# JapaneseStudent

JapaneseStudent is a Go microservices-based application to help people learn hiragana/katakana alphabet, words and attend lessons.

## Tech Stack

- **Backend**: Go 1.22+
- **Router**: go-chi/chi
- **Database**: MariaDB
- **Logging**: zap (uber-go/zap)
- **API Documentation**: Swagger/OpenAPI
- **Authentication**: JWT (JSON Web Tokens)
- **Password Hashing**: bcrypt

## Architecture

The project follows a **microservices architecture** with the following services:

1. **auth-service** - User authentication and authorization (Port: 8081)
2. **learn-service** - Character learning, word dictionary, and test management (Port: 8080)

Both services share common libraries in the `libs/` directory for:
- Authentication middleware and JWT token generation
- Configuration management
- Logging
- HTTP handlers and middleware

## Project Structure

```
JapaneseStudent/
├── libs/                          # Shared libraries
│   ├── auth/                      # Authentication library
│   │   ├── middleware/            # JWT authentication middleware
│   │   └── service/               # JWT token generation and validation
│   ├── config/                    # Configuration management
│   ├── handlers/                  # Base HTTP handler
│   ├── logger/                    # Structured logging
│   │   └── middleware/           # Request logging middleware
│   └── middlewares/               # Shared HTTP middleware
│       ├── cors.go                # CORS middleware
│       ├── recovery.go            # Panic recovery middleware
│       ├── request_id.go          # Request ID middleware
│       └── request_size.go       # Request size limit middleware
├── services/
│   ├── auth-service/              # Authentication microservice
│   │   ├── cmd/
│   │   │   └── main.go            # Service entry point
│   │   ├── internal/
│   │   │   ├── handlers/          # HTTP handlers
│   │   │   │   └── auth_handler.go
│   │   │   ├── models/           # Domain models
│   │   │   │   ├── user.go
│   │   │   │   └── user_token.go
│   │   │   ├── repositories/      # Data access layer
│   │   │   │   ├── user_repository.go
│   │   │   │   ├── user_repository_test.go
│   │   │   │   ├── user_token_repository.go
│   │   │   │   └── user_token_repository_test.go
│   │   │   └── services/          # Business logic layer
│   │   │       ├── auth_service.go
│   │   │       └── auth_service_test.go
│   │   ├── migrations/            # Database migrations
│   │   │   ├── 000001_create_users_table.up.sql
│   │   │   ├── 000001_create_users_table.down.sql
│   │   │   ├── 000002_create_user_tokens_table.up.sql
│   │   │   └── 000002_create_user_tokens_table.down.sql
│   │   ├── test/
│   │   │   └── integration/       # Integration tests
│   │   │       └── auth_test.go
│   │   ├── Dockerfile
│   │   ├── go.mod
│   │   └── go.sum
│   └── learn-service/             # Learning microservice
│       ├── cmd/
│       │   └── main.go            # Service entry point
│       ├── internal/
│       │   ├── handlers/          # HTTP handlers
│       │   │   ├── character_handler.go
│       │   │   ├── dictionary_handler.go
│       │   │   └── test_result_handler.go
│       │   ├── models/           # Domain models
│       │   │   ├── character.go
│       │   │   ├── character_learn_history.go
│       │   │   ├── dictionary_history.go
│       │   │   └── word.go
│       │   ├── repositories/      # Data access layer
│       │   │   ├── character_repository.go
│       │   │   ├── character_learn_history_repository.go
│       │   │   ├── dictionary_history_repository.go
│       │   │   ├── word_repository.go
│       │   │   └── repository_test.go
│       │   └── services/         # Business logic layer
│       │       ├── character_service.go
│       │       ├── dictionary_service.go
│       │       ├── test_result_service.go
│       │       └── service_test.go
│       ├── migrations/            # Database migrations
│       │   ├── 000001_create_characters_table.up.sql
│       │   ├── 000001_create_characters_table.down.sql
│       │   ├── 000002_create_character_learn_history_table.up.sql
│       │   ├── 000002_create_character_learn_history_table.down.sql
│       │   ├── 000003_create_words_table.up.sql
│       │   ├── 000003_create_words_table.down.sql
│       │   ├── 000004_create_dictionary_history_table.up.sql
│       │   └── 000004_create_dictionary_history_table.down.sql
│       ├── test/
│       │   └── integration/      # Integration tests
│       │       └── characters_test.go
│       ├── Dockerfile
│       ├── go.mod
│       └── go.sum
├── docker-compose.yml             # Docker Compose configuration
├── TESTING.md                     # Comprehensive testing guide
└── README.md                      # This file
```

## Prerequisites

- Go 1.22 or higher
- MariaDB 10.11 or higher
- Docker and Docker Compose (for containerized deployment)

## Environment Variables

All services can use one `.env` file, create one for each other or configure environment variables in docker containers. Create `.env` files based on the following template:

