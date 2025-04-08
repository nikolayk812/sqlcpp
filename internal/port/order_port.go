package port

import (
	"context"
	"github.com/google/uuid"
	"github.com/nikolayk812/sqlcpp/internal/domain"
)

type OrderRepository interface {
	GetOrder(ctx context.Context, orderID uuid.UUID) (domain.Order, error)
	GetOrderJoin(ctx context.Context, orderID uuid.UUID) (domain.Order, error)

	SearchOrders(ctx context.Context, filter domain.OrderFilter) ([]domain.Order, error)

	InsertOrder(ctx context.Context, order domain.Order) (uuid.UUID, error)

	SoftDeleteOrder(ctx context.Context, orderID uuid.UUID) error
	SoftDeleteOrderItem(ctx context.Context, orderID, productID uuid.UUID) error

	DeleteOrder(ctx context.Context, orderID uuid.UUID) error
}
