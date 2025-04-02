package domain

import (
	"github.com/google/uuid"
	"time"
)

type Order struct {
	ID        uuid.UUID
	OwnerID   string
	Items     []OrderItem
	CreatedAt time.Time
	UpdatedAt time.Time
}

type OrderItem struct {
	ProductID uuid.UUID
	Price     Money

	CreatedAt time.Time
}
