package kafkaconf

import (
	"log/slog"
	"strings"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

func deliveryReport(deliveryChan chan kafka.Event) {
	for e := range deliveryChan {
		switch ev := e.(type) {
		case *kafka.Message:
			if ev.TopicPartition.Error != nil {
				slog.Error("delivery failed", "error", ev.TopicPartition.Error)
			} else {
				slog.Debug("message delivered", "partition", ev.TopicPartition)
			}
		}
	}
}

type KafkaFactory struct {
	brokerList string
	autoOffset string
}

func NewKafkaFactory(brokers []string, autoOffset string) *KafkaFactory {
	brokerList := strings.Join(brokers, ",")

	return &KafkaFactory{
		brokerList: brokerList,
		autoOffset: autoOffset,
	}
}

func (kf *KafkaFactory) NewProducer() (*kafka.Producer, error) {
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": kf.brokerList,
		"retries":           1,
		"socket.timeout.ms": 5000,
	})

	if err != nil {
		slog.Error("failed to create Kafka producer", "error", err)
		return nil, err
	}

	go deliveryReport(p.Events())

	return p, nil
}

func (kf *KafkaFactory) NewConsumer(groupID string) (*kafka.Consumer, error) {
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":     kf.brokerList,
		"group.id":              groupID,
		"auto.offset.reset":     kf.autoOffset,
		"session.timeout.ms":    10000,
		"heartbeat.interval.ms": 3000,
		// PRODUCTION READY: Manual commit for data integrity
		// Offset is only committed after successful message processing
		"enable.auto.commit":   false,
		"max.poll.interval.ms": 300000,
	})
	if err != nil {
		slog.Error("failed to create Kafka consumer", "error", err)
		return nil, err
	}
	return c, nil
}
