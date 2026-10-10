package libvirt

import (
	"encoding/xml"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Loe159/lpic-daily/internal/runner"
)

func TestBuildDomainXMLContainsOnlyManagedVirtualResources(t *testing.T) {
	stateRoot := filepath.Join(t.TempDir(), "state")
	rootDisk := filepath.Join(stateRoot, "vm", "root.qcow2")
	dataDisk := filepath.Join(stateRoot, "vm", "data.qcow2")

	payload, err := BuildDomainXML(DomainSpec{
		Name:         "lpic-daily-storage-abc123",
		LabID:        "lpic1.104.1.partition-disk",
		OwnerScope:   testManagedOwnerScope,
		MemoryMB:     1024,
		CPUPercent:   150,
		Firmware:     runner.FirmwareUEFI,
		RootDiskPath: rootDisk,
		ExtraDisks: []DiskPath{{
			ID:   "data",
			Path: dataDisk,
		}},
		NetworkName:       "lpic-daily-net-abc123",
		NetworkFilterName: "lpic-daily-net-abc123",
	}, stateRoot)
	if err != nil {
		t.Fatalf("BuildDomainXML() error = %v", err)
	}
	if !hasManagedMetadata(payload, testManagedOwnerScope) {
		t.Fatalf("domain XML missing LPIC Daily ownership metadata:\n%s", payload)
	}

	for _, want := range []string{
		`<domain type="kvm">`,
		"<name>lpic-daily-storage-abc123</name>",
		`<memory unit="MiB">1024</memory>`,
		`<vcpu placement="static">2</vcpu>`,
		"<global_period>100000</global_period>",
		"<global_quota>150000</global_quota>",
		`<os firmware="efi">`,
		"<source file=\"" + rootDisk + "\"></source>",
		`<target dev="vda" bus="virtio"></target>`,
		"<source file=\"" + dataDisk + "\"></source>",
		`<target dev="vdb" bus="virtio"></target>`,
		`<source network="lpic-daily-net-abc123"></source>`,
		`<model type="virtio"></model>`,
		`<filterref filter="lpic-daily-net-abc123"></filterref>`,
		`<serial type="pty">`,
		`<console type="pty">`,
		`<channel type="unix">`,
		`<target type="virtio" name="org.qemu.guest_agent.0"></target>`,
	} {
		if !strings.Contains(payload, want) {
			t.Fatalf("domain XML missing %q:\n%s", want, payload)
		}
	}
	var decoded domainXML
	if err := xml.Unmarshal([]byte(payload), &decoded); err != nil {
		t.Fatalf("domain XML does not round-trip: %v", err)
	}
	if decoded.CPUTune.GlobalPeriod != 100000 || decoded.CPUTune.GlobalQuota != 150000 {
		t.Fatalf("unexpected global CPU limit: %#v", decoded.CPUTune)
	}

	if len(decoded.Devices.Channels) != 1 ||
		decoded.Devices.Channels[0].Type != "unix" ||
		decoded.Devices.Channels[0].Target.Type != "virtio" ||
		decoded.Devices.Channels[0].Target.Name != "org.qemu.guest_agent.0" {
		t.Fatalf("unexpected guest-agent channels: %#v", decoded.Devices.Channels)
	}

	for _, forbidden := range []string{
		"<hostdev",
		"<filesystem",
		"<graphics",
		"<emulator>",
		"qemu:commandline",
		"<redirdev",
	} {
		if strings.Contains(payload, forbidden) {
			t.Fatalf("domain XML contains forbidden %q:\n%s", forbidden, payload)
		}
	}
}

func TestDomainSpecRejectsHostPathsAndUnmanagedNetwork(t *testing.T) {
	stateRoot := filepath.Join(t.TempDir(), "state")
	base := DomainSpec{
		Name:         "lpic-daily-test-abc",
		LabID:        "lpic1.104.1.test",
		OwnerScope:   testManagedOwnerScope,
		MemoryMB:     512,
		CPUPercent:   100,
		Firmware:     runner.FirmwareBIOS,
		RootDiskPath: filepath.Join(stateRoot, "root.qcow2"),
	}
	if err := base.Validate(stateRoot); err != nil {
		t.Fatalf("valid spec rejected: %v", err)
	}

	badPath := base
	badPath.RootDiskPath = "/etc/passwd"
	if err := badPath.Validate(stateRoot); err == nil || !strings.Contains(err.Error(), "escapes trusted root") {
		t.Fatalf("bad path error = %v", err)
	}

	badNetwork := base
	badNetwork.NetworkName = "default"
	if err := badNetwork.Validate(stateRoot); err == nil || !strings.Contains(err.Error(), "managed network") {
		t.Fatalf("bad network error = %v", err)
	}
}

