#!/usr/bin/env python3
"""Prüft, dass das Frontend nur die Tokens aus docs/design-konzept.md benutzt.

Zwei Regeln, beide aus §17:

1. Keine Klasse aus einer Tailwind-Standardpalette. Die Farben der Anwendung
   stehen als Tokens in `frontend/src/index.css`; `stone-700` oder `amber-600`
   umgehen sie und lassen sich später nicht umstellen.
2. Kein Hex-Literal im Komponentencode. Es ist die Bedingung für den
   Dunkelmodus (§16).

`bg-white`, `text-white` und `bg-black` bleiben erlaubt: Sie sind die reinen
Endpunkte, keine Palette, und stehen für die Fläche auf der Schale.
"""

from __future__ import annotations

import re
import sys
from pathlib import Path

SRC = Path(__file__).resolve().parent.parent / 'frontend' / 'src'

# Die vollständige Tailwind-Farbpalette. Wir prüfen auf `name-<zahl>`, damit
# eigene Tokens wie `accent-light` oder `shell-line` nicht anschlagen.
PALETTES = (
    'slate gray zinc neutral stone red orange amber yellow lime green emerald '
    'teal cyan sky blue indigo violet purple fuchsia pink rose'
).split()

PALETTE_RE = re.compile(r'\b(?:' + '|'.join(PALETTES) + r')-(?:50|[1-9]00|950)\b')
HEX_RE = re.compile(r'#[0-9a-fA-F]{3,8}\b')

# index.css definiert die Tokens, dort gehören die Hex-Werte hin. Die Flagge ist
# in ihrer eigenen Datei begründet.
HEX_EXEMPT = {'index.css', 'GermanFlag.tsx'}


def main() -> int:
    findings: list[str] = []

    for path in sorted(SRC.rglob('*')):
        if path.suffix not in {'.ts', '.tsx', '.css'} or not path.is_file():
            continue
        relative = path.relative_to(SRC.parent.parent)

        for number, line in enumerate(path.read_text(encoding='utf-8').splitlines(), 1):
            for match in PALETTE_RE.finditer(line):
                findings.append(f'{relative}:{number}: alte Palette `{match.group()}`')
            if path.name not in HEX_EXEMPT:
                for match in HEX_RE.finditer(line):
                    findings.append(f'{relative}:{number}: Hex-Literal `{match.group()}`')

    if findings:
        print('Verstöße gegen das Design-Konzept (§17):\n')
        for finding in findings:
            print(f'  {finding}')
        print(f'\n{len(findings)} Stellen. Die Tokens stehen in frontend/src/index.css.')
        return 1

    print('Design-Tokens: keine alten Paletten, keine Hex-Literale.')
    return 0


if __name__ == '__main__':
    sys.exit(main())
