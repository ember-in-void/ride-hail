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

type updateLocationUseCase struct {
	locationRepo out.LocationRepository
	eventPub     out.EventPublisher
	log          *logger.Logger
}

func NewUpdateLocationUseCase(
	locationRepo out.LocationRepository,
	eventPub out.EventPublisher,
	log *logger.Logger,
) in.UpdateLocationUseCase {
	return &updateLocationUseCase{
		locationRepo: locationRepo,
		eventPub:     eventPub,
		log:          log,
	}
}

func (uc *updateLocationUseCase) Execute(ctx context.Context, input in.UpdateLocationInput) (*in.UpdateLocationOutput, error) {
	// Валидация координат
	if input.Latitude < -90 || input.Latitude > 90 || input.Longitude < -180 || input.Longitude > 180 {
		return nil, domain.ErrInvalidCoordinates
	}

	// Rate-limit: не чаще 1 раза в 3 секунды (согласно регламенту)
	lastUpdateTime, err := uc.locationRepo.GetLastLocationUpdateTime(ctx, input.DriverID)
	if err == nil && lastUpdateTime != nil {
		lastUpdate, parseErr := time.Parse(time.RFC3339, *lastUpdateTime)
		if parseErr == nil && time.Since(lastUpdate) < 3*time.Second {
			uc.log.Debug(logger.Entry{
				Action:  "location_update_rate_limited",
				Message: "location update too frequent",
				Additional: map[string]any{
					"driver_id":        input.DriverID,
					"last_update":      *lastUpdateTime,
					"seconds_since":    time.Since(lastUpdate).Seconds(),
					"required_seconds": 3,
				},
			})
			return nil, domain.ErrLocationUpdateTooFrequent
		}
	}

	// Сохраняем координату
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
			Action:  "location_update_save_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
			Additional: map[string]any{
				"driver_id": input.DriverID,
			},
		})
		return nil, fmt.Errorf("save coordinate: %w", err)
	}

	// Сохраняем в location_history
	if err := uc.locationRepo.SaveLocationHistory(
		ctx,
		input.DriverID,
		input.Latitude,
		input.Longitude,
		input.AccuracyMeters,
		input.SpeedKmh,
		input.HeadingDegrees,
		nil, // rideID = nil (пока не привязано к поездке)
	); err != nil {
		uc.log.Error(logger.Entry{
			Action:  "location_update_history_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		// Не фатальная ошибка
	}

	// Публикуем обновление локации в location_fanout
	if err := uc.eventPub.PublishLocationUpdate(
		ctx,
		input.DriverID,
		nil, // rideID = nil
		input.Latitude,
		input.Longitude,
		input.SpeedKmh,
		input.HeadingDegrees,
	); err != nil {
		uc.log.Error(logger.Entry{
			Action:  "location_update_publish_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		// Не фатальная ошибка
	}

	uc.log.Debug(logger.Entry{
		Action:  "location_updated",
		Message: "driver location updated",
		Additional: map[string]any{
			"driver_id": input.DriverID,
			"latitude":  input.Latitude,
			"longitude": input.Longitude,
		},
	})

	return &in.UpdateLocationOutput{
		CoordinateID: coord.ID,
		UpdatedAt:    coord.UpdatedAt.Format(time.RFC3339),
	}, nil
}
