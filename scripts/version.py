#!/usr/bin/env python3
"""Setzt und prüft die Fassung in den Paketdateien.

Die Fassung im Programm kommt aus dem Git-Tag (Taskfile.yml, VERSION). Daneben
tragen Paketdateien eine feste Fassung: Info.plist, die Windows-Ressourcen,
MSIX, NSIS, nfpm, das MCPB-Manifest und das Claude-Plugin. `task release:check`
bricht ab, wenn eine davon oder die Änderungshistorie nicht zur Fassung passt.

    python3 scripts/version.py set 0.3.1      Fassung in alle Paketdateien schreiben
    python3 scripts/version.py check [0.3.1]  Paketdateien und Änderungshistorie prüfen;
                                              ohne Fassung gilt build/config.yml
    python3 scripts/version.py notes 0.3.1    Eintrag der Änderungshistorie ausgeben
"""

from __future__ import annotations

import re
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent
CHANGELOG = REPO / "internal" / "changelog" / "CHANGELOG.md"
VERSION_RE = re.compile(r"^\d+\.\d+\.\d+(-[0-9A-Za-z.]+)?$")

PLIST = r"(<key>CFBundle(?:ShortVersionString|Version)</key>\s*<string>)([^<]*)(</string>)"
JSON_TOP = r'^(  "version": ")([^"]*)(")'

# (Datei, Muster mit der Fassung in Gruppe 2, Anzahl Fundstellen, Form, Pflicht)
# Form "msix": MSIX verlangt vier Zahlen ohne Vorabkennung.
RULES = [
    ("build/config.yml", r'^(  version: ")([^"]*)(")', 1, "semver", True),
    ("build/darwin/Info.plist", PLIST, 2, "semver", True),
    ("build/darwin/Info.dev.plist", PLIST, 2, "semver", True),
    ("build/windows/info.json", r'("(?:file_version|ProductVersion)": ")([^"]*)(")', 2, "semver", True),
    ("build/windows/wails.exe.manifest",
     r'(<assemblyIdentity type="win32" name="(?!Microsoft\.)[^"]*" version=")([^"]*)(")', 1, "semver", True),
    ("build/windows/msix/app_manifest.xml", r'(\sVersion=")([^"]*)(")', 1, "msix", True),
    ("build/windows/msix/template.xml", r'(\sVersion=")([^"]*)(")', 1, "msix", True),
    ("build/windows/nsis/wails_tools.nsh", r'(!define INFO_PRODUCTVERSION ")([^"]*)(")', 1, "semver", True),
    ("build/windows/nsis/project.nsi", r'(## !define INFO_PRODUCTVERSION ")([^"]*)(")', 1, "semver", True),
    ("build/windows/nsis/project.nsi", r'(INFO_PRODUCTVERSION "[^"]*"\s+# Default ")([^"]*)(")', 1, "semver", True),
    ("build/linux/nfpm/nfpm.yaml", r'^(version: ")([^"]*)(")', 1, "semver", True),
    ("packaging/claude-desktop/manifest.json", JSON_TOP, 1, "semver", True),
    ("plugins/buchfink-buchhaltung/.claude-plugin/plugin.json", JSON_TOP, 1, "semver", True),
]


def render(version: str, form: str) -> str:
    if form == "msix":
        return version.split("-", 1)[0] + ".0"
    return version


def config_version() -> str:
    text = (REPO / "build" / "config.yml").read_text(encoding="utf-8")
    match = re.search(RULES[0][1], text, re.MULTILINE)
    if not match:
        sys.exit("build/config.yml: keine Fassung unter info.version")
    return match.group(2)


def changelog_entry(version: str) -> tuple[str, str] | None:
    """Liefert Datum und Text des Eintrags „## v<Fassung> — <Datum>"."""
    lines = CHANGELOG.read_text(encoding="utf-8").splitlines()
    heading = re.compile(rf"^## v{re.escape(version)}(?: [—–-] (.*))?$")
    for i, line in enumerate(lines):
        match = heading.match(line.strip())
        if not match:
            continue
        body = []
        for following in lines[i + 1:]:
            if following.startswith("## "):
                break
            body.append(following)
        return (match.group(1) or "").strip(), "\n".join(body).strip()
    return None


def set_version(version: str) -> None:
    for path, pattern, count, form, required in RULES:
        file = REPO / path
        if not file.exists():
            if required:
                sys.exit(f"{path}: fehlt")
            continue
        text = file.read_text(encoding="utf-8")
        new, found = re.subn(pattern, lambda m: m.group(1) + render(version, form) + m.group(3),
                             text, flags=re.MULTILINE)
        if found != count:
            sys.exit(f"{path}: {found} statt {count} Fundstellen")
        file.write_text(new, encoding="utf-8")
    print(f"Fassung {version} gesetzt. Eintrag „## v{version} — <Datum>“ in {CHANGELOG.relative_to(REPO)} nicht vergessen.")


def check(version: str) -> int:
    problems = []
    for path, pattern, count, form, required in RULES:
        file = REPO / path
        if not file.exists():
            if required:
                problems.append(f"{path}: fehlt")
            continue
        values = [m.group(2) for m in re.finditer(pattern, file.read_text(encoding="utf-8"), re.MULTILINE)]
        want = render(version, form)
        if len(values) != count:
            problems.append(f"{path}: {len(values)} statt {count} Fundstellen")
        problems += [f"{path}: {value}, erwartet {want}" for value in values if value != want]
    entry = changelog_entry(version)
    if entry is None:
        problems.append(f"{CHANGELOG.relative_to(REPO)}: kein Eintrag „## v{version} — <Datum>“")
    elif not re.fullmatch(r"\d{4}-\d{2}-\d{2}", entry[0]):
        problems.append(f"{CHANGELOG.relative_to(REPO)}: v{version} ohne Datum JJJJ-MM-TT")
    for problem in problems:
        print(problem, file=sys.stderr)
    if problems:
        print(f"Abhilfe: python3 scripts/version.py set {version}", file=sys.stderr)
        return 1
    print(f"Fassung {version} stimmt in allen Paketdateien und der Änderungshistorie.")
    return 0


def main(argv: list[str]) -> int:
    if len(argv) < 1 or argv[0] not in ("set", "check", "notes"):
        print(__doc__, file=sys.stderr)
        return 2
    command, rest = argv[0], argv[1:]
    if command != "check" and len(rest) != 1:
        print(__doc__, file=sys.stderr)
        return 2
    version = rest[0].removeprefix("v") if rest else config_version()
    if not VERSION_RE.match(version):
        print(f"keine Fassung X.Y.Z: {version}", file=sys.stderr)
        return 2
    if command == "set":
        set_version(version)
        return 0
    if command == "check":
        return check(version)
    entry = changelog_entry(version)
    if entry is None:
        print(f"kein Eintrag für v{version} in {CHANGELOG.relative_to(REPO)}", file=sys.stderr)
        return 1
    print(unwrap(entry[1]))
    return 0


def unwrap(text: str) -> str:
    """Fügt umbrochene Zeilen zusammen; GitHub zeigt jeden Umbruch als Zeilenende."""
    out: list[str] = []
    for line in text.splitlines():
        line = line.strip()
        if not line or line.startswith("- ") or not out or not out[-1]:
            out.append(line)
        else:
            out[-1] += " " + line
    return "\n".join(out)


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
