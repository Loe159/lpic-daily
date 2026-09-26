package store

import (
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type Migration struct {
	Version int
	Name    string
	SQL     string
}

func Migrations() ([]Migration, error) {
	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return nil, err
	}
	migrations := make([]Migration, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		parts := strings.SplitN(entry.Name(), "_", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid migration filename %q", entry.Name())
		}
		version, err := strconv.Atoi(parts[0])
		if err != nil || version <= 0 {
			return nil, fmt.Errorf("invalid migration version in %q", entry.Name())
		}
		data, err := migrationFiles.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return nil, err
		}
		migrations = append(migrations, Migration{Version: version, Name: entry.Name(), SQL: string(data)})
	}
	sort.Slice(migrations, func(i, j int) bool { return migrations[i].Version < migrations[j].Version })
	for i, migration := range migrations {
		expected := i + 1
		if migration.Version != expected {
			return nil, fmt.Errorf("migration sequence gap: expected %d, got %d", expected, migration.Version)
		}
		if strings.TrimSpace(migration.SQL) == "" {
			return nil, fmt.Errorf("migration %s is empty", migration.Name)
		}
	}
	return migrations, nil
}
