package cmd

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mhirii/huma-template/internal/config"
	"github.com/mhirii/huma-template/migrations"
	"github.com/mhirii/huma-template/pkg/db"
	"github.com/mhirii/huma-template/pkg/logging"
	"github.com/uptrace/bun/migrate"
)

func Migrate(action string) {
	l := logging.L()
	l.Info().Msg("Starting Migration")

	cfg := config.GetAPICfg()
	dsn := db.GetDSN(&cfg.DB)
	dbConn, err := db.New(dsn)
	if err != nil {
		l.Error().Err(err).Msg("Failed to connect to database")
		panic(err)
	}
	ctx := context.Background()
	migrator := migrate.NewMigrator(dbConn, migrations.Migrations)

	switch action {
	case "up":
		l.Info().Msg("Migrating up")

		if err := migrator.Init(ctx); err != nil {
			l.Error().Err(err).Msg("Failed to initialize migrations")
			return
		}
		if err := migrator.Lock(ctx); err != nil {
			l.Error().Err(err).Msg("Failed to lock migrations")
			return
		}
		defer migrator.Unlock(ctx)
		mg, err := migrator.Migrate(ctx)
		if err != nil {
			if mg != nil {
				l.Error().Err(err).Str("msg", mg.String()).Msg("Failed to migrate")
			} else {
				l.Error().Err(err).Msg("Failed to migrate")
			}
			return
		} else {
			if len(mg.Migrations) > 0 {
				l.Info().Msg(fmt.Sprintf("Applied %d migrations: %v", len(mg.Migrations), mg.Migrations.Applied()))
			} else {
				l.Info().Msg("No new migrations to apply")
			}
		}
		l.Info().Msg("Migrations applied successfully.")
		return
	case "down":
		l.Info().Msg("Rolling back one migration")
		if err := migrator.Init(ctx); err != nil {
			l.Error().Err(err).Msg("Failed to initialize migrations")
			return
		}
		if err := migrator.Lock(ctx); err != nil {
			l.Error().Err(err).Msg("Failed to lock migrations")
			return
		}
		defer migrator.Unlock(ctx)
		r, err := migrator.Rollback(ctx)
		if err != nil {
			l.Error().Err(err).Msg("Failed to rollback")
			return
		}
		l.Info().Msg(fmt.Sprintf("Rolled back %d migrations: %v", len(r.Migrations), r.Migrations.Applied()))
		return
	case "status":
		l.Info().Msg("Showing migration status")
		if err := migrator.Init(ctx); err != nil {
			l.Error().Err(err).Msg("Failed to initialize migrations")
			return
		}
		ms, err := migrator.MigrationsWithStatus(ctx)
		if err != nil {
			l.Error().Err(err).Msg("Failed to get migrations status")
			return
		}
		for _, m := range ms {
			status := "pending"
			if m.IsApplied() {
				status = "applied"
			}
			l.Info().Msg(fmt.Sprintf("%s %s %s", m.Name, m.Comment, status))
		}
		return

	case "help":
		fmt.Println("\033[1;36m✨ Migration Help ✨\033[0m")
		fmt.Println("\033[1;33mUsage:\033[0m \033[1;32mmigrate [up|down|status]\033[0m")
		fmt.Println("\033[1;32m  up    \033[0m- \033[0;36mMigrates the database \033[1;32mUP\033[0;36m\033[0m")
		fmt.Println("\033[1;31m  down  \033[0m- \033[0;36mMigrates the database one step \033[1;31mDOWN\033[0;36m\033[0m")
		fmt.Println("\033[1;34m  status\033[0m - \033[0;36mShows the current migration \033[1;34mSTATUS\033[0;36m\033[0m")
		return
	default:
		slog.Error("Invalid migration type")
		panic("Invalid migration type")
	}
}
