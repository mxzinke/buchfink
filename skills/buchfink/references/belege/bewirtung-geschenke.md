# Bewirtung, Geschenke, Repräsentation

Rechtsstand: 17. Oktober 2026. Wortlaut von EStG, KStG und UStG am 17.10.2026
gelesen. Werte mit Stand Oktober 2026.

## Inhalt

1. Bewirtung
2. Geschenke
3. Pauschalierung nach § 37b EStG
4. Repräsentation
5. Buchungsgruppen und Konten

## 1. Bewirtung

- Bewirtung aus geschäftlichem Anlass ist zu 70 % der angemessenen,
  nachgewiesenen Aufwendungen abziehbar
  ([§ 4 Abs. 5 Satz 1 Nr. 2 Satz 1 EStG][estg4]). Nachweis: Ort, Tag,
  Teilnehmer, Anlass und Höhe; in der Gaststätte Anlass und Teilnehmer mit
  der Rechnung (Sätze 2, 3). Ohne einzelne, getrennte Aufzeichnung entfällt
  der Abzug ganz ([§ 4 Abs. 7 EStG][estg4]).
- Vorsteuer voll, soweit angemessen und nachgewiesen
  ([§ 15 Abs. 1a Satz 2 UStG][ustg15]); ein Verstoß nur gegen § 4 Abs. 7 EStG
  schadet nicht ([UStAE 15.6 Abs. 2 Satz 5][ustae156]). Fehlen Teilnehmer oder
  Anlass, ist sie strittig (Wertungsfrage; Urteile in
  [bewirtung-nachweis.md](bewirtung-nachweis.md)).
- Rechnung über 250 € brutto an die GmbH ([§ 14 Abs. 4 Satz 1 Nr. 1 UStG][ustg14]),
  bis 250 € Kleinbetragsrechnung ([§ 33 UStDV][ustdv33]). Speisen 7 %,
  Getränke 19 % ([§ 12 Abs. 2 Nr. 15 UStG][ustg12]).
- Anforderungen an den Bewirtungsbeleg ab 2025 (BMF vom 19.11.2025, Kassensystem
  mit TSE, Trinkgeld): [bewirtung-nachweis.md](bewirtung-nachweis.md).

Ablauf:

1. Rechnung und Bewirtungsbeleg zusammen ablegen, den Bewirtungsbeleg als
   `anhang`.
2. Frag nach Teilnehmern (Namen, bei Gästen Firma) und Anlass.
3. `beleg_buchen_mit_rueckfrage` mit Gruppe `bewirtung` und Angabe
   `bewirtung`; ein Trinkgeld als eigene Position der Gruppe mit Satz 0 und
   `nicht_steuerbar: Trinkgeld`. Fehlen Teilnehmer oder Anlass, buchst du nicht;
   Teilnehmer erfindest du nie.

Wertungsfragen: geschäftlicher Anlass oder private Feier; Bewirtung oder
Werbung (Verkostung eigener Produkte); Feier nur mit Mitarbeitern (auch Lohn,
[lohn/ablauf.md](../lohn/ablauf.md)).

## 2. Geschenke

- Geschenke an Nicht-Arbeitnehmer sind nicht abziehbar, außer die Kosten je
  Empfänger und Wirtschaftsjahr übersteigen 50 € nicht
  ([§ 4 Abs. 5 Satz 1 Nr. 1 EStG][estg4]; Stand: Oktober 2026). Freigrenze:
  Darüber ist der ganze Betrag nicht abziehbar. Kosten ohne abziehbare
  Vorsteuer ([§ 9b Abs. 1 EStG][estg9b]).
- Über der Grenze entfällt die Vorsteuer ([§ 15 Abs. 1a Satz 1 UStG][ustg15]);
  eine schon gezogene wird berichtigt ([§ 17 Abs. 2 Nr. 5 UStG][ustg17]).
- Empfänger und Aufwendungen einzeln aufzeichnen ([§ 4 Abs. 7 EStG][estg4]).

Ablauf:

