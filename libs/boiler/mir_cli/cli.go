// Offers a ready to used cli based on the config style for Mir

// TODO:
// [ ]: global config
// [ ]: no object?
// [ ]: Add Mircli.parse which also print the manual
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
// 	fmt.Println(mircli.Manual)
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

// TODO rework ideas
// - switch to kong
// - use a struct that the app can pass
// - this structs becomes cli flags
// - default struc

package mir_cli

import (
	"flag"
	"fmt"
	"os"
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

type mirCli struct {
	appName    string
	desc       string
	Manual     string
	longDesc   string
	envVars    bool
	extra      string
	defaultCfg any
}

var cli *mirCli
var flagManualOut bool

func Setup(appName string, options ...func(*mirCli)) {
	cli = &mirCli{
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
	cli.Manual += "\n" + cli.extra

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", cli.appName)
		if cli.desc != "" {
			fmt.Fprintf(flag.CommandLine.Output(), cli.desc+"\n")
		}
		fmt.Fprintf(flag.CommandLine.Output(), "Args:\n")
		flag.PrintDefaults()
	}
}

func Parse() error {
	// Load
	flag.Parse()
	if flagManualOut {
		fmt.Println(cli.Manual)
		os.Exit(0)
	}
	return nil
}

func Args() []string {
	return flag.Args()
}

func WithDescription(desc string) func(*mirCli) {
	return func(cli *mirCli) {
		cli.desc = desc
	}
}

func WithManual(desc string, defaultConfig any, envVars bool, extra string) func(*mirCli) {
	return func(cli *mirCli) {
		cli.longDesc = desc
		cli.defaultCfg = defaultConfig
		cli.envVars = envVars
		cli.extra = extra
		flag.BoolVar(&flagManualOut, "manual", false, "diplay usage manual")
	}
}

func WithConfigFilePath(out *string) func(*mirCli) {
	return func(cli *mirCli) {
		flag.StringVar(out, "config", "", "extra configuration filepath")
	}
}

func WithOsFlag(fn func()) func(*mirCli) {
	return func(cli *mirCli) {
		fn()
	}
}

func WithLogLevel(out *LogLevel) func(*mirCli) {
	return func(cli *mirCli) {
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

func WithLogDebug(out *bool) func(*mirCli) {
	return func(cli *mirCli) {
		flag.BoolVar(out, "debug", false, "set log level to debug")
	}
}
