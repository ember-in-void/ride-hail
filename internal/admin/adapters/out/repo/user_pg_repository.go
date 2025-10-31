package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"ridehail/internal/admin/application/ports/in"
	"ridehail/internal/admin/application/ports/out"
	"ridehail/internal/admin/domain"
	"ridehail/internal/shared/logger"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserPgRepository — Postgres реализация UserRepository
type UserPgRepository struct {
	pool *pgxpool.Pool
	log  *logger.Logger
}

// NewUserPgRepository создает новый репозиторий пользователей
func NewUserPgRepository(pool *pgxpool.Pool, log *logger.Logger) *UserPgRepository {
	return &UserPgRepository{
		pool: pool,
		log:  log,
	}
}

// Create создает нового пользователя (и driver record если роль DRIVER)
func (r *UserPgRepository) Create(ctx context.Context, user *domain.User) error {
	attrsJSON, err := json.Marshal(user.Attrs)
	if err != nil {
		return fmt.Errorf("marshal attrs: %w", err)
	}

	// Начинаем транзакцию
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx) // Откатываем если не закоммитили
	}()

	// Создаем пользователя
	userQuery := `
		INSERT INTO users (id, email, role, status, password_hash, attrs, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err = tx.Exec(ctx, userQuery,
		user.ID,
		user.Email,
		user.Role,
		user.Status,
		user.PasswordHash,
		attrsJSON,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		// Проверяем на unique_violation (pgx error)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			// Проверяем, что именно email constraint
			if strings.Contains(pgErr.ConstraintName, "email") || strings.Contains(pgErr.Detail, "email") {
				return domain.ErrUserAlreadyExists
			}
		}
		return fmt.Errorf("insert user: %w", err)
	}

	// Если роль DRIVER, создаем запись в таблице drivers
	if user.Role == "DRIVER" {
		licenseNumber := "UNKNOWN"
		vehicleType := "ECONOMY"
		var vehicleAttrs map[string]interface{}

		// Извлекаем license_number и vehicle_type из attrs
		if user.Attrs != nil {
			if ln, ok := user.Attrs["license_number"].(string); ok && ln != "" {
				licenseNumber = ln
			}
			if vt, ok := user.Attrs["vehicle_type"].(string); ok && vt != "" {
				vehicleType = vt
			}
			// Собираем vehicle_attrs из остальных полей
			vehicleAttrs = make(map[string]interface{})
			for k, v := range user.Attrs {
				if strings.HasPrefix(k, "vehicle_") && k != "vehicle_type" {
					vehicleAttrs[strings.TrimPrefix(k, "vehicle_")] = v
				}
			}
		}

		vehicleAttrsJSON, err := json.Marshal(vehicleAttrs)
		if err != nil {
			return fmt.Errorf("marshal vehicle attrs: %w", err)
		}

		driverQuery := `
			INSERT INTO drivers (id, license_number, vehicle_type, vehicle_attrs, status, is_verified, created_at, updated_at)
			VALUES ($1, $2, $3, $4, 'OFFLINE', true, $5, $6)
		`

		_, err = tx.Exec(ctx, driverQuery,
			user.ID,
			licenseNumber,
			vehicleType,
			vehicleAttrsJSON,
			user.CreatedAt,
			user.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("insert driver: %w", err)
		}
	}

	// Коммитим транзакцию
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// FindByID находит пользователя по ID
func (r *UserPgRepository) FindByID(ctx context.Context, userID string) (*domain.User, error) {
	query := `
		SELECT id, email, role, status, password_hash, attrs, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user domain.User
	var attrsJSON []byte

	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&user.ID,
		&user.Email,
		&user.Role,
		&user.Status,
		&user.PasswordHash,
		&attrsJSON,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("query user by id: %w", err)
	}

	// Парсим JSONB attrs
	if len(attrsJSON) > 0 {
		if err := json.Unmarshal(attrsJSON, &user.Attrs); err != nil {
			r.log.Debug(logger.Entry{
				Action:  "unmarshal_user_attrs_failed",
				Message: err.Error(),
			})
			user.Attrs = make(map[string]interface{})
		}
	} else {
		user.Attrs = make(map[string]interface{})
	}

	return &user, nil
}

// FindByEmail находит пользователя по email
func (r *UserPgRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, email, role, status, password_hash, attrs, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user domain.User
	var attrsJSON []byte

	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.Role,
		&user.Status,
		&user.PasswordHash,
		&attrsJSON,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("query user by email: %w", err)
	}

	// Парсим JSONB attrs
	if len(attrsJSON) > 0 {
		if err := json.Unmarshal(attrsJSON, &user.Attrs); err != nil {
			r.log.Debug(logger.Entry{
				Action:  "unmarshal_user_attrs_failed",
				Message: err.Error(),
			})
			user.Attrs = make(map[string]interface{})
		}
	} else {
		user.Attrs = make(map[string]interface{})
	}

	return &user, nil
}

