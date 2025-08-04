package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"

	infra "github.com/maxthom/mir/infra/compose"
)

type InfraCmd struct {
	IncludeMir bool     `short:"m" help:"Include Mir server"`
	Up         UpCmd    `cmd:"" passthrough:"" help:"Run infra docker compose up"`
	Down       DownCmd  `cmd:"" passthrough:"" help:"Run infra docker compose down"`
	Ps         PsCmd    `cmd:"" passthrough:"" help:"Run infra docker compose ps"`
	Rm         RmCmd    `cmd:"" passthrough:"" help:"Run infra docker compose rm"`
	Print      PrintCmd `cmd:"" help:"Write to disk Mir set of docker compose"`
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

const (
	composeInfraPath        = "/local_support/compose.yaml"
	composeInfraWithMirPath = "/local_mir_support/compose.yaml"
)

func (d *UpCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *UpCmd) Run(infraCmd *InfraCmd) error {
	composePath := composeInfraPath
	if infraCmd.IncludeMir {
		composePath = composeInfraWithMirPath
	}

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

	if err := RecreateFS(infra.LocalInfraFS, filePath, true); err != nil {
		return fmt.Errorf("canno't write Mir compose files to %s: %w", filePath, err)
	}

	d.Args = append([]string{"compose", "-f", path.Join(filePath, composePath), "up"}, d.Args...)
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

func (d *DownCmd) Run(infraCmd *InfraCmd) error {
	composePath := composeInfraPath
	if infraCmd.IncludeMir {
		composePath = composeInfraWithMirPath
	}
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

	if err := RecreateFS(infra.LocalInfraFS, filePath, true); err != nil {
		return fmt.Errorf("unable to write Mir compose files to %s: %w", filePath, err)
	}

	d.Args = append([]string{"compose", "-f", filePath + composePath, "down"}, d.Args...)
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
	if err := RecreateFS(infra.LocalInfraFS, path.Join(d.Path, "infra"), true); err != nil {
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

func (d *PsCmd) Run(infraCmd *InfraCmd) error {
	composePath := composeInfraPath
	if infraCmd.IncludeMir {
		composePath = composeInfraWithMirPath
	}
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

	if err := RecreateFS(infra.LocalInfraFS, filePath, true); err != nil {
		return fmt.Errorf("canno't write Mir compose files to %s: %w", filePath, err)
	}

	d.Args = append([]string{"compose", "-f", filePath + composePath, "ps"}, d.Args...)
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

func (d *RmCmd) Run(infraCmd *InfraCmd) error {
	composePath := composeInfraPath
	if infraCmd.IncludeMir {
		composePath = composeInfraWithMirPath
	}

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

	if err := RecreateFS(infra.LocalInfraFS, filePath, true); err != nil {
		return fmt.Errorf("canno't write Mir compose files to %s: %w", filePath, err)
	}

	d.Args = append([]string{"compose", "-f", filePath + composePath, "rm"}, d.Args...)
	cmd := exec.Command("docker", d.Args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		errs = errors.Join(errs, fmt.Errorf("failed to run docker compose rm: %w", err))
	}

	return errs
}
