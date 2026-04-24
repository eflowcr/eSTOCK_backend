# Build stage
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o main ./cmd

# Run stage — minimal image, no secrets baked in, runs as non-root
FROM alpine:3.20
RUN apk add --no-cache ca-certificates curl && \
    addgroup -S -g 1001 appgroup && \
    adduser -S -u 1001 -G appgroup appuser
WORKDIR /app
COPY --from=builder --chown=appuser:appgroup /app/main .
COPY --chown=appuser:appgroup db/migrations ./db/migrations

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

CMD ["/app/main"]
