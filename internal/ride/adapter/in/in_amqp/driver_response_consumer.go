package inamqp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ridehail/internal/ride/adapter/in/in_ws"
	"ridehail/internal/ride/application/ports/in"
	"ridehail/internal/shared/logger"
	"ridehail/internal/shared/mq"

	amqp "github.com/rabbitmq/amqp091-go"
)

// ============================================================================
// АДАПТЕР ВХОДЯЩИХ СООБЩЕНИЙ (Inbound Adapter)
// ============================================================================
// Слушает очередь RabbitMQ и преобразует сообщения в вызовы бизнес-логики.
//
// ПОТОК ДАННЫХ:
// 1. Driver нажимает "Принять" в мобильном приложении
// 2. WebSocket → Driver Service
// 3. Driver Service публикует: driver.response.{ride_id} → RabbitMQ
// 4. RabbitMQ → ЭТО CONSUMER → Use Case → Database
// 5. Use Case возвращает PassengerID
// 6. Consumer отправляет уведомление пассажиру через WebSocket
//
// АРХИТЕКТУРНЫЙ ПАТТЕРН: Adapter Pattern (Hexagonal Architecture)
// - Внешний слой (адаптер) зависит от внутреннего (use case)
// - Use case НЕ знает о RabbitMQ, WebSocket, HTTP
// ============================================================================

// DriverResponseMessage — формат JSON сообщения из RabbitMQ.
//
// Публикуется Driver Service в топик "driver_topic" с routing key:
// "driver.response.{ride_id}"
//
// Пример сообщения:
//
//	{
//	  "ride_id": "uuid-123",
//	  "driver_id": "uuid-456",
//	  "accepted": true,
//	  "estimated_arrival_minutes": 5,
//	  "driver_location": {"lat": 43.238, "lng": 76.889},
//	  "driver_info": {
//	    "name": "Иван Петров",
//	    "rating": 4.8,
//	    "vehicle": {"make": "Toyota", "model": "Camry", "plate": "KZ 777 ABC"}
//	  }
//	}
type DriverResponseMessage struct {
	RideID                  string  `json:"ride_id"`                             // UUID поездки
	DriverID                string  `json:"driver_id"`                           // UUID водителя
	Accepted                bool    `json:"accepted"`                            // true=принял, false=отклонил
	EstimatedArrivalMinutes int     `json:"estimated_arrival_minutes,omitempty"` // ETA в минутах
	DriverLocation          *LocDTO `json:"driver_location,omitempty"`           // Координаты водителя
	DriverInfo              *DrvDTO `json:"driver_info,omitempty"`               // Данные для пассажира
	CorrelationID           string  `json:"correlation_id,omitempty"`            // ID для трейсинга
}

// LocDTO — координаты (lat/lng).
type LocDTO struct {
	Lat float64 `json:"lat"` // Широта (Latitude)
	Lng float64 `json:"lng"` // Долгота (Longitude)
}

// DrvDTO — информация о водителе для отображения пассажиру.
type DrvDTO struct {
	Name    string      `json:"name"`              // Имя водителя
	Rating  float64     `json:"rating"`            // Рейтинг (1.0 - 5.0)
	Vehicle *VehicleDTO `json:"vehicle,omitempty"` // Данные автомобиля
}

// VehicleDTO — данные автомобиля.
type VehicleDTO struct {
	Make  string `json:"make"`  // Марка (Toyota, BMW, ...)
	Model string `json:"model"` // Модель (Camry, X5, ...)
	Color string `json:"color"` // Цвет (White, Black, ...)
	Plate string `json:"plate"` // Гос. номер (KZ 777 ABC)
}

// DriverResponseConsumer — слушатель очереди RabbitMQ для ответов водителей.
//
// Зависимости:
//   - mqConn: подключение к RabbitMQ
//   - handleDriverResponseUseCase: бизнес-логика (через интерфейс!)
//   - passengerWS: WebSocket hub для отправки уведомлений пассажирам
//   - log: структурированное логирование
//
// Паттерн Dependency Injection:
// Все зависимости передаются через конструктор, а не создаются внутри.
// Это позволяет легко тестировать с mock-объектами.
type DriverResponseConsumer struct {
	mqConn                      *mq.RabbitMQ                   // RabbitMQ connection
	handleDriverResponseUseCase in.HandleDriverResponseUseCase // Бизнес-логика (интерфейс)
	passengerWS                 *in_ws.PassengerWSHandler      // WebSocket для пассажиров
	log                         *logger.Logger                 // Логгер
}

