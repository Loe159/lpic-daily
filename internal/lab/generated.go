package lab

import (
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"github.com/Loe159/lpic-daily/internal/content"
	"github.com/Loe159/lpic-daily/internal/curriculum"
)

func appendStandaloneLabs(fsys fs.FS, labs []Lab, seen map[string]struct{}) ([]Lab, error) {
	curriculumBundle, err := curriculum.Load(fsys)
	if err != nil {
		return nil, fmt.Errorf("load curriculum: %w", err)
	}
	guideByObjective := make(map[string]curriculum.ObjectiveStudyGuide)
	for _, guide := range curriculumBundle.StudyGuides.Guides {
		guideByObjective[guide.ObjectiveID] = guide
	}

	for _, objective := range curriculumBundle.Objectives.Objectives {
		if !objective.Active {
			continue
		}
		guide, exists := guideByObjective[objective.ID]
		if !exists {
			return nil, fmt.Errorf("objective %s has no study guide", objective.ID)
		}
		concepts := standaloneConceptsForObjective(curriculumBundle.Concepts.Concepts, objective.ID)
		for _, concept := range concepts {
			for variant := 0; variant < 2; variant++ {
				generated := generatedStandaloneLab(objective, concept, guide, variant)
				if err := validateStandaloneLab(generated); err != nil {
					return nil, fmt.Errorf("%s semantic validation: %w", generated.Definition.ID, err)
				}
				if _, duplicate := seen[generated.Definition.ID]; duplicate {
					continue
				}
				if _, err := generated.RunnerDefinition(); err != nil {
					return nil, fmt.Errorf("%s runner definition: %w", generated.Definition.ID, err)
				}
				if _, err := generated.CompileChecks(); err != nil {
					return nil, fmt.Errorf("%s checks: %w", generated.Definition.ID, err)
				}
				seen[generated.Definition.ID] = struct{}{}
				labs = append(labs, generated)
			}
		}
	}
	slices.SortFunc(labs, func(a, b Lab) int {
		return strings.Compare(a.Definition.ID, b.Definition.ID)
	})
	return labs, nil
}

func validateStandaloneLab(authored Lab) error {
	definition := authored.Definition
	if definition.SchemaVersion != "1.0.0" || definition.ID == "" || strings.TrimSpace(definition.TitleFR) == "" {
		return fmt.Errorf("schema version, id and title are required")
	}
	if strings.TrimSpace(definition.PracticeContext) == "" ||
		len(strings.TrimSpace(definition.BriefFR)) < 20 ||
		len(strings.TrimSpace(definition.DebriefFR)) < 20 {
		return fmt.Errorf("practice context, learner brief and debrief are required")
	}
	if len(definition.ObjectiveIDs) != 1 || len(definition.ConceptIDs) == 0 ||
		len(definition.Checks) == 0 || definition.ResetPolicy != "disposable" {
		return fmt.Errorf("objective, concepts, checks and disposable reset policy are required")
	}
	checked := make(map[string]bool, len(definition.ConceptIDs))
	declared := make(map[string]bool, len(definition.ConceptIDs))
	for _, conceptID := range definition.ConceptIDs {
		declared[conceptID] = true
	}
	for index, check := range definition.Checks {
		if len(check.ConceptIDs) == 0 {
			return fmt.Errorf("check %d has no concept mapping", index+1)
		}
		for _, conceptID := range check.ConceptIDs {
			if !declared[conceptID] {
				return fmt.Errorf("check %d maps undeclared concept %s", index+1, conceptID)
			}
			checked[conceptID] = true
		}
	}
	for _, conceptID := range definition.ConceptIDs {
		if !checked[conceptID] {
			return fmt.Errorf("concept %s has no check", conceptID)
		}
	}
	if len(authored.Hints) != 4 || len(definition.HintIDs) != 4 {
		return fmt.Errorf("exactly four graduated hints are required")
	}
	for index, hint := range authored.Hints {
		if hint.ID != definition.HintIDs[index] || hint.Level != index+1 {
			return fmt.Errorf("hint ladder order mismatch at level %d", index+1)
		}
		if err := validateHint(definition.ID, hint); err != nil {
			return err
		}
	}
	switch definition.Environment.Backend {
	case "podman":
		if definition.Environment.Network != "none" || definition.Setup.ExecutionScope != "sandbox" ||
			strings.TrimSpace(authored.SetupScript) == "" {
			return fmt.Errorf("Podman standalone lab requires network=none and an in-memory sandbox setup")
		}
	case "libvirt":
		if definition.Environment.Machine == nil || definition.Setup.ExecutionScope != "none" ||
			len(definition.Environment.WritableGuestPaths) != 0 {
			return fmt.Errorf("libvirt standalone lab requires machine settings, setup=none and no host writable paths")
		}
	default:
		return fmt.Errorf("unsupported backend %q", definition.Environment.Backend)
	}
	return nil
}

