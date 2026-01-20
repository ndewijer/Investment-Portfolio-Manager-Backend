package service

import (
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/repository"
)

// IbkrService handles fund-related business logic operations.
type IbkrService struct {
	ibkrRepo *repository.IbkrRepository
}

// NewIbkrService creates a new IbkrService with the provided repository dependencies.
func NewIbkrService(
	ibkrRepo *repository.IbkrRepository,
) *IbkrService {
	return &IbkrService{
		ibkrRepo: ibkrRepo,
	}
}

func (s *IbkrService) GetIbkrConfig() (model.IbkrConfig, error) {
	return s.ibkrRepo.GetIbkrConfig()
}
