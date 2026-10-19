# Quellenliste des Skills buchfink

Liste für den Skill unter `skills/buchfink/`. Jede Fundstelle, die eine Datei
des Skills zitiert, steht hier mit Link, Zweck und Prüfvermerk. Prüfer gehen
die Liste von oben nach unten durch.

## Regeln für Einträge

- Eine Zeile je Norm oder Schreiben. Bei Gesetzen eine Zeile je Paragraph.
- Spalte „Geprüft“: Datum und Weg. `Text` heißt: Wortlaut am 17.10.2026 (bzw.
  am genannten Tag) gelesen und die zitierte Aussage abgeglichen. `Titel` heißt:
  nur Erreichbarkeit und Überschrift geprüft; die Aussage im Skill ist allgemein
  gehalten oder muss vom Fach-Agenten noch am Wortlaut geprüft werden.
- Spalte „Genutzt in“: Bereich und Datei unter `references/`, kurz
  (`grundlagen/gobd`); weitere Dateien desselben Bereichs folgen ohne
  Bereich. `SKILL` ist `skills/buchfink/SKILL.md`, `ablauf` die Ablauf-Referenz
  eines Bereichs.
- Werte, die sich jährlich ändern, bekommen einen eigenen Eintrag unter
  „Werte mit Stand“.
- Nicht verifizierbare Angaben kommen unter „Offen“, nicht in einen Skill.

## Zugang zu den Quellen (Stand 17.10.2026)

- gesetze-im-internet.de ist per `curl` und WebFetch erreichbar; einzelne
  Abrufe brechen ab und gelingen beim zweiten Versuch. Paragraphen-URL:
  `https://www.gesetze-im-internet.de/<gesetz>/__<paragraph>.html`. Umlaute im
  Kürzel werden zu `_` (z. B. `r_ckabzinsv`).
- bundesfinanzministerium.de und amtliche-handbuecher.bundesfinanzministerium.de
  antworten auf `curl` und WebFetch mit 503 („Security Check“). BMF-Schreiben
  deshalb über den Spiegel des Bayerischen Landesamts für Steuern
  (lfst.bayern.de) oder anderer Landesfinanzbehörden lesen und im Skill diesen
  Link angeben. Die BMF-Fundstelle (Datum, GZ, BStBl) steht trotzdem dabei.

## 1. Gesetze

### HGB

| Norm | Link | Wofür | Geprüft | Genutzt in |
|---|---|---|---|---|
| § 6 HGB | https://www.gesetze-im-internet.de/hgb/__6.html | Handelsgesellschaft gilt als Kaufmann | Text 17.10.2026 | gruendung/eroeffnungsbilanz; grundlagen (Hintergrund) |
| § 238 HGB | https://www.gesetze-im-internet.de/hgb/__238.html | Buchführungspflicht, sachverständiger Dritter, Verfolgbarkeit (Belegprinzip) | Text 17.10.2026 | grundlagen/gobd |
| § 239 HGB | https://www.gesetze-im-internet.de/hgb/__239.html | lebende Sprache, vollständig/richtig/zeitgerecht/geordnet, Unveränderbarkeit, Datenträger | Text 17.10.2026 | grundlagen/aufbewahrung, gobd, pflichten |
| § 240 HGB | https://www.gesetze-im-internet.de/hgb/__240.html | Inventar; Geschäftsjahr höchstens zwölf Monate (Abs. 2 Satz 2) | Text 17.10.2026 | grundlagen/arbeitsweise, pflichten; gruendung/eroeffnungsbilanz |
| § 242 HGB | https://www.gesetze-im-internet.de/hgb/__242.html | Eröffnungsbilanz, Jahresabschluss | Text 17.10.2026 | grundlagen/pflichten; jahresabschluss/ablauf, rueckstellungen; gruendung/eroeffnungsbilanz |
| § 245 HGB | https://www.gesetze-im-internet.de/hgb/__245.html | Unterzeichnung des Jahresabschlusses | Titel 17.10.2026 | grundlagen/aufbewahrung, pflichten; gruendung/eroeffnungsbilanz |
| § 257 HGB | https://www.gesetze-im-internet.de/hgb/__257.html | Aufbewahrung: Unterlagen (Abs. 1), Handelsbriefe (Abs. 2), Wiedergabe (Abs. 3), Fristen 10/8/6 Jahre (Abs. 4), Beginn (Abs. 5) | Text 17.10.2026 | grundlagen/aufbewahrung; gruendung/eroeffnungsbilanz |
| § 264 HGB | https://www.gesetze-im-internet.de/hgb/__264.html | Aufstellung durch gesetzliche Vertreter, Fristen 3/6 Monate | Text 17.10.2026 | grundlagen/arbeitsweise, pflichten; gruendung/rumpfjahr |
| § 267 HGB | https://www.gesetze-im-internet.de/hgb/__267.html | Größenklassen | Titel 17.10.2026 | grundlagen/grenzen |
| § 290 HGB | https://www.gesetze-im-internet.de/hgb/__290.html | Konzernabschluss | Titel 17.10.2026 | grundlagen/grenzen; beteiligungen/erwerb-ausweis |
| § 316 HGB | https://www.gesetze-im-internet.de/hgb/__316.html | Prüfungspflicht | Titel 17.10.2026 | grundlagen/grenzen |
| § 325 HGB | https://www.gesetze-im-internet.de/hgb/__325.html | Offenlegung des festgestellten Abschlusses | Text 17.10.2026 | grundlagen/arbeitsweise, pflichten; gruendung/rumpfjahr |
| Art. 95 EGHGB | https://www.gesetze-im-internet.de/hgbeg/art_95.html | Übergang Frist acht Jahre für Buchungsbelege | Text 17.10.2026 | grundlagen/aufbewahrung |

### AO und EGAO

| Norm | Link | Wofür | Geprüft | Genutzt in |
|---|---|---|---|---|
| § 140 AO | https://www.gesetze-im-internet.de/ao_1977/__140.html | handelsrechtliche Pflichten gelten für die Besteuerung | Text 17.10.2026 | grundlagen/gobd |
| § 145 AO | https://www.gesetze-im-internet.de/ao_1977/__145.html | Nachvollziehbarkeit für sachverständige Dritte | Text 17.10.2026 | grundlagen/gobd |
| § 146 AO | https://www.gesetze-im-internet.de/ao_1977/__146.html | Einzeln, vollständig, richtig, zeitgerecht, geordnet; Kasse täglich; Inland/EU/Drittstaat, Verzögerungsgeld 2.500–250.000 € (Abs. 2–2c); Sprache (Abs. 3); Unveränderbarkeit (Abs. 4) | Text 17.10.2026 | grundlagen/aufbewahrung, gobd, pflichten |
| § 147 AO | https://www.gesetze-im-internet.de/ao_1977/__147.html | Aufbewahrungspflichtige Unterlagen, Wiedergabe, Fristen 10/8/6, Ablaufhemmung (Abs. 3 Satz 5), Beginn (Abs. 4), Lesbarmachung (Abs. 5), Datenzugriff Z1–Z3 und Systemwechsel (Abs. 6) | Text 17.10.2026 | grundlagen/aufbewahrung; gruendung/eroeffnungsbilanz |
| § 153 AO | https://www.gesetze-im-internet.de/ao_1977/__153.html | Berichtigung unrichtiger Erklärungen | Text 17.10.2026 | rechnungen/storno-berichtigung; monatsabschluss/berichtigung |
| § 158 AO | https://www.gesetze-im-internet.de/ao_1977/__158.html | Beweiskraft der Buchführung | Text 17.10.2026 | grundlagen/gobd |
| § 162 AO | https://www.gesetze-im-internet.de/ao_1977/__162.html | Schätzung | Text 17.10.2026 | grundlagen/gobd, pflichten |
| § 169 AO | https://www.gesetze-im-internet.de/ao_1977/__169.html | Festsetzungsfrist | Text (Abs. 1) 17.10.2026 | grundlagen/aufbewahrung |
| § 170 AO | https://www.gesetze-im-internet.de/ao_1977/__170.html | Beginn der Festsetzungsfrist | Titel 17.10.2026 | grundlagen/aufbewahrung |
| § 171 AO | https://www.gesetze-im-internet.de/ao_1977/__171.html | Ablaufhemmung | Titel 17.10.2026 | grundlagen/aufbewahrung |
| § 371 AO | https://www.gesetze-im-internet.de/ao_1977/__371.html | Selbstanzeige | Titel 17.10.2026 | grundlagen/grenzen |
| § 379 AO | https://www.gesetze-im-internet.de/ao_1977/__379.html | Steuergefährdung: unrichtige Belege, unrichtiges Verbuchen, Aufbewahrung | Text 17.10.2026 | grundlagen/gobd |
| Art. 97 § 19a EGAO | https://www.gesetze-im-internet.de/aoeg_1977/art_97__19a.html | Übergang Frist acht Jahre (Abs. 2) | Text 17.10.2026 | grundlagen/aufbewahrung |

### GmbHG, EStG, KStG, UStG, InsO, AStG

| Norm | Link | Wofür | Geprüft | Genutzt in |
|---|---|---|---|---|
| § 5a GmbHG | https://www.gesetze-im-internet.de/gmbhg/__5a.html | UG: Volleinzahlung, keine Sacheinlage, gesetzliche Rücklage, Einberufung bei drohender Zahlungsunfähigkeit (Abs. 4) | Text 17.10.2026 | grundlagen/grenzen, pflichten |
| § 13 GmbHG | https://www.gesetze-im-internet.de/gmbhg/__13.html | GmbH als Handelsgesellschaft | Text 17.10.2026 | grundlagen/arbeitsweise; gruendung/ablauf, eroeffnungsbilanz |
| § 41 GmbHG | https://www.gesetze-im-internet.de/gmbhg/__41.html | Geschäftsführer sorgen für ordnungsmäßige Buchführung | Text 17.10.2026 | grundlagen/gobd |
| § 42a GmbHG | https://www.gesetze-im-internet.de/gmbhg/__42a.html | Vorlage, Feststellung binnen 8 bzw. 11 Monaten | Text 17.10.2026 | grundlagen/arbeitsweise, pflichten; gruendung/rumpfjahr |
| § 49 GmbHG | https://www.gesetze-im-internet.de/gmbhg/__49.html | Einberufung bei Verlust der Hälfte des Stammkapitals (Abs. 3) | Text 17.10.2026 | grundlagen/grenzen |
| § 4a EStG | https://www.gesetze-im-internet.de/estg/__4a.html | Wirtschaftsjahr, Umstellung nur im Einvernehmen mit dem Finanzamt | Text 17.10.2026 | gruendung/rumpfjahr |
| § 7 KStG | https://www.gesetze-im-internet.de/kstg_1977/__7.html | Wirtschaftsjahr bei Buchführungspflichtigen (Abs. 4) | Text 17.10.2026 | grundlagen/pflichten |
| § 8c KStG | https://www.gesetze-im-internet.de/kstg_1977/__8c.html | Verlustabzug bei schädlichem Beteiligungserwerb | Text (Abs. 1 Satz 1) 17.10.2026 | steuererklaerungen/kst |
| § 14 KStG | https://www.gesetze-im-internet.de/kstg_1977/__14.html | Organschaft | Titel 17.10.2026 | grundlagen/grenzen |
| § 17 KStG | https://www.gesetze-im-internet.de/kstg_1977/__17.html | Organschaft anderer Kapitalgesellschaften (GmbH) | Text (Abs. 1) 17.10.2026 | grundlagen/grenzen |
| § 2 UStG | https://www.gesetze-im-internet.de/ustg_1980/__2.html | Unternehmer (Abs. 1), umsatzsteuerliche Organschaft (Abs. 2 Nr. 2) | Text (Abs. 2) 17.10.2026 | grundlagen/grenzen; beteiligungen/holding-umsatzsteuer |
| § 9 UStG | https://www.gesetze-im-internet.de/ustg_1980/__9.html | Verzicht auf Steuerbefreiungen | Titel 17.10.2026 | grundlagen/grenzen; beteiligungen/holding-umsatzsteuer |
| § 13 UStG | https://www.gesetze-im-internet.de/ustg_1980/__13.html | Entstehung der Steuer; Anzahlungen Abs. 1 Nr. 1 Buchst. a Satz 4 | Text 17.10.2026 | rechnungen/anzahlungen |
| § 13b UStG | https://www.gesetze-im-internet.de/ustg_1980/__13b.html | Steuerschuldnerschaft des Leistungsempfängers; Bauleistungen Abs. 2 Nr. 4 | Text (Abs. 2 Nr. 4) 17.10.2026 | grundlagen/grenzen; monatsabschluss/ustva |
| § 14 UStG | https://www.gesetze-im-internet.de/ustg_1980/__14.html | Rechnung, E-Rechnung (Abs. 1 Satz 3, 6), Gutschrift (Abs. 2 Satz 5) | Text (Abs. 1, 2) 17.10.2026 | grundlagen/aufbewahrung, grenzen, pflichten |
| § 14b UStG | https://www.gesetze-im-internet.de/ustg_1980/__14b.html | Aufbewahrung von Rechnungen acht Jahre, Echtheit/Unversehrtheit/Lesbarkeit, Beginn | Text 17.10.2026 | grundlagen/aufbewahrung |
| § 15 UStG | https://www.gesetze-im-internet.de/ustg_1980/__15.html | Vorsteuerabzug setzt Rechnung nach §§ 14, 14a voraus (Abs. 1 Satz 1 Nr. 1 Satz 2) | Text 17.10.2026 | grundlagen/arbeitsweise, gobd; beteiligungen/holding-umsatzsteuer |
| §§ 18i, 18j, 18k UStG | https://www.gesetze-im-internet.de/ustg_1980/__18i.html, …/__18j.html, …/__18k.html | OSS/IOSS | Titel 17.10.2026 | grundlagen/grenzen |
| § 19 UStG | https://www.gesetze-im-internet.de/ustg_1980/__19.html | Kleinunternehmer | Titel 17.10.2026 | grundlagen/grenzen; gruendung/steuerliche-erfassung |
| § 20 UStG | https://www.gesetze-im-internet.de/ustg_1980/__20.html | Istversteuerung | Titel 17.10.2026 | grundlagen/grenzen; gruendung/steuerliche-erfassung |
| § 22 UStG | https://www.gesetze-im-internet.de/ustg_1980/__22.html | Aufzeichnungspflichten | Titel 17.10.2026 | grundlagen/gobd |
| §§ 25, 25a, 25b UStG | https://www.gesetze-im-internet.de/ustg_1980/__25.html, …/__25a.html, …/__25b.html | Reiseleistungen, Differenzbesteuerung, Dreiecksgeschäft | Titel 17.10.2026 | grundlagen/grenzen |
| § 15a InsO | https://www.gesetze-im-internet.de/inso/__15a.html | Antragspflicht, 3 Wochen Zahlungsunfähigkeit, 6 Wochen Überschuldung | Text 17.10.2026 | grundlagen/grenzen |
| § 19 InsO | https://www.gesetze-im-internet.de/inso/__19.html | Überschuldung mit Fortführungsprognose | Text 17.10.2026 | grundlagen/grenzen; bank/darlehen |
| § 1 AStG | https://www.gesetze-im-internet.de/astg/__1.html | Berichtigung von Einkünften (nahestehende Personen) | Titel 17.10.2026 | grundlagen/grenzen |

## 2. Verwaltungsanweisungen

| Schreiben | Link | Wofür | Geprüft | Genutzt in |
|---|---|---|---|---|
| GoBD, BMF-Schreiben vom 28.11.2019, IV A 4 - S 0316/19/10003 :001, BStBl I S. 1269 | https://www.lfst.bayern.de/fileadmin/RESSOURCEN/INFORMATIONEN/Steuerinfos/Weitere_Themen/Aussenpruefung/2019-11-28-GoBD-1.pdf | Rz. 21 (Verantwortung), 30–35 (Nachvollziehbarkeit), 36–43 (Vollständigkeit), 44 (Richtigkeit), 45–52 (zeitgerecht, 10 Tage, Folgemonat), 53–57 (Ordnung), 58–60, 107–112 (Unveränderbarkeit, Festschreibung), 61–81 (Belegwesen, Eigenbeleg, Rz. 77 Mindestinhalt), 85–99 (Grundaufzeichnung, Journal, Konten), 100–106 (IKS, Datensicherheit), 113–144 (Aufbewahrung, Scannen, Systemwechsel), 151–155 (Verfahrensdokumentation), 158 ff. (Datenzugriff) | Text 17.10.2026 (PDF über LfSt Bayern) | grundlagen/arbeitsweise, aufbewahrung, gobd, pflichten |
| 1. Änderung der GoBD, BMF-Schreiben vom 11.03.2024, IV D 2 - S 0316/21/10001 :002, BStBl I S. 374 | https://www.lfst.bayern.de/fileadmin/RESSOURCEN/INFORMATIONEN/Steuerinfos/Weitere_Themen/Aussenpruefung/2024-03-11-aenderung-gobd.pdf | geänderte Rz. 6, 11, 15, 39, 43, 94, 124 (aufgehoben), 128, 136, 159, 160, 161, 167–169, 176–178, 182–184 | Text 17.10.2026 | grundlagen/aufbewahrung, gobd, pflichten |
| 2. Änderung der GoBD, BMF-Schreiben vom 14.07.2025, IV D 2 - S 0316/00128/005/088 | https://www.lfst.bayern.de/fileadmin/RESSOURCEN/INFORMATIONEN/Steuerinfos/Weitere_Themen/Aussenpruefung/2025-07-14-GoBD-2-aenderung.pdf (BMF-Seite: https://www.bundesfinanzministerium.de/Content/DE/Downloads/BMF_Schreiben/Weitere_Steuerthemen/Abgabenordnung/2025-07-14-GoBD-2-aenderung.html, nicht abrufbar) | E-Rechnung: strukturierter Teil genügt (Rz. 119, 131), Kontoauszug/E-Rechnung als strukturierte Datei (Rz. 76), Beispiel ZUGFeRD (Rz. 125), Z2 (Rz. 166), Anwendung ab 14.07.2025 (Rz. 185) | Text 17.10.2026 | grundlagen/aufbewahrung, gobd, pflichten |

## 3. Werte mit Stand

| Wert | Stand | Quelle | Genutzt in |
|---|---|---|---|
| Aufbewahrung 10 / 8 / 6 Jahre | 17.10.2026, gilt seit 01.01.2025 | § 147 Abs. 3 AO, § 257 Abs. 4 HGB, § 14b Abs. 1 UStG, Art. 97 § 19a EGAO, Art. 95 EGHGB | grundlagen/aufbewahrung |
| Erfassung unbarer Vorgänge binnen 10 Tagen | GoBD i. d. F. 14.07.2025 | GoBD Rz. 47 | SKILL; grundlagen/gobd |
| Verzögerungsgeld 2.500 bis 250.000 € | 17.10.2026 | § 146 Abs. 2c AO | grundlagen/aufbewahrung |
| Insolvenzantrag: 3 Wochen Zahlungsunfähigkeit, 6 Wochen Überschuldung | 17.10.2026 | § 15a Abs. 1 InsO | grundlagen/grenzen |
| Feststellung binnen 8 Monaten, kleine GmbH 11 Monate | 17.10.2026 | § 42a Abs. 2 GmbHG | grundlagen (Hintergrund) |

## 4. Buchfink-eigene Quellen

| Quelle | Pfad | Wofür |
|---|---|---|
| MCP-Werkzeugkatalog | `docs/entwicklung/mcp.md` | Werkzeuge, Risikoklassen, Direktregeln, Fehlercodes, Idempotenz, Pfade |
| Umsetzungsstand | `docs/projekt/umsetzungsstand.md` | Funktionsumfang und Grenzen |
| Anforderungskatalog | `docs/anforderungen/*.md` | Prüfraster, Abgrenzungen |
| Journal-Storno | `internal/service/journal_service.go` (`ReverseOn`) | Storno mit heutigem Datum, Generalumkehr |
| Abschlussstände | `internal/domain/fiscalyear.go` | offen, aufgestellt, festgestellt, offengelegt |
| Agent-Skills-Spezifikation | https://agentskills.io/specification | Frontmatter, Ordneraufbau |

## 5. Offen

- BMF-Originallinks (bundesfinanzministerium.de) ließen sich am 17.10.2026
  nicht abrufen (503, auch beim zweiten Prüfgang). Ein Prüfer mit
  Browserzugang gleicht die Spiegel mit den BMF-PDFs ab. Die Pfade der
  Schreiben vom 15.10.2025 (E-Rechnung) und 19.11.2025 (Bewirtung) nennt der
  PwC-Steuerblog gleichlautend; der Pfad `2025-07-14-GoBD-2-aenderung.html` ist
  nirgends bestätigt. Skills verlinken nur die Spiegel.
- Prüfgang 17.10.2026, zweiter Teil: Die 13 Gesetzes-URLs ohne frühere Kopie
  (Art. 97 § 19a und § 36 EGAO, §§ 640, 641 BGB, § 84 EStDV, §§ 251, 270, 276,
  277, 285 HGB, Art. 93, 95 EGHGB, PDF der Anlage 1 GmbHG) sind live abgerufen
  (200) und gegen die zitierten Aussagen gelesen.

## 6. Ergänzungen der Fachbereiche

Fach-Agenten tragen ihre Quellen in die Tabellen oben ein (sortiert nach
Gesetz und Paragraph) und ihre jährlich wechselnden Werte unter „Werte mit
Stand“. Zeilen, die schon existieren, bekommen in „Genutzt in“ einen weiteren
Eintrag.

## Bereich belege

Quellen des Bereichs `references/belege/` (ablauf.md und Referenzen). Alle
Gesetzeslinks am 17.10.2026 mit HTTP 200 geprüft. „Text“ = Wortlaut gelesen und
Aussage abgeglichen.

### Gesetze

| Norm | Link | Wofür | Geprüft | Genutzt in |
|---|---|---|---|---|
| § 1a UStG | https://www.gesetze-im-internet.de/ustg_1980/__1a.html | innergemeinschaftlicher Erwerb, Voraussetzungen (Abs. 1) | Text 17.10.2026 | belege/steuerfaelle |
| § 3a UStG | https://www.gesetze-im-internet.de/ustg_1980/__3a.html | Ort B2B (Abs. 2), Grundstück (Abs. 3 Nr. 1), kurzfristige Vermietung Beförderungsmittel (Abs. 3 Nr. 2) | Text 17.10.2026 | belege/steuerfaelle |
| § 3d UStG | https://www.gesetze-im-internet.de/ustg_1980/__3d.html | Ort des ig. Erwerbs, fremde USt-IdNr. | Text 17.10.2026 | belege/steuerfaelle |
| § 4 UStG | https://www.gesetze-im-internet.de/ustg_1980/__4.html | Nr. 8 Buchst. a (Kredit), d (Zahlungsverkehr), e (Wertpapiere, ohne Verwahrung), Nr. 10 Buchst. a (Versicherung), Nr. 12 Buchst. a (Vermietung) | Text 17.10.2026 | belege/steuerfaelle; beteiligungen/holding-umsatzsteuer |
| § 12 UStG | https://www.gesetze-im-internet.de/ustg_1980/__12.html | 19 % (Abs. 1); 7 % Nr. 10 Personenbeförderung, Nr. 11 Beherbergung ohne Nebenleistungen, Nr. 15 Restaurant- und Verpflegungsdienstleistungen ohne Getränke (ohne Befristung im Wortlaut); Nullsatz Abs. 3 | Text 17.10.2026 (UStG zuletzt geändert durch Art. 5 G v. 29.06.2026) | belege/bewirtung-geschenke, steuerfaelle |
| § 13 UStG | https://www.gesetze-im-internet.de/ustg_1980/__13.html | Entstehung Erwerbsteuer (Abs. 1 Nr. 6) | Text 17.10.2026 | belege/steuerfaelle |
| § 13b UStG | https://www.gesetze-im-internet.de/ustg_1980/__13b.html | Steuerschuldnerschaft: Abs. 1 (EU-Leistung, VAZ der Leistung), Abs. 2 Nr. 1 (Rechnung, spätestens Folgemonat), Nr. 4, 8, 10, 11 (5.000 €), 12; Abs. 3, 4 Satz 2, 5 Satz 1/5/6, 6, 7 | Text 17.10.2026 | belege/steuerfaelle |
| § 14 UStG | https://www.gesetze-im-internet.de/ustg_1980/__14.html | Rechnungsbegriff, E-Rechnung, sonstige Rechnung (Abs. 1 Satz 1–6); E-Rechnungspflicht (Abs. 2 Satz 2 Nr. 1, Satz 3); Gutschrift (Abs. 2 Satz 5); Kontrollverfahren (Abs. 3 Satz 4, 5); Pflichtangaben (Abs. 4 Satz 1 Nr. 1–10); Berichtigung kein rückwirkendes Ereignis (Abs. 4 Satz 4); ausländischer Leistender bei § 13b (Abs. 7) | Text 17.10.2026 | belege/bewirtung-geschenke, e-rechnung-empfang, pflichtangaben |
| § 14a UStG | https://www.gesetze-im-internet.de/ustg_1980/__14a.html | Hinweis „Steuerschuldnerschaft des Leistungsempfängers“ (Abs. 5) | Text 17.10.2026 | belege/pflichtangaben, steuerfaelle |
| § 14c UStG | https://www.gesetze-im-internet.de/ustg_1980/__14c.html | unrichtiger (Abs. 1) und unberechtigter (Abs. 2) Steuerausweis | Text 17.10.2026 | belege/pflichtangaben, steuerfaelle |
| § 15 UStG | https://www.gesetze-im-internet.de/ustg_1980/__15.html | Vorsteuer: Abs. 1 Satz 1 Nr. 1 (Satz 2 Rechnung besitzen, Satz 3 Anzahlung), Nr. 2–4; Satz 2 (10 %); Abs. 1a (Ausschluss, Bewirtung Satz 2), 1b, 2, 3, 4 (Aufteilung) | Text 17.10.2026 | belege/bewirtung-geschenke, pflichtangaben, steuerfaelle, vorsteuer |
| § 15a UStG | https://www.gesetze-im-internet.de/ustg_1980/__15a.html | Vorsteuerberichtigung, 5/10 Jahre (Abs. 1), Fünftel/Zehntel (Abs. 5) | Text 17.10.2026 | belege/vorsteuer |
| § 16 UStG | https://www.gesetze-im-internet.de/ustg_1980/__16.html | Umrechnung Fremdwährung mit dem BMF-Durchschnittskurs des Monats der Leistung oder der Vereinnahmung davor, Tageskurs auf Gestattung (Abs. 6 Satz 1, 3) | Text 19.10.2026 | belege/fremdwaehrung-auslagen |
| § 17 UStG | https://www.gesetze-im-internet.de/ustg_1980/__17.html | Minderung der Bemessungsgrundlage (Abs. 1 Satz 2, 8), Uneinbringlichkeit (Abs. 2 Nr. 1), Rückgängigmachung (Nr. 3), Aufwendungen § 15 Abs. 1a (Nr. 5) | Text 17.10.2026 | belege/bewirtung-geschenke, vorsteuer |
| § 18g UStG | https://www.gesetze-im-internet.de/ustg_1980/__18g.html | Vorsteuervergütung in anderem Mitgliedstaat über BZSt | Text 17.10.2026 | belege/steuerfaelle |
| § 19 UStG | https://www.gesetze-im-internet.de/ustg_1980/__19.html | Kleinunternehmer: steuerfrei, 25.000 € Vorjahr / 100.000 € laufendes Jahr (Abs. 1) | Text 17.10.2026 | belege/steuerfaelle |
| § 22 UStG | https://www.gesetze-im-internet.de/ustg_1980/__22.html | getrennte Aufzeichnung; Abs. 4: Berechnungsgrundlagen für die Berichtigung nach § 15a aufzeichnen | Titel 17.10.2026 (Bezug über GoBD Rz. 55); Abs. 4 Text 17.10.2026 | belege/steuerfaelle, vorsteuer |
| § 27 UStG | https://www.gesetze-im-internet.de/ustg_1980/__27.html | Leistungszeitpunkt entscheidet über Rechtsänderung (Abs. 1 Satz 1); E-Rechnung Übergang (Abs. 38 Nr. 1–3, mit Übermittlungsfristen 31.12.2026 bzw. 31.12.2027) | Text 17.10.2026 | belege/e-rechnung-empfang, steuerfaelle |
| § 31 UStDV | https://www.gesetze-im-internet.de/ustdv_1980/__31.html | mehrere Dokumente (Abs. 1), Name/Anschrift (Abs. 2), Abkürzungen (Abs. 3), Kalendermonat (Abs. 4), Berichtigung (Abs. 5) | Text 17.10.2026 | belege/pflichtangaben |
| § 33 UStDV | https://www.gesetze-im-internet.de/ustdv_1980/__33.html | Kleinbetragsrechnung bis 250 €, nicht bei §§ 3c, 6a, 13b (Satz 3), immer sonstige Rechnung zulässig (Satz 4) | Text 17.10.2026 | belege/bewirtung-geschenke, pflichtangaben; bank/kreditkarte-geldtransit |
| § 34 UStDV | https://www.gesetze-im-internet.de/ustdv_1980/__34.html | Fahrausweise (Abs. 1, 2) | Text 17.10.2026 | belege/e-rechnung-empfang, pflichtangaben |
| § 34a UStDV | https://www.gesetze-im-internet.de/ustdv_1980/__34a.html | Rechnungen von Kleinunternehmern | Text 17.10.2026 | belege/e-rechnung-empfang, pflichtangaben, steuerfaelle |
| § 44 UStDV | https://www.gesetze-im-internet.de/ustdv_1980/__44.html | Bagatellgrenzen § 15a: 1.000 € (Abs. 1), 10 Prozentpunkte und 1.000 € (Abs. 2), 6.000 € (Abs. 3) | Text 17.10.2026 | belege/vorsteuer |
| § 4 EStG | https://www.gesetze-im-internet.de/estg/__4.html | Geschenke 50 € (Abs. 5 Satz 1 Nr. 1), Bewirtung 70 % und Nachweis (Nr. 2 Satz 1–3), Gästehaus, Jagd, Jacht (Nr. 3, 4), unangemessene Aufwendungen (Nr. 7), getrennte Aufzeichnung (Abs. 7) | Text 17.10.2026 | belege/bewirtung-geschenke, vorsteuer |
| § 6 EStG | https://www.gesetze-im-internet.de/estg/__6.html | GWG 800 €, Verzeichnis über 250 € (Abs. 2 Satz 1–5); Sammelposten über 250 € bis 1.000 €, Fünftel, einheitlich (Abs. 2a Satz 1–5) | Text 17.10.2026 | belege/anlagen-gwg; beteiligungen/bewertung-veraeusserung, erwerb-ausweis |
| § 7 EStG | https://www.gesetze-im-internet.de/estg/__7.html | lineare AfA, Zwölftel, Firmenwert 15 Jahre (Abs. 1 Satz 1–4); degressive AfA bis 30 % für Anschaffung 01.07.2025–31.12.2027 (Abs. 2) | Text 17.10.2026 | belege/anlagen-gwg |
| § 9b EStG | https://www.gesetze-im-internet.de/estg/__9b.html | abziehbare Vorsteuer keine Anschaffungskosten (Abs. 1) | Text 17.10.2026 | belege/anlagen-gwg, bewirtung-geschenke, vorsteuer |
| § 12 EStG | https://www.gesetze-im-internet.de/estg/__12.html | Kosten der Lebensführung (Nr. 1) | Text 17.10.2026 | belege/vorsteuer |
| § 37b EStG | https://www.gesetze-im-internet.de/estg/__37b.html | Pauschalierung 30 %, einheitlich, 10.000 €, als Lohnsteuer anmelden, Empfänger unterrichten | Text 17.10.2026 | belege/bewirtung-geschenke |
| § 52 EStG | https://www.gesetze-im-internet.de/estg/__52.html | Geschenkegrenze 50 € ab Wirtschaftsjahr nach dem 31.12.2023 (Abs. 6 Satz 10) | Text 17.10.2026 | belege/bewirtung-geschenke |
| § 10 KStG | https://www.gesetze-im-internet.de/kstg_1977/__10.html | nicht abziehbare Vorsteuer auf Aufwendungen nach § 4 Abs. 5 Nr. 1–4, 7, Abs. 7 EStG (Nr. 2) | Text 17.10.2026 | belege/vorsteuer |
| § 31 KStG | https://www.gesetze-im-internet.de/kstg_1977/__31.html | § 37b EStG gilt entsprechend (Abs. 1 Satz 3) | Text 17.10.2026 | belege/bewirtung-geschenke |
| § 253 HGB | https://www.gesetze-im-internet.de/hgb/__253.html | planmäßige Abschreibung (Abs. 3 Satz 1) | Text 17.10.2026 | belege/anlagen-gwg; beteiligungen/bewertung-veraeusserung, erwerb-ausweis |
| § 255 HGB | https://www.gesetze-im-internet.de/hgb/__255.html | Anschaffungskosten, Nebenkosten, Preisminderungen (Abs. 1); Herstellungskosten, wesentliche Verbesserung (Abs. 2 Satz 1) | Text 17.10.2026 | belege/anlagen-gwg; beteiligungen/erwerb-ausweis |
| § 256a HGB | https://www.gesetze-im-internet.de/hgb/__256a.html | Umrechnung zum Stichtag (Verweis auf jahresabschluss) | Text 17.10.2026 | jahresabschluss/forderungen |

### Verwaltungsanweisungen

