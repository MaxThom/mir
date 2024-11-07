package cli

import (
	"github.com/rs/zerolog"
)

var l zerolog.Logger

type CLI struct {
	Target string
}

func New(log zerolog.Logger, mirUrl string) CLI {
	l = log.With().Str("app", "cli").Logger()
	return CLI{Target: mirUrl}
}
