# Используем официальный образ Go 1.24.6
FROM golang:1.24-alpine AS builder

WORKDIR /src

# Устанавливаем необходимые зависимости
RUN apk add --no-cache ca-certificates git

# Копируем go.mod и go.sum для кеширования зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь проект
COPY . .

# Собираем бинарник с оптимизацией
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /out/ride-hail-system \
    .

# Финальный образ
FROM alpine:3.20

WORKDIR /app

# Устанавливаем необходимые пакеты
RUN apk add --no-cache ca-certificates tzdata

# Копируем бинарник
COPY --from=builder /out/ride-hail-system ./ride-hail-system

# Переменная окружения для конфигурации
ENV CONFIG_DIR=/app/config

# Точка входа
CMD ["./ride-hail-system", "-service=all"]