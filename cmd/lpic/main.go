package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	lpicdaily "github.com/Loe159/lpic-daily"
	"github.com/Loe159/lpic-daily/internal/curriculum"
	"github.com/Loe159/lpic-daily/internal/doctor"
	"github.com/Loe159/lpic-daily/internal/lab"
)

const version = "0.0.0-dev"

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
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
		return runLabs(stdout)
	case "lab":
		return runLab(args[1:], stdout)
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

func runLabs(stdout io.Writer) error {
	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		return err
	}
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
}

func runLab(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		printLabUsage(stdout)
		return nil
	}

	authored, err := findLab(args)
	if err != nil {
		return err
	}

	switch args[0] {
	case "show":
		if len(args) != 2 {
			return fmt.Errorf("usage: lpic lab show <lab-id>")
		}
		printLab(authored, stdout)
		return nil
	case "hint":
		if len(args) != 3 {
			return fmt.Errorf("usage: lpic lab hint <lab-id> <1-4>")
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
		fmt.Fprintf(stdout, "%s\n\n%s\n", authored.Definition.TitleFR, authored.Definition.DebriefFR)
		return nil
	case "help", "-h", "--help":
		printLabUsage(stdout)
		return nil
	default:
		return fmt.Errorf("unknown lab command %q", args[0])
	}
}

func findLab(args []string) (lab.Lab, error) {
	if len(args) < 2 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		return lab.Lab{}, nil
	}

	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		return lab.Lab{}, err
	}
	for _, authored := range labs {
		if authored.Definition.ID == args[1] {
			return authored, nil
		}
	}
	return lab.Lab{}, fmt.Errorf("unknown lab %q", args[1])
}

func printLab(authored lab.Lab, stdout io.Writer) {
	definition := authored.Definition
	fmt.Fprintf(stdout, "%s\n", definition.TitleFR)
	fmt.Fprintf(stdout, "ID: %s\n", definition.ID)
	fmt.Fprintf(stdout, "Objectifs LPIC: %s\n", strings.Join(definition.ObjectiveIDs, ", "))
	fmt.Fprintf(stdout, "Environnement: %s / %s / network=%s\n", definition.Environment.Backend, definition.Environment.Distribution, definition.Environment.Network)
	fmt.Fprintf(stdout, "Durée estimée: %d min\n", definition.EstimatedMinutes)
	fmt.Fprintf(stdout, "Labels: %s\n\n", strings.Join(definition.Labels, ", "))
	fmt.Fprintln(stdout, definition.BriefFR)
	fmt.Fprintln(stdout, "\nCritères de réussite:")
	for _, criterion := range definition.SuccessCriteriaFR {
		fmt.Fprintf(stdout, "- %s\n", criterion)
	}
	fmt.Fprintf(stdout, "\n%d niveaux d'indices disponibles. Le debrief est masqué jusqu'à demande explicite.\n", len(authored.Hints))
}

func printUsage(stdout io.Writer) {
	fmt.Fprint(stdout, `LPIC Daily

Usage:
  lpic validate                  validate embedded curriculum and labs
  lpic doctor                    check local prerequisites without changing the host
  lpic labs                      list built-in labs
  lpic lab show <id>             show a lab without spoilers
  lpic lab hint <id> <1-4>       reveal one graduated hint
  lpic lab debrief <id>          reveal the lab debrief
  lpic version                   print application version
  lpic help                      show this help

The interactive lab terminal and daily TUI are introduced incrementally during Phase 1.
`)
}

func printLabUsage(stdout io.Writer) {
	fmt.Fprint(stdout, `Usage:
  lpic lab show <id>
  lpic lab hint <id> <1-4>
  lpic lab debrief <id>
`)
}
