package lab

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Loe159/lpic-daily/internal/checker"
	"github.com/Loe159/lpic-daily/internal/runner"
)

const labGlob = "labs/lpic-1-v5/*/*/lab.json"

func LoadAll(fsys fs.FS) ([]Lab, error) {
	paths, err := fs.Glob(fsys, labGlob)
	if err != nil {
		return nil, fmt.Errorf("glob built-in labs: %w", err)
	}
	if len(paths) == 0 {
		return nil, errors.New("no built-in labs found")
	}
	slices.Sort(paths)

	labs := make([]Lab, 0, len(paths))
	seen := make(map[string]struct{}, len(paths))
	for _, labPath := range paths {
		loaded, err := loadOne(fsys, labPath)
		if err != nil {
			return nil, err
		}
		if _, exists := seen[loaded.Definition.ID]; exists {
			return nil, fmt.Errorf("duplicate lab ID %s", loaded.Definition.ID)
		}
		seen[loaded.Definition.ID] = struct{}{}
		labs = append(labs, loaded)
	}
	return labs, nil
}

func loadOne(fsys fs.FS, labPath string) (Lab, error) {
	var definition Definition
	if err := decodeStrictFile(fsys, labPath, &definition); err != nil {
		return Lab{}, err
	}
	base := path.Dir(labPath)
	if err := validateDefinition(fsys, base, definition); err != nil {
		return Lab{}, fmt.Errorf("%s: %w", labPath, err)
	}

	var setupScript string
	if definition.Setup.ExecutionScope == "sandbox" {
		setupPath, err := resolveLocalRef(base, definition.Setup.ScriptRef)
		if err != nil {
			return Lab{}, fmt.Errorf("%s setup: %w", definition.ID, err)
		}
		setupBytes, err := fs.ReadFile(fsys, setupPath)
		if err != nil {
			return Lab{}, fmt.Errorf("read setup for %s: %w", definition.ID, err)
		}
		setupScript = string(setupBytes)
	}

	hintPaths, err := fs.Glob(fsys, path.Join(base, "hints", "*.json"))
	if err != nil {
		return Lab{}, fmt.Errorf("glob hints for %s: %w", definition.ID, err)
	}
	slices.Sort(hintPaths)

	hintByID := make(map[string]Hint, len(hintPaths))
	for _, hintPath := range hintPaths {
		var hint Hint
		if err := decodeStrictFile(fsys, hintPath, &hint); err != nil {
			return Lab{}, err
		}
		if err := validateHint(definition.ID, hint); err != nil {
			return Lab{}, fmt.Errorf("%s: %w", hintPath, err)
		}
		if _, exists := hintByID[hint.ID]; exists {
			return Lab{}, fmt.Errorf("%s: duplicate hint ID %s", definition.ID, hint.ID)
		}
		hintByID[hint.ID] = hint
	}

	hints := make([]Hint, 0, len(definition.HintIDs))
	for _, hintID := range definition.HintIDs {
		hint, exists := hintByID[hintID]
		if !exists {
			return Lab{}, fmt.Errorf("%s: missing referenced hint %s", definition.ID, hintID)
		}
		hints = append(hints, hint)
		delete(hintByID, hintID)
	}
	if len(hintByID) != 0 {
		return Lab{}, fmt.Errorf("%s: unreferenced hint files remain", definition.ID)
	}
	slices.SortFunc(hints, func(a, b Hint) int {
		return a.Level - b.Level
	})

	return Lab{
		Definition:           definition,
		Hints:                hints,
		SetupScript:          setupScript,
		ReferenceSolutionRef: definition.ReferenceSolutionRef,
	}, nil
}

func (lab Lab) RunnerDefinition() (runner.Definition, error) {
	definition := runner.Definition{
		LabID:             lab.Definition.ID,
		ImageRef:          lab.Definition.Environment.ImageRef,
		Distribution:      lab.Definition.Environment.Distribution,
		Network:           runner.NetworkMode(lab.Definition.Environment.Network),
		CapabilityProfile: lab.Definition.Environment.CapabilityProfile,
		MemoryMB:          lab.Definition.Resources.MemoryMB,
		CPUPercent:        lab.Definition.Resources.CPUPercent,
		PIDs:              lab.Definition.Resources.PIDs,
		Timeout:           time.Duration(lab.Definition.Resources.TimeoutSeconds) * time.Second,
	}
	if machine := lab.Definition.Environment.Machine; machine != nil {
		definition.Machine = &runner.MachineDefinition{
			Firmware: runner.FirmwareMode(machine.Firmware),
		}
		for _, disk := range machine.ExtraDisks {
			definition.Machine.ExtraDisks = append(definition.Machine.ExtraDisks, runner.VirtualDisk{
				ID:     disk.ID,
				SizeMB: disk.SizeMB,
			})
		}
	}
	if err := definition.Validate(); err != nil {
		return runner.Definition{}, err
	}
	return definition, nil
}

