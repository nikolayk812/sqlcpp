.PHONY: generate test build

sqlc:
	sqlc generate
	./scripts/post-sqlc-generate.sh

test:
	TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock go test -v -race -cover ./...

build:
	go build ./...