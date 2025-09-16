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
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"go.uber.org/goleak"
	"golang.org/x/text/currency"
	"net/url"
	"os"
	"sort"
	"testing"
	"time"
)

type orderRepositorySuite struct {
	suite.Suite

	pool      *pgxpool.Pool
	repo      port.OrderRepository
	container testcontainers.Container
}

// entry point to run the tests in the suite
func TestOrderRepositorySuite(t *testing.T) {
	require.NoError(t, os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true"))

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

	suite.repo, err = repository.NewOrder(suite.pool)
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

	tests := []struct {
		name      string
		orderFn   func() domain.Order
		wantError string
	}{
		{
			name:    "valid order with all fields: ok",
			orderFn: fakeOrder,
		},
		{
			name: "invalid order, no items: fail",
			orderFn: func() domain.Order {
				o := fakeOrder()
				o.Items = nil
				return o
			},
			wantError: "no items in order",
		},
		{
			name:    "valid order, nil tags, nil url: ok",
			orderFn: fakeOrder,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			t := suite.T()
			ctx := t.Context()

			o := tt.orderFn()

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

func (suite *orderRepositorySuite) TestUpdateOrderStatus() {
	tests := []struct {
		name              string
		orderFn           func() domain.Order
		newStatus         domain.OrderStatus
		orderIDToUpdateFn func() uuid.UUID
		setup             func(uuid.UUID) error
		wantError         string
	}{
		{
			name:      "update status of existing order: ok",
			orderFn:   fakeOrder,
			newStatus: domain.OrderStatusShipped,
		},
		{
			name:      "update status of non-existing order: not found",
			orderFn:   fakeOrder,
			newStatus: domain.OrderStatusShipped,
			orderIDToUpdateFn: func() uuid.UUID {
				return uuid.MustParse(gofakeit.UUID())
			},
			wantError: "q.UpdateOrderStatus: order not found",
		},
		{
			name:      "update status with empty order ID: error",
			orderFn:   fakeOrder,
			newStatus: domain.OrderStatusShipped,
			orderIDToUpdateFn: func() uuid.UUID {
				return uuid.Nil
			},
			wantError: "orderID is empty",
		},
		{
			name:      "update status with empty status: error",
			orderFn:   fakeOrder,
			newStatus: "",
			wantError: "status is empty",
		},
		{
			name:      "update status of soft-deleted order: not found",
			orderFn:   fakeOrder,
			newStatus: domain.OrderStatusShipped,
			setup: func(u uuid.UUID) error {
				return suite.repo.SoftDeleteOrder(suite.T().Context(), u)
			},
			wantError: "q.UpdateOrderStatus: order not found",
		},
		{
			name:      "update status of deleted order: not found",
			orderFn:   fakeOrder,
			newStatus: domain.OrderStatusShipped,
			setup: func(u uuid.UUID) error {
				return suite.repo.DeleteOrder(suite.T().Context(), u)
			},
			wantError: "q.UpdateOrderStatus: order not found",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			defer suite.deleteAll()

			t := suite.T()
			ctx := t.Context()

			order := tt.orderFn()

			orderID, err := suite.repo.InsertOrder(ctx, order)
			require.NoError(t, err)

			if tt.setup != nil {
				err := tt.setup(orderID)
				require.NoError(t, err)
			}

			orderIDToUpdate := orderID
			if tt.orderIDToUpdateFn != nil {
				orderIDToUpdate = tt.orderIDToUpdateFn()
			}

			// Perform the status update
			err = suite.repo.UpdateOrderStatus(ctx, orderIDToUpdate, tt.newStatus)
			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)

			updatedOrder, err := suite.repo.GetOrder(ctx, orderID)
			require.NoError(t, err)

			expected := order
			expected.Status = tt.newStatus

			assertOrder(t, expected, updatedOrder)
		})
	}
}

func (suite *orderRepositorySuite) TestGetOrderJoin() {
	defer suite.deleteAll()

	tests := []struct {
		name      string
		orderFn   func() domain.Order
		wantError string
	}{
		{
			name:    "existing order: ok",
			orderFn: fakeOrder,
		},
		{
			name:      "non-existing order: not ok",
			orderFn:   func() domain.Order { return domain.Order{ID: uuid.MustParse(gofakeit.UUID())} },
			wantError: "order not found",
		},
		{
			name: "single order, most fields nil: ok",
			orderFn: func() domain.Order {
				order := fakeOrder()
				order.Tags = nil
				order.Url = nil
				order.Payload = []byte(`{}`)
				order.PayloadB = nil
				return order
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			t := suite.T()

			order := tt.orderFn()

			// Determine if we need to create a new order or use an existing ID
			orderID := order.ID
			if orderID == uuid.Nil {
				// Insert a new order since no ID was provided
				var err error
				orderID, err = suite.repo.InsertOrder(t.Context(), order)
				require.NoError(t, err)
			}

			actualOrder, err := suite.repo.GetOrderJoin(t.Context(), orderID)
			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)

			expected := order
			assertOrder(t, expected, actualOrder)
		})
	}
}

func (suite *orderRepositorySuite) TestSearchOrders() {
	defer suite.deleteAll()

	order1 := fakeOrder()
	order2 := fakeOrder()
	orderIDs := suite.insertOrders(order1, order2)

	tests := []struct {
		name       string
		filter     domain.OrderFilter
		wantOrders []domain.Order
		wantError  string
	}{
		{
			name:      "empty filter: error",
			filter:    domain.OrderFilter{},
			wantError: "filter.Validate: all fields are empty",
		},
		{
			name: "search by ids: 1 found",
			filter: domain.OrderFilter{
				IDs: []uuid.UUID{orderIDs[0]},
			},
			wantOrders: []domain.Order{order1},
		},
		{
			name: "search by ids: 2 found",
			filter: domain.OrderFilter{
				IDs: []uuid.UUID{orderIDs[0], orderIDs[1]},
			},
			wantOrders: []domain.Order{order1, order2},
		},
		{
			name: "search by ids: not found",
			filter: domain.OrderFilter{
				IDs: []uuid.UUID{uuid.MustParse(gofakeit.UUID())},
			},
		},
		{
			name: "search by owner ids: 1 found",
			filter: domain.OrderFilter{
				OwnerIDs: []string{order1.OwnerID},
			},
			wantOrders: []domain.Order{order1},
		},
		{
			name: "search by owner ids: 2 found",
			filter: domain.OrderFilter{
				OwnerIDs: []string{order1.OwnerID, order2.OwnerID},
			},
			wantOrders: []domain.Order{order1, order2},
		},
		{
			name: "search by owner ids: not found",
			filter: domain.OrderFilter{
				OwnerIDs: []string{"not found"},
			},
		},
		{
			name: "search by URL patterns: 1 found",
			filter: domain.OrderFilter{
				UrlPatterns: []string{order1.Url.String()},
			},
			wantOrders: []domain.Order{order1},
		},
		{
			name: "search by URL patterns: not found",
			filter: domain.OrderFilter{
				UrlPatterns: []string{"not found"},
			},
		},
		{
			name: "search by status pending: 2 found",
			filter: domain.OrderFilter{
				Statuses: []domain.OrderStatus{domain.OrderStatusPending},
			},
			wantOrders: []domain.Order{order1, order2},
		},
		{
			name: "search by status shipped: not found",
			filter: domain.OrderFilter{
				Statuses: []domain.OrderStatus{domain.OrderStatusShipped},
			},
		},
		{
			name: "search by tags: 1 found",
			filter: domain.OrderFilter{
				Tags: []string{order1.Tags[0]},
			},
			wantOrders: []domain.Order{order1},
		},
		{
			name: "search by tags: not found",
			filter: domain.OrderFilter{
				Tags: []string{"not found"},
			},
		},
		{
			name: "search by createdAt after: 2 found",
			filter: domain.OrderFilter{
				CreatedAt: lo.ToPtr(domain.TimeRange{
					After: lo.ToPtr(time.Now().UTC().Add(-1 * time.Minute)),
				}),
			},
			wantOrders: []domain.Order{order1, order2},
		},
		{
			name: "search by createdAt after: not found",
			filter: domain.OrderFilter{
				CreatedAt: lo.ToPtr(domain.TimeRange{
					After: lo.ToPtr(time.Now().UTC().Add(1 * time.Minute)),
				}),
			},
		},
		{
			name: "search by createdAt before: not found",
			filter: domain.OrderFilter{
				CreatedAt: lo.ToPtr(domain.TimeRange{
					Before: lo.ToPtr(time.Now().UTC().Add(-1 * time.Minute)),
				}),
			},
		},
		{
			name: "search by createdAt before: 2 found",
			filter: domain.OrderFilter{
				CreatedAt: lo.ToPtr(domain.TimeRange{
					Before: lo.ToPtr(time.Now().UTC().Add(1 * time.Minute)),
				}),
			},
			wantOrders: []domain.Order{order1, order2},
		},
		{
			name: "search by createdAt empty: error",
			filter: domain.OrderFilter{
				CreatedAt: lo.ToPtr(domain.TimeRange{}),
			},
			wantError: "filter.Validate: createdAt: both Before and After are nil",
		},
		{
			name: "search by createdAt before and after: 2 found",
			filter: domain.OrderFilter{
				CreatedAt: lo.ToPtr(domain.TimeRange{
					Before: lo.ToPtr(time.Now().UTC().Add(1 * time.Minute)),
					After:  lo.ToPtr(time.Now().UTC().Add(-1 * time.Minute)),
				}),
			},
			wantOrders: []domain.Order{order1, order2},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			t := suite.T()

			orders, err := suite.repo.SearchOrders(t.Context(), tt.filter)
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
	tests := []struct {
		name            string
		orderFn         func() domain.Order
		setup           func(uuid uuid.UUID) error
		orderIDToDelete func() uuid.UUID
		wantError       string
	}{
		{
			name:    "delete existing order: ok",
			orderFn: fakeOrder,
		},
		{
			name:    "delete non-existing order: not found",
			orderFn: fakeOrder,
			orderIDToDelete: func() uuid.UUID {
				return uuid.MustParse(gofakeit.UUID())
			},
			wantError: "withTx: q.DeleteOrderItems: order not found",
		},
		{
			name:    "delete with empty order ID: error",
			orderFn: fakeOrder,
			orderIDToDelete: func() uuid.UUID {
				return uuid.Nil
			},
			wantError: "orderID is empty",
		},
		{
			name:    "delete soft-deleted order: ok",
			orderFn: fakeOrder,
			setup: func(orderID uuid.UUID) error {
				return suite.repo.SoftDeleteOrder(suite.T().Context(), orderID)
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			t := suite.T()
			ctx := t.Context()

			order := tt.orderFn()
			orderID, err := suite.repo.InsertOrder(t.Context(), order)
			require.NoError(t, err)

			if tt.setup != nil {
				err := tt.setup(orderID)
				require.NoError(t, err)
			}

			toDeleteOrderID := orderID
			if tt.orderIDToDelete != nil {
				toDeleteOrderID = tt.orderIDToDelete()
			}

			err = suite.repo.DeleteOrder(ctx, toDeleteOrderID)
			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)

			// Verify the order is deleted
			_, err = suite.repo.GetOrder(ctx, orderID)
			assert.EqualError(t, err, "withTx: q.GetOrder: order not found")
		})
	}
}

func (suite *orderRepositorySuite) TestSoftDeleteOrder() {
	tests := []struct {
		name                string
		orderFn             func() domain.Order
		orderIDToSoftDelete func() uuid.UUID
		setup               func(uuid.UUID) error
		wantError           string
	}{
		{
			name:    "soft-delete existing order: ok",
			orderFn: fakeOrder,
		},
		{
			name:    "soft-delete non-existing order: not found",
			orderFn: fakeOrder,
			orderIDToSoftDelete: func() uuid.UUID {
				return uuid.MustParse(gofakeit.UUID())
			},
			wantError: "q.SoftDeleteOrder: order not found",
		},
		{
			name:    "soft-delete with empty order ID: error",
			orderFn: fakeOrder,
			orderIDToSoftDelete: func() uuid.UUID {
				return uuid.Nil
			},
			wantError: "orderID is empty",
		},
		{
			name:    "soft-delete deleted order: not found",
			orderFn: fakeOrder,
			setup: func(orderID uuid.UUID) error {
				return suite.repo.DeleteOrder(suite.T().Context(), orderID)
			},
			wantError: "q.SoftDeleteOrder: order not found",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			t := suite.T()
			ctx := t.Context()

			// Insert the order if needed
			order := tt.orderFn()
			orderID, err := suite.repo.InsertOrder(ctx, order)
			require.NoError(t, err)

			toDeleteOrderID := orderID
			if tt.orderIDToSoftDelete != nil {
				toDeleteOrderID = tt.orderIDToSoftDelete()
			}

			if tt.setup != nil {
				err := tt.setup(orderID)
				require.NoError(t, err)
			}

			err = suite.repo.SoftDeleteOrder(ctx, toDeleteOrderID)
			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)

			// Verify the order is soft-deleted
			_, err = suite.repo.GetOrder(ctx, orderID)
			assert.EqualError(t, err, "withTx: q.GetOrder: order not found")
		})
	}
}

