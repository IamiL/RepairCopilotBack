#!/bin/bash

echo "Generating protobuf files..."

# Создание папки для генерации
mkdir -p pkg/user/v1

# Генерация Go кода
protoc \
    --proto_path=api/proto \
    --go_out=pkg \
    --go_opt=paths=source_relative \
    --go-grpc_out=pkg \
    --go-grpc_opt=paths=source_relative \
    api/proto/user/v1/user.proto

if [ $? -ne 0 ]; then
    echo "Error: Protobuf generation failed!"
    exit 1
fi

echo "Protobuf generation completed successfully!"