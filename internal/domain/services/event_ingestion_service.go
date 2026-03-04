package services

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"monitoring-energy-service/internal/domain/entities"
	domainerrors "monitoring-energy-service/internal/domain/errors"
	"monitoring-energy-service/internal/domain/ports/output"
	"monitoring-energy-service/internal/infrastructure/adapters/metrics"
	"monitoring-energy-service/internal/infrastructure/adapters/telegram"
	"monitoring-energy-service/internal/infrastructure/tracing"

	"github.com/google/uuid"
)

// EventIngestionService encapsula toda la lógica de negocio para procesar eventos
// PROPÓSITO: Extraer la lógica del IntakeHandler, haciendo el código:
// - Testeable sin Kafka
// - Reutilizable en otros contextos
// - Enfocado en reglas de negocio, no en infraestructura
type EventIngestionService struct {
	dualWriter            output.DualEventWriterInterface
	energyPlantRepository output.EnergyPlantRepositoryInterface
	plantStatusRepository output.PlantStatusRepositoryInterface
	alertEvaluator        *AlertEvaluationService
	telegramNotifier      *telegram.Notifier
	useAsyncWrite         bool
}

// NewEventIngestionService crea una nueva instancia del servicio
func NewEventIngestionService(
	dualWriter output.DualEventWriterInterface,
	energyPlantRepository output.EnergyPlantRepositoryInterface,
	plantStatusRepository output.PlantStatusRepositoryInterface,
	alertEvaluator *AlertEvaluationService,
	telegramNotifier *telegram.Notifier,
	useAsyncWrite bool,
) *EventIngestionService {
	return &EventIngestionService{
		dualWriter:            dualWriter,
		energyPlantRepository: energyPlantRepository,
		plantStatusRepository: plantStatusRepository,
		alertEvaluator:        alertEvaluator,
		telegramNotifier:      telegramNotifier,
		useAsyncWrite:         useAsyncWrite,
	}
}

// deterministicID genera un UUID v5 a partir del hash SHA-256 del payload
// Garantiza idempotencia: el mismo mensaje siempre produce el mismo ID
func deterministicID(payload []byte) uuid.UUID {
	hash := sha256.Sum256(payload)
	return uuid.NewSHA1(uuid.NameSpaceDNS, hash[:])
}

// ProcessEvent es el punto de entrada principal del servicio
// Orquesta todas las validaciones, transformaciones y persistencias
func (s *EventIngestionService) ProcessEvent(ctx context.Context, payload []byte, data map[string]interface{}) error {
	startTime := time.Now()
	log := tracing.Logger(ctx)

	// Validar estructura básica del JSON
	if data == nil || len(data) == 0 {
		log.Error("empty event data")
		metrics.EventsValidationErrors.WithLabelValues("empty_data").Inc()
		return domainerrors.NewPermanentError("empty event data", nil)
	}

	// Extraer y validar plant_source_id
	plantSourceId, eventType, source, err := s.extractAndValidatePlantInfo(data)
	if err != nil {
		return err
	}

	// Validar que la planta existe en BD
	if err := s.validatePlantExists(ctx, plantSourceId); err != nil {
		return err
	}

	// Validar datos del evento (reglas de negocio específicas)
	if err := s.validateEventData(data, eventType); err != nil {
		return err
	}

	// Crear eventos para persistencia (operational y analytical)
	eventOp, eventAn, err := s.buildEventEntities(payload, data, plantSourceId, eventType, source)
	if err != nil {
		return err
	}

	// Persistir eventos en BD
	if err := s.persistEvents(ctx, eventOp, eventAn); err != nil {
		return err
	}

	// Actualizar Digital Twin (estado actual de planta)
	s.updatePlantStatus(ctx, plantSourceId, eventType, source, payload, data)

	// Paso 4: Evaluar alertas en tiempo real (asincronamente para no bloquear)
	// Las alertas se envían asincronamente a través de Telegram sin bloquear el evento principal
	if s.alertEvaluator != nil {
		plantName := "unknown"
		if pn, ok := data["plant_name"].(string); ok {
			plantName = pn
		}

		s.alertEvaluator.EvaluateEvent(
			eventOp.ID.String(),
			eventType,
			plantSourceId.String(),
			plantName,
			data,
		)
	}

	// Registrar métricas de éxito
	metrics.EventsIngestedTotal.WithLabelValues(eventType, plantSourceId.String(), "success").Inc()
	metrics.EventProcessingDuration.WithLabelValues(eventType).Observe(time.Since(startTime).Seconds())

	return nil
}

