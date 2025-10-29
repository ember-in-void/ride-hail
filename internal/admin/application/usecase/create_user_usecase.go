package usecase

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"ridehail/internal/admin/application/ports/in"
	"ridehail/internal/admin/application/ports/out"
	"ridehail/internal/admin/domain"
	"ridehail/internal/shared/logger"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// CreateUserService реализует CreateUserUseCase
type CreateUserService struct {
	userRepo out.UserRepository
	log      *logger.Logger
}

// NewCreateUserService создает новый сервис создания пользователя
func NewCreateUserService(userRepo out.UserRepository, log *logger.Logger) *CreateUserService {
	return &CreateUserService{
		userRepo: userRepo,
		log:      log,
	}
}

// Execute создает нового пользователя
func (s *CreateUserService) Execute(ctx context.Context, input in.CreateUserInput) (*in.CreateUserOutput, error) {
	// Валидация email
	if !emailRegex.MatchString(input.Email) {
		return nil, domain.ErrInvalidEmail
	}

	// Проверка уникальности email
	existingUser, err := s.userRepo.FindByEmail(ctx, input.Email)
	if err == nil && existingUser != nil {
		return nil, domain.ErrUserAlreadyExists
	}

	// Валидация роли (админ может создавать только PASSENGER и DRIVER)
	if input.Role != domain.RolePassenger && input.Role != domain.RoleDriver {
		return nil, domain.ErrInvalidRole
	}

	// Валидация пароля
	if len(input.Password) < 8 {
		return nil, domain.ErrPasswordTooShort
	}

	// Хешируем пароль
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		s.log.Error(logger.Entry{
			Action:  "hash_password_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		return nil, fmt.Errorf("hash password: %w", err)
	}

	// Устанавливаем статус (по умолчанию ACTIVE)
	status := input.Status
	if status == "" {
		status = domain.StatusActive
	}

	if !domain.IsValidStatus(status) {
		return nil, domain.ErrInvalidStatus
	}

	// Создаем доменную модель
	now := time.Now().UTC()
	user := &domain.User{
		ID:           uuid.New().String(),
		Email:        input.Email,
		Role:         input.Role,
		Status:       status,
		PasswordHash: string(passwordHash),
		Attrs:        input.Attrs,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Сохраняем в БД
	if err := s.userRepo.Create(ctx, user); err != nil {
		s.log.Error(logger.Entry{
			Action:  "create_user_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
			Additional: map[string]interface{}{
				"email": input.Email,
				"role":  input.Role,
			},
		})
		return nil, fmt.Errorf("create user: %w", err)
	}

	s.log.Info(logger.Entry{
		Action:  "user_created",
		Message: fmt.Sprintf("user %s created", user.Email),
		Additional: map[string]interface{}{
			"user_id": user.ID,
			"email":   user.Email,
			"role":    user.Role,
		},
	})

	// Формируем ответ
	return &in.CreateUserOutput{
		UserID:    user.ID,
		Email:     user.Email,
		Role:      user.Role,
		Status:    user.Status,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	}, nil
}
