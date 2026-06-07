# Stage 1: Build the Go binary
FROM golang:1.26-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set the working directory inside the container
WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build a statically linked Go binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app/api ./cmd/api

# Stage 2: Create the final minimal image
FROM alpine:3.19

# Install ca-certificates and timezone data
RUN apk add --no-cache ca-certificates tzdata

# Create a non-root user for secure execution
RUN adduser -D -g '' appuser

# Set the working directory
WORKDIR /app

# Copy the compiled binary from the builder stage
COPY --from=builder /app/api .

# Use the non-root user
USER appuser

# Expose port (Railway overrides this, but good for documentation/local running)
EXPOSE 8082

# Run the binary
CMD ["./api"]
