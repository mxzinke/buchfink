# Ablauf: Gesellschafter

Rechtsstand: 17. Oktober 2026. Werkzeugstand: Buchfink 0.3.0.

Dieser Ablauf ordnet Vorgänge zwischen der GmbH oder UG und ihren
Gesellschaftern ein und bucht sie. Dividenden, die die GmbH selbst erhält,
regelt [beteiligungen/dividenden.md](../beteiligungen/dividenden.md).

## Inhalt

1. Vorgang einordnen
2. Ablauf für jeden Vorgang
3. Grundregeln
4. Darlehen
5. Ausschüttung
6. Verdeckte Gewinnausschüttung und verdeckte Einlage
7. Wertungsfragen
8. Nur der Mensch
9. Stolperfallen

## 1. Vorgang einordnen

Die GmbH ist eine eigene Person. Jede Zahlung zwischen ihr und einem
Gesellschafter braucht einen Rechtsgrund: Vertrag, Beschluss, Rechnung oder
Erstattungsanspruch. Ohne Rechtsgrund droht eine verdeckte Gewinnausschüttung
(vGA) oder eine verdeckte Einlage.

Abgrenzung: Den Beleg einer Auslage bucht der
[Belegablauf](../belege/fremdwaehrung-auslagen.md#2-auslage-eines-gesellschafters)
(`bezahlung: auslage`). Lohn des Geschäftsführers bucht
[lohn/ablauf.md](../lohn/ablauf.md). Die Ergebnisverwendung im Abschluss
beschreibt
[jahresabschluss/ergebnisverwendung-ug.md](../jahresabschluss/ergebnisverwendung-ug.md).

Werkzeuge: `beleg_ablegen`, `buchung_erfassen`, `bankumsatz_buchen`,
`darlehen_erfassen`, `ergebnisverwendung_buchen`, `anlass_erfassen` (alle
Rückfrage), `konten`, `buchungen_suchen`, `auswertung`, `offene_posten`.
Gesellschafter-, Kapital- und Rücklagenkonten buchst du nach dem Ja selbst;
Kapitalerhaltung und Beschlusslage prüfst du vorher (Abschnitte 3 und 5).

| Vorgang | Einordnung | Konto (SKR04) | Lies |
|---|---|---|---|
| GmbH zahlt etwas Privates des Gesellschafters, er zahlt es zurück | Forderung gegen Gesellschafter | 1307 Forderungen gegen GmbH-Gesellschafter | [verrechnungskonto.md](verrechnungskonto.md) |
| GmbH zahlt etwas Privates, keine Rückzahlung vereinbart | vGA-Verdacht | – | [vga-einlage.md](vga-einlage.md) |
| Gesellschafter bezahlt eine Firmenausgabe privat | Erstattungsanspruch des Gesellschafters | 3640 Verbindlichkeiten gegenüber GmbH-Gesellschaftern | [belege/fremdwaehrung-auslagen.md](../belege/fremdwaehrung-auslagen.md#2-auslage-eines-gesellschafters) |
| Gesellschafter leiht der GmbH Geld | Gesellschafterdarlehen | 3641 bis 3643 nach Restlaufzeit | [bank/darlehen.md](../bank/darlehen.md#8-gesellschafter-leiht-der-gmbh) |
| GmbH leiht dem Gesellschafter Geld | Darlehen an Gesellschafter | 1308, 1309 oder 0960 | [bank/darlehen.md](../bank/darlehen.md#9-gmbh-leiht-dem-gesellschafter) |
| Gesellschafter zahlt ohne Gegenleistung ins Eigenkapital | Einlage in die Kapitalrücklage | 2928 Kapitalrücklage durch Zuzahlungen in das Eigenkapital | [vga-einlage.md](vga-einlage.md#5-offene-und-verdeckte-einlage) |
| Gesellschafter verzichtet auf eine Forderung gegen die GmbH | verdeckte Einlage möglich | – | [vga-einlage.md](vga-einlage.md#5-offene-und-verdeckte-einlage) |
| Gesellschafter vermietet oder liefert an die GmbH mit Rechnung | Lieferant ([belege/ablauf.md](../belege/ablauf.md)) plus Fremdvergleich | – | [vga-einlage.md](vga-einlage.md#3-prüfliste-typischer-fälle) |
| Gewinnausschüttung | Beschluss, Kapitalertragsteuer | 3519, 3760 | [ausschuettung.md](ausschuettung.md) |
| Gesellschafter tritt ein oder aus, Anteile wechseln | Anlass `gesellschafterwechsel` (`anlass_erfassen`); Pflichten Gesellschafterliste und Transparenzregister | – | [grundlagen/pflichten.md#gesellschafterliste](../grundlagen/pflichten.md#gesellschafterliste) |

Kontonummern stammen aus dem SKR04-Katalog von Buchfink. Die Konten der
Personengesellschaften (z. B. Privatkonten, 2020 Gesellschafter-Darlehen)
sperrt Buchfink für GmbH und UG.

## 2. Ablauf für jeden Vorgang

1. **Sachverhalt klären.** Wer zahlt wem was, wofür, auf welcher Grundlage?
   Frag nach Vertrag, Beschluss oder Rechnung. Fertig, wenn Rechtsgrund, Betrag,
   Datum und Gesellschafter feststehen.
2. **Beleg ablegen.** Vertrag, Beschluss oder Rechnung mit `beleg_ablegen`.
   Fehlt ein Fremdbeleg, erfasse mit dem Menschen einen Eigenbeleg mit Grund
   ([grundlagen/gobd.md](../grundlagen/gobd.md#8-belegwesen-und-eigenbeleg)).
   Fertig, wenn eine `beleg_id` vorliegt.
3. **Einordnen** nach Abschnitt 1. Ist eine Wertungsfrage offen (Abschnitt 7),
   leg sie vor, bevor du buchst.
4. **Rückfrage** im Format aus [SKILL.md](../../SKILL.md#3-den-menschen-fragen):
   Sachverhalt, Einordnung, Fundstelle, Buchungszeilen.
5. **Buchen** nach dem Ja:
   - Darlehen: erst den Vertrag mit `darlehen_erfassen`, dann den Bankumsatz
     mit einer Zeile `darlehen_id` (Abschnitt 4).
   - Liegt sonst ein Bankumsatz vor: `bankumsatz_buchen` mit einer Zeile
     `konto` auf das Konto aus Abschnitt 1.
   - Sonst: `buchung_erfassen` mit `beleg_id` und `kontakt_id` des
     Gesellschafters an der Zeile, falls vorhanden.
   Fertig, wenn die Antwort die Buchung mit den vorgelegten Zeilen enthält.

## 3. Grundregeln

- **Klare Vereinbarung im Voraus.** Mit einem beherrschenden Gesellschafter
  braucht jede Leistung eine zivilrechtlich wirksame, klare, im Voraus
  geschlossene und tatsächlich durchgeführte Vereinbarung, sonst droht eine
  vGA ([vga-einlage.md](vga-einlage.md#1-was-eine-vga-ist)). Frag deshalb bei
  jedem Vertrag mit einem Gesellschafter nach Datum und Form.
- **Ein-Personen-GmbH.** Ist der einzige Gesellschafter zugleich alleiniger
  Geschäftsführer, gilt für seine Geschäfte mit der GmbH § 181 BGB; er braucht
  eine Befreiung. Jedes solche Geschäft ist unverzüglich in eine Niederschrift
  aufzunehmen ([§ 35 Abs. 3 GmbHG][gmbhg35], [§ 181 BGB][bgb181]). Bitte den
  Menschen, die Niederschrift als Beleg abzulegen.
- **Kapitalerhaltung.** Vermögen, das zur Erhaltung des Stammkapitals nötig ist,
  darf nicht an Gesellschafter ausgezahlt werden
  ([§ 30 Abs. 1 GmbHG][gmbhg30]). Verbotene Zahlungen sind zu erstatten
  ([§ 31 GmbHG][gmbhg31]).
- **Fremdvergleich.** Was die GmbH einem fremden Dritten nicht zugestehen würde,
  ist verdächtig. Wertungsfrage; lies [vga-einlage.md](vga-einlage.md).

## 4. Darlehen

Vertrag, Konten, Zinsen, Nachrang und Insolvenzfolgen stehen in
[bank/darlehen.md](../bank/darlehen.md). Die Werkzeugfolge:

1. Frag nach dem Vertrag (Betrag, Zins, Laufzeit, Tilgung, gegebenenfalls
   Sicherheit, vor der Auszahlung geschlossen) und lege ihn mit
   `beleg_ablegen` ab. Den Kontakt des Gesellschafters findest oder anlegst
   du wie bei einer
   [Auslage](../belege/fremdwaehrung-auslagen.md#2-auslage-eines-gesellschafters).
2. Erfasse den Vertrag nach dem Ja mit `darlehen_erfassen`: Gesellschafter,
   Richtung, Betrag, Zinssatz und Zinsberechnung, Vertragsdatum, Beginn,
   Fälligkeit oder unbestimmte Laufzeit, Sicherheit, belegter Vergleichszins,
   Rangrücktritt. Die Feldnamen nennt das Schema des Werkzeugs.
3. Eine Berichtigung (`darlehen_id` mit `grund`) ändert nur die Angaben, die
   du nennst; alles andere bleibt, wie es ist.
4. Auszahlung und Tilgung bucht `bankumsatz_buchen` mit einer Zeile
   `darlehen_id`.
5. Die Zinsen bucht `darlehen_erfassen` mit `art: zinsen` bis `zinsen_bis`:
   Buchfink rechnet sie aus dem Vertrag, legt einen Eigenbeleg mit der
   Zinsrechnung an und vermerkt die Zinsperiode. Liegt eine Zinsabrechnung
   vor, gib sie als `beleg_id` mit. Zinsen nie mit `buchung_erfassen` buchen,
   sonst bleibt die Zinsperiode offen. Die Zinszahlung bucht
   `bankumsatz_buchen` mit einer Zeile `konto`.

## 5. Ausschüttung

Voraussetzungen, Grenzen, Steuerabzug und Buchungen stehen in
[ausschuettung.md](ausschuettung.md).

1. Jahresabschluss festgestellt und Ergebnisverwendung beschlossen
   ([ausschuettung.md](ausschuettung.md#1-voraussetzungen); Pflichten
   `feststellung` und `ergebnisverwendung` in `fristen`). Fertig, wenn das
   Beschlussprotokoll als Beleg abgelegt ist und Betrag sowie Auszahlungstag
   feststehen.
2. Prüfen: Kapitalerhaltung, bei der UG die gesetzliche Rücklage,
   Zahlungsfähigkeit ([ausschuettung.md](ausschuettung.md#2-grenzen-der-ausschüttung)).
   Bei Zweifel: anhalten, Steuerberater.
3. Lege dem Menschen die Beträge je Gesellschafter vor und buche den
   Beschluss nach dem Ja mit `ergebnisverwendung_buchen` (Ablauf in
   [jahresabschluss/ergebnisverwendung-ug.md](../jahresabschluss/ergebnisverwendung-ug.md),
   Buchungen in [ausschuettung.md](ausschuettung.md#5-buchungen-in-buchfink)).
4. Am Zuflusstag: Kapitalertragsteuer anmelden und abführen
   ([ausschuettung.md](ausschuettung.md#4-zufluss-anmeldung-abführung)).
   `fristen` führt das als Pflicht `kapitalertragsteuer`, sobald der Beschluss
   erfasst ist; Frist, Weg und Folgen:
   [grundlagen/pflichten.md#kapitalertragsteuer](../grundlagen/pflichten.md#kapitalertragsteuer).
   Nennt der Beschluss einen anderen Auszahlungstag, als Buchfink rechnet, gilt
   der Beschluss; nenne dem Menschen das Datum. Du bereitest die Zahlen vor;
   der Mensch sendet die Anmeldung in Mein ELSTER und zahlt.
5. Auszahlung an die Gesellschafter und Zahlung an das Finanzamt ordnest du
   nach [bank/ablauf.md](../bank/ablauf.md) zu. Fertig, wenn 3519 und 3760 für
   diesen Beschluss null sind (`auswertung` mit `art: kontoblatt`).
6. Steuerbescheinigung auf Verlangen
   ([steuerbescheinigung.md](steuerbescheinigung.md)).

## 6. Verdeckte Gewinnausschüttung und verdeckte Einlage

Erkennst du einen Verdacht, halte vor dem Buchen an und geh nach
[vga-einlage.md](vga-einlage.md#4-wie-du-mit-einem-verdacht-umgehst) vor.
Gebucht wird der tatsächliche Vorgang (Gehalt, Miete, Zahlung); bei privaten
Ausgaben ohne Rückzahlungsvereinbarung schlag eine Forderung gegen den
Gesellschafter vor (1307). Folgen für Körperschaftsteuer und
Kapitalertragsteuer, verdeckte Einlage und Einlagekonto stehen in
[vga-einlage.md](vga-einlage.md).

## 7. Wertungsfragen

Leite jede mit „Wertungsfrage:“ ein, nenne Grundlage und Optionen; der Mensch
entscheidet, bei Zweifel mit Steuerberater.

- Angemessenheit von Geschäftsführergehalt, Tantieme, Pensionszusage, Miete,
  Zins ([vga-einlage.md](vga-einlage.md#3-prüfliste-typischer-fälle)).
- Ob eine private Mitveranlassung vorliegt (Reise, Fahrzeug, Bewirtung).
- Ob ein Gesellschafter beherrschend ist und ob die Vereinbarung klar, im
  Voraus und tatsächlich durchgeführt ist.
- Ob ein Forderungsverzicht oder eine Zahlung eine Einlage ist.
- Ob ein Darlehen an den Gesellschafter vollwertig ist.

## 8. Nur der Mensch

- Beschlüsse fassen und unterschreiben; Verträge abschließen.
- Gesellschafter und Anteile in den Stammdaten der App ändern.
- Jahresabschluss feststellen.
- Kapitalertragsteuer-Anmeldung in Mein ELSTER übermitteln und die
  Übermittlung in der App bestätigen; das Protokoll legst du auf seine
  Anweisung aus dem Eingang ab.
- Kirchensteuer-Abfrage beim BZSt.
- Steuerbescheinigung unterschreiben oder freigeben und aushändigen.

## 9. Stolperfallen

- Soll eine private Ausgabe eines Gesellschafters als Betriebsausgabe gebucht
  werden, halte an und sprich den Menschen an
  ([grundlagen/grenzen.md](../grundlagen/grenzen.md#2-warnzeichen-die-du-sofort-meldest)).
  Der Weg ist eine Forderung gegen den Gesellschafter oder die Klärung mit dem
  Steuerberater.
- Die Kapitalertragsteuer auf eine Ausschüttung ist am Zuflusstag anzumelden
  und abzuführen, nicht erst am 10. des Folgemonats
  ([ausschuettung.md](ausschuettung.md#4-zufluss-anmeldung-abführung)).
- Wechseln innerhalb von fünf Jahren mehr als 50 % der Anteile, können nicht
  genutzte Verluste untergehen. Sag das dem Menschen vor dem Wechsel
  ([grundlagen/grenzen.md](../grundlagen/grenzen.md#2-warnzeichen-die-du-sofort-meldest));
  Ausnahmen prüft ein Steuerberater.
- Buchfink rechnet Kapitalertragsteuer und Solidaritätszuschlag auf den vollen
  Betrag. Kirchensteuer, Nichtveranlagungs-Bescheinigung und
  Einlagenrückgewähr bildet es nicht ab
  ([ausschuettung.md](ausschuettung.md#6-was-buchfink-nicht-rechnet)).
- Ein Soll-Saldo auf 1307 über längere Zeit wirkt wie ein zinsloses Darlehen
  an den Gesellschafter. Weise beim Monatsabschluss darauf hin
  ([verrechnungskonto.md](verrechnungskonto.md#4-laufende-kontrolle)).
- Prüfe vor jeder größeren Auszahlung an Gesellschafter die Liquidität mit
  `auswertung` und `offene_posten`; der Geschäftsführer haftet für Zahlungen,
  die zur Zahlungsunfähigkeit führen mussten
  ([ausschuettung.md](ausschuettung.md#2-grenzen-der-ausschüttung)).
- Die Rückzahlung eines Gesellschafterdarlehens im letzten Jahr vor einem
  Insolvenzantrag ist anfechtbar
  ([bank/darlehen.md](../bank/darlehen.md#11-insolvenz-nachrang-rangrücktritt)).

[gmbhg30]: https://www.gesetze-im-internet.de/gmbhg/__30.html
[gmbhg31]: https://www.gesetze-im-internet.de/gmbhg/__31.html
[gmbhg35]: https://www.gesetze-im-internet.de/gmbhg/__35.html
[bgb181]: https://www.gesetze-im-internet.de/bgb/__181.html
