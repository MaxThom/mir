package form

import (
	"fmt"
	"regexp"

	tea "github.com/charmbracelet/bubbletea"
)

type Control interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (Control, tea.Cmd)
	View() string
	Blur()
	Focus() tea.Cmd
	GetLabel() string
	GetTooltip() string
	Focused() bool
	GetErr() error
}

type ValidateFn func(s string) error

const keyValuePairPattern = `^(\w+\s*=\s*[^;]+)(?:;\s*(\w+\s*=\s*[^;]+))*;?\s*$`

var keyValuePairRx = regexp.MustCompile(keyValuePairPattern)

func KeyValueMapValidator(s string) error {
	if s == "" {
		return nil
	}
	match := keyValuePairRx.MatchString(s)
	if !match {
		return fmt.Errorf("must be key value pairs <k1>=<v1>;<k2>=<v2>;...")
	}
	return nil
}
