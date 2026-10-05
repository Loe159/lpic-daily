package content

import (
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/Loe159/lpic-daily/internal/curriculum"
)

const standaloneSupplementHeading = "## Complément LPIC Daily autonome"

func synthesizeStandaloneContent(
	curriculumBundle *curriculum.Bundle,
	bundle *Bundle,
	knownObjectives map[string]struct{},
	knownConcepts map[string]curriculum.Concept,
	seenIDs map[string]string,
) error {
	objectiveByID := make(map[string]curriculum.Objective)
	for _, objective := range curriculumBundle.Objectives.Objectives {
		if objective.Active {
			objectiveByID[objective.ID] = objective
		}
	}
	guideByObjective := make(map[string]curriculum.ObjectiveStudyGuide)
	for _, guide := range curriculumBundle.StudyGuides.Guides {
		guideByObjective[guide.ObjectiveID] = guide
	}

	focusedIntroduction := make(map[string]int)
	for index := range bundle.Lessons {
		lesson := &bundle.Lessons[index]
		if lesson.Stage != "introduce" || len(lesson.ConceptIDs) != 1 {
			continue
		}
		conceptID := lesson.ConceptIDs[0]
		focusedIntroduction[conceptID]++
		concept, exists := knownConcepts[conceptID]
		if !exists {
			continue
		}
		objective := objectiveByID[concept.ObjectiveID]
		guide := guideByObjective[concept.ObjectiveID]
		supplement := standaloneSupplement(objective, concept, guide)
		if !strings.Contains(lesson.BodyMarkdown, standaloneSupplementHeading) {
			lesson.BodyMarkdown = strings.TrimRight(lesson.BodyMarkdown, "\n") + "\n\n" + supplement
		}
	}

	for _, objective := range curriculumBundle.Objectives.Objectives {
		if !objective.Active {
			continue
		}
		for _, term := range objective.TermsFilesUtilities {
			if !hasSpecificStandaloneTermExplanation(term, objective.ID) {
				return fmt.Errorf("objective %s term %q has no specific standalone explanation", objective.ID, term)
			}
		}
		guide, exists := guideByObjective[objective.ID]
		if !exists {
			return fmt.Errorf("objective %s has no study guide", objective.ID)
		}
		concepts := conceptsForObjective(curriculumBundle.Concepts.Concepts, objective.ID)
		for index, concept := range concepts {
			if focusedIntroduction[concept.ID] == 0 {
				lesson := generatedIntroduction(objective, concept, guide, index, len(concepts))
				if err := validateLesson(lesson, knownObjectives, knownConcepts); err != nil {
					return fmt.Errorf("%s: %w", lesson.ID, err)
				}
				if previous, duplicate := seenIDs[lesson.ID]; duplicate {
					return fmt.Errorf("generated lesson %s conflicts with %s", lesson.ID, previous)
				}
				seenIDs[lesson.ID] = "generated standalone lesson"
				bundle.Lessons = append(bundle.Lessons, lesson)
				focusedIntroduction[concept.ID] = 1
			}
			if focusedIntroduction[concept.ID] != 1 {
				return fmt.Errorf("concept %s has %d focused introductions, want exactly one", concept.ID, focusedIntroduction[concept.ID])
			}

			for _, question := range []Question{
				generatedRecallQuestion(objective, concept, index, len(concepts)),
				generatedRecognitionQuestion(objective, concept, concepts, index),
			} {
				if err := validateQuestion(question, knownObjectives, knownConcepts); err != nil {
					return fmt.Errorf("%s: %w", question.ID, err)
				}
				if previous, duplicate := seenIDs[question.ID]; duplicate {
					return fmt.Errorf("generated question %s conflicts with %s", question.ID, previous)
				}
				seenIDs[question.ID] = "generated standalone question"
				bundle.Questions = append(bundle.Questions, question)
			}
		}
	}
	return nil
}

