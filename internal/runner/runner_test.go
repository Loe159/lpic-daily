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

func TestDefinitionRejectsResourceLimitsAboveSchemaMaximums(t *testing.T) {
	base := runner.Definition{
		LabID: "lab", ImageRef: "sha256:deadbeef", Distribution: "fedora",
		Network: runner.NetworkNone, CapabilityProfile: "baseline",
		MemoryMB: 256, CPUPercent: 100, PIDs: 128, Timeout: time.Minute,
	}

	tests := []struct {
		name   string
		mutate func(*runner.Definition)
	}{
		{name: "memory", mutate: func(definition *runner.Definition) { definition.MemoryMB = 16385 }},
		{name: "pids", mutate: func(definition *runner.Definition) { definition.PIDs = 4097 }},
		{name: "timeout", mutate: func(definition *runner.Definition) { definition.Timeout = 2*time.Hour + time.Second }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			definition := base
			test.mutate(&definition)
			if err := definition.Validate(); err == nil {
				t.Fatal("Validate() unexpectedly accepted an above-schema resource limit")
			}
		})
	}
}
