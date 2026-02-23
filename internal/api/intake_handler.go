package api

import (
	"encoding/json"
	"log/slog"

	domainerrors "monitoring-energy-service/internal/domain/errors"
	"monitoring-energy-service/internal/domain/ports/input"
	"monitoring-energy-service/internal/domain/services"
	"monitoring-energy-service/internal/infrastructure/adapters/metrics"
)

// IntakeHandler procesa mensajes consumidos desde Kafka
// CAMBIO: Refactorizado para usar EventIngestionService
// RAZÓN: Separación clara de responsabilidades:
//   - IntakeHandler: Solo deserialización (infraestructura)
//   - EventIngestionService: Lógica de negocio completa
type IntakeHandler struct {
	ingestionService *services.EventIngestionService
}

var _ input.MessageHandler = &IntakeHandler{}

func NewIntakeHandler(ingestionService *services.EventIngestionService) *IntakeHandler {
	return &IntakeHandler{
		ingestionService: ingestionService,
	}
}

// HandleMessage deserializa el evento y lo procesa a través del servicio de dominio
func (h *IntakeHandler) HandleMessage(message []byte) error {
	// Deserializar el JSON
	var data map[string]interface{}
	if err := json.Unmarshal(message, &data); err != nil {
		slog.Error("invalid JSON message", "error", err)
		metrics.EventsValidationErrors.WithLabelValues("invalid_json").Inc()
		return domainerrors.NewPermanentError("invalid JSON", err)
	}

	// Delegar toda la lógica de negocio al servicio
	return h.ingestionService.ProcessEvent(message, data)
}
