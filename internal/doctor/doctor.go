package doctor

import (
	"fmt"
	"os"
	"path/filepath"
)

type Check struct {
	Name   string
	Status string
	Detail string
}

type Report struct {
	Checks []Check
}

func Run() Report {
	report := Report{}

	if os.Geteuid() == 0 {
		report.Checks = append(report.Checks, Check{
			Name:   "unprivileged-process",
			Status: "warn",
			Detail: "LPIC Daily is designed to run as a regular user, not root",
		})
	} else {
		report.Checks = append(report.Checks, Check{
			Name:   "unprivileged-process",
			Status: "ok",
			Detail: fmt.Sprintf("running as uid %d", os.Geteuid()),
		})
	}

	socket := podmanSocket()
	if _, err := os.Stat(socket); err == nil {
		report.Checks = append(report.Checks, Check{
			Name:   "rootless-podman-socket",
			Status: "ok",
			Detail: socket,
		})
	} else {
		report.Checks = append(report.Checks, Check{
			Name:   "rootless-podman-socket",
			Status: "warn",
			Detail: fmt.Sprintf("%s not available yet; Phase 1 runner will fail closed until configured", socket),
		})
	}

	return report
}

func podmanSocket() string {
	if runtimeDir := os.Getenv("XDG_RUNTIME_DIR"); runtimeDir != "" {
		return filepath.Join(runtimeDir, "podman", "podman.sock")
	}
	return fmt.Sprintf("/run/user/%d/podman/podman.sock", os.Geteuid())
}
