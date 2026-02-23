package rest

import (
	"monitoring-energy-service/internal/infrastructure/container"

	"github.com/gin-gonic/gin"
)

// SetupRoutes configura todas las rutas REST de la aplicación
//
// CAMBIO: Refactorizado para inyectar dependencias directamente en los handlers
// RAZÓN: Elimina el anti-patrón Service Locator (pasar el container completo)
func SetupRoutes(router *gin.RouterGroup, c *container.Container) {
	api := router.Group("/api/v1")
	{
		// Inicializar handlers con sus dependencias específicas
		exampleHandlers := NewExampleHandlers(c.ExampleRepository)
		eventHandlers := NewEventHandlers(c.EventRepository, c.EventOperationalRepo, c.EventAnalyticalRepo)
		plantStatusHandlers := NewPlantStatusHandlers(c.PlantStatusRepository)
		testHandlers := NewTestHandlers(c.EnergyPlantRepository, c.TelegramNotifier)

		// Examples endpoints
		examples := api.Group("/examples")
		{
			examples.GET("", exampleHandlers.ListExamples())
			examples.POST("", exampleHandlers.CreateExample())
			examples.GET("/:id", exampleHandlers.GetExample())
			examples.PUT("/:id", exampleHandlers.UpdateExample())
			examples.DELETE("/:id", exampleHandlers.DeleteExample())
		}

		// Events endpoints
		events := api.Group("/events")
		{
			events.GET("", eventHandlers.ListEvents())
			events.GET("/:id", eventHandlers.GetEvent())
			events.GET("/type/:type", eventHandlers.GetEventsByType())

			// Operational data (hot data)
			operational := events.Group("/operational")
			{
				operational.GET("", eventHandlers.ListOperationalEvents())
				operational.GET("/:id", eventHandlers.GetOperationalEvent())
			}

			// Analytical data (TimescaleDB)
			analytical := events.Group("/analytical")
			{
				analytical.GET("", eventHandlers.ListAnalyticalEvents())
				analytical.GET("/hourly", eventHandlers.GetHourlyAggregation())
				analytical.GET("/daily", eventHandlers.GetDailyAggregation())
			}
		}

		// Digital Twin: Real-time plant status endpoints
		plants := api.Group("/plants")
		{
			plants.GET("/status", plantStatusHandlers.ListPlantStatus())
			plants.GET("/status/:id", plantStatusHandlers.GetPlantStatus())
			plants.GET("/status/filter/:status", plantStatusHandlers.ListPlantStatusByStatus())
		}

		// Test endpoints
		test := api.Group("/test")
		{
			test.POST("/intake", testHandlers.TestIntake())
		}
	}
}
