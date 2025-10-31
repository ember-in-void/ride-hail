package in

import "context"

// UpdateLocationInput — входные данные для обновления локации
type UpdateLocationInput struct {
	DriverID       string  `json:"driver_id"`
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
	AccuracyMeters float64 `json:"accuracy_meters,omitempty"`
	SpeedKmh       float64 `json:"speed_kmh,omitempty"`
	HeadingDegrees float64 `json:"heading_degrees,omitempty"`
}

// UpdateLocationOutput — результат обновления локации
type UpdateLocationOutput struct {
	CoordinateID string `json:"coordinate_id"`
	UpdatedAt    string `json:"updated_at"`
}

// UpdateLocationUseCase — use-case для обновления локации водителя
type UpdateLocationUseCase interface {
	Execute(ctx context.Context, input UpdateLocationInput) (*UpdateLocationOutput, error)
}
