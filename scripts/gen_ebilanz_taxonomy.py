#!/usr/bin/env python3
"""
Erzeugt den eingebetteten Auszug der HGB-/E-Bilanz-Taxonomie unter
internal/ebilanz/taxonomy/ (de-6.9.json, de-6.10.json).

Eingabe sind die Elementlisten, die aus dem amtlichen XBRL-Paket der
Finanzverwaltung (esteuer.de, Rubrik Taxonomien) extrahiert wurden: je Version
eine JSON-Datei mit den Listen deGaapCi und deGcd (Name, Typ, Periode, Saldo,
Mussfeld-Kennzeichen, Rechtsform, GKV/UKV, notPermittedFor, Gültigkeit,
Presentation- und Calculation-Kanten, Tupel- und Auswahlstruktur). Herkunft und
Prüfsumme des Pakets stehen in deren meta.source und gehen in den Auszug über.

Der Auszug enthält nur, was Buchfink für die E-Bilanz einer GmbH oder UG
braucht: Bilanz, GuV nach dem Gesamtkostenverfahren (auch MicroBilG),
Ergebnisverwendung, steuerliche Überleitung, steuerlichen
Betriebsvermögensvergleich, Kontennachweis und die Stammdaten (GCD), die
Pflicht sind oder geschrieben werden. Weggelassen werden Positionen, die für
Kapitalgesellschaften nicht zulässig sind, solche des Umsatzkostenverfahrens,
solche mit „notPermittedFor = Einreichung an Finanzverwaltung" und solche, deren
ValidThrough vor dem 01.01.2025 liegt.

Aufruf aus dem Projektwurzelverzeichnis:
    python3 -I scripts/gen_ebilanz_taxonomy.py elements-6.9.json elements-6.10.json

Die Ausgabe ist bei gleicher Eingabe Byte für Byte gleich.
"""

import json
import os
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
OUT_DIR = os.path.join(ROOT, "internal", "ebilanz", "taxonomy")

COPYRIGHT = "(c) XBRL Deutschland e.V. - see http://www.xbrl.de"
NOTICE = ("Abgeleiteter Auszug, bearbeitet: maschinell gekürzt auf die Positionen, "
          "die Buchfink für die E-Bilanz einer Kapitalgesellschaft braucht. Authentisch "
          "ist nur das Taxonomiepaket auf www.xbrl.de bzw. de.xbrl.org.")

GAAP_AREAS = {
    "bilanz", "guv", "guv_microbilg", "ergebnisverwendung", "ueberleitung",
    "kontennachweis", "betriebsvermoegensvergleich",
}
ROLES = {
    "balanceSheet", "incomeStatement", "incomeStatementMicroBilG",
    "appropriationProfits", "transfersCommercialCodeToTax", "transfersTaxAssets",
    "detailedInformation",
}
# Stammdaten, die Buchfink schreibt, obwohl sie kein Mussfeld sind.
GCD_EXTRA = {"genInfo.company.id.sizeClass"}
# Frühester Bilanzstichtag, für den Buchfink eine E-Bilanz erzeugt: die
# Taxonomie 6.9 gilt für Wirtschaftsjahre, die nach dem 31.12.2024 beginnen.
EARLIEST = "2025-01-01"
NOT_FOR_SUBMISSION = "Einreichung an Finanzverwaltung"


def first(values):
    if not values:
        return ""
    return min(values) if isinstance(values, list) else values


def short_type(t):
    if not t:
        return ""
    return t[:-len("ItemType")] if t.endswith("ItemType") else t


def requirement(fiscal):
    code = (fiscal or {}).get("requirementCode")
    return {"MUSSFELD": "M", "SUMMENMUSSFELD": "S", "RECHNERISCH_NOTWENDIG": "R"}.get(code, "")


def forms(fiscal):
    allowed = (fiscal or {}).get("allowedFor") or {}
    return [f for f in ("EU", "KSt", "PG") if allowed.get(f, True)]


def slim(e, ns):
    fiscal = e.get("fiscal") or {}
    hgb = e.get("hgb") or {}
    out = {"name": e["name"], "ns": ns, "kind": e["kind"]}
    if e.get("type"):
        out["type"] = short_type(e["type"])
    if e.get("periodType"):
        out["period"] = e["periodType"]
    if e.get("balance"):
        out["balance"] = e["balance"]
    if e.get("abstract"):
        out["abstract"] = True
    if requirement(fiscal):
        out["req"] = requirement(fiscal)
    f = forms(fiscal)
    if f != ["EU", "KSt", "PG"]:
        out["forms"] = f
    if fiscal.get("typeOperatingResult") in ("GKV", "UKV"):
        out["gkvUkv"] = fiscal["typeOperatingResult"]
    np = sorted(x for x in (fiscal.get("notPermittedFor") or []) if x != NOT_FOR_SUBMISSION)
    if np:
        out["notPermittedFor"] = np
    for key, value in (("validSince", first(hgb.get("ValidSince"))),
                       ("validThrough", first(hgb.get("ValidThrough"))),
                       ("fiscalValidSince", first(fiscal.get("fiscalValidSince"))),
                       ("fiscalValidThrough", first(fiscal.get("fiscalValidThrough")))):
        if value:
            out[key] = value
    out["label"] = e.get("labelDe") or ""
    if e.get("variants"):
        out["shells"] = sorted(e["variants"])
    # Kanten als [Rolle, Oberposition, Reihenfolge bzw. Gewicht]. Die Rolle
    # bestimmt das Einstiegsschema; ein eigenes Feld dafür wäre doppelt.
    pres = sorted([p["role"], p["parent"], p["order"]]
                  for p in e.get("presentation") or [] if p["role"] in ROLES)
    if pres:
        out["pres"] = pres
    calc = sorted([c["role"], c["parent"], c["weight"]]
                  for c in e.get("calcParents") or [] if c["role"] in ROLES)
    if calc:
        out["calc"] = calc
    tup = e.get("tuple")
    if tup:
        out["tuple"] = {"model": tup["model"], "children": [
            {"ref": c["ref"], "min": int(c.get("minOccurs") or 1),
             "max": -1 if c.get("maxOccurs") == "unbounded" else int(c.get("maxOccurs") or 1)}
            for c in tup["children"]]}
    if e.get("choiceHead"):
        out["choiceHead"] = e["choiceHead"]
    if e.get("choiceAlternatives"):
        out["alternatives"] = list(e["choiceAlternatives"])
    return out


