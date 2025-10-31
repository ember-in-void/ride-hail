package out

import (
	"context"

	"ridehail/internal/driver/domain"
)

// SessionRepository — репозиторий для работы с сессиями водителей
type SessionRepository interface {
	// Create — создать новую сессию
	Create(ctx context.Context, session *domain.DriverSession) error

	// FindActiveByDriverID — найти активную сессию водителя
	FindActiveByDriverID(ctx context.Context, driverID string) (*domain.DriverSession, error)

	// EndSession — завершить сессию
	EndSession(ctx context.Context, sessionID string) (*domain.DriverSession, error)

	// UpdateSessionStats — обновить статистику сессии
	UpdateSessionStats(ctx context.Context, sessionID string, additionalRides int, additionalEarnings float64) error
}
