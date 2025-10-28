.PHONY: help build run clean docker-build docker-up docker-down docker-logs docker-restart

# Отключаем BuildKit
export DOCKER_BUILDKIT=0
export COMPOSE_DOCKER_CLI_BUILD=0

help: ## Показать справку
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Собрать бинарник локально
	@echo "Building ride-hail-system..."
	@mkdir -p build
	@go build -o build/ride-hail-system .
	@echo "Build complete: build/ride-hail-system"

docker-build: ## Собрать Docker образ
	@echo "Building Docker image without buildx..."
	@DOCKER_BUILDKIT=0 docker build -f deployments/Dockerfile -t ridehail-app:latest .

docker-up: docker-build ## Запустить все сервисы в Docker
	@echo "Starting services..."
	@cd deployments && docker compose up -d
	@echo "Services started. View logs with: make docker-logs"

docker-up-attached: docker-build ## Запустить все сервисы с выводом логов
	@echo "Starting services with logs..."
	@cd deployments && docker compose up

docker-down: ## Остановить все сервисы
	@echo "Stopping services..."
	@cd deployments && docker compose down

docker-down-volumes: ## Остановить сервисы и удалить volumes
	@echo "Stopping services and removing volumes..."
	@cd deployments && docker compose down -v

docker-logs: ## Показать логи всех сервисов
	@cd deployments && docker compose logs -f

docker-logs-ride: ## Показать логи Ride service
	@cd deployments && docker compose logs -f app-ride

docker-logs-driver: ## Показать логи Driver service
	@cd deployments && docker compose logs -f app-driver

docker-logs-admin: ## Показать логи Admin service
	@cd deployments && docker compose logs -f app-admin

docker-restart: docker-down docker-up ## Перезапустить все сервисы

docker-ps: ## Показать статус контейнеров
	@cd deployments && docker compose ps

db-shell: ## Подключиться к PostgreSQL
	@docker exec -it ridehail-postgres psql -U ridehail_user -d ridehail_db

test: ## Запустить тесты
	@go test -v -race ./...

clean: ## Очистить артефакты
	@rm -rf build/
	@echo "Cleaned build artifacts"

.DEFAULT_GOAL := help