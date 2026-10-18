# Bewirtung, Geschenke, Repräsentation

Rechtsstand: 17. Oktober 2026. Wortlaut von EStG, KStG und UStG am 17.10.2026
gelesen. Werte mit Stand Oktober 2026; sie ändern sich durch Gesetz, prüfe sie
für frühere Wirtschaftsjahre am Gesetzestext der Zeit.

## Inhalt

1. Bewirtung
2. Geschenke
3. Pauschalierung nach § 37b EStG
4. Repräsentation
5. Buchungsgruppen und Konten

## 1. Bewirtung

**Regel.** Aufwendungen für die Bewirtung von Personen aus geschäftlichem
Anlass mindern den Gewinn nur zu 70 % der angemessenen Aufwendungen, deren Höhe
und betriebliche Veranlassung nachgewiesen sind
([§ 4 Abs. 5 Satz 1 Nr. 2 Satz 1 EStG][estg4]).

**Nachweis.** Schriftlich: Ort, Tag, Teilnehmer, Anlass, Höhe der Aufwendungen.
In einer Gaststätte genügen Anlass und Teilnehmer; die Rechnung ist beizufügen
(Sätze 2 und 3). Die Aufwendungen sind einzeln und getrennt aufzuzeichnen, sonst
entfällt der Abzug ganz ([§ 4 Abs. 7 EStG][estg4]).

**Vorsteuer.** Voll abziehbar, soweit die Kosten angemessen und nachgewiesen
sind ([§ 15 Abs. 1a Satz 2 UStG][ustg15]). Unangemessene Bewirtungskosten
sperren die Vorsteuer (Satz 1). Ein Verstoß nur gegen die einzelne und
getrennte Aufzeichnung nach § 4 Abs. 7 EStG schließt die Vorsteuer für
angemessene Bewirtungskosten nicht aus ([UStAE 15.6 Abs. 2 Satz 5][ustae156]).
Fehlen Teilnehmer oder Anlass, ist die Vorsteuer strittig: FG
Berlin-Brandenburg vom 09.04.2019, 5 K 5119/18, ließ sie zu ([Haufe][fgbb]);
FG Münster vom 27.04.2020, 5 K 1722/18 U, verlangt den Nachweis in der Form
des § 4 Abs. 5 Satz 1 Nr. 2 Satz 2, 3 EStG ([NWB][fgms]). Wertungsfrage: leg sie
dem Menschen vor.

**Rechnung.** Über 250 € brutto muss die Rechnung die GmbH als
Leistungsempfänger nennen ([§ 14 Abs. 4 Satz 1 Nr. 1 UStG][ustg14]); bis 250 €
genügt eine Kleinbetragsrechnung ([§ 33 UStDV][ustdv33]). In der Gaststätte
gelten für Speisen 7 % und für Getränke 19 %
([§ 12 Abs. 2 Nr. 15 UStG][ustg12], Wortlaut am 17.10.2026); übernimm beide
Sätze aus der Rechnung.

**Verwaltungsanweisung.** Für Bewirtungen ab 2025 gibt es das BMF-Schreiben vom
19.11.2025 (IV C 6 - S 2145/00026/005/033) mit Anforderungen an die Rechnung
der Gaststätte. Sein Wortlaut war am 17.10.2026 nicht abrufbar; die Inhalte nach
Sekundärquellen stehen im Quellenverzeichnis `skills/QUELLEN.md` im Buchfink-Repository unter „Offen“. Zieh sie erst heran, wenn
der Wortlaut geprüft ist.

**Ablauf in Buchfink.**

1. Rechnung und Bewirtungsbeleg (falls schon erstellt) zusammen ablegen; den
   Bewirtungsbeleg als `anhang`.
2. Den Menschen nach Teilnehmern (Namen, bei Gästen Firma) und Anlass fragen.
   Ort und Tag stehen meist auf der Rechnung.
3. `beleg_buchen_mit_rueckfrage` mit Gruppe `bewirtung` und
   `bewirtung {ort, tag, teilnehmer, anlass}`. Buchfink teilt 70 % auf 6640 und
   30 % auf 6644 und zieht die volle Vorsteuer.
4. Fehlen Teilnehmer oder Anlass, buchst du nicht. Ohne Aufzeichnung ist auch
   der abziehbare Teil verloren.

**Wertungsfragen.** Geschäftlicher Anlass oder private Feier; Bewirtung oder
Werbung (etwa Verkostung von eigenen Produkten); Feier nur mit Mitarbeitern
(betrifft auch den Lohn, buchfink-lohn). Leg Optionen und Folgen vor.

## 2. Geschenke

**Regel.** Geschenke an Personen, die nicht Arbeitnehmer der GmbH sind, mindern
den Gewinn nicht. Ausnahme: Die Anschaffungs- oder Herstellungskosten der dem
Empfänger im Wirtschaftsjahr zugewendeten Gegenstände übersteigen insgesamt
50 € nicht ([§ 4 Abs. 5 Satz 1 Nr. 1 EStG][estg4]). Stand: Oktober 2026.

- Es ist eine **Freigrenze**: Wird sie überschritten, ist der ganze Betrag an
  diesen Empfänger nicht abziehbar.
- Anschaffungskosten ohne abziehbare Vorsteuer
  ([§ 9b Abs. 1 EStG][estg9b]).
- Über der Grenze entfällt auch die Vorsteuer
  ([§ 15 Abs. 1a Satz 1 UStG][ustg15]). Wurde sie schon gezogen, wird sie
  berichtigt ([§ 17 Abs. 2 Nr. 5 UStG][ustg17]).
- Empfänger und Aufwendungen sind einzeln aufzuzeichnen
  ([§ 4 Abs. 7 EStG][estg4]).
