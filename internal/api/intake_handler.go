package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"monitoring-energy-service/internal/domain/entities"
	domainerrors "monitoring-energy-service/internal/domain/errors"
	"monitoring-energy-service/internal/domain/ports/input"
	"monitoring-energy-service/internal/domain/ports/output"
	"monitoring-energy-service/internal/infrastructure/adapters/metrics"
	"monitoring-energy-service/internal/infrastructure/adapters/telegram"

	"github.com/google/uuid"
)

// IntakeHandler procesa mensajes consumidos desde Kafka
type IntakeHandler struct {
	dualWriter            output.DualEventWriterInterface
	energyPlantRepository output.EnergyPlantRepositoryInterface
	plantStatusRepository output.PlantStatusRepositoryInterface
	telegramNotifier      *telegram.Notifier
	useAsyncWrite         bool
}

var _ input.MessageHandler = &IntakeHandler{}

func NewIntakeHandler(
	dualWriter output.DualEventWriterInterface,
	energyPlantRepository output.EnergyPlantRepositoryInterface,
	plantStatusRepository output.PlantStatusRepositoryInterface,
	telegramNotifier *telegram.Notifier,
	useAsyncWrite bool,
) *IntakeHandler {
	return &IntakeHandler{
		dualWriter:            dualWriter,
		energyPlantRepository: energyPlantRepository,
		plantStatusRepository: plantStatusRepository,
		telegramNotifier:      telegramNotifier,
		useAsyncWrite:         useAsyncWrite,
	}
}

// deterministicID generates a UUID v5 from the SHA-256 hash of the payload.
// This guarantees idempotency: the same message always produces the same ID.
func deterministicID(payload []byte) uuid.UUID {
	hash := sha256.Sum256(payload)
	// Use the first 16 bytes of the SHA-256 hash as a UUID v5 namespace seed.
	return uuid.NewSHA1(uuid.NameSpaceDNS, hash[:])
}

