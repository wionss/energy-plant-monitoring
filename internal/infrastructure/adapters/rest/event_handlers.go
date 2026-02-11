package rest

// event_handlers.go - Handlers REST para consultar eventos guardados en PostgreSQL
//
// PROPÓSITO:
// Expone endpoints HTTP para consultar los eventos que fueron enviados a Kafka
// y guardados en PostgreSQL por el IntakeHandler.
//
// CAMBIO REALIZADO: Archivo creado desde cero
// RAZÓN: Necesitábamos una API REST para consultar eventos desde cualquier cliente HTTP
//
// ENDPOINTS CREADOS:
// - GET /api/v1/events           - Lista todos los eventos (ordenados por fecha DESC)
// - GET /api/v1/events/:id       - Obtiene un evento específico por UUID
// - GET /api/v1/events/type/:type - Filtra eventos por tipo (power_reading, alert, etc.)

import (
	"errors"
	"net/http"
	"time"

	domainerrors "monitoring-energy-service/internal/domain/errors"
	"monitoring-energy-service/internal/infrastructure/container"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ListEvents obtiene todos los eventos de la base de datos
// CAMBIO: Handler nuevo
// RAZÓN: Permite consultar el histórico completo de eventos vía HTTP
//
// ListEvents godoc
// @Summary      List all events
// @Description  Get all events from the database ordered by creation time
// @Tags         events
// @Accept       json
// @Produce      json
// @Success      200  {array}   entities.EventEntity
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/events [get]
func ListEvents(c *container.Container) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		events, err := c.EventRepository.FindAll()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusOK, events)
	}
}

// GetEvent obtiene un evento específico por su ID
// CAMBIO: Handler nuevo
// RAZÓN: Permite recuperar un evento individual para análisis detallado
// CAMBIO: Ahora distingue entre errores 404 (not found) y 500 (internal)
// RAZÓN: Proporciona respuestas HTTP más precisas según el tipo de error
//
// GetEvent godoc
// @Summary      Get an event by ID
// @Description  Get a single event by its UUID
// @Tags         events
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Event ID (UUID)"
// @Success      200  {object}  entities.EventEntity
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /api/v1/events/{id} [get]
func GetEvent(c *container.Container) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		idStr := ctx.Param("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format"})
			return
		}

		event, err := c.EventRepository.FindByID(id)
		if err != nil {
			if errors.Is(err, domainerrors.ErrNotFound) {
				ctx.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
				return
			}
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		ctx.JSON(http.StatusOK, event)
	}
}

// GetEventsByType filtra eventos por tipo
// CAMBIO: Handler nuevo
// RAZÓN: Permite análisis de eventos específicos (ej: solo "alerts" o solo "power_reading")
//
// GetEventsByType godoc
// @Summary      Get events by type
// @Description  Get all events filtered by event type
// @Tags         events
// @Accept       json
// @Produce      json
// @Param        type   path      string  true  "Event Type"
// @Success      200  {array}   entities.EventEntity
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/events/type/{type} [get]
func GetEventsByType(c *container.Container) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		eventType := ctx.Param("type")

		events, err := c.EventRepository.FindByEventType(eventType)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusOK, events)
	}
}

// ============================================================================
// HANDLERS PARA DATOS OPERACIONALES (operational.events_std)
// ============================================================================

// ListOperationalEvents obtiene todos los eventos operacionales
//
// ListOperationalEvents godoc
// @Summary      List operational events
// @Description  Get all events from operational schema (hot data)
// @Tags         events-operational
// @Accept       json
// @Produce      json
// @Success      200  {array}   entities.EventOperational
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/events/operational [get]
func ListOperationalEvents(c *container.Container) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		events, err := c.EventOperationalRepo.FindAll()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusOK, events)
	}
}

// GetOperationalEvent obtiene un evento operacional por ID
//
// GetOperationalEvent godoc
// @Summary      Get operational event by ID
// @Description  Get a single event from operational schema by UUID
// @Tags         events-operational
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Event ID (UUID)"
// @Success      200  {object}  entities.EventOperational
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /api/v1/events/operational/{id} [get]
func GetOperationalEvent(c *container.Container) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		idStr := ctx.Param("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format"})
			return
		}

		event, err := c.EventOperationalRepo.FindByID(id)
		if err != nil {
			if errors.Is(err, domainerrors.ErrNotFound) {
				ctx.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
				return
			}
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		ctx.JSON(http.StatusOK, event)
	}
}

