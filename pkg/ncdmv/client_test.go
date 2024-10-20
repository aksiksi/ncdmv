package ncdmv

import (
	"context"
	"path"
	"testing"
	"time"
)

func getDBPath(t *testing.T) string {
	d := t.TempDir()
	return path.Join(d, "ncdmv.db")
}

func TestSmoke(t *testing.T) {
	ctx := context.Background()

	client, err := NewClientFromOptions(ctx, ClientOptions{
		DatabasePath:  getDBPath(t),
		StopOnFailure: true,
		Headless:      true,
		DisableGpu:    true,
		Debug:         true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := client.RunForLocations(ctx, AppointmentTypeDriverLicense, []Location{LocationCary}, 3*time.Minute); err != nil {
		t.Error(err)
	}
}
