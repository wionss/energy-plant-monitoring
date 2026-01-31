package api

import (
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	domainerrors "monitoring-energy-service/internal/domain/errors"
	"monitoring-energy-service/internal/domain/ports/input"
	"monitoring-energy-service/internal/domain/ports/output"
)

const (
	dlqTopic   = "intake_dlq"
	maxRetries = 3
)

type KafkaService struct {
	kafkaAdapter  output.KafkaAdapterInterface
	topicHandlers map[string]input.MessageHandler
	stopChan      chan struct{}
}

var _ input.KafkaServiceInterface = &KafkaService{}

func NewKafkaService(adapter output.KafkaAdapterInterface) *KafkaService {
	return &KafkaService{
		kafkaAdapter:  adapter,
		topicHandlers: make(map[string]input.MessageHandler),
		stopChan:      make(chan struct{}),
	}
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

func (ks *KafkaService) ConsumeEvents() {
	slog.Info("starting Kafka consumer")

	topics := make([]string, 0, len(ks.topicHandlers))
	for topic := range ks.topicHandlers {
		topics = append(topics, topic)
	}

	if len(topics) == 0 {
		slog.Info("no topics registered, skipping Kafka consumer")
		return
	}

	if err := ks.kafkaAdapter.SubscribeTopics(topics); err != nil {
		slog.Error("failed to subscribe to topics", "error", err)
		return
	}

	disconnectedCount := 0
	connectionRefusedCount := 0

	for {
		select {
		case <-ks.stopChan:
			slog.Info("stopping Kafka event consumption")
			return
		default:
			message, topic, err := ks.kafkaAdapter.ReadMessage()
			if err != nil {
				slog.Error("error reading message", "error", err)

				if strings.Contains(err.Error(), "Disconnected") {
					disconnectedCount++
					if disconnectedCount > 10 {
						slog.Error("disconnected from Kafka too many times, panicking")
						panic("disconnected from kafka too many times")
					}
				}

				if strings.Contains(err.Error(), "Connection refused") {
					connectionRefusedCount++
					if connectionRefusedCount > 10 {
						slog.Error("connection refused from Kafka too many times, panicking")
						panic("connection refused from kafka too many times")
					}
				}

				continue
			}

			// Reset counters on successful read
			disconnectedCount = 0
			connectionRefusedCount = 0

			handler, ok := ks.topicHandlers[topic]
			if !ok {
				slog.Warn("no handler registered for topic", "topic", topic)
				continue
			}

			slog.Info("handling message", "topic", topic)
			ks.handleWithRetry(handler, message, topic)
		}
	}
}

func (ks *KafkaService) handleWithRetry(handler input.MessageHandler, message []byte, topic string) {
	err := handler.HandleMessage(message)
	if err == nil {
		return
	}

	// Permanent errors go directly to DLQ
	if domainerrors.IsPermanent(err) {
		slog.Warn("permanent error, sending to DLQ", "topic", topic, "error", err)
		ks.SendToDLQ(message, err.Error())
		return
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
		time.Sleep(backoff)

		err = handler.HandleMessage(message)
		if err == nil {
			return
		}

		if domainerrors.IsPermanent(err) {
			slog.Warn("permanent error on retry, sending to DLQ", "topic", topic, "error", err)
			ks.SendToDLQ(message, err.Error())
			return
		}
	}

	// Exhausted retries
	slog.Error("max retries exhausted, sending to DLQ",
		"topic", topic,
		"retries", maxRetries,
		"error", err,
	)
	ks.SendToDLQ(message, "max retries exhausted: "+err.Error())
}

func (ks *KafkaService) StopConsuming() {
	close(ks.stopChan)
}
