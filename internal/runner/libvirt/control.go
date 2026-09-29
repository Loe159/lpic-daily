package libvirt

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

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
	RebootDomain(string) error
	DomainState(string) (DomainState, error)
	DestroyDomain(string) error
	UndefineDomain(string, bool) error
	Close() error
}

type NetworkControlPlane interface {
	DefineNetwork(string, string) error
	StartNetwork(string) error
	NetworkActive(string) (bool, error)
	DestroyNetwork(string) error
	UndefineNetwork(string) error
}

type ResourceInventory interface {
	ListManagedDomains() ([]string, error)
	ListManagedNetworks() ([]string, error)
}

type ConsoleControlPlane interface {
	OpenConsole(context.Context, string, io.Reader, io.Writer) error
}

type AgentControlPlane interface {
	AgentCommand(string, string, int32) (string, error)
}

type rawLibvirt interface {
	ConnectGetLibVersion() (uint64, error)
	ConnectGetCapabilities() (string, error)
	DomainDefineXMLFlags(string, golibvirt.DomainDefineFlags) (golibvirt.Domain, error)
	DomainLookupByName(string) (golibvirt.Domain, error)
	DomainCreateWithFlags(golibvirt.Domain, uint32) (golibvirt.Domain, error)
	DomainReboot(golibvirt.Domain, golibvirt.DomainRebootFlagValues) error
	DomainOpenConsoleBidirectional(golibvirt.Domain, golibvirt.OptString, io.Reader, io.Writer, uint32) error
	QEMUDomainAgentCommand(golibvirt.Domain, string, int32, uint32) (golibvirt.OptString, error)
	DomainGetState(golibvirt.Domain, uint32) (int32, int32, error)
	DomainGetXMLDesc(golibvirt.Domain, golibvirt.DomainXMLFlags) (string, error)
	DomainDestroyFlags(golibvirt.Domain, golibvirt.DomainDestroyFlagsValues) error
	DomainUndefineFlags(golibvirt.Domain, golibvirt.DomainUndefineFlagsValues) error
	NetworkDefineXML(string) (golibvirt.Network, error)
	NetworkLookupByName(string) (golibvirt.Network, error)
	NetworkCreate(golibvirt.Network) error
	NetworkIsActive(golibvirt.Network) (int32, error)
	NetworkGetXMLDesc(golibvirt.Network, uint32) (string, error)
	NetworkDestroy(golibvirt.Network) error
	NetworkUndefine(golibvirt.Network) error
	ConnectListAllDomains(int32, golibvirt.ConnectListAllDomainsFlags) ([]golibvirt.Domain, uint32, error)
	ConnectListAllNetworks(int32, golibvirt.ConnectListAllNetworksFlags) ([]golibvirt.Network, uint32, error)
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
	if !hasManagedMetadata(domainXML) {
		return errors.New("domain XML is missing LPIC Daily ownership metadata")
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

func (control *RPCControlPlane) RebootDomain(name string) error {
	domain, err := control.lookupManagedDomain(name)
	if err != nil {
		return err
	}
	if err := control.raw.DomainReboot(domain, golibvirt.DomainRebootDefault); err != nil {
		return fmt.Errorf("reboot domain %s: %w", name, err)
	}
	return nil
}

func (control *RPCControlPlane) OpenConsole(
	ctx context.Context,
	name string,
	input io.Reader,
	output io.Writer,
) error {
	if input == nil {
		return errors.New("console input is required")
	}
	if output == nil {
		return errors.New("console output is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	domain, err := control.lookupManagedDomain(name)
	if err != nil {
		return err
	}

	done := make(chan error, 1)
	go func() {
		done <- control.raw.DomainOpenConsoleBidirectional(
			domain,
			golibvirt.OptString(nil),
			input,
			output,
			0,
		)
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("open console for domain %s: %w", name, err)
		}
		return nil
	case <-ctx.Done():
		// The go-libvirt bidirectional helper has no context parameter. Powering
		// off only this disposable lab domain closes the stream while keeping the
		// control connection usable for normal teardown.
		destroyErr := control.raw.DomainDestroyFlags(domain, 0)
		timer := time.NewTimer(2 * time.Second)
		defer timer.Stop()
		select {
		case <-done:
		case <-timer.C:
		}
		if destroyErr != nil {
			return errors.Join(ctx.Err(), fmt.Errorf("interrupt console for domain %s: %w", name, destroyErr))
		}
		return ctx.Err()
	}
}

func (control *RPCControlPlane) AgentCommand(name, command string, timeoutSeconds int32) (string, error) {
	if strings.TrimSpace(command) == "" {
		return "", errors.New("guest-agent command is required")
	}
	if timeoutSeconds <= 0 || timeoutSeconds > 60 {
		return "", errors.New("guest-agent timeout must be between 1 and 60 seconds")
	}
	domain, err := control.lookupManagedDomain(name)
	if err != nil {
		return "", err
	}
	result, err := control.raw.QEMUDomainAgentCommand(domain, command, timeoutSeconds, 0)
	if err != nil {
		return "", fmt.Errorf("guest-agent command for domain %s: %w", name, err)
	}
	if len(result) != 1 {
		return "", fmt.Errorf("guest-agent command for domain %s returned %d results", name, len(result))
	}
	return result[0], nil
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

func (control *RPCControlPlane) DefineNetwork(name, networkXML string) error {
	if err := validateManagedResourceName("network", name); err != nil {
		return err
	}
	if strings.TrimSpace(networkXML) == "" {
		return errors.New("network XML is required")
	}
	if !hasManagedMetadata(networkXML) {
		return errors.New("network XML is missing LPIC Daily ownership metadata")
	}
	network, err := control.raw.NetworkDefineXML(networkXML)
	if err != nil {
		return fmt.Errorf("define network %s: %w", name, err)
	}
	if network.Name != name {
		return fmt.Errorf("libvirt defined unexpected network %q, expected %q", network.Name, name)
	}
	return nil
}

func (control *RPCControlPlane) StartNetwork(name string) error {
	network, err := control.lookupManagedNetwork(name)
	if err != nil {
		return err
	}
	if err := control.raw.NetworkCreate(network); err != nil {
		return fmt.Errorf("start network %s: %w", name, err)
	}
	return nil
}

func (control *RPCControlPlane) NetworkActive(name string) (bool, error) {
	network, err := control.lookupManagedNetwork(name)
	if err != nil {
		return false, err
	}
	active, err := control.raw.NetworkIsActive(network)
	if err != nil {
		return false, fmt.Errorf("get network %s active state: %w", name, err)
	}
	return active != 0, nil
}

func (control *RPCControlPlane) DestroyNetwork(name string) error {
	network, err := control.lookupManagedNetwork(name)
	if err != nil {
		return err
	}
	if err := control.raw.NetworkDestroy(network); err != nil {
		return fmt.Errorf("destroy network %s: %w", name, err)
	}
	return nil
}

func (control *RPCControlPlane) UndefineNetwork(name string) error {
	network, err := control.lookupManagedNetwork(name)
	if err != nil {
		return err
	}
	if err := control.raw.NetworkUndefine(network); err != nil {
		return fmt.Errorf("undefine network %s: %w", name, err)
	}
	return nil
}

func (control *RPCControlPlane) ListManagedDomains() ([]string, error) {
	domains, _, err := control.raw.ConnectListAllDomains(
		1,
		golibvirt.ConnectListDomainsActive|golibvirt.ConnectListDomainsInactive,
	)
	if err != nil {
		return nil, fmt.Errorf("list libvirt domains: %w", err)
	}
	names := make([]string, 0, len(domains))
	for _, domain := range domains {
		if !managedNamePattern.MatchString(domain.Name) {
			continue
		}
		resourceXML, err := control.raw.DomainGetXMLDesc(domain, 0)
		if err != nil {
			return nil, fmt.Errorf("inspect domain %s ownership metadata: %w", domain.Name, err)
		}
		if hasManagedMetadata(resourceXML) {
			names = append(names, domain.Name)
		}
	}
	return names, nil
}

func (control *RPCControlPlane) ListManagedNetworks() ([]string, error) {
	networks, _, err := control.raw.ConnectListAllNetworks(
		1,
		golibvirt.ConnectListNetworksActive|golibvirt.ConnectListNetworksInactive,
	)
	if err != nil {
		return nil, fmt.Errorf("list libvirt networks: %w", err)
	}
	names := make([]string, 0, len(networks))
	for _, network := range networks {
		if !managedNamePattern.MatchString(network.Name) {
			continue
		}
		resourceXML, err := control.raw.NetworkGetXMLDesc(network, 0)
		if err != nil {
			return nil, fmt.Errorf("inspect network %s ownership metadata: %w", network.Name, err)
		}
		if hasManagedMetadata(resourceXML) {
			names = append(names, network.Name)
		}
	}
	return names, nil
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

func (control *RPCControlPlane) lookupManagedNetwork(name string) (golibvirt.Network, error) {
	if err := validateManagedResourceName("network", name); err != nil {
		return golibvirt.Network{}, err
	}
	network, err := control.raw.NetworkLookupByName(name)
	if err != nil {
		return golibvirt.Network{}, fmt.Errorf("lookup network %s: %w", name, err)
	}
	if network.Name != name {
		return golibvirt.Network{}, fmt.Errorf("libvirt returned unexpected network %q for %q", network.Name, name)
	}
	return network, nil
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
