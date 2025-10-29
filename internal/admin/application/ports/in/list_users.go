package in

import (
	"context"
)

// ListUsersInput — входные данные для получения списка пользователей
type ListUsersInput struct {
	Role   string // фильтр по роли (опционально)
	Status string // фильтр по статусу (опционально)
	Limit  int    // по умолчанию 50
	Offset int    // для пагинации
}

// UserDTO — DTO пользователя для списка
type UserDTO struct {
	UserID    string                 `json:"user_id"`
	Email     string                 `json:"email"`
	Role      string                 `json:"role"`
	Status    string                 `json:"status"`
	Attrs     map[string]interface{} `json:"attrs,omitempty"`
	CreatedAt string                 `json:"created_at"`
	UpdatedAt string                 `json:"updated_at"`
}

// ListUsersOutput — результат получения списка
type ListUsersOutput struct {
	Users      []UserDTO `json:"users"`
	TotalCount int       `json:"total_count"`
	Limit      int       `json:"limit"`
	Offset     int       `json:"offset"`
}

// ListUsersUseCase — интерфейс use case получения списка пользователей
type ListUsersUseCase interface {
	Execute(ctx context.Context, input ListUsersInput) (*ListUsersOutput, error)
}
