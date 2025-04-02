package domain

import (
	"github.com/google/uuid"
	"time"
)

type Order struct {
	ID      uuid.UUID
	OwnerID string
	Items   []OrderItem
}

type OrderItem struct {
	ProductID uuid.UUID
	Price     Money

	CreatedAt time.Time
}
