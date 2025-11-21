# Reverse Proxy Deployment Guide

This application is designed to run behind a reverse proxy. The proxy handles security concerns while the Go backend focuses on application logic.

## Architecture

```
Internet -> Reverse Proxy (Port 80/443) -> Go Backend (Port 8080)
          │
          ├─ TLS/HTTPS termination
          ├─ CORS enforcement
          ├─ Rate limiting
          ├─ Request size limits
          ├─ Static file caching
          └─ Access logging
```

---

## Nginx Configuration

### Basic Setup

```nginx
upstream adventure_voter {
    server localhost:8080;
}

server {
    listen 80;
    server_name yourapp.com;

    # Redirect HTTP to HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name yourapp.com;

    # SSL Configuration
    ssl_certificate /etc/letsencrypt/live/yourapp.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/yourapp.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    # Security Headers
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;

    # CORS Headers
    add_header Access-Control-Allow-Origin "https://yourapp.com" always;
    add_header Access-Control-Allow-Methods "GET, POST, OPTIONS" always;
    add_header Access-Control-Allow-Headers "Content-Type" always;

    # Rate Limiting Zones
    limit_req_zone $binary_remote_addr zone=api_limit:10m rate=10r/s;
    limit_req_zone $binary_remote_addr zone=ws_limit:10m rate=5r/s;

    # Request Size Limits
    client_max_body_size 1M;

    # WebSocket endpoint
    location /ws {
        limit_req zone=ws_limit burst=10 nodelay;

        proxy_pass http://adventure_voter;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSocket timeouts
        proxy_read_timeout 3600s;
        proxy_send_timeout 3600s;
    }

    # API endpoints
    location /api/ {
        limit_req zone=api_limit burst=20 nodelay;

        proxy_pass http://adventure_voter;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Static files
    location / {
        proxy_pass http://adventure_voter;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;

        # Cache static files
        expires 1h;
        add_header Cache-Control "public, immutable";
    }

    # Access logs
    access_log /var/log/nginx/adventure_voter_access.log combined;
    error_log /var/log/nginx/adventure_voter_error.log warn;
}
```

### Testing Nginx Config

```bash
# Test configuration
sudo nginx -t

# Reload
sudo systemctl reload nginx
```

---

## Traefik Configuration

### docker-compose.yml

```yaml
version: '3.8'

services:
  traefik:
    image: traefik:v2.10
    command:
      - "--api.insecure=true"
      - "--providers.docker=true"
      - "--entrypoints.web.address=:80"
      - "--entrypoints.websecure.address=:443"
      - "--certificatesresolvers.letsencrypt.acme.email=you@example.com"
      - "--certificatesresolvers.letsencrypt.acme.storage=/letsencrypt/acme.json"
      - "--certificatesresolvers.letsencrypt.acme.httpchallenge.entrypoint=web"
    ports:
      - "80:80"
      - "443:443"
      - "8080:8080" # Dashboard
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./letsencrypt:/letsencrypt

  adventure-voter:
    build: backend
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.adventure.rule=Host(`yourapp.com`)"
      - "traefik.http.routers.adventure.entrypoints=websecure"
      - "traefik.http.routers.adventure.tls.certresolver=letsencrypt"

      # Rate limiting
      - "traefik.http.middlewares.ratelimit.ratelimit.average=10"
      - "traefik.http.middlewares.ratelimit.ratelimit.burst=20"
      - "traefik.http.routers.adventure.middlewares=ratelimit"

      # CORS
      - "traefik.http.middlewares.cors.headers.accesscontrolallowmethods=GET,POST,OPTIONS"
      - "traefik.http.middlewares.cors.headers.accesscontrolalloworigin=https://yourapp.com"
      - "traefik.http.middlewares.cors.headers.accesscontrolmaxage=100"
```

---

## Caddy Configuration

### Caddyfile

```caddy
yourapp.com {
    # Automatic HTTPS

    # Rate limiting
    rate_limit {
        zone api {
            key {remote_host}
            events 10
            window 1s
        }
    }

    # CORS
    @cors_preflight method OPTIONS
    handle @cors_preflight {
        header Access-Control-Allow-Origin "https://yourapp.com"
        header Access-Control-Allow-Methods "GET, POST, OPTIONS"
        header Access-Control-Allow-Headers "Content-Type"
        respond 204
    }

    header {
        Access-Control-Allow-Origin "https://yourapp.com"
        X-Frame-Options "SAMEORIGIN"
        X-Content-Type-Options "nosniff"
        Strict-Transport-Security "max-age=31536000"
    }

    # WebSocket support
    @websocket {
        path /ws
    }
    reverse_proxy @websocket localhost:8080 {
        header_up X-Real-IP {remote_host}
    }

    # API endpoints
    reverse_proxy /api/* localhost:8080 {
        header_up X-Real-IP {remote_host}
    }

    # Static files
    reverse_proxy localhost:8080
}
```

---

## Docker Deployment

### Full Stack with Nginx

**docker-compose.yml:**

```yaml
version: '3.8'

services:
  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - ./ssl:/etc/ssl/certs:ro
      - ./logs:/var/log/nginx
    depends_on:
      - backend
    restart: unless-stopped

  backend:
    build: ./backend
    expose:
      - "8080"
    volumes:
      - ./content:/app/content:ro
      - ./frontend:/app/frontend:ro
    environment:
      - ADDR=:8080
    restart: unless-stopped
```

**Dockerfile (backend):**

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o server ./main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /build/server .
COPY content ./content
COPY frontend ./frontend

EXPOSE 8080
CMD ["./server", "-addr=:8080"]
```
