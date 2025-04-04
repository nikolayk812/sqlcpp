package port

import (
	"context"
	"github.com/google/uuid"
	"github.com/nikolayk812/sqlcpp/internal/domain"
)

type OrderRepository interface {
	GetOrder(ctx context.Context, orderID uuid.UUID) (domain.Order, error)
	GetOrderJoin(ctx context.Context, orderID uuid.UUID) (domain.Order, error)

	InsertOrder(ctx context.Context, order domain.Order) (uuid.UUID, error)
}
