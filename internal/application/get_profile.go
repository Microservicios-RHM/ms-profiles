package application

import (
	"context"

	"github.com/Microservicios-RHM/ms-profiles/internal/domain"
)

type GetProfile struct {
	repository domain.ProfileRepository
}

func NewGetProfile(repository domain.ProfileRepository) *GetProfile {
	return &GetProfile{repository: repository}
}

// Execute retorna domain.ErrProfileNotFound si el empleado no tiene perfil; la capa HTTP lo
// traduce a 404.
func (uc *GetProfile) Execute(ctx context.Context, empleadoID string) (domain.Profile, error) {
	return uc.repository.FindByEmployeeID(ctx, empleadoID)
}
