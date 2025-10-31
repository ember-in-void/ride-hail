package in

import "context"

// GoOnlineInput — входные данные для перехода водителя в онлайн
type GoOnlineInput struct {
	DriverID  string  `json:"driver_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// GoOnlineOutput — результат перехода в онлайн
type GoOnlineOutput struct {
	SessionID string `json:"session_id"`
	Status    string `json:"status"`
	Message   string `json:"message"`
}

// GoOnlineUseCase — use-case для перехода водителя в онлайн
type GoOnlineUseCase interface {
	Execute(ctx context.Context, input GoOnlineInput) (*GoOnlineOutput, error)
}
