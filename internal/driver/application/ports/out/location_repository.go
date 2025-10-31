package out

import (
	"context"

	"ridehail/internal/driver/domain"
)

// LocationRepository определяет операции с координатами и историей локаций
type LocationRepository interface {
	// CreateCoordinate создает новую запись координат
	CreateCoordinate(ctx context.Context, coord *CreateCoordinateDTO) (string, error)

	// UpdateCurrentLocation обновляет текущую локацию (устанавливает is_current=false для старых)
	UpdateCurrentLocation(ctx context.Context, entityID, entityType string, lat, lng float64) (string, error)

	// GetCurrentLocation получает текущую локацию сущности
	GetCurrentLocation(ctx context.Context, entityID, entityType string) (*domain.DriverLocation, error)

	// ArchiveToHistory архивирует координаты в location_history
	ArchiveToHistory(ctx context.Context, history *LocationHistoryDTO) error

	// CheckRateLimit проверяет, можно ли обновить локацию (макс 1 раз в 3 сек)
	CheckRateLimit(ctx context.Context, driverID string) (bool, error)
}

// CreateCoordinateDTO — DTO для создания координат
type CreateCoordinateDTO struct {
	EntityID        string  `json:"entity_id"`
	EntityType      string  `json:"entity_type"` // "driver" или "passenger"
	Address         string  `json:"address"`
	Latitude        float64 `json:"latitude"`
	Longitude       float64 `json:"longitude"`
	FareAmount      float64 `json:"fare_amount,omitempty"`
	DistanceKm      float64 `json:"distance_km,omitempty"`
	DurationMinutes int     `json:"duration_minutes,omitempty"`
	IsCurrent       bool    `json:"is_current"`
}

// LocationHistoryDTO — DTO для архивации в location_history
type LocationHistoryDTO struct {
	CoordinateID   string  `json:"coordinate_id,omitempty"`
	DriverID       string  `json:"driver_id"`
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
	AccuracyMeters float64 `json:"accuracy_meters,omitempty"`
	SpeedKmh       float64 `json:"speed_kmh,omitempty"`
	HeadingDegrees float64 `json:"heading_degrees,omitempty"`
	RideID         *string `json:"ride_id,omitempty"`
}
