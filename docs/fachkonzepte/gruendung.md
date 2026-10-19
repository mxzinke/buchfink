# Gründung und Eröffnung

[Fachkonzepte](README.md) · [Belege und Buchungen](beleg-buchungsflow.md)

Anforderungen: GOB-01, GOB-06. Code: `service.FoundationService`,
`service/foundation_capital.go`, `service/foundation_opening.go`,
`service/opening_balance.go`, `internal/accounting/gruendung.go`.

Diese Seite beschreibt den buchhalterischen Teil. Behördenwege und Reihenfolge
der Gründung begleitet der Agent nach
[`skills/buchfink/references/gruendung/`](../../skills/buchfink/references/gruendung/ablauf.md); die
Pflichten daraus stehen im [Pflichtenkalender](pflichten-und-erklaerungen.md).

## Wann die Bücher beginnen

Mit der Beurkundung des Gesellschaftsvertrags entsteht die Vorgesellschaft; sie
ist buchführungspflichtig. Als GmbH entsteht sie mit der Eintragung
([§ 11 GmbHG][gmbhg11]). Die Eröffnungsbilanz steht auf den Tag der
Beurkundung ([§ 242 Abs. 1 HGB][hgb242]); das erste Geschäftsjahr beginnt dort
und ist meist ein Rumpfjahr. Gründungskosten wie Notar und Gericht sind sofort
Aufwand ([§ 248 Abs. 1 Nr. 1 HGB][hgb248]), etwa auf **6825** Rechts- und
Beratungskosten.

Wer mit bestehenden Büchern umsteigt, beginnt mit der Eröffnungsbilanz aus dem
Altsystem. Ihre Eröffnungsbuchungen rechnet
`service.JournalService.BookOpeningBalance`; weder App noch MCP rufen sie auf.

## Kapitalaufbringung

| Rechtsform | Mindestkapital | Vor der Anmeldung einzuzahlen |
|---|---|---|
| GmbH | 25.000 € | je Geschäftsanteil ein Viertel, zusammen mindestens 12.500 € ([§ 7 Abs. 2 GmbHG][gmbhg7]) |
| UG (haftungsbeschränkt) | 1 € | das ganze Stammkapital, keine Sacheinlagen ([§ 5a GmbHG][gmbhg5a]) |

Die Untergrenze der GmbH ist die Hälfte des Mindeststammkapitals, nicht des
vereinbarten. Bei 100.000 € Stammkapital sind 25.000 € einzuzahlen, ein Viertel
je Anteil. `accounting.FoundationRulesFor` rechnet beides und nimmt den höheren
Wert.

## Buchungen

Beispiel: 25.000 € Stammkapital, 12.500 € sofort, der Rest auf Beschluss.

| Schritt | Buchung |
|---|---|
| Zeichnung bei der Beurkundung | **1298** Ausstehende Einlagen, eingefordert 12.500,00 + **2910** Ausstehende Einlagen, nicht eingefordert 12.500,00 an **2900** Gezeichnetes Kapital 25.000,00 |
| Einzahlung | **1800** Bank 12.500,00 an **1298** 12.500,00 |
| Einforderung des Rests | **1298** 12.500,00 an **2910** 12.500,00 |
| Einzahlung des Rests | **1800** Bank 12.500,00 an **1298** 12.500,00 |

Der nicht eingeforderte Teil steht offen vom gezeichneten Kapital abgesetzt
([§ 272 Abs. 1 HGB][hgb272]). Einfordern beschließen die Gesellschafter
([§ 46 Nr. 2 GmbHG][gmbhg46]); der Beschluss ist der Beleg der Einforderung.
Eine Sacheinlage bucht Buchfink nicht von selbst, weil nur der Gründer weiß,
auf welches Konto der Gegenstand gehört.

Zeichnung und Einforderung bucht ein Agent nach Zustimmung im Chat
(`buchung_erfassen`), die Einzahlungen über den Bankumsatz. Gesellschafter,
Anteile und Eintragung ins Handelsregister pflegt der Mensch in der App
(`FoundationService.Save`, `FoundationService.Register`).

Den Tag der Beurkundung führt Buchfink an einer Stelle: als Gründungsdatum der
Unternehmensdaten (`CompanySettings.FoundedOn`). Die Erfassung der Gründung
schreibt ihn dorthin, die Stammdaten berichtigen ihn. Beginn des
Gründungsjahres, Zeichnung, Eröffnungsbilanz, Pflichten und Erklärungen lesen
ihn von dort. Ändert er sich, zieht Buchfink den Beginn des Gründungsjahres
nach, solange es weder festgestellt noch festgeschrieben ist und keine Buchung
vor dem neuen Beginn liegt.

