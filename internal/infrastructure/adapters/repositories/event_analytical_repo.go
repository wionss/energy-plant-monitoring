package repositories

import (
	"context"
	"time"

	"monitoring-energy-service/internal/domain/entities"
	"monitoring-energy-service/internal/domain/ports/output"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type EventAnalyticalRepository struct {
	db *gorm.DB
}

var _ output.EventAnalyticalRepositoryInterface = &EventAnalyticalRepository{}

func NewEventAnalyticalRepository(db *gorm.DB) *EventAnalyticalRepository {
	return &EventAnalyticalRepository{db: db}
}

func (r *EventAnalyticalRepository) Create(ctx context.Context, entity *entities.EventAnalytical) (*entities.EventAnalytical, error) {
	model := ToEventAnalyticalModel(entity)
	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(model)
	if result.Error != nil {
		return nil, result.Error
	}
	return ToEventAnalyticalEntity(model), nil
}

func (r *EventAnalyticalRepository) FindByTimeRange(ctx context.Context, start, end time.Time) ([]*entities.EventAnalytical, error) {
	var models []*EventAnalyticalModel
	if err := r.db.WithContext(ctx).Where("created_at >= ? AND created_at <= ?", start, end).
		Order("created_at DESC").
		Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]*entities.EventAnalytical, len(models))
	for i, m := range models {
		result[i] = ToEventAnalyticalEntity(m)
	}
	return result, nil
}

func (r *EventAnalyticalRepository) GetHourlyAggregation(ctx context.Context, plantId uuid.UUID, start, end time.Time) ([]output.AggregatedEvent, error) {
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

	if err := r.db.WithContext(ctx).Raw(query, plantId, start, end).Scan(&results).Error; err != nil {
		return nil, err
	}

	return results, nil
}

func (r *EventAnalyticalRepository) GetDailyAggregation(ctx context.Context, plantId uuid.UUID, start, end time.Time) ([]output.AggregatedEvent, error) {
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

	if err := r.db.WithContext(ctx).Raw(query, plantId, start, end).Scan(&results).Error; err != nil {
		return nil, err
	}

	return results, nil
}
