package configuration

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

var (
	// Postgres config
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBType     string
	Key        string
	Secret     string
	Version    string
)

func LoadConfig() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	DBHost = os.Getenv("DB_HOST")
	DBPort = os.Getenv("DB_PORT")
	DBUser = os.Getenv("DB_USER")
	DBPassword = os.Getenv("DB_PASSWORD")
	DBName = os.Getenv("DB_NAME")
	DBType = os.Getenv("DB_TYPE")
	Key = os.Getenv("Key")
	Secret = os.Getenv("Secret")
	Version = os.Getenv("Version")
}
