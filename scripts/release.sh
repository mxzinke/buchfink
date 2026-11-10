#!/usr/bin/env bash
# Release von Hand: prüfen, bauen, veröffentlichen. Aufruf über das Taskfile,
# Ablauf in docs/entwicklung/README.md („Release“).
#
#   scripts/release.sh check X.Y.Z        alle Prüfungen, vor dem Tag
#   scripts/release.sh build X.Y.Z        macOS-App, Windows-App, Brücke, Erweiterung, Skill (auf macOS)
#   scripts/release.sh build-linux X.Y.Z  Linux-App (auf Linux amd64 oder in Docker)
#   scripts/release.sh publish X.Y.Z      Prüfsummen und GitHub-Release
#
# Die Dateien liegen unter dist/release/vX.Y.Z/. Ihre Namen enthalten keine
# Fassung, damit Links auf releases/latest/download/<datei> stabil bleiben.
set -euo pipefail

cd "$(dirname "$0")/.."
ROOT=$(pwd)

MCPB_VERSION=2.1.2
TASK_VERSION=v3.40.1
WAILS_VERSION=v3.0.0-beta.9
LINUX_IMAGE=golang:1.25-bookworm

ASSETS=(
  buchfink-darwin-universal.zip
  buchfink-windows-amd64.exe
  buchfink-linux-amd64.tar.gz
  buchfink-mcp-darwin-universal
  buchfink-mcp-windows-amd64.exe
  buchfink-mcp-linux-amd64
  buchfink-mcp-linux-arm64
  buchfink.mcpb
  buchfink.zip
)

die() {
  echo "release: $*" >&2
  exit 1
}

step() { printf '\n==> %s\n' "$*"; }

need_version() {
  [ -n "${VERSION:-}" ] || die "Fassung fehlt, etwa: task release:check VERSION=0.3.1"
  [[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$ ]] || die "ungültige Fassung: $VERSION (angeben mit VERSION=X.Y.Z)"
}

need_clean_tree() {
  [ -z "$(git status --porcelain)" ] || die "ungesicherte oder unversionierte Änderungen; erst committen oder aufräumen"
}

need_tag_at_head() {
  local tag="v$VERSION"
  local tagged
  tagged=$(git rev-parse -q --verify "refs/tags/$tag^{commit}" || true)
  [ "$tagged" = "$(git rev-parse HEAD)" ] ||
    die "HEAD trägt nicht den Tag $tag; erst taggen: git tag -a $tag -m \"Buchfink $VERSION\""
}

out_dir() { echo "$ROOT/dist/release/v$VERSION"; }

cmd_check() {
  need_version
  need_clean_tree
  local tmp
  tmp=$(mktemp -d)
  trap 'rm -rf "$tmp"' EXIT

  step "Fassung in den Paketdateien und Eintrag der Änderungshistorie"
  python3 scripts/version.py check "$VERSION"

  step "go mod tidy ändert nichts"
  go mod tidy
  git diff --exit-code -- go.mod go.sum || die "go mod tidy ändert go.mod oder go.sum"

  step "Frontend: Abhängigkeiten, TypeScript, Build, Tests"
  npm --prefix frontend ci
  (cd frontend && npx tsc --noEmit)
  npm --prefix frontend run build
  npm --prefix frontend test

  step "Brücke ohne CGO"
  CGO_ENABLED=0 go build -o "$tmp/buchfink-mcp" ./cmd/buchfink-mcp

  step "task check (gofmt, vet, Tests, Prüfskripte)"
  task check

  step "Tests der Wails-Bridge"
  task test:bridge

  step "Tests unter extremen Zeitzonen"
  task test:timezones

  step "E-Bilanz gegen die amtlichen Schemata"
  task test:ebilanz-xsd

  step "Skill-Paket"
  python3 -I scripts/package_skills.py --out "$tmp/skills"

  [ -z "$(git status --porcelain)" ] || die "die Prüfungen haben Dateien im Repository geändert: $(git status --porcelain | head -5)"
  printf '\nAlle Prüfungen für %s bestanden. Weiter mit: git tag -a v%s -m "Buchfink %s"\n' "$VERSION" "$VERSION" "$VERSION"
}

