package domain

import (
	"github.com/google/uuid"
	"time"
)

type Cart struct {
	OwnerID string
	Items   []CartItem
}

type CartItem struct {
	ProductID uuid.UUID
	Price     Money

	CreatedAt time.Time
}