func standaloneConceptsForObjective(all []curriculum.Concept, objectiveID string) []curriculum.Concept {
	var concepts []curriculum.Concept
	for _, concept := range all {
		if concept.Active && concept.ObjectiveID == objectiveID {
			concepts = append(concepts, concept)
		}
	}
	slices.SortFunc(concepts, func(a, b curriculum.Concept) int {
		return a.PedagogyOrder - b.PedagogyOrder
	})
	return concepts
}

func generatedStandaloneLab(
	objective curriculum.Objective,
	concept curriculum.Concept,
	_ curriculum.ObjectiveStudyGuide,
	variant int,
) Lab {
	contextName := "diagnostic"
	contextFR := "diagnostic"
	if variant == 1 {
		contextName = "transfer"
		contextFR = "transfert"
	}
	labID := concept.ID + ".standalone-" + contextName
	backend, imageRef, distribution := standaloneLabEnvironment(objective, variant)
	root := "/workspace/lpic-daily-evidence"
	setup := Setup{ExecutionScope: "sandbox", ScriptRef: "generated-setup.sh"}
	setupScript := "set -eu\ninstall -d -m 0777 /workspace/lpic-daily-evidence\n: > /workspace/lpic-daily-evidence/command-history.log\nchmod 0666 /workspace/lpic-daily-evidence/command-history.log\n"
	network := "none"
	if strings.HasPrefix(objective.ID, "109.") || objective.ID == "110.3" {
		network = "isolated"
	}
	environment := Environment{
		Backend:            "podman",
		ImageRef:           imageRef,
		Distribution:       distribution,
		Network:            network,
		CapabilityProfile:  "baseline",
		WritableGuestPaths: []string{"/workspace"},
	}
	resources := Resources{
		MemoryMB:       256,
		CPUPercent:     100,
		PIDs:           128,
		TimeoutSeconds: 3600,
	}
	if backend == "libvirt" {
		root = "/root/lpic-daily-evidence"
		setup = Setup{ExecutionScope: "none"}
		setupScript = ""
		environment = Environment{
			Backend:           "libvirt",
			ImageRef:          imageRef,
			Distribution:      distribution,
			Network:           network,
			CapabilityProfile: "full-machine",
			Machine:           standaloneMachineForObjective(objective),
		}
		resources = Resources{
			MemoryMB:       1024,
			CPUPercent:     100,
			PIDs:           256,
			TimeoutSeconds: 5400,
		}
	}

	commandHistoryPath := root + "/command-history.log"
	filename := standaloneConceptFilename(concept)
	terms := slices.Clone(concept.AnchorTerms)
	termsLine := strings.Join(terms, ",")
	checks := []CheckDefinition{{
		Type: "file-content-regex",
		Path: root + "/" + filename + ".txt",
		Pattern: "(?m)^CONCEPT=" + regexEscape(concept.ID) +
			"\\nTERMS=" + regexEscape(termsLine) +
			"\\nCOMMAND=.+\\nOBSERVATION=.+\\nEXPLANATION=.+\\n?$",
		ConceptIDs: []string{concept.ID},
	}}
	if objective.Exam == "101" {
		if pattern := standaloneCommandEvidencePattern(concept); pattern != "" {
			checks = append(checks, CheckDefinition{
				Type:       "file-content-regex",
				Path:       commandHistoryPath,
				Pattern:    pattern,
				ConceptIDs: []string{concept.ID},
			})
		}
	}

	brief := fmt.Sprintf(
		"## Objectif\n%s\n\nRepères : %s\n\n## À faire\n%s\n\nQuand tu as terminé, tape `:check`.",
		concept.TitleFR,
		standaloneAnchorList(concept),
		standaloneLearnerTask(concept, variant),
	)

	hints := generatedStandaloneHints(labID, concept)
	return Lab{
		Definition: Definition{
			SchemaVersion: "1.0.0",
			ID:            labID,
			TitleFR:       fmt.Sprintf("%s — %s — pratique %s", objective.ID, concept.TitleFR, contextFR),
			BriefFR:       brief,
			SuccessCriteriaFR: []string{
				"La commande ou l'observation demandée a été réalisée.",
				fmt.Sprintf("Tu peux relier %s à « %s ».", standaloneAnchorList(concept), concept.TitleFR),
			},
			DebriefFR: standaloneDebrief(concept),
			ObjectiveIDs:     []string{objective.ID},
			ConceptIDs:       []string{concept.ID},
			Labels:           []string{"lpic-required"},
			EstimatedMinutes: 15,
			PracticeContext: fmt.Sprintf(
				"%s-c%02d-standalone-%s",
				strings.ReplaceAll(objective.ID, ".", "-"),
				concept.PedagogyOrder,
				contextName,
			),
			Environment: environment,
			Resources:   resources,
			Setup:       setup,
			Checks:      checks,
			HintIDs:     []string{hints[0].ID, hints[1].ID, hints[2].ID, hints[3].ID},
			ResetPolicy: "disposable",
		},
		Hints:       hints,
		SetupScript: setupScript,
	}
}