| Schreiben | Link | Wofür | Geprüft | Genutzt in |
|---|---|---|---|---|
| GoBD, BMF vom 28.11.2019 (bestehender Eintrag oben) | https://www.lfst.bayern.de/fileadmin/RESSOURCEN/INFORMATIONEN/Steuerinfos/Weitere_Themen/Aussenpruefung/2019-11-28-GoBD-1.pdf | Rz. 52 (Folgemonat), 55 (getrennte Verbuchung steuerpflichtig/steuerfrei/nicht steuerbar), 67 (Belegsicherung), 77 (Mindestinhalt, Währung und Kurs) | Text 17.10.2026 | belege/ablauf, fremdwaehrung-auslagen, steuerfaelle |
| BMF vom 15.10.2024, III C 2 - S 7287-a/23/10001 :007, BStBl I S. 1320 (Einführung E-Rechnung) | https://www.lfst.bayern.de/fileadmin/RESSOURCEN/INFORMATIONEN/Steuerinfos/Weitere_Themen/E-Rechnung/2024-10-15-einfuehrung-e-rechnung.pdf | Rn. 7 (sonstige Rechnung, PDF ohne Datensatz), 23 (Gesamtbetrag 250 €), 40 (Empfang, E-Mail-Postfach), 42 (Annahmeverweigerung), 55–59 (Vorsteuer bei falschem Format, Berichtigung wirkt zurück, objektive Nachweise, Vertrauensschutz), 62–65 (Übergangsregeln, kein Übergang beim Empfang) | Text 17.10.2026 (PDF über LfSt Bayern) | belege/e-rechnung-empfang, pflichtangaben |
| BMF vom 15.10.2025, III C 2 - S 7287-a/00019/007/243 (Änderung des Schreibens vom 15.10.2024 und des UStAE) | https://www.steuerschroeder.de/blog/wp-content/uploads/2025/10/2025-10-15-einfuehrung-obligatorische-e-rechnung.pdf (Kanzleispiegel; BMF: https://www.bundesfinanzministerium.de/Content/DE/Downloads/BMF_Schreiben/Steuerarten/Umsatzsteuer/Umsatzsteuer-Anwendungserlass/2025-10-15-einfuehrung-obligatorische-e-rechnung.pdf, 503) | Rn. 6a (Formatfehler), 6b (Geschäftsregelfehler), 17 (Gutschrift, § 13b zwischen Inländern), 35 (Pflichtangaben im strukturierten Teil), 35a (Inhaltsfehler, Validierung), 51a (keine Berichtigung bei § 17-Minderung); UStAE 14.1 Abs. 5 (Empfangspflicht), 14c.1 Abs. 4a (hybride Rechnung, Vorsteuer nur aus strukturiertem Teil), 15.2a Abs. 1 Satz 3, 4 | Text 17.10.2026 (Kanzleispiegel; LfSt Bayern verlinkt nur auf das BMF) | belege/ablauf, e-rechnung-empfang, pflichtangaben, vorsteuer; bank/skonto-differenzen |
| BMF vom 22.02.2022, IV C 3 - S 2190/21/10002 :025 (Nutzungsdauer Computerhardware und Software) | https://www.steuerschroeder.de/blog/wp-content/uploads/2022/02/Nutzungsdauer-von-Computerhardware-und-Software-zur-Dateneingabe-und-verarbeitung.pdf (Kanzleispiegel) | Rz. 1 (ein Jahr), 1.1 (keine Sofortabschreibung, kein Wahlrecht § 5 Abs. 1 EStG), 1.2 (Beginn, Bestandsverzeichnis, Abweichung möglich), 1.4 (volle AfA im Anschaffungsjahr nicht beanstandet), 2–5 (Begriffe), 6 (Anwendung ab Wj. nach 31.12.2020); ersetzt BMF vom 26.02.2021 (BStBl I S. 298) | Text 17.10.2026 (Kanzleispiegel; BStBl-Fundstelle nicht geprüft) | belege/anlagen-gwg |
| AfA-Tabelle für die allgemein verwendbaren Anlagegüter, BMF vom 15.12.2000, BStBl I S. 1532 | – | Erfahrungswerte Nutzungsdauer | Titel (Fundstelle aus BMF vom 22.02.2022, Rz. 8) | belege/anlagen-gwg |
| UStAE 15.6 Abs. 2 Satz 5 (Vorsteuer bei Bewirtung trotz Verstoß gegen § 4 Abs. 7 EStG) | https://datenbank.nwb.de/Dokument/378652_15___6/ (NWB-Datenbank, Fassung „VZ 2010–2026 (aktuell)“) | Verstoß nur gegen die einzelne und getrennte Aufzeichnung schließt die Vorsteuer für angemessene Bewirtungsaufwendungen nicht aus | Text 17.10.2026 (Verlagsdatenbank; amtlicher UStAE 503) | belege/bewirtung-geschenke |
| FG Berlin-Brandenburg vom 09.04.2019, 5 K 5119/18 | https://www.haufe.de/steuern/rechtsprechung/vorsteuerabzug-aus-bewirtungsaufwendungen-trotz-fehlendem-beleg_166_496064.html | Vorsteuer aus Bewirtung trotz fehlender Angaben, Nachholung im Einspruch | Text der Besprechung 17.10.2026 (Sekundärquelle) | belege/bewirtung-geschenke |
| FG Münster vom 27.04.2020, 5 K 1722/18 U | https://datenbank.nwb.de/Dokument/831949/ | Leitsatz 2: Nachweis der Bewirtungsaufwendungen für die Vorsteuer in der Form des § 4 Abs. 5 Satz 1 Nr. 2 Satz 2, 3 EStG | Leitsatz 17.10.2026 (Volltext kostenpflichtig) | belege/bewirtung-geschenke |
| BMF vom 19.11.2025, IV C 6 - S 2145/00026/005/033 (Bewirtungsaufwendungen) | https://www.bundesfinanzministerium.de/Content/DE/Downloads/BMF_Schreiben/Steuerarten/Einkommensteuer/2025-11-19-bewirtungskosten-als-betriebsausgaben.pdf (503) | nur Existenz und GZ genannt; Inhalt siehe Offen | Titel über Sekundärquellen (Deloitte Tax-News, PwC-Blog) | belege/bewirtung-geschenke |

### Werte mit Stand

| Wert | Stand | Quelle | Genutzt in |
|---|---|---|---|
| Kleinbetragsrechnung bis 250 € brutto | 17.10.2026 | § 33 UStDV | belege/ablauf, e-rechnung-empfang, pflichtangaben |
| E-Rechnung: sonstige Rechnung zulässig für Leistungen 2025/2026 (bis 31.12.2026), 2027 nur bei Vorjahresumsatz Aussteller bis 800.000 € (bis 31.12.2027), ab 2028 nicht | 17.10.2026 | § 27 Abs. 38 UStG | belege/ablauf, e-rechnung-empfang |
| § 13b-Grenze Mobilfunkgeräte, Schaltkreise, Metalle: 5.000 € je Vorgang | 17.10.2026 | § 13b Abs. 2 Nr. 10, 11 UStG | belege/ablauf, steuerfaelle |
| Kleinunternehmer: 25.000 € Vorjahr, 100.000 € laufendes Jahr | 17.10.2026 | § 19 Abs. 1 UStG | belege/steuerfaelle |
| Unternehmensnutzung mindestens 10 % für Vorsteuer auf Gegenstände | 17.10.2026 | § 15 Abs. 1 Satz 2 UStG | belege/vorsteuer |
| Vorsteuerberichtigung: 5 Jahre, Grundstücke 10 Jahre; Bagatellgrenzen 1.000 €, 10 Prozentpunkte/1.000 €, 6.000 € | 17.10.2026 | § 15a Abs. 1 UStG, § 44 UStDV | belege/anlagen-gwg, vorsteuer |
| Bewirtung: 70 % abziehbar | 17.10.2026 | § 4 Abs. 5 Satz 1 Nr. 2 EStG | belege/ablauf, bewirtung-geschenke |
| Geschenke: Freigrenze 50 € je Empfänger und Wirtschaftsjahr (ab Wj. nach 31.12.2023; davor 35 €) | 17.10.2026 | § 4 Abs. 5 Satz 1 Nr. 1, § 52 Abs. 6 Satz 10 EStG; Altwert aus `tax_params.go` | belege/ablauf, bewirtung-geschenke |
| § 37b EStG: 30 %, Ausschluss über 10.000 € | 17.10.2026 | § 37b Abs. 1 EStG | belege/bewirtung-geschenke |
| GWG 800 € netto; Verzeichnis über 250 €; Sammelposten über 250 € bis 1.000 €, fünf Jahre | 17.10.2026 | § 6 Abs. 2, 2a EStG | belege/ablauf, anlagen-gwg |
| Degressive AfA: höchstens dreifacher linearer Satz, höchstens 30 %, Anschaffung 01.07.2025 bis 31.12.2027 | 17.10.2026 | § 7 Abs. 2 EStG | belege/anlagen-gwg |
| Nutzungsdauer Computerhardware/Software: ein Jahr möglich | BMF vom 22.02.2022 | BMF-Schreiben | belege/anlagen-gwg |
| Steuersätze 19 % / 7 % | 17.10.2026 | § 12 Abs. 1, 2 UStG | belege/steuerfaelle |
| Leistungsnachweis ab 1.000 € brutto (Buchfink-Vorgabe, einstellbar) | Code 17.10.2026 | `internal/service/service_proof.go` | belege/pflichtangaben |

### Buchfink-eigene Quellen

| Quelle | Wofür |
|---|---|
| `internal/accounting/posting_groups.go` | Gruppen `bewirtung` (6640/6644), `geschenke` (6610/6620), `geschenke_betrieblich` (6625), `repraesentation` (6645), `gwg` (6260), `geldverkehr`, `versicherungen`, `miete` |
| `internal/accounting/tax_params.go` | datierte Werte: Kleinbetrag 150/250 €, Geschenke 35/50 €, Bewirtung 70 %, § 44 UStDV, Steuersätze 2020 |
| `internal/accounting/afa_rules.json`, `asset_accounts.go` | GWG/Sammelposten-Grenzen, degressive AfA, Kontenkatalog mit Nutzungsdauervorschlägen |
| `internal/accounting/vat_period.go` | Zuordnung zum Voranmeldungszeitraum: Vorsteuer nach spätestem Datum (Leistung, Beleg, Eingang); § 13b Abs. 1 nach Leistungszeitraum, Abs. 2 und Erwerb nach Rechnungsdatum, spätestens Folgemonat; Storno übernimmt die Daten der Ursprungsbuchung |
| `internal/accounting/tax_keys.go`, `tax_skr04.go` | Steuerschlüssel je Steuerfall; `reverse_charge_eu` → Kz 46/47, `reverse_charge` → Kz 84/85 |
| `internal/service/posting_service.go` (`reverseChargeParagraphWarning`) | Warnung `reverse_charge_eu_supplier`; inländische Bauleistung mit `reverse_charge` zulässig |
| `internal/service/posting_currency.go` (`vatRateDateOf`), `currency_service.go` | EZB-Tageskurs des Belegdatums, BMF-Kurs des Monats des Leistungsendes oder der früheren Zahlung, Kursdifferenz 6880/4840 |
| `internal/service/posting_service.go` | USt-IdNr. am Kontakt für EU-Fälle, Anzahlung erst mit Zahlung, Rückstellungsverbrauch |
| `internal/service/receipt_findings.go`, `service_proof.go` | Befundklassen, Leistungsnachweis |
| `internal/accounting/tax_skr04.go`, `input_tax_correction.go` | geschützte Steuerkonten, Konten 1396–1399 |

### Offen

- BMF-Schreiben vom 15.10.2025 und vom 22.02.2022 nur über Kanzleispiegel
  (steuerschroeder.de) gelesen; mit dem BMF-Original abgleichen.
