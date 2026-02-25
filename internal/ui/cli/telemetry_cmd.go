package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/maxthom/mir/internal/libs/external/grafana"
	"github.com/maxthom/mir/internal/ui"
	mir_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/mir_api/v1"
	"github.com/maxthom/mir/pkgs/mir_v1"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/rs/zerolog"
)

// TODO find how to move output here instead of per command
// TODO set yaml indent to two spaces
// TODO check if json to remove key value pair should be NONE or NULL. check json doc
type TelemetryCmd struct {
	List  TelemetryListCmd  `cmd:"" aliases:"ls" help:"Explore device telemetry"`
	Query TelemetryQueryCmd `cmd:"" aliases:"qry" help:"Query device telemetry"`
}

type TelemetryListCmd struct {
	Target        `embed:"" prefix:"target."`
	NameNs        string            `name:"name/namespace" arg:"" optional:"" help:"filter on name and/or namespace"`
	Measurements  []string          `short:"m" help:"list of measurements to display. correspond to proto messages of type telemetry"`
	Filters       map[string]string `short:"f" help:"labels to filter measurements"`
	ShowFields    bool              `short:"s" help:"show fields of the measurements" default:"false"`
	PrintQuery    bool              `short:"q" help:"Print Influx query" default:"false"`
	RefreshSchema bool              `short:"r" help:"Refresh schema from device even if in store" default:"false"`
}

type TelemetryQueryCmd struct {
	Output      string `short:"o" help:"output format for response [pretty|json|yaml|csv]" default:"pretty"`
	Target      `embed:"" prefix:"target."`
	NameNs      string    `name:"name/namespace" arg:"" optional:"" help:"filter on name and/or namespace"`
	Measurement string    `short:"m" help:"measurement to display. correspond to proto messages of type telemetry"`
	Fields      []string  `short:"f" help:"specific fields of the measurement to query,"`
	From        time.Time `help:"Set starting date to filter data. (eg: 2025-05-01T00:00:00.00Z)"`
	To          time.Time `help:"Set ending date to filter data. Default to now. (eg: 2025-05-02T00:00:00.00Z)"`
}

func (d *TelemetryCmd) Run() error {
	return nil
}

func (d *TelemetryListCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}

	if len(d.Target.Ids) == 0 &&
		len(d.Target.Names) == 0 &&
		len(d.Target.Namespaces) == 0 &&
		len(d.Target.Labels) == 0 &&
		d.NameNs == "" {
		err.Details = append(err.Details, "Must specify targets")
	}

	if d.NameNs != "" {
		d.Target = getTargetFromNameNs(d.NameNs)
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *TelemetryListCmd) Run(log zerolog.Logger, m *mir.Mir, cfg ui.Config) error {
	ctxCfg, _ := cfg.GetCurrentContext()

	req := &mir_apiv1.ListTelemetryRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids:        d.Target.Ids,
			Names:      d.Target.Names,
			Namespaces: d.Target.Namespaces,
			Labels:     d.Target.Labels,
		},
		Measurements:  d.Measurements,
		Filters:       d.Filters,
		RefreshSchema: d.RefreshSchema,
	}
	resp, err := m.Client().ListTelemetry().Request(req)
	if err != nil {
		return fmt.Errorf("error publishing telemtry list request: %w", err)
	}

	var sb strings.Builder
	i := 0
	tlmsErr := []*mir_apiv1.DevicesTelemetry{}
	for _, tlms := range resp {
		if tlms.Error != "" {
			tlmsErr = append(tlmsErr, tlms)
			continue
		} else {
			i += 1
			sb.WriteString(fmt.Sprintf("%d. ", i))

			devsTitle := []string{}
			for _, devId := range tlms.Ids {
				if devId.Name == "" && devId.Namespace == "" {
					devsTitle = append(devsTitle, devId.DeviceId)
				} else {
					devsTitle = append(devsTitle, devId.Name+"/"+devId.Namespace)
				}
			}
			if len(devsTitle) > 10 {
				sb.WriteString(strings.Join(devsTitle[0:10], ", ") + " & " + fmt.Sprintf("%d more", len(devsTitle)-10))
			} else {
				sb.WriteString(strings.Join(devsTitle, ", "))
			}
			sb.WriteString("\n")

			for _, tlm := range tlms.TlmDescriptors {
				sb.WriteString(tlm.Name)
				sb.WriteString("{")
				sb.WriteString(mapToSortedString(tlm.Labels))
				sb.WriteString("} ")
				sb.WriteString(FormatHyperlink(ctxCfg.Grafana+"/explore", grafana.CreateExploreLink(ctxCfg.Grafana, tlm.ExploreQuery)))
				sb.WriteString("\n")
				if tlm.Error != "" {
					sb.WriteString(tlm.Error)
					sb.WriteString("\n")
				} else if len(tlm.Fields) > 0 && d.ShowFields {
					for _, f := range tlm.Fields {
						sb.WriteString("    ")
						sb.WriteString(f)
						sb.WriteString("\n")
					}
				}
				if d.PrintQuery {
					sb.WriteString("  ")
					sb.WriteString(strings.ReplaceAll(tlm.ExploreQuery, "\n", "\n  "))
					sb.WriteString("\n")
				}
			}
		}
		sb.WriteString("\n")
	}

	for _, tlms := range tlmsErr {
		i += 1
		sb.WriteString(fmt.Sprintf("%d. ", i))

		devsTitle := []string{}
		for _, devId := range tlms.Ids {
			if devId.Name == "" && devId.Namespace == "" {
				devsTitle = append(devsTitle, devId.DeviceId)
			} else {
				devsTitle = append(devsTitle, devId.Name+"/"+devId.Namespace)
			}
		}
		if len(devsTitle) > 10 {
			sb.WriteString(strings.Join(devsTitle[0:10], ", ") + " & " + fmt.Sprintf("%d more", len(devsTitle)-10))
		} else {
			sb.WriteString(strings.Join(devsTitle, ", "))
		}

		sb.WriteString("\n")
		sb.WriteString(tlms.Error)
	}

	fmt.Println(sb.String())
	return nil
}

