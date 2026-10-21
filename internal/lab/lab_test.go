package lab_test

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	lpicdaily "github.com/Loe159/lpic-daily"
	"github.com/Loe159/lpic-daily/internal/lab"
	"github.com/Loe159/lpic-daily/internal/runner"
)

const (
	shellEnvironmentRepairID = "lpic1.103.1.shell-environment-repair"
	sharedDropboxID          = "lpic1.104.5.shared-dropbox"
	stuckWorkerID            = "lpic1.103.5.stuck-worker"
)

func loadBuiltinLab(t *testing.T, id string) lab.Lab {
	t.Helper()
	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	for _, authored := range labs {
		if authored.Definition.ID == id {
			return authored
		}
	}
	t.Fatalf("built-in lab %s not found", id)
	return lab.Lab{}
}

func TestLoadBuiltinShellEnvironmentRepair(t *testing.T) {
	got := loadBuiltinLab(t, shellEnvironmentRepairID)

	if len(got.Definition.ConceptIDs) != 7 {
		t.Fatalf("concepts = %d, want 7", len(got.Definition.ConceptIDs))
	}
	if len(got.Hints) != 4 {
		t.Fatalf("hints = %d, want 4", len(got.Hints))
	}
	if !strings.Contains(got.SetupScript, "/opt/lpic/shadow/bin") || !strings.Contains(got.SetupScript, "REPORT_FILE=") {
		t.Fatal("shell environment setup script is incomplete")
	}

	checks, err := got.CompileChecks()
	if err != nil {
		t.Fatalf("CompileChecks() error = %v", err)
	}
	if len(checks) != 7 {
		t.Fatalf("checks = %d, want 7", len(checks))
	}
}

func TestLoadBuiltinSharedDropbox(t *testing.T) {
	got := loadBuiltinLab(t, sharedDropboxID)

	if len(got.Definition.ConceptIDs) != 8 || !strings.Contains(strings.Join(got.Definition.ConceptIDs, "\n"), "lpic1.104.5.suid") {
		t.Fatalf("shared-dropbox concepts = %v, want all 8 permission concepts", got.Definition.ConceptIDs)
	}
	if len(got.Hints) != 4 {
		t.Fatalf("hints = %d, want 4", len(got.Hints))
	}
	for index, hint := range got.Hints {
		if hint.Level != index+1 {
			t.Fatalf("hint %d level = %d", index, hint.Level)
		}
	}
	if !strings.Contains(got.SetupScript, "getent group project") || !strings.Contains(got.SetupScript, "/srv/shared") {
		t.Fatal("setup script was not loaded")
	}
	if got.ReferenceSolutionRef == "" {
		t.Fatal("reference solution ref should remain traceable")
	}

	checks, err := got.CompileChecks()
	if err != nil {
		t.Fatalf("CompileChecks() error = %v", err)
	}
	if len(checks) != 9 {
		t.Fatalf("checks = %d, want 9", len(checks))
	}
}

func TestLoadBuiltinStuckWorker(t *testing.T) {
	got := loadBuiltinLab(t, stuckWorkerID)

	if len(got.Definition.ConceptIDs) != 7 {
		t.Fatalf("concepts = %d, want 7", len(got.Definition.ConceptIDs))
	}
	if len(got.Hints) != 4 {
		t.Fatalf("hints = %d, want 4", len(got.Hints))
	}
	if !strings.Contains(got.SetupScript, "stuck-worker") || !strings.Contains(got.SetupScript, "lpic-signal-probe") {
		t.Fatal("stuck-worker setup script is incomplete")
	}

	checks, err := got.CompileChecks()
	if err != nil {
		t.Fatalf("CompileChecks() error = %v", err)
	}
	if len(checks) != 11 {
		t.Fatalf("checks = %d, want 11", len(checks))
	}
	if !got.Definition.NeedsPersistentShell {
		t.Fatal("stuck-worker must require the persistent PTY shell")
	}
}

func TestBuiltinLabsCompileRunnerDefinitionsAndChecksAtLoadTime(t *testing.T) {
	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	for _, authored := range labs {
		if _, err := authored.RunnerDefinition(); err != nil {
			t.Fatalf("%s RunnerDefinition() error = %v", authored.Definition.ID, err)
		}
		if _, err := authored.CompileChecks(); err != nil {
			t.Fatalf("%s CompileChecks() error = %v", authored.Definition.ID, err)
		}
	}
}

