.PHONY: help build up down restart logs ps clean dev-up dev-down test

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the Docker image
	docker-compose build

up: ## Start the service in detached mode
	docker-compose up -d

down: ## Stop and remove containers
	docker-compose down

restart: ## Restart the service
	docker-compose restart

logs: ## Show service logs (follow mode)
	docker-compose logs -f

ps: ## Show running containers
	docker-compose ps

clean: ## Stop containers and remove volumes
	docker-compose down -v

dev-up: ## Start service with live reload (local development)
	uvicorn LLMRequester.app.main:app --reload --port 8020

dev-down: ## Stop development server (for consistency)
	@echo "Press Ctrl+C to stop the development server"

test: ## Run tests (placeholder - add your test command here)
	@echo "No tests configured yet"

shell: ## Open a shell in the running container
	docker-compose exec llm-requester /bin/sh

rebuild: down build up ## Rebuild and restart the service
