package bootstrap

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Loe159/lpic-daily/internal/appstate"
	libvirtrunner "github.com/Loe159/lpic-daily/internal/runner/libvirt"
)

const (
	Phase1Image         = "localhost/lpic-daily/fedora-phase1:1"
	Phase1BaseImage     = "registry.fedoraproject.org/fedora:44"
	RequiredVMImageID   = "fedora-44-x86_64-v2"
	firstRunMarker      = "bootstrap-v1"
	notificationService = "packaging/systemd/lpic-daily-notify.service"
	notificationTimer   = "packaging/systemd/lpic-daily-notify.timer"
	desktopEntry        = "packaging/desktop/lpic-daily.desktop"
	vmProvisionScript   = "scripts/provision_vm_storage.sh"
	vmBuildScript       = "scripts/build_vm_image.py"
	vmSourcesManifest   = "packaging/vm-images/sources.json"
)

var ErrDeclined = errors.New("bootstrap action declined")

type Runner interface {
	LookPath(string) (string, error)
	Run(context.Context, string, ...string) ([]byte, error)
	Interactive(context.Context, io.Reader, io.Writer, io.Writer, string, ...string) error
}

type OSRunner struct{}

func (OSRunner) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

func (OSRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, name, args...)
	return command.CombinedOutput()
}

func (OSRunner) Interactive(
	ctx context.Context,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	name string,
	args ...string,
) error {
	command := exec.CommandContext(ctx, name, args...)
	command.Stdin = stdin
	command.Stdout = stdout
	command.Stderr = stderr
	return command.Run()
}

type Options struct {
	Runner    Runner
	Stdin     io.Reader
	Stdout    io.Writer
	Stderr    io.Writer
	PrepareVM bool
	AssumeYes bool
	FirstRun  bool
}

type installer struct {
	assets fs.FS
	opts   Options
	input  *bufio.Reader
}

func Install(ctx context.Context, assets fs.FS, opts Options) error {
	setup, err := newInstaller(assets, opts)
	if err != nil {
		return err
	}
	fmt.Fprintln(setup.opts.Stdout, "LPIC Daily — configuration de l'environnement")

	if err := setup.installUserBinary(); err != nil {
		return err
	}
	if err := setup.configureDesktopIntegration(ctx); err != nil {
		return err
	}
	if err := setup.ensurePodman(ctx); err != nil {
		if errors.Is(err, ErrDeclined) {
			fmt.Fprintln(setup.opts.Stdout, "Labs Podman: préparation ignorée; elle sera reproposée au premier lab.")
		} else {
			return err
		}
	}
	if setup.opts.PrepareVM {
		if err := setup.ensureVM(ctx, RequiredVMImageID); err != nil {
			if errors.Is(err, ErrDeclined) {
				fmt.Fprintln(setup.opts.Stdout, "Labs VM: préparation ignorée; elle sera reproposée au premier lab VM.")
			} else {
				return err
			}
		}
	} else {
		fmt.Fprintln(setup.opts.Stdout, "VM KVM/libvirt: préparation différée jusqu'au premier lab VM.")
	}

	if setup.opts.FirstRun {
		if err := writeFirstRunMarker(); err != nil {
			return fmt.Errorf("record initial setup: %w", err)
		}
	}
	fmt.Fprintln(setup.opts.Stdout, "Configuration LPIC Daily terminée.")
	return nil
}

func EnsureFirstRun(ctx context.Context, assets fs.FS, opts Options) error {
	done, err := firstRunCompleted()
	if err != nil {
		return err
	}
	if done {
		return nil
	}
	setup, err := newInstaller(assets, opts)
	if err != nil {
		return err
	}
	fmt.Fprintln(setup.opts.Stdout, "Première utilisation: configuration automatique de LPIC Daily…")
	if err := setup.installUserBinary(); err != nil {
		return err
	}
	if err := setup.configureDesktopIntegration(ctx); err != nil {
		return err
	}
	if _, err := setup.ensurePodmanRuntime(ctx); err != nil {
		return err
	}
	fmt.Fprintln(setup.opts.Stdout, "Les images de labs seront préparées automatiquement à leur première utilisation.")
	if err := writeFirstRunMarker(); err != nil {
		return fmt.Errorf("record initial setup: %w", err)
	}
	return nil
}