func TestBuiltinLabsUseBackendSpecificCapabilityContracts(t *testing.T) {
	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	for _, authored := range labs {
		switch authored.Definition.Environment.Backend {
		case "podman":
			if authored.Definition.Environment.CapabilityProfile == "full-machine" {
				t.Fatalf("%s uses VM capability profile on Podman", authored.Definition.ID)
			}
		case "libvirt":
			if authored.Definition.Environment.CapabilityProfile != "full-machine" {
				t.Fatalf("%s libvirt profile = %q, want full-machine", authored.Definition.ID, authored.Definition.Environment.CapabilityProfile)
			}
			if len(authored.Definition.Environment.WritableGuestPaths) != 0 {
				t.Fatalf("%s libvirt writable_guest_paths = %#v, want empty", authored.Definition.ID, authored.Definition.Environment.WritableGuestPaths)
			}
		}
	}
}

func TestBuiltinHintLaddersAreCompleteAndMatchEvidencePolicy(t *testing.T) {
	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	for _, authored := range labs {
		if len(authored.Hints) != 4 {
			t.Fatalf("%s hints = %d, want 4", authored.Definition.ID, len(authored.Hints))
		}
		for index, hint := range authored.Hints {
			wantLevel := index + 1
			if hint.Level != wantLevel {
				t.Fatalf("%s hint[%d] level = %d, want %d", authored.Definition.ID, index, hint.Level, wantLevel)
			}
			switch hint.Level {
			case 1:
				if hint.EvidenceImpact != "none" && hint.EvidenceImpact != "minor" {
					t.Fatalf("%s level-1 impact = %q", authored.Definition.ID, hint.EvidenceImpact)
				}
			case 2, 3:
				if hint.EvidenceImpact != "material" {
					t.Fatalf("%s level-%d impact = %q, want material", authored.Definition.ID, hint.Level, hint.EvidenceImpact)
				}
			case 4:
				if hint.EvidenceImpact != "solution-revealed" {
					t.Fatalf("%s level-4 impact = %q, want solution-revealed", authored.Definition.ID, hint.EvidenceImpact)
				}
			}
		}
	}
}

func TestSessionRunsSetupAndStateChecks(t *testing.T) {
	authored := loadBuiltinLab(t, sharedDropboxID)
	fake := &fakeRunner{}

	session, err := lab.Start(context.Background(), authored, fake)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if !fake.started {
		t.Fatal("runner was not started")
	}
	if !strings.Contains(strings.Join(fake.exec.Argv, " "), "/usr/bin/bash -eu -c") {
		t.Fatalf("setup argv = %v", fake.exec.Argv)
	}
	if !strings.Contains(fake.exec.Argv[len(fake.exec.Argv)-1], "getent group project") {
		t.Fatal("setup script was not passed into sandbox exec")
	}

	results, err := session.Evaluate(context.Background())
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	for _, result := range results {
		if !result.Pass {
			t.Fatalf("check failed: %#v", result)
		}
	}

	if err := session.Close(context.Background()); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if !fake.destroyed {
		t.Fatal("runner was not destroyed")
	}
}

func TestSessionResetRestartsAndReplaysSetup(t *testing.T) {
	authored := loadBuiltinLab(t, sharedDropboxID)
	fake := &fakeRunner{}

	session, err := lab.Start(context.Background(), authored, fake)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer session.Close(context.Background())

	if fake.startCalls != 1 || fake.execCalls != 1 {
		t.Fatalf("initial lifecycle start=%d exec=%d, want 1/1", fake.startCalls, fake.execCalls)
	}
	if err := session.Reset(context.Background()); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}
	if fake.resetCalls != 1 {
		t.Fatalf("reset calls = %d, want 1", fake.resetCalls)
	}
	if fake.startCalls != 2 || fake.execCalls != 2 {
		t.Fatalf("reset lifecycle start=%d exec=%d, want 2/2", fake.startCalls, fake.execCalls)
	}
	if !strings.Contains(fake.exec.Argv[len(fake.exec.Argv)-1], "getent group project") {
		t.Fatal("reset did not replay the lab setup")
	}
}

