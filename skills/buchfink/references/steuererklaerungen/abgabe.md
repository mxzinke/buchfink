# Abgabe über Mein ELSTER

Rechtsstand: 17. Oktober 2026. ELSTER nach der Formularliste auf elster.de
(geprüft am 17.10.2026) und dem Registrierungsleitfaden des Landesamts für
Steuern Rheinland-Pfalz (Stand 09.2025).

Die Eingabe in Mein ELSTER (Abschnitt 4 und 5) gilt für Voranmeldung, ZM und
Jahreserklärungen. Fristen je Erklärung nennt `fristen`; die Regeln stehen in
[grundlagen/pflichten.md](../grundlagen/pflichten.md).

## Inhalt

1. Wer welche Erklärung sendet
2. Organisationszertifikat
3. Was Buchfink liefert
4. Eingabe in Mein ELSTER
5. Nach dem Senden

## 1. Wer welche Erklärung sendet

Buchfink übermittelt nicht. Es zeigt jede Erklärung formularnah; der Mensch
überträgt die Werte in Mein ELSTER und sendet. Ist ein Steuerberater
beauftragt, sendet er ([ablauf.md](ablauf.md#3-fristen-und-steuerberater)).

| Erklärung | Formular und Weg |
|---|---|
| Umsatzsteuererklärung | [grundlagen/pflichten.md#ust-erklaerung](../grundlagen/pflichten.md#ust-erklaerung) |
| Gewerbesteuererklärung, Zerlegung bei Betriebsstätten in mehreren Gemeinden | [grundlagen/pflichten.md#gewst-erklaerung](../grundlagen/pflichten.md#gewst-erklaerung) |
| Körperschaftsteuererklärung mit Einlagekonto | [grundlagen/pflichten.md#kst-erklaerung](../grundlagen/pflichten.md#kst-erklaerung) |
| E-Bilanz | kein Formular; die XBRL-Datei geht über einen Online-Dienst ([ebilanz.md](ebilanz.md#5-übermittlungsweg)) |

## 2. Organisationszertifikat

- Die GmbH braucht ein ELSTER-Zertifikat „für eine Organisation“. Es wird auf
  Basis der Steuernummer erstellt ([Leitfaden LfSt RLP][rlpelster],
  Abschnitt 2.2). Aktiviert wird es mit einer Aktivierungs-ID per E-Mail und
  einem Aktivierungs-Code per Brief an die Anschrift, die zur Steuernummer
  gespeichert ist ([Anleitung Organisationszertifikate, Sachsen][snelster],
  Abschnitte 1.1 und 1.2; Anleitung einer Landesbehörde für ein anderes
  Fachverfahren, keine der Steuerverwaltung). Plane dafür Postlaufzeit ein.
- Zertifikatsdatei und Passwort bedient nur der Mensch. Lass sie dir nie
  nennen oder schicken.

## 3. Was Buchfink liefert

| Erklärung | Export |
|---|---|
| Umsatzsteuer | `erklaerung_exportieren` (`art: ust_jahr`, `jahr`), JSON zur Ablage |
| Gewerbesteuer | `erklaerung_exportieren` (`art: gewst`), JSON zur Ablage |
| Körperschaftsteuer | `erklaerung_exportieren` (`art: kst`), JSON zur Ablage |
| E-Bilanz | `ebilanz_exportieren` (`jahr`), XBRL für den Online-Dienst; `format: susa_csv` als Rückfallweg ([ebilanz.md](ebilanz.md#3-was-buchfink-liefert)) |

`erklaerung` liefert je Zeile Formular, Abschnitt, Zeile (wo belegt),
Bezeichnung, Wert und Herkunft, in der Reihenfolge der Anlagen. Die App zeigt
dieselbe Ansicht als Eingabehilfe.

Der Export nennt `pfad`, `sha256`, `zahlen_hash` und `naechster_schritt`.
Exportiere erst nach der Freigabe der Zahlen ([ablauf.md](ablauf.md#2-ablauf),
Schritt 8). Ändert
sich danach eine Buchung, rechne neu und exportiere neu; der Dialog für das
Transferticket vergleicht den `zahlen_hash` mit dem aktuellen Stand.

## 4. Eingabe in Mein ELSTER

1. Lege eine Tabelle vor: Formular, Abschnitt, Bezeichnung der Zeile,
   Zeilennummer (wo Buchfink sie nennt), Wert, Herkunft (Konto, Beleg,
   Bescheid). Zeilennummern ändern sich jährlich; nenne deshalb immer die
   Bezeichnung.
2. Der Mensch meldet sich mit dem Organisationszertifikat an, wählt das
   Formular des Erklärungsjahres und fügt die Anlagen hinzu.
3. Er trägt die Werte ein; du gehst die Tabelle Abschnitt für Abschnitt mit
   ihm durch. Hat dein Client Zugriff auf seinen Browser und bittet er darum,
   trägst du die Werte ins geöffnete Formular ein; Anmeldung und Senden bleiben
   bei ihm.
4. Er lässt Mein ELSTER prüfen und vergleicht die Vorschau mit der Tabelle.
   Weicht ein von ELSTER berechneter Wert ab, auch nur um Cent, nennst du
   Abweichung und Ursache (etwa Rundung); er sendet erst, wenn die Prüfung
   ohne Fehler endet und jeder Wert geklärt ist.
5. Er sendet und nennt dir Zeitpunkt und Transferticket aus dem
   Übermittlungsprotokoll. Bitte ihn, das Protokoll in den Eingang zu
   speichern; lege es dann mit `beleg_ablegen` ab.

## 5. Nach dem Senden

- Jede Erklärung wird genau einmal gesendet. Endet ein Senden unklar
  (Abbruch, Zeitüberschreitung, Fehler nach dem Senden), bitte den Menschen,
  in Mein ELSTER nachzusehen, ob die Erklärung angekommen ist, bevor er erneut
  sendet.
- Das Transferticket erfasst der Mensch in der App
  ([ablauf.md](ablauf.md#2-ablauf), Schritt 10).
- Notiere die erwartete Abschlusszahlung oder Erstattung für die Buchung nach
  [bank/steuerzahlungen.md](../bank/steuerzahlungen.md).

[rlpelster]: https://lfst.rlp.de/fileadmin/lfst.rlp.de/ELSTER/Klickanleitungen/Leitfaden_Registrierung_mit_Zertifikatsdatei_Stand_09.2025.pdf
[snelster]: https://www.luft.sachsen.de/download/Schritt-fuer-Schritt-Anleitung-ELSTER-Organisationszertifikate-SN-2025.pdf
