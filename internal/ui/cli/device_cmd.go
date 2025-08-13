package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/maxthom/mir/pkgs/mir_v1"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"
)

// TODO get command which is ls but with -o yaml
type DeviceCmd struct {
	List   DeviceListCmd   `cmd:"" aliases:"ls" help:"List devices"`
	Create DeviceCreateCmd `cmd:"" help:"Create a new device"`
	Update DeviceUpdateCmd `cmd:"" help:"Update a device"`
	Edit   DeviceEditCmd   `cmd:"" help:"Interactive editing of devices"`
	Apply  DeviceApplyCmd  `cmd:"" help:"Update a device using a declarative format"`
	Merge  DeviceMergeCmd  `cmd:"" help:"Update a device using a merge operation"`
	Delete DeviceDeleteCmd `cmd:"" help:"Delete a device"`

	Config    ConfigCmd    `cmd:"" aliases:"cfg" help:"Explore Mir devices configuration"`
	Telemetry TelemetryCmd `cmd:"" aliases:"tlm" help:"Explore Mir devices telemetry"`
	Command   CommandCmd   `cmd:"" aliases:"cmd" help:"Send and explore commands to devices"`
	Schema    SchemaCmd    `cmd:"" aliases:"sch" help:"Upload and explore device proto schema"`
}

type DeviceListCmd struct {
	Output string `short:"o" help:"output format for response [pretty|json|yaml]" default:"pretty"`
	NameNs string `name:"name/namespace" arg:"" optional:"" help:"list single device."`
	Target `embed:"" prefix:"target."`
}

type DeviceCreateCmd struct {
	Output string `short:"o" help:"output format for response [pretty|json|yaml]" default:"yaml"`
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
	Output string `short:"o" help:"output format for response [pretty|json|yaml]" default:"yaml"`
	Target `embed:"" prefix:"target."`
	NameNs string `name:"name/namespace" arg:"" optional:"" help:"shortcut to set name and namespace"`

	Name      *string           `help:"Set device name"`
	Namespace *string           `help:"Set device namespace"`
	Id        *string           `help:"Set device id"`
	Disabled  *bool             `help:"If not enabled, communication is cut"`
	Labels    map[string]string `help:"Set labels to uniquely tag the device (set to null, none or nil to remove)"`
	Anno      map[string]string `help:"Set annotations to add extra information to the devie (set to null to remove)"`
}

type DeviceDeleteCmd struct {
	Output string `short:"o" help:"output format for response [pretty|json|yaml]" default:"yaml"`
	NameNs string `name:"name/namespace" arg:"" optional:"" help:"delete single device."`
	Target `embed:"" prefix:"target."`
}

type Target struct {
	Ids        []string          `help:"List of device to fetch by ids"`
	Names      []string          `help:"List of device to fetch by names"`
	Namespaces []string          `help:"List of device to fetch by namespaces"`
	Labels     map[string]string `help:"Set of labels to filter devices"`
}

