package domain

import (
	"github.com/google/uuid"
	"net/url"
	"time"
)

type Order struct {
	ID       uuid.UUID
	OwnerID  string
	Price    Money
	Items    []OrderItem
	Url      *url.URL
	Status   OrderStatus
	Tags     []string
	Payload  []byte
	PayloadB []byte

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

type OrderItem struct {
	ProductID uuid.UUID
	Price     Money

	CreatedAt time.Time
	DeletedAt *time.Time
}
