package domain

import (
	"github.com/google/uuid"
	"time"
)

// OrderFilter has AND semantics across fields, OR semantics within each field slice
type OrderFilter struct {
	IDs         []uuid.UUID
	OwnerIDs    []string
	UrlPatterns []string
	Statuses    []OrderStatus
	Tags        []string
	CreatedAt   *TimeRange
	UpdatedAt   *TimeRange
}

type TimeRange struct {
	Before *time.Time
	After  *time.Time
}
