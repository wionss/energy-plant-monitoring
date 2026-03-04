package rest

import (
	"monitoring-energy-service/internal/infrastructure/tracing"

	"github.com/gin-gonic/gin"
)

const requestIDHeader = "X-Request-ID"

// RequestIDMiddleware reads X-Request-ID from the incoming request (or generates a new UUID),
// stores it in the request context via tracing, and echoes it back in the response header.
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(requestIDHeader)
		if id == "" {
			id = tracing.NewCorrelationID()
		}

		// Propagate into request context
		ctx := tracing.WithCorrelationID(c.Request.Context(), id)
		c.Request = c.Request.WithContext(ctx)

		// Echo back so clients can trace their request
		c.Header(requestIDHeader, id)

		c.Next()
	}
}