func EnsurePodmanLab(ctx context.Context, assets fs.FS, opts Options) error {
	setup, err := newInstaller(assets, opts)
	if err != nil {
		return err
	}
	return setup.ensurePodman(ctx)
}

func EnsureVMLab(ctx context.Context, assets fs.FS, opts Options) error {
	return EnsureVMLabImage(ctx, assets, RequiredVMImageID, opts)
}

func EnsureVMLabImage(ctx context.Context, assets fs.FS, imageID string, opts Options) error {
	if strings.TrimSpace(imageID) == "" {
		return errors.New("VM image ID is required")
	}
	setup, err := newInstaller(assets, opts)
	if err != nil {
		return err
	}
	return setup.ensureVM(ctx, imageID)
}

func newInstaller(assets fs.FS, opts Options) (*installer, error) {
	if assets == nil {
		return nil, errors.New("bootstrap assets are required")
	}
	if opts.Runner == nil {
		opts.Runner = OSRunner{}
	}
	if opts.Stdin == nil {
		opts.Stdin = os.Stdin
	}
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}
	return &installer{
		assets: assets,
		opts:   opts,
		input:  bufio.NewReader(opts.Stdin),
	}, nil
}

func (setup *installer) installUserBinary() error {
	if strings.TrimSpace(os.Getenv("LPIC_DAILY_SKIP_SELF_INSTALL")) == "1" {
		return nil
	}
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve current LPIC Daily executable: %w", err)
	}
	source, err := os.Open(executable)
	if err != nil {
		return fmt.Errorf("open current LPIC Daily executable: %w", err)
	}
	defer source.Close()

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	binDir := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		return fmt.Errorf("create user bin directory: %w", err)
	}
	targetPath := filepath.Join(binDir, "lpic")
	sourceInfo, err := source.Stat()
	if err != nil {
		return err
	}
	if existingInfo, statErr := os.Stat(targetPath); statErr == nil &&
		existingInfo.Size() == sourceInfo.Size() {
		if same, compareErr := filesEqual(executable, targetPath); compareErr == nil && same {
			return nil
		}
	}

	temp, err := os.CreateTemp(binDir, ".lpic-install-*")
	if err != nil {
		return fmt.Errorf("create temporary user binary: %w", err)
	}
	tempPath := temp.Name()
	cleanup := func() {
		_ = temp.Close()
		_ = os.Remove(tempPath)
	}
	if _, err := io.Copy(temp, source); err != nil {
		cleanup()
		return fmt.Errorf("copy LPIC Daily executable: %w", err)
	}
	if err := temp.Chmod(0o755); err != nil {
		cleanup()
		return err
	}
	if err := temp.Sync(); err != nil {
		cleanup()
		return err
	}
	if err := temp.Close(); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	if err := os.Rename(tempPath, targetPath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("install LPIC Daily executable: %w", err)
	}
	fmt.Fprintf(setup.opts.Stdout, "Binaire utilisateur: %s\n", targetPath)
	return nil
}

func filesEqual(left, right string) (bool, error) {
	leftFile, err := os.Open(left)
	if err != nil {
		return false, err
	}
	defer leftFile.Close()
	rightFile, err := os.Open(right)
	if err != nil {
		return false, err
	}
	defer rightFile.Close()
	leftInfo, err := leftFile.Stat()
	if err != nil {
		return false, err
	}
	rightInfo, err := rightFile.Stat()
	if err != nil {
		return false, err
	}
	if leftInfo.Size() != rightInfo.Size() {
		return false, nil
	}
	const chunk = 64 * 1024
	leftBuffer := make([]byte, chunk)
	rightBuffer := make([]byte, chunk)
	for {
		leftN, leftErr := leftFile.Read(leftBuffer)
		rightN, rightErr := rightFile.Read(rightBuffer)
		if leftN != rightN || !bytes.Equal(leftBuffer[:leftN], rightBuffer[:rightN]) {
			return false, nil
		}
		if leftErr == io.EOF && rightErr == io.EOF {
			return true, nil
		}
		if leftErr != nil && leftErr != io.EOF {
			return false, leftErr
		}
		if rightErr != nil && rightErr != io.EOF {
			return false, rightErr
		}
	}
}

