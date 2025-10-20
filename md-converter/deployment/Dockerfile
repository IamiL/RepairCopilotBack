FROM python:3.11-slim

WORKDIR /app

# Копируем файл зависимостей
COPY requirements.txt .

# Устанавливаем зависимости
RUN pip install --no-cache-dir -r requirements.txt

# Копируем исходный код приложения
COPY app/ ./app/
COPY main.py .

# Открываем порт
EXPOSE 8089

# Запускаем приложение
CMD ["python", "main.py"]