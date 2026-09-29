package libvirt

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Loe159/lpic-daily/internal/runner"
)

type fakeControlPlane struct {
	ownerScope       string
	defined          map[string]string
	active           map[string]bool
	networks         map[string]string
	networkActive    map[string]bool
	starts           int
	reboots          int
	destroys         int
	undefines        int
	networkStarts    int
	networkDestroys  int
	networkUndefines int
	removeNVRAM      bool
	consoleOpens     int
	agentResponses   []string
	agentErrors      []error
	agentCommands    []string
	destroyErr       error
	undefineErr      error
	closed           bool
}

func newFakeControlPlane() *fakeControlPlane {
	return &fakeControlPlane{
		defined:       make(map[string]string),
		active:        make(map[string]bool),
		networks:      make(map[string]string),
		networkActive: make(map[string]bool),
	}
}

func (fake *fakeControlPlane) SetManagedOwnerScope(scope string) error {
	if err := validateManagedOwnerScope(scope); err != nil {
		return err
	}
	if fake.ownerScope != "" && fake.ownerScope != scope {
		return errors.New("owner scope already configured")
	}
	fake.ownerScope = scope
	return nil
}
func (fake *fakeControlPlane) LibVersion() (uint64, error) { return 1000000, nil }
func (fake *fakeControlPlane) Capabilities() (string, error) {
	return "<arch>x86_64</arch>", nil
}
func (fake *fakeControlPlane) DefineDomain(name, xml string) error {
	if _, exists := fake.defined[name]; exists {
		return errors.New("domain already defined")
	}
	fake.defined[name] = xml
	fake.active[name] = false
	return nil
}
func (fake *fakeControlPlane) StartDomain(name string) error {
	if _, exists := fake.defined[name]; !exists {
		return errors.New("domain missing")
	}
	fake.starts++
	fake.active[name] = true
	return nil
}
func (fake *fakeControlPlane) RebootDomain(name string) error {
	if !fake.active[name] {
		return errors.New("domain not active")
	}
	fake.reboots++
	return nil
}
func (fake *fakeControlPlane) OpenConsole(_ context.Context, name string, input io.Reader, output io.Writer) error {
	if !fake.active[name] {
		return errors.New("domain not active")
	}
	fake.consoleOpens++
	_, err := io.Copy(output, input)
	return err
}

func (fake *fakeControlPlane) AgentCommand(name, command string, timeoutSeconds int32) (string, error) {
	if !fake.active[name] {
		return "", errors.New("domain not active")
	}
	if timeoutSeconds <= 0 {
		return "", errors.New("invalid timeout")
	}
	fake.agentCommands = append(fake.agentCommands, command)
	if len(fake.agentErrors) != 0 {
		err := fake.agentErrors[0]
		fake.agentErrors = fake.agentErrors[1:]
		if err != nil {
			return "", err
		}
	}
	if len(fake.agentResponses) == 0 {
		return "", errors.New("no fake guest-agent response")
	}
	response := fake.agentResponses[0]
	fake.agentResponses = fake.agentResponses[1:]
	return response, nil
}

