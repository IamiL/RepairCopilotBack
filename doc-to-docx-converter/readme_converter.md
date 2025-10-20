# DOC to DOCX Converter Service

Микросервис для конвертации документов Microsoft Word из старого формата DOC в современный DOCX, работающий в Docker контейнере.

## Возможности

- ✅ Конвертация одного файла через REST API
- ✅ Пакетная конвертация нескольких файлов
- ✅ Автоматическая документация API (Swagger UI)
- ✅ Health check endpoint
- ✅ Поддержка CORS
- ✅ Логирование операций
- ✅ Docker контейнеризация

## Быстрый старт

### 1. Сборка и запуск с Docker Compose

```bash
# Клонирование или создание проекта
mkdir word-converter && cd word-converter

# Создание необходимых файлов:
# - app.py
# - Dockerfile
# - requirements.txt
# - docker-compose.yml

# Создание директорий для файлов
mkdir uploads outputs

# Сборка и запуск
docker-compose up --build
```

### 2. Запуск только с Docker

```bash
# Сборка образа
docker build -t word-converter .

# Запуск контейнера
docker run -d \
  --name word-converter \
  -p 8000:8000 \
  -v $(pwd)/uploads:/tmp/uploads \
  -v $(pwd)/outputs:/tmp/outputs \
  word-converter
```

## Использование API

### Проверка состояния сервиса

```bash
curl http://localhost:8000/health
```

### Конвертация одного файла

```bash
curl -X POST \
  http://localhost:8000/convert \
  -F "file=@document.doc" \
  --output converted.docx
```

### Конвертация с указанием имени выходного файла

```bash
curl -X POST \
  http://localhost:8000/convert?output_filename=my_document.docx \
  -F "file=@document.doc" \
  --output my_document.docx
```

### Пакетная конвертация

```bash
curl -X POST \
  http://localhost:8000/convert-batch \
  -F "files=@doc1.doc" \
  -F "files=@doc2.doc" \
  -F "files=@doc3.doc"
```

### Python клиент

```python
import requests

# Конвертация одного файла
with open('document.doc', 'rb') as f:
    files = {'file': f}
    response = requests.post('http://localhost:8000/convert', files=files)
    
    if response.status_code == 200:
        with open('converted.docx', 'wb') as output:
            output.write(response.content)
```

## API Endpoints

| Метод | Endpoint | Описание |
|-------|----------|----------|
| GET | `/` | Информация о сервисе |
| GET | `/health` | Проверка состояния |
| GET | `/docs` | Swagger UI документация |
| POST | `/convert` | Конвертация одного файла |
| POST | `/convert-batch` | Пакетная конвертация |

## Структура проекта

```
word-converter/
├── app.py              # Основной код сервиса
├── Dockerfile          # Конфигурация Docker образа
├── requirements.txt    # Python зависимости
├── docker-compose.yml  # Конфигурация Docker Compose
├── uploads/           # Директория для загруженных файлов
└── outputs/           # Директория для конвертированных файлов
```

## Настройка для Production

### 1. Добавьте nginx.conf

```nginx
events {
    worker_connections 1024;
}

http {
    upstream app {
        server word-converter:8000;
    }
    
    server {
        listen 80;
        client_max_body_size 100M;
        
        location / {
            proxy_pass http://app;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        }
    }
}
```

### 2. Запустите с профилем production

```bash
docker-compose --profile production up -d
```

## Переменные окружения

| Переменная | Описание | По умолчанию |
|------------|----------|--------------|
| `HOME` | Домашняя директория для LibreOffice | `/tmp` |
| `PYTHONUNBUFFERED` | Отключение буферизации вывода | `1` |

## Ограничения

- Максимальный размер файла: 100MB (настраивается)
- Таймаут конвертации: 30 секунд
- Поддерживаются только файлы с расширением .doc

## Мониторинг и логи

```bash
# Просмотр логов
docker-compose logs -f word-converter

# Просмотр статистики контейнера
docker stats word-converter
```

## Troubleshooting

### Ошибка "LibreOffice not available"

Убедитесь, что LibreOffice установлен в контейнере:

```bash
docker exec word-converter libreoffice --version
```

### Файлы не конвертируются

1. Проверьте логи контейнера
2. Убедитесь, что файл действительно в формате DOC
3. Проверьте права доступа к директориям uploads/outputs

### Недостаточно памяти

Увеличьте лимиты в docker-compose.yml:

```yaml
deploy:
  resources:
    limits:
      memory: 4G
```

## Лицензия

MIT