package podman

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Loe159/lpic-daily/internal/runner"
	"go.podman.io/podman/v6/pkg/specgen"
)

func validDefinition() runner.Definition {
	return runner.Definition{
		LabID:             "104.5.shared-dropbox",
		ImageRef:          "localhost/lpic-daily/fedora:phase1",
		Distribution:      "fedora",
		Network:           runner.NetworkNone,
		CapabilityProfile: "identity-files",
		MemoryMB:          256,
		PIDs:              128,
		Timeout:           20 * time.Minute,
	}
}

func TestBuildSpecIsFailClosed(t *testing.T) {
	definition := validDefinition()
	spec, err := buildSpec(definition, "lpic-daily-test")
	if err != nil {
		t.Fatalf("buildSpec() error = %v", err)
	}

	if spec.Privileged == nil || *spec.Privileged {
		t.Fatal("container must explicitly be non-privileged")
	}
	if spec.NoNewPrivileges == nil || !*spec.NoNewPrivileges {
		t.Fatal("no-new-privileges must be enabled")
	}
	if len(spec.CapDrop) != 1 || spec.CapDrop[0] != "ALL" {
		t.Fatalf("cap drop = %v, want [ALL]", spec.CapDrop)
	}
	if spec.NetNS.NSMode != specgen.NoNetwork {
		t.Fatalf("network namespace = %q, want none", spec.NetNS.NSMode)
	}
	if spec.PidNS.NSMode != specgen.Private || spec.IpcNS.NSMode != specgen.Private || spec.UtsNS.NSMode != specgen.Private {
		t.Fatal("PID/IPC/UTS namespaces must be private")
	}
	if len(spec.Mounts) != 0 || len(spec.Volumes) != 0 || len(spec.Devices) != 0 {
		t.Fatal("Phase 1 spec unexpectedly contains mounts, volumes, or devices")
	}
	if spec.ImageVolumeMode != "ignore" {
		t.Fatalf("image volume mode = %q, want ignore", spec.ImageVolumeMode)
	}
	if spec.ResourceLimits == nil || spec.ResourceLimits.Memory == nil || spec.ResourceLimits.Memory.Limit == nil {
		t.Fatal("memory limit is required")
	}
	if got, want := *spec.ResourceLimits.Memory.Limit, int64(256*1024*1024); got != want {
		t.Fatalf("memory limit = %d, want %d", got, want)
	}
	if spec.ResourceLimits.Pids == nil || spec.ResourceLimits.Pids.Limit != 128 {
		t.Fatal("PID limit is required")
	}
}

func TestBuildSpecRejectsIsolatedUntilImplemented(t *testing.T) {
	definition := validDefinition()
	definition.Network = runner.NetworkIsolated

	_, err := buildSpec(definition, "test")
	if !errors.Is(err, runner.ErrNotSupported) {
		t.Fatalf("error = %v, want ErrNotSupported", err)
	}
}

func TestInstanceNameIsSafeAndNamespaced(t *testing.T) {
	name, err := instanceName("../../Shared DropBox !!")
	if err != nil {
		t.Fatalf("instanceName() error = %v", err)
	}
	if !strings.HasPrefix(name, "lpic-daily-") {
		t.Fatalf("name = %q", name)
	}
	if strings.ContainsAny(name, "/ !") {
		t.Fatalf("unsafe container name = %q", name)
	}
}

func TestDefaultURIUsesXDGRuntimeDir(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", "/run/user/4242")
	if got, want := DefaultURI(), "unix:///run/user/4242/podman/podman.sock"; got != want {
		t.Fatalf("DefaultURI() = %q, want %q", got, want)
	}
}
