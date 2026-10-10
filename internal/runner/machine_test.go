package runner_test

import (
	"testing"

	"github.com/Loe159/lpic-daily/internal/runner"
)

func TestMachineDefinitionValidation(t *testing.T) {
	valid := runner.MachineDefinition{
		Firmware: runner.FirmwareUEFI,
		ExtraDisks: []runner.VirtualDisk{
			{ID: "data", SizeMB: 512},
			{ID: "swap", SizeMB: 256},
		},
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid machine rejected: %v", err)
	}

	duplicate := valid
	duplicate.ExtraDisks = append(duplicate.ExtraDisks, runner.VirtualDisk{ID: "data", SizeMB: 128})
	if err := duplicate.Validate(); err == nil {
		t.Fatal("duplicate disk ID unexpectedly accepted")
	}

	sata := valid
	sata.ExtraDisks = []runner.VirtualDisk{{ID: "archive", SizeMB: 256, Bus: runner.VirtualDiskBusSATA}}
	if err := sata.Validate(); err != nil {
		t.Fatalf("SATA scratch disk rejected: %v", err)
	}

	badBus := valid
	badBus.ExtraDisks = []runner.VirtualDisk{{ID: "archive", SizeMB: 256, Bus: "nvme"}}
	if err := badBus.Validate(); err == nil {
		t.Fatal("unsupported NVMe bus unexpectedly accepted")
	}

	tooLarge := valid
	tooLarge.ExtraDisks = []runner.VirtualDisk{{ID: "data", SizeMB: 9000}}
	if err := tooLarge.Validate(); err == nil {
		t.Fatal("oversized scratch disk unexpectedly accepted")
	}
}
