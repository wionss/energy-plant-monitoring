package entities

import (
	"time"

	"github.com/google/uuid"
)

// EnergyPlants representa una planta de energía en el dominio.
type EnergyPlants struct {
	ID         uuid.UUID  `json:"id"`
	PlantName  string     `json:"plant_name"`
	PlantType  string     `json:"plant_type"`
	Location   string     `json:"location"`
	Latitude   *float64   `json:"latitude"`
	Longitude  *float64   `json:"longitude"`
	CapacityMW float64    `json:"capacity_mw"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"`
}
