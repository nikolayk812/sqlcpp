.PHONY: generate test build

sqlc:
	sqlc generate
	./scripts/post-sqlc-generate.sh

test:
	go test -v -race -cover ./...

build:
	go build ./...