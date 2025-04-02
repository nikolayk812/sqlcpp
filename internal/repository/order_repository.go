package repository

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nikolayk812/sqlcpp/internal/db"
	"github.com/nikolayk812/sqlcpp/internal/domain"
	"golang.org/x/text/currency"
)

type OrderRepository interface {
	GetOrder(ctx context.Context, orderID uuid.UUID) (domain.Order, error)
	InsertOrder(ctx context.Context, order domain.Order) (uuid.UUID, error)
}

type orderRepository struct {
	q    *db.Queries
	pool *pgxpool.Pool
}

func NewOrderRepository(pool *pgxpool.Pool) OrderRepository {
	return &orderRepository{
		q:    db.New(pool),
		pool: pool,
	}
}

func NewOrderRepositoryWithTx(tx pgx.Tx) OrderRepository {
	return &orderRepository{
		q:    db.New(tx),
		pool: nil, // use provided transaction instead
	}
}

func (r *orderRepository) GetOrder(ctx context.Context, orderID uuid.UUID) (domain.Order, error) {
	var o domain.Order

	order, err := r.withTxOrder(ctx, func(q *db.Queries) (domain.Order, error) {
		dbOrder, err := q.GetOrder(ctx, orderID)
		if err != nil {
			return o, fmt.Errorf("q.GetOrder: %w", err)
		}

		dbOrderItems, err := q.GetOrderItems(ctx, orderID)
		if err != nil {
			return o, fmt.Errorf("q.GetOrderItems: %w", err)
		}

		domainOrder, err := mapDBOrderToDomainOrder(dbOrder, dbOrderItems)
		if err != nil {
			return o, fmt.Errorf("mapDBOrderToDomainOrder: %w", err)
		}

		return domainOrder, nil
	})
	if err != nil {
		return o, fmt.Errorf("r.withTxOrder: %w", err)
	}

	return order, nil
}

func (r *orderRepository) InsertOrder(ctx context.Context, order domain.Order) (uuid.UUID, error) {
	var orderID uuid.UUID

	orderID, err := r.withTxUUID(ctx, func(q *db.Queries) (uuid.UUID, error) {
		// Insert the order and get the generated order ID
		orderID, err := q.InsertOrder(ctx, order.OwnerID)
		if err != nil {
			return uuid.Nil, fmt.Errorf("q.InsertOrder: %w", err)
		}

		// Insert each order item
		for _, item := range order.Items {
			arg := db.InsertOrderItemParams{
				OrderID:       orderID,
				ProductID:     item.ProductID,
				PriceAmount:   item.Price.Amount,
				PriceCurrency: item.Price.Currency.String(),
			}
			if err := q.InsertOrderItem(ctx, arg); err != nil {
				return uuid.Nil, fmt.Errorf("q.InsertOrderItem: %w", err)
			}
		}

		return orderID, nil
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("r.withTxOrder: %w", err)
	}

	return orderID, nil
}

func (r *orderRepository) withTxOrder(ctx context.Context, fn func(q *db.Queries) (domain.Order, error)) (domain.Order, error) {
	return withTx(ctx, r.pool, r.q, fn)
}

func (r *orderRepository) withTxUUID(ctx context.Context, fn func(q *db.Queries) (uuid.UUID, error)) (uuid.UUID, error) {
	return withTx(ctx, r.pool, r.q, fn)
}

func mapGetOrderRowToDomainOrderItem(row db.GetOrderItemsRow) (domain.OrderItem, error) {
	parsedCurrency, err := currency.ParseISO(row.PriceCurrency)
	if err != nil {
		return domain.OrderItem{}, fmt.Errorf("currency[%s] is not valid: %w", row.PriceCurrency, err)
	}

	return domain.OrderItem{
		ProductID: row.ProductID,
		Price:     domain.Money{Amount: row.PriceAmount, Currency: parsedCurrency},
		CreatedAt: row.CreatedAt,
	}, nil
}

func mapGetOrderRowsToDomainOrderItems(rows []db.GetOrderItemsRow) ([]domain.OrderItem, error) {
	var items []domain.OrderItem
	for _, row := range rows {
		item, err := mapGetOrderRowToDomainOrderItem(row)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func mapDBOrderToDomainOrder(dbOrder db.Order, dbOrderItems []db.GetOrderItemsRow) (domain.Order, error) {
	items, err := mapGetOrderRowsToDomainOrderItems(dbOrderItems)
	if err != nil {
		return domain.Order{}, fmt.Errorf("mapGetOrderRowsToDomainOrderItems: %w", err)
	}

	return domain.Order{
		ID:        dbOrder.ID,
		OwnerID:   dbOrder.OwnerID,
		Items:     items,
		CreatedAt: dbOrder.CreatedAt,
		UpdatedAt: dbOrder.UpdatedAt,
	}, nil
}
