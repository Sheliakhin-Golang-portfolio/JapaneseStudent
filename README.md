# JapaneseStudent

JapaneseStudent is a Go microservices-based application to help people learn hiragana/katakana alphabet, words and attend lessons.

## Tech Stack

- **Backend**: Go 1.22+
- **Router**: go-chi/chi
- **Database**: MariaDB
- **Task Queue**: Redis + Asynq (for task-service)
- **Logging**: zap (uber-go/zap)
- **API Documentation**: Swagger/OpenAPI
- **Authentication**: JWT (JSON Web Tokens)
- **Password Hashing**: bcrypt

## Architecture

The project follows a **microservices architecture** with the following services:

1. **auth-service** - User authentication, authorization, and profile management (Port: 8081)
2. **learn-service** - Character learning, word dictionary, and test management (Port: 8080)
3. **media-service** - Media file management (upload, download, metadata) (Port: 8082)
4. **task-service** - Task queue and scheduler for immediate and scheduled tasks (Port: 8083)

All services share common libraries in the `libs/` directory for:
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
│   │   │   │   ├── auth_handler.go
│   │   │   │   ├── admin_handler.go
│   │   │   │   ├── profile_handler.go
│   │   │   │   ├── user_settings_handler.go
│   │   │   │   └── token_cleaning_handler.go
│   │   │   ├── models/           # Domain models
│   │   │   │   ├── user.go
│   │   │   │   ├── user_token.go
│   │   │   │   └── user_settings.go
│   │   │   ├── repositories/      # Data access layer
│   │   │   │   ├── user_repository.go
│   │   │   │   ├── user_repository_test.go
│   │   │   │   ├── user_token_repository.go
│   │   │   │   ├── user_token_repository_test.go
│   │   │   │   ├── user_settings_repository.go
│   │   │   │   └── user_settings_repository_test.go
│   │   │   └── services/          # Business logic layer
│   │   │       ├── auth_service.go
│   │   │       ├── auth_service_test.go
│   │   │       ├── admin_service.go
│   │   │       ├── admin_service_test.go
│   │   │       ├── profile_service.go
│   │   │       ├── profile_service_test.go
│   │   │       ├── user_settings_service.go
│   │   │       └── user_settings_service_test.go
│   │   ├── migrations/            # Database migrations
│   │   │   ├── 000001_create_users_table.up.sql
│   │   │   ├── 000001_create_users_table.down.sql
│   │   │   ├── 000002_create_user_tokens_table.up.sql
│   │   │   ├── 000002_create_user_tokens_table.down.sql
│   │   │   ├── 000003_create_user_settings_table.up.sql
│   │   │   ├── 000003_create_user_settings_table.down.sql
│   │   │   ├── 000004_add_avatar_to_users_table.up.sql
│   │   │   └── 000004_add_avatar_to_users_table.down.sql
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
│       │   │   ├── test_result_handler.go
│       │   │   ├── admin_character_handler.go
│       │   │   ├── admin_word_handler.go
│       │   │   ├── user_lesson_handler.go
│       │   │   ├── tutor_lesson_handler.go
│       │   │   └── admin_lesson_handler.go
│       │   ├── models/           # Domain models
│       │   │   ├── character.go
│       │   │   ├── character_learn_history.go
│       │   │   ├── dictionary_history.go
│       │   │   ├── word.go
│       │   │   ├── course.go
│       │   │   ├── lesson.go
│       │   │   ├── lesson_block.go
│       │   │   ├── lesson_user_history.go
│       │   │   └── tutor_media.go
│       │   ├── repositories/      # Data access layer
│       │   │   ├── character_repository.go
│       │   │   ├── character_learn_history_repository.go
│       │   │   ├── dictionary_history_repository.go
│       │   │   ├── word_repository.go
│       │   │   ├── course_repository.go
│       │   │   ├── course_repository_test.go
│       │   │   ├── lesson_repository.go
│       │   │   ├── lesson_repository_test.go
│       │   │   ├── lesson_block_repository.go
│       │   │   ├── lesson_block_repository_test.go
│       │   │   ├── lesson_user_history_repository.go
│       │   │   ├── lesson_user_history_repository_test.go
│       │   │   ├── tutor_media_repository.go
│       │   │   └── repository_test.go
│       │   └── services/         # Business logic layer
│       │       ├── character_service.go
│       │       ├── dictionary_service.go
│       │       ├── test_result_service.go
│       │       ├── service_test.go
│       │       ├── admin_character_service.go
│       │       ├── admin_character_service_test.go
│       │       ├── admin_word_service.go
│       │       ├── admin_word_service_test.go
│       │       ├── user_lesson_service.go
│       │       ├── user_lesson_service_test.go
│       │       ├── tutor_lesson_service.go
│       │       └── tutor_lesson_service_test.go
│       ├── migrations/            # Database migrations
│       │   ├── 000001_create_characters_table.up.sql
│       │   ├── 000001_create_characters_table.down.sql
│       │   ├── 000002_create_character_learn_history_table.up.sql
│       │   ├── 000002_create_character_learn_history_table.down.sql
│       │   ├── 000003_create_words_table.up.sql
│       │   ├── 000003_create_words_table.down.sql
│       │   ├── 000004_create_dictionary_history_table.up.sql
│       │   ├── 000004_create_dictionary_history_table.down.sql
│       │   ├── 000005_add_audio_to_characters_table.up.sql
│       │   ├── 000005_add_audio_to_characters_table.down.sql
│       │   ├── 000006_add_audio_to_words_table.up.sql
│       │   ├── 000006_add_audio_to_words_table.down.sql
│       │   ├── 000007_create_courses_table.up.sql
│       │   ├── 000007_create_courses_table.down.sql
│       │   ├── 000008_create_lessons_table.up.sql
│       │   ├── 000008_create_lessons_table.down.sql
│       │   ├── 000009_create_lesson_blocks_table.up.sql
│       │   ├── 000009_create_lesson_blocks_table.down.sql
│       │   ├── 000010_create_lesson_user_history_table.up.sql
│       │   ├── 000010_create_lesson_user_history_table.down.sql
│       │   ├── 000011_create_tutor_media_table.up.sql
│       │   └── 000011_create_tutor_media_table.down.sql
│       ├── test/
│       │   └── integration/      # Integration tests
│       │       └── characters_test.go
│       ├── Dockerfile
│       ├── go.mod
│       └── go.sum
│   └── media-service/             # Media management microservice
│       ├── cmd/
│       │   └── main.go            # Service entry point
│       ├── internal/
│       │   ├── handlers/          # HTTP handlers
│       │   │   └── media_handler.go
│       │   ├── models/           # Domain models
│       │   │   └── metadata.go
│       │   ├── repositories/      # Data access layer
│       │   │   ├── metadata_repository.go
│       │   │   └── metadata_repository_test.go
│       │   ├── services/         # Business logic layer
│       │   │   ├── media_service.go
│       │   │   └── media_service_test.go
│       │   └── storage/          # File storage
│       │       ├── storage.go
│       │       └── utils.go
│       ├── migrations/            # Database migrations
│       │   ├── 000001_create_metadata_table.up.sql
│       │   └── 000001_create_metadata_table.down.sql
│       ├── Dockerfile
│       ├── go.mod
│       └── go.sum
│   └── task-service/             # Task queue and scheduler microservice
│       ├── cmd/
│       │   ├── api/
│       │   │   └── main.go        # API service entry point
│       │   ├── scheduler/
│       │   │   └── main.go        # Scheduler service entry point
│       │   └── worker/
│       │       └── main.go        # Worker service entry point
│       ├── internal/
│       │   ├── handlers/          # HTTP handlers
│       │   │   ├── task_handler.go
│       │   │   └── admin_handler.go
│       │   ├── models/           # Domain models
│       │   │   ├── email_template.go
│       │   │   ├── immediate_task.go
│       │   │   ├── scheduled_task.go
│       │   │   └── scheduled_task_log.go
│       │   ├── repositories/      # Data access layer
│       │   │   ├── email_template_repository.go
│       │   │   ├── email_template_repository_test.go
│       │   │   ├── immediate_task_repository.go
│       │   │   ├── immediate_task_repository_test.go
│       │   │   ├── scheduled_task_repository.go
│       │   │   ├── scheduled_task_repository_test.go
│       │   │   ├── scheduled_task_log_repository.go
│       │   │   └── scheduled_task_log_repository_test.go
│       │   └── services/         # Business logic layer
│       │       ├── email_template_service.go
│       │       ├── immediate_task_service.go
│       │       ├── scheduled_task_service.go
│       │       └── task_log_service.go
│       ├── migrations/            # Database migrations
│       │   ├── 000001_create_email_templates_table.up.sql
│       │   ├── 000001_create_email_templates_table.down.sql
│       │   ├── 000002_create_immediate_tasks_table.up.sql
│       │   ├── 000002_create_immediate_tasks_table.down.sql
│       │   ├── 000003_create_scheduled_tasks_table.up.sql
│       │   ├── 000003_create_scheduled_tasks_table.down.sql
│       │   ├── 000004_create_scheduled_task_logs_table.up.sql
│       │   └── 000004_create_scheduled_task_logs_table.down.sql
│       ├── test/
│       │   └── integration/      # Integration tests
│       │       └── task_service_test.go
│       ├── Dockerfile
│       ├── go.mod
│       └── go.sum
├── docker-compose.yml             # Docker Compose configuration
├── TESTING.md                     # Comprehensive testing guide
└── README.md                      # This file
```

## Prerequisites

- Go 1.24 or higher
- MariaDB 10.11 or higher
- Redis 6.0 or higher (required for task-service)
- Docker and Docker Compose (for containerized deployment)

## Environment Variables

All services can use one `.env` file, create one for each other or configure environment variables in docker containers. Create `.env` files based on the following template:

### Common Configuration (all services)

```env
# Database Configuration
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=password
DB_NAME=japanesestudent

