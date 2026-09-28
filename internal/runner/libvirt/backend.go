package libvirt

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Loe159/lpic-daily/internal/runner"
	"golang.org/x/sys/unix"
)

var domainSlugUnsafe = regexp.MustCompile(`[^a-z0-9.-]+`)

var (
	_ runner.ConsoleRunner = (*Backend)(nil)
	_ runner.RebootRunner  = (*Backend)(nil)
	_ runner.StorageProbe  = (*Backend)(nil)
)

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
	Definition     runner.Definition
	Paths          OverlayPaths
	DomainDefined  bool
	NetworkName    string
	NetworkDefined bool
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

	var networkControl NetworkControlPlane
	networkName := ""
	if definition.Network == runner.NetworkIsolated {
		var ok bool
		networkControl, ok = backend.control.(NetworkControlPlane)
		if !ok {
			return fmt.Errorf(
				"%w: libvirt control plane has no isolated-network capability",
				runner.ErrNotSupported,
			)
		}
		networkName = name
	}

	paths, err := backend.overlays.Create(ctx, name, image, *definition.Machine)
	if err != nil {
		return fmt.Errorf("create VM disks for %s: %w", name, err)
	}
	cleanup := true
	networkDefined := false
	defer func() {
		if !cleanup {
			return
		}
		if networkDefined && networkControl != nil {
			if active, activeErr := networkControl.NetworkActive(networkName); activeErr == nil && active {
				_ = networkControl.DestroyNetwork(networkName)
			}
			_ = networkControl.UndefineNetwork(networkName)
		}
		_ = backend.overlays.Destroy(name)
		_ = releaseInstanceLease(paths.Lease)
	}()

	if networkControl != nil {
		networkXML, err := BuildIsolatedNetworkXML(networkName, networkSubnetOctet(networkName))
		if err != nil {
			return fmt.Errorf("build isolated network XML for %s: %w", networkName, err)
		}
		if err := networkControl.DefineNetwork(networkName, networkXML); err != nil {
			return err
		}
		networkDefined = true
		if err := networkControl.StartNetwork(networkName); err != nil {
			return err
		}
	}

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
		NetworkName:  networkName,
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
		Definition:     definition,
		Paths:          paths,
		DomainDefined:  true,
		NetworkName:    networkName,
		NetworkDefined: networkDefined,
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

