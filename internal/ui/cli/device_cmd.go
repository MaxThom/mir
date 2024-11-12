package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/maxthom/mir/internal/clients/core_client"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	common_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/common_api"
	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
	"github.com/maxthom/mir/pkgs/mir_models"
	"gopkg.in/yaml.v3"
)

// TODO set yaml indent to two spaces
type DeviceCmd struct {
	List   DeviceListCmd   `cmd:"" help:"List devices"`
	Create DeviceCreateCmd `cmd:"" help:"Create a new device"`
	Update DeviceUpdateCmd `cmd:"" help:"Update a device"`
	Edit   DeviceEditCmd   `cmd:"" help:"Interactive editing of devices"`
	Apply  DeviceApplyCmd  `cmd:"" help:"Update a device using a declarative format"`
	Delete DeviceDeleteCmd `cmd:"" help:"Delete a device"`
}

type DeviceListCmd struct {
	Output string `short:"o" help:"output format for response [pretty|json|yaml]" default:"pretty"`
	NameNs string `name:"name/namespace" arg:"" optional:"" help:"list single device."`
	Target `embed:"" prefix:"target."`
}

type DeviceCreateCmd struct {
	Output string `short:"o" help:"output format for response [pretty|json|yaml]" default:"pretty"`
	NameNs string `name:"name/namespace" arg:"" optional:"" help:"shortcut to set name and namespace"`

	ShowJsonTemplate bool              `short:"j" help:"Show json template for creating a device"`
	Path             string            `short:"f" help:"Filepath to device definition. You can also pipe file content. Tips: use 'mir device create -j > device.yaml' to get initial content"`
	RandomId         bool              `short:"r" help:"Set a random device id"`
	Id               string            `help:"Set device id"`
	Name             string            `help:"Set device name"`
	Namespace        string            `help:"Set device namespace"`
	Desc             string            `help:"Set device description"`
	Disabled         bool              `help:"If disabled, communication is cut"`
	Labels           map[string]string `help:"Set labels to uniquely tag the device"`
	Anno             map[string]string `help:"Set annotations to add extra information to the device"`
}

type DeviceUpdateCmd struct {
	Output string `short:"o" help:"output format for response [pretty|json|yaml]" default:"pretty"`
	Target `embed:"" prefix:"target."`

	Name      *string           `help:"Set device name"`
	Namespace *string           `help:"Set device namespace"`
	Desc      *string           `help:"Set device description"`
	Id        *string           `help:"Set device id"`
	Disabled  *bool             `help:"If not enabled, communication is cut"`
	Labels    map[string]string `help:"Set labels to uniquely tag the device (set to null, none or nil to remove)"`
	Anno      map[string]string `help:"Set annotations to add extra information to the devie (set to null to remove)"`
}

type DeviceDeleteCmd struct {
	Output string `short:"o" help:"output format for response [pretty|json|yaml]" default:"pretty"`
	NameNs string `name:"name/namespace" arg:"" optional:"" help:"delete single device."`
	Target `embed:"" prefix:"target."`
}

type Target struct {
	Ids        []string          `help:"List of device to fetch by ids"`
	Names      []string          `help:"List of device to fetch by names"`
	Namespaces []string          `help:"List of device to fetch by namespaces"`
	Labels     map[string]string `help:"Set of labels to filter devices"`
}

func (d *DeviceCmd) Run() error {
	return nil
}

