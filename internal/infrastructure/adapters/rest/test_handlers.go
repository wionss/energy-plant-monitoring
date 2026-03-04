package rest

// test_handlers.go - Handlers REST para probar funcionalidades del sistema
//
// PROPÓSITO:
// Expone endpoints HTTP para probar el envío de notificaciones a Telegram
// simulando mensajes como los que llegan desde Kafka.
//
// CAMBIO: Refactorizado para inyectar dependencias
// RAZÓN: Elimina el anti-patrón Service Locator

import (
	"fmt"
	"net/http"

	"monitoring-energy-service/internal/domain/ports/output"
	"monitoring-energy-service/internal/infrastructure/adapters/telegram"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TestIntakeRequest representa el payload para probar el intake
type TestIntakeRequest struct {
	EventType     string `json:"event_type" example:"power_reading"`
	PlantName     string `json:"plant_name" example:"planta_solar_1"`
	PlantSourceID string `json:"plant_source_id" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// TestIntakeResponse representa la respuesta del endpoint de prueba
type TestIntakeResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	TelegramSent bool   `json:"telegram_sent"`
	Error        string `json:"error,omitempty"`
}

// TestHandlers agrupa todos los handlers de prueba
type TestHandlers struct {
	energyPlantRepo  output.EnergyPlantRepositoryInterface
	telegramNotifier *telegram.Notifier
}

// NewTestHandlers crea una nueva instancia de TestHandlers
func NewTestHandlers(
	energyPlantRepo output.EnergyPlantRepositoryInterface,
	telegramNotifier *telegram.Notifier,
) *TestHandlers {
	return &TestHandlers{
		energyPlantRepo:  energyPlantRepo,
		telegramNotifier: telegramNotifier,
	}
}

// TestIntake simula el procesamiento de un mensaje de Kafka para probar validaciones
//
// TestIntake godoc
// @Summary      Test intake message processing
// @Description  Simulates a Kafka message to test validation and Telegram notifications
// @Tags         test
// @Accept       json
// @Produce      json
// @Param        request  body      TestIntakeRequest  true  "Test intake data"
// @Success      200  {object}  TestIntakeResponse
// @Failure      400  {object}  TestIntakeResponse
// @Failure      404  {object}  TestIntakeResponse
// @Router       /api/v1/test/intake [post]
func (h *TestHandlers) TestIntake() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req TestIntakeRequest
		if err := ctx.ShouldBindJSON(&req); err != nil {
			ctx.JSON(http.StatusBadRequest, TestIntakeResponse{
				Success: false,
				Message: "Invalid JSON payload",
				Error:   err.Error(),
			})
			return
		}

		// Validar que se proporcionó plant_source_id
		if req.PlantSourceID == "" {
			// Notificar campo faltante a Telegram
			telegramErr := h.telegramNotifier.SendValidationError(
				"plant_source_id",
				"campo ausente",
				fmt.Sprintf("El campo plant_source_id no está presente en el mensaje. EventType: %s, PlantName: %s", req.EventType, req.PlantName),
			)

			ctx.JSON(http.StatusBadRequest, TestIntakeResponse{
				Success:      false,
				Message:      "Validation failed: missing plant_source_id",
				TelegramSent: telegramErr == nil,
				Error:        "missing plant_source_id field in message",
			})
			return
		}

		// Validar formato UUID
		plantSourceID, err := uuid.Parse(req.PlantSourceID)
		if err != nil {
			// Notificar error de UUID inválido a Telegram
			telegramErr := h.telegramNotifier.SendUUIDError(
				"plant_source_id",
				req.PlantSourceID,
				fmt.Sprintf("Error al parsear UUID: %v. EventType: %s, PlantName: %s", err, req.EventType, req.PlantName),
			)

			ctx.JSON(http.StatusBadRequest, TestIntakeResponse{
				Success:      false,
				Message:      "Validation failed: invalid UUID format",
				TelegramSent: telegramErr == nil,
				Error:        fmt.Sprintf("invalid plant_source_id format: %v", err),
			})
			return
		}

		// Validar que la planta existe en la base de datos
		exists, err := h.energyPlantRepo.Exists(ctx.Request.Context(), plantSourceID)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, TestIntakeResponse{
				Success: false,
				Message: "Database error while validating plant",
				Error:   err.Error(),
			})
			return
		}

		if !exists {
			// Notificar planta inexistente a Telegram
			telegramErr := h.telegramNotifier.SendValidationError(
				"plant_source_id",
				plantSourceID.String(),
				fmt.Sprintf("La planta con ID %s no existe en la base de datos. EventType: %s, PlantName: %s", plantSourceID, req.EventType, req.PlantName),
			)

			ctx.JSON(http.StatusNotFound, TestIntakeResponse{
				Success:      false,
				Message:      "Validation failed: plant does not exist",
				TelegramSent: telegramErr == nil,
				Error:        fmt.Sprintf("plant_source_id=%s does not exist in database", plantSourceID),
			})
			return
		}

		// Todas las validaciones pasaron
		ctx.JSON(http.StatusOK, TestIntakeResponse{
			Success:      true,
			Message:      fmt.Sprintf("All validations passed for plant %s", plantSourceID),
			TelegramSent: false,
		})
	}
}
