package in_amqp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ridehail/internal/driver/adapters/in/in_ws"
	"ridehail/internal/driver/adapters/out/repo"
	"ridehail/internal/shared/logger"
	"ridehail/internal/shared/mq"

	amqp "github.com/rabbitmq/amqp091-go"
)

// RideRequestMessage структура входящего запроса на поездку
type RideRequestMessage struct {
	RideID         string       `json:"ride_id"`
	RideNumber     string       `json:"ride_number"`
	PickupLocation LocationData `json:"pickup_location"`
	DestLocation   LocationData `json:"destination_location"`
	VehicleType    string       `json:"ride_type"` // ECONOMY, PREMIUM, XL
	EstimatedFare  float64      `json:"estimated_fare"`
	MaxDistanceKm  float64      `json:"max_distance_km"`
	TimeoutSeconds int          `json:"timeout_seconds"`
	CorrelationID  string       `json:"correlation_id"`
}

// LocationData данные локации
type LocationData struct {
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
	Address string  `json:"address"`
}

// NearbyDriver информация о близком водителе
type NearbyDriver struct {
	DriverID   string
	DistanceKm float64
}

// RideRequestConsumer обрабатывает запросы на поездки
type RideRequestConsumer struct {
	mqConn       *mq.RabbitMQ
	locationRepo *repo.LocationRepository
	driverWS     *in_ws.DriverWSHandler
	log          *logger.Logger
}

// NewRideRequestConsumer создает новый consumer
func NewRideRequestConsumer(
	mqConn *mq.RabbitMQ,
	locationRepo *repo.LocationRepository,
	driverWS *in_ws.DriverWSHandler,
	log *logger.Logger,
) *RideRequestConsumer {
	return &RideRequestConsumer{
		mqConn:       mqConn,
		locationRepo: locationRepo,
		driverWS:     driverWS,
		log:          log,
	}
}

// Start запускает consumer
func (c *RideRequestConsumer) Start(ctx context.Context) error {
	c.log.Info(logger.Entry{
		Action:  "ride_request_consumer_starting",
		Message: "starting ride request consumer",
	})

	// Создаем канал
	ch := c.mqConn.Channel()
	if ch == nil {
		return fmt.Errorf("failed to get channel from RabbitMQ")
	}

	// Объявляем очередь для матчинга
	queueName := "driver_matching"
	_, err := ch.QueueDeclare(
		queueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Подписываемся на сообщения
	msgs, err := ch.Consume(
		queueName,
		"driver-service-ride-matcher", // consumer tag
		false,                         // auto-ack (мы будем ack вручную)
		false,                         // exclusive
		false,                         // no-local
		false,                         // no-wait
		nil,                           // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	c.log.Info(logger.Entry{
		Action:  "ride_request_consumer_started",
		Message: fmt.Sprintf("listening on queue: %s", queueName),
	})

	// Обрабатываем сообщения
	for {
		select {
		case <-ctx.Done():
			c.log.Info(logger.Entry{
				Action:  "ride_request_consumer_stopped",
				Message: "context cancelled",
			})
			return nil

		case msg, ok := <-msgs:
			if !ok {
				c.log.Warn(logger.Entry{
					Action:  "ride_request_consumer_channel_closed",
					Message: "message channel closed",
				})
				return fmt.Errorf("message channel closed")
			}

			// Обрабатываем сообщение
			if err := c.handleRideRequest(ctx, msg); err != nil {
				c.log.Error(logger.Entry{
					Action:  "ride_request_processing_failed",
					Message: err.Error(),
					Error:   &logger.ErrObj{Msg: err.Error()},
				})
				// Отклоняем сообщение (nack) и возвращаем в очередь
				_ = msg.Nack(false, true)
			} else {
				// Подтверждаем обработку
				_ = msg.Ack(false)
			}
		}
	}
}

