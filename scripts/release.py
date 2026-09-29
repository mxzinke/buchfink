#!/usr/bin/env python3
"""Veröffentlicht eine Fassung von Buchfink.

Ablauf und Voraussetzungen stehen in docs/entwicklung/veroeffentlichung.md.

    python3 scripts/release.py prepare 0.3.0   # Fassung setzen, prüfen, Commit und Tag
    python3 scripts/release.py publish         # Push und GitHub-Release aus dem Changelog
    python3 scripts/release.py package         # Pakete dieses Betriebssystems nach dist/
    python3 scripts/release.py upload          # dist/ an das Release hängen
    python3 scripts/release.py notes [v0.3]    # Text der Veröffentlichung ausgeben

Die Fassung „0.3.0" heißt als Tag und im Changelog „v0.3", „0.3.1" heißt
„v0.3.1". Der Changelog-Eintrag wird vor `prepare` von Hand geschrieben; sein
Text ist zugleich der Text der Veröffentlichung.
"""

from __future__ import annotations

import argparse
import datetime
import glob
import os
import platform
import re
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
CHANGELOG = ROOT / "internal/changelog/CHANGELOG.md"
DIST = ROOT / "dist"
MAIN_BRANCH = "master"
APP_NAME = "buchfink"

# Jede Datei, die die Programmfassung trägt, mit dem Muster ihrer Stelle.
# Gruppe 1 ist der Text vor der Fassung, Gruppe 2 der danach. Windows verlangt
# im Manifest und im MSIX-Paket vier Stellen.
VERSION_SITES: list[tuple[str, str, str]] = [
    ("build/config.yml", r'(\n  version: ")[^"]*(")', "semver"),
    ("build/darwin/Info.plist", r"(<key>CFBundleVersion</key>\s*<string>)[^<]*(</string>)", "semver"),
    ("build/darwin/Info.plist", r"(<key>CFBundleShortVersionString</key>\s*<string>)[^<]*(</string>)", "semver"),
    ("build/darwin/Info.dev.plist", r"(<key>CFBundleVersion</key>\s*<string>)[^<]*(</string>)", "semver"),
    ("build/darwin/Info.dev.plist", r"(<key>CFBundleShortVersionString</key>\s*<string>)[^<]*(</string>)", "semver"),
    ("build/windows/info.json", r'("file_version": ")[^"]*(")', "semver"),
    ("build/windows/info.json", r'("ProductVersion": ")[^"]*(")', "semver"),
    ("build/windows/wails.exe.manifest", r'(name="com\.buchfink\.app" version=")[^"]*(")', "quad"),
    ("build/windows/nsis/wails_tools.nsh", r'(!define INFO_PRODUCTVERSION ")[^"]*(")', "semver"),
    ("build/windows/msix/app_manifest.xml", r'(\n    Version=")[^"]*(")', "quad"),
    ("build/windows/msix/template.xml", r'(\n        Version=")[^"]*(")', "quad"),
    ("build/linux/nfpm/nfpm.yaml", r'(\nversion: ")[^"]*(")', "semver"),
]


class ReleaseError(Exception):
    pass


# --- Fassung -----------------------------------------------------------------


def parse_version(text: str) -> tuple[int, int, int]:
    match = re.fullmatch(r"v?(\d+)\.(\d+)(?:\.(\d+))?", text.strip())
    if not match:
        raise ReleaseError(f"„{text}\" ist keine Fassung der Form 0.3.0")
    return int(match[1]), int(match[2]), int(match[3] or 0)


def tag_for(version: tuple[int, int, int]) -> str:
    major, minor, patch = version
    return f"v{major}.{minor}" if patch == 0 else f"v{major}.{minor}.{patch}"


def semver(version: tuple[int, int, int]) -> str:
    return "{}.{}.{}".format(*version)


def current_version() -> str:
    text = (ROOT / "build/config.yml").read_text(encoding="utf-8")
    match = re.search(r'\n  version: "([^"]*)"', text)
    if not match:
        raise ReleaseError("build/config.yml nennt keine Fassung")
    return match[1]


def set_version(version: tuple[int, int, int]) -> list[str]:
    forms = {"semver": semver(version), "quad": semver(version) + ".0"}
    touched: list[str] = []
    for rel, pattern, form in VERSION_SITES:
        path = ROOT / rel
        text = path.read_text(encoding="utf-8")
        new, count = re.subn(pattern, lambda m: m[1] + forms[form] + m[2], text, count=1)
        if count != 1:
            raise ReleaseError(f"{rel}: Fassung nicht gefunden (Muster {pattern})")
        if new != text:
            path.write_text(new, encoding="utf-8")
            if rel not in touched:
                touched.append(rel)
    return touched


