package usecase

import (
	"context"

	"ridehail/internal/admin/application/ports/in"
	"ridehail/internal/admin/application/ports/out"
	"ridehail/internal/shared/logger"
)

// ListUsersService реализует ListUsersUseCase
type ListUsersService struct {
	userRepo out.UserRepository
	log      *logger.Logger
}

// NewListUsersService создает новый сервис получения списка пользователей
func NewListUsersService(userRepo out.UserRepository, log *logger.Logger) *ListUsersService {
	return &ListUsersService{
		userRepo: userRepo,
		log:      log,
	}
}

// Execute получает список пользователей с фильтрами
func (s *ListUsersService) Execute(ctx context.Context, input in.ListUsersInput) (*in.ListUsersOutput, error) {
	// Устанавливаем лимит по умолчанию
	limit := input.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	// Формируем фильтры
	filters := out.ListUsersFilters{
		Role:   input.Role,
		Status: input.Status,
		Limit:  limit,
		Offset: input.Offset,
	}

	// Получаем пользователей из репозитория
	users, totalCount, err := s.userRepo.List(ctx, filters)
	if err != nil {
		s.log.Error(logger.Entry{
			Action:  "list_users_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		return nil, err
	}

	// Маппим доменные модели в DTO
	userDTOs := make([]in.UserDTO, 0, len(users))
	for _, user := range users {
		userDTOs = append(userDTOs, in.UserDTO{
			UserID:    user.ID,
			Email:     user.Email,
			Role:      user.Role,
			Status:    user.Status,
			Attrs:     user.Attrs,
			CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt: user.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	return &in.ListUsersOutput{
		Users:      userDTOs,
		TotalCount: totalCount,
		Limit:      limit,
		Offset:     input.Offset,
	}, nil
}
