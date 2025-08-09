#!/bin/bash

set -e

echo "🚀 Starting protobuf generation..."

# Проверка наличия необходимых инструментов
check_command() {
    if ! command -v "$1" &> /dev/null; then
        echo "❌ Error: $1 is not installed"
        echo "📦 Installing $1..."
        return 1
    fi
    echo "✅ $1 is available"
    return 0
}

# Установка protoc через Homebrew для macOS
install_protoc() {
    echo "📦 Installing protoc via Homebrew..."
    if command -v brew &> /dev/null; then
        brew install protobuf
    else
        echo "❌ Homebrew not found. Please install Homebrew first: https://brew.sh"
        exit 1
    fi
}

# Установка Go плагинов для protoc
install_go_plugins() {
    echo "📦 Installing Go plugins for protoc..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
    
    # Добавляем GOPATH/bin к PATH если его нет
    export PATH="$PATH:$(go env GOPATH)/bin"
    echo "🔧 Added \$(go env GOPATH)/bin to PATH"
}

# Проверка и установка protoc
if ! check_command "protoc"; then
    install_protoc
fi

# Проверка и установка Go плагинов
if ! check_command "protoc-gen-go"; then
    install_go_plugins
fi

if ! check_command "protoc-gen-go-grpc"; then
    install_go_plugins
fi

# Убедимся что GOPATH/bin в PATH
export PATH="$PATH:$(go env GOPATH)/bin"

echo "🏗️  Creating directories..."
mkdir -p pkg/tz/v1

echo "🔧 Generating protobuf files..."

# Генерация Go кода с более подробным выводом
protoc \
    --proto_path=proto \
    --go_out=pkg \
    --go_opt=paths=source_relative \
    --go_opt=Mproto/tz/v1/tz.proto=repairCopilotBot/tz-bot/pkg/tz/v1 \
    --go-grpc_out=pkg \
    --go-grpc_opt=paths=source_relative \
    --go-grpc_opt=Mproto/tz/v1/tz.proto=repairCopilotBot/tz-bot/pkg/tz/v1 \
    proto/tz/v1/tz.proto

if [ $? -eq 0 ]; then
    echo "✅ Protobuf generation completed successfully!"
    echo "📁 Generated files:"
    find pkg -name "*.pb.go" -exec echo "  - {}" \;
else
    echo "❌ Error: Protobuf generation failed!"
    echo "🔍 Debug info:"
    echo "  - protoc version: $(protoc --version 2>/dev/null || echo 'not found')"
    echo "  - protoc-gen-go: $(which protoc-gen-go 2>/dev/null || echo 'not found')"
    echo "  - protoc-gen-go-grpc: $(which protoc-gen-go-grpc 2>/dev/null || echo 'not found')"
    echo "  - GOPATH: $(go env GOPATH)"
    echo "  - PATH: $PATH"
    exit 1
fi