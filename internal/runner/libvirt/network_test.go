package libvirt

import (
	"context"
	"errors"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestChooseIsolatedSubnetAvoidsExistingPrefixes(t *testing.T) {
	first, err := chooseIsolatedSubnet("lpic-daily-test-network", nil)
	if err != nil {
		t.Fatalf("chooseIsolatedSubnet() error = %v", err)
	}
	if first.Bits() != isolatedSubnetBits {
		t.Fatalf("prefix = %s, want /%d", first, isolatedSubnetBits)
	}

	second, err := chooseIsolatedSubnet("lpic-daily-test-network", []netip.Prefix{first})
	if err != nil {
		t.Fatalf("chooseIsolatedSubnet(collision) error = %v", err)
	}
	if second == first || prefixOverlapsAny(second, []netip.Prefix{first}) {
		t.Fatalf("collision was not avoided: first=%s second=%s", first, second)
	}
}

func TestChooseIsolatedSubnetAvoidsBroadHostRoutes(t *testing.T) {
	used := []netip.Prefix{
		netip.MustParsePrefix("10.0.0.0/8"),
		netip.MustParsePrefix("172.16.0.0/12"),
	}
	prefix, err := chooseIsolatedSubnet("lpic-daily-fallback-network", used)
	if err != nil {
		t.Fatalf("chooseIsolatedSubnet() error = %v", err)
	}
	if !netip.MustParsePrefix("192.168.0.0/16").Contains(prefix.Addr()) {
		t.Fatalf("prefix = %s, want fallback inside 192.168.0.0/16", prefix)
	}
}

func TestParseProcIPv4Routes(t *testing.T) {
	payload := strings.NewReader(
		"Iface\tDestination\tGateway\tFlags\tRefCnt\tUse\tMetric\tMask\tMTU\tWindow\tIRTT\n" +
			"eth0\t00000000\t010011AC\t0003\t0\t0\t0\t00000000\t0\t0\t0\n" +
			"eth0\t000011AC\t00000000\t0001\t0\t0\t0\t0000FFFF\t0\t0\t0\n" +
			"wg0\t0000000A\t00000000\t0001\t0\t0\t0\t000000FF\t0\t0\t0\n",
	)
	routes, err := parseProcIPv4Routes(payload)
	if err != nil {
		t.Fatalf("parseProcIPv4Routes() error = %v", err)
	}
	want := map[netip.Prefix]bool{
		netip.MustParsePrefix("172.17.0.0/16"): false,
		netip.MustParsePrefix("10.0.0.0/8"):     false,
	}
	if len(routes) != len(want) {
		t.Fatalf("routes = %#v, want %d entries", routes, len(want))
	}
	for _, route := range routes {
		if _, exists := want[route]; !exists {
			t.Fatalf("unexpected route %s", route)
		}
		want[route] = true
	}
	for prefix, seen := range want {
		if !seen {
			t.Fatalf("missing route %s", prefix)
		}
	}
}

func TestNetworkPrefixesFromXML(t *testing.T) {
	prefixes, err := networkPrefixesFromXML(`<network>
  <ip address="192.168.122.1" netmask="255.255.255.0"/>
  <ip family="ipv6" address="fd00::1" prefix="64"/>
  <ip address="10.20.30.1" prefix="28"/>
</network>`)
	if err != nil {
		t.Fatalf("networkPrefixesFromXML() error = %v", err)
	}
	want := map[netip.Prefix]bool{
		netip.MustParsePrefix("192.168.122.0/24"): false,
		netip.MustParsePrefix("10.20.30.0/28"):    false,
	}
	if len(prefixes) != len(want) {
		t.Fatalf("prefixes = %#v", prefixes)
	}
	for _, prefix := range prefixes {
		if _, exists := want[prefix]; !exists {
			t.Fatalf("unexpected prefix %s", prefix)
		}
		want[prefix] = true
	}
}


func TestNetworkAllocationLockRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.WriteFile(target, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "lock")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := acquireNetworkAllocationLock(context.Background(), link); err == nil ||
		!strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("symlink lock error = %v", err)
	}
}

func TestNetworkAllocationLockSerializesConcurrentAllocators(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "network-allocation.lock")
	if err := os.WriteFile(lockPath, nil, 0o600); err != nil {
		t.Fatalf("WriteFile(lock) error = %v", err)
	}
	first, err := acquireNetworkAllocationLock(context.Background(), lockPath)
	if err != nil {
		t.Fatalf("first acquireNetworkAllocationLock() error = %v", err)
	}

	waitCtx, cancel := context.WithTimeout(context.Background(), 75*time.Millisecond)
	defer cancel()
	if _, err := acquireNetworkAllocationLock(waitCtx, lockPath); !errors.Is(err, context.DeadlineExceeded) {
		_ = releaseNetworkAllocationLock(first)
		t.Fatalf("second acquire error = %v, want context deadline", err)
	}
	if err := releaseNetworkAllocationLock(first); err != nil {
		t.Fatalf("release first allocation lock: %v", err)
	}

	second, err := acquireNetworkAllocationLock(context.Background(), lockPath)
	if err != nil {
		t.Fatalf("acquire after release error = %v", err)
	}
	if err := releaseNetworkAllocationLock(second); err != nil {
		t.Fatalf("release second allocation lock: %v", err)
	}
}
