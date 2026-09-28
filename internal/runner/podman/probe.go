package podman

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Loe159/lpic-daily/internal/runner"
)

type topResponse struct {
	Titles    []string   `json:"Titles"`
	Processes [][]string `json:"Processes"`
}

func (backend *Backend) Stat(
	ctx context.Context,
	instance runner.Instance,
	guestPath string,
) (runner.FileInfo, error) {
	if instance.ID == "" {
		return runner.FileInfo{}, errors.New("instance ID is required")
	}
	if err := validateGuestPath(guestPath); err != nil {
		return runner.FileInfo{}, err
	}

	header, err := backend.archiveHeader(ctx, instance, guestPath)
	if err != nil {
		return runner.FileInfo{}, err
	}

	// Podman's archive endpoint is useful for bounded filesystem probing, but
	// ownership metadata is not stable across rootless storage drivers and
	// versions: some releases report host-remapped root:root for a guest path
	// that is root:project inside the container. Ask stat inside the already
	// isolated guest for the canonical guest-visible ownership.
	uid, gid, user, group, err := backend.guestOwnership(ctx, instance, guestPath)
	if err != nil {
		return runner.FileInfo{}, err
	}

	return runner.FileInfo{
		Path:  guestPath,
		Mode:  uint32(header.Mode),
		UID:   uid,
		GID:   gid,
		User:  user,
		Group: group,
		IsDir: header.FileInfo().IsDir(),
	}, nil
}

func (backend *Backend) guestOwnership(
	ctx context.Context,
	instance runner.Instance,
	guestPath string,
) (uint32, uint32, string, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	result, err := backend.Exec(ctx, instance, runner.ExecRequest{
		Argv: []string{
			"/usr/bin/stat",
			"--printf=%u\\n%g\\n%U\\n%G\\n",
			"--",
			guestPath,
		},
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		return 0, 0, "", "", fmt.Errorf("stat guest ownership for %s: %w", guestPath, err)
	}
	if result.ExitCode != 0 {
		return 0, 0, "", "", fmt.Errorf(
			"stat guest ownership for %s exited %d: %s",
			guestPath,
			result.ExitCode,
			strings.TrimSpace(stderr.String()),
		)
	}

	lines := strings.Split(strings.TrimSuffix(stdout.String(), "\n"), "\n")
	if len(lines) != 4 {
		return 0, 0, "", "", fmt.Errorf(
			"stat guest ownership for %s returned %d fields, want 4",
			guestPath,
			len(lines),
		)
	}
	uid64, err := strconv.ParseUint(lines[0], 10, 32)
	if err != nil {
		return 0, 0, "", "", fmt.Errorf("parse guest uid %q for %s: %w", lines[0], guestPath, err)
	}
	gid64, err := strconv.ParseUint(lines[1], 10, 32)
	if err != nil {
		return 0, 0, "", "", fmt.Errorf("parse guest gid %q for %s: %w", lines[1], guestPath, err)
	}
	if lines[2] == "" || lines[3] == "" {
		return 0, 0, "", "", fmt.Errorf("stat guest ownership for %s returned empty names", guestPath)
	}
	return uint32(uid64), uint32(gid64), lines[2], lines[3], nil
}

func (backend *Backend) ReadFile(
	ctx context.Context,
	instance runner.Instance,
	guestPath string,
	maxBytes int64,
) ([]byte, error) {
	if instance.ID == "" {
		return nil, errors.New("instance ID is required")
	}
	if err := validateGuestPath(guestPath); err != nil {
		return nil, err
	}
	if maxBytes <= 0 {
		return nil, errors.New("maxBytes must be positive")
	}
	return backend.readArchiveFile(ctx, instance, guestPath, maxBytes)
}

func (backend *Backend) Processes(
	ctx context.Context,
	instance runner.Instance,
) ([]runner.Process, error) {
	if instance.ID == "" {
		return nil, errors.New("instance ID is required")
	}

	query := url.Values{
		"stream":  {"false"},
		"ps_args": {"pid,comm,args"},
	}
	var response topResponse
	if err := backend.doJSON(
		ctx,
		http.MethodGet,
		apiBase+"/containers/"+url.PathEscape(instance.ID)+"/top",
		query,
		nil,
		&response,
	); err != nil {
		return nil, fmt.Errorf("list container processes: %w", err)
	}

	pidIndex := titleIndex(response.Titles, "PID")
	commandIndex := titleIndex(response.Titles, "COMMAND", "COMM")
	argsIndex := titleIndex(response.Titles, "ARGS")
	if argsIndex < 0 {
		// Some Podman versions label both comm and args columns as COMMAND
		// even when ps_args=pid,comm,args. In that case the second COMMAND
		// column is the full argument vector.
		commandColumns := titleIndexes(response.Titles, "COMMAND")
		if len(commandColumns) >= 2 {
			commandIndex = commandColumns[0]
			argsIndex = commandColumns[1]
		}
	}
	if pidIndex < 0 || commandIndex < 0 || argsIndex < 0 {
		return nil, fmt.Errorf("unexpected Podman top titles: %v", response.Titles)
	}

	processes := make([]runner.Process, 0, len(response.Processes))
	for rowNumber, row := range response.Processes {
		maxIndex := max(pidIndex, commandIndex, argsIndex)
		if len(row) <= maxIndex {
			return nil, fmt.Errorf("Podman top row %d has %d columns, need %d", rowNumber, len(row), maxIndex+1)
		}
		pid, err := strconv.Atoi(strings.TrimSpace(row[pidIndex]))
		if err != nil {
			return nil, fmt.Errorf("parse Podman top PID %q: %w", row[pidIndex], err)
		}
		processes = append(processes, runner.Process{
			PID:     pid,
			Command: strings.TrimSpace(row[commandIndex]),
			Args:    strings.Fields(row[argsIndex]),
		})
	}
	return processes, nil
}

func (backend *Backend) archiveHeader(
	ctx context.Context,
	instance runner.Instance,
	guestPath string,
) (*tar.Header, error) {
	response, err := backend.archiveResponse(ctx, instance, guestPath)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	reader := tar.NewReader(response.Body)
	header, err := reader.Next()
	if err != nil {
		return nil, fmt.Errorf("read archive header for %s: %w", guestPath, err)
	}
	copy := *header
	return &copy, nil
}

func (backend *Backend) readArchiveFile(
	ctx context.Context,
	instance runner.Instance,
	guestPath string,
	maxBytes int64,
) ([]byte, error) {
	response, err := backend.archiveResponse(ctx, instance, guestPath)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	reader := tar.NewReader(response.Body)
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("%s: archive contained no regular file", guestPath)
		}
		if err != nil {
			return nil, fmt.Errorf("read archive for %s: %w", guestPath, err)
		}
		if !header.FileInfo().Mode().IsRegular() {
			continue
		}
		if header.Size > maxBytes {
			return nil, fmt.Errorf("%s is %d bytes, exceeds %d-byte read limit", guestPath, header.Size, maxBytes)
		}
		content, err := io.ReadAll(io.LimitReader(reader, maxBytes+1))
		if err != nil {
			return nil, fmt.Errorf("read %s from archive: %w", guestPath, err)
		}
		if int64(len(content)) > maxBytes {
			return nil, fmt.Errorf("%s exceeded %d-byte read limit", guestPath, maxBytes)
		}
		return content, nil
	}
}

