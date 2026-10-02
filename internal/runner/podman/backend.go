package podman

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Loe159/lpic-daily/internal/runner"
)

const (
	apiBase                            = "/v6.0.0/libpod"
	phase1WritablePathLimitBytes int64 = 32 << 20
	managedLabel                       = "io.lpic-daily.managed"
	labIDLabel                         = "io.lpic-daily.lab-id"
	expiresAtLabel                     = "io.lpic-daily.expires-at"
	abandonedCleanupGrace              = time.Minute
)

var (
	safeNamePart = regexp.MustCompile(`[^a-z0-9_.-]+`)
	edgeDashes   = regexp.MustCompile(`^-+|-+$`)
)

type Backend struct {
	socketPath string
	client     *http.Client

	mu          sync.RWMutex
	definitions map[string]runner.Definition
}

type infoResponse struct {
	Host *struct {
		CgroupsVersion string `json:"cgroupVersion"`
		Security       struct {
			Rootless bool `json:"rootless"`
		} `json:"security"`
	} `json:"host"`
}

type namespace struct {
	NSMode string `json:"nsmode"`
}

type linuxResources struct {
	Memory *linuxMemory `json:"memory,omitempty"`
	CPU    *linuxCPU    `json:"cpu,omitempty"`
	Pids   *linuxPids   `json:"pids,omitempty"`
}

type linuxMemory struct {
	Limit *int64 `json:"limit,omitempty"`
}

type linuxCPU struct {
	Quota  *int64  `json:"quota,omitempty"`
	Period *uint64 `json:"period,omitempty"`
}

type linuxPids struct {
	Limit int64 `json:"limit"`
}

type ociMount struct {
	Destination string   `json:"destination"`
	Type        string   `json:"type"`
	Source      string   `json:"source"`
	Options     []string `json:"options,omitempty"`
}

type createRequest struct {
	Name               string            `json:"name"`
	Image              string            `json:"image"`
	RawImageName       string            `json:"raw_image_name"`
	Command            []string          `json:"command"`
	EnvHost            *bool             `json:"env_host"`
	HTTPProxy          *bool             `json:"httpproxy"`
	Terminal           *bool             `json:"terminal"`
	Stdin              *bool             `json:"stdin"`
	Labels             map[string]string `json:"labels"`
	Timeout            uint              `json:"timeout"`
	Privileged         *bool             `json:"privileged"`
	CapAdd             []string          `json:"cap_add,omitempty"`
	CapDrop            []string          `json:"cap_drop"`
	NoNewPrivileges    *bool             `json:"no_new_privileges"`
	NetNS              namespace         `json:"netns"`
	PidNS              namespace         `json:"pidns"`
	UtsNS              namespace         `json:"utsns"`
	IpcNS              namespace         `json:"ipcns"`
	ImageVolumeMode    string            `json:"image_volume_mode"`
	ReadOnlyFilesystem *bool             `json:"read_only_filesystem"`
	ReadWriteTmpfs     *bool             `json:"read_write_tmpfs"`
	Mounts             []ociMount        `json:"mounts,omitempty"`
	ResourceLimits     *linuxResources   `json:"resource_limits"`
}

type createResponse struct {
	ID string `json:"Id"`
}

type imageInspectResponse struct {
	ID string `json:"Id"`
}

type containerSummary struct {
	ID     string            `json:"Id"`
	Names  []string          `json:"Names"`
	Labels map[string]string `json:"Labels"`
}

type apiError struct {
	Cause   string `json:"cause"`
	Message string `json:"message"`
}

func DefaultURI() string {
	if runtimeDir := os.Getenv("XDG_RUNTIME_DIR"); runtimeDir != "" {
		return "unix://" + filepath.Join(runtimeDir, "podman", "podman.sock")
	}
	return fmt.Sprintf("unix:///run/user/%d/podman/podman.sock", os.Geteuid())
}

func Open(ctx context.Context, uri string) (*Backend, error) {
	if uri == "" {
		uri = DefaultURI()
	}
	socketPath, err := socketPathFromURI(uri)
	if err != nil {
		return nil, err
	}

	transport := &http.Transport{
		Proxy: nil,
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, "unix", socketPath)
		},
		DisableCompression: true,
	}
	client := &http.Client{
		Transport: transport,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return errors.New("Podman API redirects are not allowed")
		},
	}

	backend := &Backend{
		socketPath:  socketPath,
		client:      client,
		definitions: make(map[string]runner.Definition),
	}

	var info infoResponse
	if err := backend.doJSON(ctx, http.MethodGet, apiBase+"/info", nil, nil, &info); err != nil {
		return nil, fmt.Errorf("inspect Podman service: %w", err)
	}
	if info.Host == nil {
		return nil, errors.New("Podman service returned no host information")
	}
	if !info.Host.Security.Rootless {
		return nil, errors.New("refusing rootful Podman service: LPIC Daily requires rootless Podman")
	}
	if info.Host.CgroupsVersion != "v2" {
		return nil, fmt.Errorf("unsupported cgroups version %q: Phase 1 requires cgroups v2 for rootless limits", info.Host.CgroupsVersion)
	}

	return backend, nil
}