func conceptsForObjective(all []curriculum.Concept, objectiveID string) []curriculum.Concept {
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

func generatedIntroduction(
	objective curriculum.Objective,
	concept curriculum.Concept,
	guide curriculum.ObjectiveStudyGuide,
	index int,
	conceptCount int,
) Lesson {
	return Lesson{
		SchemaVersion:          "1.0.0",
		ID:                     concept.ID + ".lesson.autonomous",
		TitleFR:                concept.TitleFR + " — cours complet",
		ObjectiveIDs:           []string{objective.ID},
		ConceptIDs:             []string{concept.ID},
		PrerequisiteConceptIDs: slices.Clone(concept.PrerequisiteConceptIDs),
		Stage:                  "introduce",
		EstimatedMinutes:       8,
		BodyMarkdown:           generatedLessonBody(objective, concept, guide, index, conceptCount),
		Labels:                 conceptLabels(concept),
		Distribution:           "generic",
		SourceRefs: []SourceRef{{
			Title: "Objectifs officiels LPIC-1 v5.0",
			URL:   "https://www.lpi.org/our-certifications/exam-101-102-objectives/",
		}},
	}
}

func standaloneSupplement(
	objective curriculum.Objective,
	concept curriculum.Concept,
	guide curriculum.ObjectiveStudyGuide,
) string {
	concepts := []curriculum.Concept{concept}
	return standaloneSupplementHeading + "\n\n" +
		generatedLessonBody(objective, concept, guide, 0, len(concepts))
}

func generatedLessonBody(
	objective curriculum.Objective,
	concept curriculum.Concept,
	guide curriculum.ObjectiveStudyGuide,
	index int,
	conceptCount int,
) string {
	terms := termsForConcept(objective.TermsFilesUtilities, concept.TitleFR, index, conceptCount)
	var termList strings.Builder
	for _, term := range objective.TermsFilesUtilities {
		fmt.Fprintf(&termList, "- `%s` — %s", term, standaloneTermExplanation(term, objective.ID))
		if usage := standaloneTermUsage(term); usage != "" {
			fmt.Fprintf(&termList, "  \n  **À savoir pratiquer :** `%s`", usage)
		}
		termList.WriteByte('\n')
	}
	var evidence strings.Builder
	for _, item := range objective.AssessmentEvidence {
		fmt.Fprintf(&evidence, "- %s\n", item)
	}
	return fmt.Sprintf(
		"# %s\n\n"+
			"Ce concept appartient à **%s — %s**. L'objectif est de savoir l'expliquer, le reconnaître dans un scénario d'examen et l'utiliser ou le diagnostiquer sur un système Linux.\n\n"+
			"## Modèle mental\n\n%s\n\n"+
			"## Focus sur ce concept\n\n"+
			"Travaille particulièrement **%s**. Les repères techniques associés à ce module sont %s. Pour chacun, sache ce qu'il observe ou modifie, quand l'utiliser et comment vérifier le résultat.\n\n"+
			"## Mise en pratique\n\n%s\n\n"+
			"## Pièges et distinctions\n\n%s\n\n"+
			"## Termes, fichiers et utilitaires à connaître pour %s\n\n%s\n"+
			"## Ce que l'examen peut te demander de démontrer\n\n%s\n"+
			"Avant de continuer, reformule le concept sans relire le titre, cite au moins un outil ou fichier pertinent et décris comment tu vérifierais ton résultat.\n",
		concept.TitleFR,
		objective.ID,
		objective.TitleFR,
		guide.Overview,
		concept.TitleFR,
		inlineCodeList(terms),
		guide.Practice,
		guide.Pitfalls,
		objective.ID,
		termList.String(),
		evidence.String(),
	)
}

func generatedRecallQuestion(
	objective curriculum.Objective,
	concept curriculum.Concept,
	index int,
	conceptCount int,
) Question {
	terms := termsForConcept(objective.TermsFilesUtilities, concept.TitleFR, index, conceptCount)
	primary := terms[0]
	description := standaloneTermExplanation(primary, objective.ID)
	return Question{
		SchemaVersion: "1.0.0",
		ID:            concept.ID + ".q.autonomous-recall",
		ObjectiveIDs:  []string{objective.ID},
		ConceptIDs:    []string{concept.ID},
		Type:          "fill-in",
		Usage:         "daily",
		PromptFR: fmt.Sprintf(
			"Dans %s, quel terme, fichier ou utilitaire correspond à cette description : %s ?",
			objective.ID,
			description,
		),
		Grading: Grading{
			Strategy:        "exact-text",
			AcceptedAnswers: []string{primary},
			CaseSensitive:   false,
		},
		EvidenceKindOnSuccess: "recall",
		Labels:                conceptLabels(concept),
		Distribution:          "generic",
		ExplanationFR: fmt.Sprintf(
			"%s : %s",
			primary,
			description,
		),
	}
}

func generatedRecognitionQuestion(
	objective curriculum.Objective,
	concept curriculum.Concept,
	concepts []curriculum.Concept,
	index int,
) Question {
	terms := termsForConcept(objective.TermsFilesUtilities, concept.TitleFR, index, len(concepts))
	primary := terms[0]
	correctDescription := standaloneTermExplanation(primary, objective.ID)

	distractorTerms := make([]string, 0, 3)
	for offset := 1; len(distractorTerms) < 3 && offset <= len(objective.TermsFilesUtilities); offset++ {
		candidate := objective.TermsFilesUtilities[(index+offset)%len(objective.TermsFilesUtilities)]
		if candidate == primary || slices.Contains(distractorTerms, candidate) {
			continue
		}
		distractorTerms = append(distractorTerms, candidate)
	}
	for len(distractorTerms) < 3 {
		distractorTerms = append(distractorTerms, "Linux")
	}

	choices := []Choice{
		{ID: "correct", LabelFR: correctDescription},
		{ID: "other-1", LabelFR: standaloneTermExplanation(distractorTerms[0], objective.ID)},
		{ID: "other-2", LabelFR: standaloneTermExplanation(distractorTerms[1], objective.ID)},
		{ID: "other-3", LabelFR: standaloneTermExplanation(distractorTerms[2], objective.ID)},
	}
	rotation := index % len(choices)
	choices[0], choices[rotation] = choices[rotation], choices[0]

	return Question{
		SchemaVersion: "1.0.0",
		ID:            concept.ID + ".q.autonomous-recognition",
		ObjectiveIDs:  []string{objective.ID},
		ConceptIDs:    []string{concept.ID},
		Type:          "multiple-choice",
		Usage:         "daily",
		PromptFR: fmt.Sprintf(
			"Dans %s, quelle description correspond correctement à %s ?",
			objective.ID,
			inlineCodeList([]string{primary}),
		),
		Choices: choices,
		Grading: Grading{
			Strategy:          "choice-ids",
			AcceptedChoiceIDs: []string{"correct"},
		},
		EvidenceKindOnSuccess: "recognition",
		Labels:                conceptLabels(concept),
		Distribution:          "generic",
		ExplanationFR: fmt.Sprintf(
			"%s : %s",
			primary,
			correctDescription,
		),
	}
}

func termsForConcept(terms []string, title string, index int, conceptCount int) []string {
	if len(terms) == 0 {
		return []string{"Linux"}
	}
	normalizedTitle := normalizeSearchText(title)
	var matches []string
	for _, term := range terms {
		normalizedTerm := normalizeSearchText(term)
		if normalizedTerm != "" && strings.Contains(normalizedTitle, normalizedTerm) {
			matches = append(matches, term)
		}
	}
	slices.SortFunc(matches, func(a, b string) int {
		return len(normalizeSearchText(b)) - len(normalizeSearchText(a))
	})
	for _, hint := range conceptTermHints {
		if !strings.Contains(normalizedTitle, hint.keyword) {
			continue
		}
		for _, wanted := range hint.terms {
			if !slices.Contains(matches, wanted) {
				matches = append(matches, wanted)
			}
		}
	}
	if len(matches) != 0 {
		return matches
	}
	if conceptCount <= 0 {
		conceptCount = 1
	}
	for termIndex, term := range terms {
		if termIndex%conceptCount == index {
			matches = append(matches, term)
		}
	}
	if len(matches) == 0 {
		matches = append(matches, terms[index%len(terms)])
	}
	return matches
}

type conceptTermHint struct {
	keyword string
	terms   []string
}

var conceptTermHints = []conceptTermHint{
	{keyword: "module", terms: []string{"modprobe", "lsmod"}},
	{keyword: "pci", terms: []string{"lspci"}},
	{keyword: "usb", terms: []string{"lsusb"}},
	{keyword: "udev", terms: []string{"udev", "sysfs", "D-Bus"}},
	{keyword: "boot", terms: []string{"bootloader", "kernel", "initramfs", "grub-install", "grub-mkconfig"}},
	{keyword: "grub", terms: []string{"GRUB Legacy", "GRUB 2", "grub-install", "grub-mkconfig", "grub.cfg"}},
	{keyword: "variable", terms: []string{"env", "export", "set", "unset", "LANG", "LC_ALL"}},
	{keyword: "path", terms: []string{"PATH", "type", "which"}},
	{keyword: "histor", terms: []string{"history", ".bash_history"}},
	{keyword: "documentation", terms: []string{"man", "type", "which"}},
	{keyword: "systeme", terms: []string{"uname", "pwd"}},
	{keyword: "compression", terms: []string{"gzip", "bzip2", "xz", "zcat", "bzcat", "xzcat"}},
	{keyword: "archive", terms: []string{"tar", "cpio"}},
	{keyword: "checksum", terms: []string{"md5sum", "sha256sum", "sha512sum"}},
	{keyword: "stderr", terms: []string{"stderr", "2>", "2>&1"}},
	{keyword: "pipeline", terms: []string{"|"}},
	{keyword: "tee", terms: []string{"tee"}},
	{keyword: "xargs", terms: []string{"xargs"}},
	{keyword: "signaux", terms: []string{"kill", "pkill", "killall"}},
	{keyword: "multiplex", terms: []string{"screen", "tmux"}},
	{keyword: "priorite", terms: []string{"nice", "renice"}},
	{keyword: "regex", terms: []string{"grep", "regex(7)", "BRE", "ERE"}},
	{keyword: "sed", terms: []string{"sed"}},
	{keyword: "inode", terms: []string{"ln", "inode"}},
	{keyword: "symlink", terms: []string{"ln", "symbolic link"}},
	{keyword: "locale", terms: []string{"locale", "LANG", "LC_ALL", "LC_*"}},
	{keyword: "timezone", terms: []string{"TZ", "timedatectl", "tzselect", "/etc/localtime"}},
	{keyword: "ntp", terms: []string{"ntpd", "ntpdate", "chronyc", "pool.ntp.org"}},
	{keyword: "journal", terms: []string{"journalctl", "/var/log/journal", "rsyslog", "logrotate"}},
	{keyword: "rsyslog", terms: []string{"/etc/rsyslog.conf", "logger"}},
	{keyword: "ipv6", terms: []string{"IPv6"}},
	{keyword: "cidr", terms: []string{"CIDR", "subnet"}},
	{keyword: "dns", terms: []string{"dig", "host", "/etc/resolv.conf", "/etc/nsswitch.conf"}},
	{keyword: "ssh", terms: []string{"ssh", "ssh-keygen", "ssh-agent", "ssh-add"}},
	{keyword: "gpg", terms: []string{"gpg", "gpg-agent", "~/.gnupg"}},
}

func normalizeSearchText(value string) string {
	var builder strings.Builder
	for _, r := range strings.ToLower(value) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
		} else {
			builder.WriteByte(' ')
		}
	}
	return strings.Join(strings.Fields(builder.String()), " ")
}

func inlineCodeList(values []string) string {
	if len(values) == 0 {
		return "`Linux`"
	}
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, "`"+value+"`")
	}
	return strings.Join(quoted, ", ")
}

func standaloneTermUsage(term string) string {
	return standaloneTermUsages[term]
}

