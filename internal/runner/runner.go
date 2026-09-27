package runner

import (
	"context"
	"errors"
	"io"
	"time"
)

var ErrNotSupported = errors.New("operation not supported by runner")

type NetworkMode string

const (
	NetworkNone     NetworkMode = "none"
	NetworkIsolated NetworkMode = "isolated"
)

type FirmwareMode string

const (
	FirmwareBIOS FirmwareMode = "bios"
	FirmwareUEFI FirmwareMode = "uefi"
)

type VirtualDisk struct {
	ID     string
	SizeMB int
}

type MachineDefinition struct {
	Firmware   FirmwareMode
	ExtraDisks []VirtualDisk
}

func (machine MachineDefinition) Validate() error {
	if machine.Firmware != FirmwareBIOS && machine.Firmware != FirmwareUEFI {
		return errors.New("machine firmware must be bios or uefi")
	}
	if len(machine.ExtraDisks) > 4 {
		return errors.New("machine supports at most 4 extra disks")
	}
	seen := make(map[string]struct{}, len(machine.ExtraDisks))
	for _, disk := range machine.ExtraDisks {
		if disk.ID == "" {
			return errors.New("extra disk ID is required")
		}
		if _, exists := seen[disk.ID]; exists {
			return errors.New("extra disk IDs must be unique")
		}
		seen[disk.ID] = struct{}{}
		if disk.SizeMB < 64 || disk.SizeMB > 8192 {
			return errors.New("extra disk size must be between 64 and 8192 MiB")
		}
	}
	return nil
}

type Definition struct {
	LabID             string
	ImageRef          string
	Distribution      string
	Network           NetworkMode
	CapabilityProfile string
	MemoryMB          int
	CPUPercent        int
	PIDs              int
	Timeout           time.Duration
	Machine           *MachineDefinition
}

func (definition Definition) Validate() error {
	if definition.LabID == "" {
		return errors.New("lab ID is required")
	}
	if definition.ImageRef == "" {
		return errors.New("image ref is required")
	}
	if definition.Network != NetworkNone && definition.Network != NetworkIsolated {
		return errors.New("network must be none or isolated")
	}
	if definition.CapabilityProfile == "" {
		return errors.New("capability profile is required")
	}
	if definition.MemoryMB < 64 {
		return errors.New("memory limit must be at least 64 MiB")
	}
	if definition.CPUPercent < 10 || definition.CPUPercent > 400 {
		return errors.New("CPU limit must be between 10 and 400 percent")
	}
	if definition.PIDs < 16 {
		return errors.New("PID limit must be at least 16")
	}
	if definition.Timeout < 30*time.Second {
		return errors.New("timeout must be at least 30 seconds")
	}
	if definition.Machine != nil {
		if err := definition.Machine.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type Instance struct {
	ID string
}

type TerminalSize struct {
	Width  uint
	Height uint
}

func (size TerminalSize) Validate() error {
	if size.Width == 0 || size.Height == 0 {
		return errors.New("terminal width and height must both be positive")
	}
	return nil
}

type ExecRequest struct {
	Argv        []string
	Env         map[string]string
	WorkingDir  string
	Stdin       io.Reader
	Stdout      io.Writer
	Stderr      io.Writer
	TTY         bool
	InitialSize TerminalSize
	Resize      <-chan TerminalSize
}

func (request ExecRequest) Validate() error {
	if len(request.Argv) == 0 || request.Argv[0] == "" {
		return errors.New("argv[0] is required")
	}
	if request.WorkingDir != "" && request.WorkingDir[0] != '/' {
		return errors.New("working directory must be absolute")
	}

	hasInitialSize := request.InitialSize.Width != 0 || request.InitialSize.Height != 0
	if !request.TTY && (hasInitialSize || request.Resize != nil) {
		return errors.New("terminal size/resize requires TTY")
	}
	if hasInitialSize {
		if err := request.InitialSize.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type ExecResult struct {
	ExitCode int
}

type FileInfo struct {
	Path  string
	Mode  uint32
	UID   uint32
	GID   uint32
	User  string
	Group string
	IsDir bool
}

type Process struct {
	PID     int
	Command string
	Args    []string
}

type Runner interface {
	Prepare(context.Context, Definition) (Instance, error)
	Start(context.Context, Instance) error
	Exec(context.Context, Instance, ExecRequest) (ExecResult, error)
	Stat(context.Context, Instance, string) (FileInfo, error)
	ReadFile(context.Context, Instance, string, int64) ([]byte, error)
	Processes(context.Context, Instance) ([]Process, error)
	Reset(context.Context, Instance) error
	Destroy(context.Context, Instance) error
}
