package transport

// GoOnlineRequest — запрос на переход в онлайн
type GoOnlineRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// GoOnlineResponse — ответ на переход в онлайн
type GoOnlineResponse struct {
	Status    string `json:"status"`
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

// GoOfflineResponse — ответ на переход в офлайн
type GoOfflineResponse struct {
	Status         string                 `json:"status"`
	SessionID      string                 `json:"session_id"`
	SessionSummary SessionSummaryResponse `json:"session_summary"`
	Message        string                 `json:"message"`
}

// SessionSummaryResponse — сводка по сессии
type SessionSummaryResponse struct {
	DurationHours  float64 `json:"duration_hours"`
	RidesCompleted int     `json:"rides_completed"`
	Earnings       float64 `json:"earnings"`
}

// UpdateLocationRequest — запрос на обновление локации
type UpdateLocationRequest struct {
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
	AccuracyMeters float64 `json:"accuracy_meters,omitempty"`
	SpeedKmh       float64 `json:"speed_kmh,omitempty"`
	HeadingDegrees float64 `json:"heading_degrees,omitempty"`
}

// UpdateLocationResponse — ответ на обновление локации
type UpdateLocationResponse struct {
	CoordinateID string `json:"coordinate_id"`
	UpdatedAt    string `json:"updated_at"`
}

// StartRideRequest — запрос на начало поездки
type StartRideRequest struct {
	RideID    string  `json:"ride_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// StartRideResponse — ответ на начало поездки
type StartRideResponse struct {
	RideID    string `json:"ride_id"`
	Status    string `json:"status"`
	StartedAt string `json:"started_at"`
	Message   string `json:"message"`
}

// CompleteRideRequest — запрос на завершение поездки
type CompleteRideRequest struct {
	RideID                string  `json:"ride_id"`
	FinalLatitude         float64 `json:"final_latitude"`
	FinalLongitude        float64 `json:"final_longitude"`
	ActualDistanceKm      float64 `json:"actual_distance_km"`
	ActualDurationMinutes int     `json:"actual_duration_minutes"`
}

// CompleteRideResponse — ответ на завершение поездки
type CompleteRideResponse struct {
	RideID         string  `json:"ride_id"`
	Status         string  `json:"status"`
	CompletedAt    string  `json:"completed_at"`
	DriverEarnings float64 `json:"driver_earnings"`
	Message        string  `json:"message"`
}

// ErrorResponse — стандартный ответ об ошибке
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}
