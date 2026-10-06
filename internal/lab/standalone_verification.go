package lab

import (
	"fmt"
	"strings"

	"github.com/Loe159/lpic-daily/internal/curriculum"
)

func deterministicStandaloneExercise(
	objective curriculum.Objective,
	concept curriculum.Concept,
	root string,
	variant int,
) (string, []CheckDefinition, string) {
	result := root + "/" + standaloneConceptFilename(concept) + ".result"
	source := root + "/" + standaloneConceptFilename(concept) + ".source"

	if objective.ID == "101.3" {
		switch concept.PedagogyOrder {
		case 3, 5, 8:
			return deterministicSystemControlExercise(concept, root, variant)
		}
	}
	if objective.ID == "102.2" && concept.PedagogyOrder == 5 {
		return deterministicGRUBGenerationExercise(concept, root, variant)
	}

	switch objective.ID {
	case "102.1", "104.1", "104.2", "104.3":
		return deterministicStorageExercise(objective.ID, concept, root, result, variant)
	case "103.1":
		return deterministicShellExercise(concept, root, result, variant)
	case "103.2":
		return deterministicTextExercise(concept, source, result, variant)
	case "103.3":
		return deterministicFileExercise(concept, root, source, result, variant)
	case "103.4":
		return deterministicRedirectionExercise(concept, root, result, variant)
	case "103.5":
		return deterministicProcessExercise(concept, root, result, variant)
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
	case "104.7":
		return deterministicFHSExercise(concept, root, result, variant)
	default:
		diagnostic, transfer := deterministicInspectionProbes(objective.ID, concept)
		if variant == 0 {
			return deterministicSnapshotTask(concept, result, diagnostic, false)
		}
		return deterministicSnapshotTask(concept, result, transfer, true)
	}
}

func deterministicCommandCheck(conceptID, script string) CheckDefinition {
	return CheckDefinition{
		Type:         "command-exit",
		Argv:         []string{"/bin/sh", "-c", script},
		ExpectedExit: 0,
		ConceptIDs:   []string{conceptID},
	}
}

func deterministicSnapshotTask(
	concept curriculum.Concept,
	result string,
	probe string,
	digest bool,
) (string, []CheckDefinition, string) {
	if !digest {
		script := fmt.Sprintf(
			"set -eu\nactual=$(mktemp)\ntrap 'rm -f \"$actual\"' EXIT\n( %s ) >\"$actual\"\ntest -f %q\ncmp -s %q \"$actual\"",
			probe,
			result,
			result,
		)
		return fmt.Sprintf(
			"Diagnostic : inspecte l'état réel lié à %s et reproduis exactement le relevé canonique dans %s. Le checker recalcule ce relevé depuis le système.",
			concept.TitleFR,
			result,
		), []CheckDefinition{deterministicCommandCheck(concept.ID, script)}, ""
	}

	script := fmt.Sprintf(
		"set -eu\nexpected=$( ( %s ) | sha256sum | awk '{print $1}')\ntest -f %q\ntest \"$(tr -d '[:space:]' < %q)\" = \"$expected\"",
		probe,
		result,
		result,
	)
	return fmt.Sprintf(
		"Transfert : utilise un second angle d'inspection pour %s et écris uniquement dans %s le SHA-256 du relevé obtenu. Le checker dérive indépendamment la même empreinte.",
		concept.TitleFR,
		result,
	), []CheckDefinition{deterministicCommandCheck(concept.ID, script)}, ""
}

