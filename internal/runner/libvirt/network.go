package libvirt

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	golibvirt "github.com/digitalocean/go-libvirt"
	"golang.org/x/sys/unix"
)

const isolatedSubnetBits = 28

type NetworkAddressInventory interface {
	ListNetworkIPv4Prefixes() ([]netip.Prefix, error)
}

type networkAddressXML struct {
	IPs []networkIPXML `xml:"ip"`
}

type networkIPXML struct {
	Address string `xml:"address,attr"`
	Netmask string `xml:"netmask,attr"`
	Prefix  string `xml:"prefix,attr"`
}

func (control *RPCControlPlane) ListNetworkIPv4Prefixes() ([]netip.Prefix, error) {
	if control == nil || control.raw == nil {
		return nil, errors.New("libvirt control plane is not initialized")
	}
	networks, _, err := control.raw.ConnectListAllNetworks(
		1,
		golibvirt.ConnectListNetworksActive|golibvirt.ConnectListNetworksInactive,
	)
	if err != nil {
		return nil, fmt.Errorf("list libvirt networks for address allocation: %w", err)
	}

	seen := map[netip.Prefix]struct{}{}
	for _, network := range networks {
		payload, err := control.raw.NetworkGetXMLDesc(network, 0)
		if err != nil {
			return nil, fmt.Errorf("inspect network %s address allocation: %w", network.Name, err)
		}
		prefixes, err := networkPrefixesFromXML(payload)
		if err != nil {
			return nil, fmt.Errorf("parse network %s address allocation: %w", network.Name, err)
		}
		for _, prefix := range prefixes {
			seen[prefix] = struct{}{}
		}
	}

	result := make([]netip.Prefix, 0, len(seen))
	for prefix := range seen {
		result = append(result, prefix)
	}
	return result, nil
}

func networkPrefixesFromXML(payload string) ([]netip.Prefix, error) {
	var document networkAddressXML
	if err := xml.Unmarshal([]byte(payload), &document); err != nil {
		return nil, err
	}
	result := make([]netip.Prefix, 0, len(document.IPs))
	for _, item := range document.IPs {
		address, err := netip.ParseAddr(strings.TrimSpace(item.Address))
		if err != nil {
			return nil, fmt.Errorf("invalid network address %q: %w", item.Address, err)
		}
		if !address.Is4() {
			continue
		}

		bits := 0
		switch {
		case strings.TrimSpace(item.Prefix) != "":
			bits, err = strconv.Atoi(item.Prefix)
			if err != nil || bits < 0 || bits > 32 {
				return nil, fmt.Errorf("invalid IPv4 prefix %q", item.Prefix)
			}
		case strings.TrimSpace(item.Netmask) != "":
			maskIP := net.ParseIP(item.Netmask).To4()
			if maskIP == nil {
				return nil, fmt.Errorf("invalid IPv4 netmask %q", item.Netmask)
			}
			mask := net.IPMask(maskIP)
			var total int
			bits, total = mask.Size()
			if total != 32 {
				return nil, fmt.Errorf("non-contiguous IPv4 netmask %q", item.Netmask)
			}
		default:
			return nil, fmt.Errorf("IPv4 network %s has neither prefix nor netmask", address)
		}
		result = append(result, netip.PrefixFrom(address, bits).Masked())
	}
	return result, nil
}

func hostIPv4Routes() ([]netip.Prefix, error) {
	file, err := os.Open("/proc/net/route")
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open host IPv4 route table: %w", err)
	}
	defer file.Close()
	return parseProcIPv4Routes(file)
}

func parseProcIPv4Routes(reader io.Reader) ([]netip.Prefix, error) {
	scanner := bufio.NewScanner(reader)
	first := true
	var result []netip.Prefix
	for scanner.Scan() {
		if first {
			first = false
			continue
		}
		fields := strings.Fields(scanner.Text())
		if len(fields) < 8 {
			continue
		}
		destination, err := strconv.ParseUint(fields[1], 16, 32)
		if err != nil {
			return nil, fmt.Errorf("parse host route destination %q: %w", fields[1], err)
		}
		maskValue, err := strconv.ParseUint(fields[7], 16, 32)
		if err != nil {
			return nil, fmt.Errorf("parse host route mask %q: %w", fields[7], err)
		}
		address := procRouteAddr(uint32(destination))
		maskAddr := procRouteAddr(uint32(maskValue))
		maskBytes := maskAddr.As4()
		mask := net.IPMask(maskBytes[:])
		ones, total := mask.Size()
		if total != 32 {
			return nil, fmt.Errorf("host route has non-contiguous mask %s", maskAddr)
		}
		if ones == 0 {
			// The default route does not reserve the whole IPv4 address space.
			continue
		}
		result = append(result, netip.PrefixFrom(address, ones).Masked())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read host IPv4 route table: %w", err)
	}
	return result, nil
}

func procRouteAddr(value uint32) netip.Addr {
	var bytes [4]byte
	binary.LittleEndian.PutUint32(bytes[:], value)
	return netip.AddrFrom4(bytes)
}

type privateIPv4Pool struct {
	prefix netip.Prefix
}

