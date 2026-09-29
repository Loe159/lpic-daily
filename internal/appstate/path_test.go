package appstate_test

import (
	"os"
	"path/filepath"
	"strconv"
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

func TestVMRootsDefaultToSystemLibvirtImageTree(t *testing.T) {
	t.Setenv("LPIC_DAILY_VM_STATE_DIR", "")
	t.Setenv("LPIC_DAILY_VM_IMAGE_DIR", "")
	wantRoot := filepath.Join("/var/lib/libvirt/images/lpic-daily", strconv.Itoa(os.Geteuid()))
	if got := appstate.VMStorageRoot(); got != wantRoot {
		t.Fatalf("VMStorageRoot() = %q, want %q", got, wantRoot)
	}

	stateRoot, err := appstate.VMStateRoot()
	if err != nil {
		t.Fatalf("VMStateRoot() error = %v", err)
	}
	if want := filepath.Join(wantRoot, "state"); stateRoot != want {
		t.Fatalf("VMStateRoot() = %q, want %q", stateRoot, want)
	}

	imageRoot, err := appstate.VMImageRoot()
	if err != nil {
		t.Fatalf("VMImageRoot() error = %v", err)
	}
	if want := filepath.Join(wantRoot, "images"); imageRoot != want {
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

func TestVMRootsHonorAbsoluteOverrides(t *testing.T) {
	root := t.TempDir()
	state := filepath.Join(root, "state")
	images := filepath.Join(root, "images")
	t.Setenv("LPIC_DAILY_VM_STATE_DIR", state)
	t.Setenv("LPIC_DAILY_VM_IMAGE_DIR", images)

	if got, err := appstate.VMStateRoot(); err != nil || got != state {
		t.Fatalf("VMStateRoot() = %q, %v", got, err)
	}
	if got, err := appstate.VMImageRoot(); err != nil || got != images {
		t.Fatalf("VMImageRoot() = %q, %v", got, err)
	}
}

func TestVMRootsRequireAbsoluteOverrides(t *testing.T) {
	t.Setenv("LPIC_DAILY_VM_STATE_DIR", "relative/state")
	if _, err := appstate.VMStateRoot(); err == nil {
		t.Fatal("relative VM state override unexpectedly accepted")
	}
	t.Setenv("LPIC_DAILY_VM_STATE_DIR", "")
	t.Setenv("LPIC_DAILY_VM_IMAGE_DIR", "relative/images")
	if _, err := appstate.VMImageRoot(); err == nil {
		t.Fatal("relative VM image override unexpectedly accepted")
	}
}