func (d *DeviceListCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}
	if strings.ToLower(d.Output) != "pretty" && strings.ToLower(d.Output) != "yaml" && strings.ToLower(d.Output) != "json" {
		d.Output = "pretty"
	}

	if d.NameNs != "" {
		d.Target = getTargetFromNameNs(d.NameNs)
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

	resp, err := core_client.PublishDeviceListRequest(msgBus, &core_apiv1.ListDeviceRequest{
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

	if resp.GetError() != nil {
		e := MirResponseError{
			Route: "device.list",
			e: MirHttpError{
				Code:    resp.GetError().GetCode(),
				Message: resp.GetError().GetMessage(),
				Details: resp.GetError().GetDetails(),
			}}
		return e
	} else {
		list := mir_models.NewDeviceListFromProtoDevices(resp.GetOk().Devices)
		if d.Output == "pretty" {
			fmt.Println(prettyStringDevices(list))
		} else {
			if out, e := MarshalResponse(d.Output, list); e != nil {
				return e
			} else {
				fmt.Println(out)
			}
		}

	}
	return nil
}

func (d *DeviceCreateCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}
	if strings.ToLower(d.Output) != "pretty" && strings.ToLower(d.Output) != "yaml" && strings.ToLower(d.Output) != "json" {
		d.Output = "pretty"
	}

	if d.NameNs != "" {
		target := getTargetFromNameNs(d.NameNs)
		if len(target.Names) == 1 {
			d.Name = target.Names[0]
		}
		if len(target.Namespaces) == 1 {
			d.Namespace = target.Namespaces[0]
		}
	}

	if !d.ShowJsonTemplate && !isPipedStdIn() && d.Path == "" && d.Id == "" && !d.RandomId {
		err.Details = append(err.Details, "A device id must be provided, or a file path to a device definition.")
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

	if d.ShowJsonTemplate {
		if d.Output == "pretty" {
			d.Output = "yaml"
		}
		if out, e := MarshalResponse(d.Output, mir_models.NewDevice()); e != nil {
			return e
		} else {
			fmt.Println(out)
		}
		return nil
	}

	devs := []*mir_models.Device{}
	if isPipedStdIn() || d.Path != "" {
		devs, err = unmarshalTypeFromStdInOrFile[mir_models.Device](d.Path)
	} else {
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
			d.Id = t.String()[:8]
		}
		if d.Desc != "" {
			d.Anno["mir/device/description"] = d.Desc
		}

		dev := mir_models.NewDevice()
		dev.Meta.Name = d.Name
		dev.Meta.Namespace = d.Namespace
		dev.Meta.Labels = d.Labels
		dev.Meta.Annotations = d.Anno
		dev.Spec.DeviceId = d.Id
		dev.Spec.Disabled = d.Disabled
		devs = append(devs, &dev)
	}

	respDevs := []*core_apiv1.Device{}
	var errs error
	for _, d := range devs {
		resp, err := core_client.PublishDeviceCreateRequest(msgBus, mir_models.NewCreateDeviceReqFromDevice(*d))
		if err != nil {
			e := MirRequestError{Route: "device.create", e: err}
			fmt.Println(e)
			return e
		}
		if resp.GetError() != nil {
			errs = errors.Join(errs, errors.New(resp.GetError().Message))
		} else if resp.GetOk() != nil {
			respDevs = append(respDevs, resp.GetOk())
		}
	}

	if len(respDevs) > 0 {
		list := mir_models.NewDeviceListFromProtoDevices(respDevs)
		if d.Output == "pretty" {
			fmt.Println(prettyStringDevices(list))
		} else {
			if out, e := MarshalResponse(d.Output, list); e != nil {
				return e
			} else {
				fmt.Println(out)
			}
		}
	}
	return errs
}

func (d *DeviceUpdateCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}

	if strings.ToLower(d.Output) != "pretty" && strings.ToLower(d.Output) != "yaml" && strings.ToLower(d.Output) != "json" {
		d.Output = "pretty"
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

func (d *DeviceUpdateCmd) Run(c CLI) error {
	var err error
	msgBus, err := bus.New(c.Target)
	if err != nil {
		e := MirConnectionError{Target: c.Target, e: err}
		return e
	}
	defer msgBus.Close()

	labels := map[string]*common_apiv1.OptString{}
	for k, v := range d.Labels {
		if strings.ToLower(v) == "null" || strings.ToLower(v) == "nil" || strings.ToLower(v) == "none" {
			labels[k] = &common_apiv1.OptString{
				Value: nil,
			}
		} else {
			labels[k] = &common_apiv1.OptString{
				Value: &v,
			}
		}
	}
	anno := map[string]*common_apiv1.OptString{}
	for k, v := range d.Anno {
		if strings.ToLower(v) == "null" || strings.ToLower(v) == "nil" || strings.ToLower(v) == "none" {
			anno[k] = &common_apiv1.OptString{
				Value: nil,
			}
		} else {
			anno[k] = &common_apiv1.OptString{
				Value: &v,
			}
		}
	}
	if d.Desc != nil {
		anno["mir/device/description"] = &common_apiv1.OptString{
			Value: d.Desc,
		}
	}
	req := &core_apiv1.UpdateDeviceRequest{
		Targets: &core_apiv1.Targets{
			Ids:        d.Target.Ids,
			Names:      d.Target.Names,
			Namespaces: d.Target.Namespaces,
			Labels:     d.Target.Labels,
		},
		Meta: &core_apiv1.UpdateDeviceRequest_Meta{
			Labels:      labels,
			Annotations: anno,
		},
		Spec: &core_apiv1.UpdateDeviceRequest_Spec{},
	}

	if d.Name != nil {
		req.Meta.Name = d.Name
	}
	if d.Namespace != nil {
		req.Meta.Namespace = d.Namespace
	}
	if d.Disabled != nil {
		req.Spec.Disabled = d.Disabled
	}
	if d.Id != nil {
		req.Spec.DeviceId = d.Id
	}

	resp, err := core_client.PublishDeviceUpdateRequest(msgBus, req)
	if err != nil {
		e := MirRequestError{Route: "device.update", e: err}
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
		return e
	} else {
		list := mir_models.NewDeviceListFromProtoDevices(resp.GetOk().Devices)
		if d.Output == "pretty" {
			fmt.Println(prettyStringDevices(list))
		} else {
			if out, e := MarshalResponse(d.Output, list); e != nil {
				return e
			} else {
				fmt.Println(out)
			}
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
		d.NameNs == "" {
		err.Details = append(err.Details, "Must specify targets")
	}

	if d.NameNs != "" {
		d.Target = getTargetFromNameNs(d.NameNs)
	}

	if strings.ToLower(d.Output) != "pretty" && strings.ToLower(d.Output) != "yaml" && strings.ToLower(d.Output) != "json" {
		d.Output = "pretty"
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

	resp, err := core_client.PublishDeviceDeleteRequest(msgBus, &core_apiv1.DeleteDeviceRequest{
		Targets: &core_apiv1.Targets{
			Ids:        d.Target.Ids,
			Names:      d.Target.Names,
			Namespaces: d.Target.Namespaces,
			Labels:     d.Target.Labels,
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
		return e
	} else {
		list := mir_models.NewDeviceListFromProtoDevices(resp.GetOk().Devices)
		if d.Output == "pretty" {
			fmt.Println(prettyStringDevices(list))
		} else {
			if out, e := MarshalResponse(d.Output, list); e != nil {
				return e
			} else {
				fmt.Println(out)
			}
		}
	}
	return nil
}

func MarshalResponse(format string, v any) (string, error) {
	var out []byte
	var e error

	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Slice && rv.Len() == 1 {
		v = rv.Index(0).Interface()
	}

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

func prettyStringDevices(devs []*mir_models.Device) string {
	format := "%-45s %-16s %-10s %-20s %-20s %-60s\n"
	timeFormat := "2006-01-02 15:04:05"
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(format, "NAME/NAMESPACE", "DEVICE_ID", "STATUS", "LAST_HEARTHBEAT", "LAST_SCHEMA_FETCH", "LABELS"))

	sort.Slice(devs, func(i, j int) bool {
		return devs[i].Meta.Namespace < devs[j].Meta.Namespace
	})

	for _, d := range devs {
		st := ""
		if d.Spec.Disabled {
			st = "disabled"
		} else if d.Status.Online {
			st = "online"
		} else {
			st = "offline"
		}

		hb := ""
		if !d.Status.LastHearthbeat.IsZero() {
			hb = d.Status.LastHearthbeat.Format(timeFormat)
		}
		sf := ""
		if !d.Status.Schema.LastSchemaFetch.IsZero() {
			sf = d.Status.Schema.LastSchemaFetch.Format(timeFormat)
		}

		sb.WriteString(fmt.Sprintf(format, d.Meta.Name+"/"+d.Meta.Namespace, d.Spec.DeviceId, st, hb, sf, formatLabels(d.Meta.Labels)))
	}
	return sb.String()
}

func formatLabels(m map[string]string) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	pairs := make([]string, 0, len(m))
	for _, k := range keys {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, m[k]))
	}
	return strings.Join(pairs, ",")
}
