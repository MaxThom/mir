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
	Env              EnvCmd      `cmd:"" help:"Display operator details"`
	Pull             PullCmd     `cmd:"" help:"Pull operator credentials, accounts and users"`
	Push             PushCmd     `cmd:"" help:"Push operator credentials, accounts and users"`
	List             ListCmd     `cmd:"" help:"List assests"`
	Add              AddCmd      `cmd:"" help:"Add new users to the system"`
	Delete           DeleteCmd   `cmd:"" help:"Delete users in the system"`
	GenerateResolver ResolverCmd `cmd:"" help:"Generate security config for nat server startup"`
	GenerateCreds    CredsCmd    `cmd:"" help:"Generate security config for nats users"`
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
	Operator   string `short:"o"  help:"Name of operator. Default to context name."`
	Path       string `short:"p" type:"path"`
	Kubernetes bool   `help:"Create the credentials as a Kubernetes secret (use Kubectl)" default:"false"`
	NoExec     bool   `help:"Print commands instead of executing." default:"false"`
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

type CredsCmd struct {
	Operator   string `short:"o"  help:"Name of operator. Default to context name."`
	Url        string `short:"u" help:"Url of Mir server. Default to context url."`
	Account    string `short:"a" help:"Name of main account." default:"mir"`
	User       string `arg:"" help:"Name of user"`
	Path       string `short:"p" type:"path"`
	Kubernetes bool   `help:"Create the credentials as a Kubernetes secret (use Kubectl)" default:"false"`
	NoExec     bool   `help:"Print commands instead of executing." default:"false"`
}

type ListCmd struct {
	Operators ListOperatorsCmd `cmd:"" help:"List operators"`
	Accounts  ListAccountsCmd  `cmd:"" help:"List accounts"`
	Users     ListUsersCmd     `cmd:"" help:"List users"`
}

type ListOperatorsCmd struct {
	NoExec bool `help:"Print commands instead of executing." default:"false"`
}

type ListAccountsCmd struct {
	Operator string `short:"o"  help:"Name of operator. Default to context name."`
	NoExec   bool   `help:"Print commands instead of executing." default:"false"`
}

type ListUsersCmd struct {
	Operator string `short:"o"  help:"Name of operator. Default to context name."`
	Account  string `short:"a" help:"Name of main account." default:"mir"`
	NoExec   bool   `help:"Print commands instead of executing." default:"false"`
}

type AddCmd struct {
	Client AddClientCmd `cmd:"" help:"Create a new user with scope for CLI and other operating interface"`
	Module AddModuleCmd `cmd:"" help:"Create a new user with scope for server module"`
	Device AddDeviceCmd `cmd:"" help:"Create a new user with scope for device"`
}

type AddClientCmd struct {
	Operator string `short:"o"  help:"Name of operator. Default to context name."`
	Account  string `short:"a" help:"Name of main account." default:"mir"`
	Name     string `arg:"" help:"Name of client"`
	ReadOnly bool   `help:"Set scope to Read operations only."`
	Swarm    bool   `help:"Set scope for Read and Write operations and Swarm capabilities."`
	NoExec   bool   `help:"Print commands instead of executing." default:"false"`
}

type AddModuleCmd struct {
	Operator string `short:"o"  help:"Name of operator. Default to context name."`
	Account  string `short:"a" help:"Name of main account." default:"mir"`
	Name     string `arg:"" help:"Name of module"`
	NoExec   bool   `help:"Print commands instead of executing." default:"false"`
}

type AddDeviceCmd struct {
	Operator string `short:"o"  help:"Name of operator. Default to context name."`
	Account  string `short:"a" help:"Name of main account." default:"mir"`
	Name     string `arg:"" help:"Name of device, should be deviceId"`
	Wildcard bool   `help:"Don't bind scope to deviceId, more flexible, but less secure"`
	NoExec   bool   `help:"Print commands instead of executing." default:"false"`
}