func deterministicInspectionProbes(objectiveID string, concept curriculum.Concept) (string, string) {
	if diagnostic, transfer, ok := specificInspectionProbes(objectiveID, concept.PedagogyOrder); ok {
		return diagnostic, transfer
	}
	var diagnostic []string
	var transfer []string
	for _, anchor := range concept.AnchorTerms {
		anchor = strings.TrimSpace(anchor)
		if strings.HasPrefix(anchor, "/") {
			diagnostic = append(diagnostic, fmt.Sprintf("if test -e %q; then stat -Lc 'path|%%n|%%F|%%a' %q; else printf 'missing|%%s\\n' %q; fi", anchor, anchor, anchor))
			transfer = append(transfer, fmt.Sprintf("if test -e %q; then printf 'resolved|%%s\\n' \"$(readlink -f %q)\"; else printf 'missing|%%s\\n' %q; fi", anchor, anchor, anchor))
			continue
		}
		if standaloneCommandAnchorAllowed(anchor) && standaloneDirectCommandAnchor(anchor) {
			diagnostic = append(diagnostic, fmt.Sprintf("printf 'command|%%s|' %q; command -v %s 2>/dev/null || true", anchor, anchor))
			transfer = append(transfer, fmt.Sprintf("printf 'type|%%s|' %q; command -V %s 2>/dev/null || true", anchor, anchor))
		}
	}

	switch objectiveID {
	case "101.1":
		diagnostic = append(diagnostic, "lsblk -ndo NAME,TYPE,TRAN 2>/dev/null | sort", "find /sys/class -mindepth 1 -maxdepth 1 -printf '%f\\n' 2>/dev/null | sort | head -20")
		transfer = append(transfer, "{ lspci -nn 2>/dev/null || true; lsusb 2>/dev/null || true; } | head -24", "lsmod 2>/dev/null | sed -n '1,12p'")
	case "101.2":
		diagnostic = append(diagnostic, "printf 'pid1|'; cat /proc/1/comm", "printf 'cmdline|'; cat /proc/cmdline")
		transfer = append(transfer, "if test -d /sys/firmware/efi; then echo firmware=UEFI; else echo firmware=BIOS; fi", "{ journalctl -b -n 12 --no-pager 2>/dev/null || dmesg 2>/dev/null | tail -12 || true; }")
	case "101.3":
		diagnostic = append(diagnostic, "systemctl get-default 2>/dev/null || true", "systemctl list-jobs --no-pager 2>/dev/null | sed -n '1,12p'")
		transfer = append(transfer, "systemctl list-dependencies rescue.target --plain --no-pager 2>/dev/null | sed -n '1,16p'", "who 2>/dev/null || true")
	case "102.2":
		diagnostic = append(diagnostic, "for p in /boot/grub2/grub.cfg /boot/grub/grub.cfg /boot/grub/menu.lst; do test -e \"$p\" && printf 'bootcfg|%s\\n' \"$p\"; done")
		transfer = append(transfer, "for p in /boot/grub2/grub.cfg /boot/grub/grub.cfg; do test -f \"$p\" && grep -E '^[[:space:]]*(menuentry|linux|linuxefi|initrd|initrdefi)' \"$p\" | head -16; done")
	case "102.3":
		diagnostic = append(diagnostic, "ldd /bin/sh 2>/dev/null | sed -n '1,12p'", "ldconfig -p 2>/dev/null | sed -n '1,12p'")
		transfer = append(transfer, "{ grep -hv '^[[:space:]]*#' /etc/ld.so.conf /etc/ld.so.conf.d/*.conf 2>/dev/null || true; } | sed '/^[[:space:]]*$/d' | sort", "printenv LD_LIBRARY_PATH 2>/dev/null || true")
	case "102.4":
		diagnostic = append(diagnostic, "dpkg-query -W bash 2>/dev/null || true", "apt-cache depends bash 2>/dev/null | sed -n '1,12p'")
		transfer = append(transfer, "dpkg -L bash 2>/dev/null | sed -n '1,12p'", "grep -RhEv '^[[:space:]]*(#|$)' /etc/apt/sources.list /etc/apt/sources.list.d 2>/dev/null | head -12 || true")
	case "102.5":
		diagnostic = append(diagnostic, "rpm -qa 2>/dev/null | sort | head -12", "rpm --eval '%{_dbpath}' 2>/dev/null || true")
		transfer = append(transfer, "{ dnf repolist 2>/dev/null || yum repolist 2>/dev/null || zypper lr 2>/dev/null || true; } | sed -n '1,14p'", "rpm -qf /bin/sh 2>/dev/null || rpm -qf \"$(readlink -f /bin/sh)\" 2>/dev/null || true")
	case "102.6":
		diagnostic = append(diagnostic, "systemd-detect-virt 2>/dev/null || true", "printf 'machine-id|'; cat /etc/machine-id 2>/dev/null")
		transfer = append(transfer, "printf 'hostname|'; hostname", "lsmod 2>/dev/null | grep -E 'virtio|vmw|xen|hv_' | sort || true")
	case "103.1":
		diagnostic = append(diagnostic, "printenv SHELL 2>/dev/null || true", "printenv PATH")
		transfer = append(transfer, "set | grep -E '^(PATH|SHELL|HISTFILE)=' | sort", "sh -c 'printenv PATH'")
	case "104.7":
		diagnostic = append(diagnostic, "for p in /etc /var /usr /home /tmp /boot /opt /srv; do test -e \"$p\" && stat -Lc '%n|%F' \"$p\"; done", "whereis sh")
		transfer = append(transfer, "find /etc -maxdepth 1 -type f -printf '%f\\n' 2>/dev/null | sort | head -12", "printf 'type|'; type -P sh; printf 'which|'; which sh")
	default:
		diagnostic = append(diagnostic, "uname -srm", "pwd")
		transfer = append(transfer, "id -u", "uname -r")
	}
	return strings.Join(diagnostic, "; "), strings.Join(transfer, "; ")
}

