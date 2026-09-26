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
	"github.com/Loe159/lpic-daily/internal/curriculum"
	"github.com/Loe159/lpic-daily/internal/doctor"
	"github.com/Loe159/lpic-daily/internal/lab"
	"github.com/Loe159/lpic-daily/internal/runner"
	podmanrunner "github.com/Loe159/lpic-daily/internal/runner/podman"
	"github.com/Loe159/lpic-daily/internal/terminal"
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
		fmt.Fprintf(stdout, "builtin labs OK: %d\n", len(labs))
		return nil
	case "doctor":
		if _, err := curriculum.Load(lpicdaily.BuiltinFS); err != nil {
			return fmt.Errorf("builtin curriculum: %w", err)
		}
		if _, err := lab.LoadAll(lpicdaily.BuiltinFS); err != nil {
			return fmt.Errorf("builtin labs: %w", err)
		}
		for _, check := range doctor.Run().Checks {
			fmt.Fprintf(stdout, "%-24s %-5s %s\n", check.Name, check.Status, check.Detail)
		}
		return nil
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
