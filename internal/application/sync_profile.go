package application

import (
	"context"
	"log/slog"

	"github.com/Microservicios-RHM/ms-profiles/internal/domain"
	"github.com/Microservicios-RHM/ms-profiles/internal/infrastructure/messaging"
)

type SyncProfile struct {
	repository domain.ProfileRepository
	logger     *slog.Logger
}

func NewSyncProfile(repository domain.ProfileRepository, logger *slog.Logger) *SyncProfile {
	return &SyncProfile{repository: repository, logger: logger}
}

// Handle es el handler de empleado.actualizado: sincroniza nombre y email en el perfil existente.
// No crea un perfil si no existe (eso solo pasa vía empleado.creado): esa situación se advierte
// porque no debería ocurrir en operación normal.
func (uc *SyncProfile) Handle(ctx context.Context, envelope messaging.EventEnvelope) error {
	empleadoID := envelope.StringField("id")
	nombre := envelope.StringField("nombre")
	email := envelope.StringField("email")

	wasNew, found, err := uc.repository.SyncIfNew(ctx, envelope.ID, empleadoID, nombre, email)
	if err != nil {
		return err
	}
	if !wasNew {
		uc.logger.Info("duplicate event ignored", "eventId", envelope.ID, "eventType", envelope.Type)
		return nil
	}
	if !found {
		uc.logger.Warn(
			"profile not found for empleado.actualizado; nothing synced",
			"eventId", envelope.ID, "empleadoId", empleadoID,
		)
		return nil
	}

	uc.logger.Info("profile synced", "eventId", envelope.ID, "empleadoId", empleadoID)
	return nil
}