# Server Configuration
SERVER_PORT=8081                  # For auth-service default is 8081, for learn-service - 8080, for media-service - 8082, for task-service - 8083

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

# Media Service Configuration
# Base path for storing media files (required for media-service)
MEDIA_BASE_PATH=/path/to/media/storage
# API key for service-to-service authentication (required for media-service upload/delete endpoints and auth-service avatar operations)
API_KEY=your-api-key-here
# Base URL for generating download URLs (optional, defaults to http://localhost:{PORT})
BASE_URL=http://localhost:8082
# Media service base URL (required for auth-service avatar upload/delete functionality)
MEDIA_BASE_URL=http://localhost:8082

# Auth Service Configuration (required for auth-service email verification, profile updates, and token cleaning)
# Verification URL for email verification links (required for registration flow and email updates)
VERIFICATION_URL=http://localhost:8081/api/v6/auth/verify-email
# Task service base URL for creating immediate tasks (required for sending verification emails and profile email updates)
# Can use either IMMEDIATE_TASK_BASE_URL or TASK_BASE_URL (TASK_BASE_URL is used as fallback)
IMMEDIATE_TASK_BASE_URL=http://localhost:8083/api/v6/tasks/immediate
TASK_BASE_URL=http://localhost:8083/api/v6/tasks/immediate
# Task service base URL for creating scheduled tasks (required for token cleaning task scheduling)
SCHEDULED_TASK_BASE_URL=http://localhost:8083/api/v6/tasks/scheduled

# Task Service Configuration (required for task-service)
# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=                    # Optional, leave empty if no password
REDIS_DB=0                         # Redis database number (default: 0)
```

## Local Development

### 1. Install Dependencies

For each service:

```bash
cd services/auth-service && go mod download
cd ../learn-service && go mod download
cd ../media-service && go mod download
cd ../task-service && go mod download
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
CREATE DATABASE japanesestudent_media;
CREATE DATABASE japanesestudent_task;
```
or one database for all services:
```sql
CREATE DATABASE japanesestudent;
```

**Note**: task-service also requires Redis to be running. Start Redis using Docker Compose:
```bash
docker-compose up -d redis
```

### 3. Run Migrations

Migrations run automatically when each service starts. Alternatively, you can run them manually using the migrate tool:

```bash
# For auth-service
migrate -path services/auth-service/migrations -database "mysql://user:password@tcp(localhost:3306)/japanesestudent" up

# For learn-service
migrate -path services/learn-service/migrations -database "mysql://user:password@tcp(localhost:3306)/japanesestudent" up

# For media-service
migrate -path services/media-service/migrations -database "mysql://user:password@tcp(localhost:3306)/japanesestudent" up

