package checker_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Loe159/lpic-daily/internal/checker"
	"github.com/Loe159/lpic-daily/internal/runner"
)

type storageProbe struct {
	snapshot runner.StorageSnapshot
	err      error
}

func (probe storageProbe) Stat(
	context.Context,
	runner.Instance,
	string,
) (runner.FileInfo, error) {
	return runner.FileInfo{}, errors.New("unused")
}

func (probe storageProbe) ReadFile(
	context.Context,
	runner.Instance,
	string,
	int64,
) ([]byte, error) {
	return nil, errors.New("unused")
}

func (probe storageProbe) Processes(
	context.Context,
	runner.Instance,
) ([]runner.Process, error) {
	return nil, errors.New("unused")
}

func (probe storageProbe) Exec(
	context.Context,
	runner.Instance,
	runner.ExecRequest,
) (runner.ExecResult, error) {
	return runner.ExecResult{}, errors.New("unused")
}

func (probe storageProbe) StorageSnapshot(
	context.Context,
	runner.Instance,
) (runner.StorageSnapshot, error) {
	return probe.snapshot, probe.err
}

func TestBlockDeviceStateMatchesFinalStorageState(t *testing.T) {
	swapActive := true
	probe := storageProbe{
		snapshot: runner.StorageSnapshot{
			Devices: []runner.BlockDevice{
				{
					Path:           "/dev/vdb",
					DeviceType:     "disk",
					PartitionTable: "gpt",
					SizeBytes:      512 * 1024 * 1024,
				},
				{
					Path:       "/dev/vdb1",
					DeviceType: "part",
					Filesystem: "ext4",
					Mountpoint: "/srv/data",
				},
				{
					Path:       "/dev/vdb2",
					DeviceType: "part",
					Filesystem: "swap",
					SwapActive: true,
				},
			},
		},
	}
	instance := runner.Instance{ID: "fake"}

	checks := []checker.Check{
		checker.BlockDeviceState{
			CheckID:        "disk-gpt",
			Path:           "/dev/vdb",
			DeviceType:     "disk",
			PartitionTable: "gpt",
		},
		checker.BlockDeviceState{
			CheckID:    "data-ext4",
			Path:       "/dev/vdb1",
			DeviceType: "part",
			Filesystem: "ext4",
			Mountpoint: "/srv/data",
		},
		checker.BlockDeviceState{
			CheckID:    "swap-active",
			Path:       "/dev/vdb2",
			DeviceType: "part",
			Filesystem: "swap",
			SwapActive: &swapActive,
		},
	}

	results, err := checker.EvaluateAll(context.Background(), probe, instance, checks)
	if err != nil {
		t.Fatalf("EvaluateAll() error = %v", err)
	}
	for _, result := range results {
		if !result.Pass {
			t.Fatalf("check failed: %#v", result)
		}
	}
}

func TestBlockDeviceStateReportsMismatchAndMissingDevice(t *testing.T) {
	probe := storageProbe{
		snapshot: runner.StorageSnapshot{
			Devices: []runner.BlockDevice{{
				Path:       "/dev/vdb1",
				DeviceType: "part",
				Filesystem: "xfs",
			}},
		},
	}
	instance := runner.Instance{ID: "fake"}

	wrongFS := checker.BlockDeviceState{
		CheckID:    "wrong-fs",
		Path:       "/dev/vdb1",
		Filesystem: "ext4",
	}
	if result := wrongFS.Evaluate(context.Background(), probe, instance); result.Pass ||
		result.Detail == "" {
		t.Fatalf("mismatch result = %#v", result)
	}

	missing := checker.BlockDeviceState{
		CheckID: "missing",
		Path:    "/dev/vdc",
	}
	if result := missing.Evaluate(context.Background(), probe, instance); result.Pass ||
		result.Detail == "" {
		t.Fatalf("missing result = %#v", result)
	}
}
