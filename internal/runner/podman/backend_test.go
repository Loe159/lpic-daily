package podman

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Loe159/lpic-daily/internal/runner"
)

func validDefinition() runner.Definition {
	return runner.Definition{
		LabID:              "104.5.shared-dropbox",
		ImageRef:           "localhost/lpic-daily/fedora-phase1:1",
		Distribution:       "fedora",
		Network:            runner.NetworkNone,
		CapabilityProfile:  "identity-files",
		WritableGuestPaths: []string{"/srv/shared", "/home/alice", "/home/bob"},
		MemoryMB:           256,
		CPUPercent:         100,
		PIDs:               128,
		Timeout:            20 * time.Minute,
	}
}

func TestBuildCreateRequestIsFailClosed(t *testing.T) {
	definition := validDefinition()
	request, err := buildCreateRequest(definition, "lpic-daily-test")
	if err != nil {
		t.Fatalf("buildCreateRequest() error = %v", err)
	}
	if request.Privileged == nil || *request.Privileged {
		t.Fatal("container must explicitly be non-privileged")
	}
	if request.NoNewPrivileges == nil || !*request.NoNewPrivileges {
		t.Fatal("no-new-privileges must be enabled")
	}
	if len(request.CapDrop) != 1 || request.CapDrop[0] != "ALL" {
		t.Fatalf("cap drop = %v, want [ALL]", request.CapDrop)
	}
	if request.NetNS.NSMode != "none" {
		t.Fatalf("network namespace = %q, want none", request.NetNS.NSMode)
	}
	if request.PidNS.NSMode != "private" || request.IpcNS.NSMode != "private" || request.UtsNS.NSMode != "private" {
		t.Fatal("PID/IPC/UTS namespaces must be private")
	}
	if request.ImageVolumeMode != "ignore" {
		t.Fatalf("image volume mode = %q, want ignore", request.ImageVolumeMode)
	}
	if request.ReadOnlyFilesystem == nil || !*request.ReadOnlyFilesystem {
		t.Fatal("container rootfs must be read-only")
	}
	if request.ReadWriteTmpfs == nil || !*request.ReadWriteTmpfs {
		t.Fatal("standard runtime tmpfs mounts must remain writable")
	}
	if len(request.Mounts) != len(definition.WritableGuestPaths) {
		t.Fatalf("tmpfs mounts = %d, want %d", len(request.Mounts), len(definition.WritableGuestPaths))
	}
	for index, mount := range request.Mounts {
		if mount.Destination != definition.WritableGuestPaths[index] || mount.Type != "tmpfs" || mount.Source != "tmpfs" {
			t.Fatalf("unsafe writable mount = %#v", mount)
		}
		if !strings.Contains(strings.Join(mount.Options, ","), fmt.Sprintf("size=%d", phase1WritablePathLimitBytes)) {
			t.Fatalf("tmpfs mount is not size-bounded: %#v", mount)
		}
	}
	if request.ResourceLimits == nil || request.ResourceLimits.Memory == nil || request.ResourceLimits.Memory.Limit == nil {
		t.Fatal("memory limit is required")
	}
	if got, want := *request.ResourceLimits.Memory.Limit, int64(256*1024*1024); got != want {
		t.Fatalf("memory limit = %d, want %d", got, want)
	}
	if request.ResourceLimits.CPU == nil ||
		request.ResourceLimits.CPU.Quota == nil ||
		request.ResourceLimits.CPU.Period == nil {
		t.Fatal("CPU quota/period limit is required")
	}
	if got, want := *request.ResourceLimits.CPU.Period, uint64(100000); got != want {
		t.Fatalf("CPU period = %d, want %d", got, want)
	}
	if got, want := *request.ResourceLimits.CPU.Quota, int64(100000); got != want {
		t.Fatalf("CPU quota = %d, want %d for 100%% CPU", got, want)
	}
	if request.ResourceLimits.Pids == nil || request.ResourceLimits.Pids.Limit != 128 {
		t.Fatal("PID limit is required")
	}

	payload, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	for _, forbidden := range []string{"volumes", "devices", "host_device_list"} {
		if strings.Contains(string(payload), `"`+forbidden+`"`) {
			t.Fatalf("request unexpectedly contains %s: %s", forbidden, payload)
		}
	}
}