// NewDriverResponseConsumer — фабрика для создания consumer.
//
// Dependency Injection: принимает все зависимости извне.
func NewDriverResponseConsumer(
	mqConn *mq.RabbitMQ,
	handleDriverResponseUseCase in.HandleDriverResponseUseCase,
	passengerWS *in_ws.PassengerWSHandler,
	log *logger.Logger,
) *DriverResponseConsumer {
	return &DriverResponseConsumer{
		mqConn:                      mqConn,
		handleDriverResponseUseCase: handleDriverResponseUseCase,
		passengerWS:                 passengerWS,
		log:                         log,
	}
}

// Start — главный метод: запускает прослушивание очереди RabbitMQ.
//
// НАСТРОЙКА RABBITMQ:
// 1. Объявляет очередь "ride_service_driver_responses" (durable=true для персистентности)
// 2. Привязывает очередь к exchange "driver_topic" с pattern "driver.response.*"
// 3. Настраивает prefetch count для fair dispatch между воркерами
// 4. Запускает бесконечный цикл обработки сообщений
//
// ROUTING KEY PATTERN:
// "driver.response.*" означает:
// - driver.response.uuid-123 ✓ (матчится)
// - driver.response.uuid-456 ✓ (матчится)
// - driver.request.uuid-789  ✗ (не матчится)
//
// ВАЖНО: Метод блокирующий, запускать в горутине!
// Пример: go consumer.Start(ctx)
func (c *DriverResponseConsumer) Start(ctx context.Context) error {
	// Получаем канал RabbitMQ из пула
	ch := c.mqConn.Channel()
	if ch == nil {
		return fmt.Errorf("failed to get RabbitMQ channel")
	}

	// ШАГ 1: Объявляем очередь
	// Durable=true значит очередь переживет рестарт RabbitMQ сервера
	queueName := "ride_service_driver_responses"
	queue, err := ch.QueueDeclare(
		queueName, // name: имя очереди
		true,      // durable: сохраняется при рестарте RabbitMQ
		false,     // auto-delete: НЕ удалять когда нет подписчиков
		false,     // exclusive: НЕ эксклюзивная (могут быть несколько consumers)
		false,     // no-wait: ждать подтверждения от сервера
		nil,       // arguments: дополнительные параметры (пока нет)
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// ШАГ 2: Привязываем очередь к exchange через routing key
	// Exchange "driver_topic" (type=topic) позволяет использовать wildcards:
	// * (звездочка) = ровно одно слово
	// # (решетка) = ноль или больше слов
	err = ch.QueueBind(
		queue.Name,          // queue name: наша очередь
		"driver.response.*", // routing key: шаблон для фильтрации сообщений
		"driver_topic",      // exchange: откуда берем сообщения
		false,               // no-wait: ждать подтверждения
		nil,                 // arguments: дополнительные параметры
	)
	if err != nil {
		return fmt.Errorf("failed to bind queue: %w", err)
	}

	// ШАГ 3: Настраиваем Quality of Service (QoS)
	// Prefetch count = 1 означает:
	// "Не давай мне следующее сообщение, пока я не обработал текущее"
	// Это обеспечивает fair dispatch между несколькими воркерами
	err = ch.Qos(
		1,     // prefetch count: максимум 1 необработанное сообщение
		0,     // prefetch size: 0 = без лимита по размеру
		false, // global: применить только к этому каналу
	)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	// ШАГ 4: Подписываемся на сообщения из очереди
	msgs, err := ch.Consume(
		queue.Name, // queue: откуда читаем
		"",         // consumer tag: автогенерируется RabbitMQ
		false,      // auto-ack: НЕТ! Будем подтверждать вручную (msg.Ack)
		false,      // exclusive: разрешаем множественных consumers
		false,      // no-local: неприменимо для AMQP 0.9.1
		false,      // no-wait: ждать подтверждения от сервера
		nil,        // args: дополнительные параметры
	)
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	// Логируем успешный старт для мониторинга
	c.log.Info(logger.Entry{
		Action:  "driver_response_consumer_started",
		Message: fmt.Sprintf("listening on driver_topic (queue: %s, pattern: driver.response.*)", queueName),
	})

	// ШАГ 5: Бесконечный цикл обработки сообщений
	// Блокируется здесь до ctx.Done() или закрытия msgs канала
	for {
		select {
		// Case 1: Контекст отменен (graceful shutdown)
		case <-ctx.Done():
			c.log.Info(logger.Entry{
				Action:  "driver_response_consumer_stopping",
				Message: "context cancelled",
			})
			return ctx.Err()

		// Case 2: Получено новое сообщение из RabbitMQ
		case msg, ok := <-msgs:
			// Проверяем, не закрыт ли канал (произошел дисконнект)
			if !ok {
				c.log.Warn(logger.Entry{
					Action:  "driver_response_consumer_channel_closed",
					Message: "message channel closed",
				})
				return fmt.Errorf("message channel closed")
			}

			// Обрабатываем сообщение
			// handleDriverResponse может вернуть ошибку если:
			// - JSON невалидный
			// - Use case вернул ошибку (поездка не найдена, уже назначена)
			// - WebSocket упал при отправке уведомления
			if err := c.handleDriverResponse(ctx, msg); err != nil {
				c.log.Error(logger.Entry{
					Action:  "handle_driver_response_failed",
					Message: err.Error(),
					Error:   &logger.ErrObj{Msg: err.Error()},
				})
				// ВАЖНО: Nack с requeue=true
				// Сообщение вернется в очередь для повторной обработки
				// Это защищает от потери данных при временных сбоях
				_ = msg.Nack(false, true) // multiple=false, requeue=true
			} else {
				// SUCCESS: Подтверждаем успешную обработку
				// Сообщение удаляется из очереди
				_ = msg.Ack(false) // multiple=false
			}
		}
	}
}

// handleDriverResponse — обработчик одного сообщения.
//
// ПОТОК ВЫПОЛНЕНИЯ:
// 1. Парсинг JSON → DriverResponseMessage
// 2. Логирование для мониторинга
// 3. Преобразование в Input для use case
// 4. Вызов бизнес-логики (use case)
// 5. Отправка WebSocket уведомления пассажиру
//
// ОБРАБОТКА ОШИБОК:
// - JSON parse error → возвращаем ошибку (Nack + requeue)
// - Use case error → возвращаем ошибку (Nack + requeue)
// - WebSocket error → логируем, но НЕ возвращаем ошибку (Ack сообщение)
//
// Почему WebSocket ошибка не критична?
// Потому что данные уже сохранены в БД. Пассажир увидит обновление
// при следующем pull или reconnect к WebSocket.
func (c *DriverResponseConsumer) handleDriverResponse(ctx context.Context, msg amqp.Delivery) error {
	// ШАГ 1: Десериализация JSON в структуру Go
	var response DriverResponseMessage
	if err := json.Unmarshal(msg.Body, &response); err != nil {
		return fmt.Errorf("failed to parse driver response: %w", err)
	}

	// ШАГ 2: Логирование для отладки и мониторинга
	// Routing key содержит ride_id: driver.response.{ride_id}
	c.log.Info(logger.Entry{
		Action:  "driver_response_received",
		Message: fmt.Sprintf("ride=%s, driver=%s, accepted=%t", response.RideID, response.DriverID, response.Accepted),
		RideID:  response.RideID,
		Additional: map[string]any{
			"driver_id":   response.DriverID,
			"accepted":    response.Accepted,
			"routing_key": msg.RoutingKey,
		},
	})

	// Извлекаем ride_id из routing key (driver.response.{ride_id})
	parts := strings.Split(msg.RoutingKey, ".")
	if len(parts) != 3 {
		return fmt.Errorf("invalid routing key format: %s", msg.RoutingKey)
	}
	rideID := parts[2]

	// Вызываем use case для обработки ответа водителя
	useCaseInput := in.HandleDriverResponseInput{
		RideID:                  rideID,
		DriverID:                response.DriverID,
		Accepted:                response.Accepted,
		EstimatedArrivalMinutes: response.EstimatedArrivalMinutes,
	}

	if response.DriverLocation != nil {
		useCaseInput.DriverLocationLat = response.DriverLocation.Lat
		useCaseInput.DriverLocationLng = response.DriverLocation.Lng
	}

	output, err := c.handleDriverResponseUseCase.Execute(ctx, useCaseInput)
	if err != nil {
		c.log.Error(logger.Entry{
			Action:  "handle_driver_response_usecase_failed",
			Message: err.Error(),
			RideID:  rideID,
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		return fmt.Errorf("execute use case: %w", err)
	}

	if output.DriverAssigned {
		// Водитель назначен - отправляем уведомление пассажиру через WebSocket
		c.log.Info(logger.Entry{
			Action:  "sending_ride_matched_notification",
			Message: fmt.Sprintf("driver matched for ride %s", rideID),
			RideID:  rideID,
			Additional: map[string]any{
				"driver_id":    response.DriverID,
				"passenger_id": output.PassengerID,
			},
		})

		// TODO: Отправить WebSocket уведомление пассажиру
		// notification := map[string]any{
		//   "type": "ride_matched",
		//   "ride_id": rideID,
		//   "driver_id": response.DriverID,
		//   "estimated_arrival_minutes": response.EstimatedArrivalMinutes,
		//   "driver_info": response.DriverInfo,
		//   "driver_location": response.DriverLocation,
		// }
		// c.passengerWS.SendToPassenger(output.PassengerID, notification)
	}

	return nil
}
