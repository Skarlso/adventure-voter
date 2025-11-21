# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and frontend for embedding
COPY backend/ ./backend/
COPY frontend/ ./frontend/
COPY main.go .

# Build the application with embedded frontend
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/adventure .

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder (frontend is embedded)
COPY --from=builder /app/adventure .

# Copy content only (frontend is embedded in binary)
COPY content/ ./content/

# Expose port
EXPOSE 8080

# Run the server (uses embedded frontend by default)
CMD ["./adventure", "-addr=:8080", "-content=./content/chapters", "-story=./content/story.yaml"]
