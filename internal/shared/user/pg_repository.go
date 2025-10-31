package user

import (
	"context"
	"errors"
	"fmt"

	"ridehail/internal/admin/application/ports/in"
	"ridehail/internal/shared/logger"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgRepository — Postgres реализация Repository
type PgRepository struct {
	pool *pgxpool.Pool
	log  *logger.Logger
}

// NewPgRepository создает новый репозиторий пользователей
func NewPgRepository(pool *pgxpool.Pool, log *logger.Logger) *PgRepository {
	return &PgRepository{
		pool: pool,
		log:  log,
	}
}

// FindByID находит пользователя по ID
func (r *PgRepository) FindByID(ctx context.Context, userID string) (*User, error) {
	query := `
		SELECT id, email, role, status, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user User
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&user.ID,
		&user.Email,
		&user.Role,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("query user by id: %w", err)
	}

	return &user, nil
}

// Exists проверяет существование пользователя
func (r *PgRepository) Exists(ctx context.Context, userID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`

	var exists bool
	err := r.pool.QueryRow(ctx, query, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check user exists: %w", err)
	}

	return exists, nil
}

// GetSystemMetrics получает метрики системы для admin dashboard
func (r *PgRepository) GetSystemMetrics(ctx context.Context) (*in.SystemMetrics, error) {
	query := `
		WITH ride_stats AS (
			SELECT
				COUNT(*) FILTER (WHERE status IN ('MATCHED', 'EN_ROUTE', 'ARRIVED', 'IN_PROGRESS')) as active_rides,
				COUNT(*) FILTER (WHERE created_at >= CURRENT_DATE) as total_rides_today,
				COALESCE(SUM(final_fare) FILTER (WHERE created_at >= CURRENT_DATE AND status = 'COMPLETED'), 0) as revenue_today,
				COALESCE(AVG(EXTRACT(EPOCH FROM (matched_at - requested_at))/60) FILTER (WHERE matched_at IS NOT NULL), 0) as avg_wait_time,
				COALESCE(AVG(EXTRACT(EPOCH FROM (completed_at - started_at))/60) FILTER (WHERE completed_at IS NOT NULL AND started_at IS NOT NULL), 0) as avg_duration,
				COUNT(*) FILTER (WHERE created_at >= CURRENT_DATE AND status = 'CANCELLED')::float / NULLIF(COUNT(*) FILTER (WHERE created_at >= CURRENT_DATE), 0) as cancel_rate
			FROM rides
		),
		driver_stats AS (
			SELECT
				COUNT(*) FILTER (WHERE status = 'AVAILABLE') as available_drivers,
				COUNT(*) FILTER (WHERE status = 'BUSY') as busy_drivers
			FROM drivers
		)
		SELECT
			COALESCE(r.active_rides, 0),
			COALESCE(d.available_drivers, 0),
			COALESCE(d.busy_drivers, 0),
			COALESCE(r.total_rides_today, 0),
			COALESCE(r.revenue_today, 0),
			COALESCE(r.avg_wait_time, 0),
			COALESCE(r.avg_duration, 0),
			COALESCE(r.cancel_rate, 0)
		FROM ride_stats r
		CROSS JOIN driver_stats d
	`

	var metrics in.SystemMetrics
	err := r.pool.QueryRow(ctx, query).Scan(
		&metrics.ActiveRides,
		&metrics.AvailableDrivers,
		&metrics.BusyDrivers,
		&metrics.TotalRidesToday,
		&metrics.TotalRevenueToday,
		&metrics.AverageWaitTimeMinutes,
		&metrics.AverageRideDurationMinutes,
		&metrics.CancellationRate,
	)
	if err != nil {
		return nil, fmt.Errorf("get system metrics: %w", err)
	}

	return &metrics, nil
}

// GetDriverDistribution получает распределение водителей по типам транспорта
func (r *PgRepository) GetDriverDistribution(ctx context.Context) (map[string]int, error) {
	query := `
		SELECT
			vehicle_type,
			COUNT(*)
		FROM drivers
		WHERE is_online = true
		GROUP BY vehicle_type
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query driver distribution: %w", err)
	}
	defer rows.Close()

	distribution := make(map[string]int)
	for rows.Next() {
		var vehicleType string
		var count int
		if err := rows.Scan(&vehicleType, &count); err != nil {
			return nil, fmt.Errorf("scan driver distribution: %w", err)
		}
		distribution[vehicleType] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate driver distribution: %w", err)
	}

	return distribution, nil
}

// GetHotspots получает горячие точки (зоны повышенного спроса)
func (r *PgRepository) GetHotspots(ctx context.Context) ([]in.Hotspot, error) {
	// Упрощенная реализация - топ адресов pickup
	query := `
		WITH pickup_addresses AS (
			SELECT
				c.address,
				COUNT(*) FILTER (WHERE r.status IN ('MATCHED', 'EN_ROUTE', 'ARRIVED', 'IN_PROGRESS')) as active_rides
			FROM rides r
			JOIN coordinates c ON r.pickup_coordinate_id = c.id
			WHERE r.created_at >= CURRENT_DATE - INTERVAL '1 hour'
			GROUP BY c.address
			HAVING COUNT(*) FILTER (WHERE r.status IN ('MATCHED', 'EN_ROUTE', 'ARRIVED', 'IN_PROGRESS')) > 0
			ORDER BY active_rides DESC
			LIMIT 5
		)
		SELECT
			pa.address,
			pa.active_rides,
			0 as waiting_drivers
		FROM pickup_addresses pa
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query hotspots: %w", err)
	}
	defer rows.Close()

	var hotspots []in.Hotspot
	for rows.Next() {
		var h in.Hotspot
		if err := rows.Scan(&h.Location, &h.ActiveRides, &h.WaitingDrivers); err != nil {
			return nil, fmt.Errorf("scan hotspot: %w", err)
		}
		hotspots = append(hotspots, h)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate hotspots: %w", err)
	}

	return hotspots, nil
}

