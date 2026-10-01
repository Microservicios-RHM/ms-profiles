package application

import (
	"context"

	"github.com/Microservicios-RHM/ms-profiles/internal/domain"
)

type ListProfiles struct {
	repository domain.ProfileRepository
}

func NewListProfiles(repository domain.ProfileRepository) *ListProfiles {
	return &ListProfiles{repository: repository}
}

func (uc *ListProfiles) Execute(ctx context.Context) ([]domain.Profile, error) {
	return uc.repository.ListAll(ctx)
}
