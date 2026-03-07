package storage

import (
	"database/sql"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

// InitDB sets up the database connection pool and ensures the schema exists via migrations.
func InitDB(connStr string) *sql.DB {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to open db connection: %v", err)
	}

	// Configure connection pool to prevent overwhelming PostgreSQL (default limit 100)
	// We allow MAX_DB_CONNS to dynamically set limits based on service type.
	maxConns := 15
	if maxConnsEnv := os.Getenv("MAX_DB_CONNS"); maxConnsEnv != "" {
		if parsed, err := strconv.Atoi(maxConnsEnv); err == nil {
			maxConns = parsed
		}
	}
	db.SetMaxOpenConns(maxConns)
	db.SetMaxIdleConns(maxConns)
	db.SetConnMaxIdleTime(5 * time.Minute)
	db.SetConnMaxLifetime(10 * time.Minute)

	for i := 0; i < 10; i++ {
		err = db.Ping()
		if err == nil {
			break
		}
		log.Printf("Waiting for postgres... attempt %d/10: %v", i+1, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("Cannot connect to postgres after 10 attempts: %v", err)
	}

	// Initialize the postgres driver for the migration tool
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatalf("Could not start sql migration driver: %v", err)
	}

	// Create a new migrate instance looking at the local migrations folder
	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres", driver)
	if err != nil {
		log.Fatalf("Migration failed to init: %v", err)
	}

	// Run all 'up' migrations. Ignore ErrNoChange which just means database is already up to date.
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("An error occurred while running migrations: %v", err)
	}

	log.Println("Database migrated successfully!")

	return db
}
