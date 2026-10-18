---
name: buchfink-gesellschafter
description: "Vorgänge zwischen der GmbH oder UG und ihren Gesellschaftern in Buchfink: Verrechnungskonto, Darlehen in beide Richtungen, verdeckte Gewinnausschüttung, verdeckte Einlage, Gewinnausschüttung mit Kapitalertragsteuer und Steuerbescheinigung, Ausweis in der Bilanz. Lade den Skill, wenn Geld zwischen Gesellschaft und Gesellschafter fließt, wenn private Ausgaben über die GmbH laufen oder ein Gesellschafter Firmenausgaben privat bezahlt, wenn ein Gesellschafter der GmbH Geld leiht oder sich Geld von ihr leiht, wenn ein Vertrag mit einem Gesellschafter (Gehalt, Tantieme, Miete, Darlehen, Firmenwagen) zu prüfen ist, wenn eine Ausschüttung beschlossen oder ausgezahlt wird, wenn Kapitalertragsteuer anzumelden ist oder fristen sie meldet, wenn ein Gesellschafter eine Steuerbescheinigung verlangt und wenn Gesellschafter oder Anteile wechseln. Eingehende Dividenden aus Beteiligungen regelt buchfink-beteiligungen."
license: EUPL-1.2
compatibility: "Benötigt die laufende Buchfink-App mit eingeschaltetem MCP-Server und einem für den Client zugelassenen Mandanten. Setzt den Skill buchfink-grundlagen voraus."
metadata:
  buchfink-rechtsstand: "2026-10-17"
  buchfink-werkzeugstand: "Buchfink 0.3.0"
---

# Buchfink: Gesellschafter

Setzt buchfink-grundlagen voraus; lade ihn, falls noch nicht geschehen.

Die GmbH ist eine eigene Person. Jede Zahlung zwischen ihr und einem
Gesellschafter braucht einen Rechtsgrund: Vertrag, Beschluss, Rechnung oder
Erstattungsanspruch. Ohne Rechtsgrund droht eine verdeckte Gewinnausschüttung
(vGA) oder eine verdeckte Einlage. Dieser Skill ordnet solche Vorgänge ein und
bucht sie.

Abgrenzung: Den Beleg einer Auslage bucht buchfink-belege (`bezahlung: auslage`).
Lohn des Geschäftsführers bucht buchfink-lohn. Die Ergebnisverwendung im
Abschluss beschreibt buchfink-jahresabschluss. Dividenden, die die GmbH selbst
erhält, regelt buchfink-beteiligungen.

Werkzeuge: `beleg_ablegen`, `buchung_erfassen`, `bankumsatz_buchen`,
`darlehen_erfassen`, `ergebnisverwendung_buchen`, `anlass_erfassen` (alle
Rückfrage), `konten`, `buchungen_suchen`, `auswertung`, `offene_posten`.
Gesellschafter-, Kapital- und Rücklagenkonten buchst du nach dem Ja selbst;
Kapitalerhaltung und Beschlusslage prüfst du vorher (Abschnitte 3 und 5).

## 1. Vorgang einordnen

