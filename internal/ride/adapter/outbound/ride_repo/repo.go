package ride_repo

import (
	"database/sql"

	"ridehail/internal/logger"
)

type RideRepo struct {
	db     *sql.DB
	logger *logger.Logger
}

func NewRideRepo(db *sql.DB, logger *logger.Logger) *RideRepo {
	return &RideRepo{
		db:     db,
		logger: logger,
	}
}

func (r *RideRepo) Close() error {
	return r.db.Close()
}
