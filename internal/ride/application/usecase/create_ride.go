package usecase

import "ridehail/internal/ride/domain"

func (s *Service) CreateRide(passengerID string, req domain.RideRequest) (domain.Ride, error) {
	// Логика создания поездки
	return domain.Ride{}, nil
}
