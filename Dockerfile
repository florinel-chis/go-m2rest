# Build stage
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the test binary
RUN go test -c -o test-runner ./tests

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates

# Set working directory
WORKDIR /app

# Copy the test binary from builder
COPY --from=builder /app/test-runner .
COPY --from=builder /app/.env.example .env

# Set environment variables (can be overridden at runtime)
ENV MAGENTO_HOST=http://magento.local \
    MAGENTO_BEARER_TOKEN=your_token_here \
    MAGENTO_STORE_CODE=all \
    MAGENTO_API_VERSION=V1 \
    MAGENTO_REST_PREFIX=/rest \
    TEST_TIMEOUT=60s \
    TEST_DEBUG=true

# Default command runs all tests
CMD ["./test-runner", "-test.v"]