func (suite *orderRepositorySuite) insertOrders(orders ...domain.Order) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(orders))

	for _, order := range orders {
		id, err := suite.repo.InsertOrder(suite.T().Context(), order)
		suite.NoError(err)
		ids = append(ids, id)
	}

	return ids
}

func (suite *orderRepositorySuite) deleteAll() {
	_, err := suite.pool.Exec(suite.T().Context(), "TRUNCATE TABLE orders, order_items CASCADE")
	suite.NoError(err)
}

func fakeOrder() domain.Order {
	currencyUnit := fakeCurrency() // it has to be the same for all items
	orderAmount := decimal.Zero

	var items []domain.OrderItem
	for i := 0; i < gofakeit.Number(1, 5); i++ {
		fakeItem := fakeOrderItem()
		fakeItem.Price.Currency = currencyUnit
		orderAmount = orderAmount.Add(fakeItem.Price.Amount)
		items = append(items, fakeOrderItem())
	}

	var tags []string
	for i := 0; i < gofakeit.Number(1, 3); i++ {
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
		Price: domain.Money{
			Amount:   orderAmount,
			Currency: currencyUnit,
		},
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
	assert.Nil(t, actual.DeletedAt)
	assert.NotEqual(t, uuid.Nil, actual.ID)
}

func assertOrders(t *testing.T, expected []domain.Order, actual []domain.Order) {
	t.Helper()

	sortOrders := func(orders []domain.Order) {
		sort.Slice(orders, func(i, j int) bool {
			return orders[i].OwnerID < orders[j].OwnerID
		})
	}

	sortOrders(expected)
	sortOrders(actual)

	require.Equal(t, len(expected), len(actual))

	for i := range expected {
		assertOrder(t, expected[i], actual[i])
	}
}
