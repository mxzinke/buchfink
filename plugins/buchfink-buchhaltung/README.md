# Buchfink Buchhaltung für Claude Code

Dieses Plugin verbindet Claude Code mit der Buchfink-App auf Ihrem Rechner und
bringt die Buchfink-Skills mit. Claude übernimmt auf Ihre Anweisung Belege und
Kontoauszüge aus dem Eingang, bucht, ordnet Bankumsätze zu, stellt Rechnungen
aus und bereitet Abschlüsse und Erklärungen vor. Was keine Routine ist, legt
es Ihnen vorher im Chat vor. Festschreiben, Übermittlungen bestätigen,
Aufstellen, Feststellen und Offenlegen bleiben in Buchfink und damit bei
Ihnen.

## Inhalt

- MCP-Server `buchfink` unter `http://127.0.0.1:7465/mcp`. Buchfink startet ihn
  mit der App. Die Anmeldung bestätigen Sie im Fenster von Buchfink.
- Elf Skills: `buchfink-grundlagen`, `buchfink-belege`, `buchfink-bank`,
  `buchfink-rechnungen`, `buchfink-monatsabschluss`,
  `buchfink-jahresabschluss`, `buchfink-steuererklaerungen`,
  `buchfink-gruendung`, `buchfink-lohn`, `buchfink-gesellschafter`,
  `buchfink-beteiligungen`.

## Installieren

Buchfink muss laufen, ein Unternehmen muss geöffnet sein. Dann in Claude Code:

```text
/plugin marketplace add mxzinke/buchfink
/plugin install buchfink-buchhaltung@buchfink
/mcp
```

Wählen Sie in `/mcp` den Server `plugin:buchfink-buchhaltung:buchfink` und
melden Sie sich an. Die vollständige Anleitung mit Fehlerbildern steht unter
[docs/nutzung/ki-agenten/claude-code.md](../../docs/nutzung/ki-agenten/claude-code.md).

## Für Mitwirkende

Die Skills liegen einmal unter [`skills/`](../../skills) im Wurzelordner des
Repositorys. `skills/<name>` in diesem Ordner ist ein relativer Symlink auf
`../../../skills/<name>`. Claude Code kopiert beim Installieren aus einem
Git-Marketplace das Ziel jedes Symlinks in seinen Cache (beobachtet mit Claude
Code 2.1.302 unter Linux). Die Dokumentation von Claude Code sagt dazu nichts
Verbindliches; wiederholen Sie deshalb vor jedem Release die Installation aus
dem Git-Marketplace, auch unter Windows und macOS, und halten Sie das Ergebnis
hier fest. Unter Windows ungeprüft: Git for Windows legt Symlinks ohne
`core.symlinks=true` (und Entwicklermodus) als kleine Textdateien an; das
Plugin hätte dann leere Skill-Ordner.

Nach einem neuen oder umbenannten Skill:

```sh
ln -s ../../../skills/<name> plugins/buchfink-buchhaltung/skills/<name>
python3 scripts/package_skills.py --check-only
```

Das Skript meldet fehlende, überzählige oder falsch gesetzte Links.
