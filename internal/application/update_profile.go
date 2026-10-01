package application

import (
	"context"

	"github.com/Microservicios-RHM/ms-profiles/internal/domain"
)

// UpdateProfileInput usa punteros para permitir actualización parcial: un campo nil conserva el
// valor actual, un campo presente (incluso "") lo reemplaza.
type UpdateProfileInput struct {
	Telefono  *string
	Direccion *string
	Ciudad    *string
	Biografia *string
}

type UpdateProfile struct {
	repository domain.ProfileRepository
}

func NewUpdateProfile(repository domain.ProfileRepository) *UpdateProfile {
	return &UpdateProfile{repository: repository}
}

// Execute lee el perfil actual, fusiona los campos provistos y persiste el resultado. Retorna
// domain.ErrProfileNotFound si el empleado no tiene perfil.
func (uc *UpdateProfile) Execute(ctx context.Context, empleadoID string, input UpdateProfileInput) (domain.Profile, error) {
	existing, err := uc.repository.FindByEmployeeID(ctx, empleadoID)
	if err != nil {
		return domain.Profile{}, err
	}

	telefono := existing.Telefono
	if input.Telefono != nil {
		telefono = *input.Telefono
	}
	direccion := existing.Direccion
	if input.Direccion != nil {
		direccion = *input.Direccion
	}
	ciudad := existing.Ciudad
	if input.Ciudad != nil {
		ciudad = *input.Ciudad
	}
	biografia := existing.Biografia
	if input.Biografia != nil {
		biografia = *input.Biografia
	}

	return uc.repository.UpdateContactInfo(ctx, empleadoID, telefono, direccion, ciudad, biografia)
}
