package cli

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/maxthom/mir/pkgs/mir_v1"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/rs/zerolog"
)

// TODO get command which is ls but with -o yaml
type EventCmd struct {
	List   EventListCmd   `cmd:"" aliases:"ls" help:"List events"`
	Delete EventDeleteCmd `cmd:"" help:"Delete events"`
}

type EventListCmd struct {
	Output      string `short:"o" help:"output format for response [pretty|json|yaml]" default:"pretty"`
	NameNs      string `name:"name/namespace" arg:"" optional:"" help:"list single event."`
	TargetEvent `embed:"" prefix:"target."`
	Limit       int       `help:"Limit number of events in the ouput"`
	From        time.Time `help:"Set starting date to filter event. (eg: 2025-05-01T00:00:00.00Z)"`
	To          time.Time `help:"Set ending date to filter event. Default to now. (eg: 2025-05-02T00:00:00.00Z)"`
}

type EventDeleteCmd struct {
	Output      string `short:"o" help:"output format for response [pretty|json|yaml]" default:"yaml"`
	NameNs      string `name:"name/namespace" arg:"" optional:"" help:"delete single event."`
	TargetEvent `embed:"" prefix:"target."`
	From        time.Time `help:"Set starting date to filter event. (eg: 2025-05-01T00:00:00.00Z)"`
	To          time.Time `help:"Set ending date to filter event. Default to now. (eg: 2025-05-02T00:00:00.00Z)"`
}

type TargetEvent struct {
	Names      []string          `help:"List of events to fetch by names"`
	Namespaces []string          `help:"List of events to fetch by namespaces"`
	Labels     map[string]string `help:"Set of labels to filter events"`
}

func (d EventCmd) Run() error {
	return nil
}

func (d *EventListCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}
	if strings.ToLower(d.Output) != "pretty" && strings.ToLower(d.Output) != "yaml" && strings.ToLower(d.Output) != "json" {
		d.Output = "pretty"
	}

	if d.NameNs != "" {
		tar := getTargetFromNameNs(d.NameNs)
		d.TargetEvent = TargetEvent{
			Names:      tar.Names,
			Namespaces: tar.Namespaces,
			Labels:     tar.Labels,
		}
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *EventListCmd) Run(log zerolog.Logger, m *mir.Mir, cfg Config) error {
	list, err := m.Server().ListEvents().Request(
		mir_v1.EventTarget{
			ObjectTarget: mir_v1.ObjectTarget{
				Names:      d.Names,
				Namespaces: d.Namespaces,
				Labels:     d.Labels,
			},
			DateFilter: mir_v1.DateFilter{
				From: d.From,
				To:   d.To,
			},
			Limit: d.Limit,
		})
	if err != nil {
		return fmt.Errorf("error publising list event request: %w", err)
	}

	if d.Output == "pretty" && len(list) == 1 {
		d.Output = "yaml"
	}
	if str, err := stringifyEvents(d.Output, list); err != nil {
		return fmt.Errorf("error marshalling response: %w", err)
	} else {
		fmt.Println(str)
	}

	return nil
}

func (d *EventDeleteCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}
	if strings.ToLower(d.Output) != "pretty" && strings.ToLower(d.Output) != "yaml" && strings.ToLower(d.Output) != "json" {
		d.Output = "pretty"
	}

	if d.NameNs != "" {
		tar := getTargetFromNameNs(d.NameNs)
		d.TargetEvent = TargetEvent{
			Names:      tar.Names,
			Namespaces: tar.Namespaces,
			Labels:     tar.Labels,
		}
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *EventDeleteCmd) Run(log zerolog.Logger, m *mir.Mir, cfg Config) error {
	list, err := m.Server().DeleteEvents().Request(mir_v1.EventTarget{
		ObjectTarget: mir_v1.ObjectTarget{
			Names:      d.Names,
			Namespaces: d.Namespaces,
			Labels:     d.Labels,
		},
		DateFilter: mir_v1.DateFilter{
			From: d.From,
			To:   d.To,
		},
	})
	if err != nil {
		return fmt.Errorf("error publising delete event request: %w", err)
	}

	if d.Output == "pretty" && len(list) == 1 {
		d.Output = "yaml"
	}
	if str, err := stringifyEvents(d.Output, list); err != nil {
		return fmt.Errorf("error marshalling response: %w", err)
	} else {
		fmt.Println(str)
	}

	return nil
}

func stringifyEvents(output string, events []mir_v1.Event) (string, error) {
	switch output {
	case "json":
		return marshalResponse(output, events)
	case "yaml":
		return marshalResponse(output, events)
	case "pretty":
		return prettyStringEvents(events), nil
	}
	return "", errors.New("invalid output format")
}

func prettyStringEvents(events []mir_v1.Event) string {
	format := "%-16s %-30s %-8s %-20s %-60s\n"
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(format, "AGE", "NAMESPACE/NAME", "TYPE", "REASON", "MESSAGE"))

	// TODO sort namespace, time
	sort.Slice(events, func(i, j int) bool {
		if events[i].Meta.Namespace == events[j].Meta.Namespace {
			if events[i].Status.FirstAt.Equal(events[j].Status.FirstAt) {
				return events[i].Meta.Name < events[j].Meta.Name
			} else {
				return events[i].Status.FirstAt.After(events[j].Status.FirstAt)
			}
		} else {
			return events[i].Meta.Namespace < events[j].Meta.Namespace
		}
	})

	for _, d := range events {
		st := ""
		if d.Spec.Type == mir_v1.EventTypeNormal {
			st = "normal"
		} else if d.Spec.Type == mir_v1.EventTypeWarning {
			st = "warning"
		} else {
			st = "normal"
		}

		age := ""
		if !d.Status.FirstAt.IsZero() {
			age = prettyDuration(time.Now().UTC().Sub(d.Status.FirstAt))
		}

		sb.WriteString(fmt.Sprintf(format, age, d.Meta.Namespace+"/"+d.Meta.Name, st, d.Spec.Reason, d.Spec.Message))
	}
	return sb.String()
}

func prettyDuration(d time.Duration) string {
	days := d / (24 * time.Hour)
	d -= days * 24 * time.Hour

	hours := d / time.Hour
	d -= hours * time.Hour

	minutes := d / time.Minute
	d -= minutes * time.Minute

	seconds := d / time.Second

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}
