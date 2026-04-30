package ui

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

var (
	LoadedConfigPath  string
	DefaultConfigPath string
)

func init() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("$HOME is not defined")
		DefaultConfigPath = filepath.Join("/etc", "mir", "cli.yaml")
	} else {
		DefaultConfigPath = filepath.Join(userHomeDir, ".config", "mir", "cli.yaml")
	}
}

type Config struct {
	LogLevel       string    `yaml:"logLevel"`
	CurrentContext string    `yaml:"currentContext"`
	Contexts       []Context `yaml:"contexts"`
}

func NewDefaultConfig() Config {
	return Config{
		LogLevel:       "info",
		CurrentContext: "local",
		Contexts: []Context{
			{
				Name:    "local",
				Target:  "nats://localhost:4222",
				Grafana: "localhost:3000",
			},
		},
	}
}

type Context struct {
	Name        string `yaml:"name"`
	Target      string `yaml:"target"`
	WebTarget   string `yaml:"webTarget"`
	Grafana     string `yaml:"grafana"`
	Credentials string `yaml:"credentials"`
	RootCA      string `yaml:"rootCA"`
	TlsCert     string `yaml:"tlsCert"`
	TlsKey      string `yaml:"tlsKey"`
	Password    string `yaml:"password" cfg:"secret"`
}

func (c Config) GetCurrentContext() (Context, bool) {
	for _, v := range c.Contexts {
		if v.Name == c.CurrentContext {
			return v, true
		}
	}
	return Context{}, false
}

func (c *Config) SetCurrentContext(ctx string) bool {
	for _, v := range c.Contexts {
		if v.Name == ctx {
			c.CurrentContext = ctx
			return true
		}
	}
	return false
}

func (c Config) WriteConfig() error {
	if LoadedConfigPath == "" {
		LoadedConfigPath = DefaultConfigPath
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	data = append([]byte("# Mir CLI Configuration\n"), data...)

	dir := filepath.Dir(LoadedConfigPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}
	return os.WriteFile(LoadedConfigPath, data, 0644)
}

func (c Config) PrintConfig() ([]byte, error) {
	return yaml.Marshal(c)
}
