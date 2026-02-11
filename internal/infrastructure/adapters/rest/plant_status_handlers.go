package rest

import (
	"net/http"

	"monitoring-energy-service/internal/infrastructure/container"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ListPlantStatus returns the current status of all plants (Digital Twin)
// @Summary List current status of all plants
// @Description Returns the real-time status of all energy plants
// @Tags plants
// @Produce json
// @Success 200 {array} entities.PlantCurrentStatus
// @Failure 500 {object} map[string]string
// @Router /api/v1/plants/status [get]
func ListPlantStatus(c *container.Container) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		statuses, err := c.PlantStatusRepository.GetAll()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
func GetPlantStatus(c *container.Container) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		idStr := ctx.Param("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid plant ID format"})
			return
		}

		status, err := c.PlantStatusRepository.GetByPlantID(id)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
func ListPlantStatusByStatus(c *container.Container) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		statusFilter := ctx.Param("status")

		statuses, err := c.PlantStatusRepository.GetByStatus(statusFilter)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(http.StatusOK, statuses)
	}
}
