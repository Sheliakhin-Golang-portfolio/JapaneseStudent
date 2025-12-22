# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy libs directory first (needed for local module replacement)
COPY libs ./libs

# Copy go mod files for learn-service
COPY services/learn-service/go.mod services/learn-service/go.sum ./services/learn-service/
WORKDIR /app/services/learn-service
RUN go mod download

# Copy source code
COPY services/learn-service .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/bin/learn-service ./cmd

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/bin/learn-service .

# Copy migrations
COPY --from=builder /app/services/learn-service/migrations ./migrations

# Expose port
EXPOSE 8080

# Run the application
CMD ["./learn-service"]
