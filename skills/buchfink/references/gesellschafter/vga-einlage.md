# Verdeckte Gewinnausschüttung und Einlage

Rechtsstand: 17. Oktober 2026. KStR 2022 gelesen über einen
[nichtamtlichen Spiegel][kstr85]; das amtliche KSt-Handbuch des BMF war nicht
abrufbar.

## Inhalt

1. Was eine vGA ist
2. Folgen
3. Prüfliste typischer Fälle
4. Wie du mit einem Verdacht umgehst
5. Offene und verdeckte Einlage
6. Steuerliches Einlagekonto

## 1. Was eine vGA ist

- Eine vGA ist eine Vermögensminderung oder verhinderte Vermögensmehrung, die
  durch das Gesellschaftsverhältnis veranlasst ist, sich auf den Gewinn auswirkt
  und nicht auf einem ordnungsgemäßen Gewinnverteilungsbeschluss beruht
  ([R 8.5 Abs. 1 Satz 1 KStR 2022][kstr85]).
- Veranlasst ist sie auch, wenn der Vorteil einer dem Gesellschafter
  nahestehenden Person zufließt ([R 8.5 Abs. 1 Satz 3 KStR 2022][kstr85]).
- Beim beherrschenden Gesellschafter reicht in der Regel schon das Fehlen
  einer zivilrechtlich wirksamen, klaren, im Voraus geschlossenen Vereinbarung
  oder deren Nichtdurchführung ([R 8.5 Abs. 2 KStR 2022][kstr85]).
- Maßstab ist der Fremdvergleich: Hätte die GmbH einem fremden Dritten dasselbe
  gewährt? Das ist immer eine Wertungsfrage.

## 2. Folgen

| Ebene | Folge | Grundlage |
|---|---|---|
| GmbH, Körperschaftsteuer | Die vGA mindert das Einkommen nicht; die Korrektur gehört in die Körperschaftsteuererklärung | [§ 8 Abs. 3 Satz 2 KStG][kstg8] |
| GmbH, Kapitalertragsteuer | Die vGA ist Kapitalertrag; die GmbH hat Kapitalertragsteuer einzubehalten, anzumelden und haftet | [§ 20 Abs. 1 Nr. 1 Satz 2][estg20], [§ 43 Abs. 1 Satz 1 Nr. 1][estg43], [§ 44 Abs. 5 EStG][estg44] |
| Gesellschafter | Einkünfte aus Kapitalvermögen | [§ 20 Abs. 1 Nr. 1 Satz 2 EStG][estg20] |
| Geschäftsführer | Haftung für nicht abgeführte Steuern bei grober Fahrlässigkeit | [§ 69 AO][ao69] |

Die Zeile zur vGA in der Körperschaftsteuererklärung verantwortet der Mensch:
Buchfink meldet `vga_pruefen`, wenn auf typischen Konten für Leistungen an
Gesellschafter gebucht ist, setzt den Betrag aber nicht selbst. Den Weg in die
Erklärung regelt [steuererklaerungen/kst.md](../steuererklaerungen/kst.md).

## 3. Prüfliste typischer Fälle

Leg jeden Treffer als Wertungsfrage vor.

