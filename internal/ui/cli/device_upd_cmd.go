package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/maxthom/mir/internal/libs/editor"
	"github.com/maxthom/mir/internal/ui"
	"github.com/maxthom/mir/pkgs/mir_v1"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/rs/zerolog"
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

func (d *DeviceEditCmd) Run(log zerolog.Logger, m *mir.Mir, cfg ui.Config) error {
	tar := mir_v1.DeviceTarget{
		Ids:        d.Ids,
		Names:      d.Names,
		Namespaces: d.Namespaces,
		Labels:     d.Labels,
	}
	devs, err := m.Server().ListDevice().Request(tar, false)
	if err != nil {
		return fmt.Errorf("error publising device list request: %w", err)
	}

	header := []string{
		"Edit the device below",
		"To remove a field, you must explicitly set it to null",
		"Only fields under meta, spec and properties.desired are editable",
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
	list := []mir_v1.Device{}
	for _, d := range devs {
		list, err = m.Server().UpdateDevice().Request(tar, d)
		if err != nil {
			errs = errors.Join(errs, errors.New("error sending update device request"))
		}
	}
	if len(list) > 0 {
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

	if d.Path == "" && !isPipedStdIn() {
		err.Details = append(err.Details, "No device definition provided via pipe or file")
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *DeviceApplyCmd) Run(log zerolog.Logger, m *mir.Mir, cfg ui.Config) error {
	devs := []*mir_v1.Device{}
	var err error
	if isPipedStdIn() || d.Path != "" {
		devs, err = unmarshalTypeFromStdInOrFile[mir_v1.Device](d.Path)
		if err != nil {
			return fmt.Errorf("error reading devices from file: %w", err)
		}
	}

	var errs error
	list := []mir_v1.Device{}
	for _, d := range devs {
		resp, err := m.Server().UpdateDevice().RequestSingle(*d)
		if err != nil {
			errs = errors.Join(errs, errors.New("error sending update device request"))
			continue
		}
		list = append(list, resp...)
	}

	if len(list) > 0 {
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

	if d.Patch == "" && !isPipedStdIn() {
		err.Details = append(err.Details, "No device patch provided via pipe or file")
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *DeviceMergeCmd) Run(log zerolog.Logger, m *mir.Mir, cfg ui.Config) error {
	devs := []*mir_v1.Device{}
	var err error
	if isPipedStdIn() || d.Patch != "" {
		devs, err = unmarshalTypeFromStdInOrString[mir_v1.Device](d.Patch)
		if err != nil {
			return fmt.Errorf("error reading patch from stdin or string: %w", err)
		}
	}

	var errs error
	list := []mir_v1.Device{}
	for _, dev := range devs {
		resp, err := m.Server().UpdateDevice().Request(
			mir_v1.DeviceTarget{
				Ids:        d.Ids,
				Names:      d.Names,
				Namespaces: d.Namespaces,
				Labels:     d.Labels,
			}, *dev)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("error publishing device update request: %w", err))
			continue
		}
		list = append(list, resp...)
	}

	if len(list) > 0 {
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
