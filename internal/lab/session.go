package lab

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Loe159/lpic-daily/internal/checker"
	"github.com/Loe159/lpic-daily/internal/runner"
)

type Session struct {
	Lab      Lab
	Runner   runner.Runner
	Instance runner.Instance
	closed   bool
}

func Start(ctx context.Context, authored Lab, backend runner.Runner) (*Session, error) {
	if backend == nil {
		return nil, errors.New("runner is required")
	}

	definition, err := authored.RunnerDefinition()
	if err != nil {
		return nil, fmt.Errorf("compile runner definition: %w", err)
	}
	instance, err := backend.Prepare(ctx, definition)
	if err != nil {
		return nil, fmt.Errorf("prepare lab %s: %w", authored.Definition.ID, err)
	}

	cleanup := func() {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		_ = backend.Destroy(cleanupCtx, instance)
	}

	if err := startAndSetup(ctx, authored, backend, instance); err != nil {
		cleanup()
		return nil, err
	}

	return &Session{
		Lab:      authored,
		Runner:   backend,
		Instance: instance,
	}, nil
}

func startAndSetup(ctx context.Context, authored Lab, backend runner.Runner, instance runner.Instance) error {
	if err := backend.Start(ctx, instance); err != nil {
		return fmt.Errorf("start lab %s: %w", authored.Definition.ID, err)
	}

	switch authored.Definition.Setup.ExecutionScope {
	case "sandbox":
		result, err := backend.Exec(ctx, instance, runner.ExecRequest{
			Argv: []string{"/usr/bin/bash", "-eu", "-c", authored.SetupScript},
		})
		if err != nil {
			return fmt.Errorf("run setup for %s: %w", authored.Definition.ID, err)
		}
		if result.ExitCode != 0 {
			return fmt.Errorf("setup for %s exited with code %d", authored.Definition.ID, result.ExitCode)
		}
	case "none":
		// The trusted VM image + disposable disks are the complete initial state.
	default:
		return fmt.Errorf(
			"unsupported setup execution scope %q for %s",
			authored.Definition.Setup.ExecutionScope,
			authored.Definition.ID,
		)
	}
	return nil
}

func (session *Session) Reset(ctx context.Context) error {
	if session == nil || session.Runner == nil {
		return errors.New("session is not initialized")
	}
	if session.closed {
		return errors.New("session is closed")
	}
	if session.Lab.Definition.ResetPolicy != "disposable" {
		return fmt.Errorf("unsupported reset policy %q", session.Lab.Definition.ResetPolicy)
	}

	if err := session.Runner.Reset(ctx, session.Instance); err != nil {
		return fmt.Errorf("reset lab %s: %w", session.Lab.Definition.ID, err)
	}
	if err := startAndSetup(ctx, session.Lab, session.Runner, session.Instance); err != nil {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		cleanupErr := session.Runner.Destroy(cleanupCtx, session.Instance)
		session.closed = true
		if cleanupErr != nil {
			return errors.Join(
				fmt.Errorf("restore lab %s after reset: %w", session.Lab.Definition.ID, err),
				fmt.Errorf("cleanup failed reset: %w", cleanupErr),
			)
		}
		return fmt.Errorf("restore lab %s after reset: %w", session.Lab.Definition.ID, err)
	}
	return nil
}

func (session *Session) Evaluate(ctx context.Context) ([]checker.Result, error) {
	if session == nil || session.Runner == nil {
		return nil, errors.New("session is not initialized")
	}
	if session.closed {
		return nil, errors.New("session is closed")
	}

	checks, err := session.Lab.CompileChecks()
	if err != nil {
		return nil, err
	}
	return checker.EvaluateAll(ctx, session.Runner, session.Instance, checks)
}

func (session *Session) Close(ctx context.Context) error {
	if session == nil || session.Runner == nil {
		return errors.New("session is not initialized")
	}
	if session.closed {
		return nil
	}
	if err := session.Runner.Destroy(ctx, session.Instance); err != nil {
		return err
	}
	session.closed = true
	return nil
}
