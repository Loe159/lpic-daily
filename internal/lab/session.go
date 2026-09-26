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

	if err := backend.Start(ctx, instance); err != nil {
		cleanup()
		return nil, fmt.Errorf("start lab %s: %w", authored.Definition.ID, err)
	}

	result, err := backend.Exec(ctx, instance, runner.ExecRequest{
		Argv: []string{"/usr/bin/bash", "-eu", "-c", authored.SetupScript},
	})
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("run setup for %s: %w", authored.Definition.ID, err)
	}
	if result.ExitCode != 0 {
		cleanup()
		return nil, fmt.Errorf("setup for %s exited with code %d", authored.Definition.ID, result.ExitCode)
	}

	return &Session{
		Lab:      authored,
		Runner:   backend,
		Instance: instance,
	}, nil
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