func (setup *installer) configureDesktopIntegration(ctx context.Context) error {
	configHome, dataHome, err := userHomes()
	if err != nil {
		return err
	}
	systemdDir := filepath.Join(configHome, "systemd", "user")
	applicationsDir := filepath.Join(dataHome, "applications")
	if err := os.MkdirAll(systemdDir, 0o700); err != nil {
		return fmt.Errorf("create user systemd directory: %w", err)
	}
	if err := os.MkdirAll(applicationsDir, 0o700); err != nil {
		return fmt.Errorf("create applications directory: %w", err)
	}

	changed := false
	for _, target := range []struct {
		source string
		path   string
	}{
		{notificationService, filepath.Join(systemdDir, "lpic-daily-notify.service")},
		{notificationTimer, filepath.Join(systemdDir, "lpic-daily-notify.timer")},
		{desktopEntry, filepath.Join(applicationsDir, "lpic-daily.desktop")},
	} {
		updated, err := writeEmbeddedFile(setup.assets, target.source, target.path, 0o644)
		if err != nil {
			return err
		}
		changed = changed || updated
	}

	notifierReady, err := setup.ensureExecutable(ctx, "notify-send", []string{"libnotify"})
	if err != nil {
		return err
	}
	if _, err := setup.opts.Runner.LookPath("systemctl"); err != nil {
		fmt.Fprintln(setup.opts.Stderr, "Avertissement: systemctl absent; notifications automatiques non activées.")
		return nil
	}

	if changed {
		if output, err := setup.opts.Runner.Run(ctx, "systemctl", "--user", "daemon-reload"); err != nil {
			fmt.Fprintf(setup.opts.Stderr, "Avertissement: systemctl --user daemon-reload: %v (%s)\n", err, strings.TrimSpace(string(output)))
		}
	}
	if notifierReady {
		if output, err := setup.opts.Runner.Run(
			ctx,
			"systemctl",
			"--user",
			"enable",
			"--now",
			"lpic-daily-notify.timer",
		); err != nil {
			fmt.Fprintf(setup.opts.Stderr, "Avertissement: activation des notifications: %v (%s)\n", err, strings.TrimSpace(string(output)))
		} else {
			fmt.Fprintln(setup.opts.Stdout, "Notifications quotidiennes: configurées.")
		}
	}
	return nil
}

func (setup *installer) ensurePodman(ctx context.Context) error {
	ready, err := setup.ensurePodmanRuntime(ctx)
	if err != nil {
		return err
	}
	if !ready {
		return fmt.Errorf("%w: Podman installation was not approved", ErrDeclined)
	}

	revision, err := phase1ImageRevision(setup.assets)
	if err != nil {
		return err
	}
	if _, err := setup.opts.Runner.Run(ctx, "podman", "image", "exists", Phase1Image); err == nil {
		output, inspectErr := setup.opts.Runner.Run(
			ctx,
			"podman",
			"image",
			"inspect",
			"--format",
			`{{ index .Labels "io.lpic-daily.source-digest" }}`,
			Phase1Image,
		)
		if inspectErr == nil && strings.TrimSpace(string(output)) == revision {
			fmt.Fprintln(setup.opts.Stdout, "Image des labs Podman: prête.")
			return nil
		}
		fmt.Fprintln(setup.opts.Stdout, "Image des labs Podman obsolète: reconstruction automatique…")
	}

	if !setup.confirm(
		"La première préparation du lab Podman construit une image Fedora et peut télécharger des paquets. Continuer ?",
		true,
	) {
		return fmt.Errorf("%w: Phase-1 lab image build was not approved", ErrDeclined)
	}

	if _, err := setup.opts.Runner.Run(ctx, "podman", "image", "exists", Phase1BaseImage); err != nil {
		if !setup.confirm(
			"Image de base Fedora 44 absente. Autoriser son téléchargement depuis registry.fedoraproject.org ?",
			true,
		) {
			return fmt.Errorf("%w: Phase-1 base image download was not approved", ErrDeclined)
		}
		fmt.Fprintln(setup.opts.Stdout, "Téléchargement explicite de l'image de base Fedora 44…")
		if err := setup.opts.Runner.Interactive(
			ctx,
			setup.opts.Stdin,
			setup.opts.Stdout,
			setup.opts.Stderr,
			"podman",
			"pull",
			Phase1BaseImage,
		); err != nil {
			return fmt.Errorf("pull Phase-1 Fedora base image: %w", err)
		}
	}

	fmt.Fprintln(setup.opts.Stdout, "Image des labs Podman absente: construction automatique…")
	contextDir, cleanup, err := setup.extractPhase1Context()
	if err != nil {
		return err
	}
	defer cleanup()
	if err := setup.opts.Runner.Interactive(
		ctx,
		setup.opts.Stdin,
		setup.opts.Stdout,
		setup.opts.Stderr,
		"podman",
		"build",
		"--pull=never",
		"--label",
		"io.lpic-daily.source-digest="+revision,
		"--tag",
		Phase1Image,
		"--file",
		filepath.Join(contextDir, "Containerfile"),
		contextDir,
	); err != nil {
		return fmt.Errorf("build Phase-1 Podman image: %w", err)
	}
	if _, err := setup.opts.Runner.Run(ctx, "podman", "image", "exists", Phase1Image); err != nil {
		return errors.New("Phase-1 Podman image build completed but the expected image is still unavailable")
	}
	fmt.Fprintln(setup.opts.Stdout, "Image des labs Podman: installée.")
	return nil
}

