package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunAlwaysIncludesDesktopChecksEvenWithoutPodman(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	report := Run()

	seen := map[string]bool{}
	for _, check := range report.Checks {
		seen[check.Name] = true
	}
	for _, name := range []string{
		"unprivileged-process",
		"rootless-podman-service",
		"qemu-img",
		"vm-storage",
		"vm-image-catalog",
		"system-libvirt",
		"desktop-notifications",
		"terminal-launcher",
	} {
		if !seen[name] {
			t.Fatalf("missing doctor check %q: %#v", name, report.Checks)
		}
	}
}

func TestVMStorageCheckRejectsSymlinkedRoots(t *testing.T) {
	root := t.TempDir()
	imageRoot := filepath.Join(root, "images")
	if err := os.MkdirAll(imageRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "state-target")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	stateRoot := filepath.Join(root, "state")
	if err := os.Symlink(target, stateRoot); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LPIC_DAILY_VM_IMAGE_DIR", imageRoot)
	t.Setenv("LPIC_DAILY_VM_STATE_DIR", stateRoot)

	check := vmStorageCheck()
	if check.Status != "warn" || !strings.Contains(check.Detail, "symbolic link") {
		t.Fatalf("vmStorageCheck() = %#v, want symlink warning", check)
	}
}

func TestVMStorageCheckRejectsNonTraversableRoot(t *testing.T) {
	root := t.TempDir()
	imageRoot := filepath.Join(root, "images")
	stateRoot := filepath.Join(root, "state")
	if err := os.MkdirAll(imageRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(stateRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(imageRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LPIC_DAILY_VM_IMAGE_DIR", imageRoot)
	t.Setenv("LPIC_DAILY_VM_STATE_DIR", stateRoot)

	check := vmStorageCheck()
	if check.Status != "warn" || !strings.Contains(check.Detail, "not traversable") {
		t.Fatalf("vmStorageCheck() = %#v, want traversal warning", check)
	}
}
