#!/usr/bin/env python3
"""Prüft relative Links in den Markdown-Dateien des Repositorys.

Ein Link zeigt auf eine Datei oder einen Ordner, der existiert; ein Anker auf
eine Überschrift der Zieldatei, gebildet wie bei GitHub. Externe Adressen
prüft das Skript nicht.

Aufruf: python3 scripts/check_doc_links.py
"""

from __future__ import annotations

import re
import subprocess
import sys
import unicodedata
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent

LINK = re.compile(r'(?<!!)\[[^\]]*\]\(([^)\s]+)(?:\s+"[^"]*")?\)|^\[[^\]]+\]:\s*(\S+)', re.M)
FENCE = re.compile(r'```.*?```', re.S)
HEADING = re.compile(r'^#{1,6}\s+(.*?)\s*#*\s*$', re.M)
# Feste Anker wie <a id="ustva"></a> im Pflichtenkalender der Skills.
HTML_ANCHOR = re.compile(r'<a\s+(?:id|name)="([^"]+)"')


def markdown_files() -> list[Path]:
    out = subprocess.run(
        ['git', 'ls-files', '--cached', '--others', '--exclude-standard', '*.md'],
        cwd=ROOT, capture_output=True, text=True, check=True,
    ).stdout.split()
    return [ROOT / f for f in out if not f.startswith(('frontend/node_modules/', 'internal/einvoice/testdata/'))]


def slug(heading: str) -> str:
    text = re.sub(r'`([^`]*)`', r'\1', heading)
    text = re.sub(r'\[([^\]]*)\]\([^)]*\)', r'\1', text).strip().lower()
    kept = [ch for ch in text if ch in '-_ ' or ch.isalnum() or unicodedata.category(ch).startswith('L')]
    return ''.join(kept).replace(' ', '-')


_anchors: dict[Path, set[str]] = {}


def anchors(path: Path) -> set[str]:
    if path not in _anchors:
        text = FENCE.sub('', path.read_text(encoding='utf-8'))
        found: set[str] = set(HTML_ANCHOR.findall(text))
        seen: dict[str, int] = {}
        for match in HEADING.finditer(text):
            base = slug(match.group(1))
            count = seen.get(base, 0)
            seen[base] = count + 1
            found.add(base if count == 0 else f'{base}-{count}')
        _anchors[path] = found
    return _anchors[path]


def main() -> int:
    findings: list[str] = []
    for source in markdown_files():
        if not source.exists():
            continue
        text = FENCE.sub('', source.read_text(encoding='utf-8'))
        for match in LINK.finditer(text):
            target = match.group(1) or match.group(2)
            if re.match(r'^[a-z][a-z0-9+.-]*:', target) or target.startswith('//'):
                continue
            path, _, fragment = target.partition('#')
            dest = source if path == '' else (source.parent / path).resolve()
            name = source.relative_to(ROOT).as_posix()
            if not dest.exists():
                findings.append(f'{name}: Ziel fehlt: {target}')
            elif fragment and dest.suffix == '.md' and fragment not in anchors(dest):
                findings.append(f'{name}: Überschrift fehlt: {target}')
    for finding in findings:
        print(finding)
    return 1 if findings else 0


if __name__ == '__main__':
    sys.exit(main())
