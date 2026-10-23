# Ablauf: Gesellschafter

Rechtsstand: 17. Oktober 2026. Werkzeugstand: Buchfink 0.3.0.

Vorgänge zwischen der GmbH oder UG und ihren Gesellschaftern. Dividenden, die
die GmbH selbst erhält:
[beteiligungen/dividenden.md](../beteiligungen/dividenden.md).

## Inhalt

1. Vorgang einordnen
2. Ablauf für jeden Vorgang
3. Grundregeln
4. Darlehen
5. Ausschüttung
6. Verdeckte Gewinnausschüttung und verdeckte Einlage
7. Wertungsfragen
8. Nur der Mensch

## 1. Vorgang einordnen

Jede Zahlung zwischen GmbH und Gesellschafter braucht einen Rechtsgrund:
Vertrag, Beschluss, Rechnung oder Erstattungsanspruch. Fehlt er, prüfe
verdeckte Gewinnausschüttung (vGA) oder verdeckte Einlage.

| Vorgang | Einordnung | Konto (SKR04) | Lies |
|---|---|---|---|
| GmbH zahlt etwas Privates des Gesellschafters, er zahlt es zurück | Forderung gegen Gesellschafter | 1307 | [verrechnungskonto.md](verrechnungskonto.md) |
| GmbH zahlt etwas Privates, keine Rückzahlung vereinbart | vGA-Verdacht | – | [vga-einlage.md](vga-einlage.md) |
| Gesellschafter bezahlt eine Firmenausgabe privat | Auslage, Erstattungsanspruch des Gesellschafters | 3640 | [belege/fremdwaehrung-auslagen.md](../belege/fremdwaehrung-auslagen.md#2-auslage-eines-gesellschafters) |
| Gesellschafter leiht der GmbH Geld oder umgekehrt | Gesellschafterdarlehen | aus `darlehen_erfassen` | Abschnitt 4 |
| Gesellschafter zahlt ohne Gegenleistung ins Eigenkapital | Einlage in die Kapitalrücklage | 2928 | [vga-einlage.md](vga-einlage.md#5-offene-und-verdeckte-einlage) |
| Gesellschafter verzichtet auf eine Forderung gegen die GmbH | verdeckte Einlage möglich | – | [vga-einlage.md](vga-einlage.md#5-offene-und-verdeckte-einlage) |
| Gesellschafter vermietet oder liefert an die GmbH mit Rechnung | Lieferant ([belege/ablauf.md](../belege/ablauf.md)) plus Fremdvergleich | – | [vga-einlage.md](vga-einlage.md#3-prüfliste-typischer-fälle) |
| Gehalt des Gesellschafter-Geschäftsführers | Arbeitslohn | – | [lohn/ablauf.md](../lohn/ablauf.md) |
| Gewinnausschüttung | Beschluss, Kapitalertragsteuer | 3519, 3760 | Abschnitt 5 |
| Gesellschafter tritt ein oder aus, Anteile wechseln | Anlass `gesellschafterwechsel` | – | [grundlagen/pflichten.md#gesellschafterliste](../grundlagen/pflichten.md#gesellschafterliste) |

Wechseln innerhalb von fünf Jahren mehr als 50 % der Anteile, können nicht
genutzte Verluste untergehen ([§ 8c Abs. 1 KStG][kstg8c]). Sag das dem
Menschen vor dem Wechsel
([grundlagen/grenzen.md](../grundlagen/grenzen.md#2-warnzeichen-die-du-sofort-meldest));
Ausnahmen prüft ein Steuerberater.

## 2. Ablauf für jeden Vorgang

1. **Sachverhalt klären.** Wer zahlt wem was, wofür, auf welcher Grundlage?
   Fertig, wenn Rechtsgrund, Betrag, Datum und Gesellschafter feststehen.
2. **Beleg ablegen.** Vertrag, Beschluss oder Rechnung mit `beleg_ablegen`;
   fehlt ein Fremdbeleg, ein Eigenbeleg mit Grund
   ([grundlagen/gobd.md](../grundlagen/gobd.md#8-belegwesen-und-eigenbeleg)).
   Fertig, wenn eine `beleg_id` vorliegt.
3. **Einordnen** nach Abschnitt 1 und vorlegen: Sachverhalt, Einordnung,
   Fundstelle, Buchungszeilen. Offene Wertungsfragen (Abschnitt 7) klärst du
   vor dem Buchen.
4. **Buchen** nach dem Ja: Darlehen nach Abschnitt 4; mit Bankumsatz
   `bankumsatz_buchen` mit einer Zeile `konto` aus Abschnitt 1; sonst
   `buchung_erfassen` mit `beleg_id` und der `kontakt_id` des Gesellschafters
   an der Zeile. Fertig, wenn die Antwort die Buchung mit den vorgelegten
   Zeilen enthält.

## 3. Grundregeln

- **Klare Vereinbarung im Voraus.** Mit einem beherrschenden Gesellschafter
  braucht jede Leistung eine zivilrechtlich wirksame, klare, im Voraus
  geschlossene und tatsächlich durchgeführte Vereinbarung, sonst droht eine
  vGA ([vga-einlage.md](vga-einlage.md#1-was-eine-vga-ist)). Frag bei jedem
  Vertrag mit einem Gesellschafter nach Datum und Form.
- **Ein-Personen-GmbH.** Geschäfte des alleinigen
  Gesellschafter-Geschäftsführers mit der GmbH brauchen eine Befreiung von
  [§ 181 BGB][bgb181] und eine unverzügliche Niederschrift
  ([§ 35 Abs. 3 GmbHG][gmbhg35]); lass sie als Beleg ablegen.
- **Kapitalerhaltung.** Vermögen, das zur Erhaltung des Stammkapitals nötig
  ist, darf nicht an Gesellschafter fließen ([§ 30 Abs. 1 GmbHG][gmbhg30]);
  verbotene Zahlungen sind zu erstatten ([§ 31 GmbHG][gmbhg31]).
- **Zahlungsfähigkeit.** Für Zahlungen an Gesellschafter, die zur
  Zahlungsunfähigkeit führen mussten, haftet der Geschäftsführer
  ([§ 15b Abs. 5 InsO][inso15b]). Prüfe vor jeder größeren Auszahlung die
  Liquidität mit `auswertung` und `offene_posten`.
- **Fremdvergleich.** Was die GmbH einem fremden Dritten nicht zugestehen
  würde, ist verdächtig; Wertungsfrage nach [vga-einlage.md](vga-einlage.md).

## 4. Darlehen

Vertrag, Konten, Zinsen, Nachrang und Insolvenzfolgen:
[bank/darlehen.md](../bank/darlehen.md). Frag nach dem Vertrag (Betrag, Zins,
Laufzeit, Tilgung, Sicherheit, vor der Auszahlung geschlossen), lege ihn mit
`beleg_ablegen` (`art: sonstiges`) ab und erfasse ihn nach dem Ja mit
`darlehen_erfassen` und `beleg_id`; der Vertrag ist dann dem Darlehen
zugeordnet und wird nicht gebucht.
Fertig, wenn `auswertung` (`art: darlehen`) den Vertrag zeigt und der Mensch
die Befunde zum Fremdvergleich gesehen hat.

## 5. Ausschüttung

Einzelheiten: [ausschuettung.md](ausschuettung.md).

1. **Beschluss** über die Ergebnisverwendung nach festgestelltem Abschluss
   ([ausschuettung.md, Abschnitt 1](ausschuettung.md#1-voraussetzungen)).
   Fertig, wenn er als Beleg abliegt und Betrag und Auszahlungstag
   feststehen.
2. **Grenzen** prüfen
   ([ausschuettung.md, Abschnitt 2](ausschuettung.md#2-grenzen-der-ausschüttung));
   bei Zweifel anhalten, Steuerberater.
3. **Buchen:** Beträge je Gesellschafter vorlegen, nach dem Ja
   `ergebnisverwendung_buchen`.
4. **Kapitalertragsteuer** am Zuflusstag anmelden und abführen lassen
   ([ausschuettung.md, Abschnitt 4](ausschuettung.md#4-zufluss-anmeldung-abführung)).
   Weicht der Auszahlungstag im Beschluss von der Pflicht
   `kapitalertragsteuer` in `fristen` ab, gilt der Beschluss; nenne das Datum.
5. **Zahlungen** nach [bank/ablauf.md](../bank/ablauf.md) zuordnen. Fertig wie
   in [ausschuettung.md, Abschnitt 5](ausschuettung.md#5-buchungen-in-buchfink).
6. **Steuerbescheinigung** auf Verlangen
   ([steuerbescheinigung.md](steuerbescheinigung.md)).

## 6. Verdeckte Gewinnausschüttung und verdeckte Einlage

Bei einem Verdacht halte vor dem Buchen an und geh nach
[vga-einlage.md, Abschnitt 4](vga-einlage.md#4-wie-du-mit-einem-verdacht-umgehst)
vor.

## 7. Wertungsfragen

Leite jede mit „Wertungsfrage:“ ein, nenne Grundlage und Optionen; der Mensch
entscheidet, bei Zweifel mit Steuerberater.

- Angemessenheit von Gehalt, Tantieme, Pensionszusage, Miete, Zins
  ([vga-einlage.md](vga-einlage.md#3-prüfliste-typischer-fälle)).
- Private Mitveranlassung (Reise, Fahrzeug, Bewirtung).
- Beherrschung; Vereinbarung klar, im Voraus, tatsächlich durchgeführt.
- Einlage durch Forderungsverzicht oder Zahlung.
- Vollwertigkeit eines Darlehens an den Gesellschafter.

## 8. Nur der Mensch

- Beschlüsse fassen und unterschreiben; Verträge abschließen.
- Gesellschafter und Anteile in den Stammdaten der App ändern.
- Jahresabschluss feststellen.
- Kapitalertragsteuer-Anmeldung in Mein ELSTER übermitteln und in der App
  bestätigen; das Protokoll legst du auf seine Anweisung aus dem Eingang ab.
- Kirchensteuer-Abfrage beim BZSt.
- Steuerbescheinigung unterschreiben oder freigeben und aushändigen.

[gmbhg30]: https://www.gesetze-im-internet.de/gmbhg/__30.html
[gmbhg31]: https://www.gesetze-im-internet.de/gmbhg/__31.html
[gmbhg35]: https://www.gesetze-im-internet.de/gmbhg/__35.html
[bgb181]: https://www.gesetze-im-internet.de/bgb/__181.html
[inso15b]: https://www.gesetze-im-internet.de/inso/__15b.html
[kstg8c]: https://www.gesetze-im-internet.de/kstg_1977/__8c.html
