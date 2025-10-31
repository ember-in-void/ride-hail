package in

import (
	"context"
)

// CreateUserInput — входные данные для создания пользователя
type CreateUserInput struct {
	Email    string
	Password string // plain text, будет захеширован
	Role     string // PASSENGER | DRIVER
	Status   string // по умолчанию ACTIVE
	Attrs    map[string]interface{}
}

// CreateUserOutput — результат создания пользователя
type CreateUserOutput struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"` // ISO8601
}

// CreateUserUseCase — интерфейс use case создания пользователя
type CreateUserUseCase interface {
	Execute(ctx context.Context, input CreateUserInput) (*CreateUserOutput, error)
}
