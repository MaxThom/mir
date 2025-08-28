package mcp_srv

import (
	"context"
	"encoding/json"

	"github.com/maxthom/mir/pkgs/mir_v1"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GetDevicesTool struct {
	m *mir.Mir
	t mcp.Tool
}
type GetDevicesParams struct {
	Name      string `json:"name,omitempty" jsonschema:"Name of the device to query"`
	Namespace string `json:"namespace,omitempty" jsonschema:"Namespace to query"`
}

func NewGetDevicesTool(m *mir.Mir) GetDevicesTool {
	return GetDevicesTool{
		m: m,
		t: mcp.Tool{
			Title:       "Get Devices",
			Name:        "get_devices",
			Description: "Get list of Mir devices",
		},
	}
}

func (t GetDevicesTool) RegisterTool(mcpSrv *mcp.Server) {
	mcp.AddTool(mcpSrv, &t.t, t.Handler)
}

func (t GetDevicesTool) Handler(ctx context.Context, request *mcp.CallToolRequest, args GetDevicesParams) (*mcp.CallToolResult, any, error) {
	names := []string{}
	if args.Name != "" {
		names = append(names, args.Name)
	}
	namespaces := []string{}
	if args.Namespace != "" {
		namespaces = append(namespaces, args.Namespace)
	}

	devs, err := t.m.Server().ListDevice().Request(mir_v1.DeviceTarget{
		Names:      names,
		Namespaces: namespaces,
	}, true)
	if err != nil {
		return nil, nil, err
	}

	for i, d := range devs {
		d.Status.Schema.CompressedSchema = nil
		devs[i] = d
	}
	j, err := json.MarshalIndent(devs, "", "  ")
	if err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(j)}},
	}, nil, nil
}
