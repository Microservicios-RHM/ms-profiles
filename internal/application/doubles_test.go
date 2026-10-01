package application

import (
	"context"
	"io"
	"log/slog"

	"github.com/Microservicios-RHM/ms-profiles/internal/domain"
	"github.com/Microservicios-RHM/ms-profiles/internal/infrastructure/messaging"
)

// fakeRepository implementa domain.ProfileRepository en memoria, con comportamiento
// configurable para simular duplicados, perfiles inexistentes, etc.
type fakeRepository struct {
	alreadyProcessed bool
	profileFound     bool // usado por SyncIfNew/ArchiveIfNew: ¿el perfil existía?
	profiles         map[string]domain.Profile

	createCalls []struct {
		eventID string
		profile domain.Profile
	}
	syncCalls []struct {
		eventID, empleadoID, nombre, email string
	}
	archiveCalls []struct {
		eventID, empleadoID string
	}
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{profiles: make(map[string]domain.Profile)}
}

func (r *fakeRepository) CreateIfNew(_ context.Context, eventID string, profile domain.Profile) (bool, error) {
	r.createCalls = append(r.createCalls, struct {
		eventID string
		profile domain.Profile
	}{eventID, profile})
	if r.alreadyProcessed {
		return false, nil
	}
	r.profiles[profile.EmpleadoID] = profile
	return true, nil
}

func (r *fakeRepository) SyncIfNew(_ context.Context, eventID, empleadoID, nombre, email string) (bool, bool, error) {
	r.syncCalls = append(r.syncCalls, struct{ eventID, empleadoID, nombre, email string }{eventID, empleadoID, nombre, email})
	if r.alreadyProcessed {
		return false, false, nil
	}
	if profile, ok := r.profiles[empleadoID]; ok {
		profile.Nombre = nombre
		profile.Email = email
		r.profiles[empleadoID] = profile
		return true, true, nil
	}
	return true, r.profileFound, nil
}

func (r *fakeRepository) ArchiveIfNew(_ context.Context, eventID, empleadoID string) (bool, bool, error) {
	r.archiveCalls = append(r.archiveCalls, struct{ eventID, empleadoID string }{eventID, empleadoID})
	if r.alreadyProcessed {
		return false, false, nil
	}
	if profile, ok := r.profiles[empleadoID]; ok {
		profile.Archivado = true
		r.profiles[empleadoID] = profile
		return true, true, nil
	}
	return true, r.profileFound, nil
}

func (r *fakeRepository) FindByEmployeeID(_ context.Context, empleadoID string) (domain.Profile, error) {
	if profile, ok := r.profiles[empleadoID]; ok {
		return profile, nil
	}
	return domain.Profile{}, domain.ErrProfileNotFound
}

func (r *fakeRepository) UpdateContactInfo(_ context.Context, empleadoID, telefono, direccion, ciudad, biografia string) (domain.Profile, error) {
	profile, ok := r.profiles[empleadoID]
	if !ok {
		return domain.Profile{}, domain.ErrProfileNotFound
	}
	profile.Telefono = telefono
	profile.Direccion = direccion
	profile.Ciudad = ciudad
	profile.Biografia = biografia
	r.profiles[empleadoID] = profile
	return profile, nil
}

func (r *fakeRepository) ListAll(_ context.Context) ([]domain.Profile, error) {
	profiles := make([]domain.Profile, 0, len(r.profiles))
	for _, p := range r.profiles {
		profiles = append(profiles, p)
	}
	return profiles, nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func employeeCreatedEnvelope() messaging.EventEnvelope {
	return messaging.EventEnvelope{
		ID:         "3f2a1c9e-7b4d-4e10-9c2a-1a2b3c4d5e6f",
		Type:       "empleado.creado",
		Version:    1,
		OccurredAt: "2026-03-01T10:00:00.000Z",
		Producer:   "empleados-service",
		Data: map[string]any{
			"id":     "E001",
			"nombre": "Juan",
			"email":  "juan.perez@empresa.com",
		},
	}
}

func employeeUpdatedEnvelope() messaging.EventEnvelope {
	return messaging.EventEnvelope{
		ID:         "8c0366ef-1e0f-42e7-835f-cd9c4e7d241f",
		Type:       "empleado.actualizado",
		Version:    1,
		OccurredAt: "2026-04-01T10:00:00.000Z",
		Producer:   "empleados-service",
		Data: map[string]any{
			"id":     "E001",
			"nombre": "Juan Actualizado",
			"email":  "juan.nuevo@empresa.com",
		},
	}
}

func employeeRetiredEnvelope() messaging.EventEnvelope {
	return messaging.EventEnvelope{
		ID:         "c1b2a3d4-5e6f-4a7b-8c9d-0e1f2a3b4c5d",
		Type:       "empleado.retirado",
		Version:    1,
		OccurredAt: "2026-06-01T10:00:00.000Z",
		Producer:   "empleados-service",
		Data: map[string]any{
			"id": "E001",
		},
	}
}