func TestVMSetupNoneSkipsGuestExec(t *testing.T) {
	authored := loadBuiltinLab(t, sharedDropboxID)
	authored.Definition.ID = "lpic1.104.1.test-vm"
	authored.Definition.Environment.Backend = "libvirt"
	authored.Definition.Environment.CapabilityProfile = "full-machine"
	authored.Definition.Environment.Machine = &lab.Machine{
		Firmware: "uefi",
		ExtraDisks: []lab.MachineDisk{{
			ID:     "data",
			SizeMB: 512,
		}},
	}
	authored.Definition.Setup = lab.Setup{ExecutionScope: "none"}
	authored.SetupScript = ""

	fake := &fakeRunner{failExec: true}
	session, err := lab.Start(context.Background(), authored, fake)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer session.Close(context.Background())

	if fake.execCalls != 0 {
		t.Fatalf("VM setup unexpectedly called Exec %d time(s)", fake.execCalls)
	}
	if fake.definition.Machine == nil || len(fake.definition.Machine.ExtraDisks) != 1 {
		t.Fatalf("VM definition = %#v", fake.definition)
	}
}

func TestVMSetupSandboxUsesStructuredGuestExec(t *testing.T) {
	authored := loadBuiltinLab(t, sharedDropboxID)
	authored.Definition.ID = "lpic1.104.2.test-vm-setup"
	authored.Definition.Environment.Backend = "libvirt"
	authored.Definition.Environment.CapabilityProfile = "full-machine"
	authored.Definition.Environment.WritableGuestPaths = nil
	authored.Definition.Environment.Machine = &lab.Machine{Firmware: "uefi"}
	authored.Definition.Setup = lab.Setup{ExecutionScope: "sandbox", ScriptRef: "setup.sh"}
	authored.SetupScript = "printf ready > /run/lpic-vm-setup"

	fake := &fakeRunner{}
	session, err := lab.Start(context.Background(), authored, fake)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer session.Close(context.Background())

	if fake.execCalls != 1 {
		t.Fatalf("VM sandbox setup Exec calls = %d, want 1", fake.execCalls)
	}
	if len(fake.exec.Argv) != 4 ||
		fake.exec.Argv[0] != "/usr/bin/bash" ||
		fake.exec.Argv[1] != "-eu" ||
		fake.exec.Argv[2] != "-c" ||
		fake.exec.Argv[3] != authored.SetupScript {
		t.Fatalf("VM setup argv = %#v, want structured bash setup", fake.exec.Argv)
	}
}

func TestDestructiveSetupCannotModifyHostSentinelThroughLabOrchestration(t *testing.T) {
	sentinel := filepath.Join(t.TempDir(), "host-sentinel")
	if err := os.WriteFile(sentinel, []byte("safe"), 0o600); err != nil {
		t.Fatalf("write sentinel: %v", err)
	}

	authored := loadBuiltinLab(t, sharedDropboxID)
	authored.SetupScript = fmt.Sprintf("printf 'pwned' > %q", sentinel)
	fake := &fakeRunner{}

	session, err := lab.Start(context.Background(), authored, fake)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer session.Close(context.Background())

	got, err := os.ReadFile(sentinel)
	if err != nil {
		t.Fatalf("read sentinel: %v", err)
	}
	if string(got) != "safe" {
		t.Fatalf("host sentinel changed to %q", got)
	}
	if len(fake.exec.Argv) != 4 || fake.exec.Argv[0] != "/usr/bin/bash" {
		t.Fatalf("setup was not delegated as structured sandbox argv: %v", fake.exec.Argv)
	}
	if fake.exec.Argv[3] != authored.SetupScript {
		t.Fatalf("setup body = %q, want %q", fake.exec.Argv[3], authored.SetupScript)
	}
}

type fakeRunner struct {
	definition      runner.Definition
	exec            runner.ExecRequest
	execCalls       int
	startCalls      int
	resetCalls      int
	failExec        bool
	started         bool
	destroyed       bool
	destroyCalls    int
	destroyFailures int
}

func (fake *fakeRunner) Prepare(_ context.Context, definition runner.Definition) (runner.Instance, error) {
	fake.definition = definition
	return runner.Instance{ID: "fake-lab"}, nil
}

func (fake *fakeRunner) Start(_ context.Context, _ runner.Instance) error {
	fake.started = true
	fake.startCalls++
	return nil
}

func (fake *fakeRunner) Exec(_ context.Context, _ runner.Instance, request runner.ExecRequest) (runner.ExecResult, error) {
	fake.execCalls++
	fake.exec = request
	if fake.failExec {
		return runner.ExecResult{}, errors.New("Exec must not be called")
	}
	return runner.ExecResult{ExitCode: 0}, nil
}

