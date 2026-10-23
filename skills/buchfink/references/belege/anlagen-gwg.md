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

- Linear über die betriebsgewöhnliche Nutzungsdauer; im Anschaffungsjahr je
  vollem Monat vor dem Anschaffungsmonat ein Zwölftel weniger
  ([§ 7 Abs. 1 Satz 1, 2, 4 EStG][estg7]). Geschäfts- oder Firmenwert:
  15 Jahre (Satz 3).
- Degressiv für bewegliche Wirtschaftsgüter, angeschafft nach dem 30.06.2025
  und vor dem 01.01.2028: höchstens das Dreifache des linearen Satzes und 30 %
  vom Restwert ([§ 7 Abs. 2 EStG][estg7]). Stand: Oktober 2026. Die Wahl trifft der Mensch.
- Computerhardware samt Peripherie und Software: Nutzungsdauer ein Jahr
  zulässig; das Gut kommt ins Bestandsverzeichnis, die volle Abschreibung im
  Anschaffungsjahr wird nicht beanstandet
  ([BMF vom 22.02.2022, Rz. 1 bis 1.4][bmfnd]; Geräte in Rz. 2 bis 5).
  Handelsrechtlich ist die kurze Dauer eine Wertungsfrage.
- Andere Güter: Erfahrungswerte in den AfA-Tabellen des BMF, etwa für
  allgemein verwendbare Anlagegüter (BMF vom 15.12.2000, BStBl I S. 1532; hier
  nicht am Wortlaut gelesen). Die Nutzungsdauer entscheidet der Mensch.

Konten mit Vorschlag von Buchfink: 0135 EDV-Software und 0690 EDV-Hardware
(12 Monate, Abweichung begründen), 0650 Büroeinrichtung (156 Monate), 0520 Pkw
(72 Monate), 0670 GWG (Sofortabzug über 6260), 0675 Sammelposten (Fünftel über
6264), 0700 Anzahlungen und Anlagen im Bau (keine Abschreibung bis zur
Fertigstellung).

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

[estg6]: https://www.gesetze-im-internet.de/estg/__6.html
[estg7]: https://www.gesetze-im-internet.de/estg/__7.html
[estg9b]: https://www.gesetze-im-internet.de/estg/__9b.html
[hgb253]: https://www.gesetze-im-internet.de/hgb/__253.html
[hgb255]: https://www.gesetze-im-internet.de/hgb/__255.html
[bmfnd]: https://www.steuerschroeder.de/blog/wp-content/uploads/2022/02/Nutzungsdauer-von-Computerhardware-und-Software-zur-Dateneingabe-und-verarbeitung.pdf
