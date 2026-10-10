package libvirt

import (
	"context"
	"crypto/rand"
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
	_ runner.RebootRunner = (*Backend)(nil)
	_ runner.ACPIButtonRunner = (*Backend)(nil)
	_ runner.BootMenuRunner = (*Backend)(nil)
	_ runner.StorageProbe  = (*Backend)(nil)
)

type Backend struct {
	control                   ControlPlane
	catalog                   *ImageCatalog
	imageRoot                 string
	stateRoot                 string
	ownerScope                string
	networkAllocationLockPath string
	overlays                  OverlayManager

	mu        sync.RWMutex
	instances map[string]managedInstance
	scenarios map[string]Scenario
}

type managedInstance struct {
	Definition     runner.Definition
	Paths          OverlayPaths
	DomainDefined  bool
	NetworkName    string
	NetworkDefined bool
	FilterName     string
	FilterDefined  bool
}

func NewBackend(
	control ControlPlane,
	catalog *ImageCatalog,
	imageRoot string,
	stateRoot string,
	networkAllocationLockPath string,
	commands CommandRunner,
) (*Backend, error) {
	return newBackendForEffectiveUID(
		os.Geteuid(),
		control,
		catalog,
		imageRoot,
		stateRoot,
		networkAllocationLockPath,
		commands,
	)
}

