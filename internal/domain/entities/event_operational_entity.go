package entities

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EventOperational representa un evento operacional (datos calientes)
// almacenado en el esquema operational.events_std (PostgreSQL estándar)
type EventOperational struct {
	ID            uuid.UUID       `json:"id"`
	EventType     string          `json:"event_type"`
	PlantSourceId uuid.UUID       `json:"plant_source_id"`
	Source        string          `json:"source"`
	Data          json.RawMessage `json:"data"`
	Metadata      json.RawMessage `json:"metadata,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}
