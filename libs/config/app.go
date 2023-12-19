// AppConfig package offers an abstraction layer with some opininated
// desicisions around how to do configuration. Each app of this repo should
// follow this approach because it lead to a consistent experience accross all apps and services
// It follows the functionnal options pattern for configuration. It tries to strike the
// right balance between opinion and consistency
//
// ConfigFiles:
// - each file is under a directory with the app name
// - not enforce, but should use EtcFilePath for prod and XdgConfigHome for dev
// Env vars:
// - prefix with the app name
// - match the config file structs nesting
//   ```
//   httpServer:
//     port: 3000
//   ```
//   the resulting env var name for the port would be
//     APPNAME__HTTP_SERVER__PORT
// - __ is for nesting
// - _ is for words, the first letter after the _ will be capitalized
//
// TODO:
// [ ]: global config
// [ ]: file watch with channels
// [ ]: config printer on startup
//
// Snippets:
//
// var appConfig = config.New(AppName,
//   config.WithEtcFilePath("config.yaml", config.Yaml, false),
// 	 config.WithXdgConfigHomeFilePath("config.yaml", config.Yaml, true),
// 	 config.WithEnvVars(),
// )
//
// var cfg ProtoProxyConfig
// if err := appConfig.LoadAndUnmarshal(&cfg); err != nil {
//  	fmt.Println(err)
// }
//

package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

var xdgConfigHome string
var isNotPidZero = os.Getpid() != 0
var configName string

func init() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("$HOME is not defined")
	}
	xdgConfigHome = filepath.Join(userHomeDir, ".config")
}

type configFormat string

const (
	Yaml configFormat = "yaml"
	Json configFormat = "json"
)

type appConfigSetup struct {
	configFiles []configFile
	envVars     bool
	appName     string
	k           *koanf.Koanf
}

type configFile struct {
	path   string
	format configFormat
}

func Empty() *appConfigSetup {
	return &appConfigSetup{}
}

// The order of configuration loading is:
//   - flags
//   - env vars
//   - config files
func New(appName string, options ...func(*appConfigSetup)) *appConfigSetup {
	cfg := &appConfigSetup{
		appName: appName,
		k:       koanf.New("."),
	}
	for _, o := range options {
		o(cfg)
	}

	return cfg
}

func (s *appConfigSetup) Load() error {
	// Load
	var errs error
	for _, f := range s.configFiles {
		var parser koanf.Parser

		switch f.format {
		case Yaml:
			parser = yaml.Parser()
		case Json:
			parser = json.Parser()
		default:
			fmt.Printf("invalid config format '%s' for file %s.\n[Json|Yaml]\n", f.format, f.path)
		}
		if err := s.k.Load(file.Provider(f.path), parser); err != nil {
			errs = errors.Join(errs, err)
		}
	}

	if errs != nil {
		return errs
	}

	// Each env var:
	// - __ for nesting.
	// - _ for multiple words where the first letter after it becomes capitalize
	if s.envVars {
		envPrefix := strings.ToUpper(s.appName) + "__"
		s.k.Load(env.Provider(envPrefix, ".", func(s string) string {
			return envVarsToYamlNomenclature(s, envPrefix)
		}), nil)
	}

	return nil
}

func (s *appConfigSetup) LoadAndUnmarshal(out any) error {
	if err := s.Load(); err != nil {
		return err
	}
	return s.Unmarshal(out)
}

func (s *appConfigSetup) Get(path string) any {
	return s.k.Get(path)
}

func (s *appConfigSetup) All() map[string]any {
	return s.k.All()
}

func (s *appConfigSetup) Set(key string, val any) error {
	return s.k.Set(key, val)
}

func (s *appConfigSetup) Unmarshal(out any) error {
	return s.k.Unmarshal("", out)
}

func WithFilePath(path string, cff configFormat, devOnly bool) func(*appConfigSetup) {
	return func(cfg *appConfigSetup) {
		if devOnly && isNotPidZero || !devOnly {
			cfg.configFiles = append(cfg.configFiles, configFile{
				path:   path,
				format: cff,
			})
		}
	}
}

func WithEtcFilePath(fileName string, cff configFormat, devOnly bool) func(*appConfigSetup) {
	return func(cfg *appConfigSetup) {
		if devOnly && isNotPidZero || !devOnly {
			path := filepath.Join("/etc", cfg.appName, fileName)
			cfg.configFiles = append(cfg.configFiles, configFile{
				path:   path,
				format: cff,
			})
		}
	}
}

func WithXdgConfigHomeFilePath(fileName string, cff configFormat, devOnly bool) func(*appConfigSetup) {
	return func(cfg *appConfigSetup) {
		if devOnly && isNotPidZero || !devOnly {
			path := filepath.Join(xdgConfigHome, cfg.appName, fileName)
			cfg.configFiles = append(cfg.configFiles, configFile{
				path:   path,
				format: cff,
			})
		}
	}
}

func WithEnvVars() func(*appConfigSetup) {
	return func(cfg *appConfigSetup) {
		cfg.envVars = true
	}
}

func envVarsToYamlNomenclature(s string, prefix string) string {
	// Remove envvar prefix, lowercase all, and __ with . for nesting
	s = strings.Replace(strings.ToLower(strings.TrimPrefix(s, prefix)), "__", ".", -1)
	// Find all _ and replace first letter after with capital
	var result []rune
	capNext := false
	for _, r := range s {
		switch {
		case r == '_':
			capNext = true
		case capNext:
			result = append(result, unicode.ToUpper(r))
			capNext = false
		default:
			result = append(result, r)
		}
	}
	return string(result)
}
