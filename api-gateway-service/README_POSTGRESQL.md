# PostgreSQL интеграция в API Gateway Service

## Добавленная функциональность

В API Gateway Service добавлена поддержка PostgreSQL для журналирования действий пользователей.

### Компоненты

1. **Конфигурация**: `internal/repository/postgres/postgres.go`
   - Настройки подключения к PostgreSQL
   - Создание пула соединений

2. **Модель**: `internal/repository/postgres/action_log/action_log.go`
   - Репозиторий для работы с журналом действий
   - Методы для создания записей и получения всех записей

3. **API эндпоинт**: `GET /api/action-logs`
   - Получение всех записей журнала действий (от новых к старым)
   - Требует аутентификации

4. **Миграция**: `migrations/001_create_action_logs.sql`
   - SQL скрипт для создания таблицы action_logs

### Таблица action_logs

```sql
CREATE TABLE action_logs (
    id SERIAL PRIMARY KEY,
    action VARCHAR(255) NOT NULL,
    user_id UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

### Конфигурация

Добавьте в `config/config.yaml`:

```yaml
postgres:
  host: localhost
  port: "5432"
  database_name: api_gateway
  username: postgres
  password: postgres
  max_connections: 10
```

### Примеры использования

1. **Логирование действия** (уже реализовано в login handler):
```go
err = actionLogRepo.CreateActionLog(ctx, "User login: "+username, userID)
```

2. **Получение всех логов** через API:
```bash
curl -X GET "http://localhost:8080/api/action-logs" -H "Cookie: auth_token=your_session_token"
```

### Настройка базы данных

1. Создайте базу данных:
```sql
CREATE DATABASE api_gateway;
```

2. Запустите миграцию:
```bash
goose -dir migrations postgres "postgresql://user:password@localhost:5432/api_gateway" up
```

### Методы репозитория

- `CreateActionLog(ctx, action, userID)` - создание новой записи
- `GetAllActionLogs(ctx)` - получение всех записей (от новых к старым)

### Интеграция в обработчики

Для добавления логирования в другие обработчики:

1. Добавьте параметр `actionLogRepo repository.ActionLogRepository` в конструктор обработчика
2. Вызывайте `actionLogRepo.CreateActionLog(ctx, "Описание действия", userID)` в нужном месте
3. Обновите конфигурацию роутера в `internal/app/http/app.go`