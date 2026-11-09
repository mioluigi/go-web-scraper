#!/usr/bin/env bash
# measure.sh — the official measurements (SFEDD §5.3), on Luis's Mac only.
#
# Refuses to run on a stale or modified copy. Then, with the cache off:
#   1. live crawl of the ten pages, -parallel 1 and -parallel 4, RUNS times each, alternating,
#      PAUSE seconds apart → results/timing.tsv (program time, whole-process time, peak memory)
#   2. the official output: results/wikipedia.jl and results/wikipedia-alltext.jl
#   3. offline benchmarks on the saved pages, with CPU and memory profiles → results/
#   4. results/machine.txt (machine, CPUs, Go version, commit measured)
# Live requests: RUNS × 2 × 10 + 20 (default 120), within the spec's limit of 150.
set -euo pipefail
cd "$(dirname "$0")/.."
RUNS="${RUNS:-5}"
PAUSE="${PAUSE:-3}"
fail() { echo "measure: $*" >&2; exit 1; }

case "${SCRAPER_CONTACT:-}" in *@*.*) ;; *) fail "set SCRAPER_CONTACT to your e-mail address first (Wikipedia requires it)." ;; esac
[ "$(uname -s)" = "Darwin" ] || fail "official measurements run on the Mac (this is $(uname -s))."
[ -z "$(git status --porcelain)" ] || fail "uncommitted changes — commit or stash them, then run again."
git fetch -q origin
[ "$(git rev-parse HEAD)" = "$(git rev-parse '@{u}')" ] || fail "this copy differs from GitHub — run git pull (or push), then run again."
scripts/check-inputs.sh >/dev/null || fail "saved pages fail the data gate (run scripts/check-inputs.sh)."

mkdir -p results
binary="$(mktemp -d)/go-web-scraper"
go build -o "$binary" .

{
  echo "machine: $(sysctl -n hw.model) · $(sysctl -n machdep.cpu.brand_string) · $(( $(sysctl -n hw.memsize) / 1073741824 )) GB · macOS $(sw_vers -productVersion)"
  echo "cpus: $(sysctl -n hw.ncpu)"
  echo "go: $(go version | awk '{print $3}')"
  echo "commit: $(git rev-parse --short HEAD)"
  echo "measured: $(date '+%Y-%m-%d %H:%M %Z')"
} > results/machine.txt

timing=results/timing.tsv
printf 'run\tparallel\tprogram_ms\tprocess_s\tmax_rss_bytes\n' > "$timing"
scratch="$(mktemp -d)"
for run in $(seq 1 "$RUNS"); do
  for parallel in 1 4; do
    sleep "$PAUSE"
    log="$scratch/run-$run-$parallel.log"
    if ! /usr/bin/time -l "$binary" -parallel "$parallel" -out "$scratch/pages.jl" 2> "$log"; then
      tail -n 15 "$log" >&2
      fail "run $run (parallel $parallel) did not save every page; stopping so Wikipedia is not retried."
    fi
    program_ms=$(sed -n 's/.*elapsed_ms=\([0-9]*\).*/\1/p' "$log")
    process_s=$(awk '/ real /{print $1}' "$log")
    max_rss=$(awk '/maximum resident set size/{print $1}' "$log")
    printf '%s\t%s\t%s\t%s\t%s\n' "$run" "$parallel" "$program_ms" "$process_s" "$max_rss" >> "$timing"
    echo "run $run, parallel $parallel: ${program_ms} ms (process ${process_s} s, ${max_rss} bytes)"
  done
done

sleep "$PAUSE"
"$binary" -out results/wikipedia.jl
sleep "$PAUSE"
"$binary" -extract alltext -out results/wikipedia-alltext.jl

go test -run '^$' -bench . -benchmem -benchtime 3x \
  -cpuprofile results/cpu.prof -memprofile results/mem.prof ./crawler/ | tee results/benchmarks.txt
go test -run '^$' -bench . -benchmem ./extract/ | tee -a results/benchmarks.txt
rm -f crawler.test

go run ./cmd/tables
echo
echo "Measurements done: results/ (commit them with the next step)."
