package service

import "github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/repository"

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
