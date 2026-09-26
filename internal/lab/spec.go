package lab

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type NetworkMode string

const (
	NetworkNone     NetworkMode = "none"
	NetworkIsolated NetworkMode = "isolated"
)

type CapabilityProfile string

const (
	ProfileNone        CapabilityProfile = "none"
	ProfilePermissions CapabilityProfile = "permissions"
)

type Resources struct {
	MemoryMB       int
	PIDs           int
	TimeoutSeconds int
}

type Spec struct {
	ID                string
	ImageRef          string
	Network           NetworkMode
	CapabilityProfile CapabilityProfile
	Resources         Resources
}

var digestImagePattern = regexp.MustCompile(`^[^[:space:]]+@sha256:[a-f0-9]{64}$`)

func (s Spec) Validate() error {
	if strings.TrimSpace(s.ID) == "" {
		return errors.New("lab ID is required")
	}
	if !digestImagePattern.MatchString(s.ImageRef) {
		return errors.New("lab image must be pinned by sha256 digest")
	}
	switch s.Network {
	case NetworkNone, NetworkIsolated:
	default:
		return fmt.Errorf("unsupported network mode %q", s.Network)
	}
	switch s.CapabilityProfile {
	case ProfileNone, ProfilePermissions:
	default:
		return fmt.Errorf("unknown capability profile %q", s.CapabilityProfile)
	}
	if s.Resources.MemoryMB < 64 || s.Resources.MemoryMB > 4096 {
		return errors.New("memory limit must be between 64 and 4096 MiB")
	}
	if s.Resources.PIDs < 16 || s.Resources.PIDs > 1024 {
		return errors.New("PID limit must be between 16 and 1024")
	}
	if s.Resources.TimeoutSeconds < 30 || s.Resources.TimeoutSeconds > 7200 {
		return errors.New("timeout must be between 30 and 7200 seconds")
	}
	return nil
}

func (s Spec) Timeout() time.Duration {
	return time.Duration(s.Resources.TimeoutSeconds) * time.Second
}
