# Руководство по системе логирования

## Что изменилось

### Удалено (не нужно для логов)
- Prometheus (метрики)
- Jaeger (трейсы)

### Оставлено (только для логов)
- Grafana - веб-интерфейс для просмотра логов
- Loki - хранилище логов
- Promtail - сборщик логов из Docker контейнеров

## Доступ к Grafana

**URL:** http://localhost:3001

**Учетные данные:**
- Логин: `admin`
- Пароль: указан в `.env` файле в переменной `GRAFANA_ADMIN_PASSWORD` (по умолчанию: `GrafanaLog2025!Secure`)

## Как пользоваться

### 1. Запуск системы логирования

```bash
docker-compose up -d
```

### 2. Открытие веб-интерфейса

1. Откройте браузер и перейдите на http://localhost:3001
2. Введите логин `admin` и пароль из `.env` файла (переменная `GRAFANA_ADMIN_PASSWORD`)
3. После входа автоматически откроется dashboard "Docker Logs Dashboard"

### 3. Просмотр логов

На главном dashboard вы увидите:

- **Выпадающий список "Контейнер"** - выберите один или несколько контейнеров для просмотра логов
  - Можно выбрать "All" для просмотра всех контейнеров сразу

- **Выпадающий список "Поток"** - выберите stdout или stderr
  - stdout - обычные логи
  - stderr - логи ошибок
  - "All" - все логи

- **Поле "Поиск"** - введите текст для поиска в логах
  - Например: "error", "WARNING", "user_id"

- **Временной диапазон** - справа вверху можно выбрать период:
  - Last 5 minutes
  - Last 1 hour
  - Last 6 hours (по умолчанию)
  - Custom range

### 4. Обновление логов

Dashboard автоматически обновляется каждые 10 секунд. Вы можете:
- Изменить интервал обновления в правом верхнем углу
- Нажать кнопку "Refresh" для ручного обновления

### 5. Альтернативный способ - Explore

Если вам нужен более гибкий поиск:

1. В левом меню выберите "Explore" (иконка компаса)
2. Убедитесь что выбран источник данных "Loki"
3. Используйте LogQL запросы, например:
   - `{container="llm-requester-service"}` - логи конкретного контейнера
   - `{container=~".*service"}` - логи всех контейнеров с названием заканчивающимся на "service"
   - `{container="prompt-builder-service"} |= "error"` - только строки с ошибками

## Сбор логов

Promtail автоматически собирает логи **ВСЕХ** Docker контейнеров в сети `common`.

Вам не нужно добавлять специальные labels к контейнерам - все логи собираются автоматически.

## Структура конфигурации

```
config/
├── grafana.ini                    # Основная конфигурация Grafana
├── grafana-datasources.yml        # Подключение Loki как источника данных
├── grafana-dashboards.yml         # Автоматическая загрузка dashboards
├── promtail.yml                   # Конфигурация сбора логов
└── dashboards/
    └── logs-dashboard.json        # Готовый dashboard для логов
```

## Изменение пароля

Если хотите изменить пароль Grafana:

1. Откройте `.env` файл в корне проекта
2. Найдите строку `GRAFANA_ADMIN_PASSWORD=GrafanaLog2025!Secure`
3. Замените пароль на свой
4. Перезапустите контейнер: `docker-compose up -d grafana`

## Решение проблем

### Не видно логов конкретного сервиса

1. Проверьте что контейнер запущен: `docker ps`
2. Проверьте что контейнер в сети `common`
3. Подождите 5-10 секунд - Promtail обновляет список контейнеров каждые 5 секунд

### Grafana не открывается

1. Проверьте что контейнеры запущены: `docker ps | grep grafana`
2. Проверьте логи: `docker logs grafana`

### Забыли пароль

Удалите volume и пересоздайте контейнер:
```bash
docker-compose down
docker volume rm repaircopilotback_grafana-data
docker-compose up -d
```

## Полезные команды

```bash
# Просмотр логов самой системы логирования
docker logs grafana
docker logs loki
docker logs promtail

# Перезапуск системы логирования
docker-compose restart grafana loki promtail

# Остановка системы логирования
docker-compose stop grafana loki promtail

# Полная очистка (удалит все сохраненные логи и настройки!)
docker-compose down
docker volume rm repaircopilotback_grafana-data
```