package domain

import (
	"time"
)

// User представляет пользователя системы
type User struct {
	ID           string
	Email        string
	Role         string // PASSENGER | DRIVER | ADMIN
	Status       string // ACTIVE | INACTIVE | BANNED
	PasswordHash string
	Attrs        map[string]interface{} // дополнительные атрибуты (JSONB)
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// UserRole — допустимые роли
const (
	RolePassenger = "PASSENGER"
	RoleDriver    = "DRIVER"
	RoleAdmin     = "ADMIN"
)

// UserStatus — допустимые статусы
const (
	StatusActive   = "ACTIVE"
	StatusInactive = "INACTIVE"
	StatusBanned   = "BANNED"
)

// IsValidRole проверяет корректность роли
func IsValidRole(role string) bool {
	switch role {
	case RolePassenger, RoleDriver, RoleAdmin:
		return true
	default:
		return false
	}
}

// IsValidStatus проверяет корректность статуса
func IsValidStatus(status string) bool {
	switch status {
	case StatusActive, StatusInactive, StatusBanned:
		return true
	default:
		return false
	}
}
