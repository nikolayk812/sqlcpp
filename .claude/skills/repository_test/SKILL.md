---
name: repository-test
description: Go repository testing with SQLC integration following hexagonal architecture. Use when writing integration tests for repository layer, testing database operations, and validating domain model persistence. Focuses on testcontainers, table-driven tests, and comprehensive error coverage.
allowed-tools: Read, Edit, Grep, Glob
---

# Repository Test - Go Data Layer Testing Skill

This skill provides guidance on testing repository implementations using real databases with comprehensive error coverage.

## Core Principles

- **Real Database Testing**: Use testcontainers with PostgreSQL, not mocks
- **Domain Model Focus**: Test with domain models only, never SQLC types
- **Comprehensive Coverage**: Test success paths and all error scenarios (validation, not found, state conflicts)
- **Table-Driven Structure**: Organize with clear test scenarios and expected outcomes
- **Test Isolation**: Use `defer suite.deleteAll()` and `TRUNCATE` for cleanup

## Key Patterns

### Suite Structure
```go
type repositorySuite struct {
    suite.Suite
    repo port.Repository
    pool *pgxpool.Pool
}
```

### Table-Driven Tests
```go
tests := []struct {
    name         string
    inputFunc    func() domain.Model
    prepareFunc  func(uuid.UUID) error  // optional setup
    targetIDFunc func() uuid.UUID       // optional ID override
    wantError    string
}{
    {name: "valid input: ok", inputFunc: randomModel},
    {name: "empty ID: error", targetIDFunc: func() uuid.UUID { return uuid.Nil }, wantError: "id is empty"},
}
```

### Error Testing Categories
```go
// Validation errors
wantError: "field is empty"

// Not found scenarios
wantError: "q.Method: record not found"

// State-dependent errors
prepareFunc: func(id uuid.UUID) error { return suite.repo.SoftDelete(ctx, id) }
```

### Custom Assertions
```go
func assertModel(t *testing.T, expected, actual domain.Model) {
    opts := cmp.Options{
        cmpopts.IgnoreFields(domain.Model{}, "CreatedAt", "UpdatedAt", "ID"),
        customComparers,
    }
    assert.Empty(t, cmp.Diff(expected, actual, opts))
    assert.False(t, actual.CreatedAt.IsZero())
}
```

## Testing Guidelines

- **Package**: Use `repository_test` package for interface testing
- **Data Generation**: Use `gofakeit` for realistic random data via helper functions
- **Container Setup**: Initialize with migration scripts via `postgres.WithInitScripts()`
- **Test Names**: Follow "action + condition: expected result" pattern
- **Error Messages**: Verify exact error messages match repository implementations
- **Cleanup**: Use `defer suite.deleteAll()` per test method for isolation