package mcp_srv

import (
	"context"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/maxthom/mir/pkgs/mir_v1"
)

func (s *MCPServer) getDevicesTool() {
	tool := mcp.NewTool("get_devices",
		mcp.WithDescription("Get list of Mir devices"),
		mcp.WithString("name",
			mcp.Description("Name of the device to query"),
		),
		mcp.WithString("namespace",
			mcp.Description("Namespace to query"),
		),
	)

	fn := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name := request.GetString("name", "")
		namespace := request.GetString("namespace", "")

		names := []string{}
		if name != "" {
			names = append(names, name)
		}
		namespaces := []string{}
		if namespace != "" {
			namespaces = append(namespaces, namespace)
		}

		devs, err := s.m.Server().ListDevice().Request(mir_v1.DeviceTarget{
			Names:      names,
			Namespaces: namespaces,
		}, true)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		var sb strings.Builder
		for _, d := range devs {
			sb.WriteString(d.GetNameNamespace())
		}

		return mcp.NewToolResultText(sb.String()), nil
	}

	s.mcp.AddTool(tool, fn)
}