func (lab Lab) CompileChecks() ([]checker.Check, error) {
	checks := make([]checker.Check, 0, len(lab.Definition.Checks))
	for index, item := range lab.Definition.Checks {
		id := fmt.Sprintf("%s.check-%02d", lab.Definition.ID, index+1)
		switch item.Type {
		case "file-exists":
			checks = append(checks, checker.FileExists{CheckID: id, Path: item.Path})
		case "file-mode":
			mode, err := strconv.ParseUint(item.Mode, 8, 32)
			if err != nil {
				return nil, fmt.Errorf("%s: invalid octal mode %q: %w", id, item.Mode, err)
			}
			checks = append(checks, checker.FileMode{CheckID: id, Path: item.Path, Mode: uint32(mode)})
		case "file-owner":
			checks = append(checks, checker.FileOwner{
				CheckID: id,
				Path:    item.Path,
				User:    item.User,
				Group:   item.Group,
			})
		case "file-content-regex":
			checks = append(checks, checker.FileContentRegex{
				CheckID: id,
				Path:    item.Path,
				Pattern: item.Pattern,
			})
		case "process-running":
			checks = append(checks, checker.ProcessState{CheckID: id, Match: item.Match, Present: true})
		case "process-absent":
			checks = append(checks, checker.ProcessState{CheckID: id, Match: item.Match, Present: false})
		case "command-exit":
			checks = append(checks, checker.CommandExit{
				CheckID:      id,
				Argv:         append([]string(nil), item.Argv...),
				ExpectedExit: item.ExpectedExit,
			})
		default:
			return nil, fmt.Errorf("%s: unsupported check type %q", id, item.Type)
		}
	}
	return checks, nil
}

func validateDefinition(fsys fs.FS, base string, definition Definition) error {
	if definition.SchemaVersion != "1.0.0" {
		return fmt.Errorf("unsupported schema version %q", definition.SchemaVersion)
	}
	if definition.ID == "" || strings.TrimSpace(definition.TitleFR) == "" {
		return errors.New("lab id and title are required")
	}
	if len(strings.TrimSpace(definition.BriefFR)) < 20 {
		return errors.New("brief_fr must contain a useful learner-facing prompt")
	}
	if len(strings.TrimSpace(definition.DebriefFR)) < 20 {
		return errors.New("debrief_fr must contain a useful explanation")
	}
	if len(definition.SuccessCriteriaFR) == 0 {
		return errors.New("success_criteria_fr must not be empty")
	}
	for _, criterion := range definition.SuccessCriteriaFR {
		if len(strings.TrimSpace(criterion)) < 5 {
			return errors.New("success criteria must contain useful text")
		}
	}
	if len(definition.ObjectiveIDs) == 0 || len(definition.ConceptIDs) == 0 {
		return errors.New("objective_ids and concept_ids are required")
	}
	if definition.Environment.Backend != "podman" && definition.Environment.Backend != "libvirt" {
		return fmt.Errorf("unsupported backend %q", definition.Environment.Backend)
	}
	if definition.Environment.Network != "none" && definition.Environment.Network != "isolated" {
		return fmt.Errorf("unsupported network %q", definition.Environment.Network)
	}
	switch definition.Environment.Backend {
	case "podman":
		if definition.Environment.Machine != nil {
			return errors.New("podman lab must not declare machine settings")
		}
		if definition.Setup.ExecutionScope != "sandbox" {
			return errors.New("podman lab setup execution scope must be sandbox")
		}
		if _, err := resolveLocalRef(base, definition.Setup.ScriptRef); err != nil {
			return fmt.Errorf("invalid setup reference: %w", err)
		}
	case "libvirt":
		if definition.Environment.Machine == nil {
			return errors.New("libvirt lab requires machine settings")
		}
		if definition.Setup.ExecutionScope != "none" {
			return errors.New("libvirt Phase-2 lab setup execution scope must be none")
		}
		if definition.Setup.ScriptRef != "" {
			return errors.New("libvirt setup=none must not declare script_ref")
		}
	}
	if definition.ResetPolicy != "disposable" {
		return errors.New("reset policy must be disposable")
	}
	if definition.EstimatedMinutes < 1 {
		return errors.New("estimated_minutes must be positive")
	}
	if len(definition.Checks) == 0 {
		return errors.New("at least one check is required")
	}
	if definition.ReferenceSolutionRef != "" {
		solutionPath, err := resolveLocalRef(base, definition.ReferenceSolutionRef)
		if err != nil {
			return fmt.Errorf("invalid reference solution: %w", err)
		}
		if _, err := fs.Stat(fsys, solutionPath); err != nil {
			return fmt.Errorf("stat reference solution: %w", err)
		}
	}
	return nil
}

func validateHint(labID string, hint Hint) error {
	if hint.SchemaVersion != "1.0.0" {
		return fmt.Errorf("unsupported hint schema version %q", hint.SchemaVersion)
	}
	if hint.ID == "" || hint.LabID != labID {
		return fmt.Errorf("hint %q has mismatched lab ID %q", hint.ID, hint.LabID)
	}
	if hint.Level < 1 || hint.Level > 4 {
		return fmt.Errorf("hint %s level %d outside 1..4", hint.ID, hint.Level)
	}
	if hint.Level == 4 && hint.EvidenceImpact != "solution-revealed" {
		return fmt.Errorf("hint %s level 4 must reveal solution", hint.ID)
	}
	if strings.TrimSpace(hint.ContentFR) == "" {
		return fmt.Errorf("hint %s has empty content", hint.ID)
	}
	return nil
}

func resolveLocalRef(base, ref string) (string, error) {
	if ref == "" {
		return "", errors.New("empty local reference")
	}
	if path.IsAbs(ref) {
		return "", errors.New("absolute local reference is forbidden")
	}
	clean := path.Clean(ref)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("reference escapes lab directory: %q", ref)
	}
	resolved := path.Join(base, clean)
	if resolved != base && !strings.HasPrefix(resolved, base+"/") {
		return "", fmt.Errorf("reference escapes lab directory: %q", ref)
	}
	return resolved, nil
}

func decodeStrictFile(fsys fs.FS, name string, out any) error {
	file, err := fsys.Open(name)
	if err != nil {
		return fmt.Errorf("open %s: %w", name, err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return fmt.Errorf("decode %s: %w", name, err)
	}

	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("decode %s: trailing JSON value", name)
		}
		return fmt.Errorf("decode %s trailing data: %w", name, err)
	}
	return nil
}
