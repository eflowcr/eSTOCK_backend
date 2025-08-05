package tools

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq" // driver PostgreSQL
)

func ConnectDB() (*sql.DB, error) {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	host, err := Decrypt(os.Getenv("DB_HOST"))
	if err != nil {
		log.Fatalf("Error decrypting DB_HOST: %v", err)
	}

	user, err := Decrypt(os.Getenv("DB_USER"))
	if err != nil {
		log.Fatalf("Error decrypting DB_USER: %v", err)
	}

	password, err := Decrypt(os.Getenv("DB_PASSWORD"))
	if err != nil {
		log.Fatalf("Error decrypting DB_PASSWORD: %v", err)
	}

	port, err := Decrypt(os.Getenv("DB_PORT"))
	if err != nil {
		log.Fatalf("Error decrypting DB_PORT: %v", err)
	}

	dbName, err := Decrypt(os.Getenv("DB_NAME"))
	if err != nil {
		log.Fatalf("Error decrypting DB_NAME: %v", err)
	}

	// Formato correcto para PostgreSQL
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		host, user, password, dbName, port)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("error abriendo conexión: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error al verificar la conexión: %w", err)
	}

	log.Println("Conectado a la base de datos PostgreSQL correctamente.")
	return db, nil
}
