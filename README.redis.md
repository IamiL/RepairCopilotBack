# Redis Development Environment Setup

## Обзор

Этот Docker Compose файл настраивает полную Redis среду разработки с веб-интерфейсами для управления данными.

## Компоненты

### 1. Redis Server (`redis:7.4-alpine`)
- **Порт**: `6379`
- **Пароль**: `Kx8#mP3vR9$wL2@nQ7`
- **Память**: 256MB с LRU eviction policy
- **Данные**: Персистентное хранение в Docker volume

### 2. RedisInsight (Официальный GUI от Redis)
- **URL**: http://localhost:5540
- **Описание**: Современный веб-интерфейс с AI-помощником
- **Функции**: 
  - Визуализация данных
  - Профилирование команд
  - Анализ памяти
  - Redis Copilot (AI-помощник)

### 3. Redis Commander (Легковесный веб-клиент)
- **URL**: http://localhost:8081/redis-commander
- **Логин**: `admin`
- **Пароль**: `Zt7$nF9kW3#mX5@pQ8`
- **Описание**: Простой браузерный интерфейс для управления Redis

### 4. Redis Stack Server (Опционально)
- **Порт**: `6380` 
- **Профиль**: `extended`
- **Модули**: RedisJSON, RedisSearch, RedisGraph, RedisTimeSeries, RedisBloom

## Запуск

### Базовая конфигурация (Redis + Web-клиенты)
```bash
docker-compose -f docker-compose.redis.yml up -d
```

### С Redis Stack (расширенные модули)
```bash
docker-compose -f docker-compose.redis.yml --profile extended up -d
```

## Подключение к Redis

### Из приложения
```bash
Host: localhost (или IP сервера)
Port: 6379
Password: Kx8#mP3vR9$wL2@nQ7
```

### Пример конфигурации для Go
```go
import "github.com/gomodule/redigo/redis"

pool := &redis.Pool{
    MaxIdle:     10,
    MaxActive:   100,
    IdleTimeout: 240 * time.Second,
    Dial: func() (redis.Conn, error) {
        c, err := redis.Dial("tcp", "localhost:6379")
        if err != nil {
            return nil, err
        }
        if _, err := c.Do("AUTH", "Kx8#mP3vR9$wL2@nQ7"); err != nil {
            c.Close()
            return nil, err
        }
        return c, nil
    },
}
```

### Пример для api-gateway-service config.yaml
```yaml
redis:
  address: "localhost:6379"
  password: "Kx8#mP3vR9$wL2@nQ7"
  db: 0
  max_idle: 10
  max_active: 100
  idle_timeout: 240s
```

## Web-интерфейсы

### RedisInsight (Рекомендуемый)
1. Откройте http://localhost:5540
2. Добавьте базу данных:
   - Host: `repair-copilot-redis`
   - Port: `6379`
   - Password: `Kx8#mP3vR9$wL2@nQ7`

**Преимущества RedisInsight:**
- Официальный инструмент от Redis
- AI-помощник Redis Copilot
- Анализ производительности
- Визуализация структур данных
- Профилирование команд в реальном времени

### Redis Commander (Альтернатива)
1. Откройте http://localhost:8081/redis-commander
2. Логин: `admin`, Пароль: `Zt7$nF9kW3#mX5@pQ8`
3. База подключается автоматически

**Преимущества Redis Commander:**
- Простой и быстрый
- Древовидная навигация по ключам
- Пакетное удаление ключей
- Легковесный

## CLI Подключение

```bash
# Подключение к основному Redis
docker exec -it repair-copilot-redis redis-cli -a "Kx8#mP3vR9$wL2@nQ7"

# Подключение к Redis Stack
docker exec -it repair-copilot-redis-stack redis-cli -a "Kx8#mP3vR9$wL2@nQ7"
```

## Мониторинг

### Просмотр логов
```bash
docker-compose -f docker-compose.redis.yml logs redis
docker-compose -f docker-compose.redis.yml logs redis-insight
docker-compose -f docker-compose.redis.yml logs redis-commander
```

### Проверка здоровья
```bash
docker-compose -f docker-compose.redis.yml ps
```

## Безопасность

### Пароли
- **Redis**: `Kx8#mP3vR9$wL2@nQ7`
- **Redis Commander**: `admin` / `Zt7$nF9kW3#mX5@pQ8`

### Отключенные команды (для безопасности)
- `FLUSHDB` - удаление базы данных
- `FLUSHALL` - удаление всех баз
- `DEBUG` - отладочные команды
- `CONFIG` - переименована в `CONFIG_b7x9m2w4p1`

## Производство

### Изменения для продакшена:
1. Смените пароли в `.env` файле
2. Ограничьте доступ по сети
3. Настройте TLS/SSL
4. Увеличьте лимиты памяти
5. Настройте бэкапы

### Пример .env файла:
```env
REDIS_PASSWORD=your_secure_password_here
REDIS_COMMANDER_USER=your_admin_user
REDIS_COMMANDER_PASSWORD=your_admin_password
```

## Остановка

```bash
docker-compose -f docker-compose.redis.yml down

# С удалением данных
docker-compose -f docker-compose.redis.yml down -v
```

## Полезные команды Redis

```bash
# Информация о сервере
INFO

# Список всех ключей
KEYS *

# Мониторинг команд в реальном времени  
MONITOR

# Статистика использования памяти
MEMORY USAGE key_name

# Медленные запросы
SLOWLOG GET 10
```