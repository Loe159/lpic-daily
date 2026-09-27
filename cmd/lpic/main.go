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
	"github.com/Loe159/lpic-daily/internal/content"
	"github.com/Loe159/lpic-daily/internal/curriculum"
	"github.com/Loe159/lpic-daily/internal/desktop"
	"github.com/Loe159/lpic-daily/internal/doctor"
	"github.com/Loe159/lpic-daily/internal/gamification"
	"github.com/Loe159/lpic-daily/internal/lab"
	"github.com/Loe159/lpic-daily/internal/learning"
	progresssqlite "github.com/Loe159/lpic-daily/internal/progress/sqlite"
	"github.com/Loe159/lpic-daily/internal/runner"
	podmanrunner "github.com/Loe159/lpic-daily/internal/runner/podman"
	"github.com/Loe159/lpic-daily/internal/study"
	"github.com/Loe159/lpic-daily/internal/terminal"
	lpicui "github.com/Loe159/lpic-daily/internal/tui"
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
	if !force {
		sent, err := store.NotificationSent(ctx, localDay)
		if err != nil {
			return err
		}
		if sent {
			fmt.Fprintln(stdout, "Notification quotidienne déjà envoyée.")
			return nil
		}
	}

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
	newItems := 0
	for _, item := range plan.Items {
		if item.Kind == learning.SessionReview {
			reviews++
		} else {
			newItems++
		}
	}
	body := fmt.Sprintf(
		"%d activité(s) due(s) · %d révision(s) · %d nouveau(x) concept(s)",
		len(plan.Items),
		reviews,
		newItems,
	)
	open, err := desktop.SendDaily(ctx, executor, desktop.Notification{
		Title: "LPIC Daily",
		Body:  body,
	})
	if err != nil {
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
	reader := bufio.NewReader(input)
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
			fmt.Fprintf(stdout, "Indice %d/4\n%s\n", hint.Level, hint.ContentFR)
			if hint.EvidenceImpact == "solution-revealed" {
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
	ctx := context.Background()
	if authored.Definition.Environment.Backend != "podman" {
		return fmt.Errorf("backend %q is not implemented by the CLI yet", authored.Definition.Environment.Backend)
	}

	backend, err := podmanrunner.Open(ctx, "")
	if err != nil {
		return fmt.Errorf("open rootless Podman backend: %w", err)
	}
	session, err := lab.Start(ctx, authored, backend)
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
	fmt.Fprintln(stdout, "Mode commandes sandboxé. Chaque ligne est exécutée dans un nouveau shell du lab.")
	fmt.Fprintln(stdout, "Commandes LPIC Daily : :shell  :check  :hint  :quit")
	fmt.Fprintln(stdout)

	scanner := bufio.NewScanner(stdin)
	scanner.Buffer(make([]byte, 4096), 256<<10)
	nextHint := 0
	highestHintLevel := 0

	for {
		fmt.Fprint(stdout, "lpic> ")
		if !scanner.Scan() {
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
			nextHint++
			if hint.Level > highestHintLevel {
				highestHintLevel = hint.Level
			}
			fmt.Fprintf(stdout, "Indice %d/4 : %s\n", hint.Level, hint.ContentFR)
			if hint.EvidenceImpact == "solution-revealed" {
				fmt.Fprintln(stdout, "Cet indice révèle la solution et réduira la force de la preuve pratique.")
			}
			continue
		case ":shell":
			fmt.Fprintln(stdout, "Ouverture d'un shell persistant dans la sandbox. Tape exit ou Ctrl-D pour revenir.")
			result, err := runPersistentShell(ctx, backend, session.Instance, stdin, stdout)
			if err != nil {
				return fmt.Errorf("interactive sandbox shell: %w", err)
			}
			fmt.Fprintln(stdout, "\n[retour LPIC Daily]")
			if result.ExitCode != 0 {
				fmt.Fprintf(stderr, "[shell exit %d]\n", result.ExitCode)
			}
			continue
		case ":check":
			results, err := session.Evaluate(ctx)
			if err != nil {
				return fmt.Errorf("evaluate lab: %w", err)
			}
			passed := true
			for _, result := range results {
				state := "OK"
				if !result.Pass {
					state = "À CORRIGER"
					passed = false
				}
				fmt.Fprintf(stdout, "  %-11s %s — %s\n", state, result.CheckID, result.Detail)
			}
			if passed {
				fmt.Fprintln(stdout, "\nLab réussi.")
				fmt.Fprintln(stdout, authored.Definition.DebriefFR)

				store, err := openProgressStore(ctx)
				if err != nil {
					return fmt.Errorf("lab succeeded but progress could not be opened: %w", err)
				}
				now := time.Now()
				recordErr := study.RecordLab(ctx, store, authored, highestHintLevel, now)
				if recordErr != nil {
					store.Close()
					return fmt.Errorf("lab succeeded but progress could not be recorded: %w", recordErr)
				}
				fmt.Fprintln(stdout, "Progression enregistrée.")
				recordGamificationBestEffort(
					ctx,
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
			continue
		}

		result, err := backend.Exec(ctx, session.Instance, runner.ExecRequest{
			Argv:   []string{"/usr/bin/bash", "-lc", line},
			Stdout: stdout,
			Stderr: stderr,
		})
		if err != nil {
			return fmt.Errorf("execute sandbox command: %w", err)
		}
		if result.ExitCode != 0 {
			fmt.Fprintf(stderr, "[exit %d]\n", result.ExitCode)
		}
	}
}

func runPersistentShell(
	ctx context.Context,
	backend runner.Runner,
	instance runner.Instance,
	stdin io.Reader,
	stdout io.Writer,
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

	result, execErr := backend.Exec(ctx, instance, runner.ExecRequest{
		Argv:        []string{"/usr/bin/bash", "-l"},
		Env:         env,
		Stdin:       stdinFile,
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
  lpic today [--quick]           build today's adaptive session from local progress
  lpic learn <lesson-id>          read a lesson and record exposure when confirmed
  lpic question <question-id>     answer a deterministic question and record evidence
  lpic validate                  validate embedded curriculum and labs
  lpic doctor                    check local prerequisites without changing the host
  lpic labs                      list built-in labs (alias of "lpic lab list")
  lpic lab list                  list built-in labs
  lpic lab show <id>             show a lab without spoilers
  lpic lab run <id>              run a lab through rootless Podman
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
