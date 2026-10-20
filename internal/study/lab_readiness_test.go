package study

import (
	"testing"

	lpicdaily "github.com/Loe159/lpic-daily"
	"github.com/Loe159/lpic-daily/internal/lab"
)

// Exercise the very same gate used by BuildPlan against the complete
// embedded lab catalog, not just the FHS scenario that exposed the bug.
func TestEveryLabRequiresEveryMappedConceptToBeStudied(t *testing.T) {
	authored, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("load embedded labs: %v", err)
	}
	if len(authored) == 0 {
		t.Fatal("embedded lab catalog unexpectedly empty")
	}
	compositeCount := 0
	for _, item := range authored {
		ids := item.Definition.ConceptIDs
		if len(ids) == 0 {
			t.Fatalf("lab %s has no concepts", item.Definition.ID)
		}
		if len(ids) > 1 {
			compositeCount++
		}
		studied := make(map[string]bool, len(ids))
		if labPrerequisitesMet(ids, studied) {
			t.Fatalf("lab %s available before any studied concepts", item.Definition.ID)
		}
		for _, id := range ids {
			studied[id] = true
		}
		if !labPrerequisitesMet(ids, studied) {
			t.Fatalf("lab %s unavailable after all its concepts were studied", item.Definition.ID)
		}
		for _, missing := range ids {
			delete(studied, missing)
			if labPrerequisitesMet(ids, studied) {
				t.Fatalf("lab %s available without prerequisite concept %s", item.Definition.ID, missing)
			}
			studied[missing] = true
		}
	}
	if compositeCount == 0 {
		t.Fatal("catalog regression: expected cross-concept scenarios")
	}
}
