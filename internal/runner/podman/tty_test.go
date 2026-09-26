package podman

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/Loe159/lpic-daily/internal/runner"
)

func TestTTYExecUsesHijackedRawStreamAndResize(t *testing.T) {
	var (
		mu          sync.Mutex
		resizeQuery string
	)
	backend, stop := openFakeBackend(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == compatBase+"/containers/ctr/exec":
			var request execCreateRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("decode exec create: %v", err)
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			if !request.Tty || !request.AttachStdin || !request.AttachStdout || !request.AttachStderr {
				t.Errorf("TTY create request = %#v", request)
			}
			if request.Privileged {
				t.Error("TTY exec must not be privileged")
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"Id":"exec-tty"}`)
		case r.Method == http.MethodPost && r.URL.Path == compatBase+"/exec/exec-tty/resize":
			mu.Lock()
			resizeQuery = r.URL.RawQuery
			mu.Unlock()
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodPost && r.URL.Path == compatBase+"/exec/exec-tty/start":
			if !strings.EqualFold(r.Header.Get("Connection"), "upgrade") || !strings.EqualFold(r.Header.Get("Upgrade"), "tcp") {
				t.Errorf("missing upgrade headers: %#v", r.Header)
			}
			hijacker, ok := w.(http.Hijacker)
			if !ok {
				t.Error("test server does not support hijacking")
				return
			}
			conn, rw, err := hijacker.Hijack()
			if err != nil {
				t.Errorf("hijack: %v", err)
				return
			}
			defer conn.Close()
			_, _ = rw.WriteString("HTTP/1.1 101 UPGRADED\r\nConnection: Upgrade\r\nUpgrade: tcp\r\n\r\n")
			_ = rw.Flush()

			input := make([]byte, 5)
			if _, err := io.ReadFull(conn, input); err != nil {
				t.Errorf("read TTY stdin: %v", err)
				return
			}
			if string(input) != "exit\n" {
				t.Errorf("TTY stdin = %q", input)
			}
			_, _ = io.WriteString(conn, "PTY READY\r\n")
		case r.Method == http.MethodGet && r.URL.Path == compatBase+"/exec/exec-tty/json":
			_, _ = io.WriteString(w, `{"ID":"exec-tty","Running":false,"ExitCode":0}`)
		default:
			http.NotFound(w, r)
		}
	})
	defer stop()

	var stdout bytes.Buffer
	result, err := backend.Exec(context.Background(), runner.Instance{ID: "ctr"}, runner.ExecRequest{
		Argv:        []string{"/usr/bin/bash", "-l"},
		Stdin:       strings.NewReader("exit\n"),
		Stdout:      &stdout,
		Stderr:      &stdout,
		TTY:         true,
		InitialSize: runner.TerminalSize{Width: 120, Height: 40},
	})
	if err != nil {
		t.Fatalf("Exec() error = %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("exit code = %d, want 0", result.ExitCode)
	}
	if stdout.String() != "PTY READY\r\n" {
		t.Fatalf("TTY stdout = %q", stdout.String())
	}
	mu.Lock()
	gotResize := resizeQuery
	mu.Unlock()
	if gotResize != "h=40&w=120" {
		t.Fatalf("resize query = %q, want h=40&w=120", gotResize)
	}
}

func TestExecRejectsNonTTYStdin(t *testing.T) {
	backend, stop := openFakeBackend(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected Podman call %s %s", r.Method, r.URL.Path)
		http.Error(w, "unexpected", http.StatusInternalServerError)
	})
	defer stop()

	_, err := backend.Exec(context.Background(), runner.Instance{ID: "ctr"}, runner.ExecRequest{
		Argv:  []string{"/bin/cat"},
		Stdin: strings.NewReader("input"),
	})
	if err == nil || !strings.Contains(err.Error(), "non-TTY stdin") {
		t.Fatalf("Exec() error = %v, want non-TTY stdin error", err)
	}
}

func TestTTYExecRejectsInvalidResize(t *testing.T) {
	request := runner.ExecRequest{
		Argv:        []string{"/bin/bash"},
		TTY:         true,
		InitialSize: runner.TerminalSize{Width: 80},
	}
	if err := request.Validate(); err == nil {
		t.Fatal("partial terminal size unexpectedly validated")
	}
}
