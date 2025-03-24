{{ define "email.default.subject" }}[{{ .Status | toUpper }}{{ if eq .Status "firing" }}:{{ .Alerts.Firing | len }}{{ end }}] {{ .GroupLabels.SortedPairs.Values | join " " }} {{ if gt (len .CommonLabels) (len .GroupLabels) }}({{ with .CommonLabels.Remove .GroupLabels.Names }}{{ .Values | join " " }}{{ end }}){{ end }}{{ end }}

{{ define "email.default.html" }}
<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <title>Prometheus Alert</title>
  <style type="text/css">
    body {
      font-family: Arial, sans-serif;
      font-size: 14px;
      line-height: 1.5;
      color: #333;
    }
    table {
      border-collapse: collapse;
      width: 100%;
      margin: 20px 0;
    }
    th, td {
      padding: 8px;
      text-align: left;
      border: 1px solid #ddd;
    }
    th {
      background-color: #f2f2f2;
      font-weight: bold;
    }
    .alert-critical {
      background-color: #f8d7da;
    }
    .alert-warning {
      background-color: #fff3cd;
    }
    .alert-info {
      background-color: #d1ecf1;
    }
    .alert-resolved {
      background-color: #d4edda;
    }
    h3 {
      margin-top: 20px;
    }
  </style>
</head>
<body>
  <h2>
    {{ if eq .Status "firing" }}
    [ALERT] {{ .GroupLabels.SortedPairs.Values | join " " }}
    {{ else }}
    [RESOLVED] {{ .GroupLabels.SortedPairs.Values | join " " }}
    {{ end }}
  </h2>

  <h3>Summary</h3>
  <ul>
    <li><strong>Number of firing alerts:</strong> {{ .Alerts.Firing | len }}</li>
    <li><strong>Number of resolved alerts:</strong> {{ .Alerts.Resolved | len }}</li>
  </ul>

  {{ if gt (len .Alerts.Firing) 0 }}
  <h3>Firing Alerts</h3>
  <table>
    <tr>
      <th>Alert</th>
      <th>Instance</th>
      <th>Summary</th>
      <th>Started At</th>
    </tr>
    {{ range .Alerts.Firing }}
    <tr class="alert-{{ .Labels.severity | toLower }}">
      <td>{{ .Labels.alertname }}</td>
      <td>{{ .Labels.instance }}</td>
      <td>{{ .Annotations.summary }}</td>
      <td>{{ .StartsAt }}</td>
    </tr>
    {{ end }}
  </table>
  {{ end }}

  {{ if gt (len .Alerts.Resolved) 0 }}
  <h3>Resolved Alerts</h3>
  <table>
    <tr>
      <th>Alert</th>
      <th>Instance</th>
      <th>Summary</th>
      <th>Resolved At</th>
    </tr>
    {{ range .Alerts.Resolved }}
    <tr class="alert-resolved">
      <td>{{ .Labels.alertname }}</td>
      <td>{{ .Labels.instance }}</td>
      <td>{{ .Annotations.summary }}</td>
      <td>{{ .EndsAt }}</td>
    </tr>
    {{ end }}
  </table>
  {{ end }}

  <p>
    <a href="http://localhost:9093">View in Alertmanager</a>
  </p>
</body>
</html>
{{ end }}
