#!/usr/bin/env bash
# capture-pages.sh — save the target pages once, politely, as test fixtures (spike 001, Part B).
#
# Usage:  scripts/capture-pages.sh [urls-file] [output-dir]
#         defaults: urls.txt  testdata/wikipedia
#
# Fetches one page at a time with a descriptive User-Agent and a pause between requests, so the
# tests can run offline forever after. Writes <output-dir>/MANIFEST.tsv with, per page: URL, HTTP
# status, bytes, SHA-256, Wikipedia revision ID, capture time (UTC). Also saves robots.txt and the
# status Wikipedia returns to Colly's default User-Agent. Stops on the first page that is not HTTP 200.
# Works with macOS bash 3.2.
set -euo pipefail

URLS_FILE="${1:-urls.txt}"
OUT_DIR="${2:-testdata/wikipedia}"
PAUSE_SECONDS="${PAUSE_SECONDS:-1}"
CONTACT_URL="${CONTACT_URL:-https://github.com/mioluigi/go-web-scraper}"
USER_AGENT="go-web-scraper-capture/0.1 (${CONTACT_URL}) curl/$(curl --version | awk 'NR==1{print $2}')"
COLLY_DEFAULT_UA="colly - https://github.com/gocolly/colly"

fail() { echo "capture-pages: $*" >&2; exit 1; }

[ -f "$URLS_FILE" ] || fail "URL list '$URLS_FILE' not found. Run from the repository root, or pass the file."
command -v curl >/dev/null || fail "curl is not installed."

if command -v sha256sum >/dev/null; then sha256() { sha256sum "$1" | awk '{print $1}'; }
else sha256() { shasum -a 256 "$1" | awk '{print $1}'; }; fi

# The file name is the last path segment, with characters that are awkward in file names replaced.
page_file_name() { echo "${1##*/}" | sed 's/[^A-Za-z0-9._-]/_/g'; }

mkdir -p "$OUT_DIR"
manifest="$OUT_DIR/MANIFEST.tsv"
printf 'url\tstatus\tbytes\tsha256\trevision_id\tcaptured_utc\tfile\n' > "$manifest"

count=0
while IFS= read -r url || [ -n "$url" ]; do
  case "$url" in ''|'#'*) continue ;; esac
  file="$OUT_DIR/$(page_file_name "$url").html"
  [ "$count" -gt 0 ] && sleep "$PAUSE_SECONDS"
  status=$(curl -sS -L -A "$USER_AGENT" -o "$file" -w '%{http_code}' "$url") || fail "could not fetch $url"
  [ "$status" = "200" ] || fail "$url answered HTTP $status (saved body in $file for inspection)."
  bytes=$(wc -c < "$file" | tr -d ' ')
  revision=$(grep -o '"wgRevisionId":[0-9]*' "$file" | head -1 | cut -d: -f2 || true)
  printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\n' "$url" "$status" "$bytes" "$(sha256 "$file")" \
    "${revision:-unknown}" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "${file##*/}" >> "$manifest"
  echo "saved ${file##*/} ($bytes bytes, revision ${revision:-unknown})"
  count=$((count + 1))
done < "$URLS_FILE"
[ "$count" -gt 0 ] || fail "no URLs in $URLS_FILE."

# robots.txt and the default-User-Agent probe use the host of the first URL.
first_url=$(grep -v -e '^#' -e '^$' "$URLS_FILE" | head -1)
site=$(echo "$first_url" | sed -E 's#^(https?://[^/]+).*#\1#')
sleep "$PAUSE_SECONDS"
curl -sS -A "$USER_AGENT" -o "$OUT_DIR/robots.txt" "$site/robots.txt" || fail "could not fetch robots.txt"
sleep "$PAUSE_SECONDS"
default_ua_status=$(curl -sS -o /dev/null -A "$COLLY_DEFAULT_UA" -w '%{http_code}' "$first_url" || echo "error")
{
  echo "probe_utc=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "descriptive_user_agent=$USER_AGENT"
  echo "colly_default_user_agent_status=$default_ua_status"
  echo "robots_disallows_all_wiki=$(grep -qx 'Disallow: /wiki/' "$OUT_DIR/robots.txt" && echo yes || echo no)"
} > "$OUT_DIR/PROBES.txt"

echo
echo "Captured $count pages into $OUT_DIR. Manifest: $manifest"
cat "$OUT_DIR/PROBES.txt"
