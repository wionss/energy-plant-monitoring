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
	PlantType  string         `gorm:"type:varchar(50);default:'solar'"`
	Location   string         `gorm:"type:varchar(255)"`
	Latitude   *float64       `gorm:"type:double precision"`
	Longitude  *float64       `gorm:"type:double precision"`
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
		PlantType:  e.PlantType,
		Location:   e.Location,
		Latitude:   e.Latitude,
		Longitude:  e.Longitude,
		CapacityMW: e.CapacityMW,
		CreatedAt:  e.CreatedAt,
		UpdatedAt:  e.UpdatedAt,
	}
}

func ToEnergyPlantsEntity(m *EnergyPlantsModel) *entities.EnergyPlants {
	return &entities.EnergyPlants{
		ID:         m.ID,
		PlantName:  m.PlantName,
		PlantType:  m.PlantType,
		Location:   m.Location,
		Latitude:   m.Latitude,
		Longitude:  m.Longitude,
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

// ============================================================================
// Analytics Worker Models
// ============================================================================

// HourlyPlantStatsModel - estadísticas horarias pre-calculadas
type HourlyPlantStatsModel struct {
	Bucket        time.Time `gorm:"type:timestamptz;primaryKey"`
	PlantSourceId uuid.UUID `gorm:"type:uuid;primaryKey"`
	// Campos desnormalizados de master.energy_plants
	PlantName string   `gorm:"type:varchar(255)"`
	PlantType string   `gorm:"type:varchar(50);index:idx_hps_plant_type"`
	Latitude  *float64 `gorm:"type:double precision"`
	Longitude *float64 `gorm:"type:double precision"`
	// Métricas calculadas
	AvgPowerGen      *float64  `gorm:"type:double precision"`
	AvgPowerCon      *float64  `gorm:"type:double precision"`
	AvgEfficiency    *float64  `gorm:"type:double precision"`
	AvgTemp          *float64  `gorm:"type:double precision"`
	SampleCount      int       `gorm:"not null;default:0"`
	LastCalculatedAt time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

func (HourlyPlantStatsModel) TableName() string {
	return "analytical.hourly_plant_stats"
}

// WebhookQueueModel - cola de despacho de webhooks
type WebhookQueueModel struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Bucket        time.Time  `gorm:"type:timestamptz;not null;uniqueIndex:idx_wq_bucket_plant,priority:1"`
	PlantSourceId uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:idx_wq_bucket_plant,priority:2"`
	Status        string     `gorm:"type:varchar(20);not null;default:'PENDING';index:idx_wq_status"`
	Attempts      int        `gorm:"not null;default:0"`
	LastAttempt   *time.Time `gorm:"type:timestamptz"`
	NextRetryAt   *time.Time `gorm:"type:timestamptz;index:idx_wq_next_retry"`
	ErrorMessage  *string    `gorm:"type:text"`
	CreatedAt     time.Time  `gorm:"type:timestamptz;not null;default:now()"`
}

func (WebhookQueueModel) TableName() string {
	return "operational.webhook_queue"
}

// Mappers for Analytics Worker

func ToHourlyPlantStatsEntity(m *HourlyPlantStatsModel) *entities.HourlyPlantStats {
	return &entities.HourlyPlantStats{
		Bucket:           m.Bucket,
		PlantSourceId:    m.PlantSourceId,
		PlantName:        m.PlantName,
		PlantType:        m.PlantType,
		Latitude:         m.Latitude,
		Longitude:        m.Longitude,
		AvgPowerGen:      m.AvgPowerGen,
		AvgPowerCon:      m.AvgPowerCon,
		AvgEfficiency:    m.AvgEfficiency,
		AvgTemp:          m.AvgTemp,
		SampleCount:      m.SampleCount,
		LastCalculatedAt: m.LastCalculatedAt,
	}
}

func ToWebhookQueueEntity(m *WebhookQueueModel) *entities.WebhookQueueItem {
	return &entities.WebhookQueueItem{
		ID:            m.ID,
		Bucket:        m.Bucket,
		PlantSourceId: m.PlantSourceId,
		Status:        entities.WebhookStatus(m.Status),
		Attempts:      m.Attempts,
		LastAttempt:   m.LastAttempt,
		NextRetryAt:   m.NextRetryAt,
		ErrorMessage:  m.ErrorMessage,
		CreatedAt:     m.CreatedAt,
	}
}

// ============================================================================
// Digital Twin: Plant Current Status
// ============================================================================

// PlantCurrentStatusModel - real-time plant status (Digital Twin)
type PlantCurrentStatusModel struct {
	PlantID       uuid.UUID      `gorm:"type:uuid;primaryKey"`
	LastEventData datatypes.JSON `gorm:"type:jsonb;not null"`
	CurrentStatus string         `gorm:"type:varchar(50);not null;default:'UNKNOWN';index:idx_pcs_status"`
	LastEventType string         `gorm:"type:varchar(100)"`
	LastEventAt   time.Time      `gorm:"type:timestamptz;not null"`
	UpdatedAt     time.Time      `gorm:"type:timestamptz;not null;default:now();index:idx_pcs_updated_at"`
}

func (PlantCurrentStatusModel) TableName() string {
	return "master.plant_current_status"
}

// Mappers for PlantCurrentStatus

func ToPlantCurrentStatusEntity(m *PlantCurrentStatusModel) *entities.PlantCurrentStatus {
	return &entities.PlantCurrentStatus{
		PlantID:       m.PlantID,
		LastEventData: json.RawMessage(m.LastEventData),
		CurrentStatus: m.CurrentStatus,
		LastEventType: m.LastEventType,
		LastEventAt:   m.LastEventAt,
		UpdatedAt:     m.UpdatedAt,
	}
}

func ToPlantCurrentStatusModel(e *entities.PlantCurrentStatus) *PlantCurrentStatusModel {
	return &PlantCurrentStatusModel{
		PlantID:       e.PlantID,
		LastEventData: datatypes.JSON(e.LastEventData),
		CurrentStatus: e.CurrentStatus,
		LastEventType: e.LastEventType,
		LastEventAt:   e.LastEventAt,
		UpdatedAt:     e.UpdatedAt,
	}
}
