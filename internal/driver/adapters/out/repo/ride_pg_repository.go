package repo

import (
	"context"
	"database/sql"
	"fmt"

	out "ridehail/internal/driver/application/ports/out"
	"ridehail/internal/driver/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ridePgRepository struct {
	pool *pgxpool.Pool
}

func NewRidePgRepository(pool *pgxpool.Pool) out.RideRepository {
	return &ridePgRepository{pool: pool}
}

func (r *ridePgRepository) FindByID(ctx context.Context, rideID string) (*out.Ride, error) {
	query := `
		SELECT id, ride_number, passenger_id, driver_id, vehicle_type, status, pickup_coordinate_id, destination_coordinate_id, estimated_fare, final_fare
		FROM rides
		WHERE id = $1
	`

	var ride out.Ride

	err := r.pool.QueryRow(ctx, query, rideID).Scan(
		&ride.ID,
		&ride.RideNumber,
		&ride.PassengerID,
		&ride.DriverID,
		&ride.VehicleType,
		&ride.Status,
		&ride.PickupCoordinateID,
		&ride.DestinationCoordinateID,
		&ride.EstimatedFare,
		&ride.FinalFare,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrRideNotFound
		}
		return nil, fmt.Errorf("query ride: %w", err)
	}

	return &ride, nil
}

func (r *ridePgRepository) UpdateRideDriver(ctx context.Context, rideID, driverID string) error {
	query := `
		UPDATE rides
		SET driver_id = $1, status = 'MATCHED', matched_at = NOW(), updated_at = NOW()
		WHERE id = $2 AND status = 'REQUESTED'
	`

	result, err := r.pool.Exec(ctx, query, driverID, rideID)
	if err != nil {
		return fmt.Errorf("update ride driver: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrRideAlreadyMatched
	}

	return nil
}

func (r *ridePgRepository) UpdateRideStatus(ctx context.Context, rideID, status string) error {
	query := `
		UPDATE rides
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := r.pool.Exec(ctx, query, status, rideID)
	if err != nil {
		return fmt.Errorf("update ride status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrRideNotFound
	}

	return nil
}

func (r *ridePgRepository) UpdateFinalFare(ctx context.Context, rideID string, finalFare float64) error {
	query := `
		UPDATE rides
		SET final_fare = $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := r.pool.Exec(ctx, query, finalFare, rideID)
	if err != nil {
		return fmt.Errorf("update final fare: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrRideNotFound
	}

	return nil
}