func (fake *fakeControlPlane) DomainState(name string) (DomainState, error) {
	if _, exists := fake.defined[name]; !exists {
		return DomainState{}, errors.New("domain missing")
	}
	return DomainState{State: 1, Active: fake.active[name]}, nil
}
func (fake *fakeControlPlane) DestroyDomain(name string) error {
	if !fake.active[name] {
		return errors.New("domain not active")
	}
	if fake.destroyErr != nil {
		return fake.destroyErr
	}
	fake.destroys++
	fake.active[name] = false
	return nil
}
func (fake *fakeControlPlane) UndefineDomain(name string, removeNVRAM bool) error {
	if _, exists := fake.defined[name]; !exists {
		return errors.New("domain missing")
	}
	if fake.active[name] {
		return errors.New("domain active")
	}
	if fake.undefineErr != nil {
		return fake.undefineErr
	}
	fake.undefines++
	fake.removeNVRAM = removeNVRAM
	delete(fake.defined, name)
	delete(fake.active, name)
	return nil
}
func (fake *fakeControlPlane) DefineNetwork(name, xml string) error {
	if _, exists := fake.networks[name]; exists {
		return errors.New("network already defined")
	}
	fake.networks[name] = xml
	fake.networkActive[name] = false
	return nil
}
func (fake *fakeControlPlane) StartNetwork(name string) error {
	if _, exists := fake.networks[name]; !exists {
		return errors.New("network missing")
	}
	fake.networkStarts++
	fake.networkActive[name] = true
	return nil
}
func (fake *fakeControlPlane) NetworkActive(name string) (bool, error) {
	if _, exists := fake.networks[name]; !exists {
		return false, errors.New("network missing")
	}
	return fake.networkActive[name], nil
}
func (fake *fakeControlPlane) DestroyNetwork(name string) error {
	if !fake.networkActive[name] {
		return errors.New("network not active")
	}
	fake.networkDestroys++
	fake.networkActive[name] = false
	return nil
}
func (fake *fakeControlPlane) UndefineNetwork(name string) error {
	if _, exists := fake.networks[name]; !exists {
		return errors.New("network missing")
	}
	if fake.networkActive[name] {
		return errors.New("network active")
	}
	fake.networkUndefines++
	delete(fake.networks, name)
	delete(fake.networkActive, name)
	return nil
}
func (fake *fakeControlPlane) ListManagedDomains() ([]string, error) {
	names := make([]string, 0, len(fake.defined))
	for name, resourceXML := range fake.defined {
		if hasManagedMetadata(resourceXML, fake.ownerScope) {
			names = append(names, name)
		}
	}
	return names, nil
}
func (fake *fakeControlPlane) ListManagedNetworks() ([]string, error) {
	names := make([]string, 0, len(fake.networks))
	for name, resourceXML := range fake.networks {
		if hasManagedMetadata(resourceXML, fake.ownerScope) {
			names = append(names, name)
		}
	}
	return names, nil
}

func (fake *fakeControlPlane) ListNetworkIPv4Prefixes() ([]netip.Prefix, error) {
	var prefixes []netip.Prefix
	for _, resourceXML := range fake.networks {
		found, err := networkPrefixesFromXML(resourceXML)
		if err != nil {
			return nil, err
		}
		prefixes = append(prefixes, found...)
	}
	return prefixes, nil
}
func (fake *fakeControlPlane) Close() error {
	fake.closed = true
	return nil
}