# For task-service
migrate -path services/task-service/migrations -database "mysql://user:password@tcp(localhost:3306)/japanesestudent" up
```

#### Migration History

**Auth Service Migrations:**
1. `000001_create_users_table` - Creates users table with email, username, password_hash, and role
2. `000002_create_user_tokens_table` - Creates user_tokens table for refresh token storage with `created_at` timestamp for expiration tracking (used for token cleaning)
3. `000003_create_user_settings_table` - Creates user_settings table for user preferences
4. `000004_add_avatar_to_users_table` - Adds avatar column to users table (VARCHAR(500), nullable)
5. `000005_add_active_to_users_table` - Adds active column to users table (BOOLEAN, default: FALSE) for email verification

**Learn Service Migrations:**
1. `000001_create_characters_table` - Creates characters table for hiragana/katakana characters
2. `000002_create_character_learn_history_table` - Creates character_learn_history table for tracking user progress
3. `000003_create_words_table` - Creates words table for Japanese vocabulary
4. `000004_create_dictionary_history_table` - Creates dictionary_history table for spaced repetition
5. `000005_add_audio_to_characters_table` - Adds audio column to characters table
6. `000006_add_audio_to_words_table` - Adds word_audio and word_example_audio columns to words table
7. `000007_create_courses_table` - Creates courses table for course management
8. `000008_create_lessons_table` - Creates lessons table for lesson management
9. `000009_create_lesson_blocks_table` - Creates lesson_blocks table for lesson content blocks
10. `000010_create_lesson_user_history_table` - Creates lesson_user_history table for tracking lesson completion
11. `000011_create_tutor_media_table` - Creates tutor_media table for tutor media file management

**Media Service Migrations:**
1. `000001_create_metadata_table` - Creates metadata table for file metadata storage

**Task Service Migrations:**
1. `000001_create_email_templates_table` - Creates email_templates table for email template storage
2. `000002_create_immediate_tasks_table` - Creates immediate_tasks table for immediate task queue
3. `000003_create_scheduled_tasks_table` - Creates scheduled_tasks table for scheduled task management
4. `000004_create_scheduled_task_logs_table` - Creates scheduled_task_logs table for task execution logs

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

#### Run media-service:

```bash
cd services/media-service
go run cmd/main.go
```

The media API will be available at `http://localhost:8082`

#### Run task-service:

The task-service consists of three sub-services that need to be run separately:

**API Service:**
```bash
cd services/task-service
go run cmd/api/main.go
```

**Scheduler Service:**
```bash
cd services/task-service
go run cmd/scheduler/main.go
```

**Worker Service:**
```bash
cd services/task-service
go run cmd/worker/main.go
```

The task API will be available at `http://localhost:8083`

**Note**: 
- media-service requires `MEDIA_BASE_PATH` environment variable to be set for file storage.
- auth-service requires `MEDIA_BASE_URL` and `API_KEY` environment variables to be set for avatar upload/delete functionality.
- auth-service requires `IMMEDIATE_TASK_BASE_URL` (or `TASK_BASE_URL`) for sending verification emails and profile email updates, and `SCHEDULED_TASK_BASE_URL` with `API_KEY` for token cleaning task scheduling functionality.
- auth-service requires `VERIFICATION_URL` environment variable for email verification links.
- task-service requires Redis to be running and configured via `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD`, and `REDIS_DB` environment variables.

## Docker Deployment

### Build and Run with Docker Compose

```bash
# Build and start all services
docker-compose up -d

# View logs
docker-compose logs -f auth-service
docker-compose logs -f learn-service
docker-compose logs -f media-service
docker-compose logs -f task-service-api
docker-compose logs -f task-service-scheduler
docker-compose logs -f task-service-worker

# Stop services
docker-compose down

# Stop and remove volumes
docker-compose down -v
```

## API Endpoints

**Note**: All services now use API version v6 in their URL paths (e.g., `/api/v6/...`).

### Authentication Service (Port 8081)

#### Authentication
- `POST /api/v6/auth/register` - Register a new user
  - Content-Type: `multipart/form-data`
  - Form Fields:
    - `email` (required): User email address
    - `username` (required): Username
    - `password` (required): User password (must meet password requirements)
    - `avatar` (optional): Avatar image file
  - Behavior:
    - Creates user account with `active=false` (requires email verification)
    - Creates default user settings asynchronously
    - Uploads avatar to media-service if provided
    - Sends verification email via task-service using `register_template`
    - Returns 201 Created with success message
    - **Note**: User cannot login until email is verified. Tokens are only issued after email verification.
  - Returns: Success message (tokens are only issued after email verification via verify-email endpoint)
  - Errors:
    - 400 Bad Request: Invalid credentials, user already exists, or validation failed
    - 202 Accepted: User created but verification email could not be sent (temporary issue)
- `GET /api/v6/auth/verify-email` - Verify user email address
  - Query Parameters:
    - `validToken` (required): Verification token from the email link
  - Behavior:
    - Activates user account (`active=true`)
    - Issues access and refresh tokens
  - Returns: Access and refresh tokens as HTTP-only cookies
  - Errors:
    - 400 Bad Request: Invalid or expired token
    - 409 Conflict: Email already verified
- `POST /api/v6/auth/resend-verification` - Resend verification email
  - Body: `{ "email": "user@example.com" }`
  - Behavior:
    - Resends verification email to unverified users
  - Returns: Success message
  - Errors:
    - 400 Bad Request: Invalid request
    - 404 Not Found: User not found
    - 409 Conflict: Email already verified
- `POST /api/v6/auth/login` - Login with email/username and password
  - Body: `{ "login": "user@example.com", "password": "Password123!" }`
  - Behavior:
    - Validates credentials
    - Checks if user account is active (email verified)
    - Returns tokens only if account is verified
  - Returns: Access and refresh tokens as HTTP-only cookies
  - Errors:
    - 400 Bad Request: Invalid credentials or email not verified
- `POST /api/v6/auth/refresh` - Refresh access token using refresh token
  - Body: `{ "refreshToken": "token" }` (or cookie)
  - Behavior:
    - Validates refresh token
    - Checks if user account is active
  - Returns: New access and refresh tokens as HTTP-only cookies
  - Errors:
    - 400 Bad Request: Invalid token or email not verified

#### Profile Management (Requires Authentication)
- `GET /api/v6/profile` - Get user profile information
  - Returns: User profile including username, email, and avatar URL
- `PATCH /api/v6/profile` - Update user profile (username and/or email)
  - Body: `{ "username": "newusername", "email": "newemail@example.com" }`
  - At least one field (username or email) must be provided
  - Email updates require email verification (user account becomes inactive until verified)
  - Returns: 204 No Content on success
  - Note: Email updates trigger a verification email via task-service
- `PUT /api/v6/profile/avatar` - Update user avatar
  - Content-Type: `multipart/form-data`
  - Form Fields:
    - `avatar` (required): Avatar image file
  - Returns: Avatar URL
  - Note: Avatar file is uploaded to media-service and old avatar is automatically deleted
- `PUT /api/v6/profile/password` - Update user password
  - Body: `{ "password": "NewPassword123!" }`
  - Password must meet password requirements (see Password Requirements section)
  - Returns: 204 No Content on success

#### User Settings (Requires Authentication)
- `GET /api/v6/settings` - Get user settings for the authenticated user
  - Returns: User settings including word counts, alphabet learn count, and language preference
- `PATCH /api/v6/settings` - Update user settings
  - Body: `{ "newWordCount": 20, "oldWordCount": 20, "alphabetLearnCount": 10, "language": "en" }`
  - At least one field must be provided
  - Returns: 204 No Content on success
