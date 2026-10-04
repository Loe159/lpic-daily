//go:build integration

package libvirt

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	lpicdaily "github.com/Loe159/lpic-daily"
	"github.com/Loe159/lpic-daily/internal/appstate"
	"github.com/Loe159/lpic-daily/internal/lab"
	"github.com/Loe159/lpic-daily/internal/runner"
	"github.com/muesli/cancelreader"
)

type integrationLockedBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (buffer *integrationLockedBuffer) Write(payload []byte) (int, error) {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	return buffer.buffer.Write(payload)
}

func (buffer *integrationLockedBuffer) String() string {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	return buffer.buffer.String()
}

func TestRealKVMIsolationScenarioAndCrashReaping(t *testing.T) {
	if os.Getenv("LPIC_DAILY_RUN_KVM_INTEGRATION") != "1" {
		t.Skip("set LPIC_DAILY_RUN_KVM_INTEGRATION=1 to run the real qemu:///system KVM test")
	}
	imageRoot := os.Getenv("LPIC_DAILY_VM_IMAGE_DIR")
	if imageRoot == "" {
		var err error
		imageRoot, err = appstate.VMImageRoot()
		if err != nil {
			t.Fatalf("VMImageRoot() error = %v", err)
		}
	}
	if !filepath.IsAbs(imageRoot) {
		t.Fatal("VM image root must be absolute")
	}
	catalog, err := LoadImageCatalog(filepath.Join(imageRoot, "catalog.json"), imageRoot)
	if err != nil {
		t.Fatalf("LoadImageCatalog() error = %v", err)
	}
	image, err := catalog.Resolve("fedora-44-x86_64-v2", imageRoot)
	if err != nil {
		t.Fatalf("resolve Fedora test image: %v", err)
	}
	baseBefore, err := integrationFileSHA256(image.Path)
	if err != nil {
		t.Fatalf("hash base image before test: %v", err)
	}

	sentinel, err := os.CreateTemp("", "lpic-daily-kvm-host-sentinel-*")
	if err != nil {
		t.Fatalf("CreateTemp() error = %v", err)
	}
	sentinelPath := sentinel.Name()
	const sentinelBody = "LPIC Daily host sentinel must not change\n"
	if _, err := sentinel.WriteString(sentinelBody); err != nil {
		_ = sentinel.Close()
		t.Fatalf("write sentinel: %v", err)
	}
	if err := sentinel.Close(); err != nil {
		t.Fatalf("close sentinel: %v", err)
	}
	defer os.Remove(sentinelPath)

	control, err := OpenSystem()
	if err != nil {
		t.Fatalf("OpenSystem() error = %v", err)
	}
	commands, err := NewExecCommandRunner()
	if err != nil {
		_ = control.Close()
		t.Fatalf("NewExecCommandRunner() error = %v", err)
	}
	stateRoot, err := appstate.VMStateRoot()
	if err != nil {
		_ = control.Close()
		t.Fatalf("VMStateRoot() error = %v", err)
	}
	backend, err := NewBackend(
		control,
		catalog,
		imageRoot,
		stateRoot,
		appstate.VMNetworkAllocationLockPath(),
		commands,
	)
	if err != nil {
		_ = control.Close()
		t.Fatalf("NewBackend() error = %v", err)
	}
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cleanupCancel()
		if err := backend.Close(cleanupCtx); err != nil {
			t.Errorf("backend Close() error = %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	definition := runner.Definition{
		LabID:             "integration.kvm.peer-a",
		ImageRef:          "fedora-44-x86_64-v2",
		Distribution:      "fedora",
		Network:           runner.NetworkIsolated,
		CapabilityProfile: "full-machine",
		MemoryMB:          1024,
		CPUPercent:        100,
		PIDs:              128,
		Timeout:           5 * time.Minute,
		Machine:           &runner.MachineDefinition{Firmware: runner.FirmwareUEFI},
	}
	peer := definition
	peer.LabID = "integration.kvm.peer-b"

	hostUplinkIP := integrationHostDefaultUplinkIPv4(t)
	hostUplinkListener, err := net.Listen("tcp4", net.JoinHostPort(hostUplinkIP, "0"))
	if err != nil {
		t.Fatalf("listen on host uplink %s: %v", hostUplinkIP, err)
	}
	defer hostUplinkListener.Close()

	scenario, err := backend.PrepareScenario(ctx, []runner.Definition{definition, peer})
	if err != nil {
		t.Fatalf("PrepareScenario() error = %v", err)
	}
	scenarioOpen := true
	defer func() {
		if scenarioOpen {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 90*time.Second)
			defer cleanupCancel()
			_ = backend.DestroyScenario(cleanupCtx, scenario)
		}
	}()
	for _, instance := range scenario.Instances {
		if err := backend.Start(ctx, instance); err != nil {
			t.Fatalf("Start(%s) error = %v", instance.ID, err)
		}
	}

	firstIP := integrationGuestIPv4(t, ctx, backend, scenario.Instances[0])
	gatewayIP := integrationGatewayForGuestIPv4(t, firstIP)
	probeInstance := scenario.Instances[1]

	gatewayListener, err := net.Listen("tcp4", net.JoinHostPort(gatewayIP, "0"))
	if err != nil {
		t.Fatalf("listen on isolated host gateway %s: %v", gatewayIP, err)
	}
	gatewayPort := gatewayListener.Addr().(*net.TCPAddr).Port
	gatewayAccepted := make(chan error, 1)
	go func() {
		connection, acceptErr := gatewayListener.Accept()
		if connection != nil {
			_ = connection.Close()
		}
		gatewayAccepted <- acceptErr
	}()

	uplinkPort := hostUplinkListener.Addr().(*net.TCPAddr).Port
	uplinkAccepted := make(chan error, 1)
	go func() {
		connection, acceptErr := hostUplinkListener.Accept()
		if connection != nil {
			_ = connection.Close()
		}
		uplinkAccepted <- acceptErr
	}()

	var networkProbeOutput bytes.Buffer
	probeScript := fmt.Sprintf(`
set +e
/usr/bin/ping -c 1 -W 3 %s >/dev/null 2>&1
printf 'peer_ping=%%d\n' "$?"
/usr/bin/timeout 3 /usr/bin/bash -c 'printf probe >/dev/tcp/%s/%d' >/dev/null 2>&1
printf 'gateway_tcp=%%d\n' "$?"
/usr/bin/ping -c 1 -W 2 1.1.1.1 >/dev/null 2>&1
printf 'public_icmp=%%d\n' "$?"
/usr/bin/timeout 3 /usr/bin/bash -c 'printf probe >/dev/tcp/1.1.1.1/443' >/dev/null 2>&1
printf 'public_tcp=%%d\n' "$?"
/usr/bin/timeout 3 /usr/bin/bash -c 'printf probe >/dev/tcp/%s/%d' >/dev/null 2>&1
printf 'host_uplink_tcp=%%d\n' "$?"
rm -f -- %s
printf guest-write > /root/lpic-daily-isolation-probe
printf 'guest_write=%%d\n' "$?"
exit 0
`,
		firstIP,
		gatewayIP,
		gatewayPort,
		hostUplinkIP,
		uplinkPort,
		shellQuote(sentinelPath),
	)
	result, err := backend.Exec(ctx, probeInstance, runner.ExecRequest{
		Argv:   []string{"/usr/bin/bash", "-c", probeScript},
		Stdout: &networkProbeOutput,
		Stderr: &networkProbeOutput,
	})
	if err != nil || result.ExitCode != 0 {
		t.Fatalf(
			"isolated-network consolidated guest probe = exit %d, err %v, output %q",
			result.ExitCode,
			err,
			networkProbeOutput.String(),
		)
	}
	statuses := integrationParseProbeStatuses(t, networkProbeOutput.String())
	if statuses["peer_ping"] != 0 {
		t.Fatalf("peer ping exit = %d, want 0; output=%q", statuses["peer_ping"], networkProbeOutput.String())
	}
	for _, blocked := range []string{"gateway_tcp", "public_icmp", "public_tcp", "host_uplink_tcp"} {
		if statuses[blocked] == 0 {
			t.Fatalf("%s unexpectedly succeeded; output=%q", blocked, networkProbeOutput.String())
		}
	}
	if statuses["guest_write"] != 0 {
		t.Fatalf("guest isolation write exit = %d, want 0; output=%q", statuses["guest_write"], networkProbeOutput.String())
	}

	_ = gatewayListener.Close()
	select {
	case acceptErr := <-gatewayAccepted:
		if acceptErr == nil {
			t.Fatalf("isolated VM connected to host bridge gateway %s:%d", gatewayIP, gatewayPort)
		}
	case <-time.After(time.Second):
		t.Fatal("host bridge gateway sentinel did not stop after listener close")
	}

	_ = hostUplinkListener.Close()
	select {
	case acceptErr := <-uplinkAccepted:
		if acceptErr == nil {
			t.Fatalf("isolated VM connected to host uplink sentinel %s:%d", hostUplinkIP, uplinkPort)
		}
	case <-time.After(time.Second):
		t.Fatal("host uplink sentinel did not stop after listener close")
	}

	if err := backend.DestroyScenario(ctx, scenario); err != nil {
		t.Fatalf("DestroyScenario() error = %v", err)
	}
	scenarioOpen = false

	// Simulate a process crash after a VM is active: drop in-memory ownership and
	// release the lease without normal libvirt/overlay teardown, then reap it.
	orphanDefinition := definition
	orphanDefinition.LabID = "integration.kvm.orphan"
	orphanDefinition.Network = runner.NetworkNone
	orphan, err := backend.Prepare(ctx, orphanDefinition)
	if err != nil {
		t.Fatalf("Prepare(orphan) error = %v", err)
	}
	if err := backend.Start(ctx, orphan); err != nil {
		t.Fatalf("Start(orphan) error = %v", err)
	}
	backend.mu.Lock()
	managed := backend.instances[orphan.ID]
	delete(backend.instances, orphan.ID)
	backend.mu.Unlock()
	if err := releaseInstanceLease(managed.Paths.Lease); err != nil {
		t.Fatalf("release simulated-crash lease: %v", err)
	}
	if err := backend.Reap(ctx); err != nil {
		t.Fatalf("Reap() error = %v", err)
	}
	domains, err := control.ListManagedDomains()
	if err != nil {
		t.Fatalf("ListManagedDomains() error = %v", err)
	}
	for _, name := range domains {
		if name == orphan.ID {
			t.Fatalf("orphan domain %s survived Reap()", orphan.ID)
		}
	}

	consoleDefinition := definition
	consoleDefinition.LabID = "integration.kvm.serial-console"
	consoleDefinition.Network = runner.NetworkNone
	consoleInstance, err := backend.Prepare(ctx, consoleDefinition)
	if err != nil {
		t.Fatalf("Prepare(console) error = %v", err)
	}
	if err := backend.Start(ctx, consoleInstance); err != nil {
		t.Fatalf("Start(console) error = %v", err)
	}

	consoleInput, consoleInputWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("create console input pipe: %v", err)
	}
	defer consoleInputWriter.Close()
	cancelableInput, err := cancelreader.NewReader(consoleInput)
	if err != nil {
		_ = consoleInput.Close()
		t.Fatalf("create cancellable console input: %v", err)
	}
	defer cancelableInput.Close()
	defer consoleInput.Close()

	consoleCtx, cancelConsole := context.WithCancel(ctx)
	consoleDone := make(chan error, 1)
	var consoleOutput integrationLockedBuffer
	go func() {
		consoleDone <- backend.OpenConsole(consoleCtx, consoleInstance, runner.ConsoleRequest{
			Stdin:  cancelableInput,
			Stdout: &consoleOutput,
		})
	}()

	outputDeadline := time.NewTimer(30 * time.Second)
	outputTicker := time.NewTicker(100 * time.Millisecond)
	for !strings.Contains(consoleOutput.String(), "GNU GRUB") {
		select {
		case err := <-consoleDone:
			outputTicker.Stop()
			outputDeadline.Stop()
			t.Fatalf("serial console closed before producing guest output: %v", err)
		case <-outputTicker.C:
		case <-outputDeadline.C:
			outputTicker.Stop()
			t.Fatalf("serial console did not expose GRUB before userland; output=%q", consoleOutput.String())
		}
	}
	outputTicker.Stop()
	if !outputDeadline.Stop() {
		select {
		case <-outputDeadline.C:
		default:
		}
	}

	cancelConsole()
	cancelableInput.Cancel()
	select {
	case err := <-consoleDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("serial console cancellation error = %v, want context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("serial console did not stop after context cancellation")
	}

	var postConsoleOutput bytes.Buffer
	result, err = backend.Exec(ctx, consoleInstance, runner.ExecRequest{
		Argv:   []string{"/usr/bin/printf", "console-stream-closed-vm-still-running"},
		Stdout: &postConsoleOutput,
	})
	if err != nil || result.ExitCode != 0 {
		t.Fatalf("guest unusable after console cancellation: exit=%d err=%v", result.ExitCode, err)
	}
	if postConsoleOutput.String() != "console-stream-closed-vm-still-running" {
		t.Fatalf("unexpected post-console output %q", postConsoleOutput.String())
	}

	if err := backend.Destroy(ctx, consoleInstance); err != nil {
		t.Fatalf("Destroy(console) after cancellation error = %v", err)
	}

	baseAfter, err := integrationFileSHA256(image.Path)
	if err != nil {
		t.Fatalf("hash base image after test: %v", err)
	}
	if baseAfter != baseBefore {
		t.Fatalf("base image changed: before=%s after=%s", baseBefore, baseAfter)
	}
	sentinelAfter, err := os.ReadFile(sentinelPath)
	if err != nil {
		t.Fatalf("host sentinel disappeared: %v", err)
	}
	if string(sentinelAfter) != sentinelBody {
		t.Fatalf("host sentinel changed: got %q", sentinelAfter)
	}
}

