#!/usr/bin/env bash
# acceptance.sh — the final checks (spec §5), run on a FRESH CLONE before submission (SFEDD phase 10).
#
# Usage: scripts/acceptance.sh        prints one line per check and "N/N checks passed";
#                                     exits 1 if any check fails. Needs no internet: the
#                                     unreachable-page check uses a closed local port.
set -uo pipefail
cd "$(dirname "$0")/.."
passed=0; total=0
check() { # check "description" command...
  total=$((total + 1))
  if output=$("${@:2}" 2>&1); then passed=$((passed + 1)); echo "  ✅ $1"
  else echo "  ❌ $1"; echo "$output" | tail -n 8 | sed 's/^/       /'; fi
}
expect_status() { # expect_status WANT command...
  "${@:2}" >/dev/null 2>&1; local status=$?
  [ "$status" -eq "$1" ] || { echo "exit status $status, want $1"; return 1; }
}
coverage_at_least() { # coverage_at_least MIN package
  local line; line=$(go test -cover "$2" 2>&1 | grep -o 'coverage: [0-9.]*%') || { echo "no coverage for $2"; return 1; }
  awk -v c="${line//[^0-9.]/}" -v min="$1" 'BEGIN { if (c + 0 < min) { print "coverage " c "% < " min "%"; exit 1 } }'
}

# The binary needs a contact address; these checks send no request to Wikipedia.
export SCRAPER_CONTACT="${SCRAPER_CONTACT:-acceptance@example.org}"
scratch=$(mktemp -d)
binary="$scratch/go-web-scraper"
echo "Acceptance checks on $(git rev-parse --short HEAD) ($(go version | awk '{print $3}'))"

echo "Code [C]"
check "gofmt: no file needs formatting" bash -c '[ -z "$(gofmt -l .)" ] || { gofmt -l .; exit 1; }'
check "go vet is clean" go vet ./...
check "go.sum exists (dependencies pinned)" test -s go.sum

echo "Tests [T] (AC-1 to AC-3, AC-6, AC-7)"
check "all tests pass under the race detector" go test -race -count=1 ./...
for package in . ./crawler ./extract ./jsonl ./urllist; do
  check "coverage ≥ 80 % in $package" coverage_at_least 80 "$package"
done
check "saved pages pass the data gate" scripts/check-inputs.sh
check "extraction quality tests ran on the ten saved pages (not skipped)" \
  bash -c 'go test -count=1 -v -run TestWikipediaArticleOnSavedPages ./extract/ | grep -c -- "--- PASS: TestWikipediaArticleOnSavedPages/" | grep -qx 10'

echo "Application [A] (AC-5, on the built binary)"
check "builds" go build -o "$binary" .
check "-h explains usage, exit 0" expect_status 0 "$binary" -h
check "no contact e-mail → exit 2" expect_status 2 env -u SCRAPER_CONTACT "$binary"
check "missing URL file → exit 2" expect_status 2 "$binary" -urls "$scratch/missing.txt"
printf 'www.example.org/no-scheme\n' > "$scratch/bad.txt"
check "malformed URL → exit 2" expect_status 2 "$binary" -urls "$scratch/bad.txt"
check "-parallel 0 → exit 2" expect_status 2 "$binary" -parallel 0
printf 'http://127.0.0.1:9/unreachable\n' > "$scratch/unreachable.txt"
check "unreachable page → exit 1 and an errors file" bash -c \
  "\"$binary\" -urls \"$scratch/unreachable.txt\" -out \"$scratch/u.jl\" >/dev/null 2>&1; [ \$? -eq 1 ] && [ -s \"$scratch/u.errors.jl\" ]"
check "Windows .exe and macOS binaries cross-compile" bash -c 'scripts/build.sh >/dev/null'

echo "Deliverables [D] [Doc] (AC-4, AC-8 to AC-10)"
check "results/wikipedia.jl: 10 valid page records" go run ./cmd/checkjl -want 10 results/wikipedia.jl
check "results/wikipedia-alltext.jl: 10 valid page records" go run ./cmd/checkjl -want 10 results/wikipedia-alltext.jl
check "results/tables.md regenerates from results/ unchanged" bash -c \
  'cp results/tables.md "'"$scratch"'/tables.md" && go run ./cmd/tables >/dev/null && diff -q results/tables.md "'"$scratch"'/tables.md"'
check "README contains the current tables" bash -c \
  'cp README.md "'"$scratch"'/README.md" && scripts/readme.sh >/dev/null && diff -q README.md "'"$scratch"'/README.md"; status=$?; cp "'"$scratch"'/README.md" README.md; exit $status'
check "README has a GenAI Tools section" grep -q '^## GenAI Tools' README.md
check "AI dialogs are indexed in docs/ai-dialogs/README.md" test -s docs/ai-dialogs/README.md

echo
echo "$passed/$total checks passed"
[ "$passed" -eq "$total" ]