# --- Changelog ---------------------------------------------------------------


HEADLINE_SEPARATOR = re.compile(r"\s+[—-]\s+")


def changelog_sections() -> list[tuple[str, str, str]]:
    """Liefert (Fassung, Datum, Text) je Eintrag, der neueste zuerst."""
    text = CHANGELOG.read_text(encoding="utf-8")
    parts = re.split(r"^## (.+)$", text, flags=re.MULTILINE)
    sections = []
    for head, body in zip(parts[1::2], parts[2::2]):
        # Wie splitHeadline in internal/changelog: Gedanken- oder Bindestrich trennt Fassung und Datum.
        name, *date = HEADLINE_SEPARATOR.split(head.strip(), maxsplit=1)
        sections.append((name, date[0] if date else "", body.strip() + "\n"))
    return sections


def release_notes(tag: str) -> str:
    for name, _, body in changelog_sections():
        if name == tag:
            return body
    raise ReleaseError(f"{CHANGELOG.relative_to(ROOT)} hat keinen Eintrag „## {tag} — <Datum>\"")


def date_changelog(tag: str, today: str) -> bool:
    sections = changelog_sections()
    if not sections or sections[0][0] != tag:
        found = sections[0][0] if sections else "keiner"
        raise ReleaseError(
            f"Der oberste Eintrag in {CHANGELOG.relative_to(ROOT)} muss „## {tag} — <Datum>\" sein "
            f"(gefunden: {found}). Schreibe zuerst, was diese Fassung ändert."
        )
    if not sections[0][2].strip():
        raise ReleaseError(f"Der Changelog-Eintrag {tag} ist leer")
    text = CHANGELOG.read_text(encoding="utf-8")
    new = re.sub(rf"^## {re.escape(tag)}(\s+[—-]\s+.*)?$", f"## {tag} — {today}", text, count=1, flags=re.MULTILINE)
    if new != text:
        CHANGELOG.write_text(new, encoding="utf-8")
        return True
    return False


# --- Werkzeuge ---------------------------------------------------------------


def run(*cmd: str, capture: bool = False, check: bool = True) -> str:
    print("$ " + " ".join(cmd), flush=True)
    result = subprocess.run(cmd, cwd=ROOT, text=True, capture_output=capture)
    if check and result.returncode != 0:
        detail = (result.stderr or "").strip() if capture else ""
        raise ReleaseError(f"„{' '.join(cmd)}\" endete mit Code {result.returncode}. {detail}".strip())
    return (result.stdout or "").strip() if capture else ""


def git(*args: str, check: bool = True) -> str:
    result = subprocess.run(("git",) + args, cwd=ROOT, text=True, capture_output=True)
    if check and result.returncode != 0:
        raise ReleaseError(f"git {' '.join(args)}: {result.stderr.strip()}")
    return result.stdout.strip()


def is_ancestor(older: str, newer: str) -> bool:
    return subprocess.run(["git", "merge-base", "--is-ancestor", older, newer], cwd=ROOT).returncode == 0


def require(tool: str, hint: str) -> None:
    if shutil.which(tool) is None:
        raise ReleaseError(f"{tool} fehlt. {hint}")


def remove_stale(patterns: list[str]) -> None:
    """Entfernt Pakete früherer Läufe aus bin/, damit keines unter neuem Namen mitreist."""
    for pattern in patterns:
        for path in (ROOT / "bin").glob(pattern):
            path.unlink()


def head_tag() -> str:
    tag = git("describe", "--tags", "--exact-match", "HEAD", check=False)
    if not tag:
        raise ReleaseError("HEAD trägt keinen Tag. Zuerst `task release:prepare -- <Fassung>` ausführen.")
    return tag


# --- Befehle -----------------------------------------------------------------