func integrationGuestIPv4(
	t *testing.T,
	ctx context.Context,
	backend *Backend,
	instance runner.Instance,
) string {
	t.Helper()
	return integrationWaitForGuestIPv4(
		t,
		ctx,
		backend,
		instance,
		"guest IPv4",
		"ip -4 -o addr show scope global | awk '{split($4,a,\"/\"); print a[1]; exit}'",
	)
}

func integrationGatewayForGuestIPv4(t *testing.T, guestIP string) string {
	t.Helper()
	address, err := netip.ParseAddr(strings.TrimSpace(guestIP))
	if err != nil || !address.Is4() {
		t.Fatalf("parse guest IPv4 %q: %v", guestIP, err)
	}
	prefix := netip.PrefixFrom(address, isolatedSubnetBits).Masked()
	gateway, _, _, _, err := isolatedSubnetAddresses(prefix)
	if err != nil {
		t.Fatalf("derive isolated gateway from guest IPv4 %s: %v", guestIP, err)
	}
	return gateway
}

func integrationWaitForGuestIPv4(
	t *testing.T,
	ctx context.Context,
	backend *Backend,
	instance runner.Instance,
	label string,
	command string,
) string {
	t.Helper()
	waitCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	lastDetail := "no probe completed"
	for {
		var stdout bytes.Buffer
		result, err := backend.Exec(waitCtx, instance, runner.ExecRequest{
			Argv:   []string{"/usr/bin/sh", "-c", command},
			Stdout: &stdout,
		})
		value := strings.TrimSpace(stdout.String())
		if err == nil && result.ExitCode == 0 {
			if parsed := net.ParseIP(value); parsed != nil && parsed.To4() != nil &&
				!strings.ContainsAny(value, " \t\r\n") {
				return value
			}
			lastDetail = fmt.Sprintf("invalid value %q", value)
		} else if err != nil {
			lastDetail = err.Error()
		} else {
			lastDetail = fmt.Sprintf("probe exit=%d output=%q", result.ExitCode, value)
		}

		select {
		case <-waitCtx.Done():
			t.Fatalf("wait for %s: %v (last probe: %s)", label, waitCtx.Err(), lastDetail)
		case <-ticker.C:
		}
	}
}

