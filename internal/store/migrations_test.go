package store

import (
	"strings"
	"testing"
)

func TestMigrationsAreSequentialAndProtectAppendOnlyLogs(t *testing.T) {
	migrations, err := Migrations()
	if err != nil {
		t.Fatal(err)
	}
	if len(migrations) != 1 {
		t.Fatalf("migration count=%d, want 1", len(migrations))
	}
	sql := migrations[0].SQL
	for _, required := range []string{
		"CREATE TABLE IF NOT EXISTS mastery_evidence",
		"CREATE TABLE IF NOT EXISTS gamification_events",
		"mastery_evidence_no_update",
		"mastery_evidence_no_delete",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("migration missing %q", required)
		}
	}
}
