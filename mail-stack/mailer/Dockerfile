# --- Build stage ---
FROM golang:1.22-alpine AS build
WORKDIR /app

# Кэш для ускорения сборок (опционально, но полезно)
ENV GOPROXY=https://proxy.golang.org,direct
# Скопируем исходник
COPY main.go .

# Инициализируем модуль и подтянем зависимости по импортам из main.go
# (если go.mod уже существует в проекте — строки init/tidy его просто обновят)
RUN go mod init intbis.ru/mailer || true \
 && go mod tidy \
 && CGO_ENABLED=0 go build -o /mailer -ldflags="-s -w" .

# --- Runtime stage ---
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /mailer /usr/local/bin/mailer
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/mailer"]

