package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	lpicdaily "github.com/Loe159/lpic-daily"
	"github.com/Loe159/lpic-daily/internal/appstate"
	"github.com/Loe159/lpic-daily/internal/assessment"
	"github.com/Loe159/lpic-daily/internal/content"
	"github.com/Loe159/lpic-daily/internal/curriculum"
	"github.com/Loe159/lpic-daily/internal/desktop"
	"github.com/Loe159/lpic-daily/internal/doctor"
	"github.com/Loe159/lpic-daily/internal/gamification"
	"github.com/Loe159/lpic-daily/internal/lab"
	"github.com/Loe159/lpic-daily/internal/learning"
	progresssqlite "github.com/Loe159/lpic-daily/internal/progress/sqlite"
	"github.com/Loe159/lpic-daily/internal/runner"
	libvirtrunner "github.com/Loe159/lpic-daily/internal/runner/libvirt"
	podmanrunner "github.com/Loe159/lpic-daily/internal/runner/podman"
	"github.com/Loe159/lpic-daily/internal/study"
	"github.com/Loe159/lpic-daily/internal/terminal"
	lpicui "github.com/Loe159/lpic-daily/internal/tui"
	"github.com/muesli/cancelreader"
)

const version = "0.0.0-dev"

func main() {
	if err := runWithIO(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	return runWithIO(args, os.Stdin, os.Stdout, os.Stderr)
}

func runWithIO(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		if inputFile, inputOK := stdin.(*os.File); inputOK && terminal.IsTerminal(inputFile) {
			if outputFile, outputOK := stdout.(*os.File); outputOK && terminal.IsTerminal(outputFile) {
				return runDashboard(stdin, stdout, stderr)
			}
		}
		printUsage(stdout)
		return nil
	}

	switch args[0] {
	case "validate":
		bundle, err := curriculum.Load(lpicdaily.BuiltinFS)
		if err != nil {
			return err
		}
		labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
		if err != nil {
			return fmt.Errorf("builtin labs: %w", err)
		}
		fmt.Fprintln(stdout, "builtin curriculum OK")
		fmt.Fprintln(stdout, curriculum.FormatSummary(curriculum.Summarize(bundle)))
		contentBundle, err := content.Load(lpicdaily.BuiltinFS)
		if err != nil {
			return fmt.Errorf("builtin content: %w", err)
		}
		fmt.Fprintf(stdout, "builtin labs OK: %d\n", len(labs))
		fmt.Fprintf(stdout, "builtin content OK: %d lessons; %d questions\n", len(contentBundle.Lessons), len(contentBundle.Questions))
		return nil
	case "doctor":
		if _, err := curriculum.Load(lpicdaily.BuiltinFS); err != nil {
			return fmt.Errorf("builtin curriculum: %w", err)
		}
		if _, err := lab.LoadAll(lpicdaily.BuiltinFS); err != nil {
			return fmt.Errorf("builtin labs: %w", err)
		}
		if _, err := content.Load(lpicdaily.BuiltinFS); err != nil {
			return fmt.Errorf("builtin content: %w", err)
		}
		for _, check := range doctor.Run().Checks {
			fmt.Fprintf(stdout, "%-24s %-5s %s\n", check.Name, check.Status, check.Detail)
		}
		return nil
	case "today":
		return runToday(args[1:], stdout)
	case "tui":
		if len(args) != 1 {
			return fmt.Errorf("usage: lpic tui")
		}
		return runDashboard(stdin, stdout, stderr)
	case "notify":
		return runNotify(args[1:], stdout)
	case "assess":
		return runAssessment(args[1:], stdin, stdout)
	case "learn":
		return runLearn(args[1:], stdin, stdout)
	case "question":
		return runQuestion(args[1:], stdin, stdout)
	case "labs":
		return runLabCommand([]string{"list"}, stdin, stdout, stderr)
	case "lab":
		return runLabCommand(args[1:], stdin, stdout, stderr)
	case "version":
		fmt.Fprintln(stdout, version)
		return nil
	case "help", "-h", "--help":
		printUsage(stdout)
		return nil
	default:
		return fmt.Errorf("unknown command %q (try: lpic help)", args[0])
	}
}

func runDashboard(stdin io.Reader, stdout, stderr io.Writer) error {
	inputFile, inputOK := stdin.(*os.File)
	outputFile, outputOK := stdout.(*os.File)
	if !inputOK || !outputOK || !terminal.IsTerminal(inputFile) || !terminal.IsTerminal(outputFile) {
		return errors.New("lpic tui requires an interactive terminal on stdin and stdout")
	}

	curriculumBundle, err := curriculum.Load(lpicdaily.BuiltinFS)
	if err != nil {
		return fmt.Errorf("load curriculum: %w", err)
	}
	contentBundle, err := content.Load(lpicdaily.BuiltinFS)
	if err != nil {
		return fmt.Errorf("load content: %w", err)
	}
	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		return fmt.Errorf("load labs: %w", err)
	}

	ctx := context.Background()
	for {
		store, err := openProgressStore(ctx)
		if err != nil {
			return err
		}
		now := time.Now()
		plan, planErr := study.BuildPlan(ctx, study.PlanInput{
			Now:        now,
			Curriculum: curriculumBundle,
			Content:    contentBundle,
			Labs:       labs,
			Evidence:   store,
			Policy:     learning.DefaultSessionPolicy(),
		})
		gameSnapshot, gameErr := loadGamificationSnapshot(ctx, store, now)
		closeErr := store.Close()
		if planErr != nil {
			return planErr
		}
		if gameErr != nil {
			return gameErr
		}
		if closeErr != nil {
			return closeErr
		}

		action, err := lpicui.Run(ctx, plan, gameSnapshot, stdin, stdout)
		if err != nil {
			return fmt.Errorf("run TUI: %w", err)
		}

		switch action.Kind {
		case lpicui.ActionNone:
			return nil
		case lpicui.ActionLesson:
			err = runLearn([]string{action.ID}, stdin, stdout)
		case lpicui.ActionQuestion:
			err = runQuestion([]string{action.ID}, stdin, stdout)
		case lpicui.ActionLab:
			err = runLabCommand([]string{"run", action.ID}, stdin, stdout, stderr)
		default:
			return fmt.Errorf("unsupported TUI action %q", action.Kind)
		}
		if err != nil {
			return err
		}

		fmt.Fprintln(stdout)
	}
}

