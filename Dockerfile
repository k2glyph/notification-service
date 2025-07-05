# --- Stage 1: Frontend Build ---
FROM node:18-alpine AS frontend-builder
LABEL MAINTAINER="Jules AI Assistant"

WORKDIR /app/web

# Copy package.json and package-lock.json (or yarn.lock)
COPY web/package.json web/package-lock.json ./
# If using yarn, it would be:
# COPY web/package.json web/yarn.lock ./

# Install dependencies
# Assuming npm, use yarn if yarn.lock was copied
RUN npm install
# If using yarn:
# RUN yarn install

# Copy the rest of the web application source code
COPY web/. ./

# Build the React application
RUN npm run build
# If using yarn:
# RUN yarn build

# --- Stage 2: Go Backend Build (existing builder) ---
FROM golang:1.21-alpine AS go-builder
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

# --- Final Stage: Runtime ---
FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the static Go binary from the go-builder stage
COPY --from=go-builder /app/notification-service /app/notification-service

# Copy the built frontend assets from the frontend-builder stage
# The Go application in server.go is configured to serve from "./web/build/"
# So, we need to place the contents of frontend-builder's /app/web/build into /app/web/build here.
COPY --from=frontend-builder /app/web/build ./web/build

# No longer need to copy the entire 'web' source directory for runtime
# COPY web ./web

EXPOSE 8080
ENTRYPOINT [ "/app/notification-service" ]