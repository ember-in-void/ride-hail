# 🔍 Как работает проект изнутри

> Детальное объяснение внутреннего устройства Ride-Hailing System

## 📑 Содержание

1. [Общая картина](#общая-картина)
2. [WebSocket - как это работает](#websocket---как-это-работает)
3. [RabbitMQ - брокер сообщений](#rabbitmq---брокер-сообщений)
4. [PostgreSQL - база данных](#postgresql---база-данных)
5. [Полный цикл запроса](#полный-цикл-запроса)
6. [Clean Architecture изнутри](#clean-architecture-изнутри)

---

## 🎯 Общая картина

### Что происходит когда пассажир создает поездку?

```
Пассажир нажимает "Заказать такси" в приложении
         │
         ▼
    HTTP запрос → Ride Service (порт 3000)
         │
         ├─► 1. JWT проверка (это реальный пользователь?)
         ├─► 2. Валидация данных (координаты корректны?)
         ├─► 3. Расчет стоимости (сколько стоит?)
         ├─► 4. Сохранение в PostgreSQL (в 3 таблицы!)
         └─► 5. Публикация в RabbitMQ → поиск водителя
                  │
                  ▼
            Driver Service получает сообщение
                  │
                  ├─► Ищет водителей в радиусе 5км (PostGIS)
                  ├─► Сортирует по рейтингу
                  └─► Отправляет через WebSocket → водителю
                           │
                           ▼
                      Водитель видит предложение
                      Нажимает "Принять"
                           │
                           ▼
                      WebSocket → Driver Service
                           │
                           ├─► Публикация в RabbitMQ
                           │
                           ▼
                      Ride Service получает ответ
                           │
                           ├─► UPDATE в PostgreSQL
                           └─► WebSocket → Пассажир
                                    │
                                    ▼
                           "Водитель найден!"
```

---

## 🔌 WebSocket - как это работает

### Что такое WebSocket простыми словами?

**HTTP** - это как письмо: ты отправил → получил ответ → соединение закрылось.  
**WebSocket** - это как телефонный звонок: соединение открыто постоянно, можно говорить в обе стороны.

### Внутреннее устройство WebSocket Hub

#### Файл: `internal/shared/ws/hub.go`

```
┌─────────────────────────────────────────────────┐
│              WebSocket Hub                       │
│  (Центральная станция для всех соединений)      │
│                                                  │
│  clients = map[clientID]*Client {                │
│    "client-1": &Client{UserID: "passenger-123"} │
│    "client-2": &Client{UserID: "driver-456"}    │
│    "client-3": &Client{UserID: "passenger-789"} │
│  }                                               │
└─────────────────────────────────────────────────┘
        ▲                    │
        │ register           │ SendToUser()
        │ unregister         ▼
   ┌────────┐         ┌──────────────┐
   │ Client │         │  Найти       │
   │        │         │  UserID в    │
   │ send   │◄────────│  map и       │
   │ chan   │         │  отправить   │
   └────────┘         └──────────────┘
```

### Как подключается клиент (пошагово)?

#### Шаг 1: HTTP Upgrade
```go
// Клиент делает HTTP запрос
GET ws://localhost:3001/ws HTTP/1.1
Upgrade: websocket
Connection: Upgrade

// Сервер "апгрейдит" соединение
conn, err := upgrader.Upgrade(w, r, nil)
// Теперь это WebSocket соединение!
```

#### Шаг 2: Создание Client
```go
// internal/shared/ws/hub.go

client := &Client{
    ID:   uuid.New().String(),    // Уникальный ID соединения
    conn: conn,                    // WebSocket соединение
    send: make(chan []byte, 256),  // Канал для отправки сообщений
    hub:  hub,                     // Ссылка на Hub
}
```

**Что такое канал (chan)?**
Канал - это труба между горутинами:
```go
send := make(chan []byte, 256)

// Горутина 1 кладет в трубу:
send <- []byte("Hello")

// Горутина 2 достает из трубы:
msg := <-send  // msg = "Hello"
```

#### Шаг 3: Регистрация в Hub
```go
// Отправляем клиента в Hub
hub.register <- client

// Hub получает в своей горутине:
case client := <-h.register:
    h.mu.Lock()
    h.clients[client.ID] = client  // Добавляем в map
    h.mu.Unlock()
```

**Зачем мьютекс (mu.Lock)?**
Представьте, что 2 человека одновременно пишут в одну тетрадь - будет каша!
Мьютекс - это "замок", только один может писать в один момент времени.

#### Шаг 4: Аутентификация (5 секунд!)
```go
// Клиент ДОЛЖЕН отправить токен в течение 5 секунд!
{
  "type": "auth",
  "token": "eyJhbGci..."
}

// Таймер в коде:
authTimer := time.NewTimer(authTimeout)  // 5 секунд
select {
case msg := <-msgChan:
    // Получили токен!
    userID, role, err := h.hub.authFunc(token)
    c.UserID = userID
    c.Role = role
case <-authTimer.C:
    // 5 секунд прошло, нет токена → закрываем!
    c.conn.Close()
}
```

#### Шаг 5: Две горутины работают постоянно

##### readPump() - читает сообщения ОТ клиента
```go
func (c *Client) readPump() {
    for {
        _, message, err := c.conn.ReadMessage()
        if err != nil {
            // Соединение разорвано
            c.hub.unregister <- c
            break
        }
        
        // Парсим JSON
        var msg struct {
            Type string          `json:"type"`
            Data json.RawMessage `json:"data"`
        }
        json.Unmarshal(message, &msg)
        
        // Вызываем обработчик
        c.hub.messageHandler(c, msg.Type, msg.Data)
    }
}
```

##### writePump() - отправляет сообщения К клиенту
```go
func (c *Client) writePump() {
    ticker := time.NewTicker(pingInterval)  // Каждые 30 сек
    
    for {
        select {
        case message := <-c.send:
            // Кто-то положил сообщение в канал!
            c.conn.WriteMessage(websocket.TextMessage, message)
            
        case <-ticker.C:
            // 30 секунд прошло, отправим ping
            c.conn.WriteMessage(websocket.PingMessage, nil)
        }
    }
}
```

### Как отправить сообщение пассажиру?

```go
// Где-то в коде (например, в RabbitMQ consumer):
hub.SendToUser("passenger-123", map[string]interface{}{
    "type": "ride_matched",
    "driver_id": "driver-456",
})

// Внутри Hub:
func (h *Hub) SendToUser(userID string, data interface{}) error {
    h.mu.RLock()  // Читаем из map
    defer h.mu.RUnlock()
    
    // Ищем клиента с таким UserID
    for _, client := range h.clients {
        if client.UserID == userID {
            msg, _ := json.Marshal(data)
            client.send <- msg  // Кладем в канал!
            // writePump() достанет и отправит
        }
    }
}
```

### Ping/Pong - проверка жизни

```
Сервер каждые 30 секунд:
    │
    ├─► Отправляет Ping
    │        │
    │        ▼
    │   Клиент получает
    │        │
    │        ├─► Автоматически отправляет Pong
    │        │
    ▼        ▼
Сервер ждет Pong (60 секунд)
    │
    ├─► Pong получен? → Все ОК
    └─► Нет Pong? → Соединение мертво, закрываем
```

Код:
```go
// writePump отправляет ping
c.conn.WriteMessage(websocket.PingMessage, nil)

// readPump настраивает таймер для pong
c.conn.SetReadDeadline(time.Now().Add(pongWait))
c.conn.SetPongHandler(func(string) error {
    c.conn.SetReadDeadline(time.Now().Add(pongWait))
    return nil
})
```

---

## 🐰 RabbitMQ - брокер сообщений

### Что такое RabbitMQ простыми словами?

RabbitMQ - это **почтовое отделение** для сервисов.

```
Ride Service хочет сказать Driver Service: "Найди водителя!"
    │
    ├─► НЕ звонит напрямую (тогда нужно ждать ответа)
    └─► Кладет письмо в RabbitMQ почтовый ящик
             │
             ▼
        Driver Service проверяет почту
             │
             └─► Достает письмо и обрабатывает
```

### Основные понятия

#### 1. Exchange (Почтовое отделение)

```
┌──────────────────────────────────┐
│      Exchange: ride_topic         │
│      Type: topic                  │
│                                   │
│  Правила маршрутизации:           │
│  ride.request.* → queue_1        │
│  ride.status.*  → queue_2        │
└──────────────────────────────────┘
```

**Topic Exchange** - это умная сортировка по шаблонам:
- `ride.request.ECONOMY` подходит под `ride.request.*` ✅
- `ride.request.PREMIUM` подходит под `ride.request.*` ✅
- `ride.status.MATCHED` подходит под `ride.status.*` ✅

**Fanout Exchange** - это радиовещание:
```
┌────────────────────────────┐
│  Exchange: location_fanout  │
│  Type: fanout               │
│                             │
│  Отправляет ВСЕМ очередям!  │
└────────────────────────────┘
       ▼       ▼       ▼
   Queue1  Queue2  Queue3
```

#### 2. Queue (Очередь)

```
Queue: driver_matching
┌────────┬────────┬────────┬────────┐
│ Msg 1  │ Msg 2  │ Msg 3  │ Msg 4  │
└────────┴────────┴────────┴────────┘
   ▲                           │
   │ enqueue                   │ dequeue
   │                           ▼
Producer                   Consumer
```

#### 3. Routing Key (Адрес)

```
"ride.request.ECONOMY"
  │      │       │
  │      │       └─ Тип поездки
  │      └─────── Действие
  └──────────── Домен
```

### Как это работает в коде?

#### Создание topology (настройка почтового отделения)

Файл: `internal/shared/mq/topology.go`

```go
func SetupTopology(ctx context.Context, conn *amqp091.Connection, log *logger.Logger) error {
    ch, _ := conn.Channel()
    
    // 1. Создаем Exchange
    ch.ExchangeDeclare(
        "ride_topic",  // name
        "topic",       // type
        true,          // durable (переживет перезагрузку)
        false,         // auto-delete
        false,         // internal
        false,         // no-wait
        nil,           // arguments
    )
    
    // 2. Создаем Queue
    ch.QueueDeclare(
        "driver_matching",  // name
        true,               // durable
        false,              // auto-delete
        false,              // exclusive
        false,              // no-wait
        nil,                // arguments
    )
    
    // 3. Связываем Queue с Exchange (binding)
    ch.QueueBind(
        "driver_matching",  // queue
        "ride.request.*",   // routing key pattern
        "ride_topic",       // exchange
        false,              // no-wait
        nil,                // arguments
    )
}
```

**Что значит durable=true?**
Если RabbitMQ перезагрузится, exchange и queue не исчезнут!

#### Публикация сообщения (отправка письма)

Файл: `internal/ride/adapter/out/out_amqp/ride_event_publisher.go`

```go
func (p *RideEventPublisher) PublishRideRequested(ctx context.Context, ride *domain.Ride) error {
    // 1. Формируем сообщение
    msg := RideRequestedMessage{
        RideID:      ride.ID,
        PassengerID: ride.PassengerID,
        VehicleType: ride.VehicleType,
        Pickup: Location{
            Lat: pickup.Latitude,
            Lng: pickup.Longitude,
        },
    }
    
    // 2. Сериализуем в JSON
    body, _ := json.Marshal(msg)
    
    // 3. Создаем AMQP сообщение
    publishing := amqp091.Publishing{
        ContentType:  "application/json",
        DeliveryMode: amqp091.Persistent,  // Сохранится на диск
        Timestamp:    time.Now(),
        Body:         body,
    }
    
    // 4. Публикуем!
    routingKey := fmt.Sprintf("ride.request.%s", ride.VehicleType)
    
    err := p.channel.PublishWithContext(ctx,
        "ride_topic",  // exchange
        routingKey,    // "ride.request.ECONOMY"
        false,         // mandatory
        false,         // immediate
        publishing,    // message
    )
    
    return err
}
```

**Что такое Persistent?**
Сообщение сохранится на диск. Если RabbitMQ упадет, сообщение не потеряется!

#### Получение сообщения (чтение почты)

Файл: `internal/driver/adapters/in/in_amqp/ride_consumer.go`

```go
func (c *RideConsumer) Start(ctx context.Context) error {
    ch, _ := c.conn.Channel()
    
    // 1. Настраиваем QoS (Quality of Service)
    ch.Qos(
        1,     // prefetch count - берем по 1 сообщению
        0,     // prefetch size
        false, // global
    )
    
    // 2. Начинаем получать сообщения
    msgs, _ := ch.Consume(
        "driver_matching",  // queue
        "",                 // consumer tag
        false,              // auto-ack (НЕ автоматически!)
        false,              // exclusive
        false,              // no-local
        false,              // no-wait
        nil,                // args
    )
    
    // 3. Обрабатываем в цикле
    for {
        select {
        case msg := <-msgs:
            // Десериализуем
            var rideReq RideRequestMessage
            json.Unmarshal(msg.Body, &rideReq)
            
            // Вызываем use case
            err := c.matchDriverUseCase.Execute(ctx, rideReq)
            
            if err != nil {
                // Ошибка! Вернем сообщение в очередь
                msg.Nack(false, true)  // requeue=true
            } else {
                // Успех! Подтверждаем
                msg.Ack(false)  // multiple=false
            }
            
        case <-ctx.Done():
            return nil  // Graceful shutdown
        }
    }
}
```

**Что такое prefetch=1?**
```
Без prefetch:
Consumer получает ВСЕ сообщения сразу → перегрузка!

С prefetch=1:
Consumer получает 1 сообщение → обрабатывает → Ack → получает следующее
```

**Что такое Ack/Nack?**

```
Ack (Acknowledgment) - "Я обработал, можно удалить"
    msg.Ack(false)
    │
    └─► RabbitMQ удаляет сообщение из очереди

Nack (Negative Ack) - "Не получилось, верни обратно"
    msg.Nack(false, true)  // requeue=true
    │
    └─► RabbitMQ возвращает сообщение в очередь
        Другой consumer (или тот же позже) попробует еще раз
```

### Полный цикл сообщения

```
1. Ride Service создает поездку
    │
    ▼
2. Публикует в RabbitMQ
    exchange: ride_topic
    routing: ride.request.ECONOMY
    │
    ▼
3. RabbitMQ маршрутизирует
    Паттерн ride.request.* подходит!
    │
    ▼
4. Сообщение попадает в очередь
    Queue: driver_matching
    │
    ▼
5. Driver Service читает из очереди
    msgs, _ := ch.Consume("driver_matching", ...)
    │
    ▼
6. Обрабатывает
    Ищет водителей в радиусе 5км
    │
    ├─► Успех? → msg.Ack(false)
    └─► Ошибка? → msg.Nack(false, true)
```

### Почему это круто?

1. **Асинхронность**: Ride Service не ждет ответа, работает дальше
2. **Надежность**: Если Driver Service упал, сообщения не потеряются
3. **Масштабирование**: Можем запустить 10 Driver Service - они разделят нагрузку
4. **Retry**: Если ошибка - Nack вернет сообщение в очередь

---

## 🗄️ PostgreSQL - база данных

### Connection Pool (бассейн соединений)

```
Приложение                  PostgreSQL
   │
   ├─► Запрос 1 ──┐
   ├─► Запрос 2 ──┤
   ├─► Запрос 3 ──┼─► Connection Pool (10 соединений)
   ├─► Запрос 4 ──┤        │
   └─► Запрос 5 ──┘        └─► Переиспользуем!
   
Без Pool:
  Каждый запрос = новое TCP соединение (медленно!)
  
С Pool:
  10 соединений открыты заранее, переиспользуем (быстро!)
```

Файл: `internal/shared/db/db.go`

```go
func NewPool(ctx context.Context, cfg config.DatabaseConfig, log *logger.Logger) (*pgxpool.Pool, error) {
    // Строка подключения
    connStr := fmt.Sprintf(
        "postgres://%s:%s@%s:%d/%s?sslmode=disable",
        cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database,
    )
    
    // Конфигурация pool
    poolConfig, _ := pgxpool.ParseConfig(connStr)
    poolConfig.MaxConns = 25                    // Максимум 25 соединений
    poolConfig.MinConns = 5                     // Минимум 5 всегда открыты
    poolConfig.MaxConnLifetime = time.Hour      // Пересоздавать через час
    poolConfig.MaxConnIdleTime = 30 * time.Minute
    poolConfig.HealthCheckPeriod = time.Minute
    
    // Создаем pool
    pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
    return pool, err
}
```

### Транзакции - все или ничего

Представьте: создание поездки требует записи в 3 таблицы:
1. `coordinates` - pickup точка
2. `coordinates` - destination точка  
3. `rides` - сама поездка

Что если после 2-й таблицы произойдет ошибка? Будут полуготовые данные!

**Транзакция** - это "атомарная операция":

```go
func (s *RequestRideService) Execute(ctx context.Context, input Input) (Output, error) {
    // Начинаем транзакцию
    tx, _ := s.db.Begin(ctx)
    defer tx.Rollback(ctx)  // Откатим если что-то пойдет не так
    
    // 1. Сохраняем pickup координаты
    pickupID, err := s.coordRepo.SaveInTx(ctx, tx, pickupCoord)
    if err != nil {
        return Output{}, err  // Rollback произойдет автоматически!
    }
    
    // 2. Сохраняем destination координаты
    destID, err := s.coordRepo.SaveInTx(ctx, tx, destCoord)
    if err != nil {
        return Output{}, err  // Rollback!
    }
    
    // 3. Создаем ride
    ride := &domain.Ride{
        PickupCoordinateID: pickupID,
        DestinationCoordinateID: destID,
        // ...
    }
    err = s.rideRepo.SaveInTx(ctx, tx, ride)
    if err != nil {
        return Output{}, err  // Rollback!
    }
    
    // ВСЕ УСПЕШНО! Коммитим транзакцию
    tx.Commit(ctx)
    
    return Output{RideID: ride.ID}, nil
}
```

**Что происходит внутри?**

```sql
BEGIN;  -- Начало транзакции

INSERT INTO coordinates (...) VALUES (...);  -- Шаг 1
INSERT INTO coordinates (...) VALUES (...);  -- Шаг 2
INSERT INTO rides (...) VALUES (...);        -- Шаг 3

-- Если все ОК:
COMMIT;  -- Сохраняем все изменения

-- Если ошибка на любом шаге:
ROLLBACK;  -- Откатываем ВСЕ изменения
```

### Race Condition Protection

Проблема: 2 водителя одновременно принимают одну поездку!

```
Время  Driver A                Driver B
t=0    SELECT * FROM rides     SELECT * FROM rides
       WHERE id='123'          WHERE id='123'
       status=REQUESTED ✓      status=REQUESTED ✓

t=1    UPDATE rides            UPDATE rides
       SET driver_id='A'       SET driver_id='B'
       WHERE id='123'          WHERE id='123'
       
Результат: Оба успешны! Поездка назначена двум водителям! ❌
```

**Решение: Атомарный UPDATE с WHERE**

```go
func (r *RidePgRepository) AssignDriver(ctx context.Context, rideID, driverID string) error {
    query := `
        UPDATE rides 
        SET driver_id = $1, 
            status = 'MATCHED',
            matched_at = NOW()
        WHERE id = $2 
          AND status = 'REQUESTED'  -- ← КЛЮЧЕВОЕ УСЛОВИЕ!
    `
    
    result, _ := r.pool.Exec(ctx, query, driverID, rideID)
    
    // Проверяем сколько строк обновилось
    rowsAffected := result.RowsAffected()
    
    if rowsAffected == 0 {
        // Статус уже не REQUESTED (кто-то успел раньше!)
        return errors.New("ride already assigned")
    }
    
    return nil
}
```

**Как это работает?**

```sql
Время  Driver A                           Driver B
t=0    UPDATE rides SET driver_id='A'     UPDATE rides SET driver_id='B'
       WHERE id='123'                      WHERE id='123'
       AND status='REQUESTED'              AND status='REQUESTED'
       
       -- PostgreSQL обрабатывает первым запрос Driver A:
       status='REQUESTED' ✓                (ждет...)
       UPDATE выполнен!
       status → 'MATCHED'
       RowsAffected = 1 ✓

t=1    (завершено)                         -- Теперь запрос Driver B:
                                            status='MATCHED' (уже не REQUESTED!)
                                            WHERE условие НЕ выполнено
                                            RowsAffected = 0 ❌
                                            
Результат: Driver A получил поездку ✓
          Driver B получил ошибку ✓
```

---

## 🔄 Полный цикл запроса

Давайте проследим ВЕСЬ путь от нажатия кнопки до уведомления:

### Создание поездки пассажиром

```
📱 Мобильное приложение пассажира
    │
    └─► POST http://localhost:3000/api/v1/rides/request
        Headers:
          Authorization: Bearer eyJhbGci...
        Body:
          {
            "pickup_latitude": 43.238949,
            "pickup_longitude": 76.889709,
            "destination_latitude": 43.222015,
            "destination_longitude": 76.851511,
            "ride_type": "ECONOMY"
          }
```

#### Слой 1: HTTP Handler (Adapter IN)

Файл: `internal/ride/adapter/in/transport/http_handler.go`

```go
func (h *HTTPHandler) handleRequestRide(w http.ResponseWriter, r *http.Request) {
    // 1. Извлекаем user из контекста (поставлен middleware)
    user := r.Context().Value("user").(*shared_user.User)
    
    // 2. Парсим JSON
    var req RequestRideRequest
    json.NewDecoder(r.Body).Decode(&req)
    
    // 3. Создаем Input для Use Case
    input := usecase.RequestRideInput{
        PassengerID:        user.ID,
        PickupLatitude:     req.PickupLatitude,
        PickupLongitude:    req.PickupLongitude,
        DestinationLatitude: req.DestinationLatitude,
        DestinationLongitude: req.DestinationLongitude,
        VehicleType:        req.RideType,
    }
    
    // 4. Вызываем Use Case
    output, err := h.requestRideUseCase.Execute(r.Context(), input)
    
    // 5. Отправляем ответ
    if err != nil {
        h.respondError(w, http.StatusBadRequest, err.Error())
        return
    }
    
    h.respondJSON(w, http.StatusCreated, output)
}
```

#### Слой 2: Use Case (Бизнес-логика)

Файл: `internal/ride/application/usecase/request_ride_usecase.go`

```go
func (s *RequestRideService) Execute(ctx context.Context, input Input) (Output, error) {
    // ШАГ 1: Валидация
    if input.PickupLatitude < -90 || input.PickupLatitude > 90 {
        return Output{}, errors.New("invalid latitude")
    }
    
    // ШАГ 2: Создаем координаты
    pickupCoord := &domain.Coordinate{
        ID:        uuid.New().String(),
        Latitude:  input.PickupLatitude,
        Longitude: input.PickupLongitude,
        Address:   input.PickupAddress,
    }
    
    destCoord := &domain.Coordinate{
        ID:        uuid.New().String(),
        Latitude:  input.DestinationLatitude,
        Longitude: input.DestinationLongitude,
        Address:   input.DestinationAddress,
    }
    
    // ШАГ 3: Рассчитываем расстояние и время
    distance := s.calculateDistance(pickupCoord, destCoord)
    duration := s.calculateDuration(distance)
    
    // ШАГ 4: Рассчитываем стоимость
    fare := s.calculateFare(input.VehicleType, distance, duration)
    
    // ШАГ 5: Генерируем ride number
    rideNumber := generateRideNumber()  // RIDE_20251031_111459_055
    
    // ШАГ 6: Создаем domain entity
    ride := &domain.Ride{
        ID:                      uuid.New().String(),
        RideNumber:              rideNumber,
        PassengerID:             input.PassengerID,
        VehicleType:             input.VehicleType,
        Status:                  "REQUESTED",
        EstimatedFare:           &fare,
        PickupCoordinateID:      pickupCoord.ID,
        DestinationCoordinateID: destCoord.ID,
    }
    
    // ШАГ 7: Сохраняем в БД (в транзакции!)
    err := s.saveRideWithCoordinates(ctx, ride, pickupCoord, destCoord)
    if err != nil {
        return Output{}, fmt.Errorf("save ride: %w", err)
    }
    
    // ШАГ 8: Публикуем событие в RabbitMQ
    s.eventPublisher.PublishRideRequested(ctx, ride, pickupCoord, destCoord)
    
    // ШАГ 9: Возвращаем результат
    return Output{
        RideID:            ride.ID,
        RideNumber:        ride.RideNumber,
        Status:            ride.Status,
        EstimatedFare:     *ride.EstimatedFare,
        EstimatedDistance: distance,
        EstimatedDuration: duration,
    }, nil
}
```

#### Слой 3: Repository (Adapter OUT - БД)

Файл: `internal/ride/adapter/out/repo/ride_pg_repository.go`

```go
func (r *RidePgRepository) saveRideWithCoordinates(
    ctx context.Context, 
    ride *domain.Ride,
    pickup, dest *domain.Coordinate,
) error {
    // Начинаем транзакцию
    tx, _ := r.pool.Begin(ctx)
    defer tx.Rollback(ctx)
    
    // 1. Сохраняем pickup coordinate
    _, err := tx.Exec(ctx, `
        INSERT INTO coordinates (id, latitude, longitude, address, entity_type, entity_id)
        VALUES ($1, $2, $3, $4, 'ride_pickup', $5)
    `, pickup.ID, pickup.Latitude, pickup.Longitude, pickup.Address, ride.ID)
    
    if err != nil {
        return err  // Автоматический rollback!
    }
    
    // 2. Сохраняем destination coordinate
    _, err = tx.Exec(ctx, `
        INSERT INTO coordinates (id, latitude, longitude, address, entity_type, entity_id)
        VALUES ($1, $2, $3, $4, 'ride_destination', $5)
    `, dest.ID, dest.Latitude, dest.Longitude, dest.Address, ride.ID)
    
    if err != nil {
        return err
    }
    
    // 3. Сохраняем ride
    _, err = tx.Exec(ctx, `
        INSERT INTO rides (
            id, ride_number, passenger_id, vehicle_type, status,
            estimated_fare, pickup_coordinate_id, destination_coordinate_id
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    `, ride.ID, ride.RideNumber, ride.PassengerID, ride.VehicleType,
       ride.Status, ride.EstimatedFare, pickup.ID, dest.ID)
    
    if err != nil {
        return err
    }
    
    // 4. Создаем ride event для audit trail
    _, err = tx.Exec(ctx, `
        INSERT INTO ride_events (id, ride_id, event_type, event_data)
        VALUES ($1, $2, 'RIDE_REQUESTED', $3)
    `, uuid.New().String(), ride.ID, `{"status": "REQUESTED"}`)
    
    if err != nil {
        return err
    }
    
    // ВСЕ УСПЕШНО! Коммитим
    return tx.Commit(ctx)
}
```

#### Слой 4: Event Publisher (Adapter OUT - RabbitMQ)

Файл: `internal/ride/adapter/out/out_amqp/ride_event_publisher.go`

```go
func (p *RideEventPublisher) PublishRideRequested(
    ctx context.Context,
    ride *domain.Ride,
    pickup, dest *domain.Coordinate,
) error {
    // Формируем сообщение
    msg := RideRequestedMessage{
        RideID:      ride.ID,
        RideNumber:  ride.RideNumber,
        PassengerID: ride.PassengerID,
        VehicleType: ride.VehicleType,
        Pickup: Location{
            Lat:     pickup.Latitude,
            Lng:     pickup.Longitude,
            Address: pickup.Address,
        },
        Destination: Location{
            Lat:     dest.Latitude,
            Lng:     dest.Longitude,
            Address: dest.Address,
        },
        EstimatedFare: *ride.EstimatedFare,
    }
    
    body, _ := json.Marshal(msg)
    
    // Публикуем в RabbitMQ
    routingKey := fmt.Sprintf("ride.request.%s", ride.VehicleType)
    
    return p.channel.PublishWithContext(ctx,
        "ride_topic",    // exchange
        routingKey,      // "ride.request.ECONOMY"
        false, false,
        amqp091.Publishing{
            ContentType:  "application/json",
            DeliveryMode: amqp091.Persistent,
            Body:         body,
        },
    )
}
```

### Driver Service получает сообщение

#### Слой 1: AMQP Consumer (Adapter IN)

Файл: `internal/driver/adapters/in/in_amqp/ride_consumer.go`

```go
func (c *RideConsumer) Start(ctx context.Context) error {
    ch, _ := c.conn.Channel()
    ch.Qos(1, 0, false)
    
    msgs, _ := ch.Consume("driver_matching", "", false, false, false, false, nil)
    
    for {
        select {
        case msg := <-msgs:
            // Парсим сообщение
            var rideReq RideRequestMessage
            json.Unmarshal(msg.Body, &rideReq)
            
            // Вызываем Use Case
            err := c.matchDriverUseCase.Execute(ctx, rideReq)
            
            if err != nil {
                msg.Nack(false, true)  // Вернуть в очередь
            } else {
                msg.Ack(false)  // Подтвердить
            }
        }
    }
}
```

#### Слой 2: Match Driver Use Case

```go
func (s *MatchDriverService) Execute(ctx context.Context, input Input) error {
    // 1. Ищем доступных водителей в радиусе 5км
    drivers, err := s.driverRepo.FindAvailableNearby(
        ctx,
        input.Pickup.Lat,
        input.Pickup.Lng,
        5000,  // 5km в метрах
    )
    
    // 2. Сортируем по рейтингу и расстоянию
    sort.Slice(drivers, func(i, j int) bool {
        return drivers[i].Rating > drivers[j].Rating
    })
    
    // 3. Отправляем предложение ТОП-5 водителям через WebSocket
    for i := 0; i < min(5, len(drivers)); i++ {
        driver := drivers[i]
        
        offer := RideOffer{
            RideID:        input.RideID,
            PassengerName: "Пассажир",
            Pickup:        input.Pickup,
            Destination:   input.Destination,
            Fare:          input.EstimatedFare,
        }
        
        // Отправляем через WebSocket!
        s.wsHub.SendToUser(driver.ID, map[string]interface{}{
            "type": "ride_offer",
            "data": offer,
        })
    }
    
    return nil
}
```

### Водитель принимает поездку

```
📱 Приложение водителя
    │
    │ WebSocket соединение активно
    │
    ├─► Получает: {"type": "ride_offer", "data": {...}}
    │
    └─► Водитель нажимает "Принять"
         │
         └─► Отправляет через WebSocket:
             {
               "type": "ride_response",
               "data": {
                 "ride_id": "uuid-123",
                 "accepted": true
               }
             }
```

#### WebSocket Handler обрабатывает

Файл: `internal/driver/adapters/in/in_ws/driver_ws.go`

```go
func (h *DriverWSHandler) handleRideResponse(client *ws.Client, data json.RawMessage) error {
    var response RideResponseMessage
    json.Unmarshal(data, &response)
    
    // Извлекаем driver_id из JWT claims
    driverID := client.UserID
    
    // Публикуем в RabbitMQ
    return h.publisher.PublishDriverResponse(
        context.Background(),
        response.RideID,
        driverID,
        response.Accepted,
    )
}
```

#### RabbitMQ Publisher

```go
func (p *DriverResponsePublisher) PublishDriverResponse(
    ctx context.Context,
    rideID, driverID string,
    accepted bool,
) error {
    msg := DriverResponseMessage{
        RideID:   rideID,
        DriverID: driverID,
        Accepted: accepted,
    }
    
    body, _ := json.Marshal(msg)
    
    routingKey := fmt.Sprintf("driver.response.%s", rideID)
    
    return p.channel.PublishWithContext(ctx,
        "driver_topic",  // exchange
        routingKey,      // "driver.response.uuid-123"
        false, false,
        amqp091.Publishing{
            ContentType:  "application/json",
            DeliveryMode: amqp091.Persistent,
            Body:         body,
        },
    )
}
```

### Ride Service обрабатывает ответ водителя

#### AMQP Consumer

Файл: `internal/ride/adapter/in/in_amqp/driver_response_consumer.go`

```go
func (c *DriverResponseConsumer) Start(ctx context.Context) error {
    ch, _ := c.conn.Channel()
    ch.Qos(1, 0, false)
    
    // Привязываемся к очереди с паттерном driver.response.*
    msgs, _ := ch.Consume("ride_service_driver_responses", "", false, false, false, false, nil)
    
    for {
        select {
        case msg := <-msgs:
            var response DriverResponseMessage
            json.Unmarshal(msg.Body, &response)
            
            // Вызываем Use Case
            input := usecase.HandleDriverResponseInput{
                RideID:   response.RideID,
                DriverID: response.DriverID,
                Accepted: response.Accepted,
            }
            
            output, err := c.handleDriverResponseUseCase.Execute(ctx, input)
            
            if err != nil {
                msg.Nack(false, true)
                continue
            }
            
            // Успешно! Отправляем уведомление пассажиру
            c.passengerWS.GetHub().SendToUser(output.PassengerID, map[string]interface{}{
                "type": "ride_matched",
                "data": map[string]interface{}{
                    "ride_id":   output.RideID,
                    "driver_id": output.DriverID,
                    "status":    "MATCHED",
                },
            })
            
            msg.Ack(false)
        }
    }
}
```

#### Use Case - назначение водителя

Файл: `internal/ride/application/usecase/handle_driver_response.go`

```go
func (s *HandleDriverResponseService) Execute(ctx context.Context, input Input) (Output, error) {
    // ШАГ 1: Проверяем что водитель принял
    if !input.Accepted {
        // Водитель отказался, просто логируем
        return Output{}, nil
    }
    
    // ШАГ 2: Загружаем поездку
    ride, err := s.rideRepo.FindByID(ctx, input.RideID)
    if err != nil {
        return Output{}, err
    }
    
    // ШАГ 3: Проверяем что поездка еще не назначена
    if ride.Status != "REQUESTED" {
        return Output{}, errors.New("ride already assigned")
    }
    
    // ШАГ 4: Атомарно назначаем водителя с race condition protection!
    err = s.rideRepo.AssignDriver(ctx, input.RideID, input.DriverID)
    if err != nil {
        return Output{}, err
    }
    
    // ШАГ 5: Возвращаем PassengerID для WebSocket уведомления
    return Output{
        RideID:      ride.ID,
        DriverID:    input.DriverID,
        PassengerID: ride.PassengerID,
        Status:      "MATCHED",
    }, nil
}
```

#### Repository - UPDATE с защитой

```go
func (r *RidePgRepository) AssignDriver(ctx context.Context, rideID, driverID string) error {
    query := `
        UPDATE rides 
        SET driver_id = $1, 
            status = 'MATCHED',
            matched_at = NOW()
        WHERE id = $2 
          AND status = 'REQUESTED'  -- ← Защита от race condition!
    `
    
    result, err := r.pool.Exec(ctx, query, driverID, rideID)
    if err != nil {
        return err
    }
    
    if result.RowsAffected() == 0 {
        return errors.New("ride not found or already assigned")
    }
    
    return nil
}
```

### Пассажир получает уведомление

```
📱 Приложение пассажира
    │
    │ WebSocket соединение активно
    │
    └─► Получает через WebSocket:
        {
          "type": "ride_matched",
          "data": {
            "ride_id": "uuid-123",
            "driver_id": "uuid-456",
            "status": "MATCHED"
          }
        }
         │
         └─► Показывает: "Водитель найден! Едет к вам"
```

---

## 🏗️ Clean Architecture изнутри

### Почему слои?

```
Плохой код (все в одном файле):
┌─────────────────────────────────────┐
│  func CreateRide() {                │
│    // Парсим JSON                   │
│    // Валидация                     │
│    // SQL запросы                   │
│    // RabbitMQ публикация           │
│    // WebSocket отправка            │
│  }                                  │
└─────────────────────────────────────┘
❌ Невозможно тестировать
❌ Нельзя заменить БД
❌ Сложно понять


Clean Architecture (слои):
┌─────────────────────────────────────┐
│         HTTP Handler                │ ← Парсит JSON
│  (Adapter IN)                       │
└──────────────┬──────────────────────┘
               ▼
┌─────────────────────────────────────┐
│         Use Case                    │ ← Бизнес-логика
│  (Application)                      │
└──────────────┬──────────────────────┘
               ▼
┌─────────────────────────────────────┐
│         Repository                  │ ← SQL запросы
│  (Adapter OUT)                      │
└─────────────────────────────────────┘
✅ Легко тестировать (mock repository)
✅ Можно заменить PostgreSQL на MongoDB
✅ Каждый слой делает одно дело
```

### Dependency Inversion (Инверсия зависимостей)

```
Плохо:
Use Case зависит от конкретной реализации
┌────────────────┐
│   Use Case     │
│                │
│  repo *Postgre │ ← Прямая зависимость!
└────────────────┘

Нельзя заменить PostgreSQL без изменения Use Case!


Хорошо:
Use Case зависит от интерфейса
┌────────────────┐
│   Use Case     │
│                │
│  repo Repository│ ← Интерфейс!
└────────┬───────┘
         │ implements
         ▼
┌────────────────┐
│PostgreRepository│
└────────────────┘

Можно легко заменить:
┌────────────────┐
│ MongoRepository │
└────────────────┘

┌────────────────┐
│InMemoryRepository│ ← Для тестов!
└────────────────┘
```

Код:

```go
// ports/out/ride_repository.go (Интерфейс)
type RideRepository interface {
    Save(ctx context.Context, ride *domain.Ride) error
    FindByID(ctx context.Context, id string) (*domain.Ride, error)
    AssignDriver(ctx context.Context, rideID, driverID string) error
}

// usecase/request_ride.go (Use Case)
type RequestRideService struct {
    rideRepo RideRepository  // ← Зависит от интерфейса!
}

// adapter/out/repo/ride_pg_repository.go (Реализация)
type RidePgRepository struct {
    pool *pgxpool.Pool
}

func (r *RidePgRepository) Save(ctx context.Context, ride *domain.Ride) error {
    // PostgreSQL implementation
}

// ✅ Implements RideRepository interface!
```

### Bootstrap (Dependency Injection)

Файл: `internal/ride/bootstrap/compose.go`

```go
func Run(ctx context.Context, cfg config.Config, log *logger.Logger) {
    // 1. Создаем инфраструктуру
    dbPool, _ := db.NewPool(ctx, cfg.Database, log)
    mqConn, _ := mq.NewRabbitMQ(ctx, cfg.RabbitMQ, log)
    wsHub := ws.NewHub(authFunc, log)
    
    // 2. Создаем Repositories (Adapters OUT)
    rideRepo := repo.NewRidePgRepository(dbPool, log)
    coordRepo := repo.NewCoordinatePgRepository(dbPool, log)
    
    // 3. Создаем Publishers (Adapters OUT)
    eventPublisher := out_amqp.NewRideEventPublisher(mqConn, log)
    rideNotifier := out_ws.NewWsRideNotifier(wsHub, log)
    
    // 4. Создаем Use Cases (Application) - ВНЕДРЯЕМ зависимости!
    requestRideUC := usecase.NewRequestRideService(
        rideRepo,        // ← Dependency Injection!
        coordRepo,
        eventPublisher,
        rideNotifier,
        log,
    )
    
    handleDriverResponseUC := usecase.NewHandleDriverResponseService(
        rideRepo,  // ← Тот же интерфейс, та же реализация
        log,
    )
    
    // 5. Создаем HTTP Handler (Adapter IN)
    httpHandler := transport.NewHTTPHandler(requestRideUC, log)
    
    // 6. Создаем AMQP Consumers (Adapters IN)
    driverResponseConsumer := inamqp.NewDriverResponseConsumer(
        mqConn,
        handleDriverResponseUC,  // ← Используем Use Case
        passengerWS,
        log,
    )
    
    // 7. Запускаем все!
    go wsHub.Run(ctx)
    go driverResponseConsumer.Start(ctx)
    http.ListenAndServe(":3000", mux)
}
```

**Что здесь происходит?**

1. **Создаем все зависимости** (БД, MQ, WS) в одном месте
2. **Внедряем зависимости** в конструкторы (Dependency Injection)
3. **Use Cases не знают** о PostgreSQL - только об интерфейсе!
4. **Легко тестировать** - подменяем real repository на mock
5. **Легко менять** - заменили PostgreSQL на MongoDB? Меняем только 1 файл!

---

## 🎓 Итоговая картина

```
┌──────────────────────────────────────────────────────────┐
│                    RIDE-HAIL SYSTEM                       │
└──────────────────────────────────────────────────────────┘

Пассажир                 Водитель
   │                        │
   │ HTTP                   │ WebSocket
   ▼                        ▼
┌─────────┐              ┌──────────┐
│  Ride   │◄─RabbitMQ───►│  Driver  │
│ Service │              │ Service  │
└────┬────┘              └─────┬────┘
     │                         │
     │ PostgreSQL              │
     ▼                         ▼
┌──────────────────────────────────┐
│         Database                 │
│  - rides                         │
│  - coordinates                   │
│  - drivers                       │
│  - location_history              │
└──────────────────────────────────┘

RabbitMQ Exchanges:
  ride_topic ──► driver_matching queue
  driver_topic ──► ride_service_driver_responses queue
  location_fanout ──► все подписчики

WebSocket Hubs:
  Passenger Hub (Ride Service:3000)
    ├─ client-1 (passenger-123)
    └─ client-2 (passenger-456)
  
  Driver Hub (Driver Service:3001)
    ├─ client-3 (driver-789)
    └─ client-4 (driver-012)
```

---

## 💡 Ключевые моменты для понимания

1. **WebSocket** - это постоянное соединение, как телефонный звонок. Сообщения идут в обе стороны в реальном времени.

2. **RabbitMQ** - это почтовое отделение. Сервисы не звонят друг другу, а отправляют письма в очереди.

3. **PostgreSQL Connection Pool** - это бассейн готовых соединений. Переиспользуем их вместо создания новых каждый раз.

4. **Транзакции** - это "все или ничего". Либо ВСЕ операции успешны, либо ВСЕ откатываются.

5. **Race Condition Protection** - используем `WHERE status='REQUESTED'` чтобы только один водитель мог принять поездку.

6. **Clean Architecture** - слои не знают о деталях друг друга. Use Case работает с интерфейсами, а не с конкретными реализациями.

7. **Dependency Injection** - создаем все зависимости в одном месте (bootstrap) и внедряем через конструкторы.

---

**Теперь вы понимаете как работает проект изнутри!** 🎉
