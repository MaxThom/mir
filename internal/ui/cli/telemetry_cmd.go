package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/maxthom/mir/internal/clients/tlm_client"
	"github.com/maxthom/mir/internal/libs/external/grafana"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
	tlm_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/tlm_api"
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
	RefreshSchema bool              `short:"r" help:"Refresh schema from device even if in store" default:"false"`
	GrafanaUrl    string            `short:"g" help:"grafana instance url to point the generated link to" default:"localhost:3000"`
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

func (d *TelemetryListCmd) Run(c CLI) error {
	var err error
	msgBus, err := bus.New(c.Target)
	if err != nil {
		e := MirConnectionError{Target: c.Target, e: err}
		return e
	}
	defer msgBus.Close()

	req := &tlm_apiv1.SendListTelemetryRequest{
		Targets: &core_apiv1.DeviceTarget{
			Ids:        d.Target.Ids,
			Names:      d.Target.Names,
			Namespaces: d.Target.Namespaces,
			Labels:     d.Target.Labels,
		},
		Measurements:  d.Measuremeants,
		Filters:       d.Filters,
		RefreshSchema: d.RefreshSchema,
	}
	resp, err := tlm_client.PublishTelemetryListRequest(msgBus, req)
	if err != nil {
		return fmt.Errorf("error publishing telemtry list request: %w", err)
	}
	if resp.GetError() != "" {
		return errors.New(resp.GetError())
	}

	var sb strings.Builder
	i := 0
	tlmsErr := []*tlm_apiv1.DevicesTelemetry{}
	for _, tlms := range resp.GetOk().DevicesTelemetry {
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
				sb.WriteString(FormatHyperlink(d.GrafanaUrl+"/explore", grafana.CreateExploreLink(d.GrafanaUrl, tlm.ExploreQuery)))
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
