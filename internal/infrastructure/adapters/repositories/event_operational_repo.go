package repositories

import (
	"context"
	"errors"

	"monitoring-energy-service/internal/domain/entities"
	domainerrors "monitoring-energy-service/internal/domain/errors"
	"monitoring-energy-service/internal/domain/ports/output"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type EventOperationalRepository struct {
	db *gorm.DB
}

var _ output.EventOperationalRepositoryInterface = &EventOperationalRepository{}

func NewEventOperationalRepository(db *gorm.DB) *EventOperationalRepository {
	return &EventOperationalRepository{db: db}
}

func (r *EventOperationalRepository) Create(ctx context.Context, entity *entities.EventOperational) (*entities.EventOperational, error) {
	model := ToEventOperationalModel(entity)
	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(model)
	if result.Error != nil {
		return nil, result.Error
	}
	return ToEventOperationalEntity(model), nil
}

func (r *EventOperationalRepository) FindAll(ctx context.Context, q output.PageQuery) (*output.Page[*entities.EventOperational], error) {
	q = output.NormalizePageQuery(q)

	db := r.db.WithContext(ctx).Preload("PlantSource").Order("created_at DESC, id DESC")
	if q.Cursor != nil {
		db = db.Where("(created_at < ?) OR (created_at = ? AND id < ?)",
			q.Cursor.CreatedAt, q.Cursor.CreatedAt, q.Cursor.ID)
	}

	var models []*EventOperationalModel
	if err := db.Limit(q.Limit + 1).Find(&models).Error; err != nil {
		return nil, err
	}

	hasMore := len(models) > q.Limit
	if hasMore {
		models = models[:q.Limit]
	}

	items := make([]*entities.EventOperational, len(models))
	for i, m := range models {
		items[i] = ToEventOperationalEntity(m)
	}

	page := &output.Page[*entities.EventOperational]{Items: items, Count: len(items)}
	if hasMore && len(models) > 0 {
		last := models[len(models)-1]
		page.NextCursor = output.EncodeCursor(last.CreatedAt, last.ID)
	}
	return page, nil
}

func (r *EventOperationalRepository) FindByID(ctx context.Context, id uuid.UUID) (*entities.EventOperational, error) {
	var model EventOperationalModel
	if err := r.db.WithContext(ctx).Preload("PlantSource").First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return ToEventOperationalEntity(&model), nil
}

func (r *EventOperationalRepository) FindByEventType(ctx context.Context, eventType string, q output.PageQuery) (*output.Page[*entities.EventOperational], error) {
	q = output.NormalizePageQuery(q)

	db := r.db.WithContext(ctx).Preload("PlantSource").Where("event_type = ?", eventType).Order("created_at DESC, id DESC")
	if q.Cursor != nil {
		db = db.Where("(created_at < ?) OR (created_at = ? AND id < ?)",
			q.Cursor.CreatedAt, q.Cursor.CreatedAt, q.Cursor.ID)
	}

	var models []*EventOperationalModel
	if err := db.Limit(q.Limit + 1).Find(&models).Error; err != nil {
		return nil, err
	}

	hasMore := len(models) > q.Limit
	if hasMore {
		models = models[:q.Limit]
	}

	items := make([]*entities.EventOperational, len(models))
	for i, m := range models {
		items[i] = ToEventOperationalEntity(m)
	}

	page := &output.Page[*entities.EventOperational]{Items: items, Count: len(items)}
	if hasMore && len(models) > 0 {
		last := models[len(models)-1]
		page.NextCursor = output.EncodeCursor(last.CreatedAt, last.ID)
	}
	return page, nil
}
