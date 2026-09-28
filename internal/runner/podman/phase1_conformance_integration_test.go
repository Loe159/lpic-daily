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
	"lpic1.103.1.shell-environment-repair": "labs/lpic-1-v5/103.1/shell-environment-repair/reference-solution.sh",
	"lpic1.103.5.stuck-worker":             "labs/lpic-1-v5/103.5/stuck-worker/reference-solution.sh",
	"lpic1.104.5.shared-dropbox":           "labs/lpic-1-v5/104.5/shared-dropbox/reference-solution.sh",
}

func TestPhase1BuiltInLabsConformOnRootlessPodman(t *testing.T) {
	if os.Getenv("LPIC_DAILY_RUN_PODMAN_INTEGRATION") != "1" {
		t.Skip("set LPIC_DAILY_RUN_PODMAN_INTEGRATION=1 to run real rootless Podman conformance tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
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
				if authored.Definition.ID == "lpic1.103.5.stuck-worker" {
					runStuckWorkerPTY(t, ctx, backend, session.Instance)
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
