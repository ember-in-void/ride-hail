package in_amqp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ridehail/internal/driver/application/ports/out"
	"ridehail/internal/shared/logger"
	"ridehail/internal/shared/mq"

	"github.com/google/uuid"
	amqp091 "github.com/rabbitmq/amqp091-go"
)

// RideConsumer слушает очередь ride.requested и отправляет ride offers водителям
type RideConsumer struct {
	mq             *mq.RabbitMQ
	driverRepo     out.DriverRepository
	driverNotifier out.DriverNotifier
	eventPublisher out.EventPublisher
	log            *logger.Logger
}

func NewRideConsumer(
	mq *mq.RabbitMQ,
	driverRepo out.DriverRepository,
	driverNotifier out.DriverNotifier,
	eventPublisher out.EventPublisher,
	log *logger.Logger,
) *RideConsumer {
	return &RideConsumer{
		mq:             mq,
		driverRepo:     driverRepo,
		driverNotifier: driverNotifier,
		eventPublisher: eventPublisher,
		log:            log,
	}
}

// Start запускает консьюмер для очереди ride.requested
func (c *RideConsumer) Start(ctx context.Context) error {
	c.log.Info(logger.Entry{
		Action:  "ride_consumer_starting",
		Message: "starting ride request consumer",
	})

	return c.mq.Consume(ctx, "ride.requested", "driver-service", func(msg amqp091.Delivery) {
		c.handleRideRequest(ctx, msg)
	})
}

func (c *RideConsumer) handleRideRequest(ctx context.Context, msg amqp091.Delivery) {
	var rideRequest map[string]any
	if err := json.Unmarshal(msg.Body, &rideRequest); err != nil {
		c.log.Error(logger.Entry{
			Action:  "ride_request_unmarshal_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		_ = msg.Nack(false, false) // dead letter queue
		return
	}

	rideID, _ := rideRequest["ride_id"].(string)
	vehicleType, _ := rideRequest["ride_type"].(string)

	pickupLoc, ok := rideRequest["pickup_location"].(map[string]any)
	if !ok {
		c.log.Warn(logger.Entry{
			Action:  "ride_request_invalid_pickup",
			Message: "pickup_location missing or invalid",
			RideID:  rideID,
		})
		_ = msg.Nack(false, false)
		return
	}

	pickupLat, _ := pickupLoc["lat"].(float64)
	pickupLng, _ := pickupLoc["lng"].(float64)
	pickupAddr, _ := pickupLoc["address"].(string)

	destLoc, ok := rideRequest["destination_location"].(map[string]any)
	if !ok {
		c.log.Warn(logger.Entry{
			Action:  "ride_request_invalid_destination",
			Message: "destination_location missing or invalid",
			RideID:  rideID,
		})
		_ = msg.Nack(false, false)
		return
	}

	destLat, _ := destLoc["lat"].(float64)
	destLng, _ := destLoc["lng"].(float64)
	destAddr, _ := destLoc["address"].(string)

	estimatedFare, _ := rideRequest["estimated_fare"].(float64)

	c.log.Info(logger.Entry{
		Action:  "ride_request_received",
		Message: "new ride request received",
		RideID:  rideID,
		Additional: map[string]any{
			"vehicle_type": vehicleType,
			"pickup_lat":   pickupLat,
			"pickup_lng":   pickupLng,
		},
	})

	// Находим ближайших доступных водителей (PostGIS, радиус 5 км)
	drivers, err := c.driverRepo.FindNearbyAvailableDrivers(
		ctx,
		pickupLat,
		pickupLng,
		5.0, // radiusKm
		out.VehicleType(vehicleType),
		10, // limit
	)
	if err != nil {
		c.log.Error(logger.Entry{
			Action:  "ride_request_find_drivers_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
			RideID:  rideID,
		})
		_ = msg.Nack(false, true) // requeue
		return
	}

	if len(drivers) == 0 {
		c.log.Warn(logger.Entry{
			Action:  "ride_request_no_drivers_available",
			Message: "no available drivers found nearby",
			RideID:  rideID,
		})
		_ = msg.Ack(false)
		return
	}

	c.log.Info(logger.Entry{
		Action:  "ride_request_drivers_found",
		Message: fmt.Sprintf("found %d available drivers", len(drivers)),
		RideID:  rideID,
		Additional: map[string]any{
			"count": len(drivers),
		},
	})

	// Отправляем ride offers водителям (первым N подключенным через WebSocket)
	offersSent := 0
	for _, dwd := range drivers {
		if !c.driverNotifier.IsDriverConnected(dwd.Driver.ID) {
			c.log.Debug(logger.Entry{
				Action:  "driver_not_connected",
				Message: "driver not connected to websocket",
				Additional: map[string]any{
					"driver_id": dwd.Driver.ID,
				},
			})
			continue
		}

		offerID := uuid.New().String()
		expiresAt := time.Now().UTC().Add(30 * time.Second)

		offer := out.RideOffer{
			OfferID:                      offerID,
			RideID:                       rideID,
			RideNumber:                   rideRequest["ride_number"].(string),
			PickupLatitude:               pickupLat,
			PickupLongitude:              pickupLng,
			PickupAddress:                pickupAddr,
			DestinationLatitude:          destLat,
			DestinationLongitude:         destLng,
			DestinationAddress:           destAddr,
			EstimatedFare:                estimatedFare,
			DriverEarnings:               estimatedFare * 0.8, // 80% для водителя
			DistanceToPickupKm:           dwd.DistanceKm,
			EstimatedRideDurationMinutes: 15, // TODO: рассчитывать динамически
			ExpiresAt:                    expiresAt.Format(time.RFC3339),
		}

		if err := c.driverNotifier.SendRideOffer(ctx, dwd.Driver.ID, offer); err != nil {
			c.log.Error(logger.Entry{
				Action:  "send_ride_offer_failed",
				Message: err.Error(),
				Error:   &logger.ErrObj{Msg: err.Error()},
				RideID:  rideID,
				Additional: map[string]any{
					"driver_id": dwd.Driver.ID,
				},
			})
			continue
		}

		offersSent++

		// Отправляем offer только первым 3 водителям (упрощенно)
		if offersSent >= 3 {
			break
		}
	}

	c.log.Info(logger.Entry{
		Action:  "ride_offers_sent",
		Message: fmt.Sprintf("sent %d ride offers", offersSent),
		RideID:  rideID,
		Additional: map[string]any{
			"offers_sent": offersSent,
		},
	})

	_ = msg.Ack(false)
}
