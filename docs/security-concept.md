# Speicherung, Verschlüsselung und Integritätsprüfung

Abgleich: 11. September 2026. Dieses Dokument beschreibt die vorhandene
Implementierung. Es ist keine Sicherheitszertifizierung.

## Daten je Unternehmen

Jedes Unternehmen hat einen eigenen Datenordner mit SQLite-Datenbank,
Schlüsseldatei, Belegen und weiteren Dokumenten. Die Anwendung verwendet einen
lokalen Bearbeiter; eine Benutzer- oder Rollenverwaltung ist nicht vorhanden.
Der schreibgeschützte Prüfermodus sperrt Änderungen in der Anwendung.

## Verschlüsselte Datenbankfelder

`internal/repository/encryption.go` verschlüsselt die entsprechend markierten
Datenbankfelder mit AES-256-GCM über einen GORM-Serializer. Dazu gehören
personenbezogene und geschäftliche Texte. Es handelt sich um Feldverschlüsselung,
nicht um eine Verschlüsselung der gesamten SQLite-Datei. Weitere Felder,
Tabellenstruktur und Dateimetadaten bleiben sichtbar.

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

Die Wiederherstellungsdatei ist getrennt vom Rechner und der Datensicherung
aufzubewahren. Die Sicherung enthält die verschlüsselte Schlüsseldatei aus dem
Datenordner; sie ersetzt die externe Wiederherstellungsdatei nicht.

`internal/service/backup_service.go` erstellt ZIP-Sicherungen mit Datenbank,
Belegen, Dokumenten, Schlüsseldatei und Prüfwerten. Ein Sicherungsziel muss
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
