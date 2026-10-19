---
name: buchfink-steuererklaerungen
description: "Jahressteuererklärungen einer GmbH oder UG mit Buchfink vorbereiten, prüfen und abgeben lassen, mit oder ohne Berater: Körperschaftsteuer mit Solidaritätszuschlag und Erklärung zum steuerlichen Einlagekonto, Gewerbesteuer mit Hinzurechnungen und Kürzungen, Umsatzsteuer-Jahreserklärung, E-Bilanz als XBRL-Datei für einen Online-Dienst wie eBilanz+, Betrag der Steuerrückstellung. Lade den Skill, wenn fristen eine Jahreserklärung oder E-Bilanz meldet; wenn die Steuerrückstellung für den Jahresabschluss berechnet werden soll; bei Fristverlängerung, Verspätungszuschlag oder Zinsen; wenn Werte in Mein ELSTER übertragen oder die E-Bilanz übermittelt werden soll; wenn nicht abziehbare Ausgaben, verdeckte Gewinnausschüttung, Spenden oder Verlustvortrag in die Erklärung einfließen; wenn ein Steuerbescheid, Messbescheid oder Vorauszahlungsbescheid eingeht; wenn ein Einspruch erwogen wird. Voranmeldung und ZM regelt buchfink-monatsabschluss, die Buchung der Rückstellung buchfink-jahresabschluss, Zahlungen buchfink-bank."
license: EUPL-1.2
compatibility: "Benötigt die laufende Buchfink-App mit eingeschaltetem MCP-Server und einem zugelassenen Mandanten. Setzt buchfink-grundlagen voraus. Senden kann nur der Mensch: in Mein ELSTER, die E-Bilanz über einen Online-Dienst mit ELSTER-Anbindung."
metadata:
  buchfink-rechtsstand: "2026-10-17"
  buchfink-werkzeugstand: "Buchfink 0.3.0"
---

# Buchfink: Steuererklärungen

Setzt buchfink-grundlagen voraus; lade ihn, falls noch nicht geschehen.

Eine GmbH oder UG gibt für jedes Jahr die Umsatzsteuer-, die Körperschaftsteuer-
und die Gewerbesteuererklärung ab, dazu die Erklärung zum steuerlichen
Einlagekonto, und übermittelt die E-Bilanz. Buchfink rechnet die Erklärungen
aus den Büchern, zeigt sie formularnah und exportiert die E-Bilanz als
XBRL-Datei; es übermittelt nicht. Du prüfst die Rechnung, trägst nach
Rückfrage ein, was nicht in den Büchern steht, und bereitest die Abgabe vor.
Der Mensch entscheidet die Wertungsfragen, gibt die Zahlen frei, überträgt sie
in Mein ELSTER und sendet.

Buchfink rechnet die Überleitung vom Jahresüberschuss zum zu versteuernden
Einkommen (nicht abziehbare Aufwendungen, § 8b KStG aus Konten und
Anlagenkartei, Spendenabzug, Verlustabzug mit Mindestbesteuerung), die
Hinzurechnungen und Kürzungen der Gewerbesteuer, Messbetrag, Steuer und das
steuerliche Einlagekonto. Jede Zeile trägt `status` (`auto`, `vorgeschlagen`,
`eingabe`, `offen`, `entfaellt`), `zustaendig` und `herkunft`. Was nicht in
den Büchern steht, steht als `offen` oder `vorgeschlagen`: Verlustvortrag und
Einlagekonto laut Bescheid, verdeckte Gewinnausschüttung, Beteiligungsquoten,
Betriebsstätten. Die Rechtslage in den references dient dir zur Prüfung, nicht
zum Nachrechnen. Sag nie zu, dass eine Erklärung steuerlich richtig ist; bei
Zweifeln empfiehl einen Steuerberater (buchfink-grundlagen, Grenzen).

