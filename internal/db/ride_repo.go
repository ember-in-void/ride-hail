package db

import "database/sql"

type RideRepo struct {
	db *sql.DB
}

func NewRideRepo(db *sql.DB) *RideRepo {
	return &RideRepo{db: db}
}

func (r *RideRepo) Close() error {
	return r.db.Close()
}
