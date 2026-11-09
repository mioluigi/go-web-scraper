#!/usr/bin/env bash
# readme.sh — copies results/tables.md into README.md between the generated-tables markers,
# so the README's numbers always come from the measured results (Constitution 1.3).
set -euo pipefail
cd "$(dirname "$0")/.."
[ -f results/tables.md ] || { echo "readme: results/tables.md missing — run go run ./cmd/tables" >&2; exit 1; }
grep -q '<!-- tables:start -->' README.md && grep -q '<!-- tables:end -->' README.md \
  || { echo "readme: README.md lacks the tables markers" >&2; exit 1; }
awk -v tables=results/tables.md '
  /<!-- tables:start -->/ { print; while ((getline line < tables) > 0) print line; skipping = 1; next }
  /<!-- tables:end -->/   { skipping = 0 }
  !skipping' README.md > README.md.tmp && mv README.md.tmp README.md
echo "readme: tables inserted"