func integrationParseProbeStatuses(t *testing.T, output string) map[string]int {
	t.Helper()
	required := map[string]struct{}{
		"peer_ping":       {},
		"gateway_tcp":     {},
		"public_icmp":     {},
		"public_tcp":      {},
		"host_uplink_tcp": {},
		"guest_write":     {},
	}
	statuses := make(map[string]int, len(required))
	for _, line := range strings.Split(output, "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		if _, wanted := required[key]; !wanted {
			continue
		}
		exitCode, err := strconv.Atoi(value)
		if err != nil {
			t.Fatalf("parse probe status %q: %v", line, err)
		}
		statuses[key] = exitCode
	}
	for key := range required {
		if _, exists := statuses[key]; !exists {
			t.Fatalf("missing probe status %s in output %q", key, output)
		}
	}
	return statuses
}

func integrationFileSHA256(path string) (string, error) {
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	defer file.Close()
	hasher := sha256.New()
	if _, err := io.CopyBuffer(hasher, file, make([]byte, 1<<20)); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func integrationHostDefaultUplinkIPv4(t *testing.T) string {
	t.Helper()

	routeTable, err := os.Open("/proc/net/route")
	if err != nil {
		t.Fatalf("open host route table: %v", err)
	}
	defer routeTable.Close()

	interfaceName := ""
	scanner := bufio.NewScanner(routeTable)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 || fields[1] != "00000000" {
			continue
		}
		flags, err := strconv.ParseUint(fields[3], 16, 64)
		if err != nil || flags&0x1 == 0 {
			continue
		}
		interfaceName = fields[0]
		break
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("read host route table: %v", err)
	}
	if interfaceName == "" {
		t.Fatal("host has no usable default IPv4 uplink")
	}

	hostInterface, err := net.InterfaceByName(interfaceName)
	if err != nil {
		t.Fatalf("resolve default interface %s: %v", interfaceName, err)
	}
	addresses, err := hostInterface.Addrs()
	if err != nil {
		t.Fatalf("list addresses for default interface %s: %v", interfaceName, err)
	}
	for _, address := range addresses {
		var ip net.IP
		switch value := address.(type) {
		case *net.IPNet:
			ip = value.IP
		case *net.IPAddr:
			ip = value.IP
		}
		if ipv4 := ip.To4(); ipv4 != nil && !ipv4.IsLoopback() && !ipv4.IsLinkLocalUnicast() {
			return ipv4.String()
		}
	}
	t.Fatalf("default interface %s has no usable IPv4 address", interfaceName)
	return ""
}

