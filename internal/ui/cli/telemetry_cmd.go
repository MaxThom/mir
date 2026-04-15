package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/maxthom/mir/internal/libs/external/grafana"
	"github.com/maxthom/mir/internal/ui"
	mir_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/mir_api/v1"
	"github.com/maxthom/mir/pkgs/mir_v1"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/rs/zerolog"
)

// TODO find how to move output here instead of per command
// TODO set yaml indent to two spaces
// TODO check if json to remove key value pair should be NONE or NULL. check json doc
type TelemetryCmd struct {
	List  TelemetryListCmd  `cmd:"" aliases:"ls" help:"Explore device telemetry"`
	Query TelemetryQueryCmd `cmd:"" aliases:"qry" help:"Query device telemetry"`
}

type TelemetryListCmd struct {
	Target        `embed:"" prefix:"target."`
	NameNs        string            `name:"name/namespace" arg:"" optional:"" help:"filter on name and/or namespace"`
	Measurements  []string          `short:"m" help:"list of measurements to display. correspond to proto messages of type telemetry"`
	Filters       map[string]string `short:"f" help:"labels to filter measurements"`
	ShowFields    bool              `short:"s" help:"show fields of the measurements" default:"false"`
	PrintQuery    bool              `short:"q" help:"Print Influx query" default:"false"`
	RefreshSchema bool              `short:"r" help:"Refresh schema from device even if in store" default:"false"`
}

type TelemetryQueryCmd struct {
	Output      string `short:"o" help:"output format for response [pretty|json|yaml|csv]" default:"pretty"`
	Target      `embed:"" prefix:"target."`
	NameNs      string        `name:"name/namespace" arg:"" optional:"" help:"filter on name and/or namespace"`
	Measurement string        `short:"m" help:"measurement to display. correspond to proto messages of type telemetry"`
	Fields      []string      `short:"f" help:"specific fields of the measurement to query,"`
	Start       time.Time     `help:"Set starting date to filter data. (eg: 2025-05-01T00:00:00.00Z)"`
	End         time.Time     `help:"Set ending date to filter data. Default to now. (eg: 2025-05-02T00:00:00.00Z)"`
	Since       time.Duration `short:"s" help:"Set duration to filter data. (eg: 5m, 1h, 24h)" default:"5m"`
	Timezone    string        `short:"z" help:"Timezone for Timestamp [local|UTC]" default:"local"`
}

func (d *TelemetryCmd) Run() error {
	return nil
}

func (d *TelemetryListCmd) Validate() error {
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

func (d *TelemetryListCmd) Run(log zerolog.Logger, m *mir.Mir, cfg ui.Config) error {
	ctxCfg, _ := cfg.GetCurrentContext()

	req := &mir_apiv1.ListTelemetryRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids:        d.Target.Ids,
			Names:      d.Target.Names,
			Namespaces: d.Target.Namespaces,
			Labels:     d.Target.Labels,
		},
		Measurements:  d.Measurements,
		Filters:       d.Filters,
		RefreshSchema: d.RefreshSchema,
	}
	resp, err := m.Client().ListTelemetry().Request(req)
	if err != nil {
		return fmt.Errorf("error publishing telemtry list request: %w", err)
	}

	var sb strings.Builder
	i := 0
	tlmsErr := []*mir_apiv1.DevicesTelemetry{}
	for _, tlms := range resp {
		if tlms.Error != "" {
			tlmsErr = append(tlmsErr, tlms)
			continue
		} else {
			i += 1
			sb.WriteString(fmt.Sprintf("%d. ", i))

			devsTitle := []string{}
			for _, devId := range tlms.Ids {
				if devId.Name == "" && devId.Namespace == "" {
					devsTitle = append(devsTitle, devId.DeviceId)
				} else {
					devsTitle = append(devsTitle, devId.Name+"/"+devId.Namespace)
				}
			}
			if len(devsTitle) > 10 {
				sb.WriteString(strings.Join(devsTitle[0:10], ", ") + " & " + fmt.Sprintf("%d more", len(devsTitle)-10))
			} else {
				sb.WriteString(strings.Join(devsTitle, ", "))
			}
			sb.WriteString("\n")

			for _, tlm := range tlms.TlmDescriptors {
				sb.WriteString(tlm.Name)
				sb.WriteString("{")
				sb.WriteString(mapToSortedString(tlm.Labels))
				sb.WriteString("} ")
				sb.WriteString(FormatHyperlink(ctxCfg.Grafana+"/explore", grafana.CreateExploreLink(ctxCfg.Grafana, tlm.ExploreQuery)))
				sb.WriteString("\n")
				if tlm.Error != "" {
					sb.WriteString(tlm.Error)
					sb.WriteString("\n")
				} else if len(tlm.Fields) > 0 && d.ShowFields {
					for _, f := range tlm.Fields {
						sb.WriteString("    ")
						sb.WriteString(f)
						sb.WriteString("\n")
					}
				}
				if d.PrintQuery {
					sb.WriteString("  ")
					sb.WriteString(strings.ReplaceAll(tlm.ExploreQuery, "\n", "\n  "))
					sb.WriteString("\n")
				}
			}
		}
		sb.WriteString("\n")
	}

	for _, tlms := range tlmsErr {
		i += 1
		sb.WriteString(fmt.Sprintf("%d. ", i))

		devsTitle := []string{}
		for _, devId := range tlms.Ids {
			if devId.Name == "" && devId.Namespace == "" {
				devsTitle = append(devsTitle, devId.DeviceId)
			} else {
				devsTitle = append(devsTitle, devId.Name+"/"+devId.Namespace)
			}
		}
		if len(devsTitle) > 10 {
			sb.WriteString(strings.Join(devsTitle[0:10], ", ") + " & " + fmt.Sprintf("%d more", len(devsTitle)-10))
		} else {
			sb.WriteString(strings.Join(devsTitle, ", "))
		}

		sb.WriteString("\n")
		sb.WriteString(tlms.Error)
	}

	fmt.Println(sb.String())
	return nil
}

