# Anschaffungen: Aufwand, GWG, Sammelposten, Anlagegut

Rechtsstand: 17. Oktober 2026. Wortlaut von EStG und HGB am 17.10.2026 gelesen.
Für frühere Jahre gelten die Wertgrenzen des Anschaffungsjahres.

Hier geht es um den Zugang; Abschreibungslauf, Anlagenspiegel und Abgang:
[jahresabschluss/abschreibungen.md](../jahresabschluss/abschreibungen.md).

## Inhalt

1. Aufwand oder Anschaffung
2. Anschaffungskosten
3. Wertgrenzen
4. Nutzungsdauer und Methode
5. Weg in Buchfink

## 1. Aufwand oder Anschaffung

- Abnutzbare Wirtschaftsgüter mit Nutzung über mehr als ein Jahr werden über
  die Nutzungsdauer abgeschrieben ([§ 7 Abs. 1 Satz 1 EStG][estg7];
  [§ 253 Abs. 3 Satz 1 HGB][hgb253]). Aufwand sind Verbrauchsmaterial, Wartung,
  Reparatur, Abonnements und laufende Nutzungsentgelte (Software als Dienst).
- Erweiterung oder wesentliche Verbesserung über den ursprünglichen Zustand
  hinaus ist Herstellungsaufwand ([§ 255 Abs. 2 Satz 1 HGB][hgb255]).
- Wertungsfrage: Reparatur oder Verbesserung, Lizenzkauf oder Abo, ein
  Wirtschaftsgut oder mehrere. Nenne die Folge (sofort Aufwand oder Verteilung
  über Jahre); der Mensch entscheidet.

## 2. Anschaffungskosten

Aufwendungen für Erwerb und Betriebsbereitschaft samt Nebenkosten (Versand,
Montage) und nachträglichen Kosten, abzüglich Preisminderungen
([§ 255 Abs. 1 HGB][hgb255]), ohne abziehbare Vorsteuer
([§ 9b Abs. 1 EStG][estg9b]). Die Wertgrenzen gelten deshalb netto.

## 3. Wertgrenzen

Stand: Oktober 2026 ([§ 6 Abs. 2, 2a EStG][estg6]); für abnutzbare,
bewegliche, selbständig nutzbare Wirtschaftsgüter des Anlagevermögens.

| Anschaffungskosten netto je Wirtschaftsgut | Weg A: GWG-Sofortabzug | Weg B: Sammelposten |
|---|---|---|
| bis 250 € | sofort Aufwand | sofort Aufwand (Abs. 2a Satz 4) |
| über 250 € bis 800 € | sofort Aufwand; Verzeichnis nötig, wenn die Angaben nicht aus der Buchführung ersichtlich sind (Abs. 2 Satz 4, 5) | Sammelposten |
| über 800 € bis 1.000 € | Anlagegut | Sammelposten |
| über 1.000 € | Anlagegut | Anlagegut |

- Weg A oder B gilt für alle Zugänge eines Wirtschaftsjahres (Abs. 2a Satz 5).
  Frag, welcher Weg gilt; das Kontoblatt 0675 (`auswertung`,
  `art: kontoblatt`, `konto: 0675`) zeigt einen Sammelposten des Jahres.
- Der Sammelposten wird im Bildungsjahr und den vier Folgejahren zu je einem
  Fünftel aufgelöst; ein Abgang mindert ihn nicht (Abs. 2a Satz 2, 3).
- Nur zusammen mit anderen, technisch abgestimmten Anlagegütern nutzbar heißt
  nicht selbständig nutzbar; dann zählt der Wert des Ganzen (Abs. 2 Satz 2, 3).
  Ob das zutrifft (Zubehör, Peripherie, Bauteile), ist eine Wertungsfrage.

## 4. Nutzungsdauer und Methode

