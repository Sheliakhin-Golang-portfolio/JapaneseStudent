# Configuration & Environment Variables

This document describes all environment variables used by the services.

All variables listed here are preserved from the original README and configuration files.

---

## General

| Variable | Description |
|--------|-------------|
| `LOG_LEVEL` | Logging level used by services |
| `API_KEY` | API key used for interservice access |

---

## Database (MariaDB)

| Variable | Description |
|--------|-------------|
| `DB_HOST` | MariaDB host |
| `DB_PORT` | MariaDB port |
| `DB_NAME` | Database name |
| `DB_USER` | Database user |
| `DB_PASSWORD` | Database password |

Used by:
- auth-service
- learn-service
- media-service
- task-service

---

## Redis

| Variable | Description |
|--------|-------------|
| `REDIS_HOST` | Redis host |
| `REDIS_PORT` | Redis port |
| `REDIS_DB` | Redis database number |
| `REDIS_PASSWORD` | Redis password (if enabled) |

Used by:
- task-service

---

## Authentication & Security

| Variable | Description |
|--------|-------------|
| `JWT_SECRET` | Secret key used to sign JWT tokens |
| `JWT_ACCESS_TOKEN_EXPIRY` | Access token expiration duration |
| `JWT_REFRESH_TOKEN_EXPIRY` | Refresh token expiration duration |

Used by:
- auth-service

---

## Service Ports

| Variable | Description |
|--------|-------------|
| `AUTH_SERVICE_PORT` | HTTP port for auth-service |
| `LEARN_SERVICE_PORT` | HTTP port for learn-service |
| `MEDIA_SERVICE_PORT` | HTTP port for media-service |
| `TASK_SERVICE_PORT` | HTTP port for task-service |

---

## Service URLs / Base URLs

These variables define how services communicate with each other.

| Variable | Description |
|--------|-------------|
| `LEARN_SERVICE_BASE_URL` | Base URL of learn-service (inner link for services) |
| `MEDIA_BASE_URL` | Base URL of media-service (inner link for services) |
| `MEDIA_ACCESS_BASE_URL` | Public access URL for media files |
| `IMMEDIATE_TASK_BASE_URL` | Base URL for immediate task management |
| `SCHEDULED_TASK_BASE_URL` | Base URL for scheduled task management |
| `VERIFICATION_URL` | Base URL used in email verification links |

---

## Media Service

| Variable | Description |
|--------|-------------|
| `MEDIA_BASE_PATH` | Filesystem path where media files are stored |

Used by:
- media-service
- auth-service (internal access)
- learn-service (internal access)

---

## Email (SMTP)

| Variable | Description |
|--------|-------------|
| `SMTP_HOST` | SMTP server host |
| `SMTP_PORT` | SMTP server port |
| `SMTP_USERNAME` | SMTP username |
| `SMTP_PASSWORD` | SMTP password |
| `SMTP_FROM` | Sender email address |

Used by:
- task-service (email delivery)

---

## CORS

| Variable | Description |
|--------|-------------|
| `CORS_ALLOWED_ORIGINS` | Allowed origins for CORS configuration |

Used by:
- HTTP-facing services

---

## Notes for Reviewers

- Environment variables are loaded at startup
- No configuration is hardcoded
- Secrets are never committed to the repository