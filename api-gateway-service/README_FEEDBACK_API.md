# Feedback API Endpoint

## POST /api/feedback

Создает фидбек для instance (invalid или missing error).

### Заголовки
- `X-Access-Token`: string (обязательный) - токен авторизации
- `Content-Type`: application/json

### Тело запроса
```json
{
  "instance_id": "uuid",           // ID instance (invalid_instance или missing_instance)
  "instance_type": "invalid|missing", // Тип instance: "invalid" или "missing"
  "feedback_mark": true|false|null,    // Оценка (optional)
  "feedback_comment": "string|null",   // Комментарий (optional)
  "user_id": "uuid"                    // ID пользователя (optional, для админов)
}
```

### Ответы

#### 201 Created
```json
{
  "success": true,
  "message": "Feedback created successfully"
}
```

#### 400 Bad Request
```json
{
  "error": "instance_id is required"
}
```
```json
{
  "error": "instance_type must be 'invalid' or 'missing'"
}
```
```json
{
  "error": "invalid instance_id format"
}
```

#### 401 Unauthorized
```json
{
  "error": "access token is missing"
}
```
```json
{
  "error": "invalid session"
}
```

#### 500 Internal Server Error
```json
{
  "error": "failed to create feedback"
}
```

### Примеры использования

#### Создание фидбека с оценкой и комментарием
```bash
curl -X POST http://localhost:8080/api/feedback \
  -H "X-Access-Token: your-token" \
  -H "Content-Type: application/json" \
  -d '{
    "instance_id": "123e4567-e89b-12d3-a456-426614174000",
    "instance_type": "invalid",
    "feedback_mark": true,
    "feedback_comment": "Это правильная ошибка"
  }'
```

#### Создание фидбека только с оценкой
```bash
curl -X POST http://localhost:8080/api/feedback \
  -H "X-Access-Token: your-token" \
  -H "Content-Type: application/json" \
  -d '{
    "instance_id": "123e4567-e89b-12d3-a456-426614174000",
    "instance_type": "missing",
    "feedback_mark": false
  }'
```

#### Создание фидбека только с комментарием
```bash
curl -X POST http://localhost:8080/api/feedback \
  -H "X-Access-Token: your-token" \
  -H "Content-Type: application/json" \
  -d '{
    "instance_id": "123e4567-e89b-12d3-a456-426614174000",
    "instance_type": "invalid",
    "feedback_comment": "Нужно уточнить формулировку"
  }'
```

### Логика работы

1. **Валидация токена** - проверка сессии пользователя
2. **Валидация параметров** - проверка обязательных полей и форматов
3. **Определение пользователя** - использование user_id из сессии или из запроса (для админов)
4. **Вызов gRPC** - отправка запроса в tz-bot сервис
5. **Логирование действия** - сохранение в action log
6. **Возврат ответа** - JSON с результатом операции

### Безопасность

- Требуется авторизация через токен
- Валидация всех входных параметров
- Логирование всех действий для аудита
- CORS настроен для разрешенных доменов