func runNotify(args []string, stdout io.Writer) error {
	force := false
	switch {
	case len(args) == 0:
	case len(args) == 1 && args[0] == "--force":
		force = true
	default:
		return fmt.Errorf("usage: lpic notify [--force]")
	}
	return runNotifyWithExecutor(
		context.Background(),
		force,
		stdout,
		desktop.OSExecutor{},
		time.Now(),
	)
}

func runNotifyWithExecutor(
	ctx context.Context,
	force bool,
	stdout io.Writer,
	executor desktop.Executor,
	now time.Time,
) error {
	curriculumBundle, err := curriculum.Load(lpicdaily.BuiltinFS)
	if err != nil {
		return fmt.Errorf("load curriculum: %w", err)
	}
	contentBundle, err := content.Load(lpicdaily.BuiltinFS)
	if err != nil {
		return fmt.Errorf("load content: %w", err)
	}
	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		return fmt.Errorf("load labs: %w", err)
	}

	store, err := openProgressStore(ctx)
	if err != nil {
		return err
	}
	defer store.Close()

	localDay := now.In(time.Local).Format("2006-01-02")

	plan, err := study.BuildPlan(ctx, study.PlanInput{
		Now:        now,
		Curriculum: curriculumBundle,
		Content:    contentBundle,
		Labs:       labs,
		Evidence:   store,
		Policy:     learning.DefaultSessionPolicy(),
	})
	if err != nil {
		return err
	}
	if len(plan.Items) == 0 {
		fmt.Fprintln(stdout, "Aucune activité due : notification non envoyée.")
		return nil
	}

	reviews := 0
	practices := 0
	newItems := 0
	for _, item := range plan.Items {
		switch item.Kind {
		case learning.SessionReview:
			reviews++
		case learning.SessionPractice:
			practices++
		case learning.SessionNew:
			newItems++
		}
	}
	body := fmt.Sprintf(
		"%d activité(s) due(s) · %d révision(s) · %d consolidation(s) · %d nouveau(x) concept(s)",
		len(plan.Items),
		reviews,
		practices,
		newItems,
	)

	claimed := false
	if !force {
		var err error
		claimed, err = store.ClaimNotification(ctx, localDay, now)
		if err != nil {
			return err
		}
		if !claimed {
			sent, err := store.NotificationSent(ctx, localDay)
			if err != nil {
				return err
			}
			if sent {
				fmt.Fprintln(stdout, "Notification quotidienne déjà envoyée.")
			} else {
				fmt.Fprintln(stdout, "Notification quotidienne déjà réservée pour aujourd'hui.")
			}
			return nil
		}
	}

	open, err := desktop.SendDaily(ctx, executor, desktop.Notification{
		Title: "LPIC Daily",
		Body:  body,
	})
	if err != nil {
		if claimed {
			if releaseErr := store.ReleaseNotificationClaim(ctx, localDay); releaseErr != nil {
				return errors.Join(err, releaseErr)
			}
		}
		return err
	}
	if err := store.MarkNotificationSent(ctx, localDay, now); err != nil {
		return err
	}
	fmt.Fprintln(stdout, "Notification quotidienne envoyée.")

	if open {
		if err := desktop.LaunchDaily(ctx, executor); err != nil {
			return fmt.Errorf("notification opened but daily session could not launch: %w", err)
		}
	}
	return nil
}

func runToday(args []string, stdout io.Writer) error {
	policy := learning.DefaultSessionPolicy()
	switch {
	case len(args) == 0:
	case len(args) == 1 && args[0] == "--quick":
		policy.MaxReviews = 2
	default:
		return fmt.Errorf("usage: lpic today [--quick]")
	}

	curriculumBundle, err := curriculum.Load(lpicdaily.BuiltinFS)
	if err != nil {
		return fmt.Errorf("load curriculum: %w", err)
	}
	contentBundle, err := content.Load(lpicdaily.BuiltinFS)
	if err != nil {
		return fmt.Errorf("load content: %w", err)
	}
	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		return fmt.Errorf("load labs: %w", err)
	}

	ctx := context.Background()
	store, err := openProgressStore(ctx)
	if err != nil {
		return err
	}
	defer store.Close()

	plan, err := study.BuildPlan(ctx, study.PlanInput{
		Now:        time.Now(),
		Curriculum: curriculumBundle,
		Content:    contentBundle,
		Labs:       labs,
		Evidence:   store,
		Policy:     policy,
	})
	if err != nil {
		return err
	}

	fmt.Fprintln(stdout, "LPIC Daily — Aujourd'hui")
	if len(plan.Items) == 0 {
		fmt.Fprintln(stdout, "Aucune activité due dans le périmètre Phase 1.")
		return nil
	}

	for index, item := range plan.Items {
		kind := "Nouveau"
		switch item.Kind {
		case learning.SessionReview:
			kind = "Révision"
		case learning.SessionPractice:
			kind = "Consolidation"
		}
		fmt.Fprintf(
			stdout,
			"\n%d. %s · %s · %s\n",
			index+1,
			kind,
			item.ObjectiveID,
			item.ConceptTitleFR,
		)
		fmt.Fprintf(stdout, "   Maîtrise: %s\n", item.MasteryStage)
		fmt.Fprintf(stdout, "   Pourquoi: %s\n", item.ReasonFR)
		if item.RecommendedLessonID != "" {
			fmt.Fprintf(stdout, "   Cours conseillé: %s\n", item.RecommendedLessonID)
		}
		if item.RecommendedQuestionID != "" {
			fmt.Fprintf(stdout, "   Question: %s\n", item.RecommendedQuestionID)
		}
		if item.RecommendedLabID != "" {
			fmt.Fprintf(stdout, "   Lab: %s\n", item.RecommendedLabID)
		}
	}
	return nil
}

func openProgressStore(ctx context.Context) (*progresssqlite.Store, error) {
	databasePath, err := appstate.ProgressDBPath()
	if err != nil {
		return nil, fmt.Errorf("resolve progress database path: %w", err)
	}
	store, err := progresssqlite.Open(ctx, databasePath)
	if err != nil {
		return nil, fmt.Errorf("open progress database: %w", err)
	}
	return store, nil
}

