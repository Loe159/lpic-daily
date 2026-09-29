package libvirt

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	golibvirt "github.com/digitalocean/go-libvirt"
)

type fakeRawLibvirt struct {
	libVersion       uint64
	capabilities     string
	domain           golibvirt.Domain
	definedXML       string
	defineFlags      golibvirt.DomainDefineFlags
	createFlags      uint32
	rebootFlags      golibvirt.DomainRebootFlagValues
	destroyFlags     golibvirt.DomainDestroyFlagsValues
	undefineFlags    golibvirt.DomainUndefineFlagsValues
	state            int32
	reason           int32
	consoleDomain    golibvirt.Domain
	consoleDevice    golibvirt.OptString
	consoleFlags     uint32
	agentCommand     string
	agentTimeout     int32
	network          golibvirt.Network
	networkXML       string
	networkActive    int32
	networkDestroyed bool
	networkUndefined bool
	disconnected     bool
	err              error
	consoleBlock     chan struct{}
	consoleStarted   chan struct{}
	consoleClosed    bool
}

const testManagedOwnerScope = "0123456789abcdef0123456789abcdef"

const managedTestDomainXML = `<domain><metadata><lpic-daily xmlns="urn:lpic-daily:managed:v1" owner="lpic-daily" version="1" scope="` + testManagedOwnerScope + `"></lpic-daily></metadata></domain>`
const managedTestNetworkXML = `<network><metadata><lpic-daily xmlns="urn:lpic-daily:managed:v1" owner="lpic-daily" version="1" scope="` + testManagedOwnerScope + `"></lpic-daily></metadata></network>`

func newScopedRPCControlPlaneForTest(t *testing.T, raw rawLibvirt) (*RPCControlPlane, error) {
	t.Helper()
	control, err := newRPCControlPlane(raw)
	if err != nil {
		return nil, err
	}
	if err := control.SetManagedOwnerScope(testManagedOwnerScope); err != nil {
		return nil, err
	}
	return control, nil
}

func (fake *fakeRawLibvirt) ConnectGetLibVersion() (uint64, error) {
	return fake.libVersion, fake.err
}

func (fake *fakeRawLibvirt) ConnectGetCapabilities() (string, error) {
	return fake.capabilities, fake.err
}

func (fake *fakeRawLibvirt) DomainDefineXMLFlags(
	xml string,
	flags golibvirt.DomainDefineFlags,
) (golibvirt.Domain, error) {
	fake.definedXML = xml
	fake.defineFlags = flags
	return fake.domain, fake.err
}

func (fake *fakeRawLibvirt) DomainLookupByName(name string) (golibvirt.Domain, error) {
	if fake.err != nil {
		return golibvirt.Domain{}, fake.err
	}
	domain := fake.domain
	if domain.Name == "" {
		domain.Name = name
	}
	return domain, nil
}

func (fake *fakeRawLibvirt) DomainCreateWithFlags(
	domain golibvirt.Domain,
	flags uint32,
) (golibvirt.Domain, error) {
	fake.createFlags = flags
	if fake.err != nil {
		return golibvirt.Domain{}, fake.err
	}
	return domain, nil
}

func (fake *fakeRawLibvirt) DomainReboot(
	_ golibvirt.Domain,
	flags golibvirt.DomainRebootFlagValues,
) error {
	fake.rebootFlags = flags
	return fake.err
}

func (fake *fakeRawLibvirt) DomainOpenConsoleBidirectional(
	domain golibvirt.Domain,
	device golibvirt.OptString,
	input io.Reader,
	output io.Writer,
	flags uint32,
) error {
	fake.consoleDomain = domain
	fake.consoleDevice = device
	fake.consoleFlags = flags
	if fake.err != nil {
		return fake.err
	}
	if fake.consoleStarted != nil {
		close(fake.consoleStarted)
	}
	if fake.consoleBlock != nil {
		<-fake.consoleBlock
		return nil
	}
	_, err := io.Copy(output, input)
	return err
}

func (fake *fakeRawLibvirt) QEMUDomainAgentCommand(
	domain golibvirt.Domain,
	command string,
	timeout int32,
	flags uint32,
) (golibvirt.OptString, error) {
	if fake.err != nil {
		return nil, fake.err
	}
	fake.consoleDomain = domain
	fake.agentCommand = command
	fake.agentTimeout = timeout
	if flags != 0 {
		return nil, errors.New("unexpected guest-agent flags")
	}
	return golibvirt.OptString{`{"return":{"ok":true}}`}, nil
}

