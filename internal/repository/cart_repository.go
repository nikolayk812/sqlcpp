package repository

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/nikolayk812/sqlcpp/internal/db"
	"github.com/nikolayk812/sqlcpp/internal/domain"
	"golang.org/x/text/currency"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CartRepository interface {
	GetCart(ctx context.Context, ownerID string) (domain.Cart, error)
	AddItem(ctx context.Context, ownerID string, item domain.CartItem) error
	DeleteItem(ctx context.Context, ownerID string, productID uuid.UUID) (bool, error)
}

type cartRepository struct {
	q *db.Queries
}

func NewCartRepository(pool *pgxpool.Pool) CartRepository {
	return &cartRepository{
		q: db.New(pool),
	}
}

func NewCartRepositoryWithTx(tx pgx.Tx) CartRepository {
	return &cartRepository{
		q: db.New(tx),
	}
}

func (r *cartRepository) GetCart(ctx context.Context, ownerID string) (domain.Cart, error) {
	cart := domain.Cart{OwnerID: ownerID}

	items, err := r.q.GetCart(ctx, ownerID)
	if err != nil {
		return cart, err
	}

	for _, item := range items {
		cartItem, err := mapGetCartRowToCartItem(item)
		if err != nil {
			return cart, err
		}

		cart.Items = append(cart.Items, cartItem)
	}

	return cart, nil
}

func (r *cartRepository) AddItem(ctx context.Context, ownerID string, item domain.CartItem) error {
	err := r.q.AddItem(ctx, db.AddItemParams{
		OwnerID:       ownerID,
		ProductID:     item.ProductID,
		PriceAmount:   item.Price.Amount,
		PriceCurrency: item.Price.Currency.String(),
	})
	if err != nil {
		return err
	}

	return nil
}

func (r *cartRepository) DeleteItem(ctx context.Context, ownerID string, productID uuid.UUID) (bool, error) {
	rowsAffected, err := r.q.DeleteItem(ctx, db.DeleteItemParams{
		OwnerID:   ownerID,
		ProductID: productID})
	if err != nil {
		return false, err
	}

	return rowsAffected > 0, nil
}

func mapGetCartRowToCartItem(row db.GetCartRow) (domain.CartItem, error) {
	parsedCurrency, err := currency.ParseISO(row.PriceCurrency)
	if err != nil {
		return domain.CartItem{}, fmt.Errorf("currency[%s] is not valid: %w", row.PriceCurrency, err)
	}

	return domain.CartItem{
		ProductID: row.ProductID,
		Price: domain.Money{
			Amount:   row.PriceAmount,
			Currency: parsedCurrency,
		},
		CreatedAt: row.CreatedAt,
	}, nil
}
