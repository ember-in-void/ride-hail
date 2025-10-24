package model

// ==== Roles ====
const (
	RolePassenger = "PASSENGER"
	RoleDriver    = "DRIVER"
	RoleAdmin     = "ADMIN"
)

// ==== User Status ====
const (
	UserStatusActive   = "ACTIVE"
	UserStatusInactive = "INACTIVE"
	UserStatusBanned   = "BANNED"
)

// ==== Ride Status ====
const (
	RideStatusRequested  = "REQUESTED"
	RideStatusMatched    = "MATCHED"
	RideStatusEnRoute    = "EN_ROUTE"
	RideStatusArrived    = "ARRIVED"
	RideStatusInProgress = "IN_PROGRESS"
	RideStatusCompleted  = "COMPLETED"
	RideStatusCancelled  = "CANCELLED"
)

// ==== Vehicle Type ====
const (
	VehicleEconomy = "ECONOMY"
	VehiclePremium = "PREMIUM"
	VehicleXL      = "XL"
)

// ==== Ride Event Type ====
const (
	EventRideRequested   = "RIDE_REQUESTED"
	EventDriverMatched   = "DRIVER_MATCHED"
	EventDriverArrived   = "DRIVER_ARRIVED"
	EventRideStarted     = "RIDE_STARTED"
	EventRideCompleted   = "RIDE_COMPLETED"
	EventRideCancelled   = "RIDE_CANCELLED"
	EventStatusChanged   = "STATUS_CHANGED"
	EventLocationUpdated = "LOCATION_UPDATED"
	EventFareAdjusted    = "FARE_ADJUSTED"
)
