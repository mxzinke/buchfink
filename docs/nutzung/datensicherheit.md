# Datensicherheit

[Dokumentation](../README.md) · [Erste Schritte](README.md)

Diese Seite beschreibt, wo Buchfink Daten ablegt, was es schützt und was nicht.
Sie ist keine Sicherheitszertifizierung.

## Wo die Daten liegen

Jeder Mandant hat einen eigenen Datenordner, den Sie beim Anlegen wählen:

| Inhalt | Ort im Datenordner |
|---|---|
| Buchungen, Stammdaten, Änderungsprotokoll | `buchfink.sqlite` |
| Belegdateien, benannt nach ihrer SHA-256-Prüfsumme | `belege/` |
| Fassungen der Verfahrensdokumentation | `dokumente/` |
| Belege, die eine verbundene KI-App ablegen darf | `Belegeingang/` |
| Dateien, die eine verbundene KI-App erzeugt | `Export/` |
| Sperrdatei, gesperrt, solange Buchfink den Mandanten geöffnet hat | `buchfink.lock` |

Die App selbst legt ihre Einstellungen unter `~/.buchfink/` ab: die Liste der
Mandanten in `config.json`, die verbundenen KI-Apps in `verbindungen.sqlite`.
Die Brücke für Claude Desktop speichert ihre Anmeldung im
Konfigurationsordner des Benutzers unter `Buchfink/mcp-bruecke/`.

Ein Datenordner hat genau einen Schreiber. Öffnet Buchfink einen Mandanten,
sperrt es den Ordner; ein zweites Buchfink kann ihn dann nicht öffnen.

## Keine Verschlüsselung

Buchfink verschlüsselt weder die Datenbank noch die Belege noch die
Sicherungen. Wer Zugriff auf Ihr Benutzerkonto oder den Datenträger hat, kann
alles lesen. Den Schutz am Rechner übernehmen Sie:

- Festplattenverschlüsselung einschalten: FileVault unter macOS, BitLocker oder
  die Geräteverschlüsselung unter Windows, LUKS unter Linux.
- Das Benutzerkonto mit Kennwort schützen und die Bildschirmsperre nutzen.
- Sicherungen auf einem verschlüsselten Datenträger oder in einem
  verschlüsselten Ordner ablegen.

Liegt der Datenordner in einem synchronisierten Cloud-Ordner, liegen die Daten
auch beim Anbieter dieses Dienstes. Wo Bücher außerhalb Deutschlands geführt
werden dürfen, regelt [§ 146 Abs. 2a und 2b AO][ao146].

## Integrität

Buchfink macht Änderungen erkennbar. Verhindern kann es sie nicht, wenn jemand
die Dateien direkt bearbeitet.

- **Hash-Ketten.** Jede Buchung enthält den SHA-256-Hash ihrer Vorgängerin und
  einen eigenen Hash über ihren Inhalt. Das Änderungsprotokoll ist auf dieselbe
  Weise verkettet. Eine geänderte oder entfernte Zeile bricht die Kette.
- **Beleg-Hash.** Über die Dateien eines Belegs und seine Kopfdaten bildet
  Buchfink einen Hash, der in die Buchung eingeht. Eine ausgetauschte Datei
  ändert ihn.
- **Zeitstempel.** Bei jeder Festschreibung schickt Buchfink den Hash des
  Kettenendes an den Zeitstempeldienst von DigiCert und speichert die Antwort
  nach [RFC 3161][rfc3161]. Andere Daten verlassen dabei den Rechner nicht. Ohne
  Netz steht die Festschreibung trotzdem; der Zeitstempel wird nachgeholt.
- **Festschreibung.** Ein festgeschriebener Zeitraum nimmt keine Buchung mehr
  an. Korrigiert wird durch Storno und neue Buchung.
- **Prüfermodus.** Er sperrt jeden schreibenden Vorgang, für die App und für
  verbundene Agenten. Sein Zustand liegt in der Mandantendatenbank.

Grenzen: Wer Datenbank und Ketten vollständig neu berechnet, fällt erst am
Zeitstempel auf, und nur für festgeschriebene Zeiträume. Buchfink prüft beim
Zeitstempel Signatur und Hash; die Zertifikatskette gleicht es nicht mit einer
Liste vertrauenswürdiger Aussteller ab. Eine Benutzerverwaltung gibt es nicht.
Jede Buchung speichert die Bearbeiterkennung aus Benutzername und Rechnername
des Betriebssystems und den Erfassungsweg.

