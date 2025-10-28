package db_conn

import (
	"context"
	"embed"
	"fmt"
	"sort"

	"ridehail/internal/shared/logger"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var MigrationsFS embed.FS

// Migrate применяет все *.sql файлы из migrations/ в лексикографическом порядке
// Каждый файл выполняется в своей транзакции
// SQL-файлы НЕ должны содержать BEGIN/COMMIT
func Migrate(ctx context.Context, pool *pgxpool.Pool, log *logger.Logger) error {
	entries, err := MigrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)

	log.Info(logger.Entry{
		Action:  "migration_start",
		Message: fmt.Sprintf("applying %d migrations", len(names)),
	})

	for _, name := range names {
		sqlb, err := MigrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin tx for %s: %w", name, err)
		}

		if _, err := tx.Exec(ctx, string(sqlb)); err != nil {
			_ = tx.Rollback(ctx)
			log.Error(logger.Entry{
				Action:  "migration_failed",
				Message: name,
				Error:   &logger.ErrObj{Msg: err.Error()},
			})
			return fmt.Errorf("migration %s failed: %w", name, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit %s failed: %w", name, err)
		}

		log.Debug(logger.Entry{
			Action:  "migration_applied",
			Message: name,
		})
	}

	log.Info(logger.Entry{
		Action:  "migration_complete",
		Message: fmt.Sprintf("all %d migrations applied", len(names)),
	})

	return nil
}