func (d *DeviceCmd) Run(log zerolog.Logger, m *mir.Mir) error {
	if err := m.Disconnect(); err != nil {
		log.Error().Err(err).Msg("error disconnecting from Mir server")
	}
	log.Info().Msg("disconnected from Mir server")
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

func (d *DeviceListCmd) Run(log zerolog.Logger, m *mir.Mir) error {
	var err error
	list, err := m.Server().ListDevice().Request(
		mir_v1.DeviceTarget{
			Ids:        d.Ids,
			Names:      d.Names,
			Namespaces: d.Namespaces,
			Labels:     d.Labels,
		}, true)
	if err != nil {
		return fmt.Errorf("error publising list device request: %w", err)
	}
	if d.Output == "pretty" && len(list) == 1 {
		d.Output = "yaml"
	}
	if str, err := stringifyDevices(d.Output, list); err != nil {
		return fmt.Errorf("error marshalling response: %w", err)
	} else {
		fmt.Println(str)
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

func (d *DeviceCreateCmd) Run(log zerolog.Logger, m *mir.Mir) error {
	if d.ShowJsonTemplate {
		if d.Output == "pretty" {
			d.Output = "yaml"
		}
		if out, e := marshalResponse(d.Output, mir_v1.NewDevice()); e != nil {
			return fmt.Errorf("error marshalling response: %w", e)
		} else {
			fmt.Println(out)
		}
		return nil
	}

	devs := []*mir_v1.Device{}
	var err error
	if isPipedStdIn() || d.Path != "" {
		devs, err = unmarshalTypeFromStdInOrFile[mir_v1.Device](d.Path)
		if err != nil {
			return fmt.Errorf("error reading devices from file: %w", err)
		}
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
				return fmt.Errorf("error generation new random uuid: %w", err)
			}
			d.Id = t.String()[:8]
		}
		if d.Desc != "" {
			d.Anno["mir/device/description"] = d.Desc
		}

		dev := mir_v1.NewDevice()
		dev.Meta.Name = d.Name
		dev.Meta.Namespace = d.Namespace
		dev.Meta.Labels = d.Labels
		dev.Meta.Annotations = d.Anno
		dev.Spec.DeviceId = d.Id
		dev.Spec.Disabled = &d.Disabled
		devs = append(devs, &dev)
	}

	list := []mir_v1.Device{}
	var errs error
	for _, d := range devs {
		respD, err := m.Server().CreateDevice().Request(*d)
		if err != nil {
			errs = errors.Join(errs, err)
		} else {
			list = append(list, respD)
		}
	}

	if len(list) > 0 {
		if str, err := stringifyDevices(d.Output, list); err != nil {
			return fmt.Errorf("error marshalling response: %w", err)
		} else {
			fmt.Println(str)
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

func (d *DeviceUpdateCmd) Run(log zerolog.Logger, m *mir.Mir) error {
	var err error
	// TODO check if label modification works
	// labels := map[string]*mir_apiv1.OptString{}
	// for k, v := range d.Labels {
	// 	if strings.ToLower(v) == "null" || strings.ToLower(v) == "nil" || strings.ToLower(v) == "none" {
	// 		labels[k] = &mir_apiv1.OptString{
	// 			Value: nil,
	// 		}
	// 	} else {
	// 		labels[k] = &mir_apiv1.OptString{
	// 			Value: &v,
	// 		}
	// 	}
	// }
	// anno := map[string]*mir_apiv1.OptString{}
	// for k, v := range d.Anno {
	// 	if strings.ToLower(v) == "null" || strings.ToLower(v) == "nil" || strings.ToLower(v) == "none" {
	// 		anno[k] = &mir_apiv1.OptString{
	// 			Value: nil,
	// 		}
	// 	} else {
	// 		anno[k] = &mir_apiv1.OptString{
	// 			Value: &v,
	// 		}
	// 	}
	// }
	target := mir_v1.DeviceTarget{
		Ids:        d.Target.Ids,
		Names:      d.Target.Names,
		Namespaces: d.Target.Namespaces,
		Labels:     d.Target.Labels,
	}
	dev := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Labels:      d.Labels,
		Annotations: d.Anno,
	})

	if d.Name != nil {
		dev.Meta.Name = *d.Name
	}
	if d.Namespace != nil {
		dev.Meta.Namespace = *d.Namespace
	}
	if d.Disabled != nil {
		dev.Spec.Disabled = d.Disabled
	}
	if d.Id != nil {
		dev.Spec.DeviceId = *d.Id
	}

	list, err := m.Server().UpdateDevice().Request(target, dev)
	if err != nil {
		return fmt.Errorf("error publising device update request: %w", err)
	}

	if str, err := stringifyDevices(d.Output, list); err != nil {
		return fmt.Errorf("error marshalling response: %w", err)
	} else {
		fmt.Println(str)
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

func (d *DeviceDeleteCmd) Run(log zerolog.Logger, m *mir.Mir) error {
	list, err := m.Server().DeleteDevice().Request(mir_v1.DeviceTarget{
		Ids:        d.Target.Ids,
		Names:      d.Target.Names,
		Namespaces: d.Target.Namespaces,
		Labels:     d.Target.Labels,
	})
	if err != nil {
		return fmt.Errorf("error publising device delete request: %w", err)
	}

	if str, err := stringifyDevices(d.Output, list); err != nil {
		return fmt.Errorf("error marshalling response: %w", err)
	} else {
		fmt.Println(str)
	}
	return nil
}

func stringifyDevices(output string, devices []mir_v1.Device) (string, error) {
	switch output {
	case "json":
		return marshalResponse(output, devices)
	case "yaml":
		return marshalResponse(output, devices)
	case "pretty":
		return prettyStringDevices(devices), nil
	}
	return "", errors.New("invalid output format")
}

func marshalResponse(format string, v any) (string, error) {
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

func prettyStringDevices(devs []mir_v1.Device) string {
	format := "%-45s %-16s %-10s %-20s %-20s %s\n"
	timeFormat := "2006-01-02 15:04:05"
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(format, "NAMESPACE/NAME", "DEVICE_ID", "STATUS", "LAST_HEARTHBEAT", "LAST_SCHEMA_FETCH", "LABELS"))

	sort.Slice(devs, func(i, j int) bool {
		return devs[i].Meta.Namespace < devs[j].Meta.Namespace
	})

	for _, d := range devs {
		st := ""
		if *d.Spec.Disabled {
			st = "disabled"
		} else if *d.Status.Online {
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

		sb.WriteString(fmt.Sprintf(format, d.Meta.Namespace+"/"+d.Meta.Name, d.Spec.DeviceId, st, hb, sf, formatLabels(d.Meta.Labels)))
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
