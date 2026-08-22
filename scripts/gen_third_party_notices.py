#!/usr/bin/env python3
# SPDX-License-Identifier: EUPL-1.2

"""
Erzeugt THIRD-PARTY-NOTICES.md aus den tatsaechlich mitgelieferten Abhaengigkeiten.

Voraussetzungen:
    go mod download                  (fuellt den Modul-Cache)
    cd frontend && npm install       (fuellt node_modules)
    cd frontend && npm run build     (erzeugt frontend/dist, das main.go einbettet)

Aufruf aus dem Projektwurzelverzeichnis:
    python3 scripts/gen_third_party_notices.py
"""

import glob
import json
import os
import re
import subprocess
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
OUT = os.path.join(ROOT, "THIRD-PARTY-NOTICES.md")
GOOS_TARGETS = ("linux", "darwin", "windows")

# Mitgelieferte Assets, die nicht ueber einen Paketmanager kommen und deshalb
# hier gepflegt werden muessen.
ASSETS = [
    {
        "name": "Manrope (Schriftfamilie)",
        "version": "in frontend/public/Manrope-*.ttf",
        "license": "OFL-1.1",
        "copyright": "Copyright 2019 The Manrope Project Authors "
                     "(https://github.com/sharanda/manrope)",
        "note": "Lizenztext liegt neben den Schriftdateien: "
                "`frontend/public/Manrope-OFL.txt`.",
        "license_file": "frontend/public/Manrope-OFL.txt",
    },
    {
        "name": "Startbildschirm-Foto",
        "version": "frontend/public/bg-startupscreen_unsplash-steven-kamenar.jpg",
        "license": "Unsplash License",
        "copyright": "Steven Kamenar (via Unsplash)",
        "note": "Die Unsplash-Lizenz erlaubt kostenlose kommerzielle Nutzung "
                "und Bearbeitung, untersagt aber den Weitervertrieb als "
                "eigenstaendiges Bildangebot. Sie ist keine Open-Source-Lizenz "
                "und deckt das Foto separat vom uebrigen Werk ab.",
        "license_file": None,
    },
]

LICENSE_PATTERNS = ("LICENSE*", "LICENCE*", "License*", "Licence*",
                    "license*", "licence*", "COPYING*")

# npm-Pakete, die im Abhaengigkeitsbaum stehen, aber nicht im ausgelieferten
# Bundle landen — reine Typdefinitionen und Pakete, die der Code nicht
# importiert. Der Wert ist die Begruendung.
NPM_EXCLUDE = {
    "@types/react": "nur Typdefinitionen, wird wegkompiliert",
    "@types/prop-types": "nur Typdefinitionen, wird wegkompiliert",
    "csstype": "nur Typdefinitionen, wird wegkompiliert",
    "@fontsource/manrope": "nicht importiert; die Schriftdateien liegen unter "
                           "frontend/public/ und stehen bei den Assets",
}

# Pakete ohne eigene Lizenzdatei: Angaben aus package.json bzw. dem Projekt,
# zu dem das Paket gehoert.
NPM_FALLBACK = {
    "@wailsio/runtime": {
        "copyright": "Copyright (c) 2018-Present Lea Anthony",
        "text_from": "github.com/wailsapp/wails/v3",
    },
}


def run(cmd, cwd=None, env=None):
    e = dict(os.environ)
    if env:
        e.update(env)
    r = subprocess.run(cmd, cwd=cwd or ROOT, env=e,
                       capture_output=True, text=True)
    return r.stdout


def escape_module_path(path):
    """Grossbuchstaben werden im Go-Modul-Cache als !kleinbuchstabe abgelegt."""
    return re.sub(r"([A-Z])", lambda m: "!" + m.group(1).lower(), path)


def detect_license(text):
    t = text.lower()
    if "apache license" in t and "version 2.0" in t:
        return "Apache-2.0"
    if "mozilla public license" in t:
        return "MPL-2.0"
    if "sil open font license" in t:
        return "OFL-1.1"
    if "permission to use, copy, modify, and/or distribute" in t:
        return "ISC"
    if "permission is hereby granted, free of charge" in t:
        return "MIT" if "without restriction" in t else "ISC"
    if "redistribution and use in source and binary forms" in t:
        if "neither the name" in t or "names of its contributors" in t:
            return "BSD-3-Clause"
        return "BSD-2-Clause"
    return "unbekannt"