func (backend *Backend) archiveResponse(
	ctx context.Context,
	instance runner.Instance,
	guestPath string,
) (*http.Response, error) {
	query := url.Values{"path": {guestPath}}
	return backend.doRaw(
		ctx,
		http.MethodGet,
		apiBase+"/containers/"+url.PathEscape(instance.ID)+"/archive",
		query,
		"application/x-tar",
	)
}

func (backend *Backend) lookupIdentity(
	ctx context.Context,
	instance runner.Instance,
	databasePath string,
	id int,
) string {
	content, err := backend.readArchiveFile(ctx, instance, databasePath, 1<<20)
	if err != nil {
		return strconv.Itoa(id)
	}

	for _, line := range bytes.Split(content, []byte{'\n'}) {
		fields := bytes.Split(line, []byte{':'})
		if len(fields) < 3 {
			continue
		}
		value, err := strconv.Atoi(string(fields[2]))
		if err == nil && value == id {
			return string(fields[0])
		}
	}
	return strconv.Itoa(id)
}

func (backend *Backend) doRaw(
	ctx context.Context,
	method string,
	apiPath string,
	query url.Values,
	accept string,
) (*http.Response, error) {
	requestURL := "http://podman" + apiPath
	if len(query) != 0 {
		requestURL += "?" + query.Encode()
	}
	request, err := http.NewRequestWithContext(ctx, method, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build Podman request: %w", err)
	}
	if accept != "" {
		request.Header.Set("Accept", accept)
	}

	response, err := backend.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("call Podman service on %s: %w", backend.socketPath, err)
	}
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return response, nil
	}
	defer response.Body.Close()

	const maxErrorResponse = 1 << 20
	payload, readErr := io.ReadAll(io.LimitReader(response.Body, maxErrorResponse+1))
	if readErr != nil {
		return nil, fmt.Errorf("read Podman error response: %w", readErr)
	}
	if len(payload) > maxErrorResponse {
		return nil, errors.New("Podman error response exceeded 1 MiB safety limit")
	}

	var apiErr apiError
	_ = json.Unmarshal(payload, &apiErr)
	message := strings.TrimSpace(apiErr.Message)
	if message == "" {
		message = strings.TrimSpace(string(payload))
	}
	return nil, &statusError{
		Code:    response.StatusCode,
		Method:  method,
		Path:    apiPath,
		Message: message,
	}
}

func validateGuestPath(guestPath string) error {
	if guestPath == "" {
		return errors.New("guest path is required")
	}
	if strings.ContainsRune(guestPath, '\x00') {
		return errors.New("guest path contains NUL")
	}
	if !filepath.IsAbs(guestPath) {
		return errors.New("guest path must be absolute")
	}
	return nil
}

func titleIndexes(titles []string, candidate string) []int {
	indexes := make([]int, 0, len(titles))
	for index, title := range titles {
		if strings.EqualFold(strings.TrimSpace(title), candidate) {
			indexes = append(indexes, index)
		}
	}
	return indexes
}

func titleIndex(titles []string, candidates ...string) int {
	for index, title := range titles {
		for _, candidate := range candidates {
			if strings.EqualFold(strings.TrimSpace(title), candidate) {
				return index
			}
		}
	}
	return -1
}
