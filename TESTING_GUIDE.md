# 🚗 Driver Service - Quick Reference

## Скрипты для тестирования

Создано 5 скриптов в директории `scripts/`:

| Скрипт | Описание |
|--------|----------|
| `setup-test-driver.sh` | Создает тестового водителя через Admin API |
| `generate-driver-token.sh` | Генерирует JWT токен для водителя |
| `test-driver-api.sh` | **Полное автоматическое тестирование всех 8 эндпоинтов** ⭐ |
| `test-driver-workflow.sh` | Тестирует полный сценарий работы водителя |
| `driver-api-helpers.sh` | Интерактивные функции для ручного тестирования |

## Быстрый старт тестирования

```bash
# 1. Запустить сервисы
cd deployments && docker-compose up -d

# 2. Создать тестового водителя
./scripts/setup-test-driver.sh
# Сохраните DRIVER_ID из вывода!

# 3. Запустить полное тестирование
export DRIVER_ID="your-driver-id-from-step-2"
./scripts/test-driver-api.sh
```

## Успешный результат

```
✅ PASSED: Health Check
✅ PASSED: Go Online
✅ PASSED: Update Location
✅ PASSED: Rate limit works correctly
✅ PASSED: Go Offline
✅ PASSED: Invalid token rejected
✅ PASSED: ID mismatch detected
✅ PASSED: Invalid coordinates rejected
```

## Подробная документация

См. [docs/DRIVER_TESTING.md](docs/DRIVER_TESTING.md)
