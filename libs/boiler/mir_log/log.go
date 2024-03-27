package mir_log

import (
	"io"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type LogLevel = string

var (
	LogLevelTrace   LogLevel = "trace"
	LogLevelDebug   LogLevel = "debug"
	LogLevelInfo    LogLevel = "info"
	LogLevelWarning LogLevel = "warn"
	LogLevelError   LogLevel = "error"
	LogLevelFatal   LogLevel = "fatal"
)

type mirLog struct {
	timeFormat string
	logLevel   string
}

var GlobalLogger *mirLog

func Setup(options ...func(*mirLog)) {
	GlobalLogger = &mirLog{}
	for _, o := range options {
		o(GlobalLogger)
	}
}

func WithTimeFormatUnix() func(*mirLog) {
	return func(log *mirLog) {
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	}
}

func WithLogLevel(logLevel LogLevel) func(*mirLog) {
	return func(log *mirLog) {
		log.logLevel = logLevel
		switch logLevel {
		case "trace":
			zerolog.SetGlobalLevel(zerolog.TraceLevel)
		case "debug":
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		case "info":
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
		case "warn":
			zerolog.SetGlobalLevel(zerolog.WarnLevel)
		case "error":
			zerolog.SetGlobalLevel(zerolog.ErrorLevel)
		case "fatal":
			zerolog.SetGlobalLevel(zerolog.FatalLevel)
		default:
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
		}
	}
}

func WithCustomWriter(w io.Writer) func(*mirLog) {
	return func(l *mirLog) {
		log.Logger = log.Output(w)
	}
}
