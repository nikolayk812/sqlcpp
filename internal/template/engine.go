package main

import (
	"log"
	"os"
	"strings"
	"text/template"
)

type Engine struct {
	fileMap map[string]string
}

func main() {
	// Read template from file
	templateFile := "internal/template/data/repo.tmpl"

	tmpl, err := template.ParseFiles(templateFile)
	if err != nil {
		log.Fatalf("Error parsing template: %v", err)
	}

	// Data to inject into the template
	data := map[string]string{
		"DomainModelName": "Cart",
		"SQLCModelsCode":  "db/models.go",

		"OrderDomainModelCode": "domain/order.go",
		"OrderRepositoryCode":  "repository/order_repository.go",

		"DomainModelCode":            "domain/cart.go",
		"DomainModelSQLCQueriesCode": "db/01_cart.sql.go",
		"DomainModelPortCode":        "port/cart_port.go",
	}

	for k, v := range data {
		if strings.HasSuffix(v, ".go") {
			path := "internal/" + v
			content, err := os.ReadFile(path)
			if err != nil {
				panic(err)
			}

			data[k] = string(content)
		}
	}

	// Execute template with data
	err = tmpl.Execute(os.Stdout, data)
	if err != nil {
		log.Fatalf("Error executing template: %v", err)
	}
}
