package cockpit_srv

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/maxthom/mir/internal/externals/mng"
	"github.com/maxthom/mir/pkgs/mir_v1"
	"github.com/rs/zerolog"
)

//go:embed welcome_seed.json
var welcomeSeedJSON []byte

// SeedWelcomeDashboard creates the system/welcome dashboard if it does not exist.
// The layout is read from welcome_seed.json, which is embedded at compile time.
// Run scripts/export_dashboard_seed.sh to regenerate it from a live dashboard.
// Idempotent — no-ops if the dashboard already exists.
func SeedWelcomeDashboard(log zerolog.Logger, store mng.DashboardStore) error {
	target := mir_v1.ObjectTarget{
		Names:      []string{"welcome"},
		Namespaces: []string{"system"},
	}
	existing, err := store.ListDashboards(target)
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		log.Debug().Msg("welcome dashboard already seeded")
		return nil
	}

	var d mir_v1.Dashboard
	if err := json.Unmarshal(welcomeSeedJSON, &d); err != nil {
		return fmt.Errorf("failed to parse welcome_seed.json: %w", err)
	}
	// Enforce identity — the seed file always targets system/welcome regardless
	// of which source dashboard was used to generate it.
	d.Meta.Name = "welcome"
	d.Meta.Namespace = "system"

	if len(existing) > 0 {
		_, err = store.UpdateDashboard(target, mir_v1.DashboardUpdate{
			Spec: &mir_v1.DashboardUpdateSpec{
				Description: &d.Spec.Description,
				Widgets:     d.Spec.Widgets,
			},
		})
		if err != nil {
			return err
		}
		log.Info().Msg("welcome dashboard updated")
		return nil
	}

	if _, err := store.CreateDashboard(d); err != nil {
		return err
	}
	log.Info().Msg("welcome dashboard seeded")
	return nil
}
