package in

import "context"

// GetOverviewInput — входные данные для получения обзора системы
type GetOverviewInput struct {
	// Пустая структура, параметры не требуются
}

// GetOverviewOutput — выходные данные обзора системы
type GetOverviewOutput struct {
	Timestamp          string         `json:"timestamp"`
	Metrics            SystemMetrics  `json:"metrics"`
	DriverDistribution map[string]int `json:"driver_distribution"`
	Hotspots           []Hotspot      `json:"hotspots,omitempty"`
}

// SystemMetrics — метрики системы
type SystemMetrics struct {
	ActiveRides                int     `json:"active_rides"`
	AvailableDrivers           int     `json:"available_drivers"`
	BusyDrivers                int     `json:"busy_drivers"`
	TotalRidesToday            int     `json:"total_rides_today"`
	TotalRevenueToday          float64 `json:"total_revenue_today"`
	AverageWaitTimeMinutes     float64 `json:"average_wait_time_minutes"`
	AverageRideDurationMinutes float64 `json:"average_ride_duration_minutes"`
	CancellationRate           float64 `json:"cancellation_rate"`
}

// Hotspot — горячая точка (зона повышенного спроса)
type Hotspot struct {
	Location       string `json:"location"`
	ActiveRides    int    `json:"active_rides"`
	WaitingDrivers int    `json:"waiting_drivers"`
}

// GetOverviewUseCase — use case для получения обзора системы
type GetOverviewUseCase interface {
	Execute(ctx context.Context, input GetOverviewInput) (*GetOverviewOutput, error)
}