func deterministicShellExercise(
	concept curriculum.Concept,
	root, result string,
	variant int,
) (string, []CheckDefinition, string) {
	scriptPath := root + "/shell-task.sh"
	token := "alpha"
	if variant == 1 {
		token = "omega"
	}
	check := func(script string) []CheckDefinition {
		return []CheckDefinition{deterministicCommandCheck(concept.ID, script)}
	}

	switch concept.PedagogyOrder {
	case 1:
		expected := token + "-A\n" + token + "-B\n" + token + "-C\n"
		verify := fmt.Sprintf(
			"set -eu; grep -Fq '&&' %q; grep -Fq '||' %q; actual=$(mktemp); expected=$(mktemp); trap 'rm -f \"$actual\" \"$expected\"' EXIT; /bin/bash %q >\"$actual\"; printf '%%b' %q >\"$expected\"; cmp -s \"$actual\" \"$expected\"",
			scriptPath, scriptPath, scriptPath, expected,
		)
		return fmt.Sprintf(
			"Crée %s : le script doit utiliser une séquence avec && et || et produire exactement trois lignes %s-A, %s-B, %s-C.",
			scriptPath, token, token, token,
		), check(verify), ""

	case 2:
		expected := fmt.Sprintf("local=%s-local\nexported=%s-exported\nchild=%s-exported\n", token, token, token)
		verify := fmt.Sprintf(
			"set -eu; grep -Fq 'export ' %q; actual=$(mktemp); expected=$(mktemp); trap 'rm -f \"$actual\" \"$expected\"' EXIT; /bin/bash %q >\"$actual\"; printf '%%b' %q >\"$expected\"; cmp -s \"$actual\" \"$expected\"",
			scriptPath, scriptPath, expected,
		)
		return fmt.Sprintf(
			"Crée %s : définis une variable shell %s-local, exporte une variable %s-exported, puis montre sa valeur dans un shell enfant.",
			scriptPath, token, token,
		), check(verify), ""

	case 3:
		toolsDir := root + "/tools-" + token
		tool := toolsDir + "/lpic-path-tool"
		setup := fmt.Sprintf("install -d -m 0777 %q\nprintf '#!/bin/sh\\nprintf %s\\\\n\\n' > %q\nchmod 0755 %q\n", toolsDir, token, tool, tool)
		verify := fmt.Sprintf(
			"set -eu; grep -Fq 'PATH=' %q; actual=$(/bin/bash %q); test \"$actual\" = %q",
			scriptPath, scriptPath, tool,
		)
		return fmt.Sprintf(
			"Un exécutable est préparé dans %s. Crée %s pour modifier PATH, résoudre lpic-path-tool sans chemin absolu et afficher uniquement son chemin résolu.",
			toolsDir, scriptPath,
		), check(verify), setup

	case 4:
		value := token + " value * literal"
		expected := fmt.Sprintf("single=$VALUE\ndouble=%s\nescaped=$VALUE\n", value)
		verify := fmt.Sprintf(
			"set -eu; grep -Fq 'single=$VALUE' %q; grep -Fq 'double=' %q; grep -Fq '\\$VALUE' %q; actual=$(mktemp); expected=$(mktemp); trap 'rm -f \"$actual\" \"$expected\"' EXIT; VALUE=%q /bin/bash %q >\"$actual\"; printf '%%b' %q >\"$expected\"; cmp -s \"$actual\" \"$expected\"",
			scriptPath, scriptPath, scriptPath, value, scriptPath, expected,
		)
		return fmt.Sprintf(
			"Crée %s pour démontrer quoting simple, double et échappement avec une valeur contenant espaces et '*'. Le résultat doit distinguer littéral et expansion.",
			scriptPath,
		), check(verify), ""

	case 5:
		historyFile := root + "/history-" + token
		marker := "lpic-history-" + token
		verify := fmt.Sprintf(
			"set -eu; rm -f %q; /bin/bash %q >/dev/null; grep -Fq %q %q",
			historyFile, scriptPath, marker, historyFile,
		)
		return fmt.Sprintf(
			"Crée %s : active l'historique Bash, utilise HISTFILE=%s, ajoute l'entrée %s avec history et persiste-la.",
			scriptPath, historyFile, marker,
		), check(verify), ""

	case 6:
		probe := "printf"
		if variant == 1 {
			probe = "sh"
		}
		canonical := fmt.Sprintf(
			"{ type -a %s 2>/dev/null || true; which %s 2>/dev/null || true; man -w %s 2>/dev/null || true; }",
			probe, probe, probe,
		)
		return deterministicSnapshotTask(concept, result, canonical, variant == 1)

	case 7:
		probe := "uname -srm; pwd"
		if variant == 1 {
			probe = "uname -a; printf 'cwd|'; pwd"
		}
		return deterministicSnapshotTask(concept, result, probe, variant == 1)

	case 8:
		expected := fmt.Sprintf("set-shell=%s-shell\nset-export=%s-env\nenv-export=%s-env\nenv-shell=absent\n", token, token, token)
		verify := fmt.Sprintf(
			"set -eu; grep -Eq '(^|[;&[:space:]])set([;&[:space:]]|$)' %q; grep -Eq '(^|[;&[:space:]])env([;&[:space:]]|$)' %q; actual=$(mktemp); expected=$(mktemp); trap 'rm -f \"$actual\" \"$expected\"' EXIT; /bin/bash %q >\"$actual\"; printf '%%b' %q >\"$expected\"; cmp -s \"$actual\" \"$expected\"",
			scriptPath, scriptPath, scriptPath, expected,
		)
		return fmt.Sprintf(
			"Crée %s pour définir une variable non exportée et une variable exportée, puis prouver avec set et env que seul l'environnement reçoit la seconde.",
			scriptPath,
		), check(verify), ""

	case 9:
		expected := fmt.Sprintf("before=%s\nchild-before=%s\nchild-after=absent\n", token, token)
		verify := fmt.Sprintf(
			"set -eu; grep -Eq '(^|[;&[:space:]])export([;&[:space:]]|$)' %q; grep -Eq '(^|[;&[:space:]])unset([;&[:space:]]|$)' %q; actual=$(mktemp); expected=$(mktemp); trap 'rm -f \"$actual\" \"$expected\"' EXIT; /bin/bash %q >\"$actual\"; printf '%%b' %q >\"$expected\"; cmp -s \"$actual\" \"$expected\"",
			scriptPath, scriptPath, scriptPath, expected,
		)
		return fmt.Sprintf(
			"Crée %s pour exporter LPIC_CHILD=%s, l'observer dans un shell enfant, faire unset, puis prouver qu'un nouvel enfant ne la reçoit plus.",
			scriptPath, token,
		), check(verify), ""

	case 10:
		outsideDir := root + "/outside-" + token
		tool := outsideDir + "/lpic-outside"
		setup := fmt.Sprintf("install -d -m 0777 %q\nprintf '#!/bin/sh\\nprintf %s\\\\n\\n' > %q\nchmod 0755 %q\n", outsideDir, token, tool, tool)
		verify := fmt.Sprintf(
			"set -eu; grep -Fq 'PATH=/usr/bin:/bin' %q; grep -Fq %q %q; test \"$(PATH=/usr/bin:/bin /bin/bash %q)\" = %q",
			scriptPath, tool, scriptPath, scriptPath, token,
		)
		return fmt.Sprintf(
			"%s n'est pas dans PATH. Crée %s qui fixe PATH=/usr/bin:/bin puis exécute explicitement cet outil hors PATH et affiche son résultat.",
			tool, scriptPath,
		), check(verify), setup

	case 11:
		historyFile := root + "/.bash_history-" + token
		marker := "persist-" + token
		verify := fmt.Sprintf(
			"set -eu; rm -f %q; /bin/bash %q >/dev/null; grep -Fq %q %q",
			historyFile, scriptPath, marker, historyFile,
		)
		return fmt.Sprintf(
			"Crée %s pour utiliser HISTFILE=%s, ajouter %s à l'historique et le persister avec history -w.",
			scriptPath, historyFile, marker,
		), check(verify), ""

	default:
		expected := fmt.Sprintf("%s\n42\n%s-child\n", token, token)
		verify := fmt.Sprintf(
			"set -eu; grep -Fq '$((40+2))' %q; grep -Fq '$(' %q; grep -Eq '(^|[;&[:space:]])echo([;&[:space:]]|$)' %q; actual=$(mktemp); expected=$(mktemp); trap 'rm -f \"$actual\" \"$expected\"' EXIT; /bin/bash %q >\"$actual\"; printf '%%b' %q >\"$expected\"; cmp -s \"$actual\" \"$expected\"",
			scriptPath, scriptPath, scriptPath, scriptPath, expected,
		)
		return fmt.Sprintf(
			"Crée %s pour utiliser echo avec expansion de variable, expansion arithmétique 40+2 et substitution de commande produisant %s-child.",
			scriptPath, token,
		), check(verify), ""
	}
}

