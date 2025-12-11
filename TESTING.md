# Testing Guide

This document describes the testing strategy and how to run tests for the JapaneseStudent backend.

## Test Structure

The project includes three types of tests:

1. **Unit Tests for Repositories** (`internal/repositories/repository_test.go`)
   - Uses `sqlmock` to mock database interactions
   - Tests all repository methods with various scenarios
   - No database connection required
   - Fast execution, suitable for CI/CD pipelines

2. **Unit Tests for Services** (`internal/services/service_test.go`)
   - Uses mock repositories to test business logic
   - Tests validation, error handling, and service methods
   - No database connection required
   - Fast execution, suitable for CI/CD pipelines

3. **Integration Tests** (`test/integration/characters_test.go`)
   - Tests the full application stack (handler → service → repository → database)
   - Requires a test database connection
   - Tests end-to-end API endpoints
   - Includes benchmark tests for performance measurement
   - Automatically sets up and tears down test data

## Running Tests

### Unit Tests (No Database Required)

Run all unit tests:
```bash
go test ./internal/...
```

Run repository tests only:
```bash
go test ./internal/repositories/... -v
```

Run service tests only:
```bash
go test ./internal/services/... -v
```

Run with coverage:
```bash
go test ./internal/... -cover
```

Generate coverage report:
```bash
go test ./internal/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Integration Tests (Database Required)

**Prerequisites:**
1. MySQL/MariaDB server running
2. Test database created (or use existing database)
3. Database credentials configured via environment variables

**Setup:**

1. Create a test database:
```sql
CREATE DATABASE japanesestudent_test;
```

2. Set environment variables for test database. The integration tests use the same environment variables as the main application:
```env
# Test database configuration (same as main app, or use a separate test database)
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=password
DB_NAME=japanesestudent_test
```

**Note:** Integration tests use the `internal/config` package to load configuration, which reads from `.env` file or environment variables.

**Run Integration Tests:**

Run all integration tests:
```bash
go test ./test/integration/... -v
```

Run integration tests with coverage:
```bash
go test ./test/integration/... -v -cover
```

Run specific integration test:
```bash
go test ./test/integration/... -v -run TestName
```

**Test Behavior:**
- Tests automatically create and drop tables before/after each test run
- Test data is seeded before each test and cleaned up afterward
- Each test runs in isolation with fresh data

### Running All Tests

Run all tests (unit + integration):
```bash
go test ./...
```

## Test Coverage

The test suite aims for comprehensive coverage:

### Repository Tests Coverage:
- ✅ `GetAll` - all alphabet types (hr, kt) and locales (en, ru)
  - Correct field selection based on alphabet type and locale
  - SQL query construction and execution
  - Result mapping to CharacterResponse models
- ✅ `GetByRowColumn` - vowel and consonant filtering
  - Filtering by vowel characters
  - Filtering by consonant characters
  - Correct field selection and result mapping
- ✅ `GetByID` - with different locales
  - Character retrieval by ID
  - Locale-based reading field selection
  - Not found scenarios
- ✅ `GetRandomForReadingTest` - random character selection with wrong options
  - Returns correct number of items (10 by default)
  - Generates 2 wrong options per correct character
  - Random selection logic
  - Shuffling of options
- ✅ `GetRandomForWritingTest` - random character selection
  - Returns correct number of items (10 by default)
  - Random character selection
- ✅ Error handling
  - Database connection errors
  - SQL query errors
  - Row scan errors
  - Context cancellation
- ✅ Input validation
  - Invalid alphabet types
  - Invalid locales
  - Invalid character parameters
- ✅ Edge cases
  - Empty result sets
  - Character not found (GetByID)
  - Insufficient characters for test generation

### Service Tests Coverage:
- ✅ `GetAll` - validation and repository integration
  - Valid alphabet types (hr, kt) and locales (en, ru)
  - Invalid alphabet types and locales
  - Repository error handling
- ✅ `GetByRowColumn` - validation and repository integration
  - Valid parameters with various characters
  - Missing character parameter
  - Invalid alphabet types and locales
- ✅ `GetByID` - validation and repository integration
  - Valid IDs with different locales
  - Invalid IDs (zero, negative)
  - Invalid locales
  - Repository error handling
- ✅ `GetReadingTest` - validation and repository integration
  - Valid alphabet types (hiragana, katakana) and locales
  - Invalid alphabet types (wrong URL path values)
  - Invalid locales
  - Returns 10 test items (testCount constant)
- ✅ `GetWritingTest` - validation and repository integration
  - Valid alphabet types and locales
  - Invalid alphabet types
  - Invalid locales
  - Returns 10 test items (testCount constant)
- ✅ `validateAlphabetType` - all cases (hr, kt, invalid)
- ✅ `validateLocale` - all cases (en, ru, invalid)
- ✅ Error handling and propagation from repository layer

### Integration Tests Coverage:
- ✅ All API endpoints end-to-end
  - `GET /api/v1/characters` - with various type and locale combinations
  - `GET /api/v1/characters/row-column` - with consonant and vowel filtering
  - `GET /api/v1/characters/{id}` - with different locales
  - `GET /api/v1/tests/{hiragana|katakana}/reading` - reading test generation
  - `GET /api/v1/tests/{hiragana|katakana}/writing` - writing test generation
- ✅ Repository layer with real database
- ✅ Service layer with real database
- ✅ Handler layer with HTTP requests
- ✅ Error scenarios (invalid inputs, not found, validation errors)
- ✅ Success scenarios with data validation
- ✅ Benchmark tests for performance measurement

## Test Data

Integration tests automatically seed test data before each test and clean up afterward. The test data includes:
- Vowel characters (a, i, u, e, o) in both hiragana and katakana
- Consonant-vowel combinations including:
  - K-row: ka, ki, ku, ke, ko
  - S-row: sa, si, su, se, so
  - And other common combinations
- Both English and Russian readings for all characters
- Full character data with consonant, vowel, hiragana, katakana, and both reading types

## Continuous Integration

For CI/CD pipelines:

1. **Unit Tests** (always run):
```bash
go test ./internal/... -cover -race
```

2. **Integration Tests** (optional, requires database):
```bash
go test ./test/integration/... -v
```

## Troubleshooting

### Integration Tests Failing

1. **Database Connection Error:**
   - Verify MySQL/MariaDB is running
   - Check database credentials in .env
   - Ensure test database exists

2. **Table Creation Error:**
   - Check database user has CREATE TABLE permissions
   - Verify database charset supports UTF-8 (utf8mb4)

3. **Test Data Issues:**
   - Tests automatically clean up data
   - If tests fail mid-run, manually clean: `DELETE FROM characters;`

### Unit Tests Failing

1. **Import Errors:**
   - Run `go mod tidy` to update dependencies
   - Verify `sqlmock` and `testify` are installed

2. **Mock Expectations:**
   - Check that all expected SQL queries are properly mocked
   - Verify query strings match exactly (including whitespace)

## Dependencies

Test dependencies (automatically added to `go.mod`):
- `github.com/DATA-DOG/go-sqlmock` - SQL mocking for unit tests
- `github.com/stretchr/testify` - Assertions and test utilities

