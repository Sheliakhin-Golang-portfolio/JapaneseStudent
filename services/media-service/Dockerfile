# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy libs directory first (needed for local module replacement)
COPY libs ./libs

# Copy go mod files for media-service
COPY services/media-service/go.mod services/media-service/go.sum ./services/media-service/
WORKDIR /app/services/media-service
RUN go mod download

# Copy source code
COPY services/media-service .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/bin/media-service ./cmd

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/bin/media-service .

# Copy migrations
COPY --from=builder /app/services/media-service/migrations ./migrations

# Expose port
EXPOSE 8082

# Run the application
CMD ["./media-service"]

