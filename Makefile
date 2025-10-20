.PHONY: up rebuild status stop down restart logs logs-user logs-api logs-tz help

.DEFAULT_GOAL := help

# Запуск всех сервисов (без пересборки, если уже запущены)
up:
	@echo "Starting all services..."
	@docker compose -f docker-compose.yml --project-name common up -d
	@docker compose -f user-service/deployment/docker-compose.yml --env-file .env --project-name user-service up -d
	@docker compose -f llm-requester/docker-compose.yml --env-file .env  --project-name llm-requester up -d
	@docker compose -f promt-builder/docker-compose.yml --env-file .env --project-name prompt-builder up -d
	@docker compose -f tz-bot/deployment/docker-compose.yml --env-file .env --project-name tz-service up -d
	@docker compose -f api-gateway-service/deployment/docker-compose.yml --env-file .env --project-name api-gateway up -d
	@docker compose -f doc-to-docx-converter/docker-compose.yml --project-name doc-to-docx-converter up -d
	@docker compose -f docx-converter/docker-compose.yml --project-name docx-parser up -d
	@docker compose -f report-generator/docker-compose.yml --project-name report-generator up -d
	@docker compose -f md-converter/deployment/docker-compose.yml --project-name html-to-markdown-converter up -d
	@echo "All services started!"

# Форсированная пересборка всех сервисов
rebuild:
	@echo "Rebuilding all services..."
	@docker compose -f docker-compose.yml --project-name common up -d
	@docker compose -f user-service/deployment/docker-compose.yml --env-file .env --project-name user-service up -d --build
	@docker compose -f llm-requester/docker-compose.yml --env-file .env  --project-name llm-requester up -d --build
	@docker compose -f promt-builder/docker-compose.yml --env-file .env --project-name prompt-builder up -d --build
	@docker compose -f tz-bot/deployment/docker-compose.yml --env-file .env --project-name tz-service up -d --build
	@docker compose -f api-gateway-service/deployment/docker-compose.yml --env-file .env --project-name api-gateway up -d --build
	@docker compose -f doc-to-docx-converter/docker-compose.yml --project-name doc-to-docx-converter up -d --build
	@docker compose -f docx-converter/docker-compose.yml --project-name docx-parser up -d --build
	@docker compose -f report-generator/docker-compose.yml --project-name report-generator up -d --build
	@docker compose -f md-converter/deployment/docker-compose.yml --project-name html-to-markdown-converter up -d --build
	@echo "All services rebuilt!"

# Статус всех сервисов
status:
	@echo "=== Common ==="
	@docker compose -f docker-compose.yml --project-name common ps
	@echo "\n=== User Service ==="
	@docker compose -f user-service/deployment/docker-compose.yml --project-name user-service ps
	@echo "\n=== LLM Requester ==="
	@docker compose -f llm-requester/docker-compose.yml --project-name llm-requester ps
	@echo "\n=== Prompt Builder ==="
	@docker compose -f promt-builder/docker-compose.yml --project-name prompt-builder ps
	@echo "\n=== TZ Service ==="
	@docker compose -f tz-bot/deployment/docker-compose.yml --project-name tz-service ps
	@echo "\n=== API Gateway ==="
	@docker compose -f api-gateway-service/deployment/docker-compose.yml --project-name api-gateway ps
	@echo "\n=== Doc to DOCX Converter ==="
	@docker compose -f doc-to-docx-converter/docker-compose.yml --project-name doc-to-docx-converter ps
	@echo "\n=== DOCX Parser ==="
	@docker compose -f docx-converter/docker-compose.yml --project-name docx-parser ps
	@echo "\n=== Report Generator ==="
	@docker compose -f report-generator/docker-compose.yml --project-name report-generator ps
	@echo "\n=== HTML to Markdown Converter ==="
	@docker compose -f md-converter/deployment/docker-compose.yml --project-name html-to-markdown-converter ps

# Остановка всех сервисов
stop:
	@echo "Stopping all services..."
	@docker compose -f docker-compose.yml --project-name common stop
	@docker compose -f user-service/deployment/docker-compose.yml --project-name user-service stop
	@docker compose -f llm-requester/docker-compose.yml --project-name llm-requester stop
	@docker compose -f promt-builder/docker-compose.yml --project-name prompt-builder stop
	@docker compose -f tz-bot/deployment/docker-compose.yml --project-name tz-service stop
	@docker compose -f api-gateway-service/deployment/docker-compose.yml --project-name api-gateway stop
	@docker compose -f doc-to-docx-converter/docker-compose.yml --project-name doc-to-docx-converter stop
	@docker compose -f docx-converter/docker-compose.yml --project-name docx-parser stop
	@docker compose -f report-generator/docker-compose.yml --project-name report-generator stop
	@docker compose -f md-converter/deployment/docker-compose.yml --project-name html-to-markdown-converter stop
	@echo "All services stopped!"

# Удаление всех контейнеров
down:
	@echo "Removing all containers..."
	@docker compose -f docker-compose.yml --project-name common down
	@docker compose -f user-service/deployment/docker-compose.yml --project-name user-service down
	@docker compose -f llm-requester/docker-compose.yml --project-name llm-requester down
	@docker compose -f promt-builder/docker-compose.yml --project-name prompt-builder down
	@docker compose -f tz-bot/deployment/docker-compose.yml --project-name tz-service down
	@docker compose -f api-gateway-service/deployment/docker-compose.yml --project-name api-gateway down
	@docker compose -f doc-to-docx-converter/docker-compose.yml --project-name doc-to-docx-converter down
	@docker compose -f docx-converter/docker-compose.yml --project-name docx-parser down
	@docker compose -f report-generator/docker-compose.yml --project-name report-generator down
	@docker compose -f md-converter/deployment/docker-compose.yml --project-name html-to-markdown-converter down
	@echo "All containers removed!"

# Перезапуск всех сервисов
restart: stop up

# Логи всех сервисов
logs:
	@docker compose -f docker-compose.yml --project-name common logs -f

# Логи конкретного сервиса (использование: make logs-service SERVICE=user-service)
logs-user:
	@docker compose -f user-service/deployment/docker-compose.yml --project-name user-service logs -f

logs-api:
	@docker compose -f api-gateway-service/deployment/docker-compose.yml --project-name api-gateway logs -f

logs-tz:
	@docker compose -f tz-bot/deployment/docker-compose.yml --project-name tz-service logs -f

# Помощь
help:
	@echo "Available commands:"
	@echo "  make up          - Start all services (without rebuild if already running)"
	@echo "  make rebuild     - Force rebuild and start all services"
	@echo "  make stop        - Stop all services"
	@echo "  make down        - Remove all containers"
	@echo "  make restart     - Restart all services"
	@echo "  make status      - Show status of all services"
	@echo "  make logs        - Show logs from common services"
	@echo "  make logs-user   - Show logs from user-service"
	@echo "  make logs-api    - Show logs from api-gateway"
	@echo "  make logs-tz     - Show logs from tz-bot"