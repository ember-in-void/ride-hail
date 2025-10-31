package in

import "context"

// GoOfflineInput — входные данные для перехода водителя в офлайн
type GoOfflineInput struct {
	DriverID string `json:"driver_id"`
}

// SessionSummary — итоги сессии водителя
type SessionSummary struct {
	DurationHours  float64 `json:"duration_hours"`
	RidesCompleted int     `json:"rides_completed"`
	Earnings       float64 `json:"earnings"`
}

// GoOfflineOutput — результат перехода в офлайн
type GoOfflineOutput struct {
	SessionID      string         `json:"session_id"`
	Status         string         `json:"status"`
	SessionSummary SessionSummary `json:"session_summary"`
	Message        string         `json:"message"`
}

// GoOfflineUseCase — use-case для перехода водителя в офлайн
type GoOfflineUseCase interface {
	Execute(ctx context.Context, input GoOfflineInput) (*GoOfflineOutput, error)
}
