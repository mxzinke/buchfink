# Verrechnungskonto mit Gesellschaftern

Rechtsstand: 17. Oktober 2026.

## Inhalt

1. Wofür das Verrechnungskonto da ist
2. Konten
3. Typische Vorgänge
4. Laufende Kontrolle

## 1. Wofür das Verrechnungskonto da ist

Eine GmbH kennt keine Privatentnahmen und Privateinlagen. Fließt Geld zwischen
ihr und einem Gesellschafter, ohne dass ein Darlehen, eine Ausschüttung oder
eine Rechnung dahintersteht, entsteht eine Forderung oder Verbindlichkeit
zwischen beiden. Das „Verrechnungskonto“ ist dafür der übliche Name; in
Buchfink sind es die Konten aus Abschnitt 2, je Gesellschafter mit
`kontakt_id`.

## 2. Konten

| Saldo | Konto (SKR04) | Bilanzposten |
|---|---|---|
| GmbH hat Anspruch gegen den Gesellschafter | 1307 Forderungen gegen GmbH-Gesellschafter (1308 Restlaufzeit bis 1 Jahr, 1309 über 1 Jahr) | Forderungen gegen Gesellschafter |
| Gesellschafter hat Anspruch gegen die GmbH | 3640 Verbindlichkeiten gegenüber GmbH-Gesellschaftern (3641 bis 1 Jahr, 3642 1 bis 5 Jahre, 3643 über 5 Jahre) | Verbindlichkeiten gegenüber Gesellschaftern |

Forderung und Verbindlichkeit gegenüber demselben Gesellschafter werden in der
Bilanz grundsätzlich nicht saldiert ([§ 246 Abs. 2 Satz 1 HGB][hgb246]).
Den Ausweis im Abschluss nennt SKILL.md unter Referenzen.

## 3. Typische Vorgänge

**Gesellschafter bezahlt eine Firmenausgabe privat (Auslage).**
Den Beleg bucht buchfink-belege mit `bezahlung: auslage` und
`gesellschafter_kontakt_id`. Buchfink stellt die Erstattung als Verbindlichkeit
gegenüber dem Gesellschafter ein. Die Erstattung per Überweisung ordnet
buchfink-bank zu. Fertig, wenn der Saldo für diese Auslage null ist.

**GmbH bezahlt eine private Rechnung des Gesellschafters.**

1. Frag, ob die Zahlung ein Versehen war und ob der Gesellschafter sie
   zurückzahlt.
2. Ja: Rückfrage, dann Bankumsatz mit `bankumsatz_buchen`, Zeile
   `konto: 1307` mit `kontakt_id` des Gesellschafters; die Zeile trägt keine
   Umsatzsteuer.
   Vorsteuer gibt es nicht; abziehbar ist nur Steuer für Leistungen an das
   Unternehmen ([§ 15 Abs. 1 Satz 1 Nr. 1 UStG][ustg15]). Die Rückzahlung
   bucht buchfink-bank gegen 1307.
3. Nein: vGA-Verdacht. Lies SKILL.md, Abschnitt 6, und leg den Fall als
   Wertungsfrage vor.

**Gesellschafter hebt Geld ab oder lässt sich Geld überweisen, ohne
Vereinbarung.** Wie oben: Rückzahlung vereinbart, dann 1307; sonst
vGA-Verdacht. Ein Geschäftsführer darf aus dem Vermögen, das das Stammkapital
deckt, keinen Kredit erhalten ([§ 43a GmbHG][gmbhg43a]).

**Verrechnung mit Gehalt oder Tantieme.** Nur mit Vereinbarung. Das Gehalt
bleibt Arbeitslohn und läuft über buchfink-lohn; verrechnet wird nur der
Auszahlungsbetrag. Lege die Vereinbarung als Beleg ab.

## 4. Laufende Kontrolle

- Prüfe beim Monatsabschluss mit `auswertung` (`art: susa` oder `kontoblatt`)
  die Salden von 1307 bis 1309 und 3640 bis 3643 je Gesellschafter.
- Ein Soll-Saldo, der über Monate stehen bleibt, wirkt wie ein zinsloses
  Darlehen. Eine verhinderte Vermögensmehrung, die durch das
  Gesellschaftsverhältnis veranlasst ist, kann eine vGA sein
  ([R 8.5 Abs. 1 KStR 2022][kstr85]). Schlag dem Menschen vor, den Saldo
  auszugleichen oder einen Darlehensvertrag mit Zins zu schließen
  (SKILL.md, Abschnitt 4).
- Ein hoher Haben-Saldo zugunsten eines Gesellschafters ist wirtschaftlich ein
  Gesellschafterdarlehen mit Nachrang in der Insolvenz
  ([§ 39 Abs. 1 Satz 1 Nr. 5 InsO][inso39]). Weise darauf hin.

[hgb246]: https://www.gesetze-im-internet.de/hgb/__246.html
[ustg15]: https://www.gesetze-im-internet.de/ustg_1980/__15.html
[gmbhg43a]: https://www.gesetze-im-internet.de/gmbhg/__43a.html
[inso39]: https://www.gesetze-im-internet.de/inso/__39.html
[kstr85]: https://www.steuerschroeder.de/steuergesetze/kstr/8.5
