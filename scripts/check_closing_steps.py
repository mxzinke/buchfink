#!/usr/bin/env python3
"""Prüft, dass jeder Baustein des Jahresabschlusses ein Ziel hat (Architektur 6.3).

Der geführte Weg verlangt, dass jeder Schritt seine Arbeit öffnet. Eine Zeile
ohne Knopf benennt eine Arbeit und führt nicht zu ihr — und wer den Weg von oben
beginnt, stünde ausgerechnet beim ersten Schritt vor einer Zeile ohne Ausgang.

Die Schlüssel stehen im Backend (internal/domain/closing_step.go); die Ziele
stehen in der Oberfläche, verteilt auf drei Zuordnungen: STEP_TABS (Reiter der
Abschlussbausteine), STEP_OBLIGATIONS (Reiter der Nebenpflichten) und STEP_PAGES
(eigene Seiten). Die Feststellung liegt auf der Abschlussseite selbst und springt
in den Abschnitt darunter; sie wird über ihren Anker erkannt. Bausteine ohne
eigene Ansicht, deren Vorschau und Buchung über den Kern laufen, stehen
ausdrücklich in STEP_CORE_ONLY; auch sie sind damit benannt und nicht vergessen.

Die Prüfung läuft gegen den Go-Quelltext und nicht gegen eine Liste in diesem
Skript: eine zweite Liste liefe auseinander, sobald ein Baustein dazukommt — und
genau dann soll die Prüfung anschlagen.

Aufrufbar über `task check:steps`; Rückgabewert 1, wenn ein Schlüssel ohne Ziel
bleibt.
"""

from __future__ import annotations

import re
import sys
from pathlib import Path

DOMAIN = Path("internal/domain/closing_step.go")
MODULES = Path("frontend/src/pages/ClosingModulesPage.tsx")
CLOSING = Path("frontend/src/pages/ClosingPage.tsx")

# Die Zuordnungen der Oberfläche: Datei und Name der Konstante.
TARGET_MAPS = (
    (MODULES, "STEP_TABS"),
    (MODULES, "STEP_OBLIGATIONS"),
    (CLOSING, "STEP_PAGES"),
)


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


def list_keys(text: str, name: str) -> set[str]:
    """Die Schlüssel einer Liste `const NAME: ... = [ ... ];`."""
    start = re.search(rf"const\s+{name}\b[^=]*=\s*\[", text)
    if start is None:
        raise SystemExit(f"Die Liste {name} steht nicht dort, wo sie erwartet wird.")
    end = text.index("];", start.end())
    return set(re.findall(r"'([a-z_]+)'", text[start.end() : end]))


def anchor_keys(text: str) -> set[str]:
    """Schlüssel, deren Ziel ein Abschnitt derselben Seite ist."""
    return set(re.findall(r"key === '([a-z_]+)'", text))


def main() -> int:
    for path in (DOMAIN, MODULES, CLOSING):
        if not path.exists():
            print(f"Datei fehlt: {path}", file=sys.stderr)
            return 1

    keys = domain_keys(DOMAIN.read_text(encoding="utf-8"))
    if not keys:
        print("In internal/domain/closing_step.go steht kein Schlüssel.", file=sys.stderr)
        return 1

    targets: set[str] = set()
    for path, name in TARGET_MAPS:
        targets |= map_keys(path.read_text(encoding="utf-8"), name)
    targets |= anchor_keys(CLOSING.read_text(encoding="utf-8"))
    core_only = list_keys(MODULES.read_text(encoding="utf-8"), "STEP_CORE_ONLY")
    targets |= core_only

    missing = [key for key in keys if key not in targets]
    if missing:
        print("Bausteine ohne Ziel im geführten Weg (Architektur 6.3):", file=sys.stderr)
        for key in missing:
            print(f"  {key}", file=sys.stderr)
        print(
            "Ein Ziel gehört in STEP_TABS, STEP_OBLIGATIONS oder STEP_PAGES, "
            "als Anker auf die Abschlussseite selbst oder, ohne eigene Ansicht, "
            "in STEP_CORE_ONLY.",
            file=sys.stderr,
        )
        return 1

    unknown = sorted(targets - set(keys))
    if unknown:
        print("Ziele ohne Baustein:", ", ".join(unknown), file=sys.stderr)
        return 1

    print(
        f"Geführter Weg: {len(keys)} Bausteine, jeder mit Ziel; "
        f"{len(core_only)} davon ohne eigene Ansicht ({', '.join(sorted(core_only))})."
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
