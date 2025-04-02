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

type CartRepository interface {
	GetCart(ctx context.Context, ownerID string) (domain.Cart, error)
	AddItem(ctx context.Context, ownerID string, item domain.CartItem) error
	DeleteItem(ctx context.Context, ownerID string, productID uuid.UUID) (bool, error)
}

type cartRepository struct {
	q    *db.Queries
	pool *pgxpool.Pool
}

func NewCartRepository(pool *pgxpool.Pool) CartRepository {
	return &cartRepository{
		q:    db.New(pool),
		pool: pool,
	}
}

func NewCartRepositoryWithTx(tx pgx.Tx) CartRepository {
	return &cartRepository{
		q:    db.New(tx),
		pool: nil, // use provided transaction instead
	}
}

func (r *cartRepository) GetCart(ctx context.Context, ownerID string) (domain.Cart, error) {
	var c domain.Cart

	cart, err := r.withTxCart(ctx, func(q *db.Queries) (domain.Cart, error) {
		dbCartItems, err := q.GetCart(ctx, ownerID)
		if err != nil {
			return c, fmt.Errorf("q.GetCart: %w", err)
		}

		cartItems, err := mapGetCartRowsToDomainCartItems(dbCartItems)
		if err != nil {
			return c, fmt.Errorf("mapGetCartRowsToDomainCartItems: %w", err)
		}

		return domain.Cart{
			OwnerID: ownerID,
			Items:   cartItems,
		}, nil
	})
	if err != nil {
		return c, fmt.Errorf("r.withTxCart: %w", err)
	}

	return cart, nil
}

func (r *cartRepository) AddItem(ctx context.Context, ownerID string, item domain.CartItem) error {
	err := r.withTxError(ctx, func(q *db.Queries) error {
		arg := db.AddItemParams{
			OwnerID:       ownerID,
			ProductID:     item.ProductID,
			PriceAmount:   item.Price.Amount,
			PriceCurrency: item.Price.Currency.String(),
		}

		if err := q.AddItem(ctx, arg); err != nil {
			return fmt.Errorf("q.AddItem: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("r.withTxError: %w", err)
	}

	return nil
}

func (r *cartRepository) DeleteItem(ctx context.Context, ownerID string, productID uuid.UUID) (bool, error) {
	var deleted bool

	deleted, err := r.withTxBool(ctx, func(q *db.Queries) (bool, error) {
		arg := db.DeleteItemParams{
			OwnerID:   ownerID,
			ProductID: productID,
		}

		rowsAffected, err := q.DeleteItem(ctx, arg)
		if err != nil {
			return false, fmt.Errorf("q.DeleteItem: %w", err)
		}

		// Check if any rows were affected by the delete operation
		return rowsAffected > 0, nil
	})
	if err != nil {
		return false, fmt.Errorf("r.withTxBool: %w", err)
	}

	return deleted, nil
}

func (r *cartRepository) withTxCart(ctx context.Context, fn func(q *db.Queries) (domain.Cart, error)) (domain.Cart, error) {
	return withTx(ctx, r.pool, r.q, fn)
}

func (r *cartRepository) withTxError(ctx context.Context, fn func(q *db.Queries) error) error {
	_, err := withTx(ctx, r.pool, r.q, func(q *db.Queries) (struct{}, error) {
		err := fn(q)
		return struct{}{}, err
	})
	return err
}

func (r *cartRepository) withTxBool(ctx context.Context, fn func(q *db.Queries) (bool, error)) (bool, error) {
	return withTx(ctx, r.pool, r.q, fn)
}

func mapGetCartRowToDomainCartItem(row db.GetCartRow) (domain.CartItem, error) {
	parsedCurrency, err := currency.ParseISO(row.PriceCurrency)
	if err != nil {
		return domain.CartItem{}, fmt.Errorf("currency[%s] is not valid: %w", row.PriceCurrency, err)
	}

	return domain.CartItem{
		ProductID: row.ProductID,
		Price:     domain.Money{Amount: row.PriceAmount, Currency: parsedCurrency},
		CreatedAt: row.CreatedAt,
	}, nil
}

func mapGetCartRowsToDomainCartItems(rows []db.GetCartRow) ([]domain.CartItem, error) {
	var cartItems []domain.CartItem
	for _, row := range rows {
		cartItem, err := mapGetCartRowToDomainCartItem(row)
		if err != nil {
			return nil, err
		}
		cartItems = append(cartItems, cartItem)
	}
	return cartItems, nil
}
