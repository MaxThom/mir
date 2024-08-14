package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/maxthom/mir/internal/clients/core_client"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	"github.com/maxthom/mir/pkgs/api/proto/v1alpha/common_api"
	"github.com/maxthom/mir/pkgs/api/proto/v1alpha/core_api"
	"gopkg.in/yaml.v3"
)

// TODO find how to move output here instead of per command
// TODO set yaml indent to two spaces
// TODO check if json to remove key value pair should be NONE or NULL. check json doc
type DeviceCmd struct {
	List   DeviceListCmd   `cmd:"" help:"List devices"`
	Create DeviceCreateCmd `cmd:"" help:"Create a new device"`
	Update DeviceUpdateCmd `cmd:"" help:"Update a device"`
	Delete DeviceDeleteCmd `cmd:"" help:"Delete a device"`
	Schema DeviceSchemaCmd `cmd:"" help:"Upload and explore device proto schema"`
}

type DeviceListCmd struct {
	Output string `short:"o" help:"output format for response" default:"json"`
	Target `embed:"" prefix:"target."`
}

type DeviceCreateCmd struct {
	Output string `short:"o" help:"output format for response [json|yaml]" default:"json"`

	RandomId  bool              `short:"r" help:"Set a random device id"`
	Id        string            `help:"Set device id"`
	Name      string            `help:"Set device name"`
	Namespace string            `help:"Set device namespace"`
	Desc      string            `help:"Set device description"`
	Disabled  bool              `help:"if disabled, communication is cut"`
	Labels    map[string]string `help:"Set labels to uniquely tag the device"`
	Anno      map[string]string `help:"Set annotations to add extra information to the device"`
}

type DeviceUpdateCmd struct {
	Output string `short:"o" help:"output format for response" default:"json"`
	Target `embed:"" prefix:"target."`

	Name      *string           `help:"Set device name"`
	Namespace *string           `help:"Set device namespace"`
	Desc      *string           `help:"Set device description"`
	Disabled  *bool             `help:"If not enabled, communication is cut"`
	Labels    map[string]string `help:"Set labels to uniquely tag the device (set to null to remove)"`
	Anno      map[string]string `help:"Set annotations to add extra information to the devie (set to null to remove)"`
}

type DeviceDeleteCmd struct {
	Output string `short:"o" help:"output format for response" default:"json"`
	Target `embed:"" prefix:"target."`
}

type Target struct {
	Ids        []string          `help:"List of device to fetch by ids"`
	Names      []string          `help:"List of device to fetch by names"`
	Namespaces []string          `help:"List of device to fetch by namespaces"`
	Labels     map[string]string `help:"Set of labels to filter devices"`
	Anno       map[string]string `help:"Set of annotations to filter devices"`
}

func (d *DeviceCmd) Run() error {
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

func (d *DeviceListCmd) Run(c CLI) error {
	var err error
	msgBus, err := bus.New(c.Target)
	if err != nil {
		e := MirConnectionError{Target: c.Target, e: err}
		fmt.Println(e)
		return e
	}
	defer msgBus.Close()

	resp, err := core_client.PublishDeviceListRequest(msgBus, &core_api.ListDeviceRequest{
		Targets: &core_api.Targets{
			Ids:         d.Ids,
			Names:       d.Names,
			Namespaces:  d.Namespaces,
			Labels:      d.Labels,
			Annotations: d.Anno,
		},
	})
	if err != nil {
		e := MirRequestError{Route: "device.list", e: err}
		fmt.Println(e)
		return e
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
	if d.Id == "" && !d.RandomId {
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

func (d *DeviceCreateCmd) Run(c CLI) error {
	var err error
	msgBus, err := bus.New(c.Target)
	if err != nil {
		e := MirConnectionError{Target: c.Target, e: err}
		fmt.Println(e)
		return e
	}
	defer msgBus.Close()

	if d.Anno == nil {
		d.Anno = make(map[string]string)
	}
	if d.Labels == nil {
		d.Labels = make(map[string]string)
	}

	if d.RandomId {
		t, err := uuid.NewRandom()
		if err != nil {
			e := MirProcessError{Msg: "error generating random device id}", e: err}
			fmt.Println(e)
			return e
		}
		d.Id = t.String()
	}
	d.Anno["mir/device/description"] = d.Desc
	resp, err := core_client.PublishDeviceCreateRequest(msgBus, &core_api.CreateDeviceRequest{
		DeviceId:    d.Id,
		Name:        d.Name,
		Namespace:   d.Namespace,
		Disabled:    d.Disabled,
		Labels:      d.Labels,
		Annotations: d.Anno,
	})

	if err != nil {
		e := MirRequestError{Route: "device.create", e: err}
		fmt.Println(e)
		return e
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

func (d *DeviceUpdateCmd) Run(c CLI) error {
	var err error
	msgBus, err := bus.New(c.Target)
	if err != nil {
		e := MirConnectionError{Target: c.Target, e: err}
		fmt.Println(e)
		return e
	}
	defer msgBus.Close()

	labels := map[string]*common_api.OptString{}
	for k, v := range d.Labels {
		if strings.ToLower(v) == "null" {
			labels[k] = &common_api.OptString{
				Value: nil,
			}
		} else {
			labels[k] = &common_api.OptString{
				Value: &v,
			}
		}
	}
	anno := map[string]*common_api.OptString{}
	for k, v := range d.Anno {
		if strings.ToLower(v) == "null" {
			anno[k] = &common_api.OptString{
				Value: nil,
			}
		} else {
			anno[k] = &common_api.OptString{
				Value: &v,
			}
		}
	}
	if d.Desc != nil {
		anno["mir/device/description"] = &common_api.OptString{
			Value: d.Desc,
		}
	}
	req := &core_api.UpdateDeviceRequest{
		Targets: &core_api.Targets{
			Ids:         d.Target.Ids,
			Names:       d.Target.Names,
			Namespaces:  d.Target.Namespaces,
			Labels:      d.Target.Labels,
			Annotations: d.Target.Anno,
		},
		Meta: &core_api.UpdateDeviceRequest_Meta{
			Name:        d.Name,
			Namespace:   d.Namespace,
			Labels:      labels,
			Annotations: anno,
		},
		Spec: &core_api.UpdateDeviceRequest_Spec{
			Disabled: d.Disabled,
		},
	}
	resp, err := core_client.PublishDeviceUpdateRequest(msgBus, req)
	if err != nil {
		e := MirRequestError{Route: "device.update", e: err}
		fmt.Println(e)
		return e
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
		len(d.Target.Names) == 0 &&
		len(d.Target.Namespaces) == 0 &&
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

func (d *DeviceDeleteCmd) Run(c CLI) error {
	var err error
	msgBus, err := bus.New(c.Target)
	if err != nil {
		e := MirConnectionError{Target: c.Target, e: err}
		fmt.Println(e)
		return e
	}
	defer msgBus.Close()

	resp, err := core_client.PublishDeviceDeleteRequest(msgBus, &core_api.DeleteDeviceRequest{
		Targets: &core_api.Targets{
			Ids:         d.Target.Ids,
			Names:       d.Target.Names,
			Namespaces:  d.Target.Namespaces,
			Labels:      d.Target.Labels,
			Annotations: d.Target.Anno,
		},
	})
	if err != nil {
		e := MirRequestError{Route: "device.delete", e: err}
		fmt.Println(e)
		return e
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