func recordLabDisclosure(
	ctx context.Context,
	labID string,
	highestHintLevel int,
	solutionRevealed bool,
) error {
	store, err := openProgressStore(ctx)
	if err != nil {
		return err
	}
	if err := store.RecordLabDisclosure(
		ctx,
		labID,
		highestHintLevel,
		solutionRevealed,
		time.Now(),
	); err != nil {
		_ = store.Close()
		return err
	}
	if err := store.Close(); err != nil {
		return fmt.Errorf("close progress database after lab disclosure: %w", err)
	}
	return nil
}

func pendingLabHintLevel(ctx context.Context, labID string) (int, error) {
	store, err := openProgressStore(ctx)
	if err != nil {
		return 0, err
	}
	disclosure, disclosureErr := store.LabDisclosure(ctx, labID)
	closeErr := store.Close()
	if disclosureErr != nil {
		return 0, disclosureErr
	}
	if closeErr != nil {
		return 0, fmt.Errorf("close progress database after reading lab disclosure: %w", closeErr)
	}
	return disclosure.HighestHintLevel, nil
}

func nextHintIndex(hints []lab.Hint, highestHintLevel int) int {
	for index, hint := range hints {
		if hint.Level > highestHintLevel {
			return index
		}
	}
	return len(hints)
}

func runAssessment(args []string, stdin io.Reader, stdout io.Writer) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: lpic assess")
	}

	curriculumBundle, err := curriculum.Load(lpicdaily.BuiltinFS)
	if err != nil {
		return fmt.Errorf("load curriculum: %w", err)
	}
	contentBundle, err := content.Load(lpicdaily.BuiltinFS)
	if err != nil {
		return fmt.Errorf("load content: %w", err)
	}
	questions, err := assessment.Questions(contentBundle)
	if err != nil {
		return err
	}

	ctx := context.Background()
	store, err := openProgressStore(ctx)
	if err != nil {
		return err
	}
	defer store.Close()

	fmt.Fprintln(stdout, "Évaluation initiale · 103.1")
	fmt.Fprintln(stdout, "Réponds sans consulter les cours. Les réponses correctes enregistrent uniquement des preuves de rappel.")
	fmt.Fprintln(stdout)

	correct := 0
	reader := bufio.NewReader(stdin)
	for index, question := range questions {
		fmt.Fprintf(stdout, "%d/%d · %s\n", index+1, len(questions), question.PromptFR)
		fmt.Fprint(stdout, "Réponse: ")

		line, err := readLine(reader)
		if err != nil {
			return fmt.Errorf("read assessment answer %d: %w", index+1, err)
		}
		answer, err := parseAnswer(question, strings.TrimSpace(line))
		if err != nil {
			return fmt.Errorf("assessment question %d: %w", index+1, err)
		}
		pass, err := question.Grade(answer)
		if err != nil {
			return fmt.Errorf("grade assessment question %d: %w", index+1, err)
		}
		now := time.Now()
		if err := study.RecordQuestion(ctx, store, question, pass, now); err != nil {
			return err
		}
		eventType := gamification.EventQuestionFailed
		if pass {
			correct++
			eventType = gamification.EventQuestionPassed
			fmt.Fprintln(stdout, "Correct.")
		} else {
			fmt.Fprintln(stdout, "Incorrect.")
		}
		recordGamificationBestEffort(
			ctx,
			store,
			eventType,
			now,
			map[string]string{
				"source_item_id": question.ID,
				"mode":           "initial-assessment",
			},
			stdout,
		)
		fmt.Fprintln(stdout)
	}

	ready, err := assessment.FoundationReady(ctx, curriculumBundle, store)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Résultat: %d/%d\n", correct, len(questions))
	if ready {
		fmt.Fprintln(stdout, "Foundation 103.1 prête : les objectifs dépendants peuvent être proposés.")
	} else {
		fmt.Fprintln(stdout, "Foundation 103.1 pas encore prête : LPIC Daily proposera les concepts manquants.")
	}
	return nil
}

func runLearn(args []string, stdin io.Reader, stdout io.Writer) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: lpic learn <lesson-id>")
	}

	bundle, err := content.Load(lpicdaily.BuiltinFS)
	if err != nil {
		return fmt.Errorf("load content: %w", err)
	}
	lesson, ok := bundle.LessonByID(args[0])
	if !ok {
		return fmt.Errorf("unknown lesson %q", args[0])
	}

	fmt.Fprintf(stdout, "%s\n\n%s\n", lesson.TitleFR, lesson.BodyMarkdown)
	fmt.Fprint(stdout, "\nMarquer ce cours comme lu ? [o/N] ")

	line, err := readLine(stdin)
	if err != nil {
		return fmt.Errorf("read lesson confirmation: %w", err)
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "o", "oui", "y", "yes":
	default:
		fmt.Fprintln(stdout, "Progression non enregistrée.")
		return nil
	}

	ctx := context.Background()
	store, err := openProgressStore(ctx)
	if err != nil {
		return err
	}
	defer store.Close()

	now := time.Now()
	if err := study.RecordLesson(ctx, store, lesson, now); err != nil {
		return err
	}
	fmt.Fprintln(stdout, "Progression enregistrée.")
	recordGamificationBestEffort(
		ctx,
		store,
		gamification.EventLessonCompleted,
		now,
		map[string]string{"source_item_id": lesson.ID},
		stdout,
	)
	return nil
}

func runQuestion(args []string, stdin io.Reader, stdout io.Writer) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: lpic question <question-id>")
	}

	bundle, err := content.Load(lpicdaily.BuiltinFS)
	if err != nil {
		return fmt.Errorf("load content: %w", err)
	}
	question, ok := bundle.QuestionByID(args[0])
	if !ok {
		return fmt.Errorf("unknown question %q", args[0])
	}

	fmt.Fprintln(stdout, question.PromptFR)
	for index, choice := range question.Choices {
		fmt.Fprintf(stdout, "  %d) %s\n", index+1, choice.LabelFR)
	}
	fmt.Fprint(stdout, "Réponse: ")

	line, err := readLine(stdin)
	if err != nil {
		return fmt.Errorf("read answer: %w", err)
	}
	answer, err := parseAnswer(question, strings.TrimSpace(line))
	if err != nil {
		return err
	}
	pass, err := question.Grade(answer)
	if err != nil {
		return fmt.Errorf("grade answer: %w", err)
	}

	ctx := context.Background()
	store, err := openProgressStore(ctx)
	if err != nil {
		return err
	}
	defer store.Close()

	now := time.Now()
	if err := study.RecordQuestion(ctx, store, question, pass, now); err != nil {
		return err
	}
	eventType := gamification.EventQuestionFailed
	if pass {
		eventType = gamification.EventQuestionPassed
	}
	recordGamificationBestEffort(
		ctx,
		store,
		eventType,
		now,
		map[string]string{"source_item_id": question.ID},
		stdout,
	)

	if pass {
		fmt.Fprintln(stdout, "Correct.")
	} else {
		fmt.Fprintln(stdout, "Incorrect.")
	}
	if question.ExplanationFR != "" {
		fmt.Fprintln(stdout, question.ExplanationFR)
	}
	return nil
}

