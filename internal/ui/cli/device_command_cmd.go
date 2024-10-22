package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/maxthom/mir/internal/clients/cmd_client"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	cmd_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/cmd_api"
	common_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/common_api"
	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
	"github.com/pkg/errors"
)

type DeviceCommandCmd struct {
	List CommandListCmd `cmd:"" help:"List all commands belonging to a set of devices"`
	Send CommandSendCmd `cmd:"" help:"Send a commands to all targeted devices"`
}

type CommandListCmd struct {
	Target        `embed:"" prefix:"target."`
	Output        string `short:"o" help:"output format for response" default:"json"`
	RefreshSchema bool   `short:"f" help:"Refresh schema from device even if in store" default:"false"`
}

type CommandSendCmd struct {
	Target           `embed:"" prefix:"target."`
	Command          string `short:"n" help:"command name to send"`
	ShowJsonTemplate bool   `short:"j" help:"show json template for command"`
	Payload          string `short:"p" help:"payload to send in json. use single quote for easier writing. e.g. '{\"key\":\"value\"}'"`
	RefreshSchema    bool   `short:"r" help:"Refresh schema from device even if in store" default:"false"`
	DryRun           bool   `help:"dry run command" default:"false"`
	NoValidation     bool   `help:"do not validate command with device's schema. Only for protobuf encoding" default:"false"`
	ForcePush        bool   `short:"f" help:"force send commands even if some devices are in error" default:"false"`
	Timeout          int    `short:"t" help:"timeout in second for command to reach device" default:"10"`
	Output           string `short:"o" help:"output format for response" default:"json"`
}

func (d *CommandListCmd) Validate() error {
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

func (d *CommandListCmd) Run(c CLI) error {
	var err error
	msgBus, err := bus.New(c.Target)
	if err != nil {
		e := MirConnectionError{Target: c.Target, e: err}
		fmt.Println(e)
		return e
	}
	defer msgBus.Close()

	req := &cmd_apiv1.SendListCommandsRequest{
		Targets: &core_apiv1.Targets{
			Ids:         d.Target.Ids,
			Names:       d.Target.Names,
			Namespaces:  d.Target.Namespaces,
			Labels:      d.Target.Labels,
			Annotations: d.Target.Anno,
		},
		RefreshSchema: d.RefreshSchema,
	}
	resp, err := cmd_client.PublishListCommandsRequest(msgBus, req)
	if err != nil {
		e := MirRequestError{Route: "command.list", e: err}
		fmt.Println(e)
		return e
	}
	if resp.GetError() != nil {
		e := MirResponseError{
			Route: "command.list",
			e: MirHttpError{
				Code:    resp.GetError().GetCode(),
				Message: resp.GetError().GetMessage(),
				Details: resp.GetError().GetDetails(),
			}}
		fmt.Println(e)
		return e
	}
	if out, e := MarhsalResponse(d.Output, resp.GetOk()); e != nil {
		fmt.Println(e)
		return e
	} else {
		fmt.Println(out)
	}
	return nil
}

func (d *CommandSendCmd) Validate() error {
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

	if d.Command == "" {
		err.Details = append(err.Details, "Must specify command name")
	}
	if piped, ok := ReadFromPipedStdIn(); ok {
		d.Payload = piped
	}
	if d.Payload == "" && !d.ShowJsonTemplate {
		err.Details = append(err.Details, "Must set payload. Use -p to see json template")
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *CommandSendCmd) Run(c CLI) error {
	var err error
	msgBus, err := bus.New(c.Target)
	if err != nil {
		e := MirConnectionError{Target: c.Target, e: err}
		fmt.Println(e)
		return e
	}
	defer msgBus.Close()

	req := &cmd_apiv1.SendCommandRequest{
		Targets: &core_apiv1.Targets{
			Ids:         d.Target.Ids,
			Names:       d.Target.Names,
			Namespaces:  d.Target.Namespaces,
			Labels:      d.Target.Labels,
			Annotations: d.Target.Anno,
		},
		Name:            d.Command,
		Payload:         []byte(d.Payload),
		PayloadEncoding: common_apiv1.Encoding_ENCODING_JSON,
		RefreshSchema:   d.RefreshSchema,
		ShowTemplate:    d.ShowJsonTemplate,
		DryRun:          d.DryRun,
		NoValidation:    d.NoValidation,
		ForcePush:       d.ForcePush,
		TimeoutSec:      uint32(d.Timeout),
	}
	resp, err := cmd_client.PublishSendCommandRequest(msgBus, req)
	if err != nil {
		e := MirRequestError{Route: "command.send", e: err}
		return e
	}
	if resp.GetError() != nil {
		e := MirResponseError{
			Route: "command.send",
			e: MirHttpError{
				Code:    resp.GetError().GetCode(),
				Message: resp.GetError().GetMessage(),
				Details: resp.GetError().GetDetails(),
			}}
		return e
	}

	var sb strings.Builder
	i := 1
	for k, v := range resp.GetOk().DeviceResponses {
		sb.WriteString(fmt.Sprintf("%d. ", i))
		sb.WriteString(k)
		sb.WriteString(" ")
		sb.WriteString(v.Status.String())
		sb.WriteString("\n")
		if v.Error != "" {
			sb.WriteString(v.Error)
			sb.WriteString("\n\n")
		} else if len(v.Payload) > 0 {
			sb.WriteString(v.Name)
			sb.WriteString("\n")

			var prettyJSON bytes.Buffer
			if err = json.Indent(&prettyJSON, v.Payload, "", "  "); err != nil {
				sb.WriteString(errors.Wrap(err, "error unmarshaling JSON in terminal").Error())
				sb.WriteString("\n\n")
			} else {
				sb.WriteString(prettyJSON.String())
				sb.WriteString("\n\n")
			}
		}
		i++
	}
	fmt.Println(sb.String())

	return nil
}
