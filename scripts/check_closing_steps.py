#!/usr/bin/env python3
"""Prüft, dass jeder Baustein des Jahresabschlusses in der Oberfläche steht.

Der Reiter „Abschlüsse → Jahr" zeigt jeden Baustein mit seinem Stand und, auf
Klick, der Kartei dahinter. Ein Baustein ohne Eintrag fehlte dort, und ein
Prüfer sähe die Kartei nicht, ohne Claude zu fragen.

Die Schlüssel stehen im Backend (internal/domain/closing_step.go); die
Zuordnung steht in STEP_VIEWS in frontend/src/pages/closings/yearSteps.tsx:
eine Kartei in der Seitenleiste, ein Verweis in einen anderen Reiter oder ein
Schritt des Menschen in der Schrittliste.

Die Prüfung läuft gegen den Go-Quelltext und nicht gegen eine Liste in diesem
Skript: eine zweite Liste liefe auseinander, sobald ein Baustein dazukommt, und
genau dann soll die Prüfung anschlagen.

Aufrufbar über `task check:steps`; Rückgabewert 1, wenn ein Schlüssel ohne
Eintrag bleibt.
"""

from __future__ import annotations

import re
import sys
from pathlib import Path

DOMAIN = Path("internal/domain/closing_step.go")
VIEWS = Path("frontend/src/pages/closings/yearSteps.tsx")


def domain_keys(text: str) -> list[str]:
    """Die Schlüssel aus den Konstanten des Backends, in ihrer Reihenfolge."""
    return re.findall(r'ClosingStepKey\s*=\s*"([a-z_]+)"', text)


def map_keys(text: str, name: str) -> set[str]:
    """Die Schlüssel einer Zuordnung `const NAME: ... = { ... };`."""
    start = re.search(rf"const\s+{name}\b[^=]*=\s*\{{", text)
    if start is None:
        raise SystemExit(f"Die Zuordnung {name} steht nicht dort, wo sie erwartet wird.")
    end = text.index("};", start.end())
    body = text[start.end() : end]
    return set(re.findall(r"^\s*([A-Za-z_][A-Za-z0-9_]*)\s*:", body, re.MULTILINE))


def main() -> int:
    for path in (DOMAIN, VIEWS):
        if not path.exists():
            print(f"Datei fehlt: {path}", file=sys.stderr)
            return 1

    keys = domain_keys(DOMAIN.read_text(encoding="utf-8"))
    if not keys:
        print("In internal/domain/closing_step.go steht kein Schlüssel.", file=sys.stderr)
        return 1

    views = VIEWS.read_text(encoding="utf-8")
    targets = map_keys(views, "STEP_VIEWS")

    missing = [key for key in keys if key not in targets]
    if missing:
        print("Bausteine ohne Eintrag in STEP_VIEWS:", file=sys.stderr)
        for key in missing:
            print(f"  {key}", file=sys.stderr)
        return 1

    unknown = sorted(targets - set(keys))
    if unknown:
        print("Einträge ohne Baustein:", ", ".join(unknown), file=sys.stderr)
        return 1

    human = sorted(re.findall(r"^\s*([a-z_]+):\s*\{\s*kind:\s*'human'", views, re.MULTILINE))
    print(
        f"Jahresabschluss: {len(keys)} Bausteine, jeder in der Oberfläche; "
        f"{len(human)} davon Schritte des Menschen ({', '.join(human)})."
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
