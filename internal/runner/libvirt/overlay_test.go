package libvirt

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Loe159/lpic-daily/internal/runner"
)

type commandCall struct {
	Name string
	Args []string
}

type fakeCommands struct {
	Calls  []commandCall
	FailAt int
}

func (fake *fakeCommands) Run(_ context.Context, name string, args ...string) error {
	fake.Calls = append(fake.Calls, commandCall{Name: name, Args: append([]string(nil), args...)})
	if fake.FailAt != 0 && len(fake.Calls) == fake.FailAt {
		return errors.New("simulated qemu-img failure")
	}
	return nil
}

func testImage(t *testing.T, imageRoot string) ImageDescriptor {
	t.Helper()
	path := filepath.Join(imageRoot, "fedora", "base.qcow2")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	content := []byte("trusted-base-image-fixture")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	digest := sha256.Sum256(content)
	return ImageDescriptor{
		ID:            "fedora-44-x86_64-v1",
		Path:          path,
		SHA256:        hex.EncodeToString(digest[:]),
		Format:        "qcow2",
		Architecture:  "x86_64",
		Distribution:  "fedora",
		VirtualSizeMB: 8192,
		FirmwareModes: []runner.FirmwareMode{runner.FirmwareBIOS, runner.FirmwareUEFI},
	}
}

func TestOverlayManagerCreatesStructuredQEMUImgCalls(t *testing.T) {
	root := t.TempDir()
	imageRoot := filepath.Join(root, "images")
	stateRoot := filepath.Join(root, "state")
	image := testImage(t, imageRoot)
	commands := &fakeCommands{}
	manager := OverlayManager{ImageRoot: imageRoot, StateRoot: stateRoot, Commands: commands}

	paths, err := manager.Create(
		context.Background(),
		"lpic-daily-storage-abc123",
		image,
		runner.MachineDefinition{
			Firmware: runner.FirmwareUEFI,
			ExtraDisks: []runner.VirtualDisk{
				{ID: "data", SizeMB: 512},
				{ID: "swap", SizeMB: 256},
			},
		},
	)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if len(commands.Calls) != 3 {
		t.Fatalf("qemu-img calls = %d, want 3", len(commands.Calls))
	}
	directoryInfo, err := os.Stat(paths.Directory)
	if err != nil {
		t.Fatalf("Stat(instance directory) error = %v", err)
	}
	if got := directoryInfo.Mode().Perm(); got != 0o711 {
		t.Fatalf("instance directory mode = %04o, want 0711 for qemu:///system traversal", got)
	}
	wantRoot := []string{
		"create", "-f", "qcow2", "-F", "qcow2", "-b", image.Path, paths.RootDisk,
	}
	if !reflect.DeepEqual(commands.Calls[0].Args, wantRoot) {
		t.Fatalf("root args = %#v, want %#v", commands.Calls[0].Args, wantRoot)
	}
	if !strings.HasSuffix(paths.Extra["data"], "disk-data.qcow2") {
		t.Fatalf("data disk path = %q", paths.Extra["data"])
	}
	if commands.Calls[1].Args[len(commands.Calls[1].Args)-1] != "512M" {
		t.Fatalf("data disk size args = %#v", commands.Calls[1].Args)
	}
	if err := manager.Destroy("lpic-daily-storage-abc123"); err != nil {
		t.Fatalf("Destroy() error = %v", err)
	}
	if _, err := os.Stat(paths.Directory); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("instance directory still exists, stat error = %v", err)
	}
}

func TestOverlayManagerCleansPartialFailure(t *testing.T) {
	root := t.TempDir()
	imageRoot := filepath.Join(root, "images")
	stateRoot := filepath.Join(root, "state")
	image := testImage(t, imageRoot)
	commands := &fakeCommands{FailAt: 2}
	manager := OverlayManager{ImageRoot: imageRoot, StateRoot: stateRoot, Commands: commands}

	_, err := manager.Create(
		context.Background(),
		"lpic-daily-storage-fail",
		image,
		runner.MachineDefinition{
			Firmware:   runner.FirmwareBIOS,
			ExtraDisks: []runner.VirtualDisk{{ID: "data", SizeMB: 128}},
		},
	)
	if err == nil || !strings.Contains(err.Error(), "scratch disk data") {
		t.Fatalf("Create() error = %v", err)
	}
	directory := filepath.Join(stateRoot, "lpic-daily-storage-fail")
	if _, err := os.Stat(directory); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("partial directory survived failed create: %v", err)
	}
}

func TestOverlayManagerRejectsChecksumMismatchBeforeQEMUImg(t *testing.T) {
	root := t.TempDir()
	imageRoot := filepath.Join(root, "images")
	stateRoot := filepath.Join(root, "state")
	image := testImage(t, imageRoot)
	image.SHA256 = strings.Repeat("0", 64)
	commands := &fakeCommands{}
	manager := OverlayManager{ImageRoot: imageRoot, StateRoot: stateRoot, Commands: commands}

	_, err := manager.Create(
		context.Background(),
		"lpic-daily-checksum-bad",
		image,
		runner.MachineDefinition{Firmware: runner.FirmwareBIOS},
	)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("Create() error = %v", err)
	}
	if len(commands.Calls) != 0 {
		t.Fatalf("qemu-img called before checksum validation: %#v", commands.Calls)
	}
}

func TestDestroyRefusesSymlinkedInstanceDirectory(t *testing.T) {
	root := t.TempDir()
	imageRoot := filepath.Join(root, "images")
	stateRoot := filepath.Join(root, "state")
	target := filepath.Join(root, "target")
	if err := os.MkdirAll(stateRoot, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.MkdirAll(target, 0o700); err != nil {
		t.Fatalf("MkdirAll() target error = %v", err)
	}
	name := "lpic-daily-symlink-test"
	if err := os.Symlink(target, filepath.Join(stateRoot, name)); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}
	manager := OverlayManager{ImageRoot: imageRoot, StateRoot: stateRoot, Commands: &fakeCommands{}}
	if err := manager.Destroy(name); err == nil || !strings.Contains(err.Error(), "symlinked") {
		t.Fatalf("Destroy() error = %v", err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("target was damaged: %v", err)
	}
}
