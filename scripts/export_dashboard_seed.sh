#!/bin/bash
# Snapshot a Cockpit dashboard as the welcome page seed.
#
# Usage: ./scripts/export_dashboard_seed.sh <namespace> <name>
#
# Fetches the named dashboard from SurrealDB, strips internal/transient
# fields, forces meta to system/welcome, and writes
# internal/servers/cockpit_srv/welcome_seed.json.
#
# After running this script, rebuild the server — the new JSON is embedded
# at compile time and pushed to the DB on next startup if the welcome
# dashboard does not yet exist.
#
# Connection defaults match infra/compose/surrealdb and serve_cmd.go.
# Override with env vars:
#   SURREAL_ENDPOINT, SURREAL_USER, SURREAL_PASS, SURREAL_NS, SURREAL_DB

set -euo pipefail

SURREAL_ENDPOINT="${SURREAL_ENDPOINT:-ws://localhost:8000}"
SURREAL_USER="${SURREAL_USER:-root}"
SURREAL_PASS="${SURREAL_PASS:-root}"
SURREAL_NS="${SURREAL_NS:-global}"
SURREAL_DB="${SURREAL_DB:-mir}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUT="$SCRIPT_DIR/../internal/servers/cockpit_srv/welcome_seed.json"

if [[ $# -ne 2 ]]; then
    echo "Usage: $0 <namespace> <name>" >&2
    exit 1
fi

NS="$1"
NAME="$2"

echo "Fetching '$NS/$NAME' from SurrealDB..."

RAW=$(echo "SELECT * OMIT id FROM dashboards WHERE meta.name = '$NAME' AND meta.namespace = '$NS' LIMIT 1;" \
    | surreal sql \
        --endpoint "$SURREAL_ENDPOINT" \
        --username "$SURREAL_USER" \
        --password "$SURREAL_PASS" \
        --namespace "$SURREAL_NS" \
        --database "$SURREAL_DB" \
        --json 2>/dev/null)

RECORD=$(echo "$RAW" | jq -e '.[0][0] // empty' 2>/dev/null) || {
    echo "Dashboard '$NS/$NAME' not found or query failed." >&2
    exit 1
}

# Write seed JSON: strip status (regenerated on create), force system/welcome identity
echo "$RECORD" \
    | jq 'del(.status) | .meta.name = "welcome" | .meta.namespace = "system"' \
    > "$OUT"

echo "Saved → $OUT"
echo "Rebuild the server to pick up the new seed."
