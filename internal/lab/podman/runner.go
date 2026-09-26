package podman

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/Loe159/lpic-daily/internal/lab"
)

type Result struct {
	Stdout string
	Stderr string
}

type Executor interface {
	Run(ctx context.Context, executable string, args ...string) (Result, error)
}

type OSExecutor struct{}

func (OSExecutor) Run(ctx context.Context, executable string, args ...string) (Result, error) {
	cmd := exec.CommandContext(ctx, executable, args...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return Result{Stdout: stdout.String(), Stderr: stderr.String()}, err
}

type Runner struct {
	executable string
	executor   Executor
	now        func() time.Time
	newID      func() (string, error)
}

func New(executor Executor) *Runner {
	if executor == nil {
		executor = OSExecutor{}
	}
	return &Runner{
		executable: "podman",
		executor:   executor,
		now:        time.Now,
		newID:      randomID,
	}
}

func (r *Runner) Doctor(ctx context.Context) error {
	result, err := r.executor.Run(ctx, r.executable, "info", "--format", "{{.Host.Security.Rootless}}")
	if err != nil {
		return fmt.Errorf("podman rootless check failed: %w: %s", err, strings.TrimSpace(result.Stderr))
	}
	if strings.TrimSpace(result.Stdout) != "true" {
		return errors.New("podman is available but not running rootless; refusing lab execution")
	}
	return nil
}

func (r *Runner) Prepare(ctx context.Context, spec lab.Spec) (lab.Session, error) {
	if err := spec.Validate(); err != nil {
		return lab.Session{}, err
	}
	if spec.Network != lab.NetworkNone {
		return lab.Session{}, errors.New("Phase 1 Podman runner only permits network=none")
	}
	if err := r.Doctor(ctx); err != nil {
		return lab.Session{}, err
	}

	random, err := r.newID()
	if err != nil {
		return lab.Session{}, fmt.Errorf("generate lab session ID: %w", err)
	}
	name := "lpic-daily-" + random
	args, err := createArgs(name, spec)
	if err != nil {
		return lab.Session{}, err
	}
	result, err := r.executor.Run(ctx, r.executable, args...)
	if err != nil {
		return lab.Session{}, fmt.Errorf("create sandbox: %w: %s", err, strings.TrimSpace(result.Stderr))
	}
	containerID := strings.TrimSpace(result.Stdout)
	if containerID == "" {
		return lab.Session{}, errors.New("podman create returned an empty container ID")
	}
	created := r.now().UTC()
	return lab.Session{
		ID:          random,
		ContainerID: containerID,
		Spec:        spec,
		CreatedAt:   created,
		ExpiresAt:   created.Add(spec.Timeout()),
	}, nil
}

func (r *Runner) Start(ctx context.Context, session lab.Session) error {
	if err := validateSession(session); err != nil {
		return err
	}
	result, err := r.executor.Run(ctx, r.executable, "start", session.ContainerID)
	if err != nil {
		return fmt.Errorf("start sandbox: %w: %s", err, strings.TrimSpace(result.Stderr))
	}
	return nil
}

func (r *Runner) Exec(ctx context.Context, session lab.Session, argv []string) (Result, error) {
	if err := validateSession(session); err != nil {
		return Result{}, err
	}
	if len(argv) == 0 || strings.TrimSpace(argv[0]) == "" {
		return Result{}, errors.New("sandbox exec requires argv")
	}
	args := []string{"exec", "--", session.ContainerID}
	args = append(args, argv...)
	result, err := r.executor.Run(ctx, r.executable, args...)
	if err != nil {
		return result, fmt.Errorf("exec in sandbox: %w", err)
	}
	return result, nil
}

func (r *Runner) Destroy(ctx context.Context, session lab.Session) error {
	if err := validateSession(session); err != nil {
		return err
	}
	result, err := r.executor.Run(ctx, r.executable, "rm", "--force", "--time", "1", session.ContainerID)
	if err != nil {
		return fmt.Errorf("destroy sandbox: %w: %s", err, strings.TrimSpace(result.Stderr))
	}
	return nil
}

func (r *Runner) Reset(ctx context.Context, session lab.Session) (lab.Session, error) {
	if err := r.Destroy(ctx, session); err != nil {
		return lab.Session{}, err
	}
	return r.Prepare(ctx, session.Spec)
}

func createArgs(name string, spec lab.Spec) ([]string, error) {
	caps, err := capabilities(spec.CapabilityProfile)
	if err != nil {
		return nil, err
	}
	args := []string{
		"create",
		"--pull=never",
		"--name", name,
		"--network=none",
		"--ipc=private",
		"--pid=private",
		"--cap-drop=ALL",
		"--security-opt=no-new-privileges",
		"--read-only",
		"--pids-limit", strconv.Itoa(spec.Resources.PIDs),
		"--memory", fmt.Sprintf("%dm", spec.Resources.MemoryMB),
		"--cpus", "1",
		"--tmpfs", "/tmp:rw,nosuid,nodev,size=64m",
		"--tmpfs", "/workspace:rw,nosuid,nodev,size=128m",
		"--workdir", "/workspace",
	}
	for _, capability := range caps {
		args = append(args, "--cap-add="+capability)
	}
	args = append(args, spec.ImageRef, "sleep", "infinity")
	return args, nil
}

func capabilities(profile lab.CapabilityProfile) ([]string, error) {
	switch profile {
	case lab.ProfileNone:
		return nil, nil
	case lab.ProfilePermissions:
		return []string{"CHOWN", "FOWNER", "DAC_OVERRIDE", "SETGID", "SETUID"}, nil
	default:
		return nil, fmt.Errorf("unknown capability profile %q", profile)
	}
}

func validateSession(session lab.Session) error {
	if strings.TrimSpace(session.ID) == "" || strings.TrimSpace(session.ContainerID) == "" {
		return errors.New("invalid lab session")
	}
	return nil
}

func randomID() (string, error) {
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}
