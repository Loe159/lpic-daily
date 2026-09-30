package libvirt

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Loe159/lpic-daily/internal/runner"
)

var (
	managedNamePattern  = regexp.MustCompile(`^lpic-daily-[a-z0-9][a-z0-9.-]{0,52}$`)
	imageIDPattern      = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{2,63}$`)
	diskIDPattern       = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,31}$`)
	managedScopePattern = regexp.MustCompile(`^[a-f0-9]{32}$`)
)

const (
	managedMetadataNamespace = "urn:lpic-daily:managed:v1"
	managedMetadataOwner     = "lpic-daily"
	managedMetadataVersion   = "1"
)

type ImageDescriptor struct {
	ID            string
	Path          string
	SHA256        string
	Format        string
	Architecture  string
	Distribution  string
	VirtualSizeMB int
	FirmwareModes []runner.FirmwareMode
}

func (image ImageDescriptor) Validate(imageRoot string) error {
	if !imageIDPattern.MatchString(image.ID) {
		return fmt.Errorf("invalid trusted image ID %q", image.ID)
	}
	if image.Format != "qcow2" {
		return fmt.Errorf("unsupported image format %q", image.Format)
	}
	if image.Architecture != "x86_64" {
		return fmt.Errorf("unsupported image architecture %q", image.Architecture)
	}
	if len(image.SHA256) != 64 {
		return errors.New("image SHA-256 must contain 64 hex characters")
	}
	for _, char := range image.SHA256 {
		if !strings.ContainsRune("0123456789abcdef", char) {
			return errors.New("image SHA-256 must be lowercase hexadecimal")
		}
	}
	if image.VirtualSizeMB < 1024 || image.VirtualSizeMB > 32768 {
		return errors.New("image virtual size must be between 1024 and 32768 MiB")
	}
	if err := pathWithinRoot(imageRoot, image.Path); err != nil {
		return fmt.Errorf("image path: %w", err)
	}
	if len(image.FirmwareModes) == 0 {
		return errors.New("image must declare at least one firmware mode")
	}
	seen := map[runner.FirmwareMode]struct{}{}
	for _, mode := range image.FirmwareModes {
		if mode != runner.FirmwareBIOS && mode != runner.FirmwareUEFI {
			return fmt.Errorf("unsupported image firmware %q", mode)
		}
		if _, exists := seen[mode]; exists {
			return fmt.Errorf("duplicate image firmware mode %q", mode)
		}
		seen[mode] = struct{}{}
	}
	return nil
}

func (image ImageDescriptor) SupportsFirmware(mode runner.FirmwareMode) bool {
	for _, supported := range image.FirmwareModes {
		if supported == mode {
			return true
		}
	}
	return false
}

type DiskPath struct {
	ID   string
	Path string
}

type DomainSpec struct {
	Name              string
	LabID             string
	OwnerScope        string
	MemoryMB          int
	CPUPercent        int
	Firmware          runner.FirmwareMode
	RootDiskPath      string
	ExtraDisks        []DiskPath
	NetworkName       string
	NetworkFilterName string
}

func (spec DomainSpec) Validate(stateRoot string) error {
	if err := validateManagedOwnerScope(spec.OwnerScope); err != nil {
		return err
	}
	if !managedNamePattern.MatchString(spec.Name) {
		return fmt.Errorf("invalid managed domain name %q", spec.Name)
	}
	if strings.TrimSpace(spec.LabID) == "" {
		return errors.New("lab ID is required")
	}
	if spec.MemoryMB < 256 || spec.MemoryMB > 16384 {
		return errors.New("VM memory must be between 256 and 16384 MiB")
	}
	if spec.CPUPercent < 10 || spec.CPUPercent > 400 {
		return errors.New("VM CPU percent must be between 10 and 400")
	}
	if spec.Firmware != runner.FirmwareBIOS && spec.Firmware != runner.FirmwareUEFI {
		return errors.New("VM firmware must be bios or uefi")
	}
	if err := pathWithinRoot(stateRoot, spec.RootDiskPath); err != nil {
		return fmt.Errorf("root disk: %w", err)
	}
	if len(spec.ExtraDisks) > 4 {
		return errors.New("VM supports at most 4 extra disks")
	}
	seen := map[string]struct{}{}
	for _, disk := range spec.ExtraDisks {
		if !diskIDPattern.MatchString(disk.ID) {
			return fmt.Errorf("invalid disk ID %q", disk.ID)
		}
		if _, exists := seen[disk.ID]; exists {
			return fmt.Errorf("duplicate disk ID %q", disk.ID)
		}
		seen[disk.ID] = struct{}{}
		if err := pathWithinRoot(stateRoot, disk.Path); err != nil {
			return fmt.Errorf("disk %s: %w", disk.ID, err)
		}
	}
	if spec.NetworkName != "" && !managedNamePattern.MatchString(spec.NetworkName) {
		return fmt.Errorf("invalid managed network name %q", spec.NetworkName)
	}
	if spec.NetworkName == "" {
		if spec.NetworkFilterName != "" {
			return errors.New("network filter requires an isolated network")
		}
	} else {
		if !managedNamePattern.MatchString(spec.NetworkFilterName) {
			return fmt.Errorf("invalid managed network filter name %q", spec.NetworkFilterName)
		}
	}
	return nil
}

