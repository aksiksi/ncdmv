package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	ncdmv "github.com/aksiksi/ncdmv/pkg/lib"
)

const (
	disableGpuEnvVar = "NCDMV_DISABLE_GPU"
)

var (
	apptType       = flag.String("appt_type", "permit", fmt.Sprintf("appointment type (options: %s)", strings.Join(ncdmv.ValidApptTypes(), ",")))
	databasePath   = flag.String("database_path", "./ncdmv.db", "path to database file")
	locations      = flag.String("locations", "cary,durham-east,durham-south", fmt.Sprintf("comma-seperated list of locations to check (options: %s)", strings.Join(ncdmv.ValidLocations(), ",")))
	discordWebhook = flag.String("discord_webhook", "", "Discord webhook URL for notifications (optional)")
	timeout        = flag.Int("timeout", 60, "timeout, in seconds")
	stopOnFailure  = flag.Bool("stop_on_failure", false, "if true, stop completely on a failure instead of just logging")
	interval       = flag.Int("interval", 30, "interval between checks, in minutes")
	debug          = flag.Bool("debug", false, "enable debug mode")
	headless       = flag.Bool("headless", true, "enable headless browser")
	disableGpu     = flag.Bool("disable_gpu", false, "disable GPU")
)

func main() {
	flag.Parse()

	ctx := context.Background()

	if *apptType == "" {
		log.Fatal("appt_type must be specified")
	}
	if *locations == "" {
		log.Fatalf("locations list must be specified: %s", *locations)
	}

	parsedApptType := ncdmv.StringToAppointmentType(*apptType)
	if parsedApptType == ncdmv.AppointmentTypeInvalid {
		log.Fatalf("Invalid appointment type specified: %q", *apptType)
	}

	var parsedLocations []ncdmv.Location
	for _, location := range strings.Split(*locations, ",") {
		parsedLocation := ncdmv.StringToLocation(location)
		if parsedLocation == ncdmv.LocationInvalid {
			log.Fatalf("Invalid location specified: %q", location)
		}
		parsedLocations = append(parsedLocations, parsedLocation)
	}

	disableGpu := os.Getenv(disableGpuEnvVar) != "" || *disableGpu

	db, err := sql.Open("sqlite", *databasePath)
	if err != nil {
		log.Fatalf("Failed to initialize DB: %s", err)
	}
	defer db.Close()

	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON;"); err != nil {
		log.Fatalf("Failed to enable foreign key support: %s", err)
	}

	// Initialize the Chrome context and open a new window.
	ctx, cancel, err := ncdmv.NewChromeContext(ctx, *headless, disableGpu, *debug)
	if err != nil {
		log.Fatalf("Failed to init Chrome context: %s", err)
	}
	defer cancel()

	client := ncdmv.NewClient(db, *discordWebhook, *stopOnFailure)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}

	parsedTimeout := time.Duration(*timeout) * time.Second
	parsedInterval := time.Duration(*interval) * time.Minute

	if err := client.Start(ctx, parsedApptType, parsedLocations, parsedTimeout, parsedInterval); err != nil {
		log.Fatal(err)
	}
}
