package appstate

import (
	"errors"
	"os"
	"path/filepath"
)

func ProgressDBPath() (string, error) {
	base := os.Getenv("LPIC_DAILY_STATE_DIR")
	if base == "" {
		base = os.Getenv("XDG_STATE_HOME")
		if base == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			if home == "" {
				return "", errors.New("home directory is empty")
			}
			base = filepath.Join(home, ".local", "state")
		}
		base = filepath.Join(base, "lpic-daily")
	}
	base = filepath.Clean(base)
	if err := os.MkdirAll(base, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(base, "progress.sqlite"), nil
}
