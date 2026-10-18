#!/usr/bin/env python3
"""Prüft die Endnutzer-Skills unter skills/ und packt sie für den Upload.

Je Skill entsteht <name>.zip mit dem Ordner <name>/ als oberster Ebene. So
erwarten Claude Desktop und claude.ai den Upload (Customize > Skills).
buchfink-skills.zip enthält alle Skill-Ordner zum Entpacken in den
Skill-Ordner eines Clients, etwa ~/.agents/skills/. SHA256SUMS listet die
Prüfsummen aller Dateien.

Geprüft wird vor dem Packen:
- Frontmatter nach dem Agent-Skills-Standard (https://agentskills.io/specification),
  davon nur name, description, license (EUPL-1.2), compatibility und metadata;
- name gleich Ordnername, SKILL.md unter MAX_SKILL_LINES Zeilen;
- jeder Link auf references/ zeigt auf eine vorhandene Datei;
- keine Symlinks innerhalb eines Skills;
- plugins/buchfink-buchhaltung/skills/ enthält für jeden Skill genau einen
  relativen Symlink ../../../skills/<name> und sonst nichts;
- marketplace.json, plugin.json und .mcp.json passen zu Ordner und Server.

Den Inhalt der Skills (Links, Anker, Pflichten) prüft scripts/check_skills.py.

Aufruf:
    python3 scripts/package_skills.py              # prüfen und nach dist/skills packen
    python3 scripts/package_skills.py --check-only # nur prüfen

Die ZIP-Dateien sind reproduzierbar: feste Reihenfolge, feste Zeitstempel
(SOURCE_DATE_EPOCH, sonst 1980-01-01), feste Dateirechte.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import sys
import time
import zipfile
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent
PLUGIN_NAME = "buchfink-buchhaltung"
MARKETPLACE_NAME = "buchfink"
MCP_URL = "http://127.0.0.1:7465/mcp"
LICENSE = "EUPL-1.2"
BUNDLE_NAME = "buchfink-skills.zip"

# Projektregel; der Standard empfiehlt unter 500.
MAX_SKILL_LINES = 300
ALLOWED_KEYS = {"name", "description", "license", "compatibility", "metadata"}
NAME_RE = re.compile(r"^[a-z0-9]+(-[a-z0-9]+)*$")
REFERENCE_LINK_RE = re.compile(r"\]\((references/[^)#\s]+)")
IGNORED_NAMES = {".DS_Store", "Thumbs.db", "__pycache__"}


class Problems:
    def __init__(self) -> None:
        self.items: list[str] = []

    def add(self, where: str, message: str) -> None:
        self.items.append(f"{where}: {message}")


def read_frontmatter(skill_md: Path) -> str | None:
    text = skill_md.read_text(encoding="utf-8")
    if not text.startswith("---\n"):
        return None
    end = text.find("\n---\n", 4)
    if end < 0:
        return None
    return text[4:end]


def parse_frontmatter(raw: str) -> dict:
    """Liest das Frontmatter mit PyYAML, falls vorhanden.

    Ohne PyYAML genügt ein kleiner Leser für die Formen, die die Skills
    verwenden: Schlüssel mit einfachem oder gequotetem Wert und eine
    eingerückte Zuordnung unter metadata.
    """
    try:
        import yaml  # type: ignore

        data = yaml.safe_load(raw)
        return data if isinstance(data, dict) else {}
    except ImportError:
        pass

    data: dict = {}
    current_map: dict | None = None
    for line in raw.splitlines():
        if not line.strip() or line.lstrip().startswith("#"):
            continue
        indented = line[0] in " \t"
        key, sep, value = line.strip().partition(":")
        if not sep:
            raise ValueError(f"Zeile ohne Schlüssel: {line!r}")
        value = unquote(value.strip())
        if indented:
            if current_map is None:
                raise ValueError(f"Eingerückte Zeile ohne Zuordnung: {line!r}")
            current_map[key] = value
            continue
        if value == "":
            current_map = {}
            data[key] = current_map
        else:
            current_map = None
            data[key] = value
    return data


def unquote(value: str) -> str:
    if len(value) >= 2 and value[0] == value[-1] == '"':
        return re.sub(r'\\(["\\])', r"\1", value[1:-1])
    if len(value) >= 2 and value[0] == value[-1] == "'":
        return value[1:-1].replace("''", "'")
    return value


def check_skill(skill_dir: Path, problems: Problems) -> None:
    where = f"skills/{skill_dir.name}"
    skill_md = skill_dir / "SKILL.md"
    if not skill_md.is_file():
        problems.add(where, "SKILL.md fehlt")
        return

    raw = read_frontmatter(skill_md)
    if raw is None:
        problems.add(where, "SKILL.md beginnt nicht mit einem Frontmatter zwischen zwei ---")
        return
    try:
        meta = parse_frontmatter(raw)
    except ValueError as err:
        problems.add(where, f"Frontmatter nicht lesbar: {err}")
        return

    unknown = set(meta) - ALLOWED_KEYS
    if unknown:
        problems.add(where, f"Felder außerhalb des Standards: {', '.join(sorted(unknown))}")

    name = meta.get("name")
    if name != skill_dir.name:
        problems.add(where, f"name {name!r} weicht vom Ordnernamen ab")
    if not isinstance(name, str) or len(name) > 64 or not NAME_RE.match(name or ""):
        problems.add(where, "name: nur Kleinbuchstaben, Ziffern und einzelne Bindestriche, höchstens 64 Zeichen")

    description = meta.get("description")
    if not isinstance(description, str) or not 1 <= len(description) <= 1024:
        problems.add(where, "description fehlt oder ist länger als 1024 Zeichen")

    if meta.get("license") != LICENSE:
        problems.add(where, f"license muss {LICENSE} sein")

    compatibility = meta.get("compatibility")
    if compatibility is not None and (not isinstance(compatibility, str) or len(compatibility) > 500):
        problems.add(where, "compatibility ist länger als 500 Zeichen")

    metadata = meta.get("metadata")
    if metadata is not None:
        if not isinstance(metadata, dict) or not all(isinstance(v, str) for v in metadata.values()):
            problems.add(where, "metadata muss Schlüssel auf Zeichenketten abbilden")

    lines = skill_md.read_text(encoding="utf-8").count("\n")
    if lines >= MAX_SKILL_LINES:
        problems.add(where, f"SKILL.md hat {lines} Zeilen, erlaubt sind unter {MAX_SKILL_LINES}")
    print(f"{skill_dir.name}: {lines} Zeilen, description {len(description or '')} Zeichen")

    for target in sorted(set(REFERENCE_LINK_RE.findall(skill_md.read_text(encoding="utf-8")))):
        if not (skill_dir / target).is_file():
            problems.add(where, f"Verweis auf {target} zeigt ins Leere")

    for path in skill_dir.rglob("*"):
        if path.is_symlink():
            problems.add(where, f"{path.relative_to(skill_dir)} ist ein Symlink; Skills enthalten nur echte Dateien")


def check_plugin_links(skills: list[str], plugin_skills: Path, problems: Problems) -> None:
    where = f"plugins/{PLUGIN_NAME}/skills"
    if not plugin_skills.is_dir():
        problems.add(where, "Ordner fehlt")
        return
    present = {p.name: p for p in plugin_skills.iterdir()}
    for name in skills:
        link = present.pop(name, None)
        expected = f"../../../skills/{name}"
        if link is None:
            problems.add(where, f"Symlink {name} fehlt (ln -s {expected} {name})")
        elif not link.is_symlink():
            problems.add(where, f"{name} ist kein Symlink")
        elif os.readlink(link) != expected:
            problems.add(where, f"{name} zeigt auf {os.readlink(link)}, erwartet {expected}")
    for extra in sorted(present):
        problems.add(where, f"{extra} gehört zu keinem Skill unter skills/")


def load_json(path: Path, problems: Problems) -> dict:
    where = str(path.relative_to(REPO)) if path.is_relative_to(REPO) else str(path)
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except FileNotFoundError:
        problems.add(where, "Datei fehlt")
        return {}
    except json.JSONDecodeError as err:
        problems.add(where, f"kein gültiges JSON: {err}")
        return {}
    if not isinstance(data, dict):
        problems.add(where, "erwartet ein JSON-Objekt")
        return {}
    return data


def check_manifests(repo: Path, plugin_dir: Path, problems: Problems) -> None:
    """Prüft, was Claude Code beim Installieren liest: Marketplace, Plugin, Server."""
    market_path = repo / ".claude-plugin" / "marketplace.json"
    market = load_json(market_path, problems)
    if market:
        where = ".claude-plugin/marketplace.json"
        if market.get("name") != MARKETPLACE_NAME:
            problems.add(where, f"name {market.get('name')!r}, erwartet {MARKETPLACE_NAME!r}")
        entries = [p for p in market.get("plugins", []) if isinstance(p, dict) and p.get("name") == PLUGIN_NAME]
        if len(entries) != 1:
            problems.add(where, f"genau ein Eintrag {PLUGIN_NAME!r} erwartet, gefunden {len(entries)}")
        for entry in entries:
            source = entry.get("source")
            if not isinstance(source, str) or (repo / source).resolve() != plugin_dir.resolve():
                problems.add(where, f"source {source!r} zeigt nicht auf plugins/{PLUGIN_NAME}")

    plugin = load_json(plugin_dir / ".claude-plugin" / "plugin.json", problems)
    if plugin:
        where = f"plugins/{PLUGIN_NAME}/.claude-plugin/plugin.json"
        if plugin.get("name") != PLUGIN_NAME:
            problems.add(where, f"name {plugin.get('name')!r}, erwartet {PLUGIN_NAME!r}")
        if plugin.get("license") != LICENSE:
            problems.add(where, f"license muss {LICENSE} sein")

    mcp = load_json(plugin_dir / ".mcp.json", problems)
    if mcp:
        where = f"plugins/{PLUGIN_NAME}/.mcp.json"
        server = mcp.get("mcpServers", {}).get("buchfink") if isinstance(mcp.get("mcpServers"), dict) else None
        if not isinstance(server, dict):
            problems.add(where, "mcpServers.buchfink fehlt")
        elif server.get("type") != "http" or server.get("url") != MCP_URL:
            problems.add(where, f"mcpServers.buchfink muss type http und url {MCP_URL} haben")


def skill_files(skill_dir: Path) -> list[Path]:
    files = []
    for path in sorted(skill_dir.rglob("*")):
        rel = path.relative_to(skill_dir)
        if any(part in IGNORED_NAMES or part.startswith(".") for part in rel.parts):
            continue
        files.append(path)
    return files


def zip_time() -> tuple[int, int, int, int, int, int]:
    epoch = os.environ.get("SOURCE_DATE_EPOCH")
    if not epoch:
        return (1980, 1, 1, 0, 0, 0)
    t = time.gmtime(max(int(epoch), 315532800))
    return (t.tm_year, t.tm_mon, t.tm_mday, t.tm_hour, t.tm_min, t.tm_sec)


def add_skill(archive: zipfile.ZipFile, skill_dir: Path, stamp: tuple) -> None:
    def entry(name: str, mode: int) -> zipfile.ZipInfo:
        info = zipfile.ZipInfo(name, date_time=stamp)
        info.external_attr = mode << 16
        info.create_system = 3
        return info

    archive.writestr(entry(f"{skill_dir.name}/", 0o40755), b"")
    for path in skill_files(skill_dir):
        name = f"{skill_dir.name}/{path.relative_to(skill_dir).as_posix()}"
        if path.is_dir():
            archive.writestr(entry(name + "/", 0o40755), b"")
        else:
            info = entry(name, 0o100644)
            info.compress_type = zipfile.ZIP_DEFLATED
            archive.writestr(info, path.read_bytes())


def package(skill_dirs: list[Path], out: Path) -> list[Path]:
    out.mkdir(parents=True, exist_ok=True)
    stamp = zip_time()
    written = []
    for skill_dir in skill_dirs:
        target = out / f"{skill_dir.name}.zip"
        with zipfile.ZipFile(target, "w") as archive:
            add_skill(archive, skill_dir, stamp)
        written.append(target)
    bundle = out / BUNDLE_NAME
    with zipfile.ZipFile(bundle, "w") as archive:
        for skill_dir in skill_dirs:
            add_skill(archive, skill_dir, stamp)
    written.append(bundle)

    sums = "".join(f"{hashlib.sha256(p.read_bytes()).hexdigest()}  {p.name}\n" for p in written)
    (out / "SHA256SUMS").write_text(sums, encoding="utf-8")
    return written


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--skills-dir", type=Path, default=REPO / "skills")
    parser.add_argument("--plugin-dir", type=Path, default=REPO / "plugins" / PLUGIN_NAME)
    parser.add_argument("--out", type=Path, default=REPO / "dist" / "skills")
    parser.add_argument("--check-only", action="store_true", help="nur prüfen, nichts schreiben")
    args = parser.parse_args()

    skill_dirs = sorted(p for p in args.skills_dir.iterdir() if p.is_dir() and not p.name.startswith("."))
    if not skill_dirs:
        print(f"Keine Skills unter {args.skills_dir}", file=sys.stderr)
        return 1

    problems = Problems()
    for skill_dir in skill_dirs:
        check_skill(skill_dir, problems)
    check_plugin_links([p.name for p in skill_dirs], args.plugin_dir / "skills", problems)
    check_manifests(REPO, args.plugin_dir, problems)

    if problems.items:
        print("Die Skills sind nicht bereit:", file=sys.stderr)
        for item in problems.items:
            print(f"  {item}", file=sys.stderr)
        return 1

    print(f"{len(skill_dirs)} Skills geprüft.")
    if args.check_only:
        return 0
    for path in package(skill_dirs, args.out):
        print(f"  {path}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
