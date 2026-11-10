# Umstieg aus einem Vorsystem

Rechtsstand: 10. November 2026.

## Inhalt

1. Wann es ein Umstieg ist
2. Voraussetzungen
3. Unterlagen
4. Eröffnung buchen
5. Abstimmen
6. Nach der Eröffnung
7. Grenzen

## 1. Wann es ein Umstieg ist

Eine bestehende GmbH oder UG führt ihre Bücher bisher in einem anderen
Programm und ab dem Beginn eines Geschäftsjahres in Buchfink. Eine
Eröffnungsbilanz verlangt [§ 242 Abs. 1 HGB][hgb242] zu Beginn des
Handelsgewerbes; beim Umstieg entsteht keine neue. Die Bücher laufen weiter
([§ 238 Abs. 1 HGB][hgb238]), und die Wertansätze zu Beginn des
Geschäftsjahres müssen mit denen der Schlussbilanz des Vorjahres
übereinstimmen ([§ 252 Abs. 1 Nr. 1 HGB][hgb252]). Du übernimmst deshalb die
Schlussbilanz, nichts anderes.

Eine in Buchfink gegründete Gesellschaft ist kein Umsteiger:
[eroeffnungsbilanz.md](eroeffnungsbilanz.md).

## 2. Voraussetzungen

- Der Mensch hat den Umstieg in der App festgelegt: bei der Einrichtung oder
  unter Unternehmen › Stammdaten, Block „Umstieg“. Den Stand liest du mit
  `auswertung` (`art: umstieg`). Ohne festgelegten Umstieg lehnt
  `eroeffnung_buchen` ab; bitte den Menschen darum.
- Der Tag ist der erste Tag eines Geschäftsjahres. Einen Umstieg im laufenden
  Jahr bildet Buchfink nicht ab (Abschnitt 7).
- Das Vorjahr ist im Vorsystem abgeschlossen: Abschreibungen, Abgrenzungen und
  Rückstellungen sind gebucht. Ist die Schlussbilanz vorläufig, sag das dem
  Menschen; ändert sie sich, wird die Eröffnung storniert und neu gebucht
  (Abschnitt 6).

## 3. Unterlagen

Bitte den Menschen um vier Unterlagen zum letzten Tag des Vorjahres:

1. Schlussbilanz oder Summen- und Saldenliste. Lege sie mit `beleg_ablegen`
   (`art: sonstiges`) ab; ihre `beleg_id` belegt jede Eröffnungsbuchung.
2. Offene-Posten-Liste der Kunden und der Lieferanten mit Rechnungsnummer,
   Rechnungsdatum, Fälligkeit, offenem Betrag und Steuersatz.
3. Anlagenverzeichnis mit Anschaffungsdatum, historischen Anschaffungskosten,
   Nutzungsdauer, Abschreibungsverfahren und bisher gebuchter Abschreibung je
   Anlagegut.
4. Bei einem anderen Kontenrahmen als SKR04 die Zuordnung der Konten. Suche
   die Konten mit `konten` und lege die Zuordnung dem Menschen vor.

Fehlende Kontakte legst du mit `kontakt_anlegen` an.

## 4. Eröffnung buchen

`eroeffnung_buchen` zuerst mit `nur_vorschau`, dem Menschen vorlegen, nach
seinem Ja buchen.

- `konten`: jedes Bilanzkonto der Schlussbilanz mit Seite und Betrag. Die
  Gegenbuchung auf 9000, 9008 und 9009 bildet Buchfink. Aufwands- und
  Ertragskonten gehören nicht hinein. Das Jahresergebnis des Vorjahres steht
  wie beim Saldenvortrag auf 2970 (Gewinnvortrag) oder 2978 (Verlustvortrag).
- Sammelkonten 1200 und 3300: ihr Bestand ist die Kontrollsumme der offenen
  Posten. Gebucht wird je Rechnung auf das Personenkonto.
- `offene_posten`: je offene Rechnung eine Zeile. `satz` ist der im Vorsystem
  erklärte Steuersatz (`0` für nicht steuerbar). Ohne ihn lässt sich der
  Posten später voll bezahlen, aber weder mit Skonto ausgleichen noch
  ausbuchen. Die Steuer bucht Buchfink nicht noch einmal.