- BMF-Schreiben Bewirtung vom 19.11.2025: Wortlaut nicht abrufbar. Nach Deloitte
  Tax-News (https://www.deloitte-tax-news.de/steuern/unternehmensteuer/bmf-steuerliche-anerkennung-von-bewirtungsaufwendungen.html)
  und PwC-Blog (https://blogs.pwc.de/de/steuern-und-recht/article/251895/): ersetzt
  das Schreiben vom 30.06.2021 (bleibt für Bewirtungen bis 31.12.2024);
  Rechnung maschinell erstellt und elektronisch aufgezeichnet, bei Kassensystem
  mit TSE; über 250 € Name des Bewirtenden; „Speisen und Getränke“ als
  Leistungsbeschreibung genügt nicht; digitaler Eigenbeleg zulässig, mit der
  Rechnung verknüpft; Trinkgeld nachzuweisen. Nach Prüfung in
  `bewirtung-geschenke.md` übernehmen.
- AfA-Tabelle AV (BMF vom 15.12.2000) nicht gelesen; die Vorschläge in
  `asset_accounts.go` (Pkw 6, Lkw 9, Ladeneinbauten 8, Büromöbel 13 Jahre) am
  Original prüfen.
- Gruppe `geschenke_betrieblich`: Grundlage (Verwaltungsregel, vermutlich R 4.10
  EStR) nicht am Wortlaut geprüft.
- Buchfink: BMF-Kurse lassen sich weder in der App noch über ein Werkzeug
  erfassen (`SaveVatRate`, `ImportVatRatesCSV` ohne Aufrufer); die
  Umsatzsteuer folgt deshalb dem EZB-Tageskurs. Eine Zahlung vor der Leistung
  erkennt Buchfink nur, wenn der Beleg mit ihr gebucht wird.
- Werkzeug: Wie eine Gutschrift eines Lieferanten (negative Beträge) mit
  `beleg_buchen_mit_rueckfrage` gebucht wird, ist nicht beschrieben.

## Bereich bank

Quellen des Bereichs `references/bank/` (ablauf.md und Referenzen). Alle Links am
17.10.2026 mit HTTP 200 geprüft.

### Gesetze

| Norm | Link | Wofür | Geprüft | Genutzt in |
|---|---|---|---|---|
| § 3 AO | https://www.gesetze-im-internet.de/ao_1977/__3.html | steuerliche Nebenleistungen (Abs. 4) | Text 17.10.2026 | bank/steuerzahlungen |
| § 146 AO | https://www.gesetze-im-internet.de/ao_1977/__146.html | jeder Vorgang einmal und vollständig (Abs. 1) | Text 17.10.2026 (Eintrag grundlagen) | grundlagen/aufbewahrung, gobd, pflichten |
| § 152 AO | https://www.gesetze-im-internet.de/ao_1977/__152.html | Verspätungszuschlag (nur Verweis) | Titel 17.10.2026 | grundlagen/pflichten; bank/steuerzahlungen |
| § 224 AO | https://www.gesetze-im-internet.de/ao_1977/__224.html | Tag der Zahlung: Überweisung bei Gutschrift, Lastschrift am Fälligkeitstag (Abs. 2 Nr. 2, 3) | Text 17.10.2026 | grundlagen/pflichten |
| § 233a AO | https://www.gesetze-im-internet.de/ao_1977/__233a.html | Verzinsung, Zinslauf 15 Monate (Abs. 1, 2 Satz 1) | Text 17.10.2026 | bank/steuerzahlungen |
| § 238 AO | https://www.gesetze-im-internet.de/ao_1977/__238.html | 0,15 % je Monat bei § 233a (Abs. 1a), volle Monate (Abs. 1 Satz 2) | Text 17.10.2026 | steuererklaerungen/fristen-zuschlaege |
| § 240 AO | https://www.gesetze-im-internet.de/ao_1977/__240.html | Säumniszuschlag 1 %, Abrundung 50 € (Abs. 1 Satz 1), Schonfrist drei Tage, nicht bei Bar/Scheck (Abs. 3) | Text 17.10.2026 | bank/steuerzahlungen |
| § 4 UStG | https://www.gesetze-im-internet.de/ustg_1980/__4.html | Kreditzinsen (Nr. 8 Buchst. a), Zahlungsverkehr (Nr. 8 Buchst. d) steuerfrei | Text 17.10.2026 | bank/darlehen, kreditkarte-geldtransit, skonto-differenzen |
| § 13 UStG | https://www.gesetze-im-internet.de/ustg_1980/__13.html | Steuer bei Anzahlung mit Vereinnahmung (Abs. 1 Nr. 1 Buchst. a Satz 4) | Text 17.10.2026 | belege/steuerfaelle; rechnungen/anzahlungen; monatsabschluss/ustva |
| § 15 UStG | https://www.gesetze-im-internet.de/ustg_1980/__15.html | Vorsteuer aus Anzahlung erst mit Zahlung (Abs. 1 Satz 1 Nr. 1 Satz 3) | Text 17.10.2026 | bank/kreditkarte-geldtransit |
| § 17 UStG | https://www.gesetze-im-internet.de/ustg_1980/__17.html | Skonto: Berichtigung Steuer und Vorsteuer (Abs. 1 Satz 1, 2), § 13b und ig. Erwerb (Satz 5), Zeitraum der Änderung (Satz 8) | Text 17.10.2026 | bank/skonto-differenzen |
| § 18 UStG | https://www.gesetze-im-internet.de/ustg_1980/__18.html | Vorauszahlung fällig am zehnten Tag (Abs. 1 Satz 4) | Text 17.10.2026 | grundlagen/pflichten; monatsabschluss/berichtigung, ustva; steuererklaerungen/ablauf, ust-jahr; gruendung/steuerliche-erfassung |
| § 46 UStDV | https://www.gesetze-im-internet.de/ustdv_1980/__46.html | Dauerfristverlängerung um einen Monat | Text 17.10.2026 | grundlagen/pflichten; monatsabschluss/dauerfristverlaengerung |
| § 4 EStG | https://www.gesetze-im-internet.de/estg/__4.html | Gewerbesteuer und Nebenleistungen keine Betriebsausgaben (Abs. 5b) | Text 17.10.2026 | bank/steuerzahlungen |
| § 4h EStG | https://www.gesetze-im-internet.de/estg/__4h.html | Zinsschranke greift nicht unter 3 Mio. € Nettozinsaufwand (Abs. 2 Satz 1 Buchst. a) | Text 17.10.2026 | bank/darlehen |
| § 20 EStG | https://www.gesetze-im-internet.de/estg/__20.html | Subsidiarität (Abs. 8 Satz 1), Sparer-Pauschbetrag (Abs. 9) | Text 17.10.2026 | bank/zinsen-kapest; beteiligungen/dividenden, steuerfreistellung |
| § 36 EStG | https://www.gesetze-im-internet.de/estg/__36.html | Anrechnung Abzugsteuer (Abs. 2 Nr. 2), nur mit Bescheinigung (Satz 2) | Text 17.10.2026 | steuererklaerungen/kst; beteiligungen/dividenden |
| § 43 EStG | https://www.gesetze-im-internet.de/estg/__43.html | Kapitalertragsteuer ist Einkommensteuer durch Abzug (Abs. 1 Satz 1); Zinsen inländischer Kreditinstitute (Nr. 7 Buchst. b) | Text 17.10.2026 | grundlagen/pflichten; bank/darlehen, zinsen-kapest; beteiligungen/bewertung-veraeusserung, dividenden, investmentfonds |
| § 43a EStG | https://www.gesetze-im-internet.de/estg/__43a.html | 25 % (Abs. 1 Satz 1 Nr. 1) | Text 17.10.2026 | bank/zinsen-kapest; beteiligungen/investmentfonds |
| § 44a EStG | https://www.gesetze-im-internet.de/estg/__44a.html | Freistellungsauftrag setzt Sparer-Pauschbetrag voraus (Abs. 1, 2) | Text 17.10.2026 | bank/zinsen-kapest; beteiligungen/dividenden |
| § 45a EStG | https://www.gesetze-im-internet.de/estg/__45a.html | Steuerbescheinigung auf Verlangen (Abs. 2) | Text 17.10.2026 | grundlagen/pflichten; bank/zinsen-kapest; beteiligungen/dividenden |
| § 8 KStG | https://www.gesetze-im-internet.de/kstg_1977/__8.html | alle Einkünfte gewerblich (Abs. 2) | Text 17.10.2026 | bank/darlehen, zinsen-kapest |
| § 8a KStG | https://www.gesetze-im-internet.de/kstg_1977/__8a.html | Zinsschranke bei Körperschaften | Titel 17.10.2026 | bank/darlehen |
| § 10 KStG | https://www.gesetze-im-internet.de/kstg_1977/__10.html | Steuern vom Einkommen und Nebenleistungen nicht abziehbar (Nr. 2) | Text 17.10.2026 | bank/steuerzahlungen, zinsen-kapest |
| § 31 KStG | https://www.gesetze-im-internet.de/kstg_1977/__31.html | EStG-Vorschriften zur Anrechnung gelten (Abs. 1 Satz 1) | Text 17.10.2026 | grundlagen/pflichten; belege/bewirtung-geschenke; steuererklaerungen/ablauf, bescheide, kst |
| § 1 SolZG | https://www.gesetze-im-internet.de/solzg_1995/__1.html | EStG/KStG entsprechend, auch Anrechnung (Abs. 2) | Text 17.10.2026 | bank/zinsen-kapest; steuererklaerungen/kst; beteiligungen/dividenden |
| § 3 SolZG | https://www.gesetze-im-internet.de/solzg_1995/__3.html | Bemessungsgrundlage Kapitalertragsteuer (Abs. 1 Nr. 5) | Text 17.10.2026 | bank/zinsen-kapest |
| § 4 SolZG | https://www.gesetze-im-internet.de/solzg_1995/__4.html | 5,5 % (Satz 1), Cent-Bruchteile außer Ansatz (Satz 3) | Text 17.10.2026 | bank/zinsen-kapest |
| § 8 GewStG | https://www.gesetze-im-internet.de/gewstg/__8.html | Hinzurechnung ein Viertel, Freibetrag 200.000 € (Nr. 1), ungewöhnliche Skonti als Entgelt für Schulden (Nr. 1 Buchst. a Satz 2) | Text 17.10.2026 | bank/darlehen, skonto-differenzen, zinsen-kapest; beteiligungen/gewst-holding |
| § 250 HGB | https://www.gesetze-im-internet.de/hgb/__250.html | Disagio als aktiver RAP, Wahlrecht (Abs. 3) | Text 17.10.2026 | bank/darlehen |
| § 252 HGB | https://www.gesetze-im-internet.de/hgb/__252.html | Periodenabgrenzung unabhängig von Zahlung (Abs. 1 Nr. 5) | Text 17.10.2026 | jahresabschluss/ablauf, abgrenzung, forderungen; beteiligungen/dividenden |
| § 253 HGB | https://www.gesetze-im-internet.de/hgb/__253.html | Verbindlichkeiten zum Erfüllungsbetrag (Abs. 1 Satz 2) | Text 17.10.2026 | bank/darlehen |
| § 255 HGB | https://www.gesetze-im-internet.de/hgb/__255.html | Skonto mindert Anschaffungskosten (Abs. 1 Satz 3) | Text 17.10.2026 | bank/skonto-differenzen |
| § 256a HGB | https://www.gesetze-im-internet.de/hgb/__256a.html | Stichtagsbewertung (Verweis) | Text 17.10.2026 | jahresabschluss/forderungen |
| § 268 HGB | https://www.gesetze-im-internet.de/hgb/__268.html | Vermerk Restlaufzeit bis/über ein Jahr (Abs. 5 Satz 1) | Text 17.10.2026 | rechnungen/anzahlungen; jahresabschluss/abgrenzung, ergebnisverwendung-ug, forderungen, groessenklassen-anhang, unfertige-leistungen |

### Verwaltungsanweisungen

| Schreiben | Link | Wofür | Geprüft | Genutzt in |
|---|---|---|---|---|
| GoBD, BMF vom 28.11.2019 (bestehender Eintrag oben) | https://www.lfst.bayern.de/fileadmin/RESSOURCEN/INFORMATIONEN/Steuerinfos/Weitere_Themen/Aussenpruefung/2019-11-28-GoBD-1.pdf | Rz. 36, 41 (vollständig, einmal), 44 (richtig), 47 (zehn Tage), 61 (Belegprinzip), 76 (Kontoauszug als strukturierte Datei), 81 (Dauersachverhalt) | Text 17.10.2026 | bank/ablauf, darlehen, skonto-differenzen, zuordnung |
| BMF vom 15.10.2025 (Eintrag unter Bereich belege) | siehe oben | Rn. 51a: keine Rechnungsberichtigung bei Skonto | Text 17.10.2026 (Kanzleispiegel) | bank/skonto-differenzen |

### Werte mit Stand

| Wert | Stand | Quelle | Genutzt in |
|---|---|---|---|
| Kapitalertragsteuer 25 % | 17.10.2026 | § 43a Abs. 1 Satz 1 Nr. 1 EStG | bank/ablauf, zinsen-kapest |
| Solidaritätszuschlag 5,5 % auf die Kapitalertragsteuer | 17.10.2026 | § 3 Abs. 1 Nr. 5, § 4 Satz 1 SolZG | bank/ablauf, zinsen-kapest |
| Säumniszuschlag 1 % je angefangenem Monat, Rückstand auf 50 € abgerundet; drei Tage Schonfrist außer Bar/Scheck | 17.10.2026 | § 240 Abs. 1, 3 AO | bank/ablauf, steuerzahlungen |
| Zinsen nach § 233a AO: 0,15 % je Monat (seit 01.01.2019); Zinslauf ab 15 Monaten nach Ablauf des Entstehungsjahres | 17.10.2026 | § 238 Abs. 1a, § 233a Abs. 2 AO | bank/ablauf, steuerzahlungen |
| Umsatzsteuer-Vorauszahlung fällig am zehnten Tag; Dauerfristverlängerung ein Monat | 17.10.2026 | § 18 Abs. 1 Satz 4 UStG, § 46 UStDV | bank/steuerzahlungen |
| Gewerbesteuer-Hinzurechnung: ein Viertel, soweit Summe 200.000 € übersteigt | 17.10.2026 | § 8 Nr. 1 GewStG | bank/darlehen |
| Zinsschranke: nicht anzuwenden unter 3 Mio. € Nettozinsaufwand | 17.10.2026 | § 4h Abs. 2 Satz 1 Buchst. a EStG | bank/darlehen |
| Kontoauszug mindestens alle zehn Tage importieren | GoBD i. d. F. 14.07.2025 | GoBD Rz. 47 | bank/ablauf |

### Buchfink-eigene Quellen

| Quelle | Wofür |
|---|---|
| `internal/service/settlement.go` (`Settle`), `internal/service/settlement_difference.go` (`cashAmount`, `differenceLines`, `skontoLines`) | Differenzarten, Skonto nach Steuerfall, Anlage-Skonto, Rundung/Kurs auf 6300/4830, Gebühr auf 6855 bei Zahlungsausgang und -eingang |
| `internal/domain/payment.go` (`SkontoAccount`) | Skontokonten 5736/5731/5730, 4736/4731/4734 |
| `internal/service/bank_service.go`, `bank_accounts.go`, `internal/bank/camt.go` | Import, Auszug als Beleg, Ablehnungen (Sammelbuchung, Fremdwährung, unbekannte IBAN, mehrere Konten, nur gebuchte Umsätze), gelernte Regeln |
| `internal/service/closing_booking_service.go` | Jahresverrechnung USt: Zahllast 3841, Erstattung 1420, Vorauszahlungen 3820/3830 |
| `internal/accounting/skr04_2026.json` | alle zitierten Konten (geprüft am 17.10.2026) |
| `docs/entwicklung/mcp.md` | Routine, `aufteilung`, `kreditkartenabrechnung`, `ignorieren`, Kreditkarte 3610 |

### Offen

- Buchfink: realisierte Kursdifferenz bei Zahlung auf 6300/4830; das SKR04 hat
  6880/4840 (Währungsumrechnung), die der Belegweg nutzt. Vereinheitlichen?
- Werkzeug: Ob eine Zeile auf das Personenkonto des Kunden in
  `bankumsatz_buchen` nach Rücklastschrift oder Überzahlung einen offenen
  Posten bildet, ist nicht beschrieben; der Skill lässt es mit
  `offene_posten` prüfen.
- Disagio: steuerliche Aktivierungspflicht (nach Sekundärquellen H 6.10 EStH,
  BFH I R 46/05) nicht am Wortlaut geprüft; Skill nennt nur Wertungsfrage.
- Zinserträge: Steuerfall nicht steuerbar oder steuerfrei (§ 4 Nr. 8 Buchst. a
  UStG) für die Voranmeldung nicht geklärt.
- Erstattungszinsen nach § 233a AO: Wahl zwischen 7105, 7106, 7107 nicht
  geklärt (steuererklaerungen/kst).

## Bereich rechnungen

Quellen des Bereichs `references/rechnungen/` (ablauf.md und Referenzen). Alle Links
am 17.10.2026 mit HTTP 200 geprüft. „Text“ = Wortlaut gelesen und Aussage
abgeglichen.

### Gesetze

| Norm | Link | Wofür | Geprüft | Genutzt in |
|---|---|---|---|---|
| § 3a UStG | https://www.gesetze-im-internet.de/ustg_1980/__3a.html | Ort: Unternehmenssitz (Abs. 1), Empfänger B2B und jur. Person mit USt-IdNr. (Abs. 2 Satz 1, 3), Grundstück (Abs. 3 Nr. 1), Kultur/Unterricht B2C (Nr. 3), Vermittlung (Nr. 4), Veranstaltung B2B (Nr. 5), Katalogleistungen Drittland B2C (Abs. 4 Satz 2), elektronische Leistungen B2C, 10.000 €, Verzicht, zwei Jahre (Abs. 5 Satz 1–5) | Text 17.10.2026 | rechnungen/ausland |
| § 4 UStG | https://www.gesetze-im-internet.de/ustg_1980/__4.html | Kreditgewährung steuerfrei (Nr. 8 Buchst. a) | Text 17.10.2026 | bank/darlehen, kreditkarte-geldtransit, skonto-differenzen |
| § 10 UStG | https://www.gesetze-im-internet.de/ustg_1980/__10.html | Entgelt abzüglich gesetzlich geschuldeter Steuer (Abs. 1 Satz 2) | Text 17.10.2026 | rechnungen/storno-berichtigung |
| § 13 UStG | https://www.gesetze-im-internet.de/ustg_1980/__13.html | Sollversteuerung, Teilleistung, Mindest-Istversteuerung (Abs. 1 Nr. 1 Buchst. a Satz 1–4) | Text 17.10.2026 | rechnungen/anzahlungen |
| § 14 UStG | https://www.gesetze-im-internet.de/ustg_1980/__14.html | Rechnungsbegriff, E-Rechnung, sonstige Rechnung, Zustimmung, Format (Abs. 1 Satz 1, 3–6); Rechnungspflicht sechs Monate, E-Rechnungspflicht, Ansässigkeit, Gutschrift (Abs. 2 Satz 2 Nr. 1–3, Satz 3, 5); Pflichtangaben (Abs. 4 Satz 1 Nr. 1–10); Anzahlung und Endrechnung (Abs. 5 Satz 1, 2) | Text 17.10.2026 | rechnungen/anzahlungen, e-rechnung-ausgang, pflichtangaben-ausgang, storno-berichtigung |
| § 14a UStG | https://www.gesetze-im-internet.de/ustg_1980/__14a.html | „Steuerschuldnerschaft des Leistungsempfängers“, Rechnung bis 15. des Folgemonats, beide USt-IdNrn. (Abs. 1 Satz 1–3); § 13b-Leistung im Inland (Abs. 5) | Text 17.10.2026 | rechnungen/ausland, pflichtangaben-ausgang |
| § 14b UStG | https://www.gesetze-im-internet.de/ustg_1980/__14b.html | Aufbewahrungshinweis bei Grundstücksleistung an Nichtunternehmer (Abs. 1 Satz 5) | Text 17.10.2026 | grundlagen/aufbewahrung; rechnungen/pflichtangaben-ausgang |
| § 14c UStG | https://www.gesetze-im-internet.de/ustg_1980/__14c.html | unrichtiger Steuerausweis und Berichtigung (Abs. 1 Satz 1, 2); unberechtigter Ausweis, Berichtigung nur mit Zustimmung (Abs. 2 Satz 1, 3–5) | Text 17.10.2026 | rechnungen/storno-berichtigung |
| § 15 UStG | https://www.gesetze-im-internet.de/ustg_1980/__15.html | Vorsteuer nur für Leistungen an das Unternehmen, Rechnung erforderlich (Abs. 1 Satz 1 Nr. 1 Satz 1, 2) | Text 17.10.2026 | rechnungen/pflichtangaben-ausgang; gesellschafter/vga-einlage |
| § 17 UStG | https://www.gesetze-im-internet.de/ustg_1980/__17.html | Minderung der Bemessungsgrundlage, Zeitraum der Änderung (Abs. 1 Satz 1, 8); Uneinbringlichkeit und spätere Zahlung (Abs. 2 Nr. 1 Satz 1, 2); Anzahlung ohne Leistung (Nr. 2); Rückgängigmachung (Nr. 3) | Text 17.10.2026 | rechnungen/anzahlungen, forderungsausfall, pflichtangaben-ausgang, storno-berichtigung |
| § 18a UStG | https://www.gesetze-im-internet.de/ustg_1980/__18a.html | Zusammenfassende Meldung (Verweis auf monatsabschluss) | Titel 17.10.2026 | grundlagen/pflichten; monatsabschluss/zm |
| § 18e UStG | https://www.gesetze-im-internet.de/ustg_1980/__18e.html | qualifizierte Bestätigung der USt-IdNr. (Nr. 1) | Text 17.10.2026 | rechnungen/ausland |
| § 18j UStG | https://www.gesetze-im-internet.de/ustg_1980/__18j.html | OSS-Verfahren (Grenze) | Titel 17.10.2026 | grundlagen/grenzen; rechnungen/ausland |
| § 27 UStG | https://www.gesetze-im-internet.de/ustg_1980/__27.html | Übergang E-Rechnung: Umsätze 2025–2026 bei Übermittlung bis 31.12.2026 (Nr. 1), Umsätze 2027 bei Vorjahresumsatz ≤ 800.000 € oder per EDI, jeweils bis 31.12.2027 (Nr. 2, 3); Umsatz 2026 mit Rechnung ab 2027 fällt unter keine Nummer | Text 17.10.2026 | rechnungen/e-rechnung-ausgang |
| § 31 UStDV | https://www.gesetze-im-internet.de/ustdv_1980/__31.html | mehrere Dokumente (Abs. 1), Name/Anschrift (Abs. 2), Abkürzungen (Abs. 3), Kalendermonat (Abs. 4), Berichtigung (Abs. 5) | Text 17.10.2026 | rechnungen/pflichtangaben-ausgang, storno-berichtigung |
| § 33 UStDV | https://www.gesetze-im-internet.de/ustdv_1980/__33.html | Kleinbetragsrechnung 250 € (Satz 1), nicht bei §§ 3c, 6a, 13b (Satz 3), immer sonstige Rechnung (Satz 4) | Text 17.10.2026 | rechnungen/e-rechnung-ausgang, pflichtangaben-ausgang |
| § 238 HGB | https://www.gesetze-im-internet.de/hgb/__238.html | Belegprinzip auch bei Rechnung an Verbraucher (Abs. 1) | Text 17.10.2026 | rechnungen/pflichtangaben-ausgang |
| § 266 HGB | https://www.gesetze-im-internet.de/hgb/__266.html | erhaltene Anzahlungen unter Verbindlichkeiten (Abs. 3 C. Nr. 3) | Text 17.10.2026 | rechnungen/anzahlungen; beteiligungen/erwerb-ausweis |
| § 268 HGB | https://www.gesetze-im-internet.de/hgb/__268.html | gesonderter Ausweis erhaltener Anzahlungen (Abs. 5 Satz 2); Restlaufzeitvermerke Forderungen (Abs. 4 Satz 1) und Verbindlichkeiten (Abs. 5 Satz 1) | Text 17.10.2026 | rechnungen/anzahlungen |
| § 153 AO | https://www.gesetze-im-internet.de/ao_1977/__153.html | Berichtigung einer übermittelten Voranmeldung (Verweis) | Text (Eintrag grundlagen) | rechnungen/storno-berichtigung |
| § 187 BGB | https://www.gesetze-im-internet.de/bgb/__187.html | Tag des Ereignisses zählt nicht mit (Abs. 1, entsprechend für Zinsbeginn) | Text 17.10.2026 | rechnungen/mahnen-verzug |
| § 195 BGB | https://www.gesetze-im-internet.de/bgb/__195.html | Verjährung drei Jahre | Text 17.10.2026 | rechnungen/mahnen-verzug |
| § 199 BGB | https://www.gesetze-im-internet.de/bgb/__199.html | Beginn mit Schluss des Jahres (Abs. 1) | Text 17.10.2026 | rechnungen/mahnen-verzug |
| § 247 BGB | https://www.gesetze-im-internet.de/bgb/__247.html | Basiszinssatz, Änderung 1.1./1.7., Bekanntgabe durch Bundesbank (Abs. 1, 2); Zahl im Gesetzestext veraltet | Text 17.10.2026 | rechnungen/mahnen-verzug |
| § 271a BGB | https://www.gesetze-im-internet.de/bgb/__271a.html | Zahlungsfrist über 60 Tage (Abs. 1), nicht für Abschläge und Verbraucher (Abs. 5) | Text 17.10.2026 | rechnungen/pflichtangaben-ausgang |
| § 286 BGB | https://www.gesetze-im-internet.de/bgb/__286.html | Verzug durch Mahnung (Abs. 1), ohne Mahnung (Abs. 2 Nr. 1–3), 30 Tage und Verbraucherhinweis (Abs. 3 Satz 1, 2), Vertretenmüssen (Abs. 4) | Text 17.10.2026 | rechnungen/mahnen-verzug |
| § 288 BGB | https://www.gesetze-im-internet.de/bgb/__288.html | Verzugszins +5/+9 Prozentpunkte (Abs. 1, 2), höhere Zinsen (Abs. 3), weiterer Schaden (Abs. 4), Pauschale 40 € auch bei Abschlägen, Anrechnung (Abs. 5), Ausschluss unwirksam (Abs. 6) | Text 17.10.2026 | rechnungen/mahnen-verzug |

### Verwaltungsanweisungen und weitere Quellen

| Quelle | Link | Wofür | Geprüft | Genutzt in |
|---|---|---|---|---|
| BMF vom 15.10.2024, III C 2 - S 7287-a/23/10001 :007, BStBl I S. 1320 (E-Rechnung) | https://www.lfst.bayern.de/fileadmin/RESSOURCEN/INFORMATIONEN/Steuerinfos/Weitere_Themen/E-Rechnung/2024-10-15-einfuehrung-e-rechnung.pdf | Rn. 48 (Endrechnung als E-Rechnung, Restrechnung, Anhang bis 31.12.2027 nicht beanstandet), Rn. 62 (kein Übergang beim Empfang) | Text 17.10.2026 (Spiegel LfSt Bayern) | rechnungen/anzahlungen, e-rechnung-ausgang |
| BMF vom 15.10.2025, III C 2 - S 7287-a/00019/007/243 (E-Rechnung, Änderung UStAE) | https://www.steuerschroeder.de/blog/wp-content/uploads/2025/10/2025-10-15-einfuehrung-obligatorische-e-rechnung.pdf (Kanzleispiegel) | Rn. 6a, 17 (Gutschrift, § 13b zwischen Inländern), 35, 60; UStAE n. F.: 14.1 Abs. 2, Abs. 3 Satz 4, Abs. 4 Satz 1, 4, 6–8, Abs. 5 Satz 4–5, Abs. 6 Satz 1–5, Abs. 7 Satz 1, 4–8, Abs. 14 (ZUGFeRD ab 2.0.1 ohne MINIMUM/BASIC-WL); 14.4 Abs. 3 (strukturierter Teil führt); 14.11 Abs. 1 Satz 7–9, Abs. 4; 14c.1 Abs. 4 Satz 7 | Text 17.10.2026 (Kanzleispiegel) | rechnungen/ablauf, anzahlungen, ausland, e-rechnung-ausgang, pflichtangaben-ausgang, storno-berichtigung |
| UStAE Abschnitt 1.3 Abs. 6 | https://www.steuerschroeder.de/steuergesetze/ustae/1.3 | Verzugszinsen, Mahngebühren, Mahnverfahrenskosten sind Schadensersatz | Text 17.10.2026 (Kanzleispiegel) | rechnungen/ablauf, mahnen-verzug |
| UStAE Abschnitt 3a.2 Abs. 9, 11, 11a | https://www.steuerschroeder.de/steuergesetze/ustae/3a.2 | USt-IdNr. als Nachweis mit qualifizierter Bestätigung (Abs. 9 Satz 4, 5); Drittland-Nachweis (Abs. 11); private Nutzung wahrscheinlich (Abs. 11a) | Text 17.10.2026 (Kanzleispiegel) | rechnungen/ausland |
| UStAE Abschnitt 13.1 Abs. 3 | https://www.steuerschroeder.de/steuergesetze/ustae/13.1 | sonstige Leistung mit Vollendung ausgeführt, Dauerleistung | Text 17.10.2026 (Kanzleispiegel) | rechnungen/ablauf, pflichtangaben-ausgang |
| UStAE Abschnitt 13.4 | https://www.steuerschroeder.de/steuergesetze/ustae/13.4 | Teilleistungen | Text 17.10.2026 (Kanzleispiegel) | rechnungen/anzahlungen |
| UStAE Abschnitt 13.5 Abs. 1, 3, 4 | https://www.steuerschroeder.de/steuergesetze/ustae/13.5 | Anzahlung: Entstehung, bestimmte Leistung, voraussichtlich steuerfrei/nicht steuerbar | Text 17.10.2026 (Kanzleispiegel) | rechnungen/anzahlungen |
| UStAE Abschnitt 14.8 Abs. 1, 2, 4, 5, 7, 9, 10, 11 | https://www.steuerschroeder.de/steuergesetze/ustae/14.8 | Anzahlungsrechnung, Endrechnung, § 14c bei fehlender Absetzung, Restrechnung | Text 17.10.2026 (Kanzleispiegel; Abs. 3 zeigt noch alte Satzzählung, also Stand vor 15.10.2025) | rechnungen/ablauf, anzahlungen, pflichtangaben-ausgang |
| UStAE Abschnitt 17.1 Abs. 5, 7 | https://www.steuerschroeder.de/steuergesetze/ustae/17.1 | Uneinbringlichkeit (Abs. 5), Rückzahlung einer Anzahlung im Zeitraum der Rückgewähr (Abs. 7) | Text 17.10.2026 (Kanzleispiegel) | rechnungen/ablauf, anzahlungen, forderungsausfall |
| Deutsche Bundesbank, Basiszinssatz nach § 247 BGB | https://www.bundesbank.de/de/bundesbank/organisation/agb-und-regelungen/basiszinssatz-607820 | Tabelle der Basiszinssätze | Text 17.10.2026 | rechnungen/mahnen-verzug |

### Werte mit Stand

| Wert | Stand | Quelle | Genutzt in |
|---|---|---|---|
| Basiszinssatz 1,27 % ab 01.01.2026, 1,52 % ab 01.07.2026 (2025: 2,27 % / 1,27 %) | Bundesbank, abgerufen 17.10.2026; ändert sich halbjährlich | § 247 BGB, Bundesbank | rechnungen/mahnen-verzug |
| Verzugszins Basiszinssatz + 9 Prozentpunkte (ohne Verbraucher), + 5 Prozentpunkte (mit Verbraucher); ab 01.07.2026 also 10,52 % bzw. 6,52 % | 17.10.2026 | § 288 Abs. 1, 2 BGB | rechnungen/mahnen-verzug |
| Verzugspauschale 40 € | 17.10.2026 | § 288 Abs. 5 BGB | rechnungen/anzahlungen, mahnen-verzug |
| Verzug 30 Tage nach Fälligkeit und Zugang | 17.10.2026 | § 286 Abs. 3 BGB | rechnungen/ablauf, mahnen-verzug |
| Zahlungsfrist über 60 Tage nur ausdrücklich und nicht grob unbillig | 17.10.2026 | § 271a Abs. 1 BGB | rechnungen/ablauf, pflichtangaben-ausgang |
| Kleinbetragsrechnung bis 250 € | 17.10.2026 | § 33 Satz 1 UStDV | rechnungen/ablauf, pflichtangaben-ausgang |
| E-Rechnung: sonstige Rechnung zulässig für Leistungen bis 31.12.2026; 2027 nur bei Vorjahres-Gesamtumsatz ≤ 800.000 €; ab 2028 E-Rechnung | 17.10.2026 | § 27 Abs. 38 UStG | rechnungen/ablauf, e-rechnung-ausgang |
| Schwelle elektronische Leistungen an EU-Verbraucher 10.000 € (Vorjahr und laufendes Jahr) | 17.10.2026 | § 3a Abs. 5 Satz 3 UStG | rechnungen/ausland |
| Rechnung binnen sechs Monaten; EU-Leistung bis 15. des Folgemonats | 17.10.2026 | § 14 Abs. 2 Satz 2, § 14a Abs. 1 Satz 2 UStG | rechnungen/ablauf |
| Regelverjährung drei Jahre ab Jahresende | 17.10.2026 | §§ 195, 199 Abs. 1 BGB | rechnungen/mahnen-verzug |

### Buchfink-eigene Quellen

| Quelle | Pfad | Wofür |
|---|---|---|
| Werkzeugkatalog | `docs/entwicklung/mcp.md` | Felder von `rechnung_vorschau`, `rechnung_ausstellen` (`art` rechnung, abschlag, schlussrechnung), `offene_posten` (`verzugszinsen`, `pauschale`); kein Werkzeug für Storno, Korrektur, Ausbuchung |
| Rechnungsarten | `internal/domain/invoice.go` | Abschlags-, Schluss-, Korrektur-, Stornorechnung (Typcodes 386/380/384), Zahlungsbedingungen, 60-Tage-Hinweis |
| Rechnungsverbund | `internal/domain/invoice_group.go`, `internal/service/advance_service.go` | Abschläge, Absetzung in der Schlussrechnung |
| Ausstellen | `internal/service/invoice_service.go` (`Issue`) | Fälligkeit 14 Tage ohne Angabe, Verbraucherhinweis, USt-IdNr.-Bestätigung, Kleinbetrag |
| Storno, Korrektur | `internal/service/invoice_correction.go` | Generalumkehr im laufenden Zeitraum; derzeit ohne Aufruf aus App oder Werkzeug |
| Ausbuchung | `internal/service/settlement_writeoff.go` | Begründungspflicht, Steuerkorrektur; derzeit ohne Aufruf aus App oder Werkzeug |
| Konten | `internal/accounting/advance_accounts.go`, `skr04_2026.json` | 3272/3260/3250, 6930/6931/6936, 4925, 4970, 7110 |
| Verzugsrechnung | `internal/accounting/default_interest.go`, `tax_params.go` | Zuschläge, 365 Tage, Basiszinstabelle |
| Voranmeldung, ZM | `internal/accounting/ustva.go`, `zm.go` | `reverse_charge_supply` → Kz 21 und ZM bei EU-Kunden, sonst Kz 45 |
| E-Rechnung | `internal/invoice/cii.go`, `zugferd.go` | Profil EN 16931 oder XRechnung CII, Kategorien AE/O |

### Offen

- Weder Werkzeug noch App für Stornorechnung, Rechnungskorrektur, Forderung
  ausbuchen, Versandvermerk und Rückzahlung einer Anzahlung im Verbund; die
  Skills sagen das dem Menschen.
- Skonto übergibt `rechnung_ausstellen` als `skonto_promille` und
  `skonto_tage`. Prüfen, ob der Hinweis in den strukturierten Teil (BT-20)
  gelangt.
- Code: `validateSmallAmount` (`invoice_service.go`) nennt „§ 33 Satz 2 UStDV“;
  nach dem Wortlaut ist es Satz 3.
- Code: Der Hinweistext zu `reverse_charge_supply` (`internal/domain/tax.go`)
  beschreibt § 13b Abs. 2 Nr. 1 UStG (im Ausland ansässiger Leistender); genutzt
  wird der Steuerfall für Leistungen an Unternehmen im übrigen EU-Gebiet
  (§ 3a Abs. 2, § 14a Abs. 1 UStG). Der Text in der E-Rechnung lautet
  „… nach § 13b UStG“. Fachlich prüfen.
- Endrechnung als E-Rechnung (BMF 15.10.2024, Rn. 48): ob die Angabe der
  Anzahlungen in BT-113 genügt.
- UStAE nur über Kanzleispiegel gelesen; Abgleich mit dem amtlichen UStAE
  (BMF) durch einen Prüfer mit Browser.

## Bereich gesellschafter

Quellen des Bereichs `references/gesellschafter/` (ablauf.md und Referenzen). Alle
Links am 17.10.2026 mit HTTP 200 geprüft.

### Gesetze

| Norm | Link | Wofür | Geprüft | Genutzt in |
|---|---|---|---|---|
| § 8c KStG | (Zeile unter 1. Gesetze) | Verlustuntergang bei Wechsel von mehr als 50 % der Anteile in fünf Jahren (Abs. 1 Satz 1) | Text 17.10.2026 | gesellschafter/ablauf |
| § 5a GmbHG | https://www.gesetze-im-internet.de/gmbhg/__5a.html | UG: gesetzliche Rücklage ein Viertel (Abs. 3) | Text 17.10.2026 | grundlagen/grenzen, pflichten; jahresabschluss/ergebnisverwendung-ug; gruendung/gruendungsweg, kapitalaufbringung |
| § 29 GmbHG | https://www.gesetze-im-internet.de/gmbhg/__29.html | Anspruch auf Jahresüberschuss (Abs. 1), Verteilung nach Anteilen (Abs. 3) | Text 17.10.2026 | grundlagen/pflichten; jahresabschluss/ergebnisverwendung-ug; beteiligungen/dividenden |
| § 30 GmbHG | https://www.gesetze-im-internet.de/gmbhg/__30.html | Kapitalerhaltung, vollwertiger Rückgewähranspruch, Rückgewähr Gesellschafterdarlehen (Abs. 1 Satz 1–3) | Text 17.10.2026 | bank/darlehen; gesellschafter/ablauf, ausschuettung |
| § 31 GmbHG | https://www.gesetze-im-internet.de/gmbhg/__31.html | Erstattung verbotener Auszahlungen | Text 17.10.2026 | gesellschafter/ablauf |
| § 35 GmbHG | https://www.gesetze-im-internet.de/gmbhg/__35.html | Ein-Personen-GmbH: § 181 BGB, Niederschrift (Abs. 3) | Text 17.10.2026 | gesellschafter/ablauf |
| § 42 GmbHG | https://www.gesetze-im-internet.de/gmbhg/__42.html | Ausweis gegenüber Gesellschaftern (Abs. 3) | Text 17.10.2026 | gesellschafter/ausweis |
| § 43a GmbHG | https://www.gesetze-im-internet.de/gmbhg/__43a.html | kein Kredit an Geschäftsführer aus gebundenem Vermögen | Text 17.10.2026 | bank/darlehen |
| § 46 GmbHG | https://www.gesetze-im-internet.de/gmbhg/__46.html | Feststellung und Ergebnisverwendung durch Gesellschafter (Nr. 1) | Text 17.10.2026 | grundlagen/pflichten; jahresabschluss/aufstellen-feststellen-offenlegen; gruendung/kapitalaufbringung; beteiligungen/dividenden |
| § 48 GmbHG | https://www.gesetze-im-internet.de/gmbhg/__48.html | Beschlussform, Textform, Niederschrift bei einem Gesellschafter (Abs. 1–3) | Text 17.10.2026 | grundlagen/pflichten; jahresabschluss/aufstellen-feststellen-offenlegen |
| § 181 BGB | https://www.gesetze-im-internet.de/bgb/__181.html | Insichgeschäft | Text 17.10.2026 | gesellschafter/ablauf |
| § 488 BGB | https://www.gesetze-im-internet.de/bgb/__488.html | Darlehensvertrag, Kündigungsfrist drei Monate (Abs. 3) | Text 17.10.2026 | bank/darlehen |
| § 246 HGB | https://www.gesetze-im-internet.de/hgb/__246.html | Verrechnungsverbot (Abs. 2 Satz 1) | Text 17.10.2026 | jahresabschluss/abgrenzung, abschlussbuchungen, forderungen |
| § 247 HGB | https://www.gesetze-im-internet.de/hgb/__247.html | Anlagevermögen: dauernd dem Geschäftsbetrieb dienen (Abs. 2) | Text 17.10.2026 | bank/darlehen; beteiligungen/erwerb-ausweis |
| § 268 HGB | (Zeile unter Bereich rechnungen) | Restlaufzeitvermerke (Abs. 4 Satz 1, Abs. 5 Satz 1) | Text 17.10.2026 | bank/darlehen; gesellschafter/ausweis |
| § 272 HGB | https://www.gesetze-im-internet.de/hgb/__272.html | Kapitalrücklage, andere Zuzahlungen (Abs. 2 Nr. 4) | Text 17.10.2026 | gruendung/eroeffnungsbilanz, kapitalaufbringung; gesellschafter/vga-einlage |
| § 8 KStG | https://www.gesetze-im-internet.de/kstg_1977/__8.html | vGA mindert Einkommen nicht (Abs. 3 Satz 2), verdeckte Einlage (Satz 3) | Text 17.10.2026 | bank/darlehen, zinsen-kapest; gesellschafter/vga-einlage |
| § 27 KStG | https://www.gesetze-im-internet.de/kstg_1977/__27.html | Einlagekonto, Einlagenrückgewähr, ausschüttbarer Gewinn (Abs. 1 Satz 1, 3, 5), Feststellung und Erklärung (Abs. 2 Satz 1, 4), Bescheinigung (Abs. 3) | Text 17.10.2026 | grundlagen/pflichten; gesellschafter/ausschuettung, steuerbescheinigung; beteiligungen/dividenden |
| § 5 EStG | https://www.gesetze-im-internet.de/estg/__5.html | Verbindlichkeit aus künftigen Gewinnen (Abs. 2a) | Text 17.10.2026 | bank/darlehen |
| § 6 EStG | https://www.gesetze-im-internet.de/estg/__6.html | verdeckte Einlage erhöht Anschaffungskosten der Beteiligung (Abs. 6 Satz 2) | Text 17.10.2026 | gesellschafter/vga-einlage |
| § 20 EStG | https://www.gesetze-im-internet.de/estg/__20.html | vGA als Kapitalertrag (Abs. 1 Nr. 1 Satz 2), Einlagekonto keine Einnahme (Satz 3), stille Beteiligung, partiarisches Darlehen (Nr. 4) | Text 17.10.2026 | bank/darlehen, zinsen-kapest; gesellschafter/ausschuettung, vga-einlage |
| § 32d EStG | https://www.gesetze-im-internet.de/estg/__32d.html | kein Sondertarif für Zinsen an ≥ 10 % Beteiligte (Abs. 2 Nr. 1 Buchst. b) | Text 17.10.2026 | bank/darlehen |
| § 43 EStG | https://www.gesetze-im-internet.de/estg/__43.html | KapESt auf Dividenden und vGA (Abs. 1 Satz 1 Nr. 1), partiarisch/still (Nr. 3), Zinsen nur bei Anleihen/Kreditinstituten (Nr. 7), Abzug trotz § 8b KStG (Satz 3) | Text 17.10.2026 | bank/darlehen, zinsen-kapest; gesellschafter/ausschuettung, vga-einlage |
| § 43a EStG | https://www.gesetze-im-internet.de/estg/__43a.html | 25 % (Abs. 1 Satz 1 Nr. 1), Minderung bei Kirchensteuer (Satz 2), voller Ertrag (Abs. 2 Satz 1) | Text 17.10.2026 | gesellschafter/ausschuettung |
| § 44 EStG | https://www.gesetze-im-internet.de/estg/__44.html | Einbehalt für Rechnung des Gläubigers (Abs. 1 Satz 3), Abführung bei Dividenden am Zufluss, sonst 10. des Folgemonats (Satz 5), Abrundung auf volle Euro (Satz 6), Zufluss am Auszahlungstag oder Tag nach Beschluss (Abs. 2 Satz 1, 2), Haftung (Abs. 5 Satz 1) | Text 17.10.2026 | grundlagen/pflichten; gesellschafter/ausschuettung, vga-einlage; beteiligungen/dividenden, investmentfonds |
| § 44a EStG | https://www.gesetze-im-internet.de/estg/__44a.html | Freistellungsauftrag nur für Genussrechte/Arbeitnehmeranteile (Abs. 1 Satz 1 Nr. 1, 2), NV-Bescheinigung (Satz 4, Abs. 2 Satz 1 Nr. 2) | Text 17.10.2026 | gesellschafter/ausschuettung |
| § 45a EStG | https://www.gesetze-im-internet.de/estg/__45a.html | Anmeldung elektronisch, auch ohne Abzug mit Grund (Abs. 1 Satz 1–3); Bescheinigung auf Verlangen, elektronisch, ohne Unterschrift (Abs. 2 Satz 1–3); Berichtigung (Abs. 5) | Text 17.10.2026 | gesellschafter/ausschuettung, steuerbescheinigung |
| § 51a EStG | https://www.gesetze-im-internet.de/estg/__51a.html | Kirchensteuerabzug, Anlass- und Regelabfrage 1.9.–31.10., Stichtag 31.8. (Abs. 2c Satz 1 Nr. 3) | Text 17.10.2026 | grundlagen/pflichten; gesellschafter/ausschuettung |
| § 3 SolZG | https://www.gesetze-im-internet.de/solzg_1995/__3.html | Bemessung nach KapESt (Abs. 1 Nr. 5) | Text 17.10.2026 | grundlagen/pflichten; bank/zinsen-kapest; steuererklaerungen/kst |
| § 4 SolZG | https://www.gesetze-im-internet.de/solzg_1995/__4.html | 5,5 % (Satz 1), Cent-Bruchteile außer Ansatz (Satz 3) | Text 19.10.2026 | bank/zinsen-kapest; steuererklaerungen/kst; gesellschafter/ausschuettung |
| § 15b InsO | https://www.gesetze-im-internet.de/inso/__15b.html | Zahlungen an Gesellschafter, die zur Zahlungsunfähigkeit führen mussten (Abs. 5) | Text 17.10.2026 | gesellschafter/ausschuettung |
| § 19 InsO | (Eintrag grundlagen) | Rangrücktritt aus der Überschuldung (Abs. 2 Satz 2) | Text 17.10.2026 | bank/darlehen |
| § 39 InsO | https://www.gesetze-im-internet.de/inso/__39.html | Nachrang Gesellschafterdarlehen (Abs. 1 Satz 1 Nr. 5), Ausnahme ≤ 10 % ohne Geschäftsführung (Abs. 5) | Text 17.10.2026 | bank/darlehen |
| § 135 InsO | https://www.gesetze-im-internet.de/inso/__135.html | Anfechtung: Sicherung zehn Jahre, Befriedigung ein Jahr (Abs. 1) | Text 17.10.2026 | bank/darlehen |
| § 69 AO | https://www.gesetze-im-internet.de/ao_1977/__69.html | Haftung der Vertreter | Text 17.10.2026 | grundlagen/pflichten; lohn/abstimmung; gesellschafter/ausschuettung, vga-einlage |
| § 240 AO | https://www.gesetze-im-internet.de/ao_1977/__240.html | Säumniszuschlag 1 % je angefangenen Monat, Abrundung 50 €, Schonfrist drei Tage (Abs. 1, 3) | Text 17.10.2026 | lohn/abstimmung |

### Verwaltungsanweisungen und weitere Quellen

| Quelle | Link | Wofür | Geprüft | Genutzt in |
|---|---|---|---|---|
| R 8.5 KStR 2022 (Verdeckte Gewinnausschüttungen) | https://www.steuerschroeder.de/steuergesetze/kstr/8.5 | Begriff vGA, nahestehende Person (Abs. 1 Satz 1, 3); beherrschender Gesellschafter, klare Vereinbarung im Voraus (Abs. 2) | Text 17.10.2026 (Kanzleispiegel; KSt-Handbuch des BMF nicht abrufbar) | bank/darlehen; gesellschafter/ablauf, verrechnungskonto, vga-einlage |
| R 8.9 KStR 2022 (Verdeckte Einlage) | https://www.steuerschroeder.de/steuergesetze/kstr/8.9 | Begriff (Abs. 1, 3), Teilwert (Abs. 4 Satz 1) | Text 17.10.2026 (Kanzleispiegel) | gesellschafter/vga-einlage |
| UStAE Abschnitt 15.23 Abs. 8 | https://www.steuerschroeder.de/steuergesetze/ustae/15.23 | Fahrzeugüberlassung an Personal und Gesellschafter-Geschäftsführer entgeltlich (Satz 1, 4) | Text 17.10.2026 (Kanzleispiegel) | lohn/ablauf, lohnjournal; gesellschafter/vga-einlage |
| BMF vom 16.05.2025, IV C 1 - S 2401/00008/014/051 (Steuerbescheinigungen nach § 45a Abs. 2 und 3 EStG) | https://www.steuerschroeder.de/blog/wp-content/uploads/2025/05/2025-05-16-kapitalertragSt-steuerbescheinigung.pdf (Kanzleispiegel) | Pflicht auch ohne Abzug (Einleitung), Muster verbindlich (Rn. 1), Muster II für GmbH (Rn. 43), erst nach Zahlung (Rn. 47), auch vGA (Rn. 48), Muster für Zuflüsse ab 01.01.2026 (Rn. 71) | Text 17.10.2026 (Kanzleispiegel; BStBl-Fundstelle nicht geprüft) | gesellschafter/steuerbescheinigung; beteiligungen/dividenden, investmentfonds |
| ELSTER, Liste aller Formulare | https://www.elster.de/eportal/formulare-leistungen/alleformulare | Formular „Kapitalertragsteuer-Anmeldung nach Einkommensteuergesetz“ | Text 17.10.2026 | gesellschafter/ausschuettung |

### Werte mit Stand

| Wert | Stand | Quelle | Genutzt in |
|---|---|---|---|
| Kapitalertragsteuer 25 % | 17.10.2026 | § 43a Abs. 1 Satz 1 Nr. 1 EStG | gesellschafter/ablauf, ausschuettung |
| Solidaritätszuschlag 5,5 % der Kapitalertragsteuer | 17.10.2026 | §§ 3 Abs. 1 Nr. 5, 4 Satz 1 SolZG | gesellschafter/ablauf, ausschuettung |
| Minderung der KapESt um 25 % der Kirchensteuer | 17.10.2026 | § 43a Abs. 1 Satz 2 EStG | gesellschafter/ausschuettung |
| Abführung und Anmeldung bei Gewinnausschüttung im Zuflusszeitpunkt | 17.10.2026 | § 44 Abs. 1 Satz 5, § 45a Abs. 1 EStG | gesellschafter/ablauf, ausschuettung |
| Kirchensteuer-Regelabfrage 1.9.–31.10., Stichtag 31.8. | 17.10.2026 | § 51a Abs. 2c Satz 1 Nr. 3 EStG | gesellschafter/ausschuettung |
| Säumniszuschlag 1 % je angefangenen Monat, Schonfrist drei Tage | 17.10.2026 | § 240 Abs. 1, 3 AO | lohn/ablauf; gesellschafter/ausschuettung |
| Anfechtung Rückzahlung Gesellschafterdarlehen: ein Jahr, Sicherung: zehn Jahre | 17.10.2026 | § 135 Abs. 1 InsO | bank/darlehen; gesellschafter/ablauf |
| Nachrang-Ausnahme bis 10 % ohne Geschäftsführung | 17.10.2026 | § 39 Abs. 5 InsO | bank/darlehen |
| Zinsen an Gesellschafter ab 10 % Beteiligung ohne Abgeltungstarif | 17.10.2026 | § 32d Abs. 2 Nr. 1 Buchst. b EStG | bank/darlehen |

### Buchfink-eigene Quellen

| Quelle | Pfad | Wofür |
|---|---|---|
| Ergebnisverwendung | `internal/service/appropriation_service.go`, `internal/domain/appropriation.go` | Buchung 2970 an 3519 und 3760; 25 % KapESt auf den Cent und 5,5 % SolZ ohne Cent-Bruchteile auf die volle Ausschüttung; abzuführender Betrag auf volle Euro abgerundet in Vorschau und Pflicht (`accounting.WithholdingTaxPayable`); keine Kirchensteuer |
| Konten | `internal/domain/skr04_accounts.go`, `skr04_2026.json` | 1307–1309, 0960, 3640–3643, 3519, 3760, 2928, 7316, 7317, 7790 |
| Sperre Personengesellschaftskonten | `internal/accounting/legal_form_accounts.go` | 2000–2799 u. a. für GmbH/UG gesperrt |
| Werkzeugkatalog | `docs/entwicklung/mcp.md` | `auslage` mit `gesellschafter_kontakt_id`; `darlehen_erfassen` und `ergebnisverwendung_buchen` (Rückfrage) |
| Gesellschafter als Kontakt | `internal/service/shareholder_identity.go` | Rolle Gesellschafter oder Geschäftsführer über den Namen in den Unternehmensdaten |

### Offen

- BMF-Schreiben vom 14.10.2002 zur Angemessenheit der Gesamtausstattung von
  Gesellschafter-Geschäftsführern (BStBl I S. 972 laut Sekundärquellen) war
  nicht abrufbar und ist im Skill nicht zitiert.
- KStR und UStAE nur über Kanzleispiegel gelesen; Abgleich mit dem amtlichen
  KSt-Handbuch und UStAE.
- Parameter von `darlehen_erfassen` und `ergebnisverwendung_buchen` mit dem
  Werkzeugschema abgleichen, sobald die Werkzeuge im Katalog stehen.
- Buchfink bildet Kirchensteuer, NV-Bescheinigung und Einlagenrückgewähr nicht
  ab.
- Abrundung der KapESt (§ 44 Abs. 1 Satz 6 EStG): Buchfink nennt den
  abzuführenden Betrag, bucht aber die einbehaltene Steuer auf den Cent auf
  3760. Wem der Abrundungsrest zusteht und wie er auszubuchen ist, sagt der
  Wortlaut nicht; ebenso, ob die Abrundung über § 1 Abs. 2 SolZG auch für den
  Solidaritätszuschlag gilt. Kommentar oder Verwaltungsanweisung nicht
  gelesen.
- Buchfink rechnet die KapESt auf die ganze Ausschüttung, nicht je
  Gesellschafter (§ 44 Abs. 1 Satz 3 EStG); bei mehreren Gesellschaftern
  können die Summen um Cent abweichen.
- Ob eine GmbH ohne geplante Ausschüttung die Kirchensteuer-Regelabfrage
  durchführen muss, ist nicht geklärt (BZSt-Hinweise prüfen).
- SKR04-Katalog: Für 7316 und 7317 zeigt `position_id` auf
  „Abschreibungen auf Finanzanlagen“ (guv_12); vermutlich Extraktionsfehler,
  erwartet „Zinsen und ähnliche Aufwendungen“.

## Bereich lohn

Quellen des Bereichs `references/lohn/` (ablauf.md und Referenzen). Alle Links am
17.10.2026 mit HTTP 200 geprüft.

### Gesetze

| Norm | Link | Wofür | Geprüft | Genutzt in |
|---|---|---|---|---|
| § 8 EStG | https://www.gesetze-im-internet.de/estg/__8.html | Sachbezug ist Arbeitslohn (Abs. 2) | Text 17.10.2026 | lohn/lohnjournal |
| § 38 EStG | https://www.gesetze-im-internet.de/estg/__38.html | Einbehalt bei jeder Lohnzahlung (Abs. 3 Satz 1) | Text 17.10.2026 | lohn/ablauf |
| § 40 EStG | https://www.gesetze-im-internet.de/estg/__40.html | Arbeitgeber übernimmt und schuldet pauschale Lohnsteuer (Abs. 3 Satz 1, 2) | Text 17.10.2026 | lohn/lohnjournal |
| § 40a EStG | https://www.gesetze-im-internet.de/estg/__40a.html | einheitliche Pauschsteuer 2 % (Abs. 2), Erhebung durch Knappschaft-Bahn-See (Abs. 6 Satz 1) | Text 17.10.2026 | lohn/ablauf, lohnjournal |
| § 41 EStG | https://www.gesetze-im-internet.de/estg/__41.html | Lohnkonto, Aufbewahrung bis Ende des sechsten Jahres (Abs. 1 Satz 9) | Text 17.10.2026 | lohn/abstimmung |
| § 41a EStG | https://www.gesetze-im-internet.de/estg/__41a.html | Anmeldung und Zahlung bis 10. Tag (Abs. 1 Satz 1), Anmeldungszeitraum 1.080 € / 5.000 € (Abs. 2) | Text 17.10.2026 | grundlagen/pflichten; lohn/ablauf, lohnjournal |
| § 42d EStG | https://www.gesetze-im-internet.de/estg/__42d.html | Haftung des Arbeitgebers (Abs. 1 Nr. 1) | Text 17.10.2026 | lohn/abstimmung |
| § 7a SGB IV | https://www.gesetze-im-internet.de/sgb_4/__7a.html | Statusfeststellung bei geschäftsführendem Gesellschafter (Abs. 1 Satz 2) | Text 17.10.2026 | lohn/ablauf |
| § 8 SGB IV | https://www.gesetze-im-internet.de/sgb_4/__8.html | Geringfügigkeitsgrenze: Mindestlohn × 130 ÷ 3, aufgerundet, Bekanntgabe BMAS (Abs. 1a) | Text 17.10.2026 | lohn/ablauf |
| § 23 SGB IV | https://www.gesetze-im-internet.de/sgb_4/__23.html | Fälligkeit drittletzter Bankarbeitstag, Rest Folgemonat, Vormonatsbetrag (Abs. 1 Satz 2, 3) | Text 17.10.2026 | lohn/ablauf |
| § 28e SGB IV | https://www.gesetze-im-internet.de/sgb_4/__28e.html | Arbeitgeber zahlt Gesamtsozialversicherungsbeitrag (Abs. 1) | Text 17.10.2026 | lohn/ablauf, lohnjournal |
| § 28f SGB IV | https://www.gesetze-im-internet.de/sgb_4/__28f.html | Entgeltunterlagen bis Ende des Jahres nach letzter Prüfung (Abs. 1 Satz 1, 4), Beitragsnachweis zwei Arbeitstage vor Fälligkeit (Abs. 3 Satz 1) | Text 17.10.2026 | lohn/ablauf, abstimmung |
| § 358 SGB III | https://www.gesetze-im-internet.de/sgb_3/__358.html | Insolvenzgeldumlage | Text 17.10.2026 | lohn/lohnjournal |
| § 1 AAG | https://www.gesetze-im-internet.de/aufag/__1.html | Erstattungsanspruch U1 (Kürzel im Link: `aufag`) | Text 17.10.2026 | lohn/ablauf |
| § 7 AAG | https://www.gesetze-im-internet.de/aufag/__7.html | Umlagen U1, U2 | Text 17.10.2026 | lohn/lohnjournal |
| § 1 MiLoV5 | https://www.gesetze-im-internet.de/milov5/__1.html | Mindestlohn 13,90 € ab 01.01.2026, 14,60 € ab 01.01.2027 | Text 17.10.2026 | lohn/ablauf |
| § 34 AO | https://www.gesetze-im-internet.de/ao_1977/__34.html | Pflichten der gesetzlichen Vertreter (Abs. 1) | Text 17.10.2026 | lohn/abstimmung |
| § 266a StGB | https://www.gesetze-im-internet.de/stgb/__266a.html | Vorenthalten von Arbeitnehmerbeiträgen strafbar (Abs. 1) | Text 17.10.2026 | lohn/abstimmung |

§ 69 und § 240 AO: Zeilen unter Bereich gesellschafter.

### Verwaltungsanweisungen

| Quelle | Link | Wofür | Geprüft | Genutzt in |
|---|---|---|---|---|
| GoBD vom 28.11.2019 (Eintrag oben) | https://www.lfst.bayern.de/fileadmin/RESSOURCEN/INFORMATIONEN/Steuerinfos/Weitere_Themen/Aussenpruefung/2019-11-28-GoBD-1.pdf | verdichtete Buchung aufgliederbar (Rz. 42), Einzelaufzeichnungen der Vorsysteme bleiben (Rz. 99), Datenzugriff auf Lohnbuchhaltung (Rz. 159; 2024 um einen Satz ergänzt) | Text 17.10.2026 | lohn/ablauf, abstimmung |
| UStAE Abschnitt 15.23 Abs. 8 | Zeile unter Bereich gesellschafter | Firmenwagen an Personal | Text 17.10.2026 | lohn/ablauf, lohnjournal |

### Werte mit Stand

| Wert | Stand | Quelle | Genutzt in |
|---|---|---|---|
| Lohnsteuer-Anmeldungszeitraum: Monat; Quartal bei Vorjahreslohnsteuer > 1.080 € bis 5.000 €; Jahr bis 1.080 € | 17.10.2026 | § 41a Abs. 2 EStG | lohn/ablauf, abstimmung |
| Lohnsteuer-Anmeldung und Zahlung bis zum 10. Tag nach Ablauf des Zeitraums | 17.10.2026 | § 41a Abs. 1 Satz 1 EStG | lohn/ablauf, abstimmung |
| SV-Beiträge am drittletzten Bankarbeitstag; Beitragsnachweis zwei Arbeitstage vorher | 17.10.2026 | § 23 Abs. 1, § 28f Abs. 3 SGB IV | lohn/ablauf, abstimmung |
| Minijob-Pauschsteuer 2 % | 17.10.2026 | § 40a Abs. 2 EStG | lohn/ablauf, lohnjournal |
| Mindestlohn 13,90 € (2026), 14,60 € (2027) | 17.10.2026 | § 1 MiLoV5 | lohn/ablauf |
| Geringfügigkeitsgrenze 603 € (2026), 633 € (2027), aus § 8 Abs. 1a SGB IV gerechnet; BMAS-Bekanntmachung nicht geprüft | 17.10.2026 | § 8 Abs. 1a SGB IV, MiLoV5 | lohn/ablauf |
| Lohnkonto aufbewahren bis Ablauf des sechsten Kalenderjahres nach letzter Lohnzahlung | 17.10.2026 | § 41 Abs. 1 Satz 9 EStG | lohn/ablauf, abstimmung |

### Buchfink-eigene Quellen

| Quelle | Pfad | Wofür |
|---|---|---|
| Werkzeugkatalog | `docs/entwicklung/mcp.md` | Felder von `lohnjournal_buchen`, Netto-Probe |
| Konten | `internal/accounting/skr04_2026.json` | 6020, 6024, 6027, 6035–6040, 6080, 6110, 6120, 6170, 6171, 1340, 3720, 3725, 3730, 3740, 3759, 3790, 4947, 4949, 4972, 6650 |
| Buchungsgruppe Gehälter | `internal/accounting/posting_groups.go` | `gehaelter` → 6020, nicht steuerbar |

### Offen

- `PayrollService.BookSummary` ist neu; welche Konten es bebucht, steht nicht
  fest. Der Skill nennt die Konten nach SKR04-Bezeichnung und verlangt die
  Prüfung der Antwort.
- Die Summenfelder kennen ein Brutto. Ob Geschäftsführergehälter (6024, 6027)
  und Minijobs (6035, 6036, 6171) getrennt gebucht werden, ist offen.
- Sonderzeilen (Vorschuss, Pfändung, vermögenswirksame Leistung, Sachbezug)
  haben keine Felder; der Skill bucht sie per `buchung_erfassen` um.
- Umsatzsteuer auf Firmenwagenüberlassung: Bemessungsgrundlage nicht im Skill
  (UStAE 15.23 Abs. 11 ff. nicht gelesen).

## Bereich monatsabschluss

Quellen des Bereichs `references/monatsabschluss/` (ablauf.md und Referenzen). Alle
Links am 17.10.2026 mit HTTP 200 geprüft. „Text“ = Wortlaut gelesen und
Aussage abgeglichen.

### Gesetze

| Norm | Link | Wofür | Geprüft | Genutzt in |
|---|---|---|---|---|
| § 13 UStG | https://www.gesetze-im-internet.de/ustg_1980/__13.html | Sollversteuerung: Steuer entsteht mit Ablauf des VAZ der Leistung; Anzahlung mit Ablauf des VAZ der Vereinnahmung (Abs. 1 Nr. 1 Buchst. a Satz 1, 4) | Text 17.10.2026 | monatsabschluss/ustva |
| § 17 UStG | https://www.gesetze-im-internet.de/ustg_1980/__17.html | Berichtigung für den Zeitraum der Änderung (Abs. 1 Satz 8) | Text 17.10.2026 | monatsabschluss/ablauf, berichtigung |
| § 18 UStG | https://www.gesetze-im-internet.de/ustg_1980/__18.html | VA bis 10. Tag, elektronisch, Härtefall Papier (Abs. 1 Satz 1, 2); § 17 entsprechend (Satz 3); Fälligkeit 10. Tag (Satz 4); VAZ Quartal / Monat über 9.000 € / Befreiung bis 2.000 € / Neugründung Monat / Vorratsgesellschaft, Mantel / 2021–2026 voraussichtliche Steuer (Abs. 2 Satz 1–6); Wahl Monat bei Überschuss über 9.000 €, Abgabe bis 10. Februar (Abs. 2a); Jahreserklärung (Abs. 3); Ermächtigung Fristverlängerung und Sondervorauszahlung (Abs. 6) | Text 17.10.2026 | monatsabschluss/berichtigung, ustva |
| § 18a UStG | https://www.gesetze-im-internet.de/ustg_1980/__18a.html | ZM: 25. Tag, Quartal für § 3a-Abs.-2-Leistungen (Abs. 2), Monatsoption mit Anzeige (Abs. 3), Inhalt (Abs. 7 Nr. 3), Meldezeitraum der Ausführung (Abs. 8 Satz 2), Berichtigung binnen eines Monats (Abs. 10), AO ohne § 152 (Abs. 11) | Text 17.10.2026 | monatsabschluss/zm |
| § 18b UStG | https://www.gesetze-im-internet.de/ustg_1980/__18b.html | gesonderte Erklärung der Leistungen nach § 3a Abs. 2 in der VA (Satz 1 Nr. 2, Satz 3); unverzügliche Berichtigung (Satz 5) | Text 17.10.2026 | monatsabschluss/ustva, zm |
| § 26a UStG | https://www.gesetze-im-internet.de/ustg_1980/__26a.html | Bußgeld nicht rechtzeitig entrichtete Vorauszahlung bis 30.000 € (Abs. 1, 3); ZM nicht/falsch/verspätet/nicht berichtigt bis 5.000 € (Abs. 2 Nr. 5, Abs. 3) | Text 17.10.2026 | grundlagen/pflichten |
| § 46 UStDV | https://www.gesetze-im-internet.de/ustdv_1980/__46.html | Fristverlängerung um einen Monat für VA und Vorauszahlung; Ablehnung/Widerruf bei Gefährdung | Text 17.10.2026 | monatsabschluss/dauerfristverlaengerung |
| § 47 UStDV | https://www.gesetze-im-internet.de/ustdv_1980/__47.html | Sondervorauszahlung nur Monatszahler, 1/11 der Vorauszahlungen des Vorjahres; Hochrechnung; Neubeginn | Text 17.10.2026 | grundlagen/pflichten; monatsabschluss/dauerfristverlaengerung |
| § 48 UStDV | https://www.gesetze-im-internet.de/ustdv_1980/__48.html | Antrag bis zum Abgabetag der ersten betroffenen VA, Datensatz, SVZ selbst berechnen und entrichten (Abs. 1); jährlich bis zur ersten VA (Abs. 2); Festsetzung (Abs. 3); Anrechnung in der letzten VA (Abs. 4) | Text 17.10.2026 | monatsabschluss/dauerfristverlaengerung |
| § 108 AO | https://www.gesetze-im-internet.de/ao_1977/__108.html | Fristende Samstag, Sonntag, Feiertag → nächster Werktag (Abs. 3) | Text 17.10.2026 | grundlagen/pflichten; steuererklaerungen/fristen-zuschlaege; gruendung/rumpfjahr |
| § 146 AO | https://www.gesetze-im-internet.de/ao_1977/__146.html | Unveränderbarkeit (Abs. 4) | Text 17.10.2026 (Eintrag grundlagen) | grundlagen/aufbewahrung, gobd, pflichten |
| § 152 AO | https://www.gesetze-im-internet.de/ao_1977/__152.html | Verspätungszuschlag Ermessen (Abs. 1); bei monatlichen/vierteljährlichen Anmeldungen und SVZ-Anmeldung keine 0,25-%-Regel, Bemessung nach Dauer, Häufigkeit, Höhe (Abs. 8); höchstens 25.000 € (Abs. 10) | Text 17.10.2026 | monatsabschluss/ustva |
| § 153 AO | https://www.gesetze-im-internet.de/ao_1977/__153.html | Berichtigungspflicht bei möglicher Steuerverkürzung (Abs. 1) | Text 17.10.2026 | monatsabschluss/berichtigung |
| § 164 AO | https://www.gesetze-im-internet.de/ao_1977/__164.html | Änderung unter Vorbehalt jederzeit beantragbar (Abs. 2) | Text 17.10.2026 | monatsabschluss/berichtigung |
| § 168 AO | https://www.gesetze-im-internet.de/ao_1977/__168.html | Anmeldung = Festsetzung unter Vorbehalt; Herabsetzung/Erstattung erst mit Zustimmung | Text 17.10.2026 | monatsabschluss/ustva |
| § 240 AO | https://www.gesetze-im-internet.de/ao_1977/__240.html | Säumniszuschlag (Verweis; Werte in bank) | Text 17.10.2026 (Eintrag bank) | grundlagen/pflichten; bank/steuerzahlungen; lohn/abstimmung |

### Verwaltungsanweisungen und amtliche Stellen

| Quelle | Link | Wofür | Geprüft | Genutzt in |
|---|---|---|---|---|
| GoBD, BMF vom 28.11.2019 (Eintrag oben) | https://www.lfst.bayern.de/fileadmin/RESSOURCEN/INFORMATIONEN/Steuerinfos/Weitere_Themen/Aussenpruefung/2019-11-28-GoBD-1.pdf | Rz. 109 (Vorerfassungen bis zum Jahresabschluss offen: unzulässig), Rz. 110 (Festschreibung als Mittel) | Text 17.10.2026 | grundlagen/arbeitsweise, aufbewahrung, gobd, pflichten; belege/fremdwaehrung-auslagen, steuerfaelle; bank/darlehen, skonto-differenzen; monatsabschluss/festschreibung; lohn/abstimmung |
| ELSTER, Liste aller Formulare | https://www.elster.de/eportal/formulare-leistungen/alleformulare | Formulare „Umsatzsteuer-Voranmeldung“, „Dauerfristverlängerung/Sondervorauszahlung (monatlich)“, „Zusammenfassende Meldung“ unter „Umsatzsteuer“ | Text 17.10.2026 (Eintrag pflichten) | grundlagen/pflichten; monatsabschluss/ablauf, dauerfristverlaengerung, ustva; steuererklaerungen/ebilanz, kst; gesellschafter/ausschuettung |
| BZSt, ZM: Elektronische Abgabe | https://www.bzst.de/DE/Unternehmen/Umsatzsteuer/ZusammenfassendeMeldung/ElektronischeAbgabe/elektronischeabgabe_zm_node.html | Übermittlung über ELSTER oder BOP; ELSTER ohne, BOP mit Anmeldung zum Fachverfahren ZM; CSV-Import; bis 1.500 Meldezeilen je Formular | Text 17.10.2026 (erneut geprüft) | grundlagen/pflichten; monatsabschluss/ablauf, zm |
| BZSt, ZM: Berichtigung | https://www.bzst.de/DE/Unternehmen/Umsatzsteuer/ZusammenfassendeMeldung/BerichtigungZM/berichtigung_node.html | Abgleich mit VA; je Meldezeitraum eigene Berichtigung; korrekte Zeilen nicht wiederholen; Storno mit Bemessungsgrundlage 0 (die Seite nennt § 26a Abs. 1 Nr. 5; nach Wortlaut Abs. 2 Nr. 5) | Text 17.10.2026 | monatsabschluss/zm |
| BZSt, ZM: Fristen | https://www.bzst.de/DE/Unternehmen/Umsatzsteuer/ZusammenfassendeMeldung/Fristen/fristen_node.html | 25. Tag; Dauerfristverlängerung gilt nicht für die ZM | Text 17.10.2026 | grundlagen/pflichten; monatsabschluss/ablauf, zm |
| BMF vom 29.12.2025, III C 3 – S 7344/00039/007/036 (Vordruckmuster USt 1 A, USt 1 H, Anleitungen USt 1 E, USt 5 E für 2026) | Sekundär: https://www.steuerschroeder.de/blog/umsatzsteuer-2026-neue-vordrucke-fuer-voranmeldung-und-dauerfristverlaengerung/, https://raw-partner.de/files/blog/umsatzsteuerformulare-2026.pdf | Existenz, Datum, GZ; Kennzahl 500 ersetzt Kennzahl 23 (vier Kategorien, Anlage „Ergänzende Angaben zur Steueranmeldung“, personelle Prüfung) | Titel über Sekundärquellen 17.10.2026; Wortlaut nicht gelesen | monatsabschluss/ablauf, dauerfristverlaengerung, ustva |
| Vordruck USt 1 A 2026 (Formular-ID 2026USt1A561, „Dezember 2025“) mit Anleitung, Wiedergabe durch Buhl Data | https://www.buhl.de/steuer/wp-content/uploads/sites/31/2026/03/formular-dauerfristverlaengerung-2026.pdf | Zeile 30: Kz 46/47 nur sonstige Leistungen nach § 3a Abs. 2 eines im übrigen Gemeinschaftsgebiet ansässigen Unternehmers (§ 13b Abs. 1); Zeile 31: Kz 73/74 GrEStG-Umsätze (§ 13b Abs. 2 Nr. 3); Zeile 32: Kz 84/85 andere Leistungen (§ 13b Abs. 2 Nr. 1, 2, 4 bis 12), laut Anleitung u. a. Werklieferungen und sonstige Leistungen eines im Ausland ansässigen Unternehmers; Zeile 55: Kz 500 mit Schlüsseln 1–4 | Text 17.10.2026 (Wiedergabe eines Softwareherstellers, nicht amtlich; BMF-Original 503) | monatsabschluss/ustva |
| UStAE 18.4 (Dauerfristverlängerung) | https://www.steuerschroeder.de/steuergesetze/ustae/18.4 (Kanzleispiegel) | gelesen zum Abgleich; der Skill stützt sich auf §§ 46–48 UStDV. Der Spiegel ist teils veraltet (18.2 nennt noch 7.500 €, 18.1 Abs. 4 den 31. Mai) | Text 17.10.2026 (Spiegel) | – |

### Sonstige Quellen

| Quelle | Link | Wofür | Geprüft | Genutzt in |
|---|---|---|---|---|

### Werte mit Stand

| Wert | Stand | Quelle | Genutzt in |
|---|---|---|---|
| VA und Vorauszahlung am 10. Tag nach Ablauf des VAZ | 17.10.2026 | § 18 Abs. 1 Satz 1, 4 UStG | monatsabschluss/ablauf, ustva |
| Dauerfristverlängerung einen Monat | 17.10.2026 | § 46 UStDV | monatsabschluss/ablauf, dauerfristverlaengerung, ustva |
| ZM am 25. Tag; Quartal für sonstige Leistungen | 17.10.2026 | § 18a Abs. 1, 2 UStG | monatsabschluss/ablauf, zm |
| VAZ: Monat über 9.000 € Vorjahressteuer, Befreiung möglich bis 2.000 €, Wahl Monat bei Überschuss über 9.000 € (bis 10. Februar) | 17.10.2026 | § 18 Abs. 2, 2a UStG | monatsabschluss/ustva |
| Neugründer: Sonderregel voraussichtliche Steuer für Besteuerungszeiträume 2021 bis 2026 | 17.10.2026; für 2027 Wortlaut prüfen | § 18 Abs. 2 Satz 6 UStG | monatsabschluss/ustva |
| Sondervorauszahlung 1/11 der Vorauszahlungen des Vorjahres, nur Monatszahler | 17.10.2026 | § 47 Abs. 1 UStDV | monatsabschluss/dauerfristverlaengerung |
| ZM-Berichtigung binnen eines Monats | 17.10.2026 | § 18a Abs. 10 UStG | monatsabschluss/berichtigung, zm |
| Bußgeld ZM bis 5.000 €, Vorauszahlung bis 30.000 € | 17.10.2026 | § 26a Abs. 1–3 UStG | monatsabschluss/ablauf, ustva, zm |
| Verspätungszuschlag höchstens 25.000 € | 17.10.2026 | § 152 Abs. 10 AO | monatsabschluss/ustva |
| ZM-Formular bis 1.500 Meldezeilen | BZSt-Seite 17.10.2026 | BZSt, Elektronische Abgabe | monatsabschluss/zm |

### Buchfink-eigene Quellen

| Quelle | Wofür |
|---|---|
| `internal/service/month_close_service.go` | Reihenfolge Prüfbericht → Festschreibung → Bestätigung; Quartal: Monate einzeln |
| `internal/service/vat_return_service.go` (`ensureCommitted`, Kennziffer 10, eine Erstanmeldung je Zeitraum) | Transferticket erst nach Festschreibung; berichtigte Anmeldung |
| `internal/accounting/ustva.go` | Kennziffernliste USt 1 A; Bemessungsgrundlage in vollen Euro abgeschnitten; Kz 83 = Steuer − Vorsteuer − Kz 39 |
| `internal/accounting/vat_period.go` | Zuordnung nach Leistungsdatum; Vorsteuer nach spätestem Datum; Storno übernimmt Daten der Ursprungsbuchung, Storno einer Ausgangsrechnung wirkt im Zeitraum des Stornos |
| `internal/service/obligation_catalog.go`, `internal/calendar/holidays.go` | Fälligkeit in `fristen` mit Verschiebung nach § 108 Abs. 3 AO und Feiertagen des Bundeslands |
| `internal/domain/check.go`, `internal/service/check_service.go` | Prüflaufregeln und Schwere |
| `internal/domain/festschreibung.go` | Festschreibung mit Zeitstempel, Monat/Quartal/Jahr |
| `docs/entwicklung/mcp.md` | `erklaerung`, `erklaerung_exportieren` (`zahlen_hash`, `naechster_schritt`), Transferticket nur App |

### Offen

- Amtlicher Datensatz: Ob ELSTER die Steuer zu Kz 81/86 aus der
  Bemessungsgrundlage selbst rechnet und Kz 83 dann um Cent von der gebuchten
  Steuer abweicht, ist nicht geprüft (Anleitung USt 1 E 2026 nicht lesbar,
  bundesfinanzministerium.de 503).
- BMF-Schreiben vom 29.12.2025 (Vordrucke 2026, Kz 500) nur über
  Sekundärquellen; den Vordruck selbst gibt die Buhl-Datei wieder.
- Summe der Vorauszahlungen für die Sondervorauszahlung: Behandlung von
  Überschussmonaten und der Vorjahres-SVZ im Wortlaut offen.
- Buchfink: Die berichtigte ZM rechnet den ganzen Zeitraum neu; das BZSt
  verlangt, korrekt gemeldete Zeilen nicht zu wiederholen. Export prüfen.
- Buchfink: Eine CSV-Datei der ZM bietet weder App noch Werkzeug; der Mensch
  trägt die Zeilen ein.

## Bereich jahresabschluss

Quellen des Bereichs `references/jahresabschluss/` (ablauf.md und Referenzen). Alle
Links am 17.10.2026 mit HTTP 200 geprüft.

### Gesetze

| Norm | Link | Wofür | Geprüft | Genutzt in |
|---|---|---|---|---|
| § 240 HGB | https://www.gesetze-im-internet.de/hgb/__240.html | Inventar zum Schluss jedes Geschäftsjahres, ordnungsmäßiger Geschäftsgang (Abs. 1, 2) | Text 17.10.2026 | jahresabschluss/abschlussbuchungen |
| § 242 HGB | https://www.gesetze-im-internet.de/hgb/__242.html | Bilanz und GuV bilden den Jahresabschluss (Abs. 3); Aufstellungspflicht | Text 17.10.2026 | jahresabschluss/ablauf, rueckstellungen |
| § 243 HGB | https://www.gesetze-im-internet.de/hgb/__243.html | GoB, klar und übersichtlich, ordnungsmäßiger Geschäftsgang | Text 17.10.2026 | jahresabschluss/aufstellen-feststellen-offenlegen |
| § 245 HGB | https://www.gesetze-im-internet.de/hgb/__245.html | Unterzeichnung mit Datum | Text 17.10.2026 | jahresabschluss/aufstellen-feststellen-offenlegen |
| § 246 HGB | https://www.gesetze-im-internet.de/hgb/__246.html | Vollständigkeit (Abs. 1), Verrechnungsverbot (Abs. 2 Satz 1), Ansatzstetigkeit (Abs. 3) | Text 17.10.2026 | jahresabschluss/abgrenzung, abschlussbuchungen, forderungen |
| § 249 HGB | https://www.gesetze-im-internet.de/hgb/__249.html | Rückstellungspflichten (Abs. 1), Verbot anderer, Auflösung nur bei Wegfall (Abs. 2) | Text 17.10.2026 | jahresabschluss/rueckstellungen, steuern-im-abschluss, unfertige-leistungen |
| § 250 HGB | https://www.gesetze-im-internet.de/hgb/__250.html | aktive/passive RAP (Abs. 1, 2), Disagio und Abschreibung (Abs. 3) | Text 17.10.2026 | jahresabschluss/abgrenzung |
| § 251 HGB | https://www.gesetze-im-internet.de/hgb/__251.html | Haftungsverhältnisse | Text 17.10.2026 | jahresabschluss/groessenklassen-anhang |
| § 252 HGB | https://www.gesetze-im-internet.de/hgb/__252.html | Bilanzidentität (Nr. 1), Einzelbewertung (Nr. 3), Vorsicht, Wertaufhellung, Realisation (Nr. 4), Periodenabgrenzung (Nr. 5), Stetigkeit (Nr. 6), Ausnahmen (Abs. 2) | Text 17.10.2026 | jahresabschluss/ablauf, abgrenzung, forderungen, unfertige-leistungen |
| § 253 HGB | https://www.gesetze-im-internet.de/hgb/__253.html | Erfüllungsbetrag (Abs. 1 Satz 2), Kleinst ohne Zeitwert (Abs. 1 Satz 5, 6), Abzinsung 7/10 Jahre, Bundesbank monatlich (Abs. 2 Satz 1, 4), planmäßig und außerplanmäßig (Abs. 3 Satz 1, 2, 5, 6), Umlaufvermögen (Abs. 4), Wertaufholung (Abs. 5) | Text 17.10.2026 | jahresabschluss/abschreibungen, forderungen, groessenklassen-anhang, rueckstellungen, unfertige-leistungen |
| § 255 HGB | https://www.gesetze-im-internet.de/hgb/__255.html | Herstellungskosten: Pflicht, Wahlrecht, Verbot (Abs. 2), Zinsen (Abs. 3) | Text 17.10.2026 | jahresabschluss/unfertige-leistungen |
| § 256a HGB | https://www.gesetze-im-internet.de/hgb/__256a.html | Devisenkassamittelkurs; Restlaufzeit bis ein Jahr ohne Höchstwert/Imparität | Text 17.10.2026 | jahresabschluss/forderungen |
| § 264 HGB | https://www.gesetze-im-internet.de/hgb/__264.html | Anhang (Abs. 1 Satz 1), Frist 3 Monate / kleine bis 6 Monate, kein Lagebericht (Satz 3, 4), Kleinst ohne Anhang (Satz 5), Firma/Sitz/Register (Abs. 1a), zusätzliche Angaben (Abs. 2 Satz 2, 4) | Text 17.10.2026 | jahresabschluss/ablauf, aufstellen-feststellen-offenlegen, groessenklassen-anhang |
| § 266 HGB | https://www.gesetze-im-internet.de/hgb/__266.html | verkürzte Bilanz kleine/Kleinst (Abs. 1 Satz 3, 4); Posten unfertige Leistungen (Abs. 2 B. I. 2.) | Text 17.10.2026 | jahresabschluss/groessenklassen-anhang, unfertige-leistungen |
| § 267 HGB | https://www.gesetze-im-internet.de/hgb/__267.html | Schwellen klein 7,5 Mio./15 Mio./50, mittel 25 Mio./50 Mio./250 (Abs. 1, 2); Zweijahresregel, Neugründung (Abs. 4); Bilanzsumme (Abs. 4a); Arbeitnehmerzahl (Abs. 5) | Text 17.10.2026 | jahresabschluss/groessenklassen-anhang |
| § 267a HGB | https://www.gesetze-im-internet.de/hgb/__267a.html | Kleinst 450.000/900.000/10 (Abs. 1); Regeln für kleine gelten (Abs. 2); reine Beteiligungsholding keine Kleinst (Abs. 3 Nr. 3) | Text 17.10.2026 | grundlagen/pflichten; jahresabschluss/groessenklassen-anhang |
| § 268 HGB | https://www.gesetze-im-internet.de/hgb/__268.html | teilweise Ergebnisverwendung, Bilanzgewinn (Abs. 1); nicht gedeckter Fehlbetrag (Abs. 3); Restlaufzeitvermerke (Abs. 4, 5); Anzahlungen (Abs. 5 Satz 2); Disagio (Abs. 6); Haftungsverhältnisse (Abs. 7) | Text 17.10.2026 | jahresabschluss/abgrenzung, ergebnisverwendung-ug, forderungen, groessenklassen-anhang, unfertige-leistungen |
| § 270 HGB | https://www.gesetze-im-internet.de/hgb/__270.html | gesetzliche Rücklagen bei Aufstellung berücksichtigen (Abs. 2) | Text 17.10.2026 | jahresabschluss/ergebnisverwendung-ug |
| § 274a HGB | https://www.gesetze-im-internet.de/hgb/__274a.html | Erleichterungen kleine: § 268 Abs. 4 Satz 2, Abs. 5 Satz 3, Abs. 6, latente Steuern | Text 17.10.2026 | jahresabschluss/abgrenzung, forderungen, groessenklassen-anhang |
| § 275 HGB | https://www.gesetze-im-internet.de/hgb/__275.html | Rücklagenveränderungen nach Jahresüberschuss (Abs. 4); Kleinst-GuV (Abs. 5) | Text 17.10.2026 | jahresabschluss/ergebnisverwendung-ug, groessenklassen-anhang |
| § 276 HGB | https://www.gesetze-im-internet.de/hgb/__276.html | Rohergebnis für kleine und mittelgroße | Text 17.10.2026 | jahresabschluss/groessenklassen-anhang |
| § 277 HGB | https://www.gesetze-im-internet.de/hgb/__277.html | außerplanmäßige Abschreibungen gesondert oder Anhang (Abs. 3 Satz 1); Abzinsung unter Zinsen, Währungsumrechnung unter sonstigen betrieblichen Posten (Abs. 5) | Text 17.10.2026 | jahresabschluss/forderungen, groessenklassen-anhang, rueckstellungen |
| § 284 HGB | https://www.gesetze-im-internet.de/hgb/__284.html | Methoden (Abs. 2 Nr. 1), Abweichungen (Nr. 2), Anlagenspiegel (Abs. 3) | Text 17.10.2026 | jahresabschluss/abschreibungen, groessenklassen-anhang |
| § 285 HGB | https://www.gesetze-im-internet.de/hgb/__285.html | Nr. 1, 3a, 7, 9 c, 12, 13, 31, 34 | Text 17.10.2026 | jahresabschluss/ergebnisverwendung-ug, groessenklassen-anhang, rueckstellungen; beteiligungen/erwerb-ausweis |
| § 288 HGB | https://www.gesetze-im-internet.de/hgb/__288.html | Befreiungen kleine (Abs. 1 Nr. 1–3) | Text 17.10.2026 | jahresabschluss/abschreibungen, ergebnisverwendung-ug, groessenklassen-anhang, rueckstellungen; beteiligungen/erwerb-ausweis |
| § 316 HGB | https://www.gesetze-im-internet.de/hgb/__316.html | Prüfungspflicht ab mittelgroß; ohne Prüfung keine Feststellung (Abs. 1) | Text 17.10.2026 | jahresabschluss/groessenklassen-anhang |
| § 325 HGB | https://www.gesetze-im-internet.de/hgb/__325.html | Offenlegung an Unternehmensregister, deutsch, elektronisch (Abs. 1); ein Jahr (Abs. 1a); Änderungen (Abs. 1b) | Text 17.10.2026 | jahresabschluss/aufstellen-feststellen-offenlegen, rueckstellungen |
| § 326 HGB | https://www.gesetze-im-internet.de/hgb/__326.html | kleine: Bilanz und Anhang ohne GuV-Angaben; Kleinst: nur Bilanz, Hinterlegung mit Mitteilung | Text 17.10.2026 | grundlagen/pflichten; jahresabschluss/aufstellen-feststellen-offenlegen |
| § 335 HGB | https://www.gesetze-im-internet.de/hgb/__335.html | Ordnungsgeld 2.500–25.000 €, auch gegen Gesellschaft (Abs. 1); sechs Wochen (Abs. 3); Herabsetzung 500/1.000/2.500 € (Abs. 4 Satz 2) | Text 17.10.2026 | grundlagen/pflichten |
| Art. 93 EGHGB | https://www.gesetze-im-internet.de/hgbeg/art_93.html | neue Schwellen ab Geschäftsjahren nach dem 31.12.2023, wahlweise nach dem 31.12.2022 | Text 17.10.2026 | jahresabschluss/groessenklassen-anhang |
| § 5 GmbHG | https://www.gesetze-im-internet.de/gmbhg/__5.html | Mindeststammkapital 25.000 € (Abs. 1) | Text 17.10.2026 | jahresabschluss/ergebnisverwendung-ug |
| § 5a GmbHG | https://www.gesetze-im-internet.de/gmbhg/__5a.html | UG-Rücklage ein Viertel nach Verlustvortrag (Abs. 3 Satz 1), Verwendung (Satz 2), Entfall ab 25.000 € (Abs. 5) | Text 17.10.2026 | jahresabschluss/ergebnisverwendung-ug |
| § 29 GmbHG | https://www.gesetze-im-internet.de/gmbhg/__29.html | Anspruch, Bilanzgewinn, Rücklagen und Vortrag, Verteilung | Text 17.10.2026 | jahresabschluss/ergebnisverwendung-ug |
| § 42 GmbHG | https://www.gesetze-im-internet.de/gmbhg/__42.html | Posten gegenüber Gesellschaftern (Abs. 3) | Text 17.10.2026 | jahresabschluss/forderungen, groessenklassen-anhang |
| § 42a GmbHG | https://www.gesetze-im-internet.de/gmbhg/__42a.html | Vorlage unverzüglich (Abs. 1); Feststellung 8/11 Monate, nicht verlängerbar (Abs. 2) | Text 17.10.2026 | jahresabschluss/aufstellen-feststellen-offenlegen, ergebnisverwendung-ug |
| § 46 GmbHG | https://www.gesetze-im-internet.de/gmbhg/__46.html | Feststellung und Ergebnisverwendung durch Gesellschafter (Nr. 1) | Text 17.10.2026 | jahresabschluss/aufstellen-feststellen-offenlegen, ergebnisverwendung-ug |
| § 48 GmbHG | https://www.gesetze-im-internet.de/gmbhg/__48.html | Beschlussform, Alleingesellschafter-Niederschrift | Text 17.10.2026 | jahresabschluss/aufstellen-feststellen-offenlegen, ergebnisverwendung-ug |
| § 4 EStG | https://www.gesetze-im-internet.de/estg/__4.html | Gewerbesteuer keine Betriebsausgabe (Abs. 5b) | Text 17.10.2026 | belege/bewirtung-geschenke, vorsteuer; bank/steuerzahlungen; jahresabschluss/steuern-im-abschluss; steuererklaerungen/kst |
| § 5 EStG | https://www.gesetze-im-internet.de/estg/__5.html | Verzeichnisse für steuerliche Wahlrechte (Abs. 1 Satz 2, 3); Drohverlustverbot (Abs. 4a); keine Rückstellung für künftige AK/HK (Abs. 4b Satz 1); RAP und Wahlrecht bis Betrag § 6 Abs. 2 Satz 1, einheitlich (Abs. 5 Satz 1, 2) | Text 17.10.2026 | jahresabschluss/abgrenzung, abschreibungen, rueckstellungen |
| § 6 EStG | https://www.gesetze-im-internet.de/estg/__6.html | Teilwert bei dauernder Wertminderung, Nachweis (Abs. 1 Nr. 1 Satz 2–4, Nr. 2 Satz 1–2); Verwaltungskosten in HK (Nr. 1b); Verbindlichkeiten (Nr. 3); Rückstellungen a–f, 5,5 % (Nr. 3a); GWG 800 € (Abs. 2 Satz 1); Sammelposten Fünftel, Ausscheiden mindert nicht (Abs. 2a Satz 2, 3) | Text 17.10.2026 | jahresabschluss/abgrenzung, abschreibungen, forderungen, rueckstellungen, unfertige-leistungen |
| § 7 EStG | https://www.gesetze-im-internet.de/estg/__7.html | lineare AfA, Zwölftel (Abs. 1 Satz 1, 2, 4); degressiv 01.07.2025–31.12.2027, dreifach, 30 % (Abs. 2) | Text 17.10.2026 | belege/anlagen-gwg |
| § 7g EStG | https://www.gesetze-im-internet.de/estg/__7g.html | Sonderabschreibung bis zu insgesamt 40 % in fünf Jahren (Abs. 5); Voraussetzungen: Gewinngrenze 200.000 € im vorangehenden Wirtschaftsjahr (Abs. 6 Nr. 1 mit Abs. 1 Satz 2 Nr. 1 Buchst. b), Nutzung (Abs. 6 Nr. 2) | Text (Abs. 1, 5, 6) 17.10.2026 | jahresabschluss/abschreibungen |
| § 36 EStG | https://www.gesetze-im-internet.de/estg/__36.html | Anrechnung Abzugsteuer nur mit Bescheinigung (Abs. 2 Nr. 2, Satz 2) | Text 17.10.2026 | steuererklaerungen/kst; beteiligungen/dividenden |
| § 10 KStG | https://www.gesetze-im-internet.de/kstg_1977/__10.html | Steuern vom Einkommen nicht abziehbar (Nr. 2) | Text 17.10.2026 | belege/vorsteuer; bank/steuerzahlungen, zinsen-kapest; jahresabschluss/steuern-im-abschluss; steuererklaerungen/kst; beteiligungen/steuerfreistellung |
| § 17 UStG | https://www.gesetze-im-internet.de/ustg_1980/__17.html | Umsatzsteuerkorrektur erst bei Uneinbringlichkeit (Abs. 2 Nr. 1) → EWB auf Nettobetrag | Text 17.10.2026 | belege/bewirtung-geschenke, vorsteuer; bank/skonto-differenzen; rechnungen/anzahlungen, forderungsausfall, pflichtangaben-ausgang, storno-berichtigung; monatsabschluss/berichtigung; jahresabschluss/forderungen |
| § 1, § 7 RückAbzinsV | https://www.gesetze-im-internet.de/r_ckabzinsv/__1.html | Bundesbank ermittelt Abzinsungssätze; monatliche Veröffentlichung für 1 bis 50 Jahre (§ 7) | Text 17.10.2026 | jahresabschluss/rueckstellungen |
| §§ 640, 641 BGB | https://www.gesetze-im-internet.de/bgb/__640.html, https://www.gesetze-im-internet.de/bgb/__641.html | Abnahme, Fälligkeit bei Abnahme und Teilabnahme | Text 17.10.2026 | jahresabschluss/unfertige-leistungen |

### Sonstige Quellen

| Quelle | Link | Wofür | Geprüft | Genutzt in |
|---|---|---|---|---|
| Deutsche Bundesbank, Abzinsungszinssätze § 253 Abs. 2 HGB, 7-Jahresdurchschnitt | https://www.bundesbank.de/de/statistiken/geld-und-kapitalmaerkte/zinssaetze-und-renditen/abzinsungszinssaetze-gemaess-253-abs-2-hgb-7-jahresdurchschnitt-650652 | Fundort der monatlichen Sätze (Werte nicht in den Skill übernommen) | Titel 17.10.2026 | jahresabschluss/rueckstellungen |
| Unternehmensregister | https://www.unternehmensregister.de | Publikations-Plattform, „Auftrag übermitteln“ für Veröffentlichung oder Hinterlegung | Text der Startseite 17.10.2026 (`curl`, 200; WebFetch erhielt 403): „Wenn Sie eine Veröffentlichung oder Hinterlegung im Unternehmensregister vornehmen möchten, können Sie über die Publikations-Plattform einen entsprechenden Auftrag übermitteln.“ | jahresabschluss/aufstellen-feststellen-offenlegen |

### Werte mit Stand

| Wert | Stand | Quelle | Genutzt in |
|---|---|---|---|
| Größenklassen: Kleinst 450.000 € / 900.000 € / 10; klein 7.500.000 € / 15.000.000 € / 50; mittel 25.000.000 € / 50.000.000 € / 250 | 17.10.2026, für Geschäftsjahre nach dem 31.12.2023 | §§ 267, 267a HGB, Art. 93 EGHGB | jahresabschluss/groessenklassen-anhang |
| Aufstellung 3 Monate, kleine bis 6 Monate | 17.10.2026 | § 264 Abs. 1 Satz 3, 4 HGB | jahresabschluss/ablauf, aufstellen-feststellen-offenlegen, groessenklassen-anhang |
| Feststellung 8 Monate, kleine 11 Monate | 17.10.2026 | § 42a Abs. 2 GmbHG | jahresabschluss/ablauf, aufstellen-feststellen-offenlegen, groessenklassen-anhang |
| Offenlegung spätestens ein Jahr nach Stichtag | 17.10.2026 | § 325 Abs. 1a HGB | jahresabschluss/ablauf, aufstellen-feststellen-offenlegen |
| Ordnungsgeld 2.500 bis 25.000 €; Herabsetzung 500 € / 1.000 € / 2.500 €; sechs Wochen | 17.10.2026 | § 335 Abs. 1, 3, 4 HGB | jahresabschluss/aufstellen-feststellen-offenlegen, groessenklassen-anhang |
| Abzinsung HGB: Durchschnitt 7 Jahre (Pensionen 10), Restlaufzeit über ein Jahr; Sätze monatlich von der Bundesbank | 17.10.2026; Sätze ändern sich monatlich | § 253 Abs. 2 HGB, RückAbzinsV | jahresabschluss/rueckstellungen |
| Abzinsung Steuer 5,5 %, ab Laufzeit 12 Monate | 17.10.2026 | § 6 Abs. 1 Nr. 3a Buchst. e EStG | jahresabschluss/rueckstellungen |
| RAP-Wahlrecht steuerlich bis 800 € je Ausgabe/Einnahme | 17.10.2026 | § 5 Abs. 5 Satz 2, § 6 Abs. 2 Satz 1 EStG | jahresabschluss/abgrenzung |
| UG-Rücklage ein Viertel; Mindeststammkapital 25.000 € | 17.10.2026 | § 5a Abs. 3, 5, § 5 Abs. 1 GmbHG | jahresabschluss/ablauf, ergebnisverwendung-ug |
| Sonderabschreibung § 7g bis zu insgesamt 40 %; Gewinngrenze 200.000 € | 17.10.2026 | § 7g Abs. 5, Abs. 6 Nr. 1, Abs. 1 Satz 2 Nr. 1 EStG | jahresabschluss/abschreibungen |
| Degressive AfA bis dreifach linear, höchstens 30 %, Anschaffung 01.07.2025–31.12.2027 | 17.10.2026 | § 7 Abs. 2 EStG | jahresabschluss/abschreibungen |

### Buchfink-eigene Quellen

| Quelle | Wofür |
|---|---|
| `internal/domain/closing_step.go` | Abschlussbausteine und Reihenfolge |
| `internal/service/closing_service.go` (`setFiscalYearStatus`, `ReopenFiscalYear`) | Stände, Datumsregeln, Jahresfestschreibung vor Feststellung, Nachweis bei Offenlegung, Wiedereröffnen auf „aufgestellt“ |
| `internal/service/ug_reserve.go`, `internal/wailsbridge/festschreibung_service.go` (`runPreCommitChecks`) | UG-Rücklage 7765/2930, Aufrundung, Prüfung vor Festschreibung bis Jahresende und vor Statuswechsel |
| `internal/service/closing_booking_service.go` | USt-Verrechnung auf 3841/1420 mit Verweis `USTV <Jahr>`; Steuerrückstellung KSt+SolZ auf 3040, GewSt 3035 |
| `internal/service/journal_service.go` (`mayWriteTaxAccount`), `internal/accounting/tax_skr04.go` (`taxAccounts`), `internal/service/manual_entry.go` | Steuerkonten nur über Steuerautomatik; Handbuchung verlangt Steuerfall |
| `internal/accounting/ustva.go` (`vatMovements`, `RevenueTreatments`) | Abschlussbuchungen (Quelle closing) außerhalb der VA; Erlöskonten mit Steuerfall zählen ohne Schlüssel |
| `internal/accounting/chart.go` | Kontenbereiche wie 1080–1089 sind bebuchbar |
| `internal/accounting/skr04_2026.json` | alle zitierten Konten (geprüft am 17.10.2026) |
| `docs/anforderungen/jahresabschluss.md`, `bewertung.md` | Hinweisgeber (Ist-Stand), nicht Beleg |

### Offen

- Buchfink: Die UG-Rücklage wird schon bei der Festschreibung eines Zeitraums
  bis Jahresende verlangt (`runPreCommitChecks`); mit Abschlussperiode sollte
  die Prüfung an die Jahresfestschreibung wandern.
- Buchfink: `groessenklasse.go` nennt „Art. 79 Abs. 1 EGHGB“; richtig ist
  Art. 93 Abs. 1 EGHGB (Art. 79 betrifft die Abschlussprüfungsreform).
- Buchfink: `asset_service.go` begründet die Ablehnung des
  Sammelposten-Abgangs mit „§ 6 Abs. 2a S. 4 EStG“; nach Wortlaut Satz 3.
- Buchfink: Anhangumfang folgt nicht der Größenklasse; Haftungsverhältnisse als
  Anhangtext statt unter der Bilanz bei Kleinst.
- Buchfink: Kein Format für das Unternehmensregister.
- Konto bei nicht abgerechneter eigener Leistung (Erlös im falschen Jahr)
  nicht festgelegt; Buchungsdatum des Erlöses bei Rechnung im Folgejahr prüfen.

## Bereich steuererklaerungen

Quellen des Bereichs `references/steuererklaerungen/` (ablauf.md und Referenzen
`kst`, `gewst`, `ust-jahr`, `ebilanz`, `fristen-zuschlaege`, `abgabe`,
`bescheide`). Alle 124 Links beider Skills (steuererklaerungen, gruendung) am
17.10.2026 mit HTTP 200 geprüft. „Text“ = Wortlaut am 17.10.2026 gelesen und
Aussage abgeglichen.

### Gesetze

| Norm | Link | Wofür | Geprüft | Genutzt in |
|---|---|---|---|---|
| § 1 KStG | https://www.gesetze-im-internet.de/kstg_1977/__1.html | unbeschränkte Steuerpflicht der GmbH, sämtliche Einkünfte (Abs. 1 Nr. 1, Abs. 2) | Text | steuererklaerungen/kst |
| § 7 KStG | https://www.gesetze-im-internet.de/kstg_1977/__7.html | zu versteuerndes Einkommen (Abs. 1, 2), Jahressteuer, Zeitraum der Steuerpflicht (Abs. 3 Satz 3), Wirtschaftsjahr, Bezug im Kalenderjahr des Endes, Umstellung nur im Einvernehmen (Abs. 4 Satz 1–3) | Text | steuererklaerungen/kst; gruendung/rumpfjahr |
| § 8 KStG | https://www.gesetze-im-internet.de/kstg_1977/__8.html | Einkommen nach EStG und KStG (Abs. 1 Satz 1), alle Einkünfte gewerblich (Abs. 2), vGA, verdeckte Einlage (Abs. 3 Satz 2, 3) | Text | steuererklaerungen/kst |
| § 8b KStG | https://www.gesetze-im-internet.de/kstg_1977/__8b.html | Beteiligungserträge außer Ansatz (Abs. 1, 2), Streubesitz unter 10 % zu Beginn des Kalenderjahres, nur Bezüge nach Abs. 1 (Abs. 4 Satz 1), 5 % nicht abziehbar (Abs. 5); nur Verweis auf beteiligungen | Text (Abs. 1, 2, 4, 5) | steuererklaerungen/kst; beteiligungen/steuerfreistellung |
| § 8c KStG | https://www.gesetze-im-internet.de/kstg_1977/__8c.html | mehr als 50 % in fünf Jahren, Verluste nicht mehr abziehbar (Abs. 1 Satz 1) | Text (Abs. 1 Satz 1–3) | steuererklaerungen/kst |
| § 9 KStG | https://www.gesetze-im-internet.de/kstg_1977/__9.html | Spenden bis 20 % des Einkommens oder 4 ‰ der Umsätze und Löhne (Abs. 1 Satz 1 Nr. 2), Vortrag (Abs. 1 Satz 9), Einkommen vor Spenden und Verlustabzug (Abs. 2 Satz 1) | Text | steuererklaerungen/kst |
| § 10 KStG | https://www.gesetze-im-internet.de/kstg_1977/__10.html | Steuern vom Einkommen, Vorsteuer auf Abzugsverbotsaufwand, Nebenleistungen (Nr. 2), Geldstrafen (Nr. 3), Hälfte Aufsichtsratsvergütung (Nr. 4) | Text | steuererklaerungen/kst |
| § 23 KStG | https://www.gesetze-im-internet.de/kstg_1977/__23.html | 15 % bis VZ 2027, 14 % 2028, 13 % 2029, 12 % 2030, 11 % 2031, 10 % ab 2032 (Abs. 1) | Text | steuererklaerungen/kst |
| § 24 KStG | https://www.gesetze-im-internet.de/kstg_1977/__24.html | Freibetrag 5.000 € nicht für Körperschaften mit Ausschüttungen nach § 20 Abs. 1 Nr. 1 EStG (Satz 2 Nr. 1) | Text | steuererklaerungen/kst |
| § 27 KStG | https://www.gesetze-im-internet.de/kstg_1977/__27.html | Einlagekonto (Abs. 1 Satz 1–3, 5), Feststellung und jährliche Erklärung (Abs. 2 Satz 1, 4) | Text | steuererklaerungen/ablauf, bescheide, kst |
| § 31 KStG | https://www.gesetze-im-internet.de/kstg_1977/__31.html | EStG-Vorschriften für Veranlagung und Anrechnung (Abs. 1 Satz 1), Rundung auf volle Euro (Abs. 1 Satz 2), elektronische Erklärung (Abs. 1a Satz 1), Vorauszahlungen bei abweichendem Wj (Abs. 2) | Text | steuererklaerungen/ablauf, bescheide, kst |
| § 3 SolZG | https://www.gesetze-im-internet.de/solzg_1995/__3.html | Bemessung nach festgesetzter KSt (Abs. 1 Nr. 1), Freigrenze nur für ESt-Pflichtige (Abs. 3) | Text | steuererklaerungen/kst |
| § 4 SolZG | https://www.gesetze-im-internet.de/solzg_1995/__4.html | 5,5 % (Satz 1), Cent-Bruchteile (Satz 3) | Text | steuererklaerungen/kst |
| § 4 EStG | https://www.gesetze-im-internet.de/estg/__4.html | Geschenke, Bewirtung, Gästehaus/Jagd/Jacht, Geldbußen (Abs. 5 Satz 1 Nr. 1–4, 8), GewSt keine Betriebsausgabe (Abs. 5b), Parteispenden (Abs. 6) | Text | steuererklaerungen/kst |
| § 5 EStG | https://www.gesetze-im-internet.de/estg/__5.html | Drohverlustrückstellung steuerlich verboten (Abs. 4a Satz 1) | Text | steuererklaerungen/kst |
| § 5b EStG | https://www.gesetze-im-internet.de/estg/__5b.html | E-Bilanz mit Kontennachweisen, Anlagenspiegel, -verzeichnis (Abs. 1 Satz 1), Überleitung/Steuerbilanz (Satz 2, 3), Eröffnungsbilanz (Satz 4), Anhang u. a. (Satz 5), Härtefall (Abs. 2) | Text | grundlagen/pflichten; steuererklaerungen/ablauf, ebilanz, kst; gruendung/eroeffnungsbilanz |
| § 10d EStG | https://www.gesetze-im-internet.de/estg/__10d.html | Rücktrag 1 Mio. €, zwei Jahre, Verzicht (Abs. 1 Satz 1, 2, 6), Vortrag 1 Mio. € + 70 % (Abs. 2 Satz 1), Feststellung (Abs. 4 Satz 1) | Text | steuererklaerungen/bescheide, kst |
| § 36 EStG | https://www.gesetze-im-internet.de/estg/__36.html | Anrechnung Vorauszahlungen und Abzugsteuer, Bescheinigung nötig (Abs. 2 Nr. 1, 2 Satz 2) | Text | steuererklaerungen/kst |
| § 37 EStG | https://www.gesetze-im-internet.de/estg/__37.html | Vorauszahlungstermine (Abs. 1 Satz 1), Bemessung und Anpassung bis 15. Monat (Abs. 3 Satz 2, 3) | Text | grundlagen/pflichten; steuererklaerungen/bescheide |
| § 52 EStG | https://www.gesetze-im-internet.de/estg/__52.html | § 5b: Kontennachweise ab Wj nach 31.12.2024, Anlagenspiegel/-verzeichnis und Satz 5, 6 ab Wj nach 31.12.2027 (Abs. 11 Satz 2, 3); § 10d Abs. 2 ab VZ 2024 (Abs. 18b Satz 3) | Text (Abs. 11, 18b) | steuererklaerungen/ebilanz, kst |
| § 50 EStDV | https://www.gesetze-im-internet.de/estdv_1955/__50.html | Zuwendungsbestätigung nach amtlichem Vordruck (Abs. 1 Satz 1) | Text | steuererklaerungen/kst |
| § 60 EStDV | https://www.gesetze-im-internet.de/estdv_1955/__60.html | Papierunterlagen bei Härtefall (Abs. 1), Anpassung/Steuerbilanz (Abs. 2) | Text | grundlagen/pflichten; steuererklaerungen/ebilanz, kst |
| § 84 EStDV | https://www.gesetze-im-internet.de/estdv_1955/__84.html | § 60 Abs. 1: Anlagenverzeichnis ab Wj nach 31.12.2027 (Abs. 3f) | Text (Abs. 3e, 3f) | steuererklaerungen/ebilanz |
| § 2 GewStG | https://www.gesetze-im-internet.de/gewstg/__2.html | GmbH stets Gewerbebetrieb (Abs. 2 Satz 1) | Text | steuererklaerungen/gewst |
| § 7 GewStG | https://www.gesetze-im-internet.de/gewstg/__7.html | Gewerbeertrag = Gewinn nach KStG ± §§ 8, 9 (Satz 1) | Text | steuererklaerungen/gewst |
| § 8 GewStG | https://www.gesetze-im-internet.de/gewstg/__8.html | Hinzurechnungen Nr. 1 a–f (Viertel, über 200.000 €), Nr. 5, 8, 9, 10 | Text | steuererklaerungen/ablauf, gewst |
| § 9 GewStG | https://www.gesetze-im-internet.de/gewstg/__9.html | Kürzung Grundsteuer (Nr. 1 Satz 1), erweiterte Kürzung (Nr. 1 Satz 2), Schachtel ab 15 % (Nr. 2a), Spenden 20 %/4 ‰ (Nr. 5) | Text | steuererklaerungen/gewst; beteiligungen/gewst-holding |
| § 10 GewStG | https://www.gesetze-im-internet.de/gewstg/__10.html | abweichendes Wj: Bezug im Erhebungszeitraum des Endes (Abs. 2) | Text | steuererklaerungen/gewst |
| § 10a GewStG | https://www.gesetze-im-internet.de/gewstg/__10a.html | Gewerbeverlust 1 Mio. € + 60 % (Satz 1, 2), Feststellung (Satz 6) | Text | steuererklaerungen/bescheide, gewst |
| § 11 GewStG | https://www.gesetze-im-internet.de/gewstg/__11.html | Abrundung 100 €, Freibeträge nur nat. Personen/PersG 24.500 € bzw. 5.000 € (Abs. 1 Satz 3), Messzahl 3,5 % (Abs. 2) | Text | steuererklaerungen/gewst |
| § 14 GewStG | https://www.gesetze-im-internet.de/gewstg/__14.html | Erhebungszeitraum Kalenderjahr, abgekürzt (Satz 2, 3) | Text | steuererklaerungen/gewst |
| § 14a GewStG | https://www.gesetze-im-internet.de/gewstg/__14a.html | Erklärung und Zerlegungserklärung elektronisch (Satz 1) | Text | grundlagen/pflichten; steuererklaerungen/ablauf, gewst |
| § 16 GewStG | https://www.gesetze-im-internet.de/gewstg/__16.html | Hebesatz der Gemeinde (Abs. 1), mindestens 280 % (Abs. 4 Satz 2, ab EZ 2027) | Text | steuererklaerungen/bescheide, gewst; gruendung/steuerliche-erfassung |
| § 19 GewStG | https://www.gesetze-im-internet.de/gewstg/__19.html | Vorauszahlungen 15.02./05./08./11., Viertel, Anpassung bis 15. Monat (Abs. 1–3) | Text | grundlagen/pflichten; steuererklaerungen/bescheide |
| § 20 GewStG | https://www.gesetze-im-internet.de/gewstg/__20.html | Abschlusszahlung binnen eines Monats, Überzahlung (Abs. 2, 3) | Text | steuererklaerungen/gewst |
| § 28 GewStG | https://www.gesetze-im-internet.de/gewstg/__28.html | Zerlegung bei mehreren Gemeinden (Abs. 1) | Text | steuererklaerungen/gewst |
| § 36 GewStG | https://www.gesetze-im-internet.de/gewstg/__36.html | Anwendung: § 8 Nr. 1 d Satz 2 bis EZ 2030 (Abs. 4), § 9 Nr. 1 Satz 1 ab EZ 2025 (Abs. 4b), § 16 Abs. 4 Satz 2 ab EZ 2027 (Abs. 5b) | Text | steuererklaerungen/gewst |
| § 16 UStG | https://www.gesetze-im-internet.de/ustg_1980/__16.html | Sollversteuerung (Abs. 1 Satz 1), Kalenderjahr (Abs. 1 Satz 2), Teil des Jahres (Abs. 3) | Text | steuererklaerungen/ust-jahr; gruendung/steuerliche-erfassung |
| § 18 UStG | https://www.gesetze-im-internet.de/ustg_1980/__18.html | Voranmeldungszeitraum, 9.000 €/2.000 €, Neugründer, 2021–2026 (Abs. 2 Satz 1–6); Jahreserklärung (Abs. 3 Satz 1, 2); Abschlusszahlung (Abs. 4 Satz 1, 2) | Text | steuererklaerungen/ablauf, ust-jahr; gruendung/steuerliche-erfassung |
| § 44 UStDV | https://www.gesetze-im-internet.de/ustdv_1980/__44.html | § 15a-Berichtigung bis 6.000 € erst in der Jahreserklärung (Abs. 3 Satz 1) | Text | steuererklaerungen/ust-jahr |
| § 48 UStDV | https://www.gesetze-im-internet.de/ustdv_1980/__48.html | Antrag Dauerfristverlängerung (Abs. 1), Sondervorauszahlung in letzter Voranmeldung (Abs. 4 Satz 1) | Text | grundlagen/pflichten; monatsabschluss/dauerfristverlaengerung |
| § 20 AO | https://www.gesetze-im-internet.de/ao_1977/__20.html | Finanzamt der Geschäftsleitung (Abs. 1) | Text | steuererklaerungen/kst; gruendung/gruendungsweg |
| § 108 AO | https://www.gesetze-im-internet.de/ao_1977/__108.html | BGB-Fristberechnung (Abs. 1), Fristende Samstag/Sonntag/Feiertag (Abs. 3) | Text | steuererklaerungen/fristen-zuschlaege; gruendung/rumpfjahr |
| § 109 AO | https://www.gesetze-im-internet.de/ao_1977/__109.html | Fristverlängerung, auch rückwirkend (Abs. 1), Berater-Fälle nur ohne Verschulden (Abs. 2) | Text | grundlagen/pflichten; steuererklaerungen/fristen-zuschlaege |
| § 122 AO | https://www.gesetze-im-internet.de/ao_1977/__122.html | Bekanntgabe Post vierter Tag (Abs. 2 Nr. 1), elektronisch vierter Tag (Abs. 2a) | Text | steuererklaerungen/bescheide |
| § 122a AO | https://www.gesetze-im-internet.de/ao_1977/__122a.html | Bereitstellung zum Abruf, vierter Tag (Abs. 4 Satz 1) | Text | steuererklaerungen/bescheide |
| § 149 AO | https://www.gesetze-im-internet.de/ao_1977/__149.html | sieben Monate (Abs. 2), Berater letzter Februartag (Abs. 3), Vorabanforderung vier Monate (Abs. 4), USt bei Beendigung (Abs. 5) | Text | grundlagen/pflichten; gruendung/rumpfjahr |
| § 150 AO | https://www.gesetze-im-internet.de/ao_1977/__150.html | Verzicht auf Datenfernübertragung bei Unzumutbarkeit, Regelbeispiele finanzieller Aufwand und Kenntnisse (Abs. 8 Satz 1, 2) | Text 17.10.2026 | grundlagen/pflichten; steuererklaerungen/ebilanz |
| § 152 AO | https://www.gesetze-im-internet.de/ao_1977/__152.html | Verspätungszuschlag Ermessen (Abs. 1), zwingend nach 14 Monaten (Abs. 2), Ausnahmen (Abs. 3), 0,25 %/mind. 25 € (Abs. 5 Satz 2), 25 € bei Messbetrag/Feststellung (Abs. 6), höchstens 25.000 € (Abs. 10) | Text | steuererklaerungen/bescheide, fristen-zuschlaege |
| § 153 AO | (Zeile unter grundlagen) | Berichtigung der Jahreserklärung (Abs. 1 Satz 1 Nr. 1) | Text | steuererklaerungen/ust-jahr |
| § 164 AO | https://www.gesetze-im-internet.de/ao_1977/__164.html | Vorbehalt der Nachprüfung, Änderung jederzeit (Abs. 1, 2) | Text | steuererklaerungen/bescheide |
| § 165 AO | https://www.gesetze-im-internet.de/ao_1977/__165.html | Vorläufigkeit, Umfang und Grund anzugeben (Abs. 1 Satz 3) | Text | steuererklaerungen/bescheide |
| § 168 AO | https://www.gesetze-im-internet.de/ao_1977/__168.html | Steueranmeldung = Vorbehaltsfestsetzung, Erstattung erst mit Zustimmung | Text | steuererklaerungen/bescheide, ust-jahr |
| § 173 AO | https://www.gesetze-im-internet.de/ao_1977/__173.html | Änderung wegen neuer Tatsachen (Abs. 1) | Text (Abs. 1) | steuererklaerungen/bescheide |
| § 182 AO | https://www.gesetze-im-internet.de/ao_1977/__182.html | Bindung an Grundlagenbescheide (Abs. 1) | Text (Abs. 1) | steuererklaerungen/bescheide |
| § 184 AO | https://www.gesetze-im-internet.de/ao_1977/__184.html | Steuermessbescheid, § 182 Abs. 1 sinngemäß (Abs. 1) | Text (Abs. 1) | steuererklaerungen/bescheide |
| § 233a AO | https://www.gesetze-im-internet.de/ao_1977/__233a.html | Verzinsung, Zinslauf 15 Monate (Abs. 1, 2 Satz 1) | Text | steuererklaerungen/bescheide, fristen-zuschlaege |
| § 238 AO | https://www.gesetze-im-internet.de/ao_1977/__238.html | volle Monate (Abs. 1 Satz 2), 0,15 % je Monat bei § 233a (Abs. 1a) | Text | steuererklaerungen/fristen-zuschlaege |
| § 351 AO | https://www.gesetze-im-internet.de/ao_1977/__351.html | Einwände nur gegen Grundlagenbescheid (Abs. 2) | Text | steuererklaerungen/bescheide |
| § 355 AO | https://www.gesetze-im-internet.de/ao_1977/__355.html | Einspruchsfrist ein Monat, bei Steueranmeldung ab Eingang (Abs. 1) | Text | steuererklaerungen/bescheide |
| § 361 AO | https://www.gesetze-im-internet.de/ao_1977/__361.html | keine aufschiebende Wirkung (Abs. 1 Satz 1), Aussetzung auf Antrag (Abs. 2 Satz 2) | Text | steuererklaerungen/bescheide |
| Art. 97 § 36 EGAO | https://www.gesetze-im-internet.de/aoeg_1977/art_97__36.html | Übergangsfristen 2020–2024 für §§ 109, 149, 152, 233a AO (Abs. 3) | Text 17.10.2026 (live abgerufen; Tabelle in fristen-zuschlaege.md gegen Abs. 3 Nr. 1–3, 5, 7 geprüft) | grundlagen/pflichten; steuererklaerungen/fristen-zuschlaege |

### Verwaltungsanweisungen und weitere Quellen

| Quelle | Link | Wofür | Geprüft | Genutzt in |
|---|---|---|---|---|
| BMF vom 27.05.2024, IV C 6 - S 2133-b/24/10001 :002, DOK 2024/0473233 (E-Bilanz, Taxonomien 6.8) | https://betriebs-berater.ruw.de/news/media/2/BMF-Schreiben-16701.pdf (Verlagsspiegel; BMF nicht abrufbar) | Taxonomie 6.8 für Wj ab 2025, 2024 nicht beanstandet; Mussfelder und NIL, ab 6.9 je Rechtsform | Text 17.10.2026 | steuererklaerungen/ebilanz |
| BMF vom 10.06.2025, IV C 6 - S 2133-b/00064/002/006, DOK COO.7005.100.3.12167045 (E-Bilanz, Taxonomien 6.9) | https://betriebs-berater.ruw.de/news/media/2/BMF-Schreiben-17367.pdf (Verlagsspiegel) | Taxonomie 6.9 für Wj ab 2026 und Eröffnungsbilanzen nach 31.12.2025, 2025 nicht beanstandet; Kontennachweis-Inhalt; Nichtbeanstandung ohne Kontennachweise; Anlagenspiegel Mussfeld seit Wj 2017 | Text 17.10.2026 | steuererklaerungen/ebilanz; gruendung (Eröffnungsbilanz) |
| BMF vom 08.06.2026, IV C 6 - S 2133-b/00067/002/023 (E-Bilanz, Taxonomien 6.10) | https://www.steuerschroeder.de/blog/?p=79671 (Sekundärquelle) | Taxonomie 6.10 für Wj ab 2027, 2026 nicht beanstandet | Titel über Sekundärquellen (Kanzleiblog, Ecovis, DATEV-Magazin-Titel) | steuererklaerungen/ebilanz |
| H 10.1 KStH „Erstattungs- und Nachzahlungszinsen“ | https://www.steuerschroeder.de/steuergesetze/ksth/10.1 | Verweis auf BFH I R 43/08 und I B 97/11 | Text (Kanzleispiegel); Urteile nicht gelesen | steuererklaerungen/kst |
| UStAE 18.1 Abs. 4 Satz 1, 2 | https://datenbank.nwb.de/Dokument/378652_18___1/ (NWB-Datenbank, Fassung „VZ 2010–2026 (aktuell)“) | Satz 1: Jahreserklärung „in der Regel bis zum 31. Juli“ (§ 149 Abs. 2 AO); Satz 2: gilt „abweichend von § 18 Abs. 3 Satz 2 UStG“ auch bei Beginn im Lauf des Jahres | Text 17.10.2026 (Verlagsdatenbank; amtlicher UStAE und Umsatzsteuer-Handausgabe 503; Kanzleispiegel steuerschroeder.de veraltet, nennt „31. Mai“) | steuererklaerungen/ust-jahr |
| ELSTER, Liste aller Formulare | https://www.elster.de/eportal/formulare-leistungen/alleformulare | KSt 1, GewSt 1 A, GewSt 1 D, Umsatzsteuererklärung, Fragebogen Kapitalgesellschaft; kein Formular für E-Bilanz | Text 17.10.2026 (über WebFetch) | steuererklaerungen/abgabe, ebilanz, gewst, kst, ust-jahr; gruendung/steuerliche-erfassung |
| Haufe, Leitfaden 2025 Vordruck KSt 1 | https://www.haufe.de/id/beitrag/leitfaden-2025-vordruck-kst-1-HI16941392.html | KSt 1 als Mantelbogen mit Einlagekonto, Vordruck GK, Anlagen ZVE, Verluste, Z, Zinsschranke, AE | Teil gelesen (Paywall), Sekundärquelle | steuererklaerungen/kst |
| LfSt Rheinland-Pfalz, Leitfaden Registrierung mit Zertifikatsdatei, Stand 09.2025 | https://lfst.rlp.de/fileadmin/lfst.rlp.de/ELSTER/Klickanleitungen/Leitfaden_Registrierung_mit_Zertifikatsdatei_Stand_09.2025.pdf | Zertifikat „für eine Organisation“ auf Basis der Steuernummer (Abschnitt 2.2); der übrige Ablauf (Aktivierung per Brief) beschreibt nur die Privatperson | Text 17.10.2026 | steuererklaerungen/abgabe |
| Freistaat Sachsen (luft.sachsen.de), Schritt-für-Schritt-Anleitung ELSTER-Organisationszertifikate (für ELiA-Online), Stand 31.03.2025 | https://www.luft.sachsen.de/download/Schritt-fuer-Schritt-Anleitung-ELSTER-Organisationszertifikate-SN-2025.pdf | Organisationszertifikat: Steuernummer der Organisation; Aktivierungs-ID per E-Mail, Aktivierungs-Code per Brief an die zur Steuernummer gespeicherte Anschrift (Abschnitte 1.1, 1.2). Keine Anleitung der Steuerverwaltung; gleicher Ablauf im älteren Faltblatt der Finanzverwaltung NRW (https://www.finanzverwaltung.nrw.de/sites/default/files/asset/document/flyer_elster_registrierung_nrw_barrierefrei_09-07-13.pdf, „ElsterOnline-Portal“, veraltet) | Text 17.10.2026 | steuererklaerungen/abgabe |

### Werte mit Stand

| Wert | Stand | Quelle | Genutzt in |
|---|---|---|---|
| Körperschaftsteuer 15 % bis VZ 2027, 14 % 2028, 13 % 2029, 12 % 2030, 11 % 2031, 10 % ab 2032 | 17.10.2026 | § 23 Abs. 1 KStG | steuererklaerungen/ablauf, kst |
| Solidaritätszuschlag 5,5 % der festgesetzten KSt, ohne Freigrenze für Körperschaften | 17.10.2026 | §§ 3 Abs. 1 Nr. 1, Abs. 3, 4 Satz 1 SolZG | steuererklaerungen/ablauf, kst |
| Steuermesszahl 3,5 %; kein Freibetrag für GmbH (24.500 € nur nat. Personen/PersG) | 17.10.2026 | § 11 Abs. 1 Satz 3, Abs. 2 GewStG | steuererklaerungen/ablauf, gewst |
| Hinzurechnung: ein Viertel der Summe über 200.000 €; Miete beweglich 1/5, unbeweglich 1/2, Lizenzen 1/4; Elektrofahrzeuge zur Hälfte bis EZ 2030 | 17.10.2026 | § 8 Nr. 1 GewStG, § 36 Abs. 4 GewStG | steuererklaerungen/gewst |
| Kürzung um die als Betriebsausgabe erfasste Grundsteuer (ab EZ 2025); Schachtelkürzung ab 15 % zu Beginn des EZ | 17.10.2026 | § 9 Nr. 1 Satz 1, Nr. 2a, § 36 Abs. 4b GewStG | steuererklaerungen/gewst |
| Spenden: 20 % des Einkommens oder 4 ‰ der Umsätze und Löhne (KSt und GewSt) | 17.10.2026 | § 9 Abs. 1 Satz 1 Nr. 2 KStG, § 9 Nr. 5 GewStG | steuererklaerungen/gewst, kst |
| Verlustrücktrag 1 Mio. € in zwei Vorjahre; Verlustvortrag 1 Mio. € + 70 % (ab VZ 2024; ab 2028 laut Sekundärquellen wieder 60 %) | 17.10.2026 | § 10d Abs. 1, 2 EStG, § 52 Abs. 18b Satz 3 EStG | steuererklaerungen/kst |
| Gewerbeverlust 1 Mio. € + 60 % | 17.10.2026 | § 10a Satz 1, 2 GewStG | steuererklaerungen/gewst |
| Mindesthebesatz 280 % ab EZ 2027 | 17.10.2026 | § 16 Abs. 4 Satz 2, § 36 Abs. 5b GewStG | steuererklaerungen/gewst |
| Vorauszahlungen KSt 10.03./10.06./10.09./10.12., GewSt 15.02./15.05./15.08./15.11.; Anpassung bis Ende des 15. Folgemonats | 17.10.2026 | § 37 Abs. 1, 3 EStG, § 31 KStG, § 19 GewStG | steuererklaerungen/bescheide, gewst, kst |
| Abgabefrist sieben Monate; mit Berater letzter Februartag des zweiten Folgejahres; 2024: 31.07.2025/30.04.2026; 2025: 31.07.2026/01.03.2027; 2026: 02.08.2027/29.02.2028 | 17.10.2026 | § 149 Abs. 2, 3 AO, Art. 97 § 36 Abs. 3 EGAO, § 108 Abs. 3 AO | steuererklaerungen/ablauf, fristen-zuschlaege |
| Verspätungszuschlag zwingend nach 14 Monaten (2024: 16, 2023: 17); 0,25 % je Monat, mind. 25 €; 25 € je Monat bei Messbetrag und Feststellungen; höchstens 25.000 € | 17.10.2026 | § 152 Abs. 2, 5, 6, 10 AO, Art. 97 § 36 Abs. 3 Nr. 5 EGAO | steuererklaerungen/ablauf, fristen-zuschlaege |
| Zinsen § 233a: Zinslauf nach 15 Monaten (2024: 17), 0,15 % je vollem Monat | 17.10.2026 | § 233a Abs. 2, § 238 Abs. 1a AO, Art. 97 § 36 Abs. 3 Nr. 7 EGAO | steuererklaerungen/fristen-zuschlaege |
| Bekanntgabe am vierten Tag; Einspruch ein Monat | 17.10.2026 | §§ 122 Abs. 2, 2a, 122a Abs. 4, 355 Abs. 1 AO | steuererklaerungen/ablauf, bescheide |
| USt-Abschlusszahlung einen Monat nach Eingang der Jahreserklärung | 17.10.2026 | § 18 Abs. 4 Satz 1 UStG | steuererklaerungen/ust-jahr |
| § 15a-Berichtigung bis 6.000 € je Wirtschaftsgut und Jahr in der Jahreserklärung | 17.10.2026 | § 44 Abs. 3 UStDV | steuererklaerungen/ust-jahr |
| E-Bilanz: Kontennachweise ab Wj nach 31.12.2024; Anlagenspiegel/-verzeichnis, Anhang, Lagebericht ab Wj nach 31.12.2027; Taxonomie 6.8 (Wj 2025), 6.9 (Wj 2026), 6.10 (Wj 2027, Sekundärquelle) | 17.10.2026; Taxonomie wechselt jährlich | § 52 Abs. 11 EStG, BMF 27.05.2024, 10.06.2025, 08.06.2026 | steuererklaerungen/ebilanz |

### Buchfink-eigene Quellen

| Quelle | Pfad | Wofür |
|---|---|---|
| Steuerschätzung | `internal/accounting/closing_tax.go` | Ergebnis + nicht abziehbare Aufwendungen, § 8b Abs. 1, 2, 5, Spendenabzug in den Höchstbeträgen, Verlustvortrag mit Mindestbesteuerung, KSt, SolZ, GewSt mit Hebesatz; Hinzurechnungen, Kürzungen, vGA und Verlustvorträge als Eingabe |
| Vorschlag Steuerrückstellung | `internal/service/closing_tax_provision.go` | Bezüge auf 7005, 7006, 7015, 7104 steuerfrei, auf 7014, 7103 steuerpflichtig; Eingaben `verlustvortrag_kst`, `verlustvortrag_gewst`, `korrekturen`, `hinzurechnungen`, `kuerzungen`; Konten 7600, 7608, 7610, 3040, 3035 |
| Steuersätze | `internal/accounting/tax_params.go` (`corporateTaxRates`) | KSt 15 % bis 2027, dann 14 % bis 10 % ab 2032; SolZ 5,5 %; Messzahl 3,5 %; Verlustabzug über 1 Mio. € 70 %, ab 2028 60 % |
| Erklärungen | `internal/declaration/kst.go`, `gewst.go`, `catalog/*.json` | Zeilen mit `status`, `zustaendig`, `herkunft`; § 8b aus Konten und Anlagenkartei, Spenden, Verlustabzug, Einlagekonto, § 8 Nr. 1, 5, 9, § 9 Nr. 1, 2a, 5 |
| Nicht abziehbare Kategorien | `internal/accounting/non_deductible.go` | Geschenke, Bewirtung, Repräsentation |
| E-Bilanz | `internal/ebilanz/skr04.go`, `checks.go`, `taxonomy/de-6.9.json`, `taxonomy/de-6.10.json` | Zuordnung gegen die amtliche HGB-Taxonomie (XBRL Deutschland), Prüfung vor dem Export, Kontennachweis |
| Pflichtenkalender | `internal/service/obligation_catalog.go`, `internal/calendar/holidays.go` | `fristen`: Jahreserklärungen mit und ohne Berater; Vorauszahlungen ab erfasstem Bescheid oder gebuchter Zahlung; Feiertage je Bundesland |
| SKR04 | `internal/accounting/skr04_2026.json` | 3035, 3040, 6390–6398, 6437, 6620–6622, 6644, 6645, 6875, 7105–7107, 7302, 7304, 7308, 7600–7643 |
| Werkzeugkatalog | `docs/entwicklung/mcp.md` | `erklaerung`, `erklaerung_exportieren`, Formate, Transferticket nur App |

### Offen

- **E-Bilanz-Übermittlung:** Mein ELSTER führt kein E-Bilanz-Formular
  (Formularliste 17.10.2026). Buchfink erzeugt die XBRL-Datei; gesendet wird
  über einen Online-Dienst mit XBRL-Import (eBilanz+, eBilanz-Online). Ob die
  Dienste Kontennachweis, Stammdaten und Überleitung beim Import vollständig
  übernehmen, dokumentiert keiner; ein Probeimport im kostenlosen Konto klärt
  es. Rückfallweg: Summen- und Saldenliste mit Elementnamen der Taxonomie
  (`ebilanz_exportieren`, `format: susa_csv`).
- Taxonomie 6.10 (BMF 08.06.2026) nur über Sekundärquellen; esteuer.de
  antwortet mit 503.
- § 10d Abs. 2 EStG: Rückkehr auf 60 % ab VZ 2028 nur Sekundärquellen; die
  künftige Fassung steht noch nicht auf gesetze-im-internet.de.
- § 16 Abs. 4 Satz 2 GewStG: Vorfassung (bis EZ 2026, laut Sekundärquellen
  200 %) nicht am Wortlaut gelesen; im Skill nicht genannt.
- KSt-Anlagen nur über Haufe-Leitfaden (Paywall); im Formular des Jahres
  prüfen.
- Erstattungszinsen auf KSt/GewSt: BFH I R 43/08, I B 97/11 nicht gelesen;
  im Skill Wertungsfrage.
- **UStAE-Spiegel steuerschroeder.de ist teils veraltet:** UStAE 18.1 Abs. 4
  nennt „31. Mai“, UStAE 19.1 die Grenze 17.500 €. ust-jahr.md zitiert 18.1
  deshalb über die NWB-Datenbank. Andere Skills, die UStAE über diesen Spiegel
  zitieren, sollten ihre Abschnitte gegen den amtlichen UStAE prüfen.
- Jahreserklärung USt im Gründungsjahr: § 18 Abs. 3 Satz 2 UStG verlangt nach
  dem Wortlaut einen Monat nach Ablauf des kürzeren Zeitraums (§ 16 Abs. 3);
  UStAE 18.1 Abs. 4 Satz 2 lässt abweichend die Regelfrist gelten. Im Skill
  als Hinweis an den Menschen.
- § 108 Abs. 3 AO auf die Monatsgrenzen des § 152 Abs. 2 AO: nicht geprüft
  (AEAO zu § 152 auf ao.bundesfinanzministerium.de 503). Skill rechnet mit dem
  Kalenderdatum und rät zur Abgabe davor.
- Feiertage je Land für § 108 Abs. 3 AO stehen im Code
  (`internal/calendar/holidays.go`); gegen die Feiertagsgesetze der Länder
  nicht geprüft.

## Bereich gruendung

Quellen des Bereichs `references/gruendung/` (ablauf.md und Referenzen
`gruendungsweg`, `kapitalaufbringung`, `gruendungskosten`,
`eroeffnungsbilanz`, `steuerliche-erfassung`, `rumpfjahr`). Links am
17.10.2026 mit HTTP 200 geprüft. Normen, die schon unter
Bereich steuererklaerungen stehen (§§ 7 KStG, 5b, 16, 18 UStG, §§ 20, 108,
149 AO, § 48 UStDV, § 14, § 16 GewStG), sind dort mit „gruendung“ vermerkt.

### Gesetze

| Norm | Link | Wofür | Geprüft | Genutzt in |
|---|---|---|---|---|
| § 2 GmbHG | https://www.gesetze-im-internet.de/gmbhg/__2.html | notarielle Form, Unterschrift aller (Abs. 1), Musterprotokoll bis drei Gesellschafter und ein GF, gilt als Gesellschafterliste (Abs. 1a), Vollmacht (Abs. 2), Video (Abs. 3) | Text | gruendung/ablauf, gruendungsweg |
| § 3 GmbHG | https://www.gesetze-im-internet.de/gmbhg/__3.html | Pflichtinhalt Gesellschaftsvertrag (Abs. 1 Nr. 1–4) | Text | gruendung/gruendungsweg |
| § 4 GmbHG | https://www.gesetze-im-internet.de/gmbhg/__4.html | Firmenzusatz | Text | gruendung/gruendungsweg |
| § 5 GmbHG | https://www.gesetze-im-internet.de/gmbhg/__5.html | 25.000 € (Abs. 1), volle Euro (Abs. 2), Summe (Abs. 3), Sacheinlage und Sachgründungsbericht (Abs. 4) | Text | gruendung/gruendungsweg, kapitalaufbringung |
| § 5a GmbHG | https://www.gesetze-im-internet.de/gmbhg/__5a.html | UG-Firma (Abs. 1), Volleinzahlung, keine Sacheinlage (Abs. 2), Rücklage ein Viertel (Abs. 3), Ende bei 25.000 € (Abs. 5) | Text | gruendung/gruendungsweg, kapitalaufbringung |
| § 6 GmbHG | https://www.gesetze-im-internet.de/gmbhg/__6.html | Geschäftsführer (Abs. 1–3) | Text | gruendung/gruendungsweg |
| § 7 GmbHG | https://www.gesetze-im-internet.de/gmbhg/__7.html | Anmeldung (Abs. 1), Viertel und 12.500 € (Abs. 2), Sacheinlagen vorab (Abs. 3) | Text | gruendung/ablauf, gruendungsweg, kapitalaufbringung |
| § 8 GmbHG | https://www.gesetze-im-internet.de/gmbhg/__8.html | Anlagen (Abs. 1), Versicherung freie Verfügung, Einzahlungsbelege (Abs. 2), Geschäftsanschrift (Abs. 4) | Text | gruendung/ablauf, gruendungsweg, kapitalaufbringung |
| § 9 GmbHG | https://www.gesetze-im-internet.de/gmbhg/__9.html | Differenzhaftung, Verjährung zehn Jahre ab Eintragung | Text | gruendung/kapitalaufbringung |
| § 9c GmbHG | https://www.gesetze-im-internet.de/gmbhg/__9c.html | Ablehnung der Eintragung (Abs. 1) | Text | gruendung/gruendungsweg |
| § 10 GmbHG | https://www.gesetze-im-internet.de/gmbhg/__10.html | Inhalt der Eintragung (Abs. 1) | Text | gruendung/gruendungsweg |
| § 11 GmbHG | https://www.gesetze-im-internet.de/gmbhg/__11.html | GmbH besteht vor Eintragung nicht (Abs. 1), Handelndenhaftung (Abs. 2) | Text | gruendung/ablauf, gruendungsweg |
| § 13 GmbHG | (Zeile unter grundlagen) | juristische Person, Haftung (Abs. 1, 2), Handelsgesellschaft (Abs. 3) | Text | gruendung/ablauf, eroeffnungsbilanz |
| § 19 GmbHG | https://www.gesetze-im-internet.de/gmbhg/__19.html | Verhältnis (Abs. 1), keine Befreiung (Abs. 2), verdeckte Sacheinlage (Abs. 4), Hin- und Herzahlen (Abs. 5), Verjährung (Abs. 6) | Text | gruendung/kapitalaufbringung |
| § 35a GmbHG | https://www.gesetze-im-internet.de/gmbhg/__35a.html | Angaben auf Geschäftsbriefen (Abs. 1) | Text (Abs. 1) | gruendung/gruendungsweg |
| § 40 GmbHG | https://www.gesetze-im-internet.de/gmbhg/__40.html | Gesellschafterliste bei Änderungen (Abs. 1) | Text (Abs. 1) | grundlagen/pflichten; gruendung/gruendungsweg |
| § 42a GmbHG | (Zeile unter grundlagen) | Feststellung 8/11 Monate (Abs. 2 Satz 1) | Text | gruendung/rumpfjahr |
| § 46 GmbHG | (Zeile unter gesellschafter) | Einforderung der Einlagen (Nr. 2) | Text | gruendung/kapitalaufbringung |
| Anlage 1 GmbHG (Musterprotokoll) | https://www.gesetze-im-internet.de/normengrafiken/bgbl1_2021/j3364_0010.pdf (Text nur im PDF; die HTML-Seite https://www.gesetze-im-internet.de/gmbhg/anlage_1.html verweist nur darauf; Skills verlinken das PDF) | Einlage sofort voll oder 50 %, UG nur voll (Nr. 3, Hinweis 3); Gründungskosten bis 300 €, höchstens Stammkapital (Nr. 5) | Text (PDF) 17.10.2026 | gruendung/ablauf, gruendungskosten, kapitalaufbringung |
| § 26 AktG | https://www.gesetze-im-internet.de/aktg/__26.html | Gründungsaufwand in der Satzung festzusetzen, sonst unwirksam (Abs. 2, 3); für die GmbH entsprechend (BGH II ZB 10/88, BFH I R 12/87) | Text | gruendung/gruendungskosten |
| § 6 HGB | (Zeile unter grundlagen) | Handelsgesellschaften wie Kaufleute (Abs. 1) | Text | gruendung/eroeffnungsbilanz |
| § 12 HGB | https://www.gesetze-im-internet.de/hgb/__12.html | Anmeldung elektronisch, öffentlich beglaubigt (Abs. 1) | Text | grundlagen/pflichten; gruendung/gruendungsweg |
| § 240 HGB | (Zeile unter grundlagen) | Inventar zu Beginn (Abs. 1), höchstens zwölf Monate (Abs. 2) | Text | gruendung/eroeffnungsbilanz, rumpfjahr |
| § 242 HGB | (Zeile unter grundlagen) | Eröffnungsbilanz, Bilanzvorschriften entsprechend (Abs. 1) | Text | gruendung/eroeffnungsbilanz, rumpfjahr |
| § 245 HGB | (Zeile unter grundlagen) | Unterzeichnung mit Datum | Text | gruendung/eroeffnungsbilanz |
| § 248 HGB | https://www.gesetze-im-internet.de/hgb/__248.html | kein Aktivposten für Gründung und Eigenkapitalbeschaffung (Abs. 1 Nr. 1, 2) | Text | gruendung/gruendungskosten |
| § 257 HGB | (Zeile unter grundlagen) | Eröffnungsbilanz nicht nur als Wiedergabe (Abs. 3) | Text | gruendung/eroeffnungsbilanz |
| § 264 HGB | (Zeile unter grundlagen) | Aufstellung drei bzw. sechs Monate (Abs. 1 Satz 3, 4) | Text | gruendung/rumpfjahr |
| § 267 HGB | (Zeile unter grundlagen) | Neugründung: erster Stichtag entscheidet (Abs. 4 Satz 2) | Text | gruendung/rumpfjahr |
| § 272 HGB | (Zeile unter gesellschafter) | Ausweis gezeichnetes Kapital und ausstehende Einlagen (Abs. 1) | Text | gruendung/ablauf, eroeffnungsbilanz, kapitalaufbringung |
| § 325 HGB | (Zeile unter grundlagen) | Offenlegung binnen eines Jahres (Abs. 1a Satz 1) | Text | gruendung/rumpfjahr |
| § 4a EStG | (Zeile unter grundlagen) | Wirtschaftsjahr, Umstellung nur im Einvernehmen (Abs. 1 Satz 2 Nr. 2) | Text | gruendung/rumpfjahr, steuerliche-erfassung |
| § 7 EStG | (Zeile unter belege) | Zwölftel im Anschaffungsjahr (Abs. 1 Satz 4) | Text | gruendung/rumpfjahr |
| § 8b EStDV | https://www.gesetze-im-internet.de/estdv_1955/__8b.html | Wirtschaftsjahr zwölf Monate, kürzer bei Eröffnung (Satz 1, 2 Nr. 1) | Text | gruendung/rumpfjahr |
| § 54 EStDV | https://www.gesetze-im-internet.de/estdv_1955/__54.html | Notar sendet Abschrift binnen zwei Wochen an das Finanzamt (Abs. 1, 2) | Text | gruendung/gruendungsweg |
| § 137 AO | https://www.gesetze-im-internet.de/ao_1977/__137.html | Anzeige der Gründung an Finanzamt und Gemeinde binnen eines Monats | Text | grundlagen/pflichten; gruendung/gruendungsweg, rumpfjahr |
| § 138 AO | https://www.gesetze-im-internet.de/ao_1977/__138.html | Gewerbe an Gemeinde, Unterrichtung des Finanzamts (Abs. 1 Satz 1), Auskünfte elektronisch, auch für § 137 (Abs. 1b), ein Monat (Abs. 4) | Text | grundlagen/pflichten; gruendung/gruendungsweg, rumpfjahr; beteiligungen/erwerb-ausweis |
| § 147 AO | (Zeile unter grundlagen) | Eröffnungsbilanz zehn Jahre, nicht als Wiedergabe (Abs. 1 Nr. 1, Abs. 2, 3) | Text | gruendung/eroeffnungsbilanz |
| § 14 UStG | (Zeile unter belege) | Name/Anschrift, Steuernummer oder USt-IdNr. (Abs. 4 Satz 1 Nr. 1, 2) | Text | gruendung/ablauf, gruendungskosten, steuerliche-erfassung |
| § 19 UStG | (Zeile unter belege) | Kleinunternehmer 25.000/100.000 € (Abs. 1 Satz 1), Verzicht unwiderruflich, mind. fünf Jahre (Abs. 3) | Text | gruendung/steuerliche-erfassung |
| § 20 UStG | (Zeile unter grundlagen) | Ist-Versteuerung bis 800.000 € Vorjahresumsatz (Satz 1 Nr. 1) | Text | gruendung/steuerliche-erfassung |
| § 27a UStG | https://www.gesetze-im-internet.de/ustg_1980/__27a.html | USt-IdNr. durch BZSt auf Antrag, Angaben (Abs. 1 Satz 1, 4, 5) | Text | gruendung/steuerliche-erfassung |
| § 46 UStDV | (Zeile unter monatsabschluss) | Dauerfristverlängerung auf Antrag | Text | gruendung/steuerliche-erfassung |
| § 14 GewO | https://www.gesetze-im-internet.de/gewo/__14.html | Gewerbeanzeige gleichzeitig mit dem Beginn (Abs. 1 Satz 1) | Text | grundlagen/pflichten; gruendung/gruendungsweg |
| § 3 GwG | https://www.gesetze-im-internet.de/gwg_2017/__3.html | wirtschaftlich Berechtigter über 25 % (Abs. 2 Satz 1), fiktiv gesetzlicher Vertreter (Abs. 2 Satz 5) | Text | grundlagen/pflichten; gruendung/gruendungsweg |
| § 20 GwG | https://www.gesetze-im-internet.de/gwg_2017/__20.html | Mitteilung an das Transparenzregister unverzüglich (Abs. 1 Satz 1) | Text | grundlagen/pflichten; gruendung/gruendungsweg |
| § 192 SGB VII | https://www.gesetze-im-internet.de/sgb_7/__192.html | Meldung an Unfallversicherungsträger binnen einer Woche, Gewerbeanzeige genügt (Abs. 1) | Text | grundlagen/pflichten; gruendung/gruendungsweg |
| § 18i SGB IV | https://www.gesetze-im-internet.de/sgb_4/__18i.html | Betriebsnummer beantragen (Abs. 1) | Text | gruendung/gruendungsweg |
| § 2 IHKG | https://www.gesetze-im-internet.de/ihkg/__2.html | Kammerzugehörigkeit bei GewSt-Veranlagung und Betriebsstätte (Abs. 1) | Text | gruendung/gruendungsweg |

### Verwaltungsanweisungen und weitere Quellen

| Quelle | Link | Wofür | Geprüft | Genutzt in |
|---|---|---|---|---|
| H 1.1 KStH „Beginn der Steuerpflicht“, „Vorgründungsgesellschaft“ | https://www.steuerschroeder.de/steuergesetze/ksth/1.1 | Steuerpflicht ab Vorgesellschaft (BFH I R 98-99/86, I R 17/92); Vorgründungsgesellschaft nicht identisch, i. d. R. nicht kstpfl. (BFH I R 174/86) | Text (Kanzleispiegel) | steuererklaerungen/kst; gruendung/ablauf, gruendungskosten |
| BFH vom 11.10.1989, I R 12/87, BStBl II 1990, 89 | https://www.haufe.de/id/entscheidung/bfh-urteil-vom-11101989-i-r-1287-HI62976.html | Gründungskosten, die die GmbH ohne Festsetzung im Gesellschaftsvertrag trägt, sind keine Betriebsausgabe (Ausschüttung); Lücke des GmbHG durch entsprechende Anwendung von § 26 Abs. 2 AktG geschlossen, mit Verweis auf BGH II ZB 10/88 | Text 17.10.2026 (Haufe-Wiedergabe) | gruendung/gruendungskosten |
| UStAE 18.7 Abs. 5 (Amtliche Umsatzsteuer-Handausgabe 2023) | https://usth.bundesfinanzministerium.de/usth/2023/A-Umsatzsteuergesetz/V-Besteuerung/Paragraf-18/ae-18-7.html | Neugründungen 2021–2026: Zeitraum nach § 18 Abs. 2 Sätze 1 und 2 UStG, keine Befreiung (Satz 3), Gründungsjahr nach voraussichtlicher Steuer, Folgejahr nach umgerechneter Steuer (Satz 4), Vorratsgesellschaften und Firmenmäntel unberührt (Satz 8) | Text 19.10.2026 | gruendung/steuerliche-erfassung |
| BGH vom 20.02.1989, II ZB 10/88, GmbHR 1989, 250 | – (Volltext nicht gelesen) | § 26 Abs. 2 AktG entsprechend für die GmbH | nur über BFH I R 12/87 und Sekundärquellen (Haufe „Gründungsaufwand bei UG und GmbH“); Fundstelle BGHZ 107, 1 nicht bestätigt | gruendung/gruendungskosten |
| H 8.5 KStH „Gründungskosten“ → BMF vom 25.06.1991, IV B 7 - S 2741 - 4/91, BStBl I S. 661 | https://www.steuerschroeder.de/steuergesetze/ksth/8.5 | Verweis der Verwaltung; Inhalt (vGA bei Übernahme ohne Satzungsregel) nur über Sekundärquellen (NWB-Vorschau, Suchergebnisse zu BFH I R 42/96) | Titel; Wortlaut des BMF-Schreibens nicht gelesen | gruendung/gruendungskosten |
| BGH vom 09.03.1981, II ZR 54/80, BGHZ 80, 129 (Vorbelastungs-/Unterbilanzhaftung) | https://www.haufe.de/id/entscheidung/bgh-urteil-vom-09031981-ii-zr-5480-veroeffentlicht-am-09031981-HI609498.html | Grundlage der Unterbilanzhaftung (laut späterer Rechtsprechung, II ZR 123/94, II ZR 115/81) | Titel über Sekundärquellen | gruendung/kapitalaufbringung |
| Finanzverwaltung NRW, Merkblatt für Existenzgründerinnen und Existenzgründer (ELSTER-Registrierung) | https://www.finanzverwaltung.nrw.de/sites/default/files/asset/document/merkblatt_fuer_existenzgruenderinnen_und_existenzgruender.pdf | Fragebogen elektronisch seit 01.01.2021; Registrierung nur mit E-Mail für Fragebögen; Umwandlung nach Steuernummer; bis zu zwei Wochen | Text 17.10.2026 (ohne Datum) | gruendung/steuerliche-erfassung |
| ELSTER, Liste aller Formulare | (Zeile oben) | „Fragebogen zur steuerlichen Erfassung: Gründung einer Kapitalgesellschaft beziehungsweise Genossenschaft“ | Text 17.10.2026 | gruendung/steuerliche-erfassung |

### Werte mit Stand

| Wert | Stand | Quelle | Genutzt in |
|---|---|---|---|
| Stammkapital GmbH mindestens 25.000 €; UG darunter, Anteil mindestens 1 € | 17.10.2026 | § 5 Abs. 1, 2, § 5a Abs. 1 GmbHG | gruendung/kapitalaufbringung |
| Vor Anmeldung: ein Viertel je Geldanteil, insgesamt 12.500 €; UG voll | 17.10.2026 | § 7 Abs. 2, § 5a Abs. 2 GmbHG | gruendung/ablauf, kapitalaufbringung |
| Musterprotokoll: höchstens drei Gesellschafter, ein GF; Gründungskosten bis 300 €, höchstens Stammkapital; GmbH 50 % oder voll sofort | 17.10.2026 | § 2 Abs. 1a, Anlage 1 GmbHG | gruendung/ablauf, gruendungskosten, gruendungsweg, kapitalaufbringung |
| Differenzhaftung und Einlageanspruch verjähren in zehn Jahren | 17.10.2026 | § 9 Abs. 2, § 19 Abs. 6 GmbHG | gruendung/kapitalaufbringung |
| Fragebogen und Anzeige nach § 137 AO binnen eines Monats; Notar binnen zwei Wochen | 17.10.2026 | §§ 137 Abs. 2, 138 Abs. 4 AO, § 54 Abs. 2 EStDV | gruendung/gruendungsweg, rumpfjahr |
| Gewerbeanzeige gleichzeitig; Berufsgenossenschaft binnen einer Woche; Transparenzregister unverzüglich; wirtschaftlich Berechtigter über 25 % | 17.10.2026 | § 14 GewO, § 192 SGB VII, §§ 3, 20 GwG | gruendung/gruendungsweg |
| Voranmeldung: Vorjahressteuer über 9.000 € monatlich, bis 2.000 € Befreiung möglich; Neugründer monatlich im Gründungs- und Folgejahr; 2021–2026 voraussichtliche Steuer | 17.10.2026 | § 18 Abs. 2 Satz 1–6 UStG | gruendung/ablauf, steuerliche-erfassung |
| Ist-Versteuerung auf Antrag bis 800.000 € Vorjahresumsatz | 17.10.2026 | § 20 Satz 1 Nr. 1 UStG | gruendung/steuerliche-erfassung |
| Kleinunternehmer 25.000 € Vorjahr / 100.000 € laufend; Verzicht bindet fünf Jahre | 17.10.2026 | § 19 Abs. 1, 3 UStG | gruendung/steuerliche-erfassung |
| Aufstellung drei Monate (klein sechs), Feststellung acht (klein elf), Offenlegung zwölf Monate | 17.10.2026 | § 264 Abs. 1 HGB, § 42a Abs. 2 GmbHG, § 325 Abs. 1a HGB | gruendung/rumpfjahr |
| Eröffnungsbilanz zehn Jahre aufbewahren, im Original | 17.10.2026 | § 147 Abs. 1 Nr. 1, Abs. 2, 3 AO, § 257 Abs. 3 HGB | gruendung/eroeffnungsbilanz |

### Buchfink-eigene Quellen

| Quelle | Pfad | Wofür |
|---|---|---|
| Gründungsregeln | `internal/accounting/gruendung.go` | `FoundationRules`, `RequiredPaidIn` (Viertel, 12.500 €, Sacheinlage voll), `RecommendedVatPeriod` |
| Gründungsdienst | `internal/service/foundation_service.go`, `internal/domain/foundation.go` | Unterbilanz, Gründungsbuchungen (Bridge ohne Oberfläche), `FoundationCostCap` |
| Eröffnungsbilanz Gründung und Umsteiger | `internal/service/foundation_opening.go`, `internal/service/opening_balance.go` | Stichtag Beurkundung; Umsteiger gegen 9000, 9008, 9009 mit Schlussbilanz als Beleg |
| E-Bilanz der Eröffnungsbilanz | `internal/service/ebilanz_service.go` (`EBilanzOptions.Kind`), `internal/mcpserver/erklaerung.go` | `ebilanz_exportieren` mit `bilanzart: eroeffnungsbilanz`: Stichtag Gründungsdatum der Stammdaten, nur im Geschäftsjahr der Gründung und mit Buchungen bis zu diesem Tag |
| Firmenzusatz | `internal/domain/settings.go` (`FirmName`) | „i. G.“ bis zur Eintragung |
| Einrichtungsassistent | `frontend/src/components/SetupAssistantScreen.tsx` | Erfassung der Gründungsangaben (nur App) |
| Buchungsgruppen | `internal/accounting/posting_groups.go` | `beratung` 6825, `beitraege` 6420 |
| SKR04 | `internal/accounting/skr04_2026.json` | 1298, 1307, 1800, 2900, 2910, 6420, 6825, 9000, 9008, 9009 |
| Fachkonzept | `docs/fachkonzepte/gruendung.md` | Hinweisgeber, teils veraltet (siehe Offen) |

### Offen

- **Gründungsbuchungen:** Zeichnung und Einzahlung bucht der Agent nach
  Rückfrage mit `buchung_erfassen` bzw. `bankumsatz_buchen`; ein eigenes
  Werkzeug gibt es nicht.
- **Unterbilanz:** `FoundationService.Unterbilanz` rechnet, aber weder App
  noch Werkzeug zeigen das Ergebnis.
- **Voranmeldungszeitraum der Gründung:** `RecommendedVatPeriod` kennt die
  voraussichtliche Steuer des Gründungsjahres nicht und erkennt
  Vorratsgesellschaften (§ 18 Abs. 2 Satz 5 UStG) nicht. Die Umrechnung des
  Gründungsjahres in eine Jahressteuer (Satz 6) regelt der Wortlaut nicht;
  Buchfink rechnet mit vollen Monaten ab dem Gründungsmonat.
- **Kleinunternehmer:** Seit der Neufassung ist eine GmbH unter den Grenzen
  des § 19 Abs. 1 UStG ohne Verzicht Kleinunternehmerin; die Anwendung im
  Gründungsjahr (aktueller UStAE 19.1) ist nicht gelesen. Buchfink bildet
  Kleinunternehmer nicht ab; das Onboarding sollte darauf hinweisen.
- **Fachkonzept falsch:** `docs/fachkonzepte/gruendung.md` nennt für die
  Gewerbeanmeldung „einen Monat“; § 14 Abs. 1 GewO verlangt die Anzeige
  „gleichzeitig“.
- Anker der Monatsfrist (§§ 137, 138 AO): Skill rechnet ab Beurkundung wie das
  Fachkonzept; nicht durch Verwaltungsanweisung belegt.
- Unterbilanzhaftung nur über Sekundärquellen; Gründungskosten-vGA über BFH I R 12/87 (Haufe-Wiedergabe), Volltext des BGH-Beschlusses II ZB 10/88 nicht gelesen.
- Felder des Fragebogens zur steuerlichen Erfassung nicht am aktuellen Formular
  geprüft.
- Vorsteuer aus Gründungs- und Vorgründungskosten: im Skill Wertungsfrage, keine
  Regel.

## Bereich beteiligungen

Quellen des Bereichs `references/beteiligungen/` (ablauf.md und Referenzen
`erwerb-ausweis`, `dividenden`, `steuerfreistellung`, `gewst-holding`,
`bewertung-veraeusserung`, `investmentfonds`, `holding-umsatzsteuer`). Alle 56
Links am 17.10.2026 mit HTTP 200 geprüft. Normen, die schon unter einem
anderen Skill stehen, sind mit „Zeile unter …“ vermerkt; die Spalte „Wofür“
nennt nur die hier genutzten Stellen.

### Gesetze

| Norm | Link | Wofür | Geprüft | Genutzt in |
|---|---|---|---|---|
| § 8b KStG | (Zeile unter steuererklaerungen) | Abs. 1 Satz 1, 2 (Bezüge, Korrespondenz vGA); Abs. 2 Satz 1–4, 6 (Veräußerung, Gewinnbegriff, Wertaufholung, frühere TWA, verdeckte Einlage); Abs. 3 Satz 1–4, 6–9 (5 %, § 3c Abs. 1 nicht anzuwenden, Gewinnminderungen, Darlehen > 25 %, Fremdvergleich, Wertaufholung Darlehen); Abs. 4 Satz 1, 4–7 (Streubesitz < 10 % zu Beginn des Kalenderjahres, Mitunternehmerschaft, Erwerb ≥ 10 % rückbezogen, kein Abs. 5); Abs. 5 (5 %); Abs. 6–8 (Mitunternehmerschaft, Institute, Versicherer) | Text 17.10.2026 | beteiligungen/ablauf, bewertung-veraeusserung, dividenden, erwerb-ausweis, gewst-holding, steuerfreistellung |
| § 8c KStG | (Zeile unter 1. Gesetze) | Verlustuntergang bei der Tochter, mehr als 50 % in fünf Jahren (Abs. 1 Satz 1) | Text 17.10.2026 | beteiligungen/ablauf, erwerb-ausweis |
| § 10 KStG | (Zeile unter belege) | Nr. 2: Steuern vom Einkommen nicht abziehbar (Kapitalertragsteuer, Solidaritätszuschlag auf 7630, 7633) | Text 17.10.2026 | beteiligungen/ablauf, steuerfreistellung |
| § 27 KStG | (Zeile unter gesellschafter) | Abs. 3: Bescheinigung der Einlagenrückgewähr | Text 17.10.2026 | beteiligungen/dividenden |
| § 31 KStG | (Zeile unter belege) | Abs. 1 Satz 1: EStG für Anrechnung entsprechend | Text 17.10.2026 | beteiligungen/ablauf, dividenden |
| § 7 GewStG | (Zeile unter steuererklaerungen) | Satz 1: Gewerbeertrag = Gewinn nach KStG ± §§ 8, 9 | Text 17.10.2026 | beteiligungen/ablauf, bewertung-veraeusserung, gewst-holding |
| § 8 GewStG | (Zeile unter bank) | Einleitungssatz (nur Abgesetztes); Nr. 1 Buchst. a (Entgelte für Schulden, 200.000 €); Nr. 5 (Streubesitzdividenden nach Abzug der 5 %); Nr. 10 (ausschüttungsbedingte Gewinnminderungen); Nr. 12 (ausländische Steuern) | Text 17.10.2026 | beteiligungen/ablauf, gewst-holding |
| § 9 GewStG | (Zeile unter steuererklaerungen) | Nr. 2a Satz 1–4 (Kürzung ab 15 % zu Beginn des Erhebungszeitraums, unmittelbare Aufwendungen, 5 % keine Gewinne aus Anteilen); Nr. 7 (Ausland ab 15 %); Nr. 8 (DBA-Mindestbeteiligung) | Text 17.10.2026 | beteiligungen/ablauf, gewst-holding |
| § 11 GewStG | (Zeile unter steuererklaerungen) | Abs. 2: Messzahl 3,5 % (Beispiel) | Text 17.10.2026 | beteiligungen/gewst-holding |
| § 6 EStG | (Zeile unter belege) | Abs. 1 Nr. 1 Satz 3, 4 (Teilwert, Nachweis in Folgejahren); Nr. 2 Satz 1–3 (Beteiligungen, Teilwert bei dauernder Minderung); Abs. 6 Satz 2 (verdeckte Einlage erhöht Anschaffungskosten) | Text 17.10.2026 | beteiligungen/ablauf, bewertung-veraeusserung, erwerb-ausweis |
| § 20 EStG | (Zeile unter bank) | Abs. 1 Nr. 1 Satz 2, 3 (vGA als Bezug, Einlagenrückgewähr keine Einnahme), Nr. 2 (Liquidation), Nr. 3 (Investmenterträge) | Text 17.10.2026 | beteiligungen/dividenden, steuerfreistellung |
| § 36 EStG | (Zeile unter bank) | Abs. 2 Nr. 2 Buchst. a, b, Satz 2: Anrechnung auch auf § 8b-freie Bezüge, nur mit Bescheinigung | Text 17.10.2026 | beteiligungen/ablauf, dividenden |
| § 36a EStG | https://www.gesetze-im-internet.de/estg/__36a.html | Mindesthaltedauer 45 Tage, Wertänderungsrisiko 70 %, drei Fünftel nicht anrechenbar (Abs. 1–3); Ausnahmen 20.000 €, ein Jahr (Abs. 5) | Text 17.10.2026 | beteiligungen/dividenden |
| § 43 EStG | (Zeile unter bank) | Abs. 1 Satz 1 Nr. 1, 1a, 5, 6, 9 (Kapitalerträge mit Abzug); Satz 3 (Abzug ungeachtet § 8b); Abs. 2 Satz 3 Nr. 1 (kein Abzug für Nr. 6, 8–12 bei unbeschränkt steuerpflichtiger Körperschaft) | Text 17.10.2026 | beteiligungen/ablauf, bewertung-veraeusserung, dividenden, investmentfonds |
| § 43a EStG | (Zeile unter bank) | Abs. 1 Satz 1 Nr. 1 (25 %); Abs. 2 Satz 1 (volle Erträge; bei Fonds nur Teilfreistellung nach § 20 Abs. 1 Satz 1 InvStG) | Text 17.10.2026 | beteiligungen/ablauf, dividenden, investmentfonds |
| § 44 EStG | (Zeile unter gesellschafter) | Abs. 1 Satz 2, 3 (Entstehung, wer einbehält); Abs. 2 Satz 1, 2 (Zufluss von Gewinnanteilen) | Text 17.10.2026 | beteiligungen/dividenden, investmentfonds |
| § 44a EStG | (Zeile unter bank) | Abs. 5 Satz 1, 4: Abstandnahme bei Dauerüberzahlern, Bescheinigung des Finanzamts; erfasst Nr. 1, nicht Nr. 1a | Text 17.10.2026 | beteiligungen/ablauf, dividenden |
| § 45a EStG | (Zeile unter bank) | Abs. 2 Satz 1 Nr. 1, 2: Bescheinigung durch Schuldner bzw. auszahlende Stelle | Text 17.10.2026 | beteiligungen/dividenden |
| § 1 SolZG | (Zeile unter bank) | Abs. 2: Anrechnung entsprechend | Text 17.10.2026 | beteiligungen/dividenden |
| § 3 SolZG | (Zeile unter bank) | Abs. 1 Nr. 5: Bemessung nach Kapitalertragsteuer | Text 17.10.2026 | beteiligungen/ablauf, dividenden |
| § 4 SolZG | (Zeile unter bank) | Satz 1, 3: 5,5 %, Bruchteile eines Cents | Text 17.10.2026 | beteiligungen/ablauf, dividenden |
| § 247 HGB | (Zeile unter gesellschafter) | Abs. 2: Anlagevermögen dauernd dienend | Text 17.10.2026 | beteiligungen/ablauf, erwerb-ausweis |
| § 252 HGB | (Zeile unter bank) | Abs. 1 Nr. 4: Realisation, vorhersehbare Risiken | Text 17.10.2026 | beteiligungen/bewertung-veraeusserung, dividenden |
| § 253 HGB | (Zeile unter belege) | Abs. 1 Satz 1, Abs. 3 Satz 1, 5, 6, Abs. 4, Abs. 5 Satz 1: Obergrenze, keine planmäßige Abschreibung bei unbegrenzter Nutzung, gemildertes und strenges Niederstwertprinzip, Wertaufholung | Text 17.10.2026 | beteiligungen/ablauf, bewertung-veraeusserung, erwerb-ausweis |
| § 255 HGB | (Zeile unter belege) | Abs. 1: Anschaffungskosten, Nebenkosten, nachträgliche Anschaffungskosten | Text 17.10.2026 | beteiligungen/ablauf, erwerb-ausweis |
| § 266 HGB | (Zeile unter rechnungen) | Abs. 1 Satz 3 (verkürzte Bilanz), Abs. 2 A. III., B. III. (Finanzanlagen, Wertpapiere) | Text 17.10.2026 | beteiligungen/erwerb-ausweis |
| § 267a HGB | (Zeile unter jahresabschluss) | Abs. 3 Nr. 3: reine Beteiligungsgesellschaft keine Kleinstkapitalgesellschaft | Text 17.10.2026 | beteiligungen/ablauf, erwerb-ausweis |
| § 271 HGB | https://www.gesetze-im-internet.de/hgb/__271.html | Abs. 1 Satz 1, 3: Beteiligung, Vermutung bei mehr als einem Fünftel; Abs. 2: verbundene Unternehmen | Text 17.10.2026 | beteiligungen/erwerb-ausweis |
| § 285 HGB | (Zeile unter jahresabschluss) | Nr. 11: Anhangangaben zu Beteiligungen | Text 17.10.2026 | beteiligungen/erwerb-ausweis |
| § 288 HGB | (Zeile unter jahresabschluss) | Abs. 1 Nr. 1: kleine Kapitalgesellschaften ohne Angabe nach § 285 Nr. 11 | Text 17.10.2026 | beteiligungen/erwerb-ausweis |
| § 290 HGB | (Zeile unter 1. Gesetze) | Abs. 1, Abs. 2 Nr. 1: Mutterunternehmen, Mehrheit der Stimmrechte | Text 17.10.2026 | beteiligungen/erwerb-ausweis |
| § 293 HGB | https://www.gesetze-im-internet.de/hgb/__293.html | größenabhängige Befreiung vom Konzernabschluss (nur genannt, Werte nicht übernommen) | Text 17.10.2026 | beteiligungen/erwerb-ausweis |
| § 15 GmbHG | https://www.gesetze-im-internet.de/gmbhg/__15.html | Abs. 3, 4: notarielle Form der Abtretung und der Verpflichtung | Text 17.10.2026 | beteiligungen/bewertung-veraeusserung, erwerb-ausweis |
| § 29 GmbHG | (Zeile unter gesellschafter) | Abs. 1: Anspruch auf den verteilbaren Gewinn | Text 17.10.2026 | beteiligungen/dividenden |
| § 46 GmbHG | (Zeile unter gesellschafter) | Nr. 1: Feststellung und Ergebnisverwendung durch die Gesellschafter | Text 17.10.2026 | beteiligungen/dividenden |
| § 138 AO | (Zeile unter gruendung) | Abs. 2 Satz 1 Nr. 3 (ausländische Beteiligungen ab 10 % oder über 150.000 €), Abs. 5 (mit der Erklärung, spätestens 14 Monate) | Text 17.10.2026 | beteiligungen/ablauf, bewertung-veraeusserung, erwerb-ausweis |
| § 1 InvStG | https://www.gesetze-im-internet.de/invstg_2018/__1.html | Abs. 1–3: Anwendungsbereich, Ausnahmen | Text 17.10.2026 | beteiligungen/investmentfonds |
| § 2 InvStG | https://www.gesetze-im-internet.de/invstg_2018/__2.html | Abs. 6, 7, 9: Aktien-, Misch-, Immobilien-, Auslands-Immobilienfonds; Abs. 11: Ausschüttung einschließlich Steuerabzug | Text 17.10.2026 | beteiligungen/investmentfonds |
| § 16 InvStG | https://www.gesetze-im-internet.de/invstg_2018/__16.html | Abs. 1: Investmenterträge; Abs. 3: kein § 8b KStG | Text 17.10.2026 | beteiligungen/investmentfonds |
| § 18 InvStG | https://www.gesetze-im-internet.de/invstg_2018/__18.html | Vorabpauschale: Basisertrag 70 %, Begrenzung, Börsenpreis, Zwölftel, Zufluss erster Werktag, Basiszins durch BMF | Text 17.10.2026 | beteiligungen/investmentfonds |
| § 19 InvStG | https://www.gesetze-im-internet.de/invstg_2018/__19.html | Abs. 1 Satz 3: Abzug der Vorabpauschalen beim Verkauf, ausdrücklich für Anteile außerhalb eines Betriebsvermögens | Text 17.10.2026 | beteiligungen/investmentfonds |
| § 20 InvStG | https://www.gesetze-im-internet.de/invstg_2018/__20.html | Abs. 1 Satz 1, 3 (30 %, 80 % für KStG-Anleger), Abs. 2 (Mischfonds halb), Abs. 3 (60 %, 80 %, Ausschluss), Abs. 4 (Nachweis), Abs. 5 (Gewerbesteuer zur Hälfte) | Text 17.10.2026 | beteiligungen/gewst-holding, investmentfonds |
| § 21 InvStG | https://www.gesetze-im-internet.de/invstg_2018/__21.html | anteiliges Abzugsverbot für Betriebsausgaben und Vermögensminderungen | Text 17.10.2026 | beteiligungen/investmentfonds |
| § 22 InvStG | https://www.gesetze-im-internet.de/invstg_2018/__22.html | Abs. 1: fiktive Veräußerung bei Änderung des Teilfreistellungssatzes | Text 17.10.2026 | beteiligungen/investmentfonds |
| § 2 UStG | (Zeile unter 1. Gesetze) | Abs. 1 (Unternehmer), Abs. 2 Nr. 2 Satz 1–3 (Organschaft) | Text 17.10.2026 | beteiligungen/ablauf, holding-umsatzsteuer |
| § 4 UStG | (Zeile unter belege) | Nr. 8 Buchst. a (Kredite), e, f (Wertpapiere, Anteile) | Text 17.10.2026 | beteiligungen/bewertung-veraeusserung, erwerb-ausweis, holding-umsatzsteuer |
| § 9 UStG | (Zeile unter 1. Gesetze) | Abs. 1: Verzicht auf Steuerbefreiung nach § 4 Nr. 8 Buchst. a–g | Text 17.10.2026 | beteiligungen/holding-umsatzsteuer |
| § 15 UStG | (Zeile unter 1. Gesetze) | Abs. 1, Abs. 2 Satz 1 Nr. 1, Abs. 4 Satz 1–3: Vorsteuer, Ausschluss bei steuerfreien Umsätzen, Aufteilung | Text 17.10.2026 | beteiligungen/ablauf, holding-umsatzsteuer |

### Verwaltungsanweisungen und weitere Quellen

| Quelle | Link | Wofür | Geprüft | Genutzt in |
|---|---|---|---|---|
| BMF vom 05.01.2024, IV C 1 - S 1980-1/19/10038 :008, BStBl I S. 154 (Fundstelle auch laut Bezugszeile des Schreibens vom 10.01.2025), Basiszins zum 02.01.2024 | https://datenbank.nwb.de/Dokument/1033913/ (Wiedergabe in der NWB-Datenbank; BMF-Original nicht gefunden) | 2,29 % | Text 19.10.2026 (NWB-Wiedergabe) | beteiligungen/investmentfonds |
| BMF vom 10.01.2025, IV C 1 - S 1980/00230/009/002, BStBl I S. 273 (Fundstelle laut Bezugszeile des Schreibens vom 13.01.2026), Basiszins zum 02.01.2025 | https://www.bundesfinanzministerium.de/Content/DE/Downloads/BMF_Schreiben/Steuerarten/Investmentsteuer/2025-01-10-basiszins-vorabpauschale-zum-2-1-2025.pdf | 2,53 %; Vorabpauschale 2025 gilt am 02.01.2026 als zugeflossen | Text 19.10.2026 (BMF-Original) | beteiligungen/ablauf, investmentfonds |
| BMF vom 13.01.2026, IV C 1 - S 1980/00230/012/001, Basiszins zum 02.01.2026; BStBl-Fundstelle nicht geprüft | https://www.bundesfinanzministerium.de/Content/DE/Downloads/BMF_Schreiben/Steuerarten/Investmentsteuer/2026-01-13-basiszins-berechnung-vorabpauschale.pdf | 3,20 %; Vorabpauschale 2026 gilt am 04.01.2027 als zugeflossen | Text 19.10.2026 (BMF-Original) | beteiligungen/ablauf, investmentfonds |
| BMF vom 16.05.2025, IV C 1 - S 2401/00008/014/051 (Steuerbescheinigungen) | (Zeile unter gesellschafter) | Rn. 43 (Muster II der leistenden GmbH), 47 (erst nach Zahlung), 57 (Muster III für Körperschaften), 59 (nur Privat-Teilfreistellung beim Abzug; nachrichtlicher Teil mit Vorabpauschalen), 71 (Muster ab 01.01.2026) | Text 17.10.2026 (Kanzleispiegel) | beteiligungen/dividenden, investmentfonds |
| UStAE Abschnitt 2.3 Abs. 2–4 | https://datenbank.nwb.de/Dokument/378652_2___3/ (NWB-Datenbank, Fassung „VZ 2010–2026 (aktuell)“) | Halten von Beteiligungen keine unternehmerische Tätigkeit, Dividenden kein Entgelt; Finanz-, Führungs-, gemischte Holding; Eingriff durch entgeltliche Leistungen; Zuordnung zum Unternehmen | Text 17.10.2026 (Verlagsdatenbank; amtlicher UStAE 503) | beteiligungen/dividenden, holding-umsatzsteuer |
| UStAE Abschnitt 15.22 Abs. 1, 2 | https://datenbank.nwb.de/Dokument/378652_15___22/ (NWB-Datenbank, Fassung „VZ 2010–2026 (aktuell)“) | Vorsteuer bei Beteiligungen im Unternehmen, Aufteilung bei gemischter Holding, Ausschluss bei Leistungen für Umsätze der Töchter (BFH XI R 24/22), Veräußerung | Text 17.10.2026 (Verlagsdatenbank) | beteiligungen/bewertung-veraeusserung, holding-umsatzsteuer |
| BFH vom 20.12.2000, I R 50/95 (mit Verweis auf GrS 2/99 vom 07.08.2000, BStBl II 2000, 632) | https://datenbank.nwb.de/Dokument/97545/ | keine phasengleiche Aktivierung von Dividendenansprüchen ohne objektiv belegten, endgültigen Ausschüttungswillen am Stichtag | Text 17.10.2026 (Verlagsdatenbank; Beschluss GrS 2/99 selbst nicht gelesen) | beteiligungen/dividenden |
| H 6.7 EStH „Teilwertvermutungen“ (BFH vom 06.11.2003, BStBl 2004 II S. 416) | https://www.steuerschroeder.de/steuergesetze/esth/6.7 | Teilwert einer Beteiligung: Anschaffungskosten beim Erwerb; Ertragslage, Ertragsaussichten, Vermögenswert, funktionale Bedeutung | Text 17.10.2026 (Kanzleispiegel; der Spiegel verweist zur Teilwertabschreibung noch auf BMF vom 16.07.2014, ist also nicht aktuell) | beteiligungen/bewertung-veraeusserung |

### Werte mit Stand

| Wert | Stand | Quelle | Genutzt in |
|---|---|---|---|
| Kapitalertragsteuer 25 %, Solidaritätszuschlag 5,5 % darauf | 17.10.2026 | § 43a Abs. 1 Satz 1 Nr. 1 EStG, § 3 Abs. 1 Nr. 5, § 4 SolZG | beteiligungen/ablauf, dividenden, investmentfonds |
| § 8b KStG: 5 % nicht abziehbar; Streubesitz unter 10 % zu Beginn des Kalenderjahres; Darlehen bei Beteiligung über 25 % | 17.10.2026 | § 8b Abs. 3, 4, 5 KStG | beteiligungen/ablauf, gewst-holding, steuerfreistellung |
| GewSt-Kürzung ab 15 % zu Beginn des Erhebungszeitraums (Inland, Ausland); Hinzurechnung Nr. 1 über 200.000 €; Messzahl 3,5 % | 17.10.2026 | § 9 Nr. 2a, 7, § 8 Nr. 1, § 11 Abs. 2 GewStG | beteiligungen/ablauf, gewst-holding |
| Teilfreistellung KStG-Anleger: Aktienfonds 80 %, Mischfonds 40 %, Immobilienfonds 60 %, Auslands-Immobilienfonds 80 %; GewSt je zur Hälfte; Privatanleger-Satz 30 % beim Abzug | 17.10.2026 | § 20 Abs. 1–3, 5 InvStG, § 43a Abs. 2 Satz 1 EStG | beteiligungen/ablauf, gewst-holding, investmentfonds |
| Basiszins 2024: 2,29 %; 2025: 2,53 %; 2026: 3,20 %; Zufluss 02.01.2026 bzw. 04.01.2027 | 19.10.2026 | BMF vom 05.01.2024, 10.01.2025, 13.01.2026 | beteiligungen/ablauf, investmentfonds |
| § 36a EStG: 45 Tage, 70 %, drei Fünftel, 20.000 €, ein Jahr | 17.10.2026 | § 36a Abs. 1–3, 5 EStG | beteiligungen/dividenden |
| Mitteilung ausländischer Beteiligungen: 10 % oder 150.000 €, spätestens 14 Monate | 17.10.2026 | § 138 Abs. 2 Satz 1 Nr. 3, Abs. 5 AO | beteiligungen/erwerb-ausweis |
| Beteiligungsvermutung: mehr als ein Fünftel des Nennkapitals | 17.10.2026 | § 271 Abs. 1 Satz 3 HGB | beteiligungen/ablauf, erwerb-ausweis |

### Buchfink-eigene Quellen

| Quelle | Pfad | Wofür |
|---|---|---|
| Investmentrechnung | `internal/accounting/investment.go`, `tax_params.go` | Fondsarten, `PartialExemptionFor` (KSt-Sätze, Anlegerstellung), `ComputeVorabpauschale`, `InvestmentBaseRateFor` (Basiszins 2024–2026), `firstWorkdayOfYear` (Montag bis Freitag, wie BMF 13.01.2026) |
| Investment-Dienst | `internal/service/asset_investment_service.go` | Vorabpauschale festhalten (Bewegung ohne Buchung, Jahr der Entstehung), Nebenrechnung bei Ertrag und Abgang (Abzug der Vorabpauschalen) |
| Anlagenkonten | `internal/accounting/asset_accounts.go` | Katalog 0800–0990, `AssetIncomeAccount` (§-8b-Konten 7005/7006/7014 für Anteile an Kapitalgesellschaften, sonst 7000/7010/7011/7100), `ImpairmentAccount` (7204/7200/7201), `WriteUpAccount` (4913/4912), `DisposalAccountsFor` (4852/4858, 6892/6898, 4851/4857, 6891/6897), `WithholdingTaxAccount` 7630 |
| Anlagen-Dienst | `internal/service/asset_service.go` | `BookAssetIncome` (eine Zeile 7630, Ausleihungszinsen steuerfrei, Dividenden nicht steuerbar), `AcquisitionCandidates`, `DisposalRequest` (Verkauf, Tilgung, Verschrottung; Zahlungsmittel oder offener Posten), Hinweise zu Finanzanlagen |
| Anlagegut | `internal/domain/asset.go`, `internal/domain/legalform.go` | `HoldingPermille`, `TaxPrivileged`, `FundClass`, `Quantity`; Anlegerstellung aus der Rechtsform |
| Oberfläche | `frontend/src/pages/books/AssetsTab.tsx` | Kartei nur lesend (Bestand, Bewegungen, Anlagenspiegel); Pflege, Ertrag, Vorabpauschale, Wertänderung und Abgang bieten weder App noch Werkzeug |
| SKR04 | `internal/accounting/skr04_2026.json` | 0800–0990, 1260–1297, 1500–1530, 4851–4858, 4904–4916, 6891–6898, 7000–7020, 7100–7110, 7200–7255, 7630–7639 mit Kennzeichen G/K und Fußnote 9 |
| Werkzeugkatalog | `docs/entwicklung/mcp.md` | `bankumsatz_buchen`, `buchung_erfassen`, `abschlussbuchung` (`anlage_aktivieren`) |
| Abschlusssteuer, Erklärung | `internal/accounting/closing_tax.go`, `internal/declaration/kst.go` (`participations`, `classifyHoldings`, `investments`), `gewst.go` | § 8b Abs. 1 bis 5 aus Konten und Kartei, Streubesitz nach Quote, § 8 Nr. 5 und § 9 Nr. 2a GewStG, Teilfreistellung |

### Offen

- **Zeilen der Vordrucke:** Zeilennummern für § 8b KStG in der
  Körperschaftsteuererklärung (Anlage GK) und für § 8 Nr. 5, § 9 Nr. 2a
  GewStG in GewSt 1 A nicht belegt; Vordrucke und Anleitungen 2025 nicht
  abrufbar (BMF 503, Formular-Management-System nur mit JavaScript). Im Skill
  ohne Zeilen.
- **Vorabpauschale beim bilanzierenden Anleger:** Jahr des Ansatzes
  (Entstehung oder fiktiver Zufluss nach § 18 Abs. 3 InvStG) nicht belegt;
  Sekundärquellen widersprechen sich (Haufe: aktiver Ausgleichsposten in der
  Steuerbilanz; Deka-Steuerleitfaden: Zufluss beim betrieblichen Anleger bei
  Anspruchsentstehung; ob die Teilfreistellung auf die Vorabpauschale wirkt,
  bewerten Deka und Kleeberg unterschiedlich). Das BMF-Anwendungsschreiben zum
  InvStG (21.05.2019, BStBl I S. 527, mit Änderungen) wurde nicht gelesen. Im
  Skill als Wertungsfrage; Teilfreistellung aus dem Wortlaut von § 16 Abs. 1
  und § 20 Abs. 1 InvStG.
- **Abzug der Vorabpauschalen beim Verkauf im Betriebsvermögen:** § 19 Abs. 1
  Satz 3 InvStG gilt nach dem Wortlaut nur außerhalb eines Betriebsvermögens.
  Buchfink zieht sie auch bei der GmbH ab; Verwaltungsauffassung nicht gelesen.
- **§ 21 InvStG bei der Gewerbesteuer:** ob das anteilige Abzugsverbot nur zur
  Hälfte wirkt, nicht belegt.
- **§ 8b Abs. 4 Satz 6 KStG:** ob Zukäufe mit dem Bestand zusammengerechnet
  werden, nicht geklärt.
- **Anrechnung bei Beschluss und Zahlung in verschiedenen Jahren:** Zuordnung
  der Kapitalertragsteuer nach dem Wortlaut von § 36 Abs. 2 Nr. 2 EStG; keine
  Verwaltungsanweisung gelesen.
- **Teilwertabschreibung börsennotierter Aktien:** BMF vom 02.09.2016
  (BStBl I S. 995) nicht gelesen (BMF 503, Kanzleispiegel veraltet); im Skill
  nicht verwendet.
- **Phasengleiche Vereinnahmung im Handelsrecht:** BGH- und EuGH-Rechtsprechung
  nicht gelesen; Skill rät wegen der Einheitsbilanz zur Buchung mit dem
  Beschluss.
- **Finanzierungsaufwand der Holding bei der Gewerbesteuer:** Zusammenspiel von
  § 8 Nr. 1 und § 9 Nr. 2a Satz 3 GewStG bei § 8b-freien Dividenden offen.
- **Einlagenrückgewähr:** handelsrechtliche Buchung und Folge bei Übersteigen
  des Buchwerts nicht belegt.
- **Finanzholding in Buchfink:** Eine Holding ohne Unternehmereigenschaft gibt
  keine Voranmeldungen ab; wie der Mandant dann einzurichten ist, ist offen.
- **Code-Befunde Buchfink (Korrekturkandidaten):**
  - `BookAssetIncome` bucht Kapitalertragsteuer und Solidaritätszuschlag als
    eine Zeile auf 7630. Aufrufer in App oder Werkzeug hat der Dienst nicht.
  - `ImpairmentAccount` bucht gekennzeichnete Anteile auch bei
    vorübergehender Minderung auf 7204 („dauerhaft“).
  - `WriteUpAccount` bucht gekennzeichnete Anteile auf 4913 (SKR04-Name:
    § 8b Abs. 3 S. 9 KStG); für die Wertaufholung eines Anteils passt 4914
    (§ 8b Abs. 2 KStG) dem Namen nach besser. Mit DATEV-Kontenbeschreibung
    abgleichen.
  - Zwölftelung der Vorabpauschale nur nach dem Erwerbsmonat des Anlageguts;
    Nachkäufe im Jahr werden nicht getrennt gekürzt.
  - `AssetService.Vorabpauschale` hat keinen Aufrufer in App oder Werkzeug.
    Basiszinsen vor 2024 sind nicht hinterlegt; die BMF-Schreiben dazu sind
    nicht gelesen.
  - Anteiliges Abzugsverbot (§ 21 InvStG) wird nicht gerechnet.
- UStAE nur über die NWB-Datenbank, KStH/EStH nur über Kanzleispiegel gelesen;
  Abgleich mit den amtlichen Handbüchern.

## pflichten

Quellen von `references/grundlagen/pflichten.md` (Pflichtenkalender;
ein Abschnitt je Pflichtart mit Anker `<a id="<art>"></a>`). Alle Links am
17.10.2026 mit HTTP 200 geprüft, außer wo vermerkt. „Text“ = Wortlaut am
17.10.2026 gelesen und Aussage abgeglichen. „Genutzt in“ nennt den Anker.

### Gesetze

| Norm | Link | Wofür | Geprüft | Genutzt in |
|---|---|---|---|---|
| § 1 AO | https://www.gesetze-im-internet.de/ao_1977/__1.html | Realsteuern: Dritter Teil (§ 108) und Fünfter Teil (§ 240) gelten für die Gemeinde entsprechend (Abs. 2 Nr. 3, 5) | Text | Fristende, gewst-vorauszahlung |
| § 69 AO | https://www.gesetze-im-internet.de/ao_1977/__69.html | Haftung der Vertreter bei Vorsatz oder grober Fahrlässigkeit, einschließlich Säumniszuschlägen | Text | lohnsteuer-anmeldung, kapitalertragsteuer |
| § 108 AO | https://www.gesetze-im-internet.de/ao_1977/__108.html | §§ 187–193 BGB entsprechend (Abs. 1); Fristende Samstag, Sonntag, Feiertag → nächster Werktag (Abs. 3) | Text | Fristende, alle steuerlichen Anker |
| § 109 AO | https://www.gesetze-im-internet.de/ao_1977/__109.html | Verlängerung auch rückwirkend (Abs. 1); im Berater-Fall über den letzten Februartag und nach Vorabanforderung über den angeordneten Termin nur ohne Verschulden (Abs. 2 Satz 1 Nr. 1, 2) | Text | Fristende, steuererklaerungen/fristen-zuschlaege |
| § 137 AO | https://www.gesetze-im-internet.de/ao_1977/__137.html | Anzeige der Gründung an Finanzamt und Gemeinde (Abs. 1) binnen eines Monats (Abs. 2) | Text | fragebogen-steuerliche-erfassung |
| § 138 AO | https://www.gesetze-im-internet.de/ao_1977/__138.html | Gemeinde unterrichtet Finanzamt (Abs. 1 Satz 1); Auskünfte nach amtlichem Datensatz, auch für Körperschaften nach § 137 (Abs. 1b); binnen eines Monats (Abs. 4) | Text | fragebogen-steuerliche-erfassung, gewerbeanmeldung |
| § 146 AO | https://www.gesetze-im-internet.de/ao_1977/__146.html | Unveränderbarkeit (Abs. 4) | Text | monatsfestschreibung |
| § 149 AO | https://www.gesetze-im-internet.de/ao_1977/__149.html | Pflicht bleibt trotz Schätzung (Abs. 1 Satz 4); sieben Monate nach Kalenderjahr oder gesetzlich bestimmtem Zeitpunkt (Abs. 2 Satz 1); Berater: letzter Februartag des zweiten Folgejahres für KSt mit Feststellungserklärung § 27 Abs. 2 Satz 4 KStG, GewSt, USt (Abs. 3 Nr. 2–4); Vorabanforderung vier Monate, nie kürzer als Abs. 2 (Abs. 4); nicht bei USt nach Beendigung (Abs. 5) | Text | Steuerberater, kst-/gewst-/ust-erklaerung |
| § 150 AO | https://www.gesetze-im-internet.de/ao_1977/__150.html | Härtefall: Verzicht auf Datenfernübertragung bei wirtschaftlicher oder persönlicher Unzumutbarkeit (Abs. 8) | Text | ebilanz |
| § 152 AO | https://www.gesetze-im-internet.de/ao_1977/__152.html | Ermessen, entschuldbar (Abs. 1); zwingend nach 14 Monaten oder nach Termin der Vorabanforderung (Abs. 2 Nr. 1, 3); Ausnahmen, u. a. SVZ-Anmeldung (Abs. 3 Nr. 1–4); 0,25 %, mind. 25 € je Monat (Abs. 5 Satz 2); GewSt-Messbetrag und Feststellungen 25 € je Monat, von den Ausnahmen nur Abs. 3 Nr. 1 (Abs. 6); Monats-/Quartalsanmeldungen, jährliche LSt-Anmeldung, SVZ: Dauer, Häufigkeit, Höhe (Abs. 8 Nr. 1, 2, 5); höchstens 25.000 € (Abs. 10) | Text | Fristende und Versäumnis-Zeilen |
| § 162 AO | https://www.gesetze-im-internet.de/ao_1977/__162.html | Schätzung der Besteuerungsgrundlagen | Text | kst-erklaerung, gewst-erklaerung |
| § 224 AO | https://www.gesetze-im-internet.de/ao_1977/__224.html | Zahlung entrichtet: Bargeld bei Eingang, Scheck drei Tage danach (Abs. 2 Nr. 1); Überweisung bei Gutschrift (Nr. 2); SEPA-Lastschrift am Fälligkeitstag (Nr. 3) | Text | Fristende |
| § 240 AO | https://www.gesetze-im-internet.de/ao_1977/__240.html | 1 % je angefangenem Monat, Abrundung auf 50 € (Abs. 1); keine Erhebung bis drei Tage, außer § 224 Abs. 2 Nr. 1 (Abs. 3) | Text | Fristende, Zahlungsanker |
| § 328 AO | https://www.gesetze-im-internet.de/ao_1977/__328.html | Zwangsmittel zur Durchsetzung eines Verwaltungsakts auf Vornahme einer Handlung | Text | ebilanz, ebilanz-eroeffnung, fragebogen-steuerliche-erfassung |
| § 329 AO | https://www.gesetze-im-internet.de/ao_1977/__329.html | Zwangsgeld höchstens 25.000 € je Festsetzung | Text | ebilanz, ebilanz-eroeffnung, fragebogen-steuerliche-erfassung |
| § 377 AO | https://www.gesetze-im-internet.de/ao_1977/__377.html | Erster Teil des OWiG gilt für Steuerordnungswidrigkeiten (Abs. 2) | Text | ustva; monatsabschluss/ustva |
| § 380 AO | https://www.gesetze-im-internet.de/ao_1977/__380.html | Gefährdung der Abzugsteuern, Geldbuße bis 25.000 € | Text | lohnsteuer-anmeldung, kapitalertragsteuer |
| Art. 97 § 36 EGAO | https://www.gesetze-im-internet.de/aoeg_1977/art_97__36.html | Übergangsfristen 2020–2024 (Abs. 3), Berater-Frist 2024: 30.04.2026 | Text | kst-erklaerung |
| § 121 BGB | https://www.gesetze-im-internet.de/bgb/__121.html | „unverzüglich“ = ohne schuldhaftes Zögern (Abs. 1 Satz 1) | Text | Fristende |
| § 193 BGB | https://www.gesetze-im-internet.de/bgb/__193.html | Willenserklärung oder Leistung an einem bestimmten Tag oder in einer Frist: Sonnabend, Sonntag, Feiertag → nächster Werktag; ob es für Fristen aus HGB, GmbHG, GwG, GewO gilt, ist offen | Text | Fristende |
| § 10 OWiG | https://www.gesetze-im-internet.de/owig_1968/__10.html | nur Vorsatz, außer Fahrlässigkeit ist ausdrücklich bedroht | Text | ustva; monatsabschluss/ustva |
| § 14 UStG | https://www.gesetze-im-internet.de/ustg_1980/__14.html | Pflichtangabe Steuernummer oder USt-IdNr. (Abs. 4 Satz 1 Nr. 2) | Text | fragebogen-steuerliche-erfassung |
| § 18 UStG | https://www.gesetze-im-internet.de/ustg_1980/__18.html | VA und Zahlung am 10. Tag (Abs. 1 Satz 1, 4); Zeitraum (Abs. 2), Wahl Monat (Abs. 2a); Jahreserklärung, ein Monat bei kürzerem Zeitraum (Abs. 3 Satz 1, 2); Abschlusszahlung ein Monat nach Eingang (Abs. 4 Satz 1) | Text | ustva, sondervorauszahlung, ust-erklaerung |
| § 18a UStG | https://www.gesetze-im-internet.de/ustg_1980/__18a.html | ZM 25. Tag nach Quartal für § 3a-Abs.-2-Leistungen (Abs. 2); Monatsoption mit Anzeige (Abs. 3); Berichtigung binnen eines Monats (Abs. 10); AO ohne § 152 (Abs. 11) | Text | zm |
| § 26a UStG | https://www.gesetze-im-internet.de/ustg_1980/__26a.html | Vorauszahlung nicht rechtzeitig bis 30.000 €, ohne Leichtfertigkeit, also nur Vorsatz (Abs. 1, 3); ZM vorsätzlich oder leichtfertig bis 5.000 € (Abs. 2 Nr. 5, Abs. 3) | Text | ustva, zm; monatsabschluss/ablauf, monatsabschluss/ustva |
| § 46 UStDV | https://www.gesetze-im-internet.de/ustdv_1980/__46.html | Dauerfristverlängerung einen Monat | Text | ustva |
| § 47 UStDV | https://www.gesetze-im-internet.de/ustdv_1980/__47.html | SVZ nur Monatszahler, 1/11, Teiljahr, Beginn (Abs. 1–3) | Text | sondervorauszahlung |
| § 48 UStDV | https://www.gesetze-im-internet.de/ustdv_1980/__48.html | Antrag bis zur ersten betroffenen VA mit SVZ (Abs. 1); jährlich bis zur ersten VA berechnen, anmelden und entrichten (Abs. 2); Festsetzung (Abs. 3) | Text | sondervorauszahlung |
| § 5b EStG | https://www.gesetze-im-internet.de/estg/__5b.html | E-Bilanz nach amtlichem Datensatz mit unverdichteten Kontennachweisen (Abs. 1 Satz 1); bei Eröffnung des Betriebs Sätze 1 bis 3 entsprechend, ohne eigene Frist (Abs. 1 Satz 4); Verzicht bei Härte, § 150 Abs. 8 AO entsprechend (Abs. 2) | Text 19.10.2026 | ebilanz, ebilanz-eroeffnung |
| § 37 EStG | https://www.gesetze-im-internet.de/estg/__37.html | Vorauszahlungstermine 10.03./06./09./12. (Abs. 1); Festsetzung durch Bescheid (Abs. 3) | Text | kst-vorauszahlung |
| § 41a EStG | https://www.gesetze-im-internet.de/estg/__41a.html | LSt-Anmeldung und Abführung am 10. Tag (Abs. 1 Satz 1 Nr. 1, 2); Befreiung ohne Arbeitnehmer nach Mitteilung (Abs. 1 Satz 4); Zeitraum nach abzuführender Lohnsteuer des Vorjahres, 1.080 € und 5.000 € (Abs. 2) | Text | lohnsteuer-anmeldung; lohn/ablauf |
| § 43 EStG | https://www.gesetze-im-internet.de/estg/__43.html | Kapitalertragsteuer auf Gewinnanteile (Abs. 1 Satz 1 Nr. 1) | Text | kapitalertragsteuer |
| § 44 EStG | https://www.gesetze-im-internet.de/estg/__44.html | Abführung bei Gewinnausschüttung im Zeitpunkt des Zuflusses (Abs. 1 Satz 5 Hs. 2); Abrundung auf volle Euro (Abs. 1 Satz 6); Zufluss am Auszahlungstag, sonst Tag nach Beschluss, bei gesetzlicher oder durch Gesetz zugelassener Satzungs-Fälligkeit der Fälligkeitstag (Abs. 2 Satz 1, 2); Haftung (Abs. 5) | Text | kapitalertragsteuer |
| § 45a EStG | https://www.gesetze-im-internet.de/estg/__45a.html | Anmeldung in der Frist des § 44 Abs. 1, auch ohne Abzug mit Grund (Abs. 1); Bescheinigung auf Verlangen (Abs. 2) | Text | kapitalertragsteuer |
| § 51a EStG | https://www.gesetze-im-internet.de/estg/__51a.html | Kirchensteuer: Anlass- und Regelabfrage 1.9. bis 31.10. (Abs. 2c Satz 1 Nr. 3) | Text | kapitalertragsteuer |
| § 52 EStG | https://www.gesetze-im-internet.de/estg/__52.html | Kontennachweise ab Wj nach 31.12.2024 (Abs. 11 Satz 2) | Text | ebilanz |
| § 60 EStDV | https://www.gesetze-im-internet.de/estdv_1955/__60.html | bei Verzicht nach § 5b Abs. 2 Abschrift von Bilanz und GuV mit unverdichteten Kontennachweisen, Anlagenspiegel und Anlagenverzeichnis, bei Eröffnung Eröffnungsbilanz (Abs. 1) | Text | ebilanz |
| § 84 EStDV | https://www.gesetze-im-internet.de/estdv_1955/__84.html | Anlagenverzeichnis nach § 60 Abs. 1 erst für Wj nach 31.12.2027 (Abs. 3f) | Text | ebilanz |
| § 7 KStG | https://www.gesetze-im-internet.de/kstg_1977/__7.html | abweichendes Wirtschaftsjahr: Gewinn im Kalenderjahr des Endes (Abs. 4 Satz 2) | Text | kst-erklaerung |
| § 27 KStG | https://www.gesetze-im-internet.de/kstg_1977/__27.html | Feststellungserklärung Einlagekonto auf jeden Schluss des Wirtschaftsjahres (Abs. 2 Satz 4) | Text | kst-erklaerung |
| § 31 KStG | https://www.gesetze-im-internet.de/kstg_1977/__31.html | EStG entsprechend (Abs. 1); elektronische Übermittlung (Abs. 1a); Vorauszahlungen bei abweichendem Wirtschaftsjahr (Abs. 2) | Text | kst-vorauszahlung, kst-erklaerung |
| § 3 SolZG | https://www.gesetze-im-internet.de/solzg_1995/__3.html | Solidaritätszuschlag auf Vorauszahlungen (Abs. 1 Nr. 2) | Text | kst-vorauszahlung |
| § 14a GewStG | https://www.gesetze-im-internet.de/gewstg/__14a.html | Messbetragserklärung und Zerlegungserklärung elektronisch | Text | gewst-erklaerung |
| § 19 GewStG | https://www.gesetze-im-internet.de/gewstg/__19.html | Termine 15.02./05./08./11., abweichendes Wj (Abs. 1 Satz 1–3); ein Viertel (Abs. 2); Anpassung, Erhöhung binnen eines Monats (Abs. 3); Festsetzung ab 50 € (Abs. 5 Satz 2) | Text | gewst-vorauszahlung |
| § 12 HGB | https://www.gesetze-im-internet.de/hgb/__12.html | Dokumente elektronisch einreichen (Abs. 2) | Text | gesellschafterliste |
| § 239 HGB | https://www.gesetze-im-internet.de/hgb/__239.html | Unveränderbarkeit (Abs. 3) | Text | monatsfestschreibung |
| § 240 HGB | https://www.gesetze-im-internet.de/hgb/__240.html | Inventar zum Schluss des Geschäftsjahres in angemessener Zeit (Abs. 2) | Text | bilanzstichtag |
| § 242 HGB | https://www.gesetze-im-internet.de/hgb/__242.html | Bilanz und GuV (Abs. 1, 2); Eröffnungsbilanz zu Beginn des Handelsgewerbes, Vorschriften über die Bilanz des Jahresabschlusses entsprechend (Abs. 1 Satz 1, 2) | Text 19.10.2026 | bilanzstichtag, eroeffnungsbilanz |
| § 243 HGB | https://www.gesetze-im-internet.de/hgb/__243.html | Aufstellung in der einem ordnungsmäßigen Geschäftsgang entsprechenden Zeit (Abs. 3) | Text 19.10.2026 | eroeffnungsbilanz |
| § 245 HGB | https://www.gesetze-im-internet.de/hgb/__245.html | Unterzeichnung mit Datum | Text | aufstellung, eroeffnungsbilanz |
| § 257 HGB | https://www.gesetze-im-internet.de/hgb/__257.html | Eröffnungsbilanzen aufbewahren (Abs. 1 Nr. 1), nicht nur als Wiedergabe (Abs. 3), zehn Jahre (Abs. 4) | Text 19.10.2026 | eroeffnungsbilanz |
| § 264 HGB | https://www.gesetze-im-internet.de/hgb/__264.html | Aufstellung in drei Monaten, kleine bis sechs Monate (Abs. 1 Satz 3, 4) | Text | aufstellung |
| § 267a HGB | https://www.gesetze-im-internet.de/hgb/__267a.html | Kleinst-Merkmale 450.000/900.000/10 (Abs. 1); kleine Regeln gelten (Abs. 2); Holding ohne Eingriff in die Verwaltung der Töchter keine Kleinstgesellschaft, Gesellschafterrechte zählen nicht (Abs. 3 Nr. 3) | Text | aufstellung, offenlegung; jahresabschluss/ablauf, jahresabschluss/aufstellen-feststellen-offenlegen |
| § 325 HGB | https://www.gesetze-im-internet.de/hgb/__325.html | festgestellter Abschluss an das Unternehmensregister (Abs. 1); ein Jahr (Abs. 1a); Änderungen (Abs. 1b) | Text | feststellung, offenlegung |
| § 326 HGB | https://www.gesetze-im-internet.de/hgb/__326.html | klein: Bilanz und Anhang ohne GuV-Angaben (Abs. 1); Kleinst: Bilanz, Hinterlegung mit Mitteilung (Abs. 2) | Text | offenlegung |
| § 335 HGB | https://www.gesetze-im-internet.de/hgb/__335.html | BfJ-Verfahren, auch gegen die Gesellschaft, unabhängig von der Aufstellung, 2.500–25.000 € (Abs. 1); sechs Wochen, Kosten (Abs. 3); Festsetzung mit erneuter Androhung, Herabsetzung 500/1.000/2.500 € (Abs. 4); Wiedereinsetzung binnen zwei Wochen, Nachholen binnen sechs Wochen (Abs. 5) | Text | aufstellung, feststellung, offenlegung |
| § 5a GmbHG | https://www.gesetze-im-internet.de/gmbhg/__5a.html | UG-Rücklage ein Viertel (Abs. 3) | Text | ergebnisverwendung |
| § 16 GmbHG | https://www.gesetze-im-internet.de/gmbhg/__16.html | Gesellschafter gegenüber der GmbH nur, wer in der Liste steht (Abs. 1) | Text | gesellschafterliste |
| § 29 GmbHG | https://www.gesetze-im-internet.de/gmbhg/__29.html | Anspruch auf Jahresüberschuss vorbehaltlich Beschluss (Abs. 1, 2) | Text | ergebnisverwendung |
| § 40 GmbHG | https://www.gesetze-im-internet.de/gmbhg/__40.html | Liste unverzüglich, unterschrieben oder mit qualifizierter elektronischer Signatur (Abs. 1); Notar statt Geschäftsführer (Abs. 2); Haftung gegenüber Betroffenen und Gläubigern als Gesamtschuldner (Abs. 3) | Text | gesellschafterliste |
| § 42a GmbHG | https://www.gesetze-im-internet.de/gmbhg/__42a.html | Vorlage unverzüglich (Abs. 1); Beschluss in acht/elf Monaten, Satzung kann nicht verlängern (Abs. 2) | Text | feststellung, ergebnisverwendung |
| § 46 GmbHG | https://www.gesetze-im-internet.de/gmbhg/__46.html | Feststellung und Ergebnisverwendung durch die Gesellschafter (Nr. 1) | Text | feststellung, ergebnisverwendung |
| § 48 GmbHG | https://www.gesetze-im-internet.de/gmbhg/__48.html | Beschlussform | Text (Eintrag jahresabschluss) | feststellung |
| § 3 GwG | https://www.gesetze-im-internet.de/gwg_2017/__3.html | wirtschaftlich Berechtigter: mehr als 25 %, mittelbar über kontrollierte Vereinigung, Kontrolle nach § 290 Abs. 2–4 HGB (Abs. 2 Satz 1–4), fiktiv der gesetzliche Vertreter (Abs. 2 Satz 5) | Text | transparenzregister |
| § 19 GwG | https://www.gesetze-im-internet.de/gwg_2017/__19.html | Angaben: Name, Geburtsdatum, Wohnort, Art und Umfang, Staatsangehörigkeiten (Abs. 1) | Text | transparenzregister |
| § 20 GwG | https://www.gesetze-im-internet.de/gwg_2017/__20.html | einholen, aufbewahren, aktuell halten, unverzüglich mitteilen (Abs. 1 Satz 1) | Text | transparenzregister |
| § 24 GwG | https://www.gesetze-im-internet.de/gwg_2017/__24.html | Jahresgebühr für die Führung des Registers (Abs. 1) | Text | transparenzregister |
| § 56 GwG | https://www.gesetze-im-internet.de/gwg_2017/__56.html | Nr. 55 (§ 20 Abs. 1), vorsätzlich oder leichtfertig; bis 150.000 € bei Vorsatz, sonst bis 100.000 € (Abs. 1 Satz 2); bis 1 Mio. € (Abs. 3); Bundesverwaltungsamt (Abs. 5 Satz 2) | Text (Abs. 5 aus der Vollfassung) | transparenzregister |
| § 57 GwG | https://www.gesetze-im-internet.de/gwg_2017/__57.html | Bekanntmachung unanfechtbarer Bußgeldentscheidungen mit Namen (Abs. 1) | Text | transparenzregister |
| TrGebV, Anlage | https://www.gesetze-im-internet.de/trgebv_2020/anlage.html | Jahresgebühr ab Gebührenjahr 2024: 19,80 € (Nr. 1) | Text | transparenzregister |
| § 14 GewO | https://www.gesetze-im-internet.de/gewo/__14.html | Anzeige gleichzeitig mit dem Beginn (Abs. 1 Satz 1); Verlegung, Wechsel oder Ausdehnung des Gegenstands, Namensänderung, Aufgabe (Abs. 1 Satz 2) | Text | gewerbeanmeldung |
| § 15 GewO | https://www.gesetze-im-internet.de/gewo/__15.html | Empfangsbescheinigung binnen drei Tagen (Abs. 1) | Text | gewerbeanmeldung |
| § 146 GewO | https://www.gesetze-im-internet.de/gewo/__146.html | Anzeige nach § 14 Abs. 1 Satz 1, auch i. V. m. Satz 2, nicht rechtzeitig (Abs. 2 Nr. 2 Buchst. b), bis 1.000 € (Abs. 3) | Text | gewerbeanmeldung |
| § 192 SGB VII | https://www.gesetze-im-internet.de/sgb_7/__192.html | Meldung an den Unfallversicherungsträger gilt mit Gewerbeanzeige binnen einer Woche als erfüllt (Abs. 1 Satz 2) | Text | gewerbeanmeldung |
| § 1 GewAnzV | https://www.gesetze-im-internet.de/gewanzv_2014/__1.html | Vordruck Anlage 1 bei Betriebsbeginn (Nr. 1) | Text | gewerbeanmeldung |
| § 2 GewAnzV | https://www.gesetze-im-internet.de/gewanzv_2014/__2.html | elektronische Gewerbeanzeige, Identitätsprüfung | Text | gewerbeanmeldung |
| § 283 StGB | https://www.gesetze-im-internet.de/stgb/__283.html | Strafbarkeit nur bei Zahlungseinstellung, Insolvenz oder Abweisung mangels Masse (Abs. 6) | Text | bilanzstichtag |
| § 283b StGB | https://www.gesetze-im-internet.de/stgb/__283b.html | Bilanz oder Inventar nicht in der vorgeschriebenen Zeit (Abs. 1 Nr. 3 Buchst. b); § 283 Abs. 6 entsprechend (Abs. 3) | Text 19.10.2026 | bilanzstichtag, aufstellung, eroeffnungsbilanz |

### Verwaltungsanweisungen und amtliche Stellen

| Quelle | Link | Wofür | Geprüft | Genutzt in |
|---|---|---|---|---|
| AEAO zu § 108 Nr. 1 (Kanzleispiegel) | https://www.steuerschroeder.de/steuergesetze/aeao/108 | „Fälligkeitstermine geben das Ende einer Frist an“, also gilt § 108 Abs. 3 AO für Zahlungstermine; im Skill wörtlich zitiert und als Spiegel gekennzeichnet | Text 17.10.2026 (erneut abgerufen, 200); der Spiegel antwortet bei parallelen Abrufen mit 500/503; amtliche Fassung auf bundesfinanzministerium.de nicht erreichbar | Fristende, Zahlungsanker |
| AEAO zu § 149 (Kanzleispiegel) | https://www.steuerschroeder.de/steuergesetze/aeao/149 | nur Anwendungsregel; nichts zur Frist der Einlagekonto-Feststellung bei abweichendem Wj | Text 17.10.2026 | kst-erklaerung (Offen) |
| GoBD, BMF vom 28.11.2019 in der Fassung vom 14.07.2025 | https://www.lfst.bayern.de/fileadmin/RESSOURCEN/INFORMATIONEN/Steuerinfos/Weitere_Themen/Aussenpruefung/2019-11-28-GoBD-1.pdf; Änderungen 2024 und 2025 wie Abschnitt 2 | Rz. 52, 87 (Buchung bis Ablauf des Folgemonats), Rz. 107–110 (Unveränderbarkeit, Vorerfassungen, Festschreibung); von den Änderungen 2024 und 2025 nicht berührt | Text 17.10.2026 | Kopf, monatsfestschreibung |
| BMF vom 10.06.2025 (E-Bilanz, Taxonomien 6.9) | https://betriebs-berater.ruw.de/news/media/2/BMF-Schreiben-17367.pdf (Verlagsspiegel) | Kontennachweise ab Wj nach 31.12.2024; Nichtbeanstandung für 6.9 ohne Kontennachweise mit Begründung in der Position „Erläuterung, warum …“; Anlagenspiegel Mussfeld | Text 17.10.2026 | ebilanz |
| ELSTER, Alle Formulare | https://www.elster.de/eportal/formulare-leistungen/alleformulare | Formularnamen und Gruppen: Umsatzsteuer-Voranmeldung, Dauerfristverlängerung/Sondervorauszahlung (monatlich), Zusammenfassende Meldung, Umsatzsteuererklärung, Körperschaftsteuererklärung (KSt 1), Gewerbesteuererklärung (GewSt 1 A), GewSt 1 D, Lohnsteuer-Anmeldung, Kapitalertragsteuer-Anmeldung nach Einkommensteuergesetz, Fragebogen Gründung Kapitalgesellschaft, Antrag auf Fristverlängerung, Antrag auf Anpassung von Vorauszahlungen; kein Formular für Bilanz oder E-Bilanz | Text | Mein ELSTER, alle ELSTER-Anker, ebilanz; steuererklaerungen/fristen-zuschlaege |
| eBilanz+, Hilfe XBRL-Import | https://ebilanzplus.de/hilfe/xbrl | Import „XBRL-Datei“ in fünf Schritten; nicht alle Berichtsbestandteile, etwa der Anlagenspiegel | Text 17.10.2026 | pflichten#ebilanz, steuererklaerungen/ebilanz |
| eBilanz+, Hilfe Import | https://ebilanzplus.de/hilfe/import | Saldenliste als CSV/Excel, Layout mit Taxonomie-Elementnamen | Text 17.10.2026 | pflichten#ebilanz, steuererklaerungen/ebilanz |
| eBilanz+, Kosten | https://ebilanzplus.de/kosten | 35 € netto je Übermittlung; Eröffnungsbilanz und Korrektur kostenlos | Text 17.10.2026 | steuererklaerungen/ebilanz |
| eBilanz-Online, FAQ XBRL-Datei | https://www.ebilanzonline.de/wissen/faq/importmoeglichkeiten/xbrl-datei/ | XBRL-Import zum Weiterbearbeiten | Text 17.10.2026 | pflichten#ebilanz, steuererklaerungen/ebilanz |
| eBilanz-Online, Preismodell | https://www.ebilanzonline.de/preismodell/ | 39 € netto je erfolgreicher Übertragung, Staffelpreise | Text 17.10.2026 | steuererklaerungen/ebilanz |
| BZSt, ZM: Elektronische Abgabe | https://www.bzst.de/DE/Unternehmen/Umsatzsteuer/ZusammenfassendeMeldung/ElektronischeAbgabe/elektronischeabgabe_zm_node.html | ELSTER oder BOP, BOP mit Anmeldung zum Fachverfahren ZM, Online-Formular mit CSV-Import | Text | zm |
| BZSt, ZM: Fristen | https://www.bzst.de/DE/Unternehmen/Umsatzsteuer/ZusammenfassendeMeldung/Fristen/fristen_node.html | 25. Tag; Dauerfristverlängerung gilt nicht | Text | zm |
| Publikations-Plattform, Startseite | https://publikations-plattform.de | Prozesseinstieg Unternehmensregister: Veröffentlichen, Hinterlegen; XML-Format nach § 11 URV, Konvertierungsentgelt für PDF/Office; Geschäftsjahre vor 2022 an den Bundesanzeiger; Identifizierungspflicht seit 01.08.2022 | Text | offenlegung |
| Publikations-Plattform, So geht's: Jahresabschluss Veröffentlichung | https://publikations-plattform.de/order/de/wissenswertes/sogehts/javeroeffentlichung | vier Schritte; Datei hochladen oder Formular (nur kleine Unternehmen): Bilanzwerte in vorbelegte Felder, Anhang in ein Feld | Text | offenlegung |
| Publikations-Plattform, So geht's: Jahresabschluss Hinterlegung | https://publikations-plattform.de/order/de/wissenswertes/sogehts/jahinterlegung | Hinterlegung nur Kleinst; Beteiligungsgesellschaften ausgeschlossen | Text | offenlegung |
| Publikations-Plattform, Meine Daten | https://publikations-plattform.de/order/de/wissenswertes/sogehts/meinedaten | Stand der Aufträge, Belege erfolgter Veröffentlichungen und Hinterlegungen; eine „Auftragsnummer“ nennt keine Seite der Plattform | Text 17.10.2026 | offenlegung; jahresabschluss/ablauf, jahresabschluss/aufstellen-feststellen-offenlegen |
| Publikations-Plattform, Registrieren/Identifizieren | https://publikations-plattform.de/order/de/wissenswertes/sogehts/registrierenidentifizeren | einmalige Registrierung; Identifizierung je übermittelnder Person vor der ersten Übermittlung; vier Verfahren; Gebühr nach JVKostG | Text | offenlegung |
| Transparenzregister, Hilfe-Center | https://www.transparenzregister.de/hilfe-center (JavaScript-Seite); Lesepfad ohne Browser: https://www.transparenzregister.de/connector/api/help-center (JSON, im Skill verlinkt) | Nutzerkonto anlegen, Aktivierung binnen 24 Stunden, Option „Eintragungen vornehmen“, Eintragungsstrecke, Versicherung der Berechtigung, Eingangsmitteilung mit 12-stelliger Vorgangsnummer 14 Tage abrufbar, Folgemitteilung mit Gültigkeitsdatum, Änderungsmitteilung GmbH i. G., Jahresgebühr nach TrGebV | Text über `https://www.transparenzregister.de/connector/api/help-center` (JSON hinter der Seite) | transparenzregister |

### Sonstige Quellen

| Quelle | Link | Wofür | Geprüft | Genutzt in |
|---|---|---|---|---|

### Werte mit Stand

| Wert | Stand | Quelle | Genutzt in |
|---|---|---|---|
| VA und Zahlung am 10. Tag; Dauerfristverlängerung + 1 Monat | 17.10.2026 | § 18 Abs. 1 UStG, § 46 UStDV | ustva |
| SVZ 1/11 bis 10. Februar | 17.10.2026 | §§ 47, 48 UStDV | sondervorauszahlung |
| ZM am 25. Tag; Bußgeld bis 5.000 € | 17.10.2026 | § 18a, § 26a UStG | zm |
| KSt-VZ 10.03./06./09./12.; GewSt-VZ 15.02./05./08./11. | 17.10.2026 | § 37 EStG, § 19 GewStG | kst-/gewst-vorauszahlung |
| LSt-Anmeldungszeitraum: Grenzen 1.080 € und 5.000 € | 17.10.2026 | § 41a Abs. 2 EStG | lohnsteuer-anmeldung |
| Jahreserklärungen: 7 Monate / letzter Februartag des zweiten Folgejahres; Zuschlag zwingend nach 14 Monaten | 17.10.2026 | §§ 149, 152 AO | kst-/gewst-/ust-erklaerung, ebilanz |
| Verspätungszuschlag 0,25 %, mind. 25 € / 25 € je Monat; höchstens 25.000 € | 17.10.2026 | § 152 Abs. 5, 6, 10 AO | Versäumnis-Zeilen |
| Aufstellung 3/6 Monate; Feststellung 8/11 Monate; Offenlegung 12 Monate | 17.10.2026 | § 264 HGB, § 42a GmbHG, § 325 HGB | Jahresabschluss-Anker |
| Ordnungsgeld 2.500–25.000 €, herabgesetzt 500/1.000/2.500 € | 17.10.2026 | § 335 HGB | offenlegung |
| Kleinst-Merkmale 450.000 € / 900.000 € / 10 | 17.10.2026 | § 267a Abs. 1 HGB | offenlegung |
| Transparenzregister: Bußgeld bis 150.000 € / 100.000 €, schwer bis 1 Mio. € | 17.10.2026 | § 56 GwG | transparenzregister |
| Transparenzregister: Jahresgebühr 19,80 € ab Gebührenjahr 2024 | 17.10.2026 | TrGebV, Anlage Nr. 1 | transparenzregister |
| GewSt-Vorauszahlung: Festsetzung ab 50 € | 17.10.2026 | § 19 Abs. 5 GewStG | gewst-vorauszahlung |
| Gewerbeanzeige: Bußgeld bis 1.000 € | 17.10.2026 | § 146 GewO | gewerbeanmeldung |
| Zwangsgeld bis 25.000 €; Bußgeld Abzugsteuern bis 25.000 € | 17.10.2026 | §§ 329, 380 AO | ebilanz, fragebogen, lohnsteuer-anmeldung, kapitalertragsteuer |
| eBilanz+ 35 € netto je Übermittlung; eBilanz-Online 39 € netto je Übertragung | 17.10.2026 | ebilanzplus.de, ebilanzonline.de | steuererklaerungen/ebilanz |

### Buchfink-eigene Quellen

Code von Buchfink:

| Quelle | Wofür |
|---|---|
| `internal/domain/obligation.go` | Pflichtarten als Anker-Schlüssel; Status offen/vorbereitet/erledigt/überfällig/entfällt; Kanäle; Zuständigkeit; Nachweisarten; Anlässe |
| `internal/service/obligation_catalog.go` | je Art Grundlage, Rhythmus, Kanal, Zuständigkeit, Herkunft des Nachweises, Verschiebung nach § 108 AO |
| `internal/service/obligation_rules.go` | Fälligkeiten: Berater-Frist und Übergangsjahre, Größenklasse für 3/6 und 8/11 Monate, E-Bilanz im Kalenderjahr des Wirtschaftsjahresendes, Kapitalertragsteuer am Tag nach dem Beschluss, Vorauszahlungen nur mit Bescheid oder Zahlungshistorie, LSt-Zeitraum aus 3730; Eröffnungsbilanz drei Monate nach der Beurkundung, ihre E-Bilanz mit der Frist der E-Bilanz des ersten Geschäftsjahres, beide nur bei einer Gründung in Buchfink |
| `internal/service/obligation_service.go` | Bestätigen, Entfallen und Zurücknehmen nur durch den Menschen; „vorbereitet“ auch durch einen verbundenen Agenten |
| `internal/service/check_service.go` (`CommitDueDate`) | Frist der Monatsfestschreibung: Ende Folgemonat, mit Dauerfristverlängerung + 1 Monat, plus Nachfrist |
| `internal/service/task_service.go` (`addObligations`, `commitOpenFrom`) | Pflichten in `aufgaben`; Festschreibung ab dem 10. des Folgemonats als offene Aufgabe |

### Offen

- **Zahltermine und § 108 Abs. 3 AO.** Buchfink verschiebt
  KSt-/GewSt-Vorauszahlung und Kapitalertragsteuer nicht (kein
  `ShiftsToWorkday` im Katalog). Nach AEAO zu § 108 Nr. 1 sind
  Fälligkeitstermine Fristenden. Die Referenz sagt, dass § 108 Abs. 3 AO gilt
  und `fristen` für die Vorauszahlungen nur das gesetzliche Datum zeigt; der
  Agent nennt die Verschiebung. Produkt: Vorauszahlungen in `fristen`
  ebenfalls verschieben, dann den Hinweis in der Referenz streichen.
- **Monatsfestschreibung und Voranmeldung.** `fristen` setzt die Festschreibung
  auf Ende Folgemonat (+ Dauerfristverlängerung + Nachfrist), die
  Voranmeldung ist am 10. fällig, und Buchfink nimmt ihr Transferticket erst
  nach der Festschreibung an. Die Skills weisen den Agenten an, den letzten
  Monat eines Voranmeldungszeitraums nach der UStVA-Fälligkeit zu planen.
  Produkt: Fälligkeit dieser Festschreibung höchstens auf die UStVA-Fälligkeit
  setzen oder das Transferticket unabhängig von der Festschreibung annehmen.
- **Nachweis Sondervorauszahlung.** Buchfink setzt „erledigt“ mit einer
  Buchung auf 3830; das belegt die Anmeldung nicht (§ 48 Abs. 2 UStDV). Die
  Lohnsteuer-Anmeldung führt ein gebuchtes Lohnjournal nur als vorbereitet.
  Produkt: auch die Sondervorauszahlung erst mit vom Menschen bestätigtem
  Transferticket erledigt.
- **Lohnsteuer-Anmeldungszeitraum.** Die Schätzung aus 3730 enthält
  Kirchensteuer und Solidaritätszuschlag und lässt sich in der App nicht
  korrigieren. Produkt: Zeitraum als Stammdatum oder Schätzung nur aus der
  Lohnsteuer.
- **Gewerbeanzeige, weitere Anlässe:** Verlegung, Gegenstandswechsel,
  Namensänderung und Aufgabe (§ 14 Abs. 1 Satz 2 GewO) bildet `fristen` nicht.
- **Kirchensteuer-Regelabfrage** (§ 51a Abs. 2c Satz 1 Nr. 3 EStG) steht nicht
  im Kalender; ob eine GmbH ohne geplante Ausschüttung abfragen muss, ist
  offen (gesellschafter/ausschuettung).
- **Gesellschafterliste ohne Notar:** Technischer Einreichungsweg zum
  Registergericht an keiner amtlichen Quelle geprüft; die Referenz verweist
  an das Gericht oder einen Notar.
- **Einlagekonto bei abweichendem Wirtschaftsjahr:** Nach § 149 Abs. 2 Satz 1
  AO endet die Frist sieben Monate nach dem Schluss des Wirtschaftsjahres,
  also vor der KSt-Frist. AEAO zu § 149 sagt dazu nichts. Die Referenz rät,
  mit dem früheren Datum zu planen.
- **§ 193 BGB** für Fristen aus HGB, GmbHG, GwG und GewO: nicht geklärt; Buchfink
  rechnet ohne Verschiebung.
- **E-Bilanz über Online-Dienste:** Weder eBilanz+ noch eBilanz-Online weist
  die Unterstützung der Taxonomien 6.9 und 6.10 ausdrücklich aus; eBilanz+
  nennt 6.9 für Wirtschaftsjahre ab 2026 in einem Blogbeitrag. Die
  Hilfeseite von eBilanz-Online zum XBRL-Import hat einen Satz, Einzelheiten
  stehen im Benutzerhandbuch nach der Anmeldung.
- **Gewerbesteuer-Säumniszuschlag und Lastschrift** bei der Gemeinde: Weg nach
  Bescheid; Gemeindesatzungen nicht geprüft.
- **Offenlegung:** Ob die Plattform für die Hinterlegung ein Webformular
  anbietet, ist nicht geprüft; Buchfink erzeugt kein XML der Plattform.
