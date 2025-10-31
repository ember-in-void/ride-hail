package out

import (
	"context"

	"ridehail/internal/admin/application/ports/in"
	"ridehail/internal/admin/domain"
)

// UserRepository — интерфейс репозитория для работы с пользователями
type UserRepository interface {
	// Create создает нового пользователя
	Create(ctx context.Context, user *domain.User) error

	// FindByID находит пользователя по ID
	FindByID(ctx context.Context, userID string) (*domain.User, error)

	// FindByEmail находит пользователя по email
	FindByEmail(ctx context.Context, email string) (*domain.User, error)

	// List возвращает список пользователей с фильтрами
	List(ctx context.Context, filters ListUsersFilters) ([]*domain.User, int, error)

	// Update обновляет пользователя
	Update(ctx context.Context, user *domain.User) error

	// Delete удаляет пользователя (soft delete через status=BANNED или hard delete)
	Delete(ctx context.Context, userID string) error

	// GetSystemMetrics получает метрики системы для admin dashboard
	GetSystemMetrics(ctx context.Context) (*in.SystemMetrics, error)

	// GetDriverDistribution получает распределение водителей по типам транспорта
	GetDriverDistribution(ctx context.Context) (map[string]int, error)

	// GetHotspots получает горячие точки (зоны повышенного спроса)
	GetHotspots(ctx context.Context) ([]in.Hotspot, error)

	// GetActiveRides получает активные поездки с пагинацией
	GetActiveRides(ctx context.Context, page, pageSize int) ([]in.ActiveRideInfo, int, error)
}

// ListUsersFilters — фильтры для списка пользователей
type ListUsersFilters struct {
	Role   string
	Status string
	Limit  int
	Offset int
}
