#!/bin/bash

# Скрипт для сборки и развертывания api-gateway-service и tz-bot под Ubuntu
# Запускается на macOS для кросс-компиляции

set -e

echo "🔨 Начинаю сборку проекта..."

# Очистка старых бинарников
echo "🧹 Удаление старых бинарников..."
rm -f api-gateway-app-*
rm -f tz-bot-app-*

# Создание временной метки для версии
TIMESTAMP=$(date +"%Y%m%d-%H%M%S")
echo "📅 Версия: v${TIMESTAMP}"

# Сборка api-gateway-service для Ubuntu
echo "🏗️  Компилирую api-gateway-service для Ubuntu..."
cd api-gateway-service
GOOS=linux GOARCH=amd64 go build -o ../api-gateway-app-v${TIMESTAMP} ./cmd/main.go
cd ..

# Сборка tz-bot для Ubuntu  
echo "🏗️  Компилирую tz-bot для Ubuntu..."
cd tz-bot
GOOS=linux GOARCH=amd64 go build -o ../tz-bot-app-v${TIMESTAMP} ./cmd/main.go
cd ..

echo "✅ Сборка завершена успешно!"
echo "📦 Созданы файлы:"
echo "   - api-gateway-app-v${TIMESTAMP}"
echo "   - tz-bot-app-v${TIMESTAMP}"

# Git операции
echo "📝 Выполняю git операции..."
git add .
git commit -m "update"
git push

echo "🚀 Развертывание завершено успешно!"