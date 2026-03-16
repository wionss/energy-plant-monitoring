package tracing

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
)

type contextKey struct{}

// NewCorrelationID generates a new random correlation ID.
func NewCorrelationID() string {
	return uuid.New().String()
}

// WithCorrelationID returns a new context carrying the given correlation ID.
func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, contextKey{}, id)
}

// CorrelationIDFromCtx extracts the correlation ID from the context.
// Returns an empty string if none is set.
func CorrelationIDFromCtx(ctx context.Context) string {
	if id, ok := ctx.Value(contextKey{}).(string); ok {
		return id
	}
	return ""
}

// Logger returns a slog.Logger enriched with the correlation_id from ctx.
func Logger(ctx context.Context) *slog.Logger {
	id := CorrelationIDFromCtx(ctx)
	if id == "" {
		return slog.Default()
	}
	return slog.Default().With("correlation_id", id)
}