// ============================================================================
// HANDLERS PARA DATOS ANALÍTICOS (analytical.events_ts - TimescaleDB)
// ============================================================================

// ListAnalyticalEvents obtiene eventos analíticos por rango de tiempo
//
// ListAnalyticalEvents godoc
// @Summary      List analytical events by time range
// @Description  Get events from analytical schema (TimescaleDB) filtered by time range
// @Tags         events-analytical
// @Accept       json
// @Produce      json
// @Param        start   query     string  true  "Start time (RFC3339)"
// @Param        end     query     string  true  "End time (RFC3339)"
// @Success      200  {array}   entities.EventAnalytical
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/events/analytical [get]
func ListAnalyticalEvents(c *container.Container) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		startStr := ctx.Query("start")
		endStr := ctx.Query("end")

		if startStr == "" || endStr == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "start and end query parameters are required"})
			return
		}

		start, err := time.Parse(time.RFC3339, startStr)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid start time format, use RFC3339"})
			return
		}

		end, err := time.Parse(time.RFC3339, endStr)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid end time format, use RFC3339"})
			return
		}

		events, err := c.EventAnalyticalRepo.FindByTimeRange(start, end)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusOK, events)
	}
}

// GetHourlyAggregation obtiene agregaciones por hora usando TimescaleDB time_bucket
//
// GetHourlyAggregation godoc
// @Summary      Get hourly event aggregation
// @Description  Get hourly aggregated event counts using TimescaleDB time_bucket
// @Tags         events-analytical
// @Accept       json
// @Produce      json
// @Param        plant_id   query     string  true  "Plant ID (UUID)"
// @Param        start      query     string  true  "Start time (RFC3339)"
// @Param        end        query     string  true  "End time (RFC3339)"
// @Success      200  {array}   output.AggregatedEvent
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/events/analytical/hourly [get]
func GetHourlyAggregation(c *container.Container) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		plantIdStr := ctx.Query("plant_id")
		startStr := ctx.Query("start")
		endStr := ctx.Query("end")

		if plantIdStr == "" || startStr == "" || endStr == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "plant_id, start, and end query parameters are required"})
			return
		}

		plantId, err := uuid.Parse(plantIdStr)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid plant_id format"})
			return
		}

		start, err := time.Parse(time.RFC3339, startStr)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid start time format, use RFC3339"})
			return
		}

		end, err := time.Parse(time.RFC3339, endStr)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid end time format, use RFC3339"})
			return
		}

		results, err := c.EventAnalyticalRepo.GetHourlyAggregation(plantId, start, end)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusOK, results)
	}
}

// GetDailyAggregation obtiene agregaciones por día usando TimescaleDB time_bucket
//
// GetDailyAggregation godoc
// @Summary      Get daily event aggregation
// @Description  Get daily aggregated event counts using TimescaleDB time_bucket
// @Tags         events-analytical
// @Accept       json
// @Produce      json
// @Param        plant_id   query     string  true  "Plant ID (UUID)"
// @Param        start      query     string  true  "Start time (RFC3339)"
// @Param        end        query     string  true  "End time (RFC3339)"
// @Success      200  {array}   output.AggregatedEvent
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/events/analytical/daily [get]
func GetDailyAggregation(c *container.Container) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		plantIdStr := ctx.Query("plant_id")
		startStr := ctx.Query("start")
		endStr := ctx.Query("end")

		if plantIdStr == "" || startStr == "" || endStr == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "plant_id, start, and end query parameters are required"})
			return
		}

		plantId, err := uuid.Parse(plantIdStr)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid plant_id format"})
			return
		}

		start, err := time.Parse(time.RFC3339, startStr)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid start time format, use RFC3339"})
			return
		}

		end, err := time.Parse(time.RFC3339, endStr)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid end time format, use RFC3339"})
			return
		}

		results, err := c.EventAnalyticalRepo.GetDailyAggregation(plantId, start, end)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusOK, results)
	}
}