Nutzungsdauer und Methode (linear oder, für Zugänge 2025 bis 2027, degressiv)
wählt der Mensch. Vorschlag: linear nach AfA-Tabelle; bei Computerhardware und
Software ein Jahr. Sätze, Quellen und die Konten, für die Buchfink eine
Nutzungsdauer vorschlägt: [anlagen-nutzungsdauer.md](anlagen-nutzungsdauer.md).

## 5. Weg in Buchfink

1. Kläre mit dem Menschen: Aufwand oder Anschaffung, Anschaffungskosten
   netto, Weg A oder B, bei Anlagegut Konto, Nutzungsdauer und Methode.
2. **Aufwand oder GWG:** `beleg_buchen_mit_rueckfrage` mit der passenden
   Gruppe, für GWG `gwg` (6260). Gegenstand, Anschaffungstag und
   Anschaffungskosten gehören in den Positionstext; dann sind die Angaben aus
   der Buchführung ersichtlich.
3. **Sammelposten oder Anlagegut:** `beleg_buchen_mit_rueckfrage` mit dem
   Anlagekonto als `konto` der Position (weitere mit `konten`, `art: suche`).
   Die Buchung erscheint in `anlagen` als Aktivierungskandidat. Lege
   Bezeichnung, Klasse, Methode, Nutzungsdauer und Tag der Inbetriebnahme vor
   und ruf nach dem Ja `abschlussbuchung` mit `art: anlage_aktivieren` auf.
4. Liegt die Vorsteuer über der Grenze des § 15a UStG
   ([vorsteuer.md](vorsteuer.md#6-berichtigung-nach--15a-ustg)), bitte den
   Menschen, den Eintrag im Verzeichnis der Vorsteuerberichtigung zu prüfen.
5. Anzahlung auf eine noch nicht gelieferte Anlage:
   `anzahlung_fuer: sachanlage` oder `immateriell`. Die Kennungen für
   `verrechnete_anzahlungen` der späteren Rechnung liefert kein Werkzeug
   ([grundlagen/arbeitsweise.md, Abschnitt 9](../grundlagen/arbeitsweise.md#9-vorgänge-ohne-werkzeug));
   eine Umbuchung der Anzahlung legst du als `buchung_erfassen` vor.

## Stammdaten pflegen

Beschreibende Anlagenstammdaten ändern Sie unter Bücher → Anlagen → Stammdaten
ändern mit Pflichtgrund. Das Änderungsprotokoll enthält Vorher und Nachher.
Kosten, Methode, Nutzungsdauer und Zugangstag bleiben unverändert. Die
betriebliche Nutzungsquote ist eine beschreibende Angabe ohne automatische
Steuerkorrektur. Stückzahlen von Finanzanlagen lassen sich nur bei einem
ungebuchten Zugang ohne Folgebewegungen und in offener Periode korrigieren.
Eine Rücknahme erfolgt als weitere begründete Änderung. `anlagen` liest den
aktuellen Stand; Änderungen sind der App vorbehalten.

[estg6]: https://www.gesetze-im-internet.de/estg/__6.html
[estg9b]: https://www.gesetze-im-internet.de/estg/__9b.html
[hgb253]: https://www.gesetze-im-internet.de/hgb/__253.html
[hgb255]: https://www.gesetze-im-internet.de/hgb/__255.html
[estg7]: https://www.gesetze-im-internet.de/estg/__7.html

## Übertragung an Gesellschafter

Unentgeltliche oder verbilligte Übertragungen haben keinen eigenen Abgangsweg.
Lege den Fall dem Menschen zur Beurteilung vor; prüfe mit
[Verdeckte Gewinnausschüttung und Einlage](../gesellschafter/vga-einlage.md)
die Folgen nach [§ 8 Abs. 3 Satz 2 KStG](https://www.gesetze-im-internet.de/kstg_1977/__8.html)
und [§ 3 Abs. 1b UStG](https://www.gesetze-im-internet.de/ustg_1980/__3.html).
Eine Abgangsart „Entnahme“ gibt es nicht. Ein gewöhnlicher Verkauf an einen
Gesellschafter wird mit Rechnung erfasst.