1. Je Empfänger eine Angabe `geschenk`; ein Präsentkorb für drei Personen sind
   drei Empfänger.
2. Frag, ob der Empfänger in diesem Wirtschaftsjahr schon etwas bekam, auch
   über andere Belege.
3. Kann der Empfänger den Gegenstand ausschließlich betrieblich nutzen, greift
   das Abzugsverbot nicht ([R 4.10 Abs. 2 Satz 4 EStR][estr410]): Gruppe
   `geschenke_betrieblich`, außerhalb der Freigrenze. Ob das zutrifft, ist
   eine Wertungsfrage; im Zweifel Steuerberater.

Wertungsfragen: Geschenk oder Gegenleistung (Provision in Sachform); Kunde
oder Arbeitnehmer; Streuartikel mit Werbeaufdruck.

## 3. Pauschalierung nach § 37b EStG

Die Pauschsteuer von 30 % auf Sachzuwendungen, einheitlich je Jahr und als
Lohnsteuer anzumelden ([§ 37b Abs. 1, 4 EStG][estg37b];
[§ 31 Abs. 1 Satz 3 KStG][kstg31]; Stand: Oktober 2026), bildet Buchfink nicht
ab (SKR04 6611, 6612, 6621, 6622). Will der Mensch pauschalieren, leg den Fall
vor und verweise auf Lohnbüro oder Steuerberater.

## 4. Repräsentation

Gästehäuser außerhalb des Betriebsorts, Jagd, Fischerei, Jachten und die
zugehörigen Bewirtungen sind nicht abziehbar
([§ 4 Abs. 5 Satz 1 Nr. 3, 4 EStG][estg4]), die Vorsteuer ebenfalls nicht
([§ 15 Abs. 1a Satz 1 UStG][ustg15]): Gruppe `repraesentation`. Ebenso
unangemessene Aufwendungen, die die Lebensführung berühren
([§ 4 Abs. 5 Satz 1 Nr. 7 EStG][estg4]; Wertungsfrage). Buchung der nicht
abziehbaren Vorsteuer:
[vorsteuer.md](vorsteuer.md#5-nicht-abziehbare-vorsteuer-in-der-buchung).

## 5. Buchungsgruppen und Konten

| Gruppe | Konto | Nicht abziehbarer Teil | Aufzeichnung |
|---|---|---|---|
| `bewirtung` | 6640 | 6644 (30 %) | Ort, Tag, Teilnehmer, Anlass |
| `geschenke` | 6610 | 6620 (alles über der Freigrenze) | Empfänger, Anlass |
| `geschenke_betrieblich` | 6625 | – | Empfänger |
| `repraesentation` | 6645 | ganz, Vorsteuer im Aufwand | – |

Diese Konten bebuchst du nur über ihre Gruppe; nur sie hält Empfänger,
Aufzeichnung und Freigrenze fest.

[estg4]: https://www.gesetze-im-internet.de/estg/__4.html
[estg9b]: https://www.gesetze-im-internet.de/estg/__9b.html
[estg37b]: https://www.gesetze-im-internet.de/estg/__37b.html
[kstg31]: https://www.gesetze-im-internet.de/kstg_1977/__31.html
[ustg12]: https://www.gesetze-im-internet.de/ustg_1980/__12.html
[ustg14]: https://www.gesetze-im-internet.de/ustg_1980/__14.html
[ustg15]: https://www.gesetze-im-internet.de/ustg_1980/__15.html
[ustg17]: https://www.gesetze-im-internet.de/ustg_1980/__17.html
[ustdv33]: https://www.gesetze-im-internet.de/ustdv_1980/__33.html
[ustae156]: https://www.bundesfinanzministerium.de/Content/DE/Downloads/BMF_Schreiben/Steuerarten/Umsatzsteuer/Umsatzsteuer-Anwendungserlass/Umsatzsteuer-Anwendungserlass-aktuell.html
[estr410]: https://esth.bundesfinanzministerium.de/esth/2025/A-Einkommensteuergesetz/II-Einkommen-2-24b/3-Gewinn-4-7i/Paragraf-4/inhalt.html