func (setup *installer) ensurePodmanRuntime(ctx context.Context) (bool, error) {
	ready, err := setup.ensureExecutable(ctx, "podman", []string{"podman"})
	if err != nil {
		return false, err
	}
	if !ready {
		fmt.Fprintln(setup.opts.Stderr, "Avertissement: Podman absent; les labs conteneur resteront indisponibles.")
		return false, nil
	}

	if _, err := setup.opts.Runner.LookPath("systemctl"); err != nil {
		fmt.Fprintln(setup.opts.Stderr, "Avertissement: systemctl absent; impossible d'activer automatiquement le socket rootless Podman.")
		return true, nil
	}
	output, runErr := setup.opts.Runner.Run(ctx, "systemctl", "--user", "enable", "--now", "podman.socket")
	if runErr != nil {
		return false, fmt.Errorf("start rootless Podman socket: %w (%s)", runErr, strings.TrimSpace(string(output)))
	}
	fmt.Fprintln(setup.opts.Stdout, "Podman rootless: prêt.")
	return true, nil
}

func (setup *installer) ensureVM(ctx context.Context, imageID string) error {
	required := []struct {
		executable string
		packages   []string
	}{
		{"qemu-img", []string{"qemu-img"}},
		{"virt-customize", []string{"guestfs-tools"}},
		{"virsh", []string{"libvirt-client"}},
		{"virtqemud", []string{"libvirt-daemon-kvm"}},
	}
	for _, requirement := range required {
		ready, err := setup.ensureExecutable(ctx, requirement.executable, requirement.packages)
		if err != nil {
			return err
		}
		if !ready {
			return fmt.Errorf("%w: %s installation was not approved", ErrDeclined, requirement.executable)
		}
	}

	imageRoot, err := appstate.VMImageRoot()
	if err != nil {
		return err
	}
	catalogPath, err := appstate.VMImageCatalogPath()
	if err != nil {
		return err
	}
	if catalog, loadErr := libvirtrunner.LoadImageCatalog(catalogPath, imageRoot); loadErr == nil {
		if _, resolveErr := catalog.Resolve(imageID, imageRoot); resolveErr == nil {
			fmt.Fprintf(setup.opts.Stdout, "Image VM %s: prête.\n", imageID)
			return nil
		}
	}

	if err := setup.ensureVMStorage(ctx); err != nil {
		return err
	}
	if !setup.confirm(
		fmt.Sprintf("Le lab VM nécessite le téléchargement et la construction de l'image de confiance %s. Continuer ?", imageID),
		false,
	) {
		return fmt.Errorf("%w: VM image preparation was not approved", ErrDeclined)
	}

	root, cleanup, err := setup.extractVMBuildTree()
	if err != nil {
		return err
	}
	defer cleanup()
	fmt.Fprintf(setup.opts.Stdout, "Construction de l'image VM %s (téléchargement vérifié + personnalisation)…\n", imageID)
	if err := setup.opts.Runner.Interactive(
		ctx,
		setup.opts.Stdin,
		setup.opts.Stdout,
		setup.opts.Stderr,
		"python3",
		filepath.Join(root, "scripts", "build_vm_image.py"),
		imageID,
		"--image-root",
		imageRoot,
	); err != nil {
		return fmt.Errorf("build trusted VM image: %w", err)
	}

	catalog, err := libvirtrunner.LoadImageCatalog(catalogPath, imageRoot)
	if err != nil {
		return fmt.Errorf("validate generated VM image catalog: %w", err)
	}
	if _, err := catalog.Resolve(imageID, imageRoot); err != nil {
		return fmt.Errorf("validate generated VM image: %w", err)
	}
	fmt.Fprintf(setup.opts.Stdout, "Image VM %s: installée et vérifiée.\n", imageID)
	return nil
}

