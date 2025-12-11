# JapaneseStudent

JapaneseStudent is a Go microservices-based application to help people learn hiragana/katakana alphabet, words and attend lessons.

## Architecture

This project uses a microservices architecture. Currently, it includes:
- **api-server**: Handles hiragana and katakana character operations

## Tech Stack

- **Backend**: Go 1.22+
- **Router**: go-chi/chi
- **Database**: MariaDB
- **Logging**: zap (uber-go/zap)
- **API Documentation**: Swagger/OpenAPI

## Project Structure

```
JapaneseStudent/
├── internal/
│   ├── config/                # Configuration management
│   ├── handlers/              # HTTP handlers
│   │   ├── base_handler.go    # Base handler with common functionality
│   │   └── character_handler.go # Character endpoints handler
│   ├── logger/                # Logger package (internal, shared)
│   ├── middleware/            # HTTP middleware
│   │   ├── cors.go            # CORS middleware
│   │   ├── logger.go          # Request logging middleware
│   │   ├── recovery.go        # Panic recovery middleware
│   │   ├── request_id.go      # Request ID middleware
│   │   └── request_size.go    # Request size limit middleware
│   ├── models/                # Domain models
│   │   └── character.go       # Character model definitions
│   ├── repositories/          # Data access layer
│   │   ├── character_repository.go # Character repository implementation
│   │   └── repository_test.go # Repository unit tests
│   └── services/              # Business logic layer
│       ├── character_service.go # Character service implementation
│       └── service_test.go    # Service unit tests
├── migrations/                # Database migrations
│   └── 000001_create_characters_table.up.sql
├── test/                      # Integration tests
│   └── integration/
│       └── characters_test.go # Character endpoints integration tests
├── configs/                  # Configuration files
│   └── .env.example          # Environment variables example
├── docker-compose.yml        # Docker Compose configuration
├── Dockerfile                # Docker image definition
├── go.mod                    # Go module dependencies
└── go.sum                    # Go module checksums
```

## Prerequisites

- Go 1.22 or higher
- MariaDB 10.11 or higher
- Docker and Docker Compose (for containerized deployment)

## Environment Variables

Create a `.env` file in the root directory based on `configs/.env.example`:

```env
# Database Configuration
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=password
DB_NAME=japanesestudent

# Server Configuration
SERVER_PORT=8080

# Logging
LOG_LEVEL=info

# CORS Configuration
# Comma-separated list of allowed origins (e.g., "http://localhost:3000,https://example.com")
# Use "*" to allow all origins (default if not specified, useful for development)
CORS_ALLOWED_ORIGINS=*
```

## Local Development

### 1. Install Dependencies

```bash
go mod download
```

### 2. Set Up Database

Start MariaDB using Docker Compose:

```bash
docker-compose up -d mariadb
```

Or use your own MariaDB instance and update the `.env` file accordingly.

### 3. Run Migrations

Migrations run automatically when the application starts. Alternatively, you can run them manually using the migrate tool:

```bash
migrate -path migrations -database "mysql://user:password@tcp(localhost:3306)/japanesestudent" up
```

### 4. Run the Application

Create a main.go file in the root directory or run your application entry point:

```bash
go run main.go
```

The API will be available at `http://localhost:8080`

## Docker Deployment

### Build and Run with Docker Compose

```bash
# Build and start all services
docker-compose up -d

# View logs
docker-compose logs -f api-server

# Stop services
docker-compose down

# Stop and remove volumes
docker-compose down -v
```

## API Endpoints

### Characters

- `GET /api/v1/characters?type={hr|kt}&locale={en|ru}` - Get all characters
- `GET /api/v1/characters/row-column?type={hr|kt}&locale={en|ru}&character={char}` - Get characters by row/column
- `GET /api/v1/characters/{id}?locale={en|ru}` - Get character by ID

### Tests (not unit or integration, but user tests for alphabet knowledge)

- `GET /api/v1/tests/{hiragana|katakana}/reading?locale={en|ru}` - Get reading test (10 random characters)
- `GET /api/v1/tests/{hiragana|katakana}/writing?locale={en|ru}` - Get writing test (10 random characters with options)

### API Documentation

- Swagger UI: `http://localhost:8080/swagger/index.html`
- Swagger JSON: `http://localhost:8080/swagger/doc.json`

## API Examples

### Get all hiragana characters (English)

```bash
curl "http://localhost:8080/api/v1/characters?type=hr&locale=en"
```

### Get katakana character by ID (Russian)

```bash
curl "http://localhost:8080/api/v1/characters/1?locale=ru"
```

### Get reading test (hiragana, English)

Returns 10 random characters for reading practice:

```bash
curl "http://localhost:8080/api/v1/tests/hiragana/reading?locale=en"
```

### Get writing test (katakana, Russian)

Returns 10 random characters with multiple choice options:

```bash
curl "http://localhost:8080/api/v1/tests/katakana/writing?locale=ru"
```

## Testing

### Run Unit Tests

```bash
go test ./internal/...
```

### Run Tests with Coverage

```bash
go test -cover ./internal/...
```

### Run Integration Tests

**Prerequisites:** A test database must be configured. See TESTING.md for detailed setup instructions.

```bash
go test ./test/integration/... -v
```

## Database Schema

### Characters Table

- `id` - Primary key
- `consonant` - Consonant character
- `vowel` - Vowel character
- `english_reading` - English reading
- `russian_reading` - Russian reading
- `katakana` - Katakana character
- `hiragana` - Hiragana character

## Middleware

The application includes the following middleware (in order of execution):

1. **Request ID** (`internal/middleware/request_id.go`): Generates unique UUID for each request and adds it to the context
2. **Recovery** (`internal/middleware/recovery.go`): Panic recovery and error handling with proper logging
3. **Request Size Limit** (`internal/middleware/request_size.go`): Limits request body size to 10MB
4. **CORS** (`internal/middleware/cors.go`): Cross-origin resource sharing support with configurable allowed origins
5. **Logging** (`internal/middleware/logger.go`): Structured JSON logging with request details, method, path, status, duration, and request ID
6. **Rate Limiting**: Rate limiter using go-chi/httprate (100 requests/minute per IP) - configured at router level

## CORS Configuration

The CORS middleware supports configurable allowed origins via the `CORS_ALLOWED_ORIGINS` environment variable.

### Configuration Options

- **Single Origin**: `CORS_ALLOWED_ORIGINS=http://localhost:3000`
- **Multiple Origins**: `CORS_ALLOWED_ORIGINS=http://localhost:3000,https://example.com,https://app.example.com`
- **Allow All Origins** (Development): `CORS_ALLOWED_ORIGINS=*` or omit the variable (defaults to `*`)

### CORS Headers

The middleware sets the following headers:
- `Access-Control-Allow-Origin`: Set to the request origin if allowed, or `*` if all origins are allowed
- `Access-Control-Allow-Methods`: `GET, POST, PUT, DELETE, OPTIONS`
- `Access-Control-Allow-Headers`: `Content-Type, Authorization, X-Request-ID`
- `Access-Control-Allow-Credentials`: `true` (when using specific origins)
- `Access-Control-Max-Age`: `3600` seconds

## Logging

The application uses structured JSON logging with the following levels:
- `info` - General information
- `warn` - Warnings
- `error` - Errors

All logs include request ID for tracing.

## License

[Add your license here]
