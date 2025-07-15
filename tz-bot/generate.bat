 @echo off
  echo Generating protobuf files...

  REM Создание папки для генерации
  if not exist pkg\tz\v1 mkdir pkg\tz\v1

  REM Генерация Go кода
  protoc ^
      --proto_path=proto ^
      --go_out=pkg ^
      --go_opt=paths=source_relative ^
      --go-grpc_out=pkg ^
      --go-grpc_opt=paths=source_relative ^
      proto/tz/v1/tz.proto

  if %errorlevel% neq 0 (
      echo Error: Protobuf generation failed!
      exit /b 1
  )

  echo Protobuf generation completed successfully!