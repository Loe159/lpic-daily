package podman

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
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

	env, err := encodeEnvironment(request.Env)
	if err != nil {
		return runner.ExecResult{}, err
	}

	if request.TTY {
		return backend.execTTY(ctx, instance, request, env)
	}
	if request.Stdin != nil {
		return runner.ExecResult{}, fmt.Errorf(
			"%w: non-TTY stdin Podman exec is not implemented yet",
			runner.ErrNotSupported,
		)
	}

	attached := request.Stdout != nil || request.Stderr != nil
	created, err := backend.createExec(ctx, instance, execCreateRequest{
		AttachStderr: attached,
		AttachStdin:  false,
		AttachStdout: attached,
		Cmd:          append([]string(nil), request.Argv...),
		Env:          env,
		Privileged:   false,
		Tty:          false,
		WorkingDir:   request.WorkingDir,
	})
	if err != nil {
		return runner.ExecResult{}, err
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

func (backend *Backend) createExec(
	ctx context.Context,
	instance runner.Instance,
	request execCreateRequest,
) (execCreateResponse, error) {
	var created execCreateResponse
	if err := backend.doJSON(
		ctx,
		http.MethodPost,
		compatBase+"/containers/"+url.PathEscape(instance.ID)+"/exec",
		nil,
		request,
		&created,
	); err != nil {
		return execCreateResponse{}, fmt.Errorf("create exec session: %w", err)
	}
	if created.ID == "" {
		return execCreateResponse{}, errors.New("Podman returned an empty exec session ID")
	}
	return created, nil
}

func (backend *Backend) execTTY(
	ctx context.Context,
	instance runner.Instance,
	request runner.ExecRequest,
	env []string,
) (runner.ExecResult, error) {
	created, err := backend.createExec(ctx, instance, execCreateRequest{
		AttachStderr: true,
		AttachStdin:  true,
		AttachStdout: true,
		Cmd:          append([]string(nil), request.Argv...),
		Env:          env,
		Privileged:   false,
		Tty:          true,
		WorkingDir:   request.WorkingDir,
	})
	if err != nil {
		return runner.ExecResult{}, err
	}

	conn, reader, err := backend.startTTYExec(ctx, created.ID)
	if err != nil {
		return runner.ExecResult{}, err
	}

	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer conn.Close()

	if request.InitialSize.Width != 0 || request.InitialSize.Height != 0 {
		if err := backend.resizeExec(streamCtx, created.ID, request.InitialSize); err != nil {
			return runner.ExecResult{}, fmt.Errorf("set initial terminal size: %w", err)
		}
	}

	resizeErr := make(chan error, 1)
	if request.Resize != nil {
		go backend.forwardResize(streamCtx, created.ID, request.Resize, resizeErr)
	}

	contextDone := make(chan struct{})
	go func() {
		select {
		case <-streamCtx.Done():
			_ = conn.Close()
		case <-contextDone:
		}
	}()
	defer close(contextDone)

	if request.Stdin != nil {
		go func() {
			_, _ = io.Copy(conn, request.Stdin)
			if unixConn, ok := conn.(*net.UnixConn); ok {
				_ = unixConn.CloseWrite()
			}
		}()
	}

	stdout := request.Stdout
	if stdout == nil {
		stdout = io.Discard
	}
	_, copyErr := io.Copy(stdout, reader)
	cancel()
	_ = conn.Close()

	select {
	case err := <-resizeErr:
		if err != nil {
			return runner.ExecResult{}, err
		}
	default:
	}
	if copyErr != nil && !errors.Is(copyErr, net.ErrClosed) && ctx.Err() == nil {
		return runner.ExecResult{}, fmt.Errorf("read TTY stream: %w", copyErr)
	}
	if err := ctx.Err(); err != nil {
		return runner.ExecResult{}, err
	}
	return backend.waitExec(ctx, created.ID)
}

func (backend *Backend) startTTYExec(
	ctx context.Context,
	execID string,
) (net.Conn, *bufio.Reader, error) {
	payload, err := json.Marshal(execStartRequest{Detach: false, Tty: true})
	if err != nil {
		return nil, nil, fmt.Errorf("encode TTY exec start request: %w", err)
	}

	conn, err := (&net.Dialer{}).DialContext(ctx, "unix", backend.socketPath)
	if err != nil {
		return nil, nil, fmt.Errorf("connect TTY stream to Podman service: %w", err)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"http://podman"+compatBase+"/exec/"+url.PathEscape(execID)+"/start",
		bytes.NewReader(payload),
	)
	if err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("build TTY exec start request: %w", err)
	}
	request.Host = "podman"
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Connection", "Upgrade")
	request.Header.Set("Upgrade", "tcp")

	if err := request.Write(conn); err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("write TTY exec start request: %w", err)
	}

	reader := bufio.NewReader(conn)
	response, err := http.ReadResponse(reader, request)
	if err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("read TTY upgrade response: %w", err)
	}
	if response.Body != nil {
		defer response.Body.Close()
	}
	if response.StatusCode != http.StatusSwitchingProtocols {
		const maxError = 1 << 20
		body, readErr := io.ReadAll(io.LimitReader(response.Body, maxError+1))
		_ = conn.Close()
		if readErr != nil {
			return nil, nil, fmt.Errorf("read TTY exec error: %w", readErr)
		}
		if len(body) > maxError {
			return nil, nil, errors.New("TTY exec error response exceeded 1 MiB safety limit")
		}
		return nil, nil, &statusError{
			Code:    response.StatusCode,
			Method:  http.MethodPost,
			Path:    compatBase + "/exec/" + execID + "/start",
			Message: strings.TrimSpace(string(body)),
		}
	}

	if !strings.EqualFold(response.Header.Get("Upgrade"), "tcp") {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("unexpected TTY upgrade protocol %q", response.Header.Get("Upgrade"))
	}
	return conn, reader, nil
}

func (backend *Backend) forwardResize(
	ctx context.Context,
	execID string,
	resize <-chan runner.TerminalSize,
	errs chan<- error,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case size, ok := <-resize:
			if !ok {
				return
			}
			if err := size.Validate(); err != nil {
				select {
				case errs <- fmt.Errorf("invalid terminal resize: %w", err):
				default:
				}
				return
			}
			if err := backend.resizeExec(ctx, execID, size); err != nil {
				select {
				case errs <- fmt.Errorf("resize terminal: %w", err):
				default:
				}
				return
			}
		}
	}
}

func (backend *Backend) resizeExec(
	ctx context.Context,
	execID string,
	size runner.TerminalSize,
) error {
	if err := size.Validate(); err != nil {
		return err
	}
	query := url.Values{
		"h": {strconv.FormatUint(uint64(size.Height), 10)},
		"w": {strconv.FormatUint(uint64(size.Width), 10)},
	}
	return backend.doJSON(
		ctx,
		http.MethodPost,
		compatBase+"/exec/"+url.PathEscape(execID)+"/resize",
		query,
		nil,
		nil,
	)
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
