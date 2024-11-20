package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"

	"github.com/maxthom/mir/infra"
)

type InfraCmd struct {
	Up    UpCmd    `cmd:"" passthrough:"" help:"Run infra docker compose up"`
	Down  DownCmd  `cmd:"" passthrough:"" help:"Run infra docker compose down"`
	Ps    PsCmd    `cmd:"" passthrough:"" help:"Run infra docker compose ps"`
	Rm    RmCmd    `cmd:"" passthrough:"" help:"Run infra docker compose rm"`
	Print PrintCmd `cmd:"" help:"Write to disk Mir set of docker compose"`
}

type UpCmd struct {
	Args []string `arg:"" optional:"" help:"Docker compose up arguments. Will be passed on."`
}

type DownCmd struct {
	Args []string `arg:"" optional:"" help:"Docker compose down arguments. Will be passed on."`
}

type PsCmd struct {
	Args []string `arg:"" optional:"" help:"Docker compose ps arguments. Will be passed on."`
}

type RmCmd struct {
	Args []string `arg:"" optional:"" help:"Docker compose rm arguments. Will be passed on."`
}

type PrintCmd struct {
	Path string `short:"p" help:"Write path for compose files" default:"."`
}

func (d *UpCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *UpCmd) Run() error {
	var errs error
	filePath := os.Getenv("XDG_CACHE_HOME")
	if filePath == "" {
		usr, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %v", err)
		}
		filePath = usr
	}
	filePath += "/.cache/mir/infra"

	if err := RecreateFS(infra.LocalInfraFS, filePath); err != nil {
		return fmt.Errorf("canno't write Mir compose files to %s: %w", filePath, err)
	}

	d.Args = append([]string{"compose", "-f", filePath + "/local/compose.yaml", "up"}, d.Args...)
	cmd := exec.Command("docker", d.Args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		errs = errors.Join(errs, fmt.Errorf("failed to run docker compose up: %v", err))
	}

	return errs
}

func (d *DownCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *DownCmd) Run() error {
	var errs error
	filePath := os.Getenv("XDG_CACHE_HOME")
	if filePath == "" {
		usr, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %v", err)
		}
		filePath = usr
	}
	filePath += "/.cache/mir/infra"

	if err := RecreateFS(infra.LocalInfraFS, filePath); err != nil {
		return fmt.Errorf("unable to write Mir compose files to %s: %w", filePath, err)
	}

	d.Args = append([]string{"compose", "-f", filePath + "/local/compose.yaml", "down"}, d.Args...)
	cmd := exec.Command("docker", d.Args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		errs = errors.Join(errs, fmt.Errorf("failed to run docker compose up: %v", err))
	}

	return errs
}

func (d *PrintCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *PrintCmd) Run() error {
	if err := RecreateFS(infra.LocalInfraFS, path.Join(d.Path, "infra")); err != nil {
		return fmt.Errorf("unable to write Mir compose files to %s: %w", d.Path, err)
	}
	return nil
}

func (d *PsCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *PsCmd) Run() error {
	var errs error
	filePath := os.Getenv("XDG_CACHE_HOME")
	if filePath == "" {
		usr, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %v", err)
		}
		filePath = usr
	}
	filePath += "/.cache/mir/infra"

	if err := RecreateFS(infra.LocalInfraFS, filePath); err != nil {
		return fmt.Errorf("canno't write Mir compose files to %s: %w", filePath, err)
	}

	d.Args = append([]string{"compose", "-f", filePath + "/local/compose.yaml", "ps"}, d.Args...)
	cmd := exec.Command("docker", d.Args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		errs = errors.Join(errs, fmt.Errorf("failed to run docker compose ps: %w", err))
	}

	return errs
}

func (d *RmCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *RmCmd) Run() error {
	var errs error
	filePath := os.Getenv("XDG_CACHE_HOME")
	if filePath == "" {
		usr, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %v", err)
		}
		filePath = usr
	}
	filePath += "/.cache/mir/infra"

	if err := RecreateFS(infra.LocalInfraFS, filePath); err != nil {
		return fmt.Errorf("canno't write Mir compose files to %s: %w", filePath, err)
	}

	d.Args = append([]string{"compose", "-f", filePath + "/local/compose.yaml", "rm"}, d.Args...)
	cmd := exec.Command("docker", d.Args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		errs = errors.Join(errs, fmt.Errorf("failed to run docker compose rm: %w", err))
	}

	return errs
}
