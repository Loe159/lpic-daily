package libvirt

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Loe159/lpic-daily/internal/runner"
	"golang.org/x/sys/unix"
)

type CommandRunner interface {
	Run(context.Context, string, ...string) error
}

type ExecCommandRunner struct {
	QEMUImgPath string
}

func NewExecCommandRunner() (*ExecCommandRunner, error) {
	path, err := exec.LookPath("qemu-img")
	if err != nil {
		return nil, fmt.Errorf("find qemu-img: %w", err)
	}
	if !filepath.IsAbs(path) {
		path, err = filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("resolve qemu-img path: %w", err)
		}
	}
	return &ExecCommandRunner{QEMUImgPath: filepath.Clean(path)}, nil
}

func (runner *ExecCommandRunner) Run(ctx context.Context, name string, args ...string) error {
	if runner == nil || runner.QEMUImgPath == "" {
		return errors.New("qemu-img runner is not configured")
	}
	if name != "qemu-img" {
		return fmt.Errorf("unsupported trusted helper %q", name)
	}

	command := exec.CommandContext(ctx, runner.QEMUImgPath, args...)
	var stderr strings.Builder
	command.Stdout = io.Discard
	command.Stderr = &boundedWriter{destination: &stderr, remaining: 64 << 10}
	if err := command.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			return fmt.Errorf("qemu-img %v: %w", args, err)
		}
		return fmt.Errorf("qemu-img %v: %w: %s", args, err, message)
	}
	return nil
}

type boundedWriter struct {
	destination io.Writer
	remaining   int64
}

func (writer *boundedWriter) Write(payload []byte) (int, error) {
	original := len(payload)
	if writer.remaining <= 0 {
		return original, nil
	}
	toWrite := payload
	if int64(len(toWrite)) > writer.remaining {
		toWrite = toWrite[:writer.remaining]
	}
	n, err := writer.destination.Write(toWrite)
	writer.remaining -= int64(n)
	if err != nil {
		return n, err
	}
	return original, nil
}

type OverlayPaths struct {
	Directory string
	RootDisk  string
	Extra     map[string]string
	Lease     *os.File
}

type OverlayManager struct {
	ImageRoot string
	StateRoot string
	Commands  CommandRunner
}

func (manager OverlayManager) Validate() error {
	if manager.Commands == nil {
		return errors.New("trusted command runner is required")
	}
	if manager.ImageRoot == "" || !filepath.IsAbs(manager.ImageRoot) {
		return errors.New("image root must be absolute")
	}
	if manager.StateRoot == "" || !filepath.IsAbs(manager.StateRoot) {
		return errors.New("VM state root must be absolute")
	}
	if filepath.Clean(manager.ImageRoot) == filepath.Clean(manager.StateRoot) {
		return errors.New("image root and VM state root must be distinct")
	}
	return nil
}

