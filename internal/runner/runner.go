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

type Definition struct {
	LabID             string
	ImageRef          string
	Distribution      string
	Network           NetworkMode
	CapabilityProfile string
	MemoryMB          int
	PIDs              int
	Timeout           time.Duration
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
	if definition.PIDs < 16 {
		return errors.New("PID limit must be at least 16")
	}
	if definition.Timeout < 30*time.Second {
		return errors.New("timeout must be at least 30 seconds")
	}
	return nil
}

type Instance struct {
	ID string
}

type ExecRequest struct {
	Argv       []string
	Env        map[string]string
	WorkingDir string
	Stdin      io.Reader
	Stdout     io.Writer
	Stderr     io.Writer
	TTY        bool
}

func (request ExecRequest) Validate() error {
	if len(request.Argv) == 0 || request.Argv[0] == "" {
		return errors.New("argv[0] is required")
	}
	if request.WorkingDir != "" && request.WorkingDir[0] != '/' {
		return errors.New("working directory must be absolute")
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