func (d *TelemetryQueryCmd) Validate() error {
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

	if strings.ToLower(d.Output) != "pretty" && strings.ToLower(d.Output) != "yaml" && strings.ToLower(d.Output) != "json" && strings.ToLower(d.Output) != "csv" {
		d.Output = "pretty"
	}

	d.Timezone = strings.ToLower(d.Timezone)

	if !d.Start.IsZero() && !d.End.IsZero() {
		if d.Start.After(d.End) {
			err.Details = append(err.Details, "start time must be before end time")
		}
	} else if !d.Start.IsZero() {
		if d.Start.After(time.Now()) {
			err.Details = append(err.Details, "start time cannot be in the future")
		}
	} else if d.Start.IsZero() {
		d.Start = time.Now().UTC().Add(-d.Since)
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *TelemetryQueryCmd) Run(log zerolog.Logger, m *mir.Mir, cfg ui.Config) error {
	if d.Measurement == "" {
		// Shortcut for ergonomics
		ls := TelemetryListCmd{
			Target: d.Target,
			NameNs: d.NameNs,
		}
		return ls.Run(log, m, cfg)
	}

	req := &mir_apiv1.QueryTelemetryRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids:        d.Target.Ids,
			Names:      d.Target.Names,
			Namespaces: d.Target.Namespaces,
			Labels:     d.Target.Labels,
		},
		Measurement: d.Measurement,
		Fields:      d.Fields,
		StartTime:   mir_v1.AsProtoTimestamp(d.Start),
		EndTime:     mir_v1.AsProtoTimestamp(d.End),
	}
	resp, err := m.Client().QueryTelemetry().Request(req)
	if err != nil {
		return fmt.Errorf("error publishing telemtry query request: %w", err)
	}

	var out string
	switch d.Output {
	case "json", "yaml":
		out, err = marshalResponse(d.Output, resp)
		if err != nil {
			return fmt.Errorf("error marhsalling telemtry query: %w", err)
		}
		fmt.Println(out)
	case "csv":
		csvStringQuery(resp, d.Timezone)
	case "pretty":
		prettyStringQuery(resp, d.Timezone)
	}

	return nil
}

func prettyStringQuery(query *mir_apiv1.QueryTelemetry, timezone string) {
	format := "%-25s %-15s"
	if len(query.Headers) < 2 {
		// Mean there is no datapoints
		return
	}
	for range query.Headers[2:] {
		format += " %15s"
	}
	headers := make([]any, len(query.Headers))
	for i, h := range query.Headers {
		if query.Units[i] != "" {
			headers[i] = h + " (" + query.Units[i] + ")"
		} else {
			headers[i] = h
		}
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, format, headers...)
	fmt.Println(sb.String())
	sb.Reset()

	dps := make([]any, len(query.Headers))
	for _, row := range query.Rows {
		for i, dp := range row.Datapoints {
			val := datapointToString(query.Datatypes[i], dp, timezone)
			if val == "" {
				val = "-"
			}
			dps[i] = val
		}
		fmt.Fprintf(&sb, format, dps...)
		fmt.Println(sb.String())
		sb.Reset()
	}
}

