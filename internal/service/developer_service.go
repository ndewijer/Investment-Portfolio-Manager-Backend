package service

import (
	"context"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/api/request"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/repository"
)

// DeveloperService handles Developer-related business logic operations.
type DeveloperService struct {
	developerRepo *repository.DeveloperRepository
}

// NewDeveloperService creates a new DeveloperService with the provided repository dependencies.
func NewDeveloperService(
	developerRepo *repository.DeveloperRepository,
) *DeveloperService {
	return &DeveloperService{
		developerRepo: developerRepo,
	}
}

func (s *DeveloperService) GetLogs(_ context.Context, filters *request.LogFilters) (*model.LogResponse, error) {
	// Add any business logic validation here if needed

	// Pass filters to repository
	return s.developerRepo.GetLogs(filters)
}
