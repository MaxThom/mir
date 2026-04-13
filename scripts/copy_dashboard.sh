#!/bin/bash
# Copy a Cockpit dashboard to a new namespace/name.
#
# Usage:
#   ./scripts/copy_dashboard.sh <src-namespace> <src-name> <dst-namespace> <dst-name>
#
# Connection defaults match infra/compose/surrealdb and serve_cmd.go defaults.
# Override with environment variables:
#   SURREAL_ENDPOINT  (default: ws://localhost:8000)
#   SURREAL_USER      (default: root)
#   SURREAL_PASS      (default: root)
#   SURREAL_NS        (default: global)
#   SURREAL_DB        (default: mir)

set -euo pipefail

SURREAL_ENDPOINT="${SURREAL_ENDPOINT:-ws://localhost:8000}"
SURREAL_USER="${SURREAL_USER:-root}"
SURREAL_PASS="${SURREAL_PASS:-root}"
SURREAL_NS="${SURREAL_NS:-global}"
SURREAL_DB="${SURREAL_DB:-mir}"

if [[ $# -ne 4 ]]; then
    echo "Usage: $0 <src-namespace> <src-name> <dst-namespace> <dst-name>" >&2
    exit 1
fi

SRC_NS="$1"
SRC_NAME="$2"
DST_NS="$3"
DST_NAME="$4"

echo "Copying dashboard '$SRC_NS/$SRC_NAME' → '$DST_NS/$DST_NAME' ..."

surreal sql \
    --endpoint "$SURREAL_ENDPOINT" \
    --username "$SURREAL_USER" \
    --password "$SURREAL_PASS" \
    --namespace "$SURREAL_NS" \
    --database "$SURREAL_DB" \
    2>/dev/null \
    << EOF
LET \$src = (SELECT * FROM dashboards WHERE meta.name = '$SRC_NAME' AND meta.namespace = '$SRC_NS' LIMIT 1)[0];
IF \$src = NONE THEN
    THROW "Dashboard '$SRC_NS/$SRC_NAME' not found";
END;
CREATE dashboards CONTENT {
    apiVersion: \$src.apiVersion,
    kind: \$src.kind,
    meta: {
        name: '$DST_NAME',
        namespace: '$DST_NS'
    },
    spec: \$src.spec,
    status: {
        createdAt: time::now(),
        updatedAt: time::now()
    }
};
EOF