// GetActiveRides получает активные поездки с пагинацией
func (r *PgRepository) GetActiveRides(ctx context.Context, page, pageSize int) ([]in.ActiveRideInfo, int, error) {
	offset := (page - 1) * pageSize

	// Получаем общее количество
	countQuery := `
		SELECT COUNT(*)
		FROM rides
		WHERE status IN ('MATCHED', 'EN_ROUTE', 'ARRIVED', 'IN_PROGRESS')
	`

	var totalCount int
	if err := r.pool.QueryRow(ctx, countQuery).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("count active rides: %w", err)
	}

	// Получаем поездки
	query := `
		SELECT
			r.id,
			r.ride_number,
			r.status,
			r.passenger_id,
			r.driver_id,
			pickup_c.address,
			dest_c.address,
			r.started_at,
			r.started_at + INTERVAL '20 minutes' as estimated_completion,
			driver_c.latitude,
			driver_c.longitude
		FROM rides r
		JOIN coordinates pickup_c ON r.pickup_coordinate_id = pickup_c.id
		JOIN coordinates dest_c ON r.destination_coordinate_id = dest_c.id
		LEFT JOIN LATERAL (
			SELECT latitude, longitude
			FROM coordinates
			WHERE entity_id = r.driver_id AND entity_type = 'driver' AND is_current = true
			LIMIT 1
		) driver_c ON true
		WHERE r.status IN ('MATCHED', 'EN_ROUTE', 'ARRIVED', 'IN_PROGRESS')
		ORDER BY r.created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.pool.Query(ctx, query, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query active rides: %w", err)
	}
	defer rows.Close()

	var rides []in.ActiveRideInfo
	for rows.Next() {
		var ride in.ActiveRideInfo
		var driverLat, driverLon *float64

		if err := rows.Scan(
			&ride.RideID,
			&ride.RideNumber,
			&ride.Status,
			&ride.PassengerID,
			&ride.DriverID,
			&ride.PickupAddress,
			&ride.DestinationAddress,
			&ride.StartedAt,
			&ride.EstimatedCompletion,
			&driverLat,
			&driverLon,
		); err != nil {
			return nil, 0, fmt.Errorf("scan active ride: %w", err)
		}

		// Если есть локация водителя
		if driverLat != nil && driverLon != nil {
			ride.CurrentDriverLocation = &in.LocationInfo{
				Latitude:  *driverLat,
				Longitude: *driverLon,
			}
		}

		rides = append(rides, ride)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate active rides: %w", err)
	}

	return rides, totalCount, nil
}
