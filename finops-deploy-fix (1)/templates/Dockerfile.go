# Multi-stage build for Go services
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install git and ca-certificates for fetching dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates wget curl

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/main .

# Expose port (override via ENV PORT)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=3 \
  CMD wget --spider -q http://localhost:${PORT:-8080}/health 2>/dev/null || curl -fsS http://localhost:${PORT:-8080}/health

CMD ["./main"]
