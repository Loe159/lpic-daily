package doctor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Loe159/lpic-daily/internal/appstate"
	libvirtrunner "github.com/Loe159/lpic-daily/internal/runner/libvirt"
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

	report.Checks = append(report.Checks, podmanCheck())
	report.Checks = append(report.Checks, executableCheck(
		"qemu-img",
		"qemu-img",
		"qemu-img is available for disposable VM overlays",
		"qemu-img is missing; full-system labs are unavailable",
	))
	report.Checks = append(report.Checks, vmStorageCheck())
	report.Checks = append(report.Checks, vmImageCatalogCheck())
	report.Checks = append(report.Checks, systemLibvirtCheck())
	report.Checks = append(report.Checks, executableCheck(
		"desktop-notifications",
		"notify-send",
		"desktop notifications available",
		"notify-send is missing; daily desktop notifications are unavailable",
	))

	launcher := "xdg-terminal-exec"
	if override := strings.TrimSpace(os.Getenv("LPIC_DAILY_TERMINAL_LAUNCHER")); override != "" {
		if fields := strings.Fields(override); len(fields) != 0 {
			launcher = fields[0]
		}
	}
	report.Checks = append(report.Checks, executableCheck(
		"terminal-launcher",
		launcher,
		fmt.Sprintf("%s can launch the daily TUI", launcher),
		fmt.Sprintf("%s is missing; notification actions cannot open the TUI", launcher),
	))

	return report
}

func podmanCheck() Check {
	socket := podmanSocket()
	if _, err := os.Stat(socket); err != nil {
		return Check{
			Name:   "rootless-podman-service",
			Status: "warn",
			Detail: fmt.Sprintf("%s is unavailable; labs fail closed until the rootless service is running", socket),
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := podmanrunner.Open(ctx, "unix://"+socket); err != nil {
		return Check{
			Name:   "rootless-podman-service",
			Status: "warn",
			Detail: fmt.Sprintf("socket exists but service is not usable: %v", err),
		}
	}

	return Check{
		Name:   "rootless-podman-service",
		Status: "ok",
		Detail: fmt.Sprintf("%s responds as rootless Podman with cgroups v2", socket),
	}
}

func vmStorageCheck() Check {
	imageRoot, imageErr := appstate.VMImageRoot()
	stateRoot, stateErr := appstate.VMStateRoot()
	if imageErr != nil {
		return Check{Name: "vm-storage", Status: "warn", Detail: imageErr.Error()}
	}
	if stateErr != nil {
		return Check{Name: "vm-storage", Status: "warn", Detail: stateErr.Error()}
	}
	for _, root := range []string{imageRoot, stateRoot} {
		info, err := os.Stat(root)
		if err != nil {
			return Check{
				Name: "vm-storage", Status: "warn",
				Detail: fmt.Sprintf("%s is not provisioned: %v; run sudo scripts/provision_vm_storage.sh", root, err),
			}
		}
		if !info.IsDir() {
			return Check{Name: "vm-storage", Status: "warn", Detail: fmt.Sprintf("%s is not a directory", root)}
		}
		probe, err := os.CreateTemp(root, ".lpic-daily-write-probe-*")
		if err != nil {
			return Check{Name: "vm-storage", Status: "warn", Detail: fmt.Sprintf("%s is not writable: %v", root, err)}
		}
		probePath := probe.Name()
		_ = probe.Close()
		_ = os.Remove(probePath)
	}
	return Check{
		Name: "vm-storage", Status: "ok",
		Detail: fmt.Sprintf("system-libvirt storage is provisioned: images=%s state=%s", imageRoot, stateRoot),
	}
}

func vmImageCatalogCheck() Check {
	imageRoot, err := appstate.VMImageRoot()
	if err != nil {
		return Check{Name: "vm-image-catalog", Status: "warn", Detail: err.Error()}
	}
	catalogPath, err := appstate.VMImageCatalogPath()
	if err != nil {
		return Check{Name: "vm-image-catalog", Status: "warn", Detail: err.Error()}
	}
	catalog, err := libvirtrunner.LoadImageCatalog(catalogPath, imageRoot)
	if err != nil {
		return Check{
			Name:   "vm-image-catalog",
			Status: "warn",
			Detail: fmt.Sprintf("%s is unavailable or invalid: %v", catalogPath, err),
		}
	}
	return Check{
		Name:   "vm-image-catalog",
		Status: "ok",
		Detail: fmt.Sprintf("%d trusted VM image(s) declared in %s", len(catalog.Images), catalogPath),
	}
}

func systemLibvirtCheck() Check {
	type probeResult struct {
		control *libvirtrunner.RPCControlPlane
		err     error
	}

	results := make(chan probeResult, 1)
	abandoned := make(chan struct{})
	go func() {
		control, err := libvirtrunner.OpenSystem()
		result := probeResult{control: control, err: err}
		select {
		case results <- result:
		case <-abandoned:
			if control != nil {
				_ = control.Close()
			}
		}
	}()

	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()

	select {
	case result := <-results:
		if result.err != nil {
			return Check{
				Name:   "system-libvirt",
				Status: "warn",
				Detail: fmt.Sprintf("qemu:///system is unavailable: %v", result.err),
			}
		}
		if err := result.control.Close(); err != nil {
			return Check{
				Name:   "system-libvirt",
				Status: "warn",
				Detail: fmt.Sprintf("qemu:///system connected but did not close cleanly: %v", err),
			}
		}
		return Check{
			Name:   "system-libvirt",
			Status: "ok",
			Detail: "qemu:///system is reachable and advertises x86_64",
		}
	case <-timer.C:
		close(abandoned)
		return Check{
			Name:   "system-libvirt",
			Status: "warn",
			Detail: "qemu:///system probe timed out after 2s; full-system labs remain fail-closed",
		}
	}
}

func executableCheck(name, executable, okDetail, missingDetail string) Check {
	if _, err := exec.LookPath(executable); err != nil {
		return Check{Name: name, Status: "warn", Detail: missingDetail}
	}
	return Check{Name: name, Status: "ok", Detail: okDetail}
}

func podmanSocket() string {
	if runtimeDir := os.Getenv("XDG_RUNTIME_DIR"); runtimeDir != "" {
		return filepath.Join(runtimeDir, "podman", "podman.sock")
	}
	return fmt.Sprintf("/run/user/%d/podman/podman.sock", os.Geteuid())
}
