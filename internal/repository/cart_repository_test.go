package repository_test

import (
	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nikolayk812/sqlcpp/internal/domain"
	"github.com/nikolayk812/sqlcpp/internal/port"
	"github.com/nikolayk812/sqlcpp/internal/repository"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"go.uber.org/goleak"
	"golang.org/x/text/currency"
	"testing"
)

type cartRepositorySuite struct {
	suite.Suite

	pool      *pgxpool.Pool
	repo      port.CartRepository
	container testcontainers.Container
}

// entry point to run the tests in the suite
func TestCartRepositorySuite(t *testing.T) {
	// Verifies no leaks after all tests in the suite run.
	defer goleak.VerifyNone(t)

	suite.Run(t, new(cartRepositorySuite))
}

// before all tests in the suite
func (suite *cartRepositorySuite) SetupSuite() {
	ctx := suite.T().Context()

	var (
		connStr string
		err     error
	)

	suite.container, connStr, err = startPostgres(ctx)
	suite.NoError(err)

	suite.pool, err = pgxpool.New(ctx, connStr)
	suite.NoError(err)

	suite.repo = repository.NewCart(suite.pool)
	suite.NoError(err)
}

// after all tests in the suite
func (suite *cartRepositorySuite) TearDownSuite() {
	ctx := suite.T().Context()

	if suite.pool != nil {
		suite.pool.Close()
	}
	if suite.container != nil {
		suite.NoError(suite.container.Terminate(ctx))
	}
}

func (suite *cartRepositorySuite) TestAddItem() {
	item1 := fakeCartItem()
	item2 := fakeCartItem()

	tests := []struct {
		name      string
		ownerID   string
		item      domain.CartItem
		wantError error
	}{
		{
			name:    "add single item: ok",
			ownerID: gofakeit.UUID(),
			item:    item1,
		},
		{
			name:    "add another item: ok",
			ownerID: gofakeit.UUID(),
			item:    item2,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			t := suite.T()
			ctx := t.Context()

			err := suite.repo.AddItem(ctx, tt.ownerID, tt.item)
			require.NoError(t, err)

			actualCart, err := suite.repo.GetCart(ctx, tt.ownerID)
			require.NoError(t, err)

			expectedCart := domain.Cart{
				OwnerID: tt.ownerID,
				Items:   []domain.CartItem{tt.item},
			}

			assertCart(t, expectedCart, actualCart)
		})
	}
}

func (suite *cartRepositorySuite) TestDeleteItem() {
	item := fakeCartItem()
	ownerID := gofakeit.UUID()

	suite.repo.AddItem(suite.T().Context(), ownerID, item)

	tests := []struct {
		name      string
		ownerID   string
		productID uuid.UUID
		wantFound bool
		wantError error
	}{
		{
			name:      "delete existing item: ok",
			ownerID:   ownerID,
			productID: item.ProductID,
			wantFound: true,
		},
		{
			name:      "delete non-existing item: not found",
			ownerID:   ownerID,
			productID: uuid.New(),
			wantFound: false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			t := suite.T()
			ctx := t.Context()

			found, err := suite.repo.DeleteItem(ctx, tt.ownerID, tt.productID)
			require.NoError(t, err)
			assert.Equal(t, tt.wantFound, found)
		})
	}
}

func fakeCartItem() domain.CartItem {
	productID := uuid.MustParse(gofakeit.UUID())

	price := gofakeit.Price(1, 100)

	currencyUnit := currency.MustParseISO(gofakeit.CurrencyShort())

	return domain.CartItem{
		ProductID: productID,
		Price: domain.Money{
			Amount:   decimal.NewFromFloat(price),
			Currency: currencyUnit,
		},
	}
}

func assertCart(t *testing.T, expected domain.Cart, actual domain.Cart) {
	t.Helper()

	// Custom comparer for Money.Currency fields
	comparer := cmp.Comparer(func(x, y currency.Unit) bool {
		return x.String() == y.String()
	})

	// Ignore the CreatedAt field in CartItem and
	// Treat empty slices as equal to nil
	opts := cmp.Options{
		cmpopts.IgnoreFields(domain.CartItem{}, "CreatedAt"),
		cmpopts.EquateEmpty(),
	}

	diff := cmp.Diff(expected, actual, comparer, opts)
	assert.Empty(t, diff)
}
