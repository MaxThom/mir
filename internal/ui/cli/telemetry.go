package cli

import (
	"fmt"
	"strings"

	"github.com/maxthom/mir/internal/clients/tlm_client"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
	tlm_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/tlm_api"
)

// TODO find how to move output here instead of per command
// TODO set yaml indent to two spaces
// TODO check if json to remove key value pair should be NONE or NULL. check json doc
type TelemetryCmd struct {
	List TelemetryListCmd `cmd:"" help:"Explore device telemetry"`
}

type TelemetryListCmd struct {
	Target        `embed:"" prefix:"target."`
	Measuremeants []string          `short:"m" help:"list of measurements to drill down. correspond to proto messages of type telemetry"`
	Filters       map[string]string `short:"f" help:"all available filters to drill down the query"`
	RefreshSchema bool              `short:"r" help:"Refresh schema from device even if in store" default:"false"`
	GrafanaUrl    string            `short:"g" help:"grafana instance url to point the generated link to" default:"localhost:3000"`
	Explore       bool              `short:"e" help:"generate the telemetry explore panel and link"`
	Query         bool              `short:"q" help:"show the telemetry query"`
}

func (d *TelemetryCmd) Run() error {
	return nil
}

func (d *TelemetryListCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
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
		fmt.Println(e)
		return e
	}
	defer msgBus.Close()

	req := &tlm_apiv1.SendListTelemetryRequest{
		Targets: &core_apiv1.Targets{
			Ids:         d.Target.Ids,
			Names:       d.Target.Names,
			Namespaces:  d.Target.Namespaces,
			Labels:      d.Target.Labels,
			Annotations: d.Target.Anno,
		},
		Measurements:  d.Measuremeants,
		Filters:       d.Filters,
		RefreshSchema: d.RefreshSchema,
	}
	resp, err := tlm_client.PublishTelemetryListRequest(msgBus, req)
	if err != nil {
		e := MirRequestError{Route: "telemetry.list", e: err}
		return e
	}
	if resp.GetError() != nil {
		e := MirResponseError{Route: "telemetry.list", e: fmt.Errorf(resp.GetError().Message)}
		return e
	}

	tpls := map[string][]string{}
	var sb strings.Builder
	for devNameNs, tlms := range resp.GetOk().DeviceTelemetry {
		if tlms.Error != "" {
			sb.WriteString(tlms.Error)
		} else {
			for _, tlm := range tlms.TlmDescriptors {
				sb.WriteString(tlm.Name)
				sb.WriteString("{")
				sb.WriteString(mapToSortedString(tlm.Labels))
				sb.WriteString("}\n")
				if tlm.Error != "" {
					sb.WriteString(tlm.Error)
					sb.WriteString("\n")
				} else if d.Query {
					// TODO pretty print
					//sb.WriteString(tlm.Query)
					sb.WriteString("\n")
				} else if len(tlm.Fields) > 0 {
					for _, f := range tlm.Fields {
						sb.WriteString("    ")
						sb.WriteString(f)
						sb.WriteString("\n")
					}
				}
			}
		}
		tpls[sb.String()] = append(tpls[sb.String()], devNameNs)
		sb.Reset()
	}

	i := 1
	for k, v := range tpls {
		sb.WriteString(fmt.Sprintf("%d. ", i))
		sb.WriteString(strings.Join(v, ", "))
		sb.WriteString("\n")
		sb.WriteString(k)
		sb.WriteString("\n")
		i++
	}
	fmt.Println(sb.String())
	return nil
}
