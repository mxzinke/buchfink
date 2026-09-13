# Komponenten und Layout

[Designkonzept](README.md) · [Dokumentation](../README.md)

## 10. Komponenten

Die Bausteine liegen in [`frontend/src/components/ui/`](../../frontend/src/components/ui).
Dort steht die verbindliche Umsetzung, hier stehen nur die Regeln, die man dem
Code nicht ansieht. Klassenketten werden nicht in Seiten geschrieben und nicht in
diesem Dokument gepflegt.

### 10.1 Verhalten kommt von Base UI

Alles, was Fokus fängt, positioniert oder auf Tasten hört, kommt aus
[`@base-ui/react`](https://base-ui.com). Selbst gebaut waren diese Teile
fehleranfällig, und die Fehler zeigen sich erst spät: ein Tooltip, der am
Bildschirmrand hinausläuft, ein Dialog, aus dem die Tabulatortaste
herausspringt, ein Menü ohne Typeahead.

Base UI liefert nur Verhalten und keine Gestalt. Die Gestalt kommt von hier, in
Tailwind-Klassen aus den Tokens in [Abschnitt 3](grundlagen.md#3-farbe). Zustände richten sich nach den
Datenattributen der Bibliothek (`data-[open]`, `data-[highlighted]`, `data-[invalid]`), das
bleibt also Tailwind ohne Zwischenschicht.

Rein darstellende Bausteine bauen wir selbst, weil es dort nichts falsch zu
machen gibt.

### 10.2 Was es gibt

| Baustein | Datei | Verhalten von |
|---|---|---|
| `Button` | `Button.tsx` | eigen |
| `Input`, `Textarea` | `Input.tsx` | Base UI (Field-Anbindung) |
| `AmountInput` | `Input.tsx` | Base UI NumberField |
| `Select` | `Select.tsx` | Base UI Select |
| `Combobox` | `Combobox.tsx` | Base UI Combobox |
| `Checkbox`, `RadioGroup`, `Switch` | `Toggle.tsx` | Base UI |
| `Field`, `FieldRow`, `FormGrid` | `Field.tsx` | Base UI Field |
| `Dialog`, `ConfirmDialog` | `Dialog.tsx` | Base UI Dialog, AlertDialog |
| `Menu` und Einträge | `Menu.tsx` | Base UI Menu |
| `Help`, `InfoPopover` | `Help.tsx` | Base UI Tooltip, Dialog und Popover |
| `Tabs`, `TabPanel`, `Separator` | `Tabs.tsx` | Base UI |
| `Progress`, `Skeleton`, `SkeletonRows`, `toast` | `Feedback.tsx` | Base UI Progress, Sonner |
| `FileDrop` | `FileDrop.tsx` | eigen |
| `Section`, `PageHeader` | `Section.tsx` | eigen |
| `StatRow`, `Stat` | `StatRow.tsx` | eigen |
| `Table` und Zellen | `Table.tsx` | eigen |
| `StatusBadge` | `StatusBadge.tsx` | eigen |
| `EmptyState` | `EmptyState.tsx` | eigen |
| `cn` | `cn.ts` | Klassen zusammenführen, letzte Angabe gewinnt |
| `SHELL_PANEL`, `SHELL_BUTTON`, `SHELL_CONTROL` | `shell.ts` | Klassenbündel für die Vollbild-Schirme ([Abschnitt 12](#12-layout)) |

Bewusst gibt es keine `Card`. Wo eine Fläche nötig ist ([Abschnitt 6.2](grundlagen.md#62-wann-eine-fläche-erlaubt-ist)), steht sie an genau
dieser Stelle im Code und nicht als Baustein, der sich unbemerkt vermehrt.

### 10.3 Abdeckung

Abgeglichen mit dem, was die zwölf Seiten heute benutzen:

| Bedarf | Heute im Code | Baustein |
|---|---|---|
| Textfeld | 20 mal | `Input` |
| Auswahlliste | 19 mal | `Select` |
| Datum | 12 mal | `Input type="date"` |
| Dialog | 8 mal | `Dialog`, `ConfirmDialog` |
| Aufklappmenü | 9 mal | `Menu` |
| Tabelle | 8 mal | `Table` |
| Rückmeldung | 26 mal | `toast` |
| Kästchen, Radio | 5 mal | `Checkbox`, `RadioGroup` |
| Zahl, Betrag | 2 mal | `AmountInput` |
| Mehrzeilig | 1 mal | `Textarea` |
| Datei | 1 mal | `FileDrop` |
| Kontosuche | fehlte | `Combobox` |
| Schalter | fehlte | `Switch` |
| Reiter | fehlte | `Tabs` |
| Ladezustand | fehlte | `SkeletonRows` |
| Fortschritt | fehlte | `Progress` |

Ein echter Datumswähler mit Kalender fehlt noch. Bis dahin bleibt es beim
nativen Feld, das im Desktop-WebView brauchbar ist.

### 10.4 Regeln, die der Code nicht zeigt

**Buttons.** Genau eine Primäraktion pro Ansicht, in Tinte. Ein deaktivierter
Button braucht eine Erklärung im `title`, sonst versteckt er seinen Grund. Beim
Laden bleibt er an seiner Stelle, behält die Breite und tauscht die Beschriftung
nicht aus.

**Felder.** Pflichtfelder haben kein Sternchen. Gekennzeichnet wird das
Seltenere: optional. Der Fehlertext ersetzt den Hinweis, solange er steht.

**Auswahl gegen Suche.** `Select` für kurze feste Listen, `Combobox` für alles,
was man suchen muss. Der SKR04-Kontenrahmen ist immer eine Suche.

**Bestätigung.** `ConfirmDialog` nur für unumkehrbare Schritte ([Abschnitt 8.2](interaktion.md#82-rückgängig-statt-rückfrage)). Er lässt
sich nicht durch einen Klick daneben schließen. Was rückgängig gemacht werden
kann, läuft ohne Rückfrage und bekommt `toast.undo`.

**Tabelle.** Die Fläche ersetzt nicht die Kopflinie: Der Kopf bleibt ohne
Füllung. Die Überschrift des Abschnitts steht über der Fläche, nicht darin,
sonst entsteht wieder eine Karte mit Kopfzeile. Keine Zebrastreifen.

**Status-Badge.** Enthält immer Text, nie nur den Marker. Der Marker ist eine
Raute, 7 mal 7 px, in der Basisfarbe der Familie. Der Kreis ist die weichste
Form, die es gibt, und in einer Statusspalte beliebig; die Raute hat vier
definierte Kanten und eine Achse.

Sie entsteht über `clip-path`, nicht über `rotate(45deg)`. Eine gedrehte Fläche
behält ihre ursprüngliche Layoutbox: Bei 5 mal 5 px malt sie 7,07 px, die
Spitzen ragen über die Box hinaus und verkürzen den Abstand zum Text auf 5 px,
obwohl 6 px gesetzt sind. Die Geometrie liegt als Utility `mark-diamond` in
`index.css`. Dieselbe Raute markiert den Integritätszustand im Fuß der
Navigation ([Abschnitt 11.4](#114-integrität-der-hash-chain)), andere Statuspunkte gibt es nicht.

**Meldungen.**

| Art | Umsetzung |
|---|---|
| Erfolg einer Aktion | `toast.success`, 4 s |
| Umkehrbarer Schritt | `toast.undo`, 8 s |
| Fachlicher Fehler im Formular | am Feld, nie als Toast |
| Fehler aus dem Backend | Hinweisfläche in Rosé über den Aktionen |
| Dauerhafter Zustand | Statusleiste, kein Toast |

---

## 11. Fachliche Muster

Der Teil, der Buchfink von einer beliebigen Anwendung unterscheidet.

### 11.1 Soll und Haben

Zwei getrennte, rechtsbündige Spalten mit `.num`, nicht eine Spalte mit
Vorzeichen. Soll links, Haben rechts, wie im Buch. Beide farblich neutral. Unter
der letzten Position steht die Summe mit der buchhalterischen Doppellinie
(`rule-total`). Soll- und Habensumme müssen sichtbar gleich sein, das ist die
Kontrolle, die Buchhalter zuerst suchen.

Kontonummern in `.code-num`, dahinter die Bezeichnung in `text-ink-muted`. Die
Nummer allein hilft niemandem, der SKR04 nicht auswendig kann.

### 11.2 Storno und Generalumkehr

Stornierte Buchungen werden nie durchgestrichen und nie ausgeblendet. Der
ursprüngliche Betrag muss lesbar bleiben.

```
Zeile:  border-l-2 border-negative bg-negative-soft/50
Badge:  Storniert, auf der Gegenbuchung: Storno zu RE-2024-014
Betrag: normale Darstellung, kein line-through
```

Stornobuchung und Original verlinken sich gegenseitig, in beide Richtungen.

### 11.3 Status-Vokabular

Ein Wort pro Zustand, in der ganzen Anwendung dasselbe. Die Liste ist
abschließend.

| Status | Familie | Gilt für |
|---|---|---|
| Entwurf | neutral | Rechnung, Buchung vor dem Festschreiben |
| Offen | Bernstein | Rechnung, Offener Posten, Beleg ohne Buchung |
| Teilweise ausgeglichen | Bernstein, nur Rand | Offener Posten |
| Zugeordnet | neutral | Banktransaktion mit Beleg |
| Gebucht | Salbei | Buchung im Journal |
| Ausgeglichen | Salbei | Offener Posten |
| Festgeschrieben | Salbei mit Schloss-Icon | Buchung, Periode |
| Überfällig | Rosé | Rechnung, Steuerfrist |
| Storniert | Rosé | Buchung, Rechnung |
| Fehlerhaft | Rosé | Import, Validierung, Integritätsprüfung |
| Aufgestellt | neutral | Jahresabschluss |
| Festgestellt | Salbei mit Schloss-Icon | Jahresabschluss, Geschäftsjahr |
| Offengelegt | Salbei mit Schloss-Icon | Jahresabschluss, Geschäftsjahr |

Synonyme wie erledigt, fertig oder abgeschlossen für denselben Zustand sind nicht
erlaubt.

Die drei letzten Stände beschreiben nicht dasselbe wie Festgeschrieben:
festgeschrieben ist der Zeitraum, festgestellt ist der Abschluss, und beschlossen
haben ihn die Gesellschafter. Aufgestellt bleibt neutral, weil noch nichts
beschlossen und nichts gesperrt ist. Ab Festgestellt zeigt das Abzeichen das
Schloss, denn ab dort nimmt das Geschäftsjahr keine Buchung mehr an.

### 11.4 Integrität der Hash-Chain

Der Zustand steht dauerhaft im Fuß der Navigation, nie als Toast.

| Zustand | Darstellung |
|---|---|
| geprüft | Raute in Salbei, "Daten unverändert", darunter der Zeitpunkt |
| wird geprüft | rotierendes Icon in `ink-subtle`, "Prüfung läuft" |
| gebrochen | Schild in Rosé, "Integrität verletzt" |

Ein Integritätsbruch ist der einzige Fall, in dem die Oberfläche laut werden darf:
ein Balken in `negative-soft` über dem gesamten Inhalt, bis er quittiert ist.

Die Zeile ist zugleich der Weg in die Prüfübersicht — Ketten, Belegdateien,
Festschreibungen, Prüfläufe, Änderungsprotokoll, Systemhistorie —, die deshalb
keinen eigenen Eintrag in der Navigation hat: die Frage „ist etwas nicht in
Ordnung" wird hier beantwortet, und wer sie genauer wissen will, klickt auf die
Antwort. Nachgeprüft wird auf der Seite selbst; ein Klick, der beides täte,
machte aus einem Blick auf den Zustand einen Lauf über die ganze Buchführung.

### 11.5 Gesperrte Perioden

Abgeschlossene Geschäftsjahre sind schreibgeschützt. Sichtbar durch `sunken`
statt `paper` als Grund, ein Schloss neben der Jahreszahl in der Kopfzeile und
einen Hinweisstreifen: "Geschäftsjahr 2024 ist abgeschlossen. Buchungen sind nur
im laufenden Jahr möglich."

Ein festgestelltes Geschäftsjahr wird wie eine gesperrte Periode behandelt:
Schloss neben der Jahreszahl, Hinweisstreifen, Bearbeitungsaktionen deaktiviert.
Der Hinweisstreifen nennt dabei den Weg zurück, weil es ihn gibt: "Geschäftsjahr
2024 ist festgestellt. Buchungen nimmt es erst wieder an, wenn die Feststellung
mit Grund zurückgesetzt wird."

Bearbeitungsaktionen werden deaktiviert und behalten ihre Position, damit die
Ansicht zwischen den Jahren gleich aussieht.

### 11.6 Belege

Zwei Spalten: links das Dokument, rechts die Erfassung. Die Belegvorschau ist der
eine Ort, an dem eine Fläche richtig ist ([Abschnitt 6.2](grundlagen.md#62-wann-eine-fläche-erlaubt-ist), Fall 2): `rounded-card border
border-line bg-surface`. Die Erfassung daneben bekommt keine.

Die Vorschau bleibt sichtbar, während gebucht wird. Der Abgleich zwischen Beleg
und Feld ist die häufigste Tätigkeit und darf keinen Fensterwechsel kosten.

Werte aus der E-Rechnung erscheinen als Vorschlag im Feld, mit dem Hinweis "aus
ZUGFeRD übernommen". Sie sind editierbar und werden nicht als gesichert
dargestellt.

---

## 12. Layout

```
┌──────────────┬─────────────────────────────────────────────┐
│  Mandant ⌄   │  Kopfzeile 56 px, Geschäftsjahr             │
│  Navigation  ├─────────────────────────────────────────────┤
│  240 px      │  Seitentitel + Primäraktion                 │
│  dunkel      │  ─────────────────────────────────────────  │
│              │  Kennzahlen und Filter auf dem Papier,      │
│  ──────────  │  die Tabelle in ihrer eigenen Fläche        │
│  Integrität  │                                             │
└──────────────┴─────────────────────────────────────────────┘
```

**Navigation.** 240 px, fünf Gruppen: Übersicht, Buchhaltung, Stammdaten,
Auswertungen, Verwaltung. Aktiver Eintrag `bg-shell-raised text-white` mit 2 px
Leiste in `accent-light` links. Kein farbiger Hintergrund, ein dauerhaft
sichtbarer Zustand darf nicht laut sein.

**Mandantenwahl.** Sie steht am Kopf der Navigation, dort wo der Name des
Mandanten steht: ein Auslöser, der die Liste aufklappt, mit dem Weg zu einem
neuen Mandanten und zur Übersicht darunter. Ein Wechsel ist eine Auswahl und
keine Reise — bis Welle 9 führte er über den Startbildschirm und zurück.

**Zeigefinger.** Was man drücken kann, zeigt ihn. Tailwind setzt seit Fassung 4
`cursor: default` auf `button`; die Regel steht deshalb einmal in der
Basisschicht von `index.css` und nicht in jedem Baustein. Gesperrte Knöpfe
behalten `cursor-not-allowed`, Einträge in Menü und Auswahlliste `cursor-default`
— sie sind Listenzeilen, keine Knöpfe.

**Seitenkopf.** Titel in `text-display`, darunter eine Zeile Kontext in
`text-caption text-ink-subtle`, rechts die Primäraktion, darunter eine Haarlinie
über die volle Breite. Kein Icon neben dem Titel, kein Kasten um den Kopf.

**Vollbild-Schirme.** Start, Einrichtung und Wiederherstellung stehen vor dem
Arbeitsbereich und laufen ganz auf der dunklen Schale, mit denselben
`shell`-Tokens wie die Navigation. Dort gibt es kein Papier, also drehen sich
die Rollen um: Die Primäraktion hat die hellste Fläche (`bg-paper
text-shell-deep`), nicht die dunkelste. Die Klassenbündel dafür stehen in
`components/ui/shell.ts` — `SHELL_PANEL`, `SHELL_BUTTON`, `SHELL_CONTROL`.
Eigene Bausteine bekommen diese Schirme nicht: Auf jedem stehen ein bis zwei
Bedienelemente, ein zweiter Bausteinsatz würde mehr kosten, als er einbringt.

Ein Vollbild-Schirm steht *vor* dem Arbeitsbereich und trägt deshalb keine
Navigation daneben. Der Startschirm beantwortet genau eine Frage — mit welchem
Mandanten wird gearbeitet —, und die Zeile des Mandanten ist die Antwort: ein
Klick öffnet ihn. Sprungmarken in einzelne Ansichten stehen dort nicht; sie
wären eine zweite Navigation vor der ersten.

**Schmale Fenster.** Buchfink ist eine Desktop-Anwendung. Telefone sind kein
Ziel, und es wird nichts dafür gebaut. Die Oberfläche muss lediglich ein kleines
Fenster überstehen: Unter 768 px wird die Navigation zur Schublade, Tabellen
scrollen seitlich in ihrer eigenen Fläche. Eigene Listenlayouts für kleine
Breiten gibt es nicht.

---
