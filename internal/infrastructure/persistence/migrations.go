package persistence

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

const advisoryLockID = 704_913

type migration struct {
	Version int
	Name    string
	SQL     string
}

// Espejo del patrón de migration.runner.ts (ms-employees) y migrations.py (ms-notifications):
// versionadas, idempotentes, con advisory lock.
var migrations = []migration{
	{
		Version: 1,
		Name:    "create_profiles_table",
		SQL: `
			CREATE TABLE perfiles (
				id UUID PRIMARY KEY,
				empleado_id VARCHAR(50) NOT NULL UNIQUE,
				nombre VARCHAR(100) NOT NULL,
				email VARCHAR(254) NOT NULL,
				telefono VARCHAR(30) NOT NULL DEFAULT '',
				direccion VARCHAR(255) NOT NULL DEFAULT '',
				ciudad VARCHAR(100) NOT NULL DEFAULT '',
				biografia TEXT NOT NULL DEFAULT '',
				archivado BOOLEAN NOT NULL DEFAULT FALSE,
				fecha_creacion TIMESTAMPTZ NOT NULL
			);
			CREATE INDEX perfiles_empleado_id_idx ON perfiles (empleado_id);

			CREATE TABLE eventos_procesados (
				id UUID PRIMARY KEY,
				procesado_en TIMESTAMPTZ NOT NULL DEFAULT NOW()
			);
		`,
	},
}

func RunMigrations(ctx context.Context, pool *pgxpool.Pool) ([]string, error) {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("no fue posible adquirir una conexión: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", advisoryLockID); err != nil {
		return nil, fmt.Errorf("no fue posible tomar el advisory lock: %w", err)
	}
	defer func() {
		_, _ = conn.Exec(ctx, "SELECT pg_advisory_unlock($1)", advisoryLockID)
	}()

	if _, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		return nil, fmt.Errorf("no fue posible crear schema_migrations: %w", err)
	}

	var applied []string
	for _, m := range migrations {
		var exists bool
		err := conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", m.Version).Scan(&exists)
		if err != nil {
			return nil, fmt.Errorf("no fue posible consultar schema_migrations: %w", err)
		}
		if exists {
			continue
		}

		tx, err := conn.Begin(ctx)
		if err != nil {
			return nil, fmt.Errorf("no fue posible iniciar la transacción de migración %d: %w", m.Version, err)
		}
		if _, err := tx.Exec(ctx, m.SQL); err != nil {
			_ = tx.Rollback(ctx)
			return nil, fmt.Errorf("migración %d (%s) falló: %w", m.Version, m.Name, err)
		}
		if _, err := tx.Exec(ctx, "INSERT INTO schema_migrations (version, name) VALUES ($1, $2)", m.Version, m.Name); err != nil {
			_ = tx.Rollback(ctx)
			return nil, fmt.Errorf("no fue posible registrar la migración %d: %w", m.Version, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("no fue posible confirmar la migración %d: %w", m.Version, err)
		}
		applied = append(applied, fmt.Sprintf("%d - %s", m.Version, m.Name))
	}
	return applied, nil
}
