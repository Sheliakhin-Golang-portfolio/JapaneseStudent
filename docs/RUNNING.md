# Running the Project Locally

This document explains how to run the JapaneseStudent project locally for development.

It preserves the setup and execution information from the original README, reorganized for clarity and ease of use.

---

## Prerequisites

Before running the project, ensure you have:

- Go (version compatible with the project)
- Docker
- Docker Compose

These tools are required to run infrastructure dependencies and services locally.

---

## Project Dependencies

The application relies on the following external services:

- **MariaDB** — primary relational database
- **Redis** — background job queues and scheduling

These dependencies are expected to run before starting application services.

---

## Configuration

All configuration is provided via **environment variables**.

Environment variables are documented in detail here:

➡️ **[CONFIGURATION.md](CONFIGURATION.md)**

No secrets or configuration values are hardcoded in the codebase.

---

## Starting Infrastructure Services

Infrastructure services can be started using Docker Compose.

From the project root:

```bash
docker compose up -d --build
```

This starts:
- MariaDB
- Redis
- All services

After that the application is ready to use.

## Running Application Services separately

Also you can start each service independently. For that, from the project root, run the desired service:

go run ./cmd/<service-name>

Examples:
- `auth-service`
- `learn-service`
- `media-service`
- `task-service`

Each service:
- reads configuration from environment variables
- exposes an HTTP API on its configured port
- can be restarted independently

Before running a service independently, make sure that all necessary containers (MariaDB and/or Redis) are already running.

## task-service Processes

The task-service consists of multiple processes and may require running more than one entry point:
- **API process** — accepts HTTP requests
- **Scheduler** — manages scheduled tasks
- **Worker** — executes background jobs

Each process is started separately using its corresponding entry point under `cmd/`.

This separation mirrors production background processing architectures.

## Development Notes
- Services are intentionally decoupled and can be developed independently
- Logs are written to stdout for local development
- Errors are returned with structured messages
- Background tasks are retried automatically according to configuration

## Stopping the Project

To stop infrastructure services:

```
docker compose down
```

Application services can be stopped using standard process termination (Ctrl+C).

## Notes for Reviewers
- The project is designed to be run locally without external dependencies
- Docker is used only for infrastructure, not for application logic
- Configuration is explicit and environment-driven