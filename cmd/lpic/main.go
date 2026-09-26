package main

import (
	"fmt"
	"os"

	lpicdaily "github.com/Loe159/lpic-daily"
	"github.com/Loe159/lpic-daily/internal/curriculum"
	"github.com/Loe159/lpic-daily/internal/doctor"
)

const version = "0.0.0-dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}

	switch args[0] {
	case "validate":
		bundle, err := curriculum.Load(lpicdaily.BuiltinFS)
		if err != nil {
			return err
		}
		fmt.Println("builtin curriculum OK")
		fmt.Println(curriculum.FormatSummary(curriculum.Summarize(bundle)))
		return nil
	case "doctor":
		if _, err := curriculum.Load(lpicdaily.BuiltinFS); err != nil {
			return fmt.Errorf("builtin curriculum: %w", err)
		}
		for _, check := range doctor.Run().Checks {
			fmt.Printf("%-24s %-5s %s\n", check.Name, check.Status, check.Detail)
		}
		return nil
	case "version":
		fmt.Println(version)
		return nil
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command %q (try: lpic help)", args[0])
	}
}

func printUsage() {
	fmt.Print(`LPIC Daily

Usage:
  lpic validate   validate the embedded LPIC curriculum/contracts
  lpic doctor     check local prerequisites without changing the host
  lpic version    print application version
  lpic help       show this help

The learning TUI and Podman lab runner are introduced later in Phase 1.
`)
}
