# ============================
# ======== BUILDER ===========
# ============================

FROM golang:1.24.2 AS builder

WORKDIR /app

# Сначала зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем всё остальное
COPY . .

ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -o bin/pr-reviewer-service ./cmd/app


# ============================
# ========= RUNNER ===========
# ============================

FROM alpine:3.19

WORKDIR /app

# SSL сертификаты, чтобы pgx мог коннектиться
RUN apk add --no-cache ca-certificates

# Копируем собранный бинарь
COPY --from=builder /app/bin/pr-reviewer-service ./pr-reviewer-service

EXPOSE 8080

CMD ["./pr-reviewer-service"]