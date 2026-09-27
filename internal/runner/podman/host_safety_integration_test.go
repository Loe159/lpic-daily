//go:build integration

package podman_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Loe159/lpic-daily/internal/lab"
	podmanrunner "github.com/Loe159/lpic-daily/internal/runner/podman"
)

func TestHostFilesystemSentinelSurvivesDestructiveLab(t *testing.T) {
	if os.Getenv("LPIC_DAILY_RUN_PODMAN_INTEGRATION") != "1" {
		t.Skip("set LPIC_DAILY_RUN_PODMAN_INTEGRATION=1 to run the real rootless Podman isolation test")
	}

	sentinel, err := os.CreateTemp("", "lpic-daily-host-sentinel-*")
	if err != nil {
		t.Fatalf("CreateTemp() error = %v", err)
	}
	sentinelPath := sentinel.Name()
	const sentinelBody = "host must survive\n"
	if _, err := sentinel.WriteString(sentinelBody); err != nil {
		_ = sentinel.Close()
		t.Fatalf("write sentinel: %v", err)
	}
	if err := sentinel.Close(); err != nil {
		t.Fatalf("close sentinel: %v", err)
	}
	defer os.Remove(sentinelPath)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	backend, err := podmanrunner.Open(ctx, "")
	if err != nil {
		t.Fatalf("Open(rootless Podman) error = %v", err)
	}

	authored := lab.Lab{
		Definition: lab.Definition{
			ID: "integration.host-filesystem-safety",
			Environment: lab.Environment{
				Backend:           "podman",
				ImageRef:          "localhost/lpic-daily/fedora-phase1:1",
				Distribution:      "fedora",
				Network:           "none",
				CapabilityProfile: "baseline",
			},
			Resources: lab.Resources{
				MemoryMB:       128,
				CPUPercent:     100,
				PIDs:           64,
				TimeoutSeconds: 60,
			},
		},
		SetupScript: fmt.Sprintf(
			"rm -rf -- %q\nprintf 'guest overwrite\\n' > %q\n",
			sentinelPath,
			sentinelPath,
		),
	}

	session, err := lab.Start(ctx, authored, backend)
	if err != nil {
		t.Fatalf("Start(destructive lab) error = %v", err)
	}
	if err := assertHostSentinel(sentinelPath, sentinelBody); err != nil {
		_ = session.Close(context.Background())
		t.Fatal(err)
	}

	if err := session.Close(ctx); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := assertHostSentinel(sentinelPath, sentinelBody); err != nil {
		t.Fatal(err)
	}
}

func assertHostSentinel(path, want string) error {
	content, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("host sentinel was removed or became unreadable: %w", err)
	}
	if string(content) != want {
		return fmt.Errorf("host sentinel changed: got %q, want %q", content, want)
	}
	return nil
}
