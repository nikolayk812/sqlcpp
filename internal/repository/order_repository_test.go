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
	"net/url"
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

	badOrder := fakeOrder()
	badOrder.Items = nil

	tests := []struct {
		name      string
		order     domain.Order
		wantError string
	}{
		{
			name:  "single order: ok",
			order: order1,
		},
		{
			name:      "single order, no items: ok",
			order:     badOrder,
			wantError: "no items in order",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			t := suite.T()
			ctx := t.Context()

			o := tt.order

			orderID, err := suite.repo.InsertOrder(ctx, o)
			if err != nil {
				assert.EqualError(t, err, tt.wantError)
				return
			}

			actualOrder, err := suite.repo.GetOrder(ctx, orderID)
			require.NoError(t, err)

			expectedOrder := domain.Order{
				ID:      orderID,
				OwnerID: o.OwnerID,
				Items:   o.Items,
				Url:     o.Url,
				Status:  domain.OrderStatusPending,
				Tags:    o.Tags,
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

	var tags []string
	for i := 0; i < gofakeit.Number(1, 5); i++ {
		tags = append(tags, gofakeit.BeerName())
	}

	return domain.Order{
		ID:      uuid.MustParse(gofakeit.UUID()),
		OwnerID: gofakeit.UUID(),
		Items:   items,
		Url:     fakeURL(),
		Tags:    tags,
	}
}

func fakeOrderItem() domain.OrderItem {
	productID := uuid.MustParse(gofakeit.UUID())

	price := gofakeit.Price(1, 100)

	currencyUnit := fakeCurrency()

	return domain.OrderItem{
		ProductID: productID,
		Price: domain.Money{
			Amount:   decimal.NewFromFloat(price),
			Currency: currencyUnit,
		},
	}
}

func fakeURL() *url.URL {
	var (
		result *url.URL
		err    error
	)

	for {
		result, err = url.Parse(gofakeit.URL())
		if err == nil {
			break
		}
	}

	return result
}

func fakeCurrency() currency.Unit {
	var (
		result currency.Unit
		err    error
	)

	for {
		// tag is not a recognized currency
		result, err = currency.ParseISO(gofakeit.CurrencyShort())
		if err == nil {
			break
		}
	}

	return result
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
