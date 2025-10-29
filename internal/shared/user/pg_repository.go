package user

import (
	"context"
	"errors"
	"fmt"

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
