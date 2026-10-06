# Web UI build stage
FROM node:22-alpine AS web
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
# Vite writes to ../internal/ui/dist (-> /internal/ui/dist)
RUN npm run build

# Build stage
FROM golang:alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=web /internal/ui/dist ./internal/ui/dist
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/arch-repo ./cmd/server

# Final stage
FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/arch-repo /app/arch-repo

EXPOSE 8080

ENTRYPOINT ["/app/arch-repo"]
