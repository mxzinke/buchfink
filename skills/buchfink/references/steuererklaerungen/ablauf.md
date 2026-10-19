# Ablauf: Steuererklärungen

Rechtsstand: 17. Oktober 2026. Werkzeugstand: Buchfink 0.3.0.

Eine GmbH oder UG gibt für jedes Jahr die Umsatzsteuer-, die Körperschaftsteuer-
und die Gewerbesteuererklärung ab, dazu die Erklärung zum steuerlichen
Einlagekonto, und übermittelt die E-Bilanz. Buchfink rechnet die Erklärungen
aus den Büchern, zeigt sie formularnah und exportiert die E-Bilanz als
XBRL-Datei; es übermittelt nicht. Du prüfst die Rechnung, trägst nach
Rückfrage ein, was nicht in den Büchern steht, und bereitest die Abgabe vor.
Der Mensch entscheidet die Wertungsfragen, gibt die Zahlen frei, überträgt sie
in Mein ELSTER und sendet; die E-Bilanz sendet er über einen Online-Dienst mit
ELSTER-Anbindung.

Buchfink rechnet die Überleitung vom Jahresüberschuss zum zu versteuernden
Einkommen (nicht abziehbare Aufwendungen, § 8b KStG aus Konten und
Anlagenkartei, Spendenabzug, Verlustabzug mit Mindestbesteuerung), die
Hinzurechnungen und Kürzungen der Gewerbesteuer, Messbetrag, Steuer und das
steuerliche Einlagekonto. Jede Zeile hat `status` (`auto`, `vorgeschlagen`,
`eingabe`, `offen`, `entfaellt`), `zustaendig` und `herkunft`. Was nicht in
den Büchern steht, steht als `offen` oder `vorgeschlagen`: Verlustvortrag und
Einlagekonto laut Bescheid, verdeckte Gewinnausschüttung, Beteiligungsquoten,
Betriebsstätten. Die Rechtslage in den Referenzen dient dir zur Prüfung, nicht
zum Nachrechnen. Sag nie zu, dass eine Erklärung steuerlich richtig ist; bei
Zweifeln empfiehl einen Steuerberater
([grundlagen/grenzen.md](../grundlagen/grenzen.md#3-wann-du-einen-steuerberater-empfiehlst)).

| Thema | Referenz |
|---|---|
| Fristen, Weg und Nachweis jeder Pflicht | [grundlagen/pflichten.md](../grundlagen/pflichten.md) |
| Umsatzsteuer-Voranmeldung, ZM | [monatsabschluss/ablauf.md](../monatsabschluss/ablauf.md) |
| Steuerrückstellung buchen | [jahresabschluss/steuern-im-abschluss.md](../jahresabschluss/steuern-im-abschluss.md) |
| Bilanz und GuV aufstellen | [jahresabschluss/ablauf.md](../jahresabschluss/ablauf.md) |
| Zahlung, Erstattung, Säumniszuschlag, Zinsen buchen | [bank/steuerzahlungen.md](../bank/steuerzahlungen.md) |
| Dividenden, § 8b KStG, Kürzung nach § 9 Nr. 2a GewStG | [beteiligungen/steuerfreistellung.md](../beteiligungen/steuerfreistellung.md), [beteiligungen/gewst-holding.md](../beteiligungen/gewst-holding.md) |
| Verdeckte Gewinnausschüttung erkennen, Einlagen erfassen | [gesellschafter/vga-einlage.md](../gesellschafter/vga-einlage.md) |

Werkzeuge: `status`, `aufgaben`, `fristen`, `erklaerung`,
`erklaerung_eingabe` (Rückfrage), `erklaerung_exportieren`,
`ebilanz_exportieren`, `pflicht_vorbereitet`, `anlass_erfassen` (Rückfrage),
`auswertung`, `konten`, `buchungen_suchen`, `belege_suchen`, `beleg_ablegen`.

## Inhalt

1. Die Erklärungen
2. Ablauf
3. Fristen und Steuerberater
4. Entscheidungsregeln
5. Wertungsfragen
6. Nur der Mensch
7. Stolperfallen

## 1. Die Erklärungen

| Erklärung | Pflicht | `erklaerung` `art` | Lies |
|---|---|---|---|
| Umsatzsteuererklärung für das Kalenderjahr | [§ 18 Abs. 3 UStG][ustg18] | `ust_jahr` | [ust-jahr.md](ust-jahr.md) |
| Körperschaftsteuererklärung KSt 1, darin die Erklärung zum steuerlichen Einlagekonto | [§ 31 Abs. 1a KStG][kstg31], [§ 27 Abs. 2 Satz 4 KStG][kstg27] | `kst` | [kst.md](kst.md) |
| Gewerbesteuererklärung GewSt 1 A (Messbetrag) | [§ 14a GewStG][gewstg14a] | `gewst` | [gewst.md](gewst.md) |
| E-Bilanz | [§ 5b Abs. 1 EStG][estg5b] | – (`ebilanz_exportieren`) | [ebilanz.md](ebilanz.md) |

Alle Erklärungen gehen elektronisch nach amtlichem Datensatz an das Finanzamt.
Papier gibt es nur auf Antrag wegen unbilliger Härte. Formulare in Mein ELSTER
und Exporte aus Buchfink: [abgabe.md](abgabe.md); die E-Bilanz geht über einen
Online-Dienst ([ebilanz.md](ebilanz.md#5-übermittlungsweg)).

Zeitraum: Gib `zeitraum` als Kalenderjahr an (`2025`). Bei abweichendem
Wirtschaftsjahr zählt für Körperschaft- und Gewerbesteuer das Kalenderjahr, in
dem es endet ([kst.md](kst.md#1-steuerpflicht-und-zeitraum),
[gewst.md](gewst.md#1-steuerpflicht-erklärung-bescheide)); die Umsatzsteuer
folgt immer dem Kalenderjahr
([ust-jahr.md](ust-jahr.md#1-pflicht-zeitraum-frist)). Prüfe in der Antwort
von `erklaerung`, welches Wirtschaftsjahr Buchfink zugrunde legt.

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
   Abweichung erklärt ist. [ust-jahr.md](ust-jahr.md#2-abgleich-mit-den-voranmeldungen)
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
   [gewst.md](gewst.md#3-hinzurechnungen). Fertig wie Schritt 4 und wenn der
   Hebesatz zum Bescheid der Gemeinde passt.
6. **Steuerrückstellung.** Vergleiche den Vorschlag von `abschlussbuchung`
   (`art: steuerrueckstellung`, `nur_vorschau: true`) mit Schritt 4 und 5.
   Verlustvorträge, Hinzurechnungen, Kürzungen und eine vGA nimmt der
   Vorschlag nur als Eingabe; nimm die Werte aus den Erklärungen
   ([jahresabschluss/steuern-im-abschluss.md](../jahresabschluss/steuern-im-abschluss.md)).
   Nenne je Steuerart Steuer, Vorauszahlungen und Rückstellung und übergib an
   den Jahresabschluss. Fertig, wenn der Mensch die Beträge bestätigt hat.
7. **E-Bilanz.** `ebilanz_exportieren` (`jahr`; `format: xbrl` ist Vorgabe)
   prüft und schreibt die XBRL-Datei in den Exportordner. Fertig, wenn kein
   blockierender Befund bleibt und der Mensch den Online-Dienst gewählt hat.
   [ebilanz.md](ebilanz.md#4-prüfpunkte)
8. **Freigabe.** Lege je Erklärung vor: Kennzahlen, Überleitung, Abweichungen
   zum Vorschlag aus Buchfink, Wertungsfragen mit Grundlage und Optionen.
   Fertig, wenn der Mensch die Zahlen ausdrücklich freigibt.
9. **Abgeben.** Melde jede Erklärung mit `pflicht_vorbereitet` als
   vorbereitet. Dann nach [abgabe.md](abgabe.md#4-eingabe-in-mein-elster): Der
   Mensch überträgt Umsatzsteuer, Körperschaftsteuer und Gewerbesteuer mit
   deiner Tabelle in Mein ELSTER und sendet; die E-Bilanz importiert, prüft und
   sendet er im Online-Dienst ([ebilanz.md](ebilanz.md#5-übermittlungsweg)).
   Mit Steuerberater übergibst du ihm die Unterlagen (Abschnitt 3). Fertig,
   wenn jede Erklärung ein Transferticket hat.
10. **Transferticket erfassen lassen (nur App).** Nenne dem Menschen Erklärung,
    Jahr, Zeitpunkt und Transferticket. Fertig, wenn `erklaerung` den Status
    `uebermittelt` zeigt und `fristen` die Pflicht als erledigt führt.
11. **Bescheid prüfen.** Bescheid mit `beleg_ablegen` ablegen, mit der
    Erklärung abgleichen, Einspruchsfrist und Zahltermin nennen. Die Zahlung
    buchst du nach [bank/steuerzahlungen.md](../bank/steuerzahlungen.md).
    Fertig, wenn jede Abweichung erklärt ist. [bescheide.md](bescheide.md)

## 3. Fristen und Steuerberater

`fristen` nennt Fälligkeit, Stand und Zuständigkeit jeder Erklärung. Frist,
Weg, Nachweis und Folgen stehen je Pflicht im Pflichtenkalender:
[kst-erklaerung](../grundlagen/pflichten.md#kst-erklaerung),
[gewst-erklaerung](../grundlagen/pflichten.md#gewst-erklaerung),
[ust-erklaerung](../grundlagen/pflichten.md#ust-erklaerung),
[ebilanz](../grundlagen/pflichten.md#ebilanz),
[kst-vorauszahlung](../grundlagen/pflichten.md#kst-vorauszahlung),
[gewst-vorauszahlung](../grundlagen/pflichten.md#gewst-vorauszahlung).
Übergangsjahre, Fristende, Zuschläge und Zinsen:
[fristen-zuschlaege.md](fristen-zuschlaege.md).

Ob ein Steuerberater erstellt und damit die längere Frist gilt, sagt das
Stammdatum „steuerliche Beratung“; was du dann tust, steht in
[grundlagen/pflichten.md](../grundlagen/pflichten.md#arbeiten-mit-fristen)
unter „Mit und ohne Steuerberater“. Zu den Unterlagen für den Berater gehören
auch die Überleitung, die
[Liste Abschlussposten](../jahresabschluss/abschlussbuchungen.md#4-liste-abschlussposten)
und die offenen Wertungsfragen (Schritte 2 bis 7).

Ist eine Frist verstrichen, sag es sofort, bereite die Erklärung vorrangig vor
und nenne dem Menschen die rückwirkende Fristverlängerung als Möglichkeit
([fristen-zuschlaege.md](fristen-zuschlaege.md#3-fristverlängerung)).

## 4. Entscheidungsregeln

- Rechne auf dem Stand des aufgestellten Abschlusses. Ändert sich danach eine
  Buchung, rechne neu und exportiere neu (`zahlen_hash`).
- Verlustvortrag, Anfangsbestand des Einlagekontos, Hebesatz und
  Vorauszahlungen übernimmst du nur aus einem Bescheid oder einer Angabe des
  Menschen.
- Die Sätze wählt Buchfink nach dem Veranlagungszeitraum. Zur Prüfung:
  [kst.md](kst.md#9-steuer-rundung-anrechnung-vorauszahlungen) und
  [gewst.md](gewst.md#2-rechenweg).
- Erkennst du eine Wertungsfrage, leg sie vor, bevor du die Zahl einsetzt.

## 5. Wertungsfragen

Leg jede nach [SKILL.md](../../SKILL.md#3-den-menschen-fragen) vor, mit
Grundlage, Optionen und Folgen. Typisch in den Erklärungen:

- verdeckte Gewinnausschüttung oder verdeckte Einlage
  ([kst.md](kst.md#4-verdeckte-gewinnausschüttung-und-verdeckte-einlage),
  [gesellschafter/vga-einlage.md](../gesellschafter/vga-einlage.md));
- nicht abziehbare Ausgaben und ihre Höhe
  ([kst.md](kst.md#3-nicht-abziehbare-aufwendungen));
- Spende an einen begünstigten Empfänger oder durch einen Gesellschafter
  veranlasst ([kst.md](kst.md#6-spenden));
- Miete, Pacht oder Lizenz unter § 8 Nr. 1 GewStG
  ([gewst.md](gewst.md#3-hinzurechnungen));
- Betriebsstätten in mehreren Gemeinden ([gewst.md](gewst.md#7-zerlegung));
- schädlicher Beteiligungserwerb ([kst.md](kst.md#8-verlustabzug));
- Abweichungen der Steuerbilanz
  ([kst.md](kst.md#7-abweichungen-der-steuerbilanz));
- Einspruch ([bescheide.md](bescheide.md#4-einspruch)).

## 6. Nur der Mensch

Zu den Vorbehalten in
[SKILL.md](../../SKILL.md#4-nur-der-mensch-in-der-app) gehören hier:

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

- Steuern vom Einkommen, Gewerbesteuer und ihre Nebenleistungen stehen in der
  GuV als Aufwand; die Überleitung rechnet sie hinzu
  ([kst.md](kst.md#2-überleitung-zum-zu-versteuernden-einkommen)).
- Die GmbH hat bei der Gewerbesteuer keinen Freibetrag; der Gewerbeertrag wird
  auf volle 100 € abgerundet ([gewst.md](gewst.md#2-rechenweg)).
- Kapitalertragsteuer wird nur mit Steuerbescheinigung angerechnet
  ([kst.md](kst.md#9-steuer-rundung-anrechnung-vorauszahlungen)).
- Im Gründungsjahr gilt je Steuer der kürzere Zeitraum
  ([kst.md](kst.md#1-steuerpflicht-und-zeitraum),
  [gewst.md](gewst.md#1-steuerpflicht-erklärung-bescheide),
  [ust-jahr.md](ust-jahr.md#1-pflicht-zeitraum-frist)); auch die
  Eröffnungsbilanz geht als E-Bilanz ([ebilanz.md](ebilanz.md#6-eröffnungsbilanz)).
- Mein ELSTER führt kein Formular für die E-Bilanz; gleiche nach jedem Import
  im Online-Dienst ab ([ebilanz.md](ebilanz.md#5-übermittlungsweg)).
- Ab der Bekanntgabe eines Bescheids läuft die Einspruchsfrist; ein Einspruch
  hält die Zahlung nicht auf ([bescheide.md](bescheide.md)).

[estg5b]: https://www.gesetze-im-internet.de/estg/__5b.html
[gewstg14a]: https://www.gesetze-im-internet.de/gewstg/__14a.html
[kstg27]: https://www.gesetze-im-internet.de/kstg_1977/__27.html
[kstg31]: https://www.gesetze-im-internet.de/kstg_1977/__31.html
[ustg18]: https://www.gesetze-im-internet.de/ustg_1980/__18.html
