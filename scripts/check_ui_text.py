#!/usr/bin/env python3
"""Prüft die Sprache der Arbeitsansichten (Architektur 6.4).

Die Regel: eine Arbeitsansicht spricht die Sprache des Vorgangs. Ein Titel
„Belege buchen" sagt, was zu tun ist; ein Titel „§ 146 Abs. 1 AO" sagt, warum es
zu tun ist — und das gehört in die Detaildialog, nicht auf den Knopf. Wer
keine Buchhaltung gelernt hat, liest einen Paragraphen als Fehlermeldung.

Die Heuristik ist bewusst grob: jede Zeile in frontend/src/pages, die ein „§" in
einer sichtbaren Zeichenkette trägt, braucht eine Help-Komponente im Umkreis von
drei Zeilen. Sie findet nicht jeden Verstoß und meldet gelegentlich einen, der
keiner ist — aber sie hält die Norm dort, wo sie erklärt wird, und nicht dort,
wo gearbeitet wird.

Kommentarzeilen bleiben außen vor — auch im Umkreis: sie erklären den Code und
stehen in keiner Ansicht, und ein Kommentar mit dem Wort „Erklärzeichen" neben
einer sichtbaren Norm wäre sonst der Weg an der Prüfung vorbei.

Aufrufbar über `task check:text`; Rückgabewert 1 bei einem Verstoß, mit Datei
und Zeile je Fund.
"""

from __future__ import annotations

import sys
from pathlib import Path

# Der Umkreis, in dem die Erklärkomponente stehen muss.
CONTEXT_LINES = 3

# Die Marken, die eine Erklärstufe kennzeichnen: die Namen der
# Erklärkomponenten (Help, InfoPopover, FormExplanation) und die
# Bezeichner, die eine Erklärung aufmachen. `explain=` ist das Prop an Feld,
# Abschnitt und Seitenkopf, das einen Help rendert (siehe
# frontend/src/components/ui/Field.tsx und Section.tsx); `explanation` ist das
# Feld `explanation:` der Erklärtabellen und die Funktion `explanations()`, die
# sie aufbaut. `help=` steht weiterhin dabei: einzelne Ansichten reichen eine
# Erklärung unter diesem Namen an eine eigene Komponente weiter. Der Vergleich
# läuft ohne Rücksicht auf Groß- und Kleinschreibung.
#
# Bewusst keine deutschen Wortfragmente: „erklär" träfe auch sichtbaren Text —
# ein Absatz mit dem Wort „erklärt" schaltete dann eine Norm daneben frei,
# obwohl er selbst in der Arbeitsansicht steht. Geprüft wird deshalb nur, was im
# Code eine Erklärstufe *ist*, nicht was von ihr spricht; die Marken sind
# englische Bezeichner und stehen in keinem sichtbaren Text dieser Oberfläche.
#
# `hint=` steht ebenfalls nicht dabei: der Hinweis ist unmittelbar sichtbar und
# damit Arbeitsansicht, nicht Erklärstufe.
HELP_MARKERS = (
    "<help ",
    "<help>",
    "dialog",
    "infopopover",
    "formexplanation",
    "help=",
    "explain",
    "explanation",
)

# Eigenschaften, die unmittelbar sichtbaren Text tragen: der Hinweis unter einem
# Feld und der Kontext unter einer Abschnittsüberschrift. Steht in einer solchen
# Zeile eine Norm, ist sie ein Verstoß — unabhängig davon, ob nebenan ein
# Erklärzeichen sitzt. Sonst genügte ein `help=` drei Zeilen weiter, um einen
# Paragraphen in der Arbeitsansicht stehen zu lassen.
VISIBLE_PROPS = ("hint=", "context=")

PAGES = Path("frontend/src/pages")


def strip_comments(lines: list[str]) -> list[str]:
    """Ersetzt Kommentare durch Leerzeichen, Zeile für Zeile.

    Ein Paragraph in einem Kommentar erklärt den Code und steht in keiner
    Oberfläche. Der Block-Kommentar wird über die Zeilen hinweg verfolgt, damit
    ein mehrzeiliger Hinweis nicht halb geprüft wird.
    """
    out: list[str] = []
    in_block = False
    for line in lines:
        result = []
        i = 0
        while i < len(line):
            if in_block:
                end = line.find("*/", i)
                if end < 0:
                    result.append(" " * (len(line) - i))
                    i = len(line)
                else:
                    result.append(" " * (end + 2 - i))
                    i = end + 2
                    in_block = False
                continue
            if line.startswith("/*", i):
                in_block = True
                result.append("  ")
                i += 2
                continue
            if line.startswith("//", i):
                result.append(" " * (len(line) - i))
                i = len(line)
                continue
            result.append(line[i])
            i += 1
        out.append("".join(result))
    return out


