package rest

import (
	"monitoring-energy-service/internal/infrastructure/container"

	"github.com/gin-gonic/gin"
)

// SetupRoutes configura todas las rutas REST de la aplicación
//
// SetupRoutes initializes all REST routes with explicit dependency injection.
// Each handler receives only its required dependencies, avoiding tight coupling to
// a global container and enabling independent testing and easier refactoring of
// individual endpoints.
func SetupRoutes(router *gin.RouterGroup, c *container.Container) {
	rateLimiter := NewRateLimiter()

	api := router.Group("/api/v1")
	api.Use(RequestIDMiddleware(), rateLimiter.Middleware())
	{
		// Inialize handlers with explicit dependencies (not from global container)
		exampleHandlers := NewExampleHandlers(c.ExampleRepository)
		eventHandlers := NewEventHandlers(c.EventOperationalRepo, c.EventAnalyticalRepo)
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
