package repositories

import (
	"context"
	"fmt"

	"monitoring-energy-service/internal/domain/entities"
	"monitoring-energy-service/internal/domain/ports/output"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PlantStatusRepository implements the Digital Twin status tracking
type PlantStatusRepository struct {
	db *gorm.DB
}

var _ output.PlantStatusRepositoryInterface = &PlantStatusRepository{}

// NewPlantStatusRepository creates a new plant status repository
func NewPlantStatusRepository(db *gorm.DB) *PlantStatusRepository {
	return &PlantStatusRepository{db: db}
}

// Upsert creates or updates the current status of a plant
// Uses INSERT ... ON CONFLICT for atomic upsert
func (r *PlantStatusRepository) Upsert(ctx context.Context, status *entities.PlantCurrentStatus) error {
	model := ToPlantCurrentStatusModel(status)

	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "plant_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"last_event_data", "current_status", "last_event_type", "last_event_at", "updated_at"}),
	}).Create(model)

	if result.Error != nil {
		return fmt.Errorf("failed to upsert plant status: %w", result.Error)
	}

	return nil
}

// GetByPlantID retrieves the current status of a specific plant
func (r *PlantStatusRepository) GetByPlantID(ctx context.Context, plantID uuid.UUID) (*entities.PlantCurrentStatus, error) {
	var model PlantCurrentStatusModel

	if err := r.db.WithContext(ctx).Where("plant_id = ?", plantID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get plant status: %w", err)
	}

	return ToPlantCurrentStatusEntity(&model), nil
}

// GetAll retrieves the current status of all plants
func (r *PlantStatusRepository) GetAll(ctx context.Context) ([]*entities.PlantCurrentStatus, error) {
	var models []PlantCurrentStatusModel

	if err := r.db.WithContext(ctx).Order("updated_at DESC").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("failed to get all plant statuses: %w", err)
	}

	result := make([]*entities.PlantCurrentStatus, len(models))
	for i := range models {
		result[i] = ToPlantCurrentStatusEntity(&models[i])
	}

	return result, nil
}

// GetByStatus retrieves plants filtered by their current status
func (r *PlantStatusRepository) GetByStatus(ctx context.Context, status string) ([]*entities.PlantCurrentStatus, error) {
	var models []PlantCurrentStatusModel

	if err := r.db.WithContext(ctx).Where("current_status = ?", status).Order("updated_at DESC").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("failed to get plants by status: %w", err)
	}

	result := make([]*entities.PlantCurrentStatus, len(models))
	for i := range models {
		result[i] = ToPlantCurrentStatusEntity(&models[i])
	}

	return result, nil
}
