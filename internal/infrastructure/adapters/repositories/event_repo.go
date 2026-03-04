package repositories

import (
	"context"
	"errors"

	"monitoring-energy-service/internal/domain/entities"
	domainerrors "monitoring-energy-service/internal/domain/errors"
	"monitoring-energy-service/internal/domain/ports/output"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EventRepository struct {
	db *gorm.DB
}

var _ output.EventRepositoryInterface = &EventRepository{}

func NewEventRepository(db *gorm.DB) *EventRepository {
	return &EventRepository{db: db}
}

func (r *EventRepository) Create(ctx context.Context, entity *entities.EventEntity) (*entities.EventEntity, error) {
	model := ToEventModel(entity)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return nil, err
	}
	return ToEventEntity(model), nil
}

func (r *EventRepository) FindAll(ctx context.Context, q output.PageQuery) (*output.Page[*entities.EventEntity], error) {
	q = output.NormalizePageQuery(q)

	db := r.db.WithContext(ctx).Preload("PlantSource").Order("created_at DESC, id DESC")
	if q.Cursor != nil {
		db = db.Where("(created_at < ?) OR (created_at = ? AND id < ?)",
			q.Cursor.CreatedAt, q.Cursor.CreatedAt, q.Cursor.ID)
	}

	var models []*EventModel
	if err := db.Limit(q.Limit + 1).Find(&models).Error; err != nil {
		return nil, err
	}

	hasMore := len(models) > q.Limit
	if hasMore {
		models = models[:q.Limit]
	}

	items := make([]*entities.EventEntity, len(models))
	for i, m := range models {
		items[i] = ToEventEntity(m)
	}

	page := &output.Page[*entities.EventEntity]{Items: items, Count: len(items)}
	if hasMore && len(models) > 0 {
		last := models[len(models)-1]
		page.NextCursor = output.EncodeCursor(last.CreatedAt, last.ID)
	}
	return page, nil
}

func (r *EventRepository) FindByID(ctx context.Context, id uuid.UUID) (*entities.EventEntity, error) {
	var model EventModel
	if err := r.db.WithContext(ctx).Preload("PlantSource").First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return ToEventEntity(&model), nil
}

func (r *EventRepository) FindByEventType(ctx context.Context, eventType string, q output.PageQuery) (*output.Page[*entities.EventEntity], error) {
	q = output.NormalizePageQuery(q)

	db := r.db.WithContext(ctx).Preload("PlantSource").Where("event_type = ?", eventType).Order("created_at DESC, id DESC")
	if q.Cursor != nil {
		db = db.Where("(created_at < ?) OR (created_at = ? AND id < ?)",
			q.Cursor.CreatedAt, q.Cursor.CreatedAt, q.Cursor.ID)
	}

	var models []*EventModel
	if err := db.Limit(q.Limit + 1).Find(&models).Error; err != nil {
		return nil, err
	}

	hasMore := len(models) > q.Limit
	if hasMore {
		models = models[:q.Limit]
	}

	items := make([]*entities.EventEntity, len(models))
	for i, m := range models {
		items[i] = ToEventEntity(m)
	}

	page := &output.Page[*entities.EventEntity]{Items: items, Count: len(items)}
	if hasMore && len(models) > 0 {
		last := models[len(models)-1]
		page.NextCursor = output.EncodeCursor(last.CreatedAt, last.ID)
	}
	return page, nil
}