cmd_build() {
  need_version
  need_clean_tree
  need_tag_at_head
  [ "$(uname -s)" = Darwin ] || die "release:build läuft auf macOS (App für macOS, lipo); die Linux-App baut release:build:linux"
  local out tmp
  out=$(out_dir)
  tmp=$(mktemp -d)
  trap 'rm -rf "$tmp"' EXIT
  mkdir -p "$out"

  step "Frontend-Abhängigkeiten"
  npm --prefix frontend ci

  step "App für macOS (arm64 und amd64, Ad-hoc-Signatur)"
  task darwin:package:universal VERSION="$VERSION"
  rm -f "$out/buchfink-darwin-universal.zip"
  ditto -c -k --keepParent bin/buchfink.app "$out/buchfink-darwin-universal.zip"

  step "App für Windows (ohne CGO)"
  task windows:build VERSION="$VERSION" ARCH=amd64
  cp bin/buchfink.exe "$out/buchfink-windows-amd64.exe"

  step "Brücke buchfink-mcp"
  build_mcp() { GOOS=$1 GOARCH=$2 task build:mcp VERSION="$VERSION" OUTPUT="$3"; }
  build_mcp darwin arm64 "$tmp/buchfink-mcp-darwin-arm64"
  build_mcp darwin amd64 "$tmp/buchfink-mcp-darwin-amd64"
  lipo -create -output "$out/buchfink-mcp-darwin-universal" \
    "$tmp/buchfink-mcp-darwin-arm64" "$tmp/buchfink-mcp-darwin-amd64"
  build_mcp windows amd64 "$out/buchfink-mcp-windows-amd64.exe"
  build_mcp linux amd64 "$out/buchfink-mcp-linux-amd64"
  build_mcp linux arm64 "$out/buchfink-mcp-linux-arm64"

  step "Erweiterung für Claude Desktop (packaging/claude-desktop/README.md)"
  local pkg="$tmp/claude-desktop"
  cp -R packaging/claude-desktop "$pkg"
  mkdir -p "$pkg/server"
  cp "$out/buchfink-mcp-darwin-universal" "$pkg/server/buchfink-mcp"
  cp "$out/buchfink-mcp-windows-amd64.exe" "$pkg/server/buchfink-mcp.exe"
  cp build/appicon.png "$pkg/icon.png"
  (cd "$pkg" &&
    npx --yes "@anthropic-ai/mcpb@$MCPB_VERSION" validate manifest.json &&
    npx --yes "@anthropic-ai/mcpb@$MCPB_VERSION" pack . "$out/buchfink.mcpb")

  step "Skill-Paket"
  SOURCE_DATE_EPOCH=$(git log -1 --format=%ct) python3 -I scripts/package_skills.py --out "$tmp/skills"
  cp "$tmp/skills/buchfink.zip" "$out/buchfink.zip"

  if [ -n "$(git status --porcelain)" ]; then
    echo "Hinweis: der Build hat versionierte Dateien neu erzeugt (etwa Symbole); prüfen und verwerfen oder committen:" >&2
    git status --porcelain >&2
  fi
  printf '\nGebaut nach %s. Es fehlt noch die Linux-App: task release:build:linux VERSION=%s\n' "$out" "$VERSION"
}