type DeleteCmd struct {
	Operator string `short:"o"  help:"Name of operator. Default to context name."`
	Account  string `short:"a" help:"Name of main account." default:"mir"`
	Name     string `arg:"" help:"Name of user"`
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
	if d.Operator == "" {
		d.Operator = ctx.Name
	}

	cmd := exec.Command("nsc", "env", "-o", d.Operator)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if !d.NoExec {
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set current operator: %w", err)
		}
	} else {
		fmt.Println(strings.Join(cmd.Args, " "))
	}

	cmd = exec.Command("nsc", "generate", "config", "--nats-resolver")
	resolverOut := []byte{}
	var err error
	if !d.NoExec {
		resolverOut, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to generate resolver: %w", err)
		}
	} else {
		fmt.Println(strings.Join(cmd.Args, " "))
	}

	if d.Kubernetes {
		cmd = exec.Command("kubectl", "create", "secret", "generic", "mir-resolver-secret", "--from-literal=resolver.conf="+string(resolverOut), "--dry-run=client", "-o", "yaml")
		if !d.NoExec {
			resolverOut, err = cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("failed to generate kubernetes secret for resolver: %w", err)
			}
		} else {
			fmt.Println(strings.Join(cmd.Args, " "))
		}
	}

	if d.Path != "" {
		file, err := os.Create(d.Path)
		if err != nil {
			return fmt.Errorf("unable to create file: %w", err)
		}
		defer file.Close()
		if _, err = file.Write(resolverOut); err != nil {
			return fmt.Errorf("unable to write to file: %w", err)
		}
	} else {
		fmt.Println(string(resolverOut))
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

// Creds Cmd

func (d *CredsCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *CredsCmd) Run(ctx ui.Context) error {
	if d.Operator == "" {
		d.Operator = ctx.Name
	}

	cmd := exec.Command("nsc", "env", "-o", d.Operator)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if !d.NoExec {
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set current operator: %w", err)
		}
	} else {
		fmt.Println(strings.Join(cmd.Args, " "))
	}

	cmd = exec.Command("nsc", "generate", "creds", "-a", d.Account, "-n", d.User)
	credsOut := []byte{}
	var err error
	if !d.NoExec {
		credsOut, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to generate credentials for user: %w", err)
		}
	} else {
		fmt.Println(strings.Join(cmd.Args, " "))
	}

	if d.Kubernetes {
		cmd = exec.Command("kubectl", "create", "secret", "generic", d.User+"-auth-secret", "--from-literal=mir.creds="+string(credsOut), "--dry-run=client", "-o", "yaml")
		if !d.NoExec {
			credsOut, err = cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("failed to generate kubernetes secret for user credentials: %w", err)
			}
		} else {
			fmt.Println(strings.Join(cmd.Args, " "))
		}
	}

	if d.Path != "" {
		file, err := os.Create(d.Path)
		if err != nil {
			return fmt.Errorf("unable to create file: %w", err)
		}
		defer file.Close()
		if _, err = file.Write(credsOut); err != nil {
			return fmt.Errorf("unable to write to file: %w", err)
		}
	} else {
		fmt.Println(string(credsOut))
	}

	return nil
}

// List Operators Cmd

func (d *ListOperatorsCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *ListOperatorsCmd) Run(ctx ui.Context) error {
	var errs error

	cmd := exec.Command("nsc", "list", "operators")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Println(strings.Join(cmd.Args, " "))
	if !d.NoExec {
		if err := cmd.Run(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to list operators: %w", err))
		}
		fmt.Println()
	}

	return nil
}

// List Accounts Cmd

func (d *ListAccountsCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *ListAccountsCmd) Run(ctx ui.Context) error {
	var errs error

	if d.Operator == "" {
		d.Operator = ctx.Name
	}

	cmd := exec.Command("nsc", "list", "accounts", "-o", d.Operator)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Println(strings.Join(cmd.Args, " "))
	if !d.NoExec {
		if err := cmd.Run(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to list accounts: %w", err))
		}
		fmt.Println()
	}

	return nil
}

