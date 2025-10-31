package usecase

import (
	"context"
	"fmt"

	"ridehail/internal/ride/application/ports/in"
	"ridehail/internal/ride/application/ports/out"
	"ridehail/internal/shared/logger"
)

// ============================================================================
// БИЗНЕС-ЛОГИКА: Обработка ответа водителя на предложение поездки
// ============================================================================
// Этот use case отвечает за:
// 1. Получение ответа водителя (принял/отклонил) через RabbitMQ
// 2. Валидацию статуса поездки (должна быть REQUESTED)
// 3. Назначение водителя на поездку в базе данных
// 4. Уведомление пассажира об успешном назначении
//
// Архитектурный паттерн: Clean Architecture
// - Не зависит от деталей реализации БД или транспорта
// - Использует порты (интерфейсы) для взаимодействия с внешним миром
// ============================================================================

// HandleDriverResponseService реализует бизнес-логику обработки ответа водителя.
//
// Зависимости:
//   - rideRepo: доступ к данным поездок (чтение, обновление статуса)
//   - log: структурированное логирование для отладки и мониторинга
type HandleDriverResponseService struct {
	rideRepo out.RideRepository // Интерфейс для работы с БД (абстракция)
	log      *logger.Logger     // Логгер для трейсинга операций
}

// NewHandleDriverResponseService — фабрика для создания сервиса.
//
// Dependency Injection: все зависимости передаются извне, что упрощает
// тестирование и позволяет легко заменять реализации.
func NewHandleDriverResponseService(
	rideRepo out.RideRepository,
	log *logger.Logger,
) *HandleDriverResponseService {
	return &HandleDriverResponseService{
		rideRepo: rideRepo,
		log:      log,
	}
}

// Execute — главный метод use case: обрабатывает ответ водителя.
//
// СЦЕНАРИЙ ИСПОЛЬЗОВАНИЯ:
// 1. Driver Service находит водителя поблизости через PostGIS (ST_DWithin)
// 2. Отправляет оффер водителю через WebSocket
// 3. Водитель нажимает "Принять" в приложении
// 4. WebSocket → RabbitMQ → этот метод
//
// БИЗНЕС-ПРАВИЛА:
// - Поездка должна быть в статусе REQUESTED (не взята другим водителем)
// - Если водитель отклонил — отправляем оффер следующему водителю (TODO)
// - Если принял — атомарно обновляем статус и driver_id в БД
//
// ВОЗВРАЩАЕМОЕ ЗНАЧЕНИЕ:
// - Output содержит passenger_id для отправки уведомления через WebSocket
// - Ошибка если поездка не найдена или уже назначена другому водителю
func (s *HandleDriverResponseService) Execute(
	ctx context.Context,
	input in.HandleDriverResponseInput,
) (*in.HandleDriverResponseOutput, error) {
	// ЛОГИРОВАНИЕ: Фиксируем начало обработки для трейсинга
	s.log.Info(logger.Entry{
		Action:  "handle_driver_response",
		Message: fmt.Sprintf("ride=%s, driver=%s, accepted=%t", input.RideID, input.DriverID, input.Accepted),
		RideID:  input.RideID,
		Additional: map[string]any{
			"driver_id": input.DriverID,
			"accepted":  input.Accepted,
		},
	})

	// СЦЕНАРИЙ 1: Водитель отклонил поездку
	if !input.Accepted {
		s.log.Info(logger.Entry{
			Action:  "driver_rejected_ride",
			Message: fmt.Sprintf("driver %s rejected ride %s", input.DriverID, input.RideID),
			RideID:  input.RideID,
		})

		// TODO (Будущая реализация):
		// 1. Получить список других доступных водителей поблизости
		// 2. Отправить оффер следующему водителю из очереди
		// 3. Если водителей нет — отменить поездку или увеличить радиус поиска
		//
		// Пока просто возвращаем статус REQUESTED (поездка ждет водителя)
		return &in.HandleDriverResponseOutput{
			RideID:         input.RideID,
			Status:         "REQUESTED",
			DriverAssigned: false,
		}, nil
	}

	// СЦЕНАРИЙ 2: Водитель принял поездку

	// ШАГ 1: Получаем актуальные данные поездки из БД
	// Важно: делаем это перед обновлением, чтобы проверить статус
	ride, err := s.rideRepo.FindByID(ctx, input.RideID)
	if err != nil {
		s.log.Error(logger.Entry{
			Action:  "find_ride_failed",
			Message: err.Error(),
			RideID:  input.RideID,
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		return nil, fmt.Errorf("find ride: %w", err)
	}

	// ШАГ 2: Валидация бизнес-правил
	// КРИТИЧНО: проверяем, что поездка еще не взята другим водителем
	// (race condition может произойти если несколько водителей одновременно приняли)
	if ride.Status != "REQUESTED" {
		s.log.Warn(logger.Entry{
			Action:  "ride_not_in_requested_status",
			Message: fmt.Sprintf("ride %s has status %s, expected REQUESTED", input.RideID, ride.Status),
			RideID:  input.RideID,
		})
		return nil, fmt.Errorf("ride is not in REQUESTED status (current: %s)", ride.Status)
	}

	// ШАГ 3: Атомарное назначение водителя в БД
	// SQL: UPDATE rides SET driver_id=$1, status='MATCHED', matched_at=NOW()
	//      WHERE id=$2 AND status='REQUESTED'
	// WHERE status='REQUESTED' защищает от race condition
	if err := s.rideRepo.AssignDriver(ctx, input.RideID, input.DriverID); err != nil {
		s.log.Error(logger.Entry{
			Action:  "assign_driver_failed",
			Message: err.Error(),
			RideID:  input.RideID,
			Error:   &logger.ErrObj{Msg: err.Error()},
		})
		return nil, fmt.Errorf("assign driver: %w", err)
	}

	// ШАГ 4: Логируем успех для мониторинга
	s.log.Info(logger.Entry{
		Action:  "driver_assigned_successfully",
		Message: fmt.Sprintf("driver %s assigned to ride %s", input.DriverID, input.RideID),
		RideID:  input.RideID,
		Additional: map[string]any{
			"driver_id": input.DriverID,
			"eta":       input.EstimatedArrivalMinutes,
		},
	})

	// ШАГ 5: Возвращаем данные для уведомления пассажира
	// passenger_id нужен для отправки WebSocket сообщения:
	// "Водитель найден! Прибудет через X минут"
	return &in.HandleDriverResponseOutput{
		RideID:         input.RideID,
		Status:         "DRIVER_ASSIGNED", // Фронтенд показывает "Водитель едет к вам"
		DriverAssigned: true,
		PassengerID:    ride.PassengerID, // Кому отправить уведомление
	}, nil
}