- `anlagen`: je Anlagegut `anschaffungskosten` (historisch) und `afa_bisher`.
  Die Differenz ist der Buchwert; je Anlagekonto ergeben die Buchwerte den
  Bestand in `konten`. `nutzungsdauer_monate` ist die gesamte Nutzungsdauer ab
  Anschaffung. Buchfink verteilt den Buchwert auf die Restnutzungsdauer.
- Rückstellungen und Rechnungsabgrenzung übernimmst du als Bestand ihres
  Kontos.

Nennt die Vorschau `differenzen`, bucht Buchfink nicht: Aktiva und Passiva
gehen auseinander, die offenen Posten ergeben nicht ihr Sammelkonto, oder die
Anlagen ergeben nicht ihr Anlagekonto. Suche die Ursache in den Unterlagen;
gleiche nie mit einem erfundenen Betrag aus.

## 5. Abstimmen

`auswertung` (`art: umstieg`) stellt je Konto die Schlussbilanz dem gebuchten
Bestand gegenüber und die Nebenbücher ihren Sachkonten. Fertig, wenn
`abgestimmt` wahr ist und der Mensch die Abstimmung gesehen hat. Solange die
Eröffnung fehlt oder eine Differenz besteht, meldet der Prüflauf einen
sperrenden Befund, und der erste Monat lässt sich nicht festschreiben.

## 6. Nach der Eröffnung

- Zahlungen auf übernommene Posten: [bank/zuordnung.md](../bank/zuordnung.md).
- Abschreibung: der AfA-Lauf des ersten Jahres rechnet ab dem übernommenen
  Buchwert ([jahresabschluss/ablauf.md](../jahresabschluss/ablauf.md)).
- Fehler in der Eröffnung: alle Eröffnungsbuchungen mit `buchung_stornieren`
  stornieren, dann `eroeffnung_buchen` mit derselben `beleg_id`. Die
  übernommenen Anlagen bleiben im Verzeichnis; `anlagen` entfällt beim zweiten
  Mal.
- Das Vorsystem und seine Daten bleiben aufzubewahren
  ([grundlagen/aufbewahrung.md, Abschnitt 7](../grundlagen/aufbewahrung.md#7-systemwechsel),
  [§ 147 Abs. 6 Satz 5 AO][ao147]). Den Tag der Umstellung hinterlegt der
  Mensch in den Einstellungen; er steht dann in der Verfahrensdokumentation.

## 7. Grenzen

Sag dem Menschen vor der Übernahme, was Buchfink nicht übernimmt:

- Umstieg im laufenden Geschäftsjahr: Salden der Aufwands- und Ertragskonten
  und bereits abgegebene Voranmeldungen. Ausweg: das Jahr im Vorsystem
  beenden oder die Vorgänge des Jahres in Buchfink nachbuchen.
- Vorjahreszahlen: Bilanz und GuV des ersten Jahres zeigen kein Vorjahr.
  Anzugeben sind sie trotzdem ([§ 265 Abs. 2 Satz 1 HGB][hgb265]); der Mensch
  ergänzt sie aus dem Abschluss des Vorjahres.
- Steuerliche Verlustvorträge: die Eröffnung kennt sie nicht. Du gibst sie
  bei der Steuerrückstellung an
  ([jahresabschluss/steuern-im-abschluss.md](../jahresabschluss/steuern-im-abschluss.md)).
- Anlagen: sofort abgezogene geringwertige Wirtschaftsgüter,
  Sonderabschreibung nach § 7g EStG, Stückzahlen von Wertpapieren und das
  Verzeichnis nach § 15a UStG aus dem Vorsystem.
- Rückstellungen stehen nur als Kontobestand, nicht im Rückstellungsspiegel;
  Verbrauch und Auflösung buchst du mit `buchung_erfassen`.
- Offene Posten in Fremdwährung und erhaltene oder geleistete Anzahlungen.

[ao147]: https://www.gesetze-im-internet.de/ao_1977/__147.html
[hgb238]: https://www.gesetze-im-internet.de/hgb/__238.html
[hgb242]: https://www.gesetze-im-internet.de/hgb/__242.html
[hgb252]: https://www.gesetze-im-internet.de/hgb/__252.html
[hgb265]: https://www.gesetze-im-internet.de/hgb/__265.html
