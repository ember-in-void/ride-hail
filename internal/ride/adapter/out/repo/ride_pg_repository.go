package repo

import (
	"context"
	"errors"
	"fmt"

	"ridehail/internal/ride/domain"
	"ridehail/internal/shared/logger"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RidePgRepository — PostgreSQL репозиторий для работы с поездками
type RidePgRepository struct {
	pool *pgxpool.Pool
	log  *logger.Logger
}

// NewRidePgRepository создает новый экземпляр репозитория
func NewRidePgRepository(pool *pgxpool.Pool, log *logger.Logger) *RidePgRepository {
	return &RidePgRepository{
		pool: pool,
		log:  log,
	}
}

// Create создает новую поездку
func (r *RidePgRepository) Create(ctx context.Context, ride *domain.Ride) error {
	query := `
		INSERT INTO rides (
			id, ride_number, passenger_id, driver_id, vehicle_type, status, priority,
			requested_at, matched_at, arrived_at, started_at, completed_at, cancelled_at,
			cancellation_reason, estimated_fare, final_fare,
			pickup_coordinate_id, destination_coordinate_id,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20
		)
	`

	_, err := r.pool.Exec(ctx, query,
		ride.ID,
		ride.RideNumber,
		ride.PassengerID,
		ride.DriverID,
		ride.VehicleType,
		ride.Status,
		ride.Priority,
		ride.RequestedAt,
		ride.MatchedAt,
		ride.ArrivedAt,
		ride.StartedAt,
		ride.CompletedAt,
		ride.CancelledAt,
		ride.CancellationReason,
		ride.EstimatedFare,
		ride.FinalFare,
		ride.PickupCoordinateID,
		ride.DestinationCoordinateID,
		ride.CreatedAt,
		ride.UpdatedAt,
	)
	if err != nil {
		r.log.Error(logger.Entry{
			Action:  "db_create_ride_failed",
			Message: err.Error(),
			RideID:  ride.ID,
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		return fmt.Errorf("insert ride: %w", err)
	}

	return nil
}

// FindByID возвращает поездку по ID
func (r *RidePgRepository) FindByID(ctx context.Context, rideID string) (*domain.Ride, error) {
	query := `
		SELECT 
			id, ride_number, passenger_id, driver_id, vehicle_type, status, priority,
			requested_at, matched_at, arrived_at, started_at, completed_at, cancelled_at,
			cancellation_reason, estimated_fare, final_fare,
			pickup_coordinate_id, destination_coordinate_id,
			created_at, updated_at
		FROM rides
		WHERE id = $1
	`

	ride := &domain.Ride{}
	err := r.pool.QueryRow(ctx, query, rideID).Scan(
		&ride.ID,
		&ride.RideNumber,
		&ride.PassengerID,
		&ride.DriverID,
		&ride.VehicleType,
		&ride.Status,
		&ride.Priority,
		&ride.RequestedAt,
		&ride.MatchedAt,
		&ride.ArrivedAt,
		&ride.StartedAt,
		&ride.CompletedAt,
		&ride.CancelledAt,
		&ride.CancellationReason,
		&ride.EstimatedFare,
		&ride.FinalFare,
		&ride.PickupCoordinateID,
		&ride.DestinationCoordinateID,
		&ride.CreatedAt,
		&ride.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrRideNotFound
		}
		r.log.Error(logger.Entry{
			Action:  "db_find_ride_by_id_failed",
			Message: err.Error(),
			RideID:  rideID,
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		return nil, fmt.Errorf("query ride by id: %w", err)
	}

	return ride, nil
}

// FindByRideNumber возвращает поездку по номеру
func (r *RidePgRepository) FindByRideNumber(ctx context.Context, rideNumber string) (*domain.Ride, error) {
	query := `
		SELECT 
			id, ride_number, passenger_id, driver_id, vehicle_type, status, priority,
			requested_at, matched_at, arrived_at, started_at, completed_at, cancelled_at,
			cancellation_reason, estimated_fare, final_fare,
			pickup_coordinate_id, destination_coordinate_id,
			created_at, updated_at
		FROM rides
		WHERE ride_number = $1
	`

	ride := &domain.Ride{}
	err := r.pool.QueryRow(ctx, query, rideNumber).Scan(
		&ride.ID,
		&ride.RideNumber,
		&ride.PassengerID,
		&ride.DriverID,
		&ride.VehicleType,
		&ride.Status,
		&ride.Priority,
		&ride.RequestedAt,
		&ride.MatchedAt,
		&ride.ArrivedAt,
		&ride.StartedAt,
		&ride.CompletedAt,
		&ride.CancelledAt,
		&ride.CancellationReason,
		&ride.EstimatedFare,
		&ride.FinalFare,
		&ride.PickupCoordinateID,
		&ride.DestinationCoordinateID,
		&ride.CreatedAt,
		&ride.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrRideNotFound
		}
		return nil, fmt.Errorf("query ride by number: %w", err)
	}

	return ride, nil
}

// Update обновляет существующую поездку
func (r *RidePgRepository) Update(ctx context.Context, ride *domain.Ride) error {
	query := `
		UPDATE rides SET
			driver_id = $2,
			status = $3,
			matched_at = $4,
			arrived_at = $5,
			started_at = $6,
			completed_at = $7,
			cancelled_at = $8,
			cancellation_reason = $9,
			final_fare = $10,
			updated_at = $11
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query,
		ride.ID,
		ride.DriverID,
		ride.Status,
		ride.MatchedAt,
		ride.ArrivedAt,
		ride.StartedAt,
		ride.CompletedAt,
		ride.CancelledAt,
		ride.CancellationReason,
		ride.FinalFare,
		ride.UpdatedAt,
	)
	if err != nil {
		r.log.Error(logger.Entry{
			Action:  "db_update_ride_failed",
			Message: err.Error(),
			RideID:  ride.ID,
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		return fmt.Errorf("update ride: %w", err)
	}

	return nil
}

// FindActiveByPassengerID возвращает активные поездки пассажира
func (r *RidePgRepository) FindActiveByPassengerID(ctx context.Context, passengerID string) ([]*domain.Ride, error) {
	query := `
		SELECT 
			id, ride_number, passenger_id, driver_id, vehicle_type, status, priority,
			requested_at, matched_at, arrived_at, started_at, completed_at, cancelled_at,
			cancellation_reason, estimated_fare, final_fare,
			pickup_coordinate_id, destination_coordinate_id,
			created_at, updated_at
		FROM rides
		WHERE passenger_id = $1 
		  AND status NOT IN ('COMPLETED', 'CANCELLED')
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, passengerID)
	if err != nil {
		return nil, fmt.Errorf("query active rides: %w", err)
	}
	defer rows.Close()

	var rides []*domain.Ride
	for rows.Next() {
		ride := &domain.Ride{}
		err := rows.Scan(
			&ride.ID,
			&ride.RideNumber,
			&ride.PassengerID,
			&ride.DriverID,
			&ride.VehicleType,
			&ride.Status,
			&ride.Priority,
			&ride.RequestedAt,
			&ride.MatchedAt,
			&ride.ArrivedAt,
			&ride.StartedAt,
			&ride.CompletedAt,
			&ride.CancelledAt,
			&ride.CancellationReason,
			&ride.EstimatedFare,
			&ride.FinalFare,
			&ride.PickupCoordinateID,
			&ride.DestinationCoordinateID,
			&ride.CreatedAt,
			&ride.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan ride: %w", err)
		}
		rides = append(rides, ride)
	}

	return rides, rows.Err()
}

// FindByStatus возвращает поездки с определенным статусом
func (r *RidePgRepository) FindByStatus(ctx context.Context, status string, limit int) ([]*domain.Ride, error) {
	query := `
		SELECT 
			id, ride_number, passenger_id, driver_id, vehicle_type, status, priority,
			requested_at, matched_at, arrived_at, started_at, completed_at, cancelled_at,
			cancellation_reason, estimated_fare, final_fare,
			pickup_coordinate_id, destination_coordinate_id,
			created_at, updated_at
		FROM rides
		WHERE status = $1
		ORDER BY priority DESC, created_at ASC
		LIMIT $2
	`

	rows, err := r.pool.Query(ctx, query, status, limit)
	if err != nil {
		return nil, fmt.Errorf("query rides by status: %w", err)
	}
	defer rows.Close()

	var rides []*domain.Ride
	for rows.Next() {
		ride := &domain.Ride{}
		err := rows.Scan(
			&ride.ID,
			&ride.RideNumber,
			&ride.PassengerID,
			&ride.DriverID,
			&ride.VehicleType,
			&ride.Status,
			&ride.Priority,
			&ride.RequestedAt,
			&ride.MatchedAt,
			&ride.ArrivedAt,
			&ride.StartedAt,
			&ride.CompletedAt,
			&ride.CancelledAt,
			&ride.CancellationReason,
			&ride.EstimatedFare,
			&ride.FinalFare,
			&ride.PickupCoordinateID,
			&ride.DestinationCoordinateID,
			&ride.CreatedAt,
			&ride.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan ride: %w", err)
		}
		rides = append(rides, ride)
	}

	return rides, rows.Err()
}

// AssignDriver назначает водителя на поездку и обновляет статус на DRIVER_ASSIGNED
func (r *RidePgRepository) AssignDriver(ctx context.Context, rideID string, driverID string) error {
	query := `
UPDATE rides 
SET 
driver_id = $1,
status = 'MATCHED',
matched_at = NOW(),
updated_at = NOW()
WHERE id = $2
  AND status = 'REQUESTED'
`

	result, err := r.pool.Exec(ctx, query, driverID, rideID)
	if err != nil {
		r.log.Error(logger.Entry{
			Action:  "db_assign_driver_failed",
			Message: err.Error(),
			RideID:  rideID,
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		return fmt.Errorf("assign driver: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("ride not found or already assigned (ride_id=%s)", rideID)
	}

	r.log.Info(logger.Entry{
		Action:  "driver_assigned_to_ride",
		Message: fmt.Sprintf("driver %s assigned to ride %s", driverID, rideID),
		RideID:  rideID,
		Additional: map[string]any{
			"driver_id": driverID,
		},
	})

	return nil
}
