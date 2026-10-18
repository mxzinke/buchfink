#!/usr/bin/env python3
"""Prüft, dass `frontend/src/services/bridge.ts` zur Go-Bridge passt.

Das Frontend ruft die Bridge namensbasiert über die Wails-Laufzeit auf, ohne
generierte Bindings. Damit ist `bridge.ts` die einzige Stelle, an der die Namen
stehen — und nichts hat bisher geprüft, ob sie stimmen. Ein falscher Name fällt
erst zur Laufzeit auf, als „Binding call failed: unknown bound method name".

Drei Regeln:

1. Der Dienst-Präfix muss der voll qualifizierte Go-Name sein, also
   `<Modulpfad aus go.mod>/internal/wailsbridge.BuchfinkBridge`. Wails legt
   gebundene Methoden unter genau diesem Schlüssel ab; der kurze Paketname
   findet nichts.
2. Jeder aufgerufene Methodenname muss eine exportierte Methode auf
   `*BuchfinkBridge` sein.
3. Keine Methode ohne Aufrufer. Wails bindet jede exportierte Methode, und
   jede ist aus der WebView aufrufbar, ob eine Ansicht sie braucht oder nicht.
   Eine Go-Methode ohne Eintrag in `bridge.ts`, ein Eintrag ohne Aufruf in
   `api.ts` und eine Methode von `Api` ohne Aufruf aus einer Ansicht sind
   deshalb Befunde: Was die Oberfläche nicht tut, erledigt der Agent über den
   Kern, und ein vergessener Schreibweg in der Bridge wäre eine Hintertür daran
   vorbei. Ausgenommen ist eine Methode, die ein Test der Bridge aufruft: Der
   Test hält Verhalten fest, das die Methode selbst trägt, etwa die Methoden-
   liste der Anlagenmaske oder die Antwort im Prüfermodus.
"""

from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
BRIDGE_TS = ROOT / 'frontend' / 'src' / 'services' / 'bridge.ts'
API_TS = ROOT / 'frontend' / 'src' / 'services' / 'api.ts'
FRONTEND_SRC = ROOT / 'frontend' / 'src'
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


def tested_methods() -> set[str]:
    """Die Methoden, die ein Test in internal/wailsbridge über die Bridge aufruft."""
    tested: set[str] = set()
    for path in BRIDGE_GO_DIR.glob('*_test.go'):
        tested.update(re.findall(r'\bb\.([A-Z]\w*)\(', path.read_text(encoding='utf-8')))
    return tested


def unused_methods(bridge_source: str, bound: set[str], called: set[str]) -> list[str]:
    """Regel 3: jede gebundene Methode hat einen Aufrufer in einer Ansicht oder einem Test."""
    findings: list[str] = []
    for method in sorted(bound - called - tested_methods()):
        findings.append(
            f'`{method}` ist gebunden, steht aber nicht in bridge.ts — '
            'aus der WebView aufrufbar, ohne dass eine Ansicht sie braucht'
        )

    api = API_TS.read_text(encoding='utf-8')
    api = api[api.index('export const Api = {'):]
    entries = list(re.finditer(r'^  ([a-z]\w*)\s*:', api, re.MULTILINE))
    bridge_by_api: dict[str, set[str]] = {}
    for index, entry in enumerate(entries):
        end = entries[index + 1].start() if index + 1 < len(entries) else len(api)
        bridge_by_api[entry.group(1)] = set(re.findall(r'Bridge\.([A-Z]\w*)', api[entry.start():end]))

    bridge_keys = set(re.findall(r'^  ([A-Z]\w*):', bridge_source, re.MULTILINE))
    used_by_api = set().union(*bridge_by_api.values()) if bridge_by_api else set()
    for key in sorted(bridge_keys - used_by_api):
        findings.append(f'`bridge.{key}` wird in api.ts nicht aufgerufen')

    used_api: set[str] = set(re.findall(r'\b(?:Api|this)\.([a-z]\w*)', api))
    for path in FRONTEND_SRC.rglob('*.ts*'):
        if 'services' in path.parts:
            continue
        used_api.update(re.findall(r'\bApi\.([a-z]\w*)', path.read_text(encoding='utf-8')))
    for name in sorted(set(bridge_by_api) - used_api):
        findings.append(f'`Api.{name}` hat keinen Aufrufer in einer Ansicht')
    return findings


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
        if path.name.endswith('_test.go'):
            continue
        bound.update(GO_METHOD_RE.findall(path.read_text(encoding='utf-8')))
    bound -= INTERNAL_METHODS

    called = sorted(set(INVOKE_RE.findall(source)))
    if not called:
        findings.append('keine invoke-Aufrufe gefunden — prüft der Ausdruck noch das Richtige?')

    for method in called:
        if method not in bound:
            findings.append(f'`{method}` ist keine exportierte Methode auf *{BRIDGE_TYPE}')

    findings.extend(unused_methods(source, bound, set(called)))

    if findings:
        print('Bridge-Bindings passen nicht zur Go-Seite oder haben keinen Aufrufer:\n')
        for finding in findings:
            print(f'  {finding}')
        print(f'\n{len(findings)} Befund(e) in {BRIDGE_TS.relative_to(ROOT)}.')
        return 1

    only_tested = sorted(bound - set(called))
    print(
        f'Bridge-Bindings: {len(called)} Methoden, Präfix und Namen stimmen, jede hat einen Aufrufer'
        f'; {len(only_tested)} weitere nur aus Tests der Bridge.'
    )
    return 0


if __name__ == '__main__':
    sys.exit(main())
