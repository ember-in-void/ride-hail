package usecase

import (
	"context"
	"time"

	"ridehail/internal/admin/application/ports/in"
	"ridehail/internal/admin/application/ports/out"
	"ridehail/internal/shared/logger"
)

// GetOverviewService реализует GetOverviewUseCase
type GetOverviewService struct {
	userRepo out.UserRepository
	log      *logger.Logger
}

// NewGetOverviewService создает новый сервис обзора системы
func NewGetOverviewService(userRepo out.UserRepository, log *logger.Logger) *GetOverviewService {
	return &GetOverviewService{
		userRepo: userRepo,
		log:      log,
	}
}

// Execute выполняет получение обзора системы
func (s *GetOverviewService) Execute(ctx context.Context, input in.GetOverviewInput) (*in.GetOverviewOutput, error) {
	s.log.Info(logger.Entry{
		Action:  "get_overview_started",
		Message: "fetching system overview",
	})

	// Получаем метрики из базы данных
	metrics, err := s.userRepo.GetSystemMetrics(ctx)
	if err != nil {
		s.log.Error(logger.Entry{
			Action:  "get_system_metrics_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		return nil, err
	}

	// Получаем распределение водителей по типам
	driverDistribution, err := s.userRepo.GetDriverDistribution(ctx)
	if err != nil {
		s.log.Error(logger.Entry{
			Action:  "get_driver_distribution_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		return nil, err
	}

	// Получаем горячие точки (hotspots) - опционально
	hotspots, err := s.userRepo.GetHotspots(ctx)
	if err != nil {
		s.log.Warn(logger.Entry{
			Action:  "get_hotspots_failed",
			Message: err.Error(),
		})
		// Не критично, продолжаем без hotspots
		hotspots = []in.Hotspot{}
	}

	output := &in.GetOverviewOutput{
		Timestamp:          time.Now().UTC().Format(time.RFC3339),
		Metrics:            *metrics,
		DriverDistribution: driverDistribution,
		Hotspots:           hotspots,
	}

	s.log.Info(logger.Entry{
		Action:  "get_overview_completed",
		Message: "system overview fetched successfully",
	})

	return output, nil
}
