package user

import "time"

// User — упрощённая модель пользователя для проверки в ride/driver сервисах
// Полная модель находится в internal/admin/domain/user.go
type User struct {
	ID        string
	Email     string
	Role      string // PASSENGER | DRIVER | ADMIN
	Status    string // ACTIVE | INACTIVE | BANNED
	CreatedAt time.Time
	UpdatedAt time.Time
}

// IsActive проверяет, активен ли пользователь
func (u *User) IsActive() bool {
	return u.Status == "ACTIVE"
}

// HasRole проверяет наличие роли
func (u *User) HasRole(role string) bool {
	return u.Role == role
}