| Fall | Worauf du achtest |
|---|---|
| Private Ausgaben über die GmbH (Reise, Möbel, Handy der Familie) | Rückzahlung vereinbart? Sonst vGA-Verdacht |
| Geschäftsführergehalt, Tantieme, Pensionszusage | Vertrag im Voraus, Höhe im Fremdvergleich, Gehaltsänderung vor dem Zeitraum beschlossen |
| Firmenwagen privat genutzt | Regelung im Anstellungsvertrag? Mit Regelung ist der Vorteil Arbeitslohn ([lohn/ablauf.md](../lohn/ablauf.md)), ohne Regelung vGA-Verdacht |
| Darlehen an den Gesellschafter | Zins, Sicherheit, Rückzahlung ([bank/darlehen.md](../bank/darlehen.md#9-gmbh-leiht-dem-gesellschafter)) |
| Darlehen, Miete, Lizenz vom Gesellschafter | Entgelt über Marktniveau |
| Leistungen der GmbH an den Gesellschafter | Preis unter dem, was Fremde zahlen |
| Rückwirkende Vereinbarung | beim beherrschenden Gesellschafter in der Regel schädlich ([R 8.5 Abs. 2 KStR 2022][kstr85]) |
| Vereinbarung, die nicht gelebt wird | z. B. Gehalt, das nie gezahlt wird; Nichtdurchführung ist schädlich ([R 8.5 Abs. 2 KStR 2022][kstr85]) |

Die Umsatzsteuer folgt eigenen Regeln: Vorsteuer aus privaten Ausgaben ist nicht
abziehbar ([§ 15 Abs. 1 Satz 1 Nr. 1 UStG][ustg15]). Überlässt die GmbH einem
Gesellschafter-Geschäftsführer einen Firmenwagen auch privat, ist das in der
Regel eine entgeltliche Leistung an ihn als Arbeitnehmer
([UStAE 15.23 Abs. 8 Satz 1, 4][ustae1523]); das bucht das Lohnjournal
([lohn/ablauf.md](../lohn/ablauf.md)).

## 4. Wie du mit einem Verdacht umgehst

1. Halte vor dem Buchen an.
2. Lege vor: Sachverhalt, warum er verdächtig ist, Fundstelle, Optionen
   (Rückforderung als Forderung 1307, Vertrag nachweisen, als vGA behandeln mit
   Kapitalertragsteuer).
3. Der Mensch entscheidet, bei Zweifel mit Steuerberater.
4. Gebucht wird der tatsächliche Vorgang (Gehalt, Miete, Zahlung) oder die
   Forderung. Ein eigenes Konto für eine vGA führt Buchfink nicht.
5. Ist die vGA bestätigt, erinnere an die Kapitalertragsteuer
   ([ausschuettung.md](ausschuettung.md#4-zufluss-anmeldung-abführung)) und
   notiere sie für die Körperschaftsteuererklärung.

## 5. Offene und verdeckte Einlage

- **Offene Einlage:** Der Gesellschafter zahlt ohne Gegenleistung in das
  Eigenkapital, z. B. zur Stärkung der Liquidität. Handelsrechtlich andere
  Zuzahlung in die Kapitalrücklage ([§ 272 Abs. 2 Nr. 4 HGB][hgb272]), SKR04
  2928 Kapitalrücklage durch Zuzahlungen in das Eigenkapital. Beleg:
  schriftliche Erklärung des Gesellschafters oder Beschluss. Buchung nach dem
  Ja: Bank an 2928.
- **Verdeckte Einlage:** Ein Gesellschafter oder eine ihm nahestehende Person
  wendet der GmbH außerhalb der gesellschaftsrechtlichen Einlagen einen
  einlagefähigen Vermögensvorteil zu, veranlasst durch das
  Gesellschaftsverhältnis ([R 8.9 Abs. 1, 3 KStR 2022][kstr89]); etwa durch
  Verzicht auf eine werthaltige Forderung oder Übertragung eines
  Wirtschaftsguts unter Wert. Bewertet wird mit dem Teilwert
  ([R 8.9 Abs. 4 Satz 1 KStR 2022][kstr89]). Sie
  erhöht das Einkommen der GmbH nicht ([§ 8 Abs. 3 Satz 3 KStG][kstg8]); beim
  Gesellschafter erhöhen sich die Anschaffungskosten der Beteiligung um den
  Teilwert eines eingelegten Wirtschaftsguts ([§ 6 Abs. 6 Satz 2 EStG][estg6]).
- Wertungsfrage: ob und in welcher Höhe eine Einlage vorliegt (Werthaltigkeit
  einer Forderung). Den Verzicht auf ein Gesellschafterdarlehen bucht der Mensch
  erst nach Klärung mit dem Steuerberater; die handelsrechtliche Buchung hängt
  davon ab.
- Eine Kapitalerhöhung gegen Einlage braucht Satzungsänderung und
  Handelsregister. Grenze: Steuerberater und Notar.

## 6. Steuerliches Einlagekonto

- Einlagen, die nicht ins Stammkapital gehen, erfasst die GmbH zum Schluss jedes
  Wirtschaftsjahres auf dem steuerlichen Einlagekonto
  ([§ 27 Abs. 1 Satz 1 KStG][kstg27]). Der Bestand wird gesondert festgestellt;
  die GmbH gibt dazu jährlich eine Erklärung ab
  ([§ 27 Abs. 2 Satz 1, 4 KStG][kstg27]). Das regelt
  [steuererklaerungen/kst.md](../steuererklaerungen/kst.md).
- Notiere jede Einlage mit Datum, Betrag und Beleg, damit die Erklärung sie
  vollständig enthält.
- Ausschüttungen aus dem Einlagekonto (Einlagenrückgewähr) sind beim
  Gesellschafter keine Einnahmen ([§ 20 Abs. 1 Nr. 1 Satz 3 EStG][estg20]) und
  brauchen eine eigene Bescheinigung ([§ 27 Abs. 3 KStG][kstg27]); siehe
  [ausschuettung.md](ausschuettung.md#6-was-buchfink-nicht-rechnet).

[kstg8]: https://www.gesetze-im-internet.de/kstg_1977/__8.html
[kstg27]: https://www.gesetze-im-internet.de/kstg_1977/__27.html
[estg6]: https://www.gesetze-im-internet.de/estg/__6.html
[estg20]: https://www.gesetze-im-internet.de/estg/__20.html
[estg43]: https://www.gesetze-im-internet.de/estg/__43.html
[estg44]: https://www.gesetze-im-internet.de/estg/__44.html
[ustg15]: https://www.gesetze-im-internet.de/ustg_1980/__15.html
[hgb272]: https://www.gesetze-im-internet.de/hgb/__272.html
[ao69]: https://www.gesetze-im-internet.de/ao_1977/__69.html
[kstr85]: https://www.steuerschroeder.de/steuergesetze/kstr/8.5
[kstr89]: https://www.steuerschroeder.de/steuergesetze/kstr/8.9
[ustae1523]: https://www.steuerschroeder.de/steuergesetze/ustae/15.23