// List возвращает список пользователей с фильтрами
func (r *UserPgRepository) List(ctx context.Context, filters out.ListUsersFilters) ([]*domain.User, int, error) {
	// Строим динамический WHERE clause
	whereClause := ""
	args := []interface{}{}
	argIndex := 1

	if filters.Role != "" {
		whereClause += fmt.Sprintf(" AND role = $%d", argIndex)
		args = append(args, filters.Role)
		argIndex++
	}

	if filters.Status != "" {
		whereClause += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, filters.Status)
		argIndex++
	}

	// Запрос для подсчета общего количества
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM users
		WHERE 1=1 %s
	`, whereClause)

	var totalCount int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	// Запрос для получения пользователей
	args = append(args, filters.Limit, filters.Offset)
	query := fmt.Sprintf(`
		SELECT id, email, role, status, password_hash, attrs, created_at, updated_at
		FROM users
		WHERE 1=1 %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query users: %w", err)
	}
	defer rows.Close()

	users := make([]*domain.User, 0)
	for rows.Next() {
		var user domain.User
		var attrsJSON []byte

		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.Role,
			&user.Status,
			&user.PasswordHash,
			&attrsJSON,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan user row: %w", err)
		}

		// Парсим JSONB attrs
		if len(attrsJSON) > 0 {
			if err := json.Unmarshal(attrsJSON, &user.Attrs); err != nil {
				user.Attrs = make(map[string]interface{})
			}
		} else {
			user.Attrs = make(map[string]interface{})
		}

		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate users: %w", err)
	}

	return users, totalCount, nil
}

// Update обновляет пользователя
func (r *UserPgRepository) Update(ctx context.Context, user *domain.User) error {
	attrsJSON, err := json.Marshal(user.Attrs)
	if err != nil {
		return fmt.Errorf("marshal attrs: %w", err)
	}

	query := `
		UPDATE users
		SET email = $2, role = $3, status = $4, password_hash = $5, attrs = $6, updated_at = $7
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query,
		user.ID,
		user.Email,
		user.Role,
		user.Status,
		user.PasswordHash,
		attrsJSON,
		user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}

	return nil
}

// Delete удаляет пользователя (hard delete)
func (r *UserPgRepository) Delete(ctx context.Context, userID string) error {
	query := `DELETE FROM users WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}

	return nil
}

// Exists проверяет существование пользователя
func (r *UserPgRepository) Exists(ctx context.Context, userID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`

	var exists bool
	err := r.pool.QueryRow(ctx, query, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check user exists: %w", err)
	}

	return exists, nil
}

// GetSystemMetrics получает метрики системы для admin dashboard
func (r *UserPgRepository) GetSystemMetrics(ctx context.Context) (*in.SystemMetrics, error) {
	query := `
		WITH ride_stats AS (
			SELECT
				COUNT(*) FILTER (WHERE status IN ('MATCHED', 'EN_ROUTE', 'ARRIVED', 'IN_PROGRESS')) as active_rides,
				COUNT(*) FILTER (WHERE created_at >= CURRENT_DATE) as total_rides_today,
				COALESCE(SUM(final_fare) FILTER (WHERE created_at >= CURRENT_DATE AND status = 'COMPLETED'), 0) as revenue_today,
				COALESCE(AVG(EXTRACT(EPOCH FROM (matched_at - requested_at))/60) FILTER (WHERE matched_at IS NOT NULL), 0) as avg_wait_time,
				COALESCE(AVG(EXTRACT(EPOCH FROM (completed_at - started_at))/60) FILTER (WHERE completed_at IS NOT NULL AND started_at IS NOT NULL), 0) as avg_duration,
				COUNT(*) FILTER (WHERE created_at >= CURRENT_DATE AND status = 'CANCELLED')::float / NULLIF(COUNT(*) FILTER (WHERE created_at >= CURRENT_DATE), 0) as cancel_rate
			FROM rides
		),
		driver_stats AS (
			SELECT
				COUNT(*) FILTER (WHERE status = 'AVAILABLE') as available_drivers,
				COUNT(*) FILTER (WHERE status = 'BUSY') as busy_drivers
			FROM drivers
		)
		SELECT
			COALESCE(r.active_rides, 0),
			COALESCE(d.available_drivers, 0),
			COALESCE(d.busy_drivers, 0),
			COALESCE(r.total_rides_today, 0),
			COALESCE(r.revenue_today, 0),
			COALESCE(r.avg_wait_time, 0),
			COALESCE(r.avg_duration, 0),
			COALESCE(r.cancel_rate, 0)
		FROM ride_stats r
		CROSS JOIN driver_stats d
	`

	var metrics in.SystemMetrics
	err := r.pool.QueryRow(ctx, query).Scan(
		&metrics.ActiveRides,
		&metrics.AvailableDrivers,
		&metrics.BusyDrivers,
		&metrics.TotalRidesToday,
		&metrics.TotalRevenueToday,
		&metrics.AverageWaitTimeMinutes,
		&metrics.AverageRideDurationMinutes,
		&metrics.CancellationRate,
	)
	if err != nil {
		return nil, fmt.Errorf("get system metrics: %w", err)
	}

	return &metrics, nil
}

// GetDriverDistribution получает распределение водителей по типам транспорта
func (r *UserPgRepository) GetDriverDistribution(ctx context.Context) (map[string]int, error) {
	query := `
		SELECT
			vehicle_type,
			COUNT(*)
		FROM drivers
		WHERE status IN ('AVAILABLE', 'BUSY', 'EN_ROUTE')
		GROUP BY vehicle_type
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query driver distribution: %w", err)
	}
	defer rows.Close()

	distribution := make(map[string]int)
	for rows.Next() {
		var vehicleType string
		var count int
		if err := rows.Scan(&vehicleType, &count); err != nil {
			return nil, fmt.Errorf("scan driver distribution: %w", err)
		}
		distribution[vehicleType] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate driver distribution: %w", err)
	}

	return distribution, nil
}

