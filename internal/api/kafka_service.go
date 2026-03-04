package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"

	domainerrors "monitoring-energy-service/internal/domain/errors"
	"monitoring-energy-service/internal/domain/ports/input"
	"monitoring-energy-service/internal/domain/ports/output"
	"monitoring-energy-service/internal/infrastructure/adapters/metrics"
	"monitoring-energy-service/internal/infrastructure/tracing"
)

const (
	dlqTopic   = "intake_dlq"
	maxRetries = 3
)

type KafkaService struct {
	kafkaAdapter    output.KafkaAdapterInterface
	topicHandlers   map[string]input.MessageHandler
	handlersMu      sync.RWMutex
	stopChan        chan struct{}
	stopOnce        sync.Once
	consumerHealthy atomic.Bool // tracks consumer goroutine health
	consumerCtx     context.Context
	consumerCancel  context.CancelFunc
}

var _ input.KafkaServiceInterface = &KafkaService{}

func NewKafkaService(adapter output.KafkaAdapterInterface) *KafkaService {
	consumerCtx, consumerCancel := context.WithCancel(context.Background())
	ks := &KafkaService{
		kafkaAdapter:   adapter,
		topicHandlers:  make(map[string]input.MessageHandler),
		stopChan:       make(chan struct{}),
		consumerCtx:    consumerCtx,
		consumerCancel: consumerCancel,
	}
	// Consumer starts as healthy by default
	ks.consumerHealthy.Store(true)
	return ks
}

func (ks *KafkaService) SendEvent(topic string, key string, event any) error {
	value, err := json.Marshal(event)
	if err != nil {
		return err
	}
	slog.Info("sending to kafka", "topic", topic)
	return ks.kafkaAdapter.SendMessage(topic, key, value)
}

func (ks *KafkaService) RegisterHandler(topic string, handler input.MessageHandler) {
	ks.handlersMu.Lock()
	defer ks.handlersMu.Unlock()
	ks.topicHandlers[topic] = handler
}

// SendToDLQ sends a message to the dead letter queue with metadata.
func (ks *KafkaService) SendToDLQ(message []byte, reason string) {
	dlqMsg := map[string]any{
		"original_message": json.RawMessage(message),
		"error_reason":     reason,
		"timestamp":        time.Now().UTC().Format(time.RFC3339),
	}

	payload, err := json.Marshal(dlqMsg)
	if err != nil {
		slog.Error("failed to marshal DLQ message", "error", err)
		return
	}

	if err := ks.kafkaAdapter.SendMessage(dlqTopic, "", payload); err != nil {
		slog.Error("failed to send message to DLQ", "error", err, "reason", reason)
	} else {
		slog.Warn("message sent to DLQ", "reason", reason)
	}
}

// sendToDLQWithCtx sends a message to the DLQ including the correlation ID from ctx.
func (ks *KafkaService) sendToDLQWithCtx(ctx context.Context, topic string, message []byte, reason string, metricReason string) {
	dlqMsg := map[string]any{
		"original_message": json.RawMessage(message),
		"error_reason":     reason,
		"timestamp":        time.Now().UTC().Format(time.RFC3339),
		"correlation_id":   tracing.CorrelationIDFromCtx(ctx),
	}

	payload, err := json.Marshal(dlqMsg)
	if err != nil {
		slog.Error("failed to marshal DLQ message", "error", err)
		return
	}

	if err := ks.kafkaAdapter.SendMessage(dlqTopic, "", payload); err != nil {
		slog.Error("failed to send message to DLQ", "error", err, "reason", reason)
	} else {
		slog.Warn("message sent to DLQ", "reason", reason, "correlation_id", tracing.CorrelationIDFromCtx(ctx))
		metrics.KafkaDLQSendsTotal.WithLabelValues(topic, metricReason).Inc()
	}
}

