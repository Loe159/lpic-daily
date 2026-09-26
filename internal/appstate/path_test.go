package appstate_test

import (
	"path/filepath"
	"testing"

	"github.com/Loe159/lpic-daily/internal/appstate"
)

func TestProgressDBPathHonorsDedicatedOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LPIC_DAILY_STATE_DIR", filepath.Join(dir, "state"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(dir, "ignored"))

	got, err := appstate.ProgressDBPath()
	if err != nil {
		t.Fatalf("ProgressDBPath() error = %v", err)
	}
	want := filepath.Join(dir, "state", "progress.sqlite")
	if got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}
}
