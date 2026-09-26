package podman

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/Loe159/lpic-daily/internal/runner"
)

const compatBase = "/v1.40"

type execCreateRequest struct {
	AttachStderr bool     `json:"AttachStderr"`
	AttachStdin  bool     `json:"AttachStdin"`
	AttachStdout bool     `json:"AttachStdout"`
	Cmd          []string `json:"Cmd"`
	Env          []string `json:"Env,omitempty"`
	Privileged   bool     `json:"Privileged"`
	Tty          bool     `json:"Tty"`
	WorkingDir   string   `json:"WorkingDir,omitempty"`
}

type execCreateResponse struct {
	ID string `json:"Id"`
}

type execStartRequest struct {
	Detach bool `json:"Detach"`
	Tty    bool `json:"Tty"`
}

type execInspectResponse struct {
	ID       string `json:"ID"`
	Running  bool   `json:"Running"`
	ExitCode int    `json:"ExitCode"`
}

func (backend *Backend) Exec(
	ctx context.Context,
	instance runner.Instance,
	request runner.ExecRequest,
) (runner.ExecResult, error) {
	if instance.ID == "" {
		return runner.ExecResult{}, errors.New("instance ID is required")
	}
	if err := request.Validate(); err != nil {
		return runner.ExecResult{}, fmt.Errorf("validate exec request: %w", err)
	}
	if request.TTY || request.Stdin != nil || request.Stdout != nil || request.Stderr != nil {
		return runner.ExecResult{}, fmt.Errorf(
			"%w: interactive/attached Podman exec is not implemented yet",
			runner.ErrNotSupported,
		)
	}

	env, err := encodeEnvironment(request.Env)
	if err != nil {
		return runner.ExecResult{}, err
	}

	create := execCreateRequest{
		AttachStderr: false,
		AttachStdin:  false,
		AttachStdout: false,
		Cmd:          append([]string(nil), request.Argv...),
		Env:          env,
		Privileged:   false,
		Tty:          false,
		WorkingDir:   request.WorkingDir,
	}

	var created execCreateResponse
	if err := backend.doJSON(
		ctx,
		http.MethodPost,
		compatBase+"/containers/"+url.PathEscape(instance.ID)+"/exec",
		nil,
		create,
		&created,
	); err != nil {
		return runner.ExecResult{}, fmt.Errorf("create exec session: %w", err)
	}
	if created.ID == "" {
		return runner.ExecResult{}, errors.New("Podman returned an empty exec session ID")
	}

	if err := backend.doJSON(
		ctx,
		http.MethodPost,
		compatBase+"/exec/"+url.PathEscape(created.ID)+"/start",
		nil,
		execStartRequest{Detach: true, Tty: false},
		nil,
	); err != nil {
		return runner.ExecResult{}, fmt.Errorf("start exec session %s: %w", created.ID, err)
	}

	const pollInterval = 25 * time.Millisecond
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		var inspected execInspectResponse
		if err := backend.doJSON(
			ctx,
			http.MethodGet,
			compatBase+"/exec/"+url.PathEscape(created.ID)+"/json",
			nil,
			nil,
			&inspected,
		); err != nil {
			return runner.ExecResult{}, fmt.Errorf("inspect exec session %s: %w", created.ID, err)
		}
		if !inspected.Running {
			return runner.ExecResult{ExitCode: inspected.ExitCode}, nil
		}

		select {
		case <-ctx.Done():
			return runner.ExecResult{}, ctx.Err()
		case <-ticker.C:
		}
	}
}

func encodeEnvironment(values map[string]string) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}

	keys := make([]string, 0, len(values))
	for key, value := range values {
		if key == "" || strings.ContainsAny(key, "=\x00") {
			return nil, fmt.Errorf("invalid environment variable name %q", key)
		}
		if strings.ContainsRune(value, '\x00') {
			return nil, fmt.Errorf("environment variable %q contains NUL", key)
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	encoded := make([]string, 0, len(keys))
	for _, key := range keys {
		encoded = append(encoded, key+"="+values[key])
	}
	return encoded, nil
}