func TestIsolatedNetworkXMLHasNoForwarding(t *testing.T) {
	payload, err := BuildIsolatedNetworkXML("lpic-daily-net-abc", netip.MustParsePrefix("10.77.0.0/28"), testManagedOwnerScope)
	if err != nil {
		t.Fatalf("BuildIsolatedNetworkXML() error = %v", err)
	}
	if strings.Contains(payload, "<forward") {
		t.Fatalf("isolated network unexpectedly forwards traffic:\n%s", payload)
	}
	if !hasManagedMetadata(payload, testManagedOwnerScope) {
		t.Fatalf("network XML missing LPIC Daily ownership metadata:\n%s", payload)
	}
	for _, want := range []string{
		"<name>lpic-daily-net-abc</name>",
		`address="10.77.0.1"`,
		`start="10.77.0.2"`,
		`end="10.77.0.14"`,
	} {
		if !strings.Contains(payload, want) {
			t.Fatalf("network XML missing %q:\n%s", want, payload)
		}
	}

	var decoded networkXML
	if err := xml.Unmarshal([]byte(payload), &decoded); err != nil {
		t.Fatalf("network XML does not round-trip: %v", err)
	}
}

func TestHostIsolationFilterBlocksHostGatewayAndIPv6(t *testing.T) {
	payload, err := BuildHostIsolationFilterXML(
		"lpic-daily-net-abc",
		netip.MustParsePrefix("10.77.0.0/28"),
		testManagedOwnerScope,
	)
	if err != nil {
		t.Fatalf("BuildHostIsolationFilterXML() error = %v", err)
	}
	if !hasManagedMetadata(payload, testManagedOwnerScope) {
		t.Fatalf("network filter XML missing LPIC Daily ownership metadata:\n%s", payload)
	}
	filterUUID, err := managedNetworkFilterUUID("lpic-daily-net-abc", testManagedOwnerScope)
	if err != nil {
		t.Fatalf("managedNetworkFilterUUID() error = %v", err)
	}
	for _, want := range []string{
		`<filter name="lpic-daily-net-abc" chain="root">`,
		"<uuid>" + filterUUID + "</uuid>",
		`<udp dstipaddr="10.77.0.1" dstportstart="67"></udp>`,
		`<ip dstipaddr="10.77.0.1"></ip>`,
		`<ipv6></ipv6>`,
	} {
		if !strings.Contains(payload, want) {
			t.Fatalf("network filter XML missing %q:\n%s", want, payload)
		}
	}
}

func TestNetworkFilterOwnershipFallsBackToDeterministicUUID(t *testing.T) {
	name := "lpic-daily-net-abc"
	filterUUID, err := managedNetworkFilterUUID(name, testManagedOwnerScope)
	if err != nil {
		t.Fatalf("managedNetworkFilterUUID() error = %v", err)
	}
	withoutMetadata := `<filter name="` + name + `"><uuid>` + filterUUID + `</uuid></filter>`
	if !hasManagedNetworkFilterOwnership(withoutMetadata, name, testManagedOwnerScope) {
		t.Fatalf("owner-scoped UUID fallback was rejected: %s", withoutMetadata)
	}

	foreignScope := "fedcba9876543210fedcba9876543210"
	conflictingMetadata := `<filter name="` + name + `"><uuid>` + filterUUID +
		`</uuid><metadata><lpic-daily xmlns="urn:lpic-daily:managed:v1" owner="lpic-daily" version="1" scope="` +
		foreignScope + `"></lpic-daily></metadata></filter>`
	if hasManagedNetworkFilterOwnership(conflictingMetadata, name, testManagedOwnerScope) {
		t.Fatal("conflicting ownership metadata unexpectedly accepted through UUID fallback")
	}
}

func TestImageDescriptorRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.qcow2")
	if err := os.WriteFile(outside, []byte("not-an-image"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "base.qcow2")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}

	image := ImageDescriptor{
		ID:            "fedora-44-x86_64-v1",
		Path:          link,
		SHA256:        strings.Repeat("a", 64),
		Format:        "qcow2",
		Architecture:  "x86_64",
		Distribution:  "fedora",
		VirtualSizeMB: 8192,
		FirmwareModes: []runner.FirmwareMode{runner.FirmwareBIOS},
	}
	if err := image.Validate(root); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("symlink escape error = %v", err)
	}
}

