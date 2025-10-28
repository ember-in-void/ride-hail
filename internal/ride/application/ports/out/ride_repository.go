package out

import (
	"context"

	"ridehail/internal/ride/domain"
)

// RideRepository — интерфейс репозитория для работы с поездками
type RideRepository interface {
	// Create создает новую поездку
	Create(ctx context.Context, ride *domain.Ride) error

	// FindByID возвращает поездку по ID
	FindByID(ctx context.Context, rideID string) (*domain.Ride, error)

	// FindByRideNumber возвращает поездку по номеру
	FindByRideNumber(ctx context.Context, rideNumber string) (*domain.Ride, error)

	// Update обновляет существующую поездку
	Update(ctx context.Context, ride *domain.Ride) error

	// FindActiveByPassengerID возвращает активные поездки пассажира
	FindActiveByPassengerID(ctx context.Context, passengerID string) ([]*domain.Ride, error)

	// FindByStatus возвращает поездки с определенным статусом
	FindByStatus(ctx context.Context, status string, limit int) ([]*domain.Ride, error)
}
