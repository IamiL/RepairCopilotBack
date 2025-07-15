#!/bin/bash

echo "Generating protobuf files..."

# Создание папки для генерации
mkdir -p pkg/tz/v1

# Генерация Go кода
protoc \
    --proto_path=proto \
    --go_out=pkg \
    --go_opt=paths=source_relative \
    --go-grpc_out=pkg \
    --go-grpc_opt=paths=source_relative \
    proto/tz/v1/tz.proto

if [ $? -ne 0 ]; then
    echo "Error: Protobuf generation failed!"
    exit 1
fi

echo "Protobuf generation completed successfully!"