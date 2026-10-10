//go:build integration

package podman_test

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"os"
	"testing"
	"time"

	lpicdaily "github.com/Loe159/lpic-daily"
	"github.com/Loe159/lpic-daily/internal/lab"
	"github.com/Loe159/lpic-daily/internal/runner"
	podmanrunner "github.com/Loe159/lpic-daily/internal/runner/podman"
)

var phase1ReferenceSolutions = map[string]string{
	"lpic1.103.1.child-environment-handoff": "labs/lpic-1-v5/103.1/child-environment-handoff/reference-solution.sh",
	"lpic1.103.1.shell-environment-repair":     "labs/lpic-1-v5/103.1/shell-environment-repair/reference-solution.sh",
	"lpic1.103.1.transfer-shell-handoff":       "labs/lpic-1-v5/103.1/transfer-shell-handoff/reference-solution.sh",
	"lpic1.103.1.environment-boundary-repair":  "labs/lpic-1-v5/103.1/environment-boundary-repair/reference-solution.sh",
	"lpic1.103.1.external-command-recovery":    "labs/lpic-1-v5/103.1/external-command-recovery/reference-solution.sh",
	"lpic1.103.1.history-handoff-recovery":     "labs/lpic-1-v5/103.1/history-handoff-recovery/reference-solution.sh",
	"lpic1.103.2.incident-report-rebuild":      "labs/lpic-1-v5/103.2/incident-report-rebuild/reference-solution.sh",
	"lpic1.103.2.compressed-manifest-recovery": "labs/lpic-1-v5/103.2/compressed-manifest-recovery/reference-solution.sh",
	"lpic1.103.3.release-tree-recovery":        "labs/lpic-1-v5/103.3/release-tree-recovery/reference-solution.sh",
	"lpic1.103.3.backup-bundle-recovery":       "labs/lpic-1-v5/103.3/backup-bundle-recovery/reference-solution.sh",
	"lpic1.103.4.batch-stream-repair":          "labs/lpic-1-v5/103.4/batch-stream-repair/reference-solution.sh",
	"lpic1.103.4.bulk-argument-dispatch":       "labs/lpic-1-v5/103.4/bulk-argument-dispatch/reference-solution.sh",
	"lpic1.103.7.auth-log-filter-repair":       "labs/lpic-1-v5/103.7/auth-log-filter-repair/reference-solution.sh",
	"lpic1.103.8.minimal-editor-repair":        "labs/lpic-1-v5/103.8/minimal-editor-repair/reference-solution.sh",
	"lpic1.103.6.priority-incident":            "labs/lpic-1-v5/103.6/priority-incident/reference-solution.sh",
	"lpic1.103.5.stuck-worker":                 "labs/lpic-1-v5/103.5/stuck-worker/reference-solution.sh",
	"lpic1.103.5.transfer-operator-session":    "labs/lpic-1-v5/103.5/transfer-operator-session/reference-solution.sh",
	"lpic1.104.7.fhs-search-recovery":          "labs/lpic-1-v5/104.7/fhs-search-recovery/reference-solution.sh",
	"lpic1.104.6.release-link-repair":          "labs/lpic-1-v5/104.6/release-link-repair/reference-solution.sh",
	"lpic1.104.5.shared-dropbox":               "labs/lpic-1-v5/104.5/shared-dropbox/reference-solution.sh",
	"lpic1.104.5.transfer-team-share-audit":    "labs/lpic-1-v5/104.5/transfer-team-share-audit/reference-solution.sh",
}