func (setup *installer) ensureVMStorage(ctx context.Context) error {
	imageRoot, imageErr := appstate.VMImageRoot()
	stateRoot, stateErr := appstate.VMStateRoot()
	lockPath := appstate.VMNetworkAllocationLockPath()
	if imageErr == nil && stateErr == nil && directoryExists(imageRoot) && directoryExists(stateRoot) && regularFileExists(lockPath) {
		return nil
	}
	if !setup.confirm(
		"Le stockage qemu:///system doit être provisionné avec sudo. Autoriser cette étape ?",
		false,
	) {
		return fmt.Errorf("%w: VM storage provisioning was not approved", ErrDeclined)
	}
	if _, err := setup.opts.Runner.LookPath("sudo"); err != nil {
		return errors.New("sudo is required to provision system-libvirt storage")
	}
	root, cleanup, err := setup.extractFiles(map[string]os.FileMode{
		vmProvisionScript: 0o755,
	})
	if err != nil {
		return err
	}
	defer cleanup()
	if err := setup.opts.Runner.Interactive(
		ctx,
		setup.opts.Stdin,
		setup.opts.Stdout,
		setup.opts.Stderr,
		"sudo",
		filepath.Join(root, vmProvisionScript),
	); err != nil {
		return fmt.Errorf("provision VM storage: %w", err)
	}
	return nil
}

func (setup *installer) ensureExecutable(
	ctx context.Context,
	executable string,
	fedoraPackages []string,
) (bool, error) {
	if _, err := setup.opts.Runner.LookPath(executable); err == nil {
		return true, nil
	}
	if distroID() != "fedora" || len(fedoraPackages) == 0 {
		fmt.Fprintf(setup.opts.Stderr, "Avertissement: %s est absent; installation automatique non disponible sur cette distribution.\n", executable)
		return false, nil
	}
	packages := append([]string(nil), fedoraPackages...)
	sort.Strings(packages)
	prompt := fmt.Sprintf(
		"%s est absent. Installer via « sudo dnf install -y %s » ?",
		executable,
		strings.Join(packages, " "),
	)
	if !setup.confirm(prompt, false) {
		return false, nil
	}
	if _, err := setup.opts.Runner.LookPath("sudo"); err != nil {
		return false, errors.New("sudo is required to install missing Fedora packages")
	}
	args := append([]string{"dnf", "install", "-y"}, packages...)
	if err := setup.opts.Runner.Interactive(
		ctx,
		setup.opts.Stdin,
		setup.opts.Stdout,
		setup.opts.Stderr,
		"sudo",
		args...,
	); err != nil {
		return false, fmt.Errorf("install Fedora package(s) %s: %w", strings.Join(packages, ", "), err)
	}
	if _, err := setup.opts.Runner.LookPath(executable); err != nil {
		return false, fmt.Errorf("%s is still unavailable after package installation", executable)
	}
	return true, nil
}

func (setup *installer) confirm(question string, defaultYes bool) bool {
	if setup.opts.AssumeYes {
		fmt.Fprintf(setup.opts.Stdout, "%s oui (--yes)\n", question)
		return true
	}
	suffix := " [o/N] "
	if defaultYes {
		suffix = " [O/n] "
	}
	fmt.Fprint(setup.opts.Stdout, question+suffix)
	answer, err := setup.input.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		fmt.Fprintf(setup.opts.Stderr, "Avertissement: lecture de la confirmation: %v\n", err)
		return false
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	if answer == "" {
		return defaultYes
	}
	return answer == "o" || answer == "oui" || answer == "y" || answer == "yes"
}

