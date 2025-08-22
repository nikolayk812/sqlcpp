# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Code Generation
```bash
make generate
```
Regenerate type-safe Go code from SQL queries and add required DB() method. Use this instead of `sqlc generate` directly.

Alternative manual approach:
```bash
sqlc generate
./scripts/post-sqlc-generate.sh
```

### Testing
```bash
make test                        # Run all tests
go test ./internal/repository/   # Run specific package tests
go test -v ./...                 # Verbose test output with details
```

### Build and Development
```bash
make build             # Build all packages
make clean             # Clean and tidy dependencies
go mod download        # Download dependencies
go vet ./...           # Static analysis
go fmt ./...           # Format code
```

## Architecture

This project follows **hexagonal architecture** with clear separation of concerns:

### Core Structure
- **`internal/domain/`** - Business entities and logic (Order, Money, OrderStatus, OrderFilter)
- **`internal/port/`** - Interface definitions (repository contracts)
- **`internal/repository/`** - Data persistence implementations using SQLC
- **`internal/db/`** - SQLC-generated code and query definitions
- **`internal/migrations/`** - Database schema evolution

### Key Patterns
- **Domain-Driven Design**: Rich domain models with value objects (`Money`) and type-safe enums (`OrderStatus`)
- **Repository Pattern**: Interface segregation with dependency injection supporting both connection pools and transactions
- **SQLC Integration**: Type-safe SQL queries generated from `.sql` files in `internal/db/queries/`
- **Transaction Support**: Repository constructors accept both `*pgxpool.Pool` and `pgx.Tx` for flexible transaction handling

### Database Integration
- Uses **PostgreSQL** with **pgx/v5** driver
- **SQLC configuration** in `sqlc.yaml` with custom type mappings:
  - UUID fields use `github.com/google/uuid`
  - Decimal fields use `github.com/shopspring/decimal`
  - Timestamps use `time.Time`
- **Testcontainers** for integration testing with real PostgreSQL instances

### Development Workflow
1. Modify domain models in `internal/domain/`
2. Update SQL schema in `internal/migrations/`
3. Add/modify queries in `internal/db/queries/`
4. Run `make generate` to update generated code
5. Implement repository methods in `internal/repository/`
6. Write integration tests using Testcontainers
7. Run `go test ./...` to validate changes

### Testing Strategy
- **Integration tests** use real PostgreSQL via Testcontainers
- Test setup automatically applies migrations
- Repository tests cover CRUD operations, filtering, and transaction handling
- Use `setupTestDB(t, ctx)` helper for test database initialization