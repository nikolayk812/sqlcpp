# SQLC++

A Go project using SQLC for type-safe database operations with domain-driven design.

> **Note**: This is a reference implementation for [sqlcpp-demo](https://github.com/nikolayk812/sqlcpp-demo).

## What is this?

This project shows how to build Go applications with:
- SQLC for database code generation
- PostgreSQL with pgx driver
- Domain-driven design patterns
- Repository pattern implementation
- Integration testing with Testcontainers

## Structure

```
internal/
├── domain/      # Business models (Order, Money, OrderStatus)
├── port/        # Repository interfaces  
├── repository/  # Repository implementations
├── db/          # Generated SQLC code
└── migrations/  # Database schema
```

## Key Files

- `internal/domain/order.go` - Main business entity
- `internal/port/order_port.go` - Repository interface
- `internal/repository/order_repository.go` - Database implementation
- `internal/db/queries/order.sql` - SQL queries for SQLC
- `sqlc.yaml` - SQLC configuration

## Database

Uses PostgreSQL with custom type mappings:
- UUID fields → `github.com/google/uuid`
- Decimal fields → `github.com/shopspring/decimal`
- Timestamps → `time.Time`

## Testing

Integration tests use Testcontainers with real PostgreSQL instances. Tests automatically set up database schema and run migrations.

## Usage

1. Define domain models in `internal/domain/`
2. Add SQL migrations in `internal/migrations/`
3. Run `sqlc generate` it would generate files in `internal/db/queries/`
4. Implement repository methods `internal/repository/` which use SQLC-generated queries
5. Add integration tests in `internal/repository/` for new repository methods