def cmd_prepare(args: argparse.Namespace) -> None:
    version = parse_version(args.version)
    tag = tag_for(version)

    if git("status", "--porcelain"):
        raise ReleaseError("Der Arbeitsstand hat uncommittete Änderungen.")
    branch = git("rev-parse", "--abbrev-ref", "HEAD")
    if branch != MAIN_BRANCH and not args.allow_branch:
        raise ReleaseError(f"Veröffentlicht wird von {MAIN_BRANCH}, nicht von {branch} (--allow-branch übergeht das).")
    git("fetch", "origin", "--tags", "--quiet")
    upstream = git("rev-parse", "--verify", "--quiet", f"origin/{branch}", check=False)
    if upstream and not is_ancestor(upstream, "HEAD"):
        raise ReleaseError(f"origin/{branch} enthält Commits, die lokal fehlen. Zuerst `git pull`.")
    if git("tag", "--list", tag):
        raise ReleaseError(f"Der Tag {tag} existiert bereits.")

    previous = current_version()
    today = args.date or datetime.date.today().isoformat()
    dated = date_changelog(tag, today)
    touched = set_version(version)
    print(f"Fassung {previous} → {semver(version)} in {len(touched)} Dateien, Changelog {tag} — {today}")

    if not args.skip_checks:
        try:
            require("task", "Installation: docs/entwicklung/README.md#voraussetzungen")
            # Vor `task check`: check_help.mjs liest die Oberfläche mit dem TypeScript aus frontend/.
            run("npm", "--prefix", "frontend", "ci", "--no-audit", "--no-fund")
            run("task", "check")
            run("npm", "--prefix", "frontend", "test")
            run("npm", "--prefix", "frontend", "run", "build")
            if not args.skip_bridge:
                run("task", "test:bridge")
        except ReleaseError as error:
            raise ReleaseError(f"{error}\nFassung und Changelog-Datum sind geändert, aber nicht committet; "
                               "`git checkout -- .` setzt sie zurück.") from error

    files = touched + ([str(CHANGELOG.relative_to(ROOT))] if dated else [])
    if files:
        git("add", "--", *files)
        git("commit", "--quiet", "-m", f"chore(release): {tag}", "-m",
            f"Setzt die Programmfassung auf {semver(version)} und datiert den Changelog-Eintrag.")
    git("tag", "--annotate", tag, "-m", f"Buchfink {tag}")

    print(f"\nFertig: Commit und Tag {tag} liegen lokal.")
    print("Weiter mit `task release:publish`, danach je Betriebssystem `task release:package`.")


def cmd_publish(args: argparse.Namespace) -> None:
    tag = head_tag()
    notes = release_notes(tag)
    branch = git("rev-parse", "--abbrev-ref", "HEAD")

    run("git", "push", "origin", branch)
    run("git", "push", "origin", tag)

    if shutil.which("gh") is None:
        print(f"\ngh fehlt; das Release bitte von Hand anlegen: "
              f"https://github.com/mxzinke/buchfink/releases/new?tag={tag}\nText:\n\n{notes}")
        return

    title = f"{tag}: {args.title}" if args.title else tag
    with tempfile.NamedTemporaryFile("w", suffix=".md", delete=False, encoding="utf-8") as handle:
        handle.write(notes)
        notes_file = handle.name
    try:
        cmd = ["gh", "release", "create", tag, "--verify-tag", "--title", title, "--notes-file", notes_file]
        if parse_version(tag)[0] == 0 or args.prerelease:
            cmd.append("--prerelease")
        if args.draft:
            cmd.append("--draft")
        run(*cmd)
    finally:
        os.unlink(notes_file)


