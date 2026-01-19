# Services Overview

JapaneseStudent is built as a **microservices-based system**.  
Each service is independently deployable, owns its own domain logic, and communicates with other services via HTTP APIs.

This document describes **each service’s responsibility, scope, and integration points**.

---

## auth-service (Port: 8081)

### Responsibility
The **auth-service** is responsible for **user identity, authentication, authorization, and profile management**.

It acts as the **entry point for user access control** across the system.

### Core Features
- User registration with email verification
- JWT-based authentication (access & refresh tokens)
- Role-based authorization (user / tutor / admin)
- User profile management
- User settings management
- Password hashing and validation
- Token lifecycle management (refresh & cleanup)

### Key Functionalities
- Issues JWT access and refresh tokens after email verification
- Stores refresh tokens in the database for revocation and cleanup
- Integrates with:
  - **media-service** for avatar upload and deletion
  - **task-service** for sending verification emails and scheduling token cleanup
  - **learn-service** for alphabet repetition logic

### Notes for Reviewers
- Authentication is **stateful only for refresh tokens**, access tokens remain stateless
- Email verification is mandatory before login
- Alphabet repeat functionality is coordinated via scheduled tasks

---

## learn-service (Port: 8080)

### Responsibility
The **learn-service** contains the **core learning logic** of the application.

It manages Japanese language learning content and tracks user progress.

### Core Features
- Hiragana and Katakana character learning
- Reading, writing, and listening tests
- Smart test generation based on user performance
- Dictionary with spaced repetition
- Courses and lessons system
- Tutor-managed educational content

### Key Functionalities
- Generates tests using **adaptive filtering**:
  - Prioritizes unseen characters
  - Falls back to characters with lowest scores
- Tracks detailed learning history per user
- Manages lesson completion and course progress
- Provides admin and tutor tooling for educational content

### Integration Points
- Called by **auth-service** for alphabet repeat functionality
- Uses **media-service** URLs for audio and lesson media
- Receives scheduled calls from **task-service** to drop user marks

### Notes for Reviewers
- Learning logic is separated from authentication concerns
- Business rules are isolated in service layer
- Designed to support future frontend/mobile clients

---

## media-service (Port: 8082)

### Responsibility
The **media-service** manages **file storage and media access** for the entire system.

It centralizes file handling to keep other services stateless.

### Core Features
- Media upload and deletion via API key authentication
- Public and protected media access
- HTTP range requests for audio/video streaming
- Metadata storage for uploaded files

### Managed Media Types
- Character audio
- Word and example audio
- Lesson audio, video, and documents
- User avatars

### Integration Points
- Used by **auth-service** for user avatars
- Used by **learn-service** for character and lesson media
- Used by **task-service** indirectly via stored URLs

### Notes for Reviewers
- Media access rules vary by type (public vs authenticated)
- Designed to scale independently from business logic
- Keeps other services free from filesystem concerns

---

## task-service (Port: 8083)

### Responsibility
The **task-service** handles **background processing, scheduling, and asynchronous execution**.

It is designed for reliability and idempotent execution.

### Architecture
The service consists of **three separate processes**:

1. **API Service**
   - Accepts HTTP requests to manage tasks
2. **Scheduler**
   - Manages scheduled tasks using Redis ZSET
   - Enqueues due tasks
3. **Worker**
   - Executes tasks using Asynq queues

### Core Features
- Immediate background task execution
- Cron-based scheduled tasks
- Email delivery via templates
- Webhook execution
- Execution logs and retry handling

### Key Functionalities
- Prevents duplicate scheduled tasks per user and URL
- Stores task state in database for observability
- Logs execution results and errors
- Supports both email-based and webhook-based tasks

### Integration Points
- Used by **auth-service** for:
  - Email verification
  - Email updates
  - Token cleanup scheduling
- Used by **learn-service** for alphabet repetition scheduling

### Notes for Reviewers
- Separating scheduler and worker processes mirrors production systems
- Redis + Asynq is a common Go industry stack
- Explicit task logging improves debuggability

---

## Service Communication Summary

| From Service | To Service     | Purpose |
|-------------|----------------|---------|
| auth-service | media-service | Avatar upload and deletion |
| auth-service | task-service  | Email delivery and scheduling |
| auth-service | learn-service | Alphabet repetition logic |
| learn-service | media-service | Audio and lesson media |
| task-service | learn-service | Scheduled mark reduction |