func deterministicFHSExercise(
	concept curriculum.Concept,
	root, result string,
	variant int,
) (string, []CheckDefinition, string) {
	check := func(script string) []CheckDefinition {
		return []CheckDefinition{deterministicCommandCheck(concept.ID, script)}
	}
	token := "alpha"
	if variant == 1 {
		token = "omega"
	}

	switch concept.PedagogyOrder {
	case 1:
		probe := "for p in /etc /var /usr /home /tmp /boot /opt /srv; do test -e \"$p\" && stat -Lc '%n|%F' \"$p\"; done"
		if variant == 1 {
			probe = "for p in /etc /var /usr /home /tmp /boot /opt /srv; do test -e \"$p\" && printf '%s|' \"$p\" && findmnt -T \"$p\" -n -o FSTYPE 2>/dev/null || true; done"
		}
		return deterministicSnapshotTask(concept, result, probe, variant == 1)

	case 2:
		tree := root + "/search-" + token
		db := root + "/locate-" + token + ".db"
		findResult := root + "/find-" + token + ".result"
		locateResult := root + "/locate-" + token + ".result"
		target := tree + "/nested/needle-" + token + ".txt"
		setup := fmt.Sprintf("install -d -m 0777 %q\nprintf '%s\\n' > %q\nchmod -R a+rwX %q\n", tree+"/nested", token, target, tree)
		verify := fmt.Sprintf(
			"set -eu; expected=%q; test \"$(cat %q)\" = \"$expected\"; test \"$(cat %q)\" = \"$expected\"; locate -d %q %q >/dev/null",
			target, findResult, locateResult, db, "needle-"+token,
		)
		return fmt.Sprintf(
			"Dans %s, retrouve needle-%s.txt d'abord en temps réel avec find vers %s, puis construis %s avec updatedb et retrouve le même chemin avec locate -d vers %s.",
			tree, token, findResult, db, locateResult,
		), check(verify), setup

	case 3:
		tree := root + "/updatedb-" + token
		db := root + "/updatedb-" + token + ".db"
		target := tree + "/entry-" + token
		setup := fmt.Sprintf("install -d -m 0777 %q\nprintf '%s\\n' > %q\nchmod -R a+rwX %q\n", tree, token, target, tree)
		verify := fmt.Sprintf(
			"set -eu; test -s %q; test \"$(locate -d %q %q)\" = %q",
			db, db, "entry-"+token, target,
		)
		return fmt.Sprintf(
			"Construis une base locate dédiée %s pour l'arbre %s avec updatedb -U/-o, puis vérifie que entry-%s y est indexé.",
			db, tree, token,
		), check(verify), setup

	case 4:
		probe := "whereis sh"
		if variant == 1 {
			probe = "whereis bash"
		}
		return deterministicSnapshotTask(concept, result, probe, variant == 1)

	default:
		probe := "printf 'type='; type -P sh; printf 'which='; which sh"
		if variant == 1 {
			probe = "printf 'type='; type -P bash; printf 'which='; which bash"
		}
		return deterministicSnapshotTask(concept, result, probe, variant == 1)
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
		task, verify = "recopie exactement le flux source", fmt.Sprintf("cmp -s %q %q", source, result)
	case 2:
		task, verify = "extrais uniquement la première colonne avec ':' comme séparateur", fmt.Sprintf("cut -d: -f1 %q | cmp -s - %q", source, result)
	case 3:
		task, verify = "trie puis déduplique le flux", fmt.Sprintf("sort %q | uniq | cmp -s - %q", source, result)
	case 4:
		task, verify = "assemble les deux premières lignes avec ':'", fmt.Sprintf("head -2 %q | paste -sd: - | cmp -s - %q", source, result)
	case 5:
		task, verify = "transforme les minuscules ASCII en majuscules", fmt.Sprintf("tr '[:lower:]' '[:upper:]' < %q | cmp -s - %q", source, result)
	case 6:
		task, verify = "écris uniquement le SHA-256 de la source", fmt.Sprintf("test \"$(tr -d '[:space:]' < %q)\" = \"$(sha256sum %q | awk '{print $1}')\"", result, source)
	default:
		compressed := source + ".gz"
		setup += fmt.Sprintf("gzip -c %q > %q\\n", source, compressed)
		task, verify = "lis le gzip en flux et reconstitue son contenu", fmt.Sprintf("gzip -cd %q | cmp -s - %q", compressed, result)
		source = compressed
	}
	return fmt.Sprintf("À partir de %s, %s et écris le résultat dans %s.", source, task, result), []CheckDefinition{deterministicCommandCheck(concept.ID, verify)}, setup
}

func deterministicFileExercise(concept curriculum.Concept, root, source, result string, variant int) (string, []CheckDefinition, string) {
	setup := fmt.Sprintf("printf 'variant-%d\\n' > %q\\nmkdir -p %q/tree/a %q/tree/b\\nprintf 'one\\n' > %q/tree/a/one.log\\nprintf 'two\\n' > %q/tree/b/two.txt\\n", variant+1, source, root, root, root, root)
	var task, verify string
	switch concept.PedagogyOrder {
	case 1:
		task, verify = "copie puis déplace la source vers la cible", fmt.Sprintf("cmp -s %q %q", source, result)
	case 2:
		task, verify = "produis par globbing la liste triée des fichiers .log", fmt.Sprintf("printf '%%s\\n' %s/tree/*/*.log | sort | cmp -s - %q", root, result)
	case 3:
		task, verify = "trouve les fichiers réguliers .log et trie leurs chemins", fmt.Sprintf("find %q/tree -type f -name '*.log' | sort | cmp -s - %q", root, result)
	case 4:
		task, verify = "applique le mode 0640 à tous les fichiers avec find", fmt.Sprintf("test -z \"$(find %q/tree -type f ! -perm 0640 -print -quit)\"", root)
	case 5:
		result += ".tar"
		task, verify = "crée une archive tar du répertoire tree", fmt.Sprintf("tar -tf %q | grep -q 'tree/a/one.log' && tar -tf %q | grep -q 'tree/b/two.txt'", result, result)
	case 6:
		task, verify = "copie la source octet pour octet avec dd", fmt.Sprintf("cmp -s %q %q", source, result)
	case 7:
		result += ".gz"
		task, verify = "compresse la source avec gzip", fmt.Sprintf("gzip -cd %q | cmp -s - %q", result, source)
	default:
		task, verify = "écris la description courte produite par file", fmt.Sprintf("file -b %q | cmp -s - %q", source, result)
	}
	return fmt.Sprintf("Contexte %d : %s. Source %s, cible %s.", variant+1, task, source, result), []CheckDefinition{deterministicCommandCheck(concept.ID, verify)}, setup
}