// GetHotspots получает горячие точки (зоны повышенного спроса)
func (r *UserPgRepository) GetHotspots(ctx context.Context) ([]in.Hotspot, error) {
	// Упрощенная реализация - топ адресов pickup
	query := `
		WITH pickup_addresses AS (
			SELECT
				c.address,
				COUNT(*) FILTER (WHERE r.status IN ('MATCHED', 'EN_ROUTE', 'ARRIVED', 'IN_PROGRESS')) as active_rides
			FROM rides r
			JOIN coordinates c ON r.pickup_coordinate_id = c.id
			WHERE r.created_at >= CURRENT_DATE - INTERVAL '1 hour'
			GROUP BY c.address
			HAVING COUNT(*) FILTER (WHERE r.status IN ('MATCHED', 'EN_ROUTE', 'ARRIVED', 'IN_PROGRESS')) > 0
			ORDER BY active_rides DESC
			LIMIT 5
		)
		SELECT
			pa.address,
			pa.active_rides,
			0 as waiting_drivers
		FROM pickup_addresses pa
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query hotspots: %w", err)
	}
	defer rows.Close()

	var hotspots []in.Hotspot
	for rows.Next() {
		var h in.Hotspot
		if err := rows.Scan(&h.Location, &h.ActiveRides, &h.WaitingDrivers); err != nil {
			return nil, fmt.Errorf("scan hotspot: %w", err)
		}
		hotspots = append(hotspots, h)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate hotspots: %w", err)
	}

	return hotspots, nil
}

// GetActiveRides получает активные поездки с пагинацией
func (r *UserPgRepository) GetActiveRides(ctx context.Context, page, pageSize int) ([]in.ActiveRideInfo, int, error) {
	offset := (page - 1) * pageSize

	// Получаем общее количество
	countQuery := `
		SELECT COUNT(*)
		FROM rides
		WHERE status IN ('MATCHED', 'EN_ROUTE', 'ARRIVED', 'IN_PROGRESS')
	`

	var totalCount int
	if err := r.pool.QueryRow(ctx, countQuery).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("count active rides: %w", err)
	}

	// Получаем поездки
	query := `
		SELECT
			r.id,
			r.ride_number,
			r.status,
			r.passenger_id,
			r.driver_id,
			pickup_c.address,
			dest_c.address,
			r.started_at,
			r.started_at + INTERVAL '20 minutes' as estimated_completion,
			driver_c.latitude,
			driver_c.longitude
		FROM rides r
		JOIN coordinates pickup_c ON r.pickup_coordinate_id = pickup_c.id
		JOIN coordinates dest_c ON r.destination_coordinate_id = dest_c.id
		LEFT JOIN LATERAL (
			SELECT latitude, longitude
			FROM coordinates
			WHERE entity_id = r.driver_id AND entity_type = 'driver' AND is_current = true
			LIMIT 1
		) driver_c ON true
		WHERE r.status IN ('MATCHED', 'EN_ROUTE', 'ARRIVED', 'IN_PROGRESS')
		ORDER BY r.created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.pool.Query(ctx, query, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query active rides: %w", err)
	}
	defer rows.Close()

	var rides []in.ActiveRideInfo
	for rows.Next() {
		var ride in.ActiveRideInfo
		var driverLat, driverLon *float64

		if err := rows.Scan(
			&ride.RideID,
			&ride.RideNumber,
			&ride.Status,
			&ride.PassengerID,
			&ride.DriverID,
			&ride.PickupAddress,
			&ride.DestinationAddress,
			&ride.StartedAt,
			&ride.EstimatedCompletion,
			&driverLat,
			&driverLon,
		); err != nil {
			return nil, 0, fmt.Errorf("scan active ride: %w", err)
		}

		// Если есть локация водителя
		if driverLat != nil && driverLon != nil {
			ride.CurrentDriverLocation = &in.LocationInfo{
				Latitude:  *driverLat,
				Longitude: *driverLon,
			}
		}

		rides = append(rides, ride)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate active rides: %w", err)
	}

	return rides, totalCount, nil
}
