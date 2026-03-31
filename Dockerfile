# Stage 1: Build frontend
FROM node:20-alpine AS frontend
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Stage 2: Build Go binary
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Copy built frontend to embed location (required for //go:embed dist)
COPY --from=frontend /app/web/dist ./internal/webserver/static/dist/
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.Version=0.0.1 -X spoutmc/internal/webserver.WriteRoutesOnStart=false" \
    -o spoutmc \
    ./cmd/spoutmc

# Stage 3: Minimal runtime image
FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/spoutmc ./spoutmc
EXPOSE 3000
ENTRYPOINT ["/app/spoutmc"]
