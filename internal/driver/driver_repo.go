package driver

import "database/sql"

type DriverRepo struct {
	db *sql.DB
}

func NewDriverRepo(db *sql.DB) *DriverRepo {
	return &DriverRepo{db: db}
}

func (r *DriverRepo) Close() error {
	return r.db.Close()
}

type DriverRepoInterface interface{}