func readLine(input io.Reader) (string, error) {
	reader, ok := input.(*bufio.Reader)
	if !ok {
		reader = bufio.NewReader(input)
	}
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	if errors.Is(err, io.EOF) && line == "" {
		return "", errors.New("no input provided")
	}
	return line, nil
}

func parseAnswer(question content.Question, line string) (content.Answer, error) {
	switch question.Type {
	case "multiple-choice", "ordering":
		if line == "" {
			return content.Answer{}, errors.New("answer cannot be empty")
		}
		parts := strings.Split(line, ",")
		ids := make([]string, 0, len(parts))
		for _, raw := range parts {
			token := strings.TrimSpace(raw)
			if token == "" {
				return content.Answer{}, errors.New("empty choice in answer")
			}
			if number, err := strconv.Atoi(token); err == nil {
				if number < 1 || number > len(question.Choices) {
					return content.Answer{}, fmt.Errorf("choice number %d outside 1..%d", number, len(question.Choices))
				}
				ids = append(ids, question.Choices[number-1].ID)
				continue
			}

			found := false
			for _, choice := range question.Choices {
				if choice.ID == token {
					ids = append(ids, choice.ID)
					found = true
					break
				}
			}
			if !found {
				return content.Answer{}, fmt.Errorf("unknown choice %q", token)
			}
		}
		return content.Answer{ChoiceIDs: ids}, nil
	default:
		return content.Answer{Text: line}, nil
	}
}

func loadGamificationSnapshot(
	ctx context.Context,
	store *progresssqlite.Store,
	now time.Time,
) (gamification.Snapshot, error) {
	events, err := store.GamificationEvents(ctx)
	if err != nil {
		return gamification.Snapshot{}, fmt.Errorf("load gamification: %w", err)
	}
	snapshot, err := gamification.Project(events, time.Local, now)
	if err != nil {
		return gamification.Snapshot{}, fmt.Errorf("project gamification: %w", err)
	}
	return snapshot, nil
}

func recordGamificationBestEffort(
	ctx context.Context,
	store *progresssqlite.Store,
	eventType gamification.EventType,
	at time.Time,
	metadata map[string]string,
	out io.Writer,
) {
	snapshot, unlocked, err := gamification.RecordActivity(
		ctx,
		store,
		eventType,
		at,
		metadata,
		time.Local,
	)
	if err != nil {
		fmt.Fprintf(out, "Avertissement: gamification non enregistrée: %v\n", err)
		return
	}
	fmt.Fprintf(out, "XP: %d · série: %d jour(s)\n", snapshot.XP, snapshot.CurrentStreakDays)
	for _, achievement := range unlocked {
		fmt.Fprintf(out, "Achievement débloqué: %s (+%d XP)\n", achievement.TitleFR, achievement.XPReward)
	}
}

func runLabCommand(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printLabUsage(stdout)
		return nil
	}

	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		return fmt.Errorf("load built-in labs: %w", err)
	}

	switch args[0] {
	case "list":
		for _, authored := range labs {
			definition := authored.Definition
			fmt.Fprintf(
				stdout,
				"%-34s %3d min  %-7s %-8s %s\n",
				definition.ID,
				definition.EstimatedMinutes,
				definition.Environment.Backend,
				definition.Environment.Distribution,
				definition.TitleFR,
			)
		}
		return nil
	case "show":
		if len(args) != 2 {
			return fmt.Errorf("usage: lpic lab show <lab-id>")
		}
		authored, err := findLab(labs, args[1])
		if err != nil {
			return err
		}
		printLab(authored, stdout)
		return nil
	case "run":
		if len(args) != 2 {
			return fmt.Errorf("usage: lpic lab run <lab-id>")
		}
		authored, err := findLab(labs, args[1])
		if err != nil {
			return err
		}
		return runInteractiveLab(authored, stdin, stdout, stderr)
	case "hint":
		if len(args) != 3 {
			return fmt.Errorf("usage: lpic lab hint <lab-id> <1-4>")
		}
		authored, err := findLab(labs, args[1])
		if err != nil {
			return err
		}
		level, err := strconv.Atoi(args[2])
		if err != nil || level < 1 || level > 4 {
			return fmt.Errorf("hint level must be an integer from 1 to 4")
		}
		for _, hint := range authored.Hints {
			if hint.Level != level {
				continue
			}
			solutionRevealed := hint.EvidenceImpact == "solution-revealed"
			if err := recordLabDisclosure(
				context.Background(),
				authored.Definition.ID,
				hint.Level,
				solutionRevealed,
			); err != nil {
				return fmt.Errorf("record hint disclosure: %w", err)
			}
			fmt.Fprintf(stdout, "Indice %d/4\n%s\n", hint.Level, hint.ContentFR)
			if solutionRevealed {
				fmt.Fprintln(stdout, "\nCet indice révèle la solution et réduit la force de la preuve pratique.")
			}
			return nil
		}
		return fmt.Errorf("lab %s has no hint level %d", authored.Definition.ID, level)
	case "debrief":
		if len(args) != 2 {
			return fmt.Errorf("usage: lpic lab debrief <lab-id>")
		}
		authored, err := findLab(labs, args[1])
		if err != nil {
			return err
		}
		if err := recordLabDisclosure(
			context.Background(),
			authored.Definition.ID,
			4,
			true,
		); err != nil {
			return fmt.Errorf("record debrief disclosure: %w", err)
		}
		fmt.Fprintf(stdout, "%s\n\n%s\n", authored.Definition.TitleFR, authored.Definition.DebriefFR)
		return nil
	case "help", "-h", "--help":
		printLabUsage(stdout)
		return nil
	default:
		return fmt.Errorf("unknown lab command %q", args[0])
	}
}

