package db_conn

import (
	"database/sql"
	"fmt"

	"ridehail/internal/shared/config"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func OpenDB(cfg config.DBConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	return db, nil
}