func backendFixture(t *testing.T) (*Backend, *fakeControlPlane, *fakeCommands, runner.Definition) {
	t.Helper()
	root := t.TempDir()
	imageRoot := filepath.Join(root, "images")
	stateRoot := filepath.Join(root, "state")
	if err := os.MkdirAll(stateRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll(state root) error = %v", err)
	}
	imagePath := filepath.Join(imageRoot, "fedora", "base.qcow2")
	if err := os.MkdirAll(filepath.Dir(imagePath), 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	content := []byte("trusted VM base")
	if err := os.WriteFile(imagePath, content, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	digest := sha256.Sum256(content)

	catalog := &ImageCatalog{
		SchemaVersion: "1.0.0",
		Images: []ImageCatalogEntry{{
			ID:            "fedora-44-x86_64-v1",
			RelativePath:  "fedora/base.qcow2",
			SHA256:        hex.EncodeToString(digest[:]),
			Format:        "qcow2",
			Architecture:  "x86_64",
			Distribution:  "fedora",
			Version:       "44",
			VirtualSizeMB: 8192,
			Firmware:      []string{"bios", "uefi"},
			Provenance: Provenance{
				SourceURL:   "https://example.invalid/image",
				BuildRecipe: "test",
				BuiltAt:     "2026-09-27T00:00:00Z",
			},
		}},
	}
	control := newFakeControlPlane()
	commands := &fakeCommands{}
	networkLockPath := filepath.Join(root, "network-allocation.lock")
	if err := os.WriteFile(networkLockPath, nil, 0o600); err != nil {
		t.Fatalf("WriteFile(network lock) error = %v", err)
	}
	backend, err := NewBackend(control, catalog, imageRoot, stateRoot, networkLockPath, commands)
	if err != nil {
		t.Fatalf("NewBackend() error = %v", err)
	}
	definition := runner.Definition{
		LabID:             "lpic1.104.1.partition-disk",
		ImageRef:          "fedora-44-x86_64-v1",
		Distribution:      "fedora",
		Network:           runner.NetworkNone,
		CapabilityProfile: "full-machine",
		MemoryMB:          1024,
		CPUPercent:        100,
		PIDs:              128,
		Timeout:           20 * time.Minute,
		Machine: &runner.MachineDefinition{
			Firmware: runner.FirmwareUEFI,
			ExtraDisks: []runner.VirtualDisk{{
				ID:     "data",
				SizeMB: 512,
			}},
		},
	}
	return backend, control, commands, definition
}

func TestManagedOwnerScopeSeparatesStateRoots(t *testing.T) {
	firstRoot := filepath.Join(t.TempDir(), "state-a")
	secondRoot := filepath.Join(t.TempDir(), "state-b")

	first, err := managedOwnerScope(firstRoot)
	if err != nil {
		t.Fatalf("managedOwnerScope(first) error = %v", err)
	}
	again, err := managedOwnerScope(firstRoot)
	if err != nil {
		t.Fatalf("managedOwnerScope(first again) error = %v", err)
	}
	second, err := managedOwnerScope(secondRoot)
	if err != nil {
		t.Fatalf("managedOwnerScope(second) error = %v", err)
	}
	if first != again {
		t.Fatalf("owner scope is not stable: first=%q again=%q", first, again)
	}
	if first == second {
		t.Fatalf("different state roots share owner scope %q", first)
	}
}

func TestBackendPrepareStartDestroyLifecycle(t *testing.T) {
	backend, control, commands, definition := backendFixture(t)
	ctx := context.Background()

	instance, err := backend.Prepare(ctx, definition)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if !strings.HasPrefix(instance.ID, "lpic-daily-") {
		t.Fatalf("instance ID = %q", instance.ID)
	}
	xml := control.defined[instance.ID]
	if !strings.Contains(xml, `<target dev="vdb" bus="virtio"></target>`) {
		t.Fatalf("domain XML missing scratch disk:\n%s", xml)
	}
	if len(commands.Calls) != 2 {
		t.Fatalf("qemu-img calls = %d, want root overlay + scratch disk", len(commands.Calls))
	}

	if err := backend.Start(ctx, instance); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if control.starts != 1 || !control.active[instance.ID] {
		t.Fatalf("start state = %#v", control)
	}

	if err := backend.Destroy(ctx, instance); err != nil {
		t.Fatalf("Destroy() error = %v", err)
	}
	if control.destroys != 1 || control.undefines != 1 || !control.removeNVRAM {
		t.Fatalf("cleanup state = %#v", control)
	}
	if err := backend.Destroy(ctx, instance); err != nil {
		t.Fatalf("second Destroy() must be idempotent: %v", err)
	}
}

func TestBackendDestroyFailureKeepsInstanceTrackedAndDisksIntact(t *testing.T) {
	backend, control, _, definition := backendFixture(t)
	ctx := context.Background()

	instance, err := backend.Prepare(ctx, definition)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if err := backend.Start(ctx, instance); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	managed, err := backend.instance(instance)
	if err != nil {
		t.Fatalf("instance() error = %v", err)
	}

	control.destroyErr = errors.New("simulated destroy failure")
	if err := backend.Destroy(ctx, instance); err == nil {
		t.Fatal("Destroy() unexpectedly succeeded")
	}
	if !control.active[instance.ID] {
		t.Fatal("failed destroy unexpectedly marked the domain inactive")
	}
	if _, err := backend.instance(instance); err != nil {
		t.Fatalf("failed destroy lost tracked instance: %v", err)
	}
	if _, err := os.Stat(managed.Paths.Directory); err != nil {
		t.Fatalf("failed destroy removed VM disks: %v", err)
	}

	control.destroyErr = nil
	if err := backend.Destroy(ctx, instance); err != nil {
		t.Fatalf("Destroy() retry error = %v", err)
	}
	if _, err := os.Stat(managed.Paths.Directory); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("successful retry left VM disks behind: %v", err)
	}
}

func TestBackendUndefineFailureKeepsCleanupRetryable(t *testing.T) {
	backend, control, _, definition := backendFixture(t)
	ctx := context.Background()

	instance, err := backend.Prepare(ctx, definition)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if err := backend.Start(ctx, instance); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	managed, err := backend.instance(instance)
	if err != nil {
		t.Fatalf("instance() error = %v", err)
	}

	control.undefineErr = errors.New("simulated undefine failure")
	if err := backend.Destroy(ctx, instance); err == nil {
		t.Fatal("Destroy() unexpectedly succeeded")
	}
	if control.active[instance.ID] {
		t.Fatal("domain should have been stopped before undefine failure")
	}
	if _, exists := control.defined[instance.ID]; !exists {
		t.Fatal("failed undefine unexpectedly removed the domain")
	}
	if _, err := backend.instance(instance); err != nil {
		t.Fatalf("failed undefine lost tracked instance: %v", err)
	}
	if _, err := os.Stat(managed.Paths.Directory); err != nil {
		t.Fatalf("failed undefine removed VM disks: %v", err)
	}

	control.undefineErr = nil
	if err := backend.Destroy(ctx, instance); err != nil {
		t.Fatalf("Destroy() retry error = %v", err)
	}
	if _, err := os.Stat(managed.Paths.Directory); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("successful retry left VM disks behind: %v", err)
	}
}

