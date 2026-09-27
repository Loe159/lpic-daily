package libvirt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/Loe159/lpic-daily/internal/runner"
)

const (
	guestAgentCallTimeoutSeconds = int32(5)
	guestAgentReadyTimeout       = 20 * time.Second
	guestAgentReadyRetry         = 100 * time.Millisecond
	maxGuestAgentResponseBytes   = 2 << 20
	maxGuestExecOutputBytes      = 1 << 20
)

type qgaRequest struct {
	Execute   string `json:"execute"`
	Arguments any    `json:"arguments,omitempty"`
}

type qgaEnvelope struct {
	Return json.RawMessage `json:"return"`
	Error  *qgaError       `json:"error,omitempty"`
}

type qgaError struct {
	Class string `json:"class"`
	Desc  string `json:"desc"`
}

type guestExecArguments struct {
	Path          string   `json:"path"`
	Arg           []string `json:"arg,omitempty"`
	Env           []string `json:"env,omitempty"`
	CaptureOutput bool     `json:"capture-output"`
}

type guestExecStarted struct {
	PID int `json:"pid"`
}

type guestExecStatusArguments struct {
	PID int `json:"pid"`
}

type guestExecStatus struct {
	Exited   bool   `json:"exited"`
	ExitCode *int   `json:"exitcode,omitempty"`
	Signal   *int   `json:"signal,omitempty"`
	OutData  []byte `json:"out-data,omitempty"`
	ErrData  []byte `json:"err-data,omitempty"`
}

func (backend *Backend) Exec(
	ctx context.Context,
	instance runner.Instance,
	request runner.ExecRequest,
) (runner.ExecResult, error) {
	if err := request.Validate(); err != nil {
		return runner.ExecResult{}, fmt.Errorf("validate VM exec request: %w", err)
	}
	if request.TTY || request.Stdin != nil || request.Resize != nil ||
		request.InitialSize.Width != 0 || request.InitialSize.Height != 0 {
		return runner.ExecResult{}, fmt.Errorf(
			"%w: VM interactive execution uses the serial console, not guest-exec",
			runner.ErrNotSupported,
		)
	}
	if request.WorkingDir != "" {
		return runner.ExecResult{}, fmt.Errorf(
			"%w: QEMU guest-exec does not provide a working-directory parameter",
			runner.ErrNotSupported,
		)
	}
	if err := ctx.Err(); err != nil {
		return runner.ExecResult{}, err
	}
	if _, err := backend.instance(instance); err != nil {
		return runner.ExecResult{}, err
	}
	state, err := backend.control.DomainState(instance.ID)
	if err != nil {
		return runner.ExecResult{}, err
	}
	if !state.Active {
		return runner.ExecResult{}, fmt.Errorf("VM instance %s is not active", instance.ID)
	}
	if err := backend.waitForGuestAgent(ctx, instance.ID); err != nil {
		return runner.ExecResult{}, err
	}

	environment, err := encodeGuestEnvironment(request.Env)
	if err != nil {
		return runner.ExecResult{}, err
	}
	var started guestExecStarted
	if err := backend.agentJSON(ctx, instance.ID, qgaRequest{
		Execute: "guest-exec",
		Arguments: guestExecArguments{
			Path:          request.Argv[0],
			Arg:           append([]string(nil), request.Argv[1:]...),
			Env:           environment,
			CaptureOutput: true,
		},
	}, &started); err != nil {
		return runner.ExecResult{}, fmt.Errorf("start guest exec: %w", err)
	}
	if started.PID <= 0 {
		return runner.ExecResult{}, fmt.Errorf("guest-exec returned invalid pid %d", started.PID)
	}

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		var status guestExecStatus
		if err := backend.agentJSON(ctx, instance.ID, qgaRequest{
			Execute:   "guest-exec-status",
			Arguments: guestExecStatusArguments{PID: started.PID},
		}, &status); err != nil {
			return runner.ExecResult{}, fmt.Errorf("inspect guest exec %d: %w", started.PID, err)
		}
		if status.Exited {
			if len(status.OutData) > maxGuestExecOutputBytes ||
				len(status.ErrData) > maxGuestExecOutputBytes {
				return runner.ExecResult{}, errors.New("guest exec output exceeded 1 MiB safety limit")
			}
			if err := writeGuestOutput(request.Stdout, status.OutData); err != nil {
				return runner.ExecResult{}, fmt.Errorf("write guest stdout: %w", err)
			}
			if err := writeGuestOutput(request.Stderr, status.ErrData); err != nil {
				return runner.ExecResult{}, fmt.Errorf("write guest stderr: %w", err)
			}
			if status.ExitCode != nil {
				return runner.ExecResult{ExitCode: *status.ExitCode}, nil
			}
			if status.Signal != nil {
				return runner.ExecResult{ExitCode: 128 + *status.Signal}, nil
			}
			return runner.ExecResult{}, errors.New("guest exec exited without exit code or signal")
		}

		select {
		case <-ctx.Done():
			return runner.ExecResult{}, ctx.Err()
		case <-ticker.C:
		}
	}
}