var standaloneTermUsages = map[string]string{
	"modprobe":         "modprobe <module> ; modprobe -r <module>",
	"lsmod":            "lsmod",
	"lspci":            "lspci ; lspci -k",
	"lsusb":            "lsusb ; lsusb -v",
	"dmesg":            "dmesg ; dmesg -T | less",
	"journalctl":       "journalctl -b ; journalctl -u <unit> ; journalctl -p err ; journalctl --since <time>",
	"shutdown":         "shutdown -h now ; shutdown -r +5 ; shutdown -c",
	"init":             "init <runlevel>",
	"telinit":          "telinit <runlevel>",
	"systemctl":        "systemctl status <unit> ; systemctl enable --now <unit> ; systemctl isolate <target> ; systemctl get-default",
	"wall":             "wall < message.txt",
	"grub-install":     "grub-install <device-ou-options-UEFI>",
	"grub-mkconfig":    "grub-mkconfig -o <grub.cfg>",
	"ldd":              "ldd <binaire>",
	"ldconfig":         "ldconfig ; ldconfig -p",
	"dpkg":             "dpkg -i <paquet.deb> ; dpkg -l ; dpkg -L <paquet> ; dpkg -S <fichier>",
	"dpkg-reconfigure": "dpkg-reconfigure <paquet>",
	"apt-get":          "apt-get update ; apt-get install <paquet> ; apt-get remove <paquet>",
	"apt-cache":        "apt-cache search <mot> ; apt-cache show <paquet> ; apt-cache depends <paquet>",
	"apt":              "apt update ; apt install <paquet> ; apt remove <paquet> ; apt list --installed",
	"rpm":              "rpm -q <paquet> ; rpm -qi <paquet> ; rpm -ql <paquet> ; rpm -qf <fichier> ; rpm -V <paquet>",
	"rpm2cpio":         "rpm2cpio <paquet.rpm> | cpio -idmv",
	"yum":              "yum install <paquet> ; yum remove <paquet> ; yum repolist",
	"dnf":              "dnf install <paquet> ; dnf remove <paquet> ; dnf info <paquet> ; dnf repoquery <paquet>",
	"zypper":           "zypper repos ; zypper search <mot> ; zypper install <paquet> ; zypper remove <paquet>",
	"bash":             "bash ; bash <script> ; bash -x <script>",
	"PATH":             "printf '%s\\n' \"$PATH\" ; PATH=\"/opt/tools:$PATH\" ; command -v <commande>",
	"echo":             "echo \"$VAR\" ; echo -n <texte>",
	"env":              "env ; env VAR=valeur commande",
	"export":           "export VAR=valeur ; export VAR",
	"pwd":              "pwd ; pwd -P",
	"set":              "set ; set -o ; set -e",
	"unset":            "unset VAR",
	"type":             "type commande ; type -a commande",
	"which":            "which commande",
	"man":              "man commande ; man 5 fichier ; man -k mot",
	"uname":            "uname -a ; uname -r ; uname -m",
	"history":          "history ; history -c ; history -w ; history -r",
	"cat":              "cat fichier1 fichier2",
	"cut":              "cut -d: -f1 /etc/passwd ; cut -c1-10 fichier",
	"head":             "head -n 20 fichier",
	"tail":             "tail -n 20 fichier ; tail -f fichier",
	"less":             "less fichier",
	"nl":               "nl -ba fichier",
	"od":               "od -c fichier ; od -x fichier",
	"paste":            "paste fichier1 fichier2",
	"sed":              "sed 's/ancien/nouveau/g' fichier ; sed -n '1,10p' fichier",
	"sort":             "sort fichier ; sort -n fichier ; sort -k2 fichier",
	"split":            "split -l 100 fichier prefixe",
	"tr":               "tr '[:lower:]' '[:upper:]'",
	"uniq":             "sort fichier | uniq ; sort fichier | uniq -c",
	"wc":               "wc -l fichier ; wc -w fichier",
	"md5sum":           "md5sum fichier ; md5sum -c checksums",
	"sha256sum":        "sha256sum fichier ; sha256sum -c checksums",
	"sha512sum":        "sha512sum fichier ; sha512sum -c checksums",
	"zcat":             "zcat fichier.gz",
	"bzcat":            "bzcat fichier.bz2",
	"xzcat":            "xzcat fichier.xz",
	"cp":               "cp source destination ; cp -a répertoire destination",
	"mv":               "mv source destination",
	"ls":               "ls -la ; ls -li ; ls -lh",
	"rm":               "rm fichier ; rm -r répertoire",
	"mkdir":            "mkdir -p chemin",
	"rmdir":            "rmdir répertoire-vide",
	"touch":            "touch fichier",
	"find":             "find chemin -type f -name '*.log' ; find chemin -mtime +7 ; find chemin -exec commande {} \\; ; find / -perm -4000 -type f 2>/dev/null",
	"tar":              "tar -cf archive.tar fichiers ; tar -tf archive.tar ; tar -xf archive.tar ; tar -czf archive.tar.gz répertoire",
	"cpio":             "find . -print | cpio -o > archive.cpio ; cpio -id < archive.cpio",
	"dd":               "dd if=<source> of=<destination> bs=<taille> status=progress",
	"file":             "file chemin",
	"gzip":             "gzip fichier ; gzip -d fichier.gz",
	"gunzip":           "gunzip fichier.gz",
	"bzip2":            "bzip2 fichier ; bzip2 -d fichier.bz2",
	"bunzip2":          "bunzip2 fichier.bz2",
	"xz":               "xz fichier ; xz -d fichier.xz",
	"unxz":             "unxz fichier.xz",
	"<":                "commande < entrée",
	">":                "commande > sortie",
	">>":               "commande >> sortie",
	"2>":               "commande 2> erreurs",
	"2>&1":             "commande > tout.log 2>&1",
	"|":                "commande1 | commande2",
	"tee":              "commande | tee fichier ; commande | tee -a fichier",
	"xargs":            "printf '%s\\0' ... | xargs -0 commande",
	"bg":               "bg %<job>",
	"fg":               "fg %<job>",
	"jobs":             "jobs -l",
	"kill":             "kill <PID> ; kill -TERM <PID> ; kill -KILL <PID>",
	"nohup":            "nohup commande > sortie.log 2>&1 &",
	"ps":               "ps aux ; ps -ef ; ps -o pid,ppid,ni,stat,cmd",
	"top":              "top",
	"free":             "free -h",
	"uptime":           "uptime",
	"pgrep":            "pgrep -a <nom>",
	"pkill":            "pkill -TERM <nom>",
	"killall":          "killall -TERM <nom>",
	"watch":            "watch -n 2 commande",
	"screen":           "screen ; screen -ls ; screen -r <session>",
	"tmux":             "tmux new -s nom ; tmux ls ; tmux attach -t nom",
	"nice":             "nice -n 10 commande",
	"renice":           "renice 10 -p <PID>",
	"grep":             "grep motif fichier ; grep -E 'regex' fichier ; grep -r motif répertoire",
	"egrep":            "grep -E 'regex' fichier",
	"fgrep":            "grep -F 'texte littéral' fichier",
	"vi":               "vi fichier",
	"vim":              "vim fichier",
	"fdisk":            "fdisk -l ; fdisk <disque>",
	"gdisk":            "gdisk -l <disque> ; gdisk <disque>",
	"parted":           "parted <disque> print ; parted <disque> mklabel gpt",
	"mkfs":             "mkfs.ext4 <partition> ; mkfs.xfs <partition>",
	"mkswap":           "mkswap <partition> ; swapon <partition>",
	"du":               "du -sh chemin ; du -xhd1 chemin",
	"df":               "df -h ; df -i",
	"fsck":             "fsck -N <device> ; fsck <device>",
	"e2fsck":           "e2fsck -f <device>",
	"mke2fs":           "mke2fs -t ext4 <device>",
	"tune2fs":          "tune2fs -l <device>",
	"xfs_repair":       "xfs_repair <device>",
	"xfs_fsr":          "xfs_fsr <montage-ou-fichier>",
	"xfs_db":           "xfs_db -r <device>",
	"mount":            "mount <device> <point> ; mount -a",
	"umount":           "umount <point-ou-device>",
	"blkid":            "blkid ; blkid <device>",
	"lsblk":            "lsblk -f ; lsblk -o NAME,FSTYPE,UUID,MOUNTPOINTS",
	"chmod":            "chmod 640 fichier ; chmod u+x fichier ; chmod g+s répertoire ; chmod +t répertoire",
	"umask":            "umask ; umask 027",
	"chown":            "chown utilisateur:groupe chemin ; chown -R utilisateur:groupe répertoire",
	"chgrp":            "chgrp groupe chemin",
	"ln":               "ln source hardlink ; ln -s cible symlink",
	"locate":           "locate motif",
	"updatedb":         "updatedb",
	"whereis":          "whereis commande",
	".":                ". ./script-env.sh",
	"source":           "source ./script-env.sh",
	"for":              "for x in liste; do commande; done",
	"while":            "while condition; do commande; done",
	"test":             "test -f fichier ; [ -n \"$VAR\" ]",
	"if":               "if commande; then ...; elif ...; else ...; fi",
	"read":             "read -r variable",
	"seq":              "seq 1 10",
	"exec":             "exec commande",
	"&&":               "commande1 && commande2",
	"||":               "commande1 || commande2",
	"xauth":            "xauth list ; xauth info",
	"xhost":            "xhost ; xhost +SI:localuser:utilisateur",
	"useradd":          "useradd -m -s /bin/bash utilisateur",
	"usermod":          "usermod -aG groupe utilisateur ; usermod -L utilisateur",
	"userdel":          "userdel -r utilisateur",
	"groupadd":         "groupadd groupe",
	"groupmod":         "groupmod -n nouveau ancien",
	"groupdel":         "groupdel groupe",
	"passwd":           "passwd utilisateur ; passwd -l utilisateur ; passwd -S utilisateur",
	"chage":            "chage -l utilisateur ; chage -M 90 utilisateur",
	"getent":           "getent passwd utilisateur ; getent hosts nom",
	"crontab":          "crontab -e ; crontab -l ; crontab -r",
	"at":               "echo 'commande' | at 23:00",
	"atq":              "atq",
	"atrm":             "atrm <job>",
	"systemd-run":      "systemd-run --on-calendar='...' commande",
	"locale":           "locale ; locale -a",
	"tzselect":         "tzselect",
	"timedatectl":      "timedatectl ; timedatectl set-timezone Europe/Paris",
	"date":             "date ; date -u ; date '+%F %T %z'",
	"iconv":            "iconv -f ISO-8859-1 -t UTF-8 entrée > sortie",
	"hwclock":          "hwclock --show ; hwclock --systohc ; hwclock --hctosys",
	"ntpd":             "ntpd -qg",
	"ntpdate":          "ntpdate <serveur>",
	"chronyc":          "chronyc tracking ; chronyc sources -v",
	"logger":           "logger -p user.notice 'message'",
	"logrotate":        "logrotate -d /etc/logrotate.conf ; logrotate -f <config>",
	"systemd-cat":      "commande | systemd-cat -t tag",
	"sendmail":         "printf 'Subject: test\\n\\nbody\\n' | sendmail utilisateur",
	"newaliases":       "newaliases",
	"mail":             "echo corps | mail -s sujet utilisateur",
	"mailq":            "mailq",
	"lpr":              "lpr -P imprimante fichier",
	"lpq":              "lpq -P imprimante",
	"lprm":             "lprm -P imprimante <job>",
	"nmcli":            "nmcli connection show ; nmcli device status ; nmcli con mod <nom> ipv4.addresses ...",
	"hostnamectl":      "hostnamectl ; hostnamectl set-hostname nom",
	"ifup":             "ifup <interface>",
	"ifdown":           "ifdown <interface>",
	"ip":               "ip link ; ip addr ; ip route ; ip neigh",
	"ss":               "ss -lntup ; ss -tan",
	"ping":             "ping -c 4 <adresse>",
	"ping6":            "ping -6 -c 4 <adresse>",
	"traceroute":       "traceroute <hôte>",
	"traceroute6":      "traceroute -6 <hôte>",
	"tracepath":        "tracepath <hôte>",
	"tracepath6":       "tracepath -6 <hôte>",
	"netcat":           "nc -vz <hôte> <port> ; nc -l <port>",
	"ifconfig":         "ifconfig -a",
	"netstat":          "netstat -lntup ; netstat -rn",
	"route":            "route -n",
	"host":             "host nom ; host -t MX domaine",
	"dig":              "dig nom A ; dig domaine MX ; dig @serveur nom",
	"fuser":            "fuser -v fichier ; fuser -n tcp 22",
	"lsof":             "lsof fichier ; lsof -i :22",
	"nmap":             "nmap -sT <hôte> ; nmap -p 22,80 <hôte>",
	"sudo":             "sudo -l ; sudo commande ; sudo -u utilisateur commande",
	"su":               "su - utilisateur ; su -c 'commande' utilisateur",
	"ulimit":           "ulimit -a ; ulimit -n",
	"who":              "who",
	"w":                "w",
	"last":             "last",
	"ssh":              "ssh utilisateur@hôte ; ssh -L 8080:cible:80 hôte ; ssh -R 2222:localhost:22 hôte ; ssh -D 1080 hôte",
	"ssh-keygen":       "ssh-keygen -t ed25519 ; ssh-keygen -lf <clé.pub>",
	"ssh-agent":        "eval \"$(ssh-agent)\"",
	"ssh-add":          "ssh-add <clé> ; ssh-add -l",
	"gpg":              "gpg --gen-key ; gpg --encrypt -r destinataire fichier ; gpg --decrypt fichier.gpg ; gpg --detach-sign fichier ; gpg --verify fichier.sig fichier",
	"gpg-agent":        "gpgconf --launch gpg-agent",
}

