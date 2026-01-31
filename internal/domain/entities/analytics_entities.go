package entities

import (
	"time"

	"github.com/google/uuid"
)

// WebhookStatus represents the status of a webhook queue item
type WebhookStatus string

const (
	WebhookStatusPending  WebhookStatus = "PENDING"
	WebhookStatusSent     WebhookStatus = "SENT"
	WebhookStatusFailed   WebhookStatus = "FAILED"
	WebhookStatusRetrying WebhookStatus = "RETRYING"
)

// HourlyPlantStats represents pre-calculated hourly statistics for a plant
type HourlyPlantStats struct {
	Bucket           time.Time
	PlantSourceId    uuid.UUID
	AvgPowerGen      *float64
	AvgPowerCon      *float64
	AvgEfficiency    *float64
	AvgTemp          *float64
	SampleCount      int
	LastCalculatedAt time.Time
}

// WebhookQueueItem represents an item in the webhook dispatch queue
type WebhookQueueItem struct {
	ID            uuid.UUID
	Bucket        time.Time
	PlantSourceId uuid.UUID
	Status        WebhookStatus
	Attempts      int
	LastAttempt   *time.Time
	NextRetryAt   *time.Time
	ErrorMessage  *string
	CreatedAt     time.Time
}
