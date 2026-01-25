package repositories

import (
	"errors"

	"monitoring-energy-service/internal/domain/entities"
	domainerrors "monitoring-energy-service/internal/domain/errors"
	"monitoring-energy-service/internal/domain/ports/output"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// EventOperationalRepository implementa la capa de persistencia para eventos operacionales
// Tabla: operational.events_std (PostgreSQL estándar)
//
// PROPÓSITO:
// Proporciona acceso a la base de datos PostgreSQL para operaciones CRUD sobre
// eventos operacionales (datos calientes para consultas frecuentes)
type EventOperationalRepository struct {
	db *gorm.DB
}

var _ output.EventOperationalRepositoryInterface = &EventOperationalRepository{}

// NewEventOperationalRepository crea una nueva instancia del repositorio
func NewEventOperationalRepository(db *gorm.DB) *EventOperationalRepository {
	return &EventOperationalRepository{db: db}
}

// Create guarda un nuevo evento operacional en la base de datos
func (r *EventOperationalRepository) Create(entity *entities.EventOperational) (*entities.EventOperational, error) {
	if err := r.db.Create(entity).Error; err != nil {
		return nil, err
	}
	return entity, nil
}

// FindAll obtiene todos los eventos operacionales ordenados por fecha (más recientes primero)
func (r *EventOperationalRepository) FindAll() ([]*entities.EventOperational, error) {
	var events []*entities.EventOperational
	if err := r.db.Preload("PlantSource").Order("created_at DESC").Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}

// FindByID busca un evento operacional por su ID
func (r *EventOperationalRepository) FindByID(id uuid.UUID) (*entities.EventOperational, error) {
	var entity entities.EventOperational
	if err := r.db.Preload("PlantSource").First(&entity, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return &entity, nil
}

// FindByEventType filtra eventos operacionales por tipo
func (r *EventOperationalRepository) FindByEventType(eventType string) ([]*entities.EventOperational, error) {
	var events []*entities.EventOperational
	if err := r.db.Preload("PlantSource").Where("event_type = ?", eventType).Order("created_at DESC").Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}
