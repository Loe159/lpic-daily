package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLabCheckBridgeRunsInChildAndReturnsDetailedOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "check.sock")
	calls := 0
	bridge, err := startLabCheckBridge(path, func(out io.Writer, reflection []string) (bool, error) {
		calls++
		fmt.Fprintln(out, "OK          SERVICE_MODE est exportée")
		fmt.Fprintln(out, "À CORRIGER   LOCAL_NOTE ne doit pas être exportée")
		return false, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer bridge.Close()

	result, err := requestLabCheck(context.Background(), path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || result.Passed || result.NeedsReflection ||
		!strings.Contains(result.Output, "LOCAL_NOTE ne doit pas être exportée") {
		t.Fatalf("unexpected remote check: calls=%d result=%+v", calls, result)
	}
}

func TestLabCheckBridgeSupportsLegacyReflectionFromParent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "check.sock")
	bridge, err := startLabCheckBridge(path, func(out io.Writer, reflection []string) (bool, error) {
		if len(reflection) != 2 {
			return false, errLabReflectionRequired
		}
		fmt.Fprintln(out, "Validation terminée.")
		return true, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer bridge.Close()

	initial, err := requestLabCheck(context.Background(), path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !initial.NeedsReflection || initial.Passed {
		t.Fatalf("missing request for reflection: %+v", initial)
	}
	result, err := requestLabCheck(context.Background(), path, []string{"observation", "explication"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Passed || result.Error != "" || !strings.Contains(result.Output, "Validation terminée") {
		t.Fatalf("remote reflection failed: %+v", result)
	}
}

func TestLabShellCommandReaderShowsCommandAndKeepsShellActive(t *testing.T) {
	var echoed bytes.Buffer
	commands := 0
	reader := &labShellCommandReader{
		input:     strings.NewReader(":check\r:checkx\b\r:quit\r"),
		echo:      &echoed,
		lineStart: true,
		onCommand: func(command string) (bool, error) {
			if command == ":check" {
				commands++
				return false, nil
			}
			if command == ":quit" {
				return true, nil
			}
			return false, errors.New("unexpected command")
		},
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if commands != 2 || string(data) != "\r\rexit\r" {
		t.Fatalf("shell should remain live after both checks: checks=%d input=%q", commands, data)
	}
	if !strings.Contains(echoed.String(), ":check\r\n") ||
		!strings.Contains(echoed.String(), ":checkx\b \b\r\n") {
		t.Fatalf("typed host command was not visible: %q", echoed.String())
	}
}

func TestLabCheckBridgeAllowsSlowChildStartup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "check.sock")
	ready := make(chan struct{})
	go func() {
		time.Sleep(3500 * time.Millisecond)
		bridge, err := startLabCheckBridge(path, func(out io.Writer, reflection []string) (bool, error) {
			fmt.Fprintln(out, "Lab disponible.")
			return true, nil
		})
		if err == nil {
			defer bridge.Close()
		}
		close(ready)
		if err == nil {
			time.Sleep(2 * time.Second)
		}
	}()

	result, err := requestLabCheck(context.Background(), path, nil)
	if err != nil {
		t.Fatalf("slow child startup should not fail: %v", err)
	}
	if !result.Passed {
		t.Fatalf("unexpected check result: %+v", result)
	}
	<-ready
}