def indent_of(line: str) -> int:
    return len(line) - len(line.lstrip())


def opening_tag(lines: list[str], index: int) -> str:
    """Der ganze Kopf des Elements, das in dieser Zeile beginnt oder endet.

    Ein JSX-Kopf steht selten auf einer Zeile: der Name oben, die Eigenschaften
    darunter, das schließende „>" wieder auf der Höhe des Namens. Wer nur die
    Zeile mit dem „>" liest, sieht den Namen der Komponente nicht — und damit
    nicht, ob hier eine Erklärstufe aufgeht. Deshalb wird von ihr aus nach oben
    bis zum „<" gelesen und der Kopf als ein Text zurückgegeben.
    """
    text = lines[index]
    if not text.lstrip().startswith("<"):
        for i in range(index - 1, -1, -1):
            if not lines[i].strip():
                continue
            text = lines[i] + " " + text
            if lines[i].lstrip().startswith("<"):
                break
            if indent_of(lines[i]) < indent_of(lines[index]):
                break
    return text


def in_explanation_block(lines: list[str], index: int) -> bool:
    """Meldet, ob die Zeile in einem Erklärblock steht.

    Der Umkreis von drei Zeilen reicht für ein Feld mit `help=`, nicht für einen
    langen Text hinter dem Erklärzeichen: dort steht die Marke am Kopf des
    Blocks und der Paragraph vierzig Zeilen darunter. Deshalb wird zusätzlich
    nach außen gelesen — Zeile für Zeile zurück, und jede Zeile, die den Block
    öffnet (geringere Einrückung als alles bisher Gesehene), wird auf die Marke
    geprüft; bei einem JSX-Element samt seinem mehrzeiligen Kopf. Trägt einer
    davon die Marke, ist der Paragraph in der Erklärstufe.
    """
    limit = indent_of(lines[index])
    for i in range(index - 1, -1, -1):
        line = lines[i]
        if not line.strip():
            continue
        current = indent_of(line)
        if current >= limit:
            continue
        limit = current
        if any(marker in opening_tag(lines, i).lower() for marker in HELP_MARKERS):
            return True
        if current == 0:
            break
    return False


def violations(path: Path) -> list[tuple[int, str]]:
    lines = path.read_text(encoding="utf-8").splitlines()
    code = strip_comments(lines)
    found: list[tuple[int, str]] = []
    for index, line in enumerate(code):
        if "§" not in line:
            continue
        if any(prop in line for prop in VISIBLE_PROPS):
            found.append((index + 1, lines[index].strip()))
            continue
        # Geprüft wird gegen den kommentarfreien Text und nicht gegen die
        # Originalzeilen: ein Kommentar, der zufällig eine Erklärkomponente
        # nennt, steht in keiner Ansicht — er dürfte die Prüfung für eine
        # sichtbare Norm daneben nicht freischalten.
        start = max(0, index - CONTEXT_LINES)
        end = min(len(code), index + CONTEXT_LINES + 1)
        window = "\n".join(code[start:end]).lower()
        if any(marker in window for marker in HELP_MARKERS):
            continue
        if in_explanation_block(code, index):
            continue
        found.append((index + 1, lines[index].strip()))
    return found


def main() -> int:
    if not PAGES.is_dir():
        print(f"{PAGES} nicht gefunden — das Skript läuft im Wurzelverzeichnis des Projekts.")
        return 2

    total = 0
    for path in sorted(PAGES.rglob("*.tsx")) + sorted(PAGES.rglob("*.ts")):
        for line_number, text in violations(path):
            total += 1
            print(f"{path}:{line_number}: Norm ohne Erklärstufe: {text}")

    if total:
        print()
        print(f"{total} Zeile(n) nennen eine Norm ohne Erklärstufe: weder eine Marke "
              f"({', '.join(HELP_MARKERS)}) im Umkreis von {CONTEXT_LINES} Zeilen "
              "noch ein Erklärblock darüber — oder die Norm steht in einer "
              f"unmittelbar sichtbaren Eigenschaft ({', '.join(VISIBLE_PROPS)}).")
        print("Arbeitsansichten sprechen die Sprache des Vorgangs; die Norm gehört in die "
              "Detaildialog (Architektur 6.4).")
        return 1

    print("Sprache der Arbeitsansichten: keine Norm außerhalb der Erklärstufen.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
