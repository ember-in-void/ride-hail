package repo

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"ridehail/internal/driver/application/ports/out"
	"ridehail/internal/driver/domain"
)

type driverPgRepository struct {
	pool *pgxpool.Pool
}

func NewDriverPgRepository(pool *pgxpool.Pool) out.DriverRepository {
	return &driverPgRepository{pool: pool}
}

func (r *driverPgRepository) FindByID(ctx context.Context, driverID string) (*domain.Driver, error) {
	query := `
		SELECT id, license_number, vehicle_type, vehicle_attrs, rating, 
		       total_rides, total_earnings, status, is_verified, created_at, updated_at
		FROM drivers
		WHERE id = $1
	`

	var driver domain.Driver
	err := r.pool.QueryRow(ctx, query, driverID).Scan(
		&driver.ID,
		&driver.LicenseNumber,
		&driver.VehicleType,
		&driver.VehicleAttrs,
		&driver.Rating,
		&driver.TotalRides,
		&driver.TotalEarnings,
		&driver.Status,
		&driver.IsVerified,
		&driver.CreatedAt,
		&driver.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrDriverNotFound
		}
		return nil, err
	}

	return &driver, nil
}

func (r *driverPgRepository) UpdateStatus(ctx context.Context, driverID string, status domain.DriverStatus) error {
	query := `
		UPDATE drivers
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := r.pool.Exec(ctx, query, status, driverID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return domain.ErrDriverNotFound
	}

	return nil
}

func (r *driverPgRepository) UpdateEarnings(ctx context.Context, driverID string, additionalEarnings float64) error {
	query := `
		UPDATE drivers
		SET total_earnings = total_earnings + $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := r.pool.Exec(ctx, query, additionalEarnings, driverID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return domain.ErrDriverNotFound
	}

	return nil
}

func (r *driverPgRepository) IncrementRidesCount(ctx context.Context, driverID string) error {
	query := `
		UPDATE drivers
		SET total_rides = total_rides + 1, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, driverID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return domain.ErrDriverNotFound
	}

	return nil
}

func (r *driverPgRepository) FindAvailableDriversNearby(
	ctx context.Context,
	latitude, longitude float64,
	radiusKM float64,
	vehicleType string,
	limit int,
) ([]domain.DriverWithLocation, error) {
	query := `
		SELECT d.id, u.email, d.rating, d.vehicle_attrs, c.latitude, c.longitude,
		       ST_Distance(
		         ST_MakePoint(c.longitude, c.latitude)::geography,
		         ST_MakePoint($1, $2)::geography
		       ) / 1000 as distance_km
		FROM drivers d
		JOIN users u ON d.id = u.id
		JOIN coordinates c ON c.entity_id = d.id
		  AND c.entity_type = 'driver'
		  AND c.is_current = true
		WHERE d.status = 'AVAILABLE'
		  AND d.vehicle_type = $3
		  AND ST_DWithin(
		        ST_MakePoint(c.longitude, c.latitude)::geography,
		        ST_MakePoint($1, $2)::geography,
		        $4
		      )
		ORDER BY distance_km, d.rating DESC
		LIMIT $5
	`

	radiusMeters := radiusKM * 1000

	rows, err := r.pool.Query(ctx, query, longitude, latitude, vehicleType, radiusMeters, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var drivers []domain.DriverWithLocation
	for rows.Next() {
		var dwl domain.DriverWithLocation
		var email string

		err := rows.Scan(
			&dwl.Driver.ID,
			&email,
			&dwl.Driver.Rating,
			&dwl.Driver.VehicleAttrs,
			&dwl.Latitude,
			&dwl.Longitude,
			&dwl.DistanceKM,
		)
		if err != nil {
			return nil, err
		}

		drivers = append(drivers, dwl)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return drivers, nil
}
