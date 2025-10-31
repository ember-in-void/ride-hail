package in

import "context"

// ============================================================================
// ВХОДНОЙ ПОРТ (Input Port) - Интерфейс для бизнес-логики
// ============================================================================
// Определяет ЧТО должна делать система (бизнес-требования),
// но НЕ определяет КАК это будет реализовано (детали реализации).
//
// Это контракт между внешним миром (адаптеры) и бизнес-логикой (use cases).
// ============================================================================

// HandleDriverResponseInput — DTO (Data Transfer Object) для передачи данных в use case.
//
// Содержит всю информацию, полученную от водителя через:
// - WebSocket (водитель нажал "Принять" в приложении)
// - RabbitMQ (сообщение driver.response.{ride_id})
//
// Поля:
//   - RideID: UUID поездки из базы данных
//   - DriverID: UUID водителя из JWT токена
//   - Accepted: true = принял, false = отклонил
//   - EstimatedArrivalMinutes: сколько минут водитель едет до точки pickup
//   - DriverLocation*: текущие координаты водителя для отображения на карте
type HandleDriverResponseInput struct {
	RideID                  string  // UUID поездки
	DriverID                string  // UUID водителя
	Accepted                bool    // Принял (true) или отклонил (false)
	EstimatedArrivalMinutes int     // Расчетное время прибытия (минуты)
	DriverLocationLat       float64 // Широта водителя (для карты пассажира)
	DriverLocationLng       float64 // Долгота водителя
}

// HandleDriverResponseOutput — результат выполнения use case.
//
// Возвращается адаптеру (RabbitMQ consumer) для:
// - Отправки уведомления пассажиру через WebSocket
// - Логирования успешной операции
// - Публикации события в event sourcing (опционально)
//
// Поля:
//   - RideID: ID обработанной поездки
//   - Status: Новый статус ("MATCHED" если принял, "REQUESTED" если отклонил)
//   - DriverAssigned: Булев флаг для быстрой проверки
//   - PassengerID: Кому отправить WebSocket уведомление "Водитель найден!"
type HandleDriverResponseOutput struct {
	RideID         string // UUID поездки
	Status         string // Статус: REQUESTED | MATCHED | ...
	DriverAssigned bool   // true если водитель успешно назначен
	PassengerID    string // UUID пассажира (для WebSocket уведомления)
}

// HandleDriverResponseUseCase — интерфейс (порт) для use case.
//
// АРХИТЕКТУРНОЕ НАЗНАЧЕНИЕ:
// Этот интерфейс позволяет:
// 1. Тестировать адаптеры (consumers) с mock-реализацией
// 2. Менять реализацию бизнес-логики без изменения адаптеров
// 3. Соблюдать Dependency Inversion Principle (SOLID)
//
// DEPENDENCY FLOW (Clean Architecture):
// RabbitMQ Consumer → [Interface] ← Use Case Implementation
//
//	(адаптер)         (этот порт)     (бизнес-логика)
//
// Адаптер зависит от интерфейса, а не от конкретной реализации!
type HandleDriverResponseUseCase interface {
	// Execute выполняет бизнес-логику обработки ответа водителя.
	//
	// Параметры:
	//   - ctx: контекст для отмены операции или таймаутов
	//   - input: данные от водителя (см. HandleDriverResponseInput)
	//
	// Возвращает:
	//   - Output с результатом (PassengerID для уведомления)
	//   - Ошибку если поездка не найдена или уже назначена
	Execute(ctx context.Context, input HandleDriverResponseInput) (*HandleDriverResponseOutput, error)
}
