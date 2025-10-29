package user

import (
	"context"
	"errors"
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
}
