package podman

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Loe159/lpic-daily/internal/lab"
)

type call struct {
	executable string
	args       []string
}

type fakeExecutor struct {
	calls   []call
	results []Result
	errors  []error
}

func (f *fakeExecutor) Run(_ context.Context, executable string, args ...string) (Result, error) {
	f.calls = append(f.calls, call{executable: executable, args: append([]string(nil), args...)})
	index := len(f.calls) - 1
	var result Result
	if index < len(f.results) {
		result = f.results[index]
	}
	var err error
	if index < len(f.errors) {
		err = f.errors[index]
	}
	return result, err
}

func validSpec() lab.Spec {
	return lab.Spec{
		ID:                "permissions-1",
		ImageRef:          "localhost/lpic-fedora@sha256:" + strings.Repeat("a", 64),
		Network:           lab.NetworkNone,
		CapabilityProfile: lab.ProfilePermissions,
		Resources: lab.Resources{
			MemoryMB:       256,
			PIDs:           64,
			TimeoutSeconds: 900,
		},
	}
}

func TestDoctorRequiresRootless(t *testing.T) {
	fake := &fakeExecutor{results: []Result{{Stdout: "false\n"}}}
	runner := New(fake)
	if err := runner.Doctor(context.Background()); err == nil {
		t.Fatal("expected non-rootless Podman to be rejected")
	}
}

func TestPrepareBuildsFailClosedSandbox(t *testing.T) {
	fake := &fakeExecutor{results: []Result{{Stdout: "true\n"}, {Stdout: "container-123\n"}}}
	runner := New(fake)
	runner.newID = func() (string, error) { return "session123", nil }
	runner.now = func() time.Time { return time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC) }

	session, err := runner.Prepare(context.Background(), validSpec())
	if err != nil {
		t.Fatal(err)
	}
	if session.ContainerID != "container-123" || session.ID != "session123" {
		t.Fatalf("unexpected session: %#v", session)
	}
	if len(fake.calls) != 2 {
		t.Fatalf("calls=%d, want 2", len(fake.calls))
	}
	args := fake.calls[1].args
	joined := strings.Join(args, " ")
	for _, required := range []string{
		"--pull=never",
		"--network=none",
		"--ipc=private",
		"--pid=private",
		"--cap-drop=ALL",
		"--security-opt=no-new-privileges",
		"--read-only",
		"--pids-limit 64",
		"--memory 256m",
		"--cpus 1",
		"--cap-add=CHOWN",
		"--cap-add=SETUID",
	} {
		if !strings.Contains(joined, required) {
			t.Fatalf("create args missing %q: %s", required, joined)
		}
	}
	for _, forbidden := range []string{"--privileged", "--volume", " -v ", "/var/run/docker.sock", "/run/podman/podman.sock"} {
		if strings.Contains(" "+joined+" ", forbidden) {
			t.Fatalf("create args contain forbidden %q: %s", forbidden, joined)
		}
	}
}

func TestPrepareRejectsNetworkBeforeCallingPodman(t *testing.T) {
	fake := &fakeExecutor{}
	runner := New(fake)
	spec := validSpec()
	spec.Network = lab.NetworkIsolated
	if _, err := runner.Prepare(context.Background(), spec); err == nil {
		t.Fatal("expected isolated network to be rejected until a dedicated network manager exists")
	}
	if len(fake.calls) != 0 {
		t.Fatalf("Podman should not be called, calls=%#v", fake.calls)
	}
}

func TestPrepareRejectsUnpinnedImage(t *testing.T) {
	fake := &fakeExecutor{}
	runner := New(fake)
	spec := validSpec()
	spec.ImageRef = "fedora:latest"
	if _, err := runner.Prepare(context.Background(), spec); err == nil {
		t.Fatal("expected unpinned image to be rejected")
	}
	if len(fake.calls) != 0 {
		t.Fatal("Podman should not be called for invalid spec")
	}
}

func TestExecUsesStructuredArgv(t *testing.T) {
	fake := &fakeExecutor{results: []Result{{Stdout: "ok\n"}}}
	runner := New(fake)
	session := lab.Session{ID: "s1", ContainerID: "c1", Spec: validSpec()}
	argv := []string{"printf", "%s", "hello; rm -rf /"}
	result, err := runner.Exec(context.Background(), session, argv)
	if err != nil {
		t.Fatal(err)
	}
	if result.Stdout != "ok\n" {
		t.Fatalf("stdout=%q", result.Stdout)
	}
	want := []string{"exec", "--", "c1", "printf", "%s", "hello; rm -rf /"}
	if !reflect.DeepEqual(fake.calls[0].args, want) {
		t.Fatalf("args=%#v, want %#v", fake.calls[0].args, want)
	}
}

func TestResetDestroysThenCreatesFreshContainer(t *testing.T) {
	fake := &fakeExecutor{results: []Result{
		{},
		{Stdout: "true\n"},
		{Stdout: "new-container\n"},
	}}
	runner := New(fake)
	runner.newID = func() (string, error) { return "new-session", nil }
	runner.now = func() time.Time { return time.Unix(1000, 0).UTC() }
	old := lab.Session{ID: "old", ContainerID: "old-container", Spec: validSpec()}

	fresh, err := runner.Reset(context.Background(), old)
	if err != nil {
		t.Fatal(err)
	}
	if fresh.ContainerID != "new-container" || fresh.ID != "new-session" {
		t.Fatalf("unexpected fresh session: %#v", fresh)
	}
	if got := fake.calls[0].args; !reflect.DeepEqual(got, []string{"rm", "--force", "--time", "1", "old-container"}) {
		t.Fatalf("destroy args=%#v", got)
	}
}

func TestDoctorPropagatesMissingPodman(t *testing.T) {
	fake := &fakeExecutor{errors: []error{errors.New("executable not found")}}
	runner := New(fake)
	if err := runner.Doctor(context.Background()); err == nil {
		t.Fatal("expected missing Podman to fail closed")
	}
}
