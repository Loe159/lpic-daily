package libvirt

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"

	golibvirt "github.com/digitalocean/go-libvirt"
)

const systemURI = "qemu:///system"

type DomainState struct {
	State  int32
	Reason int32
	Active bool
}

type ControlPlane interface {
	LibVersion() (uint64, error)
	Capabilities() (string, error)
	DefineDomain(string, string) error
	StartDomain(string) error
	DomainState(string) (DomainState, error)
	DestroyDomain(string) error
	UndefineDomain(string, bool) error
	Close() error
}

type ConsoleControlPlane interface {
	OpenConsole(string, io.Reader, io.Writer) error
}

type rawLibvirt interface {
	ConnectGetLibVersion() (uint64, error)
	ConnectGetCapabilities() (string, error)
	DomainDefineXMLFlags(string, golibvirt.DomainDefineFlags) (golibvirt.Domain, error)
	DomainLookupByName(string) (golibvirt.Domain, error)
	DomainCreateWithFlags(golibvirt.Domain, uint32) (golibvirt.Domain, error)
	DomainOpenConsoleBidirectional(golibvirt.Domain, golibvirt.OptString, io.Reader, io.Writer, uint32) error
	DomainGetState(golibvirt.Domain, uint32) (int32, int32, error)
	DomainDestroyFlags(golibvirt.Domain, golibvirt.DomainDestroyFlagsValues) error
	DomainUndefineFlags(golibvirt.Domain, golibvirt.DomainUndefineFlagsValues) error
	Disconnect() error
}

type RPCControlPlane struct {
	raw rawLibvirt
}

func OpenSystem() (*RPCControlPlane, error) {
	uri, err := url.Parse(systemURI)
	if err != nil {
		return nil, fmt.Errorf("parse built-in libvirt URI: %w", err)
	}
	client, err := golibvirt.ConnectToURI(uri)
	if err != nil {
		return nil, fmt.Errorf("connect to local system libvirt: %w", err)
	}

	control := &RPCControlPlane{raw: client}
	if _, err := control.LibVersion(); err != nil {
		_ = control.Close()
		return nil, fmt.Errorf("probe libvirt version: %w", err)
	}
	capabilities, err := control.Capabilities()
	if err != nil {
		_ = control.Close()
		return nil, fmt.Errorf("probe libvirt capabilities: %w", err)
	}
	if err := validateSystemCapabilities(capabilities); err != nil {
		_ = control.Close()
		return nil, err
	}
	return control, nil
}

func newRPCControlPlane(raw rawLibvirt) (*RPCControlPlane, error) {
	if raw == nil {
		return nil, errors.New("raw libvirt client is required")
	}
	return &RPCControlPlane{raw: raw}, nil
}

func (control *RPCControlPlane) LibVersion() (uint64, error) {
	if control == nil || control.raw == nil {
		return 0, errors.New("libvirt control plane is not initialized")
	}
	return control.raw.ConnectGetLibVersion()
}

func (control *RPCControlPlane) Capabilities() (string, error) {
	if control == nil || control.raw == nil {
		return "", errors.New("libvirt control plane is not initialized")
	}
	return control.raw.ConnectGetCapabilities()
}

func (control *RPCControlPlane) DefineDomain(name, domainXML string) error {
	if err := validateManagedResourceName("domain", name); err != nil {
		return err
	}
	if strings.TrimSpace(domainXML) == "" {
		return errors.New("domain XML is required")
	}
	domain, err := control.raw.DomainDefineXMLFlags(domainXML, 0)
	if err != nil {
		return fmt.Errorf("define domain %s: %w", name, err)
	}
	if domain.Name != name {
		return fmt.Errorf("libvirt defined unexpected domain %q, expected %q", domain.Name, name)
	}
	return nil
}

