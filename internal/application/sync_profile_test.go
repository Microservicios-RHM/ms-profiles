package application

import (
	"context"
	"testing"

	"github.com/Microservicios-RHM/ms-profiles/internal/domain"
)

func TestSyncProfile_SincronizaNombreYEmail(t *testing.T) {
	repo := newFakeRepository()
	repo.profiles["E001"] = domain.Profile{EmpleadoID: "E001", Nombre: "Juan", Email: "juan.perez@empresa.com"}
	uc := NewSyncProfile(repo, testLogger())

	if err := uc.Handle(context.Background(), employeeUpdatedEnvelope()); err != nil {
		t.Fatalf("Handle() error = %v, want nil", err)
	}

	profile := repo.profiles["E001"]
	if profile.Nombre != "Juan Actualizado" || profile.Email != "juan.nuevo@empresa.com" {
		t.Errorf("perfil no sincronizado: %+v", profile)
	}
}

func TestSyncProfile_AdvierteSiElPerfilNoExiste(t *testing.T) {
	repo := newFakeRepository() // vacío: E001 nunca llegó por empleado.creado
	uc := NewSyncProfile(repo, testLogger())

	if err := uc.Handle(context.Background(), employeeUpdatedEnvelope()); err != nil {
		t.Fatalf("Handle() error = %v, want nil", err)
	}
	if len(repo.syncCalls) != 1 {
		t.Fatalf("got %d calls, want 1", len(repo.syncCalls))
	}
}

func TestSyncProfile_NoFallaConEventoDuplicado(t *testing.T) {
	repo := newFakeRepository()
	repo.alreadyProcessed = true
	uc := NewSyncProfile(repo, testLogger())

	if err := uc.Handle(context.Background(), employeeUpdatedEnvelope()); err != nil {
		t.Fatalf("Handle() error = %v, want nil", err)
	}
}
