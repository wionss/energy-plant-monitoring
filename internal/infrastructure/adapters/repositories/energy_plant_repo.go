package repositories

import (
	"errors"

	"monitoring-energy-service/internal/domain/entities"
	domainerrors "monitoring-energy-service/internal/domain/errors"
	"monitoring-energy-service/internal/domain/ports/output"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EnergyPlantRepository struct {
	db *gorm.DB
}

var _ output.EnergyPlantRepositoryInterface = &EnergyPlantRepository{}

func NewEnergyPlantRepository(db *gorm.DB) *EnergyPlantRepository {
	return &EnergyPlantRepository{db: db}
}

func (r *EnergyPlantRepository) FindByID(id uuid.UUID) (*entities.EnergyPlants, error) {
	var model EnergyPlantsModel
	if err := r.db.First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return ToEnergyPlantsEntity(&model), nil
}

func (r *EnergyPlantRepository) Exists(id uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&EnergyPlantsModel{}).Where("id = ?", id).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