# Baut die Linux-App aus dem Stand des Tags. Linux braucht CGO für GTK 3 und
# WebKit2GTK 4.1; auf einem anderen Rechner läuft der Build in einem
# amd64-Container. Der Container bekommt den Quellstand über git archive und
# schreibt nur die fertige Datei zurück, damit node_modules und bin/ des
# Rechners unberührt bleiben.
cmd_build_linux() {
  need_version
  need_clean_tree
  need_tag_at_head
  local out
  out=$(out_dir)
  mkdir -p "$out"

  if [ "$(uname -s)" = Linux ] && [ "$(uname -m)" = x86_64 ]; then
    step "Linux-App auf diesem Rechner"
    npm --prefix frontend ci
    task linux:build VERSION="$VERSION" EXTRA_TAGS=gtk3
    tar -C bin -czf "$out/buchfink-linux-amd64.tar.gz" buchfink
  else
    command -v docker >/dev/null 2>&1 || die "Linux-App: auf Linux amd64 ausführen oder Docker installieren"
    docker info >/dev/null 2>&1 || die "Docker läuft nicht; Docker Desktop oder OrbStack starten"
    step "Linux-App in $LINUX_IMAGE (linux/amd64)"
    git archive --format=tar "v$VERSION" |
      docker run --rm -i --platform linux/amd64 \
        -v "$out":/out \
        -e VERSION="$VERSION" -e TASK_VERSION="$TASK_VERSION" -e WAILS_VERSION="$WAILS_VERSION" \
        "$LINUX_IMAGE" bash -c '
          set -euo pipefail
          mkdir /build && tar -x -C /build && cd /build
          export DEBIAN_FRONTEND=noninteractive
          apt-get update -qq </dev/null
          apt-get install -y -qq --no-install-recommends libgtk-3-dev libwebkit2gtk-4.1-dev xz-utils </dev/null >/dev/null
          node=$(curl -fsSL https://nodejs.org/dist/latest-v22.x/SHASUMS256.txt | awk "/linux-x64.tar.xz\$/ {print \$2}")
          curl -fsSL "https://nodejs.org/dist/latest-v22.x/$node" | tar -xJ -C /usr/local --strip-components=1
          go install "github.com/go-task/task/v3/cmd/task@$TASK_VERSION"
          go install -tags gtk3 "github.com/wailsapp/wails/v3/cmd/wails3@$WAILS_VERSION"
          npm --prefix frontend ci
          task linux:build VERSION="$VERSION" EXTRA_TAGS=gtk3
          tar -C bin -czf /out/buchfink-linux-amd64.tar.gz buchfink
        '
  fi
  printf '\nGebaut: %s/buchfink-linux-amd64.tar.gz\n' "$out"
}

cmd_publish() {
  need_version
  need_clean_tree
  need_tag_at_head
  local out tmp missing=0
  out=$(out_dir)
  tmp=$(mktemp -d)
  trap 'rm -rf "$tmp"' EXIT

  step "Dateien vollständig"
  for file in "${ASSETS[@]}"; do
    [ -f "$out/$file" ] || { echo "fehlt: $out/$file" >&2; missing=1; }
  done
  # Jede Datei, auf die Dokumentation oder Webseite verweisen, muss im Release liegen.
  for file in $(git grep -hoE 'releases/latest/download/[A-Za-z0-9._-]+' -- '*.md' '*.html' |
    sed -e 's|.*/||' -e 's/\.*$//' | sort -u); do
    [ "$file" = SHA256SUMS ] && continue
    [ -f "$out/$file" ] || { echo "verlinkt, aber nicht gebaut: $file" >&2; missing=1; }
  done
  [ "$missing" = 0 ] || die "Dateien fehlen; erst release:build und release:build:linux"

  step "Prüfsummen"
  (cd "$out" && shasum -a 256 -- "${ASSETS[@]}" >SHA256SUMS && cat SHA256SUMS)

  step "Tag auf GitHub"
  git ls-remote --exit-code --tags origin "refs/tags/v$VERSION" >/dev/null ||
    die "der Tag v$VERSION ist nicht gepusht: git push origin v$VERSION"

  step "GitHub-Release"
  python3 scripts/version.py notes "$VERSION" >"$tmp/notes.md"
  local prerelease=()
  case "$VERSION" in *-*) prerelease=(--prerelease) ;; esac
  gh release create "v$VERSION" "${ASSETS[@]/#/$out/}" "$out/SHA256SUMS" \
    --verify-tag \
    --title "Buchfink $VERSION" \
    --notes-file "$tmp/notes.md" \
    ${prerelease[@]+"${prerelease[@]}"}
}

COMMAND=${1:-}
VERSION=${2:-${VERSION:-}}
case "$COMMAND" in
  check) cmd_check ;;
  build) cmd_build ;;
  build-linux) cmd_build_linux ;;
  publish) cmd_publish ;;
  *) die "Aufruf: scripts/release.sh check|build|build-linux|publish X.Y.Z" ;;
esac
