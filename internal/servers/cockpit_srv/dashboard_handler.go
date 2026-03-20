package cockpit_srv

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/maxthom/mir/internal/externals/mng"
	"github.com/maxthom/mir/pkgs/mir_v1"
)

// Routes:
//   GET  /api/v1/dashboards
//   POST /api/v1/dashboards                          body: Dashboard (apiVersion, kind, meta, spec)
//   GET  /api/v1/dashboards/{namespace}/{name}
//   PUT  /api/v1/dashboards/{namespace}/{name}       body: { spec: { description } }
//   DEL  /api/v1/dashboards/{namespace}/{name}
//   PUT  /api/v1/dashboards/{namespace}/{name}/widgets  body: { widgets: [...] }

func (s *CockpitServer) dashboardsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		dashboards, err := s.store.ListDashboards()
		if err != nil {
			s.log.Error().Err(err).Msg("failed to list dashboards")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		writeJSON(w, dashboards)
	case http.MethodPost:
		var d mir_v1.Dashboard
		if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		if d.Meta.Name == "" {
			http.Error(w, "meta.name is required", http.StatusBadRequest)
			return
		}
		created, err := s.store.CreateDashboard(d)
		if err != nil {
			s.log.Error().Err(err).Msg("failed to create dashboard")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, created)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// dashboardHandler handles all /api/v1/dashboards/{namespace}/{name}[/widgets] routes.
func (s *CockpitServer) dashboardHandler(w http.ResponseWriter, r *http.Request) {
	namespace, name, sub := parseDashboardPath(r.URL.Path)
	if namespace == "" || name == "" {
		http.Error(w, "path must be /{namespace}/{name}", http.StatusBadRequest)
		return
	}

	switch sub {
	case "":
		switch r.Method {
		case http.MethodGet:
			d, err := s.store.GetDashboard(name, namespace)
			if err != nil {
				dashboardError(w, err)
				return
			}
			writeJSON(w, d)
		case http.MethodPut:
			var req struct {
				Spec struct {
					Description string `json:"description"`
				} `json:"spec"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}
			d, err := s.store.UpdateDashboard(name, namespace, req.Spec.Description)
			if err != nil {
				dashboardError(w, err)
				return
			}
			writeJSON(w, d)
		case http.MethodDelete:
			d, err := s.store.DeleteDashboard(name, namespace)
			if err != nil {
				dashboardError(w, err)
				return
			}
			writeJSON(w, d)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}

	case "widgets":
		if r.Method != http.MethodPut {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Widgets []mir_v1.DashboardWidget `json:"widgets"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		d, err := s.store.SaveWidgets(name, namespace, req.Widgets)
		if err != nil {
			dashboardError(w, err)
			return
		}
		writeJSON(w, d)

	default:
		http.NotFound(w, r)
	}
}

func dashboardError(w http.ResponseWriter, err error) {
	if errors.Is(err, mng.ErrorDashboardNotFound) {
		http.Error(w, "Dashboard not found", http.StatusNotFound)
	} else {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// parseDashboardPath splits /api/v1/dashboards/{namespace}/{name}[/{sub}]
// into (namespace, name, sub). sub is empty for the base resource path.
func parseDashboardPath(path string) (namespace, name, sub string) {
	rest := strings.TrimPrefix(path, "/api/v1/dashboards/")
	parts := strings.SplitN(rest, "/", 3)
	switch len(parts) {
	case 2:
		return parts[0], parts[1], ""
	case 3:
		return parts[0], parts[1], parts[2]
	}
	return "", "", ""
}
