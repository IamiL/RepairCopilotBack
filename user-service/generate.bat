@echo off
echo Generating protobuf files...

REM Создание папки для генерации
if not exist "pkg\user\v1" mkdir pkg\user\v1

REM Генерация Go кода
protoc ^
    --proto_path=api\proto ^
    --go_out=pkg ^
    --go_opt=paths=source_relative ^
    --go-grpc_out=pkg ^
    --go-grpc_opt=paths=source_relative ^
    api\proto\user\v1\user.proto

if %errorlevel% neq 0 (
    echo Error: Protobuf generation failed!
    pause
    exit /b %errorlevel%
)

echo Protobuf generation completed successfully!
pause