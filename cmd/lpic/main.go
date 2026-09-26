package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	lpicdaily "github.com/Loe159/lpic-daily"
	"github.com/Loe159/lpic-daily/internal/curriculum"
	"github.com/Loe159/lpic-daily/internal/doctor"
	"github.com/Loe159/lpic-daily/internal/lab"
	"github.com/Loe159/lpic-daily/internal/runner"
	podmanrunner "github.com/Loe159/lpic-daily/internal/runner/podman"
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
		fmt.Fprintf(stdout, "builtin labs: %d OK\n", len(labs))
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
			fmt.Fprintf(
				stdout,
				"%-32s %3d min  %s\n",
				authored.Definition.ID,
				authored.Definition.EstimatedMinutes,
				authored.Definition.TitleFR,
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
	fmt.Fprintf(out, "%s\n%s\n\n", authored.Definition.TitleFR, authored.Definition.ID)
	fmt.Fprintln(out, authored.Definition.BriefFR)
	fmt.Fprintln(out, "\nCritères de réussite :")
	for _, criterion := range authored.Definition.SuccessCriteriaFR {
		fmt.Fprintf(out, "  - %s\n", criterion)
	}
	fmt.Fprintf(
		out,
		"\nDurée estimée : %d min | backend=%s | réseau=%s | hints=%d\n",
		authored.Definition.EstimatedMinutes,
		authored.Definition.Environment.Backend,
		authored.Definition.Environment.Network,
		len(authored.Hints),
	)
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
	fmt.Fprintln(stdout, "Commandes LPIC Daily : :check  :hint  :quit")
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

func printUsage(out io.Writer) {
	fmt.Fprint(out, `LPIC Daily

Usage:
  lpic validate             validate embedded curriculum, contracts and labs
  lpic doctor               check local prerequisites without changing the host
  lpic lab list             list built-in labs
  lpic lab show <lab-id>    show a lab brief without revealing its solution
  lpic lab run <lab-id>     run a lab through rootless Podman
  lpic version              print application version
  lpic help                 show this help
`)
}

func printLabUsage(out io.Writer) {
	fmt.Fprint(out, `Usage:
  lpic lab list
  lpic lab show <lab-id>
  lpic lab run <lab-id>
`)
}
