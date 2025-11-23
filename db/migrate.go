package db

import (
	"database/sql"
	"embed"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"go.uber.org/zap"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

func SetupPostgres(pool *pgxpool.Pool, logger *zap.Logger) {
	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		logger.Fatal("cannot set goose dialect", zap.Error(err))
	}

	var db *sql.DB = stdlib.OpenDBFromPool(pool)

	if err := goose.Up(db, "migrations"); err != nil {
		logger.Fatal("migration failed", zap.Error(err))
	}

	logger.Info("migrations applied successfully")
}
