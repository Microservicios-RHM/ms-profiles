package persistence

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Microservicios-RHM/ms-profiles/internal/domain"
)

type ProfileRepository struct {
	pool *pgxpool.Pool
}

func NewProfileRepository(pool *pgxpool.Pool) *ProfileRepository {
	return &ProfileRepository{pool: pool}
}

// CreateIfNew: INSERT ... ON CONFLICT DO NOTHING RETURNING id sobre eventos_procesados, seguido
// del INSERT del perfil, en una sola transacción — igual que save_if_new en ms-notifications.
// Si eventID ya existía, ni siquiera se ejecuta el segundo INSERT.
func (r *ProfileRepository) CreateIfNew(ctx context.Context, eventID string, profile domain.Profile) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("no fue posible iniciar la transacción: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	wasNew, err := markProcessed(ctx, tx, eventID)
	if err != nil || !wasNew {
		return wasNew, err
	}

	_, err = tx.Exec(
		ctx,
		`INSERT INTO perfiles (id, empleado_id, nombre, email, telefono, direccion, ciudad, biografia, archivado, fecha_creacion)
		 VALUES ($1, $2, $3, $4, '', '', '', '', false, $5)`,
		profile.ID, profile.EmpleadoID, profile.Nombre, profile.Email, profile.FechaCreacion,
	)
	if err != nil {
		return false, fmt.Errorf("no fue posible crear el perfil: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("no fue posible confirmar la transacción: %w", err)
	}
	return true, nil
}

// markProcessed inserta eventID en eventos_procesados dentro de una transacción ya abierta.
// Retorna wasNew=false (sin error) si el id ya existía — el llamador debe hacer rollback/no
// aplicar ningún efecto en ese caso.
func markProcessed(ctx context.Context, tx pgx.Tx, eventID string) (wasNew bool, err error) {
	var insertedID string
	err = tx.QueryRow(
		ctx,
		"INSERT INTO eventos_procesados (id) VALUES ($1) ON CONFLICT DO NOTHING RETURNING id",
		eventID,
	).Scan(&insertedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("no fue posible registrar el evento procesado: %w", err)
	}
	return true, nil
}

// SyncIfNew aplica empleado.actualizado: sincroniza nombre/email si el perfil existe. found
// indica si había un perfil para ese empleado; si no lo había, no se modifica nada (el llamador
// registra una advertencia).
func (r *ProfileRepository) SyncIfNew(ctx context.Context, eventID, empleadoID, nombre, email string) (wasNew bool, found bool, err error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, false, fmt.Errorf("no fue posible iniciar la transacción: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	wasNew, err = markProcessed(ctx, tx, eventID)
	if err != nil || !wasNew {
		return wasNew, false, err
	}

	tag, err := tx.Exec(ctx, "UPDATE perfiles SET nombre = $2, email = $3 WHERE empleado_id = $1", empleadoID, nombre, email)
	if err != nil {
		return false, false, fmt.Errorf("no fue posible sincronizar el perfil: %w", err)
	}
	found = tag.RowsAffected() > 0

	if err := tx.Commit(ctx); err != nil {
		return false, false, fmt.Errorf("no fue posible confirmar la transacción: %w", err)
	}
	return true, found, nil
}

// ArchiveIfNew aplica empleado.retirado: marca archivado = true si el perfil existe. Nunca borra
// la fila.
func (r *ProfileRepository) ArchiveIfNew(ctx context.Context, eventID, empleadoID string) (wasNew bool, found bool, err error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, false, fmt.Errorf("no fue posible iniciar la transacción: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	wasNew, err = markProcessed(ctx, tx, eventID)
	if err != nil || !wasNew {
		return wasNew, false, err
	}

	tag, err := tx.Exec(ctx, "UPDATE perfiles SET archivado = true WHERE empleado_id = $1", empleadoID)
	if err != nil {
		return false, false, fmt.Errorf("no fue posible archivar el perfil: %w", err)
	}
	found = tag.RowsAffected() > 0

	if err := tx.Commit(ctx); err != nil {
		return false, false, fmt.Errorf("no fue posible confirmar la transacción: %w", err)
	}
	return true, found, nil
}

const profileColumns = `id, empleado_id, nombre, email, telefono, direccion, ciudad, biografia, archivado, fecha_creacion`

func scanProfile(row pgx.Row) (domain.Profile, error) {
	var p domain.Profile
	err := row.Scan(&p.ID, &p.EmpleadoID, &p.Nombre, &p.Email, &p.Telefono, &p.Direccion, &p.Ciudad, &p.Biografia, &p.Archivado, &p.FechaCreacion)
	return p, err
}

func (r *ProfileRepository) FindByEmployeeID(ctx context.Context, empleadoID string) (domain.Profile, error) {
	row := r.pool.QueryRow(ctx, "SELECT "+profileColumns+" FROM perfiles WHERE empleado_id = $1", empleadoID)
	profile, err := scanProfile(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Profile{}, domain.ErrProfileNotFound
	}
	if err != nil {
		return domain.Profile{}, fmt.Errorf("no fue posible consultar el perfil: %w", err)
	}
	return profile, nil
}

func (r *ProfileRepository) UpdateContactInfo(ctx context.Context, empleadoID, telefono, direccion, ciudad, biografia string) (domain.Profile, error) {
	row := r.pool.QueryRow(
		ctx,
		`UPDATE perfiles
		    SET telefono = $2, direccion = $3, ciudad = $4, biografia = $5
		  WHERE empleado_id = $1
		  RETURNING `+profileColumns,
		empleadoID, telefono, direccion, ciudad, biografia,
	)
	profile, err := scanProfile(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Profile{}, domain.ErrProfileNotFound
	}
	if err != nil {
		return domain.Profile{}, fmt.Errorf("no fue posible actualizar el perfil: %w", err)
	}
	return profile, nil
}

func (r *ProfileRepository) ListAll(ctx context.Context) ([]domain.Profile, error) {
	rows, err := r.pool.Query(ctx, "SELECT "+profileColumns+" FROM perfiles ORDER BY fecha_creacion DESC")
	if err != nil {
		return nil, fmt.Errorf("no fue posible listar los perfiles: %w", err)
	}
	defer rows.Close()

	profiles := make([]domain.Profile, 0)
	for rows.Next() {
		profile, err := scanProfile(rows)
		if err != nil {
			return nil, fmt.Errorf("no fue posible leer un perfil: %w", err)
		}
		profiles = append(profiles, profile)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("no fue posible leer los perfiles: %w", err)
	}
	return profiles, nil
}
