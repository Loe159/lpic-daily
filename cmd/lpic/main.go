package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Loe159/lpic-daily/internal/curriculum"
	"github.com/Loe159/lpic-daily/internal/learning"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "validate":
		err = runValidate(os.Args[2:])
	case "plan":
		err = runPlan(os.Args[2:])
	case "help", "-h", "--help":
		usage()
		return
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: lpic <validate|plan> [options]")
}

func runValidate(args []string) error {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	dir := fs.String("curriculum", "curriculum/lpic-1-v5", "curriculum directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	catalog, err := curriculum.LoadDir(*dir)
	if err != nil {
		return err
	}
	fmt.Printf("OK %s %s: %d objectives, %d concepts, Phase 1=%v
",
		catalog.Certification, catalog.SyllabusVersion, len(catalog.Objectives), len(catalog.Concepts), catalog.Phase1Slice.SelectedObjectives)
	return nil
}

func runPlan(args []string) error {
	fs := flag.NewFlagSet("plan", flag.ContinueOnError)
	dir := fs.String("curriculum", "curriculum/lpic-1-v5", "curriculum directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	catalog, err := curriculum.LoadDir(*dir)
	if err != nil {
		return err
	}
	config := learning.DefaultSchedulerConfig(catalog.Phase1Slice.SelectedObjectives)
	scheduler, err := learning.NewScheduler(config)
	if err != nil {
		return err
	}
	plan, err := scheduler.Plan(catalog, nil, time.Now())
	if err != nil {
		return err
	}
	fmt.Printf("Reviews: %d
", len(plan.Reviews))
	if plan.New == nil {
		fmt.Println("New: none")
		return nil
	}
	fmt.Printf("New: %s — %s (%s)
", plan.New.ObjectiveID, plan.New.TitleFR, plan.New.ReasonFR)
	return nil
}