func (control *RPCControlPlane) StartDomain(name string) error {
	domain, err := control.lookupManagedDomain(name)
	if err != nil {
		return err
	}
	started, err := control.raw.DomainCreateWithFlags(domain, 0)
	if err != nil {
		return fmt.Errorf("start domain %s: %w", name, err)
	}
	if started.Name != name {
		return fmt.Errorf("libvirt started unexpected domain %q, expected %q", started.Name, name)
	}
	return nil
}

func (control *RPCControlPlane) OpenConsole(name string, input io.Reader, output io.Writer) error {
	if input == nil {
		return errors.New("console input is required")
	}
	if output == nil {
		return errors.New("console output is required")
	}
	domain, err := control.lookupManagedDomain(name)
	if err != nil {
		return err
	}
	if err := control.raw.DomainOpenConsoleBidirectional(
		domain,
		golibvirt.OptString(nil),
		input,
		output,
		0,
	); err != nil {
		return fmt.Errorf("open console for domain %s: %w", name, err)
	}
	return nil
}

func (control *RPCControlPlane) DomainState(name string) (DomainState, error) {
	domain, err := control.lookupManagedDomain(name)
	if err != nil {
		return DomainState{}, err
	}
	state, reason, err := control.raw.DomainGetState(domain, 0)
	if err != nil {
		return DomainState{}, fmt.Errorf("get domain %s state: %w", name, err)
	}
	return DomainState{
		State:  state,
		Reason: reason,
		Active: state != int32(golibvirt.DomainShutoff) && state != int32(golibvirt.DomainNostate),
	}, nil
}

func (control *RPCControlPlane) DestroyDomain(name string) error {
	domain, err := control.lookupManagedDomain(name)
	if err != nil {
		return err
	}
	if err := control.raw.DomainDestroyFlags(domain, 0); err != nil {
		return fmt.Errorf("destroy domain %s: %w", name, err)
	}
	return nil
}

func (control *RPCControlPlane) UndefineDomain(name string, removeNVRAM bool) error {
	domain, err := control.lookupManagedDomain(name)
	if err != nil {
		return err
	}
	var flags golibvirt.DomainUndefineFlagsValues
	if removeNVRAM {
		flags |= golibvirt.DomainUndefineNvram
	}
	if err := control.raw.DomainUndefineFlags(domain, flags); err != nil {
		return fmt.Errorf("undefine domain %s: %w", name, err)
	}
	return nil
}

func (control *RPCControlPlane) Close() error {
	if control == nil || control.raw == nil {
		return nil
	}
	if err := control.raw.Disconnect(); err != nil {
		return fmt.Errorf("disconnect libvirt: %w", err)
	}
	control.raw = nil
	return nil
}

func (control *RPCControlPlane) lookupManagedDomain(name string) (golibvirt.Domain, error) {
	if err := validateManagedResourceName("domain", name); err != nil {
		return golibvirt.Domain{}, err
	}
	domain, err := control.raw.DomainLookupByName(name)
	if err != nil {
		return golibvirt.Domain{}, fmt.Errorf("lookup domain %s: %w", name, err)
	}
	if domain.Name != name {
		return golibvirt.Domain{}, fmt.Errorf("libvirt returned unexpected domain %q for %q", domain.Name, name)
	}
	return domain, nil
}

func validateManagedResourceName(kind, name string) error {
	if !managedNamePattern.MatchString(name) {
		return fmt.Errorf("%s name %q is not LPIC Daily-managed", kind, name)
	}
	return nil
}

func validateSystemCapabilities(capabilities string) error {
	capabilities = strings.TrimSpace(capabilities)
	if capabilities == "" {
		return errors.New("libvirt returned empty capabilities")
	}
	if !strings.Contains(capabilities, "<arch>x86_64</arch>") &&
		!strings.Contains(capabilities, "<arch name='x86_64'>") &&
		!strings.Contains(capabilities, `<arch name="x86_64">`) {
		return errors.New("libvirt host does not advertise x86_64 capabilities")
	}
	return nil
}