func deterministicRedirectionExercise(concept curriculum.Concept, root, result string, variant int) (string, []CheckDefinition, string) {
	first, second := "alpha", "omega"
	if variant == 1 {
		first, second = "delta", "sigma"
	}
	var task, verify string
	switch concept.PedagogyOrder {
	case 1:
		task, verify = fmt.Sprintf("sépare stdout %s et stderr %s dans %s.out et %s.err", first, second, result, result), fmt.Sprintf("test \"$(cat %q.out)\" = %q && test \"$(cat %q.err)\" = %q", result, first, result, second)
	case 2:
		task, verify = fmt.Sprintf("écris %s puis ajoute %s avec append dans %s", first, second, result), fmt.Sprintf("test \"$(cat %q)\" = \"$(printf '%%s\\n%%s' %q %q)\"", result, first, second)
	case 3:
		task, verify = fmt.Sprintf("redirige uniquement stderr dans %s", result), fmt.Sprintf("grep -Fq %q %q", second, result)
	case 4:
		task, verify = fmt.Sprintf("utilise un pipeline pour ne conserver que %s dans %s", second, result), fmt.Sprintf("test \"$(tr -d '\\n' < %q)\" = %q", result, second)
	case 5:
		copyPath := result + ".copy"
		task, verify = fmt.Sprintf("avec tee, duplique %s dans %s et %s", first, result, copyPath), fmt.Sprintf("test \"$(cat %q)\" = %q && cmp -s %q %q", result, first, result, copyPath)
	case 6:
		task, verify = fmt.Sprintf("via xargs, crée les fichiers %s/%s et %s/%s", root, first, root, second), fmt.Sprintf("test -f %q/%s && test -f %q/%s", root, first, root, second)
	default:
		task, verify = fmt.Sprintf("réunis stdout %s et stderr %s dans %s", first, second, result), fmt.Sprintf("grep -Fq %q %q && grep -Fq %q %q", first, result, second, result)
	}
	return "Manipulation de flux : " + task + ".", []CheckDefinition{deterministicCommandCheck(concept.ID, verify)}, ""
}

func deterministicProcessExercise(concept curriculum.Concept, root, result string, variant int) (string, []CheckDefinition, string) {
	pidPath := root + "/process.pid"
	seconds := 600
	if variant == 1 {
		seconds = 900
	}
	switch concept.PedagogyOrder {
	case 1, 2, 5:
		script := fmt.Sprintf("test -s %q && kill -0 \"$(cat %q)\" 2>/dev/null && ps -o args= -p \"$(cat %q)\" | grep -q 'sleep %d'", pidPath, pidPath, pidPath, seconds)
		return fmt.Sprintf("Démarre sleep %d en arrière-plan ou via nohup selon le concept, puis écris son PID dans %s.", seconds, pidPath), []CheckDefinition{deterministicCommandCheck(concept.ID, script)}, ""
	case 3, 6:
		name := fmt.Sprintf("lpic-daily-probe-%d", variant+1)
		setup := fmt.Sprintf("sh -c 'exec -a %s sleep %d' & echo $! > %q\\n", name, seconds, pidPath)
		script := fmt.Sprintf("test -s %q && ! kill -0 \"$(cat %q)\" 2>/dev/null", pidPath, pidPath)
		return fmt.Sprintf("Termine le processus préparé %s avec le mécanisme adapté.", name), []CheckDefinition{deterministicCommandCheck(concept.ID, script)}, setup
	case 4:
		setup := fmt.Sprintf("sleep %d & echo $! > %q\\n", seconds, pidPath)
		script := fmt.Sprintf("test \"$(tr -d '[:space:]' < %q)\" = \"$(cat %q)\" && kill -0 \"$(cat %q)\" 2>/dev/null", result, pidPath, pidPath)
		return fmt.Sprintf("Retrouve le processus préparé avec ps ou top et écris son PID dans %s.", result), []CheckDefinition{deterministicCommandCheck(concept.ID, script)}, setup
	default:
		session := "lpicdaily-diagnostic"
		if variant == 1 {
			session = "lpicdaily-transfer"
		}
		script := fmt.Sprintf("(tmux has-session -t %q 2>/dev/null) || (screen -ls 2>/dev/null | grep -Fq %q)", session, session)
		return fmt.Sprintf("Crée une session détachée persistante nommée %s avec tmux ou screen.", session), []CheckDefinition{deterministicCommandCheck(concept.ID, script)}, ""
	}
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
		if variant == 0 {
			script := fmt.Sprintf("test -s %q && test \"$(ps -o ni= -p \"$(cat %q)\" | tr -d ' ')\" = 0", pidPath, pidPath)
			return fmt.Sprintf("Démarre sleep 600 à priorité normale et écris son PID dans %s.", pidPath), []CheckDefinition{deterministicCommandCheck(concept.ID, script)}, ""
		}
		setup := fmt.Sprintf("sleep 900 & echo $! > %q\n", pidPath)
		script := fmt.Sprintf("test -s %q && kill -0 \"$(cat %q)\" 2>/dev/null && test \"$(tr -d '[:space:]' < %q)\" = 0 && test \"$(ps -o ni= -p \"$(cat %q)\" | tr -d ' ')\" = 0", pidPath, pidPath, result, pidPath)
		return fmt.Sprintf("Un processus à priorité par défaut est préparé dans %s. Observe sa nice value avec ps/top et écris uniquement la valeur dans %s.", pidPath, result), []CheckDefinition{deterministicCommandCheck(concept.ID, script)}, setup
	case 4:
		setup := fmt.Sprintf("sleep 600 & echo $! > %q\\n", pidPath)
		script := fmt.Sprintf("test \"$(ps -o ni= -p \"$(cat %q)\" | tr -d ' ')\" = %d", pidPath, target)
		return fmt.Sprintf("Change avec renice la nice value du PID contenu dans %s vers %d.", pidPath, target), []CheckDefinition{deterministicCommandCheck(concept.ID, script)}, setup
	default:
		setup := fmt.Sprintf("nice -n %d sleep 600 & echo $! > %q\\n", target, pidPath)
		script := fmt.Sprintf("test \"$(tr -d '[:space:]' < %q)\" = \"$(ps -o ni= -p \"$(cat %q)\" | tr -d ' ')\"", result, pidPath)
		return fmt.Sprintf("Observe la nice value du PID dans %s et écris-la dans %s.", pidPath, result), []CheckDefinition{deterministicCommandCheck(concept.ID, script)}, setup
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
		return fmt.Sprintf("Avec sed, remplace les valeurs numériques finales par N dans %s puis écris %s.", source, result), []CheckDefinition{deterministicCommandCheck(concept.ID, verify)}, setup
	}
	pattern := "^[a-z]+:[0-9]+$"
	switch concept.PedagogyOrder {
	case 2:
		pattern = "^[[:alpha:]]+:[0-9]+$"
	case 3:
		pattern = ":[0-9][0-9]*$"
	case 4:
		pattern = "^(alpha|node):[0-9]+$"
	case 5:
		pattern = "^[a-z]+:"
	}
	verify := fmt.Sprintf("grep -E %q %q | cmp -s - %q", pattern, source, result)
	return fmt.Sprintf("Filtre %s avec l'expression régulière adaptée et écris les lignes retenues dans %s.", source, result), []CheckDefinition{deterministicCommandCheck(concept.ID, verify)}, setup
}

