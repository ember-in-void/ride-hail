# 🏗️ Архитектура системы: Обработка назначения водителя

> **Для начинающих разработчиков**: Этот документ объясняет, как работает критический функционал системы - назначение водителя на поездку.

## 📚 Оглавление
1. [Общая картина](#общая-картина)
2. [Clean Architecture](#clean-architecture)
3. [Детальный поток данных](#детальный-поток-данных)
4. [Защита от ошибок](#защита-от-ошибок)
5. [Словарь терминов](#словарь-терминов)

---

## 🎯 Общая картина

### Что происходит когда пассажир заказывает поездку?

```
1. 👤 ПАССАЖИР нажимает "Заказать" в приложении
   ↓
2. 📱 Ride Service создает поездку (status=REQUESTED)
   ↓
3. 🐰 RabbitMQ получает сообщение: ride.request.ECONOMY
   ↓
4. 🚗 Driver Service ищет водителей поблизости через PostGIS
   ↓
5. 📲 WebSocket отправляет оффер водителю: "Новая поездка!"
   ↓
6. 👨‍✈️ ВОДИТЕЛЬ нажимает "Принять"
   ↓
7. 🐰 RabbitMQ получает: driver.response.{ride_id}
   ↓
8. 📱 Ride Service НАЗНАЧАЕТ водителя (← ЭТО НАША ТЕМА)
   ↓
9. 💬 WebSocket уведомляет пассажира: "Водитель найден!"
```

---

## 🏛️ Clean Architecture

### Что такое Clean Architecture простыми словами?

Представь, что твой код — это дом с несколькими комнатами:
- **Самая важная комната (Use Case)** — здесь живут правила бизнеса
- **Коридоры (Порты/Интерфейсы)** — через них комнаты общаются
- **Внешние двери (Адаптеры)** — связь с внешним миром (БД, RabbitMQ)

### Почему это важно?

❌ **Плохой код (без Clean Architecture)**:
```go
// Бизнес-логика вперемешку с БД и RabbitMQ!
func handleDriverResponse(msg amqp.Delivery) {
    var data DriverResponseMessage
    json.Unmarshal(msg.Body, &data)
    
    // SQL прямо в обработчике RabbitMQ! 😱
    db.Exec("UPDATE rides SET driver_id=$1 WHERE id=$2", 
        data.DriverID, data.RideID)
    
    // Невозможно протестировать без запуска RabbitMQ и PostgreSQL
}
```

✅ **Хороший код (Clean Architecture)**:
```go
// Адаптер: только парсит сообщения
func (c *Consumer) handleMessage(msg amqp.Delivery) {
    var data DriverResponseMessage
    json.Unmarshal(msg.Body, &data)
    
    // Вызываем бизнес-логику через интерфейс
    output, err := c.useCase.Execute(ctx, input)
}

// Use Case: чистая бизнес-логика
func (s *Service) Execute(input Input) (Output, error) {
    // 1. Валидация
    // 2. Вызов БД через интерфейс
    // 3. Возврат результата
}

// Легко тестировать с mock-объектами!
```

### Слои архитектуры (от центра к краям)

```
┌─────────────────────────────────────────────┐
│  DOMAIN (Бизнес-сущности)                   │
│  ├── Ride, Driver, Passenger                │
│  └── Бизнес-правила (неизменны)             │
└─────────────────────────────────────────────┘
           ↑
┌─────────────────────────────────────────────┐
│  APPLICATION (Use Cases)                     │
│  ├── HandleDriverResponseService            │
│  ├── RequestRideService                     │
│  └── Бизнес-процессы (оркестрация)          │
└─────────────────────────────────────────────┘
           ↑
┌─────────────────────────────────────────────┐
│  PORTS (Интерфейсы)                          │
│  ├── In: HandleDriverResponseUseCase        │
│  └── Out: RideRepository                    │
└─────────────────────────────────────────────┘
           ↑
┌─────────────────────────────────────────────┐
│  ADAPTERS (Внешний мир)                      │
│  ├── In: RabbitMQ Consumer, HTTP Handler    │
│  └── Out: PostgreSQL Repository             │
└─────────────────────────────────────────────┘
```

**Правило зависимостей**: Стрелки направлены ВНУТРЬ!
- Бизнес-логика НЕ знает о БД или RabbitMQ
- Адаптеры зависят от интерфейсов, а не наоборот

---

## 🔄 Детальный поток данных

### Шаг 1: Водитель принимает поездку

```javascript
// Мобильное приложение водителя отправляет WebSocket сообщение
{
  "type": "ride_response",
  "data": {
    "ride_id": "uuid-123",
    "accepted": true,
    "current_location": {"lat": 43.238, "lng": 76.889}
  }
}
```

↓ **WebSocket соединение** ↓

```
Driver Service (порт 3001)
```

### Шаг 2: Driver Service публикует в RabbitMQ

```go
// internal/driver/adapters/in/in_ws/driver_ws.go

// Получили WebSocket сообщение
func (h *DriverWSHandler) handleMessage(client *Client, msgType string, data json.RawMessage) {
    if msgType == "ride_response" {
        var resp RideResponseMessage
        json.Unmarshal(data, &resp)
        
        // Публикуем в RabbitMQ
        dto := &out.DriverResponseDTO{
            RideID:   resp.RideID,
            DriverID: client.UserID, // Из JWT токена
            Accepted: resp.Accepted,
        }
        
        h.msgPublisher.PublishDriverResponse(ctx, dto)
        // → Exchange: driver_topic
        // → Routing Key: driver.response.{ride_id}
    }
}
```

### Шаг 3: RabbitMQ маршрутизирует сообщение

```
Exchange: driver_topic (type=topic)
    ↓
Routing Key: driver.response.uuid-123
    ↓
Queue Binding: driver.response.* → ride_service_driver_responses
    ↓
Queue: ride_service_driver_responses (durable)
    ↓
Consumer: Ride Service
```

### Шаг 4: Ride Service получает сообщение

```go
// internal/ride/adapter/in/in_amqp/driver_response_consumer.go

// Бесконечный цикл обработки
for {
    select {
    case msg := <-msgs:
        // 1. Парсим JSON
        var response DriverResponseMessage
        json.Unmarshal(msg.Body, &response)
        
        // 2. Преобразуем в Input для use case
        input := in.HandleDriverResponseInput{
            RideID:   response.RideID,
            DriverID: response.DriverID,
            Accepted: response.Accepted,
        }
        
        // 3. Вызываем бизнес-логику
        output, err := c.handleDriverResponseUseCase.Execute(ctx, input)
        
        // 4. Подтверждаем обработку
        msg.Ack(false)
    }
}
```

### Шаг 5: Use Case выполняет бизнес-логику

```go
// internal/ride/application/usecase/handle_driver_response.go

func (s *HandleDriverResponseService) Execute(input Input) (Output, error) {
    // 1. Проверяем, что водитель принял
    if !input.Accepted {
        return &Output{Status: "REQUESTED"}, nil
    }
    
    // 2. Читаем поездку из БД
    ride, err := s.rideRepo.FindByID(ctx, input.RideID)
    
    // 3. Валидируем статус
    if ride.Status != "REQUESTED" {
        return nil, errors.New("ride already assigned")
    }
    
    // 4. Атомарно назначаем водителя
    err = s.rideRepo.AssignDriver(ctx, input.RideID, input.DriverID)
    
    // 5. Возвращаем PassengerID для уведомления
    return &Output{
        RideID:      input.RideID,
        Status:      "MATCHED",
        PassengerID: ride.PassengerID,
    }, nil
}
```

### Шаг 6: Repository обновляет БД

```go
// internal/ride/adapter/out/repo/ride_pg_repository.go

func (r *RidePgRepository) AssignDriver(ctx context.Context, rideID, driverID string) error {
    query := `
        UPDATE rides 
        SET 
            driver_id = $1,
            status = 'MATCHED',
            matched_at = NOW()
        WHERE id = $2
          AND status = 'REQUESTED'  -- КРИТИЧНО: защита от race condition
    `
    
    result, err := r.pool.Exec(ctx, query, driverID, rideID)
    
    if result.RowsAffected() == 0 {
        return errors.New("ride already assigned")
    }
    
    return nil
}
```

### Шаг 7: Уведомление пассажира

```go
// Возвращаемся в consumer

// Use case вернул PassengerID
output, err := c.useCase.Execute(ctx, input)

// Отправляем WebSocket уведомление пассажиру
notification := map[string]any{
    "type": "ride_matched",
    "data": map[string]any{
        "ride_id":   output.RideID,
        "driver_id": input.DriverID,
        "eta":       input.EstimatedArrivalMinutes,
    },
}

c.passengerWS.SendToUser(output.PassengerID, notification)
// → Пассажир видит: "Водитель найден! Прибудет через 5 минут"
```

---

## 🛡️ Защита от ошибок

### 1. Race Condition (два водителя принимают одну поездку)

**Проблема**: Driver_A и Driver_B одновременно нажимают "Принять"

**Решение**: SQL с WHERE status='REQUESTED'
```sql
UPDATE rides 
SET driver_id = $1, status = 'MATCHED'
WHERE id = $2 AND status = 'REQUESTED'
```

**Как работает**:
1. Driver_A: UPDATE успешен (RowsAffected=1) ✅
2. Driver_B: UPDATE не сработал (RowsAffected=0) ❌
3. Driver_B получает ошибку "ride already assigned"

### 2. Потеря сообщений RabbitMQ

**Проблема**: Сервер упал во время обработки сообщения

**Решение 1: Manual Ack**
```go
msgs, err := ch.Consume(
    queue.Name,
    "",    // consumer tag
    false, // auto-ack = FALSE!
    ...
)

for msg := range msgs {
    if err := handleMessage(msg); err != nil {
        msg.Nack(false, true) // Вернуть в очередь
    } else {
        msg.Ack(false) // Подтвердить обработку
    }
}
```

**Решение 2: Durable очереди и сообщения**
```go
// Очередь переживет рестарт RabbitMQ
queue, err := ch.QueueDeclare(
    queueName,
    true,  // durable = true
    false, // auto-delete = false
    ...
)

// Сообщения сохраняются на диск
err = ch.Publish(
    exchange,
    routingKey,
    false, // mandatory
    false, // immediate
    amqp.Publishing{
        DeliveryMode: amqp.Persistent, // ← ВАЖНО
        Body:         jsonBytes,
    },
)
```

### 3. Database Connection Pool Exhaustion

**Проблема**: Слишком много одновременных запросов к БД

**Решение**: Правильная настройка pgxpool
```go
config, _ := pgxpool.ParseConfig(connString)

// Максимум соединений
config.MaxConns = 25

// Минимум для быстрого старта
config.MinConns = 5

// Время жизни соединения
config.MaxConnLifetime = time.Hour
config.MaxConnIdleTime = 30 * time.Minute

pool, _ := pgxpool.NewWithConfig(ctx, config)
```

### 4. Context Timeout

**Проблема**: Запрос к БД висит вечно

**Решение**: Всегда передавать ctx с таймаутом
```go
// В HTTP handler
ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
defer cancel()

output, err := useCase.Execute(ctx, input)

// БД запрос автоматически отменится через 5 секунд
```

---

## 📖 Словарь терминов

### Архитектурные паттерны

**Clean Architecture**
- Разделение кода на слои с четкими границами
- Бизнес-логика не зависит от внешних технологий
- Легко тестировать, легко менять реализацию

**Hexagonal Architecture (Ports & Adapters)**
- Синоним Clean Architecture
- "Порт" = интерфейс
- "Адаптер" = реализация интерфейса

**Dependency Injection (DI)**
- Передача зависимостей через конструктор
- Вместо: `service := NewService()` внутри
- Пишем: `service := NewService(repo, logger)` снаружи

**Use Case (Прецедент использования)**
- Одна конкретная задача системы
- Пример: "Назначить водителя на поездку"
- Содержит бизнес-логику без технических деталей

### Технические термины

**DTO (Data Transfer Object)**
- Простая структура для передачи данных
- Без методов, только поля
- Пример: `HandleDriverResponseInput`

**Repository Pattern**
- Абстракция для работы с БД
- Скрывает SQL от бизнес-логики
- Интерфейс: `RideRepository`
- Реализация: `RidePgRepository` (PostgreSQL)

**Message Broker (RabbitMQ)**
- Посредник для async коммуникации между сервисами
- Publisher → Exchange → Queue → Consumer
- Гарантирует доставку через Ack/Nack

**WebSocket**
- Двусторонняя связь клиент ↔ сервер
- Для real-time уведомлений
- Пример: "Водитель найден!"

**PostgreSQL + PostGIS**
- PostgreSQL: реляционная БД
- PostGIS: расширение для геоданных
- `ST_DWithin(location, point, 5000)` = поиск в радиусе 5км

**Connection Pool**
- Пул переиспользуемых соединений к БД
- Не создаем новое соединение для каждого запроса
- pgxpool автоматически управляет пулом

### RabbitMQ концепции

**Exchange**
- Маршрутизатор сообщений
- Type=topic: поддерживает wildcards (*,#)
- Пример: `driver_topic`

**Queue**
- Очередь сообщений FIFO
- Durable=true: переживет рестарт
- Пример: `ride_service_driver_responses`

**Routing Key**
- Строка для маршрутизации
- Пример: `driver.response.uuid-123`
- Pattern: `driver.response.*` (матчит любой ride_id)

**Binding**
- Связь Exchange ↔ Queue
- Exchange `driver_topic` + Pattern `driver.response.*` → Queue `ride_service_driver_responses`

**Ack/Nack**
- Ack: подтверждение обработки (удалить из очереди)
- Nack: отклонение (вернуть в очередь)
- Requeue=true: попробовать еще раз

---

## 🎓 Советы для изучения

### С чего начать читать код?

1. **Сначала бизнес-логика** (Use Case)
   - `internal/ride/application/usecase/handle_driver_response.go`
   - Здесь видна суть: что делает система

2. **Потом интерфейсы** (Ports)
   - `internal/ride/application/ports/in/handle_driver_response.go`
   - Понимаем контракты

3. **Затем адаптеры** (Adapters)
   - Consumer: `internal/ride/adapter/in/in_amqp/driver_response_consumer.go`
   - Repository: `internal/ride/adapter/out/repo/ride_pg_repository.go`

4. **Наконец, bootstrap** (Dependency Injection)
   - `internal/ride/bootstrap/compose.go`
   - Как все собирается вместе

### Как дебажить?

1. **Логи** - первое место для поиска
   ```bash
   docker logs ridehail-ride --tail 100 | grep "driver_assigned"
   ```

2. **RabbitMQ Management UI**
   - http://localhost:15672
   - Смотрим очереди, сообщения, bindings

3. **БД напрямую**
   ```sql
   SELECT id, status, driver_id, matched_at 
   FROM rides 
   WHERE id = 'uuid-123';
   ```

4. **Breakpoints в IDE**
   - Ставим в Use Case на критичных местах
   - Смотрим значения переменных

### Полезные вопросы для самопроверки

1. Почему Use Case не знает о RabbitMQ?
2. Что произойдет если два водителя примут поездку одновременно?
3. Зачем нужен PassengerID в Output?
4. Что случится если БД упадет во время UPDATE?
5. Почему мы используем Manual Ack вместо Auto Ack?

---

**Создано**: 2025-10-31  
**Автор**: AI Assistant с любовью к чистому коду ❤️
