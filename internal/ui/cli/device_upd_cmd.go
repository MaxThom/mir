package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/maxthom/mir/internal/clients/core_client"
	"github.com/maxthom/mir/internal/libs/editor"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
	"github.com/maxthom/mir/pkgs/mir_models"
	"gopkg.in/yaml.v3"
)

type DeviceEditCmd struct {
	// TODO put back to yaml when ill figure it out
	Output string `short:"o" help:"output format for response [json|yaml]" default:"json"`
	Target `embed:"" prefix:"target."`
}

func (d *DeviceEditCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}

	if strings.ToLower(d.Output) != "yaml" && strings.ToLower(d.Output) != "json" {
		d.Output = "json"
	}

	if len(d.Target.Ids) == 0 &&
		len(d.Target.Names) == 0 &&
		len(d.Target.Namespaces) == 0 &&
		len(d.Target.Labels) == 0 {
		err.Details = append(err.Details, "Must specify targets")
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *DeviceEditCmd) Run(c CLI) error {
	var err error
	msgBus, err := bus.New(c.Target)
	if err != nil {
		e := MirConnectionError{Target: c.Target, e: err}
		fmt.Println(e)
		return e
	}
	defer msgBus.Close()

	respList, err := core_client.PublishDeviceListRequest(msgBus, &core_apiv1.ListDeviceRequest{
		Targets: &core_apiv1.Targets{
			Ids:        d.Ids,
			Names:      d.Names,
			Namespaces: d.Namespaces,
			Labels:     d.Labels,
		},
	})
	if err != nil {
		e := MirRequestError{Route: "device.list", e: err}
		return e
	}
	if respList.GetError() != nil {
		e := MirRequestError{Route: "device.list", e: errors.New(respList.GetError().Message)}
		return e
	}

	devs := mir_models.NewDeviceListFromProtoDevices(respList.GetOk().Devices)
	header := []string{
		"Edit the device below",
		"To remove a field, you must explicitly set it to null",
		"Only fields under meta, spec and properties.desired are editable",
	}
	if d.Output == "yaml" {
		if len(devs) == 1 {
			err = editor.EditYamlDocument(devs[0], header)
		} else {
			err = editor.EditYamlDocument(&devs, header)
		}
		if err != nil {
			e := MirEditError{e: err}
			return e
		}
	} else if d.Output == "json" {
		var err error
		if len(devs) == 1 {
			err = editor.EditJsonDocument(devs[0], header)
		} else {
			err = editor.EditJsonDocument(&devs, header)
		}
		if err != nil {
			e := MirEditError{e: err}
			return e
		}
	}

	var errs error
	respDevs := []*core_apiv1.Device{}
	for _, d := range devs {
		req := mir_models.NewUpdateDeviceReqFromDevice(*d)
		resp, err := core_client.PublishDeviceUpdateRequest(msgBus, req)
		if err != nil {
			e := MirRequestError{Route: "device.update", e: err}
			errs = errors.Join(errs, e)
		}
		if resp.GetError() != nil {
			e := MirResponseError{
				Route: "device.update",
				e: MirHttpError{
					Code:    resp.GetError().GetCode(),
					Message: resp.GetError().GetMessage(),
					Details: resp.GetError().GetDetails(),
				}}
			errs = errors.Join(errs, e)
		}
		respDevs = append(respDevs, resp.GetOk().Devices...)
	}
	if errs != nil {
		return errs
	}

	if d.Output == "pretty" {
		fmt.Println(prettyStringDevices(respDevs))
	} else {
		if out, e := MarshalResponse(d.Output, respDevs); e != nil {
			return e
		} else {
			fmt.Println(out)
		}
	}

	return nil
}

type DeviceApplyCmd struct {
	Path string `short:"f" help:"filepath to device definition. You can also pipe file content. Tips: use 'mir device list --targets... -o yaml > name.yaml' to get initial content"`
}

func (d *DeviceApplyCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *DeviceApplyCmd) Run(c CLI) error {
	var err error
	msgBus, err := bus.New(c.Target)
	if err != nil {
		e := MirConnectionError{Target: c.Target, e: err}
		return e
	}
	defer msgBus.Close()

	content, ok := ReadFromPipedStdIn()
	if !ok {
		contentB, err := os.ReadFile(d.Path)
		content = string(contentB)
		if err != nil {
			e := MirDeserializationError{e: err}
			return e
		}
	}

	var devs []*mir_models.Device
	var output string
	if isJsonString(content) {
		output = "json"
		if isJsonArray(content) {
			err = json.Unmarshal([]byte(content), &devs)
			if err != nil {
				e := MirDeserializationError{e: err}
				return e
			}
		} else {
			dev := &mir_models.Device{}
			err = json.Unmarshal([]byte(content), dev)
			if err != nil {
				e := MirDeserializationError{e: err}
				return e
			}
			devs = append(devs, dev)
		}
	} else {
		output = "yaml"
		if isYamlArray(content) {
			err = yaml.Unmarshal([]byte(content), &devs)
			if err != nil {
				e := MirDeserializationError{e: err}
				return e
			}
		} else {
			dev := &mir_models.Device{}
			err = yaml.Unmarshal([]byte(content), dev)
			if err != nil {
				e := MirDeserializationError{e: err}
				return e
			}
			devs = append(devs, dev)
		}
	}

	var errs error
	respDevs := []*core_apiv1.Device{}
	for _, d := range devs {
		req := mir_models.NewUpdateDeviceReqFromDevice(*d)
		resp, err := core_client.PublishDeviceUpdateRequest(msgBus, req)
		if err != nil {
			e := MirRequestError{Route: "device.update", e: err}
			errs = errors.Join(errs, e)
		}
		if resp.GetError() != nil {
			e := MirResponseError{
				Route: "device.update",
				e: MirHttpError{
					Code:    resp.GetError().GetCode(),
					Message: resp.GetError().GetMessage(),
					Details: resp.GetError().GetDetails(),
				}}
			errs = errors.Join(errs, e)
		}
		respDevs = append(respDevs, resp.GetOk().Devices...)
	}
	if errs != nil {
		return errs
	}

	if out, e := MarshalResponse(output, respDevs); e != nil {
		return e
	} else {
		fmt.Println(out)
	}

	return nil
}
