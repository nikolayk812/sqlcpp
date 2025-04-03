package port

import (
	"context"
	"github.com/google/uuid"
	"github.com/nikolayk812/sqlcpp/internal/domain"
)

type CartRepository interface {
	GetCart(ctx context.Context, ownerID string) (domain.Cart, error)
	AddItem(ctx context.Context, ownerID string, item domain.CartItem) error
	DeleteItem(ctx context.Context, ownerID string, productID uuid.UUID) (bool, error)
}
