package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RunMigrations(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations dir %s: %w", dir, err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		files = append(files, entry.Name())
	}
	sort.Strings(files)

	if err := recordExistingSchema(ctx, pool, files); err != nil {
		return err
	}

	for _, name := range files {
		var applied bool
		err := pool.QueryRow(
			ctx,
			`SELECT TRUE FROM schema_migrations WHERE filename = $1`,
			name,
		).Scan(&applied)
		if err == nil {
			continue
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("check %s: %w", name, err)
		}

		sqlBytes, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}

		if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
			_ = tx.Rollback(ctx)
			if !isAlreadyExists(err) {
				return fmt.Errorf("apply %s: %w", name, err)
			}
			if _, recErr := pool.Exec(
				ctx,
				`INSERT INTO schema_migrations (filename) VALUES ($1) ON CONFLICT DO NOTHING`,
				name,
			); recErr != nil {
				return fmt.Errorf("record %s: %w", name, recErr)
			}
			slog.Info("migration already applied", "filename", name)
			continue
		}

		if _, err := tx.Exec(
			ctx,
			`INSERT INTO schema_migrations (filename) VALUES ($1)`,
			name,
		); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("record %s: %w", name, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return err
		}

		slog.Info("applied migration", "filename", name)
	}

	return nil
}

func recordExistingSchema(
	ctx context.Context,
	pool *pgxpool.Pool,
	files []string,
) error {
	var businessesExists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = 'public'
			  AND table_name = 'businesses'
		)
	`).Scan(&businessesExists); err != nil {
		return err
	}

	var recorded int
	if err := pool.QueryRow(
		ctx,
		`SELECT COUNT(*) FROM schema_migrations`,
	).Scan(&recorded); err != nil {
		return err
	}

	if !businessesExists || recorded > 0 {
		return nil
	}

	for _, name := range files {
		if _, err := pool.Exec(
			ctx,
			`INSERT INTO schema_migrations (filename) VALUES ($1) ON CONFLICT DO NOTHING`,
			name,
		); err != nil {
			return fmt.Errorf("baseline %s: %w", name, err)
		}
	}

	slog.Info("recorded existing database as already migrated")
	return nil
}

func isAlreadyExists(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}

	switch pgErr.Code {
	case "42P07", // duplicate_table
		"42710", // duplicate_object
		"42701": // duplicate_column
		return true
	default:
		return false
	}
}