func TestRealKVMPhase2ReferenceLabs(t *testing.T) {
	if os.Getenv("LPIC_DAILY_RUN_KVM_INTEGRATION") != "1" {
		t.Skip("set LPIC_DAILY_RUN_KVM_INTEGRATION=1 to run the real qemu:///system KVM test")
	}
	imageRoot := os.Getenv("LPIC_DAILY_VM_IMAGE_DIR")
	if imageRoot == "" {
		var err error
		imageRoot, err = appstate.VMImageRoot()
		if err != nil {
			t.Fatalf("VMImageRoot() error = %v", err)
		}
	}
	catalog, err := LoadImageCatalog(filepath.Join(imageRoot, "catalog.json"), imageRoot)
	if err != nil {
		t.Fatalf("LoadImageCatalog() error = %v", err)
	}
	control, err := OpenSystem()
	if err != nil {
		t.Fatalf("OpenSystem() error = %v", err)
	}
	commands, err := NewExecCommandRunner()
	if err != nil {
		_ = control.Close()
		t.Fatalf("NewExecCommandRunner() error = %v", err)
	}
	stateRoot, err := appstate.VMStateRoot()
	if err != nil {
		_ = control.Close()
		t.Fatalf("VMStateRoot() error = %v", err)
	}
	backend, err := NewBackend(
		control,
		catalog,
		imageRoot,
		stateRoot,
		appstate.VMNetworkAllocationLockPath(),
		commands,
	)
	if err != nil {
		_ = control.Close()
		t.Fatalf("NewBackend() error = %v", err)
	}
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cleanupCancel()
		if err := backend.Close(cleanupCtx); err != nil {
			t.Errorf("backend Close() error = %v", err)
		}
	}()

	authoredLabs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	wanted := map[string]string{
		"lpic1.104.1.partition-filesystems": "labs/lpic-1-v5/104.1/partition-filesystems/reference-solution.sh",
		"lpic1.102.2.grub-kernel-parameter": "labs/lpic-1-v5/102.2/grub-kernel-parameter/reference-solution.sh",
	}
	found := map[string]bool{}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()

	for _, authored := range authoredLabs {
		solutionPath, ok := wanted[authored.Definition.ID]
		if !ok {
			continue
		}
		found[authored.Definition.ID] = true
		authored := authored
		t.Run(authored.Definition.ID, func(t *testing.T) {
			solution, err := fs.ReadFile(lpicdaily.BuiltinFS, solutionPath)
			if err != nil {
				t.Fatalf("read reference solution: %v", err)
			}
			session, err := lab.Start(ctx, authored, backend)
			if err != nil {
				t.Fatalf("Start() error = %v", err)
			}
			defer func() {
				cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 90*time.Second)
				defer cleanupCancel()
				if err := session.Close(cleanupCtx); err != nil {
					t.Errorf("Close() error = %v", err)
				}
			}()

			assertPhase2LabNotSolved(t, ctx, session)
			runPhase2ReferenceSolution(t, ctx, backend, session.Instance, solution)
			if authored.Definition.ID == "lpic1.102.2.grub-kernel-parameter" {
				if err := backend.Reboot(ctx, session.Instance); err != nil {
					t.Fatalf("Reboot() error = %v", err)
				}
			}
			assertPhase2LabSolved(t, ctx, session)
		})
	}
	for id := range wanted {
		if !found[id] {
			t.Errorf("Phase-2 reference lab %s was not loaded", id)
		}
	}
}

