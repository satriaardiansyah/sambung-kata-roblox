# Step 1: Build the Go application
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy the go.mod first to leverage Docker cache
COPY go.mod ./

# Copy the rest of the source code
COPY . .

# Build main.go directly to avoid package declaration conflict with killerEnding.go (which is package data)
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server main.go

# Step 2: Run the Go application in a lightweight container
FROM alpine:latest

WORKDIR /app

# Install ca-certificates in case the app needs to make outbound HTTPS calls
RUN apk --no-cache add ca-certificates

# Copy binary from builder
COPY --from=builder /app/server /app/server

# Copy required runtime assets
COPY --from=builder /app/templates /app/templates
COPY --from=builder /app/deleted_words.json /app/deleted_words.json
COPY --from=builder /app/suggested_suffixes.json /app/suggested_suffixes.json

# Cloud Run defaults to listening on port 8080 or PORT env, main.go respects PORT env
EXPOSE 8000

# Set entrypoint
ENTRYPOINT ["/app/server"]
