# Welcome Page Design

**Date:** 2026-04-12  
**Branch:** cockpit/welcome  
**Status:** Approved

---

## Context

Cockpit currently lands on the user dashboard page (`/`). The goal is to introduce a dedicated welcome/landing page that greets operators with a pre-built, read-only view of the system — device list, recent events, a get-started guide, and release notes — without any user configuration required.

The welcome page reuses the existing dashboard widget system but is not user-editable. Its layout is defined once by the developer (as a `Dashboard` JSON), stored in SurrealDB under `system/welcome`, and auto-seeded at server startup.

---

## Architecture

### Routing Changes

| Route | Before | After |
|---|---|---|
| `/` | Dashboard page (tabs, edit mode) | Welcome page (read-only, no tabs) |
| `/dashboards` | — | Dashboard page (moved here, unchanged) |

The sidebar nav is updated accordingly: a **Home** entry points to `/`, **Dashboards** points to `/dashboards`.

### Frontend Components

**New files:**
- `src/routes/+page.svelte` — Welcome page (replaces current dashboard page at `/`)
- `src/routes/dashboards/+page.svelte` — Existing dashboard page (moved from `/`)
- `src/lib/domains/welcome/stores/welcome.svelte.ts` — `welcomeStore`

**Modified files:**
- `src/lib/domains/dashboards/components/dashboard-grid.svelte` — Add `readonly?: boolean` prop
- `src/lib/domains/sidebar/data/sidebar-data.ts` — Add Home nav item, update Dashboards link

**Welcome Store (`welcomeStore`):**
```typescript
class WelcomeStore {
  dashboard: Dashboard | null = null
  isLoading: boolean = false
  error: string | null = null

  async load(): Promise<void>  // GET /api/v1/dashboards/system/welcome
  async refresh(): Promise<void>
}
```
Strictly read-only by contract. No create/update/delete methods.

**DashboardGrid `readonly` prop:**
When `readonly=true`:
- GridStack initialized with `staticGrid: true` (no drag, no resize)
- No drag handles rendered in `WidgetWrapper`
- No edit/add-widget/delete controls

**Welcome Page (`/+page.svelte`):**
- Minimal toolbar: page title ("Welcome") + Refresh button only
- No tab bar, no edit button, no add-widget button
- Calls `welcomeStore.load()` on mount
- Passes `dashboard.spec.widgets` to `<DashboardGrid readonly={true} />`

### Backend — Seeder

The `system/welcome` dashboard is seeded at Cockpit server startup.

**Seeder behavior:**
1. On startup, attempt `GET /api/v1/dashboards/system/welcome` (or query SurrealDB directly)
2. If not found → `CREATE` the dashboard from the hardcoded seed JSON
3. If found → do nothing (never overwrite user-tweaked data)

**Seed location:** `internal/servers/cockpit_srv/welcome_seed.go`

```go
var welcomeSeedDashboard = mir_v1.Dashboard{
    // ... hardcoded Dashboard struct matching the API JSON format
    // Copy-pasteable from a real dashboard JSON exported via the UI
}

func seedWelcomeDashboard(store mng.DashboardStore) error {
    target := mir_v1.ObjectTarget{Name: "welcome", Namespace: "system"}
    existing, _ := store.ListDashboards(target)
    if len(existing) > 0 {
        return nil // already seeded
    }
    _, err := store.CreateDashboard(welcomeSeedDashboard)
    return err
}
```

**Developer workflow to update the seed:**
1. Open Cockpit, create a regular dashboard and configure widgets
2. Fetch its JSON: `GET /api/v1/dashboards/{namespace}/{name}`
3. Copy the `spec.widgets` array
4. Paste into `welcomeSeedDashboard` in `welcome_seed.go`

### Predefined Widget Layout (seed)

| Widget | Type | Position | Content |
|---|---|---|---|
| Device List | `device-list` | w=8, h=4 | All devices, shows online/offline |
| Events | `events` | w=4, h=4 | All devices, last 20 events |
| Get Started | `text` | w=4, h=6 | Static markdown with docs links |
| Release Notes | `text` | w=8, h=6 | Static markdown with changelog |

Namespace/name: `system` / `welcome`

---

## Data Flow

```
Page mount
  → welcomeStore.load()
    → GET /api/v1/dashboards/system/welcome
      → Dashboard{} returned
        → DashboardGrid readonly={true}
          → widgets rendered (no controls)

Refresh button
  → welcomeStore.refresh()
    → re-fetch from API
      → widgets re-render
```

---

## What Is Not Changing

- All existing widget components (`widget-telemetry.svelte`, `widget-events.svelte`, etc.) — untouched
- `dashboardStore` and the dashboard API client — untouched
- Existing dashboard page behavior at `/dashboards` — identical to current `/`
- Backend dashboard API endpoints — untouched

---

## Verification

1. **Seeding**: Start server fresh with empty DB → confirm `GET /api/v1/dashboards/system/welcome` returns the seeded dashboard
2. **Welcome page**: Open `/` → confirm widgets render, no edit controls visible, no drag handles
3. **Dashboard page**: Open `/dashboards` → confirm existing behavior unchanged (tabs, edit mode, add widget)
4. **Sidebar**: Confirm Home (`/`) and Dashboards (`/dashboards`) links both work
5. **Refresh**: Click Refresh on welcome page → widgets reload data
6. **Readonly grid**: Attempt to drag a widget on welcome page → confirm nothing moves
