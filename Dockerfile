# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS builder
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/server ./cmd/server

FROM alpine:3.20
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata && adduser -D -g '' appuser

COPY --from=builder /out/server /app/server
COPY --chown=appuser:appuser configs /app/configs
COPY --chown=appuser:appuser .env.example /app/.env.example

# Writable data dir for the SQLite DB (user→diagnosis mapping). Owned by
# appuser so the non-root process can create/write the DB when a volume is
# mounted here in production.
RUN mkdir -p /app/data && chown appuser:appuser /app/data

USER appuser

EXPOSE 8080

ENV PORT=8080
ENV BOT_CONFIG_PATH=/app/configs/faq_seed.yaml

HEALTHCHECK --interval=30s --timeout=5s --retries=3 CMD wget -qO- http://127.0.0.1:8080/healthz || exit 1

ENTRYPOINT ["/app/server"]
