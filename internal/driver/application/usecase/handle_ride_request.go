package usecase

import (
	"ridehail/internal/driver/adapters/out/repo"
	"ridehail/internal/shared/logger"
)

type Service struct {
	repo   repo.DriverRepoInterface
	logger *logger.Logger
}

func NewService(repo repo.RideRepoInterface, logger *logger.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}

type ServiceInterface interface {
	// Define driver-related service methods here
}

// func (s *Service) () {
// 	 Implement driver-related logic here
// }
