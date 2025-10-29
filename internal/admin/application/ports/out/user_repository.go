package out

import (
	"context"

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
}

// ListUsersFilters — фильтры для списка пользователей
type ListUsersFilters struct {
	Role   string
	Status string
	Limit  int
	Offset int
}