func findLab(labs []lab.Lab, id string) (lab.Lab, error) {
	for _, authored := range labs {
		if authored.Definition.ID == id {
			return authored, nil
		}
	}
	return lab.Lab{}, fmt.Errorf("unknown lab %q", id)
}

func printLab(authored lab.Lab, out io.Writer) {
	definition := authored.Definition
	fmt.Fprintf(out, "%s\n", definition.TitleFR)
	fmt.Fprintf(out, "ID: %s\n", definition.ID)
	fmt.Fprintf(out, "Objectifs LPIC: %s\n", strings.Join(definition.ObjectiveIDs, ", "))
	fmt.Fprintf(out, "Environnement: %s / %s / network=%s\n", definition.Environment.Backend, definition.Environment.Distribution, definition.Environment.Network)
	fmt.Fprintf(out, "Durée estimée: %d min\n", definition.EstimatedMinutes)
	fmt.Fprintf(out, "Labels: %s\n\n", strings.Join(definition.Labels, ", "))
	fmt.Fprintln(out, definition.BriefFR)
	fmt.Fprintln(out, "\nCritères de réussite:")
	for _, criterion := range definition.SuccessCriteriaFR {
		fmt.Fprintf(out, "- %s\n", criterion)
	}
	fmt.Fprintf(out, "\n%d niveaux d'indices disponibles. Le debrief est masqué jusqu'à réussite ou demande explicite.\n", len(authored.Hints))
}

func runInteractiveLab(authored lab.Lab, stdin io.Reader, stdout, stderr io.Writer) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	switch authored.Definition.Environment.Backend {
	case "podman":
		backend, err := podmanrunner.Open(ctx, "")
		if err != nil {
			return fmt.Errorf("open rootless Podman backend: %w", err)
		}
		return runInteractiveLabWithBackend(
			ctx,
			authored,
			backend,
			true,
			stdin,
			stdout,
			stderr,
		)
	case "libvirt":
		backend, err := openLibvirtBackend()
		if err != nil {
			return err
		}
		defer func() {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := backend.Close(cleanupCtx); err != nil {
				fmt.Fprintln(stderr, "warning: close libvirt backend:", err)
			}
		}()
		return runInteractiveLabWithBackend(
			ctx,
			authored,
			backend,
			false,
			stdin,
			stdout,
			stderr,
		)
	default:
		return fmt.Errorf(
			"unsupported lab backend %q",
			authored.Definition.Environment.Backend,
		)
	}
}

func requireUnprivilegedVMProcess(euid int) error {
	if euid == 0 {
		return errors.New("libvirt VM labs must run as a regular user, not root")
	}
	return nil
}

func openLibvirtBackend() (*libvirtrunner.Backend, error) {
	imageRoot, err := appstate.VMImageRoot()
	if err != nil {
		return nil, fmt.Errorf("resolve VM image root: %w", err)
	}
	catalogPath, err := appstate.VMImageCatalogPath()
	if err != nil {
		return nil, fmt.Errorf("resolve VM image catalog path: %w", err)
	}
	catalog, err := libvirtrunner.LoadImageCatalog(catalogPath, imageRoot)
	if err != nil {
		return nil, fmt.Errorf("load VM image catalog: %w", err)
	}

	stateRoot, err := appstate.VMStateRoot()
	if err != nil {
		return nil, fmt.Errorf("resolve VM state root: %w", err)
	}
	if err := requireUnprivilegedVMProcess(os.Geteuid()); err != nil {
		return nil, err
	}
	commands, err := libvirtrunner.NewExecCommandRunner()
	if err != nil {
		return nil, err
	}
	control, err := libvirtrunner.OpenSystem()
	if err != nil {
		return nil, err
	}

	backend, err := libvirtrunner.NewBackend(
		control,
		catalog,
		imageRoot,
		stateRoot,
		appstate.VMNetworkAllocationLockPath(),
		commands,
	)
	if err != nil {
		_ = control.Close()
		return nil, fmt.Errorf("initialize libvirt backend: %w", err)
	}
	reapCtx, cancelReap := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelReap()
	if err := backend.Reap(reapCtx); err != nil {
		_ = backend.Close(context.Background())
		return nil, fmt.Errorf("reap abandoned VM resources: %w", err)
	}
	return backend, nil
}

