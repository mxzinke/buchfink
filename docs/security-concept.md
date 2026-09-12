# Speicherung, Verschlüsselung und Integritätsprüfung

Abgleich: 12. September 2026. Dieses Dokument beschreibt die vorhandene
Implementierung. Es ist keine Sicherheitszertifizierung.

## Daten je Unternehmen

Jedes Unternehmen hat einen eigenen Datenordner mit SQLite-Datenbank,
Schlüsseldatei, Belegen und weiteren Dokumenten. Die Anwendung verwendet einen
lokalen Bearbeiter; eine Benutzer- oder Rollenverwaltung ist nicht vorhanden.
Der schreibgeschützte Prüfermodus sperrt Änderungen in der Anwendung.

## Verschlüsselte Datenbankfelder

`internal/repository/encryption.go` verschlüsselt markierte Datenbankfelder
mit AES-256-GCM über GORM-Serializer. Dazu gehören Buchungstexte,
Verwendungszwecke, Kontaktadressen, E-Mail-Adressen und Vorher-/Nachher-Werte
im Änderungsprotokoll. Seit den Schemafassungen 9 und 10 kommen hinzu:

- Kontaktnamen und Namen von Bank-Zahlungspartnern;
- Beschreibungen der Rechnungspositionen;
- Kontobezeichnungen und Kontobeschreibungen, die auch Partnernamen enthalten;
- Beschreibungstexte des Änderungsprotokolls, die solche Namen wiederholen.

Neue Werte dieser Felder tragen die Kennzeichnung `buchfink:enc:2:`. Beim Öffnen
eines älteren Bestands verschlüsselt eine Migration die bisherigen Klartexte.
Sie verändert weder fachliche Werte und Zeitangaben noch die bestehenden
Hashketten. `secure_delete`, `VACUUM` und ein abschließender WAL-Checkpoint
bereinigen die alten Seiten der aktiven SQLite-Dateien. Ein Fehler bei Migration
oder Bereinigung verhindert die erfolgreiche Initialisierung.

Alte Sicherungen können mit ihren bisherigen Klartextfeldern lesend geprüft
werden. Markierte verschlüsselte Werte werden ohne passenden Schlüssel oder
bei einer beschädigten Hülle abgewiesen. Die Migration schützt keine bereits
vorher angelegten Sicherungen, Dateisystem-Snapshots oder Kopien nachträglich.

Es handelt sich um Feldverschlüsselung. Die gesamte SQLite-Datei ist nicht
verschlüsselt. Tabellenstruktur, Beträge, Datumsangaben, Kontonummern,
Ereignisarten, Bearbeiterkennungen und weitere Metadaten bleiben sichtbar.
Auch die eigenen Unternehmensstammdaten sind nicht vollständig verschlüsselt.

Ein zufälliger Datenschlüssel gehört zum Unternehmen. Die Datei
`buchfink.keyfile.json` enthält ihn in verschlüsselter Form. Ein Geheimnis zum
Entsperren liegt im Schlüsselbund des Betriebssystems. Für die normale Bedienung
ist deshalb keine Passworteingabe nötig. Die Umsetzung liegt in
`internal/security/crypto.go` und `keyring.go`.

Der Zugriff auf den entsperrten Rechner erlaubt auch den Zugriff über die
Anwendung. Die Feldverschlüsselung ersetzt keine Zugriffssicherung des Rechners.

## Belege und Dokumente

Belegdateien bleiben unverschlüsselt im Originalformat. Das ist die aktuelle
Speicherentscheidung von Buchfink. Aus der Pflicht, den Originalbeleg zu
erhalten, folgt kein allgemeines Verbot einer verschlüsselten Ablage.

Die Ablage verwendet den SHA-256-Prüfwert des Inhalts als Dateinamen.
`internal/receiptstore/store.go` bietet mit `Verify` den Vergleich einer Datei
mit ihrem hinterlegten Prüfwert. So werden fehlende oder veränderte Dateien erkennbar.
Ein Prüfwert allein verhindert weder Löschung noch eine absichtliche Neuberechnung.

## Wiederherstellung und Sicherung

