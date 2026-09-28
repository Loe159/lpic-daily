package runner_test

import (
	"testing"
	"time"

	"github.com/Loe159/lpic-daily/internal/runner"
)

func TestDefinitionRejectsPublicOrUnknownNetworkMode(t *testing.T) {
	definition := runner.Definition{
		LabID:             "lab",
		ImageRef:          "sha256:deadbeef",
		Distribution:      "fedora",
		Network:           runner.NetworkMode("public"),
		CapabilityProfile: "baseline",
		MemoryMB:          256,
		PIDs:              128,
		Timeout:           time.Minute,
	}
	if err := definition.Validate(); err == nil {
		t.Fatal("Validate() unexpectedly accepted public network mode")
	}
}

func TestExecRequestRequiresStructuredArgv(t *testing.T) {
	if err := (runner.ExecRequest{Argv: []string{"/bin/sh", "-c", "echo ok"}}).Validate(); err != nil {
		t.Fatalf("structured argv should validate: %v", err)
	}
	if err := (runner.ExecRequest{}).Validate(); err == nil {
		t.Fatal("empty argv unexpectedly validated")
	}
}

func TestPhase1CapabilityProfilesAreAllowlisted(t *testing.T) {
	if _, err := runner.Phase1CapabilityProfile("privileged"); err == nil {
		t.Fatal("unknown/privileged profile unexpectedly accepted")
	}
	profile, err := runner.Phase1CapabilityProfile("identity-files")
	if err != nil {
		t.Fatalf("identity-files profile: %v", err)
	}
	if profile.Name != "identity-files" {
		t.Fatalf("profile name = %q", profile.Name)
	}
	want := []string{"CHOWN", "FOWNER", "FSETID"}
	if len(profile.Capabilities) != len(want) {
		t.Fatalf("capabilities = %v, want %v", profile.Capabilities, want)
	}
	for index, capability := range want {
		if profile.Capabilities[index] != capability {
			t.Fatalf("capabilities = %v, want %v", profile.Capabilities, want)
		}
	}

	shared, err := runner.Phase1CapabilityProfile("shared-files-users")
	if err != nil {
		t.Fatalf("shared-files-users profile: %v", err)
	}
	sharedWant := []string{"CHOWN", "FOWNER", "FSETID", "SETGID", "SETUID"}
	if len(shared.Capabilities) != len(sharedWant) {
		t.Fatalf("shared capabilities = %v, want %v", shared.Capabilities, sharedWant)
	}
	for index, capability := range sharedWant {
		if shared.Capabilities[index] != capability {
			t.Fatalf("shared capabilities = %v, want %v", shared.Capabilities, sharedWant)
		}
	}
}

func TestDefinitionRejectsUnsafeWritableGuestPaths(t *testing.T) {
	base := runner.Definition{
		LabID: "lab", ImageRef: "sha256:deadbeef", Distribution: "fedora",
		Network: runner.NetworkNone, CapabilityProfile: "baseline",
		MemoryMB: 256, CPUPercent: 100, PIDs: 128, Timeout: time.Minute,
	}
	for _, guestPath := range []string{"/", "relative", "/tmp/../etc", "/proc/escape", "/sys/kernel", "/dev/shm"} {
		definition := base
		definition.WritableGuestPaths = []string{guestPath}
		if err := definition.Validate(); err == nil {
			t.Fatalf("Validate() unexpectedly accepted writable guest path %q", guestPath)
		}
	}
}