func runInteractiveLabWithBackend(
	ctx context.Context,
	authored lab.Lab,
	backend runner.Runner,
	persistentShell bool,
	stdin io.Reader,
	stdout, stderr io.Writer,
) error {
	timeout := time.Duration(authored.Definition.Resources.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		return errors.New("lab runtime timeout must be positive")
	}
	sessionCtx, cancelSession := context.WithTimeout(ctx, timeout)
	defer cancelSession()

	if inputFile, ok := stdin.(*os.File); ok {
		if deadline, ok := sessionCtx.Deadline(); ok {
			if err := inputFile.SetReadDeadline(deadline); err == nil {
				defer inputFile.SetReadDeadline(time.Time{})
			}
		}
	}

	highestHintLevel, err := pendingLabHintLevel(sessionCtx, authored.Definition.ID)
	if err != nil {
		return fmt.Errorf("load pending lab disclosure: %w", err)
	}

	session, err := lab.Start(sessionCtx, authored, backend)
	if err != nil {
		return err
	}
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := session.Close(cleanupCtx); err != nil {
			fmt.Fprintln(stderr, "warning: cleanup lab:", err)
		}
	}()

	printLab(authored, stdout)
	fmt.Fprintln(stdout)
	if persistentShell {
		fmt.Fprintln(stdout, "Mode commandes sandboxé. Chaque ligne est exécutée dans un nouveau shell du lab.")
		fmt.Fprintln(stdout, "Commandes LPIC Daily : :shell  :check  :hint  :reset  :quit")
	} else {
		fmt.Fprintln(stdout, "Mode commandes VM. Chaque ligne est exécutée dans la VM via QEMU Guest Agent.")
		fmt.Fprintln(stdout, "Commandes LPIC Daily : :console  :reboot  :check  :hint  :reset  :quit")
	}
	fmt.Fprintln(stdout)

	scanner := bufio.NewScanner(stdin)
	scanner.Buffer(make([]byte, 4096), 256<<10)
	nextHint := nextHintIndex(authored.Hints, highestHintLevel)
	usedPersistentShell := false
	jobControlEvidence := &jobControlInteractionEvidence{}

	for {
		if err := sessionCtx.Err(); err != nil {
			return fmt.Errorf("lab session ended: %w", err)
		}
		fmt.Fprint(stdout, "lpic> ")
		if !scanner.Scan() {
			if err := sessionCtx.Err(); err != nil {
				return fmt.Errorf("lab session ended: %w", err)
			}
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("read command: %w", err)
			}
			fmt.Fprintln(stdout)
			return nil
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		switch line {
		case ":quit", ":q", "exit":
			return nil
		case ":hint":
			if nextHint >= len(authored.Hints) {
				fmt.Fprintln(stdout, "Aucun indice supplémentaire.")
				continue
			}
			hint := authored.Hints[nextHint]
			solutionRevealed := hint.EvidenceImpact == "solution-revealed"
			if err := recordLabDisclosure(
				sessionCtx,
				authored.Definition.ID,
				hint.Level,
				solutionRevealed,
			); err != nil {
				return fmt.Errorf("record hint disclosure: %w", err)
			}
			nextHint++
			if hint.Level > highestHintLevel {
				highestHintLevel = hint.Level
			}
			fmt.Fprintf(stdout, "Indice %d/4 : %s\n", hint.Level, hint.ContentFR)
			if solutionRevealed {
				fmt.Fprintln(stdout, "Cet indice révèle la solution et réduira la force de la preuve pratique.")
			}
			continue
		case ":shell":
			if !persistentShell {
				fmt.Fprintln(stdout, "Le shell PTY n'est pas disponible pour ce backend ; utilise le mode commandes.")
				continue
			}
			fmt.Fprintln(stdout, "Ouverture d'un shell persistant dans la sandbox. Tape exit ou Ctrl-D pour revenir.")
			result, err := runPersistentShell(
				sessionCtx,
				backend,
				session.Instance,
				stdin,
				stdout,
				jobControlEvidence,
			)
			if err != nil {
				return fmt.Errorf("interactive sandbox shell: %w", err)
			}
			usedPersistentShell = true
			fmt.Fprintln(stdout, "\n[retour LPIC Daily]")
			if result.ExitCode != 0 {
				fmt.Fprintf(stderr, "[shell exit %d]\n", result.ExitCode)
			}
			continue
		case ":console":
			if persistentShell {
				fmt.Fprintln(stdout, "La console série est réservée aux labs VM.")
				continue
			}
			fmt.Fprintln(stdout, "Console série brute ouverte. Ctrl-] revient à LPIC Daily.")
			if err := runVMConsole(sessionCtx, backend, session.Instance, stdin, stdout); err != nil {
				return fmt.Errorf("VM serial console: %w", err)
			}
			fmt.Fprintln(stdout, "\n[retour LPIC Daily]")
			continue
		case ":reboot":
			if persistentShell {
				fmt.Fprintln(stdout, "Le reboot VM n'est pas disponible pour ce backend.")
				continue
			}
			rebooter, ok := backend.(runner.RebootRunner)
			if !ok {
				return fmt.Errorf("%w: backend has no reboot capability", runner.ErrNotSupported)
			}
			if err := rebooter.Reboot(sessionCtx, session.Instance); err != nil {
				return fmt.Errorf("reboot VM: %w", err)
			}
			fmt.Fprintln(stdout, "Reboot demandé. La console série permet de suivre le prochain boot.")
			continue
		case ":reset":
			if err := session.Reset(sessionCtx); err != nil {
				return fmt.Errorf("reset lab: %w", err)
			}
			// Reset restores sandbox state only. Disclosure state is persisted
			// across resets/restarts and remains attached to the learning attempt.
			nextHint = nextHintIndex(authored.Hints, highestHintLevel)
			usedPersistentShell = false
			jobControlEvidence.Reset()
			fmt.Fprintln(stdout, "Lab réinitialisé dans son état de départ.")
			continue
		case ":check":
			passed := true
			results, err := session.Evaluate(sessionCtx)
			if err != nil {
				return fmt.Errorf("evaluate lab: %w", err)
			}
			conceptResults, err := study.LabConceptResults(authored, results)
			if err != nil {
				return fmt.Errorf("derive per-concept lab results: %w", err)
			}
			requiresJobControl := labRequiresJobControl(authored.Definition)
			if authored.Definition.NeedsPersistentShell && !usedPersistentShell {
				fmt.Fprintf(
					stdout,
					"  %-11s %s.persistent-shell — ce lab exige un passage par :shell pour valider l'interaction terminal réelle\n",
					"À CORRIGER",
					authored.Definition.ID,
				)
				passed = false
				markPersistentShellConceptFailure(authored.Definition, conceptResults)
			} else if requiresJobControl && !jobControlEvidence.Complete() {
				fmt.Fprintf(
					stdout,
					"  %-11s %s.job-control — la validation exige Ctrl-Z, une observation avec jobs et un bg réussi dans :shell\n",
					"À CORRIGER",
					authored.Definition.ID,
				)
				passed = false
				markPersistentShellConceptFailure(authored.Definition, conceptResults)
			}
			for _, result := range results {
				state := "OK"
				if !result.Pass {
					state = "À CORRIGER"
					passed = false
				}
				fmt.Fprintf(stdout, "  %-11s %s — %s\n", state, result.CheckID, result.Detail)
			}
			store, err := openProgressStore(sessionCtx)
			if err != nil {
				return fmt.Errorf("lab attempt progress could not be opened: %w", err)
			}
			now := time.Now()
			if passed {
				fmt.Fprintln(stdout, "\nLab réussi.")
				fmt.Fprintln(stdout, authored.Definition.DebriefFR)

				recordErr := study.RecordLab(sessionCtx, store, authored, highestHintLevel, now)
				if recordErr != nil {
					_ = store.Close()
					return fmt.Errorf("lab succeeded but progress could not be recorded: %w", recordErr)
				}
				if err := store.ClearLabDisclosure(sessionCtx, authored.Definition.ID); err != nil {
					_ = store.Close()
					return fmt.Errorf("lab succeeded but disclosure state could not be cleared: %w", err)
				}
				fmt.Fprintln(stdout, "Progression enregistrée.")
				recordGamificationBestEffort(
					sessionCtx,
					store,
					gamification.EventLabPassed,
					now,
					map[string]string{"source_item_id": authored.Definition.ID},
					stdout,
				)
				if closeErr := store.Close(); closeErr != nil {
					return fmt.Errorf("lab succeeded but progress store could not be closed: %w", closeErr)
				}
				return nil
			}

			recordErr := study.RecordLabConceptResults(
				sessionCtx,
				store,
				authored,
				conceptResults,
				highestHintLevel,
				now,
			)
			if recordErr != nil {
				_ = store.Close()
				return fmt.Errorf("failed lab attempt could not be recorded: %w", recordErr)
			}
			if closeErr := store.Close(); closeErr != nil {
				return fmt.Errorf("failed lab attempt store could not be closed: %w", closeErr)
			}
			fmt.Fprintln(stdout, "\nTentative enregistrée.")
			continue
		}

		if err := sessionCtx.Err(); err != nil {
			return fmt.Errorf("lab session ended: %w", err)
		}

		result, err := backend.Exec(sessionCtx, session.Instance, runner.ExecRequest{
			Argv:   []string{"/usr/bin/bash", "-lc", line},
			Stdout: stdout,
			Stderr: stderr,
		})
		if err != nil {
			return fmt.Errorf("execute lab command: %w", err)
		}
		if result.ExitCode != 0 {
			fmt.Fprintf(stderr, "[exit %d]\n", result.ExitCode)
		}
	}
}

