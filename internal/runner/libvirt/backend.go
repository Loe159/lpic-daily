package libvirt

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/Loe159/lpic-daily/internal/runner"
)

var domainSlugUnsafe = regexp.MustCompile(`[^a-z0-9.-]+`)

var _ runner.ConsoleRunner = (*Backend)(nil)

type Backend struct {
	control   ControlPlane
	catalog   *ImageCatalog
	imageRoot string
	stateRoot string
	overlays  OverlayManager

	mu        sync.RWMutex
	instances map[string]managedInstance
}

type managedInstance struct {
	Definition runner.Definition
	Paths      OverlayPaths
}

func NewBackend(
	control ControlPlane,
	catalog *ImageCatalog,
	imageRoot string,
	stateRoot string,
	commands CommandRunner,
) (*Backend, error) {
	if control == nil {
		return nil, errors.New("libvirt control plane is required")
	}
	if catalog == nil {
		return nil, errors.New("VM image catalog is required")
	}
	overlays := OverlayManager{
		ImageRoot: imageRoot,
		StateRoot: stateRoot,
		Commands:  commands,
	}
	if err := overlays.Validate(); err != nil {
		return nil, err
	}
	return &Backend{
		control:   control,
		catalog:   catalog,
		imageRoot: imageRoot,
		stateRoot: stateRoot,
		overlays:  overlays,
		instances: make(map[string]managedInstance),
	}, nil
}

func (backend *Backend) Prepare(
	ctx context.Context,
	definition runner.Definition,
) (runner.Instance, error) {
	if err := definition.Validate(); err != nil {
		return runner.Instance{}, fmt.Errorf("validate VM lab definition: %w", err)
	}
	if definition.Machine == nil {
		return runner.Instance{}, errors.New("libvirt runner requires machine settings")
	}
	if definition.Network != runner.NetworkNone {
		return runner.Instance{}, fmt.Errorf(
			"%w: libvirt isolated networking is not implemented yet",
			runner.ErrNotSupported,
		)
	}
	name, err := vmInstanceName(definition.LabID)
	if err != nil {
		return runner.Instance{}, err
	}
	if err := backend.prepareNamed(ctx, name, definition); err != nil {
		return runner.Instance{}, err
	}
	return runner.Instance{ID: name}, nil
}

func (backend *Backend) prepareNamed(
	ctx context.Context,
	name string,
	definition runner.Definition,
) error {
	image, err := backend.catalog.Resolve(definition.ImageRef, backend.imageRoot)
	if err != nil {
		return err
	}
	if image.Distribution != definition.Distribution {
		return fmt.Errorf(
			"trusted image %s distribution=%s, lab requires %s",
			image.ID,
			image.Distribution,
			definition.Distribution,
		)
	}

	paths, err := backend.overlays.Create(ctx, name, image, *definition.Machine)
	if err != nil {
		return fmt.Errorf("create VM disks for %s: %w", name, err)
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = backend.overlays.Destroy(name)
		}
	}()

	extraDisks := make([]DiskPath, 0, len(definition.Machine.ExtraDisks))
	for _, requested := range definition.Machine.ExtraDisks {
		path, exists := paths.Extra[requested.ID]
		if !exists {
			return fmt.Errorf("overlay manager omitted scratch disk %s", requested.ID)
		}
		extraDisks = append(extraDisks, DiskPath{ID: requested.ID, Path: path})
	}

	xml, err := BuildDomainXML(DomainSpec{
		Name:         name,
		LabID:        definition.LabID,
		MemoryMB:     definition.MemoryMB,
		CPUPercent:   definition.CPUPercent,
		Firmware:     definition.Machine.Firmware,
		RootDiskPath: paths.RootDisk,
		ExtraDisks:   extraDisks,
	}, backend.stateRoot)
	if err != nil {
		return fmt.Errorf("build domain XML for %s: %w", name, err)
	}
	if err := backend.control.DefineDomain(name, xml); err != nil {
		return err
	}

	backend.mu.Lock()
	if _, exists := backend.instances[name]; exists {
		backend.mu.Unlock()
		_ = backend.control.UndefineDomain(name, definition.Machine.Firmware == runner.FirmwareUEFI)
		return fmt.Errorf("VM instance %s already tracked", name)
	}
	backend.instances[name] = managedInstance{
		Definition: definition,
		Paths:      paths,
	}
	backend.mu.Unlock()

	cleanup = false
	return nil
}

func (backend *Backend) Start(ctx context.Context, instance runner.Instance) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := backend.instance(instance); err != nil {
		return err
	}
	if err := backend.control.StartDomain(instance.ID); err != nil {
		return err
	}
	return nil
}

