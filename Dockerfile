# Build stage - Use latest 1.24.4 Alpine image
FROM golang:1.24.4-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download and verify dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application with build info and security flags
RUN CGO_ENABLED=0 GOOS=linux go build \
    -a \
    -installsuffix cgo \
    -ldflags='-w -s -extldflags "-static"' \
    -o torn_rw_stats \
    .

# Final stage - Use scratch for maximum security (no vulnerabilities possible)
FROM scratch

# Copy CA certificates for HTTPS (required for API calls)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary from builder stage
COPY --from=builder /app/torn_rw_stats /app/torn_rw_stats

WORKDIR /app

# Use nonroot user (UID 65532)
USER 65532:65532

# Set default command
CMD ["./torn_rw_stats"]