func deterministicEditorExercise(concept curriculum.Concept, result string, variant int) (string, []CheckDefinition, string) {
	initial, expected := "one\\ntwo\\nthree\\n", "one\\nTWO\\nthree\\n"
	if variant == 1 {
		initial, expected = "red\\ngreen\\nblue\\n", "red\\nGREEN\\nblue\\n"
	}
	setup := fmt.Sprintf("printf '%%b' %q > %q\\n", initial, result)
	verify := fmt.Sprintf("test \"$(cat %q)\" = \"$(printf '%%b' %q)\"", result, expected)
	return fmt.Sprintf("Édite %s : seule la ligne centrale doit passer en majuscules.", result), []CheckDefinition{deterministicCommandCheck(concept.ID, verify)}, setup
}

func deterministicStorageExercise(objectiveID string, concept curriculum.Concept, root, result string, variant int) (string, []CheckDefinition, string) {
	check := func(script string) []CheckDefinition {
		return []CheckDefinition{deterministicCommandCheck(concept.ID, script)}
	}
	if objectiveID == "102.1" {
		switch concept.PedagogyOrder {
		case 1, 3:
			probe := "df -P / /var /home /boot 2>/dev/null | sed -n '1,8p'"
			if variant == 1 {
				probe = "findmnt -rn -o SOURCE,TARGET,FSTYPE / /var /home /boot 2>/dev/null || true"
			}
			return deterministicSnapshotTask(concept, result, probe, variant == 1)
		case 2:
			label := "LPIC_SWAP_A"
			if variant == 1 {
				label = "LPIC_SWAP_B"
			}
			script := fmt.Sprintf("grep -q '^/dev/vdb[[:space:]]' /proc/swaps && test \"$(blkid -s LABEL -o value /dev/vdb 2>/dev/null)\" = %s", label)
			return fmt.Sprintf("Initialise /dev/vdb comme swap avec le label %s puis active-le.", label), check(script), ""
		case 4:
			label := "ESP_A"
			if variant == 1 {
				label = "ESP_B"
			}
			script := fmt.Sprintf("parted -sm /dev/vdb print 2>/dev/null | grep -q ':gpt:' && parted -sm /dev/vdb print 2>/dev/null | grep -q 'esp' && test \"$(blkid -s LABEL -o value /dev/vdb1 2>/dev/null)\" = %s", label)
			return fmt.Sprintf("Crée sur /dev/vdb une table GPT et une partition FAT32 avec drapeau ESP et label %s.", label), check(script), ""
		case 5:
			if variant == 0 {
				return "Crée sur /dev/vdb une table GPT avec une partition /dev/vdb1.", check("test -b /dev/vdb1 && test ! -b /dev/vdb2 && parted -sm /dev/vdb print 2>/dev/null | grep -q '^1:'"), ""
			}
			return "Crée sur /dev/vdb une table GPT avec deux partitions utilisables /dev/vdb1 et /dev/vdb2.", check("test -b /dev/vdb1 && test -b /dev/vdb2 && parted -sm /dev/vdb print 2>/dev/null | grep -q '^2:'"), ""
		default:
			vg, lv := "lpicvg", "lab"
			if variant == 1 {
				vg, lv = "transfervg", "data"
			}
			return fmt.Sprintf("Initialise /dev/vdb en PV LVM, crée le VG %s puis le LV %s.", vg, lv), check(fmt.Sprintf("pvs /dev/vdb >/dev/null 2>&1 && vgs %s >/dev/null 2>&1 && lvs %s/%s >/dev/null 2>&1", vg, vg, lv)), ""
		}
	}
	if objectiveID == "104.1" {
		switch concept.PedagogyOrder {
		case 1:
			table := "gpt"
			if variant == 1 {
				table = "msdos"
			}
			return fmt.Sprintf("Crée sur /dev/vdb une table de partitions %s.", table), check(fmt.Sprintf("parted -sm /dev/vdb print 2>/dev/null | head -1 | grep -q ':%s:'", table)), ""
		case 2:
			if variant == 0 {
				return "Crée une table GPT puis une première partition /dev/vdb1.", check("test -b /dev/vdb1 && test ! -b /dev/vdb2 && parted -sm /dev/vdb print 2>/dev/null | grep -q '^1:'"), ""
			}
			return "Crée une table GPT puis deux partitions utilisables /dev/vdb1 et /dev/vdb2.", check("test -b /dev/vdb1 && test -b /dev/vdb2 && parted -sm /dev/vdb print 2>/dev/null | grep -q '^2:'"), ""
		case 3:
			fs := "ext4"
			if variant == 1 {
				fs = "ext3"
			}
			return fmt.Sprintf("Crée un filesystem %s sur /dev/vdb.", fs), check(fmt.Sprintf("test \"$(blkid -s TYPE -o value /dev/vdb 2>/dev/null)\" = %s", fs)), ""
		case 4:
			label := "LPIC_XFS_A"
			if variant == 1 {
				label = "LPIC_XFS_B"
			}
			script := fmt.Sprintf("test \"$(blkid -s TYPE -o value /dev/vdb 2>/dev/null)\" = xfs && test \"$(blkid -s LABEL -o value /dev/vdb 2>/dev/null)\" = %s", label)
			return fmt.Sprintf("Crée un filesystem XFS sur /dev/vdb avec le label %s.", label), check(script), ""
		case 5:
			label := "LPIC_VFAT_A"
			if variant == 1 {
				label = "LPIC_VFAT_B"
			}
			script := fmt.Sprintf("test \"$(blkid -s TYPE -o value /dev/vdb 2>/dev/null)\" = vfat && test \"$(blkid -s LABEL -o value /dev/vdb 2>/dev/null)\" = %s", label)
			return fmt.Sprintf("Crée un filesystem VFAT sur /dev/vdb avec le label %s.", label), check(script), ""
		case 6:
			label := "LPIC_SWAP_A"
			if variant == 1 {
				label = "LPIC_SWAP_B"
			}
			script := fmt.Sprintf("grep -q '^/dev/vdb[[:space:]]' /proc/swaps && test \"$(blkid -s LABEL -o value /dev/vdb 2>/dev/null)\" = %s", label)
			return fmt.Sprintf("Initialise /dev/vdb comme swap avec le label %s puis active-le.", label), check(script), ""
		case 7:
			label := "LPIC_BTRFS_A"
			if variant == 1 {
				label = "LPIC_BTRFS_B"
			}
			script := fmt.Sprintf("test \"$(blkid -s TYPE -o value /dev/vdb 2>/dev/null)\" = btrfs && test \"$(blkid -s LABEL -o value /dev/vdb 2>/dev/null)\" = %s", label)
			return fmt.Sprintf("Crée un filesystem Btrfs sur /dev/vdb avec le label %s.", label), check(script), ""
		default:
			label := "LPIC_FS_A"
			if variant == 1 {
				label = "LPIC_FS_B"
			}
			script := fmt.Sprintf("test \"$(blkid -s TYPE -o value /dev/vdb 2>/dev/null)\" = ext4 && test \"$(blkid -s LABEL -o value /dev/vdb 2>/dev/null)\" = %s", label)
			return fmt.Sprintf("Crée un filesystem ext4 sur /dev/vdb avec mkfs ou mke2fs et le label %s.", label), check(script), ""
		}
	}
	if objectiveID == "104.2" {
		switch concept.PedagogyOrder {
		case 1, 7:
			probe := "df -P / | tail -1; df -Pi / | tail -1; du -sx /var 2>/dev/null || true"
			if variant == 1 {
				probe = "df -PT / | tail -1; df -Pi / | tail -1"
			}
			return deterministicSnapshotTask(concept, result, probe, variant == 1)
		case 2:
			fs := "ext4"
			if variant == 1 {
				fs = "ext3"
			}
			return fmt.Sprintf("Crée un %s sur /dev/vdb puis exécute une vérification hors ligne avec e2fsck.", fs), check(fmt.Sprintf("test \"$(blkid -s TYPE -o value /dev/vdb 2>/dev/null)\" = %s && e2fsck -fn /dev/vdb >/dev/null 2>&1", fs)), ""
		case 3, 6:
			if variant == 0 {
				return "Crée un ext4 sur /dev/vdb puis exécute une vérification/réparation hors ligne avec e2fsck.", check("test \"$(blkid -s TYPE -o value /dev/vdb 2>/dev/null)\" = ext4 && e2fsck -fn /dev/vdb >/dev/null 2>&1"), ""
			}
			return "Crée un XFS sur /dev/vdb puis exécute la vérification hors ligne avec xfs_repair -n.", check("test \"$(blkid -s TYPE -o value /dev/vdb 2>/dev/null)\" = xfs && xfs_repair -n /dev/vdb >/dev/null 2>&1"), ""
		case 4:
			label := "LPIC_TUNE_A"
			if variant == 1 {
				label = "LPIC_TUNE_B"
			}
			return fmt.Sprintf("Crée un ext4 sur /dev/vdb puis règle son label à %s.", label), check(fmt.Sprintf("test \"$(blkid -s LABEL -o value /dev/vdb 2>/dev/null)\" = %s", label)), ""
		default:
			if variant == 0 {
				return "Crée un XFS sur /dev/vdb puis exécute xfs_repair -n.", check("test \"$(blkid -s TYPE -o value /dev/vdb 2>/dev/null)\" = xfs && xfs_repair -n /dev/vdb >/dev/null 2>&1"), ""
			}
			script := fmt.Sprintf("test \"$(blkid -s TYPE -o value /dev/vdb 2>/dev/null)\" = xfs && xfs_db -r -c 'sb 0' -c 'p uuid' /dev/vdb 2>/dev/null | cmp -s - %q", result)
			return fmt.Sprintf("Crée un XFS sur /dev/vdb, inspecte son superbloc avec xfs_db et écris la ligne UUID canonique dans %s.", result), check(script), ""
		}
	}
	mountpoint := root + "/mnt"
	switch concept.PedagogyOrder {
	case 1:
		target := mountpoint
		if variant == 1 {
			target = root + "/manual-transfer"
		}
		return fmt.Sprintf("Crée un ext4 sur /dev/vdb puis monte-le sur %s.", target), check(fmt.Sprintf("findmnt -n %q -S /dev/vdb >/dev/null 2>&1", target)), ""
	case 2:
		target := mountpoint
		if variant == 1 {
			target = root + "/persist"
		}
		return fmt.Sprintf("Ajoute dans /etc/fstab une entrée ext4 de /dev/vdb vers %s avec defaults.", target), check(fmt.Sprintf("grep -Eq '^/dev/vdb[[:space:]]+%s[[:space:]]+ext4[[:space:]]+defaults' /etc/fstab", target)), ""
	case 3:
		label := "LPIC_DATA_A"
		if variant == 1 {
			label = "LPIC_DATA_B"
		}
		script := fmt.Sprintf("test \"$(blkid -s LABEL -o value /dev/vdb 2>/dev/null)\" = %s && test \"$(tr -d '[:space:]' < %q)\" = \"$(blkid -s UUID -o value /dev/vdb)\"", label, result)
		return fmt.Sprintf("Crée un ext4 sur /dev/vdb avec le label %s et écris son UUID dans %s.", label, result), check(script), ""
	case 4:
		option := "ro"
		if variant == 1 {
			option = "noexec"
		}
		return fmt.Sprintf("Monte /dev/vdb sur %s avec l'option %s.", mountpoint, option), check(fmt.Sprintf("findmnt -n %q -S /dev/vdb -O %s >/dev/null 2>&1", mountpoint, option)), ""
	case 5:
		marker := ".lpic-mounted-a"
		if variant == 1 {
			marker = ".lpic-mounted-b"
		}
		verifyDir := root + "/verify-unmount"
		script := fmt.Sprintf("set -eu; test \"$(blkid -s TYPE -o value /dev/vdb 2>/dev/null)\" = ext4; ! findmnt -n %q >/dev/null 2>&1; mkdir -p %q; mount /dev/vdb %q; test -f %q/%s; umount %q", mountpoint, verifyDir, verifyDir, verifyDir, marker, verifyDir)
		return fmt.Sprintf("Crée un ext4 sur /dev/vdb, monte-le sur %s, crée le marqueur %s à sa racine puis démonte-le avant validation.", mountpoint, marker), check(script), ""
	case 6:
		unit := "/etc/systemd/system/lpic-data.mount"
		options := "ro"
		if variant == 1 {
			options = "noexec"
		}
		script := fmt.Sprintf("grep -q '^\\[Mount\\]$' %q && grep -q '^What=/dev/vdb$' %q && grep -q '^Where=/mnt/lpic-data$' %q && grep -q '^Options=%s$' %q", unit, unit, unit, options, unit)
		return fmt.Sprintf("Crée l'unité systemd %s avec What=/dev/vdb, Where=/mnt/lpic-data et Options=%s.", unit, options), check(script), ""
	default:
		probe := "find /media /run/media -mindepth 1 -maxdepth 2 -type d 2>/dev/null | sort | head -12; lsblk -o NAME,UUID,LABEL,MOUNTPOINTS 2>/dev/null"
		if variant == 1 {
			probe = "findmnt -rn -o SOURCE,TARGET,FSTYPE | head -16; blkid 2>/dev/null | sort | head -12"
		}
		return deterministicSnapshotTask(concept, result, probe, variant == 1)
	}
}

