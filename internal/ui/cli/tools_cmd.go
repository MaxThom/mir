package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/maxthom/mir/examples"
	"github.com/maxthom/mir/internal/ui"
	"github.com/maxthom/mir/pkgs/device/proto"
)

type ToolsCmd struct {
	Generate GenerateCmd `cmd:"" help:"Generate project templates"`
	Install  InstallCmd  `cmd:"" help:"Install development tools"`
	Log      LogCmd      `cmd:"" help:"View and follow Mir CLI logs"`
	Config   SettingsCmd `cmd:"" help:"Manage Mir CLI configuration"`
}

type LogCmd struct {
	Lines  int  `short:"n" help:"Number of lines to display (0 for all)" default:"0"`
	Follow bool `short:"f" help:"Follow log output" default:"false"`
}

type SettingsCmd struct {
	View SettingsViewCmd `cmd:"" help:"View configuration file"`
	Edit SettingsEditCmd `cmd:"" help:"Edit configuration file"`
}

type SettingsViewCmd struct {
}

type SettingsEditCmd struct {
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

func (d *LogCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}

	if d.Lines < 0 {
		err.Details = append(err.Details, "Number of lines must be non-negative")
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *LogCmd) Run() error {
	logPath := getLogFilePath()
	if logPath == "" {
		return fmt.Errorf("failed to determine log file path")
	}

	// Check if file exists
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		return fmt.Errorf("log file not found at %s", logPath)
	}

	// Print the file path
	fmt.Printf("# %s\n", logPath)

	// If not following, just display the requested lines
	if !d.Follow {
		// If Lines is 0, display all lines
		if d.Lines == 0 {
			content, err := os.ReadFile(logPath)
			if err != nil {
				return fmt.Errorf("failed to read log file: %v", err)
			}
			fmt.Print(string(content))
			return nil
		}

		// Otherwise, read last N lines
		file, err := os.Open(logPath)
		if err != nil {
			return fmt.Errorf("failed to open log file: %v", err)
		}
		defer file.Close()

		// Read all lines into a slice
		var lines []string
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("error reading log file: %v", err)
		}

		// Display last N lines
		start := len(lines) - d.Lines
		for i := max(start, 0); i < len(lines); i++ {
			fmt.Println(lines[i])
		}

		return nil
	}

	// Follow mode - handle interrupt signal for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	errChan := make(chan error, 1)
	go func() {
		// If Lines is 0 when following, default to showing last 10 lines
		linesToShow := d.Lines
		if linesToShow == 0 {
			linesToShow = 10
		}
		errChan <- tailFile(logPath, linesToShow, true)
	}()

	select {
	case <-sigChan:
		fmt.Println("\nLog tail interrupted")
		return nil
	case err := <-errChan:
		return err
	}
}

func getLogFilePath() string {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		userHomeDir = "./"
	}
	return filepath.Join(userHomeDir, ".config", "mir", "cli.log")
}

func tailFile(path string, lines int, follow bool) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Get file info to determine initial position
	stat, err := file.Stat()
	if err != nil {
		return err
	}

	// If we need to show last N lines, read the file to find the position
	var startPos int64 = 0
	if lines > 0 {
		// Read file to count lines and find position
		scanner := bufio.NewScanner(file)
		var allLines []string
		for scanner.Scan() {
			allLines = append(allLines, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return err
		}

		// Print last N lines
		start := len(allLines) - lines
		for i := max(start, 0); i < len(allLines); i++ {
			fmt.Println(allLines[i])
		}

		// Reset to end of file for following
		startPos = stat.Size()
		file.Seek(startPos, 0)
	} else {
		// Start from end of file if no initial lines requested
		startPos = stat.Size()
		file.Seek(startPos, 0)
	}

	if !follow {
		return nil
	}

	// Follow the file for new content
	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				// Wait a bit and check for new content
				time.Sleep(250 * time.Millisecond)

				// Check if file has grown
				newStat, err := os.Stat(path)
				if err != nil {
					return fmt.Errorf("error checking file: %v", err)
				}

				// If file was truncated or rotated, restart from beginning
				if newStat.Size() < stat.Size() {
					file.Seek(0, 0)
					reader = bufio.NewReader(file)
					stat = newStat
					fmt.Println("--- File rotated ---")
				}
				continue
			}
			return err
		}
		fmt.Print(line)
	}
}

// SettingsViewCmd implementation
func (d *SettingsViewCmd) Validate() error {
	return nil
}

func (d *SettingsViewCmd) Run(cfg ui.Config) error {
	b, err := cfg.PrintConfig()
	if err != nil {
		return err
	}
	fmt.Println("# " + ui.LoadedConfigPath)
	fmt.Println(string(b))

	return nil
}

// SettingsEditCmd implementation
func (d *SettingsEditCmd) Validate() error {
	return nil
}

func (d *SettingsEditCmd) Run(cfg ui.Config) error {
	if err := cfg.WriteConfig(); err != nil {
		return err
	}

	// Determine the editor to use
	editor := os.Getenv("EDITOR")
	if editor == "" {
		switch runtime.GOOS {
		case "windows":
			editor = "notepad.exe"
		case "darwin":
			editor = "nano"
		default:
			editor = "nano"
		}
	}

	fmt.Printf("Opening %s with %s...\n", ui.LoadedConfigPath, editor)

	// Open the config file in the editor
	cmd := exec.Command(editor, ui.LoadedConfigPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to open editor: %v", err)
	}

	fmt.Println("Configuration file saved.")
	return nil
}
