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

		// CAMBIO: Agregado grupo de endpoints para eventos
		// RAZÓN: Permite consultar eventos guardados desde Kafka via HTTP REST
		events := api.Group("/events")
		{
			events.GET("", ListEvents(c))                  // GET /api/v1/events - Lista todos
			events.GET("/:id", GetEvent(c))                // GET /api/v1/events/:id - Obtiene uno por ID
			events.GET("/type/:type", GetEventsByType(c)) // GET /api/v1/events/type/:type - Filtra por tipo
		}

		// CAMBIO: Agregado grupo de endpoints para pruebas
		// RAZÓN: Permite probar validaciones y notificaciones de Telegram via Swagger
		test := api.Group("/test")
		{
			test.POST("/intake", TestIntake(c)) // POST /api/v1/test/intake - Simula mensaje de Kafka
		}
	}
}
