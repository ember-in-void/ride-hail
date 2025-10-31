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

type completeRideUseCase struct {
	driverRepo   out.DriverRepository
	sessionRepo  out.SessionRepository
	rideRepo     out.RideRepository
	locationRepo out.LocationRepository
	log          *logger.Logger
}

func NewCompleteRideUseCase(
	driverRepo out.DriverRepository,
	sessionRepo out.SessionRepository,
	rideRepo out.RideRepository,
	locationRepo out.LocationRepository,
	log *logger.Logger,
) in.CompleteRideUseCase {
	return &completeRideUseCase{
		driverRepo:   driverRepo,
		sessionRepo:  sessionRepo,
		rideRepo:     rideRepo,
		locationRepo: locationRepo,
		log:          log,
	}
}

func (uc *completeRideUseCase) Execute(ctx context.Context, input in.CompleteRideInput) (*in.CompleteRideOutput, error) {
	// Находим водителя
	driver, err := uc.driverRepo.FindByID(ctx, input.DriverID)
	if err != nil {
		uc.log.Error(logger.Entry{
			Action:  "complete_ride_driver_not_found",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
			Additional: map[string]any{
				"driver_id": input.DriverID,
				"ride_id":   input.RideID,
			},
		})
		return nil, err
	}

	// Проверяем статус водителя (должен быть BUSY)
	if driver.Status != domain.DriverStatusBusy {
		uc.log.Warn(logger.Entry{
			Action:  "complete_ride_invalid_driver_status",
			Message: "driver cannot complete ride with current status",
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
			Action:  "complete_ride_ride_not_found",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
			RideID:  input.RideID,
		})
		return nil, err
	}

	// Проверяем, что поездка назначена этому водителю
	if ride.DriverID == nil || *ride.DriverID != input.DriverID {
		uc.log.Warn(logger.Entry{
			Action:  "complete_ride_driver_mismatch",
			Message: "ride assigned to different driver",
			RideID:  input.RideID,
			Additional: map[string]any{
				"driver_id":          input.DriverID,
				"assigned_driver_id": ride.DriverID,
			},
		})
		return nil, fmt.Errorf("ride assigned to different driver")
	}

	// Рассчитываем финальную стоимость (упрощенно: используем estimated_fare или recalculate)
	finalFare := ride.EstimatedFare
	if ride.FinalFare != nil {
		finalFare = *ride.FinalFare
	}

	// Earnings для водителя: 80% от fare (согласно ТЗ, упрощенно)
	driverEarnings := finalFare * 0.8

	// Обновляем финальную стоимость поездки
	if err := uc.rideRepo.UpdateFinalFare(ctx, input.RideID, finalFare); err != nil {
		uc.log.Error(logger.Entry{
			Action:  "complete_ride_update_fare_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
			RideID:  input.RideID,
		})
		return nil, fmt.Errorf("update final fare: %w", err)
	}

	// Обновляем статус поездки на COMPLETED
	if err := uc.rideRepo.UpdateRideStatus(ctx, input.RideID, "COMPLETED"); err != nil {
		uc.log.Error(logger.Entry{
			Action:  "complete_ride_update_status_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
			RideID:  input.RideID,
		})
		return nil, fmt.Errorf("update ride status: %w", err)
	}

	// Обновляем статус водителя на AVAILABLE
	if err := uc.driverRepo.UpdateStatus(ctx, input.DriverID, domain.DriverStatusAvailable); err != nil {
		uc.log.Error(logger.Entry{
			Action:  "complete_ride_update_driver_status_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
			RideID:  input.RideID,
		})
		return nil, fmt.Errorf("update driver status: %w", err)
	}

	// Обновляем статистику водителя (total_rides, total_earnings)
	if err := uc.driverRepo.UpdateRideStats(ctx, input.DriverID, 1, driverEarnings); err != nil {
		uc.log.Error(logger.Entry{
			Action:  "complete_ride_update_driver_stats_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		// Не фатальная ошибка
	}

	// Обновляем статистику сессии
	session, _ := uc.sessionRepo.FindActiveByDriverID(ctx, input.DriverID)
	if session != nil {
		if err := uc.sessionRepo.UpdateStats(ctx, session.ID, 1, driverEarnings); err != nil {
			uc.log.Error(logger.Entry{
				Action:  "complete_ride_update_session_stats_failed",
				Message: err.Error(),
				Error:   &logger.ErrObj{Msg: err.Error()},
			})
			// Не фатальная ошибка
		}
	}

	completedAt := time.Now().UTC()

	uc.log.Info(logger.Entry{
		Action:  "ride_completed",
		Message: "ride completed successfully",
		RideID:  input.RideID,
		Additional: map[string]any{
			"driver_id":       input.DriverID,
			"completed_at":    completedAt.Format(time.RFC3339),
			"driver_earnings": driverEarnings,
			"final_fare":      finalFare,
		},
	})

	return &in.CompleteRideOutput{
		RideID:         input.RideID,
		Status:         "COMPLETED",
		CompletedAt:    completedAt.Format(time.RFC3339),
		DriverEarnings: driverEarnings,
		Message:        "Ride completed successfully",
	}, nil
}
