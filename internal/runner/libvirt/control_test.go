package libvirt

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	golibvirt "github.com/digitalocean/go-libvirt"
)

type fakeRawLibvirt struct {
	libVersion    uint64
	capabilities  string
	domain        golibvirt.Domain
	definedXML    string
	defineFlags   golibvirt.DomainDefineFlags
	createFlags   uint32
	rebootFlags   golibvirt.DomainRebootFlagValues
	destroyFlags  golibvirt.DomainDestroyFlagsValues
	undefineFlags golibvirt.DomainUndefineFlagsValues
	state         int32
	reason        int32
	consoleDomain golibvirt.Domain
	consoleDevice golibvirt.OptString
	consoleFlags  uint32
	agentCommand  string
	agentTimeout  int32
	network        golibvirt.Network
	networkXML     string
	networkActive  int32
	networkDestroyed bool
	networkUndefined bool
	disconnected  bool
	err           error
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

func (fake *fakeRawLibvirt) DomainDestroyFlags(
	_ golibvirt.Domain,
	flags golibvirt.DomainDestroyFlagsValues,
) error {
	fake.destroyFlags = flags
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

func (fake *fakeRawLibvirt) Disconnect() error {
	fake.disconnected = true
	return fake.err
}

func TestRPCControlPlaneLifecycleUsesManagedDomainOnly(t *testing.T) {
	name := "lpic-daily-vm-abc123"
	raw := &fakeRawLibvirt{
		libVersion:   1002003,
		capabilities: "<capabilities><host><cpu><arch>x86_64</arch></cpu></host></capabilities>",
		domain:       golibvirt.Domain{Name: name},
		network:      golibvirt.Network{Name: name},
		state:        1,
		reason:       2,
	}
	control, err := newRPCControlPlane(raw)
	if err != nil {
		t.Fatalf("newRPCControlPlane() error = %v", err)
	}

	if version, err := control.LibVersion(); err != nil || version != raw.libVersion {
		t.Fatalf("LibVersion() = %d, %v", version, err)
	}
	if err := validateSystemCapabilities(raw.capabilities); err != nil {
		t.Fatalf("validateSystemCapabilities() error = %v", err)
	}
	if err := control.DefineDomain(name, "<domain/>"); err != nil {
		t.Fatalf("DefineDomain() error = %v", err)
	}
	if raw.definedXML != "<domain/>" || raw.defineFlags != 0 {
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
	if err := control.OpenConsole(name, strings.NewReader("boot\n"), &console); err != nil {
		t.Fatalf("OpenConsole() error = %v", err)
	}
	if console.String() != "boot\n" {
		t.Fatalf("console output = %q", console.String())
	}
	if raw.consoleDomain.Name != name || raw.consoleFlags != 0 || len(raw.consoleDevice) != 0 {
		t.Fatalf(
			"console call domain=%q device=%v flags=%d",
			raw.consoleDomain.Name,
			raw.consoleDevice,
			raw.consoleFlags,
		)
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
	if err := control.DefineNetwork(name, "<network/>"); err != nil {
		t.Fatalf("DefineNetwork() error = %v", err)
	}
	if raw.networkXML != "<network/>" {
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
	control, err := newRPCControlPlane(raw)
	if err != nil {
		t.Fatalf("newRPCControlPlane() error = %v", err)
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
	control, _ := newRPCControlPlane(raw)
	err := control.DefineDomain("lpic-daily-expected", "<domain/>")
	if err == nil || !strings.Contains(err.Error(), "unexpected domain") {
		t.Fatalf("DefineDomain() error = %v", err)
	}
}

func TestRPCControlPlanePropagatesRawErrors(t *testing.T) {
	raw := &fakeRawLibvirt{
		domain: golibvirt.Domain{Name: "lpic-daily-vm-abc"},
		err:    errors.New("libvirt unavailable"),
	}
	control, _ := newRPCControlPlane(raw)
	if err := control.StartDomain("lpic-daily-vm-abc"); err == nil ||
		!strings.Contains(err.Error(), "libvirt unavailable") {
		t.Fatalf("StartDomain() error = %v", err)
	}
}

func TestValidateSystemCapabilitiesRequiresX8664(t *testing.T) {
	for _, capabilities := range []string{"", "<capabilities/>", "<arch>aarch64</arch>"} {
		if err := validateSystemCapabilities(capabilities); err == nil {
			t.Fatalf("capabilities %q unexpectedly accepted", capabilities)
		}
	}
	if err := validateSystemCapabilities("<arch>x86_64</arch>"); err != nil {
		t.Fatalf("x86_64 capabilities rejected: %v", err)
	}
}