func standaloneLabEnvironment(objective curriculum.Objective, variant int) (backend, imageRef, distribution string) {
	if objective.ID == "102.4" {
		return "libvirt", "debian-13-x86_64-v2", "debian"
	}
	if objective.ID == "102.5" && variant == 1 {
		return "libvirt", "opensuse-leap-16.0-x86_64-v2", "opensuse"
	}
	if strings.Contains(objective.RecommendedBackend, "podman") &&
		strings.Contains(objective.RecommendedBackend, "libvirt") {
		if variant == 0 {
			return "podman", "localhost/lpic-daily/fedora-phase1:1", "fedora"
		}
		return "libvirt", "fedora-44-x86_64-v2", "fedora"
	}
	if strings.HasPrefix(objective.ID, "101.") ||
		strings.HasPrefix(objective.ID, "109.") ||
		objective.ID == "110.3" ||
		strings.Contains(objective.RecommendedBackend, "libvirt") {
		return "libvirt", "fedora-44-x86_64-v2", "fedora"
	}
	return "podman", "localhost/lpic-daily/fedora-phase1:1", "fedora"
}

func standaloneMachineForObjective(objective curriculum.Objective) *Machine {
	machine := &Machine{Firmware: "uefi"}
	switch objective.ID {
	case "102.1", "104.1":
		machine.ExtraDisks = []MachineDisk{
			{ID: "practice-a", SizeMB: 1024},
			{ID: "practice-b", SizeMB: 1024},
		}
	case "104.2", "104.3":
		machine.ExtraDisks = []MachineDisk{
			{ID: "practice", SizeMB: 1024},
		}
	}
	return machine
}

type standalonePracticeExample struct {
	Anchor      string
	Command     string
	Explanation string
}

func standaloneAnchorList(concept curriculum.Concept) string {
	return "`" + strings.Join(concept.AnchorTerms, "`, `") + "`"
}

func standalonePracticeExamples(concept curriculum.Concept) []standalonePracticeExample {
	examples := make([]standalonePracticeExample, 0, 4)
	seen := make(map[string]struct{})
	for _, anchor := range concept.AnchorTerms {
		explanation := content.PedagogicalTermExplanation(anchor, concept.ObjectiveID)
		usage := strings.TrimSpace(content.PedagogicalTermUsage(anchor))
		if usage != "" && standaloneCommandAnchorAllowed(anchor) {
			for _, command := range strings.Split(usage, " ; ") {
				command = strings.TrimSpace(command)
				if command == "" {
					continue
				}
				if _, duplicate := seen[command]; duplicate {
					continue
				}
				seen[command] = struct{}{}
				examples = append(examples, standalonePracticeExample{
					Anchor: anchor, Command: command, Explanation: explanation,
				})
				if len(examples) == 4 {
					return examples
				}
			}
		}
		if fallback := standaloneInspectionCommand(anchor); fallback != "" {
			if _, duplicate := seen[fallback]; !duplicate {
				seen[fallback] = struct{}{}
				examples = append(examples, standalonePracticeExample{
					Anchor: anchor, Command: fallback, Explanation: explanation,
				})
				if len(examples) == 4 {
					return examples
				}
			}
		}
	}
	return examples
}