func (fake *fakeRawLibvirt) DomainGetState(
	_ golibvirt.Domain,
	_ uint32,
) (int32, int32, error) {
	return fake.state, fake.reason, fake.err
}

func (fake *fakeRawLibvirt) DomainGetXMLDesc(
	_ golibvirt.Domain,
	_ golibvirt.DomainXMLFlags,
) (string, error) {
	return fake.definedXML, fake.err
}

func (fake *fakeRawLibvirt) DomainDestroyFlags(
	_ golibvirt.Domain,
	flags golibvirt.DomainDestroyFlagsValues,
) error {
	fake.destroyFlags = flags
	if fake.consoleBlock != nil && !fake.consoleClosed {
		close(fake.consoleBlock)
		fake.consoleClosed = true
	}
	return fake.err
}

func (fake *fakeRawLibvirt) DomainUndefineFlags(
	_ golibvirt.Domain,
	flags golibvirt.DomainUndefineFlagsValues,
) error {
	fake.undefineFlags = flags
	return fake.err
}

func (fake *fakeRawLibvirt) NetworkDefineXML(xml string) (golibvirt.Network, error) {
	fake.networkXML = xml
	if fake.err != nil {
		return golibvirt.Network{}, fake.err
	}
	return fake.network, nil
}

func (fake *fakeRawLibvirt) NetworkLookupByName(name string) (golibvirt.Network, error) {
	if fake.err != nil {
		return golibvirt.Network{}, fake.err
	}
	network := fake.network
	if network.Name == "" {
		network.Name = name
	}
	return network, nil
}

func (fake *fakeRawLibvirt) NetworkCreate(_ golibvirt.Network) error {
	if fake.err != nil {
		return fake.err
	}
	fake.networkActive = 1
	return nil
}

func (fake *fakeRawLibvirt) NetworkIsActive(_ golibvirt.Network) (int32, error) {
	return fake.networkActive, fake.err
}

func (fake *fakeRawLibvirt) NetworkGetXMLDesc(_ golibvirt.Network, _ uint32) (string, error) {
	return fake.networkXML, fake.err
}

func (fake *fakeRawLibvirt) NetworkDestroy(_ golibvirt.Network) error {
	if fake.err != nil {
		return fake.err
	}
	fake.networkDestroyed = true
	fake.networkActive = 0
	return nil
}

func (fake *fakeRawLibvirt) NetworkUndefine(_ golibvirt.Network) error {
	if fake.err != nil {
		return fake.err
	}
	fake.networkUndefined = true
	return nil
}

func (fake *fakeRawLibvirt) ConnectListAllDomains(
	_ int32,
	_ golibvirt.ConnectListAllDomainsFlags,
) ([]golibvirt.Domain, uint32, error) {
	if fake.err != nil {
		return nil, 0, fake.err
	}
	if fake.domain.Name == "" {
		return nil, 0, nil
	}
	return []golibvirt.Domain{fake.domain}, 1, nil
}

func (fake *fakeRawLibvirt) ConnectListAllNetworks(
	_ int32,
	_ golibvirt.ConnectListAllNetworksFlags,
) ([]golibvirt.Network, uint32, error) {
	if fake.err != nil {
		return nil, 0, fake.err
	}
	if fake.network.Name == "" {
		return nil, 0, nil
	}
	return []golibvirt.Network{fake.network}, 1, nil
}

func (fake *fakeRawLibvirt) Disconnect() error {
	fake.disconnected = true
	if fake.consoleBlock != nil && !fake.consoleClosed {
		close(fake.consoleBlock)
		fake.consoleClosed = true
	}
	return fake.err
}