func (fake *fakeRunner) Stat(_ context.Context, _ runner.Instance, path string) (runner.FileInfo, error) {
	switch path {
	case "/srv/shared":
		return runner.FileInfo{
			Path: path, Mode: 0o3770, UID: 0, GID: 2000,
			User: "root", Group: "project", IsDir: true,
		}, nil
	case "/srv/shared/team-note":
		return runner.FileInfo{
			Path: path, Mode: 0o0660, UID: 0, GID: 2000,
			User: "root", Group: "project", IsDir: false,
		}, nil
	case "/srv/shared/audit-helper":
		return runner.FileInfo{
			Path: path, Mode: 0o0755, UID: 0, GID: 0,
			User: "root", Group: "root", IsDir: false,
		}, nil
	default:
		return runner.FileInfo{}, errors.New("not found")
	}
}

func (fake *fakeRunner) ReadFile(context.Context, runner.Instance, string, int64) ([]byte, error) {
	return nil, errors.New("not implemented in fake")
}

func (fake *fakeRunner) Processes(context.Context, runner.Instance) ([]runner.Process, error) {
	return nil, nil
}

func (fake *fakeRunner) Reset(context.Context, runner.Instance) error {
	fake.resetCalls++
	return nil
}

func (fake *fakeRunner) Destroy(context.Context, runner.Instance) error {
	fake.destroyCalls++
	if fake.destroyFailures > 0 {
		fake.destroyFailures--
		return errors.New("transient destroy failure")
	}
	fake.destroyed = true
	return nil
}

func (fake *fakeRunner) Close() error {
	return nil
}

func TestStartSurfacesCleanupFailure(t *testing.T) {
	authored := loadBuiltinLab(t, sharedDropboxID)
	fake := &fakeRunner{failExec: true, destroyFailures: 1}

	_, err := lab.Start(context.Background(), authored, fake)
	if err == nil || !strings.Contains(err.Error(), "cleanup failed start") {
		t.Fatalf("Start() error = %v, want surfaced cleanup failure", err)
	}
	if fake.destroyCalls != 1 {
		t.Fatalf("destroy calls = %d, want 1", fake.destroyCalls)
	}
}

func TestFailedResetCleanupRemainsRetryable(t *testing.T) {
	authored := loadBuiltinLab(t, sharedDropboxID)
	fake := &fakeRunner{}
	session, err := lab.Start(context.Background(), authored, fake)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	fake.failExec = true
	fake.destroyFailures = 1
	err = session.Reset(context.Background())
	if err == nil || !strings.Contains(err.Error(), "cleanup failed reset") {
		t.Fatalf("Reset() error = %v, want cleanup failure", err)
	}

	fake.failExec = false
	if err := session.Close(context.Background()); err != nil {
		t.Fatalf("Close() retry error = %v", err)
	}
	if fake.destroyCalls != 2 || !fake.destroyed {
		t.Fatalf("cleanup retry destroyCalls=%d destroyed=%v, want 2/true", fake.destroyCalls, fake.destroyed)
	}
}

func TestBuiltinLabsAreOnlyAuthoredDefinitions(t *testing.T) {
	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	paths, err := fs.Glob(lpicdaily.BuiltinFS, "labs/lpic-1-v5/*/*/lab.json")
	if err != nil {
		t.Fatalf("glob authored labs: %v", err)
	}
	if len(labs) != len(paths) {
		t.Fatalf("loaded %d labs, but found %d authored JSON files", len(labs), len(paths))
	}
	for _, authored := range labs {
		if strings.Contains(authored.Definition.ID, ".standalone-") {
			t.Fatalf("generated lab returned by LoadAll: %s", authored.Definition.ID)
		}
	}
}

func TestBuiltinStorageBusArchiveRecoveryPropagatesTransport(t *testing.T) {
	authored := loadBuiltinLab(t, "lpic1.101.1.storage-bus-archive-recovery")
	definition, err := authored.RunnerDefinition()
	if err != nil {
		t.Fatalf("RunnerDefinition() error = %v", err)
	}
	if definition.Machine == nil || len(definition.Machine.ExtraDisks) != 2 {
		t.Fatalf("expected two isolated extra disks, got %#v", definition.Machine)
	}
	archive, queue := definition.Machine.ExtraDisks[0], definition.Machine.ExtraDisks[1]
	if archive.ID != "archive" || archive.Bus != runner.VirtualDiskBusSATA {
		t.Fatalf("archive disk = %#v, want SATA", archive)
	}
	if queue.ID != "queue" || queue.Bus != runner.VirtualDiskBusVirtio {
		t.Fatalf("queue disk = %#v, want VirtIO", queue)
	}
	if len(authored.Hints) != 4 || len(authored.Definition.Checks) != 3 {
		t.Fatalf("incomplete storage scenario: %d hints, %d checks", len(authored.Hints), len(authored.Definition.Checks))
	}
}

