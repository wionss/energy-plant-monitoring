package repositories

import (
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

func (r *EventOperationalRepository) Create(entity *entities.EventOperational) (*entities.EventOperational, error) {
	model := ToEventOperationalModel(entity)
	result := r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(model)
	if result.Error != nil {
		return nil, result.Error
	}
	return ToEventOperationalEntity(model), nil
}

func (r *EventOperationalRepository) FindAll() ([]*entities.EventOperational, error) {
	var models []*EventOperationalModel
	if err := r.db.Preload("PlantSource").Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]*entities.EventOperational, len(models))
	for i, m := range models {
		result[i] = ToEventOperationalEntity(m)
	}
	return result, nil
}

func (r *EventOperationalRepository) FindByID(id uuid.UUID) (*entities.EventOperational, error) {
	var model EventOperationalModel
	if err := r.db.Preload("PlantSource").First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return ToEventOperationalEntity(&model), nil
}

func (r *EventOperationalRepository) FindByEventType(eventType string) ([]*entities.EventOperational, error) {
	var models []*EventOperationalModel
	if err := r.db.Preload("PlantSource").Where("event_type = ?", eventType).Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]*entities.EventOperational, len(models))
	for i, m := range models {
		result[i] = ToEventOperationalEntity(m)
	}
	return result, nil
}
