//go:build linux

package terminal_test

import (
	"os"
	"testing"

	"github.com/Loe159/lpic-daily/internal/terminal"
)

func TestPipeIsNotTerminal(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	defer reader.Close()
	defer writer.Close()

	if terminal.IsTerminal(reader) {
		t.Fatal("pipe unexpectedly detected as terminal")
	}
	if _, _, err := terminal.Size(reader); err == nil {
		t.Fatal("Size(pipe) unexpectedly succeeded")
	}
	if _, err := terminal.MakeRaw(reader); err == nil {
		t.Fatal("MakeRaw(pipe) unexpectedly succeeded")
	}
}
