package mir_log

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type LogLevel = string

var isNotPidZero = os.Getpid() != 0

func LowestLogLevel(a LogLevel, b LogLevel) bool {
	logLevels := map[LogLevel]int{
		LogLevelTrace:   0,
		LogLevelDebug:   1,
		LogLevelInfo:    2,
		LogLevelWarning: 3,
		LogLevelError:   4,
		LogLevelFatal:   5,
	}
	return logLevels[a] < logLevels[b]
}

func CompareLogLevel(a LogLevel, b LogLevel) string {
	if LowestLogLevel(a, b) {
		return a
	}
	return b
}

var (
	LogLevelTrace   LogLevel = "trace"
	LogLevelDebug   LogLevel = "debug"
	LogLevelInfo    LogLevel = "info"
	LogLevelWarning LogLevel = "warn"
	LogLevelError   LogLevel = "error"
	LogLevelFatal   LogLevel = "fatal"
)

type mirLog struct {
	logLevel string
	zerolog.Logger
	hasWriter bool
}

func (l *mirLog) WithKeyValue(key, value string) func(*mirLog) {
	return func(l *mirLog) {
		log.Logger = log.With().Str(key, value).Logger()
		l.Logger = l.With().Str(key, value).Logger()
	}
}

var Logger *mirLog

// This set the global logger and return a new context
// from it with the  options
func Setup(options ...func(*mirLog)) zerolog.Logger {
	Logger = &mirLog{}
	for _, o := range options {
		o(Logger)
	}
	if !Logger.hasWriter {
		Logger.Logger = Logger.Output(os.Stdout)
	}
	return Logger.With().Logger()
}

func WithTimeFormatUnix() func(*mirLog) {
	return func(log *mirLog) {
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
		log.Logger = log.Logger.With().Timestamp().Logger()
	}
}

func WithLogLevel(logLevel LogLevel) func(*mirLog) {
	return func(log *mirLog) {
		log.logLevel = logLevel
		switch logLevel {
		case "trace":
			log.Logger = log.Level(zerolog.TraceLevel)
			zerolog.SetGlobalLevel(zerolog.TraceLevel)
		case "debug":
			log.Logger = log.Level(zerolog.DebugLevel)
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		case "info":
			log.Logger = log.Level(zerolog.InfoLevel)
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
		case "warn":
			log.Logger = log.Level(zerolog.WarnLevel)
			zerolog.SetGlobalLevel(zerolog.WarnLevel)
		case "error":
			log.Logger = log.Level(zerolog.ErrorLevel)
			zerolog.SetGlobalLevel(zerolog.ErrorLevel)
		case "fatal":
			log.Logger = log.Level(zerolog.FatalLevel)
			zerolog.SetGlobalLevel(zerolog.FatalLevel)
		default:
			log.Logger = log.Level(zerolog.InfoLevel)
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
		}
	}
}

// FlagLogLevel take logs flag parameters and
// log level from file. It will select based on priority
// 1. debug flag
// 2. loglevel flag
// 3. file loglevel
// It will set the fileLogLevel to the current to keep config right
func WithFlagAndFileLogLevel(flagIsDebug bool, flagLogLevel LogLevel, fileLogLevel *LogLevel) func(*mirLog) {
	var current LogLevel
	if flagIsDebug {
		current = LogLevelDebug
	} else if flagLogLevel != "" {
		current = flagLogLevel
	} else if *fileLogLevel != "" {
		current = *fileLogLevel
	}
	*fileLogLevel = current
	return func(log *mirLog) {
		log.logLevel = current
		switch current {
		case "trace":
			log.Logger = log.Level(zerolog.TraceLevel)
			zerolog.SetGlobalLevel(zerolog.TraceLevel)
		case "debug":
			log.Logger = log.Level(zerolog.DebugLevel)
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		case "info":
			log.Logger = log.Level(zerolog.InfoLevel)
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
		case "warn":
			log.Logger = log.Level(zerolog.WarnLevel)
			zerolog.SetGlobalLevel(zerolog.WarnLevel)
		case "error":
			log.Logger = log.Level(zerolog.ErrorLevel)
			zerolog.SetGlobalLevel(zerolog.ErrorLevel)
		case "fatal":
			log.Logger = log.Level(zerolog.FatalLevel)
			zerolog.SetGlobalLevel(zerolog.FatalLevel)
		default:
			log.Logger = log.Level(zerolog.InfoLevel)
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
		}
	}
}

