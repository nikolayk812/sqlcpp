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

type orderRepositorySuite struct {
	suite.Suite

	pool      *pgxpool.Pool
	repo      port.OrderRepository
	container testcontainers.Container
}

// entry point to run the tests in the suite
func TestOrderRepositorySuite(t *testing.T) {
	// Verifies no leaks after all tests in the suite run.
	defer goleak.VerifyNone(t)

	suite.Run(t, new(orderRepositorySuite))
}

// before all tests in the suite
func (suite *orderRepositorySuite) SetupSuite() {
	ctx := suite.T().Context()

	var (
		connStr string
		err     error
	)

	suite.container, connStr, err = startPostgres(ctx)
	suite.NoError(err)

	suite.pool, err = pgxpool.New(ctx, connStr)
	suite.NoError(err)

	suite.repo = repository.NewOrder(suite.pool)
	suite.NoError(err)
}

// after all tests in the suite
func (suite *orderRepositorySuite) TearDownSuite() {
	ctx := suite.T().Context()

	if suite.pool != nil {
		suite.pool.Close()
	}
	if suite.container != nil {
		suite.NoError(suite.container.Terminate(ctx))
	}
}

func (suite *orderRepositorySuite) TestInsertOrder() {
	order1 := fakeOrder()

	order2 := fakeOrder()
	order2.Items = nil

	tests := []struct {
		name      string
		order     domain.Order
		wantError error
	}{
		{
			name:  "single order: ok",
			order: order1,
		},
		{
			name:  "single order, no items: ok",
			order: order2,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			t := suite.T()
			ctx := t.Context()

			orderID, err := suite.repo.InsertOrder(ctx, tt.order)
			if err != nil {
				require.ErrorIs(t, err, tt.wantError)
				return
			}

			actualOrder, err := suite.repo.GetOrder(ctx, orderID)
			require.NoError(t, err)

			expectedOrder := domain.Order{
				ID:      orderID,
				OwnerID: tt.order.OwnerID,
				Items:   tt.order.Items,
			}
			assertOrder(t, expectedOrder, actualOrder)

		})
	}
}

func fakeOrder() domain.Order {
	var items []domain.OrderItem
	for i := 0; i < gofakeit.Number(1, 5); i++ {
		items = append(items, fakeOrderItem())
	}

	return domain.Order{
		ID:      uuid.MustParse(gofakeit.UUID()),
		OwnerID: gofakeit.UUID(),
		Items:   items,
	}
}

func fakeOrderItem() domain.OrderItem {
	productID := uuid.MustParse(gofakeit.UUID())

	price := gofakeit.Price(1, 100)

	currencyUnit := currency.MustParseISO(gofakeit.CurrencyShort())

	return domain.OrderItem{
		ProductID: productID,
		Price: domain.Money{
			Amount:   decimal.NewFromFloat(price),
			Currency: currencyUnit,
		},
	}
}

func assertOrder(t *testing.T, expected domain.Order, actual domain.Order) {
	t.Helper()

	// Custom comparer for Money.Currency fields
	comparer := cmp.Comparer(func(x, y currency.Unit) bool {
		return x.String() == y.String()
	})

	// Ignore the CreatedAt field in OrderItem and
	// Treat empty slices as equal to nil
	opts := cmp.Options{
		cmpopts.IgnoreFields(domain.OrderItem{}, "CreatedAt"),
		cmpopts.IgnoreFields(domain.Order{}, "CreatedAt", "UpdatedAt"),
		cmpopts.EquateEmpty(),
	}

	diff := cmp.Diff(expected, actual, comparer, opts)
	assert.Empty(t, diff)

	assert.False(t, actual.CreatedAt.IsZero())
	assert.False(t, actual.UpdatedAt.IsZero())
}