def keep_gaap(e):
    fiscal = e.get("fiscal") or {}
    if e.get("area") not in GAAP_AREAS:
        return False
    if not (fiscal.get("allowedFor") or {}).get("KSt", True):
        return False
    if fiscal.get("typeOperatingResult") == "UKV":
        return False
    if NOT_FOR_SUBMISSION in (fiscal.get("notPermittedFor") or []):
        return False
    if fiscal.get("onlyPermittedForSoBil_ErgBil"):
        return False
    through = first((e.get("hgb") or {}).get("ValidThrough"))
    if through and through < EARLIEST:
        return False
    return True


def closure(start, by_name):
    """Tupel-Kinder, Auswahlköpfe und Auswahlwerte gehören zum Element."""
    seen, stack = set(), list(start)
    while stack:
        name = stack.pop()
        if name in seen or name not in by_name:
            continue
        seen.add(name)
        e = by_name[name]
        for c in (e.get("tuple") or {}).get("children") or []:
            stack.append(c["ref"])
        stack.extend(e.get("choiceAlternatives") or [])
    return seen


def excerpt(src):
    data = json.load(open(src, encoding="utf-8"))
    meta = data["meta"]
    gaap_by_name = {e["name"]: e for e in data["deGaapCi"]}
    gcd_by_name = {e["name"]: e for e in data["deGcd"]}

    gaap = {e["name"] for e in data["deGaapCi"] if keep_gaap(e) and e.get("submittable")}
    gaap = {n for n in closure(gaap, gaap_by_name) if keep_gaap(gaap_by_name[n])}

    gcd_start = {e["name"] for e in data["deGcd"]
                 if (e.get("fiscal") or {}).get("requirementCode") == "MUSSFELD"}
    gcd_start |= GCD_EXTRA
    gcd = {n for n in closure(gcd_start, gcd_by_name)
           if NOT_FOR_SUBMISSION not in ((gcd_by_name[n].get("fiscal") or {}).get("notPermittedFor") or [])}

    elements = [slim(gaap_by_name[n], "de-gaap-ci") for n in sorted(gaap)]
    elements += [slim(gcd_by_name[n], "de-gcd") for n in sorted(gcd)]
    names = {e["name"] for e in elements}
    for e in elements:
        if "alternatives" in e:
            e["alternatives"] = [a for a in e["alternatives"] if a in names]
        if "tuple" in e:
            e["tuple"]["children"] = [c for c in e["tuple"]["children"] if c["ref"] in names]

    head = {
        "version": meta["taxonomyVersion"],
        "release": meta["release"],
        "copyright": COPYRIGHT,
        "notice": NOTICE,
        "source": meta["source"],
        "namespaces": meta["namespaces"],
        "entrySchemas": {
            "gcd": meta["entrySchemas"]["gcd"],
            "fiscal": meta["entrySchemas"]["gaapFiscal"],
            "microbilg": meta["entrySchemas"]["gaapFiscalMicroBilG"],
        },
    }
    lines = ["{"]
    for key, value in head.items():
        lines.append(f"  {json.dumps(key)}: {json.dumps(value, ensure_ascii=False, sort_keys=True)},")
    lines.append('  "elements": [')
    rows = [json.dumps(e, ensure_ascii=False, separators=(",", ":")) for e in elements]
    lines.append(",\n".join("    " + r for r in rows))
    lines.append("  ]")
    lines.append("}")
    return meta["taxonomyVersion"], "\n".join(lines) + "\n", len(elements)


def main(argv):
    if len(argv) < 2:
        sys.exit("Aufruf: gen_ebilanz_taxonomy.py <elements-6.9.json> [<elements-6.10.json> …]")
    os.makedirs(OUT_DIR, exist_ok=True)
    for src in argv[1:]:
        version, text, count = excerpt(src)
        path = os.path.join(OUT_DIR, f"de-{version}.json")
        with open(path, "w", encoding="utf-8", newline="\n") as f:
            f.write(text)
        print(f"{path}: {count} Elemente, {len(text.encode('utf-8'))} Bytes")


if __name__ == "__main__":
    main(sys.argv)
