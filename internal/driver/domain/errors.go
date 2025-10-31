package domain

import "errors"

var (
	// ErrDriverNotFound — водитель не найден
	ErrDriverNotFound = errors.New("driver not found")

	// ErrDriverNotVerified — водитель не верифицирован
	ErrDriverNotVerified = errors.New("driver not verified")

	// ErrDriverNotAvailable — водитель не в статусе AVAILABLE
	ErrDriverNotAvailable = errors.New("driver not available")

	// ErrDriverBusy — водитель уже занят поездкой
	ErrDriverBusy = errors.New("driver is busy")

	// ErrSessionNotFound — сессия не найдена
	ErrSessionNotFound = errors.New("session not found")

	// ErrSessionAlreadyActive — уже есть активная сессия
	ErrSessionAlreadyActive = errors.New("session already active")

	// ErrRideNotFound — поездка не найдена
	ErrRideNotFound = errors.New("ride not found")

	// ErrRideAlreadyMatched — поездка уже сматчена с другим водителем
	ErrRideAlreadyMatched = errors.New("ride already matched")

	// ErrLocationUpdateTooFrequent — обновление локации слишком частое (rate-limit)
	ErrLocationUpdateTooFrequent = errors.New("location update too frequent")

	// ErrInvalidCoordinates — невалидные координаты
	ErrInvalidCoordinates = errors.New("invalid coordinates")
)