func (h *IntakeHandler) HandleMessage(message []byte) error {
	startTime := time.Now()
	hash := sha256.Sum256(message)
	hashHex := hex.EncodeToString(hash[:])

	previewLen := 50
	if len(message) < previewLen {
		previewLen = len(message)
	}
	preview := string(message[:previewLen])
	if len(message) > previewLen {
		preview += "..."
	}

	slog.Info("received message on intake topic",
		"size", len(message),
		"sha256", hashHex,
		"preview", preview,
	)

	// Parse JSON
	var data map[string]interface{}
	if err := json.Unmarshal(message, &data); err != nil {
		slog.Error("invalid JSON message", "error", err)
		metrics.EventsValidationErrors.WithLabelValues("invalid_json").Inc()
		return domainerrors.NewPermanentError("invalid JSON", err)
	}

	// Extract safe fields for logging
	safeEventType := "unknown"
	if et, ok := data["event_type"].(string); ok {
		safeEventType = et
	}
	safePlantName := "not_specified"
	if pn, ok := data["plant_name"].(string); ok {
		safePlantName = pn
	}
	safePlantSourceId := "not_specified"
	if psid, ok := data["plant_source_id"].(string); ok {
		safePlantSourceId = psid
	}
	slog.Info("parsed safe fields",
		"event_type", safeEventType,
		"plant_name", safePlantName,
		"plant_source_id", safePlantSourceId,
	)

	eventType := safeEventType
	source := "kafka-intake"
	if plantName, ok := data["plant_name"].(string); ok {
		source = plantName
	}

	// Validate plant_source_id
	var plantSourceId uuid.UUID
	if plantSourceIdStr, ok := data["plant_source_id"].(string); ok {
		parsedUUID, err := uuid.Parse(plantSourceIdStr)
		if err != nil {
			slog.Error("invalid plant_source_id format", "error", err)
			h.telegramNotifier.SendUUIDError(
				"plant_source_id",
				plantSourceIdStr,
				fmt.Sprintf("Error al parsear UUID: %v. EventType: %s, PlantName: %s", err, safeEventType, safePlantName),
			)
			return domainerrors.NewPermanentError("invalid plant_source_id format", err)
		}
		plantSourceId = parsedUUID
	} else {
		slog.Error("plant_source_id not found in message")
		h.telegramNotifier.SendValidationError(
			"plant_source_id",
			"campo ausente",
			fmt.Sprintf("El campo plant_source_id no está presente en el mensaje. EventType: %s, PlantName: %s", safeEventType, safePlantName),
		)
		return domainerrors.NewPermanentError("missing plant_source_id field", nil)
	}

	// Validate plant exists
	exists, err := h.energyPlantRepository.Exists(plantSourceId)
	if err != nil {
		slog.Error("failed to validate plant existence",
			"plant_source_id", plantSourceId,
			"error", err,
		)
		return domainerrors.NewTransientError("database error checking plant existence", err)
	}
	if !exists {
		slog.Error("plant does not exist",
			"plant_source_id", plantSourceId,
			"event_type", eventType,
			"source", source,
		)
		h.telegramNotifier.SendValidationError(
			"plant_source_id",
			plantSourceId.String(),
			fmt.Sprintf("La planta con ID %s no existe en la base de datos. EventType: %s, Source: %s", plantSourceId, eventType, source),
		)
		return domainerrors.NewTransientError(
			fmt.Sprintf("plant_source_id=%s does not exist", plantSourceId), nil,
		)
	}

	slog.Info("plant validated", "plant_source_id", plantSourceId)

	// Validate event data fields
	var eventData entities.EventData
	if err := json.Unmarshal(message, &eventData); err == nil {
		if err := eventData.Validate(); err != nil {
			slog.Error("event data validation failed",
				"plant_source_id", plantSourceId,
				"event_type", eventType,
				"error", err,
			)
			h.telegramNotifier.SendValidationError(
				"event_data",
				"validation failed",
				fmt.Sprintf("Validation error for plant %s: %v", plantSourceId, err),
			)
			return domainerrors.NewPermanentError("event data validation failed", err)
		}
	}

	// Use raw JSON bytes directly (no re-marshal to string)
	dataJSON := json.RawMessage(message)

	// Idempotent ID: use "id" from message if present, otherwise generate deterministic UUID
	now := time.Now()
	var eventId uuid.UUID
	if idStr, ok := data["id"].(string); ok {
		parsed, err := uuid.Parse(idStr)
		if err != nil {
			eventId = deterministicID(message)
		} else {
			eventId = parsed
		}
	} else {
		eventId = deterministicID(message)
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

	if h.useAsyncWrite {
		if err := h.dualWriter.SaveEventAsync(eventOp, eventAn); err != nil {
			slog.Error("error saving event async", "error", err)
			return domainerrors.NewTransientError("async save failed", err)
		}
		slog.Info("event enqueued for async save", "id", eventId, "type", eventType)
	} else {
		if err := h.dualWriter.SaveEvent(eventOp, eventAn); err != nil {
			slog.Error("error saving event to database", "error", err)
			return domainerrors.NewTransientError("sync save failed", err)
		}
		slog.Info("event saved to both tables", "id", eventId, "type", eventType)
	}

	// Update Digital Twin: plant current status
	if h.plantStatusRepository != nil {
		// Extract current_status from payload if present, default to "ACTIVE"
		currentStatus := "ACTIVE"
		if status, ok := data["current_status"].(string); ok && status != "" {
			currentStatus = status
		}

		plantStatus := &entities.PlantCurrentStatus{
			PlantID:       plantSourceId,
			LastEventData: dataJSON,
			CurrentStatus: currentStatus,
			LastEventType: eventType,
			LastEventAt:   now,
			UpdatedAt:     now,
		}

		if err := h.plantStatusRepository.Upsert(plantStatus); err != nil {
			slog.Error("failed to update plant current status",
				"plant_source_id", plantSourceId,
				"error", err,
			)
			// Non-blocking: log error but don't fail the message processing
		} else {
			slog.Debug("plant current status updated",
				"plant_source_id", plantSourceId,
				"current_status", currentStatus,
			)
			// Track plant status update metric
			metrics.PlantStatusUpdates.WithLabelValues(plantSourceId.String(), currentStatus).Inc()
		}
	}

	// Track successful event ingestion metrics
	metrics.EventsIngestedTotal.WithLabelValues(eventType, plantSourceId.String(), "success").Inc()
	metrics.EventProcessingDuration.WithLabelValues(eventType).Observe(time.Since(startTime).Seconds())

	return nil
}