func TestPhase1BuiltInLabsConformOnRootlessPodman(t *testing.T) {
	if os.Getenv("LPIC_DAILY_RUN_PODMAN_INTEGRATION") != "1" {
		t.Skip("set LPIC_DAILY_RUN_PODMAN_INTEGRATION=1 to run real rootless Podman conformance tests")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	backend, err := podmanrunner.Open(ctx, "")
	if err != nil {
		t.Fatalf("Open(rootless Podman) error = %v", err)
	}
	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}

	seen := make(map[string]bool, len(phase1ReferenceSolutions))
	for _, authored := range labs {
		solutionPath, wanted := phase1ReferenceSolutions[authored.Definition.ID]
		if !wanted {
			continue
		}
		seen[authored.Definition.ID] = true
		authored := authored
		t.Run(authored.Definition.ID, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
			defer cancel()
			solution, err := fs.ReadFile(lpicdaily.BuiltinFS, solutionPath)
			if err != nil {
				t.Fatalf("read reference solution: %v", err)
			}

			session, err := lab.Start(ctx, authored, backend)
			if err != nil {
				t.Fatalf("Start() error = %v", err)
			}
			defer func() {
				cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cleanupCancel()
				if err := session.Close(cleanupCtx); err != nil {
					t.Errorf("Close() error = %v", err)
				}
			}()

			solve := func() {
				t.Helper()
				switch authored.Definition.ID {
				case "lpic1.103.5.stuck-worker":
					runStuckWorkerPTY(t, ctx, backend, session.Instance)
					return
				case "lpic1.103.5.transfer-operator-session":
					runTransferOperatorPTY(t, ctx, backend, session.Instance)
					return
				}
				runReferenceSolution(t, ctx, backend, session.Instance, solution)
			}

			if authored.Definition.ID == "lpic1.103.1.shell-environment-repair" {
				assertShellLabHelpersImmutable(t, ctx, backend, session.Instance)
			}
			assertLabNotSolved(t, ctx, session)
			solve()
			assertLabSolved(t, ctx, session)

			if err := session.Reset(ctx); err != nil {
				t.Fatalf("Reset() error = %v", err)
			}
			assertLabNotSolved(t, ctx, session)
			solve()
			assertLabSolved(t, ctx, session)

			if authored.Definition.ID == "lpic1.103.5.stuck-worker" {
				if err := session.Reset(ctx); err != nil {
					t.Fatalf("Reset() before reference-solution consistency check error = %v", err)
				}
				assertLabNotSolved(t, ctx, session)
				runReferenceSolution(t, ctx, backend, session.Instance, solution)
				assertLabSolved(t, ctx, session)
			}
		})
	}

	for id := range phase1ReferenceSolutions {
		if !seen[id] {
			t.Errorf("Phase-1 conformance lab %s was not loaded", id)
		}
	}
}

func runStuckWorkerPTY(
	t *testing.T,
	ctx context.Context,
	backend runner.Runner,
	instance runner.Instance,
) {
	t.Helper()

	input, writer := io.Pipe()
	writeDone := make(chan error, 1)
	go func() {
		write := func(value string, delay time.Duration) error {
			if _, err := io.WriteString(writer, value); err != nil {
				return err
			}
			time.Sleep(delay)
			return nil
		}

		if err := write("ps -o pid=,ppid=,stat=,comm=,args= -p \"$(cat /run/lpic/healthy-worker.pid),$(cat /run/lpic/stuck-worker.pid)\" > /run/lpic/process-inspection\n", 100*time.Millisecond); err != nil {
			writeDone <- err
			return
		}
		if err := write("kill -TERM \"$(cat /run/lpic/stuck-worker.pid)\"\n", 100*time.Millisecond); err != nil {
			writeDone <- err
			return
		}
		if err := write("/usr/local/bin/lpic-signal-probe\n", 300*time.Millisecond); err != nil {
			writeDone <- err
			return
		}
		if _, err := writer.Write([]byte{0x1a}); err != nil {
			writeDone <- err
			return
		}
		time.Sleep(200 * time.Millisecond)
		if err := write("jobs -s | grep -q lpic-signal-probe\n", 100*time.Millisecond); err != nil {
			writeDone <- err
			return
		}
		if err := write("bg\n", 200*time.Millisecond); err != nil {
			writeDone <- err
			return
		}
		if err := write("jobs -r | grep -q lpic-signal-probe\n", 100*time.Millisecond); err != nil {
			writeDone <- err
			return
		}
		if err := write("kill -TERM %1\n", 200*time.Millisecond); err != nil {
			writeDone <- err
			return
		}
		if err := write("wait %1 2>/dev/null || true\n", 100*time.Millisecond); err != nil {
			writeDone <- err
			return
		}
		if err := write("nohup bash -c 'exec -a resilient-worker sleep infinity' >/run/lpic/resilient.log 2>&1 &\n", 100*time.Millisecond); err != nil {
			writeDone <- err
			return
		}
		if err := write("tmux new-session -d -s ops 'sleep infinity'\n", 100*time.Millisecond); err != nil {
			writeDone <- err
			return
		}
		if err := write("exit\n", 0); err != nil {
			writeDone <- err
			return
		}
		writeDone <- writer.Close()
	}()

	var output bytes.Buffer
	result, err := backend.Exec(ctx, instance, runner.ExecRequest{
		Argv:        []string{"/usr/bin/bash", "--noprofile", "--norc", "-i"},
		Stdin:       input,
		Stdout:      &output,
		Stderr:      &output,
		TTY:         true,
		InitialSize: runner.TerminalSize{Width: 100, Height: 30},
	})
	if err != nil {
		_ = input.Close()
		t.Fatalf("PTY job-control exec error = %v; output=%q", err, output.String())
	}
	if writeErr := <-writeDone; writeErr != nil {
		t.Fatalf("PTY input error = %v; output=%q", writeErr, output.String())
	}
	if result.ExitCode != 0 {
		t.Fatalf("PTY job-control exit = %d, want 0; output=%q", result.ExitCode, output.String())
	}

	for path, want := range map[string]string{
		"/run/lpic/probe-stopped":    "stopped",
		"/run/lpic/probe-background": "background",
	} {
		content, err := backend.ReadFile(ctx, instance, path, 64)
		if err != nil {
			t.Fatalf("read PTY marker %s: %v; output=%q", path, err, output.String())
		}
		if string(bytes.TrimSpace(content)) != want {
			t.Fatalf("PTY marker %s = %q, want %q; output=%q", path, content, want, output.String())
		}
	}
}

