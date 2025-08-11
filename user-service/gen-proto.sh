#!/bin/bash
set -e

echo "Generating protobuf files..."

PROTO_FILE="api/proto/user/v1/user.proto"
OUT_DIR="pkg/user/v1"

# Проверка наличия protoc
if ! command -v protoc >/dev/null 2>&1; then
    echo "Error: protoc not found. Please install it first."
    echo "MacOS: brew install protobuf"
    exit 1
fi

# Проверка наличия плагинов Go
if ! command -v protoc-gen-go >/dev/null 2>&1; then
    echo "Installing protoc-gen-go..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
fi

if ! command -v protoc-gen-go-grpc >/dev/null 2>&1; then
    echo "Installing protoc-gen-go-grpc..."
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
fi

# Создание директории для генерации
mkdir -p "$OUT_DIR"

# Генерация кода
protoc \
    --proto_path=api/proto \
    --go_out=pkg \
    --go_opt=paths=source_relative \
    --go-grpc_out=pkg \
    --go-grpc_opt=paths=source_relative \
    "$PROTO_FILE"

echo "✅ Protobuf generation completed successfully!"
