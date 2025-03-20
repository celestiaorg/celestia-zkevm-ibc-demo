FROM golang:1.24-alpine AS builder

# Install git and build dependencies
RUN apk add --no-cache git make gcc libc-dev

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum files first to leverage Docker cache
COPY go.mod go.sum* ./

# Download dependencies if go.mod and go.sum are available
RUN if [ -f go.sum ]; then go mod download; fi

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -o indexer .

# Create a minimal runtime image
FROM alpine:3.18

# Add necessary runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create a non-root user to run the application
RUN adduser -D -g '' indexer

# Create directory for database files with proper permissions
RUN mkdir -p /data && chown -R indexer:indexer /data

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/indexer .

# Set ownership of application files
RUN chown -R indexer:indexer /app

# Switch to non-root user
USER indexer

# Directory for persistence
VOLUME ["/data"]

# Expose API port
EXPOSE 8080

# Set environment variables with defaults
ENV CELESTIA_NODE_URL=ws://celestia-node:26658 \
    CELESTIA_NODE_AUTH_TOKEN="" \
    CELESTIA_NAMESPACE="0f0f0f0f0f0f0f0f0f0f" \
    API_PORT=8080 \
    HTTP_TIMEOUT_SECONDS=30 \
    RECONNECT_DELAY_SECONDS=5

# Run the application with the database in the volume
CMD ["./indexer"]
