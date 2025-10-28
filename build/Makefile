.PHONY: help build run test clean docker-build docker-up docker-down docker-logs

# Переменные
BINARY_NAME=ride-hail-system
BUILD_DIR=build
DOCKER_COMPOSE=docker compose -f deployments/docker-compose.yml

help: ## Показать справку
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Собрать бинарник
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

run-ride: build ## Запустить Ride service локально
	@echo "Running Ride service..."
	@./$(BUILD_DIR)/$(BINARY_NAME) -service=ride

run-driver: build ## Запустить Driver service локально
	@echo "Running Driver service..."
	@./$(BUILD_DIR)/$(BINARY_NAME) -service=driver

run-admin: build ## Запустить Admin service локально
	@echo "Running Admin service..."
	@./$(BUILD_DIR)/$(BINARY_NAME) -service=admin

run-all: build ## Запустить все сервисы локально
	@echo "Running all services..."
	@./$(BUILD_DIR)/$(BINARY_NAME) -service=all

test: ## Запустить тесты
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

lint: ## Запустить линтер
	@echo "Running linter..."
	@go vet ./...
	@go fmt ./...

tidy: ## Очистить зависимости
	@echo "Tidying dependencies..."
	@go mod tidy

clean: ## Очистить build артефакты
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.txt

docker-build: ## Собрать Docker образы
	@echo "Building Docker images..."
	@$(DOCKER_COMPOSE) build

docker-up: ## Запустить все сервисы в Docker
	@echo "Starting services in Docker..."
	@$(DOCKER_COMPOSE) up -d

docker-up-build: ## Пересобрать и запустить все сервисы в Docker
	@echo "Rebuilding and starting services in Docker..."
	@$(DOCKER_COMPOSE) up -d --build

docker-down: ## Остановить все сервисы в Docker
	@echo "Stopping services in Docker..."
	@$(DOCKER_COMPOSE) down

docker-down-volumes: ## Остановить все сервисы и удалить volumes
	@echo "Stopping services and removing volumes..."
	@$(DOCKER_COMPOSE) down -v

docker-logs: ## Показать логи всех сервисов
	@$(DOCKER_COMPOSE) logs -f

docker-logs-ride: ## Показать логи Ride service
	@$(DOCKER_COMPOSE) logs -f app-ride

docker-logs-driver: ## Показать логи Driver service
	@$(DOCKER_COMPOSE) logs -f app-driver

docker-logs-admin: ## Показать логи Admin service
	@$(DOCKER_COMPOSE) logs -f app-admin

docker-ps: ## Показать статус контейнеров
	@$(DOCKER_COMPOSE) ps

docker-restart: docker-down docker-up ## Перезапустить все сервисы

db-shell: ## Подключиться к PostgreSQL
	@docker exec -it ridehail-postgres psql -U ridehail_user -d ridehail_db

rabbitmq-shell: ## Открыть RabbitMQ Management UI
	@echo "Opening RabbitMQ Management at http://localhost:15672"
	@echo "Login: guest / guest"

generate-jwt: ## Сгенерировать JWT токен
	@go run cmd/generate-jwt/main.go

.DEFAULT_GOAL := help