func labRequiresJobControl(definition lab.Definition) bool {
	const jobControlConcept = "lpic1.103.5.jobs-du-shell"
	for _, conceptID := range definition.ConceptIDs {
		if conceptID == jobControlConcept {
			return true
		}
	}
	return false
}

type jobControlInteractionEvidence struct {
	sawCtrlZ bool
	sawJobs  bool
	sawBg    bool
}

func (evidence *jobControlInteractionEvidence) Complete() bool {
	return evidence != nil && evidence.sawCtrlZ && evidence.sawJobs && evidence.sawBg
}

func (evidence *jobControlInteractionEvidence) Reset() {
	if evidence != nil {
		*evidence = jobControlInteractionEvidence{}
	}
}

func (evidence *jobControlInteractionEvidence) observeShellEvents(raw []byte) {
	if evidence == nil {
		return
	}
	for _, event := range strings.Fields(string(raw)) {
		switch event {
		case "jobs":
			evidence.sawJobs = true
		case "bg":
			evidence.sawBg = true
		}
	}
}

type ctrlZObservingTerminalReader struct {
	file     *os.File
	evidence *jobControlInteractionEvidence
}

func (reader *ctrlZObservingTerminalReader) Read(buffer []byte) (int, error) {
	n, err := reader.file.Read(buffer)
	for _, value := range buffer[:n] {
		if value == 0x1a && reader.evidence != nil {
			reader.evidence.sawCtrlZ = true
		}
	}
	return n, err
}

func (reader *ctrlZObservingTerminalReader) Fd() uintptr {
	return reader.file.Fd()
}

const (
	jobControlShellRCPath = "/tmp/.lpic-daily-shell-rc"
	jobControlEventPath   = "/tmp/.lpic-daily-job-control-events"
)

const jobControlShellRC = `
__lpic_daily_job_control_log="/tmp/.lpic-daily-job-control-events"

jobs() {
    builtin jobs "$@"
    local status=$?
    if (( status == 0 )); then
        printf 'jobs\\n' >> "$__lpic_daily_job_control_log"
    fi
    return "$status"
}

bg() {
    builtin bg "$@"
    local status=$?
    if (( status == 0 )); then
        printf 'bg\\n' >> "$__lpic_daily_job_control_log"
    fi
    return "$status"
}
`

func prepareJobControlShell(
	ctx context.Context,
	backend runner.Runner,
	instance runner.Instance,
) error {
	for _, file := range []struct {
		path    string
		content string
	}{
		{path: jobControlShellRCPath, content: jobControlShellRC},
		{path: jobControlEventPath, content: ""},
	} {
		result, err := backend.Exec(ctx, instance, runner.ExecRequest{
			Argv: []string{
				"/usr/bin/bash",
				"-c",
				`printf '%s' "$1" > "$2"`,
				"lpic-daily",
				file.content,
				file.path,
			},
		})
		if err != nil {
			return fmt.Errorf("prepare job-control shell file %s: %w", file.path, err)
		}
		if result.ExitCode != 0 {
			return fmt.Errorf("prepare job-control shell file %s: exit %d", file.path, result.ExitCode)
		}
	}
	return nil
}

func collectJobControlEvidence(
	ctx context.Context,
	backend runner.Runner,
	instance runner.Instance,
	evidence *jobControlInteractionEvidence,
) error {
	if evidence == nil {
		return nil
	}
	raw, err := backend.ReadFile(ctx, instance, jobControlEventPath, 4096)
	if err != nil {
		return fmt.Errorf("read job-control shell evidence: %w", err)
	}
	evidence.observeShellEvents(raw)
	return nil
}

func markPersistentShellConceptFailure(
	definition lab.Definition,
	conceptResults map[string]learning.Result,
) {
	const jobControlConcept = "lpic1.103.5.jobs-du-shell"
	for _, conceptID := range definition.ConceptIDs {
		if conceptID == jobControlConcept {
			conceptResults[conceptID] = learning.ResultFail
			return
		}
	}
	for _, conceptID := range definition.ConceptIDs {
		conceptResults[conceptID] = learning.ResultFail
	}
}

type vmConsoleEscapeReader struct {
	reader  io.Reader
	escaped bool
}

func (reader *vmConsoleEscapeReader) Read(buffer []byte) (int, error) {
	if reader == nil || reader.reader == nil {
		return 0, io.EOF
	}
	if reader.escaped {
		return 0, io.EOF
	}
	n, err := reader.reader.Read(buffer)
	for index, value := range buffer[:n] {
		if value != 0x1d {
			continue
		}
		reader.escaped = true
		if index == 0 {
			return 0, io.EOF
		}
		return index, nil
	}
	return n, err
}

