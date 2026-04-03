#!/usr/bin/env bash

set -euo pipefail

total=0
total_all=0

echo "=== Go Code ==="
go_all=$(find . -name '*.go' | xargs -I {} cat {} 2>/dev/null | wc -l)
echo "All Go files: $go_all"

go_excl=$(find . -name '*.go' ! -name '*.pb.go' | xargs -I {} cat {} 2>/dev/null | wc -l)
echo "Go files (excluding generated): $go_excl"

echo ""
echo "=== Svelte App (Cockpit) ==="

svelte=$(find ./internal/ui/web/src -name '*.svelte' 2>/dev/null | xargs -I {} cat {} 2>/dev/null | wc -l || echo 0)
echo "Svelte: $svelte"

ts=$(find ./internal/ui/web/src -name '*.ts' -o -name '*.js' 2>/dev/null | grep -v node_modules | xargs -I {} cat {} 2>/dev/null | wc -l || echo 0)
echo "TypeScript/JavaScript: $ts"

css=$(find ./internal/ui/web/src -name '*.css' 2>/dev/null | xargs -I {} cat {} 2>/dev/null | wc -l || echo 0)
echo "CSS: $css"

html=$(find ./internal/ui/web/src -name '*.html' 2>/dev/null | xargs -I {} cat {} 2>/dev/null | wc -l || echo 0)
echo "HTML: $html"

total=$((go_excl + svelte + ts + css + html))
total_all=$((go_all + svelte + ts + css + html))

echo ""
echo "=== Total ==="
echo "Sum of all lines (excl. generated): $total"
echo "Sum of all lines: $total_all"
