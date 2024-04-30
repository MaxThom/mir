package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/maxthom/mir/api/gen/proto/v1alpha/core"
	bus "github.com/maxthom/mir/libs/external/natsio"
	client_core "github.com/maxthom/mir/services/core"
	"gopkg.in/yaml.v3"
)

// TODO find how to move output here instead of per command
// TODO set yaml indent to two spaces
// TODO add random guid id for device create
// TODO check if json to remove key value pair should be NONE or NULL. check json doc
type DeviceCmd struct {
	List   DeviceListCmd   `cmd:"" help:"List devices"`
	Create DeviceCreateCmd `cmd:"" help:"Create a new device"`
	Update DeviceUpdateCmd `cmd:"" help:"Update a device"`
	Delete DeviceDeleteCmd `cmd:"" help:"Delete a device"`
}

type DeviceListCmd struct {
	Output string `short:"o" help:"output format for response" default:"json"`
	Target `embed:"" prefix:"target."`
}

type DeviceCreateCmd struct {
	Output string `short:"o" help:"output format for response [json|yaml]" default:"json"`

	Id     string            `help:"Set device id"`
	Desc   string            `help:"Set device description"`
	Labels map[string]string `help:"Set labels to uniquely tag the device"`
	Anno   map[string]string `help:"Set annotations to add extra information to the device"`
}

type DeviceUpdateCmd struct {
	Output string `short:"o" help:"output format for response" default:"json"`
	Target `embed:"" prefix:"target."`

	Desc   *string           `help:"Set device description"`
	Labels map[string]string `help:"Set labels to uniquely tag the device"`
	Anno   map[string]string `help:"Set annotations to add extra information to the device"`
}

type DeviceDeleteCmd struct {
	Output string `short:"o" help:"output format for response" default:"json"`
	Target `embed:"" prefix:"target."`
}

type Target struct {
	Ids    []string          `help:"List of device to fetch by ids"`
	Labels map[string]string `help:"Set of labels to filter devices"`
	Anno   map[string]string `help:"Set of annotations to filter devices"`
}

func (d *DeviceCmd) Run(globals *Globals) error {
	return nil
}

func (d *DeviceListCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}
	if strings.ToLower(d.Output) != "yaml" && strings.ToLower(d.Output) != "json" {
		err.Details = append(err.Details, "The output format is invalid")
	}
	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *DeviceListCmd) Run(globals *Globals) error {
	var err error
	msgBus, err = bus.New(globals.Target)
	if err != nil {
		return MirConnectionError{Target: globals.Target, e: err}
	}
	defer msgBus.Close()

	resp, err := client_core.PublishDeviceListRequest(ctx, msgBus, &core.ListDeviceRequest{
		Targets: &core.Targets{
			Ids:         d.Ids,
			Labels:      d.Labels,
			Annotations: d.Anno,
		},
	})

	if err != nil {
		return MirRequestError{Route: "device.list", e: err}
	}

	if resp.GetError() != nil {
		e := MirResponseError{
			Route: "device.list",
			e: MirHttpError{
				Code:    resp.GetError().GetCode(),
				Message: resp.GetError().GetMessage(),
				Details: resp.GetError().GetDetails(),
			}}
		fmt.Println(e)
		return e
	} else {
		if out, e := MarhsalResponse(d.Output, resp.GetOk()); e != nil {
			fmt.Println(e)
			return e
		} else {
			fmt.Println(out)
		}
	}
	return nil
}

func (d *DeviceCreateCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}
	if d.Id == "" {
		err.Details = append(err.Details, "The device ID is mandatory")
	}
	if strings.ToLower(d.Output) != "yaml" && strings.ToLower(d.Output) != "json" {
		err.Details = append(err.Details, "The output format is invalid")
	}
	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *DeviceCreateCmd) Run(globals *Globals) error {
	var err error
	msgBus, err = bus.New(globals.Target)
	if err != nil {
		return MirConnectionError{Target: globals.Target, e: err}
	}
	defer msgBus.Close()

	resp, err := client_core.PublishDeviceCreateRequest(ctx, msgBus, &core.CreateDeviceRequest{
		DeviceId:    d.Id,
		Description: d.Desc,
		Labels:      d.Labels,
		Annotations: d.Anno,
	})

	if err != nil {
		return MirRequestError{Route: "device.create", e: err}
	}

	if resp.GetError() != nil {
		e := MirResponseError{
			Route: "device.create",
			e: MirHttpError{
				Code:    resp.GetError().GetCode(),
				Message: resp.GetError().GetMessage(),
				Details: resp.GetError().GetDetails(),
			}}
		fmt.Println(e)
		return e
	} else {
		if out, e := MarhsalResponse(d.Output, resp.GetOk()); e != nil {
			fmt.Println(e)
			return e
		} else {
			fmt.Println(out)
		}
	}
	return nil
}