func (backend *Backend) OpenConsole(
	ctx context.Context,
	instance runner.Instance,
	request runner.ConsoleRequest,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := request.Validate(); err != nil {
		return err
	}
	if _, err := backend.instance(instance); err != nil {
		return err
	}
	state, err := backend.control.DomainState(instance.ID)
	if err != nil {
		return err
	}
	if !state.Active {
		return fmt.Errorf("VM instance %s is not active", instance.ID)
	}
	console, ok := backend.control.(ConsoleControlPlane)
	if !ok {
		return fmt.Errorf(
			"%w: libvirt control plane has no serial console capability",
			runner.ErrNotSupported,
		)
	}
	if err := console.OpenConsole(instance.ID, request.Stdin, request.Stdout); err != nil {
		return err
	}
	return nil
}


func (backend *Backend) Stat(
	context.Context,
	runner.Instance,
	string,
) (runner.FileInfo, error) {
	return runner.FileInfo{}, fmt.Errorf(
		"%w: guest filesystem probes are not implemented yet",
		runner.ErrNotSupported,
	)
}

func (backend *Backend) ReadFile(
	context.Context,
	runner.Instance,
	string,
	int64,
) ([]byte, error) {
	return nil, fmt.Errorf(
		"%w: guest filesystem probes are not implemented yet",
		runner.ErrNotSupported,
	)
}

func (backend *Backend) Processes(
	context.Context,
	runner.Instance,
) ([]runner.Process, error) {
	return nil, fmt.Errorf(
		"%w: guest process probes are not implemented yet",
		runner.ErrNotSupported,
	)
}

func (backend *Backend) Reset(ctx context.Context, instance runner.Instance) error {
	managed, err := backend.instance(instance)
	if err != nil {
		return err
	}
	if err := backend.destroyManaged(ctx, instance.ID, managed); err != nil {
		return err
	}
	return backend.prepareNamed(ctx, instance.ID, managed.Definition)
}

func (backend *Backend) Destroy(ctx context.Context, instance runner.Instance) error {
	managed, err := backend.instance(instance)
	if errors.Is(err, errUnknownVMInstance) {
		return nil
	}
	if err != nil {
		return err
	}
	return backend.destroyManaged(ctx, instance.ID, managed)
}

func (backend *Backend) destroyManaged(
	ctx context.Context,
	name string,
	managed managedInstance,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	var errs []error
	state, stateErr := backend.control.DomainState(name)
	if stateErr != nil {
		errs = append(errs, stateErr)
	} else if state.Active {
		if err := backend.control.DestroyDomain(name); err != nil {
			errs = append(errs, err)
		}
	}

	removeNVRAM := managed.Definition.Machine != nil &&
		managed.Definition.Machine.Firmware == runner.FirmwareUEFI
	if err := backend.control.UndefineDomain(name, removeNVRAM); err != nil {
		errs = append(errs, err)
	}
	if err := backend.overlays.Destroy(name); err != nil {
		errs = append(errs, err)
	}

	backend.mu.Lock()
	delete(backend.instances, name)
	backend.mu.Unlock()

	return errors.Join(errs...)
}

func (backend *Backend) Close(ctx context.Context) error {
	if backend == nil {
		return nil
	}
	backend.mu.RLock()
	names := make([]string, 0, len(backend.instances))
	for name := range backend.instances {
		names = append(names, name)
	}
	backend.mu.RUnlock()

	var errs []error
	for _, name := range names {
		managed, err := backend.instance(runner.Instance{ID: name})
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if err := backend.destroyManaged(ctx, name, managed); err != nil {
			errs = append(errs, err)
		}
	}
	if err := backend.control.Close(); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

var errUnknownVMInstance = errors.New("unknown managed VM instance")

func (backend *Backend) instance(instance runner.Instance) (managedInstance, error) {
	if instance.ID == "" {
		return managedInstance{}, errors.New("instance ID is required")
	}
	if err := validateManagedResourceName("domain", instance.ID); err != nil {
		return managedInstance{}, err
	}
	backend.mu.RLock()
	managed, exists := backend.instances[instance.ID]
	backend.mu.RUnlock()
	if !exists {
		return managedInstance{}, fmt.Errorf("%w %q", errUnknownVMInstance, instance.ID)
	}
	return managed, nil
}

func vmInstanceName(labID string) (string, error) {
	var suffix [6]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return "", fmt.Errorf("generate VM instance name: %w", err)
	}

	slug := strings.ToLower(labID)
	slug = domainSlugUnsafe.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-.")
	if slug == "" {
		slug = "lab"
	}
	if len(slug) > 38 {
		slug = slug[:38]
		slug = strings.TrimRight(slug, "-.")
	}
	name := "lpic-daily-" + slug + "-" + hex.EncodeToString(suffix[:])
	if !managedNamePattern.MatchString(name) {
		return "", fmt.Errorf("generated invalid VM name %q", name)
	}
	return name, nil
}
