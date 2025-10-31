package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	out "ridehail/internal/driver/application/ports/out"
	"ridehail/internal/driver/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type driverPgRepository struct {
	pool *pgxpool.Pool
}

func NewDriverPgRepository(pool *pgxpool.Pool) out.DriverRepository {
	return &driverPgRepository{pool: pool}
}

func (r *driverPgRepository) FindByID(ctx context.Context, driverID string) (*domain.Driver, error) {
	query := `
		SELECT id, license_number, vehicle_type, vehicle_attrs, rating, total_rides, total_earnings, status, is_verified, created_at, updated_at
		FROM drivers
		WHERE id = $1
	`

	var d domain.Driver
	var vehicleAttrsJSON []byte

	err := r.pool.QueryRow(ctx, query, driverID).Scan(
		&d.ID,
		&d.LicenseNumber,
		&d.VehicleType,
		&vehicleAttrsJSON,
		&d.Rating,
		&d.TotalRides,
		&d.TotalEarnings,
		&d.Status,
		&d.IsVerified,
		&d.CreatedAt,
		&d.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrDriverNotFound
		}
		return nil, fmt.Errorf("query driver: %w", err)
	}

	if len(vehicleAttrsJSON) > 0 {
		if err := json.Unmarshal(vehicleAttrsJSON, &d.VehicleAttrs); err != nil {
			return nil, fmt.Errorf("unmarshal vehicle_attrs: %w", err)
		}
	}

	return &d, nil
}

func (r *driverPgRepository) UpdateStatus(ctx context.Context, driverID string, status domain.DriverStatus) error {
	query := `
		UPDATE drivers
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := r.pool.Exec(ctx, query, status, driverID)
	if err != nil {
		return fmt.Errorf("update driver status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrDriverNotFound
	}

	return nil
}

func (r *driverPgRepository) UpdateRideStats(ctx context.Context, driverID string, ridesIncrement int, earningsIncrement float64) error {
	query := `
		UPDATE drivers
		SET total_rides = total_rides + $1,
		    total_earnings = total_earnings + $2,
		    updated_at = NOW()
		WHERE id = $3
	`

	result, err := r.pool.Exec(ctx, query, ridesIncrement, earningsIncrement, driverID)
	if err != nil {
		return fmt.Errorf("update driver stats: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrDriverNotFound
	}

	return nil
}

func (r *driverPgRepository) FindNearbyAvailableDrivers(
	ctx context.Context,
	lat, lng float64,
	radiusKm float64,
	vehicleType domain.VehicleType,
	limit int,
) ([]*out.DriverWithDistance, error) {
	// PostGIS запрос согласно ТЗ
	query := `
		SELECT d.id, d.license_number, d.vehicle_type, d.vehicle_attrs, d.rating, d.total_rides, d.total_earnings, d.status, d.is_verified, d.created_at, d.updated_at,
		       c.latitude, c.longitude,
		       ST_Distance(
		         ST_MakePoint(c.longitude, c.latitude)::geography,
		         ST_MakePoint($1, $2)::geography
		       ) / 1000 as distance_km
		FROM drivers d
		JOIN coordinates c ON c.entity_id = d.id
		  AND c.entity_type = 'driver'
		  AND c.is_current = true
		WHERE d.status = 'AVAILABLE'
		  AND d.vehicle_type = $3
		  AND d.is_verified = true
		  AND ST_DWithin(
		        ST_MakePoint(c.longitude, c.latitude)::geography,
		        ST_MakePoint($1, $2)::geography,
		        $4
		      )
		ORDER BY distance_km, d.rating DESC
		LIMIT $5
	`

	radiusMeters := radiusKm * 1000

	rows, err := r.pool.Query(ctx, query, lng, lat, vehicleType, radiusMeters, limit)
	if err != nil {
		return nil, fmt.Errorf("query nearby drivers: %w", err)
	}
	defer rows.Close()

	var results []*out.DriverWithDistance

	for rows.Next() {
		var d domain.Driver
		var vehicleAttrsJSON []byte
		var distanceKm float64
		var latitude, longitude float64

		if err := rows.Scan(
			&d.ID,
			&d.LicenseNumber,
			&d.VehicleType,
			&vehicleAttrsJSON,
			&d.Rating,
			&d.TotalRides,
			&d.TotalEarnings,
			&d.Status,
			&d.IsVerified,
			&d.CreatedAt,
			&d.UpdatedAt,
			&latitude,
			&longitude,
			&distanceKm,
		); err != nil {
			return nil, fmt.Errorf("scan driver row: %w", err)
		}

		if len(vehicleAttrsJSON) > 0 {
			if err := json.Unmarshal(vehicleAttrsJSON, &d.VehicleAttrs); err != nil {
				return nil, fmt.Errorf("unmarshal vehicle_attrs: %w", err)
			}
		}

		results = append(results, &out.DriverWithDistance{
			Driver:     &d,
			DistanceKm: distanceKm,
			Latitude:   latitude,
			Longitude:  longitude,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rows: %w", err)
	}

	return results, nil
}
