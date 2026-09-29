package runner

import (
	"context"
	"errors"
	"io"
	"path"
	"regexp"
	"strings"
	"time"
)

var (
	ErrNotSupported      = errors.New("operation not supported by runner")
	virtualDiskIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,31}package runner

import (
	"context"
	"errors"
	"io"
	"path"
	"regexp"
	"strings"
	"time"
)

)
)

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
		if !virtualDiskIDPattern.MatchString(disk.ID) {
			return errors.New("extra disk ID must match ^[a-z0-9][a-z0-9-]{0,31}$")
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
	LabID              string
	ImageRef           string
	Distribution       string
	Network            NetworkMode
	CapabilityProfile  string
	WritableGuestPaths []string
	MemoryMB           int
	CPUPercent         int
	PIDs               int
	Timeout            time.Duration
	Machine            *MachineDefinition
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
	seenWritablePaths := make(map[string]struct{}, len(definition.WritableGuestPaths))
	for _, guestPath := range definition.WritableGuestPaths {
		if !strings.HasPrefix(guestPath, "/") || path.Clean(guestPath) != guestPath || guestPath == "/" {
			return errors.New("writable guest paths must be clean absolute paths below /")
		}
		for _, blocked := range []string{"/dev", "/proc", "/sys"} {
			if guestPath == blocked || strings.HasPrefix(guestPath, blocked+"/") {
				return errors.New("writable guest paths must not target /dev, /proc, or /sys")
			}
		}
		if _, exists := seenWritablePaths[guestPath]; exists {
			return errors.New("writable guest paths must be unique")
		}
		seenWritablePaths[guestPath] = struct{}{}
	}
	if definition.MemoryMB < 64 || definition.MemoryMB > 16384 {
		return errors.New("memory limit must be between 64 and 16384 MiB")
	}
	if definition.CPUPercent < 10 || definition.CPUPercent > 400 {
		return errors.New("CPU limit must be between 10 and 400 percent")
	}
	if definition.PIDs < 16 || definition.PIDs > 4096 {
		return errors.New("PID limit must be between 16 and 4096")
	}
	if definition.Timeout < 30*time.Second || definition.Timeout > 2*time.Hour {
		return errors.New("timeout must be between 30 seconds and 2 hours")
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

type BlockDevice struct {
	Path           string
	DeviceType     string
	Filesystem     string
	PartitionTable string
	Mountpoint     string
	SizeBytes      uint64
	SwapActive     bool
}

type StorageSnapshot struct {
	Devices []BlockDevice
}

// StorageProbe is an optional structured state capability. It deliberately
// exposes observations rather than guest commands so checkers remain
// independent from the transport used by a full-machine backend.
type StorageProbe interface {
	StorageSnapshot(context.Context, Instance) (StorageSnapshot, error)
}

type ConsoleRequest struct {
	Stdin  io.Reader
	Stdout io.Writer
}

func (request ConsoleRequest) Validate() error {
	if request.Stdin == nil {
		return errors.New("console stdin is required")
	}
	if request.Stdout == nil {
		return errors.New("console stdout is required")
	}
	return nil
}

// ConsoleRunner is an optional capability for full-machine backends.
// It is intentionally separate from Runner because container labs do not
// require a firmware/boot serial console.
type ConsoleRunner interface {
	OpenConsole(context.Context, Instance, ConsoleRequest) error
}

// RebootRunner is an optional capability for full-machine backends.
// It requests a guest-visible reboot without recreating the disposable disks.
type RebootRunner interface {
	Reboot(context.Context, Instance) error
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
