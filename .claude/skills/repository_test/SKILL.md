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
- **One Assert Per Domain Model**: Create `assert[ModelName]` for EVERY domain model returned by repository methods (e.g., `assertCart`, `assertOrder`)
- **Mandatory Custom Assertions**: Compare all fields except generated ones (ID, timestamps). NEVER use `assert.Equal` on individual model fields
- **Comprehensive Error Coverage**: Test validation, not found, and state-dependent errors using `arrangeState` patterns
- **Table-Driven Structure**: Organize with clear test scenarios and expected outcomes
- **Consistent Cleanup**: Use `defer suite.deleteAll()` once per test method, never per test case

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
    buildModel   func() domain.Model
    arrangeState func(uuid.UUID) error  // arrange state for test case before the operation
    useModelID   func() uuid.UUID       // override which model ID to use, if nil use the inserted one
    wantError    string
}{
    // Must test ALL error categories:
    {name: "valid input: ok", buildModel: randomModel},
    {name: "empty id: error", useModelID: func() uuid.UUID { return uuid.Nil }, wantError: "id is empty"},
    {name: "non-existing: not found", useModelID: randomUUID, wantError: "q.Method: record not found"},
    {name: "soft-deleted: not found", arrangeState: softDelete, wantError: "q.Method: record not found"},
}
```

### Forbidden Patterns
```go
// NEVER do manual field assertions
require.Equal(t, expected.OwnerID, actual.OwnerID)  // ❌ FORBIDDEN
require.Len(t, actual.Items, len(expected.Items))   // ❌ FORBIDDEN

// ALWAYS use comprehensive model assertions
assertCart(t, expected, actual)                     // ✅ REQUIRED
```

### Custom Assertions (Required for Each Model)
```go
// MUST create for every domain model - compare ALL fields except generated ones
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
- **Cleanup Pattern**: One `defer suite.deleteAll()` per test method only