`ExportTenantRecoveryFile` erstellt eine Wiederherstellungsdatei mit einem
eigenen Schlüssel. Datenordner und Wiederherstellungsdatei werden zusammen
benötigt, wenn der Eintrag im Betriebssystem-Schlüsselbund verloren geht.
`RecoverTenantFromFile` entsperrt den Datenschlüssel und richtet einen neuen
Schlüsselbund-Eintrag ein, siehe `internal/security/recovery.go`.

Der Einrichtungsassistent fordert direkt zur Schlüsselsicherung auf. Ein Aufschub
muss ausdrücklich bestätigt werden; die Aufgabenliste erinnert weiter daran.
Der Export verlangt einen Zielordner außerhalb des Unternehmensordners. Er
schreibt die Datei atomar mit Dateirechten `0600` und merkt den erfolgreichen
Export erst nach dem Schreiben und Speichern der Konfiguration vor.

Mehrere exportierte Wiederherstellungsdateien bleiben gültig: Die Schlüsseldatei
bewahrt die zugehörigen verschlüsselten Datenschlüssel auf. Ein erneuter Export
ist deshalb kein Widerruf eines früheren Schlüssels. Eine neuere
Wiederherstellungsdatei kann eine ältere Sicherung, die vor ihrer Erstellung
entstand, unter Umständen nicht öffnen. Passende Sicherung und
Wiederherstellungsdatei gemeinsam erproben und ältere Schlüssel aufbewahren.

Die Wiederherstellungsdatei ist getrennt vom Rechner und der Datensicherung
aufzubewahren. Die Sicherung enthält die verschlüsselte Schlüsseldatei aus dem
Datenordner; sie ersetzt die externe Wiederherstellungsdatei nicht.

`internal/service/backup_service.go` erstellt ZIP-Sicherungen mit Datenbank,
Belegen, Dokumenten, Schlüsseldatei und Prüfwerten. Das ZIP selbst und seine
Originaldokumente sind nicht verschlüsselt. Die Datenbank behält ihre
Feldverschlüsselung. Die Wiederherstellungsprüfung umfasst Prüfsummen,
Buchungs- und Protokollketten sowie Belege, Anlagen- und Unternehmensdokumente.
Ein Sicherungsziel muss
konfiguriert sein. Beim Start wird nach dem Abstand von 24 Stunden gesichert;
beim Beenden wird zusätzlich geprüft, ob sich seit der letzten Sicherung etwas
geändert hat. Es gibt keinen fortlaufenden täglichen Zeitgeber. Die Aufrufe
stehen in `internal/wailsbridge/backup_service.go`, `runDueBackup`.

## Journal, Protokoll und Festschreibung

Buchungen und Änderungen werden durch getrennte Hashketten verbunden.
`internal/accounting/journalhash.go` und `audithash.go` berechnen und prüfen sie.
Der Prüfwert berücksichtigt den Vorgänger und den Inhalt des jeweiligen Eintrags.
Das erkennt nachträgliche Änderungen beim Vergleich, garantiert aber allein
keinen Schutz gegen jemanden, der Daten und Kette vollständig neu berechnet.

Bei der Festschreibung wird der Kettenstand mit einem RFC-3161-Zeitstempel
verbunden. `internal/timestamp/tsa.go` sendet dafür einen Prüfwert an einen
externen Zeitstempeldienst. Ein festgeschriebener Zeitraum nimmt keine weiteren
rückdatierten Buchungen an; Korrekturen erfolgen durch Storno und Neubuchung.
Die Zeitstempelanforderung und die Schreibsperren sind getrennte Mechanismen.

Die gesetzlichen Anforderungen betreffen die gesamte Buchführung und ihre
Arbeitsabläufe, siehe [§ 146 AO](https://www.gesetze-im-internet.de/ao_1977/__146.html),
[§ 147 AO](https://www.gesetze-im-internet.de/ao_1977/__147.html) und die
[GoBD im amtlichen Handbuch](https://amtliche-handbuecher.bundesfinanzministerium.de/ao/2025/Anhaenge/BMF-Schreiben-und-gleichlautende-Laendererlasse/Anhang-33/anhang-33.html).
Die Implementierung dieser Mechanismen ist keine Zusicherung vollständiger
GoBD-Konformität. Bekannte fachliche Grenzen stehen im
[Umsetzungsstand](stand-der-umsetzung.md).
