package repositories

import (
	"errors"

	"monitoring-energy-service/internal/domain/entities"
	domainerrors "monitoring-energy-service/internal/domain/errors"
	"monitoring-energy-service/internal/domain/ports/output"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ExampleRepository struct {
	db *gorm.DB
}

var _ output.ExampleRepositoryInterface = &ExampleRepository{}

func NewExampleRepository(db *gorm.DB) *ExampleRepository {
	return &ExampleRepository{db: db}
}

func (r *ExampleRepository) FindByID(id uuid.UUID) (*entities.ExampleEntity, error) {
	var model ExampleModel
	if err := r.db.First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return ToExampleEntity(&model), nil
}

func (r *ExampleRepository) FindAll() ([]*entities.ExampleEntity, error) {
	var models []*ExampleModel
	if err := r.db.Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]*entities.ExampleEntity, len(models))
	for i, m := range models {
		result[i] = ToExampleEntity(m)
	}
	return result, nil
}

func (r *ExampleRepository) Create(entity *entities.ExampleEntity) (*entities.ExampleEntity, error) {
	model := ToExampleModel(entity)
	if err := r.db.Create(model).Error; err != nil {
		return nil, err
	}
	return ToExampleEntity(model), nil
}

func (r *ExampleRepository) Update(entity *entities.ExampleEntity) (*entities.ExampleEntity, error) {
	model := ToExampleModel(entity)
	if err := r.db.Save(model).Error; err != nil {
		return nil, err
	}
	return ToExampleEntity(model), nil
}

func (r *ExampleRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&ExampleModel{}, "id = ?", id).Error
}
