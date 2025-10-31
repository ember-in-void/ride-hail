package domain

import "errors"

var (
	// ErrDriverNotFound возникает, когда водитель не найден
	ErrDriverNotFound = errors.New("driver not found")

	// ErrDriverCannotGoOnline возникает, когда водитель не может выйти в онлайн
	ErrDriverCannotGoOnline = errors.New("driver cannot go online: invalid status or not verified")

	// ErrDriverCannotGoOffline возникает, когда водитель не может выйти в офлайн
	ErrDriverCannotGoOffline = errors.New("driver cannot go offline: not in available status")

	// ErrDriverCannotAcceptRide возникает, когда водитель не может принять поездку
	ErrDriverCannotAcceptRide = errors.New("driver cannot accept ride: not available or not verified")

	// ErrInvalidCoordinates возникает при некорректных координатах
	ErrInvalidCoordinates = errors.New("invalid coordinates")

	// ErrRateLimitExceeded возникает при превышении лимита обновлений локации
	ErrRateLimitExceeded = errors.New("location update rate limit exceeded")

	// ErrSessionNotFound возникает, когда сессия не найдена
	ErrSessionNotFound = errors.New("driver session not found")

	// ErrRideNotFound возникает, когда поездка не найдена
	ErrRideNotFound = errors.New("ride not found")

	// ErrRideAlreadyMatched возникает, когда поездка уже назначена другому водителю
	ErrRideAlreadyMatched = errors.New("ride already matched to another driver")
)
