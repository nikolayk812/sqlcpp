package domain

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/samber/lo"
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

func (f OrderFilter) Validate() error {
	// TODO: all fields are empty

	if f.CreatedAt != nil {
		if err := f.CreatedAt.Validate(); err != nil {
			return fmt.Errorf("createdAt: %w", err)
		}
	}

	if f.UpdatedAt != nil {
		if err := f.UpdatedAt.Validate(); err != nil {
			return fmt.Errorf("updatedAt: %w", err)
		}
	}

	return nil
}

type TimeRange struct {
	Before *time.Time
	After  *time.Time
}

func (t TimeRange) Validate() error {
	if t.Before == nil && t.After == nil {
		return errors.New("both Before and After are nil")
	}

	if lo.FromPtr(t.Before).IsZero() {
		return errors.New("before is zero")
	}

	if lo.FromPtr(t.After).IsZero() {
		return errors.New("after is zero")
	}

	if t.Before != nil && t.After != nil {
		if t.Before.After(*t.After) {
			return fmt.Errorf("before is after After")
		}
	}

	return nil
}