func (backend *Backend) waitForGuestAgent(ctx context.Context, domainName string) error {
	waitCtx, cancel := context.WithTimeout(ctx, guestAgentReadyTimeout)
	defer cancel()

	var lastErr error
	for {
		if err := waitCtx.Err(); err != nil {
			if lastErr != nil {
				return fmt.Errorf("wait for QEMU guest agent in %s: %w (last error: %v)", domainName, err, lastErr)
			}
			return fmt.Errorf("wait for QEMU guest agent in %s: %w", domainName, err)
		}

		var pong struct{}
		err := backend.agentJSON(waitCtx, domainName, qgaRequest{Execute: "guest-ping"}, &pong)
		if err == nil {
			return nil
		}
		lastErr = err

		timer := time.NewTimer(guestAgentReadyRetry)
		select {
		case <-waitCtx.Done():
			timer.Stop()
		case <-timer.C:
		}
	}
}

func (backend *Backend) agentJSON(
	ctx context.Context,
	domainName string,
	request qgaRequest,
	result any,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	agent, ok := backend.control.(AgentControlPlane)
	if !ok {
		return fmt.Errorf(
			"%w: libvirt control plane has no QEMU guest-agent capability",
			runner.ErrNotSupported,
		)
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("encode guest-agent request: %w", err)
	}
	raw, err := agent.AgentCommand(domainName, string(payload), guestAgentCallTimeoutSeconds)
	if err != nil {
		return err
	}
	if len(raw) > maxGuestAgentResponseBytes {
		return errors.New("guest-agent response exceeded 2 MiB safety limit")
	}

	var envelope qgaEnvelope
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		return fmt.Errorf("decode guest-agent envelope: %w", err)
	}
	if envelope.Error != nil {
		return fmt.Errorf(
			"guest-agent %s: %s",
			strings.TrimSpace(envelope.Error.Class),
			strings.TrimSpace(envelope.Error.Desc),
		)
	}
	if len(envelope.Return) == 0 {
		return errors.New("guest-agent response has no return value")
	}
	if result == nil {
		return nil
	}
	if err := json.Unmarshal(envelope.Return, result); err != nil {
		return fmt.Errorf("decode guest-agent return value: %w", err)
	}
	return nil
}

func encodeGuestEnvironment(environment map[string]string) ([]string, error) {
	if len(environment) == 0 {
		return nil, nil
	}
	keys := make([]string, 0, len(environment))
	for key, value := range environment {
		if key == "" || strings.ContainsAny(key, "=\x00") {
			return nil, fmt.Errorf("invalid environment variable name %q", key)
		}
		if strings.ContainsRune(value, '\x00') {
			return nil, fmt.Errorf("environment variable %q contains NUL", key)
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := make([]string, 0, len(keys))
	for _, key := range keys {
		result = append(result, key+"="+environment[key])
	}
	return result, nil
}

func writeGuestOutput(writer io.Writer, content []byte) error {
	if writer == nil || len(content) == 0 {
		return nil
	}
	_, err := writer.Write(content)
	return err
}