func pathWithinRoot(root, candidate string) error {
	if root == "" || !filepath.IsAbs(root) {
		return errors.New("trusted root must be absolute")
	}
	if candidate == "" || !filepath.IsAbs(candidate) {
		return errors.New("path must be absolute")
	}
	cleanRoot := filepath.Clean(root)
	cleanCandidate := filepath.Clean(candidate)
	rel, err := filepath.Rel(cleanRoot, cleanCandidate)
	if err != nil {
		return err
	}
	if rel == "." || rel == "" {
		return errors.New("path must name a file below the trusted root")
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errors.New("path escapes trusted root")
	}

	current := cleanRoot
	info, statErr := os.Lstat(current)
	if errors.Is(statErr, os.ErrNotExist) {
		return nil
	}
	if statErr != nil {
		return fmt.Errorf("inspect trusted path %s: %w", current, statErr)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("path contains symbolic link: %s", current)
	}
	for _, component := range strings.Split(rel, string(filepath.Separator)) {
		current = filepath.Join(current, component)
		info, statErr = os.Lstat(current)
		if errors.Is(statErr, os.ErrNotExist) {
			return nil
		}
		if statErr != nil {
			return fmt.Errorf("inspect trusted path %s: %w", current, statErr)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("path contains symbolic link: %s", current)
		}
	}
	return nil
}

type managedMetadataXML struct {
	XMLName xml.Name `xml:"urn:lpic-daily:managed:v1 lpic-daily"`
	Owner   string   `xml:"owner,attr"`
	Version string   `xml:"version,attr"`
	Scope   string   `xml:"scope,attr"`
}

type metadataXML struct {
	Managed managedMetadataXML `xml:"urn:lpic-daily:managed:v1 lpic-daily"`
}

func managedOwnerScope(stateRoot string) (string, error) {
	return managedOwnerScopeForUID(stateRoot, os.Geteuid())
}

func managedOwnerScopeForUID(stateRoot string, uid int) (string, error) {
	if uid < 0 {
		return "", errors.New("effective UID must not be negative")
	}
	if stateRoot == "" || !filepath.IsAbs(stateRoot) {
		return "", errors.New("VM state root must be absolute")
	}
	cleanRoot := filepath.Clean(stateRoot)
	digest := sha256.Sum256([]byte(fmt.Sprintf("uid=%d\x00state-root=%s", uid, cleanRoot)))
	return hex.EncodeToString(digest[:16]), nil
}

func validateManagedOwnerScope(scope string) error {
	if !managedScopePattern.MatchString(scope) {
		return fmt.Errorf("invalid LPIC Daily owner scope %q", scope)
	}
	return nil
}

func newManagedMetadata(scope string) metadataXML {
	return metadataXML{Managed: managedMetadataXML{
		Owner: managedMetadataOwner, Version: managedMetadataVersion, Scope: scope,
	}}
}

func hasManagedMetadata(payload, scope string) bool {
	if validateManagedOwnerScope(scope) != nil {
		return false
	}
	var document struct {
		Metadata metadataXML `xml:"metadata"`
	}
	if err := xml.Unmarshal([]byte(payload), &document); err != nil {
		return false
	}
	return document.Metadata.Managed.Owner == managedMetadataOwner &&
		document.Metadata.Managed.Version == managedMetadataVersion &&
		document.Metadata.Managed.Scope == scope
}

type domainXML struct {
	XMLName     xml.Name    `xml:"domain"`
	Type        string      `xml:"type,attr"`
	Name        string      `xml:"name"`
	Description string      `xml:"description"`
	Metadata    metadataXML `xml:"metadata"`
	Memory      memoryXML   `xml:"memory"`
	VCPU        vcpuXML     `xml:"vcpu"`
	CPUTune     cpuTuneXML  `xml:"cputune"`
	OS          osXML       `xml:"os"`
	Features    featuresXML `xml:"features"`
	Devices     devicesXML  `xml:"devices"`
	OnPoweroff  string      `xml:"on_poweroff"`
	OnReboot    string      `xml:"on_reboot"`
	OnCrash     string      `xml:"on_crash"`
}

type memoryXML struct {
	Unit  string `xml:"unit,attr"`
	Value int    `xml:",chardata"`
}