func TestRPCControlPlaneLifecycleUsesManagedDomainOnly(t *testing.T) {
	name := "lpic-daily-vm-abc123"
	raw := &fakeRawLibvirt{
		libVersion:   1002003,
		capabilities: "<capabilities><guest><arch name=\"x86_64\"><domain type=\"kvm\"/></arch></guest></capabilities>",
		domain:       golibvirt.Domain{Name: name},
		network:      golibvirt.Network{Name: name},
		state:        1,
		reason:       2,
	}
	control, err := newScopedRPCControlPlaneForTest(t, raw)
	if err != nil {
		t.Fatalf("newRPCControlPlane() error = %v", err)
	}
	consoleRaw := &fakeRawLibvirt{
		domain:     golibvirt.Domain{Name: name},
		definedXML: managedTestDomainXML,
	}
	control.consoleDial = func() (rawLibvirt, error) {
		return consoleRaw, nil
	}

	if version, err := control.LibVersion(); err != nil || version != raw.libVersion {
		t.Fatalf("LibVersion() = %d, %v", version, err)
	}
	if err := validateSystemCapabilities(raw.capabilities); err != nil {
		t.Fatalf("validateSystemCapabilities() error = %v", err)
	}
	if err := control.DefineDomain(name, managedTestDomainXML); err != nil {
		t.Fatalf("DefineDomain() error = %v", err)
	}
	if raw.definedXML != managedTestDomainXML || raw.defineFlags != 0 {
		t.Fatalf("define call = %q flags=%d", raw.definedXML, raw.defineFlags)
	}
	if err := control.StartDomain(name); err != nil {
		t.Fatalf("StartDomain() error = %v", err)
	}
	if err := control.RebootDomain(name); err != nil {
		t.Fatalf("RebootDomain() error = %v", err)
	}
	if raw.rebootFlags != golibvirt.DomainRebootDefault {
		t.Fatalf("reboot flags = %v, want default", raw.rebootFlags)
	}
	var console bytes.Buffer
	if err := control.OpenConsole(context.Background(), name, strings.NewReader("boot\n"), &console); err != nil {
		t.Fatalf("OpenConsole() error = %v", err)
	}
	if console.String() != "boot\n" {
		t.Fatalf("console output = %q", console.String())
	}
	if consoleRaw.consoleDomain.Name != name || consoleRaw.consoleFlags != 0 || len(consoleRaw.consoleDevice) != 0 {
		t.Fatalf(
			"console call domain=%q device=%v flags=%d",
			consoleRaw.consoleDomain.Name,
			consoleRaw.consoleDevice,
			consoleRaw.consoleFlags,
		)
	}
	if !consoleRaw.disconnected {
		t.Fatal("dedicated console connection was not closed after console completion")
	}
	if raw.disconnected {
		t.Fatal("normal console completion disconnected the main libvirt connection")
	}
	agentResult, err := control.AgentCommand(name, `{"execute":"guest-ping"}`, 5)
	if err != nil {
		t.Fatalf("AgentCommand() error = %v", err)
	}
	if agentResult != `{"return":{"ok":true}}` ||
		raw.agentCommand != `{"execute":"guest-ping"}` ||
		raw.agentTimeout != 5 {
		t.Fatalf(
			"agent result=%q command=%q timeout=%d",
			agentResult,
			raw.agentCommand,
			raw.agentTimeout,
		)
	}
	state, err := control.DomainState(name)
	if err != nil || state.State != 1 || state.Reason != 2 {
		t.Fatalf("DomainState() = %#v, %v", state, err)
	}
	if err := control.DefineNetwork(name, managedTestNetworkXML); err != nil {
		t.Fatalf("DefineNetwork() error = %v", err)
	}
	if raw.networkXML != managedTestNetworkXML {
		t.Fatalf("network XML = %q", raw.networkXML)
	}
	if err := control.StartNetwork(name); err != nil {
		t.Fatalf("StartNetwork() error = %v", err)
	}
	if active, err := control.NetworkActive(name); err != nil || !active {
		t.Fatalf("NetworkActive() = %v, %v", active, err)
	}
	if err := control.DestroyNetwork(name); err != nil {
		t.Fatalf("DestroyNetwork() error = %v", err)
	}
	if err := control.UndefineNetwork(name); err != nil {
		t.Fatalf("UndefineNetwork() error = %v", err)
	}
	domains, err := control.ListManagedDomains()
	if err != nil || len(domains) != 1 || domains[0] != name {
		t.Fatalf("ListManagedDomains() = %#v, %v", domains, err)
	}
	networks, err := control.ListManagedNetworks()
	if err != nil || len(networks) != 1 || networks[0] != name {
		t.Fatalf("ListManagedNetworks() = %#v, %v", networks, err)
	}
	if !raw.networkDestroyed || !raw.networkUndefined {
		t.Fatalf("network cleanup flags destroyed=%v undefined=%v", raw.networkDestroyed, raw.networkUndefined)
	}

	if err := control.DestroyDomain(name); err != nil {
		t.Fatalf("DestroyDomain() error = %v", err)
	}
	if err := control.UndefineDomain(name, true); err != nil {
		t.Fatalf("UndefineDomain() error = %v", err)
	}
	if raw.undefineFlags&golibvirt.DomainUndefineNvram == 0 {
		t.Fatalf("undefine flags = %v, want NVRAM cleanup", raw.undefineFlags)
	}
	if err := control.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if !raw.disconnected {
		t.Fatal("raw libvirt connection was not disconnected")
	}
}

