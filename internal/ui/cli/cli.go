package cli

import (
	"github.com/rs/zerolog"
)

var l zerolog.Logger

type Config struct {
	LogLevel string
	Target   string
}

func NewConfig(logLevel string, mirUrl string) Config {
	return Config{Target: mirUrl, LogLevel: logLevel}
}