func TestBackendSerialConsoleRequiresActiveManagedVM(t *testing.T) {
	backend, control, _, definition := backendFixture(t)
	ctx := context.Background()

	instance, err := backend.Prepare(ctx, definition)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	var output strings.Builder
	request := runner.ConsoleRequest{
		Stdin:  strings.NewReader("grub> help\n"),
		Stdout: &output,
	}
	if err := backend.OpenConsole(ctx, instance, request); err == nil ||
		!strings.Contains(err.Error(), "not active") {
		t.Fatalf("OpenConsole(inactive) error = %v", err)
	}
	if err := backend.Start(ctx, instance); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := backend.OpenConsole(ctx, instance, request); err != nil {
		t.Fatalf("OpenConsole() error = %v", err)
	}
	if output.String() != "grub> help\n" || control.consoleOpens != 1 {
		t.Fatalf("console output=%q opens=%d", output.String(), control.consoleOpens)
	}
	if err := backend.OpenConsole(
		ctx,
		runner.Instance{ID: "lpic-daily-not-tracked"},
		request,
	); err == nil || !errors.Is(err, errUnknownVMInstance) {
		t.Fatalf("OpenConsole(unmanaged) error = %v", err)
	}
}

func TestBackendExecUsesStructuredGuestAgentCommand(t *testing.T) {
	backend, control, _, definition := backendFixture(t)
	ctx := context.Background()

	instance, err := backend.Prepare(ctx, definition)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if err := backend.Start(ctx, instance); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	control.agentResponses = []string{
		`{"return":{}}`,
		`{"return":{"pid":42}}`,
		`{"return":{"exited":false}}`,
		`{"return":{"exited":true,"exitcode":7,"out-data":"b2sK","err-data":"ZXJyCg=="}}`,
	}

	var stdout strings.Builder
	var stderr strings.Builder
	result, err := backend.Exec(ctx, instance, runner.ExecRequest{
		Argv:   []string{"/usr/bin/false", "--example"},
		Env:    map[string]string{"Z_LAST": "z", "A_FIRST": "a"},
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		t.Fatalf("Exec() error = %v", err)
	}
	if result.ExitCode != 7 || stdout.String() != "ok\n" || stderr.String() != "err\n" {
		t.Fatalf(
			"result=%#v stdout=%q stderr=%q",
			result,
			stdout.String(),
			stderr.String(),
		)
	}
	if len(control.agentCommands) != 4 {
		t.Fatalf("agent commands = %d, want guest-ping + 3 exec calls", len(control.agentCommands))
	}
	if control.agentCommands[0] != `{"execute":"guest-ping"}` {
		t.Fatalf("first agent command = %s, want guest-ping", control.agentCommands[0])
	}
	if !strings.Contains(control.agentCommands[1], `"path":"/usr/bin/false"`) ||
		!strings.Contains(control.agentCommands[1], `"arg":["--example"]`) ||
		!strings.Contains(control.agentCommands[1], `"env":["A_FIRST=a","Z_LAST=z"]`) {
		t.Fatalf("guest-exec request = %s", control.agentCommands[1])
	}
	if strings.Contains(control.agentCommands[1], "/bin/sh") {
		t.Fatalf("guest-exec unexpectedly introduced a shell: %s", control.agentCommands[1])
	}
}

func TestBackendExecWaitsForGuestAgentReadiness(t *testing.T) {
	backend, control, _, definition := backendFixture(t)
	ctx := context.Background()

	instance, err := backend.Prepare(ctx, definition)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if err := backend.Start(ctx, instance); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	control.agentErrors = []error{errors.New("guest agent not ready"), nil}
	control.agentResponses = []string{
		`{"return":{}}`,
		`{"return":{"pid":7}}`,
		`{"return":{"exited":true,"exitcode":0}}`,
	}

	if _, err := backend.Exec(ctx, instance, runner.ExecRequest{
		Argv: []string{"/usr/bin/true"},
	}); err != nil {
		t.Fatalf("Exec() error = %v", err)
	}
	if len(control.agentCommands) != 4 {
		t.Fatalf("agent commands = %d, want two pings + exec + status", len(control.agentCommands))
	}
	if control.agentCommands[0] != `{"execute":"guest-ping"}` ||
		control.agentCommands[1] != `{"execute":"guest-ping"}` {
		t.Fatalf("readiness commands = %#v", control.agentCommands[:2])
	}
}

func TestBackendExecStopsWaitingForGuestAgentWhenContextExpires(t *testing.T) {
	backend, control, _, definition := backendFixture(t)
	ctx := context.Background()

	instance, err := backend.Prepare(ctx, definition)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if err := backend.Start(ctx, instance); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	control.agentErrors = make([]error, 16)
	for index := range control.agentErrors {
		control.agentErrors[index] = errors.New("guest agent unavailable")
	}
	deadlineCtx, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
	defer cancel()

	_, err = backend.Exec(deadlineCtx, instance, runner.ExecRequest{
		Argv: []string{"/usr/bin/true"},
	})
	if err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Exec() error = %v, want context deadline", err)
	}
}

