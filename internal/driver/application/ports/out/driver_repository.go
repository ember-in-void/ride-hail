package out

import (
	"context"

	"ridehail/internal/driver/domain"
)

// DriverRepository определяет операции с водителями в БД
type DriverRepository interface {
	// FindByID находит водителя по ID
	FindByID(ctx context.Context, driverID string) (*domain.Driver, error)

	// FindNearbyAvailable находит доступных водителей рядом с точкой
	FindNearbyAvailable(ctx context.Context, lat, lng float64, vehicleType domain.VehicleType, maxDistanceKm float64) ([]*NearbyDriverDTO, error)

	// UpdateStatus обновляет статус водителя
	UpdateStatus(ctx context.Context, driverID string, status domain.DriverStatus) error

	// CreateSession создает новую сессию водителя
	CreateSession(ctx context.Context, session *domain.DriverSession) error

	// EndSession завершает текущую сессию водителя
	EndSession(ctx context.Context, sessionID string, totalRides int, totalEarnings float64) error

	// GetActiveSession получает активную сессию водителя
	GetActiveSession(ctx context.Context, driverID string) (*domain.DriverSession, error)

	// UpdateRideStats обновляет статистику поездок водителя
	UpdateRideStats(ctx context.Context, driverID string, ridesIncrement int, earningsIncrement float64) error
}

// NearbyDriverDTO — DTO для результатов поиска водителей поблизости
type NearbyDriverDTO struct {
	DriverID    string              `json:"driver_id"`
	Email       string              `json:"email"`
	Rating      float64             `json:"rating"`
	Latitude    float64             `json:"latitude"`
	Longitude   float64             `json:"longitude"`
	DistanceKm  float64             `json:"distance_km"`
	VehicleType domain.VehicleType  `json:"vehicle_type"`
	Vehicle     domain.VehicleAttrs `json:"vehicle"`
}
