package libvirt

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Loe159/lpic-daily/internal/runner"
)

type fakeControlPlane struct {
	defined      map[string]string
	active       map[string]bool
	starts       int
	destroys     int
	undefines    int
	removeNVRAM  bool
	closed       bool
}

func newFakeControlPlane() *fakeControlPlane {
	return &fakeControlPlane{
		defined: make(map[string]string),
		active:  make(map[string]bool),
	}
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
	fake.undefines++
	fake.removeNVRAM = removeNVRAM
	delete(fake.defined, name)
	delete(fake.active, name)
	return nil
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
	backend, err := NewBackend(control, catalog, imageRoot, stateRoot, commands)
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

func TestBackendFailsClosedForNetworkingAndGuestOperations(t *testing.T) {
	backend, control, commands, definition := backendFixture(t)
	definition.Network = runner.NetworkIsolated

	if _, err := backend.Prepare(context.Background(), definition); !errors.Is(err, runner.ErrNotSupported) {
		t.Fatalf("Prepare(network=isolated) error = %v", err)
	}
	if len(control.defined) != 0 || len(commands.Calls) != 0 {
		t.Fatalf("unsupported network mutated VM state: control=%#v commands=%#v", control, commands.Calls)
	}

	instance := runner.Instance{ID: "lpic-daily-not-real"}
	if _, err := backend.Exec(context.Background(), instance, runner.ExecRequest{}); !errors.Is(err, runner.ErrNotSupported) {
		t.Fatalf("Exec() error = %v", err)
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
