package entities

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// PlantCurrentStatus represents the current real-time status of an energy plant
// This is the "Digital Twin" model that tracks the latest state of each plant
type PlantCurrentStatus struct {
	PlantID       uuid.UUID       `json:"plant_id"`
	LastEventData json.RawMessage `json:"last_event_data"`
	CurrentStatus string          `json:"current_status"`
	LastEventType string          `json:"last_event_type"`
	LastEventAt   time.Time       `json:"last_event_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}