| Thema | Skill |
|---|---|
| Fristen, Weg und Nachweis jeder Pflicht | buchfink-grundlagen ([grundlagen/pflichten.md](../grundlagen/pflichten.md)) |
| Umsatzsteuer-Voranmeldung, ZM | buchfink-monatsabschluss |
| Steuerrückstellung buchen, Bilanz und GuV aufstellen | buchfink-jahresabschluss |
| Zahlung, Erstattung, Säumniszuschlag, Zinsen buchen | buchfink-bank |
| Dividenden, § 8b KStG, Kürzung nach § 9 Nr. 2a GewStG | buchfink-beteiligungen |
| Verdeckte Gewinnausschüttung erkennen, Einlagen erfassen | buchfink-gesellschafter |

Werkzeuge: `status`, `aufgaben`, `fristen`, `erklaerung`,
`erklaerung_eingabe` (Rückfrage), `erklaerung_exportieren`,
`ebilanz_exportieren`, `pflicht_vorbereitet`, `anlass_erfassen` (Rückfrage),
`auswertung`, `konten`, `buchungen_suchen`, `belege_suchen`, `beleg_ablegen`.

## 1. Die Erklärungen

| Erklärung | Pflicht | `erklaerung` `art` | Abgabe | Lies |
|---|---|---|---|---|
| Umsatzsteuererklärung für das Kalenderjahr | [§ 18 Abs. 3 UStG][ustg18] | `ust_jahr` | Mein ELSTER, „Umsatzsteuererklärung“ | [ust-jahr.md](ust-jahr.md) |
| Körperschaftsteuererklärung KSt 1, darin die Erklärung zum steuerlichen Einlagekonto | [§ 31 Abs. 1a KStG][kstg31], [§ 27 Abs. 2 Satz 4 KStG][kstg27] | `kst` | Mein ELSTER, „Körperschaftsteuererklärung (KSt 1)“ mit Anlagen | [kst.md](kst.md) |
| Gewerbesteuererklärung GewSt 1 A (Messbetrag) | [§ 14a GewStG][gewstg14a] | `gewst` | Mein ELSTER, „Gewerbesteuererklärung (GewSt 1 A)“; bei Zerlegung GewSt 1 D | [gewst.md](gewst.md) |
| E-Bilanz | [§ 5b Abs. 1 EStG][estg5b] | – (`ebilanz_exportieren`) | XBRL-Datei aus Buchfink, importiert und gesendet in einem Online-Dienst wie eBilanz+ oder eBilanz-Online; Mein ELSTER hat kein Formular | [ebilanz.md](ebilanz.md) |

Alle Erklärungen gehen elektronisch nach amtlichem Datensatz an das Finanzamt.
Papier gibt es nur auf Antrag wegen unbilliger Härte. `erklaerung` zeigt jede
Erklärung formularnah mit Formular, Zeile, Bezeichnung, Wert und Herkunft; die
App zeigt dieselbe Ansicht als Eingabehilfe.

Zeitraum: Gib `zeitraum` als Kalenderjahr an (`2025`). Bei abweichendem
Wirtschaftsjahr gilt der Gewinn in dem Kalenderjahr als bezogen, in dem das
Wirtschaftsjahr endet ([§ 7 Abs. 4 Satz 2 KStG][kstg7],
[§ 10 Abs. 2 GewStG][gewstg10]). Die Umsatzsteuer folgt immer dem
Kalenderjahr ([§ 16 Abs. 1 Satz 2 UStG][ustg16]). Prüfe in der Antwort von
`erklaerung`, welches Wirtschaftsjahr Buchfink zugrunde legt.

## 2. Ablauf

1. **Stand klären.** `status` (Geschäftsjahre mit `stand`) und `aufgaben`.
   Die Erklärungen bauen auf dem Jahresabschluss auf. Ist er noch offen, frag
   den Menschen, ob du zuerst den Betrag der Steuerrückstellung liefern sollst
   (Schritt 6). Fertig, wenn Jahr, Abschlussstand sowie Fälligkeit und
   Zuständigkeit jeder Erklärung aus `fristen` feststehen (Abschnitt 3).
