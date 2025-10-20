# Docker конфигурация для LLM Requester

## Структура файлов

- `Dockerfile` — описание Docker-образа для сервиса
- `docker-compose.yml` — оркестрация контейнера
- `.env.example` — шаблон переменных окружения
- `.dockerignore` — файлы, исключаемые из Docker-образа
- `Makefile` — удобные команды для управления

## Быстрый старт

### 1. Настройка переменных окружения

```bash
# Скопируйте пример файла .env
cp .env.example .env

# Отредактируйте .env и установите обязательные переменные:
# - YC_API_KEY (обязательно)
# - YC_FOLDER_ID (обязательно)
```

### 2. Запуск сервиса

```bash
# Соберите и запустите контейнер
docker-compose up -d

# Или используйте Makefile
make up
```

### 3. Проверка работы

```bash
# Просмотр логов
docker-compose logs -f llm-requester

# Проверка статуса
docker-compose ps

# Открыть Swagger UI
# http://localhost:8020/docs
```

### 4. Остановка

```bash
docker-compose down

# Или
make down
```

## Команды Makefile

| Команда | Описание |
|---------|----------|
| `make help` | Показать список доступных команд |
| `make build` | Собрать Docker-образ |
| `make up` | Запустить сервис (detached mode) |
| `make down` | Остановить и удалить контейнеры |
| `make restart` | Перезапустить сервис |
| `make logs` | Показать логи (follow mode) |
| `make ps` | Показать запущенные контейнеры |
| `make clean` | Остановить контейнеры и удалить volumes |
| `make dev-up` | Запуск для разработки (без Docker) |
| `make shell` | Открыть shell в контейнере |
| `make rebuild` | Пересобрать и перезапустить |

## Конфигурация

### Переменные окружения

#### Обязательные
- `YC_API_KEY` — API ключ Yandex Cloud Foundation Models
- `YC_FOLDER_ID` — Folder ID в Yandex Cloud

#### Опциональные
- `YC_BASE_URL` — базовый URL API (по умолчанию: `https://llm.api.cloud.yandex.net/v1`)
- `MAX_CONCURRENT` — максимальное число параллельных запросов (по умолчанию: 10)
- `HTTP_TIMEOUT` — таймаут HTTP-клиента в секундах (по умолчанию: 600)
- `MAX_RETRY_PROVIDER` — количество повторов при ошибках провайдера (по умолчанию: 2)
- `MAX_RETRY_JSON` — количество повторов при ошибках парсинга JSON (по умолчанию: 2)
- `DEFAULT_MODEL` — модель по умолчанию (по умолчанию: `qwen3-235b-a22b-fp8/latest`)
- `LOG_LEVEL` — уровень логирования: DEBUG, INFO, WARNING, ERROR (по умолчанию: INFO)
- `REQUEST_LOG` — включить логирование HTTP-запросов (по умолчанию: true)
- `REQUEST_ID_HEADER` — заголовок для request ID (по умолчанию: X-Request-ID)

### Порты

- `8020` — HTTP API сервиса (FastAPI + Uvicorn)

### Сети

Сервис подключен к двум сетям:
- `llm-network` — внутренняя сеть сервиса
- `common` — общая сеть для интеграции с другими сервисами проекта (если существует)

### Health check

Контейнер настроен с проверкой здоровья:
- Интервал: 30 секунд
- Таймаут: 10 секунд
- Количество попыток: 3
- Начальная задержка: 40 секунд

### Логирование

Настроено JSON-логирование с ротацией:
- Максимальный размер файла: 10MB
- Максимальное количество файлов: 3

## Разработка

### Локальный запуск без Docker

```bash
# Установите зависимости
pip install -r requirements.txt

# Запустите сервис
uvicorn LLMRequester.app.main:app --reload --port 8020

# Или используйте Makefile
make dev-up
```

### Отладка в контейнере

```bash
# Открыть shell в запущенном контейнере
docker-compose exec llm-requester /bin/sh

# Просмотр переменных окружения
docker-compose exec llm-requester env

# Просмотр логов uvicorn
docker-compose logs -f llm-requester
```

### Пересборка после изменений

```bash
# Полная пересборка
docker-compose down
docker-compose build --no-cache
docker-compose up -d

# Или одной командой
make rebuild
```

## Интеграция с основным docker-compose.yml

Чтобы добавить сервис в общий docker-compose.yml проекта:

```yaml
services:
  llm-requester:
    build:
      context: ./llm-requester
      dockerfile: Dockerfile
    container_name: llm-requester
    hostname: llm-requester
    ports:
      - "8020:8020"
    environment:
      YC_API_KEY: ${YC_API_KEY}
      YC_FOLDER_ID: ${YC_FOLDER_ID}
      # ... другие переменные
    networks:
      - common
    restart: unless-stopped
```

## Troubleshooting

### Контейнер не запускается
```bash
# Проверьте логи
docker-compose logs llm-requester

# Проверьте переменные окружения
docker-compose config
```

### Ошибки подключения к API
- Убедитесь, что `YC_API_KEY` и `YC_FOLDER_ID` установлены правильно
- Проверьте доступность `YC_BASE_URL` из контейнера
- Увеличьте `HTTP_TIMEOUT` если запросы долгие

### Проблемы с производительностью
- Настройте `MAX_CONCURRENT` в зависимости от нагрузки
- Мониторьте использование ресурсов: `docker stats llm-requester`

### Переполнение логов
- Логи автоматически ротируются (макс. 3 файла по 10MB)
- При необходимости измените настройки в `docker-compose.yml` → `logging`
