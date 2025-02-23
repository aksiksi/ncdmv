package ncdmv

import (
	"context"
	"os/exec"
	"path"
	"testing"
	"time"
)

func isChromeAvailable() bool {
	_, err := exec.LookPath("google-chrome")
	return err == nil
}

func getDBPath(t *testing.T) string {
	d := t.TempDir()
	return path.Join(d, "ncdmv.db")
}

// Smoke integration that makes sure the E2E path works by running a simple
// search.
func TestSmoke(t *testing.T) {
	if !isChromeAvailable() {
		t.Skip("Integration test requires Chrome")
	}

	ctx := context.Background()

	client, chromeCtx, cleanup, err := NewClientFromOptions(ctx, ClientOptions{
		DatabasePath:  getDBPath(t),
		StopOnFailure: true,
		Headless:      true,
		DisableGpu:    true,
		Debug:         true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	if _, err := client.RunForLocations(chromeCtx, AppointmentTypeDriverLicense, []Location{LocationCary}, 3*time.Minute); err != nil {
		t.Error(err)
	}
}