## Sicherung

Eine Sicherung ist eine ZIP-Datei mit der Datenbank, den Belegen, den
Fassungen der Verfahrensdokumentation und einer Datei `backup.json` mit den
Prüfsummen aller Dateien. Sie ist nicht verschlüsselt.

- Automatisch sichert Buchfink erst, wenn ein Sicherungsziel eingerichtet ist.
  Dann sichert es, sobald die letzte Sicherung 24 Stunden alt ist, und beim
  Beenden, wenn sich seit der letzten Sicherung etwas geändert hat.
- Der Wiederherstellungstest entpackt eine Sicherung in einen leeren
  Arbeitsordner und prüft Prüfsummen, Hash-Ketten und Belegdateien. Ihre Daten
  bleiben dabei unverändert.
- Eine Wiederherstellung entpackt die Sicherung in einen leeren Ordner, meldet
  sie als Mandanten an und prüft sie. Das Ergebnis steht im
  Änderungsprotokoll.

Legen Sie das Sicherungsziel auf einen anderen Datenträger als den Datenordner
und führen Sie den Wiederherstellungstest gelegentlich aus.

## Verbundene Apps

KI-Apps erreichen Buchfink über einen Server, der nur auf `127.0.0.1` lauscht.
Buchfink ist dafür selbst Autorisierungsserver nach OAuth 2.1 mit PKCE.

- Zugestimmt wird nur im Fenster von Buchfink. Die Browserseite zeigt
  lediglich einen Abgleichcode. Eine Anfrage verfällt nach fünf Minuten.
- Apps dürfen sich nur mit Rücksprungadressen auf diesem Rechner registrieren.
- Die Zulassung gilt je Mandant und mit den Rechten Lesen und Buchen.
- Ein Zugriffstoken gilt zehn Minuten, ein Erneuerungstoken 30 Tage; nach 180
  Tagen ist eine neue Anmeldung nötig. Buchfink speichert Codes und Tokens nur
  als SHA-256-Hash.
- In den Einstellungen unter „Apps verbinden“ sehen Sie jede zugelassene App
  und entziehen ihr den Zugriff. Ihr nächster Aufruf scheitert.
- Zulassung und Entzug protokolliert Buchfink im Änderungsprotokoll des
  geöffneten Mandanten. Einzelne Lesezugriffe einer App protokolliert es nicht.

Grenzen: Jedes Programm, das unter Ihrem Benutzerkonto läuft, kann eine
Anmeldung anstoßen. Deshalb gilt: nur zulassen, wenn der Abgleichcode
übereinstimmt und Sie die Anmeldung selbst begonnen haben. Den Namen, unter
dem sich eine App meldet, prüft Buchfink nicht. Was eine zugelassene App mit
den gelesenen Daten tut, liegt außerhalb von Buchfink.

## KI-Anbieter

Was ein Werkzeug zurückgibt, schickt die KI-App an den Anbieter ihres
Sprachmodells: Buchungen, Beträge, Namen von Kunden, Lieferanten und
Beschäftigten, Texte und Dateien aus Belegen. Diese Übertragung verantworten
Sie. Klären Sie vor dem Einsatz mit echten Daten, ob Ihr Vertrag mit dem
Anbieter eine Auftragsverarbeitung nach [Art. 28 DSGVO][dsgvo] abdeckt und ob
Ihre Eingaben zum Training genutzt werden. Buchfink selbst überträgt keine
Buchführungsdaten an den Hersteller.

## Weitere Dienste im Netz

| Dienst | Wozu | Was übertragen wird |
|---|---|---|
| Zeitstempeldienst DigiCert | Beglaubigung der Festschreibung | ein SHA-256-Hash |
| Frankfurter-API (Referenzkurse der EZB) | Kurse für Fremdwährungsbeträge | Währung und Datum |
| Bundeszentralamt für Steuern | Bestätigung einer USt-IdNr. | eigene und fremde USt-IdNr., bei der qualifizierten Abfrage Name und Anschrift des Partners |

Im Prüfermodus holt Buchfink keine Kurse, weil ein geholter Kurs den
Datenbestand ändern würde.

[ao146]: https://www.gesetze-im-internet.de/ao_1977/__146.html
[rfc3161]: https://www.rfc-editor.org/rfc/rfc3161
[dsgvo]: https://eur-lex.europa.eu/legal-content/DE/TXT/?uri=CELEX:32016R0679
