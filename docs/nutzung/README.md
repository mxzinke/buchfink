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
gerade gründen oder mit bestehenden Büchern umsteigen, den ersten Tag der
Bücher und einen leeren Ordner für die Daten. Anschrift, Register,
Steuernummer und Bankkonto tragen Sie ein, soweit Sie sie schon haben. Drei
Fragen klären, ob die Gesellschaft gemeinnützig ist, sich in Liquidation
befindet oder Organgesellschaft ist. Für solche Gesellschaften rechnet Buchfink
weder Abschluss noch Steuerrückstellung, siehe [Umfang](umfang.md).

**2. KI-Agent verbinden.** Sie wählen Ihren Client, richten ihn nach der
[Anleitung](ki-agenten/README.md) ein und lassen die Anmeldung im Fenster von
Buchfink zu. Dabei sehen Sie, welche Daten an den Anbieter des Sprachmodells
gehen. Verbinden können Sie auch später: Der Zugang für KI-Apps läuft, solange
Buchfink geöffnet ist.

Danach:

- Ein **Sicherungsziel** in den Einstellungen wählen, am besten auf einem
  anderen Datenträger. Ohne Ziel sichert Buchfink nicht. Siehe
  [Datensicherheit](datensicherheit.md#sicherung).
- Mit dem Agenten die **Eröffnung** vorbereiten: bei einer Gründung Stammkapital
  und Einzahlung, beim Umstieg die Eröffnungsbilanz aus den bisherigen Büchern.
- Fehlende **Unternehmensdaten** nachtragen, sobald eine Aufgabe danach fragt.
  Das geht nur in der App.

## Mit dem Agenten arbeiten

Sie arbeiten im Chat Ihres KI-Clients. Der Agent ruft zu Beginn das Werkzeug
`status` auf und sieht, welcher Mandant geöffnet ist. Buchfink muss dafür
laufen.

| Sie | Der Agent | Buchfink |
|---|---|---|
| geben Belege und Kontoauszüge (CAMT.053) als Datei oder Pfad | legt sie ab, liest sie und schlägt die Buchung vor | prüft Pflichtangaben, Steuerfall und Konten |
| beantworten Rückfragen im Chat | bucht Routinefälle direkt, alles andere nach Ihrer Zustimmung | speichert an jeder Buchung, über welchen Weg sie kam |
| schreiben den Monat in der App fest | bereitet Prüfbericht und Voranmeldung vor | sperrt den Monat und holt einen Zeitstempel |
| übertragen die Voranmeldung in Mein ELSTER und tragen das Transferticket in der App ein | erinnert an Fristen aus dem Pflichtenkalender | hält die Übermittlung fest |
| stellen den Jahresabschluss auf, lassen ihn feststellen und legen ihn offen | bucht die Abschlussbausteine nach Ihrer Zustimmung, bereitet Erklärungen und E-Bilanz vor | zeigt Erklärungen formularnah und exportiert sie |

Was nur Sie in der App erledigen, steht unter
[KI-Agenten](ki-agenten/README.md#was-der-agent-darf). Bis zur Festschreibung
lässt sich jede Buchung durch Storno und neue Buchung korrigieren.

In der App haben Sie fünf Bereiche: **Übersicht** mit Aufgaben und Fristen,
**Bücher** mit Belegen, Bank, Rechnungen, Journal, Konten und Anlagen,
**Abschlüsse** für Monat, Jahr und Erklärungen, **Unternehmen** mit
Stammdaten und Kontakten, **Einstellungen** mit verbundenen Apps und
Sicherung.

## Rückmeldung geben

Melden Sie Fehler und unverständliche Abläufe über die
[GitHub-Issues](https://github.com/mxzinke/buchfink/issues). Nennen Sie
Programmversion, Betriebssystem, KI-Client, die Schritte und das erwartete
Ergebnis. Verwenden Sie erfundene Daten. Wie Sie sonst beitragen können, steht
unter [Mitwirken](../../CONTRIBUTING.md).
