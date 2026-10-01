package domain

import "context"

// ProfileRepository es el puerto de dominio. Los casos de uso dependen de esta interfaz, no de
// pgx directamente — la implementación Postgres vive en internal/infrastructure/persistence.
//
// Los métodos *IfNew contienen la deduplicación (igual que save_if_new en ms-notifications):
// verificación y efecto ocurren en una sola transacción, así que dos entregas concurrentes del
// mismo evento nunca aplican el efecto dos veces. wasNew=false significa "evento ya procesado, no
// se hizo nada"; found solo es significativo cuando wasNew=true, e indica si el perfil existía.
type ProfileRepository interface {
	CreateIfNew(ctx context.Context, eventID string, profile Profile) (wasNew bool, err error)
	SyncIfNew(ctx context.Context, eventID, empleadoID, nombre, email string) (wasNew bool, found bool, err error)
	ArchiveIfNew(ctx context.Context, eventID, empleadoID string) (wasNew bool, found bool, err error)

	// FindByEmployeeID y UpdateContactInfo son operaciones REST, sin relación con deduplicación de
	// eventos. Retornan ErrProfileNotFound si el empleado no tiene perfil.
	FindByEmployeeID(ctx context.Context, empleadoID string) (Profile, error)
	UpdateContactInfo(ctx context.Context, empleadoID, telefono, direccion, ciudad, biografia string) (Profile, error)
	ListAll(ctx context.Context) ([]Profile, error)
}
