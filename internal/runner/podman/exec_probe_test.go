package podman

import (
	"archive/tar"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Loe159/lpic-daily/internal/runner"
)

func TestExecDetachedReturnsExitCode(t *testing.T) {
	var inspectCalls atomic.Int32
	backend, stop := openFakeBackend(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == compatBase+"/containers/ctr/exec":
			var request execCreateRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("decode exec create: %v", err)
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			if request.Privileged || request.Tty || request.AttachStdin || request.AttachStdout || request.AttachStderr {
				t.Errorf("unsafe/unexpected exec request: %#v", request)
			}
			if got := strings.Join(request.Cmd, " "); got != "/bin/false --example" {
				t.Errorf("command = %q", got)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"Id":"exec-1"}`)
		case r.Method == http.MethodPost && r.URL.Path == compatBase+"/exec/exec-1/start":
			var request execStartRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("decode exec start: %v", err)
			}
			if !request.Detach || request.Tty {
				t.Errorf("start request = %#v", request)
			}
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodGet && r.URL.Path == compatBase+"/exec/exec-1/json":
			call := inspectCalls.Add(1)
			w.Header().Set("Content-Type", "application/json")
			if call == 1 {
				_, _ = io.WriteString(w, `{"ID":"exec-1","Running":true,"ExitCode":0}`)
				return
			}
			_, _ = io.WriteString(w, `{"ID":"exec-1","Running":false,"ExitCode":7}`)
		default:
			http.NotFound(w, r)
		}
	})
	defer stop()

	result, err := backend.Exec(context.Background(), runner.Instance{ID: "ctr"}, runner.ExecRequest{
		Argv: []string{"/bin/false", "--example"},
		Env: map[string]string{
			"Z_LAST":  "z",
			"A_FIRST": "a",
		},
	})
	if err != nil {
		t.Fatalf("Exec() error = %v", err)
	}
	if result.ExitCode != 7 {
		t.Fatalf("exit code = %d, want 7", result.ExitCode)
	}
	if inspectCalls.Load() < 2 {
		t.Fatalf("inspect calls = %d, want >= 2", inspectCalls.Load())
	}
}

func TestExecRejectsAttachedOrTTYBeforeCallingPodman(t *testing.T) {
	backend, stop := openFakeBackend(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected Podman call %s %s", r.Method, r.URL.Path)
		http.Error(w, "unexpected", http.StatusInternalServerError)
	})
	defer stop()

	_, err := backend.Exec(context.Background(), runner.Instance{ID: "ctr"}, runner.ExecRequest{
		Argv: []string{"/bin/bash"},
		TTY:  true,
	})
	if !errors.Is(err, runner.ErrNotSupported) {
		t.Fatalf("Exec() error = %v, want ErrNotSupported", err)
	}
}

func TestStatAndReadFileUseContainerArchive(t *testing.T) {
	backend, stop := openFakeBackend(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !strings.HasSuffix(r.URL.Path, "/archive") {
			http.NotFound(w, r)
			return
		}
		switch r.URL.Query().Get("path") {
		case "/srv/shared":
			writeTar(t, w, tar.Header{
				Name:     "shared",
				Mode:     0o3770,
				Typeflag: tar.TypeDir,
				Uid:      0,
				Gid:      2000,
				Uname:    "root",
				Gname:    "project",
			}, nil)
		case "/tmp/message":
			writeTar(t, w, tar.Header{
				Name:     "message",
				Mode:     0o640,
				Typeflag: tar.TypeReg,
				Size:     5,
				Uid:      1101,
				Gid:      2000,
				Uname:    "alice",
				Gname:    "project",
			}, []byte("hello"))
		default:
			http.NotFound(w, r)
		}
	})
	defer stop()

	instance := runner.Instance{ID: "ctr"}
	info, err := backend.Stat(context.Background(), instance, "/srv/shared")
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if got := info.Mode & 0o7777; got != 0o3770 {
		t.Fatalf("mode = %04o, want 3770", got)
	}
	if info.User != "root" || info.Group != "project" || info.UID != 0 || info.GID != 2000 || !info.IsDir {
		t.Fatalf("stat info = %#v", info)
	}

	content, err := backend.ReadFile(context.Background(), instance, "/tmp/message", 16)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(content) != "hello" {
		t.Fatalf("content = %q, want hello", content)
	}
}

func TestReadFileEnforcesLimit(t *testing.T) {
	backend, stop := openFakeBackend(t, func(w http.ResponseWriter, r *http.Request) {
		writeTar(t, w, tar.Header{
			Name:     "large",
			Mode:     0o600,
			Typeflag: tar.TypeReg,
			Size:     5,
		}, []byte("hello"))
	})
	defer stop()

	_, err := backend.ReadFile(context.Background(), runner.Instance{ID: "ctr"}, "/tmp/large", 4)
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("ReadFile() error = %v, want size-limit error", err)
	}
}

func TestProcessesParsesStableDescriptors(t *testing.T) {
	backend, stop := openFakeBackend(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !strings.HasSuffix(r.URL.Path, "/top") {
			http.NotFound(w, r)
			return
		}
		if got := r.URL.Query().Get("ps_args"); got != "pid,comm,args" {
			t.Errorf("ps_args = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
			"Titles":["PID","COMMAND","ARGS"],
			"Processes":[
				["1","sleep","sleep infinity"],
				["42","worker","worker --safe"]
			]
		}`)
	})
	defer stop()

	processes, err := backend.Processes(context.Background(), runner.Instance{ID: "ctr"})
	if err != nil {
		t.Fatalf("Processes() error = %v", err)
	}
	if len(processes) != 2 {
		t.Fatalf("process count = %d, want 2", len(processes))
	}
	if processes[1].PID != 42 || processes[1].Command != "worker" || len(processes[1].Args) != 2 {
		t.Fatalf("process = %#v", processes[1])
	}
}