2. **Unterlagen sammeln.** Bescheide des Vorjahres (Körperschaftsteuer,
   Gewerbesteuermessbetrag, Verlustfeststellungen, Einlagekonto),
   Vorauszahlungsbescheide, Steuerbescheinigungen über Kapitalertragsteuer,
   Zuwendungsbestätigungen, Hebesatz der Gemeinde, Gesellschafterbeschlüsse,
   Liste der Einlagen. Suche mit `belege_suchen`, lege Neues mit
   `beleg_ablegen` ab, frag nach Fehlendem. Fertig, wenn jede Unterlage
   vorliegt oder als fehlend benannt ist.
3. **Umsatzsteuer-Jahreserklärung.** `erklaerung` (`art: ust_jahr`). Gleiche
   sie mit der Summe der Voranmeldungen und den Konten ab. Fertig, wenn jede
   Abweichung erklärt ist. [ust-jahr.md](ust-jahr.md)
4. **Körperschaftsteuer.** `erklaerung` (`art: kst`, erst mit
   `nur_offen: true`). Prüfe jede Zeile `offen` oder `vorgeschlagen` und jeden
   Eintrag in `befunde` nach [kst.md](kst.md); die Zeilen `auto`
   prüfst du über `herkunft` gegen `auswertung` auf Plausibilität. Angaben aus
   Bescheid, Beschluss oder Beleg trägst du nach Rückfrage mit
   `erklaerung_eingabe` ein (`art`, `jahr`, `schluessel` der Zeile, `wert`,
   `quelle`, `beleg_id`). Bei `zustaendig: mensch` bleibt sie ein Vorschlag,
   bis der Mensch sie in der App bestätigt; was er gesetzt oder bestätigt hat,
   ändert nur er. Fertig, wenn keine Pflichtzeile mehr `offen` ist, kein
   blockierender Befund bleibt (`abgabereif: true`) und die Wertungsfragen
   gesammelt sind.
5. **Gewerbesteuer.** `erklaerung` (`art: gewst`) ebenso. Den Gewinn übernimmt
   Buchfink aus der Körperschaftsteuer; Hinzurechnungen nach § 8 Nr. 1 rechnet
   es aus den Konten. Prüfe die Zuordnung von Miete, Leasing und Lizenzen nach
   [gewst.md](gewst.md). Fertig wie Schritt 4 und wenn der Hebesatz
   zum Bescheid der Gemeinde passt.
6. **Steuerrückstellung.** Vergleiche den Vorschlag von `abschlussbuchung`
   (`art: steuerrueckstellung`, `nur_vorschau: true`) mit Schritt 4 und 5. Der
   Vorschlag kennt Verlustvorträge, Hinzurechnungen, Kürzungen und eine vGA nur
   als Eingabe (`verlustvortrag_kst`, `verlustvortrag_gewst`,
   `hinzurechnungen`, `kuerzungen`, `korrekturen`); nimm die Werte aus den
   Erklärungen. Nenne je Steuerart Steuer, Vorauszahlungen und Rückstellung und
   übergib an buchfink-jahresabschluss. Fertig, wenn der Mensch die Beträge
   bestätigt hat.
7. **E-Bilanz.** `ebilanz_exportieren` (`jahr`; `format: xbrl` ist Vorgabe)
   prüft und schreibt die XBRL-Datei in den Exportordner. Fertig, wenn kein
   blockierender Befund bleibt und der Mensch den Online-Dienst gewählt hat.
   [ebilanz.md](ebilanz.md)
8. **Freigabe.** Lege je Erklärung vor: Kennzahlen, Überleitung, Abweichungen
   zum Vorschlag aus Buchfink, Wertungsfragen mit Grundlage und Optionen.
   Fertig, wenn der Mensch die Zahlen ausdrücklich freigibt.
9. **Abgeben.** Melde jede Erklärung mit `pflicht_vorbereitet` als
   vorbereitet. Dann nach [abgabe.md](abgabe.md): Der Mensch
   überträgt Umsatzsteuer, Körperschaftsteuer und Gewerbesteuer mit deiner
   Tabelle in Mein ELSTER und sendet; die E-Bilanz importiert, prüft und
   sendet er im Online-Dienst. Mit Steuerberater
   übergibst du ihm die Unterlagen (Abschnitt 3). Fertig, wenn jede Erklärung
   ein Transferticket hat.
