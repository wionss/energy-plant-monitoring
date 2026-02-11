package entities

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EventAnalytical representa un evento analítico (datos fríos)
// almacenado en el esquema analytical.events_ts (TimescaleDB hypertable)
type EventAnalytical struct {
	CreatedAt     time.Time       `json:"created_at"`
	ID            uuid.UUID       `json:"id"`
	EventType     string          `json:"event_type"`
	PlantSourceId uuid.UUID       `json:"plant_source_id"`
	Source        string          `json:"source"`
	Data          json.RawMessage `json:"data"`
	Metadata      json.RawMessage `json:"metadata,omitempty"`
}
