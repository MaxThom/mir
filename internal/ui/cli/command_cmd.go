package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/maxthom/mir/internal/clients/cmd_client"
	"github.com/maxthom/mir/internal/libs/editor"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	cmd_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/cmd_api"
	common_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/common_api"
	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
	"github.com/pkg/errors"
)

type CommandCmd struct {
	List CommandListCmd `cmd:"" aliases:"ls" help:"List all commands belonging to a set of devices"`
	Send CommandSendCmd `cmd:"" help:"Send a commands to all targeted devices"`
}

type CommandListCmd struct {
	Target           `embed:"" prefix:"target."`
	NameNs           string            `name:"name/namespace" arg:"" optional:"" help:"edit single device"`
	FilterLabels     map[string]string `help:"Set of labels to filter commands"`
	RefreshSchema    bool              `short:"r" help:"Refresh schema from device even if in store" default:"false"`
	ShowJsonTemplate bool              `short:"j" help:"show json template for command"`
}

type CommandSendCmd struct {
	Target           `embed:"" prefix:"target."`
	NameNs           string `name:"name/namespace" arg:"" optional:"" help:"edit single device"`
	Command          string `short:"n" help:"command name to send"`
	ShowJsonTemplate bool   `short:"j" help:"show json template for command"`
	Payload          string `short:"p" help:"payload to send in json. use single quote for easier writing. e.g. '{\"key\":\"value\"}'"`
	Edit             bool   `short:"e" help:"Interactive edit of command payload" default:"false"`
	RefreshSchema    bool   `short:"r" help:"Refresh schema from device even if in store" default:"false"`
	DryRun           bool   `help:"dry run command" default:"false"`
	NoValidation     bool   `help:"do not validate command with device's schema. Only for protobuf encoding" default:"false"`
	ForcePush        bool   `short:"f" help:"force send commands even if some devices are in error" default:"false"`
	Timeout          int    `short:"t" help:"timeout in second for command to reach device" default:"10"`
}

func (d *CommandListCmd) Validate() error {
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

func (d *CommandListCmd) Run(c CLI) error {
	var err error
	msgBus, err := bus.New(c.Target)
	if err != nil {
		e := MirConnectionError{Target: c.Target, e: err}
		return e
	}
	defer msgBus.Close()

	req := &cmd_apiv1.SendListCommandsRequest{
		Targets: &core_apiv1.Targets{
			Ids:        d.Target.Ids,
			Names:      d.Target.Names,
			Namespaces: d.Target.Namespaces,
			Labels:     d.Target.Labels,
		},
		FilterLabels:  d.FilterLabels,
		RefreshSchema: d.RefreshSchema,
	}
	resp, err := cmd_client.PublishListCommandsRequest(msgBus, req)
	if err != nil {
		return fmt.Errorf("error publising list command request: %w", err)
	}
	if resp.GetError() != "" {
		return errors.New(resp.GetError())
	}

	tpls := map[string][]string{}
	var sb strings.Builder
	for devNameNs, cmds := range resp.GetOk().DeviceCommands {
		if cmds.Error != "" {
			sb.WriteString(cmds.Error)
		} else {
			for _, cmd := range cmds.Commands {
				sb.WriteString(cmd.Name)
				sb.WriteString("{")
				sb.WriteString(mapToSortedString(cmd.Labels))
				sb.WriteString("}\n")
				if cmd.Error != "" {
					sb.WriteString(cmd.Error)
					sb.WriteString("\n")
				} else if d.ShowJsonTemplate {
					var prettyJSON bytes.Buffer
					if err = json.Indent(&prettyJSON, []byte(cmd.Template), "", "  "); err != nil {
						sb.WriteString(err.Error())
					} else {
						sb.WriteString(prettyJSON.String())
					}
					sb.WriteString("\n")
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

func (d *CommandSendCmd) Validate() error {
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

	if piped, ok := ReadFromPipedStdIn(); ok {
		d.Payload = piped
	}
	if d.Command != "" && d.Payload == "" && !d.ShowJsonTemplate && !d.Edit {
		err.Details = append(err.Details, "Must set payload. Use -j to see json template or -e for interactive edit.")
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
		return e
	}
	defer msgBus.Close()

	if d.Command == "" {
		listCmd := CommandListCmd{
			Target:           d.Target,
			NameNs:           d.NameNs,
			RefreshSchema:    d.RefreshSchema,
			ShowJsonTemplate: d.ShowJsonTemplate,
		}
		return listCmd.Run(c)
	}

	if d.Edit {
		d.ShowJsonTemplate = true
	}

	req := &cmd_apiv1.SendCommandRequest{
		Targets: &core_apiv1.Targets{
			Ids:        d.Target.Ids,
			Names:      d.Target.Names,
			Namespaces: d.Target.Namespaces,
			Labels:     d.Target.Labels,
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
		return fmt.Errorf("error publising send command request: %w", err)
	}
	if resp.GetError() != "" {
		return errors.New(resp.GetError())
	}

	if req.ShowTemplate {
		tpls := map[string][]string{}
		for k, v := range resp.GetOk().DeviceResponses {
			if v.Error != "" {
				tpls[string(v.Error)] = append(tpls[string(v.Error)], k)
				if d.Edit {
					return fmt.Errorf("Cannot edit json template with error response: %s", v.Error)
				}
			} else {
				var prettyJSON bytes.Buffer
				if err = json.Indent(&prettyJSON, v.Payload, "", "  "); err != nil {
					tpls[err.Error()] = append(tpls[err.Error()], k)
				} else {
					tpls[prettyJSON.String()] = append(tpls[prettyJSON.String()], k)
				}
			}
		}

		var sb strings.Builder
		if len(tpls) == 1 {
			for k := range tpls {
				sb.WriteString(k)
			}
		} else {
			if d.Edit {
				return errors.New("Cannot edit multiple json templates. Refine targets to get single json or use -j to see.")
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
		}

		if d.Edit {
			header := []string{
				"Edit the command payload below",
				"On exit, the payload will be sent to the selected targets",
			}
			payload := []byte(sb.String())
			err = editor.EditRawDocument(&payload, header)
			req.ShowTemplate = false
			req.Payload = payload
			resp, err = cmd_client.PublishSendCommandRequest(msgBus, req)
			if err != nil {
				return fmt.Errorf("error publising send command request: %w", err)
			}
			if resp.GetError() != "" {
				return errors.New(resp.GetError())
			}
		} else {
			fmt.Println(sb.String())
			return nil
		}
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