### Common Configuration (both services)

```env
# Database Configuration
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=password
DB_NAME=japanesestudent

# Server Configuration
SERVER_PORT=8081                  # For auth-service default is 8081, for learn-service - 8080

# Logging
LOG_LEVEL=info

# CORS Configuration
# Comma-separated list of allowed origins (e.g., "http://localhost:3000,https://example.com")
# Use "*" to allow all origins (default if not specified, useful for development)
CORS_ALLOWED_ORIGINS=*

# JWT Configuration
# Secret key for signing JWT tokens (must be at least 32 characters for security)
JWT_SECRET=your-secret-key-here-change-in-production-minimum-32-characters
# Access token expiry duration (e.g., "1h", "30m", "2h")
JWT_ACCESS_TOKEN_EXPIRY=1h
# Refresh token expiry duration (e.g., "168h" for 7 days, "720h" for 30 days)
JWT_REFRESH_TOKEN_EXPIRY=168h
```

## Local Development

### 1. Install Dependencies

For each service:

```bash
cd services/auth-service && go mod download
cd ../learn-service && go mod download
```

### 2. Set Up Database

Start MariaDB using Docker Compose:

```bash
docker-compose up -d mariadb
```

Or use your own MariaDB instance and update the `.env` files accordingly.

Create the required databases:

```sql
CREATE DATABASE japanesestudent_auth;
CREATE DATABASE japanesestudent_learn;
```
or one database for both services:
```sql
CREATE DATABASE japanesestudent;
```

### 3. Run Migrations

Migrations run automatically when each service starts. Alternatively, you can run them manually using the migrate tool:

```bash
# For auth-service
migrate -path services/auth-service/migrations -database "mysql://user:password@tcp(localhost:3306)/japanesestudent" up

# For learn-service
migrate -path services/learn-service/migrations -database "mysql://user:password@tcp(localhost:3306)/japanesestudent" up
```

### 4. Run the Services

#### Run auth-service:

```bash
cd services/auth-service
go run cmd/main.go
```

The auth API will be available at `http://localhost:8081`

#### Run learn-service:

```bash
cd services/learn-service
go run cmd/main.go
```

The learn API will be available at `http://localhost:8080`

## Docker Deployment

### Build and Run with Docker Compose

```bash
# Build and start all services
docker-compose up -d

# View logs
docker-compose logs -f auth-service
docker-compose logs -f learn-service

# Stop services
docker-compose down

# Stop and remove volumes
docker-compose down -v
```

## API Endpoints

### Authentication Service (Port 8081)

#### Authentication
- `POST /api/v1/auth/register` - Register a new user
  - Body: `{ "email": "user@example.com", "username": "username", "password": "Password123!" }`
  - Returns: Access and refresh tokens as HTTP-only cookies
- `POST /api/v1/auth/login` - Login with email/username and password
  - Body: `{ "login": "user@example.com", "password": "Password123!" }`
  - Returns: Access and refresh tokens as HTTP-only cookies
- `POST /api/v1/auth/refresh` - Refresh access token using refresh token
  - Body: `{ "refreshToken": "token" }` (or cookie)
  - Returns: New access and refresh tokens

### Learning Service (Port 8080)

#### Characters
- `GET /api/v1/characters?type={hr|kt}&locale={en|ru}` - Get all characters
- `GET /api/v1/characters/row-column?type={hr|kt}&locale={en|ru}&character={char}` - Get characters by row/column
- `GET /api/v1/characters/{id}?locale={en|ru}` - Get character by ID

#### Tests (Requires Authentication)
- `GET /api/v1/tests/{hiragana|katakana}/reading?locale={en|ru}` - Get reading test (10 random characters with options)
- `GET /api/v1/tests/{hiragana|katakana}/writing?locale={en|ru}` - Get writing test (10 random characters)

#### Test Results (Requires Authentication)
- `POST /api/v1/test-results/{hiragana|katakana}/{reading|writing|listening}` - Submit test results
  - Body: `{ "results": [{ "characterId": 1, "passed": true }, ...] }`
- `GET /api/v1/test-results/history` - Get user's learning history

#### Dictionary / Words (Requires Authentication)
- `GET /api/v1/words?newCount={10-40}&oldCount={10-40}&locale={en|ru|de}` - Get word list
  - Query Parameters:
    - `newCount` (optional): Number of new words to return (10-40, default: 20)
    - `oldCount` (optional): Number of old words to return (10-40, default: 20)
    - `locale` (optional): Translation locale - `en`, `ru`, or `de` (default: `en`)
  - Returns: Mixed list of new and old words with locale-specific translations
- `POST /api/v1/words/results` - Submit word learning results
  - Body: `{ "results": [{ "wordId": 1, "period": 7 }, ...] }`
  - `period`: Days until next appearance (1-30)
  - Returns: 204 No Content on success

