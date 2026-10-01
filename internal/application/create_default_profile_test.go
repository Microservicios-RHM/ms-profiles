package application

import (
	"context"
	"testing"
)

func TestCreateDefaultProfile_CreaPerfilConDatosDelEvento(t *testing.T) {
	repo := newFakeRepository()
	uc := NewCreateDefaultProfile(repo, testLogger())

	if err := uc.Handle(context.Background(), employeeCreatedEnvelope()); err != nil {
		t.Fatalf("Handle() error = %v, want nil", err)
	}

	if len(repo.createCalls) != 1 {
		t.Fatalf("got %d calls, want 1", len(repo.createCalls))
	}
	call := repo.createCalls[0]
	if call.eventID != "3f2a1c9e-7b4d-4e10-9c2a-1a2b3c4d5e6f" {
		t.Errorf("eventID = %q, want envelope id", call.eventID)
	}
	if call.profile.EmpleadoID != "E001" || call.profile.Nombre != "Juan" || call.profile.Email != "juan.perez@empresa.com" {
		t.Errorf("perfil inesperado: %+v", call.profile)
	}
	if call.profile.Telefono != "" || call.profile.Direccion != "" || call.profile.Ciudad != "" || call.profile.Biografia != "" {
		t.Errorf("perfil por defecto debe tener campos opcionales vacíos: %+v", call.profile)
	}
}

func TestCreateDefaultProfile_NoFallaConEventoDuplicado(t *testing.T) {
	repo := newFakeRepository()
	repo.alreadyProcessed = true
	uc := NewCreateDefaultProfile(repo, testLogger())

	if err := uc.Handle(context.Background(), employeeCreatedEnvelope()); err != nil {
		t.Fatalf("Handle() error = %v, want nil", err)
	}
	if len(repo.createCalls) != 1 {
		t.Fatalf("got %d calls, want 1", len(repo.createCalls))
	}
}
