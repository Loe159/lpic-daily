//go:build linux

package terminal

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

type State struct {
	termios unix.Termios
}

func IsTerminal(file *os.File) bool {
	if file == nil {
		return false
	}
	_, err := unix.IoctlGetTermios(int(file.Fd()), unix.TCGETS)
	return err == nil
}

func MakeRaw(file *os.File) (*State, error) {
	if file == nil {
		return nil, errors.New("terminal file is required")
	}

	fd := int(file.Fd())
	current, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return nil, fmt.Errorf("read terminal state: %w", err)
	}

	raw := *current
	raw.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP | unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON
	raw.Oflag &^= unix.OPOST
	raw.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN
	raw.Cflag &^= unix.CSIZE | unix.PARENB
	raw.Cflag |= unix.CS8
	raw.Cc[unix.VMIN] = 1
	raw.Cc[unix.VTIME] = 0

	if err := unix.IoctlSetTermios(fd, unix.TCSETS, &raw); err != nil {
		return nil, fmt.Errorf("enable raw terminal mode: %w", err)
	}
	return &State{termios: *current}, nil
}

func Restore(file *os.File, state *State) error {
	if file == nil {
		return errors.New("terminal file is required")
	}
	if state == nil {
		return errors.New("terminal state is required")
	}
	if err := unix.IoctlSetTermios(int(file.Fd()), unix.TCSETS, &state.termios); err != nil {
		return fmt.Errorf("restore terminal state: %w", err)
	}
	return nil
}

func Size(file *os.File) (width, height uint, err error) {
	if file == nil {
		return 0, 0, errors.New("terminal file is required")
	}
	size, err := unix.IoctlGetWinsize(int(file.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return 0, 0, fmt.Errorf("read terminal size: %w", err)
	}
	if size.Col == 0 || size.Row == 0 {
		return 0, 0, errors.New("terminal reported a zero-sized window")
	}
	return uint(size.Col), uint(size.Row), nil
}