type vcpuXML struct {
	Placement string `xml:"placement,attr"`
	Value     int    `xml:",chardata"`
}

type cpuTuneXML struct {
	GlobalPeriod int64 `xml:"global_period"`
	GlobalQuota  int64 `xml:"global_quota"`
}

type osXML struct {
	Firmware string    `xml:"firmware,attr,omitempty"`
	Type     osTypeXML `xml:"type"`
}

type osTypeXML struct {
	Arch    string `xml:"arch,attr"`
	Machine string `xml:"machine,attr"`
	Value   string `xml:",chardata"`
}

type featuresXML struct {
	ACPI struct{} `xml:"acpi"`
}

type devicesXML struct {
	Disks      []diskXML      `xml:"disk"`
	Interfaces []interfaceXML `xml:"interface,omitempty"`
	Channels   []channelXML   `xml:"channel,omitempty"`
	Serial     serialXML      `xml:"serial"`
	Console    consoleXML     `xml:"console"`
}

type diskXML struct {
	Type   string        `xml:"type,attr"`
	Device string        `xml:"device,attr"`
	Driver diskDriverXML `xml:"driver"`
	Source diskSourceXML `xml:"source"`
	Target diskTargetXML `xml:"target"`
}

type diskDriverXML struct {
	Name string `xml:"name,attr"`
	Type string `xml:"type,attr"`
}

type diskSourceXML struct {
	File string `xml:"file,attr"`
}

type diskTargetXML struct {
	Dev string `xml:"dev,attr"`
	Bus string `xml:"bus,attr"`
}

type interfaceXML struct {
	Type      string             `xml:"type,attr"`
	Source    interfaceSourceXML `xml:"source"`
	Model     interfaceModelXML  `xml:"model"`
	FilterRef filterRefXML       `xml:"filterref"`
}

type filterRefXML struct {
	Filter string `xml:"filter,attr"`
}

type interfaceSourceXML struct {
	Network string `xml:"network,attr"`
}

type interfaceModelXML struct {
	Type string `xml:"type,attr"`
}

type channelXML struct {
	Type   string           `xml:"type,attr"`
	Target channelTargetXML `xml:"target"`
}

type channelTargetXML struct {
	Type string `xml:"type,attr"`
	Name string `xml:"name,attr"`
}

type serialXML struct {
	Type   string          `xml:"type,attr"`
	Target serialTargetXML `xml:"target"`
}

type serialTargetXML struct {
	Type string `xml:"type,attr"`
	Port int    `xml:"port,attr"`
}

type consoleXML struct {
	Type   string           `xml:"type,attr"`
	Target consoleTargetXML `xml:"target"`
}

type consoleTargetXML struct {
	Type string `xml:"type,attr"`
	Port int    `xml:"port,attr"`
}

func BuildDomainXML(spec DomainSpec, stateRoot string) (string, error) {
	if err := spec.Validate(stateRoot); err != nil {
		return "", err
	}

	vcpuCount := (spec.CPUPercent + 99) / 100
	if vcpuCount < 1 {
		vcpuCount = 1
	}
	if vcpuCount > 4 {
		vcpuCount = 4
	}
	const period = int64(100000)
	quota := period * int64(spec.CPUPercent) / 100

	doc := domainXML{
		Type:        "kvm",
		Name:        spec.Name,
		Description: "LPIC Daily lab " + spec.LabID,
		Metadata:    newManagedMetadata(spec.OwnerScope),
		Memory:      memoryXML{Unit: "MiB", Value: spec.MemoryMB},
		VCPU:        vcpuXML{Placement: "static", Value: vcpuCount},
		CPUTune:     cpuTuneXML{GlobalPeriod: period, GlobalQuota: quota},
		OS: osXML{
			Type: osTypeXML{Arch: "x86_64", Machine: "q35", Value: "hvm"},
		},
		OnPoweroff: "destroy",
		OnReboot:   "restart",
		OnCrash:    "destroy",
	}
	if spec.Firmware == runner.FirmwareUEFI {
		doc.OS.Firmware = "efi"
	}

	doc.Devices.Disks = append(doc.Devices.Disks, diskXML{
		Type:   "file",
		Device: "disk",
		Driver: diskDriverXML{Name: "qemu", Type: "qcow2"},
		Source: diskSourceXML{File: filepath.Clean(spec.RootDiskPath)},
		Target: diskTargetXML{Dev: "vda", Bus: "virtio"},
	})
	for index, disk := range spec.ExtraDisks {
		doc.Devices.Disks = append(doc.Devices.Disks, diskXML{
			Type:   "file",
			Device: "disk",
			Driver: diskDriverXML{Name: "qemu", Type: "qcow2"},
			Source: diskSourceXML{File: filepath.Clean(disk.Path)},
			Target: diskTargetXML{Dev: "vd" + string(rune('b'+index)), Bus: "virtio"},
		})
	}
	if spec.NetworkName != "" {
		doc.Devices.Interfaces = []interfaceXML{{
			Type:      "network",
			Source:    interfaceSourceXML{Network: spec.NetworkName},
			Model:     interfaceModelXML{Type: "virtio"},
			FilterRef: filterRefXML{Filter: spec.NetworkFilterName},
		}}
	}
	doc.Devices.Channels = []channelXML{{
		Type: "unix",
		Target: channelTargetXML{
			Type: "virtio",
			Name: "org.qemu.guest_agent.0",
		},
	}}
	doc.Devices.Serial = serialXML{Type: "pty", Target: serialTargetXML{Type: "isa-serial", Port: 0}}
	doc.Devices.Console = consoleXML{Type: "pty", Target: consoleTargetXML{Type: "serial", Port: 0}}

	payload, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal domain XML: %w", err)
	}
	return xml.Header + string(payload) + "\n", nil
}

