# Abgabe über Mein ELSTER

Stand: 17. Oktober 2026. ELSTER nach der Formularliste auf elster.de (geprüft
am 17.10.2026) und dem Registrierungsleitfaden des Landesamts für Steuern
Rheinland-Pfalz (Stand 09.2025). Fristen je Erklärung nennt `fristen`; die
Regeln stehen in buchfink-grundlagen, references/pflichten.md.

## Inhalt

1. Wer welche Erklärung sendet
2. Organisationszertifikat
3. Was Buchfink liefert
4. Eingabe in Mein ELSTER
5. Nach dem Senden

## 1. Wer welche Erklärung sendet

Buchfink übermittelt nicht. Es zeigt jede Erklärung formularnah; der Mensch
überträgt die Werte in Mein ELSTER und sendet. Ist ein Steuerberater
beauftragt, sendet er (SKILL.md, Abschnitt 3).

| Erklärung | Formular in Mein ELSTER |
|---|---|
| Umsatzsteuererklärung | „Umsatzsteuererklärung“ |
| Gewerbesteuererklärung | „Gewerbesteuererklärung (GewSt 1 A)“ |
| Zerlegung bei Betriebsstätten in mehreren Gemeinden | „Erklärung für die Zerlegung des Gewerbesteuermessbetrages (GewSt 1 D)“ |
| Körperschaftsteuererklärung mit Einlagekonto | „Körperschaftsteuererklärung (KSt 1)“ mit Anlagen |
| E-Bilanz | kein Formular; die XBRL-Datei geht über einen Online-Dienst wie eBilanz+ oder eBilanz-Online (SKILL.md, Schritt 7) |

Quelle: [Formularliste][elsterformulare], geprüft am 17.10.2026. Alle Formulare
liegen unter „Formulare und Leistungen“ → „Alle Formulare“, gruppiert nach
Steuerart.

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

| Erklärung | `erklaerung` `art` | Export |
|---|---|---|
| Umsatzsteuer | `ust_jahr` | `erklaerung_exportieren` (`art: ust_jahr`, `jahr`), JSON zur Ablage |
| Gewerbesteuer | `gewst` | `erklaerung_exportieren` (`art: gewst`), JSON zur Ablage |
| Körperschaftsteuer | `kst` | `erklaerung_exportieren` (`art: kst`), JSON zur Ablage |
| E-Bilanz | – | `ebilanz_exportieren` (`jahr`), XBRL für den Online-Dienst; `format: susa_csv` als Rückfallweg |

`erklaerung` liefert je Zeile Formular, Abschnitt, Zeile (wo belegt),
Bezeichnung, Wert und Herkunft, in der Reihenfolge der Anlagen. Die App zeigt
dieselbe Ansicht als Eingabehilfe.

Der Export nennt `pfad`, `sha256`, `zahlen_hash` und `naechster_schritt`.
Exportiere erst nach der Freigabe der Zahlen (SKILL.md, Schritt 8). Ändert
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
   Weicht ein von ELSTER berechneter Wert ab, klärt ihr den Grund, bevor er
   sendet.
5. Er sendet und nennt dir Zeitpunkt und Transferticket aus dem
   Übermittlungsprotokoll. Bitte ihn, das Protokoll in den Eingang zu
   speichern; lege es dann mit `beleg_ablegen` ab.

## 5. Nach dem Senden

- Endet ein Senden unklar (Abbruch, Zeitüberschreitung, Fehler nach dem
  Senden), bitte den Menschen, in Mein ELSTER nachzusehen, ob die Erklärung
  angekommen ist, bevor er erneut sendet.
- Der Mensch erfasst das Transferticket in der App (nur App). Prüfe danach mit
  `erklaerung`, dass der Status `uebermittelt` lautet, und mit `fristen`, dass
  die Pflicht als erledigt geführt wird.
- Notiere die erwartete Abschlusszahlung oder Erstattung für buchfink-bank.

[elsterformulare]: https://www.elster.de/eportal/formulare-leistungen/alleformulare
[rlpelster]: https://lfst.rlp.de/fileadmin/lfst.rlp.de/ELSTER/Klickanleitungen/Leitfaden_Registrierung_mit_Zertifikatsdatei_Stand_09.2025.pdf
[snelster]: https://www.luft.sachsen.de/download/Schritt-fuer-Schritt-Anleitung-ELSTER-Organisationszertifikate-SN-2025.pdf
