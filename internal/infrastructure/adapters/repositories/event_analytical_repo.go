package repositories

import (
	"time"

	"monitoring-energy-service/internal/domain/entities"
	"monitoring-energy-service/internal/domain/ports/output"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// EventAnalyticalRepository implementa la capa de persistencia para eventos analíticos
// Tabla: analytical.events_ts (TimescaleDB hypertable)
//
// PROPÓSITO:
// Proporciona acceso a la base de datos TimescaleDB para operaciones sobre
// eventos analíticos (datos fríos para análisis temporal y agregaciones)
type EventAnalyticalRepository struct {
	db *gorm.DB
}

var _ output.EventAnalyticalRepositoryInterface = &EventAnalyticalRepository{}

// NewEventAnalyticalRepository crea una nueva instancia del repositorio
func NewEventAnalyticalRepository(db *gorm.DB) *EventAnalyticalRepository {
	return &EventAnalyticalRepository{db: db}
}

// Create guarda un nuevo evento analítico en la hypertable
func (r *EventAnalyticalRepository) Create(entity *entities.EventAnalytical) (*entities.EventAnalytical, error) {
	if err := r.db.Create(entity).Error; err != nil {
		return nil, err
	}
	return entity, nil
}

// FindByTimeRange obtiene eventos analíticos dentro de un rango de tiempo
func (r *EventAnalyticalRepository) FindByTimeRange(start, end time.Time) ([]*entities.EventAnalytical, error) {
	var events []*entities.EventAnalytical
	if err := r.db.Where("created_at >= ? AND created_at <= ?", start, end).
		Order("created_at DESC").
		Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}

// GetHourlyAggregation obtiene agregaciones por hora usando time_bucket de TimescaleDB
func (r *EventAnalyticalRepository) GetHourlyAggregation(plantId uuid.UUID, start, end time.Time) ([]output.AggregatedEvent, error) {
	var results []output.AggregatedEvent

	query := `
		SELECT
			time_bucket('1 hour', created_at) AS bucket,
			plant_source_id,
			event_type,
			COUNT(*) AS event_count
		FROM analytical.events_ts
		WHERE plant_source_id = ?
			AND created_at >= ?
			AND created_at <= ?
		GROUP BY bucket, plant_source_id, event_type
		ORDER BY bucket DESC
	`

	if err := r.db.Raw(query, plantId, start, end).Scan(&results).Error; err != nil {
		return nil, err
	}

	return results, nil
}

// GetDailyAggregation obtiene agregaciones por día usando time_bucket de TimescaleDB
func (r *EventAnalyticalRepository) GetDailyAggregation(plantId uuid.UUID, start, end time.Time) ([]output.AggregatedEvent, error) {
	var results []output.AggregatedEvent

	query := `
		SELECT
			time_bucket('1 day', created_at) AS bucket,
			plant_source_id,
			event_type,
			COUNT(*) AS event_count
		FROM analytical.events_ts
		WHERE plant_source_id = ?
			AND created_at >= ?
			AND created_at <= ?
		GROUP BY bucket, plant_source_id, event_type
		ORDER BY bucket DESC
	`

	if err := r.db.Raw(query, plantId, start, end).Scan(&results).Error; err != nil {
		return nil, err
	}

	return results, nil
}
