// Offers a ready to used cli based on the config style for Mir

// TODO:
// [ ]: global config
// [ ]: no object?
// [ ]: Add MirCli.parse which also print the manual
//
// Snippets:
//cli.New(AppName,
// 	cli.WithDescription("Listen to NatsIO, deserialize protofbuf and push to puthost"),
// 	cli.WithConfigFilePath(&flagFilePath),
// 	cli.WithLogLevel(&flagLogLevel),
// 	cli.WithLogDebug(&flagDebug),
// 	cli.WithManual(&flagManual,
// 		"Listen to queues from NatsIO and receive protobuf encoding to deserialize at runtime\n"+
// 			"using an uploaded protobuf definition.The decoded data is pushed to the puthost protocol.",
// 		&cfg, true, ""),
// )
// flag.Parse()
// if flagManual {
// 	fmt.Println(mirCli.Manual)
// 	os.Exit(0)
// }

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

package cli

import (
	"flag"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type LogLevel = string

const (
	LogLevelTrace   LogLevel = "trace"
	LogLevelDebug   LogLevel = "debug"
	LogLevelInfo    LogLevel = "info"
	LogLevelWarning LogLevel = "warn"
	LogLevelError   LogLevel = "error"
	LogLevelFatal   LogLevel = "fatal"
)

func init() {
}

type MirCli struct {
	appName    string
	desc       string
	Manual     string
	longDesc   string
	envVars    bool
	extra      string
	defaultCfg any
}

func New(appName string, options ...func(*MirCli)) *MirCli {
	cli := &MirCli{
		appName: appName,
	}
	for _, o := range options {
		o(cli)
	}

	cli.Manual = "Manual of " + cli.appName + "\n"
	cli.Manual += strings.ReplaceAll("  "+cli.longDesc, "\n", "\n  ")
	cli.Manual += "\n\nDefault configuration in yaml"
	yamlData, _ := yaml.Marshal(cli.defaultCfg)
	cli.Manual += strings.ReplaceAll("\n"+string(yamlData), "\n", "\n  ")
	if cli.envVars {
		cli.Manual += "\nEnvironment variables"
		cli.Manual +=
			fmt.Sprintf("\n  Are auto defined using the following convention\n"+
				"  - prefixed by the application name\n"+
				"  - follows the yaml configuration naming\n"+
				"  - __ represent nesting\n"+
				"  - _ represent two words where the second one first letter is capitalized\n"+
				"  e.g. %s__LOG__DEBUG_LOGGING == log.debugLogging in yaml", strings.ToUpper(cli.appName))
	}
	cli.Manual += cli.extra

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", cli.appName)
		if cli.desc != "" {
			fmt.Fprintf(flag.CommandLine.Output(), cli.desc+"\n")
		}
		fmt.Fprintf(flag.CommandLine.Output(), "Args:\n")
		flag.PrintDefaults()
	}

	return cli
}

func (m *MirCli) Load() error {
	// Load

	return nil
}

func WithDescription(desc string) func(*MirCli) {
	return func(cli *MirCli) {
		cli.desc = desc
	}
}
func WithManual(out *bool, desc string, defaultConfig any, envVars bool, extra string) func(*MirCli) {
	return func(cli *MirCli) {
		cli.longDesc = desc
		cli.defaultCfg = defaultConfig
		cli.envVars = envVars
		cli.extra = extra
		flag.BoolVar(out, "manual", false, "diplay usage manual")
	}
}

func WithConfigFilePath(out *string) func(*MirCli) {
	return func(cli *MirCli) {
		flag.StringVar(out, "config", "", "extra configuration filepath")
	}
}

func WithLogLevel(out *LogLevel) func(*MirCli) {
	return func(cli *MirCli) {
		flag.StringVar(out, "loglevel", LogLevelInfo, fmt.Sprintf("[%s|%s|%s|%s|%s|%s]",
			LogLevelTrace,
			LogLevelDebug,
			LogLevelInfo,
			LogLevelWarning,
			LogLevelError,
			LogLevelFatal,
		))
	}
}

func WithLogDebug(out *bool) func(*MirCli) {
	return func(cli *MirCli) {
		flag.BoolVar(out, "debug", false, "set log level to debug")
	}
}
