# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY backend/ ./backend/

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./backend/main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server .

# Copy frontend and content
COPY frontend/ ./frontend/
COPY content/ ./content/

# Expose port
EXPOSE 8080

# Run the server
CMD ["./server", "-addr=:8080", "-content=./content/chapters", "-story=./content/story.yaml", "-static=./frontend"]