func (ks *KafkaService) ConsumeEvents() {
	slog.Info("starting Kafka consumer with manual commit enabled")

	ks.handlersMu.RLock()
	topics := make([]string, 0, len(ks.topicHandlers))
	for topic := range ks.topicHandlers {
		topics = append(topics, topic)
	}
	ks.handlersMu.RUnlock()

	if len(topics) == 0 {
		slog.Info("no topics registered, skipping Kafka consumer")
		return
	}

	if err := ks.kafkaAdapter.SubscribeTopics(topics); err != nil {
		slog.Error("failed to subscribe to topics", "error", err)
		return
	}

	connectionErrCount := 0

	for {
		select {
		case <-ks.stopChan:
			slog.Info("stopping Kafka event consumption")
			ks.consumerHealthy.Store(false)
			return
		default:
			kafkaMsg, err := ks.kafkaAdapter.ReadMessage()
			if err != nil {
				// Timeout is normal - it allows checking stopChan on each poll cycle
				if kafkaErr, ok := err.(kafka.Error); ok && kafkaErr.Code() == kafka.ErrTimedOut {
					continue
				}
				slog.Error("error reading message", "error", err)

				if kafkaErr, ok := err.(kafka.Error); ok {
					switch kafkaErr.Code() {
					case kafka.ErrTransport, kafka.ErrAllBrokersDown, kafka.ErrNetworkException:
						connectionErrCount++
						if connectionErrCount > 10 {
							slog.Error("kafka connection errors exceeded threshold, marking consumer as unhealthy",
								"error_code", kafkaErr.Code(),
								"count", connectionErrCount,
							)
							ks.consumerHealthy.Store(false)
							return
						}
						slog.Warn("kafka connection error", "error_code", kafkaErr.Code(), "count", connectionErrCount)
					}
				}

				continue
			}

			// Reset counter on successful read
			connectionErrCount = 0

			ks.handlersMu.RLock()
			handler, ok := ks.topicHandlers[kafkaMsg.Topic]
			ks.handlersMu.RUnlock()
			if !ok {
				slog.Warn("no handler registered for topic", "topic", kafkaMsg.Topic)
				// Commit even for unhandled topics to prevent reprocessing
				ks.commitOffset(kafkaMsg)
				continue
			}

			// Attach a fresh correlation ID to each message context
			msgCtx := tracing.WithCorrelationID(ks.consumerCtx, tracing.NewCorrelationID())

			slog.Info("handling message",
				"topic", kafkaMsg.Topic,
				"partition", kafkaMsg.Partition,
				"offset", kafkaMsg.Offset,
				"correlation_id", tracing.CorrelationIDFromCtx(msgCtx),
			)

			// Process message with retry logic
			success := ks.handleWithRetry(msgCtx, handler, kafkaMsg.Value, kafkaMsg.Topic)

			// PRODUCTION READY: Only commit offset after successful processing
			// This ensures at-least-once delivery semantics
			if success {
				ks.commitOffset(kafkaMsg)
			} else {
				slog.Info("processing cancelled, offset not committed - will be redelivered",
					"topic", kafkaMsg.Topic,
					"partition", kafkaMsg.Partition,
					"offset", kafkaMsg.Offset,
				)
			}
		}
	}
}

// commitOffset manually commits the Kafka offset for a message
func (ks *KafkaService) commitOffset(msg *output.KafkaMessage) {
	if err := ks.kafkaAdapter.CommitMessage(msg); err != nil {
		slog.Error("failed to commit offset",
			"topic", msg.Topic,
			"partition", msg.Partition,
			"offset", msg.Offset,
			"error", err,
		)
	}
}

// handleWithRetry returns true if the message was processed successfully (including permanent errors sent to DLQ)
func (ks *KafkaService) handleWithRetry(ctx context.Context, handler input.MessageHandler, message []byte, topic string) bool {
	err := handler.HandleMessage(ctx, message)
	if err == nil {
		return true
	}

	// Permanent errors go directly to DLQ
	if domainerrors.IsPermanent(err) {
		slog.Warn("permanent error, sending to DLQ", "topic", topic, "error", err)
		ks.sendToDLQWithCtx(ctx, topic, message, err.Error(), "permanent")
		return true // DLQ send is considered "handled"
	}

	// Transient errors: retry with exponential backoff
	for attempt := 1; attempt <= maxRetries; attempt++ {
		backoff := time.Duration(attempt*attempt) * time.Second
		slog.Warn("transient error, retrying",
			"topic", topic,
			"attempt", attempt,
			"backoff", backoff,
			"error", err,
		)
		metrics.KafkaRetryAttemptsTotal.WithLabelValues(topic).Inc()

		select {
		case <-time.After(backoff):
			// continue retry
		case <-ctx.Done():
			slog.Info("context cancelled during retry, leaving uncommitted for restart",
				"topic", topic,
				"attempt", attempt,
			)
			return false
		}

		err = handler.HandleMessage(ctx, message)
		if err == nil {
			return true
		}

		if domainerrors.IsPermanent(err) {
			slog.Warn("permanent error on retry, sending to DLQ", "topic", topic, "error", err)
			ks.sendToDLQWithCtx(ctx, topic, message, err.Error(), "permanent")
			return true // DLQ send is considered "handled"
		}
	}

	// Exhausted retries
	slog.Error("max retries exhausted, sending to DLQ",
		"topic", topic,
		"retries", maxRetries,
		"error", err,
	)
	ks.sendToDLQWithCtx(ctx, topic, message, "max retries exhausted: "+err.Error(), "max_retries")
	return true // DLQ send is considered "handled"
}

// IsConsumerHealthy returns true if the Kafka consumer goroutine is running
func (ks *KafkaService) IsConsumerHealthy() bool {
	return ks.consumerHealthy.Load()
}

func (ks *KafkaService) StopConsuming() {
	ks.stopOnce.Do(func() {
		ks.consumerCancel()
		close(ks.stopChan)
	})
}