- `GET /api/v6/profile/settings` - Get user settings (via profile handler)
  - Returns: User settings including word counts, alphabet learn count, and language preference
- `PATCH /api/v6/profile/settings` - Update user settings (via profile handler)
  - Body: `{ "newWordCount": 20, "oldWordCount": 20, "alphabetLearnCount": 10, "language": "en" }`
  - At least one field must be provided
  - Returns: 204 No Content on success

#### Admin Endpoints (Requires Authentication & Admin Role)
- `GET /api/v6/admin/users` - Get paginated list of users
  - Query Parameters:
    - `page` (optional): Page number (default: 1)
    - `count` (optional): Items per page (default: 20)
    - `role` (optional): Filter by role (integer)
    - `search` (optional): Search in email or username
  - Returns: List of users with pagination
- `GET /api/v6/admin/users/{id}` - Get user with settings by ID
  - Returns: Full user information including settings
- `POST /api/v6/admin/users` - Create a new user with settings
  - Body: `{ "email": "user@example.com", "username": "username", "password": "Password123!", "role": 0 }`
  - Returns: Created user ID
- `POST /api/v6/admin/users/{id}/settings` - Create user settings for a user
  - Returns: Success message or "Settings already exist"
- `PATCH /api/v6/admin/users/{id}` - Update user and/or settings
  - Content-Type: `multipart/form-data`
  - Form Fields:
    - `username` (optional): New username
    - `email` (optional): New email
    - `role` (optional): New role (integer)
    - `newWordCount` (optional): New word count (10-40)
    - `oldWordCount` (optional): Old word count (10-40)
    - `alphabetLearnCount` (optional): Alphabet learn count (5-15)
    - `language` (optional): Language preference (`en`, `ru`, `de`)
    - `avatar` (optional): Avatar image file
  - Returns: 204 No Content on success
  - Note: Avatar upload integrates with media-service. Old avatar is automatically deleted when uploading a new one.
- `DELETE /api/v6/admin/users/{id}` - Delete a user by ID
  - Returns: 204 No Content on success (or 200 with message if avatar deletion fails)
  - Note: Automatically deletes user's avatar from media-service if present
- `GET /api/v6/admin/tutors` - Get list of tutors
  - Returns: List of tutors (users with role = 2) with only ID and username
  - Used for course/lesson assignment where tutor needs to be selected
- `POST /api/v6/admin/tasks/schedule-token-cleaning` - Schedule token cleaning task
  - Headers:
    - `Authorization`: Bearer token (admin role required)
  - Body: `{ "tokenCleaningURL": "http://localhost:8081/api/v6/tokens/clean" }`
  - Behavior:
    - Creates a scheduled task in task-service to call token cleaning endpoint twice daily (at 00:00 and 12:00 UTC)
    - Requires task-service to be running and configured
    - Requires `TASK_BASE_URL` and `API_KEY` environment variables to be set in auth-service
  - Returns: Success message with task scheduling confirmation
  - Errors:
    - 400 Bad Request: Missing or invalid token cleaning URL
    - 500 Internal Server Error: Task-service communication error or configuration issue

#### Token Cleaning (Requires API Key)
- `GET /api/v6/tokens/clean` - Clean expired user tokens
  - Headers:
    - `X-API-Key`: API key for service-to-service authentication (required)
  - Behavior:
    - Deletes all user tokens with `created_at` older than refresh token expiry time (configured via `JWT_REFRESH_TOKEN_EXPIRY`)
    - Returns count of deleted tokens
    - Handles case where no tokens are expired (0 deleted is not an error)
  - Returns: Success message with deletion confirmation
  - Errors:
    - 401 Unauthorized: Missing or invalid API key
    - 500 Internal Server Error: Database error during token deletion
  - Note: This endpoint is typically called by a scheduled task created via the admin task scheduling endpoint

### Learning Service (Port 8080)

#### Characters
- `GET /api/v6/characters?type={hr|kt}&locale={en|ru}` - Get all characters
- `GET /api/v6/characters/row-column?type={hr|kt}&locale={en|ru}&character={char}` - Get characters by row/column
- `GET /api/v6/characters/{id}?locale={en|ru}` - Get character by ID
  - Returns: Character information including audio URL (if available)

#### Tests (Requires Authentication)
- `GET /api/v6/tests/{hiragana|katakana}/reading?locale={en|ru}&count={number}` - Get reading test
  - Query Parameters:
    - `locale` (optional): Locale for reading - `en` or `ru` (default: `en`)
    - `count` (optional): Number of characters to return (default: 10)
  - Returns: List of characters with multiple choice options
  - **Smart Filtering**: When authenticated, prioritizes characters with no learning history, then characters with lowest reading test results
- `GET /api/v6/tests/{hiragana|katakana}/writing?locale={en|ru}&count={number}` - Get writing test
  - Query Parameters:
    - `locale` (optional): Locale for reading - `en` or `ru` (default: `en`)
    - `count` (optional): Number of characters to return (default: 10)
  - Returns: List of characters for writing practice
  - **Smart Filtering**: When authenticated, prioritizes characters with no learning history, then characters with lowest writing test results
- `GET /api/v6/tests/{hiragana|katakana}/listening?locale={en|ru}&count={number}` - Get listening test
  - Query Parameters:
    - `locale` (optional): Locale for reading - `en` or `ru` (default: `en`)
    - `count` (optional): Number of characters to return (default: 10)
  - Returns: List of characters with audio URLs, correct character, and wrong options
  - **Smart Filtering**: When authenticated, prioritizes characters with no learning history, then characters with lowest listening test results
  - **Note**: Only returns characters that have audio files uploaded

#### Test Results (Requires Authentication)
- `POST /api/v6/test-results/{hiragana|katakana}/{reading|writing|listening}` - Submit test results
  - Body: `{ "results": [{ "characterId": 1, "passed": true }, ...] }`
  - Test types: `reading`, `writing`, or `listening`
- `GET /api/v6/test-results/history` - Get user's learning history
  - Returns: All learning history records for the authenticated user

#### Dictionary / Words (Requires Authentication)
- `GET /api/v6/words?newCount={10-40}&oldCount={10-40}&locale={en|ru|de}` - Get word list
  - Query Parameters:
    - `newCount` (optional): Number of new words to return (10-40, default: 20)
    - `oldWordCount` (optional): Number of old words to return (10-40, default: 20)
    - `locale` (optional): Translation locale - `en`, `ru`, or `de` (default: `en`)
  - Returns: Mixed list of new and old words with locale-specific translations
  - Note: Includes audio URLs (`wordAudio` and `wordExampleAudio`) if available
