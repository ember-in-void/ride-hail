package out

import (
	"context"

	"ridehail/internal/driver/domain"
)

// DriverRepository — репозиторий для работы с водителями
type DriverRepository interface {
	// FindByID находит водителя по ID
	FindByID(ctx context.Context, driverID string) (*domain.Driver, error)

	// UpdateStatus обновляет статус водителя
	UpdateStatus(ctx context.Context, driverID string, status domain.DriverStatus) error

	// UpdateRideStats обновляет статистику поездок (total_rides, total_earnings)
	UpdateRideStats(ctx context.Context, driverID string, ridesIncrement int, earningsIncrement float64) error

	// FindNearbyAvailableDrivers находит доступных водителей в радиусе
	FindNearbyAvailableDrivers(ctx context.Context, lat, lng float64, radiusKm float64, vehicleType domain.VehicleType, limit int) ([]*DriverWithDistance, error)
}

// DriverWithDistance — водитель с расстоянием до точки
type DriverWithDistance struct {
	Driver     *domain.Driver
	DistanceKm float64
	Latitude   float64
	Longitude  float64
}
