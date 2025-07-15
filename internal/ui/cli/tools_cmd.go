package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"

	"github.com/maxthom/mir/examples"
	"github.com/maxthom/mir/pkgs/device/proto"
)

type ToolsCmd struct {
	Generate GenerateCmd `cmd:"" help:"Generate project templates"`
	Install  InstallCmd  `cmd:"" help:"Install development tools"`
}

type GenerateCmd struct {
	MirSchema      MirSchemaCmd      `cmd:"" help:"Generate mir schema"`
	DeviceTemplate DeviceTemplateCmd `cmd:"" help:"Generate a device template project"`
}

type InstallCmd struct {
}

func (d *InstallCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *InstallCmd) Run() error {
	var errs error

	fmt.Println("installing buf...")
	cmd := exec.Command("go", "install", "github.com/bufbuild/buf/cmd/buf@latest")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		errs = errors.Join(errs, fmt.Errorf("failed to install buf: %v", err))
	}

	fmt.Println("installing protoc...")
	cmd = exec.Command("go", "install", "google.golang.org/protobuf/cmd/protoc-gen-go@latest")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		errs = errors.Join(errs, fmt.Errorf("failed to install protoc: %v", err))
	}

	fmt.Println("install complete 🚀 !")
	return errs
}

type MirSchemaCmd struct {
	ContextPath string `short:"p" help:"Context path to generate schema structure" default:"."`
}

func (d *MirSchemaCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *MirSchemaCmd) Run() error {
	return RecreateFS(proto.SchemaFS, d.ContextPath, true)
}

type DeviceTemplateCmd struct {
	ContextPath string `short:"p" help:"Context path to generate schema structure" default:"."`
	Proto       string `enum:"protoc,buf" help:"Protofiles management [protoc|buf]. Buf (recommended)." default:"buf"`
	ModulePath  string `arg:"" help:"Go project module path. eg: github.com/<user|org>/<project>"`
}

func (d *DeviceTemplateCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *DeviceTemplateCmd) Run() error {
	if err := os.MkdirAll(d.ContextPath, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	if _, err := os.Stat(path.Join(d.ContextPath, "go.mod")); os.IsNotExist(err) {
		cmd := exec.Command("go", "mod", "init", d.ModulePath)
		cmd.Dir = d.ContextPath
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to init go module: %v", err)
		}
	}

	cmd := exec.Command("go", "get", "github.com/maxthom/mir/")
	cmd.Dir = d.ContextPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add mir dependency: %v", err)
	}

	if d.Proto == "buf" {
		subFS, err := fs.Sub(examples.DeviceTemplateFS, "templates/go_device_buf")
		if err != nil {
			return fmt.Errorf("failed to create sub filesystem: %w", err)
		}
		if err = RecreateFS(subFS, d.ContextPath, true); err != nil {
			return fmt.Errorf("failed to create device template project: %w", err)
		}
	} else if d.Proto == "protoc" {
		subFS, err := fs.Sub(examples.DeviceTemplateFS, "templates/go_device_protoc")
		if err != nil {
			return fmt.Errorf("failed to create sub filesystem: %w", err)
		}
		if err = RecreateFS(subFS, d.ContextPath, true); err != nil {
			return fmt.Errorf("failed to create device template project: %w", err)
		}
	}

	cmd = exec.Command("go", "mod", "tidy")
	cmd.Dir = d.ContextPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to tidy up dependencies: %v", err)
	}

	return nil
}
