package entities

import (
	"time"

	"github.com/google/uuid"
)

// EventOperational representa un evento operacional (datos calientes)
// almacenado en el esquema operational.events_std (PostgreSQL estándar)
//
// PROPÓSITO:
// Esta tabla almacena eventos recientes para consultas operacionales frecuentes.
// Incluye FK a energy_plants para validación de integridad referencial.
type EventOperational struct {
	ID            uuid.UUID    `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	EventType     string       `gorm:"type:varchar(100);index:idx_op_event_type;not null" json:"event_type"`
	PlantSourceId uuid.UUID    `gorm:"type:uuid;index:idx_op_plant_source_id;not null" json:"plant_source_id"`
	Source        string       `gorm:"type:varchar(255)" json:"source"`
	Data          string       `gorm:"type:text" json:"data"`
	Metadata      string       `gorm:"type:text" json:"metadata,omitempty"`
	CreatedAt     time.Time    `gorm:"autoCreateTime;index:idx_op_created_at" json:"created_at"`
	PlantSource   EnergyPlants `gorm:"foreignKey:PlantSourceId;references:ID" json:"plant_source,omitempty"`
}

func (EventOperational) TableName() string {
	return "operational.events_std"
}