// List Users Cmd

func (d *ListUsersCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *ListUsersCmd) Run(ctx ui.Context) error {
	var errs error

	if d.Operator == "" {
		d.Operator = ctx.Name
	}

	cmd := exec.Command("nsc", "list", "users", "-o", d.Operator, "-a", d.Account)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Println(strings.Join(cmd.Args, " "))
	if !d.NoExec {
		if err := cmd.Run(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to list users: %w", err))
		}
		fmt.Println()
	}

	return nil
}

// Add Client Cmd

func (d *AddClientCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *AddClientCmd) Run(ctx ui.Context) error {
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

	scopes := []string{}
	if d.ReadOnly {
		scopes = []string{
			"--allow-pubsub", "_INBOX.>",
			"--allow-pub", "client.*.core.v1alpa.list",
			"--allow-pub", "client.*.cfg.v1alpa.list",
			"--allow-pub", "client.*.cmd.v1alpa.list",
			"--allow-pub", "client.*.tlm.v1alpa.list",
			"--allow-pub", "client.*.evt.v1alpa.list",
		}
	} else if d.Swarm {
		scopes = []string{
			"--allow-pubsub", "_INBOX.>",
			"--allow-pub", "client.*.>",
			"--allow-pub", "device.*.>",
			"--allow-sub", "*.>",
		}
	} else {
		scopes = []string{
			"--allow-pubsub", "_INBOX.>",
			"--allow-pub", "client.*.>",
		}
	}

	cmd = exec.Command("nsc", "add", "user", "-a", d.Account, "-n", d.Name)
	cmd.Args = append(cmd.Args, scopes...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Println(strings.Join(cmd.Args, " "))
	if !d.NoExec {
		if err := cmd.Run(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to create new user: %w", err))
		}
		fmt.Println()
	}

	return nil
}

// Add Module Cmd

func (d *AddModuleCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *AddModuleCmd) Run(ctx ui.Context) error {
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

	cmd = exec.Command("nsc", "add", "user", "-a", d.Account, "-n", d.Name,
		"--allow-pubsub", "_INBOX.>",
		"--allow-pubsub", "client.*.>",
		"--allow-pubsub", "event.*.>",
		"--allow-sub", "device.*.>",
		"--allow-pub", "*.>")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Println(strings.Join(cmd.Args, " "))
	if !d.NoExec {
		if err := cmd.Run(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to create new user: %w", err))
		}
		fmt.Println()
	}

	return nil
}

// Add Device Cmd

func (d *AddDeviceCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *AddDeviceCmd) Run(ctx ui.Context) error {
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

	scopes := []string{}
	if d.Wildcard {
		scopes = []string{
			"--allow-pubsub", "_INBOX.>",
			"--allow-pub", "device.*.>",
			"--allow-sub", "*.>",
		}
	} else {
		scopes = []string{
			"--allow-pubsub", "_INBOX.>",
			"--allow-pub", fmt.Sprintf("device.%s.>", d.Name),
			"--allow-sub", fmt.Sprintf("%s.>", d.Name),
		}
	}

	cmd = exec.Command("nsc", "add", "user", "-a", d.Account, "-n", d.Name)
	cmd.Args = append(cmd.Args, scopes...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Println(strings.Join(cmd.Args, " "))
	if !d.NoExec {
		if err := cmd.Run(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to create new user: %w", err))
		}
		fmt.Println()
	}

	return nil
}

// Delete Cmd

func (d *DeleteCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *DeleteCmd) Run(ctx ui.Context) error {
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

	cmd = exec.Command("nsc", "delete", "user", "-a", d.Account, "-n", d.Name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Println(strings.Join(cmd.Args, " "))
	if !d.NoExec {
		if err := cmd.Run(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to delete user: %w", err))
		}
		fmt.Println()
	}

	return nil
}
