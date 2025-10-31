package usecase

import (
	"context"

	"ridehail/internal/admin/application/ports/in"
	"ridehail/internal/admin/application/ports/out"
	"ridehail/internal/shared/logger"
)

// GetActiveRidesService реализует GetActiveRidesUseCase
type GetActiveRidesService struct {
	userRepo out.UserRepository
	log      *logger.Logger
}

// NewGetActiveRidesService создает новый сервис получения активных поездок
func NewGetActiveRidesService(userRepo out.UserRepository, log *logger.Logger) *GetActiveRidesService {
	return &GetActiveRidesService{
		userRepo: userRepo,
		log:      log,
	}
}

// Execute выполняет получение активных поездок
func (s *GetActiveRidesService) Execute(ctx context.Context, input in.GetActiveRidesInput) (*in.GetActiveRidesOutput, error) {
	s.log.Info(logger.Entry{
		Action:  "get_active_rides_started",
		Message: "fetching active rides",
	})

	// Валидация пагинации
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 100 {
		input.PageSize = 20
	}

	// Получаем активные поездки из базы данных
	rides, totalCount, err := s.userRepo.GetActiveRides(ctx, input.Page, input.PageSize)
	if err != nil {
		s.log.Error(logger.Entry{
			Action:  "get_active_rides_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		return nil, err
	}

	output := &in.GetActiveRidesOutput{
		Rides:      rides,
		TotalCount: totalCount,
		Page:       input.Page,
		PageSize:   input.PageSize,
	}

	s.log.Info(logger.Entry{
		Action:  "get_active_rides_completed",
		Message: "active rides fetched successfully",
	})

	return output, nil
}