func TestEncodeEnvironmentIsDeterministic(t *testing.T) {
	got, err := encodeEnvironment(map[string]string{"Z": "2", "A": "1"})
	if err != nil {
		t.Fatalf("encodeEnvironment() error = %v", err)
	}
	if strings.Join(got, ",") != "A=1,Z=2" {
		t.Fatalf("environment = %v", got)
	}
	if _, err := encodeEnvironment(map[string]string{"BAD=KEY": "x"}); err == nil {
		t.Fatal("invalid environment key unexpectedly accepted")
	}
}

func openFakeBackend(t *testing.T, handler http.HandlerFunc) (*Backend, func()) {
	t.Helper()
	socket, stop := fakePodmanSocket(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == apiBase+"/info" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"host":{"cgroupVersion":"v2","security":{"rootless":true}}}`)
			return
		}
		handler(w, r)
	})
	backend, err := Open(context.Background(), "unix://"+socket)
	if err != nil {
		stop()
		t.Fatalf("Open() error = %v", err)
	}
	return backend, stop
}

func writeTar(t *testing.T, w http.ResponseWriter, header tar.Header, content []byte) {
	t.Helper()
	w.Header().Set("Content-Type", "application/x-tar")
	writer := tar.NewWriter(w)
	if err := writer.WriteHeader(&header); err != nil {
		t.Errorf("WriteHeader: %v", err)
		return
	}
	if len(content) != 0 {
		if _, err := writer.Write(content); err != nil {
			t.Errorf("Write tar content: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Errorf("Close tar writer: %v", err)
	}
}

func TestLookupIdentityFallsBackToNumericID(t *testing.T) {
	backend, stop := openFakeBackend(t, func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	defer stop()

	if got := backend.lookupIdentity(context.Background(), runner.Instance{ID: "ctr"}, "/etc/passwd", 4242); got != strconv.Itoa(4242) {
		t.Fatalf("lookup = %q, want numeric fallback", got)
	}
}

func TestExecHonorsContextCancellationWhilePolling(t *testing.T) {
	backend, stop := openFakeBackend(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/exec") && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"Id":"exec-loop"}`)
		case strings.HasSuffix(r.URL.Path, "/start") && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusOK)
		case strings.HasSuffix(r.URL.Path, "/json") && r.Method == http.MethodGet:
			_, _ = io.WriteString(w, `{"ID":"exec-loop","Running":true,"ExitCode":0}`)
		default:
			http.NotFound(w, r)
		}
	})
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel()
	_, err := backend.Exec(ctx, runner.Instance{ID: "ctr"}, runner.ExecRequest{Argv: []string{"/bin/sleep", "10"}})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Exec() error = %v, want context deadline", err)
	}
}