func newBackendForEffectiveUID(
	effectiveUID int,
	control ControlPlane,
	catalog *ImageCatalog,
	imageRoot string,
	stateRoot string,
	networkAllocationLockPath string,
	commands CommandRunner,
) (*Backend, error) {
	if effectiveUID == 0 {
		return nil, errors.New("libvirt VM backend must run as a regular user, not root")
	}
	if effectiveUID < 0 {
		return nil, errors.New("effective UID must not be negative")
	}
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
	if err := overlays.validateProvisionedStateRoot(); err != nil {
		return nil, err
	}
	if networkAllocationLockPath == "" || !filepath.IsAbs(networkAllocationLockPath) {
		return nil, errors.New("network allocation lock path must be absolute")
	}
	ownerScope, err := managedOwnerScopeForUID(stateRoot, effectiveUID)
	if err != nil {
		return nil, err
	}
	if err := control.SetManagedOwnerScope(ownerScope); err != nil {
		return nil, fmt.Errorf("configure libvirt ownership scope: %w", err)
	}
	return &Backend{
		control:                   control,
		catalog:                   catalog,
		imageRoot:                 imageRoot,
		stateRoot:                 stateRoot,
		ownerScope:                ownerScope,
		networkAllocationLockPath: filepath.Clean(networkAllocationLockPath),
		overlays:                  overlays,
		instances:                 make(map[string]managedInstance),
		scenarios:                 make(map[string]Scenario),
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
	return backend.prepareNamedOnNetwork(ctx, name, definition, "", true)
}

func (backend *Backend) prepareNamedOnNetwork(
	ctx context.Context,
	name string,
	definition runner.Definition,
	sharedNetworkName string,
	manageNetwork bool,
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
	var filterControl NetworkFilterControlPlane
	networkName := ""
	filterName := ""
	if definition.Network == runner.NetworkIsolated {
		var ok bool
		networkControl, ok = backend.control.(NetworkControlPlane)
		if !ok {
			return fmt.Errorf(
				"%w: libvirt control plane has no isolated-network capability",
				runner.ErrNotSupported,
			)
		}
		filterControl, ok = backend.control.(NetworkFilterControlPlane)
		if !ok {
			return fmt.Errorf(
				"%w: libvirt control plane has no network-filter capability",
				runner.ErrNotSupported,
			)
		}
		networkName = name
		if sharedNetworkName != "" {
			if err := validateManagedResourceName("network", sharedNetworkName); err != nil {
				return err
			}
			networkName = sharedNetworkName
		}
		filterName = networkName
	}

	paths, err := backend.overlays.Create(ctx, name, image, *definition.Machine)
	if err != nil {
		return fmt.Errorf("create VM disks for %s: %w", name, err)
	}
	cleanup := true
	networkDefined := false
	filterDefined := false
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
		if filterDefined && filterControl != nil {
			_ = filterControl.UndefineNetworkFilter(filterName)
		}
		_ = backend.overlays.Destroy(name)
		_ = releaseInstanceLease(paths.Lease)
	}()

	if networkControl != nil && manageNetwork {
		allocationLock, err := acquireNetworkAllocationLock(ctx, backend.networkAllocationLockPath)
		if err != nil {
			return fmt.Errorf("lock isolated network allocation for %s: %w", networkName, err)
		}
		subnet, err := backend.allocateIsolatedSubnet(networkName)
		if err != nil {
			_ = releaseNetworkAllocationLock(allocationLock)
			return fmt.Errorf("allocate isolated network for %s: %w", networkName, err)
		}
		networkXML, err := BuildIsolatedNetworkXML(networkName, subnet, backend.ownerScope)
		if err != nil {
			_ = releaseNetworkAllocationLock(allocationLock)
			return fmt.Errorf("build isolated network XML for %s: %w", networkName, err)
		}
		filterXML, err := BuildHostIsolationFilterXML(filterName, subnet, backend.ownerScope)
		if err != nil {
			_ = releaseNetworkAllocationLock(allocationLock)
			return fmt.Errorf("build host-isolation filter XML for %s: %w", filterName, err)
		}
		if err := networkControl.DefineNetwork(networkName, networkXML); err != nil {
			_ = releaseNetworkAllocationLock(allocationLock)
			return err
		}
		networkDefined = true
		if err := filterControl.DefineNetworkFilter(filterName, filterXML); err != nil {
			_ = networkControl.UndefineNetwork(networkName)
			_ = releaseNetworkAllocationLock(allocationLock)
			return err
		}
		filterDefined = true
		if err := releaseNetworkAllocationLock(allocationLock); err != nil {
			_ = filterControl.UndefineNetworkFilter(filterName)
			_ = networkControl.UndefineNetwork(networkName)
			return fmt.Errorf("unlock isolated network allocation for %s: %w", networkName, err)
		}
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
		extraDisks = append(extraDisks, DiskPath{ID: requested.ID, Path: path, Bus: requested.Bus})
	}

	xml, err := BuildDomainXML(DomainSpec{
		Name:              name,
		LabID:             definition.LabID,
		OwnerScope:        backend.ownerScope,
		MemoryMB:          definition.MemoryMB,
		CPUPercent:        definition.CPUPercent,
		Firmware:          definition.Machine.Firmware,
		RootDiskPath:      paths.RootDisk,
		ExtraDisks:        extraDisks,
		NetworkName:       networkName,
		NetworkFilterName: filterName,
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
		FilterName:     filterName,
		FilterDefined:  filterDefined,
	}
	backend.mu.Unlock()

	cleanup = false
	return nil
}

type Scenario struct {
	NetworkName    string
	Instances      []runner.Instance
	lease          *os.File
	networkDefined bool
	filterDefined  bool
}

func (backend *Backend) PrepareScenario(
	ctx context.Context,
	definitions []runner.Definition,
) (Scenario, error) {
	if len(definitions) < 2 || len(definitions) > 8 {
		return Scenario{}, errors.New("isolated VM scenario requires 2..8 machines")
	}
	networkControl, ok := backend.control.(NetworkControlPlane)
	if !ok {
		return Scenario{}, fmt.Errorf(
			"%w: libvirt control plane has no isolated-network capability",
			runner.ErrNotSupported,
		)
	}
	filterControl, ok := backend.control.(NetworkFilterControlPlane)
	if !ok {
		return Scenario{}, fmt.Errorf(
			"%w: libvirt control plane has no network-filter capability",
			runner.ErrNotSupported,
		)
	}
	networkName, err := vmInstanceName("scenario-network")
	if err != nil {
		return Scenario{}, err
	}
	networkDirectory := filepath.Join(backend.stateRoot, networkName)
	if err := os.MkdirAll(networkDirectory, 0o700); err != nil {
		return Scenario{}, fmt.Errorf("create scenario network state: %w", err)
	}
	networkLease, err := acquireInstanceLease(networkDirectory)
	if err != nil {
		_ = os.RemoveAll(networkDirectory)
		return Scenario{}, fmt.Errorf("lease scenario network: %w", err)
	}
	cleanupLease := true
	defer func() {
		if cleanupLease {
			_ = releaseInstanceLease(networkLease)
			_ = os.RemoveAll(networkDirectory)
		}
	}()

	allocationLock, err := acquireNetworkAllocationLock(ctx, backend.networkAllocationLockPath)
	if err != nil {
		return Scenario{}, fmt.Errorf("lock scenario network allocation: %w", err)
	}
	subnet, err := backend.allocateIsolatedSubnet(networkName)
	if err != nil {
		_ = releaseNetworkAllocationLock(allocationLock)
		return Scenario{}, fmt.Errorf("allocate scenario network: %w", err)
	}
	networkXML, err := BuildIsolatedNetworkXML(networkName, subnet, backend.ownerScope)
	if err != nil {
		_ = releaseNetworkAllocationLock(allocationLock)
		return Scenario{}, err
	}
	filterXML, err := BuildHostIsolationFilterXML(networkName, subnet, backend.ownerScope)
	if err != nil {
		_ = releaseNetworkAllocationLock(allocationLock)
		return Scenario{}, err
	}
	if err := networkControl.DefineNetwork(networkName, networkXML); err != nil {
		_ = releaseNetworkAllocationLock(allocationLock)
		return Scenario{}, err
	}
	if err := filterControl.DefineNetworkFilter(networkName, filterXML); err != nil {
		_ = networkControl.UndefineNetwork(networkName)
		_ = releaseNetworkAllocationLock(allocationLock)
		return Scenario{}, err
	}
	if err := releaseNetworkAllocationLock(allocationLock); err != nil {
		_ = filterControl.UndefineNetworkFilter(networkName)
		_ = networkControl.UndefineNetwork(networkName)
		return Scenario{}, fmt.Errorf("unlock scenario network allocation: %w", err)
	}
	networkStarted := false
	cleanupNetwork := true
	defer func() {
		if !cleanupNetwork {
			return
		}
		if networkStarted {
			_ = networkControl.DestroyNetwork(networkName)
		}
		_ = networkControl.UndefineNetwork(networkName)
		_ = filterControl.UndefineNetworkFilter(networkName)
	}()
	if err := networkControl.StartNetwork(networkName); err != nil {
		return Scenario{}, err
	}
	networkStarted = true

	scenario := Scenario{
		NetworkName:    networkName,
		lease:          networkLease,
		networkDefined: true,
		filterDefined:  true,
	}
	rollback := func(primary error) error {
		if rollbackErr := backend.rollbackScenarioInstances(scenario.Instances); rollbackErr != nil {
			// Preserve the shared network and lease as tracked backend state so
			// Close() can retry cleanup instead of tearing networking away from
			// a guest that failed to roll back.
			backend.mu.Lock()
			backend.scenarios[networkName] = scenario
			backend.mu.Unlock()
			cleanupNetwork = false
			cleanupLease = false
			return errors.Join(primary, fmt.Errorf("rollback scenario instances: %w", rollbackErr))
		}
		return primary
	}
	for _, definition := range definitions {
		if err := definition.Validate(); err != nil {
			return Scenario{}, rollback(fmt.Errorf("validate scenario VM definition: %w", err))
		}
		if definition.Network != runner.NetworkIsolated {
			return Scenario{}, rollback(errors.New("scenario machines must use network=isolated"))
		}
		name, err := vmInstanceName(definition.LabID)
		if err != nil {
			return Scenario{}, rollback(err)
		}
		if err := backend.prepareNamedOnNetwork(ctx, name, definition, networkName, false); err != nil {
			return Scenario{}, rollback(err)
		}
		scenario.Instances = append(scenario.Instances, runner.Instance{ID: name})
	}
	backend.mu.Lock()
	backend.scenarios[networkName] = scenario
	backend.mu.Unlock()

	cleanupNetwork = false
	cleanupLease = false
	return scenario, nil
}

func (backend *Backend) DestroyScenario(ctx context.Context, scenario Scenario) error {
	if scenario.NetworkName == "" {
		return errors.New("scenario network name is required")
	}
	if err := validateManagedResourceName("network", scenario.NetworkName); err != nil {
		return err
	}

	backend.mu.RLock()
	tracked, trackedScenario := backend.scenarios[scenario.NetworkName]
	backend.mu.RUnlock()
	if !trackedScenario {
		// A successfully destroyed scenario is no longer tracked. Treat repeated
		// teardown as success so cleanup remains idempotent.
		return nil
	}
	scenario = tracked

	// Do not tear down the shared network while one of its guests is still
	// active or failed to clean up. Keeping the lease/state makes the operation
	// retryable by Close() or by a later explicit DestroyScenario call.
	if err := backend.destroyScenarioInstances(ctx, scenario.Instances); err != nil {
		return err
	}

	if scenario.networkDefined {
		networkControl, ok := backend.control.(NetworkControlPlane)
		if !ok {
			return fmt.Errorf("%w: cannot destroy scenario network", runner.ErrNotSupported)
		}
		active, err := networkControl.NetworkActive(scenario.NetworkName)
		if err != nil {
			return err
		}
		if active {
			if err := networkControl.DestroyNetwork(scenario.NetworkName); err != nil {
				return err
			}
		}
		if err := networkControl.UndefineNetwork(scenario.NetworkName); err != nil {
			return err
		}
		scenario.networkDefined = false
		backend.updateTrackedScenario(scenario)
	}

	if scenario.filterDefined {
		filterControl, ok := backend.control.(NetworkFilterControlPlane)
		if !ok {
			return fmt.Errorf("%w: cannot destroy scenario network filter", runner.ErrNotSupported)
		}
		if err := filterControl.UndefineNetworkFilter(scenario.NetworkName); err != nil {
			return err
		}
		scenario.filterDefined = false
		backend.updateTrackedScenario(scenario)
	}

	if err := backend.overlays.Destroy(scenario.NetworkName); err != nil {
		return err
	}
	if err := releaseInstanceLease(scenario.lease); err != nil {
		return err
	}
	scenario.lease = nil

	backend.mu.Lock()
	delete(backend.scenarios, scenario.NetworkName)
	backend.mu.Unlock()
	return nil
}

func (backend *Backend) updateTrackedScenario(scenario Scenario) {
	backend.mu.Lock()
	if _, exists := backend.scenarios[scenario.NetworkName]; exists {
		backend.scenarios[scenario.NetworkName] = scenario
	}
	backend.mu.Unlock()
}

func (backend *Backend) rollbackScenarioInstances(instances []runner.Instance) error {
	rollbackCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return backend.destroyScenarioInstances(rollbackCtx, instances)
}

func (backend *Backend) destroyScenarioInstances(ctx context.Context, instances []runner.Instance) error {
	var errs []error
	for index := len(instances) - 1; index >= 0; index-- {
		if err := backend.Destroy(ctx, instances[index]); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
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

// PressPowerButton is limited to a running owned libvirt instance. A lab can
// observe the kernel ACPI event without requesting a guest-agent shutdown.
func (backend *Backend) PressPowerButton(ctx context.Context, instance runner.Instance) error {
	if err := ctx.Err(); err != nil { return err }
	if _, err := backend.instance(instance); err != nil { return err }
	state, err := backend.control.DomainState(instance.ID)
	if err != nil { return err }
	if !state.Active { return fmt.Errorf("VM instance %s is not active", instance.ID) }
	control, ok := backend.control.(ACPIControlPlane)
	if !ok { return fmt.Errorf("%w: no managed ACPI event support", runner.ErrNotSupported) }
	return control.PressACPIButtonDomain(instance.ID)
}

// RebootToBootMenu is nonblocking because waiting for the guest agent would
// consume the GRUB menu window before the learner could open the serial console.
func (backend *Backend) RebootToBootMenu(ctx context.Context, instance runner.Instance) error {
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
	return backend.control.RebootDomain(instance.ID)
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
	if err := console.OpenConsole(ctx, instance.ID, request.Stdin, request.Stdout); err != nil {
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

	// Scenario guests use a shared network owned by the scenario rather than by
	// the individual VM. Preserve that attachment across reset instead of
	// accidentally creating a new per-VM isolated network.
	sharedNetworkName := ""
	if managed.NetworkName != "" && !managed.NetworkDefined {
		sharedNetworkName = managed.NetworkName
	}

	if err := backend.destroyManaged(ctx, instance.ID, managed); err != nil {
		return err
	}
	if sharedNetworkName != "" {
		return backend.prepareNamedOnNetwork(
			ctx,
			instance.ID,
			managed.Definition,
			sharedNetworkName,
			false,
		)
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

	if managed.FilterDefined {
		filterControl, ok := backend.control.(NetworkFilterControlPlane)
		if !ok {
			return fmt.Errorf(
				"%w: libvirt control plane lost network-filter capability",
				runner.ErrNotSupported,
			)
		}
		if err := filterControl.UndefineNetworkFilter(managed.FilterName); err != nil {
			return err
		}
		managed.FilterDefined = false
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
	domains, err := inventory.ListManagedDomains()
	if err != nil {
		return err
	}
	networks, err := inventory.ListManagedNetworks()
	if err != nil {
		return err
	}
	filters, err := inventory.ListManagedNetworkFilters()
	if err != nil {
		return err
	}
	domainSet := make(map[string]struct{}, len(domains))
	networkSet := make(map[string]struct{}, len(networks))
	filterSet := make(map[string]struct{}, len(filters))
	candidates := make(map[string]struct{}, len(domains)+len(networks)+len(filters))
	for _, name := range domains {
		domainSet[name] = struct{}{}
		candidates[name] = struct{}{}
	}
	for _, name := range networks {
		networkSet[name] = struct{}{}
		candidates[name] = struct{}{}
	}
	for _, name := range filters {
		filterSet[name] = struct{}{}
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
		if _, exists := filterSet[name]; exists {
			filterControl, supported := backend.control.(NetworkFilterControlPlane)
			if !supported {
				errs = append(errs, fmt.Errorf("%w: cannot reap network filter %s", runner.ErrNotSupported, name))
				_ = releaseInstanceLease(lease)
				continue
			}
			if undefineErr := filterControl.UndefineNetworkFilter(name); undefineErr != nil {
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
		if errors.Is(err, errUnknownVMInstance) {
			continue
		}
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if err := backend.destroyManaged(ctx, name, managed); err != nil {
			errs = append(errs, err)
		}
	}

	// Scenario networks and their leases are backend-owned resources too.
	// Clean them even when the caller forgot an explicit DestroyScenario().
	backend.mu.RLock()
	scenarios := make([]Scenario, 0, len(backend.scenarios))
	for _, scenario := range backend.scenarios {
		scenarios = append(scenarios, scenario)
	}
	backend.mu.RUnlock()
	for _, scenario := range scenarios {
		if err := backend.DestroyScenario(ctx, scenario); err != nil {
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
