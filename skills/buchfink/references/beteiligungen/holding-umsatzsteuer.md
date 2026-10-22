# Umsatzsteuer der Holding

Rechtsstand: 17. Oktober 2026. Wortlaut des UStG am 17.10.2026 auf
gesetze-im-internet.de gelesen. UStAE Abschnitte 2.3 und 15.22 in der Fassung
„VZ 2010–2026 (aktuell)“ über die NWB-Datenbank gelesen; der amtliche UStAE
war nicht abrufbar.

Ob eine GmbH mit Beteiligungen Unternehmerin ist und Vorsteuer abziehen darf,
ist eine **Wertungsfrage**: Leg die Formen aus Abschnitt 2 mit ihrer Folge
vor; der Mensch entscheidet, bei Zweifeln mit Steuerberater. Buchfink geht von
einer Unternehmerin aus
([grundlagen/grenzen.md](../grundlagen/grenzen.md#1-was-buchfink-nicht-abbildet)).

## Inhalt

1. Unternehmerin oder nicht
2. Drei Formen der Holding
3. Vorsteuer aus Kosten der Beteiligung
4. Leistungen an Tochtergesellschaften
5. Darlehen an Tochtergesellschaften
6. Verkauf einer Beteiligung
7. Organschaft

## 1. Unternehmerin oder nicht

- Erwerben, Halten und Veräußern von Beteiligungen ist keine
  unternehmerische Tätigkeit; Dividenden sind kein Entgelt
  ([UStAE 2.3 Abs. 2 Satz 1 bis 3][ustae23]).
- Unternehmerisch wird es bei gewerbsmäßigem Handel, wenn die Beteiligung der
  eigenen unternehmerischen Tätigkeit dient, oder wenn die Holding durch
  entgeltliche Leistungen (administrativ, finanziell, kaufmännisch, technisch)
  in die Verwaltung der Tochter eingreift
  ([UStAE 2.3 Abs. 3 Satz 5 ff.][ustae23]).
- Zum Unternehmen gehört eine Beteiligung nur bei erkennbarem, objektivem
  wirtschaftlichem Zusammenhang mit der unternehmerischen Tätigkeit
  ([UStAE 2.3 Abs. 4 Satz 1 bis 3][ustae23]).

## 2. Drei Formen der Holding

| Form | Merkmal | Folge | Fundstelle |
|---|---|---|---|
| Finanzholding | hält und verwaltet nur Beteiligungen, keine Leistungen gegen Entgelt | keine Unternehmerin, kein Vorsteuerabzug | [UStAE 2.3 Abs. 3 Satz 2][ustae23] |
| Führungs- oder Funktionsholding | greift durch entgeltliche Leistungen aktiv in das Tagesgeschäft der Töchter ein | Unternehmerin; Beteiligungen gehören zum Unternehmen | [UStAE 2.3 Abs. 3 Satz 3, 5 ff.][ustae23] |
| gemischte Holding | leitet einige Töchter, hält andere nur | unternehmerischer und nichtunternehmerischer Bereich | [UStAE 2.3 Abs. 3 Satz 4][ustae23] |

- Eine GmbH mit eigenem operativem Geschäft ist Unternehmerin; ihre
  Beteiligungen gehören nur bei dem Zusammenhang aus Abschnitt 1 zum
  Unternehmen ([UStAE 2.3 Abs. 2 Satz 5, 6][ustae23]).
- Reine Finanzholding: Grenze, Buchfinks Voranmeldung setzt eine
  Unternehmerin voraus. Ob überhaupt Voranmeldungen abzugeben sind, klärt der
  Mensch mit dem Steuerberater.

## 3. Vorsteuer aus Kosten der Beteiligung

- Gehört die Beteiligung zum Unternehmen, ist die Vorsteuer aus Kosten in
  ihrem Zusammenhang abziehbar ([UStAE 15.22 Abs. 1 Satz 1, 2][ustae1522];
  [§ 15 Abs. 1 UStG][ustg15]).
- Mit Beteiligungen im nichtunternehmerischen Bereich wird die Vorsteuer aus
  gemeinsamen Kosten (Verwaltung, allgemeine Beratung, Steuerberatung) nach
  den Ausgangsumsätzen aufgeteilt, in die sie eingehen
  ([UStAE 15.22 Abs. 1 Satz 3, 4][ustae1522]).
- Hängt eine Eingangsleistung direkt mit Umsätzen einer Tochter zusammen,
  ist die Vorsteuer bei der Holding nicht abziehbar
  ([UStAE 15.22 Abs. 1 Satz 6][ustae1522] mit BFH vom 15.02.2023, XI R 24/22).
- Buche mit `beleg_buchen_mit_rueckfrage` und dem abziehbaren Anteil je
  Position ([belege/vorsteuer.md, Abschnitt 4](../belege/vorsteuer.md#4-aufteilung));
  der Grund nennt Maßstab und Rechnung, z. B. „Beratung Kauf Beteiligung X;
  40 % den eigenen Umsätzen zuzuordnen“. Der Maßstab ist eine Wertungsfrage.

## 4. Leistungen an Tochtergesellschaften

- Entgeltliche Leitungs-, Buchhaltungs-, IT- oder Personalleistungen machen
  die Holding zur Unternehmerin; sie stellt Rechnungen mit Umsatzsteuer aus
  ([rechnungen/ablauf.md](../rechnungen/ablauf.md)).
- Die Führungsrolle allein reicht nicht ([UStAE 2.3 Abs. 3][ustae23]);
  lege den schriftlichen Leistungsvertrag als Beleg ab.
- Wertungsfrage: Angemessenheit des Entgelts; ein unangemessenes Entgelt
  kann eine verdeckte Gewinnausschüttung oder Einlage sein
  ([gesellschafter/vga-einlage.md](../gesellschafter/vga-einlage.md)).

## 5. Darlehen an Tochtergesellschaften

- Zinsen sind steuerfreie Umsätze ohne Steuerausweis
  ([bank/darlehen.md, Abschnitt 3](../bank/darlehen.md#3-zinsen-umsatzsteuer-gewerbesteuer-stichtag)).
  Vorsteuer aus Kosten dieser Umsätze ist ausgeschlossen
  ([§ 15 Abs. 2 Satz 1 Nr. 1 UStG][ustg15]), gemeinsame Kosten werden
  aufgeteilt ([§ 15 Abs. 4 UStG][ustg15]).
- Den Verzicht auf die Steuerfreiheit bei Krediten an Unternehmer
  ([§ 9 Abs. 1 UStG][ustg9]) bildet Buchfink nicht ab
  ([grundlagen/grenzen.md](../grundlagen/grenzen.md#1-was-buchfink-nicht-abbildet)).
- Prüfe die Zinsen in der Voranmeldung mit `erklaerung`
  ([monatsabschluss/ustva.md](../monatsabschluss/ustva.md)).

## 6. Verkauf einer Beteiligung

- Das Veräußern ist nur unternehmerisch, wenn die Beteiligung zum Unternehmen
  gehört ([UStAE 15.22 Abs. 2 Satz 1, 2][ustae1522]).
- Vorsteuer aus Kosten des Verkaufs (Makler, Berater, Notar) ist nicht
  abziehbar: außerhalb des Unternehmens mangels steuerbaren Umsatzes, im
  Unternehmen wegen [§ 4 Nr. 8 Buchst. e, f UStG][ustg4]
  ([UStAE 15.22 Abs. 2 Satz 3 bis 5][ustae1522]). Den Verzicht auf die
  Steuerfreiheit ([§ 9 Abs. 1 UStG][ustg9]) bildet Buchfink nicht ab;
  Steuerberater.

## 7. Organschaft

Eine Tochter, die finanziell, wirtschaftlich und organisatorisch in die
Holding eingegliedert ist, ist umsatzsteuerlich nicht selbstständig; Leistungen
zwischen beiden sind Innenleistungen ([§ 2 Abs. 2 Nr. 2 UStG][ustg2]).
Buchfink bildet die Organschaft nicht ab. Bei einer Führungsholding mit
Mehrheitsbeteiligung: weise darauf hin und empfiehl den Steuerberater.

[ustae23]: https://datenbank.nwb.de/Dokument/378652_2___3/
[ustae1522]: https://datenbank.nwb.de/Dokument/378652_15___22/
[ustg2]: https://www.gesetze-im-internet.de/ustg_1980/__2.html
[ustg4]: https://www.gesetze-im-internet.de/ustg_1980/__4.html
[ustg9]: https://www.gesetze-im-internet.de/ustg_1980/__9.html
[ustg15]: https://www.gesetze-im-internet.de/ustg_1980/__15.html
