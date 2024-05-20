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
	zerolog.Logger
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
	return Logger.With().Logger()
}

func WithTimeFormatUnix() func(*mirLog) {
	return func(log *mirLog) {
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
		log.timeFormat = zerolog.TimeFormatUnix
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

func WithCustomWriter(w io.Writer) func(*mirLog) {
	return func(l *mirLog) {
		log.Logger = log.Output(w)
		l.Logger = l.Output(w)
	}
}

func WithCustomWriters(writers []io.Writer) func(*mirLog) {
	return func(l *mirLog) {
		w := io.MultiWriter(writers...)
		log.Logger = log.Output(w)
		l.Logger = l.Output(w)
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