func runTransferOperatorPTY(
	t *testing.T,
	ctx context.Context,
	backend runner.Runner,
	instance runner.Instance,
) {
	t.Helper()

	input, writer := io.Pipe()
	writeDone := make(chan error, 1)
	go func() {
		write := func(value string, delay time.Duration) error {
			if _, err := io.WriteString(writer, value); err != nil {
				return err
			}
			time.Sleep(delay)
			return nil
		}

		commands := []struct {
			value string
			delay time.Duration
		}{
			{"ps -o pid=,ppid=,stat=,comm=,args= -p \"$(cat /run/lpic/reload-worker.pid)\" > /run/lpic/maintenance-inspection\n", 100 * time.Millisecond},
			{"kill -USR1 \"$(pgrep -f '^reload-worker' | head -n1)\"\n", 100 * time.Millisecond},
			{"/usr/local/bin/lpic-maintenance-job-probe\n", 300 * time.Millisecond},
		}
		for _, command := range commands {
			if err := write(command.value, command.delay); err != nil {
				writeDone <- err
				return
			}
		}
		if _, err := writer.Write([]byte{0x1a}); err != nil {
			writeDone <- err
			return
		}
		time.Sleep(200 * time.Millisecond)
		for _, command := range []struct {
			value string
			delay time.Duration
		}{
			{"jobs -s | grep -q lpic-maintenance-job-probe\n", 100 * time.Millisecond},
			{"bg\n", 200 * time.Millisecond},
			{"jobs -r | grep -q lpic-maintenance-job-probe\n", 100 * time.Millisecond},
			{"kill -TERM %1\n", 200 * time.Millisecond},
			{"wait %1 2>/dev/null || true\n", 100 * time.Millisecond},
			{"bash -c 'exec -a batch-worker sleep infinity' &\n", 100 * time.Millisecond},
			{"nohup bash -c 'exec -a handoff-daemon sleep infinity' >/run/lpic/handoff-daemon.log 2>&1 &\n", 100 * time.Millisecond},
			{"tmux new-session -d -s maintenance-ops 'sleep infinity'\n", 100 * time.Millisecond},
			{"exit\n", 0},
		} {
			if err := write(command.value, command.delay); err != nil {
				writeDone <- err
				return
			}
		}
		writeDone <- writer.Close()
	}()

	var output bytes.Buffer
	result, err := backend.Exec(ctx, instance, runner.ExecRequest{
		Argv:        []string{"/usr/bin/bash", "--noprofile", "--norc", "-i"},
		Stdin:       input,
		Stdout:      &output,
		Stderr:      &output,
		TTY:         true,
		InitialSize: runner.TerminalSize{Width: 100, Height: 30},
	})
	if err != nil {
		_ = input.Close()
		t.Fatalf("transfer PTY job-control exec error = %v; output=%q", err, output.String())
	}
	if writeErr := <-writeDone; writeErr != nil {
		t.Fatalf("transfer PTY input error = %v; output=%q", writeErr, output.String())
	}
	if result.ExitCode != 0 {
		t.Fatalf("transfer PTY job-control exit = %d, want 0; output=%q", result.ExitCode, output.String())
	}
}

func assertShellLabHelpersImmutable(
	t *testing.T,
	ctx context.Context,
	backend runner.Runner,
	instance runner.Instance,
) {
	t.Helper()

	result, err := backend.Exec(ctx, instance, runner.ExecRequest{
		Argv: []string{
			"/usr/bin/bash",
			"-c",
			"printf '#!/usr/bin/env bash\\nexit 0\\n' > /opt/lpic/approved/bin/report-status",
		},
	})
	if err != nil {
		t.Fatalf("immutable helper probe error = %v", err)
	}
	if result.ExitCode == 0 {
		t.Fatal("shell lab helper executable was writable")
	}
}

func runReferenceSolution(
	t *testing.T,
	ctx context.Context,
	backend runner.Runner,
	instance runner.Instance,
	script []byte,
) {
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

func assertLabSolved(t *testing.T, ctx context.Context, session *lab.Session) {
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

func assertLabNotSolved(t *testing.T, ctx context.Context, session *lab.Session) {
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
	t.Fatal("fresh/reset lab unexpectedly already satisfies every checker")
}