type networkXML struct {
	XMLName  xml.Name    `xml:"network"`
	Name     string      `xml:"name"`
	Metadata metadataXML `xml:"metadata"`
	IP       networkIP   `xml:"ip"`
}

type networkIP struct {
	Address string  `xml:"address,attr"`
	Netmask string  `xml:"netmask,attr"`
	DHCP    dhcpXML `xml:"dhcp"`
}

type dhcpXML struct {
	Range dhcpRangeXML `xml:"range"`
}

type dhcpRangeXML struct {
	Start string `xml:"start,attr"`
	End   string `xml:"end,attr"`
}

type networkFilterXML struct {
	XMLName  xml.Name            `xml:"filter"`
	Name     string              `xml:"name,attr"`
	Chain    string              `xml:"chain,attr"`
	Metadata metadataXML         `xml:"metadata"`
	Rules    []networkFilterRule `xml:"rule"`
}

type networkFilterRule struct {
	Action    string             `xml:"action,attr"`
	Direction string             `xml:"direction,attr"`
	Priority  int                `xml:"priority,attr"`
	UDP       *networkFilterUDP  `xml:"udp,omitempty"`
	IP        *networkFilterIP   `xml:"ip,omitempty"`
	IPv6      *networkFilterIPv6 `xml:"ipv6,omitempty"`
}

type networkFilterUDP struct {
	DestinationIP   string `xml:"dstipaddr,attr"`
	DestinationPort int    `xml:"dstportstart,attr"`
}

type networkFilterIP struct {
	DestinationIP string `xml:"dstipaddr,attr"`
}

type networkFilterIPv6 struct{}

func BuildHostIsolationFilterXML(name string, subnet netip.Prefix, ownerScope string) (string, error) {
	if err := validateManagedOwnerScope(ownerScope); err != nil {
		return "", err
	}
	if !managedNamePattern.MatchString(name) {
		return "", fmt.Errorf("invalid managed network filter name %q", name)
	}
	gateway, _, _, _, err := isolatedSubnetAddresses(subnet)
	if err != nil {
		return "", err
	}
	doc := networkFilterXML{
		Name:     name,
		Chain:    "root",
		Metadata: newManagedMetadata(ownerScope),
		Rules: []networkFilterRule{
			{
				Action:    "accept",
				Direction: "out",
				Priority:  100,
				UDP: &networkFilterUDP{
					DestinationIP:   gateway,
					DestinationPort: 67,
				},
			},
			{
				Action:    "drop",
				Direction: "out",
				Priority:  200,
				IP:        &networkFilterIP{DestinationIP: gateway},
			},
			{
				Action:    "drop",
				Direction: "out",
				Priority:  300,
				IPv6:      &networkFilterIPv6{},
			},
		},
	}
	payload, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal network filter XML: %w", err)
	}
	return xml.Header + string(payload) + "\n", nil
}

func BuildIsolatedNetworkXML(name string, subnet netip.Prefix, ownerScope string) (string, error) {
	if err := validateManagedOwnerScope(ownerScope); err != nil {
		return "", err
	}
	if !managedNamePattern.MatchString(name) {
		return "", fmt.Errorf("invalid managed network name %q", name)
	}
	gateway, netmask, dhcpStart, dhcpEnd, err := isolatedSubnetAddresses(subnet)
	if err != nil {
		return "", err
	}
	doc := networkXML{
		Name:     name,
		Metadata: newManagedMetadata(ownerScope),
		IP: networkIP{
			Address: gateway,
			Netmask: netmask,
			DHCP: dhcpXML{
				Range: dhcpRangeXML{Start: dhcpStart, End: dhcpEnd},
			},
		},
	}
	payload, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal network XML: %w", err)
	}
	return xml.Header + string(payload) + "\n", nil
}
