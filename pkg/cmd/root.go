package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/exp/slog"
	_ "modernc.org/sqlite"

	"github.com/aksiksi/ncdmv/pkg/models"
	"github.com/aksiksi/ncdmv/pkg/ncdmv"
)

type Args struct {
	ApptType          string
	DatabasePath      string
	Locations         []string
	DiscordWebhook    string
	Timeout           time.Duration
	Interval          time.Duration
	StopOnFailure     bool
	NotifyUnavailable bool
	Headless          bool
	DisableGpu        bool
	Debug             bool
}

func parseFlags(cmd *cobra.Command) *Args {
	args := Args{}
	cmd.PersistentFlags().StringVarP(&args.ApptType, "appt-type", "t", "permit", fmt.Sprintf("appointment type (one of: %s)", ncdmv.ValidApptTypes()))
	cmd.PersistentFlags().StringVarP(&args.DatabasePath, "database-path", "d", "", "database path")
	cmd.PersistentFlags().StringSliceVarP(&args.Locations, "locations", "l", []string{"cary", "durham-east", "durham-south"}, "locations to search")
	cmd.PersistentFlags().StringVarP(&args.DiscordWebhook, "discord-webhook", "w", "", "Discord webhook URL")
	cmd.PersistentFlags().DurationVar(&args.Timeout, "timeout", 5*time.Minute, "timeout for each search, in seconds")
	cmd.PersistentFlags().DurationVar(&args.Interval, "interval", 5*time.Minute, "interval between searches")
	cmd.PersistentFlags().BoolVar(&args.StopOnFailure, "stop-on-failure", false, "if set, completely stop on failure instead of just logging")
	cmd.PersistentFlags().BoolVar(&args.NotifyUnavailable, "notify-unavailable", true, "if set, send a notification if an appointment becomes unavailable")
	cmd.PersistentFlags().BoolVar(&args.Headless, "headless", true, "run Chrome in headless mode (no GUI)")
	cmd.PersistentFlags().BoolVar(&args.DisableGpu, "disable-gpu", false, "disable GPU acceleration")
	cmd.PersistentFlags().BoolVar(&args.Debug, "debug", false, "enable debug mode")

	cmd.MarkFlagRequired("appt-type")
	cmd.MarkFlagRequired("database-path")
	cmd.MarkFlagRequired("locations")

	return &args
}

func runCommand(args *Args) error {
	ctx := context.Background()

	if args.DatabasePath == "" {
		log.Fatal("database path must be specified")
	}
	if args.ApptType == "" {
		log.Fatal("appt type must be specified")
	}
	if len(args.Locations) == 0 {
		log.Fatalf("locations list must be specified")
	}

	parsedApptType := ncdmv.StringToAppointmentType(args.ApptType)
	if parsedApptType == ncdmv.AppointmentTypeInvalid {
		log.Fatalf("Invalid appointment type specified: %q", args.ApptType)
	}

	var locations []ncdmv.Location
	for _, location := range args.Locations {
		parsedLocation := ncdmv.StringToLocation(location)
		if parsedLocation == ncdmv.LocationInvalid {
			log.Fatalf("Invalid location specified: %q", location)
		}
		locations = append(locations, parsedLocation)
	}

	disableGpu := args.DisableGpu
	slog.Info("GPU support", "disabled", disableGpu)

	db, err := sql.Open("sqlite", args.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to initialize DB: %s", err)
	}
	defer db.Close()
	slog.Info("Loaded DB successfully")

	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON;"); err != nil {
		log.Fatalf("Failed to enable foreign key support: %s", err)
	}
	slog.Info("Enabled foreign key support")

	slog.Info("Running all up migrations...", "databasePath", args.DatabasePath)
	if err := models.RunMigrations(args.DatabasePath, 0 /* count */, false /* down */); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize the Chrome context and open a new window.
	ctx, cancel, err := ncdmv.NewChromeContext(ctx, args.Headless, disableGpu, args.Debug)
	if err != nil {
		log.Fatalf("Failed to init Chrome context: %s", err)
	}
	defer cancel()
	slog.Info("Initialized Chrome context", "headless", args.Headless, "debug", args.Debug)

	client := ncdmv.NewClient(db, args.DiscordWebhook, args.StopOnFailure, args.NotifyUnavailable)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}

	if err := client.Start(ctx, parsedApptType, locations, args.Timeout, args.Interval); err != nil {
		log.Fatal(err)
	}
	return nil
}

func Execute() error {
	var rootCmd = &cobra.Command{
		Use:   "ncdmv",
		Short: "ncdmv monitors NC DMV appointments",
	}
	args := parseFlags(rootCmd)
	rootCmd.RunE = func(_ *cobra.Command, _ []string) error {
		return runCommand(args)
	}
	return rootCmd.Execute()
}
