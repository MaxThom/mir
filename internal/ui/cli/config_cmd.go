package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/maxthom/mir/internal/libs/editor"
	"github.com/maxthom/mir/internal/ui"
	mir_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/mir_api/v1"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type ConfigCmd struct {
	List ConfigListCmd `cmd:"" aliases:"ls" help:"List all config belonging to a set of devices"`
	Send ConfigSendCmd `cmd:"" help:"Send a config to all targeted devices"`
}

type ConfigListCmd struct {
	Target            `embed:"" prefix:"target."`
	NameNs            string            `name:"name/namespace" arg:"" optional:"" help:"edit single device"`
	FilterLabels      map[string]string `help:"Set of labels to filter config"`
	RefreshSchema     bool              `short:"r" help:"Refresh schema from device even if in store" default:"false"`
	ShowJsonTemplate  bool              `short:"j" help:"show json template for config"`
	ShowCurrentValues bool              `short:"c" help:"show current values for config"`
}

type ConfigSendCmd struct {
	Target            `embed:"" prefix:"target."`
	NameNs            string `name:"name/namespace" arg:"" optional:"" help:"edit single device"`
	Config            string `short:"n" help:"config name to send"`
	ShowJsonTemplate  bool   `short:"j" help:"show json template for config"`
	ShowCurrentValues bool   `short:"c" help:"show current values for config"`
	Payload           string `short:"p" help:"payload to send in json. use single quote for easier writing. e.g. '{\"key\":\"value\"}'"`
	Edit              bool   `short:"e" help:"Interactive edit of config payload" default:"false"`
	RefreshSchema     bool   `short:"r" help:"Refresh schema from device even if in store" default:"false"`
	DryRun            bool   `help:"dry run command" default:"false"`
	ForcePush         bool   `short:"f" help:"force send commands even if some devices are in error" default:"false"`
}

func (d *ConfigListCmd) Validate() error {
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

func (d *ConfigListCmd) Run(log zerolog.Logger, m *mir.Mir, cfg ui.Config) error {
	req := &mir_apiv1.SendListConfigRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids:        d.Target.Ids,
			Names:      d.Target.Names,
			Namespaces: d.Target.Namespaces,
			Labels:     d.Target.Labels,
		},
		FilterLabels:  d.FilterLabels,
		RefreshSchema: d.RefreshSchema,
	}
	resp, err := m.Client().ListConfig().Request(req)
	if err != nil {
		return fmt.Errorf("error publishing list config request: %w", err)
	}

	tpls := map[string][]string{}
	var sb strings.Builder
	for _, cmds := range resp {
		if cmds.Error != "" {
			sb.WriteString(cmds.Error)
		} else {
			for _, cmd := range cmds.CfgDescriptors {
				sb.WriteString(cmd.Name)
				// sb.WriteString("{")
				// sb.WriteString(mapToSortedString(cmd.Labels))
				// sb.WriteString("}\n")
				if cmd.Error != "" {
					sb.WriteString(cmd.Error)
					sb.WriteString("\n")
				} else if d.ShowJsonTemplate || d.ShowCurrentValues {
					js := cmd.Template
					if d.ShowCurrentValues {
						js = cmd.Values
					}
					var prettyJSON bytes.Buffer
					if err = json.Indent(&prettyJSON, []byte(js), "", "  "); err != nil {
						sb.WriteString(err.Error())
					} else {
						sb.WriteString(prettyJSON.String())
					}
					sb.WriteString("\n")
				}
			}
		}
		devsTitle := cmds.DevicesNamens
		if len(cmds.DevicesNamens) > 3 {
			devsTitle = []string{strings.Join(cmds.DevicesNamens[0:3], ", ") + " & " + fmt.Sprintf("%d more", len(cmds.DevicesNamens)-3)}
		}
		tpls[sb.String()] = append(tpls[sb.String()], devsTitle...)
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

func (d *ConfigSendCmd) Validate() error {
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
	if d.Config != "" && d.Payload == "" && !d.ShowJsonTemplate && !d.ShowCurrentValues && !d.Edit {
		err.Details = append(err.Details, "Must set payload. Use -j to see json template, -c for current values or -e for interactive edit.")
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *ConfigSendCmd) Run(log zerolog.Logger, m *mir.Mir, cfg ui.Config) error {
	if d.Config == "" {
		listCfg := ConfigListCmd{
			Target:            d.Target,
			NameNs:            d.NameNs,
			RefreshSchema:     d.RefreshSchema,
			ShowJsonTemplate:  d.ShowJsonTemplate,
			ShowCurrentValues: d.ShowCurrentValues,
		}
		return listCfg.Run(log, m, cfg)
	}

	if d.Edit {
		d.ShowCurrentValues = true
	}

	req := &mir_apiv1.SendConfigRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids:        d.Target.Ids,
			Names:      d.Target.Names,
			Namespaces: d.Target.Namespaces,
			Labels:     d.Target.Labels,
		},
		Name:            d.Config,
		Payload:         []byte(d.Payload),
		PayloadEncoding: mir_apiv1.Encoding_ENCODING_JSON,
		RefreshSchema:   d.RefreshSchema,
		ShowTemplate:    d.ShowJsonTemplate,
		ShowValues:      d.ShowCurrentValues,
		DryRun:          d.DryRun,
		ForcePush:       d.ForcePush,
	}
	resp, err := m.Client().SendConfig().Request(req)
	if err != nil {
		return fmt.Errorf("error publishing send config request: %w", err)
	}

	if d.ShowJsonTemplate || d.ShowCurrentValues {
		tpls := map[string][]string{}
		for k, v := range resp {
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
				return errors.New("Cannot edit multiple json. Refine targets to get single json or use -j to see.")
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
				"Edit the config payload below",
				"On exit, the payload will be sent to the selected targets",
			}
			payload := []byte(sb.String())
			err = editor.EditRawDocument(&payload, header)
			req.ShowTemplate = false
			req.ShowValues = false
			req.Payload = payload
			resp, err = m.Client().SendConfig().Request(req)
			if err != nil {
				return fmt.Errorf("error publishing send config request: %w", err)
			}
		} else {
			fmt.Println(sb.String())
			return nil
		}
	}

	var sb strings.Builder
	i := 1
	for k, v := range resp {
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
