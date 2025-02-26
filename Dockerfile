# Build stage
FROM --platform=$BUILDPLATFORM golang:1.21.7 AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
ARG TARGETARCH
RUN CGO_ENABLED=0 GOARCH=$TARGETARCH go build -o /go/bin/subscriber cmd/subscriber/main.go
RUN CGO_ENABLED=0 GOARCH=$TARGETARCH go build -o /go/bin/publisher cmd/publisher/main.go

# Final stage
FROM --platform=$TARGETPLATFORM ubuntu:22.04

# Install CA certificates
RUN apt-get update && \
    apt-get install -y ca-certificates tzdata curl && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy binaries from builder
COPY --from=builder /go/bin/subscriber /usr/local/bin/subscriber
COPY --from=builder /go/bin/publisher /usr/local/bin/publisher

# Set environment variables
ENV DB_HOST=postgres \
    DB_PORT=5432 \
    DB_USER=casino \
    DB_PASSWORD=casino \
    DB_NAME=casino \
    DB_SSL_MODE=disable \
    NATS_URL=nats://nats:4222

# Command will be specified in docker-compose.yml
CMD ["subscriber"] 