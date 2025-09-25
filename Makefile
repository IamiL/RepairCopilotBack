.PHONY: generate-proto clean-proto build-chat-bot-linux help

# Default target
help:
	@echo "Available targets:"
	@echo "  generate-proto     - Generate Go code from proto files"
	@echo "  clean-proto        - Remove generated proto files"
	@echo "  build-chat-bot-linux - Build chat-bot for Linux"
	@echo "  help              - Show this help message"

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

# Build chat-bot for Linux
build-chat-bot-linux:
	@echo "Building chat-bot for Linux..."
	@mkdir -p chat-bot/bin
	@rm -f chat-bot/bin/chat-bot-linux-*
	@TIMESTAMP=$$(date +%Y%m%d_%H%M%S) && \
	cd chat-bot && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ../chat-bot-app-$$TIMESTAMP ./cmd/chat-bot/main.go && \
	echo "chat-bot built successfully for Linux at chat-bot/bin/chat-bot-linux-$$TIMESTAMP"