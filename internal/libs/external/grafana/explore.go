package grafana

import (
	"encoding/json"
	"net/url"
)

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