| Vorgang | Einordnung | Konto (SKR04) | Lies |
|---|---|---|---|
| GmbH zahlt etwas Privates des Gesellschafters, er zahlt es zurück | Forderung gegen Gesellschafter | 1307 Forderungen gegen GmbH-Gesellschafter | [verrechnungskonto.md](references/verrechnungskonto.md) |
| GmbH zahlt etwas Privates, keine Rückzahlung vereinbart | vGA-Verdacht | – | [vga-einlage.md](references/vga-einlage.md) |
| Gesellschafter bezahlt eine Firmenausgabe privat | Erstattungsanspruch des Gesellschafters | 3640 Verbindlichkeiten gegenüber GmbH-Gesellschaftern | [verrechnungskonto.md](references/verrechnungskonto.md) |
| Gesellschafter leiht der GmbH Geld | Gesellschafterdarlehen | 3641 bis 3643 nach Restlaufzeit | [darlehen.md](references/darlehen.md) |
| GmbH leiht dem Gesellschafter Geld | Darlehen an Gesellschafter | 1308, 1309 oder 0960 | [darlehen.md](references/darlehen.md) |
| Gesellschafter zahlt ohne Gegenleistung ins Eigenkapital | Einlage in die Kapitalrücklage | 2928 Kapitalrücklage durch Zuzahlungen in das Eigenkapital | [vga-einlage.md](references/vga-einlage.md) |
| Gesellschafter verzichtet auf eine Forderung gegen die GmbH | verdeckte Einlage möglich | – | [vga-einlage.md](references/vga-einlage.md) |
| Gesellschafter vermietet oder liefert an die GmbH mit Rechnung | Lieferant (buchfink-belege) plus Fremdvergleich | – | [vga-einlage.md](references/vga-einlage.md) |
| Gewinnausschüttung | Beschluss, Kapitalertragsteuer | 3519, 3760 | [ausschuettung.md](references/ausschuettung.md) |
| Gesellschafter tritt ein oder aus, Anteile wechseln | Anlass `gesellschafterwechsel` (`anlass_erfassen`); Pflichten Gesellschafterliste und Transparenzregister | – | [pflichten.md#gesellschafterliste](../buchfink-grundlagen/references/pflichten.md#gesellschafterliste) |

Kontonummern stammen aus dem SKR04-Katalog von Buchfink. Die Konten der
Personengesellschaften (z. B. Privatkonten, 2020 Gesellschafter-Darlehen)
sperrt Buchfink für GmbH und UG.

## 2. Ablauf für jeden Vorgang

1. **Sachverhalt klären.** Wer zahlt wem was, wofür, auf welcher Grundlage?
   Frag nach Vertrag, Beschluss oder Rechnung. Fertig, wenn Rechtsgrund, Betrag,
   Datum und Gesellschafter feststehen.
2. **Beleg ablegen.** Vertrag, Beschluss oder Rechnung mit `beleg_ablegen`.
   Fehlt ein Fremdbeleg, erfasse mit dem Menschen einen Eigenbeleg mit Grund
   (buchfink-grundlagen). Fertig, wenn eine `beleg_id` vorliegt.
3. **Einordnen** nach Abschnitt 1. Ist eine Wertungsfrage offen (Abschnitt 7),
   leg sie vor, bevor du buchst.
4. **Rückfrage** im Format aus buchfink-grundlagen: Sachverhalt, Einordnung,
   Fundstelle, Buchungszeilen.
5. **Buchen** nach dem Ja:
   - Darlehen: erst den Vertrag mit `darlehen_erfassen`, dann den Bankumsatz
     mit einer Zeile `darlehen_id` (Abschnitt 4).
   - Liegt sonst ein Bankumsatz vor: `bankumsatz_buchen` mit einer Zeile
     `konto` auf das Konto aus Abschnitt 1.
   - Sonst: `buchung_erfassen` mit `beleg_id` und `kontakt_id` des
     Gesellschafters an der Zeile, falls vorhanden.
   Fertig, wenn die Antwort die Buchung mit den vorgelegten Zeilen enthält.

## 3. Grundregeln

- **Klare Vereinbarung im Voraus.** Zwischen der GmbH und einem beherrschenden
  Gesellschafter gilt eine Zahlung in der Regel als durch das
  Gesellschaftsverhältnis veranlasst, wenn keine zivilrechtlich wirksame,
  klare, im Voraus geschlossene Vereinbarung besteht oder anders verfahren wird
  ([R 8.5 Abs. 2 KStR 2022][kstr85]). Folge: vGA
  ([§ 8 Abs. 3 Satz 2 KStG][kstg8]). Frag deshalb bei jedem Vertrag mit einem
  Gesellschafter nach Datum und Form.
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
  ist verdächtig. Wertungsfrage; lies [vga-einlage.md](references/vga-einlage.md).

## 4. Darlehen

- Darlehen in beide Richtungen brauchen einen Vertrag mit Betrag, Zins,
  Laufzeit, Tilgung und gegebenenfalls Sicherheit, abgeschlossen vor der
  Auszahlung.
- Ein Darlehen der GmbH an den Gesellschafter ist nur zulässig, wenn der
  Rückzahlungsanspruch vollwertig ist ([§ 30 Abs. 1 Satz 2 GmbHG][gmbhg30]).
  An Geschäftsführer darf aus dem gebundenen Vermögen kein Kredit fließen
  ([§ 43a GmbHG][gmbhg43a]).
- Lege den Vertrag mit `beleg_ablegen` ab und erfasse ihn nach dem Ja mit
  `darlehen_erfassen`: Gesellschafter, Richtung, Betrag, Zinssatz und
  Zinsberechnung, Vertragsdatum, Beginn, Fälligkeit oder unbestimmte Laufzeit,
  Sicherheit, belegter Vergleichszins, Rangrücktritt. Die Feldnamen nennt das
  Schema des Werkzeugs. Den Kontakt des Gesellschafters findest oder anlegst
  du wie bei einer Auslage (buchfink-belege, Referenz zu Auslagen).
- Auszahlung und Tilgung bucht `bankumsatz_buchen` mit einer Zeile
  `darlehen_id`; Zinsen bucht `buchung_erfassen` oder eine Zeile `konto` am
  Bankumsatz. Konten, Nachrang und Insolvenzfolgen:
  [references/darlehen.md](references/darlehen.md).

## 5. Ausschüttung

1. Jahresabschluss festgestellt und Ergebnisverwendung beschlossen
   ([§ 46 Nr. 1 GmbHG][gmbhg46], [§ 29 GmbHG][gmbhg29]; Pflichten
   `feststellung` und `ergebnisverwendung` in `fristen`). Fertig, wenn das
   Beschlussprotokoll als Beleg abgelegt ist und Betrag sowie Auszahlungstag
   feststehen.
2. Prüfen: Kapitalerhaltung, bei der UG die gesetzliche Rücklage
   ([§ 5a Abs. 3 GmbHG][gmbhg5a]), Zahlungsfähigkeit
   ([§ 15b Abs. 5 InsO][inso15b]). Bei Zweifel: anhalten, Steuerberater.
3. Lege dem Menschen die Beträge je Gesellschafter vor und buche den
   Beschluss nach dem Ja mit `ergebnisverwendung_buchen` (Ablauf in
   buchfink-jahresabschluss). Buchfink bucht 2970 an 3519 (Netto an die
   Gesellschafter) und 3760 (Kapitalertragsteuer und Solidaritätszuschlag).
4. Am Zuflusstag: Kapitalertragsteuer anmelden und abführen
   ([§ 44 Abs. 1 Satz 5, Abs. 2 EStG][estg44], [§ 45a Abs. 1 EStG][estg45a]).
   `fristen` führt das als Pflicht `kapitalertragsteuer`, sobald der Beschluss
   erfasst ist; Frist, Weg und Folgen:
   [pflichten.md#kapitalertragsteuer](../buchfink-grundlagen/references/pflichten.md#kapitalertragsteuer).
   Nennt der Beschluss einen anderen Auszahlungstag, als Buchfink rechnet, gilt
   der Beschluss; nenne dem Menschen das Datum. Du bereitest die Zahlen vor;
   der Mensch sendet die Anmeldung in Mein ELSTER und zahlt.
5. Auszahlung an die Gesellschafter und Zahlung an das Finanzamt ordnet
   buchfink-bank zu. Fertig, wenn 3519 und 3760 für diesen Beschluss null sind
   (`auswertung` mit `art: kontoblatt`).
6. Steuerbescheinigung auf Verlangen
   ([references/steuerbescheinigung.md](references/steuerbescheinigung.md)).

Sätze, Kirchensteuer, Freistellung, Rundung, Fristen:
[references/ausschuettung.md](references/ausschuettung.md).

## 6. Verdeckte Gewinnausschüttung und verdeckte Einlage

- Eine vGA mindert das Einkommen der GmbH nicht
  ([§ 8 Abs. 3 Satz 2 KStG][kstg8]) und ist beim Gesellschafter ein
  Kapitalertrag ([§ 20 Abs. 1 Nr. 1 Satz 2 EStG][estg20]). Die GmbH hat darauf
  Kapitalertragsteuer einzubehalten und abzuführen und haftet dafür
  ([§ 43 Abs. 1 Satz 1 Nr. 1 EStG][estg43], [§ 44 Abs. 1 Satz 3, Abs. 5 EStG][estg44]).
- Eine verdeckte Einlage erhöht das Einkommen nicht
  ([§ 8 Abs. 3 Satz 3 KStG][kstg8]) und geht in das steuerliche Einlagekonto
  ([§ 27 KStG][kstg27]).
- Erkennst du einen Verdacht, halte vor dem Buchen an. Lege ihn mit Fundstelle
  vor und empfiehl einen Steuerberater. Gebucht wird der tatsächliche Vorgang
  (Gehalt, Miete, Zahlung); bei privaten Ausgaben ohne Rückzahlungsvereinbarung
  schlag eine Forderung gegen den Gesellschafter vor (1307). Die Korrektur des
  Einkommens gehört in die Körperschaftsteuererklärung
  (buchfink-steuererklaerungen).

## 7. Wertungsfragen

Leite jede mit „Wertungsfrage:“ ein, nenne Grundlage und Optionen; der Mensch
entscheidet, bei Zweifel mit Steuerberater.

- Angemessenheit von Geschäftsführergehalt, Tantieme, Pensionszusage, Miete,
  Zins ([§ 8 Abs. 3 Satz 2 KStG][kstg8], [R 8.5 KStR 2022][kstr85]).
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
  werden, halte an und sprich den Menschen an (buchfink-grundlagen,
  Warnzeichen). Der Weg ist eine Forderung gegen den Gesellschafter oder die
  Klärung mit dem Steuerberater.
- Die Kapitalertragsteuer auf eine Ausschüttung ist am Zuflusstag anzumelden
  und abzuführen ([§ 44 Abs. 1 Satz 5 Halbsatz 2 EStG][estg44]). Die oft
  genannte Frist „10. des Folgemonats“ gilt für andere Kapitalerträge.
- Wechseln innerhalb von fünf Jahren mehr als 50 % der Anteile an einen
  Erwerber, gehen nicht genutzte Verluste unter
  ([§ 8c Abs. 1 Satz 1 KStG][kstg8c]). Sag das dem Menschen vor dem Wechsel;
  Ausnahmen prüft ein Steuerberater.
- Buchfink rechnet 25 % Kapitalertragsteuer und 5,5 % Solidaritätszuschlag auf
  den vollen Betrag. Kirchensteuer, Nichtveranlagungs-Bescheinigung und
  Einlagenrückgewähr bildet es nicht ab; lies
  [references/ausschuettung.md](references/ausschuettung.md).
- Ein Soll-Saldo auf 1307 über längere Zeit wirkt wie ein zinsloses Darlehen
  an den Gesellschafter ([R 8.5 Abs. 1 KStR 2022][kstr85]). Weise beim
  Monatsabschluss darauf hin.
- Zahlungen an einen Gesellschafter, die zur Zahlungsunfähigkeit führen müssen,
  machen den Geschäftsführer ersatzpflichtig
  ([§ 15b Abs. 5 InsO][inso15b]). Prüfe vor jeder größeren Auszahlung an
  Gesellschafter die Liquidität mit `auswertung` und `offene_posten`.
- Die Rückzahlung eines Gesellschafterdarlehens im letzten Jahr vor einem
  Insolvenzantrag ist anfechtbar ([§ 135 Abs. 1 Nr. 2 InsO][inso135]).

## Referenzen

- [references/verrechnungskonto.md](references/verrechnungskonto.md): Lies, wenn
  Geld zwischen GmbH und Gesellschafter fließt, ohne dass ein Darlehen oder
  eine Ausschüttung vereinbart ist.
- [references/darlehen.md](references/darlehen.md): Lies bei jedem Darlehen
  zwischen GmbH und Gesellschafter, bei Zinsen, Rangrücktritt und Tilgung.
- [references/vga-einlage.md](references/vga-einlage.md): Lies bei Verträgen mit
  Gesellschaftern, privaten Ausgaben, Verzicht, Einlage und jedem
  vGA-Verdacht.
- [references/ausschuettung.md](references/ausschuettung.md): Lies vor jeder
  Ausschüttung und bei Kapitalertragsteuer, Kirchensteuer und Freistellung.
- [references/steuerbescheinigung.md](references/steuerbescheinigung.md): Lies,
  wenn ein Gesellschafter eine Steuerbescheinigung verlangt.
- [references/ausweis.md](references/ausweis.md): Lies beim Abschluss, wenn
  Forderungen, Verbindlichkeiten oder Ausleihungen gegenüber Gesellschaftern
  auszuweisen sind.

[kstr85]: https://www.steuerschroeder.de/steuergesetze/kstr/8.5
[kstg8]: https://www.gesetze-im-internet.de/kstg_1977/__8.html
[kstg27]: https://www.gesetze-im-internet.de/kstg_1977/__27.html
[kstg8c]: https://www.gesetze-im-internet.de/kstg_1977/__8c.html
[gmbhg5a]: https://www.gesetze-im-internet.de/gmbhg/__5a.html
[gmbhg29]: https://www.gesetze-im-internet.de/gmbhg/__29.html
[gmbhg30]: https://www.gesetze-im-internet.de/gmbhg/__30.html
[gmbhg31]: https://www.gesetze-im-internet.de/gmbhg/__31.html
[gmbhg35]: https://www.gesetze-im-internet.de/gmbhg/__35.html
[gmbhg43a]: https://www.gesetze-im-internet.de/gmbhg/__43a.html
[gmbhg46]: https://www.gesetze-im-internet.de/gmbhg/__46.html
[bgb181]: https://www.gesetze-im-internet.de/bgb/__181.html
[estg20]: https://www.gesetze-im-internet.de/estg/__20.html
[estg43]: https://www.gesetze-im-internet.de/estg/__43.html
[estg44]: https://www.gesetze-im-internet.de/estg/__44.html
[estg45a]: https://www.gesetze-im-internet.de/estg/__45a.html
[inso15b]: https://www.gesetze-im-internet.de/inso/__15b.html
[inso135]: https://www.gesetze-im-internet.de/inso/__135.html