func TestImageDescriptorRejectsPathEscapesAndUnsupportedFirmware(t *testing.T) {
	root := filepath.Join(t.TempDir(), "images")
	image := ImageDescriptor{
		ID:            "fedora-44-x86_64-v1",
		Path:          filepath.Join(root, "fedora-44", "base.qcow2"),
		SHA256:        strings.Repeat("a", 64),
		Format:        "qcow2",
		Architecture:  "x86_64",
		Distribution:  "fedora",
		VirtualSizeMB: 8192,
		FirmwareModes: []runner.FirmwareMode{runner.FirmwareBIOS, runner.FirmwareUEFI},
	}
	if err := image.Validate(root); err != nil {
		t.Fatalf("valid image descriptor rejected: %v", err)
	}
	if !image.SupportsFirmware(runner.FirmwareUEFI) {
		t.Fatal("UEFI support not detected")
	}

	image.Path = "/tmp/foreign.qcow2"
	if err := image.Validate(root); err == nil {
		t.Fatal("escaped image path unexpectedly accepted")
	}
}

func TestBuildDomainXMLSupportsMixedVirtioAndSATAExtraDisks(t *testing.T) {
	stateRoot := filepath.Join(t.TempDir(), "state")
	spec := DomainSpec{
		Name: "lpic-daily-mixed-disks", LabID: "lpic1.101.1.storage",
		OwnerScope: testManagedOwnerScope, MemoryMB: 1024, CPUPercent: 100,
		Firmware: runner.FirmwareUEFI, RootDiskPath: filepath.Join(stateRoot, "root.qcow2"),
		ExtraDisks: []DiskPath{
			{ID: "legacy", Path: filepath.Join(stateRoot, "sata.qcow2"), Bus: runner.VirtualDiskBusSATA},
			{ID: "fast", Path: filepath.Join(stateRoot, "virtio.qcow2")},
		},
	}
	payload, err := BuildDomainXML(spec, stateRoot)
	if err != nil { t.Fatalf("BuildDomainXML(mixed disks): %v", err) }
	for _, expected := range []string{
		`<target dev="vda" bus="virtio"></target>`,
		`<target dev="sda" bus="sata"></target>`,
		`<target dev="vdb" bus="virtio"></target>`,
	} {
		if !strings.Contains(payload, expected) { t.Fatalf("missing %s in %s", expected, payload) }
	}
	var parsed domainXML
	if err := xml.Unmarshal([]byte(payload), &parsed); err != nil { t.Fatal(err) }
	if len(parsed.Devices.Disks) != 3 { t.Fatalf("disk count=%d", len(parsed.Devices.Disks)) }
	for _, badBus := range []runner.VirtualDiskBus{"nvme", "scsi", "hostdev"} {
		invalid := spec
		invalid.ExtraDisks = []DiskPath{{ID: "bad", Path: filepath.Join(stateRoot, "bad.qcow2"), Bus: badBus}}
		if _, err := BuildDomainXML(invalid, stateRoot); err == nil {
			t.Errorf("bus %q unexpectedly accepted", badBus)
		}
	}
}

func TestBuildDomainXMLCreatesPrivateUSBControllerAndDisk(t *testing.T) {
	root := filepath.Join(t.TempDir(), "state")
	spec := DomainSpec{
		Name: "lpic-daily-usb-incident", LabID: "lpic1.101.1.usb",
		OwnerScope: testManagedOwnerScope, MemoryMB: 1024, CPUPercent: 100,
		Firmware: runner.FirmwareUEFI, RootDiskPath: filepath.Join(root, "root.qcow2"),
		ExtraDisks: []DiskPath{
			{ID: "media", Path: filepath.Join(root, "media.qcow2"), Bus: runner.VirtualDiskBusUSB},
			{ID: "sata", Path: filepath.Join(root, "sata.qcow2"), Bus: runner.VirtualDiskBusSATA},
			{ID: "virtio", Path: filepath.Join(root, "virtio.qcow2")},
		},
	}
	payload, err := BuildDomainXML(spec, root)
	if err != nil { t.Fatal(err) }
	for _, expected := range []string{
		`<controller type="usb" index="0" model="qemu-xhci"></controller>`,
		`<target dev="sda" bus="usb"></target>`,
		`<target dev="sdb" bus="sata"></target>`,
		`<target dev="vdb" bus="virtio"></target>`,
	} {
		if !strings.Contains(payload, expected) { t.Errorf("missing %q", expected) }
	}
	for _, forbidden := range []string{"<hostdev", "<redirdev", "qemu:commandline", "<filesystem"} {
		if strings.Contains(payload, forbidden) { t.Errorf("unsafe XML element %q", forbidden) }
	}
	var parsed domainXML
	if err := xml.Unmarshal([]byte(payload), &parsed); err != nil { t.Fatal(err) }
	if len(parsed.Devices.Controllers) != 1 { t.Errorf("USB controllers=%d, want 1", len(parsed.Devices.Controllers)) }
}
