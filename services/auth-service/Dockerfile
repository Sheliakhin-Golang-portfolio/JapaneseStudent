# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install git and ca-certificates (needed for go get)
RUN apk add --no-cache git ca-certificates

# Copy libs directory first (needed for local module replacement)
COPY libs ./libs

# Copy go mod files for auth-service
COPY services/auth-service/go.mod services/auth-service/go.sum ./services/auth-service/
WORKDIR /app/services/auth-service
RUN go mod download

# Copy source code
COPY services/auth-service .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/bin/auth-service ./cmd

# Final stage
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Copy the binary from builder
COPY --from=builder /app/bin/auth-service .

# Copy migrations
COPY --from=builder /app/services/auth-service/migrations ./migrations

# Expose port
EXPOSE 8081

# Run the application
CMD ["./auth-service"]

