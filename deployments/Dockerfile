FROM golang:1.24-alpine AS builder

WORKDIR /src

# Устанавливаем зависимости для сборки
RUN apk add --no-cache ca-certificates git

# Копируем go.mod и go.sum
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь код
COPY . .

# Собираем бинарник
RUN go build -o /out/ride-hail-system .

# Финальный образ
FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /out/ride-hail-system ./ride-hail-system

# Переменная для конфигов
ENV CONFIG_DIR=/app/config

# Запуск
CMD ["./ride-hail-system", "-service=all"]