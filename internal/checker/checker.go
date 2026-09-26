package checker

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"

	"github.com/Loe159/lpic-daily/internal/runner"
)

type Probe interface {
	Stat(context.Context, runner.Instance, string) (runner.FileInfo, error)
	ReadFile(context.Context, runner.Instance, string, int64) ([]byte, error)
	Processes(context.Context, runner.Instance) ([]runner.Process, error)
}

type Check interface {
	ID() string
	Evaluate(context.Context, Probe, runner.Instance) Result
}

type Result struct {
	CheckID string
	Pass    bool
	Detail  string
	Err     error
}

type FileExists struct {
	CheckID string
	Path    string
}

func (check FileExists) ID() string {
	return check.CheckID
}

func (check FileExists) Evaluate(ctx context.Context, probe Probe, instance runner.Instance) Result {
	_, err := probe.Stat(ctx, instance, check.Path)
	if err != nil {
		return Result{CheckID: check.ID(), Pass: false, Detail: "path missing or inaccessible", Err: err}
	}
	return Result{CheckID: check.ID(), Pass: true, Detail: "path exists"}
}

type FileMode struct {
	CheckID string
	Path    string
	Mode    uint32
}

func (check FileMode) ID() string {
	return check.CheckID
}

func (check FileMode) Evaluate(ctx context.Context, probe Probe, instance runner.Instance) Result {
	info, err := probe.Stat(ctx, instance, check.Path)
	if err != nil {
		return Result{CheckID: check.ID(), Err: err}
	}
	got := info.Mode & 0o7777
	want := check.Mode & 0o7777
	if got != want {
		return Result{
			CheckID: check.ID(),
			Pass:    false,
			Detail:  fmt.Sprintf("mode=%04o, expected=%04o", got, want),
		}
	}
	return Result{CheckID: check.ID(), Pass: true, Detail: fmt.Sprintf("mode=%04o", got)}
}

type FileOwner struct {
	CheckID string
	Path    string
	User    string
	Group   string
}

func (check FileOwner) ID() string {
	return check.CheckID
}

func (check FileOwner) Evaluate(ctx context.Context, probe Probe, instance runner.Instance) Result {
	info, err := probe.Stat(ctx, instance, check.Path)
	if err != nil {
		return Result{CheckID: check.ID(), Err: err}
	}
	if info.User != check.User || info.Group != check.Group {
		return Result{
			CheckID: check.ID(),
			Pass:    false,
			Detail:  fmt.Sprintf("owner=%s:%s (%d:%d), expected=%s:%s", info.User, info.Group, info.UID, info.GID, check.User, check.Group),
		}
	}
	return Result{
		CheckID: check.ID(),
		Pass:    true,
		Detail:  fmt.Sprintf("owner=%s:%s (%d:%d)", info.User, info.Group, info.UID, info.GID),
	}
}

type FileContentRegex struct {
	CheckID  string
	Path     string
	Pattern  string
	MaxBytes int64
}

func (check FileContentRegex) ID() string {
	return check.CheckID
}

func (check FileContentRegex) Evaluate(ctx context.Context, probe Probe, instance runner.Instance) Result {
	limit := check.MaxBytes
	if limit <= 0 {
		limit = 1 << 20
	}
	content, err := probe.ReadFile(ctx, instance, check.Path, limit)
	if err != nil {
		return Result{CheckID: check.ID(), Err: err}
	}
	pattern, err := regexp.Compile(check.Pattern)
	if err != nil {
		return Result{CheckID: check.ID(), Err: fmt.Errorf("invalid checker regex: %w", err)}
	}
	if !pattern.Match(content) {
		return Result{CheckID: check.ID(), Pass: false, Detail: "file content does not match"}
	}
	return Result{CheckID: check.ID(), Pass: true, Detail: "file content matches"}
}

type ProcessState struct {
	CheckID string
	Match   string
	Present bool
}

func (check ProcessState) ID() string {
	return check.CheckID
}

func (check ProcessState) Evaluate(ctx context.Context, probe Probe, instance runner.Instance) Result {
	processes, err := probe.Processes(ctx, instance)
	if err != nil {
		return Result{CheckID: check.ID(), Err: err}
	}
	found := false
	for _, process := range processes {
		if process.Command == check.Match || slices.Contains(process.Args, check.Match) {
			found = true
			break
		}
	}
	if found != check.Present {
		state := "absent"
		if check.Present {
			state = "present"
		}
		return Result{CheckID: check.ID(), Pass: false, Detail: fmt.Sprintf("process %q expected %s", check.Match, state)}
	}
	return Result{CheckID: check.ID(), Pass: true, Detail: "process state matches"}
}

func EvaluateAll(ctx context.Context, probe Probe, instance runner.Instance, checks []Check) ([]Result, error) {
	if probe == nil {
		return nil, errors.New("probe is required")
	}

	results := make([]Result, 0, len(checks))
	var errs []error
	for _, check := range checks {
		if check == nil {
			errs = append(errs, errors.New("nil check"))
			continue
		}
		result := check.Evaluate(ctx, probe, instance)
		results = append(results, result)
		if result.Err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", result.CheckID, result.Err))
		}
	}
	return results, errors.Join(errs...)
}
