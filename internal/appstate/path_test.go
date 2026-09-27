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

func TestVMRootsUseControlledXDGDirectories(t *testing.T) {
	root := t.TempDir()
	t.Setenv("LPIC_DAILY_STATE_DIR", "")
	t.Setenv("LPIC_DAILY_VM_IMAGE_DIR", "")
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))

	stateRoot, err := appstate.VMStateRoot()
	if err != nil {
		t.Fatalf("VMStateRoot() error = %v", err)
	}
	if want := filepath.Join(root, "state", "lpic-daily", "vms"); stateRoot != want {
		t.Fatalf("VMStateRoot() = %q, want %q", stateRoot, want)
	}

	imageRoot, err := appstate.VMImageRoot()
	if err != nil {
		t.Fatalf("VMImageRoot() error = %v", err)
	}
	if want := filepath.Join(root, "data", "lpic-daily", "vm-images"); imageRoot != want {
		t.Fatalf("VMImageRoot() = %q, want %q", imageRoot, want)
	}

	catalogPath, err := appstate.VMImageCatalogPath()
	if err != nil {
		t.Fatalf("VMImageCatalogPath() error = %v", err)
	}
	if want := filepath.Join(imageRoot, "catalog.json"); catalogPath != want {
		t.Fatalf("VMImageCatalogPath() = %q, want %q", catalogPath, want)
	}
}

func TestVMImageRootRequiresAbsoluteOverride(t *testing.T) {
	t.Setenv("LPIC_DAILY_VM_IMAGE_DIR", "relative/images")
	if _, err := appstate.VMImageRoot(); err == nil {
		t.Fatal("relative VM image override unexpectedly accepted")
	}
}
