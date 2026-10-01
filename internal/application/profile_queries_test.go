package application

import (
	"context"
	"errors"
	"testing"

	"github.com/Microservicios-RHM/ms-profiles/internal/domain"
)

func TestGetProfile_RetornaErrProfileNotFound(t *testing.T) {
	repo := newFakeRepository()
	uc := NewGetProfile(repo)

	_, err := uc.Execute(context.Background(), "E999")
	if !errors.Is(err, domain.ErrProfileNotFound) {
		t.Fatalf("Execute() error = %v, want ErrProfileNotFound", err)
	}
}

func TestListProfiles_ListaTodos(t *testing.T) {
	repo := newFakeRepository()
	repo.profiles["E001"] = domain.Profile{EmpleadoID: "E001"}
	repo.profiles["E002"] = domain.Profile{EmpleadoID: "E002"}
	uc := NewListProfiles(repo)

	profiles, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if len(profiles) != 2 {
		t.Fatalf("got %d profiles, want 2", len(profiles))
	}
}

func TestUpdateProfile_FusionaSoloLosCamposProvistos(t *testing.T) {
	repo := newFakeRepository()
	repo.profiles["E001"] = domain.Profile{
		EmpleadoID: "E001",
		Telefono:   "3000000000",
		Direccion:  "Calle 1",
		Ciudad:     "Bogotá",
		Biografia:  "bio original",
	}
	uc := NewUpdateProfile(repo)

	nuevaCiudad := "Armenia"
	profile, err := uc.Execute(context.Background(), "E001", UpdateProfileInput{Ciudad: &nuevaCiudad})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	if profile.Ciudad != "Armenia" {
		t.Errorf("Ciudad = %q, want Armenia", profile.Ciudad)
	}
	// Los campos no provistos deben conservar su valor original.
	if profile.Telefono != "3000000000" || profile.Direccion != "Calle 1" || profile.Biografia != "bio original" {
		t.Errorf("campos no provistos fueron modificados: %+v", profile)
	}
}

func TestUpdateProfile_RetornaErrProfileNotFound(t *testing.T) {
	repo := newFakeRepository()
	uc := NewUpdateProfile(repo)

	_, err := uc.Execute(context.Background(), "E999", UpdateProfileInput{})
	if !errors.Is(err, domain.ErrProfileNotFound) {
		t.Fatalf("Execute() error = %v, want ErrProfileNotFound", err)
	}
}