func (backend *Backend) Reboot(ctx context.Context, instance runner.Instance) error {
	if err := ctx.Err(); err != nil {
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

	beforeBootID, err := backend.guestBootID(ctx, instance)
	if err != nil {
		return fmt.Errorf("read guest boot ID before reboot: %w", err)
	}
	if err := backend.control.RebootDomain(instance.ID); err != nil {
		return err
	}

	waitCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	var lastErr error
	for {
		bootID, probeErr := backend.guestBootID(waitCtx, instance)
		if probeErr == nil && bootID != beforeBootID {
			return nil
		}
		if probeErr != nil {
			lastErr = probeErr
		}

		select {
		case <-waitCtx.Done():
			if lastErr != nil {
				return fmt.Errorf("wait for guest reboot: %w (last probe error: %v)", waitCtx.Err(), lastErr)
			}
			return fmt.Errorf("wait for guest reboot: %w", waitCtx.Err())
		case <-ticker.C:
		}
	}
}

func (backend *Backend) guestBootID(ctx context.Context, instance runner.Instance) (string, error) {
	var output strings.Builder
	result, err := backend.Exec(ctx, instance, runner.ExecRequest{
		Argv:   []string{"/usr/bin/cat", "/proc/sys/kernel/random/boot_id"},
		Stdout: &output,
	})
	if err != nil {
		return "", err
	}
	if result.ExitCode != 0 {
		return "", fmt.Errorf("read guest boot ID: exit %d", result.ExitCode)
	}
	bootID := strings.TrimSpace(output.String())
	if bootID == "" || len(bootID) > 128 {
		return "", errors.New("guest returned invalid boot ID")
	}
	return bootID, nil
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

	if managed.DomainDefined {
		state, err := backend.control.DomainState(name)
		if err != nil {
			return err
		}
		if state.Active {
			if err := backend.control.DestroyDomain(name); err != nil {
				return err
			}
		}

		removeNVRAM := managed.Definition.Machine != nil &&
			managed.Definition.Machine.Firmware == runner.FirmwareUEFI
		if err := backend.control.UndefineDomain(name, removeNVRAM); err != nil {
			return err
		}

		managed.DomainDefined = false
		backend.mu.Lock()
		backend.instances[name] = managed
		backend.mu.Unlock()
	}

	if managed.NetworkDefined {
		networkControl, ok := backend.control.(NetworkControlPlane)
		if !ok {
			return fmt.Errorf(
				"%w: libvirt control plane lost isolated-network capability",
				runner.ErrNotSupported,
			)
		}
		active, err := networkControl.NetworkActive(managed.NetworkName)
		if err != nil {
			return err
		}
		if active {
			if err := networkControl.DestroyNetwork(managed.NetworkName); err != nil {
				return err
			}
		}
		if err := networkControl.UndefineNetwork(managed.NetworkName); err != nil {
			return err
		}
		managed.NetworkDefined = false
		backend.mu.Lock()
		backend.instances[name] = managed
		backend.mu.Unlock()
	}

	if err := backend.overlays.Destroy(name); err != nil {
		return err
	}
	if err := releaseInstanceLease(managed.Paths.Lease); err != nil {
		return fmt.Errorf("release VM instance lease: %w", err)
	}
	managed.Paths.Lease = nil

	backend.mu.Lock()
	delete(backend.instances, name)
	backend.mu.Unlock()
	return nil
}

func (backend *Backend) Reap(ctx context.Context) error {
	if backend == nil {
		return nil
	}
	inventory, ok := backend.control.(ResourceInventory)
	if !ok {
		return fmt.Errorf("%w: libvirt control plane has no resource inventory", runner.ErrNotSupported)
	}
	if err := os.MkdirAll(backend.stateRoot, 0o700); err != nil {
		return fmt.Errorf("create VM state root: %w", err)
	}

	domains, err := inventory.ListManagedDomains()
	if err != nil {
		return err
	}
	networks, err := inventory.ListManagedNetworks()
	if err != nil {
		return err
	}
	domainSet := make(map[string]struct{}, len(domains))
	networkSet := make(map[string]struct{}, len(networks))
	candidates := make(map[string]struct{}, len(domains)+len(networks))
	for _, name := range domains {
		domainSet[name] = struct{}{}
		candidates[name] = struct{}{}
	}
	for _, name := range networks {
		networkSet[name] = struct{}{}
		candidates[name] = struct{}{}
	}

	entries, err := os.ReadDir(backend.stateRoot)
	if err != nil {
		return fmt.Errorf("read VM state root: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() && managedNamePattern.MatchString(entry.Name()) {
			candidates[entry.Name()] = struct{}{}
		}
	}

	names := make([]string, 0, len(candidates))
	for name := range candidates {
		names = append(names, name)
	}
	sort.Strings(names)

	var errs []error
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			errs = append(errs, err)
			break
		}
		backend.mu.RLock()
		_, tracked := backend.instances[name]
		backend.mu.RUnlock()
		if tracked {
			continue
		}

		directory := filepath.Join(backend.stateRoot, name)
		var lease *os.File
		info, statErr := os.Lstat(directory)
		switch {
		case statErr == nil:
			if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
				errs = append(errs, fmt.Errorf("refusing orphan cleanup for unsafe state path %s", directory))
				continue
			}
			lease, err = acquireInstanceLease(directory)
			if err != nil {
				if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
					// Another LPIC Daily process owns this run; it is not abandoned.
					continue
				}
				errs = append(errs, err)
				continue
			}
		case errors.Is(statErr, os.ErrNotExist):
			// New code creates and leases state before defining libvirt resources.
			// A managed resource without state is therefore abandoned.
		default:
			errs = append(errs, fmt.Errorf("inspect orphan state %s: %w", directory, statErr))
			continue
		}

		if _, exists := domainSet[name]; exists {
			state, stateErr := backend.control.DomainState(name)
			if stateErr != nil {
				errs = append(errs, stateErr)
				_ = releaseInstanceLease(lease)
				continue
			}
			if state.Active {
				if destroyErr := backend.control.DestroyDomain(name); destroyErr != nil {
					errs = append(errs, destroyErr)
					_ = releaseInstanceLease(lease)
					continue
				}
			}
			if undefineErr := backend.control.UndefineDomain(name, true); undefineErr != nil {
				errs = append(errs, undefineErr)
				_ = releaseInstanceLease(lease)
				continue
			}
		}
		if _, exists := networkSet[name]; exists {
			networkControl, supported := backend.control.(NetworkControlPlane)
			if !supported {
				errs = append(errs, fmt.Errorf("%w: cannot reap network %s", runner.ErrNotSupported, name))
				_ = releaseInstanceLease(lease)
				continue
			}
			active, activeErr := networkControl.NetworkActive(name)
			if activeErr != nil {
				errs = append(errs, activeErr)
				_ = releaseInstanceLease(lease)
				continue
			}
			if active {
				if destroyErr := networkControl.DestroyNetwork(name); destroyErr != nil {
					errs = append(errs, destroyErr)
					_ = releaseInstanceLease(lease)
					continue
				}
			}
			if undefineErr := networkControl.UndefineNetwork(name); undefineErr != nil {
				errs = append(errs, undefineErr)
				_ = releaseInstanceLease(lease)
				continue
			}
		}
		if statErr == nil {
			if removeErr := backend.overlays.Destroy(name); removeErr != nil {
				errs = append(errs, removeErr)
				_ = releaseInstanceLease(lease)
				continue
			}
		}
		if releaseErr := releaseInstanceLease(lease); releaseErr != nil {
			errs = append(errs, releaseErr)
		}
	}
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

func networkSubnetOctet(name string) int {
	digest := sha256.Sum256([]byte(name))
	return int(digest[0])%250 + 1
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
