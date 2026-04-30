package cli

import (
	"fmt"
	"strings"

	"github.com/maxthom/mir/internal/ui"
)

type ContextCmd struct {
	Context string `arg:"" optional:"" help:"Set context"`
	Output  string `short:"o" help:"output format for response [pretty|json|yaml]" default:"pretty"`
}

func (d *ContextCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *ContextCmd) Run(cfg ui.Config) error {
	if d.Context != "" {
		ok := cfg.SetCurrentContext(d.Context)
		if !ok {
			fmt.Printf("context '%s' does not exist.\n\n", d.Context)
		} else {
			err := cfg.WriteConfig()
			if err != nil {
				return fmt.Errorf("failed to write config: %w", err)
			}
		}
	}

	data := struct {
		Contexts       []ui.Context `json:"contexts" yaml:"contexts"`
		CurrentContext string       `json:"currentContext" yaml:"currentContext"`
	}{
		Contexts:       cfg.Contexts,
		CurrentContext: cfg.CurrentContext,
	}

	var out string
	var err error
	switch d.Output {
	case "pretty":
		format := "%-20s %-30s %-30s %s\n"
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf(format, "NAME", "TARGET", "GRAFANA", "CREDENTIALS"))
		for _, c := range data.Contexts {
			st := c.Name
			if c.Name == data.CurrentContext {
				st = "*" + c.Name + ""
			}
			sb.WriteString(fmt.Sprintf(format, st, c.Target, c.Grafana, c.Sec.Credentials))
		}
		out = sb.String()
	case "json":
		out, err = marshalResponse("json", data)
	case "yaml":
		out, err = marshalResponse("yaml", data)
	}
	if err != nil {
		return err
	}
	fmt.Println(out)

	return nil
}