func (backend *Backend) Prepare(ctx context.Context, definition runner.Definition) (runner.Instance, error) {
	if err := definition.Validate(); err != nil {
		return runner.Instance{}, fmt.Errorf("validate lab definition: %w", err)
	}
	if definition.Machine != nil {
		return runner.Instance{}, fmt.Errorf("%w: Podman does not accept full-machine settings", runner.ErrNotSupported)
	}
	if definition.Network != runner.NetworkNone {
		return runner.Instance{}, fmt.Errorf("%w: Podman Phase 1 supports network=none only", runner.ErrNotSupported)
	}
	if _, err := runner.Phase1CapabilityProfile(definition.CapabilityProfile); err != nil {
		return runner.Instance{}, err
	}

	imageID, err := backend.resolveImageID(ctx, definition.ImageRef)
	if err != nil {
		return runner.Instance{}, err
	}
	definition.ImageRef = imageID

	name, err := instanceName(definition.LabID)
	if err != nil {
		return runner.Instance{}, err
	}
	if err := backend.create(ctx, definition, name); err != nil {
		return runner.Instance{}, err
	}

	backend.mu.Lock()
	backend.definitions[name] = definition
	backend.mu.Unlock()
	return runner.Instance{ID: name}, nil
}

func (backend *Backend) Start(ctx context.Context, instance runner.Instance) error {
	if err := backend.requireManagedInstance(instance); err != nil {
		return err
	}
	if err := backend.doJSON(
		ctx,
		http.MethodPost,
		apiBase+"/containers/"+url.PathEscape(instance.ID)+"/start",
		nil,
		nil,
		nil,
	); err != nil {
		return fmt.Errorf("start container %s: %w", instance.ID, err)
	}
	return nil
}

func (backend *Backend) Reset(ctx context.Context, instance runner.Instance) error {
	backend.mu.RLock()
	definition, exists := backend.definitions[instance.ID]
	backend.mu.RUnlock()
	if !exists {
		return fmt.Errorf("unknown managed instance %q", instance.ID)
	}
	if err := backend.remove(ctx, instance.ID); err != nil {
		return err
	}
	if err := backend.create(ctx, definition, instance.ID); err != nil {
		return fmt.Errorf("recreate container %s: %w", instance.ID, err)
	}
	return nil
}

func (backend *Backend) Destroy(ctx context.Context, instance runner.Instance) error {
	if err := backend.requireManagedInstance(instance); err != nil {
		return err
	}
	if err := backend.remove(ctx, instance.ID); err != nil {
		return err
	}
	backend.mu.Lock()
	delete(backend.definitions, instance.ID)
	backend.mu.Unlock()
	return nil
}

func (backend *Backend) requireManagedInstance(instance runner.Instance) error {
	if instance.ID == "" {
		return errors.New("instance ID is required")
	}
	backend.mu.RLock()
	_, exists := backend.definitions[instance.ID]
	backend.mu.RUnlock()
	if !exists {
		return fmt.Errorf("unknown managed instance %q", instance.ID)
	}
	return nil
}

