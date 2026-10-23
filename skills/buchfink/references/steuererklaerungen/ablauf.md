# Ablauf: Steuererklärungen

Rechtsstand: 17. Oktober 2026. Werkzeugstand: Buchfink 0.3.0.

Buchfink rechnet die Jahreserklärungen aus den Büchern. Du prüfst, trägst nach
Rückfrage ein, was nicht in den Büchern steht, und bereitest die Abgabe vor.
Bei Zweifeln empfiehl einen Steuerberater
([grundlagen/grenzen.md](../grundlagen/grenzen.md#3-wann-du-einen-steuerberater-empfiehlst)).

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

`zeitraum` ist das Kalenderjahr (`2025`). Bei abweichendem Wirtschaftsjahr
prüfe in der Antwort, welches Buchfink zugrunde legt (Regeln je Steuer in
Abschnitt 1 von [kst.md](kst.md#1-steuerpflicht-und-zeitraum),
[gewst.md](gewst.md#1-steuerpflicht-erklärung-bescheide) und
[ust-jahr.md](ust-jahr.md#1-pflicht-zeitraum-frist)).

## 2. Ablauf

1. **Stand klären.** `status` und `aufgaben`. Die Erklärungen bauen auf dem
   Jahresabschluss auf ([jahresabschluss/ablauf.md](../jahresabschluss/ablauf.md));
   ist er offen, frag, ob du zuerst die Steuerrückstellung liefern sollst
   (Schritt 6). Fertig, wenn Jahr, Abschlussstand, Fälligkeit und
   Zuständigkeit jeder Erklärung aus `fristen` feststehen.
2. **Unterlagen sammeln.** Vorjahresbescheide (Körperschaftsteuer,
   Messbetrag, Verlustfeststellungen, Einlagekonto), Vorauszahlungsbescheide,
   Steuerbescheinigungen, Zuwendungsbestätigungen, Hebesatz,
   Gesellschafterbeschlüsse, Liste der Einlagen. `belege_suchen`,
   `beleg_ablegen`, nach Fehlendem fragen. Fertig, wenn jede Unterlage vorliegt
   oder als fehlend benannt ist.
3. **Umsatzsteuer** (`art: ust_jahr`) nach
   [ust-jahr.md](ust-jahr.md#2-abgleich-mit-den-voranmeldungen). Fertig, wenn
   jede Abweichung zu Voranmeldungen und Konten erklärt ist.
4. **Körperschaftsteuer** (`art: kst`, zuerst `nur_offen: true`). Prüfe jede
   Zeile `offen` oder `vorgeschlagen` und jeden Befund nach [kst.md](kst.md),
   die Zeilen `auto` über `herkunft` gegen `auswertung`. Angaben aus Bescheid,
   Beschluss oder Beleg trägst du nach Rückfrage mit `erklaerung_eingabe` ein.
   Fertig, wenn keine Pflichtzeile `offen` ist, `abgabereif: true` und die
   Wertungsfragen gesammelt sind.
5. **Gewerbesteuer** (`art: gewst`) ebenso; Miete, Leasing und Lizenzen nach
   [gewst.md](gewst.md#3-hinzurechnungen). Fertig wie Schritt 4 und wenn der
   Hebesatz zum Bescheid der Gemeinde passt.
6. **Steuerrückstellung.** Vergleiche `abschlussbuchung`
   (`art: steuerrueckstellung`, `nur_vorschau: true`) mit Schritt 4 und 5;
   Verlustvorträge, Hinzurechnungen, Kürzungen und vGA nimmt der Vorschlag nur
   als Eingabe, nimm sie aus den Erklärungen
   ([jahresabschluss/steuern-im-abschluss.md](../jahresabschluss/steuern-im-abschluss.md)).
   Fertig, wenn der Mensch je Steuerart Steuer, Vorauszahlungen und
   Rückstellung bestätigt hat.
7. **E-Bilanz.** `ebilanz_exportieren` (`jahr`), Prüfpunkte nach
   [ebilanz.md](ebilanz.md#4-prüfpunkte). Fertig, wenn kein blockierender
   Befund bleibt und der Mensch den Online-Dienst gewählt hat.
8. **Freigabe.** Je Erklärung: Kennzahlen, Überleitung, Abweichungen zum
   Vorschlag aus Buchfink, Wertungsfragen mit Grundlage und Optionen. Fertig,
   wenn der Mensch die Zahlen ausdrücklich freigibt.
9. **Abgeben.** Jede Erklärung mit `pflicht_vorbereitet` melden, dann nach
   [abgabe.md](abgabe.md#4-eingabe-in-mein-elster); E-Bilanz im Online-Dienst
   ([ebilanz.md](ebilanz.md#5-übermittlungsweg)). Mit Steuerberater: Abschnitt 3.
   Fertig, wenn jede Erklärung ein Transferticket hat.
10. **Transferticket erfassen lassen (nur App).** Fertig, wenn `erklaerung`
    `uebermittelt` und `fristen` die Pflicht als erledigt zeigt.
11. **Bescheid prüfen** nach [bescheide.md](bescheide.md); Zahlung nach
    [bank/steuerzahlungen.md](../bank/steuerzahlungen.md). Fertig, wenn jede
    Abweichung erklärt ist.

## 3. Fristen und Steuerberater

`fristen` nennt Fälligkeit, Stand, Zuständigkeit und `anker`; Frist, Weg,
Nachweis und Folgen stehen unter diesem Anker in
[grundlagen/pflichten.md](../grundlagen/pflichten.md). Zuschläge und Zinsen:
[fristen-zuschlaege.md](fristen-zuschlaege.md).

Das Stammdatum „steuerliche Beratung“ bestimmt, ob die Beraterfrist gilt; dein
Vorgehen steht in
[grundlagen/pflichten.md](../grundlagen/pflichten.md#arbeiten-mit-fristen)
unter „Mit und ohne Steuerberater“. Der Berater bekommt auch Überleitung,
[Liste Abschlussposten](../jahresabschluss/abschlussbuchungen.md#4-liste-abschlussposten)
und offene Wertungsfragen.

Frist verstrichen: sofort sagen, vorrangig vorbereiten, rückwirkende
Verlängerung nennen ([fristen-zuschlaege.md](fristen-zuschlaege.md#3-fristverlängerung)).

## 4. Entscheidungsregeln

- Rechne auf dem Stand des aufgestellten Abschlusses; ändert sich eine
  Buchung, rechne und exportiere neu (`zahlen_hash`).
- Verlustvortrag, Anfangsbestand des Einlagekontos, Hebesatz und
  Vorauszahlungen nur aus Bescheid oder Angabe des Menschen.

## 5. Wertungsfragen

- vGA oder verdeckte Einlage
  ([kst.md](kst.md#4-verdeckte-gewinnausschüttung-und-verdeckte-einlage),
  [gesellschafter/vga-einlage.md](../gesellschafter/vga-einlage.md));
- nicht abziehbare Ausgaben ([kst.md](kst.md#3-nicht-abziehbare-aufwendungen));
- Spende begünstigt oder vom Gesellschafter veranlasst ([kst.md](kst.md#6-spenden));
- Miete, Pacht, Lizenz nach § 8 Nr. 1 GewStG ([gewst.md](gewst.md#3-hinzurechnungen));
- Betriebsstätten in mehreren Gemeinden ([gewst.md](gewst.md#7-zerlegung));
- schädlicher Beteiligungserwerb ([kst.md](kst.md#8-verlustabzug));
- Abweichungen der Steuerbilanz ([kst.md](kst.md#7-abweichungen-der-steuerbilanz));
- Einspruch ([bescheide.md](bescheide.md#4-einspruch)).

## 6. Nur der Mensch

Zusätzlich zu [SKILL.md](../../SKILL.md#4-nur-der-mensch-in-der-app): Zahlen
freigeben; Online-Dienst der E-Bilanz wählen oder Härtefallantrag stellen;
Fristverlängerung, Einspruch, Aussetzung der Vollziehung beantragen;
Steuerberater beauftragen und „steuerliche Beratung“ pflegen.

## 7. Stolperfallen

- Kapitalertragsteuer nur mit Steuerbescheinigung anrechnen
  ([kst.md](kst.md#9-steuer-rundung-anrechnung-vorauszahlungen)).
- Gründungsjahr: je Steuer der kürzere Zeitraum
  ([kst.md](kst.md#1-steuerpflicht-und-zeitraum),
  [gewst.md](gewst.md#1-steuerpflicht-erklärung-bescheide),
  [ust-jahr.md](ust-jahr.md#1-pflicht-zeitraum-frist)); die Eröffnungsbilanz
  geht als E-Bilanz ([ebilanz.md](ebilanz.md#6-eröffnungsbilanz)).
- Ein Einspruch hält die Zahlung nicht auf ([bescheide.md](bescheide.md)).

[estg5b]: https://www.gesetze-im-internet.de/estg/__5b.html
[gewstg14a]: https://www.gesetze-im-internet.de/gewstg/__14a.html
[kstg27]: https://www.gesetze-im-internet.de/kstg_1977/__27.html
[kstg31]: https://www.gesetze-im-internet.de/kstg_1977/__31.html
[ustg18]: https://www.gesetze-im-internet.de/ustg_1980/__18.html
