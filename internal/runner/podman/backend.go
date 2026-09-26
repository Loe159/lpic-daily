package podman

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"go.podman.io/podman/v6/pkg/bindings"
	"go.podman.io/podman/v6/pkg/bindings/containers"
	"go.podman.io/podman/v6/pkg/bindings/images"
	"go.podman.io/podman/v6/pkg/bindings/system"
	"go.podman.io/podman/v6/pkg/specgen"

	"github.com/Loe159/lpic-daily/internal/runner"
)

var safeNamePart = regexp.MustCompile(`[^a-z0-9_.-]+`)

type Backend struct {
	uri string

	mu          sync.RWMutex
	definitions map[string]runner.Definition
}

func DefaultURI() string {
	if runtimeDir := os.Getenv("XDG_RUNTIME_DIR"); runtimeDir != "" {
		return "unix://" + filepath.Join(runtimeDir, "podman", "podman.sock")
	}
	return fmt.Sprintf("unix:///run/user/%d/podman/podman.sock", os.Geteuid())
}

func Open(ctx context.Context, uri string) (*Backend, error) {
	if uri == "" {
		uri = DefaultURI()
	}

	conn, err := bindings.NewConnection(ctx, uri)
	if err != nil {
		return nil, fmt.Errorf("connect to Podman service %s: %w", uri, err)
	}

	info, err := system.Info(conn, nil)
	if err != nil {
		return nil, fmt.Errorf("inspect Podman service: %w", err)
	}
	if info == nil || info.Host == nil {
		return nil, errors.New("Podman service returned no host information")
	}
	if !info.Host.Security.Rootless {
		return nil, errors.New("refusing rootful Podman service: LPIC Daily requires rootless Podman")
	}
	if info.Host.CgroupsVersion != "v2" {
		return nil, fmt.Errorf("unsupported cgroups version %q: Phase 1 requires cgroups v2 for rootless limits", info.Host.CgroupsVersion)
	}

	return &Backend{
		uri:         uri,
		definitions: make(map[string]runner.Definition),
	}, nil
}

func (backend *Backend) Prepare(ctx context.Context, definition runner.Definition) (runner.Instance, error) {
	if err := definition.Validate(); err != nil {
		return runner.Instance{}, fmt.Errorf("validate lab definition: %w", err)
	}
	if definition.Network != runner.NetworkNone {
		return runner.Instance{}, fmt.Errorf("%w: Podman Phase 1 supports network=none only", runner.ErrNotSupported)
	}
	if _, err := runner.Phase1CapabilityProfile(definition.CapabilityProfile); err != nil {
		return runner.Instance{}, err
	}

	conn, err := backend.connection(ctx)
	if err != nil {
		return runner.Instance{}, err
	}
	exists, err := images.Exists(conn, definition.ImageRef, nil)
	if err != nil {
		return runner.Instance{}, fmt.Errorf("check local image %q: %w", definition.ImageRef, err)
	}
	if !exists {
		return runner.Instance{}, fmt.Errorf("required image %q is not present locally; implicit pulls are disabled", definition.ImageRef)
	}

	name, err := instanceName(definition.LabID)
	if err != nil {
		return runner.Instance{}, err
	}
	if err := backend.create(ctx, conn, definition, name); err != nil {
		return runner.Instance{}, err
	}

	backend.mu.Lock()
	backend.definitions[name] = definition
	backend.mu.Unlock()

	return runner.Instance{ID: name}, nil
}

func (backend *Backend) Start(ctx context.Context, instance runner.Instance) error {
	if instance.ID == "" {
		return errors.New("instance ID is required")
	}
	conn, err := backend.connection(ctx)
	if err != nil {
		return err
	}
	if err := containers.Start(conn, instance.ID, nil); err != nil {
		return fmt.Errorf("start container %s: %w", instance.ID, err)
	}
	return nil
}

func (backend *Backend) Exec(context.Context, runner.Instance, runner.ExecRequest) (runner.ExecResult, error) {
	return runner.ExecResult{}, runner.ErrNotSupported
}

func (backend *Backend) Stat(context.Context, runner.Instance, string) (runner.FileInfo, error) {
	return runner.FileInfo{}, runner.ErrNotSupported
}

func (backend *Backend) ReadFile(context.Context, runner.Instance, string, int64) ([]byte, error) {
	return nil, runner.ErrNotSupported
}

func (backend *Backend) Processes(context.Context, runner.Instance) ([]runner.Process, error) {
	return nil, runner.ErrNotSupported
}