func hasSpecificStandaloneTermExplanation(term, objectiveID string) bool {
	if _, exists := standaloneTermExplanations[term]; exists {
		return true
	}
	if objectiveID == "109.1" {
		_, exists := standalonePortServices[term]
		return exists
	}
	return false
}

func standaloneTermExplanation(term, objectiveID string) string {
	if explanation, exists := standaloneTermExplanations[term]; exists {
		return explanation
	}
	if strings.HasPrefix(term, "/") || strings.HasPrefix(term, "~/") {
		return "chemin examinable de cet objectif : sache ce qu'il contient, qui le lit ou l'écrit et comment l'inspecter sans le modifier inutilement"
	}
	if _, ok := standalonePortServices[term]; ok && objectiveID == "109.1" {
		return standalonePortServices[term]
	}
	return "terme ou utilitaire explicitement examinable : sache le reconnaître, expliquer son rôle dans cet objectif et l'utiliser ou l'inspecter dans le contexte pratique associé"
}

var standalonePortServices = map[string]string{
	"20": "FTP data (TCP)", "21": "FTP control (TCP)", "22": "SSH (TCP)",
	"23": "Telnet (TCP)", "25": "SMTP (TCP)", "53": "DNS (UDP/TCP)",
	"80": "HTTP (TCP)", "110": "POP3 (TCP)", "123": "NTP (UDP)",
	"139": "NetBIOS session service (TCP)", "143": "IMAP (TCP)",
	"161": "SNMP queries (UDP)", "162": "SNMP traps (UDP)",
	"389": "LDAP (TCP/UDP selon usage)", "443": "HTTPS (TCP)",
	"465": "SMTPS implicite (TCP)", "514": "syslog traditionnel (UDP/TCP selon implémentation)",
	"636": "LDAPS (TCP)", "993": "IMAPS (TCP)", "995": "POP3S (TCP)",
}