def copyright_lines(text):
    """Nur echte Urheberrechtszeilen — keine Saetze aus dem Lizenzfliesstext."""
    out = []
    for line in text.splitlines():
        s = line.strip().lstrip("*/#").strip()
        if not re.match(r"^(Copyright|COPYRIGHT|©|\(c\))\b", s):
            continue
        # Fliesstext aus dem Lizenzkoerper aussortieren: eine echte
        # Rechtezeile nennt ein Jahr oder traegt ein (c)/©-Zeichen.
        if not re.search(r"(\(c\)|©|\b(19|20)\d{2}\b)", s, re.I):
            continue
        if len(s) > 160:
            continue
        out.append(s)
        if len(out) >= 2:
            break
    return out


def find_license_files(directory):
    files = []
    for pat in LICENSE_PATTERNS:
        files += [f for f in glob.glob(os.path.join(directory, pat))
                  if os.path.isfile(f)]
    return sorted(set(files))


def collect_go():
    fmt = "{{if .Module}}{{.Module.Path}}@{{.Module.Version}}{{end}}"
    mods = set()
    for goos in GOOS_TARGETS:
        out = run(["go", "list", "-deps", "-f", fmt, "./..."],
                  env={"GOOS": goos})
        mods.update(m for m in out.split() if m and "@" in m)
    mods = {m for m in mods if not m.startswith("github.com/buchfink/buchfink")}
    if not mods:
        sys.exit("FEHLER: 'go list -deps' lieferte keine Module. Meist fehlt "
                 "frontend/dist, das main.go per go:embed einbindet — erst "
                 "'cd frontend && npm run build' ausfuehren.")

    cache = run(["go", "env", "GOMODCACHE"]).strip()
    items = []
    for m in sorted(mods):
        path, version = m.rsplit("@", 1)
        directory = os.path.join(cache, escape_module_path(path) + "@" + version)
        files = find_license_files(directory)
        if not files:
            print(f"WARNUNG: keine Lizenzdatei fuer {m}", file=sys.stderr)
            continue
        text = open(files[0], encoding="utf-8", errors="replace").read().strip()
        items.append({
            "name": path,
            "version": version,
            "license": detect_license(text),
            "copyright": " / ".join(copyright_lines(text)) or "—",
            "text": text,
        })
    return items


def collect_npm():
    frontend = os.path.join(ROOT, "frontend")
    out = run(["npm", "ls", "--omit=dev", "--all", "--json"], cwd=frontend)
    try:
        tree = json.loads(out)
    except json.JSONDecodeError:
        print("WARNUNG: npm ls lieferte kein JSON — npm install ausgefuehrt?",
              file=sys.stderr)
        return []

    found = {}

    def walk(node):
        for name, info in (node.get("dependencies") or {}).items():
            version = info.get("version")
            if version:
                found.setdefault(name, version)
            walk(info)

    walk(tree)

    for name, reason in NPM_EXCLUDE.items():
        if found.pop(name, None):
            print(f"Hinweis: {name} ausgelassen ({reason})", file=sys.stderr)

    items = []
    for name in sorted(found):
        directory = os.path.join(frontend, "node_modules", *name.split("/"))
        if not os.path.isdir(directory):
            continue
        pkg = json.load(open(os.path.join(directory, "package.json"),
                             encoding="utf-8"))
        files = find_license_files(directory)
        if files:
            text = open(files[0], encoding="utf-8", errors="replace").read().strip()
            license_id = detect_license(text)
            copyright = " / ".join(copyright_lines(text)) or "—"
        else:
            text = None
            license_id = pkg.get("license") or "unbekannt"
            copyright = NPM_FALLBACK.get(name, {}).get("copyright", "—")
            print(f"Hinweis: keine Lizenzdatei fuer {name}, nutze package.json",
                  file=sys.stderr)
        items.append({
            "name": name,
            "version": found[name],
            "license": license_id,
            "copyright": copyright,
            "text": text,
        })
    return items


def table(items):
    rows = ["| Komponente | Version | Lizenz | Urheberrechtshinweis |",
            "|---|---|---|---|"]
    for i in items:
        c = i["copyright"].replace("|", "\\|")
        rows.append(f"| `{i['name']}` | {i['version']} | {i['license']} | {c} |")
    return "\n".join(rows)


