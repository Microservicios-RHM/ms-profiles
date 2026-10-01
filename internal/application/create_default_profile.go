package application

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/Microservicios-RHM/ms-profiles/internal/domain"
	"github.com/Microservicios-RHM/ms-profiles/internal/infrastructure/messaging"
)

type CreateDefaultProfile struct {
	repository domain.ProfileRepository
	logger     *slog.Logger
}

func NewCreateDefaultProfile(repository domain.ProfileRepository, logger *slog.Logger) *CreateDefaultProfile {
	return &CreateDefaultProfile{repository: repository, logger: logger}
}

// Handle es el handler de empleado.creado: crea el perfil por defecto (telefono/direccion/ciudad/
// biografia vacíos, como pide el enunciado) con los únicos datos que sí trae el evento (nombre,
// email).
func (uc *CreateDefaultProfile) Handle(ctx context.Context, envelope messaging.EventEnvelope) error {
	profile := domain.Profile{
		ID:            uuid.NewString(),
		EmpleadoID:    envelope.StringField("id"),
		Nombre:        envelope.StringField("nombre"),
		Email:         envelope.StringField("email"),
		FechaCreacion: time.Now().UTC(),
	}

	created, err := uc.repository.CreateIfNew(ctx, envelope.ID, profile)
	if err != nil {
		return err
	}
	if !created {
		uc.logger.Info("duplicate event ignored", "eventId", envelope.ID, "eventType", envelope.Type)
		return nil
	}

	uc.logger.Info(
		"default profile created",
		"eventId", envelope.ID,
		"empleadoId", profile.EmpleadoID,
		"profileId", profile.ID,
	)
	return nil
}
