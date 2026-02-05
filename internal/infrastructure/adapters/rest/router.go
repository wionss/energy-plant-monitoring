package rest

import (
	"monitoring-energy-service/internal/infrastructure/container"

	"github.com/gin-gonic/gin"
)

// SetupRoutes configura todas las rutas REST de la aplicación
//
// CAMBIOS REALIZADOS:
// - Agregado grupo /api/v1/events con 3 endpoints (líneas 21-26)
//
// RAZÓN:
// Necesitábamos exponer los eventos guardados en PostgreSQL mediante API REST
// para que puedan ser consultados desde el frontend o herramientas como Postman
func SetupRoutes(router *gin.RouterGroup, c *container.Container) {
	api := router.Group("/api/v1")
	{
		examples := api.Group("/examples")
		{
			examples.GET("", ListExamples(c))
			examples.POST("", CreateExample(c))
			examples.GET("/:id", GetExample(c))
			examples.PUT("/:id", UpdateExample(c))
			examples.DELETE("/:id", DeleteExample(c))
		}

		// CAMBIO: Agregado grupo de endpoints para eventos (legacy)
		// RAZÓN: Permite consultar eventos guardados desde Kafka via HTTP REST
		events := api.Group("/events")
		{
			events.GET("", ListEvents(c))                  // GET /api/v1/events - Lista todos
			events.GET("/:id", GetEvent(c))                // GET /api/v1/events/:id - Obtiene uno por ID
			events.GET("/type/:type", GetEventsByType(c)) // GET /api/v1/events/type/:type - Filtra por tipo

			// CAMBIO: Endpoints para datos operacionales (hot data)
			// RAZÓN: Consultas frecuentes sobre eventos recientes
			operational := events.Group("/operational")
			{
				operational.GET("", ListOperationalEvents(c))      // GET /api/v1/events/operational
				operational.GET("/:id", GetOperationalEvent(c))    // GET /api/v1/events/operational/:id
			}

			// CAMBIO: Endpoints para datos analíticos (TimescaleDB)
			// RAZÓN: Consultas por rango de tiempo y agregaciones
			analytical := events.Group("/analytical")
			{
				analytical.GET("", ListAnalyticalEvents(c))        // GET /api/v1/events/analytical?start=...&end=...
				analytical.GET("/hourly", GetHourlyAggregation(c)) // GET /api/v1/events/analytical/hourly?plant_id=...&start=...&end=...
				analytical.GET("/daily", GetDailyAggregation(c))   // GET /api/v1/events/analytical/daily?plant_id=...&start=...&end=...
			}
		}

		// Digital Twin: Real-time plant status endpoints
		plants := api.Group("/plants")
		{
			plants.GET("/status", ListPlantStatus(c))                   // GET /api/v1/plants/status - All plant statuses
			plants.GET("/status/:id", GetPlantStatus(c))                // GET /api/v1/plants/status/:id - Specific plant status
			plants.GET("/status/filter/:status", ListPlantStatusByStatus(c)) // GET /api/v1/plants/status/filter/:status - Filter by status
		}

		// CAMBIO: Agregado grupo de endpoints para pruebas
		// RAZÓN: Permite probar validaciones y notificaciones de Telegram via Swagger
		test := api.Group("/test")
		{
			test.POST("/intake", TestIntake(c)) // POST /api/v1/test/intake - Simula mensaje de Kafka
		}
	}
}
