package appstate

import (
	"errors"
	"os"
	"path/filepath"
)

func ProgressDBPath() (string, error) {
	base, err := stateBase()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(base, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(base, "progress.sqlite"), nil
}

func VMStateRoot() (string, error) {
	base, err := stateBase()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "vms"), nil
}

func VMImageRoot() (string, error) {
	if override := os.Getenv("LPIC_DAILY_VM_IMAGE_DIR"); override != "" {
		if !filepath.IsAbs(override) {
			return "", errors.New("LPIC_DAILY_VM_IMAGE_DIR must be absolute")
		}
		return filepath.Clean(override), nil
	}

	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if home == "" {
			return "", errors.New("home directory is empty")
		}
		base = filepath.Join(home, ".local", "share")
	}
	if !filepath.IsAbs(base) {
		return "", errors.New("XDG_DATA_HOME must be absolute")
	}
	return filepath.Join(filepath.Clean(base), "lpic-daily", "vm-images"), nil
}

func stateBase() (string, error) {
	if override := os.Getenv("LPIC_DAILY_STATE_DIR"); override != "" {
		if !filepath.IsAbs(override) {
			return "", errors.New("LPIC_DAILY_STATE_DIR must be absolute")
		}
		return filepath.Clean(override), nil
	}

	base := os.Getenv("XDG_STATE_HOME")
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
	if !filepath.IsAbs(base) {
		return "", errors.New("XDG_STATE_HOME must be absolute")
	}
	return filepath.Join(filepath.Clean(base), "lpic-daily"), nil
}