func TestPrepareRejectsMachineSettings(t *testing.T) {
	socket, stop := fakePodmanSocket(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == apiBase+"/info" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"host":{"cgroupVersion":"v2","security":{"rootless":true}}}`))
			return
		}
		t.Errorf("unexpected Podman call after machine settings should have been rejected: %s %s", r.Method, r.URL.Path)
		http.Error(w, "unexpected", http.StatusInternalServerError)
	})
	defer stop()

	backend, err := Open(context.Background(), "unix://"+socket)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	definition := validDefinition()
	definition.Machine = &runner.MachineDefinition{Firmware: runner.FirmwareUEFI}
	if _, err := backend.Prepare(context.Background(), definition); !errors.Is(err, runner.ErrNotSupported) {
		t.Fatalf("Prepare() error = %v, want ErrNotSupported", err)
	}
}

func TestBuildCreateRequestRejectsIsolatedUntilImplemented(t *testing.T) {
	definition := validDefinition()
	definition.Network = runner.NetworkIsolated
	_, err := buildCreateRequest(definition, "test")
	if !errors.Is(err, runner.ErrNotSupported) {
		t.Fatalf("error = %v, want ErrNotSupported", err)
	}
}

func TestSocketPathRejectsRemoteSchemes(t *testing.T) {
	for _, uri := range []string{
		"tcp://127.0.0.1:8080",
		"ssh://host/run/user/1000/podman.sock",
		"http://localhost",
		"unix://relative.sock",
	} {
		if _, err := socketPathFromURI(uri); err == nil {
			t.Fatalf("socketPathFromURI(%q) unexpectedly succeeded", uri)
		}
	}
}

func TestDefaultURIUsesXDGRuntimeDir(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", "/run/user/4242")
	if got, want := DefaultURI(), "unix:///run/user/4242/podman/podman.sock"; got != want {
		t.Fatalf("DefaultURI() = %q, want %q", got, want)
	}
}

func TestInstanceNameIsSafeAndNamespaced(t *testing.T) {
	name, err := instanceName("../../Shared DropBox !!")
	if err != nil {
		t.Fatalf("instanceName() error = %v", err)
	}
	if !strings.HasPrefix(name, "lpic-daily-") {
		t.Fatalf("name = %q", name)
	}
	if strings.ContainsAny(name, "/ !") {
		t.Fatalf("unsafe container name = %q", name)
	}
}

func TestOpenRejectsRootfulAndAcceptsRootlessV2(t *testing.T) {
	tests := []struct {
		name    string
		info    string
		wantErr string
	}{
		{
			name:    "rootful rejected",
			info:    `{"host":{"cgroupVersion":"v2","security":{"rootless":false}}}`,
			wantErr: "refusing rootful",
		},
		{
			name:    "cgroups v1 rejected",
			info:    `{"host":{"cgroupVersion":"v1","security":{"rootless":true}}}`,
			wantErr: "requires cgroups v2",
		},
		{
			name: "rootless v2 accepted",
			info: `{"host":{"cgroupVersion":"v2","security":{"rootless":true}}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			socket, stop := fakePodmanSocket(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != apiBase+"/info" {
					http.NotFound(w, r)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.info))
			})
			defer stop()

			backend, err := Open(context.Background(), "unix://"+socket)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Open() error = %v", err)
				}
				if backend == nil {
					t.Fatal("Open() returned nil backend")
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Open() error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestManagedContainerLifecyclePrepareStartResetDestroy(t *testing.T) {
	var (
		createCalls int
		startCalls  int
		deleteCalls int
	)

	socket, stop := fakePodmanSocket(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == apiBase+"/info":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"host":{"cgroupVersion":"v2","security":{"rootless":true}}}`))
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, apiBase+"/images/") && strings.HasSuffix(r.URL.Path, "/exists"):
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && r.URL.Path == apiBase+"/containers/create":
			createCalls++
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Id":"podman-container-id"}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/start"):
			startCalls++
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, apiBase+"/containers/"):
			deleteCalls++
			if r.URL.Query().Get("force") != "true" || r.URL.Query().Get("ignore") != "true" {
				t.Errorf("unsafe delete query: %s", r.URL.RawQuery)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected Podman request: %s %s", r.Method, r.URL.String())
			http.Error(w, "unexpected", http.StatusNotFound)
		}
	})
	defer stop()

	ctx := context.Background()
	backend, err := Open(ctx, "unix://"+socket)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	instance, err := backend.Prepare(ctx, validDefinition())
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if instance.ID == "" {
		t.Fatal("Prepare() returned empty instance ID")
	}
	if err := backend.Start(ctx, instance); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := backend.Reset(ctx, instance); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}
	if err := backend.Destroy(ctx, instance); err != nil {
		t.Fatalf("Destroy() error = %v", err)
	}

	if createCalls != 2 {
		t.Fatalf("create calls = %d, want 2 (prepare + reset)", createCalls)
	}
	if startCalls != 1 {
		t.Fatalf("start calls = %d, want 1", startCalls)
	}
	if deleteCalls != 2 {
		t.Fatalf("delete calls = %d, want 2 (reset + destroy)", deleteCalls)
	}
	if err := backend.Reset(ctx, instance); err == nil || !strings.Contains(err.Error(), "unknown managed instance") {
		t.Fatalf("Reset() after Destroy error = %v, want unknown managed instance", err)
	}
}

func fakePodmanSocket(t *testing.T, handler http.HandlerFunc) (string, func()) {
	t.Helper()
	socket := filepath.Join(t.TempDir(), "podman.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatalf("listen unix socket: %v", err)
	}
	server := &http.Server{Handler: handler}
	done := make(chan struct{})
	go func() {
		_ = server.Serve(listener)
		close(done)
	}()
	return socket, func() {
		_ = server.Close()
		_ = listener.Close()
		<-done
		_ = os.Remove(socket)
	}
}
