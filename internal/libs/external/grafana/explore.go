package grafana

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
)

func OpenBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		return fmt.Errorf("unsupported platform")
	}

	return cmd.Start()
}

func CreateExploreLink(host string, query string) string {
	paneConfig := map[string]interface{}{
		"xyz": map[string]interface{}{
			"datasource": "mir-influxdb",
			"queries": []map[string]interface{}{
				{
					"refId": "A",
					"datasource": map[string]interface{}{
						"type": "influxdb",
						"uid":  "mir-influxdb",
					},
					"query": query,
				},
			},
			"range": map[string]string{
				"from": "now-1h",
				"to":   "now",
			},
		},
	}
	panes, _ := json.Marshal(paneConfig)

	params := url.Values{}
	params.Add("schemaVersion", "1")
	params.Add("orgId", "1")
	params.Add("panes", string(panes))
	u := url.URL{
		Scheme:   "http",
		Host:     host,
		Path:     "explore",
		RawQuery: params.Encode(),
	}

	return u.String()
}
