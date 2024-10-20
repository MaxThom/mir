package cli

import (
	"fmt"
	"io"
	"os"

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

func ReadFromPipedStdIn() (string, bool) {
	fi, e := os.Stdin.Stat()
	if e != nil {
		return "", false
	}
	// 0 equal no pipe
	if fi.Mode()&os.ModeNamedPipe != 0 {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			return "", false
		}
		if len(data) > 0 {
			return string(data), true
		}
	}
	return "", false
}
