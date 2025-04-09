package domain

import (
	"errors"
	"fmt"
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
	// TODO: add DeletedAt
}

func (f OrderFilter) Validate() error {
	if len(f.IDs) == 0 && len(f.OwnerIDs) == 0 && len(f.UrlPatterns) == 0 && len(f.Statuses) == 0 && len(f.Tags) == 0 && f.CreatedAt == nil && f.UpdatedAt == nil {
		return errors.New("all fields are empty")
	}

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

	if t.Before != nil && t.After != nil {
		if t.Before.Before(*t.After) {
			return fmt.Errorf("before is before After")
		}
	}

	return nil
}
