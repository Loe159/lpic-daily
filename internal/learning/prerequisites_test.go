package learning_test

import (
	"slices"
	"testing"

	lpicdaily "github.com/Loe159/lpic-daily"
	"github.com/Loe159/lpic-daily/internal/curriculum"
	"github.com/Loe159/lpic-daily/internal/learning"
)

func loadGraph(t *testing.T) curriculum.PrerequisitesFile {
	t.Helper()
	bundle, err := curriculum.Load(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	return bundle.Prerequisites
}

func TestInitialEligibility(t *testing.T) {
	eligible := learning.EligibleObjectives(loadGraph(t), map[string]bool{})
	want := []string{"101.1", "101.2", "103.1", "106.3", "109.1"}
	if !slices.Equal(eligible, want) {
		t.Fatalf("eligible = %v, want %v", eligible, want)
	}
}

func TestShellFoundationUnlocksPhase1Peers(t *testing.T) {
	eligible := learning.EligibleObjectives(loadGraph(t), map[string]bool{"103.1": true})
	for _, objectiveID := range []string{"103.5", "104.5"} {
		if !slices.Contains(eligible, objectiveID) {
			t.Fatalf("%s should be eligible after 103.1 evidence; got %v", objectiveID, eligible)
		}
	}
}
