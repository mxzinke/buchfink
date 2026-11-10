"""Erzeugt die Testdokumente der drei Mandanten: python3 scripts/test-tenants/gen.py [Zielordner]"""

import os
import shutil
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
if len(sys.argv) > 1:
    os.environ["BUCHFINK_TESTDATEN"] = sys.argv[1]

from common import BASE, BUILD  # noqa: E402
import m1, m2, m3  # noqa: E402

OVERVIEW = """# Buchfink-Testdaten

Drei erfundene Unternehmen mit Belegen, E-Rechnungen und Kontoauszügen, um Buchfink mit einem verbundenen Agenten
durchzuspielen: Mandant anlegen, Agent verbinden, Belege Schritt für Schritt in den Eingang legen, buchen lassen und
das Ergebnis gegen die erwarteten Werte prüfen.

Alle Angaben sind erfunden. Jedes Dokument trägt unten den Vermerk „Testdokument für Buchfink · alle Angaben
erfunden“. IBANs, USt-IdNrn., Gläubiger-IDs und die Leitweg-ID haben gültige Prüfziffern, gehören aber zu keinem
echten Konto. Mail-Adressen enden auf `.example`. Behördennamen (Finanzamt, Amtsgericht, Kammer) sind echt, die
Schreiben nicht.

## Die drei Mandanten

| Ordner | Unternehmen | Schwerpunkt | Zeitraum |
|---|---|---|---|
| [`1-Nordlicht-Digital-GmbH`](1-Nordlicht-Digital-GmbH/README.md) | GmbH, Hamburg, 2 Gesellschafter | Gründung mit Halbeinzahlung, Einforderung, Auslagen, erste Ausgangsrechnungen, XRechnung an eine Behörde | 03.08.–16.10.2026 |
| [`2-Kranich-Software-GmbH`](2-Kranich-Software-GmbH/README.md) | Einpersonen-GmbH, München | laufender Betrieb: Direktbuchung, Kreditkarte, Lohn, Sonderfälle, Voranmeldung mit Dauerfristverlängerung | 01.06.–16.10.2026 |
| [`3-Waldkauz-Projekte-UG`](3-Waldkauz-Projekte-UG/README.md) | UG, Köln, Geschäftsjahr 01.10.–30.09. | Projektgeschäft mit Abschlägen, Kundeninsolvenz, Jahresabschluss und UG-Rücklage | 01.10.2025–16.10.2026 |

## Was wo geprüft wird

| Testfall | 1 Nordlicht | 2 Kranich | 3 Waldkauz |
|---|:-:|:-:|:-:|
| Gründung, Zeichnung, Einzahlung | ✓ (Halbeinzahlung) | ✓ (voll) | ✓ (UG) |
| Einforderung der Resteinlagen | ✓ | | |
| Rumpfgeschäftsjahr / abweichendes Geschäftsjahr | ✓ Rumpfjahr | ✓ Rumpfjahr | ✓ 01.10.–30.09. |
| Auslage eines Gesellschafters | ✓ (2×) | | ✓ (Reisekosten) |
| Foto eines Kassenbons (JPG) | ✓ | | |
| ZUGFeRD-PDF | ✓ | ✓ | ✓ |
| XRechnung (CII-XML ohne PDF) | | ✓ | |
| UBL-XML (Peppol) | | ✓ | |
| PDF ohne Datensatz (Hinweis E-Rechnungspflicht) | ✓ | ✓ | ✓ |
| Direktbuchung nach drei Rechnungen mit bestätigter Standardgruppe | | ✓ | |
| Gutschrift eines Lieferanten (Typ 381) | | ✓ | |
| Dublette (gleiche Datei erneut) | | ✓ | |
| Geänderte Bankverbindung, Betrugsverdacht | | ✓ | |
| Zwei Steuersätze auf einem Beleg | | ✓ (Hotel, Bewirtung) | |
| Bewirtung 70/30, Trinkgeld | | ✓ | |
| Geschenke über 50 € | | ✓ | |
| Skonto | | ✓ | |
| Teilzahlung, Mahnung, Verzug | | ✓ (Mahnung im Testlauf nicht prüfbar) | |
| Firmenkreditkarte mit Monatsabrechnung | | ✓ | |
| Fremdwährung (USD), Auslandseinsatzentgelt | | ✓ | |
| § 13b Abs. 1 (EU-Leistung) | ✓ | ✓ | |
| § 13b Abs. 2 Nr. 1 (Drittland) | | ✓ | |
| Innergemeinschaftlicher Erwerb | | ✓ | |
| Ausländische USt ohne Vorsteuerabzug | ✓ | | |
| Ausgangsrechnung Inland / EU-Reverse-Charge / Behörde (XRechnung) | ✓ / – / ✓ | ✓ / ✓ / – | ✓ / – / – |
| Abschlagsrechnungen und Schlussrechnung | | | ✓ |
| Mindest-Istversteuerung | ✓ (Coworking Okt.) | | ✓ |
| Lohnjournal | | ✓ (GGF + Angestellter) | ✓ (GGF) |
| Gesellschafterdarlehen | | ✓ | |
| Zinsen mit Kapitalertragsteuer | | ✓ | |
| Steuerlastschrift aufteilen (USt + LSt) | | ✓ | |
| Sondervorauszahlung, Dauerfristverlängerung | | ✓ | |
| Post vom Amt ohne Buchung (Steuernummer, USt-IdNr.) | ✓ | ✓ | |
| Anlagen: Aktivierung, GWG, Nebenkosten | ✓ | ✓ | ✓ |
| Forderungsausfall, Insolvenz, § 17 UStG | | | ✓ |
| Unfertige Leistungen, Rechnungsabgrenzung, Rückstellungen | | | ✓ |
| Steuerrückstellung, UG-Rücklage, Ergebnisverwendung | | | ✓ |

Nicht abgedeckt ist der Umstieg mit Eröffnungsbilanz: Eröffnungsbuchungen eines Umsteigers kann Buchfink laut
`skills/buchfink/references/grundlagen/arbeitsweise.md` noch nicht, weder als Werkzeug noch in der App.

## Ablauf je Mandant

1. In Buchfink einen neuen Mandanten anlegen, Ablage und Eingang in **leeren Ordnern außerhalb dieses
   Testordners** (z. B. `~/Buchfink-Mandanten/<name>/`). Die Werte stehen in der README des Mandanten.
2. Agent verbinden (Einstellungen › Apps verbinden) und `status` aufrufen lassen.
3. Unternehmensdaten, Gründung und Bankkonto aus der README vom Agenten vorschlagen lassen
   (`stammdaten_vorschlagen`) und in der App nach der Vorschau übernehmen. Schreiben darf der Agent sie nicht.
4. Schrittordner der Reihe nach abarbeiten: Dateien ins Fenster ziehen, „Buche die Belege im Eingang.“ sagen,
   Rückfragen beantworten, Ergebnis gegen die Tabelle in der README prüfen.
5. Den ersten Kontoauszug jedes Mandanten liest **der Mensch in der App** ein; erst danach gelten die weiteren, die
   der Agent einliest, als geprüft.

Allgemeine Hinweise:

- Die Belege gelten als am Belegdatum eingegangen. Der Agent soll `eingang_am` setzen; sonst rechnet Buchfink die
  Vorsteuer dem Ablagetag im Oktober 2026 zu.
- Ausgangsrechnungen stellt der Agent in der angegebenen Reihenfolge aus, damit die Nummern (RE-2026-0001 …) zu den
  Verwendungszwecken der Kontoauszüge passen.
- Die Kontoauszüge (CAMT.053) haben Anfangs- und Endsaldo und schließen lückenlos aneinander an.
- Die erwarteten Buchungssätze und Kontrollwerte in den READMEs sind die fachlich erwartete Lösung. Weicht Buchfink
  ab, ist das ein Befund zum Nachsehen, nicht automatisch ein Fehler der App.

## Dateien neu erzeugen

Der Generator liegt im Repository unter `scripts/test-tenants/`:

```sh
python3 scripts/test-tenants/gen.py <Zielordner>
```

Braucht Python 3 und Typst (PDF/A-3 mit eingebettetem `factur-x.xml`), für das Foto eines Kassenbons `sips`
(macOS) oder ImageMagick. Der Generator löscht die drei Mandantenordner im Zielordner und legt sie neu an.
"""


def main():
    for t in (m1.T, m2.T, m3.T):
        if os.path.isdir(t.dir):
            shutil.rmtree(t.dir)
    if os.path.isdir(BUILD):
        shutil.rmtree(BUILD)
    for mod in (m1, m2, m3):
        text = mod.build()
        with open(os.path.join(mod.T.dir, "README.md"), "w", encoding="utf-8") as f:
            f.write(text)
        print(f"{os.path.basename(mod.T.dir)}: {len(mod.T.entries)} Dateien")
    with open(os.path.join(BASE, "README.md"), "w", encoding="utf-8") as f:
        f.write(OVERVIEW)
    shutil.rmtree(BUILD, ignore_errors=True)


if __name__ == "__main__":
    main()
