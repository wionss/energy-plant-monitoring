package output

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	DefaultPageSize = 50
	MaxPageSize     = 200
)

// PageCursor holds the position of the last seen item for cursor-based pagination.
type PageCursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

// PageQuery describes a paginated request.
type PageQuery struct {
	Limit  int
	Cursor *PageCursor
}

// Page is a generic paginated response.
type Page[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"next_cursor,omitempty"`
	Count      int    `json:"count"`
}

// EncodeCursor encodes a (time, uuid) pair as a base64 string.
func EncodeCursor(t time.Time, id uuid.UUID) string {
	raw := t.UTC().Format(time.RFC3339Nano) + "|" + id.String()
	return base64.StdEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor decodes a base64 cursor string back to a PageCursor.
func DecodeCursor(s string) (*PageCursor, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor encoding: %w", err)
	}
	parts := strings.SplitN(string(b), "|", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid cursor format")
	}
	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid cursor timestamp: %w", err)
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid cursor uuid: %w", err)
	}
	return &PageCursor{CreatedAt: t, ID: id}, nil
}

// NormalizePageQuery applies defaults and enforces max limit.
func NormalizePageQuery(q PageQuery) PageQuery {
	if q.Limit <= 0 {
		q.Limit = DefaultPageSize
	}
	if q.Limit > MaxPageSize {
		q.Limit = MaxPageSize
	}
	return q
}
