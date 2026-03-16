package rest

// test_handlers.go - REST handlers for testing system functionality
//
// Package rest provides HTTP handlers for testing notifications and event queries.
// Dependencies are injected directly into handler constructors (not retrieved from
// a global Service Locator), enabling unit testing with mocked repositories and
// improving composability across different deployment scenarios.

import (
	"fmt"
	"net/http"

	"monitoring-energy-service/internal/domain/ports/output"
	"monitoring-energy-service/internal/infrastructure/adapters/telegram"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TestIntakeRequest represents the payload to test intake
type TestIntakeRequest struct {
	EventType     string `json:"event_type" example:"power_reading"`
	PlantName     string `json:"plant_name" example:"planta_solar_1"`
	PlantSourceID string `json:"plant_source_id" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// TestIntakeResponse represents the test endpoint response
type TestIntakeResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	TelegramSent bool   `json:"telegram_sent"`
	Error        string `json:"error,omitempty"`
}

// TestHandlers groups all test handlers
type TestHandlers struct {
	energyPlantRepo  output.EnergyPlantRepositoryInterface
	telegramNotifier *telegram.Notifier
}

// NewTestHandlers creates a new TestHandlers instance
func NewTestHandlers(
	energyPlantRepo output.EnergyPlantRepositoryInterface,
	telegramNotifier *telegram.Notifier,
) *TestHandlers {
	return &TestHandlers{
		energyPlantRepo:  energyPlantRepo,
		telegramNotifier: telegramNotifier,
	}
}

// TestIntake simulates processing a Kafka message to test validations
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

		// Validate that plant_source_id was provided
		if req.PlantSourceID == "" {
			// Notify missing field via Telegram
			telegramErr := h.telegramNotifier.SendValidationError(
				"plant_source_id",
				"missing field",
				fmt.Sprintf("The plant_source_id field is not present in the message. EventType: %s, PlantName: %s", req.EventType, req.PlantName),
			)

			ctx.JSON(http.StatusBadRequest, TestIntakeResponse{
				Success:      false,
				Message:      "Validation failed: missing plant_source_id",
				TelegramSent: telegramErr == nil,
				Error:        "missing plant_source_id field in message",
			})
			return
		}

		// Validate UUID format
		plantSourceID, err := uuid.Parse(req.PlantSourceID)
		if err != nil {
			// Notify invalid UUID error via Telegram
			telegramErr := h.telegramNotifier.SendUUIDError(
				"plant_source_id",
				req.PlantSourceID,
				fmt.Sprintf("Failed to parse UUID: %v. EventType: %s, PlantName: %s", err, req.EventType, req.PlantName),
			)

			ctx.JSON(http.StatusBadRequest, TestIntakeResponse{
				Success:      false,
				Message:      "Validation failed: invalid UUID format",
				TelegramSent: telegramErr == nil,
				Error:        fmt.Sprintf("invalid plant_source_id format: %v", err),
			})
			return
		}

		// Validate that the plant exists in the database
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
			// Notify non-existent plant via Telegram
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

		// all validations passed
		ctx.JSON(http.StatusOK, TestIntakeResponse{
			Success:      true,
			Message:      fmt.Sprintf("All validations passed for plant %s", plantSourceID),
			TelegramSent: false,
		})
	}
}
