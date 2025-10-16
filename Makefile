.PHONY: all clean build deploy help generate-proto clean-proto build-chat-bot-linux
.DEFAULT_GOAL := help

# Переменные
TIMESTAMP := $(shell date +"%Y%m%d-%H%M%S")
GOOS := linux
GOARCH := amd64

# Пути к бинарникам
API_GATEWAY_BIN := api-gateway-app-v$(TIMESTAMP)
TZ_BOT_BIN := tz-bot-app-v$(TIMESTAMP)
USER_BIN := user-app-v$(TIMESTAMP)

# Исходники для отслеживания изменений
API_GATEWAY_SRC := $(shell find api-gateway-service -name '*.go' 2>/dev/null)
TZ_BOT_SRC := $(shell find tz-bot -name '*.go' 2>/dev/null)
USER_SRC := $(shell find user-service -name '*.go' 2>/dev/null)

# Default target
help:
	@echo "Доступные команды:"
	@echo "  all                - Собрать все сервисы"
	@echo "  build              - Собрать все бинарники"
	@echo "  clean              - Удалить все бинарники"
	@echo "  deploy             - Собрать и развернуть (git push)"
	@echo "  generate-proto     - Generate Go code from proto files"
	@echo "  clean-proto        - Remove generated proto files"
	@echo "  build-chat-bot-linux - Build chat-bot for Linux"
	@echo "  help               - Show this help message"

all: build

build: $(API_GATEWAY_BIN) $(TZ_BOT_BIN) $(USER_BIN)
	@echo "✅ Сборка завершена успешно!"
	@echo "📦 Созданы файлы:"
	@echo "   - $(API_GATEWAY_BIN)"
	@echo "   - $(TZ_BOT_BIN)"
	@echo "   - $(USER_BIN)"

$(API_GATEWAY_BIN): $(API_GATEWAY_SRC)
	@echo "🏗️  Компилирую api-gateway-service для Ubuntu..."
	@cd api-gateway-service && GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o ../$(API_GATEWAY_BIN) ./cmd/main.go

$(TZ_BOT_BIN): $(TZ_BOT_SRC)
	@echo "🏗️  Компилирую tz-bot для Ubuntu..."
	@cd tz-bot && GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o ../$(TZ_BOT_BIN) ./cmd/main.go

$(USER_BIN): $(USER_SRC)
	@echo "🏗️  Компилирую user-service для Ubuntu..."
	@cd user-service && GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o ../$(USER_BIN) ./cmd/main.go

clean:
	@echo "🧹 Удаление старых бинарников..."
	@rm -f api-gateway-app-*
	@rm -f tz-bot-app-*
	@rm -f user-app-*

deploy: build
	@echo "📝 Выполняю git операции..."
	@git add .
	@git commit -m "update" || true
	@git push
	@echo "🚀 Развертывание завершено успешно!"

# Generate gRPC code from proto files
generate-proto:
	@echo "Generating gRPC code..."
	protoc --go_out=chat-bot/pkg --go_opt=paths=source_relative \
		--go-grpc_out=chat-bot/pkg --go-grpc_opt=paths=source_relative \
		chat-bot/api/proto/chat/v1/chat.proto
	@echo "gRPC code generated successfully in chat-bot/pkg/chat/v1/"

# Clean generated proto files
clean-proto:
	@echo "Cleaning generated proto files..."
	@rm -rf chat-bot/pkg/chat/
	@echo "Generated proto files cleaned"

# Build search-bot for Linux
build-chat-bot-linux:
	@echo "Building chat-bot for Linux..."
	@mkdir -p chat-bot/bin
	@rm -f chat-bot/bin/chat-bot-linux-*
	@TIMESTAMP=$$(date +%Y%m%d_%H%M%S) && \
	cd chat-bot && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ../chat-bot-app-$$TIMESTAMP ./cmd/chat-bot/main.go && \
	echo "chat-bot built successfully for Linux at chat-bot/bin/chat-bot-linux-$$TIMESTAMP"