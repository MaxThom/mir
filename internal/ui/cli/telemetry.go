package cli

import (
	"fmt"

	"github.com/maxthom/mir/internal/clients/tlm_client"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
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

	if len(d.Target.Ids) == 0 &&
		len(d.Target.Names) == 0 &&
		len(d.Target.Namespaces) == 0 &&
		len(d.Target.Labels) == 0 &&
		len(d.Target.Anno) == 0 {
		err.Details = append(err.Details, "Must specify targets")
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

	req := &tlm_apiv1.SendListTelemetryRequest{}
	resp, err := tlm_client.PublishTelemetryListRequest(msgBus, req)
	if err != nil {
		e := MirRequestError{Route: "telemetry.list", e: err}
		return e
	}
	fmt.Println(resp)
	return nil
}
