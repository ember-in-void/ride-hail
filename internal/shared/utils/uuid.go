package utils

import "github.com/google/uuid"

// NewUUID генерирует новый UUID v4
func NewUUID() string {
	return uuid.New().String()
}
