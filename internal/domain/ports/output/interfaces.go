package output

import (
	"time"

	"monitoring-energy-service/internal/domain/entities"

	"github.com/google/uuid"
)

// KafkaAdapterInterface defines the contract for Kafka adapter operations
type KafkaAdapterInterface interface {
	SendMessage(topic, key string, message []byte) error
	ReadMessage() (message []byte, topic string, err error)
	SubscribeTopics(topics []string) error
}

// WebhookAdapterInterface defines the contract for webhook operations
type WebhookAdapterInterface interface {
	SendPayload(url string, payload any) error
}

// ExampleRepositoryInterface defines the contract for example data persistence
type ExampleRepositoryInterface interface {
	FindByID(id uuid.UUID) (*entities.ExampleEntity, error)
	FindAll() ([]*entities.ExampleEntity, error)
	Create(entity *entities.ExampleEntity) (*entities.ExampleEntity, error)
	Update(entity *entities.ExampleEntity) (*entities.ExampleEntity, error)
	Delete(id uuid.UUID) error
}

// EventRepositoryInterface define el contrato para la persistencia de eventos
//
// CAMBIO REALIZADO: Interface agregada (líneas 30-35)
// RAZÓN: Sigue el patrón de arquitectura hexagonal, definiendo el contrato que
// debe implementar el repositorio de eventos (EventRepository)
//
// MÉTODOS:
// - Create: Guarda un evento consumido desde Kafka
// - FindAll: Lista todos los eventos (para API REST)
// - FindByID: Obtiene un evento específico
// - FindByEventType: Filtra eventos por tipo (power_reading, alert, etc.)
type EventRepositoryInterface interface {
	Create(entity *entities.EventEntity) (*entities.EventEntity, error)
	FindAll() ([]*entities.EventEntity, error)
	FindByID(id uuid.UUID) (*entities.EventEntity, error)
	FindByEventType(eventType string) ([]*entities.EventEntity, error)
}

// EnergyPlantRepositoryInterface define el contrato para la persistencia de plantas de energía
//
// CAMBIO REALIZADO: Interface agregada
// RAZÓN: Necesitamos validar que las plantas existen antes de guardar eventos
//
// MÉTODOS:
// - FindByID: Verifica si una planta existe por su UUID
// - Exists: Método rápido para validar existencia
type EnergyPlantRepositoryInterface interface {
	FindByID(id uuid.UUID) (*entities.EnergyPlants, error)
	Exists(id uuid.UUID) (bool, error)
}

// EventOperationalRepositoryInterface define el contrato para datos operacionales (calientes)
// Esquema: operational.events_std
type EventOperationalRepositoryInterface interface {
	Create(entity *entities.EventOperational) (*entities.EventOperational, error)
	FindAll() ([]*entities.EventOperational, error)
	FindByID(id uuid.UUID) (*entities.EventOperational, error)
	FindByEventType(eventType string) ([]*entities.EventOperational, error)
}

// EventAnalyticalRepositoryInterface define el contrato para datos analíticos (fríos)
// Esquema: analytical.events_ts (TimescaleDB hypertable)
type EventAnalyticalRepositoryInterface interface {
	Create(entity *entities.EventAnalytical) (*entities.EventAnalytical, error)
	FindByTimeRange(start, end time.Time) ([]*entities.EventAnalytical, error)
	GetHourlyAggregation(plantId uuid.UUID, start, end time.Time) ([]AggregatedEvent, error)
	GetDailyAggregation(plantId uuid.UUID, start, end time.Time) ([]AggregatedEvent, error)
}

// DualEventWriterInterface define el contrato para escritura dual a ambas tablas
type DualEventWriterInterface interface {
	SaveEvent(op *entities.EventOperational, an *entities.EventAnalytical) error
	SaveEventAsync(op *entities.EventOperational, an *entities.EventAnalytical) error
	Stop()
}

// AggregatedEvent representa datos agregados de TimescaleDB time_bucket
type AggregatedEvent struct {
	Bucket        time.Time `json:"bucket"`
	PlantSourceId uuid.UUID `json:"plant_source_id"`
	EventType     string    `json:"event_type"`
	EventCount    int64     `json:"event_count"`
}

// AnalyticsWorkerRepoInterface - contrato para agregaciones del worker de analytics
type AnalyticsWorkerRepoInterface interface {
	RecalculateDirtyBuckets(lookbackWindow time.Duration) (int, error)
	GetPendingWebhooks(limit int) ([]*entities.WebhookQueueItem, error)
	UpdateWebhookStatus(id uuid.UUID, status entities.WebhookStatus, errorMsg string) error
	GetHourlyStats(bucket time.Time, plantId uuid.UUID) (*entities.HourlyPlantStats, error)
}

// AnalyticsCoordinatorInterface - contrato para el worker de analytics
type AnalyticsCoordinatorInterface interface {
	Start()
	Stop()
}
