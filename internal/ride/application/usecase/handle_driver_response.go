package usecase

import (
	"context"
	"fmt"

	"ridehail/internal/ride/application/ports/in"
	"ridehail/internal/ride/application/ports/out"
	"ridehail/internal/shared/logger"
)

// HandleDriverResponseService — use case для обработки ответа водителя
type HandleDriverResponseService struct {
	rideRepo out.RideRepository
	log      *logger.Logger
}

// NewHandleDriverResponseService создает новый сервис
func NewHandleDriverResponseService(
	rideRepo out.RideRepository,
	log *logger.Logger,
) *HandleDriverResponseService {
	return &HandleDriverResponseService{
		rideRepo: rideRepo,
		log:      log,
	}
}

// Execute обрабатывает ответ водителя на предложение поездки
func (s *HandleDriverResponseService) Execute(ctx context.Context, input in.HandleDriverResponseInput) (*in.HandleDriverResponseOutput, error) {
	s.log.Info(logger.Entry{
		Action:  "handle_driver_response",
		Message: fmt.Sprintf("ride=%s, driver=%s, accepted=%t", input.RideID, input.DriverID, input.Accepted),
		RideID:  input.RideID,
		Additional: map[string]any{
			"driver_id": input.DriverID,
			"accepted":  input.Accepted,
		},
	})

	if !input.Accepted {
		// Водитель отклонил поездку
		s.log.Info(logger.Entry{
			Action:  "driver_rejected_ride",
			Message: fmt.Sprintf("driver %s rejected ride %s", input.DriverID, input.RideID),
			RideID:  input.RideID,
		})

		// TODO: Попробовать найти другого водителя
		return &in.HandleDriverResponseOutput{
			RideID:         input.RideID,
			Status:         "REQUESTED",
			DriverAssigned: false,
		}, nil
	}

	// Получаем поездку из БД
	ride, err := s.rideRepo.FindByID(ctx, input.RideID)
	if err != nil {
		s.log.Error(logger.Entry{
			Action:  "find_ride_failed",
			Message: err.Error(),
			RideID:  input.RideID,
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		return nil, fmt.Errorf("find ride: %w", err)
	}

	// Проверяем, что поездка в статусе REQUESTED
	if ride.Status != "REQUESTED" {
		s.log.Warn(logger.Entry{
			Action:  "ride_not_in_requested_status",
			Message: fmt.Sprintf("ride %s has status %s, expected REQUESTED", input.RideID, ride.Status),
			RideID:  input.RideID,
		})
		return nil, fmt.Errorf("ride is not in REQUESTED status (current: %s)", ride.Status)
	}

	// Назначаем водителя
	if err := s.rideRepo.AssignDriver(ctx, input.RideID, input.DriverID); err != nil {
		s.log.Error(logger.Entry{
			Action:  "assign_driver_failed",
			Message: err.Error(),
			RideID:  input.RideID,
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		return nil, fmt.Errorf("assign driver: %w", err)
	}

	s.log.Info(logger.Entry{
		Action:  "driver_assigned_successfully",
		Message: fmt.Sprintf("driver %s assigned to ride %s", input.DriverID, input.RideID),
		RideID:  input.RideID,
		Additional: map[string]any{
			"driver_id": input.DriverID,
			"eta":       input.EstimatedArrivalMinutes,
		},
	})

	return &in.HandleDriverResponseOutput{
		RideID:         input.RideID,
		Status:         "DRIVER_ASSIGNED",
		DriverAssigned: true,
		PassengerID:    ride.PassengerID,
	}, nil
}
