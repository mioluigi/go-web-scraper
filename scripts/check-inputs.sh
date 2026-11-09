#!/usr/bin/env bash
# check-inputs.sh — the data gate (SFEDD §5.2, Constitution 8.1) for the saved pages.
#
# Usage: scripts/check-inputs.sh [dir]        default dir: testdata/wikipedia
# Passes when: MANIFEST.tsv lists every .html file and nothing else; every file's SHA-256
# matches the manifest; every file is ≤ 5 MB; the expected count is present (EXPECTED_PAGES,
# default 10); PROBES.txt and robots.txt exist; no stray files (.DS_Store, editor backups).
# Works with macOS bash 3.2.
set -euo pipefail
DIR="${1:-testdata/wikipedia}"
EXPECTED_PAGES="${EXPECTED_PAGES:-10}"
MAX_BYTES=$((5 * 1024 * 1024))
problems=0
problem() { echo "check-inputs: $*" >&2; problems=$((problems + 1)); }

[ -d "$DIR" ] || { echo "check-inputs: $DIR not found — run scripts/capture-pages.sh first." >&2; exit 1; }
manifest="$DIR/MANIFEST.tsv"
[ -f "$manifest" ] || { echo "check-inputs: $manifest not found." >&2; exit 1; }
[ -f "$DIR/PROBES.txt" ] || problem "PROBES.txt missing"
[ -f "$DIR/robots.txt" ] || problem "robots.txt missing"

if command -v sha256sum >/dev/null; then sha256() { sha256sum "$1" | awk '{print $1}'; }
else sha256() { shasum -a 256 "$1" | awk '{print $1}'; }; fi

listed=0
while IFS=$'\t' read -r url status bytes hash revision captured file; do
  [ "$url" = "url" ] && continue
  listed=$((listed + 1))
  path="$DIR/$file"
  if [ ! -f "$path" ]; then problem "$file is in the manifest but missing"; continue; fi
  [ "$status" = "200" ] || problem "$file was captured with HTTP $status"
  [ "$(sha256 "$path")" = "$hash" ] || problem "$file does not match its manifest hash (changed after capture?)"
  [ "$(wc -c < "$path" | tr -d ' ')" -le "$MAX_BYTES" ] || problem "$file is larger than 5 MB"
done < "$manifest"

for path in "$DIR"/*; do
  name="${path##*/}"
  case "$name" in
    MANIFEST.tsv|PROBES.txt|robots.txt) ;;
    *.html) grep -q "	$name\$" "$manifest" || problem "$name is not in the manifest" ;;
    *) problem "unexpected file $name" ;;
  esac
done
for stray in "$DIR"/.[!.]*; do [ -e "$stray" ] && problem "hidden file ${stray##*/}"; done

[ "$listed" -eq "$EXPECTED_PAGES" ] || problem "manifest lists $listed pages, expected $EXPECTED_PAGES"
if [ "$problems" -gt 0 ]; then echo "check-inputs: FAILED ($problems problem(s))" >&2; exit 1; fi
echo "check-inputs: OK — $listed pages, hashes match, sizes ≤ 5 MB, no stray files"