func phase1ImageRevision(assets fs.FS) (string, error) {
	digest := sha256.New()
	for _, source := range []string{
		"labs/images/fedora-phase1/Containerfile",
		"labs/images/fedora-phase1/report-status-approved",
		"labs/images/fedora-phase1/report-status-shadow",
	} {
		payload, err := fs.ReadFile(assets, source)
		if err != nil {
			return "", fmt.Errorf("read Phase-1 image source %s: %w", source, err)
		}
		_, _ = io.WriteString(digest, source)
		_, _ = digest.Write([]byte{0})
		_, _ = digest.Write(payload)
		_, _ = digest.Write([]byte{0})
	}
	return fmt.Sprintf("%x", digest.Sum(nil)), nil
}

func (setup *installer) extractPhase1Context() (string, func(), error) {
	files := map[string]os.FileMode{
		"labs/images/fedora-phase1/Containerfile":          0o644,
		"labs/images/fedora-phase1/report-status-approved": 0o755,
		"labs/images/fedora-phase1/report-status-shadow":   0o755,
	}
	root, cleanup, err := setup.extractFiles(files)
	if err != nil {
		return "", nil, err
	}
	return filepath.Join(root, "labs", "images", "fedora-phase1"), cleanup, nil
}

func (setup *installer) extractVMBuildTree() (string, func(), error) {
	return setup.extractFiles(map[string]os.FileMode{
		vmBuildScript:     0o755,
		vmSourcesManifest: 0o644,
	})
}

func (setup *installer) extractFiles(files map[string]os.FileMode) (string, func(), error) {
	root, err := os.MkdirTemp("", "lpic-daily-bootstrap-*")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(root) }
	for source, mode := range files {
		payload, err := fs.ReadFile(setup.assets, source)
		if err != nil {
			cleanup()
			return "", nil, fmt.Errorf("read embedded bootstrap asset %s: %w", source, err)
		}
		target := filepath.Join(root, filepath.FromSlash(source))
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			cleanup()
			return "", nil, err
		}
		if err := os.WriteFile(target, payload, mode); err != nil {
			cleanup()
			return "", nil, err
		}
	}
	return root, cleanup, nil
}

func writeEmbeddedFile(assets fs.FS, source, target string, mode os.FileMode) (bool, error) {
	payload, err := fs.ReadFile(assets, source)
	if err != nil {
		return false, fmt.Errorf("read embedded asset %s: %w", source, err)
	}
	current, err := os.ReadFile(target)
	if err == nil && bytes.Equal(current, payload) {
		return false, nil
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	if err := os.WriteFile(target, payload, mode); err != nil {
		return false, fmt.Errorf("write %s: %w", target, err)
	}
	return true, nil
}

func firstRunCompleted() (bool, error) {
	path, err := markerPath()
	if err != nil {
		return false, err
	}
	_, err = os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func writeFirstRunMarker() error {
	path, err := markerPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte("1\n"), 0o600)
}

func markerPath() (string, error) {
	base := strings.TrimSpace(os.Getenv("XDG_STATE_HOME"))
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".local", "state")
	}
	if !filepath.IsAbs(base) {
		return "", errors.New("XDG_STATE_HOME must be absolute")
	}
	return filepath.Join(filepath.Clean(base), "lpic-daily", firstRunMarker), nil
}

func userHomes() (string, string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}
	configHome := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
	if configHome == "" {
		configHome = filepath.Join(home, ".config")
	}
	dataHome := strings.TrimSpace(os.Getenv("XDG_DATA_HOME"))
	if dataHome == "" {
		dataHome = filepath.Join(home, ".local", "share")
	}
	if !filepath.IsAbs(configHome) || !filepath.IsAbs(dataHome) {
		return "", "", errors.New("XDG_CONFIG_HOME and XDG_DATA_HOME must be absolute")
	}
	return filepath.Clean(configHome), filepath.Clean(dataHome), nil
}

func distroID() string {
	payload, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(payload), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok || key != "ID" {
			continue
		}
		return strings.Trim(strings.TrimSpace(value), "\"'")
	}
	return ""
}

func directoryExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func regularFileExists(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0
}
