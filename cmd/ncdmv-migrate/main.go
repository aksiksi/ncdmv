package main

import (
	"flag"
	"log"
	"os"

	models "github.com/aksiksi/ncdmv/pkg/models"
)

const (
	dbPathEnvVar = "NCDMV_DB_PATH"
)

var (
	down           = flag.Bool("down", false, "up or down")
	count          = flag.Int("count", 0, "number of migrations")
	dbPathFlag     = flag.String("db_path", "./database/ncdmv.db", "path to SQLite DB file")
	migrationsPath = flag.String("migrations_path", "./database/migrations", "path to migrations directory")
)

func main() {
	flag.Parse()
	dbPath := os.Getenv(dbPathEnvVar)
	if dbPath == "" {
		if *dbPathFlag != "" {
			dbPath = *dbPathFlag
		} else {
			log.Fatalf("No DB path specified")
		}
	}
	if err := models.RunMigrations(dbPath, *migrationsPath, *count, *down); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
}
