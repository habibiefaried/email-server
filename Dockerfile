# Multi-stage build
FROM golang:1.22.4-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o email-server ./cmd/email-server

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates bash

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/email-server .

# Create emails directory
RUN mkdir -p emails

# Expose ports
EXPOSE 25 48080

# Set default HTTP_PORT
ENV HTTP_PORT=48080

# Run the server
CMD ["./email-server"]
