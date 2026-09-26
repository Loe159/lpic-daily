package curriculum

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRepositoryCatalogLoads(t *testing.T) {
	dir := filepath.Join("..", "..", "curriculum", "lpic-1-v5")
	if _, err := os.Stat(filepath.Join(dir, "objectives.json")); err != nil {
		t.Skip("repository curriculum is not present in this isolated unit-test workspace")
	}
	catalog, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(catalog.Objectives); got != 42 {
		t.Fatalf("objectives=%d, want 42", got)
	}
	if got := len(catalog.Concepts); got != 295 {
		t.Fatalf("concepts=%d, want 295", got)
	}
	if got := len(catalog.Phase1Slice.SelectedObjectives); got != 3 {
		t.Fatalf("phase1 objectives=%d, want 3", got)
	}
}