func (backend *Backend) ReapAbandoned(ctx context.Context, now time.Time) error {
	if now.IsZero() {
		return errors.New("reap time is required")
	}
	filters, err := json.Marshal(map[string][]string{
		"label": {managedLabel + "=true"},
	})
	if err != nil {
		return fmt.Errorf("encode Podman reap filters: %w", err)
	}
	query := url.Values{
		"all":     {"true"},
		"filters": {string(filters)},
	}
	var containers []containerSummary
	if err := backend.doJSON(
		ctx,
		http.MethodGet,
		apiBase+"/containers/json",
		query,
		nil,
		&containers,
	); err != nil {
		return fmt.Errorf("list managed Podman containers: %w", err)
	}

	var errs []error
	for _, container := range containers {
		if container.Labels[managedLabel] != "true" ||
			strings.TrimSpace(container.Labels[labIDLabel]) == "" ||
			!hasManagedContainerName(container.Names) {
			continue
		}
		expiresText := strings.TrimSpace(container.Labels[expiresAtLabel])
		if expiresText == "" {
			continue
		}
		expiresAt, err := time.Parse(time.RFC3339Nano, expiresText)
		if err != nil {
			errs = append(errs, fmt.Errorf(
				"managed container %q has invalid expiry %q: %w",
				container.ID,
				expiresText,
				err,
			))
			continue
		}
		if now.Before(expiresAt) {
			continue
		}
		if strings.TrimSpace(container.ID) == "" {
			errs = append(errs, errors.New("managed expired container has empty ID"))
			continue
		}
		if err := backend.remove(ctx, container.ID); err != nil {
			errs = append(errs, fmt.Errorf("reap expired container %s: %w", container.ID, err))
			continue
		}
		backend.mu.Lock()
		delete(backend.definitions, container.ID)
		backend.mu.Unlock()
	}
	return errors.Join(errs...)
}

func hasManagedContainerName(names []string) bool {
	for _, name := range names {
		name = strings.TrimPrefix(strings.TrimSpace(name), "/")
		if strings.HasPrefix(name, "lpic-daily-") {
			return true
		}
	}
	return false
}

func (backend *Backend) resolveImageID(ctx context.Context, imageRef string) (string, error) {
	var response imageInspectResponse
	err := backend.doJSON(
		ctx,
		http.MethodGet,
		apiBase+"/images/"+url.PathEscape(imageRef)+"/json",
		nil,
		nil,
		&response,
	)
	var statusErr *statusError
	if errors.As(err, &statusErr) && statusErr.Code == http.StatusNotFound {
		return "", fmt.Errorf("required image %q is not present locally; implicit pulls are disabled", imageRef)
	}
	if err != nil {
		return "", fmt.Errorf("inspect local image %q: %w", imageRef, err)
	}
	id := strings.TrimSpace(response.ID)
	if matched, _ := regexp.MatchString("^sha256:[0-9a-f]{64}$", id); matched {
		return id, nil
	}
	if matched, _ := regexp.MatchString("^[0-9a-f]{64}$", id); matched {
		return "sha256:" + id, nil
	}
	return "", fmt.Errorf("Podman returned invalid immutable image ID %q for %q", id, imageRef)
}

func (backend *Backend) create(ctx context.Context, definition runner.Definition, name string) error {
	request, err := buildCreateRequest(definition, name)
	if err != nil {
		return err
	}
	request.Labels[expiresAtLabel] = time.Now().UTC().
		Add(definition.Timeout + abandonedCleanupGrace).
		Format(time.RFC3339Nano)
	var response createResponse
	if err := backend.doJSON(ctx, http.MethodPost, apiBase+"/containers/create", nil, request, &response); err != nil {
		return fmt.Errorf("create container %s: %w", name, err)
	}
	if response.ID == "" {
		return fmt.Errorf("create container %s: Podman returned an empty container ID", name)
	}
	return nil
}

func (backend *Backend) remove(ctx context.Context, name string) error {
	query := url.Values{
		"force":  {"true"},
		"v":      {"true"},
		"ignore": {"true"},
	}
	if err := backend.doJSON(
		ctx,
		http.MethodDelete,
		apiBase+"/containers/"+url.PathEscape(name),
		query,
		nil,
		nil,
	); err != nil {
		return fmt.Errorf("remove container %s: %w", name, err)
	}
	return nil
}