func runVMConsole(
	ctx context.Context,
	backend runner.Runner,
	instance runner.Instance,
	stdin io.Reader,
	stdout io.Writer,
) (returnErr error) {
	console, ok := backend.(runner.ConsoleRunner)
	if !ok {
		return fmt.Errorf("%w: backend has no serial console capability", runner.ErrNotSupported)
	}
	stdinFile, ok := stdin.(*os.File)
	if !ok || !terminal.IsTerminal(stdinFile) {
		return errors.New(":console requires an interactive terminal on stdin")
	}
	stdoutFile, ok := stdout.(*os.File)
	if !ok || !terminal.IsTerminal(stdoutFile) {
		return errors.New(":console requires an interactive terminal on stdout")
	}

	state, err := terminal.MakeRaw(stdinFile)
	if err != nil {
		return err
	}
	restored := false
	defer func() {
		if restored {
			return
		}
		if err := terminal.Restore(stdinFile, state); returnErr == nil && err != nil {
			returnErr = err
		}
	}()

	cancelableInput, err := cancelreader.NewReader(stdinFile)
	if err != nil {
		return fmt.Errorf("prepare cancellable VM console input: %w", err)
	}
	defer cancelableInput.Close()

	stopCancellation := make(chan struct{})
	defer close(stopCancellation)
	go func() {
		select {
		case <-ctx.Done():
			cancelableInput.Cancel()
		case <-stopCancellation:
		}
	}()

	err = console.OpenConsole(ctx, instance, runner.ConsoleRequest{
		Stdin:  &vmConsoleEscapeReader{reader: cancelableInput},
		Stdout: stdoutFile,
	})
	if restoreErr := terminal.Restore(stdinFile, state); restoreErr != nil {
		return restoreErr
	}
	restored = true
	if err != nil {
		if errors.Is(err, cancelreader.ErrCanceled) && ctx.Err() != nil {
			return ctx.Err()
		}
		return err
	}
	return nil
}

func runPersistentShell(
	ctx context.Context,
	backend runner.Runner,
	instance runner.Instance,
	stdin io.Reader,
	stdout io.Writer,
	evidence *jobControlInteractionEvidence,
) (result runner.ExecResult, returnErr error) {
	stdinFile, ok := stdin.(*os.File)
	if !ok || !terminal.IsTerminal(stdinFile) {
		return runner.ExecResult{}, errors.New(":shell requires an interactive terminal on stdin")
	}
	stdoutFile, ok := stdout.(*os.File)
	if !ok || !terminal.IsTerminal(stdoutFile) {
		return runner.ExecResult{}, errors.New(":shell requires an interactive terminal on stdout")
	}

	width, height, err := terminal.Size(stdoutFile)
	if err != nil {
		return runner.ExecResult{}, err
	}
	if evidence != nil {
		if err := prepareJobControlShell(ctx, backend, instance); err != nil {
			return runner.ExecResult{}, err
		}
	}

	state, err := terminal.MakeRaw(stdinFile)
	if err != nil {
		return runner.ExecResult{}, err
	}
	restored := false
	defer func() {
		if restored {
			return
		}
		if err := terminal.Restore(stdinFile, state); returnErr == nil && err != nil {
			returnErr = err
		}
	}()

	resize := make(chan runner.TerminalSize, 1)
	resizeSignals := make(chan os.Signal, 1)
	signal.Notify(resizeSignals, syscall.SIGWINCH)
	resizeDone := make(chan struct{})
	go func() {
		for {
			select {
			case <-resizeDone:
				return
			case <-resizeSignals:
				newWidth, newHeight, err := terminal.Size(stdoutFile)
				if err != nil {
					continue
				}
				sendLatestResize(resize, runner.TerminalSize{
					Width:  newWidth,
					Height: newHeight,
				})
			}
		}
	}()

	env := map[string]string{}
	if value := os.Getenv("TERM"); value != "" {
		env["TERM"] = value
	}

	shellInput := io.Reader(stdinFile)
	shellArgv := []string{"/usr/bin/bash", "-l"}
	if evidence != nil {
		shellInput = &ctrlZObservingTerminalReader{file: stdinFile, evidence: evidence}
		shellArgv = []string{"/usr/bin/bash", "--noprofile", "--rcfile", jobControlShellRCPath, "-i"}
	}

	result, execErr := backend.Exec(ctx, instance, runner.ExecRequest{
		Argv:        shellArgv,
		Env:         env,
		Stdin:       shellInput,
		Stdout:      stdoutFile,
		Stderr:      stdoutFile,
		TTY:         true,
		InitialSize: runner.TerminalSize{Width: width, Height: height},
		Resize:      resize,
	})

	signal.Stop(resizeSignals)
	close(resizeDone)
	if err := terminal.Restore(stdinFile, state); err != nil {
		return runner.ExecResult{}, err
	}
	restored = true

	if evidence != nil {
		if evidenceErr := collectJobControlEvidence(ctx, backend, instance, evidence); evidenceErr != nil {
			if execErr != nil {
				return runner.ExecResult{}, errors.Join(execErr, evidenceErr)
			}
			return runner.ExecResult{}, evidenceErr
		}
	}
	if execErr != nil {
		return runner.ExecResult{}, execErr
	}
	return result, nil
}

func sendLatestResize(destination chan runner.TerminalSize, size runner.TerminalSize) {
	select {
	case destination <- size:
		return
	default:
	}

	select {
	case <-destination:
	default:
	}
	select {
	case destination <- size:
	default:
	}
}

func printUsage(out io.Writer) {
	fmt.Fprint(out, `LPIC Daily

Usage:
  lpic tui                       open the interactive daily dashboard
  lpic notify [--force]           send today's desktop notification once
  lpic assess                     run the Phase-1 initial recall assessment
  lpic today [--quick]           build today's adaptive session from local progress
  lpic learn <lesson-id>          read a lesson and record exposure when confirmed
  lpic question <question-id>     answer a deterministic question and record evidence
  lpic validate                  validate embedded curriculum and labs
  lpic doctor                    check local prerequisites without changing the host
  lpic labs                      list built-in labs (alias of "lpic lab list")
  lpic lab list                  list built-in labs
  lpic lab show <id>             show a lab without spoilers
  lpic lab run <id>              run a lab through its declared isolated backend
  lpic lab hint <id> <1-4>       reveal one graduated hint
  lpic lab debrief <id>          reveal the lab debrief
  lpic version                   print application version
  lpic help                      show this help
`)
}

func printLabUsage(out io.Writer) {
	fmt.Fprint(out, `Usage:
  lpic lab list
  lpic lab show <id>
  lpic lab run <id>
  lpic lab hint <id> <1-4>
  lpic lab debrief <id>
`)
}