### API Documentation

#### Auth Service
- Swagger UI: `http://localhost:8081/swagger/index.html`
- Swagger JSON: `http://localhost:8081/swagger/doc.json`

#### Learn Service
- Swagger UI: `http://localhost:8080/swagger/index.html`
- Swagger JSON: `http://localhost:8080/swagger/doc.json`

## Testing

For comprehensive testing information, see [TESTING.md](./TESTING.md).

### Quick Start

#### Run Unit Tests

```bash
# Run all unit tests
go test ./services/.../internal/...

# Run with coverage
go test ./services/.../internal/... -cover
```

#### Run Integration Tests

**Prerequisites:** Test databases must be configured. See TESTING.md for detailed setup instructions.

```bash
# Run all integration tests
go test ./services/.../test/integration/... -v

# Run specific service integration tests
go test ./services/auth-service/test/integration/... -v
go test ./services/learn-service/test/integration/... -v
```

## Database Schema

### Auth Service Database

#### Users Table
- `id` - Primary key (auto-increment)
- `email` - Email address (unique, indexed)
- `username` - Username (unique, indexed)
- `password_hash` - Bcrypt hashed password
- `role` - User role (default: 'user')

#### User Tokens Table
- `id` - Primary key (auto-increment)
- `user_id` - Foreign key to users.id
- `token` - Refresh token (unique, indexed)
- `updated_at` - Timestamp

### Learn Service Database

#### Characters Table
- `id` - Primary key
- `consonant` - Consonant character
- `vowel` - Vowel character
- `english_reading` - English reading
- `russian_reading` - Russian reading
- `katakana` - Katakana character
- `hiragana` - Hiragana character

#### Character Learn History Table
- `id` - Primary key (auto-increment)
- `user_id` - User ID (from auth service)
- `character_id` - Foreign key to characters.id
- `hiragana_reading_result` - Test result (0.0-1.0)
- `hiragana_writing_result` - Test result (0.0-1.0)
- `hiragana_listening_result` - Test result (0.0-1.0)
- `katakana_reading_result` - Test result (0.0-1.0)
- `katakana_writing_result` - Test result (0.0-1.0)
- `katakana_listening_result` - Test result (0.0-1.0)
- Unique constraint on (user_id, character_id)

#### Words Table
- `id` - Primary key (auto-increment)
- `word` - Japanese word (Kanji, indexed)
- `phonetic_clues` - Hiragana reading (phonetic clues)
- `russian_translation` - Russian translation
- `english_translation` - English translation
- `german_translation` - German translation
- `example` - Japanese example sentence
- `example_russian_translation` - Russian translation of example
- `example_english_translation` - English translation of example
- `example_german_translation` - German translation of example
- `easy_period` - Days until next appearance for easy difficulty
- `normal_period` - Days until next appearance for normal difficulty
- `hard_period` - Days until next appearance for hard difficulty
- `extra_hard_period` - Days until next appearance for extra hard difficulty

#### Dictionary History Table
- `id` - Primary key (auto-increment)
- `word_id` - Foreign key to words.id
- `user_id` - User ID (from auth service)
- `next_appearance` - Date when the word should appear again (indexed)
- Unique constraint on (user_id, word_id)
- Foreign key constraint on word_id with CASCADE delete

## Middleware

Both services include the following middleware (in order of execution):

1. **Request ID** (`libs/middlewares/request_id.go`): Generates unique UUID for each request and adds it to the context
2. **Recovery** (`libs/middlewares/recovery.go`): Panic recovery and error handling with proper logging
3. **Request Size Limit** (`libs/middlewares/request_size.go`): Limits request body size to 10MB
4. **CORS** (`libs/middlewares/cors.go`): Cross-origin resource sharing support with configurable allowed origins
5. **Logging** (`libs/logger/middleware/logger.go`): Structured JSON logging with request details, method, path, status, duration, and request ID
6. **Rate Limiting**: Rate limiter using go-chi/httprate (100 requests/minute per IP) - configured at router level
7. **Authentication** (`libs/auth/middleware/auth.go`): JWT token validation for protected routes

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

## Authentication

The application uses JWT (JSON Web Tokens) for authentication:

- **Access Tokens**: Short-lived tokens (default: 1 hour) used for API authentication
- **Refresh Tokens**: Long-lived tokens (default: 7 days) stored in database, used to obtain new access tokens
- **Token Storage**: Refresh tokens are stored as HTTP-only cookies for security
- **Password Security**: Passwords are hashed using bcrypt before storage

### Password Requirements

- Minimum 8 characters
- At least one lowercase letter
- At least one uppercase letter
- At least one number
- At least one special character from: `!_?^&+-=|`

## Logging

The application uses structured JSON logging with the following levels:
- `info` - General information
- `warn` - Warnings
- `error` - Errors

All logs include request ID for tracing across services.

## License

[Add your license here]
