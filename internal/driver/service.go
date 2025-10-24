package driver

import (
	"ridehail/internal/db"
	"ridehail/internal/logger"
)

type Service struct {
	repo   db.DriverRepoInterface
	logger *logger.Logger
}

func NewService(repo db.RideRepoInterface, logger *logger.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}

type ServiceInterface interface {
	// Define driver-related service methods here
}

// func (s *Service) () {
// 	 Implement driver-related logic here
// }
