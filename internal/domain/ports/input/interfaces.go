package input

import (
	"context"

	"monitoring-energy-service/internal/domain/entities"

	"github.com/google/uuid"
)

// MessageHandler is the interface for handling Kafka messages
type MessageHandler interface {
	HandleMessage(ctx context.Context, message []byte) error
}

// KafkaServiceInterface defines the contract for Kafka operations
type KafkaServiceInterface interface {
	SendEvent(topic string, key string, event any) error
	RegisterHandler(topic string, handler MessageHandler)
	ConsumeEvents()
	StopConsuming()
	SendToDLQ(message []byte, reason string)
	IsConsumerHealthy() bool
}

// ExampleServiceInterface defines the contract for example business logic
type ExampleServiceInterface interface {
	GetByID(id uuid.UUID) (*entities.ExampleEntity, error)
	GetAll() ([]*entities.ExampleEntity, error)
	Create(entity *entities.ExampleEntity) (*entities.ExampleEntity, error)
	Update(entity *entities.ExampleEntity) (*entities.ExampleEntity, error)
	Delete(id uuid.UUID) error
}