func TestRPCControlPlaneRejectsUnmanagedNamesBeforeRPC(t *testing.T) {
	raw := &fakeRawLibvirt{}
	control, err := newScopedRPCControlPlaneForTest(t, raw)
	if err != nil {
		t.Fatalf("newRPCControlPlane() error = %v", err)
	}
	consoleRaw := &fakeRawLibvirt{}
	control.consoleDial = func() (rawLibvirt, error) {
		return consoleRaw, nil
	}

	for _, action := range []func() error{
		func() error { return control.StartDomain("default") },
		func() error { return control.RebootDomain("default") },
		func() error { return control.DefineNetwork("default", "<network/>") },
		func() error { return control.StartNetwork("default") },
		func() error {
			_, err := control.DomainState("other-vm")
			return err
		},
		func() error { return control.DestroyDomain("qemu-test") },
		func() error { return control.UndefineDomain("../../vm", false) },
		func() error { return control.DefineDomain("foreign", "<domain/>") },
		func() error {
			return control.OpenConsole(
				context.Background(),
				"foreign-console",
				strings.NewReader(""),
				&bytes.Buffer{},
			)
		},
		func() error {
			_, err := control.AgentCommand("foreign-agent", `{"execute":"guest-ping"}`, 5)
			return err
		},
	} {
		if err := action(); err == nil || !strings.Contains(err.Error(), "not LPIC Daily-managed") {
			t.Fatalf("unmanaged action error = %v", err)
		}
	}
}

func TestDefineDomainRejectsMismatchedLibvirtIdentity(t *testing.T) {
	raw := &fakeRawLibvirt{
		domain: golibvirt.Domain{Name: "lpic-daily-other"},
	}
	control, _ := newScopedRPCControlPlaneForTest(t, raw)
	err := control.DefineDomain("lpic-daily-expected", managedTestDomainXML)
	if err == nil || !strings.Contains(err.Error(), "unexpected domain") {
		t.Fatalf("DefineDomain() error = %v", err)
	}
}

func TestRPCControlPlaneRejectsMissingOwnershipMetadata(t *testing.T) {
	raw := &fakeRawLibvirt{
		domain:  golibvirt.Domain{Name: "lpic-daily-owned-test"},
		network: golibvirt.Network{Name: "lpic-daily-owned-test"},
	}
	control, _ := newScopedRPCControlPlaneForTest(t, raw)
	if err := control.DefineDomain("lpic-daily-owned-test", "<domain/>"); err == nil ||
		!strings.Contains(err.Error(), "ownership metadata") {
		t.Fatalf("DefineDomain() metadata error = %v", err)
	}
	if err := control.DefineNetwork("lpic-daily-owned-test", "<network/>"); err == nil ||
		!strings.Contains(err.Error(), "ownership metadata") {
		t.Fatalf("DefineNetwork() metadata error = %v", err)
	}
}

func TestRPCControlPlaneInventoryIgnoresPrefixedForeignResources(t *testing.T) {
	name := "lpic-daily-foreign-abc123"
	raw := &fakeRawLibvirt{
		domain:     golibvirt.Domain{Name: name},
		network:    golibvirt.Network{Name: name},
		definedXML: "<domain/>",
		networkXML: "<network/>",
	}
	control, _ := newScopedRPCControlPlaneForTest(t, raw)
	if domains, err := control.ListManagedDomains(); err != nil || len(domains) != 0 {
		t.Fatalf("foreign domains = %#v, %v", domains, err)
	}
	if networks, err := control.ListManagedNetworks(); err != nil || len(networks) != 0 {
		t.Fatalf("foreign networks = %#v, %v", networks, err)
	}
}