10. **Transferticket erfassen lassen (nur App).** Nenne dem Menschen Erklärung,
    Jahr, Zeitpunkt und Transferticket. Fertig, wenn `erklaerung` den Status
    `uebermittelt` zeigt und `fristen` die Pflicht als erledigt führt.
11. **Bescheid prüfen.** Bescheid mit `beleg_ablegen` ablegen, mit der
    Erklärung abgleichen, Einspruchsfrist und Zahltermin nennen. Die Zahlung
    ordnet buchfink-bank zu. Fertig, wenn jede Abweichung erklärt ist.
    [bescheide.md](bescheide.md)

## 3. Fristen und Steuerberater

`fristen` nennt Fälligkeit, Stand und Zuständigkeit jeder Erklärung. Frist,
Weg, Nachweis und Folgen stehen je Pflicht in buchfink-grundlagen:
[kst-erklaerung](../grundlagen/pflichten.md#kst-erklaerung),
[gewst-erklaerung](../grundlagen/pflichten.md#gewst-erklaerung),
[ust-erklaerung](../grundlagen/pflichten.md#ust-erklaerung),
[ebilanz](../grundlagen/pflichten.md#ebilanz),
[kst-vorauszahlung](../grundlagen/pflichten.md#kst-vorauszahlung),
[gewst-vorauszahlung](../grundlagen/pflichten.md#gewst-vorauszahlung).
Übergangsjahre, Fristende, Zuschläge und Zinsen:
[fristen-zuschlaege.md](fristen-zuschlaege.md).

Die Fälligkeit hängt davon ab, ob ein Steuerberater mit der Erstellung
beauftragt ist ([§ 149 Abs. 3 AO][ao149]). Das sagt das Stammdatum
„steuerliche Beratung“, das nur der Mensch in der App pflegt. Ist ein Berater
eingetragen, aber nicht beauftragt, bitte den Menschen um die Korrektur.

Mit Steuerberater erstellt und sendet der Berater die Erklärungen, die ihm das
Stammdatum zuweist. Du stellst ihm Saldenliste, Bilanz, GuV, Anlagenspiegel,
Überleitung, die Liste Abschlussposten aus buchfink-jahresabschluss und die
offenen Wertungsfragen zusammen (Schritte 2 bis 7). Das Transferticket, das er
nennt, erfasst der Mensch in der App.

Ist eine Frist verstrichen, sag es sofort, bereite die Erklärung vorrangig vor
und nenne dem Menschen die rückwirkende Fristverlängerung
([§ 109 Abs. 1 Satz 2 AO][ao109]) als Möglichkeit.

## 4. Entscheidungsregeln

- Rechne auf dem Stand des aufgestellten Abschlusses. Ändert sich danach eine
  Buchung, rechne neu und exportiere neu (`zahlen_hash`).
- Verlustvortrag, Anfangsbestand des Einlagekontos, Hebesatz und
  Vorauszahlungen übernimmst du nur aus einem Bescheid oder einer Angabe des
  Menschen.
- Die Sätze wählt Buchfink nach dem Veranlagungszeitraum. Zur Prüfung:
  Körperschaftsteuer 15 % bis 2027, 14 % für 2028, 13 % für 2029, 12 % für
  2030, 11 % für 2031, 10 % ab 2032 ([§ 23 Abs. 1 KStG][kstg23]);
  Solidaritätszuschlag 5,5 % der festgesetzten Körperschaftsteuer
  ([§ 3 Abs. 1 Nr. 1 SolZG][solzg3], [§ 4 Satz 1 SolZG][solzg4]);
  Steuermesszahl 3,5 % ([§ 11 Abs. 2 GewStG][gewstg11]). Stand: Oktober 2026.
- Der Mensch sendet erst, wenn die Prüfung in Mein ELSTER ohne Fehler endet
  und jeder von ELSTER berechnete Wert zu deiner freigegebenen Tabelle passt.
  Jede Erklärung wird genau einmal gesendet.
- Erkennst du eine Wertungsfrage, leg sie vor, bevor du die Zahl einsetzt.

## 5. Wertungsfragen

Leite jede mit „Wertungsfrage:“ ein, nenne Grundlage, Optionen und Folgen.

- Liegt eine verdeckte Gewinnausschüttung oder verdeckte Einlage vor
  ([§ 8 Abs. 3 KStG][kstg8]; buchfink-gesellschafter)?
- Ist eine Ausgabe nicht abziehbar (Geschenk, Bewirtung, Strafe), und in
  welcher Höhe?
- Ist ein Spendenempfänger begünstigt, oder ist die Spende durch einen
  Gesellschafter veranlasst ([§ 9 Abs. 1 Satz 1 Nr. 2 KStG][kstg9])?
- Welche Miete, Pacht oder Lizenz fällt unter welchen Buchstaben von
  [§ 8 Nr. 1 GewStG][gewstg8]?
- Hat die GmbH Betriebsstätten in mehreren Gemeinden (Zerlegung,
  [§ 28 GewStG][gewstg28])?
- Wurden innerhalb von fünf Jahren mehr als 50 % der Anteile an einen Erwerber
  übertragen ([§ 8c Abs. 1 Satz 1 KStG][kstg8c])? Dann sind nicht genutzte
  Verluste nicht mehr abziehbar; Ausnahmen prüft ein Steuerberater.
- Weichen Ansätze der Steuerbilanz von der Handelsbilanz ab
  ([§ 5b Abs. 1 Satz 2 EStG][estg5b])?
- Lohnt ein Einspruch?

## 6. Nur der Mensch

- Zahlen freigeben.
- Jede Erklärung in Mein ELSTER senden, die E-Bilanz im Online-Dienst;
  das ELSTER-Zertifikat der GmbH bedienen.
- Online-Dienst für die E-Bilanz wählen, gegebenenfalls Härtefallantrag
  stellen.
- Transferticket in der App erfassen.
- Fristverlängerung, Einspruch, Aussetzung der Vollziehung beantragen.
- Einen Steuerberater beauftragen und das Stammdatum „steuerliche Beratung“
  in der App pflegen.

## 7. Stolperfallen

- Körperschaftsteuer, Solidaritätszuschlag und Gewerbesteuer mindern das
  Einkommen nicht, ebenso wenig ihre Nebenleistungen
  ([§ 10 Nr. 2 KStG][kstg10], [§ 4 Abs. 5b EStG][estg4]). In der GuV stehen sie
  als Aufwand; die Überleitung rechnet sie hinzu.
- Eine GmbH hat bei der Gewerbesteuer keinen Freibetrag; den gibt es nur für
  natürliche Personen und Personengesellschaften
  ([§ 11 Abs. 1 Satz 3 Nr. 1 GewStG][gewstg11]).
- Gewerbeertrag auf volle 100 € abrunden ([§ 11 Abs. 1 Satz 3 GewStG][gewstg11]).
- Kapitalertragsteuer wird nur mit Steuerbescheinigung angerechnet
  ([§ 36 Abs. 2 Nr. 2 Satz 2 EStG][estg36] über [§ 31 Abs. 1 KStG][kstg31]).
- Im Gründungsjahr ist der Zeitraum der Steuerpflicht maßgebend
  ([§ 7 Abs. 3 Satz 3 KStG][kstg7], [§ 14 Satz 3 GewStG][gewstg14],
  [§ 16 Abs. 3 UStG][ustg16]); auch die Eröffnungsbilanz ist als E-Bilanz zu
  übermitteln ([§ 5b Abs. 1 Satz 4 EStG][estg5b]).
- Mein ELSTER führt kein Formular für die E-Bilanz (Formularliste geprüft am
  17.10.2026). Ob der Online-Dienst Kontennachweis und Stammdaten aus der
  XBRL-Datei vollständig übernimmt, ist nicht dokumentiert; prüft es nach
  jedem Import ([ebilanz.md](ebilanz.md)).
- Ein elektronisch bereitgestellter Bescheid gilt am vierten Tag als bekannt
  gegeben ([§ 122a Abs. 4 AO][ao122a]); ab dann läuft die Einspruchsfrist von
  einem Monat ([§ 355 Abs. 1 AO][ao355]). Ein Einspruch hält die Zahlung nicht
  auf ([§ 361 Abs. 1 AO][ao361]).

## Referenzen

- [kst.md](kst.md): Lies für die Überleitung zum zu
  versteuernden Einkommen, nicht abziehbare Ausgaben, Spenden, Verlustabzug,
  Anrechnung, Vorauszahlungen und das steuerliche Einlagekonto.
- [gewst.md](gewst.md): Lies für Gewerbeertrag,
  Hinzurechnungen, Kürzungen, Gewerbeverlust, Messbetrag, Hebesatz und
  Zerlegung.
- [ust-jahr.md](ust-jahr.md): Lies für den Abgleich der
  Jahreserklärung mit den Voranmeldungen und die Abschlusszahlung.
- [ebilanz.md](ebilanz.md): Lies vor jedem XBRL-Export
  und wenn der Übermittlungsweg der E-Bilanz offen ist.
- [fristen-zuschlaege.md](fristen-zuschlaege.md): Lies bei
  Übergangsjahren, Fristverlängerung, Verspätungszuschlag und Zinsen.
- [abgabe.md](abgabe.md): Lies, bevor der Mensch Werte in
  Mein ELSTER überträgt, und zum Organisationszertifikat.
- [bescheide.md](bescheide.md): Lies, wenn ein Bescheid
  oder Vorauszahlungsbescheid eingeht oder ein Einspruch erwogen wird.

[ao109]: https://www.gesetze-im-internet.de/ao_1977/__109.html
[ao122a]: https://www.gesetze-im-internet.de/ao_1977/__122a.html
[ao149]: https://www.gesetze-im-internet.de/ao_1977/__149.html
[ao355]: https://www.gesetze-im-internet.de/ao_1977/__355.html
[ao361]: https://www.gesetze-im-internet.de/ao_1977/__361.html
[estg4]: https://www.gesetze-im-internet.de/estg/__4.html
[estg5b]: https://www.gesetze-im-internet.de/estg/__5b.html
[estg36]: https://www.gesetze-im-internet.de/estg/__36.html
[gewstg8]: https://www.gesetze-im-internet.de/gewstg/__8.html
[gewstg10]: https://www.gesetze-im-internet.de/gewstg/__10.html
[gewstg11]: https://www.gesetze-im-internet.de/gewstg/__11.html
[gewstg14]: https://www.gesetze-im-internet.de/gewstg/__14.html
[gewstg14a]: https://www.gesetze-im-internet.de/gewstg/__14a.html
[gewstg28]: https://www.gesetze-im-internet.de/gewstg/__28.html
[kstg7]: https://www.gesetze-im-internet.de/kstg_1977/__7.html
[kstg8]: https://www.gesetze-im-internet.de/kstg_1977/__8.html
[kstg8c]: https://www.gesetze-im-internet.de/kstg_1977/__8c.html
[kstg9]: https://www.gesetze-im-internet.de/kstg_1977/__9.html
[kstg10]: https://www.gesetze-im-internet.de/kstg_1977/__10.html
[kstg23]: https://www.gesetze-im-internet.de/kstg_1977/__23.html
[kstg27]: https://www.gesetze-im-internet.de/kstg_1977/__27.html
[kstg31]: https://www.gesetze-im-internet.de/kstg_1977/__31.html
[solzg3]: https://www.gesetze-im-internet.de/solzg_1995/__3.html
[solzg4]: https://www.gesetze-im-internet.de/solzg_1995/__4.html
[ustg16]: https://www.gesetze-im-internet.de/ustg_1980/__16.html
[ustg18]: https://www.gesetze-im-internet.de/ustg_1980/__18.html
