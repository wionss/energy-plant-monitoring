package entities

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type EnergyPlants struct {
	ID         uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	PlantName  string         `gorm:"type:varchar(255);not null"`
	Location   string         `gorm:"type:varchar(255)"`
	CapacityMW float64        `gorm:"type:float"`
	CreatedAt  time.Time      `gorm:"autoCreateTime"`
	UpdatedAt  time.Time      `gorm:"autoUpdateTime"`
	DeletedAt  gorm.DeletedAt `gorm:"index" swaggerignore:"true"`
}

func (EnergyPlants) TableName() string {
	return "master.energy_plants"
}
