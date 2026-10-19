# Erste Schritte

[Dokumentation](../README.md) · [Umfang und Grenzen](umfang.md) ·
[KI-Agenten](ki-agenten/README.md) · [Datensicherheit](datensicherheit.md)

Buchfink 0.3 ist eine Vorschau. Probieren Sie es zuerst mit einem
Testunternehmen und erfundenen Daten aus.

## Installieren

Die Installationsdateien stehen auf der
[Releases-Seite](https://github.com/mxzinke/buchfink/releases). Sie sind nicht
signiert. Das Betriebssystem warnt deshalb beim ersten Start. Laden Sie
Buchfink nur von dieser Seite.

### macOS

1. [buchfink-darwin-universal.zip][mac] herunterladen, entpacken und
   `buchfink.app` in den Ordner „Programme“ ziehen. Die App läuft auf Apple
   Silicon und Intel.
2. Buchfink starten. macOS meldet, dass es die App nicht prüfen kann.
3. Systemeinstellungen → Datenschutz & Sicherheit öffnen und bei der Meldung zu
   Buchfink „Dennoch öffnen“ wählen.

Meldet macOS, die App sei beschädigt, entfernt dieser Befehl die
Quarantäne-Markierung des Downloads:

```sh
xattr -dr com.apple.quarantine /Applications/buchfink.app
```

### Windows

1. [buchfink-windows-amd64.exe][win] herunterladen, in einen Ordner Ihrer Wahl
   legen und starten. Ein Installationsprogramm gibt es nicht.
2. Windows SmartScreen meldet „Der Computer wurde durch Windows geschützt“.
   „Weitere Informationen“ und dann „Trotzdem ausführen“ wählen.

Buchfink braucht die WebView2-Laufzeit von Microsoft. Windows 11 bringt sie
mit, unter Windows 10 kommt sie meist über Windows Update. Fehlt sie,
installieren Sie die Evergreen-Laufzeit von der
[WebView2-Seite von Microsoft](https://developer.microsoft.com/microsoft-edge/webview2/).

### Linux

[buchfink-linux-amd64.tar.gz][linux] herunterladen, entpacken und starten:

```sh
tar -xzf buchfink-linux-amd64.tar.gz
./buchfink
```

Die Datei `buchfink` können Sie in einen Ordner im `PATH` legen, etwa
`~/.local/bin`. Sie ist unter Ubuntu 24.04 für x86-64 gebaut und braucht
GTK 3, WebKitGTK 4.1 und glibc ab Version 2.34. Unter Debian und Ubuntu
installiert `sudo apt install libwebkit2gtk-4.1-0` WebKitGTK samt GTK 3, unter
Fedora `sudo dnf install webkit2gtk4.1`.

[mac]: https://github.com/mxzinke/buchfink/releases/latest/download/buchfink-darwin-universal.zip
[win]: https://github.com/mxzinke/buchfink/releases/latest/download/buchfink-windows-amd64.exe
[linux]: https://github.com/mxzinke/buchfink/releases/latest/download/buchfink-linux-amd64.tar.gz

### Aus dem Quellcode

Wer selbst baut, folgt der [Entwicklungsumgebung](../entwicklung/README.md).

## Einrichten

Beim ersten Start führt Buchfink in zwei Schritten durch die Einrichtung.

**1. Unternehmen.** Sie geben Firma und Rechtsform an (GmbH oder UG), ob Sie
gerade gründen oder mit bestehenden Büchern umsteigen, und den ersten Tag der
Bücher. Anschrift, Register, Steuernummer und Bankkonto tragen Sie ein, soweit
Sie sie schon haben. Ist die Gesellschaft gemeinnützig, in Liquidation oder
Organgesellschaft, rechnet Buchfink weder Abschluss noch Steuerrückstellung,
siehe [Umfang](umfang.md).

Darunter stehen zwei Ordner:

- **Ablage.** Ein leerer Ordner für Datenbank und Belege, vorgeschlagen nach
  der Firma. Mit „Ordner ändern“ wählen Sie einen anderen. Liegt er in iCloud
  Drive, OneDrive, Google Drive, Dropbox, Nextcloud oder Syncthing, zeigt
  Buchfink einen Hinweis: Die Datenbank verträgt keine Synchronisation,
  während Buchfink läuft. Besser ist ein lokaler Ordner mit einer Sicherung
  ([Datensicherheit](datensicherheit.md#wo-die-daten-liegen)). Weitermachen
  können Sie trotzdem.
- **Eingang.** Hier sammeln Sie Belege, E-Rechnungen und Kontoauszüge, bis
  Ihr Agent sie ablegt. Voreingestellt ist der Ordner `Eingang` in der Ablage.
  Mit „Anderen Ordner wählen“ legen Sie ihn woanders hin, auch in einen
  Cloud-Ordner, in den eine Scanner-App auf dem Telefon speichert.

Die Ablage steht nach dem Anlegen fest. Den Eingang ändern Sie später unter
Einstellungen › „Eingang & Ordner“.

**2. KI-Agent verbinden.** Sie wählen Ihren Client, richten ihn nach der
[Anleitung](ki-agenten/README.md) ein und lassen die Anmeldung im Fenster von
Buchfink zu. Was dann an den Anbieter des Sprachmodells geht, steht unter
[KI-Anbieter](datensicherheit.md#ki-anbieter). Verbinden können Sie auch
später: Der Zugang für KI-Apps läuft, solange Buchfink geöffnet ist.

Danach:

- Ein **Sicherungsziel** in den Einstellungen wählen, am besten auf einem
  anderen Datenträger. Ohne Ziel sichert Buchfink nicht. Siehe
  [Datensicherheit](datensicherheit.md#sicherung).
- Mit dem Agenten die **Eröffnung** vorbereiten: bei einer Gründung Stammkapital
  und Einzahlung, beim Umstieg die Eröffnungsbilanz aus den bisherigen Büchern.
  Nach einer Gründung speichern Sie unter Unternehmen › Stammdaten, Block
  „Gesellschafter & Kapital“, an der Pflicht „Aufstellung der Eröffnungsbilanz“
  mit „Als PDF speichern“ die Eröffnungsbilanz. Alle Geschäftsführer
  unterzeichnen sie mit Datum; danach bestätigen Sie dort die Pflicht.
- Die **Standardgruppen** Ihrer regelmäßigen Lieferanten bestätigen, sobald
  der Agent sie vorschlägt. Erst dann bucht er deren E-Rechnungen ohne
  Rückfrage ([Was der Agent darf](ki-agenten/README.md#was-der-agent-darf)).
- Fehlende **Unternehmensdaten** nachtragen, sobald eine Aufgabe danach fragt.
  Das geht nur in der App.

## Mit dem Agenten arbeiten

Sie arbeiten im Chat Ihres KI-Clients. Der Agent ruft zu Beginn das Werkzeug
`status` auf und sieht, welcher Mandant geöffnet ist. Buchfink muss dafür
laufen.

| Sie | Der Agent | Buchfink |
|---|---|---|
| legen Belege und Kontoauszüge (CAMT.053) in den Eingang und sagen „Buche die Belege im Eingang.“ | legt sie ab, liest sie und schlägt die Buchung vor | prüft Pflichtangaben, Steuerfall und Konten und verschiebt die Datei nach `Übernommen/` |
| beantworten Rückfragen im Chat | bucht Routinefälle direkt, alles andere nach Ihrer Zustimmung | speichert an jeder Buchung, über welchen Weg sie kam |
| schreiben den Monat in der App fest | bereitet Prüfbericht und Voranmeldung vor | sperrt den Monat und holt einen Zeitstempel |
| übertragen die Voranmeldung in Mein ELSTER und tragen das Transferticket in der App ein | erinnert an Fristen aus dem Pflichtenkalender | hält die Übermittlung fest |
| stellen den Jahresabschluss auf, lassen ihn feststellen und legen ihn offen | bucht die Abschlussbausteine nach Ihrer Zustimmung, bereitet Erklärungen und E-Bilanz vor | zeigt Erklärungen formularnah und exportiert sie |

Was der Agent direkt, was er mit Rückfrage und was nur Sie in der App
erledigen, steht unter [KI-Agenten](ki-agenten/README.md#was-der-agent-darf).
Bis zur Festschreibung lässt sich jede Buchung durch Storno und neue Buchung
korrigieren.

In der App haben Sie fünf Bereiche: **Übersicht** mit Aufgaben, Eingang und
Fristen, **Bücher** mit Belegen, Bank, Rechnungen, Journal, Konten und Anlagen,
**Abschlüsse** für Monat, Jahr und Erklärungen, **Unternehmen** mit
Stammdaten und Kontakten, **Einstellungen** mit Eingang und Ordnern, den
Regeln für Buchen ohne Rückfrage, verbundenen Apps und Sicherung. Was der
Agent bucht, erscheint ohne Neuladen in der offenen Ansicht.

## Belege in den Eingang legen

Belege, E-Rechnungen und Kontoauszüge kommen in den Eingang. Dafür gibt es
zwei Wege:

- Sie legen die Dateien in den Ordner, im Dateimanager oder mit einer
  Scanner-App. „Ordner öffnen“ in der Übersicht öffnet ihn.
- Sie ziehen die Dateien ins Fenster von Buchfink oder wählen in der Übersicht
  „Dateien hinzufügen“. Buchfink kopiert sie in den Eingang und meldet, was es
  übernommen hat. Angenommen werden PDF, Bilder und XML bis 50 MB; andere
  Dateien übergeht es und nennt den Grund.

Buchfink legt nichts von selbst ab und beobachtet den Ordner nicht. Sagen Sie
Ihrem Agenten, wann es so weit ist, etwa „Buche die Belege im Eingang.“ Er
legt jede Datei als Beleg oder Kontoauszug ab, bucht die Routinefälle und legt
Ihnen den Rest im Chat vor. Nach der Ablage verschiebt Buchfink die Datei im
Eingang in den Unterordner `Übernommen/<JJJJ-MM>`, nach dem Monat der Ablage.
Das Original liegt dann im Belegarchiv. Dasselbe geschieht, wenn Sie in der App
einen Beleg oder Kontoauszug aus dem Eingang ablegen.

Die Übersicht zeigt, wie viele Dateien im Eingang liegen und wie alt die
älteste ist, gemessen am Änderungsdatum der Datei. Ist sie 14 Tage oder älter,
erinnert die Übersicht daran; eine alte Rechnung aus den Downloads tut das also
sofort. Legen Sie nur echte Belege in den Eingang: eine abgelegte Datei bleibt
im Belegarchiv und in jeder Sicherung, auch wenn der Beleg später verworfen
wird. Was noch im Eingang liegt, ist in keiner Sicherung.

## E-Bilanz übermitteln

Mein ELSTER hat kein Formular für die E-Bilanz. Buchfink erzeugt die Datei,
übermitteln Sie über einen E-Bilanz-Dienst wie ebilanzplus.de oder
ebilanzonline.de.

1. Unter Abschlüsse › Erklärungen › „E-Bilanz <Jahr>“ wählen Sie „XBRL
   exportieren“. Ihr Agent kann die Datei auch in den Exportordner schreiben.
2. Laden Sie die XBRL-Datei beim Dienst hoch, ergänzen Sie dort, was er
   verlangt, und übermitteln Sie.
3. Nimmt der Dienst die XBRL-Datei nicht an, wählen Sie „Als Saldenliste
   (CSV)“ und lesen die Datei bei eBilanz+ über den Import „Summen- und
   Saldenliste“ ein, mit dem Taxonomie-Layout oder der Zuordnung nach SKR04.
   Die Saldenliste füllt nur Bilanz und Gewinn- und Verlustrechnung; die
   übrigen Angaben ergänzen Sie im Dienst. Weicht die Steuerbilanz ab, etwa
   durch eine Sonderabschreibung, enthält die Saldenliste die Handelsbilanz.
   Die Ansicht nennt dann die Abweichungen; die Überleitung zur Steuerbilanz
   tragen Sie im Dienst nach.
4. Bestätigen Sie die Übermittlung in Buchfink mit „Übermittlung bestätigen“
   und dem Transferticket aus dem Protokoll des Dienstes.

Wurde die Gesellschaft in Buchfink gegründet, geht auch die Eröffnungsbilanz
an das Finanzamt. Im Geschäftsjahr der Gründung steht sie in derselben Ansicht
über der E-Bilanz des Jahres: „Als XBRL exportieren“, beim Dienst hochladen und
übermitteln, danach „Übermittlung bestätigen“. Die Frist steht im
Pflichtenkalender.

## Rückmeldung geben

Melden Sie Fehler und unverständliche Abläufe über die
[GitHub-Issues](https://github.com/mxzinke/buchfink/issues). Nennen Sie
Programmversion, Betriebssystem, KI-Client, die Schritte und das erwartete
Ergebnis. Verwenden Sie erfundene Daten. Wie Sie sonst beitragen können, steht
unter [Mitwirken](../../CONTRIBUTING.md).
