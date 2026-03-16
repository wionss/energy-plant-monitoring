package entities

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EventAnalytical represents an analytical event (cold data)
// stored in the analytical.events_ts schema (TimescaleDB hypertable)
type EventAnalytical struct {
	CreatedAt     time.Time       `json:"created_at"`
	ID            uuid.UUID       `json:"id"`
	EventType     string          `json:"event_type"`
	PlantSourceId uuid.UUID       `json:"plant_source_id"`
	Source        string          `json:"source"`
	Data          json.RawMessage `json:"data"`
	Metadata      json.RawMessage `json:"metadata,omitempty"`
}