func standaloneInspectionCommand(anchor string) string {
	switch {
	case strings.HasPrefix(anchor, "/"), strings.HasPrefix(anchor, "~/"):
		return "ls -ld " + anchor
	case isDecimalAnchor(anchor):
		return "grep -E '[[:space:]]" + anchor + "/(tcp|udp)' /etc/services"
	default:
		return ""
	}
}

func isDecimalAnchor(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func standaloneLearnerTask(concept curriculum.Concept, variant int) string {
	if concept.ID == "lpic1.103.1.set-env-et-portee-des-variables" {
		if variant == 0 {
			return "Crée `LPIC_SCOPE=local` sans l'exporter. Vérifie sa présence avec `set` et son absence avec `env`, puis exporte-la et compare à nouveau."
		}
		return "Crée `LPIC_CHILD=visible`, exporte-la et lance `env | grep '^LPIC_CHILD='`. Fais ensuite `unset LPIC_CHILD` et vérifie qu'un nouveau processus ne la reçoit plus."
	}

	examples := standalonePracticeExamples(concept)
	if len(examples) != 0 {
		example := examples[variant%len(examples)]
		if variant == 0 {
			return fmt.Sprintf(
				"Exécute `%s`. Observe ce que le résultat montre sur `%s` : %s",
				example.Command,
				example.Anchor,
				example.Explanation,
			)
		}
		return fmt.Sprintf(
			"Utilise `%s` sur un autre exemple ou une autre cible. Vérifie que le résultat confirme le rôle de `%s` : %s",
			example.Command,
			example.Anchor,
			example.Explanation,
		)
	}

	if len(concept.AnchorTerms) >= 2 {
		first := concept.AnchorTerms[0]
		second := concept.AnchorTerms[1]
		return fmt.Sprintf(
			"Compare `%s` et `%s` dans le système du lab. Identifie un exemple concret de chacun et explique la différence : `%s` — %s ; `%s` — %s",
			first,
			second,
			first,
			content.PedagogicalTermExplanation(first, concept.ObjectiveID),
			second,
			content.PedagogicalTermExplanation(second, concept.ObjectiveID),
		)
	}

	anchor := concept.AnchorTerms[0]
	return fmt.Sprintf(
		"Trouve dans le système du lab un exemple de `%s` correspondant à cette définition : %s Note où tu l'as observé et ce qui le prouve.",
		anchor,
		content.PedagogicalTermExplanation(anchor, concept.ObjectiveID),
	)
}

func standaloneDebrief(concept curriculum.Concept) string {
	limit := 2
	if len(concept.AnchorTerms) < limit {
		limit = len(concept.AnchorTerms)
	}
	parts := make([]string, 0, limit)
	for _, anchor := range concept.AnchorTerms[:limit] {
		parts = append(parts, fmt.Sprintf(
			"`%s` : %s",
			anchor,
			content.PedagogicalTermExplanation(anchor, concept.ObjectiveID),
		))
	}
	return "À retenir — " + strings.Join(parts, " ")
}

func generatedStandaloneHints(labID string, concept curriculum.Concept) []Hint {
	examples := standalonePracticeExamples(concept)
	firstAnchor := concept.AnchorTerms[0]
	firstExplanation := content.PedagogicalTermExplanation(firstAnchor, concept.ObjectiveID)

	level1 := "Commence par repérer `" + firstAnchor + "` : " + firstExplanation
	level2 := "Vérifie que ton exemple correspond bien à `" + firstAnchor + "` : " + firstExplanation
	level4 := "Reprends exactement les repères " + standaloneAnchorList(concept) + " et vérifie-les un par un."
	if len(examples) != 0 {
		level1 = "Commence par `" + examples[0].Command + "`."
		level2 = "Dans le résultat, vérifie `" + examples[0].Anchor + "` : " + examples[0].Explanation
		if len(examples) > 1 {
			level4 = "Essaie ensuite `" + examples[1].Command + "` et compare les deux observations."
		}
	}

	return []Hint{
		{
			SchemaVersion:  "1.0.0",
			ID:             labID + ".hint-1",
			LabID:          labID,
			Level:          1,
			ContentFR:      level1,
			EvidenceImpact: "none",
		},
		{
			SchemaVersion:  "1.0.0",
			ID:             labID + ".hint-2",
			LabID:          labID,
			Level:          2,
			ContentFR:      level2,
			EvidenceImpact: "material",
		},
		{
			SchemaVersion:  "1.0.0",
			ID:             labID + ".hint-3",
			LabID:          labID,
			Level:          3,
			ContentFR:      "Avant `:check`, résume en une phrase ce que tu as observé et ce que cela démontre.",
			EvidenceImpact: "material",
		},
		{
			SchemaVersion:  "1.0.0",
			ID:             labID + ".hint-4",
			LabID:          labID,
			Level:          4,
			ContentFR:      level4,
			EvidenceImpact: "solution-revealed",
		},
	}
}

func StandaloneCommandHistoryPath(definition Definition) (string, bool) {
	if !strings.Contains(definition.ID, ".standalone-") {
		return "", false
	}
	switch definition.Environment.Backend {
	case "podman":
		return "/workspace/lpic-daily-evidence/command-history.log", true
	case "libvirt":
		return "/root/lpic-daily-evidence/command-history.log", true
	default:
		return "", false
	}
}

func StandaloneReflectionPath(definition Definition) (string, bool) {
	if !strings.Contains(definition.ID, ".standalone-") {
		return "", false
	}
	historyPath, _ := StandaloneCommandHistoryPath(definition)
	for _, check := range definition.Checks {
		if check.Type == "file-content-regex" && check.Path != "" && check.Path != historyPath {
			return check.Path, true
		}
	}
	return "", false
}

func standaloneCommandEvidencePattern(concept curriculum.Concept) string {
	// This concept is intentionally exercised inside one persistent shell:
	// a non-exported variable must survive while set/env are compared. The
	// interactive reflection is therefore the evidence source; executing env or
	// set later through the one-command runner would test a different shell.
	if concept.ID == "lpic1.103.1.set-env-et-portee-des-variables" {
		return ""
	}

	var commands []string
	for _, anchor := range concept.AnchorTerms {
		if !standaloneCommandAnchorAllowed(anchor) || !standaloneDirectCommandAnchor(anchor) {
			continue
		}
		commands = append(
			commands,
			"(?:[^;&|]*[;&|][[:space:]]*)*[[:space:]]*(?:sudo[[:space:]]+)?"+
				regexEscape(anchor)+"(?:[[:space:]]|$)",
		)
	}
	if len(commands) == 0 {
		return ""
	}
	return "(?m)^EXIT=0\\tCOMMAND=(?:" + strings.Join(commands, "|") + ").*$"
}

func standaloneDirectCommandAnchor(anchor string) bool {
	usage := strings.TrimSpace(content.PedagogicalTermUsage(anchor))
	return usage == anchor ||
		strings.HasPrefix(usage, anchor+" ") ||
		strings.HasPrefix(usage, anchor+" ;")
}

func standaloneCommandAnchorAllowed(anchor string) bool {
	switch anchor {
	case "vi", "vim", "screen", "tmux", "shutdown", "init", "telinit",
		"grub-install", "grub-mkconfig", "dpkg-reconfigure":
		return false
	default:
		return true
	}
}

func standaloneConceptFilename(concept curriculum.Concept) string {
	prefix := "lpic1." + concept.ObjectiveID + "."
	return strings.TrimPrefix(concept.ID, prefix)
}

func regexEscape(value string) string {
	var builder strings.Builder
	for _, r := range value {
		if strings.ContainsRune("\\.+*?()|[]{}^$", r) {
			builder.WriteByte('\\')
		}
		builder.WriteRune(r)
	}
	return builder.String()
}
