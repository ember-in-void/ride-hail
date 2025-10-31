# 📐 Стандарты кода и лучшие практики

> Руководство по написанию чистого и понятного кода в проекте Ride-Hail System

## 📑 Содержание

1. [Общие принципы](#общие-принципы)
2. [Структура комментариев](#структура-комментариев)
3. [Организация кода](#организация-кода)
4. [Обработка ошибок](#обработка-ошибок)
5. [Тестирование](#тестирование)
6. [Чек-лист перед коммитом](#чек-лист-перед-коммитом)

---

## 🎯 Общие принципы

### Clean Architecture

```
┌─────────────────────────────────────────────┐
│              DOMAIN (Ядро)                  │
│  ┌──────────────────────────────────┐       │
│  │  Entities, Value Objects         │       │
│  │  Ride, Driver, Coordinate        │       │
│  └──────────────────────────────────┘       │
│               ▲                             │
│               │ зависимость                 │
│  ┌────────────┴─────────────────────┐       │
│  │  USE CASES (Бизнес-логика)       │       │
│  │  RequestRide, AssignDriver       │       │
│  └──────────────────────────────────┘       │
│               ▲                             │
│               │ через интерфейсы            │
│  ┌────────────┴─────────────────────┐       │
│  │  ADAPTERS (Реализации)           │       │
│  │  PostgreSQL, RabbitMQ, HTTP      │       │
│  └──────────────────────────────────┘       │
└─────────────────────────────────────────────┘
```

**Правило зависимостей**: Стрелки всегда указывают **внутрь**.
- ✅ Use Case может использовать Domain
- ✅ Adapter может использовать Use Case (через интерфейс)
- ❌ Domain НЕ МОЖЕТ зависеть от Use Case
- ❌ Use Case НЕ МОЖЕТ зависеть от конкретной реализации Adapter

### SOLID принципы

#### 1. **S**ingle Responsibility Principle (Единственная ответственность)

❌ **Плохо** (одна функция делает всё):
```go
func CreateRide(w http.ResponseWriter, r *http.Request) {
    // Парсинг JSON
    var req RequestRideRequest
    json.NewDecoder(r.Body).Decode(&req)
    
    // Валидация
    if req.PickupLat == 0 { return }
    
    // Сохранение в БД
    db.Exec("INSERT INTO rides ...")
    
    // Отправка в RabbitMQ
    mq.Publish("ride_requested", ...)
    
    // Отправка WebSocket
    ws.Send(userID, "ride_created")
}
```

✅ **Хорошо** (каждый компонент делает одно дело):
```go
// HTTP Handler — только парсинг HTTP
func (h *HTTPHandler) CreateRide(w http.ResponseWriter, r *http.Request) {
    var req RequestRideRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.respondError(w, err)
        return
    }
    
    // Вызов Use Case
    output, err := h.requestRideUseCase.Execute(r.Context(), input)
    h.respondJSON(w, output)
}

// Use Case — только бизнес-логика
func (s *RequestRideService) Execute(ctx context.Context, input Input) (Output, error) {
    // Валидация
    if err := s.validate(input); err != nil {
        return Output{}, err
    }
    
    // Создание Ride
    ride := domain.NewRide(...)
    
    // Сохранение
    if err := s.rideRepo.Save(ctx, ride); err != nil {
        return Output{}, err
    }
    
    // Публикация события
    s.eventPublisher.Publish(ctx, "ride_requested", ...)
    
    return Output{RideID: ride.ID}, nil
}
```

#### 2. **D**ependency Inversion Principle (Инверсия зависимостей)

❌ **Плохо** (Use Case зависит от конкретной реализации):
```go
type RequestRideService struct {
    rideRepo *RidePgRepository  // ← зависимость от PostgreSQL!
}
```

✅ **Хорошо** (Use Case зависит от интерфейса):
```go
// В слое Application/Ports
type RideRepository interface {
    Save(ctx context.Context, ride *domain.Ride) error
    FindByID(ctx context.Context, id string) (*domain.Ride, error)
}

type RequestRideService struct {
    rideRepo RideRepository  // ← интерфейс, не конкретная реализация!
}

// В слое Adapters
type RidePgRepository struct {
    db *pgxpool.Pool
}

func (r *RidePgRepository) Save(ctx context.Context, ride *domain.Ride) error {
    // PostgreSQL implementation
}
```

**Преимущество**: Можно легко заменить PostgreSQL на In-Memory для тестов!

---

## 📝 Структура комментариев

### Заголовок файла

```go
// ============================================================================
// НАЗВАНИЕ КОМПОНЕНТА - Краткое описание
// ============================================================================
//
// 📦 НАЗНАЧЕНИЕ:
// Подробное объяснение, зачем нужен этот файл/пакет.
// Какую проблему он решает?
//
// 🎯 ОСНОВНЫЕ ЗАДАЧИ:
// 1. Первая задача
// 2. Вторая задача
// 3. Третья задача
//
// 💡 ПРИМЕР ИСПОЛЬЗОВАНИЯ:
//
//   // Код примера
//   handler := NewHTTPHandler(useCase, log)
//   handler.RegisterRoutes(mux)
//
// 🏗️ АРХИТЕКТУРА:
//
//   Визуальная диаграмма потока данных
//
// ============================================================================
```

### Публичные функции

```go
// ============================================================================
// НАЗВАНИЕ ФУНКЦИИ
// ============================================================================

// КраткоеОписание делает что-то полезное.
//
// ПАРАМЕТРЫ:
// - ctx: контекст для отмены операции
// - input: данные запроса
//
// ВОЗВРАЩАЕТ:
// - Output: результат выполнения
// - error: ошибка, если что-то пошло не так
//
// ОШИБКИ:
// - ErrInvalidInput: если input не валиден
// - ErrNotFound: если ресурс не найден
//
// ПРИМЕР:
//   output, err := service.Execute(ctx, input)
//   if err != nil {
//     log.Error(err)
//     return
//   }
func (s *Service) Execute(ctx context.Context, input Input) (Output, error) {
    // Реализация
}
```

### Константы и переменные

```go
const (
    // MaxRetries — максимальное количество попыток переподключения
    // к RabbitMQ перед возвратом ошибки.
    MaxRetries = 3
    
    // ConnectionTimeout — таймаут подключения к базе данных.
    // Если соединение не установлено за 10 секунд, возвращается ошибка.
    ConnectionTimeout = 10 * time.Second
)
```

### Сложная бизнес-логика

```go
func (s *Service) AssignDriver(ctx context.Context, rideID, driverID string) error {
    // ШАГ 1: Загрузка поездки из БД
    // Нужно убедиться, что поездка существует и в статусе REQUESTED
    ride, err := s.rideRepo.FindByID(ctx, rideID)
    if err != nil {
        return fmt.Errorf("find ride: %w", err)
    }
    
    // ШАГ 2: Проверка бизнес-правила
    // Нельзя назначить водителя, если поездка уже назначена
    if ride.Status != domain.StatusRequested {
        return domain.ErrRideAlreadyAssigned
    }
    
    // ШАГ 3: Атомарное обновление в БД
    // WHERE status='REQUESTED' защищает от race condition:
    // если два водителя одновременно примут поездку, только один успеет
    if err := s.rideRepo.AssignDriver(ctx, rideID, driverID); err != nil {
        return fmt.Errorf("assign driver: %w", err)
    }
    
    return nil
}
```

---

## 🗂️ Организация кода

### Структура пакетов

```
internal/
├── ride/                         # Bounded Context: Поездки
│   ├── domain/                   # Entities, Value Objects, Domain Errors
│   │   ├── ride.go               # Entity: Ride
│   │   ├── coordinate.go         # Value Object: Coordinate
│   │   └── errors.go             # Domain Errors
│   ├── application/              # Бизнес-логика
│   │   ├── ports/
│   │   │   ├── in/               # Интерфейсы Use Cases
│   │   │   │   ├── request_ride.go
│   │   │   │   └── handle_driver_response.go
│   │   │   └── out/              # Интерфейсы Repository, Publisher
│   │   │       ├── ride_repository.go
│   │   │       └── event_publisher.go
│   │   └── usecase/              # Реализации Use Cases
│   │       ├── request_ride_usecase.go
│   │       └── handle_driver_response.go
│   ├── adapter/                  # Реализации адаптеров
│   │   ├── in/                   # Входящие адаптеры (HTTP, AMQP, WS)
│   │   │   ├── transport/        # HTTP handlers
│   │   │   ├── in_amqp/          # RabbitMQ consumers
│   │   │   └── in_ws/            # WebSocket handlers
│   │   └── out/                  # Исходящие адаптеры (DB, MQ)
│   │       ├── repo/             # PostgreSQL repositories
│   │       ├── out_amqp/         # RabbitMQ publishers
│   │       └── out_ws/           # WebSocket notifiers
│   └── bootstrap/                # Dependency Injection
│       └── compose.go            # Сборка всех компонентов
├── driver/                       # Bounded Context: Водители
│   └── ...                       # Аналогичная структура
└── shared/                       # Общие компоненты
    ├── auth/                     # JWT authentication
    ├── db/                       # Database connection pool
    ├── mq/                       # RabbitMQ connection
    └── ws/                       # WebSocket hub
```

### Именование файлов

| Тип | Имя файла | Пример |
|-----|-----------|--------|
| Entity | `entity_name.go` | `ride.go`, `driver.go` |
| Use Case | `action_usecase.go` | `request_ride_usecase.go` |
| Repository | `entity_repository.go` | `ride_pg_repository.go` |
| HTTP Handler | `http_handler.go` | `http_handler.go` |
| AMQP Consumer | `event_consumer.go` | `driver_response_consumer.go` |
| Domain Errors | `errors.go` | `errors.go` |

### Порядок объявлений в файле

```go
// 1. Package и imports
package usecase

import (
    "context"
    "fmt"
)

// 2. Константы
const (
    MaxRetries = 3
)

// 3. Типы (интерфейсы, структуры)
type Input struct {
    RideID string
}

// 4. Конструкторы
func NewService(repo Repository) *Service {
    return &Service{repo: repo}
}

// 5. Публичные методы
func (s *Service) Execute(ctx context.Context, input Input) error {
    return s.execute(ctx, input)
}

// 6. Приватные методы
func (s *Service) execute(ctx context.Context, input Input) error {
    // ...
}
```

---

## ⚠️ Обработка ошибок

### Создание доменных ошибок

```go
// domain/errors.go
package domain

import "errors"

var (
    // ErrRideNotFound возвращается когда поездка не найдена в БД
    ErrRideNotFound = errors.New("ride not found")
    
    // ErrRideAlreadyAssigned возвращается при попытке назначить водителя
    // на поездку, которая уже имеет водителя
    ErrRideAlreadyAssigned = errors.New("ride already assigned to another driver")
    
    // ErrInvalidStatus возвращается при попытке изменить статус поездки
    // на недопустимый (например, из COMPLETED в REQUESTED)
    ErrInvalidStatus = errors.New("invalid ride status transition")
)
```

### Обработка ошибок в Use Cases

```go
func (s *Service) Execute(ctx context.Context, input Input) (Output, error) {
    ride, err := s.rideRepo.FindByID(ctx, input.RideID)
    if err != nil {
        // Проверяем конкретную ошибку
        if errors.Is(err, domain.ErrRideNotFound) {
            s.log.Warn(logger.Entry{
                Action:  "ride_not_found",
                Message: fmt.Sprintf("ride %s not found", input.RideID),
            })
            return Output{}, domain.ErrRideNotFound
        }
        
        // Неожиданная ошибка (например, проблема с БД)
        s.log.Error(logger.Entry{
            Action:  "find_ride_failed",
            Message: err.Error(),
            Error:   &logger.ErrObj{Msg: err.Error()},
        })
        return Output{}, fmt.Errorf("find ride: %w", err)
    }
    
    // ...
}
```

### Обработка ошибок в HTTP handlers

```go
func (h *HTTPHandler) CreateRide(w http.ResponseWriter, r *http.Request) {
    output, err := h.useCase.Execute(r.Context(), input)
    if err != nil {
        // Доменная ошибка → клиентская (4xx)
        if errors.Is(err, domain.ErrInvalidInput) {
            h.respondError(w, http.StatusBadRequest, err.Error())
            return
        }
        
        // Неожиданная ошибка → серверная (5xx)
        h.log.Error(logger.Entry{
            Action:  "create_ride_failed",
            Message: err.Error(),
            Error:   &logger.ErrObj{Msg: err.Error()},
        })
        h.respondError(w, http.StatusInternalServerError, "internal server error")
        return
    }
    
    h.respondJSON(w, http.StatusCreated, output)
}
```

### Логирование ошибок

```go
// ✅ Хорошо: структурированный лог
s.log.Error(logger.Entry{
    Action:  "assign_driver_failed",
    Message: fmt.Sprintf("failed to assign driver %s to ride %s", driverID, rideID),
    Error:   &logger.ErrObj{Msg: err.Error()},
    Extra: map[string]interface{}{
        "ride_id":   rideID,
        "driver_id": driverID,
    },
})

// ❌ Плохо: простая строка без контекста
log.Println("Error:", err)
```

---

## 🧪 Тестирование

### Unit тесты для Use Cases

```go
// usecase/request_ride_test.go
package usecase_test

import (
    "context"
    "testing"
    
    "ridehail/internal/ride/domain"
    "ridehail/internal/ride/application/usecase"
)

// Мок репозитория
type mockRideRepository struct {
    saveFunc func(ctx context.Context, ride *domain.Ride) error
}

func (m *mockRideRepository) Save(ctx context.Context, ride *domain.Ride) error {
    return m.saveFunc(ctx, ride)
}

func TestRequestRideService_Execute_Success(t *testing.T) {
    // Arrange
    ctx := context.Background()
    mockRepo := &mockRideRepository{
        saveFunc: func(ctx context.Context, ride *domain.Ride) error {
            return nil // успешное сохранение
        },
    }
    
    service := usecase.NewRequestRideService(mockRepo, nil, nil, nil, logger)
    
    input := usecase.RequestRideInput{
        PassengerID: "user-123",
        PickupLat:   55.7558,
        PickupLon:   37.6173,
    }
    
    // Act
    output, err := service.Execute(ctx, input)
    
    // Assert
    if err != nil {
        t.Fatalf("expected no error, got: %v", err)
    }
    
    if output.RideID == "" {
        t.Error("expected ride ID, got empty string")
    }
}

func TestRequestRideService_Execute_InvalidInput(t *testing.T) {
    // Arrange
    ctx := context.Background()
    service := usecase.NewRequestRideService(nil, nil, nil, nil, logger)
    
    input := usecase.RequestRideInput{
        PassengerID: "", // невалидный input
    }
    
    // Act
    _, err := service.Execute(ctx, input)
    
    // Assert
    if err == nil {
        t.Fatal("expected error for invalid input")
    }
    
    if !errors.Is(err, domain.ErrInvalidInput) {
        t.Errorf("expected ErrInvalidInput, got: %v", err)
    }
}
```

### Integration тесты с БД

```go
// adapter/repo/ride_pg_repository_test.go
package repo_test

import (
    "context"
    "testing"
    
    "ridehail/internal/shared/db"
    "ridehail/internal/ride/adapter/out/repo"
)

func TestRidePgRepository_Save(t *testing.T) {
    // Подключаемся к тестовой БД
    ctx := context.Background()
    pool, err := db.NewPool(ctx, testDBConfig, logger)
    if err != nil {
        t.Fatalf("failed to connect to test DB: %v", err)
    }
    defer pool.Close()
    
    // Создаем репозиторий
    rideRepo := repo.NewRidePgRepository(pool, logger)
    
    // Создаем тестовую поездку
    ride := &domain.Ride{
        ID:          "test-ride-123",
        PassengerID: "user-456",
        Status:      domain.StatusRequested,
    }
    
    // Сохраняем
    err = rideRepo.Save(ctx, ride)
    if err != nil {
        t.Fatalf("failed to save ride: %v", err)
    }
    
    // Проверяем, что поездка сохранилась
    saved, err := rideRepo.FindByID(ctx, ride.ID)
    if err != nil {
        t.Fatalf("failed to find saved ride: %v", err)
    }
    
    if saved.ID != ride.ID {
        t.Errorf("expected ride ID %s, got %s", ride.ID, saved.ID)
    }
}
```

---

## ✅ Чек-лист перед коммитом

### Код

- [ ] Код следует Clean Architecture (зависимости направлены внутрь)
- [ ] Все публичные функции имеют комментарии
- [ ] Нет захардкоженных значений (используются константы/конфиг)
- [ ] Обработаны все ошибки (нет игнорирования `err`)
- [ ] Логируются важные события (с правильным уровнем: Info/Warn/Error)
- [ ] Нет дублирования кода (DRY principle)

### Безопасность

- [ ] Все SQL запросы используют параметризацию (`$1, $2`) против SQL injection
- [ ] JWT токены проверяются перед доступом к ресурсам
- [ ] Пароли НЕ логируются
- [ ] Нет чувствительных данных в логах (токены, карты, etc.)

### Производительность

- [ ] Нет N+1 запросов к БД (используется JOIN где нужно)
- [ ] Используются индексы для частых запросов
- [ ] Connection pool настроен правильно
- [ ] WebSocket соединения закрываются при отключении

### Тесты

- [ ] Написаны unit тесты для бизнес-логики
- [ ] Покрытие тестами > 70%
- [ ] Все тесты проходят: `go test ./...`
- [ ] E2E сценарий работает: `./scripts/test-full-flow.sh`

### Документация

- [ ] README.md обновлен (если добавлены новые фичи)
- [ ] API документация обновлена (если изменились endpoints)
- [ ] Комментарии в коде актуальны

### Форматирование

```bash
# Форматирование кода
go fmt ./...

# Линтер
golangci-lint run

# Проверка импортов
goimports -w .
```

---

## 📚 Дополнительные ресурсы

- [Clean Architecture (Uncle Bob)](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [Effective Go](https://golang.org/doc/effective_go)
- [SOLID в Go](https://dave.cheney.net/2016/08/20/solid-go-design)
- [Error handling in Go](https://go.dev/blog/error-handling-and-go)

---

**Помни**: Хороший код — это код, который легко читать и понимать через 6 месяцев! 🚀
