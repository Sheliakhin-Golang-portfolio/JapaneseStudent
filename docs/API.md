# API Documentation

This document describes the public HTTP APIs exposed by the JapaneseStudent backend.

It preserves the API-related information from the original README and serves as a high-level reference.  
For full endpoint definitions, request schemas, and responses, refer to the Swagger UI.

---

## API Versioning

The project uses **explicit API versioning**.

Each service exposes versioned endpoints (e.g. `/api/v6/...`) to allow:
- backward compatibility
- safe evolution of APIs
- parallel support of multiple clients

Breaking changes are introduced only in new API versions.

---

## Authentication & Authorization

### Authentication Method
Authentication is implemented using **JWT (JSON Web Tokens)**.

Two token types are used:
- **Access token** — short-lived, stateless
- **Refresh token** — stored in the database and revocable

### Authentication Flow
1. User registers and verifies email
2. User logs in and receives access + refresh tokens
3. Access token is sent with each protected request
4. Refresh token is used to renew access tokens

### Authorization
Role-based access control is enforced:
- user
- tutor
- admin

Authorization checks are performed at the API layer.

---

## Email Verification & Notifications

Certain API actions trigger **asynchronous workflows**:

- User registration
- Email updates
- Token cleanup
- Alphabet repetition reminders

These actions enqueue background tasks instead of blocking HTTP requests.

---

## Background Tasks & API Interaction

Some API endpoints do not perform work synchronously.

Instead, they:
- validate the request
- persist intent to the database
- enqueue a background task via `task-service`

This approach:
- keeps APIs responsive
- improves reliability
- supports retries and observability

---

## Media Access

Media files (audio, images, lesson assets) are **not served directly by business services**.

Instead:
- APIs return media URLs
- Actual file access is handled by `media-service`

This separation avoids filesystem coupling across services.

---

## Swagger / OpenAPI

Interactive API documentation is available via **Swagger UI** when services are running locally.

Swagger provides:
- complete endpoint list
- request/response schemas
- authentication requirements
- error responses

Swagger should be treated as the **authoritative API reference**.

---

## Error Handling

APIs return structured error responses with:
- HTTP status codes
- machine-readable error identifiers
- human-readable messages

Validation errors are handled explicitly and returned to clients in a consistent format.

---

## Notes for Reviewers

- APIs are designed to be consumed by multiple clients (web, mobile)
- Business logic is isolated from HTTP handlers
- Long-running operations are intentionally asynchronous
