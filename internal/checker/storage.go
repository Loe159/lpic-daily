package checker

import (
	"context"
	"errors"
	"fmt"

	"github.com/Loe159/lpic-daily/internal/runner"
)

type BlockDeviceState struct {
	CheckID        string
	Path           string
	DeviceType     string
	Filesystem     string
	PartitionTable string
	Mountpoint     string
	SwapActive     *bool
}

func (check BlockDeviceState) ID() string {
	return check.CheckID
}

func (check BlockDeviceState) Evaluate(
	ctx context.Context,
	probe Probe,
	instance runner.Instance,
) Result {
	storage, ok := probe.(runner.StorageProbe)
	if !ok {
		return Result{
			CheckID: check.ID(),
			Err:     errors.New("runner does not expose structured storage state"),
		}
	}

	snapshot, err := storage.StorageSnapshot(ctx, instance)
	if err != nil {
		return Result{CheckID: check.ID(), Err: err}
	}
	for _, device := range snapshot.Devices {
		if device.Path != check.Path {
			continue
		}
		if check.DeviceType != "" && device.DeviceType != check.DeviceType {
			return mismatch(check.ID(), "device_type", device.DeviceType, check.DeviceType)
		}
		if check.Filesystem != "" && device.Filesystem != check.Filesystem {
			return mismatch(check.ID(), "filesystem", device.Filesystem, check.Filesystem)
		}
		if check.PartitionTable != "" && device.PartitionTable != check.PartitionTable {
			return mismatch(check.ID(), "partition_table", device.PartitionTable, check.PartitionTable)
		}
		if check.Mountpoint != "" && device.Mountpoint != check.Mountpoint {
			return mismatch(check.ID(), "mountpoint", device.Mountpoint, check.Mountpoint)
		}
		if check.SwapActive != nil && device.SwapActive != *check.SwapActive {
			return mismatch(
				check.ID(),
				"swap_active",
				fmt.Sprintf("%t", device.SwapActive),
				fmt.Sprintf("%t", *check.SwapActive),
			)
		}
		return Result{
			CheckID: check.ID(),
			Pass:    true,
			Detail:  fmt.Sprintf("block device %s matches expected state", check.Path),
		}
	}

	return Result{
		CheckID: check.ID(),
		Pass:    false,
		Detail:  fmt.Sprintf("block device %s not found", check.Path),
	}
}

func mismatch(checkID, field, got, want string) Result {
	return Result{
		CheckID: checkID,
		Pass:    false,
		Detail:  fmt.Sprintf("%s=%q, expected=%q", field, got, want),
	}
}
