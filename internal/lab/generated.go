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
			DebriefFR:        standaloneDebrief(concept),
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

var standalonePracticeOverrides = func() map[string][2]string {
	overrides := make(map[string][2]string, 120)
	overrides["lpic1.101.1.role-de-proc-sys-et-dev"] = [2]string{"Compare `ls -ld /proc /sys /dev`, puis observe `head -n 5 /proc/cpuinfo`, `ls /sys/class | head` et `ls -l /dev/null`. Associe chaque arborescence à son rôle.", "Choisis un processus dans `/proc`, une classe matérielle dans `/sys` et un nœud dans `/dev`. Explique ce que représente chacun et pourquoi ces trois arborescences ne sont pas interchangeables."}
	overrides["lpic1.101.1.sysfs-udev-et-d-bus-comme-couches-de-decouverte-gestion"] = [2]string{"Observe un périphérique avec `ls /sys/class/block`, puis affiche ses propriétés avec `udevadm info --query=property --name=/dev/null`. Distingue ce que fournit sysfs de ce qu'ajoute udev.", "Vérifie la présence du bus système avec `test -S /run/dbus/system_bus_socket && echo D-Bus-present`. Explique la différence entre l'état exposé par sysfs, les règles udev et la communication D-Bus."}
	overrides["lpic1.101.1.differences-entre-grandes-familles-de-stockage"] = [2]string{"Exécute `lsblk -o NAME,TYPE,TRAN,SIZE,MODEL`. Repère les périphériques bloc présents et relie leur nom/transport aux familles de stockage visibles.", "Compare `lsblk -d -o NAME,ROTA,TRAN` et `ls -l /dev | grep -E 'nvme|sd|vd' | head`. Explique pourquoi le nom sous `/dev` ne suffit pas toujours à identifier la technologie physique."}
	overrides["lpic1.101.2.bios-versus-uefi"] = [2]string{"Exécute `test -d /sys/firmware/efi && echo UEFI || echo BIOS-legacy`. Déduis le mode de démarrage de la machine et explique ce que ce test observe.", "Examine `ls -ld /sys/firmware/efi /boot/efi 2>/dev/null`. Explique quel élément est propre à UEFI et ce qu'on attendrait différemment sur une machine démarrée en mode BIOS."}
	overrides["lpic1.101.2.parametres-noyau-au-boot"] = [2]string{"Lis `cat /proc/cmdline`. Identifie au moins un paramètre transmis au noyau au démarrage et explique qui le lui fournit.", "Compare la ligne active de `/proc/cmdline` avec les fichiers de configuration du chargeur présents sous `/etc/default` ou `/boot`. Distingue configuration persistante et paramètres effectivement utilisés au boot courant."}
	overrides["lpic1.101.2.role-de-initramfs"] = [2]string{"Exécute `ls -lh /boot | grep -E 'initramfs|initrd'`. Repère l'image initramfs du noyau courant et explique pourquoi elle est nécessaire avant le montage du vrai système racine.", "Si `lsinitrd` est disponible, lance `lsinitrd | head -n 30`; sinon inspecte le nom de l'image dans `/boot`. Cite deux types de ressources que l'initramfs peut devoir fournir très tôt au boot."}
	overrides["lpic1.101.2.sysvinit-systemd-et-connaissance-historique-d-upstart"] = [2]string{"Exécute `ps -p 1 -o comm=` puis `systemctl get-default 2>/dev/null`. Identifie l'init réellement utilisé et relie-le à systemd, SysVinit ou Upstart.", "Compare les notions de target systemd et de runlevel SysV. Utilise `systemctl list-dependencies default.target | head` pour observer le modèle actif et explique où Upstart se situe historiquement."}
	overrides["lpic1.101.2.commandes-du-chargeur-de-demarrage"] = [2]string{"À partir d'un menu GRUB 2, indique quelles touches permettent d'éditer une entrée et d'ouvrir la console. Dans le lab, vérifie la version disponible avec `grub2-editenv --version 2>/dev/null || grub-editenv --version 2>/dev/null`.", "Explique comment modifier temporairement la ligne du noyau depuis GRUB sans rendre le changement persistant. Distingue clairement édition d'une entrée, console GRUB et régénération de `grub.cfg`."}
	overrides["lpic1.101.3.notion-d-evenements-acpi"] = [2]string{"Exécute `systemctl status acpid --no-pager 2>/dev/null || echo acpid-non-actif`. Explique quel type d'événement matériel ACPI un démon comme acpid peut traiter.", "Observe `ls /proc/acpi 2>/dev/null` et `ls /sys/class/power_supply 2>/dev/null`. Donne un exemple d'événement ACPI et distingue événement matériel de simple arrêt logiciel."}
	overrides["lpic1.102.1.separer-var-home-et-boot-selon-usage"] = [2]string{"Exécute `findmnt -no TARGET,SOURCE,FSTYPE / /var /home /boot 2>/dev/null`. Pour chaque chemin présent, explique l'intérêt ou non de le placer sur un système de fichiers séparé.", "Imagine un serveur dont `/var` grossit rapidement alors que `/home` contient des données utilisateurs. Propose une séparation de points de montage et explique quel incident elle limite."}
	overrides["lpic1.102.1.contraintes-boot-et-architecture"] = [2]string{"Inspecte `findmnt /boot 2>/dev/null` puis `ls -lh /boot | head`. Explique pourquoi `/boot` doit rester accessible au chargeur avant que le système complet soit disponible.", "Compare un démarrage BIOS et UEFI du point de vue des fichiers nécessaires avant le noyau. Indique où intervient `/boot` et quand une partition EFI séparée est requise."}
	overrides["lpic1.102.1.efi-system-partition"] = [2]string{"Exécute `findmnt /boot/efi 2>/dev/null || ls -ld /boot/efi 2>/dev/null`. Explique le rôle de l'EFI System Partition et le type de fichiers qu'elle contient.", "Utilise `lsblk -o NAME,FSTYPE,PARTTYPENAME,MOUNTPOINTS`. Repère une éventuelle EFI System Partition et explique pourquoi elle est généralement en FAT plutôt qu'en ext4."}
	overrides["lpic1.102.1.partitions-et-points-de-montage"] = [2]string{"Compare `lsblk -o NAME,TYPE,FSTYPE,MOUNTPOINTS` et `findmnt`. Associe une partition ou un volume à son point de montage sans confondre périphérique et répertoire.", "Choisis un périphérique non système du lab. Décris les étapes qui le font passer de partition disponible à système de fichiers accessible via un point de montage."}
	overrides["lpic1.102.1.principes-de-base-lvm"] = [2]string{"Exécute `pvs 2>/dev/null; vgs 2>/dev/null; lvs 2>/dev/null`. Même si aucun volume n'existe, explique la relation PV → VG → LV.", "Pour un disque ajouté au lab, décris comment tu répartirais sa capacité avec LVM. Distingue clairement partition/disque physique, physical volume, volume group et logical volume."}
	overrides["lpic1.102.2.grub-legacy-versus-grub-2"] = [2]string{"Recherche `menu.lst`, `grub.cfg` et `/etc/default/grub` sous `/boot` et `/etc`. Associe `menu.lst` à GRUB Legacy et `grub.cfg` à GRUB 2.", "Compare la manière de modifier durablement GRUB Legacy et GRUB 2. Explique pourquoi éditer directement un `grub.cfg` généré n'est généralement pas la bonne méthode sous GRUB 2."}
	overrides["lpic1.102.2.entrees-alternatives-et-recuperation"] = [2]string{"Liste les entrées GRUB visibles dans `grub.cfg` avec `grep -E '^menuentry ' /boot/grub*/grub.cfg 2>/dev/null | head`. Repère comment plusieurs noyaux ou modes de récupération peuvent coexister.", "Imagine que le noyau par défaut ne démarre plus. Explique comment sélectionner une entrée alternative au menu GRUB 2 sans modifier d'abord la configuration persistante."}
	overrides["lpic1.102.2.mbr-et-relation-avec-schema-de-boot"] = [2]string{"Exécute `lsblk -o NAME,PTTYPE,PARTTYPE`. Repère si les disques utilisent `dos`/MBR ou GPT et relie ce choix au schéma de démarrage.", "Compare le rôle du MBR dans un démarrage BIOS classique avec celui de l'EFI System Partition en UEFI. Indique ce que GPT change dans ce raisonnement."}
	overrides["lpic1.102.2.generation-de-configuration-grub-2"] = [2]string{"Repère `/etc/default/grub` et les scripts de `/etc/grub.d` s'ils existent. Explique comment ils participent à la génération de `grub.cfg`.", "Sans modifier le système, indique la commande qui régénérerait `grub.cfg` sur cette distribution et la cible correcte. Explique pourquoi la cible de sortie compte."}
	overrides["lpic1.102.2.interaction-au-menu-console-grub"] = [2]string{"Explique la différence entre sélectionner une entrée, l'éditer temporairement et ouvrir la console GRUB 2. Identifie les touches habituelles `e` et `c`.", "Dans un scénario où un paramètre noyau doit être testé une seule fois, décris précisément l'action au menu GRUB et pourquoi aucune régénération de configuration n'est nécessaire."}
	overrides["lpic1.102.3.override-via-variables-d-environnement"] = [2]string{"Exécute `printf '%s\\n' \"$LD_LIBRARY_PATH\"` puis `LD_LIBRARY_PATH=/tmp ldd /bin/sh 2>/dev/null | head`. Explique comment la variable peut modifier la recherche de bibliothèques d'un processus enfant.", "Compare un lancement normal d'un binaire et `LD_LIBRARY_PATH=/tmp commande`. Explique pourquoi cet override est local au processus lancé et pourquoi il peut être risqué."}
	overrides["lpic1.102.6.vm-versus-conteneur-applicatif-systeme"] = [2]string{"Exécute `systemd-detect-virt 2>/dev/null || true` et `uname -r`. Explique ce qu'une VM virtualise par rapport à un conteneur qui partage le noyau de l'hôte.", "Classe trois cas : VM complète, conteneur système et conteneur applicatif. Pour chacun, indique s'il possède son propre noyau et quel niveau d'isolation il vise."}
	overrides["lpic1.102.6.ressources-virtuelles-block-network"] = [2]string{"Observe `lsblk` et `ip link`. Identifie un périphérique bloc et une interface réseau présentés au système invité, puis explique pourquoi ils peuvent être virtuels.", "Imagine l'ajout à chaud d'un disque et d'une carte réseau virtuels. Indique ce que l'invité verrait et quelles commandes permettraient de confirmer leur apparition."}
	overrides["lpic1.102.6.images-clones-et-templates"] = [2]string{"Compare les notions d'image, clone et template : pour chacune, précise si elle sert de source réutilisable ou d'instance exécutable.", "Imagine dix VM créées depuis le même template. Liste deux éléments qui doivent devenir propres à chaque clone avant mise en production."}
	overrides["lpic1.102.6.identites-devant-etre-regenerees-lors-d-un-clonage"] = [2]string{"Examine `cat /etc/machine-id` et `ls /etc/ssh/ssh_host_* 2>/dev/null`. Explique pourquoi ces identités ne doivent pas être dupliquées telles quelles entre clones.", "Pour une VM clonée, indique comment traiter le D-Bus machine-id et les SSH host keys avant le premier usage réseau. Explique le risque d'une duplication."}
	overrides["lpic1.102.6.guest-drivers"] = [2]string{"Exécute `lspci -k 2>/dev/null | head -n 30`. Repère un périphérique et son pilote noyau, puis explique le rôle d'un guest driver dans une VM.", "Compare un périphérique émulé et un périphérique paravirtualisé. Explique pourquoi un pilote invité adapté peut améliorer performances et intégration."}
	overrides["lpic1.102.6.principes-cloud-init-iaas"] = [2]string{"Vérifie `cloud-init status 2>/dev/null || echo cloud-init-absent`. Explique quelles informations cloud-init peut appliquer au premier démarrage d'une instance IaaS.", "Imagine une image générique déployée chez un fournisseur IaaS. Cite trois éléments que cloud-init peut personnaliser sans reconstruire l'image."}
	overrides["lpic1.103.1.quoting-escaping"] = [2]string{"Compare `printf '<%s>\\n' '$HOME' \"$HOME\"`. Explique pourquoi les quotes simples gardent le texte littéral alors que les doubles quotes permettent l'expansion.", "Crée `FILE='deux mots'`, puis compare `printf '<%s>\\n' $FILE` et `printf '<%s>\\n' \"$FILE\"`. Explique l'effet du quoting sur le nombre d'arguments."}
	overrides["lpic1.103.3.recursivite-et-globbing"] = [2]string{"Crée `demo/a.txt` et `demo/b.log`, puis exécute `printf '%s\\n' demo/*.txt`. Explique que le glob est développé par le shell avant l'exécution de la commande.", "Compare `ls demo/*` et une opération récursive comme `find demo -type f`. Explique pourquoi globbing et récursivité répondent à des besoins différents."}
	overrides["lpic1.103.4.stdin-stdout-stderr-et-descripteurs"] = [2]string{"Exécute `sh -c 'echo sortie; echo erreur >&2' >out.txt 2>err.txt`, puis lis les deux fichiers. Associe stdin, stdout et stderr aux descripteurs 0, 1 et 2.", "Exécute `printf 'entrée\\n' | cat >copie.txt`. Identifie le flux utilisé à chaque étape et explique pourquoi stderr n'est pas inclus automatiquement dans le pipe."}
	overrides["lpic1.103.6.priorite-par-defaut"] = [2]string{"Exécute `ps -o pid,ni,comm -p $$`. Observe la valeur nice du shell et rappelle la valeur par défaut habituelle d'un processus lancé sans ajustement.", "Lance `sleep 30 &`, relève son nice avec `ps -o ni= -p $!`, puis termine-le. Compare sa priorité à celle du shell qui l'a lancé."}
	overrides["lpic1.103.7.quantificateurs-et-ancres"] = [2]string{"Teste `printf 'ab\\naab\\nb\\n' | grep -E '^a+b$'`. Identifie le rôle de `+`, `^` et `$` dans le motif.", "Compare `grep -E 'ab'` et `grep -E '^ab$'` sur plusieurs lignes contenant `ab`. Explique ce que les ancres changent dans la correspondance."}
	overrides["lpic1.103.7.groupes-alternatives"] = [2]string{"Teste `printf 'cat\\ndog\\ncow\\n' | grep -E '^(cat|dog)$'`. Identifie le groupe et l'alternative dans l'expression.", "Construis une ERE qui accepte exactement `red1` ou `blue2`, puis teste-la avec `printf`. Explique comment parenthèses et `|` structurent le motif."}
	overrides["lpic1.103.8.navigation"] = [2]string{"Ouvre un petit fichier avec `vi` ou `vim`. Déplace le curseur uniquement avec `h`, `j`, `k`, `l`, puis quitte sans modifier le fichier.", "Dans `vi`, place le curseur sur une ligne précise en combinant plusieurs mouvements `h/j/k/l`. Explique quel axe contrôle chaque touche."}
	overrides["lpic1.103.8.insertion-remplacement"] = [2]string{"Dans `vi`, teste `i`, `a` et `o` sur un fichier temporaire. Observe où commence l'insertion pour chacune des trois commandes.", "Édite un fichier temporaire avec `vi` : ajoute du texte avant le curseur, après le curseur puis sur une nouvelle ligne. Associe chaque action à `i`, `a` ou `o`."}
	overrides["lpic1.103.8.recherche"] = [2]string{"Dans `vi`, ouvre un fichier contenant plusieurs occurrences d'un mot. Utilise `/mot` puis `?mot` et observe le sens de recherche.", "Effectue une recherche vers l'avant puis vers l'arrière dans `vi`. Explique la différence entre `/` et `?` et comment répéter une recherche."}
	overrides["lpic1.103.8.sauvegarde-quitter"] = [2]string{"Dans `vi`, modifie un fichier temporaire puis teste la différence entre `:w`, `:q!` et `ZZ` sur des sessions séparées.", "Crée une modification non sauvegardée dans `vi`. Explique quand `:w!`, `:q!` et `ZZ` écrivent, quittent ou forcent l'action."}
	overrides["lpic1.103.8.editeur-par-defaut"] = [2]string{"Exécute `printf '%s\\n' \"$EDITOR\"`. Si la variable est vide, définis `EDITOR=vi` dans le shell puis vérifie sa valeur.", "Lance un processus enfant avec `EDITOR=vi env | grep '^EDITOR='`. Explique comment une application peut utiliser `EDITOR` pour choisir l'éditeur à ouvrir."}
	overrides["lpic1.104.1.mbr-versus-gpt"] = [2]string{"Exécute `lsblk -o NAME,SIZE,PTTYPE`. Repère le type de table de partitions des disques et distingue `dos`/MBR de GPT.", "Sur les disques de pratique du lab, crée ou inspecte une table GPT sur l'un et compare ses caractéristiques avec une table MBR. Ne touche pas au disque système."}
	overrides["lpic1.104.1.ext2-ext3-ext4"] = [2]string{"Exécute `lsblk -f` et repère d'éventuels ext2/ext3/ext4. Explique les différences historiques essentielles, notamment la journalisation.", "Sur un périphérique de pratique, crée un ext4 si demandé par le lab puis vérifie son type avec `blkid`. Explique pourquoi ext4 est généralement préféré à ext2 aujourd'hui."}
	overrides["lpic1.104.1.xfs"] = [2]string{"Exécute `lsblk -f | grep -i xfs || true`. Explique les caractéristiques générales de XFS et dans quels usages on le rencontre.", "Si un disque de pratique est disponible, indique la commande de création d'un XFS et la commande qui permettrait d'en vérifier le type sans toucher au disque système."}
	overrides["lpic1.104.1.vfat-exfat"] = [2]string{"Exécute `lsblk -f | grep -Ei 'vfat|exfat' || true`. Distingue VFAT et exFAT des filesystems Unix comme ext4, notamment pour les permissions POSIX.", "Pour un support amovible destiné à être partagé avec plusieurs OS, compare VFAT et exFAT et choisis lequel utiliser selon taille de fichiers et compatibilité."}
	overrides["lpic1.104.1.principes-btrfs-multi-device-compression-subvolumes"] = [2]string{"Vérifie `btrfs filesystem show 2>/dev/null || true`. Explique les notions de subvolume, compression et filesystem multi-device propres à Btrfs.", "Imagine un volume Btrfs contenant `/home`. Explique comment un subvolume diffère d'une partition et pourquoi snapshots/compression peuvent être utiles."}
	overrides["lpic1.104.3.fstab-persistant"] = [2]string{"Lis `sed -n '1,120p' /etc/fstab`. Pour une ligne non commentée, identifie source, point de montage, type, options, dump et pass.", "Compare un `mount` manuel avec une entrée `/etc/fstab`. Explique ce qui rend le montage persistant au redémarrage et comment tester une modification avant de rebooter."}
	overrides["lpic1.104.3.unites-mount-systemd"] = [2]string{"Exécute `systemctl list-units --type=mount --all | head -n 20`. Repère une unité `.mount` et relie son nom au point de montage.", "Choisis un point de montage comme `/mnt/data` et déduis le nom d'unité systemd correspondant. Explique comment systemd et `/etc/fstab` peuvent interagir."}
	overrides["lpic1.104.3.peripheriques-amovibles"] = [2]string{"Exécute `lsblk -o NAME,FSTYPE,UUID,MOUNTPOINTS`. Repère un UUID et explique pourquoi il est plus stable qu'un nom `/dev/sdX` pour identifier un volume.", "Compare les rôles de `/media` et `/mnt`. Explique où un environnement desktop monte généralement un support amovible et quand un administrateur utiliserait plutôt `/mnt`."}
	overrides["lpic1.104.5.notation-symbolique-octale"] = [2]string{"Crée un fichier temporaire, exécute `chmod u=rw,g=r,o= fichier`, puis vérifie avec `stat -c '%A %a' fichier`. Convertis le résultat en octal.", "Sur un autre fichier, applique `chmod 640 fichier` puis décris l'équivalent symbolique pour user, group et other."}
	overrides["lpic1.104.5.suid"] = [2]string{"Exécute `find /usr/bin -maxdepth 1 -perm -4000 -type f 2>/dev/null | head`. Vérifie un résultat avec `ls -l` et repère le `s` du SUID.", "Sur un fichier de test non sensible, observe comment `chmod u+s` change l'affichage de `ls -l`, puis retire le bit. Explique l'effet du SUID sur un exécutable binaire."}
	overrides["lpic1.104.5.sgid"] = [2]string{"Exécute `find /usr/bin -maxdepth 1 -perm -2000 -type f 2>/dev/null | head`. Repère le bit SGID avec `ls -l`.", "Crée un répertoire de test, applique `chmod g+s`, puis crée un fichier dedans et observe son groupe. Explique l'effet particulier du SGID sur un répertoire."}
	overrides["lpic1.104.5.sticky-bit"] = [2]string{"Exécute `ls -ld /tmp` et repère le `t`. Explique pourquoi le sticky bit est utile sur un répertoire inscriptible par plusieurs utilisateurs.", "Crée un répertoire de test, applique `chmod +t`, puis vérifie son mode avec `stat`. Explique ce que le sticky bit change pour suppression et renommage des entrées."}
	overrides["lpic1.104.5.effets-speciaux-sur-repertoires"] = [2]string{"Compare `ls -ld /tmp` avec un répertoire SGID que tu crées en test. Distingue l'effet du sticky bit de celui du SGID sur un répertoire.", "Crée deux répertoires temporaires, l'un en `1770`, l'autre en `2770`. Vérifie leurs modes et explique pourquoi ces deux configurations répondent à des besoins différents."}
	overrides["lpic1.104.6.limites-filesystem-directory-des-hard-links"] = [2]string{"Crée un fichier et un hard link dans le même répertoire, puis compare `ls -li`. Vérifie qu'ils partagent le même inode.", "Essaie de créer un hard link vers un répertoire ou vers un autre filesystem si le lab en offre un. Explique les limites observées et leur lien avec les inodes."}
	overrides["lpic1.104.6.liens-casses"] = [2]string{"Crée `cible`, puis `ln -s cible lien`; supprime `cible` et vérifie `ls -l lien` puis `test -e lien`. Explique pourquoi le lien symbolique devient cassé.", "Crée un lien symbolique vers un chemin inexistant dès le départ. Compare `-L` et `-e` avec `test` et explique ce que chacun vérifie."}
	overrides["lpic1.104.7.role-des-principaux-repertoires-fhs"] = [2]string{"Exécute `ls -ld /etc /var /usr /home /tmp /boot /opt /srv`. Associe chaque répertoire FHS à son rôle principal.", "Pour chacun de ces besoins — configuration système, logs variables, logiciels additionnels, données servies — choisis le répertoire FHS attendu et vérifie qu'il existe."}
	overrides["lpic1.105.1.login-shell-vs-shell-interactif"] = [2]string{"Compare `bash -lc 'echo login:$0'` et `bash -ic 'echo interactive:$0' 2>/dev/null`. Relie ces modes à `/etc/profile`, `~/.bash_profile` et `~/.bashrc`.", "Ajoute temporairement un marqueur dans un fichier rc de test et lance Bash avec les options adaptées. Vérifie quel fichier est lu par un login shell et par un shell interactif non-login."}
	overrides["lpic1.105.1.fonctions-aliases"] = [2]string{"Crée `alias ll='ls -l'` puis une fonction `cdup(){ cd ..; }`. Utilise `type ll` et `type cdup` pour distinguer alias et fonction.", "Lance un nouveau shell sans exporter ta configuration et vérifie si l'alias et la fonction existent encore. Explique leur portée et où les rendre persistants."}
	overrides["lpic1.105.1.etc-skel"] = [2]string{"Liste `ls -la /etc/skel`. Explique à quel moment son contenu est copié et pourquoi modifier `/etc/skel` ne change pas rétroactivement les homes existants.", "Crée un répertoire home de test en recopiant `/etc/skel` dans un emplacement temporaire. Compare le résultat au squelette d'origine."}
	overrides["lpic1.105.1.configuration-globale-vs-utilisateur"] = [2]string{"Compare les contenus pertinents de `/etc/profile`, `/etc/bash.bashrc` ou `/etc/bashrc`, et `~/.bashrc`. Distingue configuration globale et configuration utilisateur.", "Choisis un alias qui doit s'appliquer à tous les utilisateurs et un autre seulement à toi. Indique dans quel fichier placer chacun et pourquoi."}
	overrides["lpic1.105.2.shebang-et-execution"] = [2]string{"Crée un script contenant `#!/bin/sh` et `printf 'ok\\n'`, rends-le exécutable puis lance-le directement. Explique le rôle du shebang.", "Retire le bit exécutable puis compare `./script` et `sh script`. Explique pourquoi l'interpréteur explicite peut lire le fichier même quand l'exécution directe échoue."}
	overrides["lpic1.106.1.client-serveur-x11"] = [2]string{"Explique, pour une application X11 distante, quel côté est le client X et quel côté est le serveur X. Relie le serveur à l'écran/clavier de l'utilisateur.", "Dans un scénario SSH avec X forwarding, décris le trajet entre application distante, protocole X11 et serveur d'affichage local. Ne confonds pas client réseau SSH et client X11."}
	overrides["lpic1.106.1.display"] = [2]string{"Exécute `printf '%s\\n' \"$DISPLAY\"`. Décompose une valeur typique comme `:0` ou `host:10.0` en hôte, display et screen.", "Imagine deux serveurs X accessibles avec `DISPLAY=:0` et `DISPLAY=:1`. Explique ce que changer la variable modifie pour une application graphique lancée ensuite."}
	overrides["lpic1.106.1.configuration-xorg"] = [2]string{"Inspecte `ls -ld /etc/X11/xorg.conf /etc/X11/xorg.conf.d 2>/dev/null`. Explique la différence entre fichier monolithique et fragments de configuration.", "Imagine un fragment `10-keyboard.conf` sous `xorg.conf.d`. Explique pourquoi les fichiers numérotés permettent de séparer les réglages par fonction."}
	overrides["lpic1.106.1.clavier-affichage"] = [2]string{"Pour un problème de clavier sous X11, indique où chercher un fragment Xorg et quelles informations vérifier avant de modifier la configuration.", "Pour un écran mal détecté, distingue ce qui relève du serveur X11, d'un fichier sous `/etc/X11/xorg.conf.d` et de l'environnement de bureau."}
	overrides["lpic1.106.1.display-manager-window-manager"] = [2]string{"Classe GDM/SDDM, Mutter/KWin et un environnement de bureau dans les catégories display manager, window manager et desktop environment.", "Scénario : l'écran de connexion apparaît mais les fenêtres n'ont plus de décorations après ouverture de session. Identifie la couche la plus probable et explique pourquoi."}
	overrides["lpic1.106.1.concepts-de-base-wayland"] = [2]string{"Compare X11 et Wayland : indique qui joue le rôle central pour l'affichage et pourquoi le modèle client/serveur historique de X11 n'est pas identique.", "Scénario : une application X11 doit fonctionner dans une session Wayland. Explique le rôle d'une couche de compatibilité comme XWayland."}
	overrides["lpic1.106.2.roles-desktop-environment-window-manager-display-manager"] = [2]string{"Associe ces tâches aux bonnes couches : authentifier l'utilisateur graphiquement, placer/décorer les fenêtres, fournir panneaux et applications de session.", "Scénario : changer de window manager sans changer d'écran de connexion. Indique quelles couches restent identiques et laquelle est remplacée."}
	overrides["lpic1.106.2.caracteristiques-generales-kde-gnome-xfce"] = [2]string{"Compare KDE Plasma, GNOME et Xfce sur leur rôle commun d'environnement de bureau, sans les confondre avec un simple window manager.", "Choisis lequel de KDE/GNOME/Xfce tu proposerais dans un contexte de ressources limitées et explique le critère, puis rappelle que LPIC attend surtout leur identification générale."}
	overrides["lpic1.106.2.protocoles-d-acces-graphique-distant"] = [2]string{"Classe VNC, RDP, SPICE, XDMCP et X11 selon qu'ils exposent un bureau distant, une session graphique ou un protocole d'affichage historique.", "Scénario : administrer une VM via console optimisée, partager un bureau existant, puis ouvrir une session Windows distante. Associe SPICE, VNC et RDP aux trois besoins."}
	overrides["lpic1.106.2.differences-vnc-rdp-spice-xdmcp"] = [2]string{"Compare VNC, RDP, SPICE et XDMCP en indiquant pour chacun le type de session/affichage qu'il fournit et son contexte d'usage typique.", "Associe quatre situations — bureau partagé multiplateforme, session Windows, console de VM, ancien login X distant — à VNC, RDP, SPICE et XDMCP."}
	overrides["lpic1.106.3.contraste-themes"] = [2]string{"Scénario : un utilisateur distingue mal deux couleurs d'interface mais lit correctement la taille du texte. Choisis entre thème à fort contraste et grandes polices et justifie.", "Scénario inverse : les couleurs sont lisibles mais le texte est trop petit. Choisis le réglage adapté et explique pourquoi contraste et taille de police résolvent deux problèmes différents."}
	overrides["lpic1.106.3.lecteur-d-ecran"] = [2]string{"Scénario : un utilisateur aveugle doit parcourir menus, champs et texte sans écran visuel. Identifie le rôle d'un screen reader et les informations qu'il restitue.", "Compare lecteur d'écran et magnificateur pour deux utilisateurs ayant des besoins différents. Indique lequel transforme l'interface en sortie vocale/braille."}
	overrides["lpic1.106.3.braille"] = [2]string{"Scénario : un utilisateur lit le contenu textuel via une plage braille rafraîchissable. Explique le rôle du Braille display et sa relation avec les technologies d'assistance.", "Compare une plage braille à un lecteur d'écran vocal. Indique ce qu'ils ont en commun et quelle modalité de sortie les distingue."}
	overrides["lpic1.106.3.magnification"] = [2]string{"Scénario : l'utilisateur voit l'écran mais a besoin d'agrandir une zone sans modifier chaque application. Identifie le rôle d'un screen magnifier.", "Compare magnification et large fonts : donne un cas où le grossissement global/ponctuel est préférable à une simple augmentation de la taille des polices."}
	overrides["lpic1.106.3.clavier-virtuel"] = [2]string{"Scénario : l'utilisateur ne peut pas utiliser un clavier physique mais contrôle un pointeur. Explique comment un on-screen keyboard permet la saisie.", "Compare clavier virtuel et reconnaissance vocale pour saisir du texte. Donne un avantage et une limite de chaque approche."}
	overrides["lpic1.106.3.sticky-slow-bounce-toggle-keys"] = [2]string{"Associe sticky keys, slow keys, bounce keys et toggle keys aux problèmes suivants : combinaisons difficiles, touches accidentelles brèves, doubles frappes involontaires, retour sonore d'état.", "Scénario : un utilisateur ne peut maintenir Ctrl et C ensemble, tandis qu'un autre produit deux lettres à chaque pression. Choisis l'option d'accessibilité adaptée à chacun."}
	overrides["lpic1.106.3.controle-souris-au-clavier"] = [2]string{"Scénario : un utilisateur maîtrise le pavé numérique mais pas une souris. Explique le rôle de mouse keys et ce que le clavier contrôle.", "Compare mouse keys et on-screen keyboard : l'un remplace le dispositif de pointage, l'autre le clavier physique. Donne une action typique pour chacun."}
	overrides["lpic1.106.3.gestes"] = [2]string{"Scénario : une interface tactile permet d'effectuer zoom, défilement et navigation par mouvements. Explique ce que recouvrent les gestures en accessibilité.", "Compare un geste tactile à un raccourci clavier pour la même action. Explique pourquoi offrir plusieurs modalités peut améliorer l'accessibilité."}
	overrides["lpic1.106.3.reconnaissance-vocale"] = [2]string{"Scénario : un utilisateur dicte du texte et déclenche des commandes sans clavier. Explique le rôle de la speech recognition.", "Compare reconnaissance vocale, lecteur d'écran et clavier virtuel : identifie lequel fournit une entrée vocale et lesquels répondent à d'autres besoins."}
	overrides["lpic1.107.1.comptes-systeme-a-usage-limite"] = [2]string{"Lis quelques lignes de `/etc/passwd` et repère les comptes avec shell `nologin` ou `false`. Explique pourquoi un compte de service n'a généralement pas besoin d'un login interactif.", "Compare un compte utilisateur humain et un compte système à usage limité à partir de `/etc/passwd` : UID, home et shell. Explique les différences attendues."}
	overrides["lpic1.107.1.squelettes"] = [2]string{"Liste `/etc/skel` puis compare avec le contenu initial d'un home utilisateur récent si disponible. Explique quand le squelette est copié.", "Ajoute un fichier dans une copie temporaire de `/etc/skel` et simule la création d'un home. Explique pourquoi les utilisateurs existants ne sont pas modifiés."}
	overrides["lpic1.107.2.allow-deny"] = [2]string{"Repère `at.allow`, `at.deny`, `cron.allow` et `cron.deny` sous `/etc` s'ils existent. Explique comment la présence de ces fichiers contrôle l'accès aux planificateurs.", "Pour cron puis at, raisonne sur trois cas : fichier allow présent, seulement deny présent, aucun des deux. Explique qui est autorisé dans chaque cas."}
	overrides["lpic1.107.2.choix-cron-vs-timer"] = [2]string{"Compare une tâche récurrente exprimée en crontab avec un timer systemd. Identifie ce que le timer ajoute en termes d'unités, dépendances et journalisation.", "Scénario : lancer un script toutes les nuits sur un système systemd et pouvoir consulter son état avec `systemctl`. Choisis cron ou timer et justifie."}
	overrides["lpic1.107.3.lang-lc-all-priorite"] = [2]string{"Exécute `LANG=C LC_ALL=fr_FR.UTF-8 locale 2>/dev/null | head`. Explique pourquoi `LC_ALL` prend le dessus sur `LANG` et les variables `LC_*`.", "Définis `LANG=C`, puis surcharge uniquement `LC_TIME` si une locale adaptée existe. Explique la priorité entre catégorie `LC_*`, `LANG` et `LC_ALL`."}
	overrides["lpic1.107.3.utf-8-unicode-ascii-iso-8859"] = [2]string{"Crée un fichier UTF-8 contenant `é`, puis exécute `file fichier` et `od -An -tx1 fichier`. Relie Unicode, encodage UTF-8 et octets observés.", "Compare ASCII, ISO-8859 et UTF-8 : indique lesquels sont des jeux/encodages historiques, lesquels sont compatibles sur l'ASCII de base et comment UTF-8 représente Unicode."}
	overrides["lpic1.107.3.lang-c-et-scripts-reproductibles"] = [2]string{"Compare `printf '%s\\n' b A a | sort` avec `printf '%s\\n' b A a | LANG=C sort`. Observe l'ordre et explique pourquoi `LANG=C` rend le comportement de tri plus prévisible.", "Lance une commande sensible à la locale avec `LANG=C` uniquement pour ce processus. Explique pourquoi cette technique est utile dans un script qui doit produire une sortie reproductible."}
	overrides["lpic1.108.1.sources-pools"] = [2]string{"Examine `chronyc sources 2>/dev/null || ntpq -p 2>/dev/null || true`. Explique la différence entre une source NTP individuelle et un pool comme `pool.ntp.org`.", "Dans une configuration de temps, compare l'usage d'un hostname de pool et d'un serveur unique. Explique l'intérêt de disposer de plusieurs sources."}
	overrides["lpic1.108.1.configuration-de-base"] = [2]string{"Inspecte `/etc/chrony.conf` ou `/etc/ntp.conf` selon ce qui existe. Repère une directive de source de temps et explique son effet.", "Ajoute mentalement une nouvelle source sans modifier le système : indique dans quel fichier elle irait pour chrony puis pour ntpd, et quelle commande vérifierait ensuite l'état."}
	overrides["lpic1.108.2.syslog-facilities-priorities-actions"] = [2]string{"Inspecte `/etc/rsyslog.conf` et `/etc/rsyslog.d` s'ils existent. Repère une règle `facility.priority action` et décompose ses trois parties.", "Prends la règle conceptuelle `authpriv.* /var/log/secure`. Explique facility, priority et action puis propose ce qui changerait pour ne garder que les messages de priorité error et plus sévères."}
	overrides["lpic1.108.2.rsyslog-configuration"] = [2]string{"Lis les lignes non commentées de `/etc/rsyslog.conf` et `/etc/rsyslog.d/*.conf` s'ils existent. Repère au moins une destination de logs.", "Imagine que les messages `local0.info` doivent aller dans `/var/log/app.log`. Écris la règle rsyslog correspondante et indique comment vérifier la configuration avant redémarrage."}
	overrides["lpic1.108.2.connaissance-syslog-ng"] = [2]string{"Vérifie `command -v syslog-ng || true`. Même s'il n'est pas installé, explique son rôle comme implémentation de syslog distincte de rsyslog.", "Compare rsyslog et syslog-ng au niveau attendu LPIC : même objectif de collecte/routage de logs, syntaxe et implémentation différentes. Cite un fichier de configuration typique de chacun."}
	overrides["lpic1.108.2.journalisation-distante-rsyslog"] = [2]string{"Inspecte les directives contenant `@` ou `omfwd` dans la configuration rsyslog si elles existent. Explique comment une action distante diffère d'une écriture locale.", "Scénario : envoyer les logs vers `log.example:514` puis recevoir des messages distants. Décris séparément la configuration émetteur et récepteur, et distingue UDP `@` de TCP `@@`."}
	overrides["lpic1.108.3.role-mta"] = [2]string{"Vérifie `command -v sendmail postconf exim 2>/dev/null || true`. Explique le rôle d'un MTA dans l'acheminement du courrier entre systèmes.", "Classe MTA, MUA et MDA : lequel transporte entre serveurs, lequel est l'interface utilisateur, lequel dépose localement le message."}
	overrides["lpic1.108.3.forward-utilisateur"] = [2]string{"Vérifie `ls -l ~/.forward 2>/dev/null || true`. Explique comment ce fichier peut rediriger le courrier d'un utilisateur vers une autre destination.", "Écris dans un fichier temporaire le contenu qu'aurait `~/.forward` pour transférer vers `user@example.net`. Explique à quel niveau ce mécanisme agit."}
	overrides["lpic1.108.4.architecture-client-serveur-cups"] = [2]string{"Vérifie `systemctl status cups --no-pager 2>/dev/null || true` et `lpstat -r 2>/dev/null || true`. Explique le rôle du serveur CUPS et celui d'un client d'impression.", "Scénario : plusieurs postes envoient des travaux vers une imprimante gérée centralement. Identifie le serveur CUPS, les clients et la file d'impression."}
	overrides["lpic1.108.4.configuration-de-base"] = [2]string{"Inspecte `ls -la /etc/cups 2>/dev/null` puis `grep -v '^#' /etc/cups/cupsd.conf 2>/dev/null | head`. Repère le fichier principal de configuration de CUPS.", "Imagine l'ajout d'une imprimante et une modification de l'accès au serveur CUPS. Indique quels outils/fichiers tu consulterais et comment vérifier l'état des files."}
	overrides["lpic1.109.1.ipv4-et-ipv6"] = [2]string{"Exécute `ip -4 addr` puis `ip -6 addr`. Repère une adresse de chaque famille et compare leur notation.", "Compare `ip -4 route` et `ip -6 route`. Identifie la route par défaut ou la boucle locale et explique ce qui est propre à chaque famille."}
	overrides["lpic1.109.1.cidr-masques-sous-reseaux"] = [2]string{"Prends `192.168.10.34/24`. Calcule l'adresse réseau, l'adresse de broadcast et la plage d'hôtes, puis vérifie ton raisonnement avec les bits du préfixe.", "Compare `/24` et `/26` pour un même réseau IPv4. Indique combien d'adresses contient chaque sous-réseau et comment le masque change."}
	overrides["lpic1.109.1.adresses-privees-publiques"] = [2]string{"Exécute `ip -4 addr`. Pour chaque IPv4, indique si elle est privée, loopback, link-local ou potentiellement publique.", "Classe `10.0.0.1`, `172.20.1.1`, `192.168.1.1`, `169.254.1.1` et `8.8.8.8`. Explique quelles plages RFC1918 sont privées."}
	overrides["lpic1.109.1.tcp-vs-udp-vs-icmp"] = [2]string{"Exécute `ss -lnt` puis `ss -lnu`. Compare les sockets TCP et UDP, puis explique pourquoi ICMP n'apparaît pas comme un port d'écoute équivalent.", "Associe connexion fiable, datagrammes sans connexion et messages de contrôle réseau à TCP, UDP et ICMP. Donne un exemple d'usage typique pour chacun."}
	overrides["lpic1.109.1.ports-et-services-courants"] = [2]string{"Utilise `grep -E '(^ssh|^domain|^http|^https)[[:space:]]' /etc/services`. Associe 22, 53, 80 et 443 à leurs services et protocoles.", "Choisis cinq autres ports de la liste LPIC dans `/etc/services`, retrouve leur nom de service et indique s'ils utilisent TCP, UDP ou les deux."}
	overrides["lpic1.109.1.boucle-locale-et-routes-conceptuelles"] = [2]string{"Exécute `ip addr show lo` puis `ip route`. Repère la boucle locale et une route réseau, et explique pourquoi `lo` ne sort pas de la machine.", "Compare `127.0.0.1` et `::1`, puis identifie la route par défaut IPv4/IPv6 si elle existe. Explique la différence entre adresse locale et décision de routage."}
	overrides["lpic1.109.1.lecture-etc-services"] = [2]string{"Lis `grep -E '(^ssh|^domain|^http|^https)[[:space:]]' /etc/services`. Décompose une ligne en nom, port/protocole et alias.", "Recherche un port avec `grep -w '53/tcp' /etc/services` puis par nom avec `grep '^domain' /etc/services`. Explique ce que ce fichier fournit et ce qu'il ne prouve pas sur les services réellement actifs."}
	overrides["lpic1.109.1.caracteristiques-ipv6"] = [2]string{"Exécute `ip -6 addr`. Repère `::1`, une éventuelle adresse link-local `fe80::/10` et explique la notation abrégée avec `::`.", "Compare une adresse IPv4 `/24` à une IPv6 `/64`. Explique l'absence de broadcast IPv6 classique et le rôle important du link-local."}
	overrides["lpic1.109.2.hosts-nss-resolv-conf"] = [2]string{"Inspecte `/etc/hosts`, la ligne `hosts:` de `/etc/nsswitch.conf` et `/etc/resolv.conf`. Explique l'ordre entre sources locales/NSS et résolveurs DNS.", "Ajoute mentalement un nom local puis un DNS : indique quel fichier contrôle la correspondance statique, lequel l'ordre de résolution et lequel les serveurs DNS."}
	overrides["lpic1.109.2.connaissance-systemd-networkd"] = [2]string{"Exécute `systemctl status systemd-networkd --no-pager 2>/dev/null || true` et liste `/etc/systemd/network 2>/dev/null`. Explique le rôle de systemd-networkd.", "Compare systemd-networkd et NetworkManager : deux gestionnaires possibles de la configuration réseau, avec des usages et fichiers différents. Identifie lequel est actif dans le lab."}
	overrides["lpic1.109.2.persistance-et-ordre-resolution"] = [2]string{"Lis la ligne `hosts:` de `/etc/nsswitch.conf` puis vérifie `systemctl is-active NetworkManager 2>/dev/null`. Distingue ordre de résolution de noms et persistance de configuration réseau.", "Scénario : une IP statique disparaît au reboot alors que `/etc/hosts` fonctionne. Explique pourquoi NetworkManager et `nsswitch.conf` ne règlent pas le même problème."}
	overrides["lpic1.109.4.resolver-configuration"] = [2]string{"Lis `/etc/resolv.conf`. Identifie `nameserver`, `search` et `options` s'ils existent et explique leur rôle.", "Compare une résolution avec `getent hosts localhost` puis un nom DNS disponible dans le lab. Explique comment `/etc/resolv.conf` intervient lorsque NSS choisit DNS."}
	overrides["lpic1.109.4.systemd-resolved-awareness"] = [2]string{"Exécute `systemctl status systemd-resolved --no-pager 2>/dev/null || true` et `readlink -f /etc/resolv.conf`. Détermine si resolved gère le résolveur sur cette machine.", "Compare un `/etc/resolv.conf` statique à un lien géré par systemd-resolved. Explique pourquoi éditer directement le fichier généré peut ne pas être persistant."}
	overrides["lpic1.110.2.shadow-passwords"] = [2]string{"Compare les champs utilisateur de `/etc/passwd` et `/etc/shadow` avec `getent passwd root` et, si autorisé, `getent shadow root`. Explique pourquoi les hashes ne sont pas dans passwd.", "Observe les permissions de `/etc/passwd` et `/etc/shadow` avec `ls -l`. Explique pourquoi le premier est lisible largement et le second beaucoup plus restreint."}
	overrides["lpic1.110.2.login-interdit-via-nologin"] = [2]string{"Vérifie `ls -l /etc/nologin 2>/dev/null || true`. Explique l'effet attendu de ce fichier sur les connexions d'utilisateurs ordinaires.", "Compare `/etc/nologin` à un shell `/usr/sbin/nologin` dans `/etc/passwd`. Explique pourquoi ces deux mécanismes ne bloquent pas exactement la même chose."}
	overrides["lpic1.110.2.socket-activation-systemd"] = [2]string{"Exécute `systemctl list-units --type=socket --all | head`. Choisis une unité `.socket` et retrouve le service associé si possible.", "Explique le scénario d'activation à la demande : systemd écoute sur un socket, puis démarre le service lors d'une connexion. Compare avec un démon toujours actif."}
	overrides["lpic1.110.2.xinetd-historique"] = [2]string{"Vérifie `ls -ld /etc/xinetd.conf /etc/xinetd.d 2>/dev/null`. Explique le rôle historique de xinetd comme super-serveur.", "Compare xinetd à la socket activation systemd : tous deux peuvent démarrer un service à la demande, mais leur configuration et leur intégration diffèrent."}
	overrides["lpic1.110.2.tcp-wrappers-historique"] = [2]string{"Vérifie `ls -l /etc/hosts.allow /etc/hosts.deny 2>/dev/null`. Explique le rôle historique des TCP wrappers et pourquoi leur prise en charge dépend du service.", "Écris deux règles conceptuelles hosts.allow/hosts.deny pour autoriser un réseau et refuser le reste. Explique l'ordre logique sans supposer que tous les démons modernes utilisent libwrap."}
	overrides["lpic1.110.3.known-hosts-authorized-keys"] = [2]string{"Compare `~/.ssh/known_hosts` et `~/.ssh/authorized_keys` s'ils existent. Explique que l'un mémorise l'identité des serveurs et l'autre autorise des clés clientes.", "Scénario : une alerte de host key apparaît après changement de serveur, puis un utilisateur veut ajouter une nouvelle clé de connexion. Indique quel fichier concerne chaque action."}
	overrides["lpic1.101.2.sequence-firmware-bootloader-kernel-initramfs-init-systemd"] = [2]string{"Observe `test -d /sys/firmware/efi && echo UEFI || echo BIOS`, `cat /proc/cmdline`, `ls /boot | grep -E 'initramfs|initrd'` et `ps -p 1 -o comm=`. Remets les étapes firmware → bootloader → kernel → initramfs → init/systemd dans l'ordre.", "Pour chaque étape firmware, bootloader, kernel, initramfs puis init/systemd, cite un symptôme ou une commande d'observation qui permettrait de localiser un échec de démarrage."}
	overrides["lpic1.101.3.arret-redemarrage-propre"] = [2]string{"Sans éteindre le lab, consulte `shutdown --help | head -n 30` et `systemctl --help | grep -E 'poweroff|reboot'`. Distingue arrêt, redémarrage, planification et annulation.", "Écris les commandes que tu utiliserais pour programmer un redémarrage dans 10 minutes puis l'annuler avant exécution. Explique pourquoi un arrêt propre passe par le système d'init plutôt que par une coupure brutale."}
	overrides["lpic1.102.2.emplacement-configuration-du-bootloader"] = [2]string{"Inspecte `ls -ld /etc/default/grub /etc/grub.d /boot/grub* 2>/dev/null`. Distingue les sources de configuration de GRUB 2 du fichier `grub.cfg` généré.", "Retrouve le `grub.cfg` présent sur la machine avec `find /boot -name grub.cfg -print 2>/dev/null`. Explique quels fichiers tu modifierais pour un changement persistant et lequel tu régénérerais ensuite."}
	overrides["lpic1.102.4.reconfiguration"] = [2]string{"Vérifie qu'un paquet configurable comme `tzdata` est connu avec `dpkg-query -W tzdata 2>/dev/null`. Explique ce que ferait `dpkg-reconfigure tzdata` sans lancer l'interface interactive.", "Choisis un paquet Debian installé et distingue réinstallation, reconfiguration avec `dpkg-reconfigure` et modification manuelle d'un fichier de configuration."}
	overrides["lpic1.103.5.multiplexeurs-de-terminal"] = [2]string{"Vérifie `tmux -V` et `screen --version 2>/dev/null | head -n 1`. Explique ce qu'un multiplexeur conserve quand le terminal client disparaît.", "Crée une session détachée avec `tmux new-session -d -s lpicdemo 'sleep 300'`, vérifie-la avec `tmux ls`, puis supprime-la avec `tmux kill-session -t lpicdemo`. Relie détachement et rattachement à la survie de la session."}
	overrides["lpic1.103.8.connaissance-nano-emacs-vim"] = [2]string{"Exécute `command -v nano emacs vim`. Pour les éditeurs présents, affiche leur version sans ouvrir d'interface et distingue nano, Emacs et Vim comme éditeurs texte interactifs.", "Choisis entre nano, Emacs et Vim pour une modification rapide en terminal, puis explique quel éditeur est modal et lequel privilégie une interface de raccourcis plus directe."}
	overrides["lpic1.103.8.modes-normal-insertion-commande"] = [2]string{"Ouvre un fichier temporaire dans `vi` ou `vim`. Passe du mode normal au mode insertion avec `i`, reviens avec `Esc`, puis ouvre la ligne de commande avec `:` avant de quitter.", "Dans `vi`, réalise successivement une navigation en mode normal, une insertion de texte et une commande `:wq` ou `:q!`. Explique comment reconnaître les trois modes et comment passer de l'un à l'autre."}
	return overrides
}()

func standaloneLearnerTask(concept curriculum.Concept, variant int) string {
	if tasks, exists := standalonePracticeOverrides[concept.ID]; exists {
		return tasks[variant%len(tasks)]
	}
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

	return fmt.Sprintf(
		"Exercice spécifique indisponible pour %s. Utilise les repères %s et signale ce contenu à LPIC Daily.",
		concept.ID,
		standaloneAnchorList(concept),
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