func TestBuiltinUSBAuthorizationRecoveryUsesVirtualUSBTransport(t *testing.T) {
	authored := loadBuiltinLab(t, "lpic1.101.1.usb-backup-authorization-recovery")
	definition, err := authored.RunnerDefinition()
	if err != nil {
		t.Fatalf("RunnerDefinition() error = %v", err)
	}
	if definition.Machine == nil || len(definition.Machine.ExtraDisks) != 1 {
		t.Fatalf("expected one isolated USB disk, got %#v", definition.Machine)
	}
	device := definition.Machine.ExtraDisks[0]
	if device.Bus != runner.VirtualDiskBusUSB || device.ID != "usb-backup" {
		t.Fatalf("virtual disk = %#v, want USB backup", device)
	}
	if len(authored.Definition.Checks) != 3 || len(authored.Hints) != 4 {
		t.Fatalf("incomplete USB scenario: %d checks, %d hints", len(authored.Definition.Checks), len(authored.Hints))
	}
	for _, id := range []string{"lpic1.101.1.bus-pci-et-usb", "lpic1.101.1.manipulation-des-peripheriques-usb"} {
		if !slices.Contains(authored.Definition.ConceptIDs, id) {
			t.Fatalf("concept %s missing from USB scenario", id)
		}
		found := false
		for _, check := range authored.Definition.Checks {
			if slices.Contains(check.ConceptIDs, id) {
				found = true
			}
		}
		if !found {
			t.Fatalf("concept %s has no state-based check", id)
		}
	}
}

func TestBuiltinRescueTargetRecoveryRequiresRealSystemdTargets(t *testing.T) {
	authored := loadBuiltinLab(t, "lpic1.101.3.rescue-target-service-recovery")
	definition, err := authored.RunnerDefinition()
	if err != nil {
		t.Fatalf("RunnerDefinition: %v", err)
	}
	if definition.Machine == nil || authored.Definition.Environment.Backend != "libvirt" {
		t.Fatalf("rescue needs isolated full VM, got %#v", authored.Definition.Environment)
	}
	if len(authored.Hints) != 4 || len(authored.Definition.Checks) != 3 {
		t.Fatalf("rescue scenario has %d hints / %d checks", len(authored.Hints), len(authored.Definition.Checks))
	}
	if !strings.Contains(authored.SetupScript, "IgnoreOnIsolate=yes") ||
		!strings.Contains(authored.SetupScript, "isolate rescue.target") ||
		!strings.Contains(authored.SetupScript, "is-active --quiet qemu-guest-agent.service") {
		t.Fatal("rescue setup lacks real target isolation or remote-control guard")
	}
	for _, check := range authored.Definition.Checks {
		if !slices.Contains(check.ConceptIDs, "lpic1.101.3.mode-mono-utilisateur-recuperation") {
			t.Fatalf("unmapped rescue check: %#v", check)
		}
	}
}

func TestBuiltinACPIAuditRequiresRealVMEvent(t *testing.T) {
	authored := loadBuiltinLab(t, "lpic1.101.3.acpi-power-event-audit-recovery")
	definition, err := authored.RunnerDefinition()
	if err != nil {
		t.Fatal(err)
	}
	if definition.Machine == nil || authored.Definition.Environment.Backend != "libvirt" {
		t.Fatalf("ACPI lab requires disposable VM: %#v", authored.Definition.Environment)
	}
	if len(authored.Definition.Checks) != 4 || len(authored.Hints) != 4 {
		t.Fatalf("incomplete ACPI lab: %d checks %d hints", len(authored.Definition.Checks), len(authored.Hints))
	}
	if authored.Definition.Checks[3].Type != "acpi-button-sent" {
		t.Fatal("missing independent host ACPI evidence check")
	}
	for _, check := range authored.Definition.Checks {
		if !slices.Contains(check.ConceptIDs, "lpic1.101.3.notion-d-evenements-acpi") {
			t.Fatalf("ACPI concept missing on state check: %#v", check)
		}
	}
	if !strings.Contains(authored.SetupScript, "HandlePowerKey=ignore") ||
		!strings.Contains(authored.SetupScript, "action=/usr/bin/true") ||
		authored.ReferenceSolutionRef != "reference-solution.sh" {
		t.Fatal("ACPI audit scenario lacks safe power policy, broken rule, or manual event trigger")
	}
}

