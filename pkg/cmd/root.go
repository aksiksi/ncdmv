package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/exp/slog"
	_ "modernc.org/sqlite"

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

	clientOpts := ncdmv.ClientOptions{
		DatabasePath:      args.DatabasePath,
		DiscordWebhook:    args.DiscordWebhook,
		StopOnFailure:     args.StopOnFailure,
		NotifyUnavailable: args.NotifyUnavailable,
		Headless:          args.Headless,
		DisableGpu:        args.DisableGpu,
		Debug:             args.Debug,
		DebugChrome:       args.DebugChrome,
	}

	client, err := ncdmv.NewClientFromOptions(ctx, clientOpts)
	if err != nil {
		log.Fatal(err)
	}

	return client.Start(ctx, parsedApptType, locations, args.Timeout, args.Interval)
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
