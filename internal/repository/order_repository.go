package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/nikolayk812/sqlcpp/internal/db"
	"github.com/nikolayk812/sqlcpp/internal/domain"
	"github.com/nikolayk812/sqlcpp/internal/port"
	"github.com/samber/lo"
	"golang.org/x/text/currency"
	"net/url"
	"time"
)

var (
	ErrNotFound = errors.New("order not found")
)

type orderRepository struct {
	q    *db.Queries
	dbtx db.DBTX
}

// NewOrder creates a new OrderRepository with the given dbtx (pgx.Tx or pgxpool.Pool).
func NewOrder(dbtx db.DBTX) (port.OrderRepository, error) {
	if dbtx == nil {
		return nil, fmt.Errorf("dbtx is nil")
	}
	return &orderRepository{
		q:    db.New(dbtx),
		dbtx: dbtx,
	}, nil
}

func (r *orderRepository) GetOrder(ctx context.Context, orderID uuid.UUID) (domain.Order, error) {
	var o domain.Order

	order, err := withTx(ctx, r.dbtx, func(q *db.Queries) (domain.Order, error) {
		dbOrder, err := q.GetOrder(ctx, orderID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return o, fmt.Errorf("q.GetOrder: %w", ErrNotFound)
			}
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
		return o, fmt.Errorf("withTx: %w", err)
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
		return o, ErrNotFound
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

	orderID, err := withTx(ctx, r.dbtx, func(q *db.Queries) (uuid.UUID, error) {
		// Insert the order and get the generated order ID
		orderID, err := q.InsertOrder(ctx, db.InsertOrderParams{
			OwnerID:       order.OwnerID,
			Url:           lo.ToPtr(urlToString(order.Url)),
			Tags:          order.Tags,
			Payload:       emptyJSONIfNil(order.Payload),
			Payloadb:      order.PayloadB,
			PriceAmount:   order.Price.Amount,
			PriceCurrency: order.Price.Currency.String(),
		})
		if err != nil {
			return uuid.Nil, fmt.Errorf("q.InsertOrder: %w", err)
		}

		tx, ok := q.DB().(pgx.Tx)
		if !ok {
			return uuid.Nil, fmt.Errorf("q.DB() is not pgx.Tx")
		}

		if err := r.insertOrderItems(ctx, tx, orderID, order.Items); err != nil {
			return uuid.Nil, fmt.Errorf("r.insertOrderItems: %w", err)
		}

		return orderID, nil
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("withTx: %w", err)
	}

	return orderID, nil
}

func (r *orderRepository) insertOrderItems(ctx context.Context, tx pgx.Tx, orderID uuid.UUID, items []domain.OrderItem) (txErr error) {
	if tx == nil {
		return fmt.Errorf("tx is nil")
	}

	batch := &pgx.Batch{}

	for _, item := range items {
		batch.Queue(db.InsertOrderItem,
			orderID,
			item.ProductID,
			item.Price.Amount,
			item.Price.Currency.String(),
		)
	}

	results := tx.SendBatch(ctx, batch)
	defer func() {
		if err := results.Close(); err != nil {
			txErr = errors.Join(txErr, fmt.Errorf("results.Close: %w", err))
		}
	}()

	for i := 0; i < batch.Len(); i++ {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("batch item[%d]: %w", i, err)
		}
	}

	return nil
}

func (r *orderRepository) UpdateOrderStatus(ctx context.Context, orderID uuid.UUID, status domain.OrderStatus) error {
	if orderID == uuid.Nil {
		return fmt.Errorf("orderID is empty")
	}

	if status == "" {
		return fmt.Errorf("status is empty")
	}

	cmdTag, err := r.q.UpdateOrderStatus(ctx, db.UpdateOrderStatusParams{
		ID:     orderID,
		Status: string(status),
	})
	if err != nil {
		return fmt.Errorf("q.UpdateOrderStatus: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("q.UpdateOrderStatus: %w", ErrNotFound)
	}

	return nil
}

func (r *orderRepository) SearchOrders(ctx context.Context, filter domain.OrderFilter) ([]domain.Order, error) {
	if err := filter.Validate(); err != nil {
		return nil, fmt.Errorf("filter.Validate: %w", err)
	}

	dbFilter := mapDomainOrderFilterToSearchOrdersParams(filter)

	dbOrders, err := r.q.SearchOrders(ctx, dbFilter)
	if err != nil {
		return nil, fmt.Errorf("q.SearchOrders: %w", err)
	}

	// Use a map to group orders and their items
	orderMap := make(map[uuid.UUID]domain.Order)
	for _, row := range dbOrders {
		if _, exists := orderMap[row.ID]; !exists {
			order, err := mapSearchOrdersRowToDomainOrder(row)
			if err != nil {
				return nil, fmt.Errorf("mapSearchOrdersRowToDomainOrder: %w", err)
			}
			orderMap[row.ID] = order
		}

		item, err := mapSearchOrdersRowToDomainOrderItem(row)
		if err != nil {
			return nil, fmt.Errorf("mapSearchOrdersRowToDomainOrderItem: %w", err)
		}

		order := orderMap[row.ID]
		order.Items = append(order.Items, item)
		orderMap[row.ID] = order
	}

	return lo.Values(orderMap), nil
}

func (r *orderRepository) DeleteOrder(ctx context.Context, orderID uuid.UUID) error {
	if orderID == uuid.Nil {
		return fmt.Errorf("orderID is empty")
	}

	zero := struct{}{}
	_, err := withTx(ctx, r.dbtx, func(q *db.Queries) (struct{}, error) {
		cmdTag, err := q.DeleteOrderItems(ctx, orderID)
		if err != nil {
			return zero, fmt.Errorf("q.DeleteOrderItems: %w", err)
		}

		if cmdTag.RowsAffected() == 0 {
			return zero, fmt.Errorf("q.DeleteOrderItems: %w", ErrNotFound)
		}

		cmdTag, err = q.DeleteOrder(ctx, orderID)
		if err != nil {
			return zero, fmt.Errorf("q.DeleteOrder: %w", err)
		}

		if cmdTag.RowsAffected() == 0 {
			return zero, fmt.Errorf("q.DeleteOrder: %w", ErrNotFound)
		}

		return zero, nil
	})
	if err != nil {
		return fmt.Errorf("withTx: %w", err)
	}

	return nil
}

func (r *orderRepository) SoftDeleteOrder(ctx context.Context, orderID uuid.UUID) error {
	if orderID == uuid.Nil {
		return fmt.Errorf("orderID is empty")
	}

	cmdTag, err := r.q.SoftDeleteOrder(ctx, orderID)
	if err != nil {
		return fmt.Errorf("q.SoftDeleteOrder: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("q.SoftDeleteOrder: %w", ErrNotFound)
	}

	return nil
}

func (r *orderRepository) SoftDeleteOrderItem(ctx context.Context, orderID, productID uuid.UUID) error {
	if orderID == uuid.Nil {
		return fmt.Errorf("orderID is empty")
	}
	if productID == uuid.Nil {
		return fmt.Errorf("productID is empty")
	}

	zero := struct{}{}
	_, err := withTx(ctx, r.dbtx, func(q *db.Queries) (struct{}, error) {
		cmdTag, err := q.SoftDeleteOrderItem(ctx, db.SoftDeleteOrderItemParams{
			OrderID:   orderID,
			ProductID: productID,
		})
		if err != nil {
			return zero, fmt.Errorf("q.SoftDeleteOrderItem: %w", err)
		}

		if cmdTag.RowsAffected() == 0 {
			return zero, fmt.Errorf("q.SoftDeleteOrderItem: %w", ErrNotFound)
		}

		cmdTag, err = q.UpdateOrderPrice(ctx, orderID)
		if err != nil {
			return zero, fmt.Errorf("q.UpdateOrderPrice: %w", err)
		}

		if cmdTag.RowsAffected() == 0 {
			return zero, fmt.Errorf("q.UpdateOrderPrice: %w", ErrNotFound)
		}

		return zero, nil
	})
	if err != nil {
		return fmt.Errorf("withTx: %w", err)
	}

	return nil
}

func mapGetOrderItemsRowToDomain(row db.GetOrderItemsRow) (domain.OrderItem, error) {
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

func mapGetOrderItemsRowsToDomain(rows []db.GetOrderItemsRow) ([]domain.OrderItem, error) {
	var items []domain.OrderItem

	for _, row := range rows {
		item, err := mapGetOrderItemsRowToDomain(row)
		if err != nil {
			return nil, fmt.Errorf("mapGetOrderItemsRowsToDomain: %w", err)
		}

		items = append(items, item)
	}

	return items, nil
}

func mapDBOrderToDomain(dbOrder db.GetOrderRow, dbOrderItems []db.GetOrderItemsRow) (domain.Order, error) {
	var o domain.Order

	items, err := mapGetOrderItemsRowsToDomain(dbOrderItems)
	if err != nil {
		return o, fmt.Errorf("mapGetOrderItemsRowsToDomain: %w", err)
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

	parsedCurrency, err := currency.ParseISO(dbOrder.PriceCurrency)
	if err != nil {
		return o, fmt.Errorf("currency.ParseISO[%s]: %w", dbOrder.PriceCurrency, err)
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
		Payload:   dbOrder.Payload,
		PayloadB:  dbOrder.Payloadb,
		Price: domain.Money{
			Amount:   dbOrder.PriceAmount,
			Currency: parsedCurrency,
		},
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

	parsedCurrency, err := currency.ParseISO(row.PriceCurrency)
	if err != nil {
		return o, fmt.Errorf("currency.ParseISO[%s]: %w", row.PriceCurrency, err)
	}

	return domain.Order{
		ID:        row.ID,
		OwnerID:   row.OwnerID,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
		Status:    status,
		Url:       parsedURL,
		Tags:      row.Tags,
		Payload:   row.Payload,
		PayloadB:  row.Payloadb,
		Price: domain.Money{
			Amount:   row.PriceAmount,
			Currency: parsedCurrency,
		},
	}, nil
}

func mapGetOrderJoinItemsRowToDomainOrderItem(row db.GetOrderJoinItemsRow) (domain.OrderItem, error) {
	parsedCurrency, err := currency.ParseISO(row.ItemPriceCurrency)
	if err != nil {
		return domain.OrderItem{}, fmt.Errorf("item currency[%s] is not valid: %w", row.ItemPriceCurrency, err)
	}

	return domain.OrderItem{
		ProductID: row.ProductID,
		Price:     domain.Money{Amount: row.ItemPriceAmount, Currency: parsedCurrency},
		CreatedAt: row.CreatedAt,
	}, nil
}

func mapSearchOrdersRowToDomainOrder(row db.SearchOrdersRow) (domain.Order, error) {
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

	parsedCurrency, err := currency.ParseISO(row.PriceCurrency)
	if err != nil {
		return o, fmt.Errorf("currency.ParseISO[%s]: %w", row.PriceCurrency, err)
	}

	return domain.Order{
		ID:        row.ID,
		OwnerID:   row.OwnerID,
		Status:    status,
		Url:       parsedURL,
		Tags:      row.Tags,
		Payload:   row.Payload,
		PayloadB:  row.Payloadb,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
		Price: domain.Money{
			Amount:   row.PriceAmount,
			Currency: parsedCurrency,
		},
	}, nil
}

func mapSearchOrdersRowToDomainOrderItem(row db.SearchOrdersRow) (domain.OrderItem, error) {
	parsedCurrency, err := currency.ParseISO(row.ItemPriceCurrency)
	if err != nil {
		return domain.OrderItem{}, fmt.Errorf("item currency[%s] is not valid: %w", row.ItemPriceCurrency, err)
	}

	return domain.OrderItem{
		ProductID: row.ProductID,
		Price:     domain.Money{Amount: row.ItemPriceAmount, Currency: parsedCurrency},
	}, nil
}

func mapDomainOrderFilterToSearchOrdersParams(filter domain.OrderFilter) db.SearchOrdersParams {
	var statuses []string
	for _, status := range filter.Statuses {
		statuses = append(statuses, string(status))
	}

	var createdAfter, createdBefore, updatedAfter, updatedBefore *time.Time

	if filter.CreatedAt != nil {
		createdAfter = filter.CreatedAt.After
		createdBefore = filter.CreatedAt.Before
	}

	if filter.UpdatedAt != nil {
		updatedAfter = filter.UpdatedAt.After
		updatedBefore = filter.UpdatedAt.Before
	}

	return db.SearchOrdersParams{
		Ids:           nilSliceIfEmpty(filter.IDs),
		OwnerIds:      nilSliceIfEmpty(filter.OwnerIDs),
		UrlPatterns:   nilSliceIfEmpty(filter.UrlPatterns),
		Statuses:      nilSliceIfEmpty(statuses),
		Tags:          nilSliceIfEmpty(filter.Tags),
		CreatedAfter:  createdAfter,
		CreatedBefore: createdBefore,
		UpdatedAfter:  updatedAfter,
		UpdatedBefore: updatedBefore,
	}
}

func urlToString(u *url.URL) string {
	if u == nil {
		return ""
	}
	return u.String()
}

func emptyJSONIfNil(j []byte) []byte {
	if j == nil {
		return []byte(`{}`)
	}
	return j
}

func nilSliceIfEmpty[T any](s []T) []T {
	if len(s) == 0 {
		return nil
	}
	return s
}
