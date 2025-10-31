package transport

// GoOnlineRequest — HTTP request для перехода в онлайн
type GoOnlineRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// GoOfflineResponse — HTTP response для перехода в офлайн
type GoOfflineResponse struct {
	SessionID      string         `json:"session_id"`
	Status         string         `json:"status"`
	SessionSummary SessionSummary `json:"session_summary"`
	Message        string         `json:"message"`
}

type SessionSummary struct {
	DurationHours  float64 `json:"duration_hours"`
	RidesCompleted int     `json:"rides_completed"`
	Earnings       float64 `json:"earnings"`
}

// UpdateLocationRequest — HTTP request для обновления локации
type UpdateLocationRequest struct {
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
	AccuracyMeters float64 `json:"accuracy_meters,omitempty"`
	SpeedKmh       float64 `json:"speed_kmh,omitempty"`
	HeadingDegrees float64 `json:"heading_degrees,omitempty"`
}

// StartRideRequest — HTTP request для начала поездки
type StartRideRequest struct {
	RideID    string  `json:"ride_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// CompleteRideRequest — HTTP request для завершения поездки
type CompleteRideRequest struct {
	RideID                string  `json:"ride_id"`
	FinalLatitude         float64 `json:"final_latitude"`
	FinalLongitude        float64 `json:"final_longitude"`
	ActualDistanceKm      float64 `json:"actual_distance_km"`
	ActualDurationMinutes int     `json:"actual_duration_minutes"`
}

// ErrorResponse — стандартный ответ об ошибке
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}