def cmd_package(args: argparse.Namespace) -> None:
    tag = head_tag() if not args.allow_untagged else (git("describe", "--tags", "--always") or "dev")
    require("task", "Installation: docs/entwicklung/README.md#voraussetzungen")
    require("wails3", "Installation: docs/entwicklung/README.md#voraussetzungen")
    if git("status", "--porcelain"):
        raise ReleaseError("Der Arbeitsstand hat uncommittete Änderungen; die Fassung im Programm hieße dann „-dirty\".")

    system = platform.system()
    DIST.mkdir(exist_ok=True)
    produced: list[tuple[str, str]] = []  # (Quelle, Ziel)

    if system == "Darwin":
        identity = args.identity or os.environ.get("BUCHFINK_SIGN_IDENTITY", "")
        cmd = ["task", "darwin:package:release"]
        if identity:
            cmd.append(f"SIGN_IDENTITY={identity}")
        else:
            print("Hinweis: keine Signaturidentität (--identity oder BUCHFINK_SIGN_IDENTITY); signiert wird ad hoc.")
        run(*cmd)
        produced.append((f"bin/{APP_NAME}.dmg", f"Buchfink-{tag}-macOS-universal.dmg"))
        archive = DIST / f"Buchfink-{tag}-macOS-universal.zip"
        archive.unlink(missing_ok=True)
        # ditto erhält Signatur und erweiterte Attribute, zip nicht.
        run("ditto", "-c", "-k", "--keepParent", f"bin/{APP_NAME}.app", str(archive))
    elif system == "Linux":
        formats = [("*.AppImage", "AppImage"), ("*.deb", "deb"), ("*.rpm", "rpm"), ("*.pkg.tar.zst", "pkg.tar.zst")]
        remove_stale([pattern for pattern, _ in formats])
        run("task", "linux:package")
        arch = {"x86_64": "amd64", "aarch64": "arm64"}.get(platform.machine(), platform.machine())
        for pattern, suffix in formats:
            for source in glob.glob(str(ROOT / "bin" / pattern)):
                produced.append((source, f"Buchfink-{tag}-linux-{arch}.{suffix}"))
    elif system == "Windows":
        remove_stale(["*-installer.exe"])
        run("task", "windows:package")
        for source in glob.glob(str(ROOT / "bin" / "*-installer.exe")):
            arch = "arm64" if "arm64" in Path(source).name else "amd64"
            produced.append((source, f"Buchfink-{tag}-windows-{arch}-setup.exe"))
    else:
        raise ReleaseError(f"Für {system} gibt es keine Paketaufgabe.")

    for source, target in produced:
        src = ROOT / source
        if not src.exists():
            raise ReleaseError(f"Erwartetes Paket fehlt: {source}")
        shutil.copy2(src, DIST / target)

    print("\nPakete in dist/:")
    for item in sorted(DIST.iterdir()):
        print(f"  {item.name}")
    print("Anhängen mit `task release:upload`.")


def cmd_upload(args: argparse.Namespace) -> None:
    tag = head_tag()
    require("gh", "Installation: https://cli.github.com")
    files = [str(p) for p in sorted(DIST.glob(f"Buchfink-{tag}-*"))]
    if not files:
        raise ReleaseError(f"dist/ enthält keine Pakete für {tag}. Zuerst `task release:package`.")
    run("gh", "release", "upload", tag, "--clobber", *files)


def cmd_notes(args: argparse.Namespace) -> None:
    tag = args.tag or changelog_sections()[0][0]
    sys.stdout.write(release_notes(tag))


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = parser.add_subparsers(dest="command", required=True)

    p = sub.add_parser("prepare", help="Fassung setzen, prüfen, committen und taggen")
    p.add_argument("version", help="z. B. 0.3.0")
    p.add_argument("--date", help="Datum im Changelog (Standard: heute)")
    p.add_argument("--skip-checks", action="store_true", help="Prüfungen auslassen")
    p.add_argument("--skip-bridge", action="store_true", help="task test:bridge auslassen (braucht GTK/WebKit)")
    p.add_argument("--allow-branch", action="store_true", help=f"auch außerhalb von {MAIN_BRANCH}")
    p.set_defaults(func=cmd_prepare)

    p = sub.add_parser("publish", help="Branch und Tag pushen, GitHub-Release anlegen")
    p.add_argument("--title", help="Titel nach dem Tag, z. B. „Gründung und Stammdaten\"")
    p.add_argument("--draft", action="store_true", help="als Entwurf anlegen")
    p.add_argument("--prerelease", action="store_true", help="als Vorabversion markieren (bei 0.x immer)")
    p.set_defaults(func=cmd_publish)

    p = sub.add_parser("package", help="Pakete für dieses Betriebssystem nach dist/")
    p.add_argument("--identity", help="macOS: Zertifikatsname für codesign (sonst BUCHFINK_SIGN_IDENTITY)")
    p.add_argument("--allow-untagged", action="store_true", help="auch ohne Tag an HEAD, zum Ausprobieren")
    p.set_defaults(func=cmd_package)

    p = sub.add_parser("upload", help="Pakete aus dist/ an das Release hängen")
    p.set_defaults(func=cmd_upload)

    p = sub.add_parser("notes", help="Text der Veröffentlichung ausgeben")
    p.add_argument("tag", nargs="?", help="Standard: oberster Changelog-Eintrag")
    p.set_defaults(func=cmd_notes)

    args = parser.parse_args()
    try:
        args.func(args)
    except ReleaseError as error:
        print(f"Abbruch: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
