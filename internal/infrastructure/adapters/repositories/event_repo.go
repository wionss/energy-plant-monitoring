package repositories

import (
	"errors"

	"monitoring-energy-service/internal/domain/entities"
	domainerrors "monitoring-energy-service/internal/domain/errors"
	"monitoring-energy-service/internal/domain/ports/output"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// EventRepository implementa la capa de persistencia para eventos
//
// PROPÓSITO:
// Proporciona acceso a la base de datos PostgreSQL para operaciones CRUD sobre eventos.
// Sigue el patrón Repository para abstraer la lógica de acceso a datos.
//
// CAMBIO REALIZADO: Archivo creado desde cero
// RAZÓN: Necesitábamos un repositorio para gestionar la persistencia de eventos en PostgreSQL
type EventRepository struct {
	db *gorm.DB
}

var _ output.EventRepositoryInterface = &EventRepository{}

// NewEventRepository crea una nueva instancia del repositorio de eventos
// PARÁMETROS: db - Conexión GORM a PostgreSQL
func NewEventRepository(db *gorm.DB) *EventRepository {
	return &EventRepository{db: db}
}

// Create guarda un nuevo evento en la base de datos
// CAMBIO: Método nuevo
// RAZÓN: Permite al IntakeHandler guardar eventos consumidos desde Kafka
func (r *EventRepository) Create(entity *entities.EventEntity) (*entities.EventEntity, error) {
	if err := r.db.Create(entity).Error; err != nil {
		return nil, err
	}
	return entity, nil
}

// FindAll obtiene todos los eventos ordenados por fecha de creación (más recientes primero)
// CAMBIO: Método nuevo
// RAZÓN: Permite a la API REST devolver todos los eventos para consulta
// CAMBIO: Agregado Preload para cargar la relación con EnergyPlants
// RAZÓN: Permite mostrar los nombres de las plantas en las consultas de Swagger
func (r *EventRepository) FindAll() ([]*entities.EventEntity, error) {
	var entities []*entities.EventEntity
	if err := r.db.Preload("PlantSource").Order("created_at DESC").Find(&entities).Error; err != nil {
		return nil, err
	}
	return entities, nil
}

// FindByID busca un evento específico por su ID
// CAMBIO: Método nuevo
// RAZÓN: Permite a la API REST obtener un evento individual
// CAMBIO: Ahora traduce gorm.ErrRecordNotFound a domainerrors.ErrNotFound
// RAZÓN: Permite a los handlers distinguir entre 404 (not found) y 500 (internal error)
// CAMBIO: Agregado Preload para cargar la relación con EnergyPlants
// RAZÓN: Permite mostrar los nombres de las plantas en las consultas de Swagger
func (r *EventRepository) FindByID(id uuid.UUID) (*entities.EventEntity, error) {
	var entity entities.EventEntity
	if err := r.db.Preload("PlantSource").First(&entity, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return &entity, nil
}

// FindByEventType filtra eventos por tipo (power_reading, alert, etc.)
// CAMBIO: Método nuevo
// RAZÓN: Permite filtrar eventos por tipo para análisis específicos
// CAMBIO: Agregado Preload para cargar la relación con EnergyPlants
// RAZÓN: Permite mostrar los nombres de las plantas en las consultas de Swagger
func (r *EventRepository) FindByEventType(eventType string) ([]*entities.EventEntity, error) {
	var entities []*entities.EventEntity
	if err := r.db.Preload("PlantSource").Where("event_type = ?", eventType).Order("created_at DESC").Find(&entities).Error; err != nil {
		return nil, err
	}
	return entities, nil
}
