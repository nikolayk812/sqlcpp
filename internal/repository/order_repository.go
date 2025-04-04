package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nikolayk812/sqlcpp/internal/db"
	"github.com/nikolayk812/sqlcpp/internal/domain"
	"github.com/nikolayk812/sqlcpp/internal/port"
	"github.com/samber/lo"
	"golang.org/x/text/currency"
	"net/url"
)

type orderRepository struct {
	q    *db.Queries
	pool *pgxpool.Pool
}

func NewOrder(pool *pgxpool.Pool) port.OrderRepository {
	return &orderRepository{
		q:    db.New(pool),
		pool: pool,
	}
}

func NewOrderWithTx(tx pgx.Tx) port.OrderRepository {
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

		domainOrder, err := mapDBOrderToDomain(dbOrder, dbOrderItems)
		if err != nil {
			return o, fmt.Errorf("mapDBOrderToDomain: %w", err)
		}

		return domainOrder, nil
	})
	if err != nil {
		return o, fmt.Errorf("r.withTxOrder: %w", err)
	}

	return order, nil
}

func (r *orderRepository) GetOrderJoin(ctx context.Context, orderID uuid.UUID) (domain.Order, error) {
	var o domain.Order

	dbOrderItemsRows, err := r.q.GetOrderJoinItems(ctx, orderID)
	if err != nil {
		return o, fmt.Errorf("q.GetOrderJoinItems: %w", err)
	}

	if len(dbOrderItemsRows) == 0 {
		return o, fmt.Errorf("order not found")
	}

	// Map the first row to domain.Order
	order, err := mapGetOrderJoinItemsRowToDomainOrder(dbOrderItemsRows[0])
	if err != nil {
		return o, fmt.Errorf("mapGetOrderJoinItemsRowToDomainOrder: %w", err)
	}

	// Iterate over the rows and map to domain.OrderItem
	for _, row := range dbOrderItemsRows {
		item, err := mapGetOrderJoinItemsRowToDomainOrderItem(row)
		if err != nil {
			return o, fmt.Errorf("mapGetOrderJoinItemsRowToDomainOrderItem: %w", err)
		}
		order.Items = append(order.Items, item)
	}

	return order, nil
}

func (r *orderRepository) InsertOrder(ctx context.Context, order domain.Order) (uuid.UUID, error) {
	if len(order.Items) == 0 {
		return uuid.Nil, errors.New("no items in order")
	}

	var orderID uuid.UUID

	orderID, err := r.withTxUUID(ctx, func(q *db.Queries) (uuid.UUID, error) {
		// Insert the order and get the generated order ID
		orderID, err := q.InsertOrder(ctx, db.InsertOrderParams{
			OwnerID: order.OwnerID,
			Url:     lo.ToPtr(urlToString(order.Url)),
			Tags:    order.Tags,
		})
		if err != nil {
			return uuid.Nil, fmt.Errorf("q.InsertOrder: %w", err)
		}

		// TODO: join or batch
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

func (r *orderRepository) withTx(ctx context.Context, fn func(q *db.Queries) error) error {
	_, err := withTx(ctx, r.pool, r.q, func(q *db.Queries) (struct{}, error) {
		err := fn(q)
		return struct{}{}, err
	})
	return err
}

func (r *orderRepository) withTxOrder(ctx context.Context, fn func(q *db.Queries) (domain.Order, error)) (domain.Order, error) {
	return withTx(ctx, r.pool, r.q, fn)
}

func (r *orderRepository) withTxUUID(ctx context.Context, fn func(q *db.Queries) (uuid.UUID, error)) (uuid.UUID, error) {
	return withTx(ctx, r.pool, r.q, fn)
}

func mapGetOrderRowToDomain(row db.GetOrderItemsRow) (domain.OrderItem, error) {
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

func mapGetOrderRowsToDomain(rows []db.GetOrderItemsRow) ([]domain.OrderItem, error) {
	var items []domain.OrderItem

	for _, row := range rows {
		item, err := mapGetOrderRowToDomain(row)
		if err != nil {
			return nil, fmt.Errorf("mapGetOrderRowToDomain: %w", err)
		}

		items = append(items, item)
	}

	return items, nil
}

func mapDBOrderToDomain(dbOrder db.GetOrderRow, dbOrderItems []db.GetOrderItemsRow) (domain.Order, error) {
	var o domain.Order

	items, err := mapGetOrderRowsToDomain(dbOrderItems)
	if err != nil {
		return o, fmt.Errorf("mapGetOrderRowsToDomain: %w", err)
	}

	var parsedURL *url.URL

	if lo.FromPtr(dbOrder.Url) != "" {
		parsedURL, err = url.Parse(*dbOrder.Url)
		if err != nil {
			return o, fmt.Errorf("url.Parse[%s]: %w", *dbOrder.Url, err)
		}
	}

	status, err := domain.ToOrderStatus(dbOrder.Status)
	if err != nil {
		return o, fmt.Errorf("domain.ToOrderStatus[%s]: %w", dbOrder.Status, err)
	}

	return domain.Order{
		ID:        dbOrder.ID,
		OwnerID:   dbOrder.OwnerID,
		Items:     items,
		CreatedAt: dbOrder.CreatedAt,
		UpdatedAt: dbOrder.UpdatedAt,
		Status:    status,
		Url:       parsedURL,
		Tags:      dbOrder.Tags,
	}, nil
}

func mapGetOrderJoinItemsRowToDomainOrder(row db.GetOrderJoinItemsRow) (domain.Order, error) {
	var (
		o         domain.Order
		parsedURL *url.URL
		err       error
	)

	if lo.FromPtr(row.Url) != "" {
		parsedURL, err = url.Parse(*row.Url)
		if err != nil {
			return o, fmt.Errorf("url.Parse[%s]: %w", *row.Url, err)
		}
	}

	status, err := domain.ToOrderStatus(row.Status)
	if err != nil {
		return o, fmt.Errorf("domain.ToOrderStatus[%s]: %w", row.Status, err)
	}

	return domain.Order{
		ID:        row.ID,
		OwnerID:   row.OwnerID,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
		Status:    status,
		Url:       parsedURL,
		Tags:      row.Tags,
	}, nil
}

func mapGetOrderJoinItemsRowToDomainOrderItem(row db.GetOrderJoinItemsRow) (domain.OrderItem, error) {
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

func urlToString(u *url.URL) string {
	if u == nil {
		return ""
	}
	return u.String()
}