func (backend *Backend) Reset(ctx context.Context, instance runner.Instance) error {
	backend.mu.RLock()
	definition, exists := backend.definitions[instance.ID]
	backend.mu.RUnlock()
	if !exists {
		return fmt.Errorf("unknown managed instance %q", instance.ID)
	}

	conn, err := backend.connection(ctx)
	if err != nil {
		return err
	}
	if err := removeContainer(conn, instance.ID); err != nil {
		return err
	}
	if err := backend.create(ctx, conn, definition, instance.ID); err != nil {
		return fmt.Errorf("recreate container %s: %w", instance.ID, err)
	}
	return nil
}

func (backend *Backend) Destroy(ctx context.Context, instance runner.Instance) error {
	if instance.ID == "" {
		return errors.New("instance ID is required")
	}
	conn, err := backend.connection(ctx)
	if err != nil {
		return err
	}
	if err := removeContainer(conn, instance.ID); err != nil {
		return err
	}

	backend.mu.Lock()
	delete(backend.definitions, instance.ID)
	backend.mu.Unlock()
	return nil
}

func (backend *Backend) connection(ctx context.Context) (context.Context, error) {
	conn, err := bindings.NewConnection(ctx, backend.uri)
	if err != nil {
		return nil, fmt.Errorf("connect to Podman service %s: %w", backend.uri, err)
	}
	return conn, nil
}

func (backend *Backend) create(ctx context.Context, conn context.Context, definition runner.Definition, name string) error {
	spec, err := buildSpec(definition, name)
	if err != nil {
		return err
	}
	response, err := containers.CreateWithSpec(conn, spec, nil)
	if err != nil {
		return fmt.Errorf("create container %s: %w", name, err)
	}
	if response.ID == "" {
		return fmt.Errorf("create container %s: Podman returned an empty container ID", name)
	}
	return nil
}

func buildSpec(definition runner.Definition, name string) (*specgen.SpecGenerator, error) {
	profile, err := runner.Phase1CapabilityProfile(definition.CapabilityProfile)
	if err != nil {
		return nil, err
	}
	if definition.Network != runner.NetworkNone {
		return nil, fmt.Errorf("%w: network mode %q", runner.ErrNotSupported, definition.Network)
	}

	falseValue := false
	trueValue := true
	memoryBytes := int64(definition.MemoryMB) * 1024 * 1024
	pids := int64(definition.PIDs)

	spec := specgen.NewSpecGenerator(definition.ImageRef, false)
	spec.Name = name
	spec.Command = []string{"/usr/bin/sleep", "infinity"}
	spec.EnvHost = &falseValue
	spec.HTTPProxy = &falseValue
	spec.Terminal = &falseValue
	spec.Stdin = &falseValue
	spec.Labels = map[string]string{
		"io.lpic-daily.managed": "true",
		"io.lpic-daily.lab-id":  definition.LabID,
	}
	spec.Timeout = uint(definition.Timeout.Seconds())

	spec.Privileged = &falseValue
	spec.CapDrop = []string{"ALL"}
	spec.CapAdd = profile.Capabilities
	spec.NoNewPrivileges = &trueValue

	spec.NetNS = specgen.Namespace{NSMode: specgen.NoNetwork}
	spec.PidNS = specgen.Namespace{NSMode: specgen.Private}
	spec.UtsNS = specgen.Namespace{NSMode: specgen.Private}
	spec.IpcNS = specgen.Namespace{NSMode: specgen.Private}

	spec.ImageVolumeMode = "ignore"
	spec.Mounts = nil
	spec.Volumes = nil
	spec.OverlayVolumes = nil
	spec.ImageVolumes = nil
	spec.Devices = nil
	spec.HostDeviceList = nil

	spec.ResourceLimits = &specs.LinuxResources{
		Memory: &specs.LinuxMemory{Limit: &memoryBytes},
		Pids:   &specs.LinuxPids{Limit: pids},
	}
	return spec, nil
}

func removeContainer(conn context.Context, name string) error {
	options := new(containers.RemoveOptions).
		WithForce(true).
		WithVolumes(true).
		WithIgnore(true)
	reports, err := containers.Remove(conn, name, options)
	if err != nil {
		return fmt.Errorf("remove container %s: %w", name, err)
	}
	for _, report := range reports {
		if report != nil && report.Err != nil {
			return fmt.Errorf("remove container %s: %w", name, report.Err)
		}
	}
	return nil
}

func instanceName(labID string) (string, error) {
	var random [6]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", fmt.Errorf("generate instance name: %w", err)
	}

	slug := safeNamePart.ReplaceAllString(labID, "-")
	slug = regexp.MustCompile(`^-+|-+$`).ReplaceAllString(slug, "")
	if slug == "" {
		slug = "lab"
	}
	if len(slug) > 40 {
		slug = slug[:40]
	}
	return "lpic-daily-" + slug + "-" + hex.EncodeToString(random[:]), nil
}
