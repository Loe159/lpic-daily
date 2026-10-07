package runner

import (
	"fmt"
	"slices"
)

type CapabilityProfile struct {
	Name         string
	Capabilities []string
}

var phase1CapabilityProfiles = map[string]CapabilityProfile{
	"baseline": {
		Name:         "baseline",
		Capabilities: []string{},
	},
	"identity-files": {
		Name:         "identity-files",
		Capabilities: []string{"CHOWN", "FOWNER", "FSETID"},
	},
	"metadata-db": {
		Name:         "metadata-db",
		Capabilities: []string{"CHOWN"},
	},
	"process-lab": {
		Name:         "process-lab",
		Capabilities: []string{"KILL"},
	},
}

func Phase1CapabilityProfile(name string) (CapabilityProfile, error) {
	profile, exists := phase1CapabilityProfiles[name]
	if !exists {
		return CapabilityProfile{}, fmt.Errorf("unknown capability profile %q", name)
	}
	profile.Capabilities = slices.Clone(profile.Capabilities)
	return profile, nil
}

func Phase1CapabilityProfileNames() []string {
	names := make([]string, 0, len(phase1CapabilityProfiles))
	for name := range phase1CapabilityProfiles {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}
