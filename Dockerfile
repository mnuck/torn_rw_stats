# Build stage - Use latest 1.25.4 Alpine image
FROM golang:1.25.4-alpine AS builder

# Docker automatically provides these build args for multi-platform builds
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download and verify dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application with build info and security flags
# Use TARGETOS and TARGETARCH instead of hardcoded values
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -a \
    -installsuffix cgo \
    -ldflags='-w -s -extldflags "-static"' \
    -o torn_rw_stats \
    .

# Final stage - Use distroless for security and minimal attack surface
FROM gcr.io/distroless/static:nonroot

# Copy binary from builder stage
COPY --from=builder /app/torn_rw_stats /app/torn_rw_stats

WORKDIR /app

# Use built-in nonroot user (UID 65532)
USER 65532:65532

# Set default command
CMD ["./torn_rw_stats"]