- `POST /api/v6/words/results` - Submit word learning results
  - Body: `{ "results": [{ "wordId": 1, "period": 7 }, ...] }`
  - `period`: Days until next appearance (1-30)
  - Returns: 204 No Content on success

#### Courses and Lessons (Requires Authentication)
- `GET /api/v6/courses` - Get paginated list of courses
  - Query Parameters:
    - `complexityLevel` (optional): Filter by complexity level (`ab`, `b`, `i`, `ui`, `a` or full name)
    - `search` (optional): Search by course title
    - `isMine` (optional): Filter courses by user's completion history (boolean)
    - `page` (optional): Page number (default: 1)
    - `count` (optional): Items per page (default: 10)
  - Returns: List of courses with total lessons and completed lessons count
- `GET /api/v6/courses/{slug}/lessons` - Get course details with lessons list
  - Path Parameters:
    - `slug`: Course slug
  - Returns: Course details and list of lessons with completion status
- `GET /api/v6/lessons/{slug}` - Get lesson details
  - Path Parameters:
    - `slug`: Lesson slug
  - Returns: Lesson details and list of lesson blocks
- `POST /api/v6/lessons/{slug}/complete` - Toggle lesson completion
  - Path Parameters:
    - `slug`: Lesson slug
  - Returns: 204 No Content on success
  - Note: Toggles completion status (complete if not completed, uncomplete if completed)

#### Admin Endpoints (Requires Authentication & Admin Role)

##### Admin Characters
- `GET /api/v6/admin/characters` - Get full list of characters ordered by ID
  - Returns: List of all characters with full information including audio URLs
- `GET /api/v6/admin/characters/{id}` - Get character by ID
  - Returns: Full character information including audio URL (if available)
- `POST /api/v6/admin/characters` - Create a new character
  - Content-Type: `multipart/form-data`
  - Form Fields:
    - `consonant`: Consonant character
    - `vowel`: Vowel character
    - `englishReading`: English reading
    - `russianReading`: Russian reading
    - `hiragana`: Hiragana character
    - `katakana`: Katakana character
    - `audio` (optional): Audio file (MP3, WAV, etc.)
  - Returns: Created character ID
  - Note: Audio file is uploaded to media-service and URL is stored in the database
- `PATCH /api/v6/admin/characters/{id}` - Update a character (partial update)
  - Content-Type: `multipart/form-data`
  - Form Fields:
    - `consonant` (optional): Consonant character
    - `vowel` (optional): Vowel character
    - `englishReading` (optional): English reading
    - `russianReading` (optional): Russian reading
    - `hiragana` (optional): Hiragana character
    - `katakana` (optional): Katakana character
    - `audio` (optional): New audio file (replaces existing audio if provided)
  - Returns: 204 No Content on success
  - Note: If new audio is provided, old audio file is automatically deleted from media-service
- `DELETE /api/v6/admin/characters/{id}` - Delete a character by ID
  - Returns: 204 No Content on success
  - Note: Automatically deletes character's audio file from media-service if present

##### Admin Words
- `GET /api/v6/admin/words` - Get paginated list of words with optional search
  - Query Parameters:
    - `page` (optional): Page number (default: 1)
    - `count` (optional): Items per page (default: 20)
    - `search` (optional): Search in word, phonetic clues, or translations
  - Returns: List of words with pagination
- `GET /api/v6/admin/words/{id}` - Get word by ID
  - Returns: Full word information including all translations, examples, and audio URLs
- `POST /api/v6/admin/words` - Create a new word
  - Content-Type: `multipart/form-data`
  - Form Fields:
    - `word`: Japanese word (Kanji)
    - `phoneticClues`: Hiragana reading
    - `russianTranslation`: Russian translation
    - `englishTranslation`: English translation
    - `germanTranslation`: German translation
    - `example`: Japanese example sentence
    - `exampleRussianTranslation`: Russian translation of example
    - `exampleEnglishTranslation`: English translation of example
    - `exampleGermanTranslation`: German translation of example
    - `easyPeriod`: Days until next appearance for easy difficulty (1-30)
    - `normalPeriod`: Days until next appearance for normal difficulty (1-30)
    - `hardPeriod`: Days until next appearance for hard difficulty (1-30)
    - `extraHardPeriod`: Days until next appearance for extra hard difficulty (1-30)
    - `wordAudio` (optional): Word audio file (MP3, WAV, etc.)
    - `wordExampleAudio` (optional): Word example audio file (MP3, WAV, etc.)
  - Returns: Created word ID
  - Note: Audio files are uploaded to media-service and URLs are stored in the database
- `PATCH /api/v6/admin/words/{id}` - Update a word (partial update)
  - Content-Type: `multipart/form-data`
  - Form Fields (all optional):
    - `word`: Japanese word (Kanji)
    - `phoneticClues`: Hiragana reading
    - `russianTranslation`: Russian translation
    - `englishTranslation`: English translation
    - `germanTranslation`: German translation
    - `example`: Japanese example sentence
    - `exampleRussianTranslation`: Russian translation of example
    - `exampleEnglishTranslation`: English translation of example
    - `exampleGermanTranslation`: German translation of example
    - `easyPeriod`: Days until next appearance for easy difficulty (1-30)
    - `normalPeriod`: Days until next appearance for normal difficulty (1-30)
    - `hardPeriod`: Days until next appearance for hard difficulty (1-30)
    - `extraHardPeriod`: Days until next appearance for extra hard difficulty (1-30)
    - `wordAudio` (optional): New word audio file (replaces existing audio if provided)
    - `wordExampleAudio` (optional): New word example audio file (replaces existing audio if provided)
  - Returns: 204 No Content on success
  - Note: If new audio files are provided, old audio files are automatically deleted from media-service
- `DELETE /api/v6/admin/words/{id}` - Delete a word by ID
  - Returns: 204 No Content on success
  - Note: Automatically deletes word's audio files (word audio and word example audio) from media-service if present

##### Admin Courses and Lessons (Requires Authentication & Admin Role)
- `GET /api/v6/admin/courses` - Get paginated list of courses
  - Query Parameters:
    - `tutorId` (optional): Filter by tutor ID
    - `complexityLevel` (optional): Filter by complexity level (`ab`, `b`, `i`, `ui`, `a` or full name)
    - `search` (optional): Search by course title
    - `page` (optional): Page number (default: 1)
    - `count` (optional): Items per page (default: 10)
  - Returns: List of courses
- `GET /api/v6/admin/courses/short` - Get short course info (ID and title only)
  - Query Parameters:
    - `tutorId` (optional): Filter by tutor ID
  - Returns: List of courses with only ID and title
- `POST /api/v6/admin/courses` - Create a new course
  - Content-Type: `application/json`
  - Body: `{ "slug": "course-slug", "authorId": 1, "title": "Course Title", "shortSummary": "Summary", "complexityLevel": "Beginner" }`
  - Returns: Created course ID
