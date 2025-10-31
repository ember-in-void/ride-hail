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

type startRideUseCase struct {
	driverRepo   out.DriverRepository
	rideRepo     out.RideRepository
	locationRepo out.LocationRepository
	log          *logger.Logger
}

func NewStartRideUseCase(
	driverRepo out.DriverRepository,
	rideRepo out.RideRepository,
	locationRepo out.LocationRepository,
	log *logger.Logger,
) in.StartRideUseCase {
	return &startRideUseCase{
		driverRepo:   driverRepo,
		rideRepo:     rideRepo,
		locationRepo: locationRepo,
		log:          log,
	}
}

func (uc *startRideUseCase) Execute(ctx context.Context, input in.StartRideInput) (*in.StartRideOutput, error) {
	// Находим водителя
	driver, err := uc.driverRepo.FindByID(ctx, input.DriverID)
	if err != nil {
		uc.log.Error(logger.Entry{
			Action:  "start_ride_driver_not_found",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
			Additional: map[string]any{
				"driver_id": input.DriverID,
				"ride_id":   input.RideID,
			},
		})
		return nil, err
	}

	// Проверяем статус водителя (должен быть EN_ROUTE или AVAILABLE)
	if driver.Status != domain.DriverStatusEnRoute && driver.Status != domain.DriverStatusAvailable {
		uc.log.Warn(logger.Entry{
			Action:  "start_ride_invalid_driver_status",
			Message: "driver cannot start ride with current status",
			Additional: map[string]any{
				"driver_id": input.DriverID,
				"ride_id":   input.RideID,
				"status":    driver.Status,
			},
		})
		return nil, domain.ErrDriverNotAvailable
	}

	// Находим поездку
	ride, err := uc.rideRepo.FindByID(ctx, input.RideID)
	if err != nil {
		uc.log.Error(logger.Entry{
			Action:  "start_ride_ride_not_found",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
			RideID:  input.RideID,
		})
		return nil, err
	}

	// Проверяем, что поездка назначена этому водителю
	if ride.DriverID == nil || *ride.DriverID != input.DriverID {
		uc.log.Warn(logger.Entry{
			Action:  "start_ride_driver_mismatch",
			Message: "ride assigned to different driver",
			RideID:  input.RideID,
			Additional: map[string]any{
				"driver_id":          input.DriverID,
				"assigned_driver_id": ride.DriverID,
			},
		})
		return nil, fmt.Errorf("ride assigned to different driver")
	}

	// Обновляем статус поездки на IN_PROGRESS
	if err := uc.rideRepo.UpdateRideStatus(ctx, input.RideID, "IN_PROGRESS"); err != nil {
		uc.log.Error(logger.Entry{
			Action:  "start_ride_update_status_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
			RideID:  input.RideID,
		})
		return nil, fmt.Errorf("update ride status: %w", err)
	}

	// Обновляем статус водителя на BUSY
	if err := uc.driverRepo.UpdateStatus(ctx, input.DriverID, domain.DriverStatusBusy); err != nil {
		uc.log.Error(logger.Entry{
			Action:  "start_ride_update_driver_status_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
			RideID:  input.RideID,
		})
		return nil, fmt.Errorf("update driver status: %w", err)
	}

	startedAt := time.Now().UTC()

	uc.log.Info(logger.Entry{
		Action:  "ride_started",
		Message: "ride is now in progress",
		RideID:  input.RideID,
		Additional: map[string]any{
			"driver_id":  input.DriverID,
			"started_at": startedAt.Format(time.RFC3339),
		},
	})

	return &in.StartRideOutput{
		RideID:    input.RideID,
		Status:    "IN_PROGRESS",
		StartedAt: startedAt.Format(time.RFC3339),
		Message:   "Ride started successfully",
	}, nil
}
