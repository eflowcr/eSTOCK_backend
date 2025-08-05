package tools

import (
	"fmt"
	"log"

	"github.com/eflowcr/eSTOCK_backend/configuration"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func InitDB() *gorm.DB {
	var db *gorm.DB
	var err error

	switch configuration.DBType {
	case "postgres":
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
			configuration.DBHost, configuration.DBUser, configuration.DBPassword, configuration.DBName, configuration.DBPort)
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		})
	case "sqlserver":
		dsn := fmt.Sprintf("sqlserver://%s:%s@%s:%s?database=%s",
			configuration.DBUser, configuration.DBPassword, configuration.DBHost, configuration.DBPort, configuration.DBName)
		db, err = gorm.Open(sqlserver.Open(dsn), &gorm.Config{})
	default:
		log.Fatalf("unsupported database type: %s", configuration.DBType)
	}

	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	return db
}

func CloseDB(db *gorm.DB) {
	dbSQL, err := db.DB()
	if err != nil {
		log.Fatalf("failed to close database: %v", err)
	}
	dbSQL.Close()
}