- Die Grenze von 50 € gilt für Wirtschaftsjahre, die nach dem 31.12.2023
  beginnen ([§ 52 Abs. 6 Satz 10 EStG][estg52]); davor 35 €. Buchfink führt sie
  datiert (`internal/accounting/tax_params.go`).

**Ablauf in Buchfink.**

1. Je Empfänger eine Angabe `geschenk {kontakt_id oder name, anlass}`. Ein
   Präsentkorb für drei Personen sind drei Empfänger.
2. Frag den Menschen, ob der Empfänger in diesem Wirtschaftsjahr schon etwas
   bekommen hat, auch über andere Belege.
3. Buchfink summiert je Empfänger und Wirtschaftsjahr und bucht über der
   Freigrenze auf 6620 ohne Vorsteuer, sonst auf 6610.
4. Gegenstände, die der Empfänger nur betrieblich nutzen kann, führt Buchfink in
   der Gruppe `geschenke_betrieblich` (6625) außerhalb der Freigrenze. Ob ein
   Gegenstand so einzuordnen ist, ist eine Wertungsfrage; die Grundlage ist eine
   Verwaltungsregel, die hier nicht am Wortlaut geprüft ist. Frag im Zweifel
   einen Steuerberater.

**Wertungsfragen.** Geschenk oder Gegenleistung (etwa Provision in Sachform);
Kunde oder Arbeitnehmer; Streuartikel mit Werbeaufdruck.

## 3. Pauschalierung nach § 37b EStG

- Die GmbH kann die Einkommensteuer der Empfänger für Sachzuwendungen und
  Geschenke pauschal mit 30 % übernehmen; Bemessungsgrundlage sind die
  Aufwendungen einschließlich Umsatzsteuer
  ([§ 37b Abs. 1 Satz 1 und 2 EStG][estg37b]; für Körperschaften
  [§ 31 Abs. 1 Satz 3 KStG][kstg31]). Stand: Oktober 2026.
- Die Wahl gilt einheitlich für alle Zuwendungen eines Wirtschaftsjahres
  (Satz 1). Ausgeschlossen über 10.000 € je Empfänger und Jahr oder je
  Zuwendung (Satz 3). Geldgeschenke fallen nicht darunter.
- Die Pauschsteuer gilt als Lohnsteuer und wird in der
  Lohnsteuer-Anmeldung angemeldet ([§ 37b Abs. 4 EStG][estg37b]); der Empfänger
  ist zu unterrichten (Abs. 3 Satz 3).
- Buchfink bildet § 37b in der Gruppe `geschenke` nicht ab. Das SKR04 hat dafür
  eigene Konten (6611, 6621, 6612, 6622). Will der Mensch pauschalieren, leg
  den Fall vor und verweise auf Lohnbüro oder Steuerberater.

## 4. Repräsentation

Aufwendungen für Gästehäuser außerhalb des Betriebsorts sowie für Jagd,
Fischerei, Segel- oder Motorjachten und die zugehörigen Bewirtungen sind nicht
abziehbar ([§ 4 Abs. 5 Satz 1 Nr. 3 und 4 EStG][estg4]), die Vorsteuer darauf
ebenfalls nicht ([§ 15 Abs. 1a Satz 1 UStG][ustg15]). Gruppe
`repraesentation` (6645). Unangemessene Aufwendungen, die die Lebensführung
berühren, sind ebenfalls nicht abziehbar ([§ 4 Abs. 5 Satz 1 Nr. 7 EStG][estg4]);
das ist eine Wertungsfrage.

Bei der GmbH ist die Vorsteuer auf solche Aufwendungen außerdem eine nicht
abziehbare Ausgabe ([§ 10 Nr. 2 KStG][kstg10]).

## 5. Buchungsgruppen und Konten

| Gruppe | Konto | Nicht abziehbarer Teil | Aufzeichnung |
|---|---|---|---|
| `bewirtung` | 6640 | 6644 (30 %) | Ort, Tag, Teilnehmer, Anlass |
| `geschenke` | 6610 | 6620 (alles über der Freigrenze) | Empfänger, Anlass |
| `geschenke_betrieblich` | 6625 | – | Empfänger |
| `repraesentation` | 6645 | ganz, Vorsteuer im Aufwand | – |

Quelle: `internal/accounting/posting_groups.go`. Diese Konten bebuchst du nur
über ihre Gruppe.

[estg4]: https://www.gesetze-im-internet.de/estg/__4.html
[estg9b]: https://www.gesetze-im-internet.de/estg/__9b.html
[estg37b]: https://www.gesetze-im-internet.de/estg/__37b.html
[estg52]: https://www.gesetze-im-internet.de/estg/__52.html
[kstg10]: https://www.gesetze-im-internet.de/kstg_1977/__10.html
[kstg31]: https://www.gesetze-im-internet.de/kstg_1977/__31.html
[ustg12]: https://www.gesetze-im-internet.de/ustg_1980/__12.html
[ustg14]: https://www.gesetze-im-internet.de/ustg_1980/__14.html
[ustg15]: https://www.gesetze-im-internet.de/ustg_1980/__15.html
[ustg17]: https://www.gesetze-im-internet.de/ustg_1980/__17.html
[ustdv33]: https://www.gesetze-im-internet.de/ustdv_1980/__33.html
[ustae156]: https://datenbank.nwb.de/Dokument/378652_15___6/
[fgbb]: https://www.haufe.de/steuern/rechtsprechung/vorsteuerabzug-aus-bewirtungsaufwendungen-trotz-fehlendem-beleg_166_496064.html
[fgms]: https://datenbank.nwb.de/Dokument/831949/
