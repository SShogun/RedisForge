# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy dependency manifests
COPY go.mod go.sum ./
RUN go mod download

# Copy application source
COPY . .

# Build the application
# CGO_ENABLED=0 ensures a static binary, perfect for scratch/alpine
RUN CGO_ENABLED=0 GOOS=linux go build -o redisforge ./cmd/redisforge

# Final stage
FROM alpine:3.19

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/redisforge .

# Expose the application port
EXPOSE 8080

# Run the binary
CMD ["./redisforge"]