def main():
    go_items = collect_go()
    npm_items = collect_npm()

    # Pakete ohne eigene Lizenzdatei erben den Text des Projekts, zu dem sie
    # gehoeren, damit sie im selben Lizenzblock landen.
    by_name = {i["name"]: i for i in go_items}
    for item in npm_items:
        rule = NPM_FALLBACK.get(item["name"])
        if rule and not item["text"] and rule.get("text_from") in by_name:
            item["text"] = by_name[rule["text_from"]]["text"]

    for asset in ASSETS:
        if asset["license_file"]:
            p = os.path.join(ROOT, asset["license_file"])
            asset["text"] = open(p, encoding="utf-8", errors="replace").read().strip()
        else:
            asset["text"] = None

    # Identische Lizenztexte zusammenfassen.
    blocks = {}
    for i in go_items + npm_items + ASSETS:
        if not i.get("text"):
            continue
        blocks.setdefault(i["text"], []).append(f"{i['name']} {i['version']}")

    parts = [HEADER]
    parts.append("## Go-Module (in die Anwendung kompiliert)\n\n" + table(go_items))
    parts.append("## npm-Pakete (in das Frontend-Bundle kompiliert)\n\n" + table(npm_items))
    parts.append("## Mitgelieferte Assets\n\n" + table(ASSETS) + "\n\n" +
                 "\n".join(f"- **{a['name']}:** {a['note']}" for a in ASSETS))
    parts.append(SKR04_NOTE)
    parts.append(EXTERNAL)

    texts = ["## Lizenztexte\n\nIdentische Lizenztexte sind zusammengefasst; "
             "davor steht jeweils, für welche Komponenten sie gelten."]
    for n, (text, users) in enumerate(sorted(blocks.items(), key=lambda kv: kv[1][0]), 1):
        texts.append(f"### {n}. " + ", ".join(f"`{u}`" for u in sorted(users)) +
                     "\n\n```text\n" + text + "\n```")
    parts.append("\n\n".join(texts))

    open(OUT, "w", encoding="utf-8").write("\n\n---\n\n".join(parts).rstrip() + "\n")
    print(f"{OUT} geschrieben: {len(go_items)} Go-Module, {len(npm_items)} npm-Pakete, "
          f"{len(ASSETS)} Assets, {len(blocks)} Lizenztexte")


HEADER = """<!-- Diese Datei wird erzeugt von scripts/gen_third_party_notices.py.
     Nicht von Hand bearbeiten — stattdessen das Skript anpassen und neu ausführen. -->

# Hinweise zu Drittkomponenten

Buchfink selbst steht unter der [EUPL-1.2](LICENSE). Die hier aufgeführten
Komponenten stammen von Dritten, werden mit der Anwendung ausgeliefert und
bleiben unter ihren eigenen Lizenzbedingungen. Diese Bedingungen gelten
zusätzlich zur EUPL und werden von ihr nicht verdrängt.

Die Lizenzen der Go-Module und npm-Pakete sind durchgehend permissiv (MIT, BSD,
ISC) und damit mit der EUPL vereinbar: Ihre Bedingungen erschöpfen sich darin,
Urheberrechtshinweis und Lizenztext weiterzugeben. Genau dazu dient dieses
Dokument. Für die mitgelieferten Assets gelten engere eigene Bedingungen; sie
stehen beim jeweiligen Eintrag.

Erfasst ist, was tatsächlich ausgeliefert wird — also die in die Binärdatei
kompilierten Go-Module (über die Zielplattformen Linux, macOS und Windows
hinweg), die in das Frontend-Bundle kompilierten npm-Pakete und die
mitgelieferten Assets. Reine Build-Werkzeuge wie Vite, TypeScript, Tailwind CSS
oder das Wails-CLI landen nicht im Auslieferungsstand und sind hier deshalb
nicht gelistet."""

SKR04_NOTE = """## Kontenrahmen SKR 04

Die mitgelieferten Kontendaten (`skr04_2026.json`) lehnen sich an den
DATEV-Standardkontenrahmen SKR 04 in der BilRUG-Fassung 2026 an. „DATEV“ und
„SKR 04“ sind Kennzeichen der DATEV eG. Zwischen Buchfink und der DATEV eG
besteht keine Verbindung; die DATEV eG prüft oder unterstützt dieses Projekt
nicht.

Die zugrunde liegende DATEV-Veröffentlichung wird nicht mit diesem Repository
verteilt. Wer `scripts/build_positions_skr04.py` ausführen möchte, beschafft
sie selbst."""

EXTERNAL = """## Externe Werkzeuge (nicht mitgeliefert)

| Werkzeug | Lizenz | Rolle |
|---|---|---|
| Typst | Apache-2.0 | Layout-Engine für Rechnungs-PDFs. Buchfink erzeugt Typst-Markup; das Binary wird nicht mitgeliefert und derzeit auch nicht aufgerufen. |
| Tailwind CSS | MIT | Build-Werkzeug. Der Quellcode von Tailwind wird nicht ausgeliefert, wohl aber das erzeugte Stylesheet einschließlich der Preflight-Regeln. Der Hinweis auf die MIT-Lizenz steht als Kommentar im erzeugten CSS und bleibt dort erhalten. |

Externe Programme, die lediglich als eigenständiger Prozess aufgerufen werden,
gehen keine Verbindung mit dem Werk im Sinne der EUPL-Copyleft-Klausel ein."""


if __name__ == "__main__":
    main()
