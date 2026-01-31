package entities

import (
	"time"

	"github.com/google/uuid"
)

// EnergyPlants representa una planta de energía en el dominio.
type EnergyPlants struct {
	ID         uuid.UUID  `json:"id"`
	PlantName  string     `json:"plant_name"`
	Location   string     `json:"location"`
	CapacityMW float64    `json:"capacity_mw"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"`
}
