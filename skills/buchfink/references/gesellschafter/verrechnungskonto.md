# Verrechnungskonto mit Gesellschaftern

Rechtsstand: 17. Oktober 2026.

## Inhalt

1. Wofür das Verrechnungskonto da ist
2. Konten
3. Typische Vorgänge
4. Laufende Kontrolle

## 1. Wofür das Verrechnungskonto da ist

Eine GmbH kennt keine Privatentnahmen oder -einlagen. Fließt Geld mit einem
Gesellschafter ohne Darlehen, Ausschüttung oder Rechnung, entsteht eine
Forderung oder Verbindlichkeit auf den Konten aus Abschnitt 2, je
Gesellschafter mit `kontakt_id`.

## 2. Konten

| Saldo | Konto (SKR04) | Bilanzposten |
|---|---|---|
| GmbH hat Anspruch gegen den Gesellschafter | 1307 Forderungen gegen GmbH-Gesellschafter (1308 Restlaufzeit bis 1 Jahr, 1309 über 1 Jahr) | Forderungen gegen Gesellschafter |
| Gesellschafter hat Anspruch gegen die GmbH | 3640 Verbindlichkeiten gegenüber GmbH-Gesellschaftern (3641 bis 1 Jahr, 3642 1 bis 5 Jahre, 3643 über 5 Jahre) | Verbindlichkeiten gegenüber Gesellschaftern |

Forderung und Verbindlichkeit gegenüber demselben Gesellschafter werden nicht
saldiert ([ausweis.md](ausweis.md)).

## 3. Typische Vorgänge

**Gesellschafter bezahlt eine Firmenausgabe privat (Auslage):**
[belege/fremdwaehrung-auslagen.md](../belege/fremdwaehrung-auslagen.md#2-auslage-eines-gesellschafters).
Fertig, wenn der Saldo für diese Auslage null ist.

**GmbH bezahlt eine private Rechnung des Gesellschafters.**

1. Frag, ob die Zahlung ein Versehen war und der Gesellschafter sie
   zurückzahlt.
2. Ja: nach Rückfrage `bankumsatz_buchen` mit einer Zeile `konto: 1307` und
   der `kontakt_id` des Gesellschafters, ohne Umsatzsteuer und Vorsteuer
   ([vga-einlage.md](vga-einlage.md#3-prüfliste-typischer-fälle)); die
   Rückzahlung nach [bank/ablauf.md](../bank/ablauf.md) gegen 1307.
3. Nein: vGA-Verdacht nach
   [vga-einlage.md](vga-einlage.md#4-wie-du-mit-einem-verdacht-umgehst).

**Entnahme oder Überweisung an den Gesellschafter ohne Vereinbarung.** Wie
oben: Rückzahlung vereinbart, dann 1307; sonst vGA-Verdacht. Kredit an
Geschäftsführer aus dem stammkapitalgebundenen Vermögen:
[bank/darlehen.md](../bank/darlehen.md#9-gmbh-leiht-dem-gesellschafter).

**Verrechnung mit Gehalt oder Tantieme** nur mit Vereinbarung, die du als
Beleg ablegst. Das Gehalt läuft als Arbeitslohn über das Lohnjournal
([lohn/ablauf.md](../lohn/ablauf.md)); verrechnet wird nur der
Auszahlungsbetrag.

## 4. Laufende Kontrolle

- Prüfe beim Monatsabschluss mit `auswertung` (`art: susa`) die Salden von
  1307 bis 1309 und 3640 bis 3643; je Gesellschafter mit `buchungen_suchen`
  (`konto`, `kontakt_id`).
- Ein Soll-Saldo, der über Monate stehen bleibt, wirkt wie ein zinsloses
  Darlehen; die verhinderte Vermögensmehrung kann vGA sein
  ([R 8.5 Abs. 1 KStR 2022][kstr85]). Schlag Ausgleich oder einen
  Darlehensvertrag mit Zins vor ([ablauf.md](ablauf.md#4-darlehen)).
- Ein hoher Haben-Saldo zugunsten eines Gesellschafters ist wirtschaftlich
  ein Gesellschafterdarlehen mit Nachrang in der Insolvenz
  ([bank/darlehen.md](../bank/darlehen.md#11-insolvenz-nachrang-rangrücktritt));
  weise darauf hin.

[kstr85]: https://www.steuerschroeder.de/steuergesetze/kstr/8.5