var isolatedPrivatePools = []privateIPv4Pool{
	{prefix: netip.MustParsePrefix("10.0.0.0/8")},
	{prefix: netip.MustParsePrefix("172.16.0.0/12")},
	{prefix: netip.MustParsePrefix("192.168.0.0/16")},
}

func (backend *Backend) allocateIsolatedSubnet(name string) (netip.Prefix, error) {
	inventory, ok := backend.control.(NetworkAddressInventory)
	if !ok {
		return netip.Prefix{}, fmt.Errorf(
			"%w: libvirt control plane cannot inventory existing network addresses",
			ErrNetworkAddressInventoryUnsupported,
		)
	}
	used, err := hostIPv4Routes()
	if err != nil {
		return netip.Prefix{}, err
	}
	libvirtPrefixes, err := inventory.ListNetworkIPv4Prefixes()
	if err != nil {
		return netip.Prefix{}, err
	}
	used = append(used, libvirtPrefixes...)
	return chooseIsolatedSubnet(name, used)
}

var ErrNetworkAddressInventoryUnsupported = errors.New("network address inventory unavailable")

func chooseIsolatedSubnet(name string, used []netip.Prefix) (netip.Prefix, error) {
	if strings.TrimSpace(name) == "" {
		return netip.Prefix{}, errors.New("network name is required")
	}
	for poolIndex, pool := range isolatedPrivatePools {
		count := 1 << (isolatedSubnetBits - pool.prefix.Bits())
		for attempt := 0; attempt < count && attempt < 4096; attempt++ {
			digest := sha256.Sum256([]byte(fmt.Sprintf("%s:%d:%d", name, poolIndex, attempt)))
			index := binary.BigEndian.Uint32(digest[:4]) % uint32(count)
			candidate, err := subnetAt(pool.prefix, isolatedSubnetBits, index)
			if err != nil {
				return netip.Prefix{}, err
			}
			if !prefixOverlapsAny(candidate, used) {
				return candidate, nil
			}
		}
	}
	return netip.Prefix{}, errors.New("no collision-free isolated IPv4 subnet is available")
}

func subnetAt(pool netip.Prefix, bits int, index uint32) (netip.Prefix, error) {
	pool = pool.Masked()
	if !pool.Addr().Is4() || bits < pool.Bits() || bits > 32 {
		return netip.Prefix{}, errors.New("invalid IPv4 subnet pool")
	}
	count := uint32(1) << uint(bits-pool.Bits())
	if index >= count {
		return netip.Prefix{}, errors.New("subnet index outside pool")
	}
	baseBytes := pool.Addr().As4()
	base := binary.BigEndian.Uint32(baseBytes[:])
	offset := index << uint(32-bits)
	value := base + offset
	var addressBytes [4]byte
	binary.BigEndian.PutUint32(addressBytes[:], value)
	return netip.PrefixFrom(netip.AddrFrom4(addressBytes), bits).Masked(), nil
}

func prefixOverlapsAny(candidate netip.Prefix, used []netip.Prefix) bool {
	for _, existing := range used {
		if !existing.IsValid() || !existing.Addr().Is4() {
			continue
		}
		existing = existing.Masked()
		if candidate.Contains(existing.Addr()) || existing.Contains(candidate.Addr()) {
			return true
		}
	}
	return false
}

func isolatedSubnetAddresses(prefix netip.Prefix) (gateway, netmask, dhcpStart, dhcpEnd string, err error) {
	prefix = prefix.Masked()
	if !prefix.IsValid() || !prefix.Addr().Is4() || prefix.Bits() != isolatedSubnetBits {
		return "", "", "", "", fmt.Errorf("isolated network must be an IPv4 /%d", isolatedSubnetBits)
	}
	baseBytes := prefix.Addr().As4()
	base := binary.BigEndian.Uint32(baseBytes[:])
	address := func(offset uint32) string {
		var value [4]byte
		binary.BigEndian.PutUint32(value[:], base+offset)
		return netip.AddrFrom4(value).String()
	}
	mask := net.CIDRMask(isolatedSubnetBits, 32)
	return address(1), net.IP(mask).String(), address(2), address(14), nil
}


func acquireNetworkAllocationLock(ctx context.Context, stateRoot string) (*os.File, error) {
	if err := os.MkdirAll(stateRoot, 0o700); err != nil {
		return nil, fmt.Errorf("create VM state root: %w", err)
	}
	lockPath := filepath.Join(stateRoot, ".network-allocation.lock")
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open network allocation lock: %w", err)
	}
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		if err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB); err == nil {
			return file, nil
		} else if !errors.Is(err, unix.EWOULDBLOCK) && !errors.Is(err, unix.EAGAIN) {
			_ = file.Close()
			return nil, fmt.Errorf("lock network allocation: %w", err)
		}
		select {
		case <-ctx.Done():
			_ = file.Close()
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

func releaseNetworkAllocationLock(file *os.File) error {
	if file == nil {
		return nil
	}
	unlockErr := unix.Flock(int(file.Fd()), unix.LOCK_UN)
	closeErr := file.Close()
	return errors.Join(unlockErr, closeErr)
}
