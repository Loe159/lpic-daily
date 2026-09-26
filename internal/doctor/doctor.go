package doctor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	podmanrunner "github.com/Loe159/lpic-daily/internal/runner/podman"
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
	if _, err := os.Stat(socket); err != nil {
		report.Checks = append(report.Checks, Check{
			Name:   "rootless-podman-service",
			Status: "warn",
			Detail: fmt.Sprintf("%s is unavailable; labs fail closed until the rootless service is running", socket),
		})
		return report
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := podmanrunner.Open(ctx, "unix://"+socket); err != nil {
		report.Checks = append(report.Checks, Check{
			Name:   "rootless-podman-service",
			Status: "warn",
			Detail: fmt.Sprintf("socket exists but service is not usable: %v", err),
		})
		return report
	}

	report.Checks = append(report.Checks, Check{
		Name:   "rootless-podman-service",
		Status: "ok",
		Detail: fmt.Sprintf("%s responds as rootless Podman with cgroups v2", socket),
	})
	return report
}

func podmanSocket() string {
	if runtimeDir := os.Getenv("XDG_RUNTIME_DIR"); runtimeDir != "" {
		return filepath.Join(runtimeDir, "podman", "podman.sock")
	}
	return fmt.Sprintf("/run/user/%d/podman/podman.sock", os.Geteuid())
}
