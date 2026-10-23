# Abgabe über Mein ELSTER

Rechtsstand: 17. Oktober 2026.

Abschnitte 4 und 5 gelten für Voranmeldung, ZM und Jahreserklärungen. Fristen:
`fristen` und [grundlagen/pflichten.md](../grundlagen/pflichten.md).

## Inhalt

1. Wer welche Erklärung sendet
2. Organisationszertifikat
3. Was Buchfink liefert
4. Eingabe in Mein ELSTER
5. Nach dem Senden

## 1. Wer welche Erklärung sendet

Der Mensch überträgt und sendet, mit Steuerberater dieser
([ablauf.md](ablauf.md#3-fristen-und-steuerberater)). Formular und Weg stehen
im Pflichtenkalender unter
[ust-erklaerung](../grundlagen/pflichten.md#ust-erklaerung),
[gewst-erklaerung](../grundlagen/pflichten.md#gewst-erklaerung) (mit
Zerlegung) und [kst-erklaerung](../grundlagen/pflichten.md#kst-erklaerung)
(mit Einlagekonto); die E-Bilanz geht über einen Online-Dienst
([ebilanz.md](ebilanz.md#5-übermittlungsweg)).

## 2. Organisationszertifikat

- Die GmbH braucht ein ELSTER-Zertifikat „für eine Organisation“ auf Basis der
  Steuernummer ([Leitfaden LfSt RLP][rlpelster], Abschnitt 2.2). Der
  Aktivierungs-Code kommt per Brief an die Anschrift zur Steuernummer
  ([Anleitung Sachsen][snelster], Abschnitte 1.1, 1.2); plane Postlaufzeit ein.
- Zertifikatsdatei und Passwort bedient nur der Mensch; lass sie dir nie nennen
  oder schicken.

## 3. Was Buchfink liefert

`erklaerung` zeigt jede Erklärung formularnah; `erklaerung_exportieren` legt
Umsatz-, Körperschaft- und Gewerbesteuererklärung als JSON ab, die E-Bilanz
kommt aus `ebilanz_exportieren` ([ebilanz.md](ebilanz.md#3-was-buchfink-liefert)).
Exportiere erst nach der Freigabe ([ablauf.md](ablauf.md#2-ablauf), Schritt 8);
der Transferticket-Dialog vergleicht den `zahlen_hash` mit dem aktuellen Stand.

## 4. Eingabe in Mein ELSTER

1. Leg eine Tabelle vor: Formular, Abschnitt, Bezeichnung der Zeile,
   Zeilennummer (wo Buchfink sie nennt), Wert, Herkunft. Nenne immer die
   Bezeichnung; Zeilennummern ändern sich jährlich.
2. Der Mensch meldet sich mit dem Organisationszertifikat an, wählt das
   Formular des Erklärungsjahres und fügt die Anlagen hinzu.
3. Er trägt die Werte ein; ihr geht die Tabelle Abschnitt für Abschnitt durch.
   Hat dein Client Zugriff auf seinen Browser und bittet er darum, trägst du
   ein; Anmeldung und Senden bleiben bei ihm.
4. Er lässt Mein ELSTER prüfen und vergleicht die Vorschau mit der Tabelle.
   Jede Abweichung, auch um Cent, nennst du mit Ursache; gesendet wird erst,
   wenn die Prüfung fehlerfrei endet und jeder Wert geklärt ist.
5. Er sendet, nennt Zeitpunkt und Transferticket und speichert das
   Übermittlungsprotokoll in den Eingang; du legst es mit `beleg_ablegen` ab.

## 5. Nach dem Senden

- Jede Erklärung wird genau einmal gesendet. Endet ein Senden unklar, lass den
  Menschen in Mein ELSTER nachsehen, ob sie angekommen ist, bevor er erneut
  sendet.
- Notiere die erwartete Abschlusszahlung oder Erstattung für
  [bank/steuerzahlungen.md](../bank/steuerzahlungen.md).

[rlpelster]: https://lfst.rlp.de/fileadmin/lfst.rlp.de/ELSTER/Klickanleitungen/Leitfaden_Registrierung_mit_Zertifikatsdatei_Stand_09.2025.pdf
[snelster]: https://www.luft.sachsen.de/download/Schritt-fuer-Schritt-Anleitung-ELSTER-Organisationszertifikate-SN-2025.pdf
