package kafka

import (
	"log/slog"

	"monitoring-energy-service/internal/domain/ports/output"
	"monitoring-energy-service/internal/infrastructure/conf/kafkaconf"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type KafkaAdapter struct {
	producer *kafka.Producer
	consumer *kafka.Consumer
}

var _ output.KafkaAdapterInterface = &KafkaAdapter{}

func NewKafkaAdapter(factory *kafkaconf.KafkaFactory, groupID string) *KafkaAdapter {
	return &KafkaAdapter{
		producer: factory.NewProducer(),
		consumer: factory.NewConsumer(groupID),
	}
}

func (ka *KafkaAdapter) SubscribeTopics(topics []string) error {
	return ka.consumer.SubscribeTopics(topics, nil)
}

func (ka *KafkaAdapter) SendMessage(topic, key string, message []byte) error {
	err := ka.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Key:            []byte(key),
		Value:          message,
	}, nil)
	if err != nil {
		return err
	}

	ka.producer.Flush(15 * 1000)

	return nil
}

// ReadMessage reads the next message from Kafka and returns it with metadata for manual commit.
func (ka *KafkaAdapter) ReadMessage() (*output.KafkaMessage, error) {
	msg, err := ka.consumer.ReadMessage(-1)
	if err != nil {
		return nil, err
	}

	return &output.KafkaMessage{
		Value:     msg.Value,
		Topic:     *msg.TopicPartition.Topic,
		Partition: msg.TopicPartition.Partition,
		Offset:    int64(msg.TopicPartition.Offset),
	}, nil
}

// CommitMessage manually commits the offset for the given message.
// This ensures at-least-once delivery semantics by only committing after successful processing.
func (ka *KafkaAdapter) CommitMessage(msg *output.KafkaMessage) error {
	tp := kafka.TopicPartition{
		Topic:     &msg.Topic,
		Partition: msg.Partition,
		Offset:    kafka.Offset(msg.Offset + 1), // Commit the next offset to read
	}

	_, err := ka.consumer.CommitOffsets([]kafka.TopicPartition{tp})
	if err != nil {
		slog.Error("failed to commit Kafka offset",
			"topic", msg.Topic,
			"partition", msg.Partition,
			"offset", msg.Offset,
			"error", err,
		)
		return err
	}

	slog.Debug("Kafka offset committed",
		"topic", msg.Topic,
		"partition", msg.Partition,
		"offset", msg.Offset,
	)
	return nil
}
