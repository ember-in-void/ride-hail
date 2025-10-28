package usecase

import (
	"context"
	"fmt"
	"math"
	"time"

	"ridehail/internal/ride/application/ports/in"
	"ridehail/internal/ride/application/ports/out"
	"ridehail/internal/ride/domain"
	constants "ridehail/internal/shared/const"
	"ridehail/internal/shared/logger"

	"github.com/google/uuid"
)

// RequestRideService реализует RequestRideUseCase
type RequestRideService struct {
	rideRepo  out.RideRepository
	coordRepo out.CoordinateRepository
	publisher out.EventPublisher
	notifier  out.RideNotifier
	log       *logger.Logger
}

// NewRequestRideService создает новый сервис для запроса поездки
func NewRequestRideService(
	rideRepo out.RideRepository,
	coordRepo out.CoordinateRepository,
	publisher out.EventPublisher,
	notifier out.RideNotifier,
	log *logger.Logger,
) *RequestRideService {
	return &RequestRideService{
		rideRepo:  rideRepo,
		coordRepo: coordRepo,
		publisher: publisher,
		notifier:  notifier,
		log:       log,
	}
}

// Execute выполняет создание новой поездки
func (s *RequestRideService) Execute(ctx context.Context, input in.RequestRideInput) (*in.RequestRideOutput, error) {
	// Валидация координат
	if err := domain.ValidateCoordinates(input.PickupLat, input.PickupLng); err != nil {
		return nil, err
	}
	if err := domain.ValidateCoordinates(input.DestLat, input.DestLng); err != nil {
		return nil, err
	}

	// Валидация типа автомобиля
	if !isValidVehicleType(input.VehicleType) {
		return nil, domain.ErrInvalidVehicleType
	}

	// Валидация приоритета
	priority := input.Priority
	if priority < 1 || priority > 10 {
		priority = 1
	}

	// Создаем координаты pickup
	pickupCoord := &domain.Coordinate{
		ID:         uuid.New().String(),
		EntityID:   input.PassengerID,
		EntityType: "passenger",
		Address:    input.PickupAddress,
		Latitude:   input.PickupLat,
		Longitude:  input.PickupLng,
		IsCurrent:  true,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}

	if err := s.coordRepo.Create(ctx, pickupCoord); err != nil {
		s.log.Error(logger.Entry{
			Action:  "create_pickup_coordinate_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		return nil, fmt.Errorf("create pickup coordinate: %w", err)
	}

	// Создаем координаты destination
	distance := calculateDistance(input.PickupLat, input.PickupLng, input.DestLat, input.DestLng)
	estimatedFare := calculateFare(distance, input.VehicleType)
	estimatedDuration := calculateDuration(distance)

	destCoord := &domain.Coordinate{
		ID:              uuid.New().String(),
		EntityID:        input.PassengerID,
		EntityType:      "passenger",
		Address:         input.DestAddress,
		Latitude:        input.DestLat,
		Longitude:       input.DestLng,
		FareAmount:      &estimatedFare,
		DistanceKm:      &distance,
		DurationMinutes: &estimatedDuration,
		IsCurrent:       false,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}

	if err := s.coordRepo.Create(ctx, destCoord); err != nil {
		s.log.Error(logger.Entry{
			Action:  "create_destination_coordinate_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		return nil, fmt.Errorf("create destination coordinate: %w", err)
	}

	// Генерируем уникальный номер поездки
	rideNumber := generateRideNumber()

	// Создаем поездку
	now := time.Now().UTC()
	ride := &domain.Ride{
		ID:                      uuid.New().String(),
		RideNumber:              rideNumber,
		PassengerID:             input.PassengerID,
		DriverID:                nil,
		VehicleType:             input.VehicleType,
		Status:                  constants.RideStatusRequested,
		Priority:                priority,
		RequestedAt:             now,
		EstimatedFare:           &estimatedFare,
		PickupCoordinateID:      pickupCoord.ID,
		DestinationCoordinateID: destCoord.ID,
		CreatedAt:               now,
		UpdatedAt:               now,
	}

	if err := s.rideRepo.Create(ctx, ride); err != nil {
		s.log.Error(logger.Entry{
			Action:  "create_ride_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
			Additional: map[string]any{
				"ride_number":  rideNumber,
				"passenger_id": input.PassengerID,
			},
		})
		return nil, fmt.Errorf("create ride: %w", err)
	}

	s.log.Info(logger.Entry{
		Action:  "ride_created",
		Message: rideNumber,
		RideID:  ride.ID,
		Additional: map[string]any{
			"passenger_id":   input.PassengerID,
			"vehicle_type":   input.VehicleType,
			"estimated_fare": estimatedFare,
			"distance_km":    distance,
		},
	})

	// Публикуем событие в RabbitMQ
	eventData := out.RideEventData{
		RideID:      ride.ID,
		PassengerID: input.PassengerID,
		Status:      constants.RideStatusRequested,
		VehicleType: input.VehicleType,
		AdditionalData: map[string]interface{}{
			"ride_number":    rideNumber,
			"estimated_fare": estimatedFare,
			"pickup_address": input.PickupAddress,
			"dest_address":   input.DestAddress,
			"distance_km":    distance,
		},
	}

	if err := s.publisher.PublishRideEvent(ctx, constants.EventRideRequested, eventData); err != nil {
		s.log.Error(logger.Entry{
			Action:  "publish_ride_event_failed",
			Message: err.Error(),
			RideID:  ride.ID,
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		// Не возвращаем ошибку, т.к. поездка уже создана
	}

	// Отправляем WebSocket уведомление пассажиру
	notification := out.RideNotification{
		Type:    "ride_requested",
		RideID:  ride.ID,
		Message: "Your ride has been requested",
		Data: map[string]interface{}{
			"ride_number":    rideNumber,
			"estimated_fare": estimatedFare,
			"status":         constants.RideStatusRequested,
		},
	}

	if err := s.notifier.NotifyPassenger(ctx, input.PassengerID, notification); err != nil {
		s.log.Error(logger.Entry{
			Action:  "notify_passenger_failed",
			Message: err.Error(),
			RideID:  ride.ID,
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		// Не возвращаем ошибку
	}

	// Формируем ответ
	return &in.RequestRideOutput{
		RideID:        ride.ID,
		RideNumber:    rideNumber,
		Status:        constants.RideStatusRequested,
		EstimatedFare: estimatedFare,
		PickupAddress: input.PickupAddress,
		DestAddress:   input.DestAddress,
	}, nil
}

// isValidVehicleType проверяет корректность типа автомобиля
func isValidVehicleType(vType string) bool {
	switch vType {
	case constants.VehicleEconomy, constants.VehiclePremium, constants.VehicleXL:
		return true
	default:
		return false
	}
}

// calculateDistance вычисляет расстояние между двумя точками (формула Haversine)
func calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371.0 // км

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}

// calculateFare вычисляет примерную стоимость поездки
func calculateFare(distanceKm float64, vehicleType string) float64 {
	baseFare := 50.0 // базовый тариф
	perKm := 15.0    // за км

	switch vehicleType {
	case constants.VehiclePremium:
		baseFare = 100.0
		perKm = 25.0
	case constants.VehicleXL:
		baseFare = 80.0
		perKm = 20.0
	}

	fare := baseFare + (distanceKm * perKm)
	return math.Round(fare*100) / 100 // округляем до 2 знаков
}

// calculateDuration вычисляет примерное время поездки (минуты)
func calculateDuration(distanceKm float64) int {
	avgSpeedKmh := 40.0 // средняя скорость в городе
	durationHours := distanceKm / avgSpeedKmh
	return int(math.Ceil(durationHours * 60))
}

// generateRideNumber генерирует уникальный номер поездки
func generateRideNumber() string {
	now := time.Now().UTC()
	return fmt.Sprintf("RIDE-%s-%d", now.Format("20060102"), now.UnixNano()%1000000)
}