// handleRideRequest обрабатывает один запрос на поездку
func (c *RideRequestConsumer) handleRideRequest(ctx context.Context, msg amqp.Delivery) error {
	// Парсим сообщение
	var request RideRequestMessage
	if err := json.Unmarshal(msg.Body, &request); err != nil {
		return fmt.Errorf("failed to parse ride request: %w", err)
	}

	c.log.Info(logger.Entry{
		Action:  "ride_request_received",
		Message: request.RideID,
		Additional: map[string]interface{}{
			"ride_id":      request.RideID,
			"ride_number":  request.RideNumber,
			"vehicle_type": request.VehicleType,
			"pickup_lat":   request.PickupLocation.Lat,
			"pickup_lng":   request.PickupLocation.Lng,
		},
	})

	// Ищем доступных водителей поблизости
	nearbyDrivers, err := c.findNearbyDrivers(
		ctx,
		request.PickupLocation.Lat,
		request.PickupLocation.Lng,
		request.VehicleType,
		request.MaxDistanceKm,
	)
	if err != nil {
		return fmt.Errorf("failed to find nearby drivers: %w", err)
	}

	if len(nearbyDrivers) == 0 {
		c.log.Warn(logger.Entry{
			Action:  "no_drivers_found",
			Message: request.RideID,
			Additional: map[string]interface{}{
				"ride_id":      request.RideID,
				"vehicle_type": request.VehicleType,
				"radius_km":    request.MaxDistanceKm,
			},
		})
		// TODO: Отправить уведомление в Ride Service что водители не найдены
		return nil
	}

	c.log.Info(logger.Entry{
		Action:  "drivers_found",
		Message: fmt.Sprintf("found %d drivers", len(nearbyDrivers)),
		Additional: map[string]interface{}{
			"ride_id": request.RideID,
			"count":   len(nearbyDrivers),
		},
	})

	// Отправляем оффер водителям через WebSocket
	offersSent := 0
	for _, driver := range nearbyDrivers {
		// Проверяем, подключен ли водитель
		if !c.driverWS.IsDriverConnected(driver.DriverID) {
			c.log.Debug(logger.Entry{
				Action:  "driver_not_connected",
				Message: driver.DriverID,
				Additional: map[string]interface{}{
					"driver_id": driver.DriverID,
					"ride_id":   request.RideID,
				},
			})
			continue
		}

		// Формируем оффер
		offer := map[string]interface{}{
			"offer_id":    fmt.Sprintf("offer_%s_%s", request.RideID, driver.DriverID),
			"ride_id":     request.RideID,
			"ride_number": request.RideNumber,
			"pickup_location": map[string]interface{}{
				"latitude":  request.PickupLocation.Lat,
				"longitude": request.PickupLocation.Lng,
				"address":   request.PickupLocation.Address,
			},
			"destination_location": map[string]interface{}{
				"latitude":  request.DestLocation.Lat,
				"longitude": request.DestLocation.Lng,
				"address":   request.DestLocation.Address,
			},
			"estimated_fare":              request.EstimatedFare,
			"driver_earnings":             request.EstimatedFare * 0.8, // 80% водителю
			"distance_to_pickup_km":       driver.DistanceKm,
			"estimated_ride_duration_min": 15, // TODO: Рассчитать реальную длительность
			"expires_at":                  time.Now().Add(time.Duration(request.TimeoutSeconds) * time.Second).Format(time.RFC3339),
		}

		// Отправляем оффер водителю
		if err := c.driverWS.SendRideOffer(driver.DriverID, offer); err != nil {
			c.log.Error(logger.Entry{
				Action:  "send_offer_failed",
				Message: err.Error(),
				Error:   &logger.ErrObj{Msg: err.Error()},
				Additional: map[string]interface{}{
					"driver_id": driver.DriverID,
					"ride_id":   request.RideID,
				},
			})
			continue
		}

		offersSent++
		c.log.Info(logger.Entry{
			Action:  "ride_offer_sent",
			Message: driver.DriverID,
			Additional: map[string]interface{}{
				"driver_id":   driver.DriverID,
				"ride_id":     request.RideID,
				"distance_km": driver.DistanceKm,
			},
		})

		// Отправляем офферы максимум 5 водителям
		if offersSent >= 5 {
			break
		}
	}

	c.log.Info(logger.Entry{
		Action:  "ride_offers_completed",
		Message: request.RideID,
		Additional: map[string]interface{}{
			"ride_id":     request.RideID,
			"offers_sent": offersSent,
		},
	})

	return nil
}

// findNearbyDrivers ищет доступных водителей в радиусе 5км от точки pickup
func (c *RideRequestConsumer) findNearbyDrivers(
	ctx context.Context,
	pickupLat, pickupLng float64,
	vehicleType string,
	maxDistanceKm float64,
) ([]NearbyDriver, error) {
	// Используем метод из LocationRepository для поиска ближайших водителей
	nearbyDrivers, err := c.locationRepo.FindNearbyOnlineDrivers(ctx, pickupLat, pickupLng, maxDistanceKm, 10)
	if err != nil {
		return nil, fmt.Errorf("find nearby drivers: %w", err)
	}

	var drivers []NearbyDriver
	for _, nd := range nearbyDrivers {
		drivers = append(drivers, NearbyDriver{
			DriverID:   nd.DriverID,
			DistanceKm: nd.Distance / 1000.0, // конвертируем метры в км
		})
	}

	return drivers, nil
}
