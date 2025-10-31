package user

import (
	"context"
	"errors"

	"ridehail/internal/admin/application/ports/in"
)

var (
	// ErrUserNotFound пользователь не найден
	ErrUserNotFound = errors.New("user not found")

	// ErrUserInactive пользователь неактивен
	ErrUserInactive = errors.New("user is inactive")

	// ErrUserBanned пользователь забанен
	ErrUserBanned = errors.New("user is banned")
)

// Repository — интерфейс для проверки пользователей (используется в ride/driver)
type Repository interface {
	// FindByID находит пользователя по ID
	// Возвращает ErrUserNotFound если не найден
	FindByID(ctx context.Context, userID string) (*User, error)

	// Exists проверяет существование пользователя
	Exists(ctx context.Context, userID string) (bool, error)

	// GetSystemMetrics получает метрики системы для admin dashboard
	GetSystemMetrics(ctx context.Context) (*in.SystemMetrics, error)

	// GetDriverDistribution получает распределение водителей по типам транспорта
	GetDriverDistribution(ctx context.Context) (map[string]int, error)

	// GetHotspots получает горячие точки (зоны повышенного спроса)
	GetHotspots(ctx context.Context) ([]in.Hotspot, error)

	// GetActiveRides получает активные поездки с пагинацией
	GetActiveRides(ctx context.Context, page, pageSize int) ([]in.ActiveRideInfo, int, error)
}
