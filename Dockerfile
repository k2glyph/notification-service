FROM golang:1.21-alpine AS builder
LABEL MAINTAINER="Dinesh Katwal<dinesh@auzmor.com>"

WORKDIR /app

# Copy go.mod and go.sum first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download
RUN go mod verify

# Copy the rest of the application source code
COPY . .

# Build the application
# Using CGO_ENABLED=0 for a static binary if no CGO is needed, common for Alpine
# Using -ldflags="-w -s" to make the binary smaller
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags="-w -s" -o notification-service cmd/server/main.go

# --- Final Stage ---
FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the static binary from the builder stage
COPY --from=builder /app/notification-service /app/notification-service

# Copy web assets (templates)
# Ensure this path matches where your app expects to find them if not using embed
COPY web ./web

EXPOSE 8080
ENTRYPOINT [ "/app/notification-service" ]