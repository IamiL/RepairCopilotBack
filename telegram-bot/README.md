# Telegram Bot Service

Telegram-бот для общения с нейросетью с системой авторизации.

## Функциональность

- **Авторизация пользователей** через user-service (логин/пароль)
- **Чат с нейросетью** через chat-bot сервис
- **Управление состояниями пользователей** (неавторизован, ожидание логина, ожидание пароля, авторизован, в чате)
- **Хранение данных** в PostgreSQL (связь tg_user_id с user_id, текущий chat_id)

## Структура проекта

```
telegram-bot/
├── cmd/                    # Точка входа приложения
│   └── main.go
├── config/                 # Конфигурация
│   ├── config.go
│   └── local.yaml
├── internal/
│   ├── app/               # Инициализация приложения
│   │   └── app.go
│   ├── domain/            # Модели данных
│   │   └── models/
│   │       ├── user.go
│   │       └── user_state.go
│   ├── repository/        # Слой работы с БД
│   │   ├── repository.go
│   │   └── postgres_repository.go
│   ├── service/           # Бизнес-логика
│   │   └── service.go
│   └── telegram/          # Telegram bot handler
│       └── handler.go
├── migrations/            # SQL миграции
│   ├── 000001_init.up.sql
│   └── 000001_init.down.sql
├── pkg/                   # Общие пакеты
│   └── database/
│       └── postgres.go
├── .env.example          # Пример переменных окружения
├── Dockerfile
├── go.mod
└── README.md
```

## База данных

### Таблицы

#### telegram_users
Хранит связь между Telegram пользователями и пользователями системы:
- `id` - автоинкремент ID
- `tg_user_id` - ID пользователя в Telegram (уникальный)
- `user_id` - UUID пользователя из user-service
- `created_at`, `updated_at` - метки времени

#### user_states
Хранит состояние каждого пользователя:
- `id` - автоинкремент ID
- `tg_user_id` - ID пользователя в Telegram
- `state` - текущее состояние (unauthorized, awaiting_login, awaiting_password, authorized, in_chat)
- `login_attempt` - сохраненный логин при попытке входа
- `current_chat_id` - UUID текущего активного чата
- `created_at`, `updated_at` - метки времени

## Команды бота

- `/start` - Приветствие и информация о боте
- `/login` - Начать процесс авторизации
- `/startchat` - Начать новый чат с нейросетью
- `/endchat` - Завершить текущий чат

## Workflow пользователя

1. **Старт**: Пользователь отправляет `/start`
2. **Авторизация**:
   - Отправляет `/login`
   - Вводит логин
   - Вводит пароль
   - Система проверяет через user-service
   - При успехе сохраняется user_id и состояние меняется на "authorized"
3. **Начало чата**:
   - Отправляет `/startchat`
   - Система создает новый чат через chat-bot service
   - Сохраняется current_chat_id
   - Состояние меняется на "in_chat"
4. **Общение**:
   - Все текстовые сообщения отправляются в chat-bot service
   - Ответы нейросети приходят обратно пользователю
5. **Завершение чата**:
   - Отправляет `/endchat`
   - Чат завершается через chat-bot service
   - current_chat_id очищается
   - Состояние возвращается к "authorized"

## Настройка и запуск

### Требования

- Go 1.22+
- PostgreSQL
- Запущенные сервисы: user-service, chat-bot service
- Telegram Bot Token (получить у @BotFather)

### Миграции

Применить миграции к базе данных:

```bash
# Создать базу данных
createdb telegram_bot

# Применить миграции (используя migrate или вручную)
psql -d telegram_bot -f migrations/000001_init.up.sql
```

### Конфигурация

1. Скопировать `.env.example` в `.env` и заполнить:
```bash
cp .env.example .env
```

2. Обновить `config/local.yaml`:
```yaml
telegram:
  bot_token: "YOUR_TELEGRAM_BOT_TOKEN"

postgres:
  host: "localhost"
  port: "5432"
  database_name: "telegram_bot"
  username: "postgres"
  password: "postgres"

user_service:
  address: "localhost:8001"

chat_service:
  address: "localhost:50053"
```

### Запуск

```bash
# Установить зависимости
cd telegram-bot
go mod download

# Запустить бота
go run cmd/main.go --config=./config/local.yaml
```

### Docker

```bash
# Сборка образа
docker build -t telegram-bot -f telegram-bot/Dockerfile .

# Запуск контейнера
docker run -d --name telegram-bot \
  -v $(pwd)/telegram-bot/config:/root/config \
  telegram-bot
```

## Зависимости

- `github.com/go-telegram-bot-api/telegram-bot-api/v5` - Telegram Bot API
- `github.com/jmoiron/sqlx` - Расширение для database/sql
- `github.com/lib/pq` - PostgreSQL драйвер
- `github.com/google/uuid` - UUID генерация
- `github.com/ilyakaznacheev/cleanenv` - Парсинг конфигурации

## Интеграция с другими сервисами

### user-service
- Авторизация: `Login(login, password) -> user_id`
- Используется клиент из `repairCopilotBot/user-service/client`

### chat-bot service
- Создание сообщения: `CreateNewMessage(chat_id?, user_id, message) -> chat_id, response`
- Завершение чата: `FinishChat(chat_id, user_id)`
- Используется клиент из `repairCopilotBot/chat-bot/pkg/client/chat`