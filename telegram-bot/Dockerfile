FROM golang:1.23-alpine AS builder

WORKDIR /app

# Копируем go.mod и go.sum из корневого проекта
COPY go.mod go.sum ./

# Скачиваем зависимости
RUN go mod download

# Копируем весь исходный код
COPY . .

# Собираем приложение telegram-bot
RUN CGO_ENABLED=0 GOOS=linux go build -o /telegram-bot-app ./telegram-bot/cmd/main.go

# Финальный образ
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Копируем скомпилированное приложение
COPY --from=builder /telegram-bot-app .

# Копируем конфигурацию
COPY telegram-bot/config/local.yaml ./config/local.yaml

# Запускаем приложение
CMD ["./telegram-bot-app", "--config=./config/local.yaml"]