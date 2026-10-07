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
	guide curriculum.ObjectiveStudyGuide,
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

	var tasks strings.Builder
	commandHistoryPath := root + "/command-history.log"
	filename := standaloneConceptFilename(concept)
	terms := slices.Clone(concept.AnchorTerms)
	termsLine := strings.Join(terms, ",")
	fmt.Fprintf(
		&tasks,
		"- **%s** : travaille les repères `%s`, réalise une commande, une inspection ou une configuration pertinente, puis écris `%s/%s.txt`.\n",
		concept.TitleFR,
		strings.Join(terms, "`, `"),
		root,
		filename,
	)
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
		"Contexte de %s pour %s — %s, concept **%s**. Travaille uniquement ce concept ; les autres notions de l'objectif seront proposées séparément.\n\n%s\n"+
			"Utilise exactement cinq lignes dans le fichier de preuve : "+
			"CONCEPT=<id>, TERMS=<repères demandés séparés par des virgules>, "+
			"COMMAND=<commande/action réellement utilisée>, OBSERVATION=<résultat constaté>, "+
			"EXPLANATION=<pourquoi ce résultat démontre le concept>. "+
			"Crée d'abord %s si le répertoire n'existe pas. Le but n'est pas de recopier le cours : "+
			"fais la manipulation dans le sandbox, puis explique l'état observé. "+
			"Pour les concepts qui demandent une commande réelle, exécute-la directement à l'invite lpic> : LPIC Daily journalise automatiquement la commande et son code de sortie dans une preuve séparée que le lab vérifie.",
		contextFR,
		objective.ID,
		objective.TitleFR,
		concept.TitleFR,
		tasks.String(),
		root,
	)

	hints := generatedStandaloneHints(labID, root)
	return Lab{
		Definition: Definition{
			SchemaVersion: "1.0.0",
			ID:            labID,
			TitleFR:       fmt.Sprintf("%s — %s — pratique %s", objective.ID, concept.TitleFR, contextFR),
			BriefFR:       brief,
			SuccessCriteriaFR: []string{
				"Une preuve distincte est produite pour chaque concept.",
				"Chaque preuve couvre explicitement les termes/fichiers/utilitaires affectés au concept et décrit une commande ou action réellement effectuée.",
				"Chaque preuve explique le lien entre le résultat observé et le concept LPIC.",
			},
			DebriefFR: fmt.Sprintf(
				"%s %s Le contexte %s oblige à reformuler et vérifier chaque sous-concept au lieu de valider l'objectif par une seule commande.",
				guide.Overview,
				guide.Pitfalls,
				contextFR,
			),
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
		return "libvirt", "debian-13-x86_64-v3", "debian"
	}
	if objective.ID == "102.5" && variant == 1 {
		return "libvirt", "opensuse-leap-16.0-x86_64-v3", "opensuse"
	}
	if strings.Contains(objective.RecommendedBackend, "podman") &&
		strings.Contains(objective.RecommendedBackend, "libvirt") {
		if variant == 0 {
			return "podman", "localhost/lpic-daily/fedora-phase1:1", "fedora"
		}
		return "libvirt", "fedora-44-x86_64-v3", "fedora"
	}
	if strings.HasPrefix(objective.ID, "101.") ||
		strings.HasPrefix(objective.ID, "109.") ||
		objective.ID == "110.3" ||
		strings.Contains(objective.RecommendedBackend, "libvirt") {
		return "libvirt", "fedora-44-x86_64-v3", "fedora"
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

func generatedStandaloneHints(labID, root string) []Hint {
	return []Hint{
		{
			SchemaVersion:  "1.0.0",
			ID:             labID + ".hint-1",
			LabID:          labID,
			Level:          1,
			ContentFR:      "Traite un seul concept à la fois : relis son intitulé, choisis un outil ou fichier pertinent, puis observe le système avant d'écrire la preuve.",
			EvidenceImpact: "none",
		},
		{
			SchemaVersion:  "1.0.0",
			ID:             labID + ".hint-2",
			LabID:          labID,
			Level:          2,
			ContentFR:      fmt.Sprintf("Crée %s puis un fichier par concept. COMMAND décrit ce que tu as fait; OBSERVATION contient le résultat concret.", root),
			EvidenceImpact: "material",
		},
		{
			SchemaVersion:  "1.0.0",
			ID:             labID + ".hint-3",
			LabID:          labID,
			Level:          3,
			ContentFR:      "Chaque preuve doit contenir CONCEPT, TERMS, COMMAND, OBSERVATION et EXPLANATION. Recopie exactement TERMS depuis le brief puis relie le résultat au comportement Linux attendu.",
			EvidenceImpact: "material",
		},
		{
			SchemaVersion: "1.0.0",
			ID:            labID + ".hint-4",
			LabID:         labID,
			Level:         4,
			ContentFR: "Format attendu : CONCEPT=<id exact>, TERMS=<liste exacte du brief>, COMMAND=<commande/action>, " +
				"OBSERVATION=<résultat>, EXPLANATION=<raison>. Répète ce format pour chaque fichier indiqué dans le brief.",
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

func standaloneCommandEvidencePattern(concept curriculum.Concept) string {
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
