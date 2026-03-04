package rest

import (
	"log/slog"
	"net/http"

	"monitoring-energy-service/internal/domain/ports/output"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// PlantStatusHandlers agrupa todos los handlers relacionados con el estado de plantas
type PlantStatusHandlers struct {
	plantStatusRepo output.PlantStatusRepositoryInterface
}

// NewPlantStatusHandlers crea una nueva instancia de PlantStatusHandlers
func NewPlantStatusHandlers(plantStatusRepo output.PlantStatusRepositoryInterface) *PlantStatusHandlers {
	return &PlantStatusHandlers{
		plantStatusRepo: plantStatusRepo,
	}
}

// ListPlantStatus returns the current status of all plants (Digital Twin)
// @Summary List current status of all plants
// @Description Returns the real-time status of all energy plants
// @Tags plants
// @Produce json
// @Success 200 {array} entities.PlantCurrentStatus
// @Failure 500 {object} map[string]string
// @Router /api/v1/plants/status [get]
func (h *PlantStatusHandlers) ListPlantStatus() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		statuses, err := h.plantStatusRepo.GetAll(ctx.Request.Context())
		if err != nil {
			slog.Error("handler error", "error", err, "path", ctx.FullPath())
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
		ctx.JSON(http.StatusOK, statuses)
	}
}

// GetPlantStatus returns the current status of a specific plant
// @Summary Get current status of a plant
// @Description Returns the real-time status of a specific energy plant
// @Tags plants
// @Produce json
// @Param id path string true "Plant ID (UUID)"
// @Success 200 {object} entities.PlantCurrentStatus
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/plants/status/{id} [get]
func (h *PlantStatusHandlers) GetPlantStatus() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		idStr := ctx.Param("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid plant ID format"})
			return
		}

		status, err := h.plantStatusRepo.GetByPlantID(ctx.Request.Context(), id)
		if err != nil {
			slog.Error("handler error", "error", err, "path", ctx.FullPath())
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		if status == nil {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "plant status not found"})
			return
		}

		ctx.JSON(http.StatusOK, status)
	}
}

// ListPlantStatusByStatus returns plants filtered by their current status
// @Summary List plants by status
// @Description Returns all plants with the specified status
// @Tags plants
// @Produce json
// @Param status path string true "Status filter (e.g., ACTIVE, INACTIVE, UNKNOWN)"
// @Success 200 {array} entities.PlantCurrentStatus
// @Failure 500 {object} map[string]string
// @Router /api/v1/plants/status/filter/{status} [get]
func (h *PlantStatusHandlers) ListPlantStatusByStatus() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		statusFilter := ctx.Param("status")

		statuses, err := h.plantStatusRepo.GetByStatus(ctx.Request.Context(), statusFilter)
		if err != nil {
			slog.Error("handler error", "error", err, "path", ctx.FullPath())
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		ctx.JSON(http.StatusOK, statuses)
	}
}
