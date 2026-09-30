package application

import (
	"context"
	"log/slog"

	"github.com/Microservicios-RHM/ms-profiles/internal/domain"
	"github.com/Microservicios-RHM/ms-profiles/internal/infrastructure/messaging"
)

type ArchiveProfile struct {
	repository domain.ProfileRepository
	logger     *slog.Logger
}

func NewArchiveProfile(repository domain.ProfileRepository, logger *slog.Logger) *ArchiveProfile {
	return &ArchiveProfile{repository: repository, logger: logger}
}

// Handle es el handler de empleado.retirado: archiva el perfil (archivado = true). Nunca lo
// borra — misma regla de baja lógica que ms-employees aplica al propio empleado.
func (uc *ArchiveProfile) Handle(ctx context.Context, envelope messaging.EventEnvelope) error {
	empleadoID := envelope.StringField("id")

	wasNew, found, err := uc.repository.ArchiveIfNew(ctx, envelope.ID, empleadoID)
	if err != nil {
		return err
	}
	if !wasNew {
		uc.logger.Info("duplicate event ignored", "eventId", envelope.ID, "eventType", envelope.Type)
		return nil
	}
	if !found {
		uc.logger.Warn(
			"profile not found for empleado.retirado; nothing archived",
			"eventId", envelope.ID, "empleadoId", empleadoID,
		)
		return nil
	}

	uc.logger.Info("profile archived", "eventId", envelope.ID, "empleadoId", empleadoID)
	return nil
}
