package selfupdate

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const mainPackage = "github.com/Loe159/lpic-daily/cmd/lpic@main"

type Runner interface {
	LookPath(file string) (string, error)
	Run(ctx context.Context, name string, args []string, env []string, stdout, stderr io.Writer) error
}

type OSRunner struct{}

func (OSRunner) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func (OSRunner) Run(
	ctx context.Context,
	name string,
	args []string,
	env []string,
	stdout, stderr io.Writer,
) error {
	command := exec.CommandContext(ctx, name, args...)
	command.Env = env
	command.Stdout = stdout
	command.Stderr = stderr
	return command.Run()
}

type Options struct {
	Runner  Runner
	Stdout  io.Writer
	Stderr  io.Writer
	HomeDir string
}

func Update(ctx context.Context, opts Options) error {
	if opts.Runner == nil {
		opts.Runner = OSRunner{}
	}
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}

	goBinary, err := opts.Runner.LookPath("go")
	if err != nil {
		return errors.New("Go est requis pour mettre LPIC Daily à jour depuis main")
	}

	home := strings.TrimSpace(opts.HomeDir)
	if home == "" {
		home, err = os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("resolve home directory: %w", err)
		}
	}
	if !filepath.IsAbs(home) {
		return errors.New("home directory must be absolute")
	}

	binDir := filepath.Join(filepath.Clean(home), ".local", "bin")
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		return fmt.Errorf("create user bin directory: %w", err)
	}

	buildRoot, err := os.MkdirTemp("", "lpic-daily-update-*")
	if err != nil {
		return fmt.Errorf("create update workspace: %w", err)
	}
	defer os.RemoveAll(buildRoot)

	buildBin := filepath.Join(buildRoot, "bin")
	if err := os.MkdirAll(buildBin, 0o700); err != nil {
		return fmt.Errorf("create update bin directory: %w", err)
	}

	fmt.Fprintln(opts.Stdout, "Mise à jour de LPIC Daily depuis main…")
	env := withEnv(os.Environ(), "GOBIN", buildBin)
	if err := opts.Runner.Run(
		ctx,
		goBinary,
		[]string{"install", mainPackage},
		env,
		opts.Stdout,
		opts.Stderr,
	); err != nil {
		return fmt.Errorf("build latest LPIC Daily from main: %w", err)
	}

	builtPath := filepath.Join(buildBin, "lpic")
	info, err := os.Lstat(builtPath)
	if err != nil {
		return fmt.Errorf("locate updated LPIC Daily binary: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		return errors.New("updated LPIC Daily binary is not a regular executable")
	}

	targetPath := filepath.Join(binDir, "lpic")
	if err := replaceExecutable(builtPath, targetPath); err != nil {
		return err
	}

	fmt.Fprintf(opts.Stdout, "LPIC Daily mis à jour depuis main : %s\n", targetPath)
	return nil
}

func replaceExecutable(sourcePath, targetPath string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open updated LPIC Daily binary: %w", err)
	}
	defer source.Close()

	targetDir := filepath.Dir(targetPath)
	temp, err := os.CreateTemp(targetDir, ".lpic-update-*")
	if err != nil {
		return fmt.Errorf("create temporary update binary: %w", err)
	}
	tempPath := temp.Name()
	cleanup := func() {
		_ = temp.Close()
		_ = os.Remove(tempPath)
	}

	if _, err := io.Copy(temp, source); err != nil {
		cleanup()
		return fmt.Errorf("copy updated LPIC Daily binary: %w", err)
	}
	if err := temp.Chmod(0o755); err != nil {
		cleanup()
		return fmt.Errorf("mark updated LPIC Daily binary executable: %w", err)
	}
	if err := temp.Sync(); err != nil {
		cleanup()
		return fmt.Errorf("sync updated LPIC Daily binary: %w", err)
	}
	if err := temp.Close(); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("close updated LPIC Daily binary: %w", err)
	}
	if err := os.Rename(tempPath, targetPath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("install updated LPIC Daily binary: %w", err)
	}
	return nil
}

func withEnv(environ []string, key, value string) []string {
	prefix := key + "="
	result := make([]string, 0, len(environ)+1)
	for _, item := range environ {
		if !strings.HasPrefix(item, prefix) {
			result = append(result, item)
		}
	}
	return append(result, prefix+value)
}
