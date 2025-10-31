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

// RidePgRepository — PostgreSQL репозиторий для работы с поездками
type RidePgRepository struct {
	pool *pgxpool.Pool
	log  *logger.Logger
}

// NewRidePgRepository создает новый экземпляр репозитория
func NewRidePgRepository(pool *pgxpool.Pool, log *logger.Logger) *RidePgRepository {
	return &RidePgRepository{
		pool: pool,
		log:  log,
	}
}

// Create создает новую поездку
func (r *RidePgRepository) Create(ctx context.Context, ride *domain.Ride) error {
	query := `
		INSERT INTO rides (
			id, ride_number, passenger_id, driver_id, vehicle_type, status, priority,
			requested_at, matched_at, arrived_at, started_at, completed_at, cancelled_at,
			cancellation_reason, estimated_fare, final_fare,
			pickup_coordinate_id, destination_coordinate_id,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20
		)
	`

	_, err := r.pool.Exec(ctx, query,
		ride.ID,
		ride.RideNumber,
		ride.PassengerID,
		ride.DriverID,
		ride.VehicleType,
		ride.Status,
		ride.Priority,
		ride.RequestedAt,
		ride.MatchedAt,
		ride.ArrivedAt,
		ride.StartedAt,
		ride.CompletedAt,
		ride.CancelledAt,
		ride.CancellationReason,
		ride.EstimatedFare,
		ride.FinalFare,
		ride.PickupCoordinateID,
		ride.DestinationCoordinateID,
		ride.CreatedAt,
		ride.UpdatedAt,
	)
	if err != nil {
		r.log.Error(logger.Entry{
			Action:  "db_create_ride_failed",
			Message: err.Error(),
			RideID:  ride.ID,
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		return fmt.Errorf("insert ride: %w", err)
	}

	return nil
}

// FindByID возвращает поездку по ID
func (r *RidePgRepository) FindByID(ctx context.Context, rideID string) (*domain.Ride, error) {
	query := `
		SELECT 
			id, ride_number, passenger_id, driver_id, vehicle_type, status, priority,
			requested_at, matched_at, arrived_at, started_at, completed_at, cancelled_at,
			cancellation_reason, estimated_fare, final_fare,
			pickup_coordinate_id, destination_coordinate_id,
			created_at, updated_at
		FROM rides
		WHERE id = $1
	`

	ride := &domain.Ride{}
	err := r.pool.QueryRow(ctx, query, rideID).Scan(
		&ride.ID,
		&ride.RideNumber,
		&ride.PassengerID,
		&ride.DriverID,
		&ride.VehicleType,
		&ride.Status,
		&ride.Priority,
		&ride.RequestedAt,
		&ride.MatchedAt,
		&ride.ArrivedAt,
		&ride.StartedAt,
		&ride.CompletedAt,
		&ride.CancelledAt,
		&ride.CancellationReason,
		&ride.EstimatedFare,
		&ride.FinalFare,
		&ride.PickupCoordinateID,
		&ride.DestinationCoordinateID,
		&ride.CreatedAt,
		&ride.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrRideNotFound
		}
		r.log.Error(logger.Entry{
			Action:  "db_find_ride_by_id_failed",
			Message: err.Error(),
			RideID:  rideID,
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		return nil, fmt.Errorf("query ride by id: %w", err)
	}

	return ride, nil
}

// FindByRideNumber возвращает поездку по номеру
func (r *RidePgRepository) FindByRideNumber(ctx context.Context, rideNumber string) (*domain.Ride, error) {
	query := `
		SELECT 
			id, ride_number, passenger_id, driver_id, vehicle_type, status, priority,
			requested_at, matched_at, arrived_at, started_at, completed_at, cancelled_at,
			cancellation_reason, estimated_fare, final_fare,
			pickup_coordinate_id, destination_coordinate_id,
			created_at, updated_at
		FROM rides
		WHERE ride_number = $1
	`

	ride := &domain.Ride{}
	err := r.pool.QueryRow(ctx, query, rideNumber).Scan(
		&ride.ID,
		&ride.RideNumber,
		&ride.PassengerID,
		&ride.DriverID,
		&ride.VehicleType,
		&ride.Status,
		&ride.Priority,
		&ride.RequestedAt,
		&ride.MatchedAt,
		&ride.ArrivedAt,
		&ride.StartedAt,
		&ride.CompletedAt,
		&ride.CancelledAt,
		&ride.CancellationReason,
		&ride.EstimatedFare,
		&ride.FinalFare,
		&ride.PickupCoordinateID,
		&ride.DestinationCoordinateID,
		&ride.CreatedAt,
		&ride.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrRideNotFound
		}
		return nil, fmt.Errorf("query ride by number: %w", err)
	}

	return ride, nil
}

// Update обновляет существующую поездку
func (r *RidePgRepository) Update(ctx context.Context, ride *domain.Ride) error {
	query := `
		UPDATE rides SET
			driver_id = $2,
			status = $3,
			matched_at = $4,
			arrived_at = $5,
			started_at = $6,
			completed_at = $7,
			cancelled_at = $8,
			cancellation_reason = $9,
			final_fare = $10,
			updated_at = $11
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query,
		ride.ID,
		ride.DriverID,
		ride.Status,
		ride.MatchedAt,
		ride.ArrivedAt,
		ride.StartedAt,
		ride.CompletedAt,
		ride.CancelledAt,
		ride.CancellationReason,
		ride.FinalFare,
		ride.UpdatedAt,
	)
	if err != nil {
		r.log.Error(logger.Entry{
			Action:  "db_update_ride_failed",
			Message: err.Error(),
			RideID:  ride.ID,
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		return fmt.Errorf("update ride: %w", err)
	}

	return nil
}

// FindActiveByPassengerID возвращает активные поездки пассажира
func (r *RidePgRepository) FindActiveByPassengerID(ctx context.Context, passengerID string) ([]*domain.Ride, error) {
	query := `
		SELECT 
			id, ride_number, passenger_id, driver_id, vehicle_type, status, priority,
			requested_at, matched_at, arrived_at, started_at, completed_at, cancelled_at,
			cancellation_reason, estimated_fare, final_fare,
			pickup_coordinate_id, destination_coordinate_id,
			created_at, updated_at
		FROM rides
		WHERE passenger_id = $1 
		  AND status NOT IN ('COMPLETED', 'CANCELLED')
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, passengerID)
	if err != nil {
		return nil, fmt.Errorf("query active rides: %w", err)
	}
	defer rows.Close()

	var rides []*domain.Ride
	for rows.Next() {
		ride := &domain.Ride{}
		err := rows.Scan(
			&ride.ID,
			&ride.RideNumber,
			&ride.PassengerID,
			&ride.DriverID,
			&ride.VehicleType,
			&ride.Status,
			&ride.Priority,
			&ride.RequestedAt,
			&ride.MatchedAt,
			&ride.ArrivedAt,
			&ride.StartedAt,
			&ride.CompletedAt,
			&ride.CancelledAt,
			&ride.CancellationReason,
			&ride.EstimatedFare,
			&ride.FinalFare,
			&ride.PickupCoordinateID,
			&ride.DestinationCoordinateID,
			&ride.CreatedAt,
			&ride.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan ride: %w", err)
		}
		rides = append(rides, ride)
	}

	return rides, rows.Err()
}

// FindByStatus возвращает поездки с определенным статусом
func (r *RidePgRepository) FindByStatus(ctx context.Context, status string, limit int) ([]*domain.Ride, error) {
	query := `
		SELECT 
			id, ride_number, passenger_id, driver_id, vehicle_type, status, priority,
			requested_at, matched_at, arrived_at, started_at, completed_at, cancelled_at,
			cancellation_reason, estimated_fare, final_fare,
			pickup_coordinate_id, destination_coordinate_id,
			created_at, updated_at
		FROM rides
		WHERE status = $1
		ORDER BY priority DESC, created_at ASC
		LIMIT $2
	`

	rows, err := r.pool.Query(ctx, query, status, limit)
	if err != nil {
		return nil, fmt.Errorf("query rides by status: %w", err)
	}
	defer rows.Close()

	var rides []*domain.Ride
	for rows.Next() {
		ride := &domain.Ride{}
		err := rows.Scan(
			&ride.ID,
			&ride.RideNumber,
			&ride.PassengerID,
			&ride.DriverID,
			&ride.VehicleType,
			&ride.Status,
			&ride.Priority,
			&ride.RequestedAt,
			&ride.MatchedAt,
			&ride.ArrivedAt,
			&ride.StartedAt,
			&ride.CompletedAt,
			&ride.CancelledAt,
			&ride.CancellationReason,
			&ride.EstimatedFare,
			&ride.FinalFare,
			&ride.PickupCoordinateID,
			&ride.DestinationCoordinateID,
			&ride.CreatedAt,
			&ride.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan ride: %w", err)
		}
		rides = append(rides, ride)
	}

	return rides, rows.Err()
}

// AssignDriver назначает водителя на поездку и обновляет статус на DRIVER_ASSIGNED
// AssignDriver — атомарно назначает водителя на поездку.
//
// БИЗНЕС-ЛОГИКА:
// - Обновляет driver_id в таблице rides
// - Меняет статус REQUESTED → MATCHED
// - Проставляет timestamp matched_at
//
// ЗАЩИТА ОТ RACE CONDITIONS:
// SQL включает WHERE status='REQUESTED' — это критично!
// Если два водителя одновременно примут поездку, только один UPDATE сработает.
// Второй вернет RowsAffected=0 и получит ошибку.
//
// ПРИМЕР СЦЕНАРИЯ:
// 1. Passenger создает ride → status='REQUESTED', driver_id=NULL
// 2. Driver Service отправляет оффер Driver_A и Driver_B
// 3. Driver_A нажимает "Принять" → UPDATE успешен
// 4. Driver_B нажимает "Принять" → UPDATE не сработает (status уже MATCHED)
//
// ИНДЕКСЫ БД (для производительности):
// - PRIMARY KEY на rides.id
// - INDEX на rides.status (для быстрого WHERE status='REQUESTED')
//
// ТРАНЗАКЦИОННОСТЬ:
// Метод НЕ создает транзакцию, т.к. это одиночный UPDATE.
// Если нужна транзакция (например, вместе с созданием event в ride_events),
// вызывающий код должен обернуть в pgx.BeginTx().
func (r *RidePgRepository) AssignDriver(ctx context.Context, rideID string, driverID string) error {
	// SQL-запрос с параметрами ($1, $2) для защиты от SQL-injection
	query := `
		UPDATE rides 
		SET 
			driver_id = $1,           -- Назначаем водителя
			status = 'MATCHED',        -- Меняем статус (см. enum ride_status в схеме)
			matched_at = NOW(),        -- Timestamp когда водитель принял
			updated_at = NOW()         -- Обновляем для аудита
		WHERE id = $2                  -- Конкретная поездка
		  AND status = 'REQUESTED'     -- КРИТИЧНО: только если еще не взята!
	`

	// Выполняем UPDATE через connection pool
	// pgx автоматически переиспользует соединения
	result, err := r.pool.Exec(ctx, query, driverID, rideID)
	if err != nil {
		// SQL ошибка (constraint violation, connection timeout, etc.)
		r.log.Error(logger.Entry{
			Action:  "db_assign_driver_failed",
			Message: err.Error(),
			RideID:  rideID,
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		return fmt.Errorf("assign driver: %w", err)
	}

	// Проверяем результат: была ли обновлена хотя бы одна строка?
	// RowsAffected() == 0 означает:
	// 1. Поездка не найдена (неверный ride_id)
	// 2. Поездка уже назначена другому водителю (status != REQUESTED)
	if result.RowsAffected() == 0 {
		return fmt.Errorf("ride not found or already assigned (ride_id=%s)", rideID)
	}

	// SUCCESS: Логируем для мониторинга
	// Эти логи используются для:
	// - Дебаггинга (почему водитель не получил поездку?)
	// - Метрик (сколько поездок назначается в минуту?)
	// - Аудита (кто когда взял какую поездку?)
	r.log.Info(logger.Entry{
		Action:  "driver_assigned_to_ride",
		Message: fmt.Sprintf("driver %s assigned to ride %s", driverID, rideID),
		RideID:  rideID,
		Additional: map[string]any{
			"driver_id": driverID,
		},
	})

	return nil
}
