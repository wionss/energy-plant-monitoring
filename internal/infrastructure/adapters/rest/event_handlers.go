package rest

// event_handlers.go - Handlers REST para consultar eventos guardados en PostgreSQL
//
// PROPÓSITO:
// Expone endpoints HTTP para consultar los eventos que fueron enviados a Kafka
// y guardados en PostgreSQL por el IntakeHandler.
//
// CAMBIO: Refactorizado para inyectar dependencias en el constructor
// RAZÓN: Elimina el anti-patrón Service Locator (pasar el container completo)

import (
	"errors"
	"net/http"
	"time"

	domainerrors "monitoring-energy-service/internal/domain/errors"
	"monitoring-energy-service/internal/domain/ports/output"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// EventHandlers agrupa todos los handlers relacionados con eventos
type EventHandlers struct {
	eventRepo   output.EventRepositoryInterface
	eventOpRepo output.EventOperationalRepositoryInterface
	eventAnRepo output.EventAnalyticalRepositoryInterface
}

// NewEventHandlers crea una nueva instancia de EventHandlers
// Inyecta solo las dependencias necesarias (no el container completo)
func NewEventHandlers(
	eventRepo output.EventRepositoryInterface,
	eventOpRepo output.EventOperationalRepositoryInterface,
	eventAnRepo output.EventAnalyticalRepositoryInterface,
) *EventHandlers {
	return &EventHandlers{
		eventRepo:   eventRepo,
		eventOpRepo: eventOpRepo,
		eventAnRepo: eventAnRepo,
	}
}

// ListEvents obtiene todos los eventos de la base de datos
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
func (h *EventHandlers) ListEvents() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		events, err := h.eventRepo.FindAll()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusOK, events)
	}
}

// GetEvent obtiene un evento específico por su ID
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
func (h *EventHandlers) GetEvent() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		idStr := ctx.Param("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format"})
			return
		}

		event, err := h.eventRepo.FindByID(id)
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
func (h *EventHandlers) GetEventsByType() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		eventType := ctx.Param("type")

		events, err := h.eventRepo.FindByEventType(eventType)
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
func (h *EventHandlers) ListOperationalEvents() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		events, err := h.eventOpRepo.FindAll()
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
func (h *EventHandlers) GetOperationalEvent() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		idStr := ctx.Param("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format"})
			return
		}

		event, err := h.eventOpRepo.FindByID(id)
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
func (h *EventHandlers) ListAnalyticalEvents() gin.HandlerFunc {
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

		events, err := h.eventAnRepo.FindByTimeRange(start, end)
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
func (h *EventHandlers) GetHourlyAggregation() gin.HandlerFunc {
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

		results, err := h.eventAnRepo.GetHourlyAggregation(plantId, start, end)
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
func (h *EventHandlers) GetDailyAggregation() gin.HandlerFunc {
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

		results, err := h.eventAnRepo.GetDailyAggregation(plantId, start, end)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusOK, results)
	}
}