var standaloneTermExplanations = map[string]string{
	"/proc":                      "pseudo-filesystem exposant des informations noyau et processus",
	"/sys":                       "sysfs : vue structurée des périphériques et objets du noyau",
	"/dev":                       "nœuds de périphériques utilisés par l'espace utilisateur",
	"modprobe":                   "charge ou retire un module noyau en gérant ses dépendances",
	"lsmod":                      "liste les modules noyau actuellement chargés",
	"lspci":                      "énumère les périphériques du bus PCI et leurs identifiants",
	"lsusb":                      "énumère les périphériques USB et leurs identifiants",
	"sysfs":                      "interface noyau sous /sys décrivant périphériques, pilotes et attributs",
	"udev":                       "gestionnaire userspace des événements de périphériques et de /dev",
	"D-Bus":                      "bus de messages userspace utilisé par de nombreux services pour exposer événements et API",
	"dmesg":                      "affiche le ring buffer du noyau, utile notamment pour le boot et le matériel",
	"journalctl":                 "interroge le journal systemd avec filtres de boot, unité, temps et priorité",
	"BIOS":                       "firmware historique initialisant la machine avant le bootloader",
	"UEFI":                       "firmware moderne utilisant notamment une EFI System Partition",
	"bootloader":                 "programme qui choisit/charge le noyau et lui transmet ses paramètres",
	"kernel":                     "noyau Linux chargé par le bootloader",
	"initramfs":                  "filesystem initial en RAM utilisé avant le montage du système racine définitif",
	"init":                       "processus PID 1 / mécanisme historique de changement de runlevel selon contexte",
	"SysVinit":                   "système d'init historique basé sur runlevels et scripts /etc/init.d",
	"systemd":                    "init et gestionnaire de services/units utilisé sur la majorité des distributions modernes",
	"Upstart":                    "init événementiel historique à reconnaître pour l'examen",
	"shutdown":                   "programme un arrêt ou redémarrage ordonné et peut prévenir les utilisateurs",
	"telinit":                    "demande historiquement à init de changer de runlevel",
	"systemctl":                  "contrôle unités, services, targets et état systemd",
	"wall":                       "diffuse un message aux terminaux des utilisateurs connectés",
	"acpid":                      "démon historique traitant des événements ACPI",
	"ESP":                        "EFI System Partition contenant des exécutables de démarrage UEFI",
	"swap":                       "espace bloc utilisé comme extension de mémoire et parfois pour l'hibernation",
	"LVM":                        "couche volumes physiques → groupes de volumes → volumes logiques permettant un stockage flexible",
	"grub-install":               "installe les composants de GRUB pour une cible de démarrage",
	"grub-mkconfig":              "génère un grub.cfg à partir des sources de configuration",
	"grub.cfg":                   "configuration générée de GRUB 2, généralement à ne pas éditer comme source principale",
	"menu.lst":                   "fichier de configuration courant de GRUB Legacy",
	"grub.conf":                  "nom historique de configuration GRUB selon distribution",
	"GRUB Legacy":                "ancienne génération de GRUB, examinable même si obsolète",
	"GRUB 2":                     "génération actuelle de GRUB utilisée sur les systèmes Linux modernes",
	"MBR":                        "secteur/format de partitionnement historique associé au boot BIOS",
	"ldd":                        "affiche les bibliothèques dynamiques qu'un exécutable résout",
	"ldconfig":                   "met à jour le cache/liens du chargeur dynamique à partir des chemins configurés",
	"LD_LIBRARY_PATH":            "variable modifiant les chemins de recherche de bibliothèques pour un processus",
	"dpkg":                       "outil bas niveau de la base et des paquets Debian .deb",
	"dpkg-reconfigure":           "relance la configuration d'un paquet Debian déjà installé",
	"apt":                        "interface APT interactive moderne pour dépôts, dépendances et paquets",
	"apt-get":                    "interface APT scriptable/historique pour installation, mise à jour et suppression",
	"apt-cache":                  "interroge métadonnées et dépendances disponibles dans APT",
	"rpm":                        "outil et base de paquets RPM pour requêtes, installation et vérification",
	"rpm2cpio":                   "convertit un RPM en flux cpio pour inspecter son contenu",
	"yum":                        "gestionnaire haut niveau RPM historique encore examinable",
	"dnf":                        "gestionnaire haut niveau RPM moderne sur Fedora/RHEL",
	"zypper":                     "gestionnaire de paquets et dépôts d'openSUSE",
	"bash":                       "shell GNU utilisé pour interprétation, scripts, expansions, variables et historique",
	"PATH":                       "variable d'environnement contenant la liste ordonnée de répertoires où le shell recherche les commandes sans chemin explicite",
	"echo":                       "affiche ses arguments après les expansions réalisées par le shell",
	"env":                        "affiche l'environnement ou lance une commande avec un environnement ajusté",
	"export":                     "marque une variable shell pour héritage par les processus enfants",
	"pwd":                        "affiche le répertoire courant",
	"set":                        "affiche ou modifie variables/options/paramètres du shell selon son usage",
	"unset":                      "supprime une variable ou fonction du shell",
	"type":                       "indique comment le shell résout un nom : builtin, fonction, alias ou fichier",
	"which":                      "recherche typiquement un exécutable selon PATH, moins complet que type pour les constructions shell",
	"man":                        "consulte la documentation locale structurée en sections",
	"uname":                      "affiche des informations noyau/système selon ses options",
	"history":                    "affiche et manipule l'historique du shell Bash",
	".bash_history":              "fichier persistant historique utilisé par Bash pour les commandes enregistrées",
	"cat":                        "concatène des fichiers vers stdout",
	"cut":                        "extrait champs ou positions de lignes",
	"head":                       "affiche le début d'un flux",
	"tail":                       "affiche la fin d'un flux et peut suivre un fichier",
	"less":                       "pager interactif pour lire un flux sans le charger entièrement à l'écran",
	"nl":                         "numérote les lignes",
	"od":                         "affiche une représentation octale/hexadécimale ou typée des octets",
	"paste":                      "assemble des lignes de fichiers côte à côte",
	"sed":                        "éditeur de flux pour sélection, substitution et transformation",
	"sort":                       "trie des lignes selon locale/options",
	"split":                      "découpe un fichier en morceaux",
	"tr":                         "traduit ou supprime des caractères",
	"uniq":                       "filtre/compte les lignes adjacentes identiques",
	"wc":                         "compte lignes, mots ou octets/caractères",
	"md5sum":                     "calcule une empreinte MD5, utile pour intégrité non cryptographique moderne",
	"sha256sum":                  "calcule/vérifie une empreinte SHA-256",
	"sha512sum":                  "calcule/vérifie une empreinte SHA-512",
	"zcat":                       "décompresse gzip vers stdout",
	"bzcat":                      "décompresse bzip2 vers stdout",
	"xzcat":                      "décompresse xz vers stdout",
	"cp":                         "copie fichiers/répertoires selon options",
	"mv":                         "déplace ou renomme",
	"mkdir":                      "crée des répertoires",
	"rm":                         "supprime des entrées; les options récursives exigent une grande prudence",
	"rmdir":                      "supprime un répertoire vide",
	"touch":                      "crée un fichier vide ou met à jour ses timestamps",
	"find":                       "parcourt une arborescence, filtre selon attributs et peut appliquer des actions",
	"tar":                        "crée/extrait des archives et peut les combiner avec compression",
	"cpio":                       "archive/extrait via un flux de noms/fichiers",
	"dd":                         "selon le contexte, commande de copie bloc par bloc ou, dans vi, suppression de la ligne courante",
	"file":                       "détecte le type de contenu via signatures/magic",
	"gzip":                       "compression gzip",
	"gunzip":                     "décompression gzip",
	"bzip2":                      "compression bzip2",
	"bunzip2":                    "décompression bzip2",
	"xz":                         "compression xz/LZMA",
	"unxz":                       "décompression xz",
	"stdin":                      "entrée standard, descripteur 0 par convention",
	"stdout":                     "sortie standard, descripteur 1 par convention",
	"stderr":                     "sortie d'erreur standard, descripteur 2 par convention",
	"<":                          "redirige stdin depuis un fichier",
	">":                          "redirige stdout vers un fichier en l'écrasant",
	">>":                         "redirige stdout en ajout à la fin",
	"2>":                         "redirige le descripteur 2 stderr",
	"2>&1":                       "duplique stderr vers la destination actuelle de stdout; l'ordre des redirections compte",
	"|":                          "connecte stdout de la commande gauche à stdin de la commande droite",
	"tee":                        "duplique stdin vers stdout et un ou plusieurs fichiers",
	"xargs":                      "construit des arguments de commande à partir de stdin; les séparateurs doivent être gérés prudemment",
	"bg":                         "reprend un job suspendu en arrière-plan",
	"fg":                         "ramène un job au premier plan",
	"jobs":                       "liste les jobs suivis par le shell courant",
	"kill":                       "envoie un signal à un PID; SIGTERM est le défaut habituel",
	"nohup":                      "ignore SIGHUP pour aider un processus à survivre à la fermeture de session",
	"ps":                         "instantané des processus et attributs sélectionnés",
	"top":                        "vue interactive des processus et ressources",
	"free":                       "résume mémoire et swap",
	"uptime":                     "affiche durée de fonctionnement et load averages",
	"pgrep":                      "sélectionne des PID par nom/critères",
	"pkill":                      "envoie un signal aux processus correspondant à des critères",
	"killall":                    "envoie un signal aux processus portant un nom donné selon implémentation",
	"watch":                      "réexécute périodiquement une commande et affiche sa sortie",
	"screen":                     "multiplexeur de terminal historique",
	"tmux":                       "multiplexeur de terminal moderne permettant sessions détachables",
	"nice":                       "démarre une commande avec une valeur nice ajustée",
	"renice":                     "modifie la valeur nice d'un processus existant",
	"grep":                       "sélectionne des lignes correspondant à une expression",
	"egrep":                      "forme historique de grep -E pour ERE",
	"fgrep":                      "forme historique de grep -F pour chaînes littérales",
	"BRE":                        "Basic Regular Expressions",
	"ERE":                        "Extended Regular Expressions",
	"vi":                         "éditeur modal POSIX au cœur de l'objectif 103.8",
	"vim":                        "implémentation étendue compatible vi",
	"nano":                       "éditeur texte non modal à reconnaître",
	"emacs":                      "famille d'éditeurs extensibles à reconnaître",
	"EDITOR":                     "variable indiquant l'éditeur préféré à de nombreux programmes",
	"h":                          "vi: déplacement d'un caractère à gauche en mode normal",
	"j":                          "vi: déplacement d'une ligne vers le bas",
	"k":                          "vi: déplacement d'une ligne vers le haut",
	"l":                          "vi: déplacement d'un caractère à droite",
	"i":                          "vi: entre en insertion avant le curseur",
	"a":                          "vi: entre en insertion après le curseur",
	"o":                          "vi: ouvre une nouvelle ligne puis passe en insertion",
	"d":                          "vi: opérateur de suppression",
	"p":                          "vi: colle après le curseur",
	"y":                          "vi: opérateur de copie (yank)",
	"yy":                         "vi: copie la ligne courante",
	"ZZ":                         "vi: sauvegarde et quitte si nécessaire",
	":w!":                        "vi: force l'écriture lorsque les permissions/conditions de vi le permettent",
	":q!":                        "vi: quitte en abandonnant les modifications non sauvegardées",
	"fdisk":                      "outil de partitionnement interactif couramment utilisé pour MBR/GPT",
	"gdisk":                      "outil de partitionnement orienté GPT",
	"parted":                     "outil de partitionnement scriptable/interactif prenant en charge plusieurs tables",
	"mkfs":                       "interface/famille d'outils de création de filesystem",
	"mkswap":                     "initialise une zone de swap",
	"du":                         "estime l'espace occupé par fichiers et répertoires",
	"df":                         "rapporte l'utilisation des filesystems montés, blocs et éventuellement inodes",
	"fsck":                       "frontend de vérification de filesystem, à utiliser avec les contraintes du type de FS",
	"e2fsck":                     "vérifie/répare ext2/3/4 hors conditions dangereuses",
	"tune2fs":                    "inspecte/modifie paramètres des filesystems ext",
	"xfs_repair":                 "répare un filesystem XFS hors ligne selon conditions",
	"mount":                      "monte un filesystem sur un point de montage",
	"umount":                     "démonte un filesystem",
	"blkid":                      "identifie UUID, labels et types de volumes bloc",
	"lsblk":                      "présente les périphériques bloc et leur topologie",
	"chmod":                      "modifie les bits de mode/permissions",
	"umask":                      "masque des bits lors de la création de fichiers/répertoires",
	"chown":                      "change propriétaire et éventuellement groupe",
	"chgrp":                      "change le groupe",
	"SUID":                       "bit spécial faisant prendre à un exécutable l'UID effectif de son propriétaire",
	"SGID":                       "bit spécial affectant GID effectif d'un exécutable ou héritage de groupe d'un répertoire",
	"sticky bit":                 "sur répertoire partagé, limite la suppression/renommage aux propriétaires appropriés",
	"ln":                         "crée des hard links ou des liens symboliques avec -s",
	"locate":                     "cherche dans une base indexée, potentiellement obsolète",
	"updatedb":                   "met à jour la base utilisée par locate",
	"whereis":                    "localise typiquement binaire, source et pages man",
	"source":                     "exécute un fichier dans le shell courant, comme le builtin .",
	"read":                       "lit une ligne/valeurs depuis stdin dans des variables shell",
	"seq":                        "génère une séquence numérique souvent utilisée dans des boucles simples",
	"exec":                       "remplace le processus shell courant par une commande",
	"xauth":                      "gère les cookies d'autorisation X11",
	"xhost":                      "contrôle d'accès X11 par hôte/utilisateur, plus grossier que xauth",
	"DISPLAY":                    "variable identifiant le serveur/display X11 cible",
	"XDMCP":                      "protocole historique de sessions X distantes",
	"VNC":                        "famille de protocoles de bureau distant orientés framebuffer",
	"SPICE":                      "protocole d'affichage distant fréquent en virtualisation",
	"RDP":                        "Remote Desktop Protocol",
	"getent":                     "interroge les bases NSS comme passwd, group ou hosts via la résolution système",
	"useradd":                    "crée un compte utilisateur selon options et valeurs par défaut",
	"usermod":                    "modifie un compte existant",
	"userdel":                    "supprime un compte",
	"groupadd":                   "crée un groupe",
	"groupmod":                   "modifie un groupe",
	"groupdel":                   "supprime un groupe",
	"passwd":                     "gère le mot de passe/verrouillage selon options",
	"chage":                      "inspecte/modifie l'expiration de mot de passe",
	"crontab":                    "installe/liste/édite les tâches cron d'un utilisateur",
	"at":                         "programme une commande à exécution unique",
	"atq":                        "liste la file at",
	"atrm":                       "supprime un job at",
	"systemd-run":                "crée des unités/transient jobs, y compris timers selon options",
	"LC_ALL":                     "surcharge toutes les catégories de locale",
	"LANG":                       "locale par défaut utilisée lorsqu'une catégorie LC_* n'est pas plus spécifique",
	"TZ":                         "variable pouvant sélectionner un fuseau horaire pour un processus",
	"locale":                     "affiche les réglages/catégories de locale",
	"tzselect":                   "assistant shell pour choisir un identifiant de fuseau",
	"timedatectl":                "inspecte/configure heure, timezone et intégration de synchronisation systemd",
	"date":                       "affiche ou règle date/heure selon privilèges et options",
	"iconv":                      "convertit du texte entre encodages",
	"hwclock":                    "lit ou synchronise l'horloge matérielle RTC",
	"ntpd":                       "démon NTP classique",
	"ntpdate":                    "outil historique de réglage ponctuel NTP",
	"chronyc":                    "client d'inspection/contrôle de chronyd",
	"pool.ntp.org":               "service de pools DNS de serveurs NTP publics",
	"logger":                     "envoie un message au système de journalisation",
	"logrotate":                  "fait tourner, compresse et conserve des fichiers de logs selon règles",
	"systemd-cat":                "connecte la sortie d'une commande au journal systemd",
	"syslog-ng":                  "implémentation syslog alternative à connaître",
	"sendmail":                   "interface MTA historique/compatible pour soumettre un message",
	"newaliases":                 "reconstruit la base d'aliases du MTA compatible",
	"mail":                       "client mail en ligne de commande pour courrier local selon implémentation",
	"mailq":                      "affiche la file d'attente MTA via interface compatible",
	"postfix":                    "MTA courant",
	"exim":                       "MTA courant notamment sur certains systèmes Debian",
	"lpr":                        "soumet un job d'impression via interface compatible BSD",
	"lpq":                        "inspecte une file d'impression",
	"lprm":                       "annule un job d'impression",
	"IPv4":                       "adressage IP 32 bits",
	"IPv6":                       "adressage IP 128 bits avec notation hexadécimale, compression et types comme link-local/multicast",
	"CIDR":                       "notation préfixe /n définissant la partie réseau",
	"TCP":                        "transport orienté connexion avec fiabilité/ordre",
	"UDP":                        "transport datagramme sans connexion ni garantie de livraison",
	"ICMP":                       "messages de contrôle/diagnostic IP, notamment utilisés par ping",
	"nmcli":                      "CLI de NetworkManager pour profils et état réseau",
	"hostnamectl":                "inspecte/modifie le hostname via systemd-hostnamed",
	"ifup":                       "outil historique pour activer une interface selon configuration distribution",
	"ifdown":                     "outil historique pour désactiver une interface",
	"ip":                         "outil iproute2 pour liens, adresses, routes et voisins",
	"ss":                         "inspecte sockets et connexions, remplaçant moderne de nombreux usages netstat",
	"ping":                       "envoie des requêtes ICMP Echo IPv4",
	"ping6":                      "forme historique/dédiée IPv6 de ping selon distribution",
	"traceroute":                 "sonde le chemin IP via TTL/hop-limit",
	"tracepath":                  "observe le chemin et souvent le PMTU sans options privilégiées",
	"netcat":                     "outil générique de connexion/écoute TCP/UDP",
	"ifconfig":                   "outil net-tools historique pour interfaces",
	"netstat":                    "outil net-tools historique pour connexions/routes/statistiques",
	"route":                      "outil net-tools historique de table de routage",
	"host":                       "client DNS simple pour requêtes de noms",
	"dig":                        "client DNS détaillé pour interroger types, serveurs et sections de réponse",
	"fuser":                      "identifie des processus utilisant un fichier/socket et peut leur envoyer des signaux",
	"lsof":                       "liste les fichiers ouverts, sockets inclus",
	"nmap":                       "scanner réseau/ports à utiliser uniquement dans des environnements autorisés",
	"sudo":                       "exécute une commande selon une politique sudoers",
	"su":                         "change d'identité utilisateur via authentification selon politique",
	"ulimit":                     "builtin shell pour inspecter/modifier des limites de ressources du processus",
	"who":                        "liste des sessions connectées",
	"w":                          "sessions connectées avec activité/charge",
	"last":                       "historique de connexions à partir de la base wtmp",
	"ssh":                        "client OpenSSH pour connexion et tunnels",
	"ssh-keygen":                 "génère/inspecte des clés SSH",
	"ssh-agent":                  "agent conservant des clés privées déverrouillées en mémoire",
	"ssh-add":                    "ajoute/liste/retire des identités dans ssh-agent",
	"gpg":                        "outil GnuPG pour clés, chiffrement, signatures et vérification",
	"gpg-agent":                  "agent GnuPG pour opérations nécessitant des secrets",
	"/etc/inittab":               "configuration historique SysV décrivant notamment init/runlevels sur les systèmes qui l'utilisent",
	"/etc/init.d":                "répertoire historique de scripts de services SysVinit",
	"/etc/systemd":               "configuration systemd administrateur, unités et drop-ins locaux selon sous-répertoire",
	"/usr/lib/systemd":           "unités et composants systemd fournis par les paquets de la distribution",
	"/":                          "racine unique de l'arborescence Unix/Linux",
	"/var":                       "données variables : logs, spools, caches et états susceptibles de croître",
	"/home":                      "répertoire parent conventionnel des homes utilisateurs",
	"/boot":                      "fichiers nécessaires au démarrage tels que noyaux, initramfs et éléments du bootloader",
	"mount points":               "répertoires où des filesystems sont attachés à l'arborescence",
	"partitions":                 "plages définies dans une table de partitions et utilisables comme volumes bloc",
	"/etc/ld.so.conf":            "configuration des chemins additionnels du chargeur dynamique, souvent complétée par ld.so.conf.d",
	"shared libraries":           "bibliothèques chargées dynamiquement et partagées entre programmes",
	"/etc/apt/sources.list":      "source historique principale de dépôts APT, complétée par sources.list.d",
	"/etc/yum.conf":              "configuration globale historique de YUM/DNF selon distribution",
	"/etc/yum.repos.d":           "répertoire des définitions de dépôts YUM/DNF",
	"virtual machine":            "machine complète virtualisée avec matériel virtuel et noyau invité propre",
	"Linux container":            "environnement isolé partageant le noyau Linux de l'hôte",
	"application container":      "conteneur centré sur l'exécution empaquetée d'une application et de ses dépendances",
	"guest drivers":              "pilotes/outils améliorant l'intégration d'une VM avec son hyperviseur",
	"SSH host keys":              "clés identifiant cryptographiquement un serveur SSH; elles doivent être uniques après clonage",
	"D-Bus machine-id":           "identifiant machine userspace qui doit être régénéré lorsqu'une image clonée l'exige",
	"cloud-init":                 "mécanisme d'initialisation d'instances cloud à partir de metadata/user-data",
	"IaaS":                       "Infrastructure as a Service : VM, volumes et réseau fournis comme ressources cloud",
	"quoting":                    "règles shell qui contrôlent séparation des mots, globbing et expansions via quotes/escaping",
	"ls":                         "liste fichiers/répertoires et, avec options, métadonnées comme modes, propriétaires et inodes",
	"globbing":                   "expansion par le shell de motifs comme *, ? et [] avant l'exécution d'une commande",
	"&":                          "opérateur shell lançant une pipeline/commande en arrière-plan",
	"regex(7)":                   "page de manuel décrivant la syntaxe et la sémantique générales des expressions régulières",
	"?":                          "dans vi, démarre une recherche vers l'arrière; le sens dépend du contexte shell/éditeur",
	"ext2":                       "filesystem ext historique non journalisé",
	"ext3":                       "évolution ext2 avec journalisation",
	"ext4":                       "filesystem ext moderne avec journalisation et nombreuses améliorations",
	"XFS":                        "filesystem journalisé haute performance avec outils d'administration dédiés",
	"VFAT":                       "filesystem FAT compatible largement avec d'autres systèmes, sans permissions Unix natives complètes",
	"exFAT":                      "filesystem Microsoft adapté aux supports amovibles et gros fichiers",
	"Btrfs":                      "filesystem copy-on-write avec sous-volumes, compression et fonctions multi-device",
	"mke2fs":                     "outil bas niveau de création des filesystems ext2/3/4, souvent invoqué via mkfs.ext*",
	"xfs_fsr":                    "outil de réorganisation/défragmentation de fichiers XFS",
	"xfs_db":                     "outil d'inspection et diagnostic bas niveau des métadonnées XFS",
	"/etc/fstab":                 "table persistante des filesystems, points de montage et options",
	"/media":                     "emplacement FHS courant pour montages de médias amovibles",
	"UUID":                       "identifiant persistant de volume préféré aux noms de périphériques instables dans de nombreux montages",
	"labels":                     "noms symboliques de filesystem utilisables pour identifier un volume",
	"systemd mount units":        "unités .mount représentant des montages dans le graphe de dépendances systemd",
	"inode":                      "structure de métadonnées d'un fichier Unix; plusieurs hard links peuvent référencer le même inode",
	"hard link":                  "entrée de répertoire supplémentaire pointant vers le même inode",
	"symbolic link":              "fichier spécial contenant un chemin vers une cible, qui peut devenir cassé",
	"/etc/updatedb.conf":         "configuration contrôlant les chemins/filesystems inclus ou exclus de la base locate",
	"FHS":                        "Filesystem Hierarchy Standard définissant le rôle des grands répertoires Linux/Unix",
	".":                          "builtin shell équivalent à source pour exécuter un fichier dans le shell courant",
	"/etc/bash.bashrc":           "configuration interactive Bash globale sur les distributions qui l'utilisent",
	"/etc/profile":               "profil global généralement lu par les login shells compatibles",
	"~/.bash_profile":            "profil utilisateur Bash de login, prioritaire sur certains autres fichiers de login",
	"~/.bash_login":              "fichier de login Bash consulté si .bash_profile est absent",
	"~/.profile":                 "profil utilisateur générique lu par de nombreux shells de login; Bash l'utilise si ses profils spécifiques sont absents",
	"~/.bashrc":                  "configuration utilisateur des shells Bash interactifs non-login, souvent sourcée depuis le profil",
	"~/.bash_logout":             "commandes Bash exécutées lors de la sortie d'un login shell",
	"function":                   "fonction shell nommée exécutée dans le shell courant et pouvant réutiliser variables/paramètres",
	"alias":                      "substitution textuelle de commande gérée par le shell interactif",
	"/etc/skel":                  "modèle de fichiers recopiés dans le home lors de la création de nouveaux utilisateurs",
	"for":                        "boucle shell itérant sur une liste de mots/valeurs",
	"while":                      "boucle shell répétée tant qu'une commande-condition réussit",
	"test":                       "commande/builtin évaluant expressions sur chaînes, nombres et fichiers; [ est une forme courante",
	"if":                         "construction conditionnelle shell pilotée par le statut de commandes",
	"||":                         "exécute la commande/pipeline droite si celle de gauche échoue",
	"&&":                         "exécute la commande/pipeline droite si celle de gauche réussit",
	"shebang":                    "ligne #! au début d'un script indiquant l'interpréteur à lancer lors d'une exécution directe",
	"exit status":                "code 0 pour succès conventionnel et non-zéro pour erreur/condition, utilisé par if, && et ||",
	"/etc/X11/xorg.conf":         "configuration monolithique historique du serveur Xorg",
	"/etc/X11/xorg.conf.d":       "répertoire de fragments de configuration Xorg",
	"~/.xsession-errors":         "fichier historique pouvant contenir sorties/erreurs de session X selon distribution/display manager",
	"X11":                        "protocole/architecture graphique réseau client-serveur historique de l'écosystème Unix",
	"Wayland":                    "architecture graphique moderne où un compositor gère directement fenêtres, entrées et affichage",
	"KDE":                        "écosystème de bureau dont Plasma est l'environnement graphique principal",
	"GNOME":                      "environnement de bureau Linux majeur fondé notamment sur GTK",
	"Xfce":                       "environnement de bureau léger et modulaire",
	"screen reader":              "technologie lisant vocalement l'interface pour utilisateurs aveugles ou malvoyants",
	"Braille display":            "afficheur braille rafraîchissable présentant du texte tactile",
	"screen magnifier":           "agrandit une zone ou l'ensemble de l'affichage",
	"on-screen keyboard":         "clavier logiciel utilisable sans clavier physique",
	"sticky keys":                "permet de saisir des combinaisons de modificateurs séquentiellement",
	"repeat keys":                "réglages contrôlant répétition d'une touche maintenue",
	"slow keys":                  "ignore les frappes trop brèves et exige un maintien minimal",
	"bounce keys":                "ignore les répétitions très rapprochées involontaires",
	"toggle keys":                "produit un retour lors de l'activation/désactivation de touches d'état comme Caps Lock",
	"mouse keys":                 "permet de contrôler le pointeur avec le clavier",
	"gestures":                   "gestes tactiles/pointeur pouvant déclencher des actions d'accessibilité ou de bureau",
	"speech recognition":         "contrôle ou saisie via reconnaissance de la parole",
	"/etc/passwd":                "base publique des comptes : nom, UID, GID, home et shell; le hash moderne n'y réside pas",
	"/etc/shadow":                "hashes de mots de passe et paramètres d'expiration protégés",
	"/etc/group":                 "définitions des groupes locaux et membres supplémentaires",
	"cron":                       "démon/système de planification récurrente selon calendrier",
	"/etc/cron.*":                "répertoires/fichiers système de tâches périodiques selon la distribution",
	"at.allow":                   "liste optionnelle des utilisateurs autorisés à utiliser at",
	"at.deny":                    "liste optionnelle des utilisateurs interdits d'utiliser at lorsque allow n'est pas utilisé",
	"cron.allow":                 "liste optionnelle des utilisateurs autorisés à utiliser cron/crontab",
	"cron.deny":                  "liste optionnelle des utilisateurs interdits d'utiliser cron/crontab selon politique",
	"systemd timers":             "unités .timer déclenchant des unités systemd selon calendrier ou délai",
	"/etc/timezone":              "fichier de timezone utilisé notamment sur Debian et dérivés",
	"/etc/localtime":             "fichier ou symlink représentant le fuseau horaire système",
	"/usr/share/zoneinfo":        "base installée des règles de fuseaux horaires IANA",
	"LC_*":                       "famille de variables de locale par catégorie, par exemple LC_COLLATE ou LC_TIME",
	"UTF-8":                      "encodage Unicode à longueur variable dominant sur Linux moderne",
	"ISO-8859":                   "famille historique d'encodages 8 bits régionaux",
	"ASCII":                      "jeu/encodage 7 bits historique, sous-ensemble d'UTF-8 pour les caractères de base",
	"Unicode":                    "répertoire universel de points de code; UTF-8 est un encodage de ces caractères",
	"/etc/ntp.conf":              "configuration classique de ntpd",
	"/etc/chrony.conf":           "configuration principale de chronyd sur de nombreuses distributions",
	"/etc/rsyslog.conf":          "configuration principale de rsyslog : règles, modules, sources et destinations",
	"/var/log":                   "répertoire FHS principal des fichiers de logs persistants",
	"/etc/logrotate.conf":        "configuration globale de logrotate",
	"/etc/logrotate.d":           "fragments de configuration logrotate fournis par services/paquets",
	"/etc/systemd/journald.conf": "configuration de journald, notamment stockage et limites",
	"/var/log/journal":           "emplacement qui rend généralement le journal systemd persistant",
	"~/.forward":                 "fichier utilisateur historique demandant au MTA de transférer son courrier",
	"/etc/cups":                  "configuration du serveur/client CUPS selon fichiers et distribution",
	"CUPS":                       "Common UNIX Printing System, architecture de files, imprimantes et jobs d'impression",
	"/etc/services":              "base locale associant noms de services à ports et protocoles bien connus",
	"subnet":                     "sous-réseau défini par une adresse réseau et un préfixe/masque",
	"/etc/hostname":              "nom d'hôte persistant sur de nombreuses distributions",
	"/etc/hosts":                 "mappings locaux statiques noms↔adresses consultés selon NSS",
	"/etc/nsswitch.conf":         "ordre/sources NSS pour passwd, group, hosts et autres bases",
	"/etc/resolv.conf":           "configuration du resolver DNS libc : nameserver, search et options; parfois générée",
	"NetworkManager":             "service de gestion réseau par profils, contrôlable notamment avec nmcli",
	"systemd-networkd":           "service systemd de configuration réseau déclarative via fichiers .network/.netdev",
	"hostname":                   "commande/valeur représentant le nom d'hôte; hostnamectl est souvent préféré pour la persistance systemd",
	"traceroute6":                "variante/alias historique pour tracer le chemin IPv6",
	"tracepath6":                 "variante IPv6 de tracepath",
	"systemd-resolved":           "service de résolution/cache DNS systemd pouvant fournir un stub resolver local",
	"/etc/sudoers":               "politique principale sudo; doit être éditée avec des outils sûrs comme visudo",
	"/etc/nologin":               "fichier dont la présence peut bloquer les connexions de comptes non-root selon les programmes",
	"/etc/xinetd.d":              "répertoire historique de définitions de services xinetd",
	"/etc/xinetd.conf":           "configuration principale historique de xinetd",
	"systemd.socket":             "type d'unité systemd pouvant écouter un socket et activer le service associé",
	"/etc/hosts.allow":           "politique historique TCP wrappers d'autorisation",
	"/etc/hosts.deny":            "politique historique TCP wrappers de refus",
	"~/.ssh/id_rsa":              "chemin traditionnel d'une clé privée utilisateur SSH RSA",
	"~/.ssh/id_dsa":              "chemin historique d'une clé DSA, algorithme aujourd'hui obsolète mais reconnaissable",
	"~/.ssh/id_ecdsa":            "chemin traditionnel d'une clé privée SSH ECDSA",
	"~/.ssh/id_ed25519":          "chemin traditionnel d'une clé privée SSH Ed25519",
	"/etc/ssh/ssh_host_*":        "clés privées/publiques identifiant le serveur SSH",
	"~/.ssh/authorized_keys":     "clés publiques autorisées à authentifier un utilisateur",
	"ssh_known_hosts":            "base système ou concept de clés d'hôtes SSH connues; complète ~/.ssh/known_hosts",
	"~/.gnupg":                   "répertoire utilisateur GnuPG contenant keybox, trust/configuration et sockets selon version",
}

func conceptLabels(concept curriculum.Concept) []string {
	labels := slices.Clone(concept.Classification)
	title := normalizeSearchText(concept.TitleFR)
	legacyMarkers := []string{
		"historique",
		"legacy",
		"upstart",
		"grub legacy",
		"net tools",
		"xinetd",
		"tcp wrappers",
		"syslog ng",
	}
	for _, marker := range legacyMarkers {
		if strings.Contains(title, marker) && !slices.Contains(labels, "lpic-legacy") {
			labels = append(labels, "lpic-legacy")
			break
		}
	}
	if len(labels) == 0 {
		labels = []string{"lpic-required"}
	}
	return labels
}
