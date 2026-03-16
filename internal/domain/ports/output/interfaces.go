package output

import (
	"context"
	"time"

	"monitoring-energy-service/internal/domain/entities"

	"github.com/google/uuid"
)

// KafkaMessage wraps a Kafka message with metadata for manual commit support
type KafkaMessage struct {
	Value     []byte
	Topic     string
	Partition int32
	Offset    int64
}

// KafkaAdapterInterface defines the contract for Kafka adapter operations
type KafkaAdapterInterface interface {
	SendMessage(topic, key string, message []byte) error
	// ReadMessage returns a KafkaMessage with metadata for manual commit support
	ReadMessage() (*KafkaMessage, error)
	SubscribeTopics(topics []string) error
	// CommitMessage manually commits the offset for the given message
	// This should only be called after successful processing
	CommitMessage(msg *KafkaMessage) error
	// Close flushes pending messages and releases producer/consumer resources
	Close()
}

// WebhookAdapterInterface defines the contract for webhook operations
type WebhookAdapterInterface interface {
	SendPayload(url string, payload any) error
}

// ExampleRepositoryInterface defines the contract for example data persistence
type ExampleRepositoryInterface interface {
	FindByID(ctx context.Context, id uuid.UUID) (*entities.ExampleEntity, error)
	FindAll(ctx context.Context) ([]*entities.ExampleEntity, error)
	Create(ctx context.Context, entity *entities.ExampleEntity) (*entities.ExampleEntity, error)
	Update(ctx context.Context, entity *entities.ExampleEntity) (*entities.ExampleEntity, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// AlertRulesRepositoryInterface defines the contract for fetching alert rules from the DB.
type AlertRulesRepositoryInterface interface {
	FindActive() ([]entities.AlertRule, error)
}

// EnergyPlantRepositoryInterface defines the contract for persisting energy plants.
//
// EnergyPlantRepositoryInterface validates energy plant existence during event ingestion.
// This prevents saving events for non-existent plants, ensuring data consistency and
// enabling the dual-write strategy to selectively update plant status records.
// Methods:
// - FindByID: Returns plant details by UUID for full validation
// - Exists: Optimized check for rapid plant existence validation in hot paths
type EnergyPlantRepositoryInterface interface {
	FindByID(ctx context.Context, id uuid.UUID) (*entities.EnergyPlants, error)
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
}

// EventOperationalRepositoryInterface defines the contract for operational data (hot data)
// Schema: operational.events_std
type EventOperationalRepositoryInterface interface {
	Create(ctx context.Context, entity *entities.EventOperational) (*entities.EventOperational, error)
	FindAll(ctx context.Context, q PageQuery) (*Page[*entities.EventOperational], error)
	FindByID(ctx context.Context, id uuid.UUID) (*entities.EventOperational, error)
	FindByEventType(ctx context.Context, eventType string, q PageQuery) (*Page[*entities.EventOperational], error)
}

// EventAnalyticalRepositoryInterface defines the contract for analytical data (cold data)
// Schema: analytical.events_ts (TimescaleDB hypertable)
type EventAnalyticalRepositoryInterface interface {
	Create(ctx context.Context, entity *entities.EventAnalytical) (*entities.EventAnalytical, error)
	FindByTimeRange(ctx context.Context, start, end time.Time) ([]*entities.EventAnalytical, error)
	GetHourlyAggregation(ctx context.Context, plantId uuid.UUID, start, end time.Time) ([]AggregatedEvent, error)
	GetDailyAggregation(ctx context.Context, plantId uuid.UUID, start, end time.Time) ([]AggregatedEvent, error)
}

// DualEventWriterInterface defines the contract for dual-writing to both tables
type DualEventWriterInterface interface {
	SaveEvent(ctx context.Context, op *entities.EventOperational, an *entities.EventAnalytical) error
	SaveEventAsync(op *entities.EventOperational, an *entities.EventAnalytical) error // no ctx: non-blocking enqueue
	Stop()
}

// AggregatedEvent representa datos agregados de TimescaleDB time_bucket
type AggregatedEvent struct {
	Bucket        time.Time `json:"bucket"`
	PlantSourceId uuid.UUID `json:"plant_source_id"`
	EventType     string    `json:"event_type"`
	EventCount    int64     `json:"event_count"`
}

// AnalyticsWorkerRepoInterface - contract for analytics worker aggregations
type AnalyticsWorkerRepoInterface interface {
	RecalculateDirtyBuckets(lookbackWindow time.Duration) (int, error)
	GetPendingWebhooks(limit int) ([]*entities.WebhookQueueItem, error)
	UpdateWebhookStatus(id uuid.UUID, status entities.WebhookStatus, errorMsg string) error
	GetHourlyStats(bucket time.Time, plantId uuid.UUID) (*entities.HourlyPlantStats, error)
}

// AnalyticsCoordinatorInterface - contract for the analytics worker
type AnalyticsCoordinatorInterface interface {
	Start()
	Stop()
}

// PlantStatusRepositoryInterface - contract for the plant status Digital Twin
type PlantStatusRepositoryInterface interface {
	Upsert(ctx context.Context, status *entities.PlantCurrentStatus) error
	GetByPlantID(ctx context.Context, plantID uuid.UUID) (*entities.PlantCurrentStatus, error)
	GetAll(ctx context.Context) ([]*entities.PlantCurrentStatus, error)
	GetByStatus(ctx context.Context, status string) ([]*entities.PlantCurrentStatus, error)
}
