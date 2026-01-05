# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy libs directory first (needed for local module replacement)
COPY libs ./libs

# Copy go mod files for task-service
COPY services/task-service/go.mod services/task-service/go.sum ./services/task-service/
WORKDIR /app/services/task-service
RUN go mod download

# Copy source code
COPY services/task-service .

# Build scheduler binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/bin/task-scheduler ./cmd/scheduler

# Build worker binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/bin/task-worker ./cmd/worker

# Build API binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/bin/task-api ./cmd/api

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy the binaries from builder
COPY --from=builder /app/bin/task-scheduler ./task-scheduler
COPY --from=builder /app/bin/task-worker ./task-worker
COPY --from=builder /app/bin/task-api ./task-api

# Copy migrations
COPY --from=builder /app/services/task-service/migrations ./migrations

# Expose port (for API)
EXPOSE 8083

# Default command (will be overridden in docker-compose)
CMD ["./task-api"]
