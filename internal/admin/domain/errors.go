package domain

import "errors"

var (
	// ErrUserAlreadyExists пользователь с таким email уже существует
	ErrUserAlreadyExists = errors.New("user with this email already exists")

	// ErrUserNotFound пользователь не найден
	ErrUserNotFound = errors.New("user not found")

	// ErrInvalidEmail некорректный формат email
	ErrInvalidEmail = errors.New("invalid email format")

	// ErrInvalidRole некорректная роль
	ErrInvalidRole = errors.New("invalid role")

	// ErrInvalidStatus некорректный статус
	ErrInvalidStatus = errors.New("invalid status")

	// ErrPasswordTooShort пароль слишком короткий
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")

	// ErrUnauthorized недостаточно прав
	ErrUnauthorized = errors.New("unauthorized")
)