func buildCreateRequest(definition runner.Definition, name string) (createRequest, error) {
	profile, err := runner.Phase1CapabilityProfile(definition.CapabilityProfile)
	if err != nil {
		return createRequest{}, err
	}
	if definition.Network != runner.NetworkNone {
		return createRequest{}, fmt.Errorf("%w: network mode %q", runner.ErrNotSupported, definition.Network)
	}

	falseValue := false
	trueValue := true
	memoryBytes := int64(definition.MemoryMB) * 1024 * 1024
	cpuPeriod := uint64(100000)
	cpuQuota := int64(cpuPeriod) * int64(definition.CPUPercent) / 100
	mounts := make([]ociMount, 0, len(definition.WritableGuestPaths))
	for _, guestPath := range definition.WritableGuestPaths {
		mounts = append(mounts, ociMount{
			Destination: guestPath,
			Type:        "tmpfs",
			Source:      "tmpfs",
			Options: []string{
				"rw", "rprivate", "nosuid", "nodev", "tmpcopyup",
				fmt.Sprintf("size=%d", phase1WritablePathLimitBytes),
			},
		})
	}

	return createRequest{
		Name:         name,
		Image:        definition.ImageRef,
		RawImageName: definition.ImageRef,
		Command:      []string{"/usr/bin/sleep", "infinity"},
		EnvHost:      &falseValue,
		HTTPProxy:    &falseValue,
		Terminal:     &falseValue,
		Stdin:        &falseValue,
		Labels: map[string]string{
			managedLabel: "true",
			labIDLabel:   definition.LabID,
		},
		Timeout:            uint(definition.Timeout.Seconds()),
		Privileged:         &falseValue,
		CapAdd:             append([]string(nil), profile.Capabilities...),
		CapDrop:            []string{"ALL"},
		NoNewPrivileges:    &trueValue,
		NetNS:              namespace{NSMode: "none"},
		PidNS:              namespace{NSMode: "private"},
		UtsNS:              namespace{NSMode: "private"},
		IpcNS:              namespace{NSMode: "private"},
		ImageVolumeMode:    "ignore",
		ReadOnlyFilesystem: &trueValue,
		ReadWriteTmpfs:     &trueValue,
		Mounts:             mounts,
		ResourceLimits: &linuxResources{
			Memory: &linuxMemory{Limit: &memoryBytes},
			CPU:    &linuxCPU{Quota: &cpuQuota, Period: &cpuPeriod},
			Pids:   &linuxPids{Limit: int64(definition.PIDs)},
		},
	}, nil
}

type statusError struct {
	Code    int
	Method  string
	Path    string
	Message string
}

func (err *statusError) Error() string {
	if err.Message == "" {
		return fmt.Sprintf("Podman API %s %s returned HTTP %d", err.Method, err.Path, err.Code)
	}
	return fmt.Sprintf("Podman API %s %s returned HTTP %d: %s", err.Method, err.Path, err.Code, err.Message)
}

func (backend *Backend) doJSON(
	ctx context.Context,
	method string,
	apiPath string,
	query url.Values,
	body any,
	out any,
) error {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode Podman request: %w", err)
		}
		reader = bytes.NewReader(payload)
	}

	requestURL := "http://podman" + apiPath
	if len(query) != 0 {
		requestURL += "?" + query.Encode()
	}
	request, err := http.NewRequestWithContext(ctx, method, requestURL, reader)
	if err != nil {
		return fmt.Errorf("build Podman request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := backend.client.Do(request)
	if err != nil {
		return fmt.Errorf("call Podman service on %s: %w", backend.socketPath, err)
	}
	defer response.Body.Close()

	const maxResponse = 4 << 20
	payload, err := io.ReadAll(io.LimitReader(response.Body, maxResponse+1))
	if err != nil {
		return fmt.Errorf("read Podman response: %w", err)
	}
	if len(payload) > maxResponse {
		return errors.New("Podman response exceeded 4 MiB safety limit")
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var apiErr apiError
		_ = json.Unmarshal(payload, &apiErr)
		message := strings.TrimSpace(apiErr.Message)
		if message == "" {
			message = strings.TrimSpace(string(payload))
		}
		return &statusError{
			Code:    response.StatusCode,
			Method:  method,
			Path:    apiPath,
			Message: message,
		}
	}
	if out == nil || len(bytes.TrimSpace(payload)) == 0 {
		return nil
	}
	if err := json.Unmarshal(payload, out); err != nil {
		return fmt.Errorf("decode Podman response: %w", err)
	}
	return nil
}

func socketPathFromURI(uri string) (string, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("parse Podman URI: %w", err)
	}
	if parsed.Scheme != "unix" {
		return "", fmt.Errorf("unsupported Podman URI scheme %q: only unix:// is allowed", parsed.Scheme)
	}
	if parsed.Host != "" {
		return "", errors.New("unix Podman URI must not contain a host")
	}
	if parsed.Path == "" || !filepath.IsAbs(parsed.Path) {
		return "", errors.New("unix Podman URI requires an absolute socket path")
	}
	return filepath.Clean(parsed.Path), nil
}

func instanceName(labID string) (string, error) {
	var random [6]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", fmt.Errorf("generate instance name: %w", err)
	}
	slug := safeNamePart.ReplaceAllString(strings.ToLower(labID), "-")
	slug = edgeDashes.ReplaceAllString(slug, "")
	if slug == "" {
		slug = "lab"
	}
	if len(slug) > 40 {
		slug = slug[:40]
	}
	return "lpic-daily-" + slug + "-" + hex.EncodeToString(random[:]), nil
}
