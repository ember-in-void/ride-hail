package usecase

import (
	"ridehail/internal/ride/domain"
	"ridehail/internal/shared/logger"
)

type Service struct {
	repo   out.RideRepoInterface
	logger *logger.Logger
}

type ServiceInterface interface {
	CreateRide(passengerID string, req domain.RideRequest) (domain.Ride, error)
	CancelRide(rideID string, userID string) error
}

func NewService(repo ride_repo_ports.RideRepoInterface, logger *logger.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}

func (s *Service) CreateRide(passengerID string, req domain.RideRequest) (domain.Ride, error) {
	// Логика создания поездки
	return domain.Ride{}, nil
}

func (s *Service) CancelRide(rideID string, userID string) error {
	// Логика отмены поездки
	return nil
}
