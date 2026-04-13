# Tutorial Drawer — Welcome & Dashboard Pages

**Date:** 2026-04-13
**Branch:** cockpit/welcome

## Overview

The docs drawer (Sheet sliding in from the right, triggered by the help button in the toolbar) is already wired to show context-sensitive content per route. This spec covers:

1. **Welcome page (`/`)** — keep existing `doc-dashboard.svelte` as-is (Web/CLI/Go/TypeScript tabs with install & connect instructions).
2. **Dashboards page (`/dashboards`)** — add a new `doc-dashboards.svelte` (Web-only, no tabs) with a brief overview of how to use dashboards.

## What Changes

### New file: `doc-dashboards.svelte`

A plain content panel (no tab strip). Structure follows the existing doc components:

```
Sheet.Header  →  title: "Dashboards"
Separator
subtitle bar  →  "Build and manage custom dashboards from widgets."
Separator
scrollable content:
  Overview
  Create a dashboard
  Add widgets
  Manage dashboards
```

**Content sections:**

- **Overview** — Dashboards are customizable grids of widgets. Each dashboard belongs to a namespace and can be pinned to the tab bar.
- **Create** — Click `⋯` (top-right) → *New Dashboard* → enter a namespace and name → confirm with ✓.
- **Add widgets** — With the dashboard open, click `⋯` → *Edit Dashboard* (or just *Edit* if already in edit mode) → *Add Widget* → select a widget type → configure it → save with ✓.
- **Manage** — Use the `⊞` icon dropdown to pin or unpin dashboards from the tab bar. Rename or delete a dashboard via `⋯` → *Edit Dashboard*.

### Modified: `docs.ts`

Add `'dashboards'` to the `RouteKey` union type.

### Modified: `docs.svelte.ts`

- Import `DocDashboards` from the new component file.
- Add `dashboards: DocDashboards` to the `docsContent` record.
- Add `if (pathname === '/dashboards') return DocDashboards;` in `getContent()` before the default fallback.

### Untouched

`doc-dashboard.svelte` — welcome page doc stays exactly as-is.

## Design Decisions

- **No tabs for the dashboard doc** — the tab strip only adds value when there are multiple SDK targets. Dashboard creation is a Web UI action; CLI/Go/TS tabs would be empty or redundant.
- **Approach A chosen over B/C** — single-tab strip is visual noise; mixing two topics in one component is poor separation.

## Files

| File | Action |
|------|--------|
| `internal/ui/web/src/lib/domains/docs/components/doc-dashboards.svelte` | Create |
| `internal/ui/web/src/lib/domains/docs/types/docs.ts` | Modify |
| `internal/ui/web/src/lib/domains/docs/stores/docs.svelte.ts` | Modify |
| `internal/ui/web/src/lib/domains/docs/components/doc-dashboard.svelte` | No change |
