package cli

import (
	"fmt"
	"strings"

	"github.com/maxthom/mir/internal/libs/external/grafana"
	"github.com/maxthom/mir/internal/ui"
	mir_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/mir_api/v1"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/rs/zerolog"
)

// TODO find how to move output here instead of per command
// TODO set yaml indent to two spaces
// TODO check if json to remove key value pair should be NONE or NULL. check json doc
type TelemetryCmd struct {
	List TelemetryListCmd `cmd:"" aliases:"ls" help:"Explore device telemetry"`
}

type TelemetryListCmd struct {
	Target        `embed:"" prefix:"target."`
	NameNs        string            `name:"name/namespace" arg:"" optional:"" help:"filter on name and/or namespace"`
	Measuremeants []string          `short:"m" help:"list of measurements to display. correspond to proto messages of type telemetry"`
	Filters       map[string]string `short:"f" help:"labels to filter measurements"`
	ShowFields    bool              `short:"s" help:"show fields of the measurements" default:"false"`
	PrintQuery    bool              `short:"q" help:"Print Influx query" default:"false"`
	RefreshSchema bool              `short:"r" help:"Refresh schema from device even if in store" default:"false"`
}

func (d *TelemetryCmd) Run() error {
	return nil
}

func (d *TelemetryListCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
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

	req := &mir_apiv1.SendListTelemetryRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids:        d.Target.Ids,
			Names:      d.Target.Names,
			Namespaces: d.Target.Namespaces,
			Labels:     d.Target.Labels,
		},
		Measurements:  d.Measuremeants,
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
			sb.WriteString(strings.Join(tlms.DevicesNamens, ", "))
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
		sb.WriteString(strings.Join(tlms.DevicesNamens, ", "))
		sb.WriteString("\n")
		sb.WriteString(tlms.Error)
	}

	fmt.Println(sb.String())
	return nil
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
