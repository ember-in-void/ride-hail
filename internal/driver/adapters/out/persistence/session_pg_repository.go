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

type sessionPgRepository struct {
	pool *pgxpool.Pool
}

func NewSessionPgRepository(pool *pgxpool.Pool) out.SessionRepository {
	return &sessionPgRepository{pool: pool}
}

func (r *sessionPgRepository) Create(ctx context.Context, session *domain.DriverSession) error {
	session.ID = uuid.New().String()

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

func (r *sessionPgRepository) FindActiveByDriverID(ctx context.Context, driverID string) (*domain.DriverSession, error) {
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

func (r *sessionPgRepository) Close(ctx context.Context, sessionID string) (*domain.DriverSession, error) {
	query := `
		UPDATE driver_sessions
		SET ended_at = $1
		WHERE id = $2 AND ended_at IS NULL
		RETURNING id, driver_id, started_at, ended_at, total_rides, total_earnings
	`

	now := time.Now().UTC()
	var s domain.DriverSession

	err := r.pool.QueryRow(ctx, query, now, sessionID).Scan(
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
		return nil, fmt.Errorf("close session: %w", err)
	}

	return &s, nil
}

func (r *sessionPgRepository) UpdateStats(ctx context.Context, sessionID string, ridesIncrement int, earningsIncrement float64) error {
	query := `
		UPDATE driver_sessions
		SET total_rides = total_rides + $1,
		    total_earnings = total_earnings + $2
		WHERE id = $3
	`

	result, err := r.pool.Exec(ctx, query, ridesIncrement, earningsIncrement, sessionID)
	if err != nil {
		return fmt.Errorf("update session stats: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrSessionNotFound
	}

	return nil
}
