#!/usr/bin/env python3
"""Prüft den Inhalt der Endnutzer-Skills unter skills/.

Aufruf: python3 -I scripts/check_skills.py [skills-Ordner]

Prüft je Skill, dass jede Datei unter references/ aus SKILL.md verlinkt ist.
Prüft in allen Markdown-Dateien: relative Links zeigen auf vorhandene Dateien
und Anker, Referenz-Links sind definiert, Verweise `pflichten.md#<art>`
treffen einen Anker. Prüft die 21 Anker in pflichten.md und meldet verbotene
Wörter. Frontmatter, Zeilenzahl, Plugin und Manifeste prüft
scripts/package_skills.py.
"""
import re
import sys
import unicodedata
from pathlib import Path

ANCHORS = [
    "ustva", "sondervorauszahlung", "zm", "kst-vorauszahlung", "gewst-vorauszahlung",
    "lohnsteuer-anmeldung", "monatsfestschreibung", "bilanzstichtag", "aufstellung",
    "feststellung", "ergebnisverwendung", "ebilanz", "kst-erklaerung", "gewst-erklaerung",
    "ust-erklaerung", "offenlegung", "transparenzregister", "gesellschafterliste",
    "fragebogen-steuerliche-erfassung", "gewerbeanmeldung", "kapitalertragsteuer",
]
FORBIDDEN = [re.compile(r"viking", re.I), re.compile(r"\bpincmd\b"), re.compile(r"myebilanz", re.I)]

INLINE_LINK = re.compile(r"(?<!\!)\[([^\]]+)\]\(([^)\s]+)\)")
REF_USE = re.compile(r"\[([^\]]+)\]\[([^\]]*)\]")
REF_DEF = re.compile(r"^\[([^\]]+)\]:\s*(\S+)", re.M)
PFLICHT_REF = re.compile(r"pflichten\.md#([A-Za-z0-9_-]+)")
CODE_SPAN = re.compile(r"`[^`\n]*`")
FENCE = re.compile(r"^```.*?^```", re.M | re.S)

errors = []
warnings = []


def err(path, msg):
    errors.append(f"FEHLER {path}: {msg}")


def warn(path, msg):
    warnings.append(f"HINWEIS {path}: {msg}")


def slug(heading):
    text = heading.strip().lower()
    text = text.replace("`", "")
    text = re.sub(r"\[([^\]]*)\]\([^)]*\)", r"\1", text)
    out = []
    for ch in text:
        if ch in " -":
            out.append("-" if ch == " " else ch)
        elif unicodedata.category(ch)[0] in ("L", "N") or ch == "_":
            out.append(ch)
    return "".join(out)


def anchors_of(path, cache={}):
    if path in cache:
        return cache[path]
    text = path.read_text(encoding="utf-8")
    text_nf = FENCE.sub("", text)
    found = {}
    for m in re.finditer(r'<a\s+id="([^"]+)"\s*>\s*</a>', CODE_SPAN.sub("", text_nf)):
        found[m.group(1)] = found.get(m.group(1), 0) + 1
    seen = {}
    for m in re.finditer(r"^#{1,6}\s+(.+?)\s*#*\s*$", text_nf, re.M):
        s = slug(m.group(1))
        n = seen.get(s, 0)
        seen[s] = n + 1
        key = s if n == 0 else f"{s}-{n}"
        found.setdefault(key, 0)
        found[key] += 1
    cache[path] = found
    return found


def check_links(path, root):
    text = path.read_text(encoding="utf-8")
    body = FENCE.sub("", text)
    body_nocode = CODE_SPAN.sub("", body)
    defs = {m.group(1).lower(): m.group(2) for m in REF_DEF.finditer(body_nocode)}
    targets = []
    for m in INLINE_LINK.finditer(body_nocode):
        targets.append(m.group(2))
    for m in REF_USE.finditer(body_nocode):
        label = (m.group(2) or m.group(1)).lower()
        if label not in defs:
            err(path.relative_to(root), f"Referenz-Link ohne Definition: [{m.group(1)}][{m.group(2)}]")
        else:
            targets.append(defs[label])
    used = {(m.group(2) or m.group(1)).lower() for m in REF_USE.finditer(body_nocode)}
    for label in defs:
        if label not in used:
            warn(path.relative_to(root), f"Referenz-Definition ungenutzt: [{label}]")
    for t in targets:
        if re.match(r"^[a-z][a-z0-9+.-]*:", t, re.I):
            continue
        file_part, _, frag = t.partition("#")
        target = path if not file_part else (path.parent / file_part).resolve()
        if not target.exists():
            err(path.relative_to(root), f"toter Link: {t}")
            continue
        if frag and target.suffix == ".md":
            if frag not in anchors_of(target):
                err(path.relative_to(root), f"Anker fehlt: {t}")
    # Verweise auf Pflichten-Anker im Fließtext oder in Codespans
    pf = root / "buchfink-grundlagen" / "references" / "pflichten.md"
    if pf.exists():
        anchors = anchors_of(pf)
        for m in PFLICHT_REF.finditer(body):
            if m.group(1) not in anchors:
                err(path.relative_to(root), f"pflichten.md-Anker fehlt: #{m.group(1)}")
    for rx in FORBIDDEN:
        lines = sorted({text.count("\n", 0, m.start()) + 1 for m in rx.finditer(text)})
        if lines:
            err(path.relative_to(root), f"verbotenes Wort /{rx.pattern}/ in Zeilen {', '.join(map(str, lines))}")


def check_skill(d, root):
    skill = d / "SKILL.md"
    if not skill.exists():
        err(d.relative_to(root), "SKILL.md fehlt")
        return
    rel = skill.relative_to(root)
    text = skill.read_text(encoding="utf-8")
    refs = d / "references"
    if refs.is_dir():
        for f in sorted(refs.glob("*.md")):
            if f"references/{f.name}" not in text:
                err(rel, f"references/{f.name} ist aus SKILL.md nicht verlinkt")
        for f in sorted(refs.iterdir()):
            if f.suffix != ".md":
                warn(f.relative_to(root), "Datei ohne .md in references/")


def check_pflichten(root):
    pf = root / "buchfink-grundlagen" / "references" / "pflichten.md"
    if not pf.exists():
        err("buchfink-grundlagen/references", "pflichten.md fehlt")
        return
    text = pf.read_text(encoding="utf-8")
    ids = re.findall(r'<a\s+id="([^"]+)"\s*>\s*</a>', CODE_SPAN.sub("", FENCE.sub("", text)))
    for a in ANCHORS:
        n = ids.count(a)
        if n != 1:
            err(pf.relative_to(root), f"Anker {a} {n}-mal (erwartet 1)")
    for a in set(ids) - set(ANCHORS):
        err(pf.relative_to(root), f"unerwarteter Anker {a}")


def main():
    default = Path(__file__).resolve().parent.parent / "skills"
    root = (Path(sys.argv[1]) if len(sys.argv) > 1 else default).resolve()
    skills = sorted(p for p in root.iterdir() if p.is_dir() and not p.name.startswith("."))
    for d in skills:
        check_skill(d, root)
    for f in sorted(root.rglob("*.md")):
        check_links(f, root)
    check_pflichten(root)
    for w in warnings:
        print(w)
    for e in errors:
        print(e)
    print(f"{len(skills)} Skills, {len(errors)} Fehler, {len(warnings)} Hinweise")
    sys.exit(1 if errors else 0)


if __name__ == "__main__":
    main()
