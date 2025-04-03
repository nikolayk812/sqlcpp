package template

import (
	"fmt"
	"strings"
)

func BuildRepositoryDataMap(domainModelName string) map[string]string {
	lower := strings.ToLower(domainModelName)

	return map[string]string{
		"DomainModelName": lower,
		"SQLCModelsCode":  "internal/db/models.go",

		"OrderDomainModelCode":    "internal/domain/order.go",
		"OrderRepositoryCode":     "internal/repository/order_repository.go",
		"OrderRepositoryTestCode": "internal/repository/order_repository_test.go",

		"DomainModelCode":            fmt.Sprintf("internal/domain/%s.go", lower),
		"DomainModelSQLCQueriesCode": fmt.Sprintf("internal/db/%s.sql.go", lower),
		"DomainModelPortCode":        fmt.Sprintf("internal/port/%s_port.go", lower),
		"DomainModelRepositoryCode":  fmt.Sprintf("internal/repository/%s_repository.go", lower),
	}
}
