 Build stage - Use latest .* Alpine image# Build stage - Use latest 1.26.2 Alpine image
 Build stage - Use latest .* Alpine imageFROM golang:1.26.4-alpine AS builder
 Build stage - Use latest .* Alpine image
 Build stage - Use latest .* Alpine image# Docker automatically provides these build args for multi-platform builds
 Build stage - Use latest .* Alpine imageARG TARGETOS
 Build stage - Use latest .* Alpine imageARG TARGETARCH
 Build stage - Use latest .* Alpine image
 Build stage - Use latest .* Alpine imageWORKDIR /app
 Build stage - Use latest .* Alpine image
 Build stage - Use latest .* Alpine image# Copy go mod files
 Build stage - Use latest .* Alpine imageCOPY go.mod go.sum ./
 Build stage - Use latest .* Alpine image
 Build stage - Use latest .* Alpine image# Download and verify dependencies
 Build stage - Use latest .* Alpine imageRUN go mod download && go mod verify
 Build stage - Use latest .* Alpine image
 Build stage - Use latest .* Alpine image# Copy source code
 Build stage - Use latest .* Alpine imageCOPY . .
 Build stage - Use latest .* Alpine image
 Build stage - Use latest .* Alpine image# Build the application with build info and security flags
 Build stage - Use latest .* Alpine image# Use TARGETOS and TARGETARCH instead of hardcoded values
 Build stage - Use latest .* Alpine imageRUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
 Build stage - Use latest .* Alpine image    -a \
 Build stage - Use latest .* Alpine image    -installsuffix cgo \
 Build stage - Use latest .* Alpine image    -ldflags='-w -s -extldflags "-static"' \
 Build stage - Use latest .* Alpine image    -o torn_rw_stats \
 Build stage - Use latest .* Alpine image    .
 Build stage - Use latest .* Alpine image
 Build stage - Use latest .* Alpine image# Final stage - Use distroless for security and minimal attack surface
 Build stage - Use latest .* Alpine imageFROM gcr.io/distroless/static:nonroot
 Build stage - Use latest .* Alpine image
 Build stage - Use latest .* Alpine image# Copy binary from builder stage
 Build stage - Use latest .* Alpine imageCOPY --from=builder /app/torn_rw_stats /app/torn_rw_stats
 Build stage - Use latest .* Alpine image
 Build stage - Use latest .* Alpine imageWORKDIR /app
 Build stage - Use latest .* Alpine image
 Build stage - Use latest .* Alpine image# Use built-in nonroot user (UID 65532)
 Build stage - Use latest .* Alpine imageUSER 65532:65532
 Build stage - Use latest .* Alpine image
 Build stage - Use latest .* Alpine image# Set default command
 Build stage - Use latest .* Alpine imageCMD ["./torn_rw_stats"]