func TestBuiltinLegacyInitServiceMigrationPreservesArchive(t *testing.T) {
	authored := loadBuiltinLab(t, "lpic1.101.2.legacy-init-service-migration")
	definition, err := authored.RunnerDefinition()
	if err != nil {
		t.Fatal(err)
	}
	if definition.Machine == nil || authored.Definition.Environment.Backend != "libvirt" {
		t.Fatalf("systemd integration requires actual isolated VM: %#v", authored.Definition.Environment)
	}
	if len(authored.Definition.Checks) != 3 || len(authored.Hints) != 4 {
		t.Fatalf("incomplete init migration lab: %d checks %d hints", len(authored.Definition.Checks), len(authored.Hints))
	}
	for _, check := range authored.Definition.Checks {
		if !slices.Contains(check.ConceptIDs, "lpic1.101.2.sysvinit-systemd-et-connaissance-historique-d-upstart") {
			t.Fatalf("unmapped legacy init check: %#v", check)
		}
	}
	for _, marker := range []string{"ExecStart=/etc/init/lpic-legacy-indexer.conf", "sha256sum", "start on runlevel", "### BEGIN INIT INFO", "heartbeat"} {
		if !strings.Contains(authored.SetupScript, marker) {
			t.Errorf("expected real compatibility evidence %q", marker)
		}
	}
}

func TestBuiltinGRUBConsoleRequiresTemporaryBootAndUnmodifiedConfig(t *testing.T) {
	authored := loadBuiltinLab(t, "lpic1.102.2.grub-serial-one-time-diagnostics")
	definition, err := authored.RunnerDefinition()
	if err != nil {
		t.Fatal(err)
	}
	if definition.Machine == nil || authored.Definition.Environment.Backend != "libvirt" {
		t.Fatalf("GRUB console lab needs a disposable full VM: %#v", authored.Definition.Environment)
	}
	if len(authored.Definition.Checks) != 3 || len(authored.Hints) != 4 {
		t.Fatalf("incomplete GRUB console lab: %d checks %d hints", len(authored.Definition.Checks), len(authored.Hints))
	}
	for _, cid := range []string{"lpic1.101.2.commandes-du-chargeur-de-demarrage", "lpic1.102.2.interaction-au-menu-console-grub"} {
		if !slices.Contains(authored.Definition.ConceptIDs, cid) {
			t.Fatalf("missing concept %s", cid)
		}
		mapped := false
		for _, check := range authored.Definition.Checks {
			if slices.Contains(check.ConceptIDs, cid) {
				mapped = true
			}
		}
		if !mapped {
			t.Fatalf("no state check for %s", cid)
		}
	}
	if !strings.Contains(authored.SetupScript, "GRUB_TIMEOUT=90") ||
		!strings.Contains(authored.SetupScript, "preserved-boot-config.sha256") ||
		!strings.Contains(authored.SetupScript, "grub2-mkconfig") ||
		authored.ReferenceSolutionRef != "reference-solution.sh" {
		t.Fatal("GRUB lab missing serial-menu window, immutable config evidence, or manual procedure")
	}
}

func TestBuiltinGRUBLegacyMigrationRetainsHistoricArchive(t *testing.T) {
	authored := loadBuiltinLab(t, "lpic1.102.2.grub-legacy-migration-recovery")
	definition, err := authored.RunnerDefinition()
	if err != nil {
		t.Fatal(err)
	}
	if definition.Machine == nil || authored.Definition.Environment.Backend != "libvirt" {
		t.Fatalf("legacy GRUB migration needs full VM: %#v", authored.Definition.Environment)
	}
	if len(authored.Definition.Checks) != 3 || len(authored.Hints) != 4 {
		t.Fatalf("incomplete GRUB migration: %d checks %d hints", len(authored.Definition.Checks), len(authored.Hints))
	}
	for _, check := range authored.Definition.Checks {
		if !slices.Contains(check.ConceptIDs, "lpic1.102.2.grub-legacy-versus-grub-2") {
			t.Fatalf("unmapped legacy GRUB check: %#v", check)
		}
	}
	for _, marker := range []string{"menu.lst", "root (hd0,0)", "kernel /vmlinuz-", "sha256sum", "grub2-set-default"} {
		if !strings.Contains(authored.SetupScript, marker) {
			t.Errorf("setup missing historic GRUB evidence %q", marker)
		}
	}
}

