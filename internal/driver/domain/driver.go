package domain

import "time"

// DriverStatus — статус водителя согласно ТЗ
type DriverStatus string

const (
	DriverStatusOffline   DriverStatus = "OFFLINE"   // не принимает поездки
	DriverStatusAvailable DriverStatus = "AVAILABLE" // доступен для поездок
	DriverStatusBusy      DriverStatus = "BUSY"      // занят поездкой
	DriverStatusEnRoute   DriverStatus = "EN_ROUTE"  // едет к точке pickup
)

// VehicleType — тип транспорта
type VehicleType string

const (
	VehicleTypeEconomy VehicleType = "ECONOMY"
	VehicleTypePremium VehicleType = "PREMIUM"
	VehicleTypeXL      VehicleType = "XL"
)

// Driver — доменная модель водителя
type Driver struct {
	ID            string       `json:"id" db:"id"`
	LicenseNumber string       `json:"license_number" db:"license_number"`
	VehicleType   VehicleType  `json:"vehicle_type" db:"vehicle_type"`
	VehicleAttrs  VehicleAttrs `json:"vehicle_attrs" db:"vehicle_attrs"`
	Rating        float64      `json:"rating" db:"rating"`
	TotalRides    int          `json:"total_rides" db:"total_rides"`
	TotalEarnings float64      `json:"total_earnings" db:"total_earnings"`
	Status        DriverStatus `json:"status" db:"status"`
	IsVerified    bool         `json:"is_verified" db:"is_verified"`
	CreatedAt     time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at" db:"updated_at"`
}

// VehicleAttrs — атрибуты транспорта (хранится как JSONB)
type VehicleAttrs struct {
	Make  string `json:"vehicle_make"`
	Model string `json:"vehicle_model"`
	Color string `json:"vehicle_color"`
	Plate string `json:"vehicle_plate"`
	Year  int    `json:"vehicle_year"`
}

// DriverSession — сессия водителя (online/offline период)
type DriverSession struct {
	ID            string     `json:"id" db:"id"`
	DriverID      string     `json:"driver_id" db:"driver_id"`
	StartedAt     time.Time  `json:"started_at" db:"started_at"`
	EndedAt       *time.Time `json:"ended_at,omitempty" db:"ended_at"`
	TotalRides    int        `json:"total_rides" db:"total_rides"`
	TotalEarnings float64    `json:"total_earnings" db:"total_earnings"`
}

// DriverLocation — текущее состояние водителя (используется для кэша/быстрого доступа)
type DriverLocation struct {
	DriverID  string    `json:"driver_id" db:"driver_id"`
	Latitude  float64   `json:"latitude" db:"latitude"`
	Longitude float64   `json:"longitude" db:"longitude"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// LocationUpdate — DTO от WebSocket или HTTP клиента
type LocationUpdate struct {
	DriverID       string  `json:"driver_id"`
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
	AccuracyMeters float64 `json:"accuracy_meters,omitempty"`
	SpeedKmh       float64 `json:"speed_kmh,omitempty"`
	HeadingDegrees float64 `json:"heading_degrees,omitempty"`
}

// User — общая модель пользователя (реюз из ride service)
type User struct {
	ID           string    `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	Role         string    `json:"role" db:"role"`
	Status       string    `json:"status" db:"status"`
	PasswordHash string    `json:"-" db:"password_hash"`
	Attrs        any       `json:"attrs" db:"attrs"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// Event — доменное событие
type Event struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data"`
}

// CanAcceptRide проверяет, может ли водитель принять поездку
func (d *Driver) CanAcceptRide() bool {
	return d.Status == DriverStatusAvailable && d.IsVerified
}

// CanGoOnline проверяет, может ли водитель перейти в онлайн
func (d *Driver) CanGoOnline() bool {
	return d.Status == DriverStatusOffline && d.IsVerified
}

// CanGoOffline проверяет, может ли водитель перейти в офлайн
func (d *Driver) CanGoOffline() bool {
	return d.Status == DriverStatusAvailable
}