func csvStringQuery(query *mir_apiv1.QueryTelemetry, timezone string) {
	var sb strings.Builder
	sb.WriteString(strings.Join(query.Headers, ","))
	fmt.Println(sb.String())
	sb.Reset()
	for _, row := range query.Rows {
		for i, dp := range row.Datapoints {
			sb.WriteString(datapointToString(query.Datatypes[i], dp, timezone))
			if i < len(row.Datapoints)-1 {
				sb.WriteString(",")
			}
		}
		fmt.Println(sb.String())
		sb.Reset()
	}
}

// OSC 8 hyperlink escape sequence
// Supported by most new terminal
func FormatHyperlink(text, url string) string {
	reset := "\033[0m"
	blue := "\033[34m"
	underline := "\033[4m"
	text = blue + underline + text + reset
	return fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", url, text)
}

func datapointToString(tp mir_apiv1.DataType, dp *mir_apiv1.QueryTelemetry_Row_DataPoint, timezone string) string {
	switch tp {
	case mir_apiv1.DataType_DATA_TYPE_UNSPECIFIED:
		return ""
	case mir_apiv1.DataType_DATA_TYPE_BOOL:
		if dp.ValueBool == nil {
			return ""
		}
		return fmt.Sprintf("%t", *dp.ValueBool)
	case mir_apiv1.DataType_DATA_TYPE_INT32:
		if dp.ValueInt32 == nil {
			return ""
		}
		return fmt.Sprintf("%d", *dp.ValueInt32)
	case mir_apiv1.DataType_DATA_TYPE_INT64:
		if dp.ValueInt64 == nil {
			return ""
		}
		return fmt.Sprintf("%d", *dp.ValueInt64)
	case mir_apiv1.DataType_DATA_TYPE_SINT32:
		if dp.ValueSint32 == nil {
			return ""
		}
		return fmt.Sprintf("%d", *dp.ValueSint32)
	case mir_apiv1.DataType_DATA_TYPE_SINT64:
		if dp.ValueSint64 == nil {
			return ""
		}
		return fmt.Sprintf("%d", *dp.ValueSint64)
	case mir_apiv1.DataType_DATA_TYPE_UINT32:
		if dp.ValueUint32 == nil {
			return ""
		}
		return fmt.Sprintf("%d", *dp.ValueUint32)
	case mir_apiv1.DataType_DATA_TYPE_UINT64:
		if dp.ValueUint64 == nil {
			return ""
		}
		return fmt.Sprintf("%d", *dp.ValueUint64)
	case mir_apiv1.DataType_DATA_TYPE_FIXED32:
		if dp.ValueFixed32 == nil {
			return ""
		}
		return fmt.Sprintf("%d", *dp.ValueFixed32)
	case mir_apiv1.DataType_DATA_TYPE_FIXED64:
		if dp.ValueFixed64 == nil {
			return ""
		}
		return fmt.Sprintf("%d", *dp.ValueFixed64)
	case mir_apiv1.DataType_DATA_TYPE_SFIXED32:
		if dp.ValueSfixed32 == nil {
			return ""
		}
		return fmt.Sprintf("%d", *dp.ValueSfixed32)
	case mir_apiv1.DataType_DATA_TYPE_SFIXED64:
		if dp.ValueSfixed64 == nil {
			return ""
		}
		return fmt.Sprintf("%d", *dp.ValueSfixed64)
	case mir_apiv1.DataType_DATA_TYPE_FLOAT:
		if dp.ValueFloat == nil {
			return ""
		}
		return fmt.Sprintf("%.4f", *dp.ValueFloat)
	case mir_apiv1.DataType_DATA_TYPE_DOUBLE:
		if dp.ValueDouble == nil {
			return ""
		}
		return fmt.Sprintf("%.4f", *dp.ValueDouble)
	case mir_apiv1.DataType_DATA_TYPE_STRING:
		if dp.ValueString == nil {
			return ""
		}
		return fmt.Sprintf("%s", *dp.ValueString)
	case mir_apiv1.DataType_DATA_TYPE_BYTES:
		if dp.ValueBytes == nil {
			return ""
		}
		return fmt.Sprintf("%s", dp.ValueBytes)
	case mir_apiv1.DataType_DATA_TYPE_TIMESTAMP:
		if dp.ValueTimestamp == nil {
			return ""
		}
		if timezone == "utc" {
			return fmt.Sprintf("%s", mir_v1.AsGoTime(dp.ValueTimestamp).UTC().Format(time.RFC3339))
		} else {
			return fmt.Sprintf("%s", mir_v1.AsGoTime(dp.ValueTimestamp).Local().Format(time.RFC3339))
		}
	}
	return ""
}
