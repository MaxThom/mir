package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/maxthom/mir/internal/ui"
)

type SecurityCmd struct {
	Init             InitCmd     `cmd:"" help:"Create operator to manage a Mir server"`
	Edit             EditCmd     `cmd:"" help:"Update operator server url"`
	GenerateResolver ResolverCmd `cmd:"" help:"Generate security config for nat server startup"`
	Env              EnvCmd      `cmd:"" help:"Display operator details"`
	Pull             PullCmd     `cmd:"" help:"Pull operator credentials, accounts and users"`
	Push             PushCmd     `cmd:"" help:"Push operator credentials, accounts and users"`
}

type InitCmd struct {
	Operator string `short:"o"  help:"Name of operator. Default to context name."`
	Url      string `short:"u" help:"Url of Mir server. Default to context url."`
	Account  string `short:"a" help:"Name of main account." default:"mir"`
	NoExec   bool   `help:"Print commands instead of executing." default:"false"`
}

type EditCmd struct {
	Operator string `short:"o"  help:"Name of operator. Default to context name."`
	Url      string `short:"u" help:"Url of Mir server. Default to context url."`
	NoExec   bool   `help:"Print commands instead of executing." default:"false"`
}

type ResolverCmd struct {
	Operator string `short:"o"  help:"Name of operator. Default to context name."`
	Path     string `arg:"" type:"path" default:"./resolver.conf"`
	NoExec   bool   `help:"Print commands instead of executing." default:"false"`
}

type EnvCmd struct {
	Operator string `short:"o"  help:"Name of operator. Default to context name."`
	NoExec   bool   `help:"Print commands instead of executing." default:"false"`
}

type PullCmd struct {
	Operator string `short:"o"  help:"Name of operator. Default to context name."`
	NoExec   bool   `help:"Print commands instead of executing." default:"false"`
}

type PushCmd struct {
	Operator string `short:"o"  help:"Name of operator. Default to context name."`
	NoExec   bool   `help:"Print commands instead of executing." default:"false"`
}

// Init Cmd

func (d *InitCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *InitCmd) Run(ctx ui.Context) error {
	var errs error

	if d.Operator == "" {
		d.Operator = ctx.Name
	}
	if d.Url == "" {
		d.Url = ctx.Target
	}

	cmd := exec.Command("nsc", "add", "operator", "--generate-signing-key", "--sys", "--name", d.Operator)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Println(strings.Join(cmd.Args, " "))
	if !d.NoExec {
		if err := cmd.Run(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to create operator: %v", err))
		}
		fmt.Println()
	}

	cmd = exec.Command("nsc", "edit", "operator", "--service-url", d.Url, "--account-jwt-server-url", d.Url)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Println(strings.Join(cmd.Args, " "))
	if !d.NoExec {
		if err := cmd.Run(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to edit operator: %v", err))
		}
		fmt.Println()
	}

	cmd = exec.Command("nsc", "add", "account", "-n", d.Account)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Println(strings.Join(cmd.Args, " "))
	if !d.NoExec {
		if err := cmd.Run(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to add account to operator: %v", err))
		}
		fmt.Println()
	}

	return nil
}

// Edit Cmd

func (d *EditCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *EditCmd) Run(ctx ui.Context) error {
	var errs error

	if d.Operator == "" {
		d.Operator = ctx.Name
	}
	if d.Url == "" {
		d.Url = ctx.Target
	}

	cmd := exec.Command("nsc", "env", "-o", d.Operator)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	fmt.Println(strings.Join(cmd.Args, " "))
	if !d.NoExec {
		if err := cmd.Run(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to set current operator: %w", err))
		}
	}

	cmd = exec.Command("nsc", "edit", "operator", "--service-url", d.Url, "--account-jwt-server-url", d.Url)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Println(strings.Join(cmd.Args, " "))
	if !d.NoExec {
		if err := cmd.Run(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to edit operator: %w", err))
		}
		fmt.Println()
	}

	return nil
}

// Resolver Cmd

func (d *ResolverCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *ResolverCmd) Run(ctx ui.Context) error {
	var errs error

	if d.Operator == "" {
		d.Operator = ctx.Name
	}

	cmd := exec.Command("nsc", "env", "-o", d.Operator)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	fmt.Println(strings.Join(cmd.Args, " "))
	if !d.NoExec {
		if err := cmd.Run(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to set current operator: %w", err))
		}
	}

	file, err := os.Create(d.Path)
	if err != nil {
		return fmt.Errorf("unable to create file: %w", err)
	}
	defer file.Close()

	cmd = exec.Command("nsc", "generate", "config", "--nats-resolver")
	cmd.Stdout = file
	cmd.Stderr = file
	fmt.Println(strings.Join(cmd.Args, " "), ">", d.Path)
	if !d.NoExec {
		if err := cmd.Run(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to generate resolver.conf: %w", err))
		}
		fmt.Println()
	}

	return nil
}

// Env Cmd

func (d *EnvCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *EnvCmd) Run(ctx ui.Context) error {
	var errs error

	if d.Operator == "" {
		d.Operator = ctx.Name
	}

	cmd := exec.Command("nsc", "env", "-o", d.Operator)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Println(strings.Join(cmd.Args, " "))
	if !d.NoExec {
		if err := cmd.Run(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to set current operator: %w", err))
		}
		fmt.Println()
	}

	return nil
}

// Pull Cmd

func (d *PullCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *PullCmd) Run(ctx ui.Context) error {
	var errs error

	if d.Operator == "" {
		d.Operator = ctx.Name
	}

	cmd := exec.Command("nsc", "env", "-o", d.Operator)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	fmt.Println(strings.Join(cmd.Args, " "))
	if !d.NoExec {
		if err := cmd.Run(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to set current operator: %w", err))
		}
	}

	cmd = exec.Command("nsc", "pull", "-A")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Println(strings.Join(cmd.Args, " "))
	if !d.NoExec {
		if err := cmd.Run(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to pull operator: %w", err))
		}
		fmt.Println()
	}

	return nil
}

// Push Cmd

func (d *PushCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *PushCmd) Run(ctx ui.Context) error {
	var errs error

	if d.Operator == "" {
		d.Operator = ctx.Name
	}

	cmd := exec.Command("nsc", "env", "-o", d.Operator)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	fmt.Println(strings.Join(cmd.Args, " "))
	if !d.NoExec {
		if err := cmd.Run(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to set current operator: %w", err))
		}
	}

	cmd = exec.Command("nsc", "push", "-A")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Println(strings.Join(cmd.Args, " "))
	if !d.NoExec {
		if err := cmd.Run(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to push changes to operator: %w", err))
		}
		fmt.Println()
	}

	return nil
}
