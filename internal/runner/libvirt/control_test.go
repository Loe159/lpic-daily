package libvirt

import (
	"errors"
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
	destroyFlags  golibvirt.DomainDestroyFlagsValues
	undefineFlags golibvirt.DomainUndefineFlagsValues
	state         int32
	reason        int32
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

func (fake *fakeRawLibvirt) Disconnect() error {
	fake.disconnected = true
	return fake.err
}

func TestRPCControlPlaneLifecycleUsesManagedDomainOnly(t *testing.T) {
	name := "lpic-daily-vm-abc123"
	raw := &fakeRawLibvirt{
		libVersion:   1002003,
		capabilities: "<capabilities><host><cpu><arch>x86_64</arch></cpu></host></capabilities>",
		domain:        golibvirt.Domain{Name: name},
		state:         1,
		reason:        2,
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
	state, err := control.DomainState(name)
	if err != nil || state.State != 1 || state.Reason != 2 {
		t.Fatalf("DomainState() = %#v, %v", state, err)
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
		func() error {
			_, err := control.DomainState("other-vm")
			return err
		},
		func() error { return control.DestroyDomain("qemu-test") },
		func() error { return control.UndefineDomain("../../vm", false) },
		func() error { return control.DefineDomain("foreign", "<domain/>") },
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