func TestBuiltinOnboardSATAControllerReenableUsesRealPCI(t *testing.T) {
	authored := loadBuiltinLab(t, "lpic1.101.1.onboard-sata-controller-reenable")
	definition, err := authored.RunnerDefinition()
	if err != nil {
		t.Fatal(err)
	}
	if definition.Machine == nil || len(definition.Machine.ExtraDisks) != 1 {
		t.Fatalf("expected isolated SATA controller and guest-only disk: %#v", definition.Machine)
	}
	if definition.Machine.ExtraDisks[0].Bus != runner.VirtualDiskBusSATA {
		t.Fatalf("missing true SATA bus: %#v", definition.Machine.ExtraDisks)
	}
	if len(authored.Definition.Checks) != 3 || len(authored.Hints) != 4 {
		t.Fatalf("incomplete PCI controller recovery: %d checks %d hints", len(authored.Definition.Checks), len(authored.Hints))
	}
	for _, check := range authored.Definition.Checks {
		if !slices.Contains(check.ConceptIDs, "lpic1.101.1.activation-desactivation-peripheriques-integres") {
			t.Fatalf("unmapped PCI controller check: %#v", check)
		}
	}
	for _, marker := range []string{"/sys/bus/pci/drivers/ahci/unbind", "readlink -f", "udevadm settle", "mkfs.ext4"} {
		if !strings.Contains(authored.SetupScript, marker) {
			t.Errorf("missing physical PCI state evidence %q", marker)
		}
	}
}

// Test the actual authored shell checker with synthetic efibootmgr outputs.
// This runs in CI without firmware, so BootOrder parsing cannot silently
// regress while the host baseline check is tested independently.
func TestBuiltinUEFIBootOrderShellCheck(t *testing.T) {
	authored := loadBuiltinLab(t, "lpic1.101.2.uefi-boot-order-recovery")
	if len(authored.Definition.Checks) < 3 {
		t.Fatalf("missing UEFI checks: %d", len(authored.Definition.Checks))
	}
	check := authored.Definition.Checks[0]
	if check.Type != "command-exit" || len(check.Argv) != 3 {
		t.Fatalf("UEFI shell check shape changed: %#v", check)
	}
	command := check.Argv[2]
	const precondition = "test -d /sys/firmware/efi/efivars"
	if !strings.Contains(command, precondition) {
		t.Fatalf("missing firmware precondition")
	}
	command = strings.Replace(command, precondition, "true", 1)
	command = "efibootmgr() { cat \"$LPIC_TEST_EFI_STATE\"; }\n" + command

	entries := "Boot0005* LPIC-primary HD(1,GPT,abc)\n" +
		"Boot0004* LPIC-rescue HD(1,GPT,abc)\n" +
		"Boot0000* Fedora\n" +
		"Boot0001  UEFI:PXE\n"
	cases := []struct {
		name     string
		order    string
		entries  string
		wantPass bool
	}{
		{name: "restored complete order", order: "0005,0004,0000,0001", entries: entries, wantPass: true},
		{name: "unrepaired rescue first", order: "0004,0005,0000,0001", entries: entries},
		{name: "other entry before primary", order: "0000,0005,0004,0001", entries: entries},
		{name: "duplicate primary", order: "0005,0004,0000,0005", entries: entries},
		{name: "missing rescue entry", order: "0005,0004,0000,0001", entries: strings.Replace(entries, "Boot0004* LPIC-rescue HD(1,GPT,abc)\n", "", 1)},
		{name: "no preserved entries", order: "0005,0004", entries: entries},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			statePath := filepath.Join(t.TempDir(), "efibootmgr.txt")
			state := "BootCurrent: 0000\nBootOrder: " + tc.order + "\n" + tc.entries
			if err := os.WriteFile(statePath, []byte(state), 0o600); err != nil {
				t.Fatal(err)
			}
			process := exec.Command("/usr/bin/bash", "-c", command)
			process.Env = append(os.Environ(), "LPIC_TEST_EFI_STATE="+statePath)
			output, err := process.CombinedOutput()
			if (err == nil) != tc.wantPass {
				t.Fatalf("UEFI check pass=%v, want %v: err=%v output=%q", err == nil, tc.wantPass, err, output)
			}
		})
	}
}

