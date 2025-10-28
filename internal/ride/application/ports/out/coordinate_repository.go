package out

import (
	"context"

	"ridehail/internal/ride/domain"
)

// CoordinateRepository — интерфейс репозитория для работы с координатами
type CoordinateRepository interface {
	// Create создает новую координату
	Create(ctx context.Context, coord *domain.Coordinate) error

	// FindByID возвращает координату по ID
	FindByID(ctx context.Context, coordID string) (*domain.Coordinate, error)

	// FindCurrentByEntity возвращает текущую координату для entity
	FindCurrentByEntity(ctx context.Context, entityID, entityType string) (*domain.Coordinate, error)

	// MarkAsNotCurrent помечает координаты entity как не текущие
	MarkAsNotCurrent(ctx context.Context, entityID, entityType string) error
}
