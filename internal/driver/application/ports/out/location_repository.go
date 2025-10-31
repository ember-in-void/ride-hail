package out

import (
	"context"

	"ridehail/internal/driver/domain"
)

// LocationRepository — репозиторий для работы с локациями
type LocationRepository interface {
	// SaveCoordinate сохраняет координату в coordinates table
	SaveCoordinate(ctx context.Context, coord *domain.Coordinates) error

	// GetCurrentDriverLocation получает текущую локацию водителя
	GetCurrentDriverLocation(ctx context.Context, driverID string) (*domain.Coordinates, error)

	// SaveLocationHistory сохраняет в location_history
	SaveLocationHistory(ctx context.Context, driverID string, lat, lng float64, accuracy, speed, heading float64, rideID *string) error

	// GetLastLocationUpdateTime получает время последнего обновления локации (для rate-limit)
	GetLastLocationUpdateTime(ctx context.Context, driverID string) (*string, error)
}