func (d *DeviceUpdateCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}

	if strings.ToLower(d.Output) != "yaml" && strings.ToLower(d.Output) != "json" {
		err.Details = append(err.Details, "The output format is invalid")
	}

	if len(d.Target.Ids) == 0 &&
		len(d.Target.Labels) == 0 &&
		len(d.Target.Anno) == 0 {
		err.Details = append(err.Details, "Must specify targets")
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *DeviceUpdateCmd) Run(globals *Globals) error {
	var err error
	msgBus, err = bus.New(globals.Target)
	if err != nil {
		return MirConnectionError{Target: globals.Target, e: err}
	}
	defer msgBus.Close()

	labels := map[string]*core.UpdateDeviceRequest_OptString{}
	for k, v := range d.Labels {
		if strings.ToLower(v) == "none" {
			labels[k] = &core.UpdateDeviceRequest_OptString{
				Value: nil,
			}
		} else {
			labels[k] = &core.UpdateDeviceRequest_OptString{
				Value: &v,
			}
		}
	}
	anno := map[string]*core.UpdateDeviceRequest_OptString{}
	for k, v := range d.Anno {
		if strings.ToLower(v) == "none" {
			anno[k] = &core.UpdateDeviceRequest_OptString{
				Value: nil,
			}
		} else {
			anno[k] = &core.UpdateDeviceRequest_OptString{
				Value: &v,
			}
		}
	}
	resp, err := client_core.PublishDeviceUpdateRequest(ctx, msgBus, &core.UpdateDeviceRequest{
		Targets: &core.Targets{
			Ids:         d.Target.Ids,
			Labels:      d.Target.Labels,
			Annotations: d.Target.Anno,
		},
		Description: d.Desc,
		Labels:      labels,
		Annotations: anno,
	})

	if err != nil {
		return MirRequestError{Route: "device.update", e: err}
	}

	if resp.GetError() != nil {
		e := MirResponseError{
			Route: "device.update",
			e: MirHttpError{
				Code:    resp.GetError().GetCode(),
				Message: resp.GetError().GetMessage(),
				Details: resp.GetError().GetDetails(),
			}}
		fmt.Println(e)
		return e
	} else {
		if out, e := MarhsalResponse(d.Output, resp.GetOk()); e != nil {
			fmt.Println(e)
			return e
		} else {
			fmt.Println(out)
		}
	}

	return nil
}

func (d *DeviceDeleteCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}

	if len(d.Target.Ids) == 0 &&
		len(d.Target.Labels) == 0 &&
		len(d.Target.Anno) == 0 {
		err.Details = append(err.Details, "Must specify targets")
	}

	if strings.ToLower(d.Output) != "yaml" && strings.ToLower(d.Output) != "json" {
		err.Details = append(err.Details, "The output format is invalid")
	}
	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *DeviceDeleteCmd) Run(globals *Globals) error {
	var err error
	msgBus, err = bus.New(globals.Target)
	if err != nil {
		return MirConnectionError{Target: globals.Target, e: err}
	}
	defer msgBus.Close()

	resp, err := client_core.PublishDeviceDeleteRequest(ctx, msgBus, &core.DeleteDeviceRequest{
		Targets: &core.Targets{
			Ids:         d.Ids,
			Labels:      d.Labels,
			Annotations: d.Anno,
		},
	})

	if err != nil {
		return MirRequestError{Route: "device.delete", e: err}
	}

	if resp.GetError() != nil {
		e := MirResponseError{
			Route: "device.delete",
			e: MirHttpError{
				Code:    resp.GetError().GetCode(),
				Message: resp.GetError().GetMessage(),
				Details: resp.GetError().GetDetails(),
			}}
		fmt.Println(e)
		return e
	} else {
		if out, e := MarhsalResponse(d.Output, resp.GetOk()); e != nil {
			fmt.Println(e)
			return e
		} else {
			fmt.Println(out)
		}
	}
	return nil
}

func MarhsalResponse(format string, v any) (string, error) {
	var out []byte
	var e error
	if strings.ToLower(format) == "json" {
		var err error
		out, err = json.MarshalIndent(v, "", "  ")
		if err != nil {
			e = MirSerializationError{
				Format: "JSON",
				e:      err,
			}
		}
	} else if strings.ToLower(format) == "yaml" {
		// TODO find how to set indentation to two spaces
		var err error
		out, err = yaml.Marshal(v)
		if err != nil {
			e = MirSerializationError{
				Format: "YAML",
				e:      err,
			}
		}
	}

	return string(out), e
}
