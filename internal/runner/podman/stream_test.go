package podman

import (
	"bytes"
	"encoding/binary"
	"errors"
	"strings"
	"testing"
)

func TestDemultiplexDockerStream(t *testing.T) {
	var raw bytes.Buffer
	writeFrame := func(stream byte, payload string) {
		header := make([]byte, 8)
		header[0] = stream
		binary.BigEndian.PutUint32(header[4:], uint32(len(payload)))
		raw.Write(header)
		raw.WriteString(payload)
	}
	writeFrame(1, "hello\n")
	writeFrame(2, "warning\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := demultiplexDockerStream(&raw, &stdout, &stderr); err != nil {
		t.Fatalf("demultiplexDockerStream() error = %v", err)
	}
	if stdout.String() != "hello\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if stderr.String() != "warning\n" {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestDemultiplexRejectsOversizedFrame(t *testing.T) {
	var raw bytes.Buffer
	header := make([]byte, 8)
	header[0] = 1
	binary.BigEndian.PutUint32(header[4:], uint32((16<<20)+1))
	raw.Write(header)

	if err := demultiplexDockerStream(&raw, nil, nil); err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("error = %v, want frame-size error", err)
	}
}

func TestDemultiplexRejectsTruncatedHeader(t *testing.T) {
	err := demultiplexDockerStream(bytes.NewReader([]byte{1, 2, 3}), nil, nil)
	if err == nil || !errors.Is(err, errors.New("truncated Docker stream header")) && !strings.Contains(err.Error(), "truncated") {
		t.Fatalf("error = %v, want truncated-header error", err)
	}
}