func deterministicPermissionExercise(concept curriculum.Concept, root string, variant int) (string, []CheckDefinition, string) {
	path := root + "/perm-diagnostic"
	if variant == 1 {
		path = root + "/perm-transfer"
	}
	switch concept.PedagogyOrder {
	case 1, 2:
		mode := "0640"
		if variant == 1 {
			mode = "0750"
		}
		return fmt.Sprintf("Crée %s puis fixe ses permissions à %s.", path, mode), []CheckDefinition{{Type: "file-mode", Path: path, Mode: mode, ConceptIDs: []string{concept.ID}}}, ""
	case 3:
		return fmt.Sprintf("Crée %s et fixe son propriétaire/groupe à root:root.", path), []CheckDefinition{{Type: "file-owner", Path: path, User: "root", Group: "root", ConceptIDs: []string{concept.ID}}}, ""
	case 4:
		mode := "0640"
		if variant == 1 {
			mode = "0660"
		}
		return fmt.Sprintf("Choisis une umask adaptée puis crée %s avec le mode final %s.", path, mode), []CheckDefinition{{Type: "file-mode", Path: path, Mode: mode, ConceptIDs: []string{concept.ID}}}, ""
	case 5:
		return fmt.Sprintf("Crée %s exécutable puis active son bit SUID.", path), []CheckDefinition{deterministicCommandCheck(concept.ID, fmt.Sprintf("test -u %q", path))}, ""
	case 6:
		return fmt.Sprintf("Crée le répertoire %s puis active son bit SGID.", path), []CheckDefinition{deterministicCommandCheck(concept.ID, fmt.Sprintf("test -d %q && test -g %q", path, path))}, ""
	case 7:
		return fmt.Sprintf("Crée le répertoire %s puis active son sticky bit.", path), []CheckDefinition{deterministicCommandCheck(concept.ID, fmt.Sprintf("test -d %q && test -k %q", path, path))}, ""
	default:
		mode := "3770"
		if variant == 1 {
			mode = "3775"
		}
		return fmt.Sprintf("Crée le répertoire partagé %s avec mode %s.", path, mode), []CheckDefinition{{Type: "file-mode", Path: path, Mode: mode, ConceptIDs: []string{concept.ID}}}, ""
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
		return fmt.Sprintf("Crée %s comme lien symbolique vers %s.", link, target), []CheckDefinition{deterministicCommandCheck(concept.ID, script)}, setup
	case 4:
		setup += fmt.Sprintf("ln -s %q %q\\nrm -f %q\\n", target, link, target)
		return fmt.Sprintf("Le lien %s est préparé comme symlink cassé. Diagnostique-le sans recréer la cible.", link), []CheckDefinition{deterministicCommandCheck(concept.ID, fmt.Sprintf("test -L %q && test ! -e %q", link, link))}, setup
	case 5:
		copyPath, hardPath, symPath := root+"/copy", root+"/hard", root+"/sym"
		script := fmt.Sprintf("cmp -s %q %q && test %q -ef %q && test -L %q", target, copyPath, target, hardPath, symPath)
		return fmt.Sprintf("À partir de %s, crée une copie %s, un hard link %s et un symlink %s.", target, copyPath, hardPath, symPath), []CheckDefinition{deterministicCommandCheck(concept.ID, script)}, setup
	default:
		script := fmt.Sprintf("test -L %q && test \"$(readlink -f %q)\" = \"$(readlink -f %q)\"", link, link, target)
		return fmt.Sprintf("Crée %s comme symlink vers %s.", link, target), []CheckDefinition{deterministicCommandCheck(concept.ID, script)}, setup
	}
}
