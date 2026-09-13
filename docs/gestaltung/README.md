# Design-Konzept

[Dokumentation](../README.md) · [Entwicklung](../entwicklung/README.md)

Die visuelle und interaktive Grundlage von Buchfink. Dieses Dokument ist die
Referenz für Code-Reviews. Eine Ansicht, die gegen die Regeln hier verstößt, ist
ein Fehler und keine Geschmacksfrage.

Zwei Stellen setzen es um:

- [`frontend/src/index.css`](../../frontend/src/index.css) hält die Tokens als
  Tailwind-`@theme`. Nutzbar als Utility (`bg-paper`, `text-ink`) und als
  CSS-Variable (`var(--color-ink)`).
- [`frontend/src/components/ui/`](../../frontend/src/components/ui) hält die
  Bausteine. Wo dieses Dokument einen Baustein beschreibt, steht dort die
  verbindliche Umsetzung. Klassenketten gehören in die Komponente, nicht in
  dieses Dokument und nicht in eine Seite.

---

## Themen

| Thema | Inhalt |
|---|---|
| [Visuelle Grundlagen](grundlagen.md) | Leitidee, Farben, Typografie, Flächen, Bewegung und Dunkelmodus |
| [Interaktion](interaktion.md) | Eingaben, Rückmeldungen, Icons, Zustände und Barrierefreiheit |
| [Komponenten und Layout](komponenten.md) | UI-Bausteine, fachliche Muster und Seitenaufbau |
| [Hilfetexte](hilfetexte.md) | Textbudget, Fragezeichen und Detaildialog |
| [Schreibweise](schreibweise.md) | Sprache für Oberfläche, Webseite und Dokumentation |

Die Abschnittsnummern bleiben als Verweise zwischen den Themenseiten erhalten.

## Prüfliste für jeden Pull Request

- [ ] Keine neue Fläche außerhalb der fünf Fälle aus [Abschnitt 6.2](grundlagen.md#62-wann-eine-fläche-erlaubt-ist), keine Fläche in einer Fläche
- [ ] Abschnitte durch Überschrift, Abstand und Haarlinie getrennt, nicht durch Kästen
- [ ] Tabellenüberschrift über der Fläche, Kopfzeile ohne Füllung
- [ ] Kein Fließtext in der Ansicht, Erklärungen hinter dem Erklärzeichen ([Abschnitt 15](hilfetexte.md#15-text))
- [ ] Textbudget je Ort eingehalten ([Abschnitt 15.1](hilfetexte.md#151-textbudget))
- [ ] Keine Hex-Farben, keine Klassen aus den alten Paletten
- [ ] `-text` für Text, Basis für Marker, `-soft` mit `-line` für Flächen
- [ ] Schriftgrößen aus der Skala in [Abschnitt 4](grundlagen.md#4-typografie)
- [ ] Radien nur `control`, `overlay`, `full`
- [ ] Schatten nur an Popover und Dialog
- [ ] Jede Zahlenspalte hat `.num` und ist rechtsbündig
- [ ] Nur Bewegungen aus der Tabelle in [Abschnitt 7](grundlagen.md#7-bewegung)
- [ ] Farbige Zustände haben zusätzlich Text oder Icon
- [ ] Leer-, Lade- und Fehlerzustand vorhanden
- [ ] Mit der Tastatur bedienbar, Fokus sichtbar, Fokusrückgabe geregelt
- [ ] Beträge und Daten über `utils/formatters.ts`
- [ ] `task check:design` läuft durch
