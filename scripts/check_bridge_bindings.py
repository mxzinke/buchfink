#!/usr/bin/env python3
"""Prüft, dass `frontend/src/services/bridge.ts` zur Go-Bridge passt.

Das Frontend ruft die Bridge namensbasiert über die Wails-Laufzeit auf, ohne
generierte Bindings. Damit ist `bridge.ts` die einzige Stelle, an der die Namen
stehen — und nichts hat bisher geprüft, ob sie stimmen. Ein falscher Name fällt
erst zur Laufzeit auf, als „Binding call failed: unknown bound method name".

Zwei Regeln:

1. Der Dienst-Präfix muss der voll qualifizierte Go-Name sein, also
   `<Modulpfad aus go.mod>/internal/wailsbridge.BuchfinkBridge`. Wails legt
   gebundene Methoden unter genau diesem Schlüssel ab; der kurze Paketname
   findet nichts.
2. Jeder aufgerufene Methodenname muss eine exportierte Methode auf
   `*BuchfinkBridge` sein.

Die Gegenrichtung — eine Go-Methode ohne Eintrag in `bridge.ts` — ist kein
Fehler: Nicht jede Methode braucht das Frontend.
"""

from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
BRIDGE_TS = ROOT / 'frontend' / 'src' / 'services' / 'bridge.ts'
BRIDGE_GO_DIR = ROOT / 'internal' / 'wailsbridge'
GO_MOD = ROOT / 'go.mod'

BRIDGE_TYPE = 'BuchfinkBridge'
BRIDGE_PACKAGE = 'internal/wailsbridge'

MODULE_RE = re.compile(r'^module\s+(\S+)', re.MULTILINE)
SERVICE_RE = re.compile(r"^const SERVICE = '([^']+)';", re.MULTILINE)
INVOKE_RE = re.compile(r"invoke<[^\n]*?>\(\s*'([A-Za-z]\w*)'")
GO_METHOD_RE = re.compile(r'^func \(\w+ \*' + BRIDGE_TYPE + r'\) ([A-Z]\w*)\(', re.MULTILINE)

# Wails behandelt diese Methoden intern und bindet sie nicht ans Frontend.
INTERNAL_METHODS = {'ServiceName', 'ServiceStartup', 'ServiceShutdown', 'ServeHTTP'}


def main() -> int:
    module_path = MODULE_RE.search(GO_MOD.read_text(encoding='utf-8'))
    if module_path is None:
        print(f'{GO_MOD.name}: keine module-Zeile gefunden.')
        return 1
    expected_service = f'{module_path.group(1)}/{BRIDGE_PACKAGE}.{BRIDGE_TYPE}'

    source = BRIDGE_TS.read_text(encoding='utf-8')
    findings: list[str] = []

    service = SERVICE_RE.search(source)
    if service is None:
        findings.append("keine Zeile `const SERVICE = '…';` gefunden")
    elif service.group(1) != expected_service:
        findings.append(
            f'Dienst-Präfix `{service.group(1)}` — erwartet `{expected_service}`.\n'
            '    Wails schlägt gebundene Methoden unter dem vollen Importpfad nach.'
        )

    bound = set()
    for path in sorted(BRIDGE_GO_DIR.glob('*.go')):
        bound.update(GO_METHOD_RE.findall(path.read_text(encoding='utf-8')))
    bound -= INTERNAL_METHODS

    called = sorted(set(INVOKE_RE.findall(source)))
    if not called:
        findings.append('keine invoke-Aufrufe gefunden — prüft der Ausdruck noch das Richtige?')

    for method in called:
        if method not in bound:
            findings.append(f'`{method}` ist keine exportierte Methode auf *{BRIDGE_TYPE}')

    if findings:
        print('Bridge-Bindings passen nicht zur Go-Seite:\n')
        for finding in findings:
            print(f'  {finding}')
        print(f'\n{len(findings)} Befund(e) in {BRIDGE_TS.relative_to(ROOT)}.')
        return 1

    print(f'Bridge-Bindings: {len(called)} Methoden, Präfix und Namen stimmen.')
    return 0


if __name__ == '__main__':
    sys.exit(main())
