package podman

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	if request.TTY || request.Stdin != nil {
		return runner.ExecResult{}, fmt.Errorf(
			"%w: TTY/stdin Podman exec is not implemented yet",
			runner.ErrNotSupported,
		)
	}

	env, err := encodeEnvironment(request.Env)
	if err != nil {
		return runner.ExecResult{}, err
	}

	attached := request.Stdout != nil || request.Stderr != nil
	create := execCreateRequest{
		AttachStderr: attached,
		AttachStdin:  false,
		AttachStdout: attached,
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

	if attached {
		if err := backend.startAttachedExec(ctx, created.ID, request.Stdout, request.Stderr); err != nil {
			return runner.ExecResult{}, err
		}
	} else {
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
	}

	return backend.waitExec(ctx, created.ID)
}

func (backend *Backend) startAttachedExec(
	ctx context.Context,
	execID string,
	stdout io.Writer,
	stderr io.Writer,
) error {
	payload, err := json.Marshal(execStartRequest{Detach: false, Tty: false})
	if err != nil {
		return fmt.Errorf("encode exec start request: %w", err)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"http://podman"+compatBase+"/exec/"+url.PathEscape(execID)+"/start",
		bytes.NewReader(payload),
	)
	if err != nil {
		return fmt.Errorf("build attached exec request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/vnd.docker.raw-stream")

	response, err := backend.client.Do(request)
	if err != nil {
		return fmt.Errorf("start attached exec %s: %w", execID, err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		const maxError = 1 << 20
		body, readErr := io.ReadAll(io.LimitReader(response.Body, maxError+1))
		if readErr != nil {
			return fmt.Errorf("read attached exec error: %w", readErr)
		}
		return &statusError{
			Code:    response.StatusCode,
			Method:  http.MethodPost,
			Path:    compatBase + "/exec/" + execID + "/start",
			Message: strings.TrimSpace(string(body)),
		}
	}

	if err := demultiplexDockerStream(response.Body, stdout, stderr); err != nil {
		return fmt.Errorf("read attached exec %s output: %w", execID, err)
	}
	return nil
}

func (backend *Backend) waitExec(ctx context.Context, execID string) (runner.ExecResult, error) {
	const pollInterval = 25 * time.Millisecond
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		var inspected execInspectResponse
		if err := backend.doJSON(
			ctx,
			http.MethodGet,
			compatBase+"/exec/"+url.PathEscape(execID)+"/json",
			nil,
			nil,
			&inspected,
		); err != nil {
			return runner.ExecResult{}, fmt.Errorf("inspect exec session %s: %w", execID, err)
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

func demultiplexDockerStream(source io.Reader, stdout, stderr io.Writer) error {
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}

	var header [8]byte
	for {
		if _, err := io.ReadFull(source, header[:]); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			if errors.Is(err, io.ErrUnexpectedEOF) {
				return errors.New("truncated Docker stream header")
			}
			return err
		}

		size := int64(binary.BigEndian.Uint32(header[4:8]))
		if size > 16<<20 {
			return fmt.Errorf("Docker stream frame too large: %d bytes", size)
		}

		var destination io.Writer
		switch header[0] {
		case 1:
			destination = stdout
		case 2:
			destination = stderr
		default:
			return fmt.Errorf("unsupported Docker stream id %d", header[0])
		}

		if _, err := io.CopyN(destination, source, size); err != nil {
			return fmt.Errorf("copy Docker stream frame: %w", err)
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
