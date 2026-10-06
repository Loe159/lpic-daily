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
	labels := []string{"lpic-required"}
	checks := []CheckDefinition(nil)
	successCriteria := []string(nil)

	if objective.Exam == "101" {
		task, stateChecks, extraSetup := deterministicStandaloneExercise(objective, concept, root, variant)
		fmt.Fprintf(&tasks, "- %s\\n", task)
		checks = stateChecks
		labels = append(labels, "deterministic-state")
		successCriteria = []string{
			"Le checker recalcule ou inspecte directement l'état Linux attendu.",
			"Une simple déclaration écrite ou un code de sortie arbitraire ne suffit pas.",
			"Le contexte diagnostic et le contexte transfert demandent des productions distinctes.",
		}
		if setup.ExecutionScope == "sandbox" && extraSetup != "" {
			setupScript += extraSetup
		}
		if pattern := standaloneCommandEvidencePattern(concept); pattern != "" {
			checks = append(checks, CheckDefinition{
				Type:       "file-content-regex",
				Path:       commandHistoryPath,
				Pattern:    pattern,
				ConceptIDs: []string{concept.ID},
			})
		}
	} else {
		fmt.Fprintf(
			&tasks,
			"- %s : travaille les repères %s, réalise une commande, une inspection ou une configuration pertinente, puis écris %s/%s.txt.\\n",
			concept.TitleFR,
			strings.Join(terms, ", "),
			root,
			filename,
		)
		checks = []CheckDefinition{{
			Type: "file-content-regex",
			Path: root + "/" + filename + ".txt",
			Pattern: "(?m)^CONCEPT=" + regexEscape(concept.ID) +
				"\\\\nTERMS=" + regexEscape(termsLine) +
				"\\\\nCOMMAND=.+\\\\nOBSERVATION=.+\\\\nEXPLANATION=.+\\\\n?$",
			ConceptIDs: []string{concept.ID},
		}}
		successCriteria = []string{
			"Une preuve distincte est produite pour chaque concept.",
			"Chaque preuve couvre explicitement les termes/fichiers/utilitaires affectés au concept.",
			"Chaque preuve explique le lien entre le résultat observé et le concept LPIC.",
		}
	}

	brief := fmt.Sprintf(
		"Contexte de %s pour %s — %s, concept **%s**. Travaille uniquement ce concept ; les autres notions de l'objectif seront proposées séparément.\\n\\n%s\\n"+
			"Crée %s si nécessaire. Exécute réellement les inspections ou manipulations demandées : le checker vérifie l'état du sandbox/VM et non une auto-déclaration.",
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
			SchemaVersion:     "1.0.0",
			ID:                labID,
			TitleFR:           fmt.Sprintf("%s — %s — pratique %s", objective.ID, concept.TitleFR, contextFR),
			BriefFR:           brief,
			SuccessCriteriaFR: successCriteria,
			DebriefFR: fmt.Sprintf(
				"%s %s Le contexte %s oblige à reformuler et vérifier chaque sous-concept au lieu de valider l'objectif par une seule commande.",
				guide.Overview,
				guide.Pitfalls,
				contextFR,
			),
			ObjectiveIDs:     []string{objective.ID},
			ConceptIDs:       []string{concept.ID},
			Labels:           labels,
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

func deterministicStandaloneExercise(
	objective curriculum.Objective,
	concept curriculum.Concept,
	root string,
	variant int,
) (string, []CheckDefinition, string) {
	result := root + "/" + standaloneConceptFilename(concept) + ".result"
	source := root + "/" + standaloneConceptFilename(concept) + ".source"

	switch objective.ID {
	case "103.2":
		return deterministicTextExercise(concept, source, result, variant)
	case "103.3":
		return deterministicFileExercise(concept, root, source, result, variant)
	case "103.4":
		return deterministicRedirectionExercise(concept, root, result, variant)
	case "103.6":
		return deterministicPriorityExercise(concept, root, result, variant)
	case "103.7":
		return deterministicRegexExercise(concept, source, result, variant)
	case "103.8":
		return deterministicEditorExercise(concept, result, variant)
	case "104.5":
		return deterministicPermissionExercise(concept, root, variant)
	case "104.6":
		return deterministicLinkExercise(concept, root, variant)
	}

	probe := deterministicInspectionProbe(objective, concept)
	if variant == 0 {
		script := fmt.Sprintf(
			"set -eu; a=$(mktemp); trap 'rm -f \"$a\"' EXIT; ( %s ) >\"$a\"; test -f %q; cmp -s %q \"$a\"",
			probe, result, result,
		)
		return fmt.Sprintf(
			"Diagnostic : inspecte l'état Linux réel lié à %s et écris le relevé canonique dans %s. Le checker recalcule cet état indépendamment.",
			concept.TitleFR, result,
		), []CheckDefinition{deterministicCommandCheck(concept.ID, script)}, ""
	}

	script := fmt.Sprintf(
		"set -eu; expected=$(( %s ) | sha256sum | awk '{print $1}'); test -f %q; test \"$(tr -d '[:space:]' < %q)\" = \"$expected\"",
		probe, result, result,
	)
	return fmt.Sprintf(
		"Transfert : réinspecte %s dans ce second contexte, dérive le SHA-256 du relevé canonique et écris uniquement l'empreinte dans %s. Le checker dérive sa propre empreinte.",
		concept.TitleFR, result,
	), []CheckDefinition{deterministicCommandCheck(concept.ID, script)}, ""
}

func deterministicInspectionProbe(objective curriculum.Objective, concept curriculum.Concept) string {
	var parts []string
	for _, anchor := range concept.AnchorTerms {
		trimmed := strings.TrimSpace(anchor)
		if strings.HasPrefix(trimmed, "/") {
			parts = append(parts, fmt.Sprintf("if test -e %q; then stat -Lc '%%n|%%F|%%a' %q; else printf 'missing|%%s\\n' %q; fi", trimmed, trimmed, trimmed))
			continue
		}
		if standaloneCommandAnchorAllowed(trimmed) && standaloneDirectCommandAnchor(trimmed) {
			parts = append(parts, fmt.Sprintf("printf 'command|%%s|' %q; command -v %s 2>/dev/null || true", trimmed, trimmed))
		}
	}
	switch objective.ID {
	case "101.1":
		parts = append(parts, "lsblk -ndo NAME,TYPE,TRAN 2>/dev/null | sort", "find /sys/class -mindepth 1 -maxdepth 1 -printf '%f\\n' 2>/dev/null | sort | head -20")
	case "101.2":
		parts = append(parts, "printf 'pid1|'; cat /proc/1/comm; printf 'cmdline|'; cat /proc/cmdline", "if test -d /sys/firmware/efi; then echo firmware=UEFI; else echo firmware=BIOS; fi")
	case "101.3":
		parts = append(parts, "systemctl get-default 2>/dev/null || true", "systemctl list-jobs --no-pager 2>/dev/null | sed -n '1,12p'")
	case "102.1", "104.1":
		parts = append(parts, "lsblk -o NAME,TYPE,SIZE,PTTYPE,FSTYPE,MOUNTPOINTS 2>/dev/null")
	case "102.2":
		parts = append(parts, "for p in /boot/grub2/grub.cfg /boot/grub/grub.cfg /boot/grub/menu.lst; do test -e \"$p\" && printf 'bootcfg|%s\\n' \"$p\"; done")
	case "102.3":
		parts = append(parts, "ldd /bin/sh 2>/dev/null | sed -n '1,12p'", "ldconfig -p 2>/dev/null | sed -n '1,12p'")
	case "102.4":
		parts = append(parts, "dpkg-query -W bash 2>/dev/null || true", "apt-cache depends bash 2>/dev/null | sed -n '1,10p'")
	case "102.5":
		parts = append(parts, "rpm -qa 2>/dev/null | sort | head -10", "{ dnf repolist 2>/dev/null || yum repolist 2>/dev/null || zypper lr 2>/dev/null || true; } | sed -n '1,12p'")
	case "102.6":
		parts = append(parts, "systemd-detect-virt 2>/dev/null || true", "printf 'machine-id|'; cat /etc/machine-id 2>/dev/null; printf 'hostname|'; hostname")
	case "103.1":
		parts = append(parts, "printf 'path|%s\\n' \"$PATH\"; command -V sh; command -V printf")
	case "104.2":
		parts = append(parts, "df -P / | tail -1", "df -Pi / | tail -1")
	case "104.3":
		parts = append(parts, "findmnt -rn -o SOURCE,TARGET,FSTYPE,OPTIONS | head -12", "grep -Ev '^[[:space:]]*(#|$)' /etc/fstab 2>/dev/null || true")
	case "104.7":
		parts = append(parts, "for p in /etc /var /usr /home /tmp /boot /opt /srv; do test -e \"$p\" && stat -Lc '%n|%F' \"$p\"; done", "whereis sh")
	}
	if len(parts) == 0 {
		parts = append(parts, "uname -srm", "pwd")
	}
	return strings.Join(parts, "; ")
}

func deterministicCommandCheck(conceptID, script string) CheckDefinition {
	return CheckDefinition{
		Type:         "command-exit",
		Argv:         []string{"/bin/sh", "-c", script},
		ExpectedExit: 0,
		ConceptIDs:   []string{conceptID},
	}
}

func deterministicTextExercise(concept curriculum.Concept, source, result string, variant int) (string, []CheckDefinition, string) {
	payload := "delta:3\\nalpha:1\\nbeta:2\\nalpha:1\\n"
	if variant == 1 {
		payload = "kiwi:7\\npear:4\\nkiwi:7\\napple:2\\n"
	}
	setup := fmt.Sprintf("printf '%%b' %q > %q\\n", payload, source)
	var task, verify string
	switch concept.PedagogyOrder {
	case 1:
		task, verify = "Recopie exactement le flux source dans le fichier résultat avec un outil de lecture de texte.", fmt.Sprintf("cmp -s %q %q", source, result)
	case 2:
		task, verify = "Extrais uniquement la première colonne séparée par ':' dans le fichier résultat.", fmt.Sprintf("cut -d: -f1 %q | cmp -s - %q", source, result)
	case 3:
		task, verify = "Trie puis déduplique le flux source dans le fichier résultat.", fmt.Sprintf("sort %q | uniq | cmp -s - %q", source, result)
	case 4:
		task, verify = "Assemble dans le fichier résultat les deux premières lignes de la source avec ':' comme séparateur.", fmt.Sprintf("head -2 %q | paste -sd: - | cmp -s - %q", source, result)
	case 5:
		task, verify = "Transforme toutes les minuscules ASCII de la source en majuscules.", fmt.Sprintf("tr '[:lower:]' '[:upper:]' < %q | cmp -s - %q", source, result)
	case 6:
		task, verify = "Écris uniquement le SHA-256 de la source.", fmt.Sprintf("test \"$(tr -d '[:space:]' < %q)\" = \"$(sha256sum %q | awk '{print $1}')\"", result, source)
	default:
		compressed := source + ".gz"
		setup += fmt.Sprintf("gzip -c %q > %q\\n", source, compressed)
		task, verify = "Lis le fichier gzip en flux et reconstitue exactement son contenu.", fmt.Sprintf("gzip -cd %q | cmp -s - %q", compressed, result)
	}
	return fmt.Sprintf("%s Source: %s. Résultat attendu dans %s.", task, source, result), []CheckDefinition{deterministicCommandCheck(concept.ID, verify)}, setup
}

func deterministicFileExercise(concept curriculum.Concept, root, source, result string, variant int) (string, []CheckDefinition, string) {
	setup := fmt.Sprintf("printf 'variant-%d\\n' > %q\\nmkdir -p %q/tree/a %q/tree/b\\nprintf 'one\\n' > %q/tree/a/one.log\\nprintf 'two\\n' > %q/tree/b/two.txt\\n", variant+1, source, root, root, root, root)
	var task, verify string
	switch concept.PedagogyOrder {
	case 1:
		task, verify = "Copie la source puis déplace la copie vers le résultat final.", fmt.Sprintf("cmp -s %q %q", source, result)
	case 2:
		task, verify = "Utilise le globbing pour produire la liste triée des fichiers .log de tree.", fmt.Sprintf("printf '%%s\\n' %s/tree/*/*.log | sort | cmp -s - %q", root, result)
	case 3:
		task, verify = "Trouve uniquement les fichiers réguliers .log sous tree et écris les chemins triés.", fmt.Sprintf("find %q/tree -type f -name '*.log' | sort | cmp -s - %q", root, result)
	case 4:
		task, verify = "Avec find, applique le mode 0640 à tous les fichiers sous tree.", fmt.Sprintf("test -z \"$(find %q/tree -type f ! -perm 0640 -print -quit)\"", root)
	case 5:
		archive := result + ".tar"
		task, verify = "Crée une archive tar du répertoire tree.", fmt.Sprintf("tar -tf %q | grep -q 'tree/a/one.log' && tar -tf %q | grep -q 'tree/b/two.txt'", archive, archive)
	case 6:
		task, verify = "Copie la source octet pour octet avec dd vers le résultat.", fmt.Sprintf("cmp -s %q %q", source, result)
	case 7:
		compressed := result + ".gz"
		task, verify = "Compresse la source avec gzip sans altérer la source.", fmt.Sprintf("gzip -cd %q | cmp -s - %q", compressed, source)
	default:
		task, verify = "Écris la description courte produite par file pour la source.", fmt.Sprintf("file -b %q | cmp -s - %q", source, result)
	}
	return fmt.Sprintf("%s Source: %s. Résultat: %s.", task, source, result), []CheckDefinition{deterministicCommandCheck(concept.ID, verify)}, setup
}

func deterministicRedirectionExercise(concept curriculum.Concept, root, result string, variant int) (string, []CheckDefinition, string) {
	first, second := "alpha", "omega"
	if variant == 1 {
		first, second = "delta", "sigma"
	}
	var task, verify string
	switch concept.PedagogyOrder {
	case 1:
		task, verify = "Sépare stdout et stderr dans deux fichiers .out et .err.", fmt.Sprintf("test \"$(cat %q.out)\" = %q && test \"$(cat %q.err)\" = %q", result, first, result, second)
	case 2:
		task, verify = "Écris la première valeur puis ajoute la seconde avec >>.", fmt.Sprintf("test \"$(cat %q)\" = \"$(printf '%%s\\n%%s' %q %q)\"", result, first, second)
	case 3:
		task, verify = "Redirige uniquement stderr dans le résultat.", fmt.Sprintf("grep -Fq %q %q", second, result)
	case 4:
		task, verify = "Utilise un pipeline pour ne conserver que la seconde valeur.", fmt.Sprintf("test \"$(tr -d '\\n' < %q)\" = %q", result, second)
	case 5:
		task, verify = "Utilise tee pour dupliquer la première valeur dans le résultat et une copie .copy.", fmt.Sprintf("test \"$(cat %q)\" = %q && cmp -s %q %q.copy", result, first, result, result)
	case 6:
		task, verify = "Utilise xargs pour transformer les deux valeurs reçues sur stdin en deux fichiers dans le répertoire de preuve.", fmt.Sprintf("test -f %q/%s && test -f %q/%s", root, first, root, second)
	default:
		task, verify = "Réunis stdout et stderr dans le même fichier en respectant l'ordre des redirections.", fmt.Sprintf("grep -Fq %q %q && grep -Fq %q %q", first, result, second, result)
	}
	return fmt.Sprintf("%s Valeurs: %s et %s. Cible: %s.", task, first, second, result), []CheckDefinition{deterministicCommandCheck(concept.ID, verify)}, ""
}

func deterministicPriorityExercise(concept curriculum.Concept, root, result string, variant int) (string, []CheckDefinition, string) {
	pidPath := root + "/priority.pid"
	target := 7
	if variant == 1 {
		target = 11
	}
	switch concept.PedagogyOrder {
	case 1, 3:
		script := fmt.Sprintf("test -s %q && test \"$(ps -o ni= -p \"$(cat %q)\" | tr -d ' ')\" = %d", pidPath, pidPath, target)
		return fmt.Sprintf("Démarre sleep 600 avec une nice value de %d et écris son PID dans %s.", target, pidPath), []CheckDefinition{deterministicCommandCheck(concept.ID, script)}, ""
	case 2:
		script := fmt.Sprintf("test -s %q && test \"$(ps -o ni= -p \"$(cat %q)\" | tr -d ' ')\" = 0", pidPath, pidPath)
		return fmt.Sprintf("Démarre sleep 600 avec la priorité par défaut et écris son PID dans %s.", pidPath), []CheckDefinition{deterministicCommandCheck(concept.ID, script)}, ""
	case 4:
		setup := fmt.Sprintf("sleep 600 & echo $! > %q\\n", pidPath)
		script := fmt.Sprintf("test \"$(ps -o ni= -p \"$(cat %q)\" | tr -d ' ')\" = %d", pidPath, target)
		return fmt.Sprintf("Change avec renice la nice value du PID contenu dans %s vers %d.", pidPath, target), []CheckDefinition{deterministicCommandCheck(concept.ID, script)}, setup
	default:
		setup := fmt.Sprintf("nice -n %d sleep 600 & echo $! > %q\\n", target, pidPath)
		script := fmt.Sprintf("test \"$(tr -d '[:space:]' < %q)\" = \"$(ps -o ni= -p \"$(cat %q)\" | tr -d ' ')\"", result, pidPath)
		return fmt.Sprintf("Observe avec ps ou top la nice value du PID dans %s et écris uniquement la valeur dans %s.", pidPath, result), []CheckDefinition{deterministicCommandCheck(concept.ID, script)}, setup
	}
}

func deterministicRegexExercise(concept curriculum.Concept, source, result string, variant int) (string, []CheckDefinition, string) {
	payload := "alpha:12\\nbeta:7\\nALPHA:42\\ngamma:x\\n"
	if variant == 1 {
		payload = "node:31\\nNODE:8\\nedge:x\\nnode:55\\n"
	}
	setup := fmt.Sprintf("printf '%%b' %q > %q\\n", payload, source)
	if concept.PedagogyOrder == 6 {
		verify := fmt.Sprintf("sed -E 's/:[0-9]+$/:N/' %q | cmp -s - %q", source, result)
		return fmt.Sprintf("Avec sed, remplace les valeurs numériques finales par N dans %s et écris le flux dans %s.", source, result), []CheckDefinition{deterministicCommandCheck(concept.ID, verify)}, setup
	}
	pattern := "^[a-z]+:[0-9]+$"
	if concept.PedagogyOrder == 2 {
		pattern = "^[[:alpha:]]+:[0-9]+$"
	}
	if concept.PedagogyOrder == 3 {
		pattern = ":[0-9][0-9]*$"
	}
	if concept.PedagogyOrder == 4 {
		pattern = "^(alpha|node):[0-9]+$"
	}
	if concept.PedagogyOrder == 5 {
		pattern = "^[a-z]+:"
	}
	verify := fmt.Sprintf("grep -E %q %q | cmp -s - %q", pattern, source, result)
	return fmt.Sprintf("Filtre %s avec la regex adaptée à %s et écris uniquement les lignes retenues dans %s.", source, concept.TitleFR, result), []CheckDefinition{deterministicCommandCheck(concept.ID, verify)}, setup
}

func deterministicEditorExercise(concept curriculum.Concept, result string, variant int) (string, []CheckDefinition, string) {
	initial, expected := "one\\ntwo\\nthree\\n", "one\\nTWO\\nthree\\n"
	if variant == 1 {
		initial, expected = "red\\ngreen\\nblue\\n", "red\\nGREEN\\nblue\\n"
	}
	setup := fmt.Sprintf("printf '%%b' %q > %q\\n", initial, result)
	verify := fmt.Sprintf("test \"$(cat %q)\" = \"$(printf '%%b' %q)\"", result, expected)
	return fmt.Sprintf("Édite %s avec un éditeur concerné par %s : seule la ligne centrale doit passer en majuscules.", result, concept.TitleFR), []CheckDefinition{deterministicCommandCheck(concept.ID, verify)}, setup
}

func deterministicPermissionExercise(concept curriculum.Concept, root string, variant int) (string, []CheckDefinition, string) {
	path := root + "/perm-target"
	switch concept.PedagogyOrder {
	case 1, 2:
		mode := "0640"
		if variant == 1 {
			mode = "0750"
		}
		return fmt.Sprintf("Crée %s puis fixe exactement ses permissions à %s.", path, mode), []CheckDefinition{{Type: "file-mode", Path: path, Mode: mode, ConceptIDs: []string{concept.ID}}}, ""
	case 3:
		return fmt.Sprintf("Crée %s et fixe son propriétaire/groupe à root:root.", path), []CheckDefinition{{Type: "file-owner", Path: path, User: "root", Group: "root", ConceptIDs: []string{concept.ID}}}, ""
	case 4:
		return fmt.Sprintf("Applique umask 027 puis crée %s ; son mode final doit être 0640.", path), []CheckDefinition{{Type: "file-mode", Path: path, Mode: "0640", ConceptIDs: []string{concept.ID}}}, ""
	case 5:
		return fmt.Sprintf("Crée %s exécutable et active le bit SUID.", path), []CheckDefinition{deterministicCommandCheck(concept.ID, fmt.Sprintf("test -u %q", path))}, ""
	case 6:
		return fmt.Sprintf("Crée le répertoire %s et active le bit SGID.", path), []CheckDefinition{deterministicCommandCheck(concept.ID, fmt.Sprintf("test -d %q && test -g %q", path, path))}, ""
	case 7:
		return fmt.Sprintf("Crée le répertoire %s et active le sticky bit.", path), []CheckDefinition{deterministicCommandCheck(concept.ID, fmt.Sprintf("test -d %q && test -k %q", path, path))}, ""
	default:
		return fmt.Sprintf("Crée le répertoire partagé %s en mode 3770.", path), []CheckDefinition{{Type: "file-mode", Path: path, Mode: "3770", ConceptIDs: []string{concept.ID}}}, ""
	}
}

func deterministicLinkExercise(concept curriculum.Concept, root string, variant int) (string, []CheckDefinition, string) {
	target, link := root+"/target", root+"/link"
	setup := fmt.Sprintf("printf 'variant-%d\\n' > %q\\n", variant+1, target)
	switch concept.PedagogyOrder {
	case 1, 3:
		script := fmt.Sprintf("test %q -ef %q && test \"$(stat -c '%%i' %q)\" = \"$(stat -c '%%i' %q)\"", target, link, target, link)
		return fmt.Sprintf("Crée %s comme hard link de %s.", link, target), []CheckDefinition{deterministicCommandCheck(concept.ID, script)}, setup
	case 2:
		script := fmt.Sprintf("test -L %q && test \"$(readlink -f %q)\" = \"$(readlink -f %q)\"", link, link, target)
		return fmt.Sprintf("Crée %s comme symlink vers %s.", link, target), []CheckDefinition{deterministicCommandCheck(concept.ID, script)}, setup
	case 4:
		setup += fmt.Sprintf("ln -s %q %q\\nrm -f %q\\n", target, link, target)
		return fmt.Sprintf("Le lien %s doit rester un symlink cassé ; diagnostique-le sans recréer la cible.", link), []CheckDefinition{deterministicCommandCheck(concept.ID, fmt.Sprintf("test -L %q && test ! -e %q", link, link))}, setup
	case 5:
		copyPath, hardPath, symPath := root+"/copy", root+"/hard", root+"/sym"
		script := fmt.Sprintf("cmp -s %q %q && test %q -ef %q && test -L %q", target, copyPath, target, hardPath, symPath)
		return fmt.Sprintf("À partir de %s, crée une copie %s, un hard link %s et un symlink %s.", target, copyPath, hardPath, symPath), []CheckDefinition{deterministicCommandCheck(concept.ID, script)}, setup
	default:
		script := fmt.Sprintf("test -L %q && test \"$(readlink -f %q)\" = \"$(readlink -f %q)\"", link, link, target)
		return fmt.Sprintf("Crée %s comme symlink administratif vers %s.", link, target), []CheckDefinition{deterministicCommandCheck(concept.ID, script)}, setup
	}
}

func deterministicStandaloneLab(definition Definition) bool {
	return strings.Contains(definition.ID, ".standalone-") && slices.Contains(definition.Labels, "deterministic-state")
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
