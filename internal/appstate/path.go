package appstate

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
)

const defaultVMStorageBase = "/var/lib/libvirt/images/lpic-daily"

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

func VMStorageRoot() string {
	return filepath.Join(defaultVMStorageBase, strconv.Itoa(os.Geteuid()))
}

func VMNetworkAllocationLockPath() string {
	return filepath.Join(defaultVMStorageBase, ".network-allocation.lock")
}

func VMStateRoot() (string, error) {
	if override := os.Getenv("LPIC_DAILY_VM_STATE_DIR"); override != "" {
		if !filepath.IsAbs(override) {
			return "", errors.New("LPIC_DAILY_VM_STATE_DIR must be absolute")
		}
		return filepath.Clean(override), nil
	}
	return filepath.Join(VMStorageRoot(), "state"), nil
}

func VMImageRoot() (string, error) {
	if override := os.Getenv("LPIC_DAILY_VM_IMAGE_DIR"); override != "" {
		if !filepath.IsAbs(override) {
			return "", errors.New("LPIC_DAILY_VM_IMAGE_DIR must be absolute")
		}
		return filepath.Clean(override), nil
	}
	return filepath.Join(VMStorageRoot(), "images"), nil
}

func VMImageCatalogPath() (string, error) {
	root, err := VMImageRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "catalog.json"), nil
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
