package repository_test

import (
	"encoding/json"
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
	defer suite.deleteAll()

	order1 := fakeOrder()

	order2 := fakeOrder()
	order2.Tags = nil
	order2.Url = nil

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
		{
			name:  "single order, nil tags, nil url: ok",
			order: order2,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			t := suite.T()
			ctx := t.Context()

			o := tt.order

			orderID, err := suite.repo.InsertOrder(ctx, o)
			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)
				return
			}

			actualOrder, err := suite.repo.GetOrder(ctx, orderID)
			require.NoError(t, err)

			expected := o
			expected.ID = orderID
			expected.Status = domain.OrderStatusPending

			assertOrder(t, expected, actualOrder)
		})
	}
}

func (suite *orderRepositorySuite) TestGetOrderJoin() {
	defer suite.deleteAll()

	order1 := fakeOrder()

	order2 := fakeOrder()
	order2.Tags = nil
	order2.Url = nil
	order2.Payload = nil
	order2.PayloadB = nil

	tests := []struct {
		name       string
		order      domain.Order
		expectFunc func(*domain.Order)
		wantError  string
	}{
		{
			name:  "existing order: ok",
			order: order1,
		},
		{
			name:      "non-existing order: not ok",
			order:     domain.Order{ID: uuid.MustParse(gofakeit.UUID())},
			wantError: "order not found",
		},
		{
			name:  "single order, most fields nil: ok",
			order: order2,
			expectFunc: func(o *domain.Order) {
				o.Payload = []byte(`{}`)
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			t := suite.T()

			o := tt.order

			var (
				orderID uuid.UUID
				err     error
			)

			if o.ID == uuid.Nil {
				orderID, err = suite.repo.InsertOrder(t.Context(), o)
				require.NoError(t, err)
			} else {
				orderID = o.ID
			}

			actualOrder, err := suite.repo.GetOrderJoin(t.Context(), orderID)
			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)

			expected := o
			if tt.expectFunc != nil {
				tt.expectFunc(&expected)
			}

			assertOrder(t, expected, actualOrder)
		})
	}
}

func (suite *orderRepositorySuite) TestSearchOrders() {
	order1 := fakeOrder()
	order2 := fakeOrder()

	tests := []struct {
		name       string
		orders     []domain.Order
		filter     domain.OrderFilter
		wantOrders []domain.Order
		wantError  string
	}{
		{
			name:   "search by owner ids",
			orders: []domain.Order{order1, order2},
			filter: domain.OrderFilter{
				OwnerIDs: []string{order1.OwnerID},
			},
			wantOrders: []domain.Order{order1},
		},
		{
			name:   "search by owner empty",
			orders: []domain.Order{order1, order2},
			filter: domain.OrderFilter{
				OwnerIDs: []string{order1.OwnerID},
			},
			wantOrders: []domain.Order{order1},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			defer suite.deleteAll()

			t := suite.T()
			ctx := t.Context()

			for _, order := range tt.orders {
				_, err := suite.repo.InsertOrder(ctx, order)
				require.NoError(t, err)
			}

			orders, err := suite.repo.SearchOrders(ctx, tt.filter)
			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)

			assertOrders(t, tt.wantOrders, orders)
		})
	}
}

func (suite *orderRepositorySuite) TestDeleteOrder() {
	order1 := fakeOrder()

	tests := []struct {
		name        string
		order       domain.Order
		orderIDFunc func(uuid.UUID) uuid.UUID
		wantError   string
	}{
		{
			name:  "delete existing order: ok",
			order: order1,
		},
		{
			name:  "delete non-existing order: not found",
			order: order1,
			orderIDFunc: func(_ uuid.UUID) uuid.UUID {
				return uuid.MustParse(gofakeit.UUID())
			},
			wantError: "r.withTx: q.DeleteOrderItems: not found",
		},
		{
			name:  "delete with empty order ID: error",
			order: order1,
			orderIDFunc: func(_ uuid.UUID) uuid.UUID {
				return uuid.Nil
			},
			wantError: "orderID is empty",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			defer suite.deleteAll()

			t := suite.T()
			ctx := t.Context()
			o := tt.order

			orderID, err := suite.repo.InsertOrder(t.Context(), o)
			require.NoError(t, err)

			toDeleteOrderID := orderID
			if tt.orderIDFunc != nil {
				toDeleteOrderID = tt.orderIDFunc(orderID)
			}

			err = suite.repo.DeleteOrder(ctx, toDeleteOrderID)
			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)

			// Verify the order is deleted
			_, err = suite.repo.GetOrder(ctx, orderID)
			assert.EqualError(t, err, "r.withTxOrder: q.GetOrder: not found")
		})
	}
}

func (suite *orderRepositorySuite) deleteAll() {
	_, err := suite.pool.Exec(suite.T().Context(), "TRUNCATE TABLE orders, order_items CASCADE")
	suite.NoError(err)
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
		ID:       uuid.Nil,
		OwnerID:  gofakeit.UUID(),
		Items:    items,
		Url:      fakeURL(),
		Tags:     tags,
		Payload:  fakeJson(),
		PayloadB: fakeJson(),
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

func fakeJson() []byte {
	var (
		result []byte
		err    error
	)

	for {
		result, err = gofakeit.JSON(nil)
		if err == nil {
			break
		}
	}

	return result
}

func assertOrder(t *testing.T, expected domain.Order, actual domain.Order) {
	t.Helper()

	currencyComparer := cmp.Comparer(func(x, y currency.Unit) bool {
		return x.String() == y.String()
	})

	jsonComparer := cmp.Comparer(func(x, y []byte) bool {
		if x == nil && y == nil {
			return true
		}

		var normalizedX, normalizedY interface{}

		if err := json.Unmarshal(x, &normalizedX); err != nil {
			return false
		}
		if err := json.Unmarshal(y, &normalizedY); err != nil {
			return false
		}

		return cmp.Equal(normalizedX, normalizedY)
	})

	// Ignore the CreatedAt field in OrderItem and
	// Treat empty slices as equal to nil
	opts := cmp.Options{
		cmpopts.IgnoreFields(domain.OrderItem{}, "CreatedAt"),
		cmpopts.IgnoreFields(domain.Order{}, "CreatedAt", "UpdatedAt", "ID", "Status"),
		// cmpopts.EquateEmpty(),
		currencyComparer,
		cmp.FilterPath(func(p cmp.Path) bool {
			return p.Last().String() == ".Payload" || p.Last().String() == ".PayloadB"
		}, jsonComparer),
	}

	diff := cmp.Diff(expected, actual, opts)
	assert.Empty(t, diff)

	assert.False(t, actual.CreatedAt.IsZero())
	assert.False(t, actual.UpdatedAt.IsZero())
	assert.Equal(t, domain.OrderStatusPending, actual.Status)
	assert.NotEqual(t, uuid.Nil, actual.ID)
}

func assertOrders(t *testing.T, expected []domain.Order, actual []domain.Order) {
	t.Helper()

	require.Equal(t, len(expected), len(actual))

	for i := range expected {
		assertOrder(t, expected[i], actual[i])
	}
}
