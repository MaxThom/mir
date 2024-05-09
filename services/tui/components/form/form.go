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
	GetValue() string
}

type ValidateFn func(s string) error

const keyValuePairPattern = `^(\w+\s*=\s*[^;]+)(?:;\s*(\w+\s*=\s*[^;]+))*;?\s*$`

var keyValuePairRx = regexp.MustCompile(keyValuePairPattern)

func WithKeyValueMapValidator() func(s string) error {
	return func(s string) error {
		if s == "" {
			return nil
		}
		match := keyValuePairRx.MatchString(s)
		if !match {
			return fmt.Errorf("must be key value pairs <k1>=<v1>;<k2>=<v2>;...")
		}
		return nil
	}
}

func WithMandatoryValidator() func(s string) error {
	return func(s string) error {
		if s == "" {
			return fmt.Errorf("must not be empty")
		}
		return nil
	}
}

// The priority of error is the order of the functions
// IDEA could rework to join the errors together
func MirValidators(opts ...func(s string) error) func(s string) error {
	return func(s string) error {
		for _, o := range opts {
			if err := o(s); err != nil {
				return err
			}
		}
		return nil
	}
}