func TestRPCControlPlaneRejectsForeignScopedResources(t *testing.T) {
	name := "lpic-daily-foreign-scope-abc123"
	foreignScope := "fedcba9876543210fedcba9876543210"
	raw := &fakeRawLibvirt{
		domain:      golibvirt.Domain{Name: name},
		network:     golibvirt.Network{Name: name},
		definedXML:  strings.Replace(managedTestDomainXML, testManagedOwnerScope, foreignScope, 1),
		networkXML:  strings.Replace(managedTestNetworkXML, testManagedOwnerScope, foreignScope, 1),
	}
	control, err := newScopedRPCControlPlaneForTest(t, raw)
	if err != nil {
		t.Fatalf("new control plane: %v", err)
	}

	if err := control.StartDomain(name); err == nil || !strings.Contains(err.Error(), "not owned by this LPIC Daily instance") {
		t.Fatalf("StartDomain(foreign scope) error = %v", err)
	}
	if err := control.StartNetwork(name); err == nil || !strings.Contains(err.Error(), "not owned by this LPIC Daily instance") {
		t.Fatalf("StartNetwork(foreign scope) error = %v", err)
	}
	if domains, err := control.ListManagedDomains(); err != nil || len(domains) != 0 {
		t.Fatalf("foreign scoped domains = %#v, %v", domains, err)
	}
	if networks, err := control.ListManagedNetworks(); err != nil || len(networks) != 0 {
		t.Fatalf("foreign scoped networks = %#v, %v", networks, err)
	}
}

func TestRPCControlPlaneRequiresOwnerScopeForManagedOperations(t *testing.T) {
	raw := &fakeRawLibvirt{domain: golibvirt.Domain{Name: "lpic-daily-test-abc123"}}
	control, err := newRPCControlPlane(raw)
	if err != nil {
		t.Fatalf("newRPCControlPlane() error = %v", err)
	}
	if err := control.StartDomain("lpic-daily-test-abc123"); err == nil || !strings.Contains(err.Error(), "owner scope is not configured") {
		t.Fatalf("StartDomain(unscoped) error = %v", err)
	}
}

func TestRPCControlPlaneConsoleCancellationDisconnectsOnlyConsoleStream(t *testing.T) {
	name := "lpic-daily-console-abc123"
	mainRaw := &fakeRawLibvirt{
		domain:     golibvirt.Domain{Name: name},
		definedXML: managedTestDomainXML,
	}
	consoleRaw := &fakeRawLibvirt{
		domain:         golibvirt.Domain{Name: name},
		definedXML:     managedTestDomainXML,
		consoleBlock:   make(chan struct{}),
		consoleStarted: make(chan struct{}),
	}
	control, _ := newScopedRPCControlPlaneForTest(t, mainRaw)
	control.consoleDial = func() (rawLibvirt, error) {
		return consoleRaw, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- control.OpenConsole(ctx, name, strings.NewReader(""), &bytes.Buffer{})
	}()
	select {
	case <-consoleRaw.consoleStarted:
	case <-time.After(time.Second):
		t.Fatal("console did not start")
	}
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("OpenConsole() cancellation error = %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("OpenConsole() did not return after cancellation")
	}
	if !consoleRaw.disconnected || !consoleRaw.consoleClosed {
		t.Fatal("console cancellation did not close the dedicated console connection")
	}
	if mainRaw.disconnected {
		t.Fatal("console cancellation disconnected the main libvirt control connection")
	}
	if mainRaw.destroyFlags != 0 {
		t.Fatalf("console cancellation destroyed the VM: flags=%v", mainRaw.destroyFlags)
	}
	if _, err := control.DomainState(name); err != nil {
		t.Fatalf("main libvirt control connection unusable after console cancellation: %v", err)
	}
}

func TestRPCControlPlanePropagatesRawErrors(t *testing.T) {
	raw := &fakeRawLibvirt{
		domain: golibvirt.Domain{Name: "lpic-daily-vm-abc"},
		err:    errors.New("libvirt unavailable"),
	}
	control, _ := newScopedRPCControlPlaneForTest(t, raw)
	if err := control.StartDomain("lpic-daily-vm-abc"); err == nil ||
		!strings.Contains(err.Error(), "libvirt unavailable") {
		t.Fatalf("StartDomain() error = %v", err)
	}
}

func TestValidateSystemCapabilitiesRequiresX8664KVM(t *testing.T) {
	for _, capabilities := range []string{
		"",
		"<capabilities/>",
		"<capabilities><guest><arch name=\"aarch64\"><domain type=\"kvm\"/></arch></guest></capabilities>",
		"<capabilities><guest><arch name=\"x86_64\"><domain type=\"qemu\"/></arch></guest></capabilities>",
		"<not-closed>",
	} {
		if err := validateSystemCapabilities(capabilities); err == nil {
			t.Fatalf("capabilities %q unexpectedly accepted", capabilities)
		}
	}
	valid := "<capabilities><guest><arch name=\"x86_64\"><domain type=\"qemu\"/><domain type=\"kvm\"/></arch></guest></capabilities>"
	if err := validateSystemCapabilities(valid); err != nil {
		t.Fatalf("x86_64 KVM capabilities rejected: %v", err)
	}
}
