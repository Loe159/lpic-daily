//go:build integration

package libvirt

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	lpicdaily "github.com/Loe159/lpic-daily"
	"github.com/Loe159/lpic-daily/internal/appstate"
	"github.com/Loe159/lpic-daily/internal/lab"
	"github.com/Loe159/lpic-daily/internal/runner"
	"github.com/muesli/cancelreader"
)

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
	image, err := catalog.Resolve("fedora-44-x86_64-v1", imageRoot)
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
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cleanupCancel()
	defer func() {
		if err := backend.Close(cleanupCtx); err != nil {
			t.Errorf("backend Close() error = %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	definition := runner.Definition{
		LabID:             "integration.kvm.peer-a",
		ImageRef:          "fedora-44-x86_64-v1",
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

	scenario, err := backend.PrepareScenario(ctx, []runner.Definition{definition, peer})
	if err != nil {
		t.Fatalf("PrepareScenario() error = %v", err)
	}
	scenarioOpen := true
	defer func() {
		if scenarioOpen {
			_ = backend.DestroyScenario(cleanupCtx, scenario)
		}
	}()
	for _, instance := range scenario.Instances {
		if err := backend.Start(ctx, instance); err != nil {
			t.Fatalf("Start(%s) error = %v", instance.ID, err)
		}
	}

	firstIP := integrationGuestIPv4(t, ctx, backend, scenario.Instances[0])
	ping := runner.ExecRequest{
		Argv: []string{"/usr/bin/ping", "-c", "1", "-W", "3", firstIP},
	}
	result, err := backend.Exec(ctx, scenario.Instances[1], ping)
	if err != nil {
		t.Fatalf("peer ping error = %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("peer ping exit = %d, want 0", result.ExitCode)
	}

	result, err = backend.Exec(ctx, scenario.Instances[0], runner.ExecRequest{
		Argv: []string{"/usr/bin/ping", "-c", "1", "-W", "2", "1.1.1.1"},
	})
	if err != nil {
		t.Fatalf("isolated-network public egress probe error = %v", err)
	}
	if result.ExitCode == 0 {
		t.Fatal("isolated VM unexpectedly reached a public Internet address")
	}

	// Attempt writes at host/base-looking paths from inside the guest. These must
	// stay inside the VM; the host sentinel and immutable base are checked below.
	var guestOutput bytes.Buffer
	result, err = backend.Exec(ctx, scenario.Instances[0], runner.ExecRequest{
		Argv: []string{"/usr/bin/sh", "-c", "rm -f -- " + shellQuote(sentinelPath) + "; printf guest-write > /root/lpic-daily-isolation-probe"},
		Stdout: &guestOutput,
		Stderr: &guestOutput,
	})
	if err != nil || result.ExitCode != 0 {
		t.Fatalf("guest destructive probe = exit %d, err %v, output %q", result.ExitCode, err, guestOutput.String())
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
	var consoleOutput bytes.Buffer
	go func() {
		consoleDone <- backend.OpenConsole(consoleCtx, consoleInstance, runner.ConsoleRequest{
			Stdin:  cancelableInput,
			Stdout: &consoleOutput,
		})
	}()

	select {
	case err := <-consoleDone:
		t.Fatalf("serial console closed before cancellation: %v", err)
	case <-time.After(500 * time.Millisecond):
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
	var stdout bytes.Buffer
	result, err := backend.Exec(ctx, instance, runner.ExecRequest{
		Argv: []string{"/usr/bin/sh", "-c", "ip -4 -o addr show scope global | awk '{split($4,a,\"/\"); print a[1]; exit}'"},
		Stdout: &stdout,
	})
	if err != nil {
		t.Fatalf("read guest IPv4: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("read guest IPv4 exit = %d", result.ExitCode)
	}
	value := strings.TrimSpace(stdout.String())
	if value == "" || strings.ContainsAny(value, " \t\r\n") {
		t.Fatalf("invalid guest IPv4 %q", value)
	}
	return value
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
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cleanupCancel()
	defer func() {
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
		"lpic1.102.2.grub-kernel-parameter":  "labs/lpic-1-v5/102.2/grub-kernel-parameter/reference-solution.sh",
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
	result, err := backend.Exec(ctx, instance, runner.ExecRequest{
		Argv: []string{"/usr/bin/bash", "-eu", "-c", string(script)},
	})
	if err != nil {
		t.Fatalf("reference solution exec error = %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("reference solution exit = %d, want 0", result.ExitCode)
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