func (d *TelemetryQueryCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}

	if len(d.Target.Ids) == 0 &&
		len(d.Target.Names) == 0 &&
		len(d.Target.Namespaces) == 0 &&
		len(d.Target.Labels) == 0 &&
		d.NameNs == "" {
		err.Details = append(err.Details, "Must specify targets")
	}

	if d.NameNs != "" {
		d.Target = getTargetFromNameNs(d.NameNs)
	}

	if strings.ToLower(d.Output) != "pretty" && strings.ToLower(d.Output) != "yaml" && strings.ToLower(d.Output) != "json" && strings.ToLower(d.Output) != "csv" {
		d.Output = "pretty"
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *TelemetryQueryCmd) Run(log zerolog.Logger, m *mir.Mir, cfg ui.Config) error {
	req := &mir_apiv1.QueryTelemetryRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids:        d.Target.Ids,
			Names:      d.Target.Names,
			Namespaces: d.Target.Namespaces,
			Labels:     d.Target.Labels,
		},
		Measurement: d.Measurement,
		Fields:      d.Fields,
		StartTime:   mir_v1.AsProtoTimestamp(d.To),
		EndTime:     mir_v1.AsProtoTimestamp(d.From),
	}
	resp, err := m.Client().QueryTelemetry().Request(req)
	if err != nil {
		return fmt.Errorf("error publishing telemtry query request: %w", err)
	}

	var out string
	switch d.Output {
	case "json", "yaml":
		out, err = marshalResponse(d.Output, resp)
		if err != nil {
			return fmt.Errorf("error marhsalling telemtry query: %w", err)
		}
	case "csv":
		out = csvStringQuery(resp)
	case "pretty":
		out = prettyStringQuery(resp)
	}

	fmt.Println(out)
	return nil
}

func prettyStringQuery(query *mir_apiv1.QueryTelemetry) string {
	// format := "%-45s %-25s %-10s %-20s %-20s %s\n"
	// timeFormat := "2006-01-02 15:04:05"
	var sb strings.Builder
	// sb.WriteString(fmt.Sprintf(format, "NAMESPACE/NAME", "DEVICE_ID", "STATUS", "LAST_HEARTHBEAT", "LAST_SCHEMA_FETCH", "LABELS"))

	// 	sb.WriteString(fmt.Sprintf(format, d.Meta.Namespace+"/"+d.Meta.Name, d.Spec.DeviceId, st, hb, sf, formatLabels(d.Meta.Labels)))
	return sb.String()
}

func csvStringQuery(query *mir_apiv1.QueryTelemetry) string {
	// format := "%-45s %-25s %-10s %-20s %-20s %s\n"
	// timeFormat := "2006-01-02 15:04:05"
	var sb strings.Builder
	// sb.WriteString(fmt.Sprintf(format, "NAMESPACE/NAME", "DEVICE_ID", "STATUS", "LAST_HEARTHBEAT", "LAST_SCHEMA_FETCH", "LABELS"))

	// 	sb.WriteString(fmt.Sprintf(format, d.Meta.Namespace+"/"+d.Meta.Name, d.Spec.DeviceId, st, hb, sf, formatLabels(d.Meta.Labels)))
	return sb.String()
}

// OSC 8 hyperlink escape sequence
// Supported by most new terminal
func FormatHyperlink(text, url string) string {
	reset := "\033[0m"
	blue := "\033[34m"
	underline := "\033[4m"
	text = blue + underline + text + reset
	return fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", url, text)
}