func TestEnvironmentBoundaryRepairChecksAreBounded(t *testing.T) {
	authored := loadBuiltinLab(t, "lpic1.103.1.environment-boundary-repair")
	if len(authored.Definition.Checks) != 5 {
		t.Fatalf("checks = %d, want 5", len(authored.Definition.Checks))
	}
	if !authored.Definition.NeedsPersistentShell {
		t.Fatal("environment grading must observe the persistent learner shell")
	}
	if !strings.Contains(authored.SetupScript, "PROMPT_COMMAND=__lpic_capture_environment") {
		t.Fatal("shell startup must capture the live state before the prompt")
	}
	for i, check := range authored.Definition.Checks {
		if check.Type != "command-exit" || check.TimeoutSeconds != 5 ||
			check.BatchID != "environment-inspection" || check.FailureBit != 1<<i {
			t.Fatalf("check %d must use one grouped, bounded command: %#v", i+1, check)
		}
		want := []string{"/usr/bin/timeout", "-k", "1s", "3s", "/usr/bin/bash", "--noprofile", "--norc", "-c"}
		if len(check.Argv) != len(want)+1 || !slices.Equal(check.Argv[:len(want)], want) {
			t.Fatalf("check %d must use a bounded noninteractive shell: %#v", i+1, check.Argv)
		}
		if !strings.Contains(check.Argv[len(want)], "/run/lpic/.environment-check-state") ||
			strings.Contains(check.Argv[len(want)], "/root/.bashrc") {
			t.Fatalf("check %d must observe a live shell snapshot, not source .bashrc: %#v", i+1, check.Argv)
		}
		if i > 0 && !slices.Equal(authored.Definition.Checks[0].Argv, check.Argv) {
			t.Fatalf("check %d duplicates a different command", i+1)
		}
	}
}

// The active-shell probe must accept commands entered in the learner's shell,
// not require hidden edits to .bashrc or a fresh login.
func TestEnvironmentBoundaryShellExportsWithoutBashrcEdits(t *testing.T) {
	authored := loadBuiltinLab(t, "lpic1.103.1.environment-boundary-repair")
	tmp := t.TempDir()
	lpic := filepath.Join(tmp, "lpic")
	setup := strings.ReplaceAll(authored.SetupScript, "/run/lpic", lpic)
	command := exec.Command("/usr/bin/bash", "-c", setup)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("setup shell: %v: %s", err, output)
	}
	runShell := func(source string, want string) {
		t.Helper()
		command := exec.Command("/usr/bin/bash", "--noprofile", "--norc", "-c", source)
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("shell probe: %v: %s", err, output)
		}
		got, err := os.ReadFile(filepath.Join(lpic, ".environment-check-state"))
		if err != nil {
			t.Fatal(err)
		}
		if strings.TrimSpace(string(got)) != want {
			t.Fatalf("observed state = %q, want %q", got, want)
		}
	}
	source := ". " + filepath.Join(lpic, ".lpic-shell-rc") + "; "
	runShell(source+"__lpic_capture_environment", "30")
	runShell(source+"export SERVICE_MODE=production; __lpic_capture_environment", "26")
	runShell(source+"export SERVICE_MODE=production; export -n LOCAL_NOTE; unset LEGACY_TOKEN; "+
		"printf '%s|%s\\n' \"$SERVICE_MODE\" \"$LOCAL_NOTE\" > "+filepath.Join(lpic, "env-summary")+
		"; __lpic_capture_environment", "0")
	if _, err := os.Stat(filepath.Join(tmp, "home", ".bashrc")); !os.IsNotExist(err) {
		t.Fatalf("the exercise should not create an unrelated .bashrc: %v", err)
	}
}

// All built-in scenarios must grade state without opening a second interactive
// Bash. Such shells can hang without a TTY and block :check for a full minute.
func TestBuiltinCheckCommandsNeverLaunchInteractiveShell(t *testing.T) {
	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatal(err)
	}
	for _, authored := range labs {
		for i, check := range authored.Definition.Checks {
			if check.Type != "command-exit" {
				continue
			}
			for _, arg := range check.Argv {
				switch arg {
				case "-i", "-ic", "--interactive", "--rcfile":
					t.Errorf("%s check %d launches an interactive Bash: %v", authored.Definition.ID, i+1, check.Argv)
				}
			}
		}
	}
}