// extractAndValidatePlantInfo extrae y valida los campos principales
func (s *EventIngestionService) extractAndValidatePlantInfo(data map[string]interface{}) (uuid.UUID, string, string, error) {
	// Extraer event_type
	eventType := "unknown"
	if et, ok := data["event_type"].(string); ok {
		eventType = et
	}

	// Extraer plant_name para source
	source := "kafka-intake"
	if plantName, ok := data["plant_name"].(string); ok {
		source = plantName
	}

	// Extraer y validar plant_source_id
	if plantSourceIdStr, ok := data["plant_source_id"].(string); ok {
		plantSourceId, err := uuid.Parse(plantSourceIdStr)
		if err != nil {
			slog.Error("invalid plant_source_id format", "error", err)
			metrics.EventsValidationErrors.WithLabelValues("invalid_uuid").Inc()
			s.telegramNotifier.SendUUIDError(
				"plant_source_id",
				plantSourceIdStr,
				fmt.Sprintf("Error al parsear UUID: %v. EventType: %s, PlantName: %s", err, eventType, source),
			)
			return uuid.Nil, "", "", domainerrors.NewPermanentError("invalid plant_source_id format", err)
		}
		return plantSourceId, eventType, source, nil
	}

	// plant_source_id es obligatorio
	slog.Error("plant_source_id not found in message")
	metrics.EventsValidationErrors.WithLabelValues("missing_plant_id").Inc()
	s.telegramNotifier.SendValidationError(
		"plant_source_id",
		"campo ausente",
		fmt.Sprintf("El campo plant_source_id no está presente en el mensaje. EventType: %s, PlantName: %s", eventType, source),
	)
	return uuid.Nil, "", "", domainerrors.NewPermanentError("missing plant_source_id field", nil)
}

// validatePlantExists verifica que la planta existe en la BD
func (s *EventIngestionService) validatePlantExists(ctx context.Context, plantSourceId uuid.UUID) error {
	log := tracing.Logger(ctx)
	exists, err := s.energyPlantRepository.Exists(ctx, plantSourceId)
	if err != nil {
		log.Error("failed to validate plant existence",
			"plant_source_id", plantSourceId,
			"error", err,
		)
		metrics.EventsValidationErrors.WithLabelValues("db_error").Inc()
		return domainerrors.NewTransientError("database error checking plant existence", err)
	}

	if !exists {
		log.Error("plant does not exist", "plant_source_id", plantSourceId)
		metrics.EventsValidationErrors.WithLabelValues("plant_not_found").Inc()
		s.telegramNotifier.SendValidationError(
			"plant_source_id",
			plantSourceId.String(),
			fmt.Sprintf("La planta con ID %s no existe en la base de datos", plantSourceId),
		)
		return domainerrors.NewTransientError(
			fmt.Sprintf("plant_source_id=%s does not exist", plantSourceId), nil,
		)
	}

	log.Info("plant validated", "plant_source_id", plantSourceId)
	return nil
}

// validateEventData valida los campos específicos del evento
func (s *EventIngestionService) validateEventData(data map[string]interface{}, eventType string) error {
	// Intentar parsear como EventData para validaciones estructurales
	// Si no se puede parsear, no es un error fatal
	rawData, err := json.Marshal(data)
	if err != nil {
		slog.Warn("could not re-marshal event data for validation", "error", err)
		return nil
	}

	var eventData entities.EventData
	if err := json.Unmarshal(rawData, &eventData); err == nil {
		if err := eventData.Validate(); err != nil {
			slog.Error("event data validation failed", "event_type", eventType, "error", err)
			metrics.EventsValidationErrors.WithLabelValues("validation_failed").Inc()
			s.telegramNotifier.SendValidationError(
				"event_data",
				"validation failed",
				fmt.Sprintf("Validation error for event type %s: %v", eventType, err),
			)
			return domainerrors.NewPermanentError("event data validation failed", err)
		}
	}

	return nil
}

