package entities

import (
	"time"

	"github.com/google/uuid"
)

// EventAnalytical representa un evento analítico (datos fríos)
// almacenado en el esquema analytical.events_ts (TimescaleDB hypertable)
//
// PROPÓSITO:
// Esta tabla almacena el histórico completo de eventos para análisis temporal.
// Usa TimescaleDB hypertable para optimizar queries por rango de tiempo.
//
// NOTAS IMPORTANTES:
// - PK compuesta (created_at, id) requerida por TimescaleDB
// - Sin FK a energy_plants por limitaciones de TimescaleDB con tablas particionadas
// - La validación de plant_source_id se hace en la aplicación antes de insertar
type EventAnalytical struct {
	CreatedAt     time.Time `gorm:"primaryKey;index:idx_an_created_at" json:"created_at"`
	ID            uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	EventType     string    `gorm:"type:varchar(100);index:idx_an_event_type;not null" json:"event_type"`
	PlantSourceId uuid.UUID `gorm:"type:uuid;index:idx_an_plant_source_id;not null" json:"plant_source_id"`
	Source        string    `gorm:"type:varchar(255)" json:"source"`
	Data          string    `gorm:"type:text" json:"data"`
	Metadata      string    `gorm:"type:text" json:"metadata,omitempty"`
}

func (EventAnalytical) TableName() string {
	return "analytical.events_ts"
}
