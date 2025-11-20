# Owner: JeelRupapara (zeelrupapara@gmail.com)
# Multi-stage optimized Dockerfile for Platform Blueprint Service

# Stage 1: Dependencies caching
FROM golang:1.23-alpine AS deps
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Stage 2: Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    ca-certificates \
    git \
    make \
    protoc \
    protobuf-dev

# Set build environment
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    GO111MODULE=on

WORKDIR /app

# Copy dependencies from deps stage
COPY --from=deps /go/pkg /go/pkg

# Copy source code
COPY go.mod go.sum ./
COPY . .

# Build the application with optimizations
RUN go build -ldflags="-w -s -X main.version=$(git describe --tags --always --dirty) -X main.buildTime=$(date -u +%Y%m%d-%H%M%S)" \
    -a -installsuffix cgo -o blueprint cmd/main.go

# Stage 3: Security scanner (optional)
FROM aquasec/trivy:latest AS scanner
COPY --from=builder /app/blueprint /blueprint
RUN trivy fs --no-progress --security-checks vuln --exit-code 0 /blueprint

# Stage 4: Final minimal runtime
FROM gcr.io/distroless/static:nonroot

# Copy certificates for TLS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary
COPY --from=builder /app/blueprint /blueprint

# Use non-root user
USER nonroot:nonroot

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/blueprint", "-health"]

# Expose gRPC port
EXPOSE 50051

# Run the binary
ENTRYPOINT ["/blueprint"]