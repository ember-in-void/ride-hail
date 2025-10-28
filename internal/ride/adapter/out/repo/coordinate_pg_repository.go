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

// CoordinatePgRepository — PostgreSQL репозиторий для работы с координатами
type CoordinatePgRepository struct {
	pool *pgxpool.Pool
	log  *logger.Logger
}

// NewCoordinatePgRepository создает новый экземпляр репозитория
func NewCoordinatePgRepository(pool *pgxpool.Pool, log *logger.Logger) *CoordinatePgRepository {
	return &CoordinatePgRepository{
		pool: pool,
		log:  log,
	}
}

// Create создает новую координату
func (r *CoordinatePgRepository) Create(ctx context.Context, coord *domain.Coordinate) error {
	query := `
		INSERT INTO coordinates (
			id, entity_id, entity_type, address, latitude, longitude,
			fare_amount, distance_km, duration_minutes, is_current,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
	`

	_, err := r.pool.Exec(ctx, query,
		coord.ID,
		coord.EntityID,
		coord.EntityType,
		coord.Address,
		coord.Latitude,
		coord.Longitude,
		coord.FareAmount,
		coord.DistanceKm,
		coord.DurationMinutes,
		coord.IsCurrent,
		coord.CreatedAt,
		coord.UpdatedAt,
	)
	if err != nil {
		r.log.Error(logger.Entry{
			Action:  "db_create_coordinate_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		return fmt.Errorf("insert coordinate: %w", err)
	}

	return nil
}

// FindByID возвращает координату по ID
func (r *CoordinatePgRepository) FindByID(ctx context.Context, coordID string) (*domain.Coordinate, error) {
	query := `
		SELECT 
			id, entity_id, entity_type, address, latitude, longitude,
			fare_amount, distance_km, duration_minutes, is_current,
			created_at, updated_at
		FROM coordinates
		WHERE id = $1
	`

	coord := &domain.Coordinate{}
	err := r.pool.QueryRow(ctx, query, coordID).Scan(
		&coord.ID,
		&coord.EntityID,
		&coord.EntityType,
		&coord.Address,
		&coord.Latitude,
		&coord.Longitude,
		&coord.FareAmount,
		&coord.DistanceKm,
		&coord.DurationMinutes,
		&coord.IsCurrent,
		&coord.CreatedAt,
		&coord.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("coordinate not found")
		}
		return nil, fmt.Errorf("query coordinate: %w", err)
	}

	return coord, nil
}

// FindCurrentByEntity возвращает текущую координату для entity
func (r *CoordinatePgRepository) FindCurrentByEntity(ctx context.Context, entityID, entityType string) (*domain.Coordinate, error) {
	query := `
		SELECT 
			id, entity_id, entity_type, address, latitude, longitude,
			fare_amount, distance_km, duration_minutes, is_current,
			created_at, updated_at
		FROM coordinates
		WHERE entity_id = $1 AND entity_type = $2 AND is_current = true
		ORDER BY created_at DESC
		LIMIT 1
	`

	coord := &domain.Coordinate{}
	err := r.pool.QueryRow(ctx, query, entityID, entityType).Scan(
		&coord.ID,
		&coord.EntityID,
		&coord.EntityType,
		&coord.Address,
		&coord.Latitude,
		&coord.Longitude,
		&coord.FareAmount,
		&coord.DistanceKm,
		&coord.DurationMinutes,
		&coord.IsCurrent,
		&coord.CreatedAt,
		&coord.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("current coordinate not found")
		}
		return nil, fmt.Errorf("query current coordinate: %w", err)
	}

	return coord, nil
}

// MarkAsNotCurrent помечает координаты entity как не текущие
func (r *CoordinatePgRepository) MarkAsNotCurrent(ctx context.Context, entityID, entityType string) error {
	query := `
		UPDATE coordinates 
		SET is_current = false, updated_at = NOW()
		WHERE entity_id = $1 AND entity_type = $2 AND is_current = true
	`

	_, err := r.pool.Exec(ctx, query, entityID, entityType)
	if err != nil {
		r.log.Error(logger.Entry{
			Action:  "db_mark_coordinates_not_current_failed",
			Message: err.Error(),
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		return fmt.Errorf("mark coordinates as not current: %w", err)
	}

	return nil
}
