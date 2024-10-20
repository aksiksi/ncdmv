package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
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
	DebugChrome       bool
}

func parseFlags(cmd *cobra.Command) *Args {
	args := Args{}
	cmd.Flags().StringVarP(&args.ApptType, "appt-type", "t", "permit", fmt.Sprintf("appointment type (one of: %s)", ncdmv.ValidApptTypes()))
	cmd.Flags().StringVarP(&args.DatabasePath, "database-path", "d", "", "database path")
	cmd.Flags().StringSliceVarP(&args.Locations, "locations", "l", []string{"cary", "durham-east", "durham-south"}, "locations to search")
	cmd.Flags().StringVarP(&args.DiscordWebhook, "discord-webhook", "w", "", "Discord webhook URL")
	cmd.Flags().DurationVar(&args.Timeout, "timeout", 5*time.Minute, "timeout for each search, in seconds")
	cmd.Flags().DurationVar(&args.Interval, "interval", 5*time.Minute, "interval between searches")
	cmd.Flags().BoolVar(&args.StopOnFailure, "stop-on-failure", false, "if set, completely stop on failure instead of just logging")
	cmd.Flags().BoolVar(&args.NotifyUnavailable, "notify-unavailable", true, "if set, send a notification if an appointment becomes unavailable")
	cmd.Flags().BoolVar(&args.Headless, "headless", true, "run Chrome in headless mode (no GUI)")
	cmd.Flags().BoolVar(&args.DisableGpu, "disable-gpu", false, "disable GPU acceleration")
	cmd.Flags().BoolVar(&args.Debug, "debug", false, "enable debug mode")
	cmd.Flags().BoolVar(&args.DebugChrome, "debug-chrome", false, "enable debug mode for Chrome")

	cmd.MarkFlagRequired("appt-type")
	cmd.MarkFlagRequired("database-path")
	cmd.MarkFlagRequired("locations")

	return &args
}

func runCommand(args *Args) error {
	ctx := context.Background()

	// Setup logger
	level := &slog.LevelVar{}
	if args.Debug {
		level.Set(slog.LevelDebug)
	} else {
		level.Set(slog.LevelInfo)
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	}))
	slog.SetDefault(logger)
	slog.InfoContext(ctx, "Setup logger", "debug", logger.Enabled(ctx, slog.LevelDebug))

	if args.DatabasePath == "" {
		log.Fatal("--database-path must be non-empty")
	}
	if args.ApptType == "" {
		log.Fatal("--appt-type must be specified")
	}
	if len(args.Locations) == 0 {
		log.Fatalf("--locations must be specified")
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
	slog.InfoContext(ctx, "GPU support", "disabled", disableGpu)

	db, err := sql.Open("sqlite", args.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to initialize DB: %s", err)
	}
	defer db.Close()
	slog.InfoContext(ctx, "Loaded DB successfully")

	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON;"); err != nil {
		log.Fatalf("Failed to enable foreign key support: %s", err)
	}
	slog.InfoContext(ctx, "Enabled foreign key support")

	slog.InfoContext(ctx, "Running all up migrations...", "databasePath", args.DatabasePath)
	if err := models.RunMigrations(args.DatabasePath, 0 /* count */, false /* down */); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize the Chrome context and open a new window.
	ctx, cancel, err := ncdmv.NewChromeContext(ctx, args.Headless, disableGpu, args.DebugChrome)
	if err != nil {
		log.Fatalf("Failed to init Chrome context: %s", err)
	}
	defer cancel()
	slog.InfoContext(ctx, "Initialized Chrome context", "headless", args.Headless, "debug", args.DebugChrome)

	client := ncdmv.NewClient(db, args.DiscordWebhook, args.StopOnFailure, args.NotifyUnavailable)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}
	slog.InfoContext(ctx, "Created ncdmv client",
		"webhook", args.DiscordWebhook != "",
		"stopOnFailure", args.StopOnFailure,
		"notifyUnavailable", args.NotifyUnavailable,
	)

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