func runPhase2ReferenceSolution(t *testing.T, ctx context.Context, backend runner.Runner, instance runner.Instance, script []byte) {
	t.Helper()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	result, err := backend.Exec(ctx, instance, runner.ExecRequest{
		Argv:   []string{"/usr/bin/bash", "-eu", "-c", string(script)},
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		t.Fatalf("reference solution exec error = %v; stdout=%q stderr=%q", err, stdout.String(), stderr.String())
	}
	if result.ExitCode != 0 {
		t.Fatalf(
			"reference solution exit = %d, want 0; stdout=%q stderr=%q",
			result.ExitCode,
			stdout.String(),
			stderr.String(),
		)
	}
}

func assertPhase2LabSolved(t *testing.T, ctx context.Context, session *lab.Session) {
	t.Helper()
	results, err := session.Evaluate(ctx)
	if err != nil {
		t.Fatalf("Evaluate() solved state error = %v", err)
	}
	if len(results) == 0 {
		t.Fatal("lab has no checker results")
	}
	for _, result := range results {
		if !result.Pass {
			t.Fatalf("check %s failed after reference solution: %s (%v)", result.CheckID, result.Detail, result.Err)
		}
	}
}

func assertPhase2LabNotSolved(t *testing.T, ctx context.Context, session *lab.Session) {
	t.Helper()
	results, err := session.Evaluate(ctx)
	if err != nil {
		return
	}
	if len(results) == 0 {
		t.Fatal("lab has no checker results")
	}
	for _, result := range results {
		if !result.Pass {
			return
		}
	}
	t.Fatal("fresh Phase-2 lab unexpectedly already satisfies every checker")
}
