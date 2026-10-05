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

	dailyQuestions := make(map[string]int)
	recallQuestions := make(map[string]bool)
	for _, question := range bundle.Questions {
		if question.Usage == "initial-assessment" {
			continue
		}
		for _, conceptID := range question.ConceptIDs {
			dailyQuestions[conceptID]++
			if question.EvidenceKindOnSuccess == "recall" {
				recallQuestions[conceptID] = true
			}
		}
	}

	for _, objective := range curriculumBundle.Objectives.Objectives {
		if !objective.Active {
			continue
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

			if !recallQuestions[concept.ID] {
				question := generatedRecallQuestion(objective, concept, index, len(concepts))
				if err := validateQuestion(question, knownObjectives, knownConcepts); err != nil {
					return fmt.Errorf("%s: %w", question.ID, err)
				}
				if previous, duplicate := seenIDs[question.ID]; duplicate {
					return fmt.Errorf("generated question %s conflicts with %s", question.ID, previous)
				}
				seenIDs[question.ID] = "generated standalone recall question"
				bundle.Questions = append(bundle.Questions, question)
				dailyQuestions[concept.ID]++
				recallQuestions[concept.ID] = true
			}
			if dailyQuestions[concept.ID] < 2 {
				question := generatedRecognitionQuestion(objective, concept, concepts, index)
				if err := validateQuestion(question, knownObjectives, knownConcepts); err != nil {
					return fmt.Errorf("%s: %w", question.ID, err)
				}
				if previous, duplicate := seenIDs[question.ID]; duplicate {
					return fmt.Errorf("generated question %s conflicts with %s", question.ID, previous)
				}
				seenIDs[question.ID] = "generated standalone recognition question"
				bundle.Questions = append(bundle.Questions, question)
				dailyQuestions[concept.ID]++
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
		fmt.Fprintf(&termList, "- `%s`\n", term)
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
	return Question{
		SchemaVersion: "1.0.0",
		ID:            concept.ID + ".q.autonomous-recall",
		ObjectiveIDs:  []string{objective.ID},
		ConceptIDs:    []string{concept.ID},
		Type:          "fill-in",
		Usage:         "daily",
		PromptFR: fmt.Sprintf(
			"Pour « %s », donne le terme, fichier ou utilitaire canonique mis en avant dans le cours comme premier repère pratique.",
			concept.TitleFR,
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
			"Le premier repère de ce module est %s. Il faut également comprendre le concept, pas seulement mémoriser ce nom.",
			primary,
		),
	}
}

func generatedRecognitionQuestion(
	objective curriculum.Objective,
	concept curriculum.Concept,
	concepts []curriculum.Concept,
	index int,
) Question {
	choices := []Choice{{ID: "correct", LabelFR: concept.TitleFR}}
	for offset := 1; len(choices) < 4 && offset < len(concepts)+1; offset++ {
		candidate := concepts[(index+offset)%len(concepts)]
		if candidate.ID == concept.ID {
			continue
		}
		choices = append(choices, Choice{
			ID:      fmt.Sprintf("other-%d", len(choices)),
			LabelFR: candidate.TitleFR,
		})
	}
	for len(choices) < 4 {
		choices = append(choices, Choice{
			ID:      fmt.Sprintf("other-%d", len(choices)),
			LabelFR: "une autre compétence de l'objectif",
		})
	}
	rotation := index % len(choices)
	choices[0], choices[rotation] = choices[rotation], choices[0]
	correctID := "correct"

	terms := termsForConcept(objective.TermsFilesUtilities, concept.TitleFR, index, len(concepts))
	return Question{
		SchemaVersion: "1.0.0",
		ID:            concept.ID + ".q.autonomous-recognition",
		ObjectiveIDs:  []string{objective.ID},
		ConceptIDs:    []string{concept.ID},
		Type:          "multiple-choice",
		Usage:         "daily",
		PromptFR: fmt.Sprintf(
			"Le repère technique %s apparaît dans ce module. Quelle compétence faut-il lui associer en priorité ?",
			inlineCodeList(terms),
		),
		Choices: choices,
		Grading: Grading{
			Strategy:          "choice-ids",
			AcceptedChoiceIDs: []string{correctID},
		},
		EvidenceKindOnSuccess: "recognition",
		Labels:                conceptLabels(concept),
		Distribution:          "generic",
		ExplanationFR: fmt.Sprintf(
			"%s est travaillé ici dans le cadre de « %s ».",
			inlineCodeList(terms),
			concept.TitleFR,
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
	for _, hint := range conceptTermHints {
		if !strings.Contains(normalizedTitle, hint.keyword) {
			continue
		}
		for _, wanted := range hint.terms {
			for _, term := range terms {
				if strings.EqualFold(term, wanted) && !slices.Contains(matches, term) {
					matches = append(matches, term)
				}
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
