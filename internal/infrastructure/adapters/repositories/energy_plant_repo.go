package repositories

import (
	"context"
	"errors"
	"log/slog"

	"monitoring-energy-service/internal/domain/entities"
	domainerrors "monitoring-energy-service/internal/domain/errors"
	"monitoring-energy-service/internal/domain/ports/output"

	"github.com/google/uuid"
	lru "github.com/hashicorp/golang-lru/v2"
	"gorm.io/gorm"
)

type EnergyPlantRepository struct {
	db    *gorm.DB
	cache *lru.Cache[uuid.UUID, bool]
}

var _ output.EnergyPlantRepositoryInterface = &EnergyPlantRepository{}

func NewEnergyPlantRepository(db *gorm.DB) *EnergyPlantRepository {
	// Cache for 1000 plant IDs
	cache, err := lru.New[uuid.UUID, bool](1000)
	if err != nil {
		slog.Error("failed to create LRU cache for energy plants", "error", err)
	}
	return &EnergyPlantRepository{db: db, cache: cache}
}

func (r *EnergyPlantRepository) FindByID(ctx context.Context, id uuid.UUID) (*entities.EnergyPlants, error) {
	var model EnergyPlantsModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return ToEnergyPlantsEntity(&model), nil
}

func (r *EnergyPlantRepository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	// 1. Check cache first
	if r.cache != nil {
		if exists, found := r.cache.Get(id); found {
			return exists, nil // Instant return from cache
		}
	}

	// 2. Check DB (only if not in cache)
	var count int64
	err := r.db.WithContext(ctx).Model(&EnergyPlantsModel{}).Where("id = ?", id).Count(&count).Error
	if err != nil {
		return false, err
	}
	exists := count > 0

	// 3. Store in cache (even if false to prevent cache stampede)
	if r.cache != nil {
		r.cache.Add(id, exists)
	}

	return exists, nil
}