func (manager OverlayManager) Create(
	ctx context.Context,
	instanceName string,
	image ImageDescriptor,
	machine runner.MachineDefinition,
) (OverlayPaths, error) {
	if err := manager.Validate(); err != nil {
		return OverlayPaths{}, err
	}
	if !managedNamePattern.MatchString(instanceName) {
		return OverlayPaths{}, fmt.Errorf("invalid managed instance name %q", instanceName)
	}
	if err := image.Validate(manager.ImageRoot); err != nil {
		return OverlayPaths{}, fmt.Errorf("validate image descriptor: %w", err)
	}
	if err := machine.Validate(); err != nil {
		return OverlayPaths{}, fmt.Errorf("validate machine: %w", err)
	}
	if !image.SupportsFirmware(machine.Firmware) {
		return OverlayPaths{}, fmt.Errorf("image %s does not support firmware %s", image.ID, machine.Firmware)
	}
	if err := VerifyImageFile(ctx, image); err != nil {
		return OverlayPaths{}, err
	}

	directory := filepath.Join(filepath.Clean(manager.StateRoot), instanceName)
	if err := pathWithinRoot(manager.StateRoot, directory); err != nil {
		return OverlayPaths{}, fmt.Errorf("instance directory: %w", err)
	}
	if _, err := os.Lstat(directory); err == nil {
		return OverlayPaths{}, fmt.Errorf("instance directory already exists: %s", directory)
	} else if !errors.Is(err, os.ErrNotExist) {
		return OverlayPaths{}, fmt.Errorf("inspect instance directory: %w", err)
	}
	if err := os.MkdirAll(directory, 0o711); err != nil {
		return OverlayPaths{}, fmt.Errorf("create instance directory: %w", err)
	}
	if err := os.Chmod(directory, 0o711); err != nil {
		_ = os.RemoveAll(directory)
		return OverlayPaths{}, fmt.Errorf("set instance directory permissions: %w", err)
	}

	lease, err := acquireInstanceLease(directory)
	if err != nil {
		_ = os.RemoveAll(directory)
		return OverlayPaths{}, err
	}

	cleanup := true
	defer func() {
		if cleanup {
			_ = releaseInstanceLease(lease)
			_ = os.RemoveAll(directory)
		}
	}()

	paths := OverlayPaths{
		Directory: directory,
		RootDisk:  filepath.Join(directory, "root.qcow2"),
		Extra:     make(map[string]string, len(machine.ExtraDisks)),
		Lease:     lease,
	}
	if err := manager.Commands.Run(
		ctx,
		"qemu-img",
		"create",
		"-f", "qcow2",
		"-F", "qcow2",
		"-b", filepath.Clean(image.Path),
		paths.RootDisk,
	); err != nil {
		return OverlayPaths{}, fmt.Errorf("create root overlay: %w", err)
	}

	for _, disk := range machine.ExtraDisks {
		path := filepath.Join(directory, "disk-"+disk.ID+".qcow2")
		if err := manager.Commands.Run(
			ctx,
			"qemu-img",
			"create",
			"-f", "qcow2",
			path,
			fmt.Sprintf("%dM", disk.SizeMB),
		); err != nil {
			return OverlayPaths{}, fmt.Errorf("create scratch disk %s: %w", disk.ID, err)
		}
		paths.Extra[disk.ID] = path
	}

	cleanup = false
	return paths, nil
}

func (manager OverlayManager) Destroy(instanceName string) error {
	if err := manager.Validate(); err != nil {
		return err
	}
	if !managedNamePattern.MatchString(instanceName) {
		return fmt.Errorf("invalid managed instance name %q", instanceName)
	}
	directory := filepath.Join(filepath.Clean(manager.StateRoot), instanceName)
	if err := pathWithinRoot(manager.StateRoot, directory); err != nil {
		return err
	}
	info, err := os.Lstat(directory)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect instance directory: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return errors.New("refusing to remove symlinked VM instance directory")
	}
	if err := os.RemoveAll(directory); err != nil {
		return fmt.Errorf("remove VM instance directory: %w", err)
	}
	return nil
}

func acquireInstanceLease(directory string) (*os.File, error) {
	path := filepath.Join(directory, ".lease")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open VM instance lease: %w", err)
	}
	if err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("lock VM instance lease: %w", err)
	}
	return file, nil
}

func releaseInstanceLease(file *os.File) error {
	if file == nil {
		return nil
	}
	unlockErr := unix.Flock(int(file.Fd()), unix.LOCK_UN)
	closeErr := file.Close()
	return errors.Join(unlockErr, closeErr)
}

func VerifyImageFile(ctx context.Context, image ImageDescriptor) error {
	file, err := os.Open(filepath.Clean(image.Path))
	if err != nil {
		return fmt.Errorf("open base image %s: %w", image.ID, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat base image %s: %w", image.ID, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("base image %s is not a regular file", image.ID)
	}

	hasher := sha256.New()
	reader := bufio.NewReaderSize(file, 1<<20)
	buffer := make([]byte, 1<<20)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		count, readErr := reader.Read(buffer)
		if count > 0 {
			if _, err := hasher.Write(buffer[:count]); err != nil {
				return fmt.Errorf("hash base image %s: %w", image.ID, err)
			}
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return fmt.Errorf("read base image %s: %w", image.ID, readErr)
		}
	}
	got := hex.EncodeToString(hasher.Sum(nil))
	if got != image.SHA256 {
		return fmt.Errorf("base image %s checksum mismatch: got %s", image.ID, got)
	}
	return nil
}