func WithFlagLogLevel(flagIsDebug bool, flagLogLevel LogLevel) func(*mirLog) {
	var current LogLevel
	if flagIsDebug {
		current = LogLevelDebug
	} else if flagLogLevel != "" {
		current = flagLogLevel
	}
	return func(log *mirLog) {
		log.logLevel = current
		switch current {
		case "trace":
			log.Logger = log.Level(zerolog.TraceLevel)
			zerolog.SetGlobalLevel(zerolog.TraceLevel)
		case "debug":
			log.Logger = log.Level(zerolog.DebugLevel)
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		case "info":
			log.Logger = log.Level(zerolog.InfoLevel)
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
		case "warn":
			log.Logger = log.Level(zerolog.WarnLevel)
			zerolog.SetGlobalLevel(zerolog.WarnLevel)
		case "error":
			log.Logger = log.Level(zerolog.ErrorLevel)
			zerolog.SetGlobalLevel(zerolog.ErrorLevel)
		case "fatal":
			log.Logger = log.Level(zerolog.FatalLevel)
			zerolog.SetGlobalLevel(zerolog.FatalLevel)
		default:
			log.Logger = log.Level(zerolog.InfoLevel)
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
		}
	}
}

func WithCustomWriter(w io.Writer) func(*mirLog) {
	return func(l *mirLog) {
		log.Logger = log.Output(w)
		l.Logger = l.Output(w)
		l.hasWriter = true
	}
}

func WithCustomWriters(writers []io.Writer) func(*mirLog) {
	return func(l *mirLog) {
		w := io.MultiWriter(writers...)
		log.Logger = log.Output(w)
		l.Logger = l.Output(w)
		l.hasWriter = true
	}
}

// Will only set pretty logger if the process is not PID 0
// aka not running in container
func WithDevOnlyPrettyLogger() func(*mirLog) {
	return func(l *mirLog) {
		if isNotPidZero {
			out := log.Output(zerolog.ConsoleWriter{
				Out:     os.Stdout,
				NoColor: false,
			})
			log.Logger = out
			l.Logger = out
			l.hasWriter = true
		}
	}
}

func WithPrettyLogger() func(*mirLog) {
	return func(l *mirLog) {
		out := log.Output(zerolog.ConsoleWriter{
			Out:     os.Stdout,
			NoColor: false,
		})
		log.Logger = out
		l.Logger = out
		l.hasWriter = true
	}
}

func WithXdgConfigHomeLogFile(path string, file *os.File) func(*mirLog) {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("$HOME is not defined")
		userHomeDir = "./"
	}
	filePath := filepath.Join(userHomeDir, ".config", path)
	if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
		fmt.Println("Failed to create directories:", err)
		os.Exit(1)
	}
	file, err = os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Failed to open log file:", err)
		os.Exit(1)
	}
	return func(l *mirLog) {
		log.Logger = log.Output(file)
		l.Logger = l.Output(file)
		l.hasWriter = true
	}
}

func WithAppName(name string) func(*mirLog) {
	return func(l *mirLog) {
		log.Logger = log.With().Str("cmd", name).Logger()
		l.Logger = l.With().Str("cmd", name).Logger()
	}
}

func WithKeyValue(key, value string) func(*mirLog) {
	return func(l *mirLog) {
		log.Logger = log.With().Str(key, value).Logger()
		l.Logger = l.With().Str(key, value).Logger()
	}
}
