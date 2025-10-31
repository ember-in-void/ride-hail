package usecase

import (
	"context"
	"fmt"
	"time"

	"ridehail/internal/driver/application/ports/in"
	"ridehail/internal/driver/application/ports/out"
	"ridehail/internal/driver/domain"
	"ridehail/internal/shared/logger"
	"ridehail/internal/shared/utils"
)

// DriverService реализует бизнес-логику управления водителем
type DriverService struct {
	driverRepo   out.DriverRepository
	locationRepo out.LocationRepository
	rideRepo     out.RideRepository
	msgPublisher out.MessagePublisher
	log          *logger.Logger
}

// NewDriverService создает новый сервис управления водителем
func NewDriverService(
	driverRepo out.DriverRepository,
	locationRepo out.LocationRepository,
	rideRepo out.RideRepository,
	msgPublisher out.MessagePublisher,
	log *logger.Logger,
) *DriverService {
	return &DriverService{
		driverRepo:   driverRepo,
		locationRepo: locationRepo,
		rideRepo:     rideRepo,
		msgPublisher: msgPublisher,
		log:          log,
	}
}

// GoOnline переводит водителя в статус AVAILABLE
func (s *DriverService) GoOnline(ctx context.Context, input in.GoOnlineInput) (in.GoOnlineOutput, error) {
	// Валидация координат
	if err := validateCoordinates(input.Latitude, input.Longitude); err != nil {
		s.log.Error(logger.Entry{
			Action:  "go_online_invalid_coordinates",
			Message: err.Error(),
			Error: &logger.ErrObj{
				Msg: err.Error(),
			},
		})
		return in.GoOnlineOutput{}, fmt.Errorf("invalid coordinates: %w", err)
	}

	// Получаем водителя из БД
	driver, err := s.driverRepo.FindByID(ctx, input.DriverID)
	if err != nil {
		s.log.Error(logger.Entry{
			Action:  "go_online_driver_not_found",
			Message: err.Error(),
			Error: &logger.ErrObj{
				Msg: err.Error(),
			},
		})
		return in.GoOnlineOutput{}, fmt.Errorf("find driver: %w", err)
	}

	// Проверяем, может ли водитель выйти в онлайн
	if !driver.CanGoOnline() {
		err := domain.ErrDriverCannotGoOnline
		s.log.Error(logger.Entry{
			Action:  "go_online_invalid_status",
			Message: fmt.Sprintf("driver status: %s, verified: %t", driver.Status, driver.IsVerified),
			Error: &logger.ErrObj{
				Msg: err.Error(),
			},
		})
		return in.GoOnlineOutput{}, err
	}

	// Создаем новую сессию
	sessionID := utils.NewUUID()
	session := &domain.DriverSession{
		ID:            sessionID,
		DriverID:      input.DriverID,
		StartedAt:     time.Now().UTC(),
		EndedAt:       nil,
		TotalRides:    0,
		TotalEarnings: 0,
	}

	if err := s.driverRepo.CreateSession(ctx, session); err != nil {
		s.log.Error(logger.Entry{
			Action:  "go_online_create_session_failed",
			Message: err.Error(),
			Error: &logger.ErrObj{
				Msg: err.Error(),
			},
		})
		return in.GoOnlineOutput{}, fmt.Errorf("create session: %w", err)
	}

	// Обновляем статус водителя на AVAILABLE
	if err := s.driverRepo.UpdateStatus(ctx, input.DriverID, domain.DriverStatusAvailable); err != nil {
		s.log.Error(logger.Entry{
			Action:  "go_online_update_status_failed",
			Message: err.Error(),
			Error: &logger.ErrObj{
				Msg: err.Error(),
			},
		})
		return in.GoOnlineOutput{}, fmt.Errorf("update status: %w", err)
	}

	// Создаем координату для текущей локации водителя
	coordinateID, err := s.locationRepo.CreateCoordinate(ctx, &out.CreateCoordinateDTO{
		EntityID:   input.DriverID,
		EntityType: "driver",
		Address:    "Current Location",
		Latitude:   input.Latitude,
		Longitude:  input.Longitude,
		IsCurrent:  true,
	})
	if err != nil {
		s.log.Error(logger.Entry{
			Action:  "go_online_create_coordinate_failed",
			Message: err.Error(),
			Error: &logger.ErrObj{
				Msg: err.Error(),
			},
		})
		return in.GoOnlineOutput{}, fmt.Errorf("create coordinate: %w", err)
	}

	// Публикуем событие изменения статуса водителя
	if err := s.msgPublisher.PublishDriverStatus(ctx, &out.DriverStatusDTO{
		DriverID:  input.DriverID,
		Status:    domain.DriverStatusAvailable,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		// Логируем ошибку, но не прерываем поток (eventual consistency)
		s.log.Error(logger.Entry{
			Action:  "go_online_publish_status_failed",
			Message: err.Error(),
			Error: &logger.ErrObj{
				Msg: err.Error(),
			},
		})
	}

	s.log.Info(logger.Entry{
		Action:  "driver_went_online",
		Message: fmt.Sprintf("driver_id=%s, session_id=%s, coordinate_id=%s", input.DriverID, sessionID, coordinateID),
	})

	return in.GoOnlineOutput{
		Status:    string(domain.DriverStatusAvailable),
		SessionID: sessionID,
		Message:   "You are now online and ready to accept rides",
	}, nil
}

// GoOffline переводит водителя в статус OFFLINE
func (s *DriverService) GoOffline(ctx context.Context, input in.GoOfflineInput) (in.GoOfflineOutput, error) {
	// Получаем водителя из БД
	driver, err := s.driverRepo.FindByID(ctx, input.DriverID)
	if err != nil {
		s.log.Error(logger.Entry{
			Action:  "go_offline_driver_not_found",
			Message: err.Error(),
			Error: &logger.ErrObj{
				Msg: err.Error(),
			},
		})
		return in.GoOfflineOutput{}, fmt.Errorf("find driver: %w", err)
	}

	// Проверяем, может ли водитель выйти в оффлайн
	if driver.Status != domain.DriverStatusAvailable {
		err := domain.ErrDriverCannotGoOffline
		s.log.Error(logger.Entry{
			Action:  "go_offline_invalid_status",
			Message: fmt.Sprintf("driver status: %s", driver.Status),
			Error: &logger.ErrObj{
				Msg: err.Error(),
			},
		})
		return in.GoOfflineOutput{}, err
	}

	// Получаем активную сессию
	session, err := s.driverRepo.GetActiveSession(ctx, input.DriverID)
	if err != nil {
		s.log.Error(logger.Entry{
			Action:  "go_offline_session_not_found",
			Message: err.Error(),
			Error: &logger.ErrObj{
				Msg: err.Error(),
			},
		})
		return in.GoOfflineOutput{}, fmt.Errorf("get active session: %w", err)
	}

	// Завершаем сессию
	if err := s.driverRepo.EndSession(ctx, session.ID, session.TotalRides, session.TotalEarnings); err != nil {
		s.log.Error(logger.Entry{
			Action:  "go_offline_end_session_failed",
			Message: err.Error(),
			Error: &logger.ErrObj{
				Msg: err.Error(),
			},
		})
		return in.GoOfflineOutput{}, fmt.Errorf("end session: %w", err)
	}

	// Обновляем статус водителя на OFFLINE
	if err := s.driverRepo.UpdateStatus(ctx, input.DriverID, domain.DriverStatusOffline); err != nil {
		s.log.Error(logger.Entry{
			Action:  "go_offline_update_status_failed",
			Message: err.Error(),
			Error: &logger.ErrObj{
				Msg: err.Error(),
			},
		})
		return in.GoOfflineOutput{}, fmt.Errorf("update status: %w", err)
	}

	// Публикуем событие изменения статуса водителя
	if err := s.msgPublisher.PublishDriverStatus(ctx, &out.DriverStatusDTO{
		DriverID:  input.DriverID,
		Status:    domain.DriverStatusOffline,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		// Логируем ошибку, но не прерываем поток
		s.log.Error(logger.Entry{
			Action:  "go_offline_publish_status_failed",
			Message: err.Error(),
			Error: &logger.ErrObj{
				Msg: err.Error(),
			},
		})
	}

	// Вычисляем продолжительность сессии
	durationHours := time.Since(session.StartedAt).Hours()

	s.log.Info(logger.Entry{
		Action:  "driver_went_offline",
		Message: fmt.Sprintf("driver_id=%s, session_id=%s", input.DriverID, session.ID),
	})

	return in.GoOfflineOutput{
		Status:    string(domain.DriverStatusOffline),
		SessionID: session.ID,
		SessionSummary: in.SessionSummaryOutput{
			DurationHours:  durationHours,
			RidesCompleted: session.TotalRides,
			Earnings:       session.TotalEarnings,
		},
		Message: "You are now offline",
	}, nil
}

// UpdateLocation обновляет локацию водителя
func (s *DriverService) UpdateLocation(ctx context.Context, input in.UpdateLocationInput) (in.UpdateLocationOutput, error) {
	// Валидация координат
	if err := validateCoordinates(input.Latitude, input.Longitude); err != nil {
		s.log.Error(logger.Entry{
			Action:  "update_location_invalid_coordinates",
			Message: err.Error(),
			Error: &logger.ErrObj{
				Msg: err.Error(),
			},
		})
		return in.UpdateLocationOutput{}, fmt.Errorf("invalid coordinates: %w", err)
	}

	// Проверяем rate limit (макс 1 раз в 3 секунды)
	allowed, err := s.locationRepo.CheckRateLimit(ctx, input.DriverID)
	if err != nil {
		s.log.Error(logger.Entry{
			Action:  "update_location_rate_limit_check_failed",
			Message: err.Error(),
			Error: &logger.ErrObj{
				Msg: err.Error(),
			},
		})
		return in.UpdateLocationOutput{}, fmt.Errorf("check rate limit: %w", err)
	}

	if !allowed {
		s.log.Warn(logger.Entry{
			Action:  "update_location_rate_limit_exceeded",
			Message: fmt.Sprintf("driver_id=%s", input.DriverID),
		})
		return in.UpdateLocationOutput{}, domain.ErrRateLimitExceeded
	}

	// Обновляем текущую локацию
	coordinateID, err := s.locationRepo.UpdateCurrentLocation(ctx, input.DriverID, "driver", input.Latitude, input.Longitude)
	if err != nil {
		s.log.Error(logger.Entry{
			Action:  "update_location_failed",
			Message: err.Error(),
			Error: &logger.ErrObj{
				Msg: err.Error(),
			},
		})
		return in.UpdateLocationOutput{}, fmt.Errorf("update location: %w", err)
	}

	// Архивируем в location_history
	if err := s.locationRepo.ArchiveToHistory(ctx, &out.LocationHistoryDTO{
		CoordinateID:   coordinateID,
		DriverID:       input.DriverID,
		Latitude:       input.Latitude,
		Longitude:      input.Longitude,
		AccuracyMeters: input.AccuracyMeters,
		SpeedKmh:       input.SpeedKmh,
		HeadingDegrees: input.HeadingDegrees,
	}); err != nil {
		// Логируем ошибку, но не прерываем поток
		s.log.Error(logger.Entry{
			Action:  "update_location_archive_failed",
			Message: err.Error(),
			Error: &logger.ErrObj{
				Msg: err.Error(),
			},
		})
	}

	// Публикуем обновление локации в fanout exchange
	if err := s.msgPublisher.PublishLocationUpdate(ctx, &out.LocationUpdateDTO{
		DriverID: input.DriverID,
		Location: out.LocationDTO{
			Lat: input.Latitude,
			Lng: input.Longitude,
		},
		SpeedKmh:       input.SpeedKmh,
		HeadingDegrees: input.HeadingDegrees,
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		// Логируем ошибку, но не прерываем поток
		s.log.Error(logger.Entry{
			Action:  "update_location_publish_failed",
			Message: err.Error(),
			Error: &logger.ErrObj{
				Msg: err.Error(),
			},
		})
	}

	s.log.Debug(logger.Entry{
		Action:  "location_updated",
		Message: fmt.Sprintf("driver_id=%s, coordinate_id=%s", input.DriverID, coordinateID),
	})

	return in.UpdateLocationOutput{
		CoordinateID: coordinateID,
		UpdatedAt:    time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// StartRide начинает поездку
func (s *DriverService) StartRide(ctx context.Context, input in.StartRideInput) (in.StartRideOutput, error) {
	// Валидация координат
	if err := validateCoordinates(input.Latitude, input.Longitude); err != nil {
		s.log.Error(logger.Entry{
			Action:  "start_ride_invalid_coordinates",
			Message: err.Error(),
			Error: &logger.ErrObj{
				Msg: err.Error(),
			},
		})
		return in.StartRideOutput{}, fmt.Errorf("invalid coordinates: %w", err)
	}

	// Получаем поездку
	ride, err := s.rideRepo.FindByID(ctx, input.RideID)
	if err != nil {
		s.log.Error(logger.Entry{
			Action:  "start_ride_not_found",
			Message: err.Error(),
			RideID:  input.RideID,
			Error: &logger.ErrObj{
				Msg: err.Error(),
			},
		})
		return in.StartRideOutput{}, fmt.Errorf("find ride: %w", err)
	}

	// Проверяем, что водитель назначен на эту поездку
	if ride.DriverID == nil || *ride.DriverID != input.DriverID {
		s.log.Error(logger.Entry{
			Action:  "start_ride_driver_mismatch",
			Message: fmt.Sprintf("driver_id=%s, ride_driver_id=%v", input.DriverID, ride.DriverID),
			RideID:  input.RideID,
		})
		return in.StartRideOutput{}, fmt.Errorf("driver not assigned to this ride")
	}

	// Обновляем статус поездки на IN_PROGRESS
	if err := s.rideRepo.UpdateRideStatus(ctx, input.RideID, "IN_PROGRESS"); err != nil {
		s.log.Error(logger.Entry{
			Action:  "start_ride_update_status_failed",
			Message: err.Error(),
			RideID:  input.RideID,
			Error: &logger.ErrObj{
				Msg: err.Error(),
			},
		})
		return in.StartRideOutput{}, fmt.Errorf("update ride status: %w", err)
	}

	// Обновляем статус водителя на BUSY
	if err := s.driverRepo.UpdateStatus(ctx, input.DriverID, domain.DriverStatusBusy); err != nil {
		s.log.Error(logger.Entry{
			Action:  "start_ride_update_driver_status_failed",
			Message: err.Error(),
			RideID:  input.RideID,
			Error: &logger.ErrObj{
				Msg: err.Error(),
			},
		})
		return in.StartRideOutput{}, fmt.Errorf("update driver status: %w", err)
	}

	// Публикуем изменение статуса водителя
	if err := s.msgPublisher.PublishDriverStatus(ctx, &out.DriverStatusDTO{
		DriverID:  input.DriverID,
		Status:    domain.DriverStatusBusy,
		RideID:    input.RideID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		s.log.Error(logger.Entry{
			Action:  "start_ride_publish_status_failed",
			Message: err.Error(),
			RideID:  input.RideID,
			Error: &logger.ErrObj{
				Msg: err.Error(),
			},
		})
	}

	startedAt := time.Now().UTC().Format(time.RFC3339)

	s.log.Info(logger.Entry{
		Action:  "ride_started",
		Message: fmt.Sprintf("driver_id=%s, ride_id=%s", input.DriverID, input.RideID),
		RideID:  input.RideID,
	})

	return in.StartRideOutput{
		RideID:    input.RideID,
		Status:    "IN_PROGRESS",
		StartedAt: startedAt,
		Message:   "Ride started successfully",
	}, nil
}

// CompleteRide завершает поездку
func (s *DriverService) CompleteRide(ctx context.Context, input in.CompleteRideInput) (in.CompleteRideOutput, error) {
	// Валидация координат
	if err := validateCoordinates(input.FinalLatitude, input.FinalLongitude); err != nil {
		s.log.Error(logger.Entry{
			Action:  "complete_ride_invalid_coordinates",
			Message: err.Error(),
			RideID:  input.RideID,
			Error: &logger.ErrObj{
				Msg: err.Error(),
			},
		})
		return in.CompleteRideOutput{}, fmt.Errorf("invalid coordinates: %w", err)
	}

	// Получаем поездку
	ride, err := s.rideRepo.FindByID(ctx, input.RideID)
	if err != nil {
		s.log.Error(logger.Entry{
			Action:  "complete_ride_not_found",
			Message: err.Error(),
			RideID:  input.RideID,
			Error: &logger.ErrObj{
				Msg: err.Error(),
			},
		})
		return in.CompleteRideOutput{}, fmt.Errorf("find ride: %w", err)
	}

	// Проверяем, что водитель назначен на эту поездку
	if ride.DriverID == nil || *ride.DriverID != input.DriverID {
		s.log.Error(logger.Entry{
			Action:  "complete_ride_driver_mismatch",
			Message: fmt.Sprintf("driver_id=%s, ride_driver_id=%v", input.DriverID, ride.DriverID),
			RideID:  input.RideID,
		})
		return in.CompleteRideOutput{}, fmt.Errorf("driver not assigned to this ride")
	}

	// Вычисляем финальную стоимость (можно использовать базовую логику)
	// Берем estimate или пересчитываем на основе actual distance/duration
	var finalFare float64
	if ride.EstimatedFare != nil {
		// Используем примерную оценку, можно улучшить расчет
		finalFare = *ride.EstimatedFare
		// Можно добавить коррекцию на основе фактических данных
		if input.ActualDistanceKm > 0 {
			// Простая логика: базовая ставка + стоимость за км + стоимость за минуту
			baseRate := 500.0 // для ECONOMY
			ratePerKm := 100.0
			ratePerMin := 50.0
			finalFare = baseRate + (input.ActualDistanceKm * ratePerKm) + (float64(input.ActualDurationMinutes) * ratePerMin)
		}
	} else {
		finalFare = 1000.0 // fallback
	}

	// Обновляем финальную стоимость поездки
	if err := s.rideRepo.UpdateFinalFare(ctx, input.RideID, finalFare); err != nil {
		s.log.Error(logger.Entry{
			Action:  "complete_ride_update_fare_failed",
			Message: err.Error(),
			RideID:  input.RideID,
			Error: &logger.ErrObj{
				Msg: err.Error(),
			},
		})
		return in.CompleteRideOutput{}, fmt.Errorf("update final fare: %w", err)
	}

	// Обновляем статус поездки на COMPLETED
	if err := s.rideRepo.UpdateRideStatus(ctx, input.RideID, "COMPLETED"); err != nil {
		s.log.Error(logger.Entry{
			Action:  "complete_ride_update_status_failed",
			Message: err.Error(),
			RideID:  input.RideID,
			Error: &logger.ErrObj{
				Msg: err.Error(),
			},
		})
		return in.CompleteRideOutput{}, fmt.Errorf("update ride status: %w", err)
	}

	// Обновляем статус водителя на AVAILABLE
	if err := s.driverRepo.UpdateStatus(ctx, input.DriverID, domain.DriverStatusAvailable); err != nil {
		s.log.Error(logger.Entry{
			Action:  "complete_ride_update_driver_status_failed",
			Message: err.Error(),
			RideID:  input.RideID,
			Error: &logger.ErrObj{
				Msg: err.Error(),
			},
		})
		return in.CompleteRideOutput{}, fmt.Errorf("update driver status: %w", err)
	}

	// Рассчитываем заработок водителя (например, 80% от стоимости)
	driverEarnings := finalFare * 0.8

	// Обновляем статистику водителя
	if err := s.driverRepo.UpdateRideStats(ctx, input.DriverID, 1, driverEarnings); err != nil {
		s.log.Error(logger.Entry{
			Action:  "complete_ride_update_stats_failed",
			Message: err.Error(),
			RideID:  input.RideID,
			Error: &logger.ErrObj{
				Msg: err.Error(),
			},
		})
		// Не прерываем поток, это не критичная ошибка
	}

	// Публикуем изменение статуса водителя
	if err := s.msgPublisher.PublishDriverStatus(ctx, &out.DriverStatusDTO{
		DriverID:  input.DriverID,
		Status:    domain.DriverStatusAvailable,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		s.log.Error(logger.Entry{
			Action:  "complete_ride_publish_status_failed",
			Message: err.Error(),
			RideID:  input.RideID,
			Error: &logger.ErrObj{
				Msg: err.Error(),
			},
		})
	}

	completedAt := time.Now().UTC().Format(time.RFC3339)

	s.log.Info(logger.Entry{
		Action:  "ride_completed",
		Message: fmt.Sprintf("driver_id=%s, ride_id=%s, fare=%.2f", input.DriverID, input.RideID, finalFare),
		RideID:  input.RideID,
	})

	return in.CompleteRideOutput{
		RideID:         input.RideID,
		Status:         "COMPLETED",
		CompletedAt:    completedAt,
		DriverEarnings: driverEarnings,
		Message:        "Ride completed successfully",
	}, nil
}

// validateCoordinates проверяет корректность координат
func validateCoordinates(lat, lng float64) error {
	if lat < -90 || lat > 90 {
		return fmt.Errorf("latitude must be between -90 and 90, got %f", lat)
	}
	if lng < -180 || lng > 180 {
		return fmt.Errorf("longitude must be between -180 and 180, got %f", lng)
	}
	return nil
}
