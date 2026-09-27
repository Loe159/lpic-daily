//go:build integration

package podman_test

import (
	"context"
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
				cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cleanupCancel()
				if err := session.Close(cleanupCtx); err != nil {
					t.Errorf("Close() error = %v", err)
				}
			}()

			assertLabNotSolved(t, ctx, session)
			runReferenceSolution(t, ctx, backend, session.Instance, solution)
			assertLabSolved(t, ctx, session)

			if err := session.Reset(ctx); err != nil {
				t.Fatalf("Reset() error = %v", err)
			}
			assertLabNotSolved(t, ctx, session)
			runReferenceSolution(t, ctx, backend, session.Instance, solution)
			assertLabSolved(t, ctx, session)
		})
	}

	for id := range phase1ReferenceSolutions {
		if !seen[id] {
			t.Errorf("Phase-1 conformance lab %s was not loaded", id)
		}
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