- `GET /api/v6/admin/courses/{id}/lessons` - Get lessons for a course
  - Returns: Course details and list of lessons
- `PATCH /api/v6/admin/courses/{id}` - Update a course (partial update)
  - Content-Type: `application/json`
  - Body: Partial course data (all fields optional)
  - Returns: 204 No Content on success
- `DELETE /api/v6/admin/courses/{id}` - Delete a course by ID
  - Returns: 204 No Content on success
  - Note: Cascades to delete all lessons and lesson blocks
- `POST /api/v6/admin/lessons` - Create a new lesson
  - Content-Type: `application/json`
  - Body: `{ "slug": "lesson-slug", "courseId": 1, "title": "Lesson Title", "shortSummary": "Summary", "order": 1 }`
  - Returns: Created lesson ID
- `GET /api/v6/admin/lessons/short` - Get short lesson info (ID and title only)
  - Query Parameters:
    - `courseId` (optional): Filter by course ID
  - Returns: List of lessons with only ID and title
- `GET /api/v6/admin/lessons/{id}` - Get full lesson information
  - Returns: Lesson details with all blocks
- `PATCH /api/v6/admin/lessons/{id}` - Update a lesson (partial update)
  - Content-Type: `application/json`
  - Body: Partial lesson data (all fields optional)
  - Returns: 204 No Content on success
- `DELETE /api/v6/admin/lessons/{id}` - Delete a lesson by ID
  - Returns: 204 No Content on success
  - Note: Cascades to delete all lesson blocks
- `POST /api/v6/admin/blocks` - Create a lesson block
  - Content-Type: `application/json`
  - Body: `{ "lessonId": 1, "blockType": "video", "blockOrder": 1, "blockData": {...} }`
  - Block types: `video`, `audio`, `text`, `document`, `list`
  - Returns: Created block ID
- `PATCH /api/v6/admin/blocks/{id}` - Update a lesson block
  - Content-Type: `application/json`
  - Body: Partial block data (all fields optional)
  - Returns: 204 No Content on success
- `DELETE /api/v6/admin/blocks/{id}` - Delete a lesson block
  - Returns: 204 No Content on success
- `GET /api/v6/admin/media` - Get tutor media list
  - Query Parameters:
    - `tutorId` (optional): Filter by tutor ID
    - `mediaType` (optional): Filter by media type (`video`, `doc`, `audio`)
  - Returns: List of tutor media
- `POST /api/v6/admin/media` - Create tutor media
  - Content-Type: `multipart/form-data`
  - Form Fields:
    - `tutorId`: Tutor ID
    - `slug`: Media slug
    - `mediaType`: Media type (`video`, `doc`, `audio`)
    - `file`: Media file
  - Returns: Created media ID
  - Note: File is uploaded to media-service and URL is stored in the database
- `DELETE /api/v6/admin/media/{id}` - Delete tutor media
  - Returns: 204 No Content on success
  - Note: Automatically deletes media file from media-service

##### Tutor Courses and Lessons (Requires Authentication & Tutor Role)
- All admin course and lesson endpoints are available to tutors
- Tutors can only manage courses they own (author_id matches tutor ID)
- Tutors can manage lessons within their own courses
- Tutor endpoints use the same paths as admin endpoints but with ownership validation

### Media Service (Port 8082)

#### Media Management (Requires API Key for Upload/Delete)
- `GET /api/v6/media/{id}` - Get file metadata by ID
  - Returns: Metadata information (content type, size, URL, type)
  - Public endpoint (no authentication required)
- `GET /api/v6/media/{mediaType}/{filename}` - Download media file
  - Path Parameters:
    - `mediaType`: Type of media (`character`, `word`, `word_example`, `lesson_audio`, `lesson_video`, `lesson_doc`)
    - `filename`: Name of the file to download
  - Headers:
    - `Range` (optional): For partial content requests (audio/video files)
  - Returns: File content
  - Character files are public; other types require JWT authentication
  - Audio/video files support HTTP range requests (206 Partial Content)
- `POST /api/v6/media/{mediaType}` - Upload media file
  - Headers:
    - `X-API-Key`: API key for service-to-service authentication (required)
  - Body: `multipart/form-data` with `file` field
  - Path Parameters:
    - `mediaType`: Type of media (see above)
  - Returns: Download URL as plain text
  - Maximum file size: 50MB
- `DELETE /api/v6/media/{mediaType}/{filename}` - Delete media file
  - Headers:
    - `X-API-Key`: API key for service-to-service authentication (required)
  - Path Parameters:
    - `mediaType`: Type of media
    - `filename`: Name of the file to delete
  - Returns: 204 No Content on success

**Media Types**:
- `character` - Character images (public access)
- `word` - Word images (requires authentication)
- `word_example` - Word example images (requires authentication)
- `lesson_audio` - Lesson audio files (requires authentication, supports range requests)
- `lesson_video` - Lesson video files (requires authentication, supports range requests)
- `lesson_doc` - Lesson documents (requires authentication)
- `avatar` - User avatar images (used by auth-service for user profile pictures)

### Task Service (Port 8083)

The task-service consists of three sub-services:
- **API Service**: HTTP API for creating and managing tasks
- **Scheduler Service**: Manages scheduled tasks in Redis ZSET and enqueues due tasks
- **Worker Service**: Processes tasks from Asynq queues (immediate and scheduled tasks)

#### Task Management (Requires API Key)
- `POST /api/v6/tasks/immediate` - Create an immediate task
  - Headers:
    - `X-API-Key`: API key for service-to-service authentication (required)
  - Body: `{ "userId": 1, "emailSlug": "welcome-email", "content": "user@example.com;John Doe" }`
  - Returns: Task ID and success message
  - Note: Content field uses semicolon (`;`) as separator - first value is recipient email, rest are template variables
- `POST /api/v6/tasks/scheduled` - Create a scheduled task
  - Headers:
    - `X-API-Key`: API key for service-to-service authentication (required)
  - Body: `{ "userId": 1, "emailSlug": "reminder-email", "url": "http://example.com/webhook", "content": "data", "cron": "0 9 * * *" }`
  - Returns: Task ID and success message
  - Note: At least one of `emailSlug` or `url` must be provided. Cron expression follows standard cron format.

#### Admin Endpoints (Requires Authentication & Admin Role)

##### Email Templates
- `GET /api/v6/admin/email-templates` - Get paginated list of email templates
  - Query Parameters:
    - `page` (optional): Page number (default: 1)
    - `count` (optional): Items per page (default: 20)
    - `search` (optional): Search in template slug
  - Returns: List of email templates with pagination
