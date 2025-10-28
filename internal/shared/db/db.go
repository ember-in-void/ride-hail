package db_conn

import (
	"context"
	"fmt"
	"time"

	"ridehail/internal/shared/config"
	"ridehail/internal/shared/logger"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool создает connection pool для БД с таймаутами и ограничениями
func NewPool(ctx context.Context, cfg config.DBConfig, log *logger.Logger) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("parse db config: %w", err)
	}

	// Production-ready настройки пула
	poolCfg.MaxConns = 20
	poolCfg.MinConns = 2
	poolCfg.MaxConnLifetime = time.Hour
	poolCfg.MaxConnIdleTime = 30 * time.Minute
	poolCfg.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	// Проверяем подключение
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	log.Info(logger.Entry{
		Action:  "db_connected",
		Message: fmt.Sprintf("connected to %s:%d/%s", cfg.Host, cfg.Port, cfg.Database),
	})

	return pool, nil
}

// Close безопасно закрывает пул с логированием
func Close(pool *pgxpool.Pool, log *logger.Logger) {
	if pool != nil {
		pool.Close()
		log.Info(logger.Entry{Action: "db_closed", Message: "database pool closed"})
	}
}
