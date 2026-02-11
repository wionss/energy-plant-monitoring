package repositories

import (
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

func (r *EventRepository) Create(entity *entities.EventEntity) (*entities.EventEntity, error) {
	model := ToEventModel(entity)
	if err := r.db.Create(model).Error; err != nil {
		return nil, err
	}
	return ToEventEntity(model), nil
}

func (r *EventRepository) FindAll() ([]*entities.EventEntity, error) {
	var models []*EventModel
	if err := r.db.Preload("PlantSource").Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]*entities.EventEntity, len(models))
	for i, m := range models {
		result[i] = ToEventEntity(m)
	}
	return result, nil
}

func (r *EventRepository) FindByID(id uuid.UUID) (*entities.EventEntity, error) {
	var model EventModel
	if err := r.db.Preload("PlantSource").First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return ToEventEntity(&model), nil
}

func (r *EventRepository) FindByEventType(eventType string) ([]*entities.EventEntity, error) {
	var models []*EventModel
	if err := r.db.Preload("PlantSource").Where("event_type = ?", eventType).Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]*entities.EventEntity, len(models))
	for i, m := range models {
		result[i] = ToEventEntity(m)
	}
	return result, nil
}