## Unterbilanz

Bleibt das Reinvermögen am Tag der Eintragung hinter dem Stammkapital zurück,
schulden die Gesellschafter die Differenz anteilig. Buchfink rechnet sie aus dem
Journal über alle Geschäftsjahre bis zum Stichtag:

```
Reinvermögen = Aktiva + ausstehende Einlagen − Schulden
Unterdeckung = Stammkapital − Reinvermögen   (nie negativ)
gedeckt      = min(Gründungsaufwand laut Satzung, Unterdeckung)
Haftung      = Unterdeckung − gedeckt, verteilt nach Geschäftsanteilen
```

Ausstehende Einlagen zählen zum Reinvermögen, weil sie Ansprüche gegen die
Gesellschafter sind. Vor der Eintragung rechnet Buchfink auf heute und nennt die
Zahl vorläufig. Die App zeigt die Rechnung unter Unternehmen › Stammdaten im
Block „Gesellschafter & Kapital“, ein Agent liest sie mit `auswertung`
(`art: unterbilanz`).

## Eröffnungsbilanz

Bei einer Gründung stellt Buchfink die Eröffnungsbilanz aus den Buchungen bis
zum Tag der Beurkundung auf (`FoundationService.OpeningBalance`); gebucht wird
dafür nichts Zusätzliches. Stichtag ist das Gründungsdatum der
Unternehmensdaten, derselbe Tag wie für ihre E-Bilanz und die Pflichten im
Kalender (`openingBalanceDate`).

Als PDF zum Unterzeichnen gibt sie die App aus: „Als PDF speichern“ an der
Pflicht unter Unternehmen › Stammdaten, Block „Gesellschafter & Kapital“
(`FoundationService.OpeningBalancePDF`, Bridge `SaveOpeningBalanceAs`, auch im
Prüfermodus). Das PDF nennt Firma, Sitz, Stichtag und die Posten mit Betrag
und hat je Geschäftsführer aus den Unternehmensdaten eine Zeile für Ort und
Datum und eine für die Unterschrift: alle Geschäftsführer unterzeichnen es mit
Datum ([§ 245 HGB][hgb245]).
Buchfink legt es nicht ab: die Zahlen stehen im Journal, und aus ihnen
entsteht dasselbe Dokument neu. Es entsteht nicht, solange ein Befund
besteht:

- Aktiva und Passiva gehen auseinander, oder zum Stichtag ist nichts gebucht.
- Gezeichnetes oder nicht eingefordertes Kapital zum Stichtag weicht von der
  erfassten Gründung ab.
- Firma, Sitz oder Geschäftsführung fehlen in den Unternehmensdaten; nach der
  Eintragung auch Registergericht und -nummer.

Als E-Bilanz ([§ 5b Abs. 1 Satz 4 EStG][estg5b]) geben sie die App bei der
E-Bilanz des ersten Geschäftsjahres und `ebilanz_exportieren` mit
`bilanzart: eroeffnungsbilanz` aus. Der Pflichtenkalender führt Aufstellung
und E-Bilanz der Eröffnungsbilanz als Pflichten
([Pflichten und Erklärungen](pflichten-und-erklaerungen.md)).

Beim Umstieg bucht Buchfink die Eröffnungsbilanz gegen die Saldenvortragskonten
**9000** (Sachkonten), **9008** (Debitoren) und **9009** (Kreditoren). Offene
Posten werden einzeln mit Nummer und Fälligkeit übernommen, damit sie sich im
ersten Jahr ausgleichen lassen. Die Buchungen entstehen alle oder keine.

[estg5b]: https://www.gesetze-im-internet.de/estg/__5b.html
[gmbhg5a]: https://www.gesetze-im-internet.de/gmbhg/__5a.html
[gmbhg7]: https://www.gesetze-im-internet.de/gmbhg/__7.html
[gmbhg11]: https://www.gesetze-im-internet.de/gmbhg/__11.html
[gmbhg46]: https://www.gesetze-im-internet.de/gmbhg/__46.html
[hgb242]: https://www.gesetze-im-internet.de/hgb/__242.html
[hgb245]: https://www.gesetze-im-internet.de/hgb/__245.html
[hgb248]: https://www.gesetze-im-internet.de/hgb/__248.html
[hgb272]: https://www.gesetze-im-internet.de/hgb/__272.html
