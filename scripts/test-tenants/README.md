# Testmandanten

Erzeugt drei erfundene Unternehmen mit Belegen, E-Rechnungen und
Kontoauszügen, um Buchfink mit einem verbundenen Agenten durchzuspielen. Was
die Mandanten abdecken und wie ein Testlauf geht, steht unter
[Prüfszenarien](../../docs/entwicklung/pruefszenarien.md#testmandanten-mit-einem-agenten).

```sh
python3 scripts/test-tenants/gen.py ~/Buchfink-Testdaten
```

Ohne Zielordner schreibt der Generator nach `.cache/testdaten/`. Er löscht die
drei Mandantenordner im Ziel und legt sie neu an, mit je einer README:
Stammdaten, Schrittordner, Dateien mit Testfall, erwartete Buchungssätze und
Kontrollwerte. Die README im Zielordner gibt die Übersicht über alle drei.

Braucht Python 3 und [Typst](https://typst.app), für das Foto eines
Kassenbons `sips` (macOS) oder ImageMagick. Die Schriften der Vorlage sind die
von macOS; auf anderen Systemen setzt Typst Ersatzschriften.

| Datei | Inhalt |
|---|---|
| `gen.py` | Einstieg und Übersicht im Zielordner |
| `common.py` | Beträge, Prüfziffern, E-Rechnung in CII und UBL, ZUGFeRD, CAMT.053, Aufruf von Typst |
| `m1.py`, `m2.py`, `m3.py` | je ein Mandant: Dokumente, README, Kontrollwerte |
| `main.typ`, `lib.typ` | Vorlage der PDF-Dokumente, PDF/A-3 mit eingebettetem Datensatz |

Alle Angaben sind erfunden. IBANs, USt-IdNrn., Gläubiger-IDs und die
Leitweg-ID haben gültige Prüfziffern, gehören aber zu keinem echten Konto.
Behördennamen sind echt, die Schreiben nicht.
