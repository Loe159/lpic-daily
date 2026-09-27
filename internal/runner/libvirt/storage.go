package libvirt

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Loe159/lpic-daily/internal/runner"
)

type lsblkDocument struct {
	BlockDevices []lsblkDevice `json:"blockdevices"`
}

type lsblkDevice struct {
	Path       string        `json:"path"`
	DeviceType string        `json:"type"`
	Filesystem string        `json:"fstype"`
	Mountpoint string        `json:"mountpoint"`
	PTType     string        `json:"pttype"`
	Size       flexibleText  `json:"size"`
	Children   []lsblkDevice `json:"children"`
}

type flexibleText string

func (value *flexibleText) UnmarshalJSON(payload []byte) error {
	if bytes.Equal(payload, []byte("null")) {
		*value = ""
		return nil
	}
	var text string
	if err := json.Unmarshal(payload, &text); err == nil {
		*value = flexibleText(text)
		return nil
	}
	var number json.Number
	if err := json.Unmarshal(payload, &number); err == nil {
		*value = flexibleText(number.String())
		return nil
	}
	return fmt.Errorf("expected string, number, or null, got %s", payload)
}

func (backend *Backend) StorageSnapshot(
	ctx context.Context,
	instance runner.Instance,
) (runner.StorageSnapshot, error) {
	var lsblkOutput bytes.Buffer
	result, err := backend.Exec(ctx, instance, runner.ExecRequest{
		Argv: []string{
			"/usr/bin/lsblk",
			"--json",
			"--bytes",
			"--output",
			"PATH,TYPE,FSTYPE,MOUNTPOINT,PTTYPE,SIZE",
		},
		Stdout: &lsblkOutput,
	})
	if err != nil {
		return runner.StorageSnapshot{}, fmt.Errorf("collect lsblk state: %w", err)
	}
	if result.ExitCode != 0 {
		return runner.StorageSnapshot{}, fmt.Errorf("lsblk exited with code %d", result.ExitCode)
	}

	var swapsOutput bytes.Buffer
	result, err = backend.Exec(ctx, instance, runner.ExecRequest{
		Argv:   []string{"/usr/bin/cat", "/proc/swaps"},
		Stdout: &swapsOutput,
	})
	if err != nil {
		return runner.StorageSnapshot{}, fmt.Errorf("collect swap state: %w", err)
	}
	if result.ExitCode != 0 {
		return runner.StorageSnapshot{}, fmt.Errorf("read /proc/swaps exited with code %d", result.ExitCode)
	}

	return parseStorageSnapshot(lsblkOutput.Bytes(), swapsOutput.Bytes())
}

func parseStorageSnapshot(
	lsblkPayload []byte,
	swapsPayload []byte,
) (runner.StorageSnapshot, error) {
	var document lsblkDocument
	decoder := json.NewDecoder(bytes.NewReader(lsblkPayload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil {
		return runner.StorageSnapshot{}, fmt.Errorf("decode lsblk JSON: %w", err)
	}
	if len(document.BlockDevices) == 0 {
		return runner.StorageSnapshot{}, errors.New("lsblk returned no block devices")
	}

	swaps := parseSwapPaths(swapsPayload)
	snapshot := runner.StorageSnapshot{}
	for _, device := range document.BlockDevices {
		if err := appendBlockDevice(&snapshot, device, swaps); err != nil {
			return runner.StorageSnapshot{}, err
		}
	}
	return snapshot, nil
}

func appendBlockDevice(
	snapshot *runner.StorageSnapshot,
	device lsblkDevice,
	swaps map[string]bool,
) error {
	if strings.TrimSpace(device.Path) == "" {
		return errors.New("lsblk device has empty path")
	}
	var size uint64
	if device.Size != "" {
		parsed, err := strconv.ParseUint(string(device.Size), 10, 64)
		if err != nil {
			return fmt.Errorf("parse size for %s: %w", device.Path, err)
		}
		size = parsed
	}
	snapshot.Devices = append(snapshot.Devices, runner.BlockDevice{
		Path:           device.Path,
		DeviceType:     device.DeviceType,
		Filesystem:     device.Filesystem,
		PartitionTable: device.PTType,
		Mountpoint:     device.Mountpoint,
		SizeBytes:      size,
		SwapActive:     swaps[device.Path],
	})
	for _, child := range device.Children {
		if err := appendBlockDevice(snapshot, child, swaps); err != nil {
			return err
		}
	}
	return nil
}

func parseSwapPaths(payload []byte) map[string]bool {
	result := make(map[string]bool)
	lines := strings.Split(string(payload), "\n")
	for index, line := range lines {
		if index == 0 {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 || !strings.HasPrefix(fields[0], "/dev/") {
			continue
		}
		result[fields[0]] = true
	}
	return result
}
