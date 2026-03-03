package tools

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/jackc/pgx/v5/pgxpool"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func InitDB(cfg configuration.Config) *gorm.DB {
	var db *gorm.DB
	var err error

	if cfg.DBSource != "" {
		db, err = openFromDSN(cfg.DBSource)
	} else {
		db, err = openFromParts(cfg)
	}

	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	return db
}

// openFromDSN opens the database using a single URL (DB_SOURCE or DATABASE_URL). Driver is inferred from scheme.
func openFromDSN(dsn string) (*gorm.DB, error) {
	dsnLower := strings.ToLower(strings.TrimSpace(dsn))
	switch {
	case strings.HasPrefix(dsnLower, "postgresql://") || strings.HasPrefix(dsnLower, "postgres://"):
		return gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
			NamingStrategy: schema.NamingStrategy{
				TablePrefix:   "",
				SingularTable: false,
				NoLowerCase:   false,
				NameReplacer:  nil,
			},
		})
	case strings.HasPrefix(dsnLower, "sqlserver://"):
		return gorm.Open(sqlserver.Open(dsn), &gorm.Config{})
	default:
		return nil, fmt.Errorf("unsupported DSN scheme (use postgres://, postgresql://, or sqlserver://)")
	}
}

func openFromParts(cfg configuration.Config) (*gorm.DB, error) {
	switch cfg.DBType {
	case "postgres":
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
			cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort)
		return gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
			NamingStrategy: schema.NamingStrategy{
				TablePrefix:   "",
				SingularTable: false,
				NoLowerCase:   false,
				NameReplacer:  nil,
			},
		})
	case "sqlserver":
		dsn := fmt.Sprintf("sqlserver://%s:%s@%s:%s?database=%s",
			cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)
		return gorm.Open(sqlserver.Open(dsn), &gorm.Config{})
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.DBType)
	}
}

func CloseDB(db *gorm.DB) {
	dbSQL, err := db.DB()
	if err != nil {
		log.Fatalf("failed to close database: %v", err)
	}
	dbSQL.Close()
}

// InitPgxPool creates a pgx connection pool for PostgreSQL. Returns (nil, nil) when DB is not postgres (e.g. sqlserver).
// Used by sqlc-generated code (ArticlesRepositorySQLC). Call pool.Close() when shutting down.
func InitPgxPool(cfg configuration.Config) (*pgxpool.Pool, error) {
	connStr := configuration.DatabaseURL(cfg)
	if connStr == "" {
		return nil, nil
	}
	dsnLower := strings.ToLower(strings.TrimSpace(connStr))
	if !strings.HasPrefix(dsnLower, "postgres://") && !strings.HasPrefix(dsnLower, "postgresql://") {
		return nil, nil // not postgres, skip pool (e.g. sqlserver)
	}
	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		return nil, fmt.Errorf("pgx pool: %w", err)
	}
	return pool, nil
}
