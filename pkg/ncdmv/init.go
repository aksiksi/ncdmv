package ncdmv

import (
	"context"
	"database/sql"
	"fmt"

	"golang.org/x/exp/slog"
	_ "modernc.org/sqlite"

	"github.com/aksiksi/ncdmv/pkg/models"
)

type ClientOptions struct {
	DatabasePath      string
	DiscordWebhook    string
	StopOnFailure     bool
	NotifyUnavailable bool
	Headless          bool
	DisableGpu        bool
	Debug             bool
	DebugChrome       bool
}

func NewClientFromOptions(ctx context.Context, opts ClientOptions) (*Client, error) {
	if opts.DatabasePath == "" {
		return nil, fmt.Errorf("database-path must be non-empty")
	}

	disableGpu := opts.DisableGpu
	slog.InfoContext(ctx, "GPU support", "disabled", disableGpu)

	db, err := sql.Open("sqlite", opts.DatabasePath)
	if err != nil {
		return nil, fmt.Errorf("Failed to initialize DB: %w", err)
	}
	defer db.Close()
	slog.InfoContext(ctx, "Loaded DB successfully")

	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON;"); err != nil {
		return nil, fmt.Errorf("Failed to enable foreign key support: %w", err)
	}
	slog.InfoContext(ctx, "Enabled foreign key support")

	slog.InfoContext(ctx, "Running all up migrations...", "databasePath", opts.DatabasePath)
	if err := models.RunMigrations(opts.DatabasePath, 0 /* count */, false /* down */); err != nil {
		return nil, fmt.Errorf("Failed to run migrations: %w", err)
	}

	// Initialize the Chrome context and open a new window.
	ctx, cancel, err := NewChromeContext(ctx, opts.Headless, disableGpu, opts.DebugChrome)
	if err != nil {
		return nil, fmt.Errorf("Failed to init Chrome context: %w", err)
	}
	defer cancel()
	slog.InfoContext(ctx, "Initialized Chrome context", "headless", opts.Headless, "debug", opts.DebugChrome)

	client := NewClient(db, opts.DiscordWebhook, opts.StopOnFailure, opts.NotifyUnavailable)
	slog.InfoContext(ctx, "Created ncdmv client",
		"webhook", opts.DiscordWebhook != "",
		"stopOnFailure", opts.StopOnFailure,
		"notifyUnavailable", opts.NotifyUnavailable,
	)

	return client, nil
}