func TestBackendExecRejectsTTYAndWorkingDirectory(t *testing.T) {
	backend, _, _, definition := backendFixture(t)
	ctx := context.Background()
	instance, err := backend.Prepare(ctx, definition)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if err := backend.Start(ctx, instance); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	for _, request := range []runner.ExecRequest{
		{Argv: []string{"/bin/bash"}, TTY: true},
		{Argv: []string{"/bin/pwd"}, WorkingDir: "/tmp"},
	} {
		if _, err := backend.Exec(ctx, instance, request); !errors.Is(err, runner.ErrNotSupported) {
			t.Fatalf("Exec(%#v) error = %v, want ErrNotSupported", request, err)
		}
	}
}

func TestBackendResetRecreatesSameManagedIdentity(t *testing.T) {
	backend, control, commands, definition := backendFixture(t)
	ctx := context.Background()

	instance, err := backend.Prepare(ctx, definition)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if err := backend.Start(ctx, instance); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := backend.Reset(ctx, instance); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}
	if _, exists := control.defined[instance.ID]; !exists {
		t.Fatal("reset did not redefine the same domain identity")
	}
	if control.active[instance.ID] {
		t.Fatal("reset domain should be prepared but not started")
	}
	if len(commands.Calls) != 4 {
		t.Fatalf("qemu-img calls = %d, want two per preparation", len(commands.Calls))
	}
}

