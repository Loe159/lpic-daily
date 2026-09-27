package doctor

import "testing"

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
