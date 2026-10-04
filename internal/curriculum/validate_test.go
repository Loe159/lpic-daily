package curriculum_test

import (
	"testing"

	lpicdaily "github.com/Loe159/lpic-daily"
	"github.com/Loe159/lpic-daily/internal/curriculum"
)

func TestBuiltinBundleValidates(t *testing.T) {
	bundle, err := curriculum.Load(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	summary := curriculum.Summarize(bundle)
	if summary.ActiveObjectives != 42 {
		t.Fatalf("active objectives = %d, want 42", summary.ActiveObjectives)
	}
	if summary.Exam101Weight != 60 || summary.Exam102Weight != 60 {
		t.Fatalf("weights = (%d, %d), want (60, 60)", summary.Exam101Weight, summary.Exam102Weight)
	}
	if summary.Concepts != 295 {
		t.Fatalf("concepts = %d, want 295", summary.Concepts)
	}
	if summary.Phase1Concepts != 22 {
		t.Fatalf("phase1 concepts = %d, want 22", summary.Phase1Concepts)
	}
	if summary.Phase3Objectives != 23 || summary.Phase3Concepts != 153 {
		t.Fatalf("phase3 scope = (%d objectives, %d concepts), want (23, 153)", summary.Phase3Objectives, summary.Phase3Concepts)
	}
	if summary.SchemaFiles != 10 {
		t.Fatalf("schemas = %d, want 10", summary.SchemaFiles)
	}
}
