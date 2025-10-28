package domain

import "errors"

var (
	// ErrRideNotFound возвращается когда поездка не найдена
	ErrRideNotFound = errors.New("ride not found")

	// ErrInvalidCoordinates возвращается при невалидных координатах
	ErrInvalidCoordinates = errors.New("invalid coordinates")

	// ErrInvalidVehicleType возвращается при неподдерживаемом типе авто
	ErrInvalidVehicleType = errors.New("invalid vehicle type")

	// ErrRideAlreadyCancelled возвращается при попытке изменить отмененную поездку
	ErrRideAlreadyCancelled = errors.New("ride already cancelled")

	// ErrRideAlreadyCompleted возвращается при попытке изменить завершенную поездку
	ErrRideAlreadyCompleted = errors.New("ride already completed")

	// ErrUnauthorized возвращается при отсутствии прав доступа
	ErrUnauthorized = errors.New("unauthorized")

	// ErrInvalidStatus возвращается при невалидном статусе поездки
	ErrInvalidStatus = errors.New("invalid ride status")
)
