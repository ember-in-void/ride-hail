package out

import (
	"context"

	"ridehail/internal/driver/domain"
)

// SessionRepository — репозиторий для работы с сессиями водителей
type SessionRepository interface {
	// Create создает новую сессию
	Create(ctx context.Context, session *domain.DriverSession) error

	// FindActiveByDriverID находит активную сессию водителя
	FindActiveByDriverID(ctx context.Context, driverID string) (*domain.DriverSession, error)

	// Close закрывает сессию (устанавливает ended_at)
	Close(ctx context.Context, sessionID string) (*domain.DriverSession, error)

	// UpdateStats обновляет статистику сессии (rides, earnings)
	UpdateStats(ctx context.Context, sessionID string, ridesIncrement int, earningsIncrement float64) error
}
