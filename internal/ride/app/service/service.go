package service

import (
	"ridehail/internal/logger"
	"ridehail/internal/ride/ports/ride_repo_ports"
	"ridehail/internal/ride/ride_models"
)

type Service struct {
	repo   ride_repo_ports.RideRepoInterface
	logger *logger.Logger
}

type ServiceInterface interface {
	CreateRide(passengerID string, req ride_models.RideRequest) (ride_models.Ride, error)
	CancelRide(rideID string, userID string) error
}

func NewService(repo ride_repo_ports.RideRepoInterface, logger *logger.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}

func (s *Service) CreateRide(passengerID string, req ride_models.RideRequest) (ride_models.Ride, error) {
	// Логика создания поездки
	return ride_models.Ride{}, nil
}

func (s *Service) CancelRide(rideID string, userID string) error {
	// Логика отмены поездки
	return nil
}
