# Используем официальный Python образ
FROM python:3.11-slim

# Устанавливаем рабочую директорию
WORKDIR /app

# Копируем файл зависимостей
COPY requirements.txt .

# Устанавливаем зависимости
RUN pip install --no-cache-dir -r requirements.txt

# Копируем все файлы приложения
COPY . .

# Создаем директорию для логов
RUN mkdir -p /app/logs

# Открываем порт
EXPOSE 8082

# Запускаем приложение
CMD ["python", "main.py"]