- `GET /api/v6/admin/email-templates/{id}` - Get email template by ID
  - Returns: Full email template information
- `POST /api/v6/admin/email-templates` - Create a new email template
  - Body: `{ "slug": "welcome-email", "subjectTemplate": "Welcome {{.Name}}!", "bodyTemplate": "Hello {{.Name}}, welcome to our platform!" }`
  - Returns: Created template ID
- `PATCH /api/v6/admin/email-templates/{id}` - Update an email template (partial update)
  - Body: Partial template data (all fields optional)
  - Returns: 204 No Content on success
- `DELETE /api/v6/admin/email-templates/{id}` - Delete an email template by ID
  - Returns: 204 No Content on success

##### Immediate Tasks
- `GET /api/v6/admin/immediate-tasks` - Get paginated list of immediate tasks
  - Query Parameters:
    - `page` (optional): Page number (default: 1)
    - `count` (optional): Items per page (default: 20)
    - `user_id` (optional): Filter by user ID
    - `template_id` (optional): Filter by template ID
    - `status` (optional): Filter by status (`Enqueued`, `Completed`, `Failed`)
  - Returns: List of immediate tasks with pagination
- `GET /api/v6/admin/immediate-tasks/{id}` - Get immediate task by ID
  - Returns: Full immediate task information
- `POST /api/v6/admin/immediate-tasks` - Create a new immediate task
  - Body: `{ "userId": 1, "templateId": 1, "content": "user@example.com;John Doe" }`
  - Returns: Created task ID
- `PATCH /api/v6/admin/immediate-tasks/{id}` - Update an immediate task (partial update)
  - Body: Partial task data (all fields optional)
  - Returns: 204 No Content on success
- `DELETE /api/v6/admin/immediate-tasks/{id}` - Delete an immediate task by ID
  - Returns: 204 No Content on success

##### Scheduled Tasks
- `GET /api/v6/admin/scheduled-tasks` - Get paginated list of scheduled tasks
  - Query Parameters:
    - `page` (optional): Page number (default: 1)
    - `count` (optional): Items per page (default: 20)
    - `user_id` (optional): Filter by user ID
    - `template_id` (optional): Filter by template ID
    - `active` (optional): Filter by active status (boolean)
  - Returns: List of scheduled tasks with pagination
- `GET /api/v6/admin/scheduled-tasks/{id}` - Get scheduled task by ID
  - Returns: Full scheduled task information
- `POST /api/v6/admin/scheduled-tasks` - Create a new scheduled task
  - Body: `{ "userId": 1, "templateId": 1, "url": "http://example.com/webhook", "content": "data", "cron": "0 9 * * *" }`
  - Returns: Created task ID
  - Note: At least one of `templateId` or `url` must be provided
- `PATCH /api/v6/admin/scheduled-tasks/{id}` - Update a scheduled task (partial update)
  - Body: Partial task data (all fields optional)
  - Returns: 204 No Content on success
  - Note: If `active` is set to false, task is removed from Redis ZSET. If `active` is set to true or `nextRun` is updated, task is added/updated in Redis ZSET.
- `DELETE /api/v6/admin/scheduled-tasks/{id}` - Delete a scheduled task by ID
  - Returns: 204 No Content on success
  - Note: Automatically removes task from Redis ZSET

##### Scheduled Task Logs
- `GET /api/v6/admin/scheduled-task-logs` - Get paginated list of scheduled task logs
  - Query Parameters:
    - `page` (optional): Page number (default: 1)
    - `count` (optional): Items per page (default: 20)
    - `task_id` (optional): Filter by task ID
    - `job_id` (optional): Filter by job ID
    - `status` (optional): Filter by status (`Completed`, `Failed`)
  - Returns: List of task logs with pagination
- `GET /api/v6/admin/scheduled-task-logs/{id}` - Get scheduled task log by ID
  - Returns: Full task log information

### API Documentation

#### Auth Service
- Swagger UI: `http://localhost:8081/swagger/index.html`
- Swagger JSON: `http://localhost:8081/swagger/doc.json`

#### Learn Service
- Swagger UI: `http://localhost:8080/swagger/index.html`
- Swagger JSON: `http://localhost:8080/swagger/doc.json`

#### Media Service
- Swagger UI: `http://localhost:8082/swagger/index.html`
- Swagger JSON: `http://localhost:8082/swagger/doc.json`

#### Task Service
- Swagger UI: `http://localhost:8083/swagger/index.html`
- Swagger JSON: `http://localhost:8083/swagger/doc.json`

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

**Prerequisites:** 
- Test databases must be configured (MySQL/MariaDB)
- Redis server running (required for task-service integration tests)
- Asynq client/server (required for task-service integration tests)

See TESTING.md for detailed setup instructions.

```bash
# Run all integration tests
go test ./services/.../test/integration/... -v

# Run specific service integration tests
go test ./services/auth-service/test/integration/... -v
go test ./services/learn-service/test/integration/... -v
go test ./services/task-service/test/integration/... -v
# Note: media-service integration tests are not yet implemented

# Run integration tests with coverage
go test ./services/.../test/integration/... -v -cover
```

**Integration Test Coverage:**
- ✅ **auth-service**: Full end-to-end API tests, repository layer tests, service layer tests, profile handler tests (60+ test cases)
- ✅ **learn-service**: Character tests, dictionary tests, course/lesson tests, repository and service layer tests (60+ test cases)
- ✅ **task-service**: Repository and service layer tests with Redis and Asynq integration (10+ test cases)
- ⚠️ **media-service**: Integration tests not yet implemented

All integration tests automatically set up test data, run tests in isolation, and clean up afterward.

## Database Schema

### Auth Service Database

#### Users Table
- `id` - Primary key (auto-increment)
- `email` - Email address (unique, indexed)
- `username` - Username (unique, indexed)
- `password_hash` - Bcrypt hashed password
- `role` - User role (default: 'user')
- `avatar` - Avatar image URL (optional, VARCHAR(500))
- `active` - Account active status (BOOLEAN, default: FALSE)

#### User Tokens Table
- `id` - Primary key (auto-increment)
- `user_id` - Foreign key to users.id
- `token` - Refresh token (unique, indexed)
- `created_at` - Timestamp when token was created (used for token expiration tracking)

#### User Settings Table
- `id` - Primary key (auto-increment)
- `user_id` - Foreign key to users.id (unique)
- `new_word_count` - Number of new words to show (10-40, default: 20)
- `old_word_count` - Number of old words to show (10-40, default: 20)
- `alphabet_learn_count` - Number of characters for alphabet tests (5-15, default: 10)
- `language` - Preferred language for translations (`en`, `ru`, or `de`, default: `en`)
- `created_at` - Timestamp
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
- `audio` - Audio file URL (VARCHAR(500), nullable) - URL to audio file stored on media-service

