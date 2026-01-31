package repositories

import (
	"encoding/json"
	"time"

	"monitoring-energy-service/internal/domain/entities"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ============================================================================
// GORM Persistence Models
// ============================================================================

// EventOperationalModel is the GORM persistence model for operational events.
type EventOperationalModel struct {
	ID            uuid.UUID        `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	EventType     string           `gorm:"type:varchar(100);index:idx_op_event_type;not null"`
	PlantSourceId uuid.UUID        `gorm:"type:uuid;index:idx_op_plant_source_id;not null"`
	Source        string           `gorm:"type:varchar(255)"`
	Data          datatypes.JSON   `gorm:"type:jsonb;not null"`
	Metadata      datatypes.JSON   `gorm:"type:jsonb"`
	CreatedAt     time.Time        `gorm:"autoCreateTime;index:idx_op_created_at"`
	PlantSource   EnergyPlantsModel `gorm:"foreignKey:PlantSourceId;references:ID"`
}

func (EventOperationalModel) TableName() string {
	return "operational.events_std"
}

// EventAnalyticalModel is the GORM persistence model for analytical events.
type EventAnalyticalModel struct {
	CreatedAt     time.Time      `gorm:"primaryKey;index:idx_an_created_at"`
	ID            uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	EventType     string         `gorm:"type:varchar(100);index:idx_an_event_type;not null"`
	PlantSourceId uuid.UUID      `gorm:"type:uuid;index:idx_an_plant_source_id;not null"`
	Source        string         `gorm:"type:varchar(255)"`
	Data          datatypes.JSON `gorm:"type:jsonb;not null"`
	Metadata      datatypes.JSON `gorm:"type:jsonb"`
}

func (EventAnalyticalModel) TableName() string {
	return "analytical.events_ts"
}

// EnergyPlantsModel is the GORM persistence model for energy plants.
type EnergyPlantsModel struct {
	ID         uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	PlantName  string         `gorm:"type:varchar(255);not null"`
	Location   string         `gorm:"type:varchar(255)"`
	CapacityMW float64        `gorm:"type:float"`
	CreatedAt  time.Time      `gorm:"autoCreateTime"`
	UpdatedAt  time.Time      `gorm:"autoUpdateTime"`
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

func (EnergyPlantsModel) TableName() string {
	return "master.energy_plants"
}

// EventModel is the GORM persistence model for legacy events.
type EventModel struct {
	ID            uuid.UUID         `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	EventType     string            `gorm:"type:varchar(100);index:idx_event_type;not null"`
	PlantSourceId uuid.UUID         `gorm:"type:uuid;index:idx_plant_source_id;not null"`
	Source        string            `gorm:"type:varchar(255)"`
	Data          datatypes.JSON    `gorm:"type:jsonb;not null"`
	Metadata      datatypes.JSON    `gorm:"type:jsonb"`
	CreatedAt     time.Time         `gorm:"autoCreateTime;index:idx_created_at"`
	PlantSource   EnergyPlantsModel `gorm:"foreignKey:PlantSourceId;references:ID"`
}

func (EventModel) TableName() string {
	return "events"
}

// ============================================================================
// Mappers: Entity <-> Model
// ============================================================================

func ToEventOperationalModel(e *entities.EventOperational) *EventOperationalModel {
	return &EventOperationalModel{
		ID:            e.ID,
		EventType:     e.EventType,
		PlantSourceId: e.PlantSourceId,
		Source:        e.Source,
		Data:          datatypes.JSON(e.Data),
		Metadata:      datatypes.JSON(e.Metadata),
		CreatedAt:     e.CreatedAt,
	}
}

func ToEventOperationalEntity(m *EventOperationalModel) *entities.EventOperational {
	return &entities.EventOperational{
		ID:            m.ID,
		EventType:     m.EventType,
		PlantSourceId: m.PlantSourceId,
		Source:        m.Source,
		Data:          json.RawMessage(m.Data),
		Metadata:      json.RawMessage(m.Metadata),
		CreatedAt:     m.CreatedAt,
	}
}

func ToEventAnalyticalModel(e *entities.EventAnalytical) *EventAnalyticalModel {
	return &EventAnalyticalModel{
		CreatedAt:     e.CreatedAt,
		ID:            e.ID,
		EventType:     e.EventType,
		PlantSourceId: e.PlantSourceId,
		Source:        e.Source,
		Data:          datatypes.JSON(e.Data),
		Metadata:      datatypes.JSON(e.Metadata),
	}
}

func ToEventAnalyticalEntity(m *EventAnalyticalModel) *entities.EventAnalytical {
	return &entities.EventAnalytical{
		CreatedAt:     m.CreatedAt,
		ID:            m.ID,
		EventType:     m.EventType,
		PlantSourceId: m.PlantSourceId,
		Source:        m.Source,
		Data:          json.RawMessage(m.Data),
		Metadata:      json.RawMessage(m.Metadata),
	}
}

func ToEnergyPlantsModel(e *entities.EnergyPlants) *EnergyPlantsModel {
	return &EnergyPlantsModel{
		ID:         e.ID,
		PlantName:  e.PlantName,
		Location:   e.Location,
		CapacityMW: e.CapacityMW,
		CreatedAt:  e.CreatedAt,
		UpdatedAt:  e.UpdatedAt,
	}
}

func ToEnergyPlantsEntity(m *EnergyPlantsModel) *entities.EnergyPlants {
	return &entities.EnergyPlants{
		ID:         m.ID,
		PlantName:  m.PlantName,
		Location:   m.Location,
		CapacityMW: m.CapacityMW,
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
	}
}

func ToEventModel(e *entities.EventEntity) *EventModel {
	return &EventModel{
		ID:            e.ID,
		EventType:     e.EventType,
		PlantSourceId: e.PlantSourceId,
		Source:        e.Source,
		Data:          datatypes.JSON(e.Data),
		Metadata:      datatypes.JSON(e.Metadata),
		CreatedAt:     e.CreatedAt,
	}
}

func ToEventEntity(m *EventModel) *entities.EventEntity {
	return &entities.EventEntity{
		ID:            m.ID,
		EventType:     m.EventType,
		PlantSourceId: m.PlantSourceId,
		Source:        m.Source,
		Data:          json.RawMessage(m.Data),
		Metadata:      json.RawMessage(m.Metadata),
		CreatedAt:     m.CreatedAt,
	}
}

// ExampleModel is the GORM persistence model for examples.
type ExampleModel struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name        string    `gorm:"type:varchar(255);not null"`
	Description string    `gorm:"type:text"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

func (ExampleModel) TableName() string {
	return "examples"
}

func ToExampleModel(e *entities.ExampleEntity) *ExampleModel {
	return &ExampleModel{
		ID:          e.ID,
		Name:        e.Name,
		Description: e.Description,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
}

func ToExampleEntity(m *ExampleModel) *entities.ExampleEntity {
	return &entities.ExampleEntity{
		ID:          m.ID,
		Name:        m.Name,
		Description: m.Description,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}
