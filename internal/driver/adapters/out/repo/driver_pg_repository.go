package repo

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

// CreateSession создает новую сессию водителя
func (r *driverPgRepository) CreateSession(ctx context.Context, session *domain.DriverSession) error {
	query := `
		INSERT INTO driver_sessions (id, driver_id, started_at, total_rides, total_earnings)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.pool.Exec(ctx, query,
		session.ID,
		session.DriverID,
		session.StartedAt,
		session.TotalRides,
		session.TotalEarnings,
	)
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}

	return nil
}

// GetActiveSession получает активную сессию водителя
func (r *driverPgRepository) GetActiveSession(ctx context.Context, driverID string) (*domain.DriverSession, error) {
	query := `
		SELECT id, driver_id, started_at, ended_at, total_rides, total_earnings
		FROM driver_sessions
		WHERE driver_id = $1 AND ended_at IS NULL
		ORDER BY started_at DESC
		LIMIT 1
	`

	var s domain.DriverSession

	err := r.pool.QueryRow(ctx, query, driverID).Scan(
		&s.ID,
		&s.DriverID,
		&s.StartedAt,
		&s.EndedAt,
		&s.TotalRides,
		&s.TotalEarnings,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrSessionNotFound
		}
		return nil, fmt.Errorf("query session: %w", err)
	}

	return &s, nil
}

// EndSession завершает текущую сессию водителя
func (r *driverPgRepository) EndSession(ctx context.Context, sessionID string, totalRides int, totalEarnings float64) error {
	query := `
		UPDATE driver_sessions
		SET ended_at = NOW(),
		    total_rides = $1,
		    total_earnings = $2
		WHERE id = $3 AND ended_at IS NULL
	`

	result, err := r.pool.Exec(ctx, query, totalRides, totalEarnings, sessionID)
	if err != nil {
		return fmt.Errorf("end session: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrSessionNotFound
	}

	return nil
}

// FindNearbyAvailable находит доступных водителей рядом с точкой
func (r *driverPgRepository) FindNearbyAvailable(ctx context.Context, lat, lng float64, vehicleType domain.VehicleType, maxDistanceKm float64) ([]*out.NearbyDriverDTO, error) {
	// PostGIS запрос для поиска водителей в радиусе
	query := `
		SELECT d.id, u.email, d.rating, c.latitude, c.longitude,
		       ST_Distance(
		         ST_MakePoint(c.longitude, c.latitude)::geography,
		         ST_MakePoint($1, $2)::geography
		       ) / 1000 as distance_km,
		       d.vehicle_type, d.vehicle_attrs
		FROM drivers d
		JOIN users u ON u.id = d.id
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
		LIMIT 10
	`

	radiusMeters := maxDistanceKm * 1000

	rows, err := r.pool.Query(ctx, query, lng, lat, vehicleType, radiusMeters)
	if err != nil {
		return nil, fmt.Errorf("query nearby drivers: %w", err)
	}
	defer rows.Close()

	var results []*out.NearbyDriverDTO

	for rows.Next() {
		var dto out.NearbyDriverDTO
		var vehicleAttrsJSON []byte

		if err := rows.Scan(
			&dto.DriverID,
			&dto.Email,
			&dto.Rating,
			&dto.Latitude,
			&dto.Longitude,
			&dto.DistanceKm,
			&dto.VehicleType,
			&vehicleAttrsJSON,
		); err != nil {
			return nil, fmt.Errorf("scan driver row: %w", err)
		}

		if len(vehicleAttrsJSON) > 0 {
			if err := json.Unmarshal(vehicleAttrsJSON, &dto.Vehicle); err != nil {
				return nil, fmt.Errorf("unmarshal vehicle_attrs: %w", err)
			}
		}

		results = append(results, &dto)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rows: %w", err)
	}

	return results, nil
}
