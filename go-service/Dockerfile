FROM golang:1.25-bookworm AS builder

RUN apt-get update \
    && apt-get install -y --no-install-recommends build-essential pkg-config libsqlite3-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=1 GOOS=linux
RUN go build -o /out/api ./cmd/api

FROM debian:bookworm-slim AS runtime

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates libsqlite3-0 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /out/api /app/api

RUN mkdir -p /data

ENV PORT=5000
ENV AUTH_DB_DSN="file:/data/auth.db?_foreign_keys=on"
ENV LOGS_DB_DSN="file:/data/logs.db?_foreign_keys=on"

EXPOSE 5000

ENTRYPOINT ["/app/api"]