func TestBackendIsolatedNetworkLifecycle(t *testing.T) {
	backend, control, _, definition := backendFixture(t)
	definition.Network = runner.NetworkIsolated
	ctx := context.Background()

	instance, err := backend.Prepare(ctx, definition)
	if err != nil {
		t.Fatalf("Prepare(network=isolated) error = %v", err)
	}
	if len(control.networks) != 1 || !control.networkActive[instance.ID] {
		t.Fatalf("isolated network state = %#v", control)
	}
	networkXML := control.networks[instance.ID]
	if strings.Contains(networkXML, "<forward") {
		t.Fatalf("isolated network forwards traffic:\n%s", networkXML)
	}
	domainXML := control.defined[instance.ID]
	if !strings.Contains(domainXML, `<source network="`+instance.ID+`"></source>`) {
		t.Fatalf("domain XML does not attach isolated network:\n%s", domainXML)
	}
	if err := backend.Start(ctx, instance); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := backend.Destroy(ctx, instance); err != nil {
		t.Fatalf("Destroy() error = %v", err)
	}
	if len(control.networks) != 0 || control.networkDestroys != 1 || control.networkUndefines != 1 {
		t.Fatalf("network cleanup state = %#v", control)
	}
}

func TestBackendRebootRequiresActiveManagedVM(t *testing.T) {
	backend, control, _, definition := backendFixture(t)
	ctx := context.Background()
	instance, err := backend.Prepare(ctx, definition)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if err := backend.Reboot(ctx, instance); err == nil || !strings.Contains(err.Error(), "not active") {
		t.Fatalf("Reboot(inactive) error = %v", err)
	}
	if err := backend.Start(ctx, instance); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	control.agentResponses = []string{
		`{"return":{}}`,
		`{"return":{"pid":10}}`,
		`{"return":{"exited":true,"exitcode":0,"out-data":"Ym9vdC1hCg=="}}`,
		`{"return":{}}`,
		`{"return":{"pid":11}}`,
		`{"return":{"exited":true,"exitcode":0,"out-data":"Ym9vdC1iCg=="}}`,
	}
	if err := backend.Reboot(ctx, instance); err != nil {
		t.Fatalf("Reboot() error = %v", err)
	}
	if control.reboots != 1 {
		t.Fatalf("reboots = %d, want 1", control.reboots)
	}
}

func TestBackendFailsClosedForUnsupportedGuestOperations(t *testing.T) {
	backend, _, _, _ := backendFixture(t)
	instance := runner.Instance{ID: "lpic-daily-not-real"}
	if _, err := backend.Exec(
		context.Background(),
		instance,
		runner.ExecRequest{Argv: []string{"/usr/bin/true"}},
	); !errors.Is(err, errUnknownVMInstance) {
		t.Fatalf("Exec(unmanaged) error = %v, want errUnknownVMInstance", err)
	}
	if _, err := backend.Stat(context.Background(), instance, "/etc/passwd"); !errors.Is(err, runner.ErrNotSupported) {
		t.Fatalf("Stat() error = %v", err)
	}
}

func TestBackendRejectsImageDistributionMismatchBeforeOverlayCreation(t *testing.T) {
	backend, control, commands, definition := backendFixture(t)
	definition.Distribution = "debian"

	if _, err := backend.Prepare(context.Background(), definition); err == nil ||
		!strings.Contains(err.Error(), "distribution=fedora") {
		t.Fatalf("Prepare() error = %v", err)
	}
	if len(control.defined) != 0 || len(commands.Calls) != 0 {
		t.Fatalf("distribution mismatch mutated VM state")
	}
}