// buildEventEntities construye las entidades para persistencia
func (s *EventIngestionService) buildEventEntities(
	payload []byte,
	data map[string]interface{},
	plantSourceId uuid.UUID,
	eventType string,
	source string,
) (*entities.EventOperational, *entities.EventAnalytical, error) {
	// Usar el payload raw para data JSON
	dataJSON := json.RawMessage(payload)

	// Generar ID idempotente
	now := time.Now()
	var eventId uuid.UUID
	if idStr, ok := data["id"].(string); ok {
		if parsed, err := uuid.Parse(idStr); err == nil {
			eventId = parsed
		} else {
			eventId = deterministicID(payload)
		}
	} else {
		eventId = deterministicID(payload)
	}

	eventOp := &entities.EventOperational{
		ID:            eventId,
		EventType:     eventType,
		PlantSourceId: plantSourceId,
		Source:        source,
		Data:          dataJSON,
		CreatedAt:     now,
	}

	eventAn := &entities.EventAnalytical{
		CreatedAt:     now,
		ID:            eventId,
		EventType:     eventType,
		PlantSourceId: plantSourceId,
		Source:        source,
		Data:          dataJSON,
	}

	return eventOp, eventAn, nil
}

// persistEvents persiste los eventos en BD
func (s *EventIngestionService) persistEvents(
	ctx context.Context,
	eventOp *entities.EventOperational,
	eventAn *entities.EventAnalytical,
) error {
	log := tracing.Logger(ctx)
	if s.useAsyncWrite {
		if err := s.dualWriter.SaveEventAsync(eventOp, eventAn); err != nil {
			log.Error("error saving event async", "error", err)
			metrics.EventsIngestedTotal.WithLabelValues(eventOp.EventType, eventOp.PlantSourceId.String(), "failed").Inc()
			return domainerrors.NewTransientError("async save failed", err)
		}
		log.Info("event enqueued for async save", "id", eventOp.ID, "type", eventOp.EventType)
	} else {
		if err := s.dualWriter.SaveEvent(ctx, eventOp, eventAn); err != nil {
			log.Error("error saving event to database", "error", err)
			metrics.EventsIngestedTotal.WithLabelValues(eventOp.EventType, eventOp.PlantSourceId.String(), "failed").Inc()
			return domainerrors.NewTransientError("sync save failed", err)
		}
		log.Info("event saved to both tables", "id", eventOp.ID, "type", eventOp.EventType)
	}
	return nil
}

// updatePlantStatus actualiza el Digital Twin (estado actual de planta)
// No-blocking: errores aquí no fallan el procesamiento del evento
func (s *EventIngestionService) updatePlantStatus(
	ctx context.Context,
	plantSourceId uuid.UUID,
	eventType string,
	source string,
	payload []byte,
	data map[string]interface{},
) {
	if s.plantStatusRepository == nil {
		return
	}

	currentStatus := "ACTIVE"
	if status, ok := data["current_status"].(string); ok && status != "" {
		currentStatus = status
	}

	plantStatus := &entities.PlantCurrentStatus{
		PlantID:       plantSourceId,
		LastEventData: json.RawMessage(payload),
		CurrentStatus: currentStatus,
		LastEventType: eventType,
		LastEventAt:   time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.plantStatusRepository.Upsert(ctx, plantStatus); err != nil {
		slog.Error("failed to update plant current status",
			"plant_source_id", plantSourceId,
			"error", err,
		)
	} else {
		slog.Debug("plant current status updated",
			"plant_source_id", plantSourceId,
			"current_status", currentStatus,
		)
		metrics.PlantStatusUpdates.WithLabelValues(plantSourceId.String(), currentStatus).Inc()
	}
}
