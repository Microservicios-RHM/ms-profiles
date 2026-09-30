package application

import (
	"context"
	"testing"

	"github.com/Microservicios-RHM/ms-profiles/internal/domain"
)

func TestArchiveProfile_ArchivaSinBorrar(t *testing.T) {
	repo := newFakeRepository()
	repo.profiles["E001"] = domain.Profile{EmpleadoID: "E001", Nombre: "Juan"}
	uc := NewArchiveProfile(repo, testLogger())

	if err := uc.Handle(context.Background(), employeeRetiredEnvelope()); err != nil {
		t.Fatalf("Handle() error = %v, want nil", err)
	}

	profile, ok := repo.profiles["E001"]
	if !ok {
		t.Fatal("el perfil fue borrado; debía conservarse archivado")
	}
	if !profile.Archivado {
		t.Errorf("profile.Archivado = false, want true")
	}
}

func TestArchiveProfile_AdvierteSiElPerfilNoExiste(t *testing.T) {
	repo := newFakeRepository()
	uc := NewArchiveProfile(repo, testLogger())

	if err := uc.Handle(context.Background(), employeeRetiredEnvelope()); err != nil {
		t.Fatalf("Handle() error = %v, want nil", err)
	}
	if len(repo.archiveCalls) != 1 {
		t.Fatalf("got %d calls, want 1", len(repo.archiveCalls))
	}
}

func TestArchiveProfile_NoFallaConEventoDuplicado(t *testing.T) {
	repo := newFakeRepository()
	repo.alreadyProcessed = true
	uc := NewArchiveProfile(repo, testLogger())

	if err := uc.Handle(context.Background(), employeeRetiredEnvelope()); err != nil {
		t.Fatalf("Handle() error = %v, want nil", err)
	}
}
