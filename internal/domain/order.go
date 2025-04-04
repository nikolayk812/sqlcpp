package domain

import (
	"github.com/google/uuid"
	"net/url"
	"time"
)

type Order struct {
	ID      uuid.UUID
	OwnerID string
	Items   []OrderItem
	Url     *url.URL
	Status  OrderStatus
	Tags    []string

	CreatedAt time.Time
	UpdatedAt time.Time
}

type OrderItem struct {
	ProductID uuid.UUID
	Price     Money

	CreatedAt time.Time
}
