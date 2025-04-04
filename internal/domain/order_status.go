package domain

import "errors"

type OrderStatus string

// remember to add new statuses to the validOrderStatuses map
const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusShipped   OrderStatus = "shipped"
	OrderStatusDelivered OrderStatus = "delivered"
	OrderStatusCancelled OrderStatus = "cancelled"
)

var validOrderStatuses = map[OrderStatus]struct{}{
	OrderStatusPending:   {},
	OrderStatusShipped:   {},
	OrderStatusDelivered: {},
	OrderStatusCancelled: {},
}

func ToOrderStatus(s string) (OrderStatus, error) {
	status := OrderStatus(s)
	if _, ok := validOrderStatuses[status]; ok {
		return status, nil
	}

	return "", errors.New("invalid order status")
}

func OrderStatuses() []OrderStatus {
	result := make([]OrderStatus, 0, len(validOrderStatuses))
	for status := range validOrderStatuses {
		result = append(result, status)
	}
	return result
}
