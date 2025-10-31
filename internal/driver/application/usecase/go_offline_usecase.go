package usecase

import (
	"context"
	"fmt"
	"time"

	in "ridehail/internal/driver/application/ports/in"
	out "ridehail/internal/driver/application/ports/out"
	"ridehail/internal/driver/domain"
	"ridehail/internal/shared/logger"
)

type goOfflineUseCase struct {
	driverRepo  out.DriverRepository
	sessionRepo out.SessionRepository
	eventPub    out.EventPublisher
	log         *logger.Logger
}

func NewGoOfflineUseCase(
	driverRepo out.DriverRepository,
	sessionRepo out.SessionRepository,
	eventPub out.EventPublisher,
	log *logger.Logger,
) in.GoOfflineUseCase {
	return &goOfflineUseCase{
		driverRepo:  driverRepo,
		sessionRepo: sessionRepo,
		eventPub:    eventPub,
		log:         log,
	}
}

func (uc *goOfflineUseCase) Execute(ctx context.Context, input in.GoOfflineInput) (*in.GoOfflineOutput, error) {
	// Находим водителя
	driver, err := uc.driverRepo.FindByID(ctx, input.DriverID)
	if err != nil {
		uc.log.Error(logger.Entry{
			Action:  "go_offline_driver_not_found",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
			Additional: map[string]any{
				"driver_id": input.DriverID,
			},
		})
		return nil, err
	}

	// Проверяем, может ли водитель перейти в офлайн
	if !driver.CanGoOffline() {
		uc.log.Warn(logger.Entry{
			Action:  "go_offline_cannot_go_offline",
			Message: "driver cannot go offline",
			Additional: map[string]any{
				"driver_id": input.DriverID,
				"status":    driver.Status,
			},
		})
		return nil, domain.ErrDriverBusy
	}

	// Находим активную сессию
	session, err := uc.sessionRepo.FindActiveByDriverID(ctx, input.DriverID)
	if err != nil {
		uc.log.Error(logger.Entry{
			Action:  "go_offline_session_not_found",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
			Additional: map[string]any{
				"driver_id": input.DriverID,
			},
		})
		return nil, domain.ErrSessionNotFound
	}

	// Закрываем сессию
	closedSession, err := uc.sessionRepo.Close(ctx, session.ID)
	if err != nil {
		uc.log.Error(logger.Entry{
			Action:  "go_offline_close_session_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
			Additional: map[string]any{
				"driver_id":  input.DriverID,
				"session_id": session.ID,
			},
		})
		return nil, fmt.Errorf("close session: %w", err)
	}

	// Обновляем статус водителя на OFFLINE
	if err := uc.driverRepo.UpdateStatus(ctx, input.DriverID, domain.DriverStatusOffline); err != nil {
		uc.log.Error(logger.Entry{
			Action:  "go_offline_update_status_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
			Additional: map[string]any{
				"driver_id": input.DriverID,
			},
		})
		return nil, fmt.Errorf("update status: %w", err)
	}

	// Публикуем событие изменения статуса
	if err := uc.eventPub.PublishDriverStatusChanged(ctx, input.DriverID, string(domain.DriverStatusOffline)); err != nil {
		uc.log.Error(logger.Entry{
			Action:  "go_offline_publish_event_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		// Не фатальная ошибка
	}

	// Рассчитываем summary
	duration := time.Duration(0)
	if closedSession.EndedAt != nil {
		duration = closedSession.EndedAt.Sub(closedSession.StartedAt)
	}

	uc.log.Info(logger.Entry{
		Action:  "driver_went_offline",
		Message: "driver is now offline",
		Additional: map[string]any{
			"driver_id":       input.DriverID,
			"session_id":      closedSession.ID,
			"duration_hours":  duration.Hours(),
			"rides_completed": closedSession.TotalRides,
			"earnings":        closedSession.TotalEarnings,
		},
	})

	return &in.GoOfflineOutput{
		SessionID: closedSession.ID,
		Status:    string(domain.DriverStatusOffline),
		SessionSummary: in.SessionSummary{
			DurationHours:  duration.Hours(),
			RidesCompleted: closedSession.TotalRides,
			Earnings:       closedSession.TotalEarnings,
		},
		Message: "You are now offline",
	}, nil
}
