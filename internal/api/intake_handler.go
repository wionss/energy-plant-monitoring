package api

import (
	"context"
	"encoding/json"
	"log/slog"

	domainerrors "monitoring-energy-service/internal/domain/errors"
	"monitoring-energy-service/internal/domain/ports/input"
	"monitoring-energy-service/internal/domain/services"
	"monitoring-energy-service/internal/infrastructure/adapters/metrics"
)

// IntakeHandler processes messages consumed from Kafka.
// IntakeHandler deserializes Kafka intake messages and delegates business logic processing.
// Uses EventIngestionService to maintain separation of concerns: this handler handles
// only deserialization and error classification (infrastructure), while the service
// manages event persistence, validation, alert evaluation, and status updates (domain).
type IntakeHandler struct {
	ingestionService *services.EventIngestionService
}

var _ input.MessageHandler = &IntakeHandler{}

func NewIntakeHandler(ingestionService *services.EventIngestionService) *IntakeHandler {
	return &IntakeHandler{
		ingestionService: ingestionService,
	}
}

// HandleMessage deserializes the message and processes it via the domain service
func (h *IntakeHandler) HandleMessage(ctx context.Context, message []byte) error {
	// Deserialize JSON
	var data map[string]interface{}
	if err := json.Unmarshal(message, &data); err != nil {
		slog.Error("invalid JSON message", "error", err)
		metrics.EventsValidationErrors.WithLabelValues("invalid_json").Inc()
		return domainerrors.NewPermanentError("invalid JSON", err)
	}

	// Delegate all business logic to the service
	return h.ingestionService.ProcessEvent(ctx, message, data)
}