#### Character Learn History Table
- `id` - Primary key (auto-increment)
- `user_id` - User ID (from auth service)
- `character_id` - Foreign key to characters.id
- `hiragana_reading_result` - Test result (0.0-1.0)
- `hiragana_writing_result` - Test result (0.0-1.0)
- `hiragana_listening_result` - Test result (0.0-1.0) - Used for listening test results
- `katakana_reading_result` - Test result (0.0-1.0)
- `katakana_writing_result` - Test result (0.0-1.0)
- `katakana_listening_result` - Test result (0.0-1.0) - Used for listening test results
- Unique constraint on (user_id, character_id)
- **Note**: Test results are used for smart filtering in test endpoints to prioritize characters that need more practice

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
- `word_audio` - Word audio file URL (VARCHAR(500), nullable) - URL to audio file stored on media-service
- `word_example_audio` - Word example audio file URL (VARCHAR(500), nullable) - URL to example audio file stored on media-service

#### Dictionary History Table
- `id` - Primary key (auto-increment)
- `word_id` - Foreign key to words.id
- `user_id` - User ID (from auth service)
- `next_appearance` - Date when the word should appear again (indexed)
- Unique constraint on (user_id, word_id)
- Foreign key constraint on word_id with CASCADE delete

#### Courses Table
- `id` - Primary key (auto-increment)
- `slug` - Unique URL-friendly identifier (unique, indexed)
- `author_id` - Tutor/author user ID (indexed)
- `title` - Course title (indexed)
- `short_summary` - Course summary/description
- `complexity_level` - Complexity level enum (Absolute beginner, Beginner, Intermediate, Upper Intermediate, Advanced) (indexed)

#### Lessons Table
- `id` - Primary key (auto-increment)
- `slug` - Unique URL-friendly identifier (unique, indexed)
- `course_id` - Foreign key to courses.id (indexed, CASCADE delete)
- `title` - Lesson title
- `short_summary` - Lesson summary/description
- `order` - Order within course (indexed with course_id)

#### Lesson Blocks Table
- `id` - Primary key (auto-increment)
- `lesson_id` - Foreign key to lessons.id (indexed, CASCADE delete)
- `block_type` - Block type enum (video, audio, text, document, list)
- `block_order` - Order within lesson (indexed with lesson_id)
- `block_data` - JSON data containing block content

#### Lesson User History Table
- `id` - Primary key (auto-increment)
- `user_id` - User ID (from auth service) (indexed)
- `course_id` - Foreign key to courses.id (indexed, CASCADE delete)
- `lesson_id` - Foreign key to lessons.id (indexed, CASCADE delete)
- Unique constraint on (user_id, course_id, lesson_id)
- Tracks which lessons users have completed

#### Tutor Media Table
- `id` - Primary key (auto-increment)
- `tutor_id` - Tutor user ID (indexed)
- `slug` - Unique media identifier (unique, indexed)
- `media_type` - Media type enum (video, doc, audio) (indexed)
- `url` - Media file URL (VARCHAR(500)) - URL to media file stored on media-service

### Media Service Database

#### Metadata Table
- `id` - Primary key (filename/ID)
- `content_type` - MIME type of the file (e.g., "image/jpeg", "video/mp4")
- `size` - File size in bytes
- `url` - Download URL for the file
- `type` - Media type (`character`, `word`, `word_example`, `lesson_audio`, `lesson_video`, `lesson_doc`)

### Task Service Database

#### Email Templates Table
- `id` - Primary key (auto-increment)
- `slug` - Unique template identifier (unique, indexed)
- `subject_template` - Email subject template (TEXT)
- `body_template` - Email body template (TEXT)
- `created_at` - Timestamp
- `updated_at` - Timestamp

#### Immediate Tasks Table
- `id` - Primary key (auto-increment)
- `user_id` - User ID (not a foreign key, indexed)
- `template_id` - Foreign key to email_templates.id (nullable, indexed, ON DELETE SET NULL)
- `content` - Task content (TEXT) - uses semicolon (`;`) as separator
- `created_at` - Timestamp (indexed)
- `status` - Task status (`Enqueued`, `Completed`, `Failed`, indexed)
- `error` - Error message if task failed (TEXT)

#### Scheduled Tasks Table
- `id` - Primary key (auto-increment)
- `user_id` - User ID (nullable, not a foreign key, indexed)
- `template_id` - Foreign key to email_templates.id (nullable, indexed, ON DELETE SET NULL)
- `url` - Webhook URL (VARCHAR(500), nullable)
- `content` - Task content (TEXT)
- `created_at` - Timestamp
- `next_run` - Next execution time (DATETIME, indexed)
- `previous_run` - Last execution time (DATETIME, nullable)
- `active` - Whether task is active (BOOLEAN, default: TRUE, indexed)
- `cron` - Cron expression for scheduling (VARCHAR(100))

#### Scheduled Task Logs Table
- `id` - Primary key (auto-increment)
- `task_id` - Foreign key to scheduled_tasks.id (indexed)
- `job_id` - Asynq job ID (VARCHAR(255))
- `status` - Execution status (`Completed`, `Failed`)
- `http_status` - HTTP status code from webhook (INTEGER)
- `error` - Error message if execution failed (TEXT)
- `created_at` - Timestamp

## Middleware

All services include the following middleware (in order of execution):

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
- **Email Verification**: Users must verify their email address before they can login. New users are created with `active=false` and must verify via the email link to activate their account.
- **Role-Based Access Control**: Users have roles (default: `user`). Different endpoints require different role levels.

### User Roles

The system supports the following user roles:
- **User** (role = 0): Default role for regular users. Can access user endpoints and complete lessons.
- **Tutor** (role = 2): Can create and manage their own courses, lessons, and media files. Tutors can only manage content they own.
- **Admin** (role = 1 or higher): Can manage all users, courses, lessons, characters, and words regardless of ownership.

### Password Requirements

- Minimum 8 characters
- At least one lowercase letter
- At least one uppercase letter
- At least one number
- At least one special character from: `!_?^&+-=|`

### Admin Access

Admin endpoints require:
1. Valid JWT authentication token
2. User role must be set to admin (role value: 1 or higher)
3. Admin middleware validates both authentication and role permissions

### Tutor Access

Tutor endpoints require:
1. Valid JWT authentication token
2. User role must be set to tutor (role value: 2)
3. Tutors can only manage courses they own (where `author_id` matches their user ID)
4. Tutor middleware validates both authentication and role permissions, plus ownership for course/lesson operations

## Logging

The application uses structured JSON logging with the following levels:
- `info` - General information
- `warn` - Warnings
- `error` - Errors

All logs include request ID for tracing across services.

## License

[Add your license here]
