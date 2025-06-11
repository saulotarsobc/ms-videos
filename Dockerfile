FROM golang:1.21-alpine AS builder

# Install ffmpeg
RUN apk add --no-cache ffmpeg

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ms-videos ./cmd/ms-videos

# Final stage
FROM alpine:latest

# Install ffmpeg and ca-certificates
RUN apk --no-cache add ffmpeg ca-certificates

# Create app directory
WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/ms-videos .

# Expose port (if needed for health checks)
EXPOSE 8080

# Run the binary
CMD ["./ms-videos"]

