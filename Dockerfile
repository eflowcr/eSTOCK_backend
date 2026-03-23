# Build stage
FROM golang:1.24.0-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o main ./cmd

# Run stage — minimal image, no secrets baked in
FROM alpine:3.20
RUN apk add --no-cache ca-certificates curl postgresql-client netcat-openbsd
WORKDIR /app
COPY --from=builder /app/main .
COPY db/migrations ./db/migrations
COPY scripts/wait-for.sh .
COPY scripts/start.sh .
RUN chmod +x wait-for.sh start.sh

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

CMD ["/app/main"]
ENTRYPOINT ["/app/start.sh"]