func TestBackendScenarioSharesOneIsolatedNetwork(t *testing.T) {
	backend, control, _, definition := backendFixture(t)
	definition.Network = runner.NetworkIsolated
	second := definition
	second.LabID = "lpic1.109.2.peer-network"
	ctx := context.Background()

	scenario, err := backend.PrepareScenario(ctx, []runner.Definition{definition, second})
	if err != nil {
		t.Fatalf("PrepareScenario() error = %v", err)
	}
	if len(scenario.Instances) != 2 {
		t.Fatalf("scenario instances = %d, want 2", len(scenario.Instances))
	}
	if len(control.networks) != 1 || !control.networkActive[scenario.NetworkName] {
		t.Fatalf("scenario network state = %#v", control)
	}
	for _, instance := range scenario.Instances {
		xml := control.defined[instance.ID]
		if !strings.Contains(xml, `<source network="`+scenario.NetworkName+`"></source>`) {
			t.Fatalf("domain %s does not use shared network:\n%s", instance.ID, xml)
		}
	}
	if err := backend.DestroyScenario(ctx, scenario); err != nil {
		t.Fatalf("DestroyScenario() error = %v", err)
	}
	if len(control.defined) != 0 || len(control.networks) != 0 {
		t.Fatalf("scenario cleanup leaked resources: domains=%#v networks=%#v", control.defined, control.networks)
	}
	if control.networkStarts != 1 || control.networkDestroys != 1 || control.networkUndefines != 1 {
		t.Fatalf("scenario network lifecycle = starts:%d destroys:%d undefines:%d",
			control.networkStarts, control.networkDestroys, control.networkUndefines)
	}
}

func TestBackendReapRemovesAbandonedResources(t *testing.T) {
	backend, control, _, _ := backendFixture(t)
	name := "lpic-daily-orphan-abc123"
	directory := filepath.Join(backend.stateRoot, name)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(directory, "root.qcow2"), []byte("orphan"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	control.defined[name] = managedTestDomainXML
	control.active[name] = true
	control.networks[name] = managedTestNetworkXML
	control.networkActive[name] = true

	if err := backend.Reap(context.Background()); err != nil {
		t.Fatalf("Reap() error = %v", err)
	}
	if _, exists := control.defined[name]; exists {
		t.Fatal("orphan domain still defined")
	}
	if _, exists := control.networks[name]; exists {
		t.Fatal("orphan network still defined")
	}
	if _, err := os.Stat(directory); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("orphan state directory still exists: %v", err)
	}
}

func TestBackendReapDoesNotTouchPrefixedForeignLibvirtResources(t *testing.T) {
	backend, control, _, _ := backendFixture(t)
	name := "lpic-daily-foreign-abc123"
	control.defined[name] = "<domain/>"
	control.active[name] = true
	control.networks[name] = "<network/>"
	control.networkActive[name] = true

	if err := backend.Reap(context.Background()); err != nil {
		t.Fatalf("Reap() error = %v", err)
	}
	if _, exists := control.defined[name]; !exists {
		t.Fatal("foreign prefixed domain was reaped")
	}
	if _, exists := control.networks[name]; !exists {
		t.Fatal("foreign prefixed network was reaped")
	}
}

func TestBackendReapSkipsLiveLease(t *testing.T) {
	backend, control, _, _ := backendFixture(t)
	name := "lpic-daily-live-abc123"
	directory := filepath.Join(backend.stateRoot, name)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	lease, err := acquireInstanceLease(directory)
	if err != nil {
		t.Fatalf("acquireInstanceLease() error = %v", err)
	}
	defer releaseInstanceLease(lease)
	control.defined[name] = managedTestDomainXML
	control.active[name] = true

	if err := backend.Reap(context.Background()); err != nil {
		t.Fatalf("Reap() error = %v", err)
	}
	if _, exists := control.defined[name]; !exists {
		t.Fatal("live leased domain was reaped")
	}
	if _, err := os.Stat(directory); err != nil {
		t.Fatalf("live state directory removed: %v", err)
	}
}

func TestVMInstanceNameIsSafe(t *testing.T) {
	name, err := vmInstanceName("../../Storage Lab !!")
	if err != nil {
		t.Fatalf("vmInstanceName() error = %v", err)
	}
	if !managedNamePattern.MatchString(name) {
		t.Fatalf("generated name = %q", name)
	}
	if strings.ContainsAny(name, "/ !") {
		t.Fatalf("unsafe generated name = %q", name)
	}
}
