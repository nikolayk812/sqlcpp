---
name: repository
description: Go repository implementation with SQLC integration following hexagonal architecture. Use when implementing data persistence layer, creating repository interfaces, or working with SQLC-generated queries. Focuses on domain model conversion, transaction handling, and error patterns.
allowed-tools: Read, Edit, Grep, Glob
---

# Repository - Go Data Layer Skill

This skill provides guidance on implementing repository pattern in Go using SQLC with hexagonal architecture principles.

## Core Principles

- **Domain Focus**: Accept and return domain models only, never SQLC-generated types
- **Interface Segregation**: Implement port interfaces from `internal/port/`
- **Transaction Support**: Handle both connection pools and transactions via `db.DBTX`
- **Transaction Usage**: Use transaction wrappers (`withTx`) only for repository methods that execute multiple SQLC queries; single-query methods can execute directly without transaction wrapping
- **SQLC Delegation**: Repository methods delegate to SQLC queries, then map results using consistent naming:
  - `map[SQLCType]ToDomain[DomainType](row db.Row) (domain.Type, error)`
  - `mapDomain[Type]To[SQLCParams](model domain.Type) db.Params`
  - Parameter names: `row` for single records, `rows` for slices

## Key Patterns

### Constructor
```go
func New(dbtx db.DBTX) (port.Repository, error) {
    if dbtx == nil {
        return nil, fmt.Errorf("dbtx is nil")
    }
    return &repo{q: db.New(dbtx), dbtx: dbtx}, nil
}
```

### Method Structure
```go
func (r *repo) Method(ctx context.Context, id uuid.UUID) (domain.Model, error) {
    if id == uuid.Nil {
        return domain.Model{}, fmt.Errorf("id is empty")
    }

    result, err := r.q.SQLCMethod(ctx, id)
    if err != nil {
        return domain.Model{}, fmt.Errorf("q.SQLCMethod: %w", err)
    }

    return mapSQLCToDomain(result)
}
```

### Transaction Handling
```go
result, err := withTx(ctx, r.dbtx, func(q *db.Queries) (Type, error) {
    if err := q.First(ctx, params); err != nil {
        return zero, fmt.Errorf("q.First: %w", err)
    }

    second, err := q.Second(ctx, params)
    if err != nil {
        return zero, fmt.Errorf("q.Second: %w", err)
    }

    return second, nil
})
```

### Error Patterns
```go
// Not found handling
if errors.Is(err, pgx.ErrNoRows) {
    return Model{}, fmt.Errorf("q.Method: %w", ErrNotFound)
}

// Affected rows check
if cmdTag.RowsAffected() == 0 {
    return fmt.Errorf("q.Method: %w", ErrNotFound)
}
```

### Mapping Conventions
```go
// SQLC to Domain - always with error return
func mapGetOrderRowToDomainOrder(row db.GetOrderRow) (domain.Order, error) {
    currency, err := currency.ParseISO(row.Currency)
    if err != nil {
        return domain.Order{}, fmt.Errorf("currency[%s]: %w", row.Currency, err)
    }
    return domain.Order{Currency: currency}, nil
}

// Domain to SQLC - typically no error
func mapDomainFilterToParams(filter domain.Filter) db.Params {
    return db.Params{IDs: nilSliceIfEmpty(filter.IDs)}
}
```