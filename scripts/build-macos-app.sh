#!/usr/bin/env bash
# build-macos-app.sh — dist/WebScraper.app, for the rubric's .app requirement (adapted from Week 6).
#
# The program is a command-line tool and a double-clicked app has no terminal, so the bundle is
# a thin wrapper: its launcher opens Terminal and runs the program on the ten built-in Wikipedia
# pages (asking for the contact e-mail Wikipedia requires), writing ~/Desktop/WebScraper-output/wikipedia.jl, and waits for Return so the summary
# stays on screen. The program itself is unchanged.
set -euo pipefail
cd "$(dirname "$0")/.."
app=dist/WebScraper.app
rm -rf "$app"
mkdir -p "$app/Contents/MacOS" "$app/Contents/Resources"
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -o "$app/Contents/Resources/go-web-scraper" .

cat >"$app/Contents/Resources/run-in-terminal.command" <<'SCRIPT'
#!/bin/bash
clear
program="$(dirname "$0")/go-web-scraper"
output="$HOME/Desktop/WebScraper-output/wikipedia.jl"
if [ -z "${SCRAPER_CONTACT:-}" ]; then
  echo "Wikipedia asks automated tools for a contact e-mail address (sent only in the request header)."
  read -r -p "Your e-mail address: " SCRAPER_CONTACT
  export SCRAPER_CONTACT
  echo
fi
echo "Fetching ten Wikipedia articles on intelligent systems and robotics, four at a time,"
echo "and saving their text as JSON Lines."
echo
if "$program" -out "$output"; then
  echo
  echo "Done. The file is on your Desktop: WebScraper-output/wikipedia.jl"
else
  echo
  echo "Some pages could not be fetched; see the messages above (check the internet connection)."
fi
echo
read -r -p "Press Return to close this window. " _
SCRIPT

cat >"$app/Contents/MacOS/WebScraper" <<'SCRIPT'
#!/bin/bash
open -a Terminal "$(dirname "$0")/../Resources/run-in-terminal.command"
SCRIPT
chmod +x "$app/Contents/MacOS/WebScraper" "$app/Contents/Resources/run-in-terminal.command"

cat >"$app/Contents/Info.plist" <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleName</key>               <string>WebScraper</string>
  <key>CFBundleDisplayName</key>        <string>Web Scraper</string>
  <key>CFBundleIdentifier</key>         <string>com.github.mioluigi.webscraper</string>
  <key>CFBundleExecutable</key>         <string>WebScraper</string>
  <key>CFBundlePackageType</key>        <string>APPL</string>
  <key>CFBundleShortVersionString</key> <string>1.0</string>
  <key>CFBundleVersion</key>            <string>1</string>
  <key>LSMinimumSystemVersion</key>     <string>13.0</string>
  <key>LSArchitecturePriority</key>     <array><string>arm64</string></array>
</dict>
</plist>
PLIST
echo "Built $app — double-click it in Finder, or run: open $app"
