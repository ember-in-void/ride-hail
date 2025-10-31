package repo

import (
	"context"
	"fmt"
	"time"

	"ridehail/internal/driver/application/ports/out"
	"ridehail/internal/driver/domain"
	"ridehail/internal/shared/utils"

	"github.com/jackc/pgx/v5/pgxpool"
)

// LocationRepository реализует out.LocationRepository для PostgreSQL
type LocationRepository struct {
	db *pgxpool.Pool
}

// NewLocationRepository создает новый Postgres репозиторий для локаций
func NewLocationRepository(db *pgxpool.Pool) *LocationRepository {
	return &LocationRepository{db: db}
}

// CreateCoordinate создает новую запись координат
func (r *LocationRepository) CreateCoordinate(ctx context.Context, coord *out.CreateCoordinateDTO) (string, error) {
	coordinateID := utils.NewUUID()

	query := `
		INSERT INTO coordinates (
			id, entity_id, entity_type, address, latitude, longitude,
			fare_amount, distance_km, duration_minutes, is_current
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.db.Exec(ctx, query,
		coordinateID,
		coord.EntityID,
		coord.EntityType,
		coord.Address,
		coord.Latitude,
		coord.Longitude,
		coord.FareAmount,
		coord.DistanceKm,
		coord.DurationMinutes,
		coord.IsCurrent,
	)
	if err != nil {
		return "", fmt.Errorf("insert coordinate: %w", err)
	}

	return coordinateID, nil
}

// UpdateCurrentLocation обновляет текущую локацию (устанавливает is_current=false для старых)
func (r *LocationRepository) UpdateCurrentLocation(ctx context.Context, entityID, entityType string, lat, lng float64) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Устанавливаем is_current=false для всех текущих координат
	_, err = tx.Exec(ctx, `
		UPDATE coordinates
		SET is_current = false, updated_at = now()
		WHERE entity_id = $1 AND entity_type = $2 AND is_current = true
	`, entityID, entityType)
	if err != nil {
		return "", fmt.Errorf("update old coordinates: %w", err)
	}

	// Создаем новую координату с is_current=true
	coordinateID := utils.NewUUID()
	_, err = tx.Exec(ctx, `
		INSERT INTO coordinates (
			id, entity_id, entity_type, address, latitude, longitude, is_current
		)
		VALUES ($1, $2, $3, $4, $5, $6, true)
	`, coordinateID, entityID, entityType, "Current Location", lat, lng)
	if err != nil {
		return "", fmt.Errorf("insert new coordinate: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit transaction: %w", err)
	}

	return coordinateID, nil
}

// GetCurrentLocation получает текущую локацию сущности
func (r *LocationRepository) GetCurrentLocation(ctx context.Context, entityID, entityType string) (*domain.DriverLocation, error) {
	query := `
		SELECT entity_id, latitude, longitude, updated_at
		FROM coordinates
		WHERE entity_id = $1 AND entity_type = $2 AND is_current = true
		LIMIT 1
	`

	var loc domain.DriverLocation
	err := r.db.QueryRow(ctx, query, entityID, entityType).Scan(
		&loc.DriverID,
		&loc.Latitude,
		&loc.Longitude,
		&loc.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("query current location: %w", err)
	}

	return &loc, nil
}

// ArchiveToHistory архивирует координаты в location_history
func (r *LocationRepository) ArchiveToHistory(ctx context.Context, history *out.LocationHistoryDTO) error {
	query := `
		INSERT INTO location_history (
			coordinate_id, driver_id, latitude, longitude,
			accuracy_meters, speed_kmh, heading_degrees, ride_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.Exec(ctx, query,
		history.CoordinateID,
		history.DriverID,
		history.Latitude,
		history.Longitude,
		history.AccuracyMeters,
		history.SpeedKmh,
		history.HeadingDegrees,
		history.RideID,
	)
	if err != nil {
		return fmt.Errorf("insert location history: %w", err)
	}

	return nil
}

// CheckRateLimit проверяет, можно ли обновить локацию (макс 1 раз в 3 сек)
func (r *LocationRepository) CheckRateLimit(ctx context.Context, driverID string) (bool, error) {
	query := `
		SELECT updated_at
		FROM coordinates
		WHERE entity_id = $1 AND entity_type = 'driver' AND is_current = true
		LIMIT 1
	`

	var lastUpdate time.Time
	err := r.db.QueryRow(ctx, query, driverID).Scan(&lastUpdate)
	if err != nil {
		// Если записи нет — разрешаем обновление
		return true, nil
	}

	// Проверяем, прошло ли 3 секунды с последнего обновления
	elapsed := time.Since(lastUpdate)
	return elapsed >= 3*time.Second, nil
}

// NearbyDriverInfo информация о ближайшем водителе
type NearbyDriverInfo struct {
	DriverID string
	Distance float64 // в метрах
}

// FindNearbyOnlineDrivers находит ближайших онлайн водителей в радиусе
func (r *LocationRepository) FindNearbyOnlineDrivers(
	ctx context.Context,
	pickupLat, pickupLng float64,
	radiusKm float64,
	limit int,
) ([]NearbyDriverInfo, error) {
	query := `
		SELECT 
			d.id as driver_id,
			ST_Distance(
				ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
				ST_SetSRID(ST_MakePoint(c.longitude, c.latitude), 4326)::geography
			) as distance
		FROM drivers d
		INNER JOIN coordinates c ON c.entity_id = d.id AND c.entity_type = 'driver' AND c.is_current = true
		WHERE d.is_online = true
		  AND d.current_status = 'available'
		  AND ST_DWithin(
				ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
				ST_SetSRID(ST_MakePoint(c.longitude, c.latitude), 4326)::geography,
				$3
			)
		ORDER BY distance ASC, d.rating DESC
		LIMIT $4
	`

	rows, err := r.db.Query(ctx, query, pickupLng, pickupLat, radiusKm*1000, limit)
	if err != nil {
		return nil, fmt.Errorf("query nearby drivers: %w", err)
	}
	defer rows.Close()

	var drivers []NearbyDriverInfo
	for rows.Next() {
		var driver NearbyDriverInfo
		if err := rows.Scan(&driver.DriverID, &driver.Distance); err != nil {
			return nil, fmt.Errorf("scan driver: %w", err)
		}
		drivers = append(drivers, driver)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return drivers, nil
}
