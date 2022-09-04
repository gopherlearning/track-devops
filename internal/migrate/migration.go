package migrate

import (
	"context"
	"embed"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"go.uber.org/zap"
)

// MigrateFromDir executes database migrations
func MigrateFromDir(ctx context.Context, db *pgx.Conn, migrationDir string, logger *zap.Logger) error {
	if logger == nil {
		logger, _ = zap.NewDevelopment()
	}
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	createMigrationTable := `
		CREATE TABLE IF NOT EXISTS migration(
			id          varchar(255) primary key,
			modified_at timestamp not null
		);
	`
	if _, err = tx.Exec(ctx, createMigrationTable); err != nil {
		return err
	}

	if _, err = tx.Exec(ctx, `LOCK TABLE migration;`); err != nil {
		return err
	}

	files, err := os.ReadDir(migrationDir)
	if err != nil {
		err = tx.Rollback(ctx)
		if err != nil {
			return err
		}
		return err
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	for _, f := range files {
		fileName := f.Name()

		if !strings.HasSuffix(fileName, ".sql") {
			continue
		}

		filePath := path.Join(migrationDir, fileName)

		r := tx.QueryRow(ctx, `SELECT id, modified_at FROM migration WHERE id = $1;`, fileName)

		type migrationItem struct {
			ModifiedAt time.Time
			ID         string
		}

		mi := &migrationItem{}
		err := r.Scan(&mi.ID, &mi.ModifiedAt)

		if err != nil && err != pgx.ErrNoRows {
			err = tx.Rollback(ctx)
			if err != nil {
				return err
			}
			return err
		} else if err == nil {
			continue
		}

		script, err := os.ReadFile(filePath)
		if err != nil {
			err = tx.Rollback(ctx)
			if err != nil {
				return err
			}
			return err
		}
		logger.Info(string(script))
		if _, err := tx.Exec(ctx, string(script)); err != nil {
			err = tx.Rollback(ctx)
			if err != nil {
				return err
			}
			return err
		}

		if _, err := tx.Exec(ctx,
			`INSERT INTO migration (id, modified_at) VALUES($1, $2) ON CONFLICT (id) DO UPDATE SET modified_at = $2;`,
			fileName, time.Now().UTC(),
		); err != nil {
			err = tx.Rollback(ctx)
			if err != nil {
				return err
			}
			return err
		}
	}
	return tx.Commit(ctx)
}

type PgxIface interface {
	Begin(context.Context) (pgx.Tx, error)
	Close(context.Context) error
}

// MigrateFromFS executes database migrations from emdebed files
func MigrateFromFS(ctx context.Context, db *pgxpool.Pool, migrations *embed.FS, logger *zap.Logger) error {
	if logger == nil {
		logger, _ = zap.NewDevelopment()
	}
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	createMigrationTable := `
		CREATE TABLE IF NOT EXISTS migration(
			id          varchar(255) primary key,
			modified_at timestamp not null
		);
	`
	if _, err = tx.Exec(ctx, createMigrationTable); err != nil {
		return err
	}

	if _, err = tx.Exec(ctx, `LOCK TABLE migration;`); err != nil {
		return err
	}
	files, err := migrations.ReadDir(".")
	if err != nil {
		err = tx.Rollback(ctx)
		if err != nil {
			logger.Error(err.Error())
			return err
		}
		return err
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	for _, f := range files {
		fileName := f.Name()
		r := tx.QueryRow(ctx, `SELECT id, modified_at FROM migration WHERE id = $1;`, fileName)
		type migrationItem struct {
			ModifiedAt time.Time
			ID         string
		}

		mi := &migrationItem{}
		err = r.Scan(&mi.ID, &mi.ModifiedAt)

		if err != nil && err != pgx.ErrNoRows {
			err = tx.Rollback(ctx)
			if err != nil {
				logger.Error(err.Error())
				return err
			}
			return err
		} else if err == nil {
			continue
		}
		var script []byte
		script, err = migrations.ReadFile(fileName)
		if err != nil {
			err = tx.Rollback(ctx)
			if err != nil {
				logger.Error(err.Error())
				return err
			}
			return err
		}
		logger.Info(string(script))
		if _, err := tx.Exec(ctx, string(script)); err != nil {
			logger.Error(err.Error())
			err = tx.Rollback(ctx)
			if err != nil {
				logger.Error(err.Error())
				return err
			}
			return err
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO migration (id, modified_at) VALUES($1, $2) ON CONFLICT (id) DO UPDATE SET modified_at = $2;`,
			fileName, time.Now().UTC(),
		); err != nil {
			err = tx.Rollback(ctx)
			if err != nil {
				logger.Error(err.Error())
				return err
			}
			return err
		}
	}
	return tx.Commit(ctx)
}
