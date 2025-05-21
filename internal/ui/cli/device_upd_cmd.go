package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/maxthom/mir/internal/clients/core_client"
	"github.com/maxthom/mir/internal/libs/editor"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	mir_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/mir_api/v1"
	"github.com/maxthom/mir/pkgs/mir_v1"
)

type DeviceEditCmd struct {
	Output string `short:"o" help:"output format for response [pretty|json|yaml]" default:"yaml"`
	NameNs string `name:"name/namespace" arg:"" optional:"" help:"edit single device"`
	Target `embed:"" prefix:"target."`
}

func (d *DeviceEditCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}

	if strings.ToLower(d.Output) != "pretty" && strings.ToLower(d.Output) != "yaml" && strings.ToLower(d.Output) != "json" {
		d.Output = "pretty"
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
		return e
	}
	defer msgBus.Close()

	respList, err := core_client.PublishDeviceListRequest(msgBus, &mir_apiv1.ListDeviceRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids:        d.Ids,
			Names:      d.Names,
			Namespaces: d.Namespaces,
			Labels:     d.Labels,
		},
	})
	if err != nil {
		return fmt.Errorf("error publising device list request: %w", err)
	}
	if respList.GetError() != "" {
		return errors.New(respList.GetError())
	}

	if len(respList.GetOk().Devices) == 0 {
		return errors.New("No devices found for the given targets")
	}

	devs := mir_v1.NewDeviceListFromProtoDevices(respList.GetOk().Devices)
	header := []string{
		"Edit the device below",
		"To remove a field, you must explicitly set it to null",
		"Only fields under meta, spec and properties.desired are editable",
	}

	targetNameNs := []mir_v1.NameNs{}
	for _, d := range devs {
		targetNameNs = append(targetNameNs, d.GetNameNs())
	}
	if d.Output == "json" {
		if len(devs) == 1 {
			err = editor.EditJsonDocument(&devs[0], header)
		} else {
			err = editor.EditJsonDocument(&devs, header)
		}
		if err != nil {
			e := MirEditError{e: err}
			return e
		}
	} else {
		var err error
		if len(devs) == 1 {
			err = editor.EditYamlDocument(&devs[0], header)
		} else {
			err = editor.EditYamlDocument(&devs, header)
		}
		if err != nil {
			e := MirEditError{e: err}
			return e
		}
	}

	var errs error
	respDevs := []*mir_apiv1.Device{}
	for i, d := range devs {
		req := mir_v1.NewUpdateDeviceReqFromDeviceWithNameNs(targetNameNs[i], d)
		resp, err := core_client.PublishDeviceUpdateRequest(msgBus, req)
		if err != nil {
			errs = errors.Join(errs, errors.New("error sending update device request"))
		}
		if resp.GetError() != "" {
			errs = errors.Join(errs, errors.New(resp.GetError()))
		}
		if resp.GetOk() != nil {
			respDevs = append(respDevs, resp.GetOk().Devices...)
		}
	}

	if len(respDevs) > 0 {
		list := mir_v1.NewDeviceListFromProtoDevices(respDevs)
		if str, err := stringifyDevices(d.Output, list); err != nil {
			return fmt.Errorf("error marshalling response: %w", err)
		} else {
			fmt.Println(str)
		}
	}

	if errs != nil {
		return errs
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

	devs := []*mir_apiv1.Device{}
	if isPipedStdIn() || d.Path != "" {
		devs, err = unmarshalTypeFromStdInOrFile[mir_apiv1.Device](d.Path)
		if err != nil {
			return fmt.Errorf("error reading devices from file: %w", err)
		}
	}

	var errs error
	respDevs := []*mir_apiv1.Device{}
	for _, d := range devs {
		req := mir_v1.NewUpdateDeviceReqFromProtoDevice(&mir_apiv1.DeviceTarget{
			Names:      []string{d.GetMeta().GetName()},
			Namespaces: []string{d.GetMeta().GetNamespace()},
		}, d)
		resp, err := core_client.PublishDeviceUpdateRequest(msgBus, req)
		if err != nil {
			errs = errors.Join(errs, errors.New("error sending update device request"))
		}
		if resp.GetError() != "" {
			return errors.New(resp.GetError())
		}
		if resp.GetOk() != nil {
			respDevs = append(respDevs, resp.GetOk().Devices...)
		}
	}

	if len(respDevs) > 0 {
		list := mir_v1.NewDeviceListFromProtoDevices(respDevs)
		if str, err := stringifyDevices("yaml", list); err != nil {
			return fmt.Errorf("error marshalling response: %w", err)
		} else {
			fmt.Println(str)
		}
	}

	if errs != nil {
		return errs
	}

	return nil
}

type DeviceMergeCmd struct {
	Output string `short:"o" help:"output format for response [pretty|json|yaml]" default:"yaml"`
	NameNs string `name:"name/namespace" arg:"" optional:"" help:"edit single device"`
	Target `embed:"" prefix:"target."`
	Patch  string `short:"p" help:"Json patch of device. You can also pipe file content. Tips: use 'mir device list --targets... -o yaml > name.yaml' to get initial content"`
}

func (d *DeviceMergeCmd) Validate() error {
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

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *DeviceMergeCmd) Run(c CLI) error {
	var err error
	msgBus, err := bus.New(c.Target)
	if err != nil {
		e := MirConnectionError{Target: c.Target, e: err}
		return e
	}
	defer msgBus.Close()

	devs := []*mir_v1.Device{}
	if isPipedStdIn() || d.Patch != "" {
		devs, err = unmarshalTypeFromStdInOrString[mir_v1.Device](d.Patch)
		if err != nil {
			return fmt.Errorf("error reading patch from stdin or string: %w", err)
		}
	}

	var errs error
	respDevs := []*mir_apiv1.Device{}
	for _, de := range devs {
		req := mir_v1.NewUpdateDeviceReqFromDeviceWithTarget(mir_v1.DeviceTarget{
			Ids:        d.Ids,
			Names:      d.Names,
			Namespaces: d.Namespaces,
			Labels:     d.Labels,
		}, *de)
		resp, err := core_client.PublishDeviceUpdateRequest(msgBus, req)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("error publishing device update request: %w", err))
		}
		if resp.GetError() != "" {
			errs = errors.Join(errs, errors.New(resp.GetError()))
		}
		if resp.GetOk() != nil {
			respDevs = append(respDevs, resp.GetOk().Devices...)
		}
	}

	if len(respDevs) > 0 {
		list := mir_v1.NewDeviceListFromProtoDevices(respDevs)
		if str, err := stringifyDevices("yaml", list); err != nil {
			return fmt.Errorf("error marshalling response: %w", err)
		} else {
			fmt.Println(str)
		}
	}

	if errs != nil {
		return errs
	}

	return nil
}
