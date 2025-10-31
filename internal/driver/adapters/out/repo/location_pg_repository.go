package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	out "ridehail/internal/driver/application/ports/out"
	"ridehail/internal/driver/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type locationPgRepository struct {
	pool *pgxpool.Pool
}

func NewLocationPgRepository(pool *pgxpool.Pool) out.LocationRepository {
	return &locationPgRepository{pool: pool}
}

func (r *locationPgRepository) SaveCoordinate(ctx context.Context, coord *domain.Coordinates) error {
	// Сначала обновляем предыдущие координаты (is_current = false)
	updateQuery := `
		UPDATE coordinates
		SET is_current = false
		WHERE entity_id = $1 AND entity_type = $2 AND is_current = true
	`

	_, err := r.pool.Exec(ctx, updateQuery, coord.EntityID, coord.EntityType)
	if err != nil {
		return fmt.Errorf("update previous coordinates: %w", err)
	}

	// Вставляем новую координату
	coord.ID = uuid.New().String()

	insertQuery := `
		INSERT INTO coordinates (id, entity_id, entity_type, address, latitude, longitude, is_current, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err = r.pool.Exec(ctx, insertQuery,
		coord.ID,
		coord.EntityID,
		coord.EntityType,
		coord.Address,
		coord.Latitude,
		coord.Longitude,
		coord.IsCurrent,
		coord.CreatedAt,
		coord.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert coordinate: %w", err)
	}

	return nil
}

func (r *locationPgRepository) GetCurrentDriverLocation(ctx context.Context, driverID string) (*domain.Coordinates, error) {
	query := `
		SELECT id, entity_id, entity_type, address, latitude, longitude, is_current, created_at, updated_at
		FROM coordinates
		WHERE entity_id = $1 AND entity_type = 'driver' AND is_current = true
		LIMIT 1
	`

	var c domain.Coordinates

	err := r.pool.QueryRow(ctx, query, driverID).Scan(
		&c.ID,
		&c.EntityID,
		&c.EntityType,
		&c.Address,
		&c.Latitude,
		&c.Longitude,
		&c.IsCurrent,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("current location not found")
		}
		return nil, fmt.Errorf("query current location: %w", err)
	}

	return &c, nil
}

func (r *locationPgRepository) SaveLocationHistory(
	ctx context.Context,
	driverID string,
	lat, lng float64,
	accuracy, speed, heading float64,
	rideID *string,
) error {
	query := `
		INSERT INTO location_history (id, driver_id, latitude, longitude, accuracy_meters, speed_kmh, heading_degrees, recorded_at, ride_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	id := uuid.New().String()
	recordedAt := time.Now().UTC()

	_, err := r.pool.Exec(ctx, query,
		id,
		driverID,
		lat,
		lng,
		accuracy,
		speed,
		heading,
		recordedAt,
		rideID,
	)
	if err != nil {
		return fmt.Errorf("insert location history: %w", err)
	}

	return nil
}

func (r *locationPgRepository) GetLastLocationUpdateTime(ctx context.Context, driverID string) (*string, error) {
	query := `
		SELECT updated_at
		FROM coordinates
		WHERE entity_id = $1 AND entity_type = 'driver' AND is_current = true
		LIMIT 1
	`

	var updatedAt time.Time

	err := r.pool.QueryRow(ctx, query, driverID).Scan(&updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query last update time: %w", err)
	}

	timeStr := updatedAt.Format(time.RFC3339)
	return &timeStr, nil
}
