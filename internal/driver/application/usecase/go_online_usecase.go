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

type goOnlineUseCase struct {
	driverRepo   out.DriverRepository
	sessionRepo  out.SessionRepository
	locationRepo out.LocationRepository
	eventPub     out.EventPublisher
	log          *logger.Logger
}

func NewGoOnlineUseCase(
	driverRepo out.DriverRepository,
	sessionRepo out.SessionRepository,
	locationRepo out.LocationRepository,
	eventPub out.EventPublisher,
	log *logger.Logger,
) in.GoOnlineUseCase {
	return &goOnlineUseCase{
		driverRepo:   driverRepo,
		sessionRepo:  sessionRepo,
		locationRepo: locationRepo,
		eventPub:     eventPub,
		log:          log,
	}
}

func (uc *goOnlineUseCase) Execute(ctx context.Context, input in.GoOnlineInput) (*in.GoOnlineOutput, error) {
	// Валидация координат
	if input.Latitude < -90 || input.Latitude > 90 || input.Longitude < -180 || input.Longitude > 180 {
		return nil, domain.ErrInvalidCoordinates
	}

	// Находим водителя
	driver, err := uc.driverRepo.FindByID(ctx, input.DriverID)
	if err != nil {
		uc.log.Error(logger.Entry{
			Action:  "go_online_driver_not_found",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
			Additional: map[string]any{
				"driver_id": input.DriverID,
			},
		})
		return nil, err
	}

	// Проверяем, может ли водитель перейти в онлайн
	if !driver.CanGoOnline() {
		uc.log.Warn(logger.Entry{
			Action:  "go_online_cannot_go_online",
			Message: "driver cannot go online",
			Additional: map[string]any{
				"driver_id": input.DriverID,
				"status":    driver.Status,
				"verified":  driver.IsVerified,
			},
		})
		return nil, domain.ErrDriverNotAvailable
	}

	// Проверяем, нет ли уже активной сессии
	existingSession, _ := uc.sessionRepo.FindActiveByDriverID(ctx, input.DriverID)
	if existingSession != nil {
		uc.log.Warn(logger.Entry{
			Action:  "go_online_session_already_active",
			Message: "driver already has active session",
			Additional: map[string]any{
				"driver_id":  input.DriverID,
				"session_id": existingSession.ID,
			},
		})
		return nil, domain.ErrSessionAlreadyActive
	}

	// Создаем новую сессию
	session := &domain.DriverSession{
		DriverID:      input.DriverID,
		StartedAt:     time.Now().UTC(),
		TotalRides:    0,
		TotalEarnings: 0,
	}

	if err := uc.sessionRepo.Create(ctx, session); err != nil {
		uc.log.Error(logger.Entry{
			Action:  "go_online_create_session_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
			Additional: map[string]any{
				"driver_id": input.DriverID,
			},
		})
		return nil, fmt.Errorf("create session: %w", err)
	}

	// Обновляем статус водителя на AVAILABLE
	if err := uc.driverRepo.UpdateStatus(ctx, input.DriverID, domain.DriverStatusAvailable); err != nil {
		uc.log.Error(logger.Entry{
			Action:  "go_online_update_status_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
			Additional: map[string]any{
				"driver_id": input.DriverID,
			},
		})
		return nil, fmt.Errorf("update status: %w", err)
	}

	// Сохраняем начальную локацию
	coord := &domain.Coordinates{
		EntityID:   input.DriverID,
		EntityType: "driver",
		Address:    "",
		Latitude:   input.Latitude,
		Longitude:  input.Longitude,
		IsCurrent:  true,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}

	if err := uc.locationRepo.SaveCoordinate(ctx, coord); err != nil {
		uc.log.Error(logger.Entry{
			Action:  "go_online_save_location_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		// Не фатальная ошибка, продолжаем
	}

	// Публикуем событие изменения статуса
	if err := uc.eventPub.PublishDriverStatusChanged(ctx, input.DriverID, string(domain.DriverStatusAvailable)); err != nil {
		uc.log.Error(logger.Entry{
			Action:  "go_online_publish_event_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		// Не фатальная ошибка
	}

	uc.log.Info(logger.Entry{
		Action:  "driver_went_online",
		Message: "driver is now available",
		Additional: map[string]any{
			"driver_id":  input.DriverID,
			"session_id": session.ID,
		},
	})

	return &in.GoOnlineOutput{
		SessionID: session.ID,
		Status:    string(domain.DriverStatusAvailable),
		Message:   "You are now online and ready to accept rides",
	}, nil
}
