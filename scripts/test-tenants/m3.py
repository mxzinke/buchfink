"""Mandant 3: Waldkauz Projekte UG (haftungsbeschränkt), Köln. Geschäftsjahr 01.10.–30.09., erstes Jahr bis zum Abschluss."""

from common import *

T = Tenant("3-Waldkauz-Projekte-UG")
F1, F2, F3, F4, F5, F6 = ("01-Gruendung-Okt-2025", "02-Q4-2025", "03-Q1-2026", "04-Q2-2026", "05-Q3-2026", "06-Oktober-2026-Abschluss")

ME = Party("Waldkauz Projekte UG (haftungsbeschränkt)", "Venloer Straße 241", "50823", "Köln",
           vat_id=de_vat("32519846"), tax_no="215/5712/3456", email="kontakt@waldkauz-projekte.example",
           phone="0221 9874 330", contact="Tobias Kern",
           iban=iban("DE", "37069922" + "0310447700"), bic="GENODED1RTB", bank="Rheinland Testbank eG",
           register="Amtsgericht Köln HRB 123987", gf="Tobias Kern")
ME_IG = Party(ME.name + " i. G.", ME.street, ME.zip, ME.city)
ME_B = Party(ME.name, ME.street, ME.zip, ME.city)
ME_V = Party(ME.name, ME.street, ME.zip, ME.city, vat_id=ME.vat_id)
TOBIAS_IBAN = iban("DE", "37050198" + "1934522800")

NOTAR = Party("Dr. Annette Kühn, Notarin", "Apostelnstraße 9", "50667", "Köln", tax_no="214/5003/1187",
              email="buero@notarin-kuehn.example", phone="0221 2577 40",
              iban=iban("DE", "37050198" + "0018827733"), bic="COLSDE33XXX", bank="Testsparkasse Köln")
DOMTECH = Party("Domtech Computer GmbH", "Hohe Straße 52", "50667", "Köln", vat_id=de_vat("12266491"),
                email="rechnung@domtech.example", phone="0221 2300 88", contact="Ilse Brück",
                iban=iban("DE", "37040044" + "0533002100"), bic="COBADEFFXXX", bank="Testbank Köln",
                register="Amtsgericht Köln HRB 44120", gf="Ilse Brück")
AKADEMIE = Party("Akademie Rheinland GmbH", "Poppelsdorfer Allee 32", "53115", "Bonn", vat_id=de_vat("18839007"),
                 email="rechnung@akademie-rheinland.example", phone="0228 7766 50", contact="Seminarbüro",
                 iban=iban("DE", "38050000" + "0047711200"), bic="COLSDE33XXX", bank="Testsparkasse Bonn",
                 register="Amtsgericht Bonn HRB 21877", gf="Prof. Dr. Ute Wenzel")
RHEINWERK = Party("Rheinwerk Coworking GmbH", "Schanzenstraße 6", "51063", "Köln", tax_no="218/5788/0912",
                  iban=iban("DE", "37050198" + "1000277199"))
VERS = Party("Rheinische Testversicherung AG", "Konrad-Adenauer-Ufer 41", "50668", "Köln",
             iban=iban("DE", "37040044" + "0100888800"), bic="COBADEFFXXX", bank="Testbank Köln",
             register="Amtsgericht Köln HRB 1188", phone="0221 4400 0")
WEBER = Party("Jonas Weber IT-Beratung", "Bonner Talweg 81", "53113", "Bonn", tax_no="205/2147/3390",
              email="mail@jonasweber-it.example", phone="0176 5512 0098",
              iban=iban("DE", "38060186" + "4410022300"), bic="GENODED1BRS", bank="Testvolksbank Bonn")
FA_IBAN = iban("DE", "37000000" + "0037001508")
LK_IBAN = iban("DE", "30050000" + "0004000111")
BRENNER = Party("Möbelhaus Brenner GmbH", "Vogelsanger Straße 300", "50827", "Köln", vat_id=de_vat("11408826"), iban=iban("DE", "37050198" + "0091228800"))
RHEINF = Party("Rheinfähre Logistik GmbH", "Deutz-Mülheimer Straße 51", "51063", "Köln", vat_id=de_vat("27066154"), iban=iban("DE", "37040044" + "0702201100"))
KLEINKRAM = Party("Kleinkram Werbeagentur GmbH", "Kaiserswerther Straße 115", "40474", "Düsseldorf", vat_id=de_vat("24470118"))
DOMSTADT = Party("Domstadt Immobilien GmbH", "Bayenthalgürtel 13", "50968", "Köln", vat_id=de_vat("29338410"), iban=iban("DE", "37050198" + "0022667100"))

KM = [("08.07.2026", "Köln – Düsseldorf und zurück, Kleinkram Werbeagentur (Forderung, Abstimmung)", 92),
      ("16.07.2026", "Köln-Ehrenfeld, Möbelhaus Brenner (Nachbetreuung ERP)", 24),
      ("04.08.2026", "Köln-Marienburg, Domstadt Immobilien (Kick-off DMS)", 30),
      ("18.08.2026", "Köln-Marienburg, Domstadt Immobilien (Workshop)", 30),
      ("01.09.2026", "Köln-Marienburg, Domstadt Immobilien (Workshop)", 30),
      ("15.09.2026", "Köln-Marienburg, Domstadt Immobilien (Workshop)", 30),
      ("24.09.2026", "Köln – Bonn und zurück, Jonas Weber (Abstimmung Fremdleistung)", 62),
      ("29.09.2026", "Köln-Marienburg, Domstadt Immobilien (Statustermin)", 30)]
KM_TOTAL = sum(k for _, _, k in KM)
KM_EUR = r2(D(KM_TOTAL) * D("0.30"))


def lohnjournal(month: str, folder: str):
    mm, yyyy = month[5:], month[:4]
    name = f"{yyyy}-{mm}-25_Lohnjournal_{mm}-{yyyy}.pdf"
    pages = [{"kind": "a4", "font": "Arial", "accent": "#3f6212", "footer": [["Lohnservice Rhein-Erft GmbH", "Hauptstraße 12, 50226 Frechen"], ["Mandant 2207"]],
              "blocks": [
                  {"type": "letterhead", "name": "Lohnservice Rhein-Erft GmbH", "size": 14, "tagline": "Entgeltabrechnung", "right": ["Mandant 2207", ME.name]},
                  {"type": "title", "text": f"Lohnjournal {mm}/{yyyy}", "sub": "Monatssummen · alle Beträge in Euro"},
                  {"type": "table", "widths": [12, "1fr", 34, 22, 20, 14, 14, 16, 22], "align": ["l", "l", "l", "r", "r", "r", "r", "r", "r"], "size": 8,
                   "header": ["PNr.", "Name", "Merkmale", "Brutto", "LSt", "SolZ", "KiSt", "SV-AN", "Netto"],
                   "rows": [["1", "Kern, Tobias", "GGF 100 %, SV-frei, StKl I", "4.000,00", "694,25", "0,00", "0,00", "0,00", "3.305,75"]]},
                  {"type": "para", "text": "Arbeitgeberanteile Sozialversicherung: keine (beherrschender Gesellschafter-Geschäftsführer). Umlagen U1/U2: keine."},
                  {"type": "table", "widths": ["1fr", 40, 30], "align": ["l", "l", "r"], "header": ["Zahlung", "fällig", "Betrag"], "rows": [
                      ["Nettogehalt Tobias Kern", "letzter Bankarbeitstag", "3.305,75"],
                      ["Lohnsteuer an Finanzamt Köln-Mitte (Anmeldungszeitraum Quartal)", "10.10.2026 (Quartalssumme)", "694,25"]]},
                  {"type": "para", "size": 8.5, "text": "Vergütung laut Gesellschafterbeschluss vom 15.06.2026. Werte erfunden, in sich stimmig."}]}]
    compile_doc(T.path(folder, name), f"Lohnjournal {mm}/{yyyy}", "Lohnservice Rhein-Erft", pages)
    T.add(folder, name, f"{yyyy}-{mm}-25", "PDF", "Lohnservice Rhein-Erft GmbH", "Brutto 4.000,00 €", "Lohnjournal: 6024 an 3720/3730")


def build() -> str:
    # ------------------------------------------------------------ Gründung
    vertrag = [{"kind": "a4", "font": "Georgia", "accent": "#222222", "footer": [], "blocks": [
        {"type": "lines", "lines": ["UR-Nr. 1544/2025 K", "Ausfertigung"], "size": 8.5}, {"type": "spacer", "mm": 4},
        {"type": "title", "text": "Errichtung einer Unternehmergesellschaft (haftungsbeschränkt)", "sub": "Verhandelt zu Köln am 01. Oktober 2025"},
        {"type": "para", "justify": True, "text": "Vor mir, Dr. Annette Kühn, Notarin in Köln, erschien Herr Tobias Kern, geb. 05.05.1990, wohnhaft Subbelrather Straße 410, 50825 Köln. "
         "Er errichtet eine Unternehmergesellschaft (haftungsbeschränkt) mit dem folgenden Gesellschaftsvertrag. Ein Musterprotokoll wird nicht verwendet, weil das Geschäftsjahr vom Kalenderjahr abweicht."},
        {"type": "h", "text": "Gesellschaftsvertrag"},
        {"type": "para", "justify": True, "text": [{"t": "§ 1 Firma, Sitz. ", "b": True}, "Waldkauz Projekte UG (haftungsbeschränkt), Sitz Köln."]},
        {"type": "para", "justify": True, "text": [{"t": "§ 2 Gegenstand. ", "b": True}, "Projektsteuerung, Einführung betrieblicher Software und Prozessberatung."]},
        {"type": "para", "justify": True, "text": [{"t": "§ 3 Stammkapital. ", "b": True}, "3.000,00 €, Geschäftsanteil Nr. 1 übernommen von Tobias Kern. Die Einlage ist in Geld und vor der Anmeldung in voller Höhe zu leisten (§ 5a Abs. 2 GmbHG)."]},
        {"type": "para", "justify": True, "text": [{"t": "§ 4 Geschäftsjahr. ", "b": True}, "Das Geschäftsjahr beginnt am 1. Oktober und endet am 30. September. Das erste Geschäftsjahr beginnt mit der Errichtung und endet am 30.09.2026."]},
        {"type": "para", "justify": True, "text": [{"t": "§ 5 Geschäftsführung. ", "b": True}, "Geschäftsführer ist Tobias Kern; er vertritt allein und ist von § 181 BGB befreit."]},
        {"type": "para", "justify": True, "text": [{"t": "§ 6 Rücklage. ", "b": True}, "In die Bilanz ist eine gesetzliche Rücklage einzustellen, in die ein Viertel des um einen Verlustvortrag geminderten Jahresüberschusses einzustellen ist (§ 5a Abs. 3 GmbHG)."]},
        {"type": "para", "justify": True, "text": [{"t": "§ 7 Gründungsaufwand. ", "b": True}, "Die Gesellschaft trägt die Gründungskosten bis 900,00 €."]},
        {"type": "sig", "items": [{"label": "Tobias Kern", "sig": "T. Kern"}, {"label": "Dr. Annette Kühn, Notarin", "sig": "Kühn"}]}]}]
    n = "2025-10-01_Gesellschaftsvertrag_UR-1544-2025.pdf"
    compile_doc(T.path(F1, n), "Gesellschaftsvertrag Waldkauz Projekte UG", "Notarin Dr. Kühn", vertrag)
    T.add(F1, n, "2025-10-01", "PDF", "Notarin Dr. Kühn", "3.000,00 € Stammkapital", "UG: Volleinzahlung in bar, abweichendes Geschäftsjahr 01.10.–30.09.")

    notar = Invoice("KB 1544/2025", "2025-10-08", NOTAR, ME_IG, [
        Line("Beurkundung Gesellschaftsvertrag, Einpersonengründung (Geschäftswert 30.000 €), 1,0-Gebühr", 1, D("125.00"), unit_label="pausch."),
        Line("Anmeldung zum Handelsregister, 0,5-Gebühr", 1, D("62.50"), unit_label="pausch."),
        Line("Vollzug: Gesellschafterliste, 0,3-Gebühr", 1, D("37.50"), unit_label="pausch."),
        Line("Erzeugung strukturierter Daten (XML), 0,3-Gebühr", 1, D("37.50"), unit_label="pausch."),
        Line("Beglaubigung von Abschriften", 1, D("10.00"), unit_label="pausch."),
        Line("Dokumentenpauschale", 1, D("14.00"), unit_label="pausch."),
        Line("Post- und Telekommunikationsentgelte", 1, D("20.00"), unit_label="pausch."),
    ], delivery="2025-10-01", due="2025-10-22")
    efile(T, F1, "2025-10-08_Notarin-Kuehn_Kostenberechnung.pdf", notar, fmt="pdf", issuer="Notarin Dr. Kühn", case="Gründungskosten (bis 900 € laut Satzung)",
          pdf_kwargs=dict(font="Georgia", accent="#5a3e1b", title="Kostenberechnung (§ 19 GNotKG)", intro="UR-Nr. 1544/2025 K – Errichtung der Waldkauz Projekte UG (haftungsbeschränkt)"))

    dom = Invoice("RE 2025-30981", "2025-10-09", DOMTECH, ME_IG, [
        Line("Notebook ProBook 440 G11, 16 GB, 512 GB SSD", 1, D("1149.00"), detail="Seriennr. 5CD4471KQ2"),
        Line("Monitor 27\" QHD, höhenverstellbar", 1, D("289.00")),
    ], delivery="2025-10-09", due="2025-10-23", customer_no="88412",
        means={"code": "58", "iban": DOMTECH.iban, "bic": DOMTECH.bic, "account_name": DOMTECH.name}, terms="Zahlbar innerhalb von 14 Tagen.")
    efile(T, F1, "2025-10-09_Domtech_Rechnung_2025-30981.pdf", dom, fmt="zugferd", issuer="Domtech Computer GmbH",
          case="Notebook aktivieren (AfA im Abschluss), Monitor GWG",
          pdf_kwargs=dict(font="Avenir Next", accent="#7c2d12", tagline="Computer am Dom"))

    ag = letter_pages(Party("Amtsgericht Köln – Registergericht", "Reichenspergerplatz 1", "50670", "Köln"), ME.address_lines(),
                      meta=[["Kassenzeichen", "2510 4471 3302"], ["Datum", "27.10.2025"], ["Aktenzeichen", "HRB 123987"]],
                      title="Kostenrechnung", font="Arial", accent="#30475e",
                      paras=["In der Registersache Waldkauz Projekte UG (haftungsbeschränkt) werden folgende Kosten erhoben:",
                             {"type": "table", "widths": ["1fr", 30], "align": ["l", "r"], "header": ["Gegenstand", "Betrag"],
                              "rows": [["Ersteintragung (Nr. 2100 GV HRegGebV), eingetragen am 20.10.2025", "150,00 €"]]},
                             {"type": "totals", "rows": [["Zu zahlen", "150,00 €", True]]},
                             f"Zahlbar bis 10.11.2025 an die Landeskasse NRW, IBAN {fmt_iban(LK_IBAN)}."])
    n = "2025-10-27_Amtsgericht-Koeln_Kostenrechnung_HRB-123987.pdf"
    compile_doc(T.path(F1, n), "Kostenrechnung HRB 123987", "Amtsgericht Köln", ag)
    T.add(F1, n, "2025-10-27", "PDF", "Amtsgericht Köln", "150,00 €", "Gerichtsgebühr, nicht steuerbar")

    # ------------------------------------------------------------ Q4/2025
    best = letter_pages(Party("Möbelhaus Brenner GmbH", "Vogelsanger Straße 300", "50827", "Köln", phone="0221 5544 100"), ME.address_lines(),
                        meta=[["Bestellnummer", "BR-EK-2025-1187"], ["Datum", "20.10.2025"]], title="Bestellung: Einführung ERP-System",
                        font="Helvetica", accent="#0f766e",
                        paras=["Sehr geehrter Herr Kern,", "auf Grundlage Ihres Angebots vom 13.10.2025 beauftragen wir Sie mit der Einführung unseres ERP-Systems (Projektleitung, Datenmigration, Schulung) zum Festpreis von 24.000,00 € zzgl. Umsatzsteuer.",
                               {"type": "table", "widths": ["1fr", 40], "align": ["l", "r"], "header": ["Zahlungsplan", "netto"], "rows": [
                                   ["1. Abschlag bei Auftrag (Rechnung Anfang November 2025)", "8.000,00 €"],
                                   ["2. Abschlag nach Datenmigration (Januar 2026)", "8.000,00 €"],
                                   ["Schlussrechnung nach Abnahme (geplant April 2026)", "8.000,00 €"]]},
                               "Mit freundlichen Grüßen", "Möbelhaus Brenner GmbH, Einkauf"])
    n = "2025-10-20_Brenner_Bestellung_ERP.pdf"
    compile_doc(T.path(F2, n), "Bestellung ERP", "Möbelhaus Brenner", best)
    T.add(F2, n, "2025-10-20", "PDF", "Möbelhaus Brenner GmbH", "24.000,00 € netto",
          "Kein Buchungsbeleg: Grundlage für den Rechnungsverbund (Abschläge, Schlussrechnung)")

    # ------------------------------------------------------------ Q1/2026
    akad = Invoice("AR-26-0211", "2026-02-12", AKADEMIE, ME_V, [
        Line("Online-Seminar „Agiles Projektmanagement mit Scrum“, 10.–11.02.2026, Teilnehmer Tobias Kern", 1, D("390.00"), unit_label="Teiln."),
    ], delivery="2026-02-11", due="2026-02-26", means={"code": "58", "iban": AKADEMIE.iban, "bic": AKADEMIE.bic, "account_name": AKADEMIE.name},
        terms="Zahlbar innerhalb von 14 Tagen.")
    efile(T, F3, "2026-02-12_Akademie-Rheinland_AR-26-0211.pdf", akad, fmt="zugferd", issuer="Akademie Rheinland GmbH",
          case="Fortbildung, E-Rechnung", pdf_kwargs=dict(font="Gill Sans", accent="#1e40af", tagline="Weiterbildung für Unternehmen"))

    # ------------------------------------------------------------ Q2/2026
    abn = [{"kind": "a4", "font": "Helvetica", "accent": "#0f766e", "footer": [], "blocks": [
        {"type": "title", "text": "Abnahmeprotokoll", "sub": "Projekt ERP-Einführung, Bestellung BR-EK-2025-1187"},
        {"type": "kv", "rows": [["Auftraggeber", "Möbelhaus Brenner GmbH, Köln"], ["Auftragnehmer", ME.name],
                                ["Leistung", "Einführung ERP-System: Projektleitung, Datenmigration, Schulung von 14 Mitarbeitenden"],
                                ["Tag der Abnahme", "24.04.2026"], ["Ergebnis", "Abgenommen ohne wesentliche Mängel. Zwei Restpunkte (Druckvorlagen) bis 08.05.2026."]]},
        {"type": "sig", "items": [{"label": "Möbelhaus Brenner GmbH, Leitung IT", "sig": "R. Brenner"}, {"label": "Tobias Kern", "sig": "T. Kern"}]}]}]
    n = "2026-04-24_Brenner_Abnahmeprotokoll.pdf"
    compile_doc(T.path(F4, n), "Abnahmeprotokoll", "Möbelhaus Brenner", abn)
    T.add(F4, n, "2026-04-24", "PDF", "Brenner / Waldkauz", "–", "Leistungsnachweis für die Schlussrechnung (Leistungsdatum 24.04.2026)")

    rw = Invoice("26-0522-17", "2026-05-22", RHEINWERK, ME_B, [
        Line("Tagespass Flex Desk, Mai 2026", 5, D("25.00"), unit="DAY", unit_label="Tag"),
    ], delivery="2026-05-22")
    efile(T, F4, "2026-05-22_Rheinwerk-Coworking_26-0522-17.pdf", rw, fmt="pdf", issuer="Rheinwerk Coworking GmbH",
          case="Kleinbetragsrechnung (≤ 250 €), überwiesen",
          pdf_kwargs=dict(font="Futura", accent="#be123c", tagline="Arbeiten am Rhein", pay_lines=["Bitte überweisen Sie den Betrag bis 05.06.2026."]))

    gfb = [{"kind": "a4", "font": "Georgia", "accent": "#222222", "footer": [], "blocks": [
        {"type": "title", "text": "Gesellschafterbeschluss", "sub": ME.name},
        {"type": "para", "justify": True, "text": "Der alleinige Gesellschafter Tobias Kern beschließt unter Verzicht auf Form und Frist:"},
        {"type": "para", "justify": True, "text": "1. Der Geschäftsführer Tobias Kern erhält ab dem 01.07.2026 ein Gehalt von 4.000,00 € brutto monatlich, zahlbar zum Monatsende. "
         "Weitere Bezüge (Tantieme, Sachbezüge) werden nicht gewährt. 2. Der Geschäftsführer erstattet Fahrten mit dem Privat-Pkw mit 0,30 € je Kilometer gegen Abrechnung."},
        {"type": "para", "text": "Köln, den 15.06.2026"},
        {"type": "sig", "items": [{"label": "Tobias Kern (Gesellschafter)", "sig": "T. Kern"}]}]}]
    n = "2026-06-15_Gesellschafterbeschluss_GF-Verguetung.pdf"
    compile_doc(T.path(F4, n), "Gesellschafterbeschluss GF-Vergütung", ME.name, gfb)
    T.add(F4, n, "2026-06-15", "PDF", "Gesellschafter", "4.000,00 € / Monat", "Kein Buchungsbeleg: Grundlage für Lohn und Reisekosten (Fremdvergleich)")

    # ------------------------------------------------------------ Q3/2026
    vers = letter_pages(VERS, ME.address_lines(), meta=[["Versicherungsschein", "VS-4471-2290"], ["Datum", "01.07.2026"]],
                        title="Beitragsrechnung", font="Helvetica Neue", accent="#0e7490",
                        paras=["Betriebs- und Vermögensschadenhaftpflichtversicherung für IT-Dienstleister.",
                               {"type": "table", "widths": ["1fr", 30], "align": ["l", "r"], "header": ["Position", "Betrag"], "rows": [
                                   ["Jahresbeitrag 01.07.2026 – 30.06.2027 (netto)", "1.200,00 €"], ["Versicherungsteuer 19 %", "228,00 €"]]},
                               {"type": "totals", "rows": [["Zu zahlen", "1.428,00 €", True]]},
                               f"Fällig am 15.07.2026. Bitte überweisen Sie auf IBAN {fmt_iban(VERS.iban)} unter Angabe des Versicherungsscheins.",
                               "Versicherungsleistungen sind nach § 4 Nr. 10 UStG umsatzsteuerfrei; die Versicherungsteuer ist keine Umsatzsteuer und nicht als Vorsteuer abziehbar."],
                        footer=[["Rheinische Testversicherung AG", "Konrad-Adenauer-Ufer 41, 50668 Köln"], ["Amtsgericht Köln HRB 1188"]])
    n = "2026-07-01_Rheinische-Testversicherung_Beitragsrechnung.pdf"
    compile_doc(T.path(F5, n), "Beitragsrechnung", "Rheinische Testversicherung AG", vers)
    T.add(F5, n, "2026-07-01", "PDF", "Rheinische Testversicherung AG", "1.428,00 €",
          "Versicherung ohne Vorsteuer; Jahresbeitrag bis 30.06.2027 → aktive Rechnungsabgrenzung 9/12 = 1.071,00 € zum 30.09.")

    best2 = letter_pages(Party("Domstadt Immobilien GmbH", "Bayenthalgürtel 13", "50968", "Köln"), ME.address_lines(),
                         meta=[["Auftrag", "DI-2026-044"], ["Datum", "20.07.2026"]], title="Auftrag: Einführung Dokumentenmanagement",
                         font="Helvetica", accent="#9a3412",
                         paras=["Sehr geehrter Herr Kern,", "wir beauftragen Sie mit der Einführung eines Dokumentenmanagementsystems zum Festpreis von 18.000,00 € netto. "
                                "Start 03.08.2026, Abnahme geplant 30.11.2026. Zahlungsplan: Abschlag 6.000,00 € netto bei Projektstart, Rest nach Abnahme.",
                                "Mit freundlichen Grüßen", "Domstadt Immobilien GmbH"])
    n = "2026-07-20_Domstadt_Auftrag_DMS.pdf"
    compile_doc(T.path(F5, n), "Auftrag DMS", "Domstadt Immobilien", best2)
    T.add(F5, n, "2026-07-20", "PDF", "Domstadt Immobilien GmbH", "18.000,00 € netto", "Kein Buchungsbeleg: Grundlage für Verbund und unfertige Leistung")

    for m in ("2026-07", "2026-08", "2026-09"):
        lohnjournal(m, F5)

    ins = letter_pages(Party("Dr. Felix Brandt, Rechtsanwalt, Insolvenzverwalter", "Königsallee 60", "40212", "Düsseldorf"), ME.address_lines(),
                       meta=[["Az. Insolvenzgericht", "501 IN 88/26"], ["Unser Zeichen", "FB-KK-0441"], ["Datum", "04.09.2026"]],
                       title="Insolvenzverfahren über das Vermögen der Kleinkram Werbeagentur GmbH", font="Georgia", accent="#374151",
                       paras=["Sehr geehrte Damen und Herren,",
                              "das Amtsgericht Düsseldorf hat am 01.09.2026 das Insolvenzverfahren über das Vermögen der Kleinkram Werbeagentur GmbH, "
                              "Kaiserswerther Straße 115, 40474 Düsseldorf, eröffnet und mich zum Insolvenzverwalter bestellt.",
                              "Nach den Unterlagen der Schuldnerin steht Ihnen eine Forderung aus der Rechnung RE-2025-0006 vom 29.05.2026 über 4.188,80 € zu. "
                              "Bitte melden Sie Forderungen bis zum 15.10.2026 schriftlich zur Insolvenztabelle an (§ 174 InsO). Zahlungen an Sie aus dem "
                              "Vermögen der Schuldnerin sind nicht mehr möglich.",
                              "Nach derzeitigem Stand rechne ich mit einer Quote von etwa 3 bis 5 Prozent; eine Ausschüttung ist nicht vor 2028 zu erwarten.",
                              "Mit freundlichen Grüßen", "Dr. Felix Brandt, Insolvenzverwalter"])
    n = "2026-09-04_Insolvenzverwalter_Kleinkram.pdf"
    compile_doc(T.path(F5, n), "Insolvenzverfahren Kleinkram", "Dr. Felix Brandt", ins)
    T.add(F5, n, "2026-09-04", "PDF", "Insolvenzverwalter Dr. Brandt", "Forderung 4.188,80 €",
          "Forderung uneinbringlich: USt-Berichtigung 668,80 € in III/2026 (§ 17 Abs. 2 Nr. 1 UStG), Wertberichtigung zum 30.09.")

    weber = Invoice("2026-019", "2026-09-30", WEBER, ME_B, [
        Line("IT-Beratung Projekt DMS-Einführung Domstadt Immobilien, Leistungszeitraum 01.–30.09.2026", 6, D("650.00"), unit="DAY", unit_label="Tag"),
    ], period=("2026-09-01", "2026-09-30"), due="2026-10-14",
        means={"code": "58", "iban": WEBER.iban, "bic": WEBER.bic, "account_name": "Jonas Weber"}, terms="Zahlbar innerhalb von 14 Tagen.")
    efile(T, F5, "2026-09-30_Jonas-Weber_Rechnung_2026-019.pdf", weber, fmt="pdf", issuer="Jonas Weber IT-Beratung",
          case="Fremdleistung für das laufende Projekt (5906); zum 30.09. offen; Teil der unfertigen Leistung",
          pdf_kwargs=dict(font="Helvetica", accent="#334155", tagline="Freiberuflicher IT-Berater",
                          extra_notes=["Leistungsnachweis: Stundenzettel September 2026 liegt dem Projektordner bei."]))

    rk_rows = [[a, b, str(c), de(r2(D(c) * D("0.30")))] for a, b, c in KM]
    rk = [{"kind": "a4", "font": "Helvetica", "accent": "#3f6212", "footer": [], "blocks": [
        {"type": "title", "text": "Reisekostenabrechnung III/2026", "sub": f"{ME.name} · Geschäftsführer Tobias Kern · Privat-Pkw K-TK 2207"},
        {"type": "table", "widths": [22, "1fr", 14, 22], "align": ["l", "l", "r", "r"], "header": ["Datum", "Anlass, Strecke", "km", "€ (0,30/km)"], "rows": rk_rows},
        {"type": "totals", "rows": [[f"Summe {KM_TOTAL} km × 0,30 €", de(KM_EUR, "€"), True]]},
        {"type": "para", "text": "Erstattung nach Gesellschafterbeschluss vom 15.06.2026; steuerfrei nach § 3 Nr. 16 EStG. Auszahlung auf IBAN " + fmt_iban(TOBIAS_IBAN) + "."},
        {"type": "sig", "items": [{"label": "Köln, 30.09.2026 · Tobias Kern", "sig": "T. Kern"}]}]}]
    n = "2026-09-30_Reisekostenabrechnung_Q3-2026.pdf"
    compile_doc(T.path(F5, n), "Reisekostenabrechnung", ME.name, rk)
    T.add(F5, n, "2026-09-30", "PDF (Eigenbeleg)", "Tobias Kern", de(KM_EUR, "€"),
          "Kilometergeld ohne Vorsteuer (6663 bzw. 6670), Erstattungsschuld gegenüber dem Gesellschafter-GF (3640), gezahlt erst am 02.10.")

    wip = [{"kind": "a4", "font": "Helvetica", "accent": "#9a3412", "footer": [], "blocks": [
        {"type": "title", "text": "Projektstatus zum 30.09.2026", "sub": "DMS-Einführung Domstadt Immobilien GmbH, Auftrag DI-2026-044 · Bewertung der unfertigen Leistung"},
        {"type": "kv", "rows": [["Festpreis", "18.000,00 € netto"], ["Fertigstellungsgrad", "ca. 40 % (Konzept und Migration Teil 1 abgeschlossen)"],
                                ["Abnahme", "geplant 30.11.2026"], ["Erhaltene Anzahlung", "6.000,00 € netto (RE-2025-0008, bezahlt 17.08.2026)"]]},
        {"type": "table", "widths": ["1fr", 26, 26, 30], "align": ["l", "r", "r", "r"], "header": ["Kostenart", "Menge", "Satz", "Herstellungskosten"], "rows": [
            ["Eigenleistung Geschäftsführer (Gehalt 4.000 € / 20 Arbeitstage)", "16 Tage", "200,00 €", "3.200,00 €"],
            ["Fremdleistung Jonas Weber (RE 2026-019)", "6 Tage", "650,00 €", "3.900,00 €"],
            [{"t": "Summe Herstellungskosten (ohne Gewinnaufschlag, ohne Vertriebskosten)", "b": True}, "", "", {"t": "7.100,00 €", "b": True}]]},
        {"type": "para", "text": "Verlustfreie Bewertung: Erwartete Restkosten 7.500 €, Restumsatz 12.000 € netto, kein drohender Verlust."},
        {"type": "sig", "items": [{"label": "Köln, 30.09.2026 · Tobias Kern", "sig": "T. Kern"}]}]}]
    n = "2026-09-30_Projektstatus_Domstadt_unfertige-Leistung.pdf"
    compile_doc(T.path(F5, n), "Projektstatus", ME.name, wip)
    T.add(F5, n, "2026-09-30", "PDF (Inventurbeleg)", "Tobias Kern", "7.100,00 €",
          "Unfertige Leistung zum Stichtag (1080 bzw. Bestandsveränderung 4815)")

    # ------------------------------------------------------------ Oktober 2026
    ihk = letter_pages(Party("IHK Köln", "Unter Sachsenhausen 10–26", "50667", "Köln", phone="0221 1640-0"), ME.address_lines(),
                       meta=[["Mitgliedsnummer", "71 4471 22"], ["Datum", "05.10.2026"]], title="Beitragsbescheid 2025 und 2026",
                       font="Helvetica Neue", accent="#7a1e2c",
                       paras=["Sehr geehrte Damen und Herren,", "Ihr Unternehmen ist seit dem 20.10.2025 kammerzugehörig. Wir setzen folgende Beiträge fest:",
                              {"type": "table", "widths": ["1fr", 30], "align": ["l", "r"], "header": ["Beitragsart", "Betrag"], "rows": [
                                  ["Grundbeitrag 2025 (anteilig, ab Oktober)", "30,00 €"], ["Grundbeitrag 2026", "120,00 €"],
                                  ["Umlage 2026 (vorläufig, Gewerbeertrag nicht bekannt)", "0,00 €"]]},
                              {"type": "totals", "rows": [["Festgesetzt", "150,00 €", True]]},
                              "Fällig am 05.11.2026. Die Beiträge unterliegen nicht der Umsatzsteuer."])
    n = "2026-10-05_IHK-Koeln_Beitragsbescheid.pdf"
    compile_doc(T.path(F6, n), "Beitragsbescheid", "IHK Köln", ihk)
    T.add(F6, n, "2026-10-05", "PDF", "IHK Köln", "150,00 €",
          "Bescheid nach dem Stichtag für Zeiträume davor → Rückstellung oder Verbindlichkeit zum 30.09.?")

    # ------------------------------------------------------------ Bank
    B = dict(owner=ME.name, iban=ME.iban, bic=ME.bic, bank=ME.bank)
    fee = lambda date, q: Tx(date, D("-26.70"), "Rheinland Testbank eG", "", f"Rechnungsabschluss {q} Kontofuehrung 3 x 8,90", kind="Entgelt")
    fa = lambda date, amt, text: Tx(date, amt, "Finanzamt Koeln-Mitte", FA_IBAN, f"StNr 215/5712/3456 {text}", kind="Lastschrift", mandate="FA5215-5712-3456")
    sal = lambda date, m: Tx(date, D("-3305.75"), "Tobias Kern", TOBIAS_IBAN, f"Gehalt {m}", kind="Lohn")

    q4 = D("1520.00") + D("2006.40") - (notar.tax + dom.tax)
    q1 = D("1520.00") + D("2508.00") - akad.tax
    q2 = D("1520.00") + D("668.80") + D("1672.00") - rw.tax
    q3 = D("1140.00") - D("668.80") - weber.tax
    lst3 = D("694.25") * 3

    s1 = Statement(**B, seq=1, start="2025-10-02", end="2025-12-31", opening=D("0"), txs=[
        Tx("2025-10-06", D("3000.00"), "Tobias Kern", TOBIAS_IBAN, "Einzahlung Stammkapital Waldkauz Projekte UG haftungsbeschraenkt i.G."),
        Tx("2025-10-13", -dom.gross, DOMTECH.name, DOMTECH.iban, "RE 2025-30981 Kd 88412"),
        Tx("2025-10-15", -notar.gross, NOTAR.name, NOTAR.iban, "KB 1544/2025"),
        Tx("2025-11-03", D("-150.00"), "Landeskasse NRW", LK_IBAN, "Kassenzeichen 2510 4471 3302 HRB 123987"),
        Tx("2025-11-14", D("9520.00"), BRENNER.name, BRENNER.iban, "1. Abschlag ERP RE-2025-0001 Best. BR-EK-2025-1187"),
        fee("2025-12-31", "IV/2025")])
    s2 = Statement(**B, seq=2, start="2026-01-01", end="2026-03-31", opening=s1.closing, txs=[
        Tx("2026-01-09", D("12566.40"), RHEINF.name, RHEINF.iban, "RE-2025-0002"),
        fa("2026-01-12", -q4, "USt IV/2025"),
        Tx("2026-01-27", D("9520.00"), BRENNER.name, BRENNER.iban, "2. Abschlag ERP RE-2025-0003"),
        Tx("2026-02-20", -akad.gross, AKADEMIE.name, AKADEMIE.iban, "AR-26-0211 Seminar Kern"),
        fee("2026-03-31", "I/2026")])
    s3 = Statement(**B, seq=3, start="2026-04-01", end="2026-06-30", opening=s2.closing, txs=[
        fa("2026-04-10", -q1, "USt I/2026"),
        Tx("2026-04-22", D("15708.00"), RHEINF.name, RHEINF.iban, "RE-2025-0004"),
        Tx("2026-05-18", D("9520.00"), BRENNER.name, BRENNER.iban, "Schlussrechnung ERP RE-2025-0005"),
        Tx("2026-05-26", -rw.gross, RHEINWERK.name, RHEINWERK.iban, "Rg 26-0522-17 Tagespaesse Mai"),
        fee("2026-06-30", "II/2026")])
    s4 = Statement(**B, seq=4, start="2026-07-01", end="2026-09-30", opening=s3.closing, txs=[
        fa("2026-07-10", -q2, "USt II/2026"),
        Tx("2026-07-15", D("-1428.00"), VERS.name, VERS.iban, "VS-4471-2290 Beitrag 07/2026-06/2027"),
        Tx("2026-07-22", D("10472.00"), RHEINF.name, RHEINF.iban, "RE-2025-0007"),
        sal("2026-07-31", "07/2026"),
        Tx("2026-08-17", D("7140.00"), DOMSTADT.name, DOMSTADT.iban, "Abschlag DMS RE-2025-0008 Auftrag DI-2026-044"),
        sal("2026-08-31", "08/2026"),
        sal("2026-09-30", "09/2026"), fee("2026-09-30", "III/2026")])
    s5 = Statement(**B, seq=5, start="2026-10-01", end="2026-10-16", opening=s4.closing, txs=[
        Tx("2026-10-02", -KM_EUR, "Tobias Kern", TOBIAS_IBAN, "Reisekosten III/2026"),
        Tx("2026-10-09", -weber.gross, WEBER.name, WEBER.iban, "RE 2026-019"),
        fa("2026-10-12", -lst3, "LSt III/2026")])
    for folder, st in [(F2, s1), (F3, s2), (F4, s3), (F5, s4), (F6, s5)]:
        name = f"{st.end}_Kontoauszug_{st.start[:7]}_bis_{st.end[:7]}_camt053.xml"
        T.write(folder, name, camt(st))
        T.add(folder, name, st.end, "CAMT.053", ME.bank, f"{len(st.txs)} Umsätze, Endsaldo {de(st.closing, '€')}",
              "Kontoauszug" + (" – den ersten liest der Mensch in der App ein" if st.seq == 1 else ""))
    return readme(locals())


def readme(v) -> str:
    s = [v[f"s{i}"] for i in range(1, 6)]
    q4, q1, q2, q3, lst3 = v["q4"], v["q1"], v["q2"], v["q3"], v["lst3"]
    return f"""# Mandant 3: Waldkauz Projekte UG (haftungsbeschränkt) (Köln)

**Schwerpunkt: abweichendes Geschäftsjahr und Jahresabschluss.** Eine UG mit 3.000 € Stammkapital, gegründet am
01.10.2025, Geschäftsjahr 01.10.–30.09. Das erste Geschäftsjahr ist am 30.09.2026 zu Ende; der Abschluss steht jetzt an.
Projektgeschäft mit Abschlagsrechnungen und Schlussrechnung (Mindest-Istversteuerung), eine Kundeninsolvenz, ein
Projekt über den Stichtag, Versicherung im Voraus, Reisekosten des Gesellschafter-Geschäftsführers, Rückstellungen,
Steuerrückstellung und die gesetzliche Rücklage der UG.

Weil das Jahr in der Vergangenheit liegt, stellt der Agent die Ausgangsrechnungen mit dem damaligen Datum aus.
Die Reihenfolge unten ergibt die Nummern RE-2025-0001 bis RE-2025-0008 (Buchfink zählt das Geschäftsjahr 2025/26
als „2025“); die Kontoauszüge nennen genau diese Nummern.

## 1. Einrichtung in der App

| Feld im Einrichtungsassistenten | Wert |
|---|---|
| Firma | Waldkauz Projekte UG (haftungsbeschränkt) |
| Rechtsform | UG (haftungsbeschränkt) |
| Wie beginnen die Bücher? | Neu gegründet |
| Tag der Beurkundung | 01.10.2025 |
| Geschäftsjahr | beginnt im Oktober (01.10.–30.09.) |
| Bundesland | Nordrhein-Westfalen |

**Unternehmensdaten:**

| Feld | Wert |
|---|---|
| Anschrift | Venloer Straße 241, 50823 Köln |
| Geschäftsführung, Ansprechpartner | Tobias Kern, 0221 9874 330, kontakt@waldkauz-projekte.example |
| Register | Amtsgericht Köln HRB 123987, eingetragen 20.10.2025 |
| Notar, Urkunde | Dr. Annette Kühn, Köln; UR-Nr. 1544/2025 K |
| Steuernummer, Finanzamt | 215/5712/3456, Finanzamt Köln-Mitte |
| USt-IdNr. | {ME.vat_id} |
| Voranmeldung | vierteljährlich, keine Dauerfristverlängerung |
| Lohnsteuer | vierteljährlich |
| Gewerbesteuer-Hebesatz | Köln 475 % |
| Steuerliche Beratung | keine |
| Bankkonto | {ME.bank}, IBAN {fmt_iban(ME.iban)}, BIC {ME.bic}, Konto für Rechnungen |

**Gründung:** Stammkapital 3.000 €, Geschäftsanteil Nr. 1 Tobias Kern (100 %), sofort voll in Geld; Gründungsaufwand
bis 900 €. Tobias Kern als Gesellschafter-Kontakt anlegen (Reisekosten, Gehalt).

## 2. Testschritte

Die Belege gelten als am Belegdatum eingegangen (`eingang_am`), sonst landet die Vorsteuer im Oktober 2026.

### Schritt 1 – Gründung (`{F1}/`)

{T.table(F1)}

Erwartet: Zeichnung 1298 an 2900 3.000,00 (UG: alles sofort eingefordert). Die Pflicht „Eröffnungsbilanz aufstellen und
unterzeichnen“ war am 01.01.2026 fällig (drei Monate nach der Beurkundung, keine Verschiebung) und ist überfällig; die
E-Bilanz der Eröffnungsbilanz gehört zur E-Bilanz des ersten Geschäftsjahres (Veranlagungszeitraum 2026, fällig 02.08.2027). Notebook 1.149 € als Anlage, Monitor
289 € als GWG. Gründungskosten 456,50 € netto bleiben unter 900 €.

### Schritt 2 – IV/2025 (`{F2}/`)

{T.table(F2)}

Ausgangsrechnungen (Agent, mit Vorschau und Rückfrage, damaliges Datum):

| Nr. (erwartet) | Art | Kunde | Datum | Inhalt | Brutto |
|---|---|---|---|---|---:|
| RE-2025-0001 | Abschlag, neuer Verbund „ERP-Einführung“ 24.000 € netto, 19 % | Möbelhaus Brenner GmbH, Vogelsanger Straße 300, 50827 Köln, USt-IdNr. {BRENNER.vat_id} | 03.11.2025 | 1. Abschlag 8.000 € netto | 9.520,00 € |
| RE-2025-0002 | Rechnung | Rheinfähre Logistik GmbH, Deutz-Mülheimer Straße 51, 51063 Köln, USt-IdNr. {RHEINF.vat_id} | 19.12.2025 | „Prozessberatung Lager“, 12 Tage × 880 €, Leistung 03.11.–18.12.2025 | 12.566,40 € |

Voranmeldung IV/2025: USt 3.526,40 (Abschlag nach Zahlung am 14.11., Mindest-Istversteuerung), Vorsteuer
{de(v['notar'].tax + v['dom'].tax, '€')}, Zahllast **{de(q4, '€')}** (Lastschrift 12.01.2026). Endsaldo 31.12.2025: **{de(s[0].closing, '€')}**.

### Schritt 3 – I/2026 (`{F3}/`)

{T.table(F3)}

Ausgangsrechnungen: **RE-2025-0003** 2. Abschlag Brenner, 15.01.2026, 8.000 € netto (9.520,00 €);
**RE-2025-0004** Rheinfähre, 31.03.2026, 15 Tage × 880 €, Leistung 07.01.–27.03.2026 (15.708,00 €).

Voranmeldung I/2026: Zahllast **{de(q1, '€')}**. Endsaldo 31.03.: **{de(s[1].closing, '€')}**.

### Schritt 4 – II/2026 (`{F4}/`)

{T.table(F4)}

Ausgangsrechnungen: **RE-2025-0005** Schlussrechnung Brenner, 30.04.2026, Leistung 24.04.2026, verrechnet beide
Abschläge, Zahlbetrag 9.520,00 €; **RE-2025-0006** Kleinkram Werbeagentur GmbH, Kaiserswerther Straße 115, 40474
Düsseldorf, USt-IdNr. {KLEINKRAM.vat_id}, 29.05.2026, „Kampagnenplanung Tooling“ 4 Tage × 880 € (4.188,80 €, wird nie
bezahlt); **RE-2025-0007** Rheinfähre, 30.06.2026, 10 Tage × 880 €, Leistung 01.04.–26.06.2026 (10.472,00 €).

Voranmeldung II/2026: Zahllast **{de(q2, '€')}**. Endsaldo 30.06.: **{de(s[2].closing, '€')}**.

### Schritt 5 – III/2026 (`{F5}/`)

{T.table(F5)}

Ausgangsrechnung: **RE-2025-0008** Abschlag, neuer Verbund „DMS-Einführung“ 18.000 € netto bei Domstadt Immobilien
GmbH, Bayenthalgürtel 13, 50968 Köln, USt-IdNr. {DOMSTADT.vat_id}, 03.08.2026, 6.000 € netto (7.140,00 €).

Voranmeldung III/2026: USt 1.140,00 (Abschlag Domstadt) − 668,80 (Kleinkram uneinbringlich), Vorsteuer 741,00 (Weber),
Ergebnis **{de(q3, '€')}** (Erstattung, nicht mehr im Auszug). Endsaldo 30.09.: **{de(s[3].closing, '€')}**.

### Schritt 6 – Oktober 2026 und Jahresabschluss (`{F6}/`)

{T.table(F6)}

Danach den Abschluss durchgehen (`abschluss_status`, Skill `buchfink-jahresabschluss`). Erwartete Bausteine:

| Baustein | Erwartung |
|---|---|
| Bank zum 30.09. | 1800 = {de(s[3].closing, '€')} |
| Abschreibung | Notebook 1.149 € ab 10/2025 (Nutzungsdauer 3 Jahre oder 1 Jahr als digitales Wirtschaftsgut); Monitor GWG |
| Rechnungsabgrenzung | Versicherung 1.428 € × 9/12 = 1.071,00 € auf 1900 |
| Unfertige Leistung | Domstadt 7.100,00 € (Beleg Projektstatus); erhaltene Anzahlung 6.000 € netto |
| Forderung Kleinkram | netto 3.520 €; Quote 3–5 % → Wertberichtigung rund 3.344 € (95 %); USt 668,80 € schon in III/2026 berichtigt |
| Verbindlichkeiten | Weber 4.641,00 €; Reisekosten Tobias Kern {de(KM_EUR, '€')} (3640); Lohnsteuer III 2.082,75 € |
| Rückstellungen | Abschlusskosten (z. B. 300 €, 3095), Aufbewahrung (z. B. 250 €, 3070), IHK 150 € (Bescheid nach dem Stichtag) |
| Umsatzsteuer | Verrechnung; Erstattungsanspruch III/2026 {de(-q3, '€')} |
| Steuerrückstellung | KSt 15 % + SolZ, GewSt mit Hebesatz 475 % |
| UG-Rücklage | 25 % des Jahresüberschusses auf 2930 (kein Verlustvortrag) |
| Aufstellen, Feststellen, Offenlegen | nur der Mensch in der App; danach `ergebnisverwendung_buchen` |

Prüfpunkte, an denen Buchfink bei einem abweichenden Geschäftsjahr leicht danebenliegt:

- Das Geschäftsjahr heißt in Buchfink „2025“. Körperschaft- und Gewerbesteuer gehören aber in den
  Veranlagungszeitraum **2026** (Wirtschaftsjahr endet 2026, § 7 Abs. 4 KStG). Erklärungsjahr, Vordrucke und Fristen
  (31.07.2027 ohne Berater) prüfen.
- E-Bilanz: Wirtschaftsjahr beginnt am 01.10.2025. Welche Taxonomie wählt Buchfink (6.9 oder 6.10)?
- Größenklasse: Kleinstkapitalgesellschaft; Fristen für Aufstellung, Feststellung (§ 42a GmbHG) und Offenlegung ab 30.09.2026.
- Die Voranmeldungen laufen nach Kalendervierteljahren quer zum Geschäftsjahr.
- `fristen` zeigt nach dem 30.09.2026 die Pflichten für den Abschluss des ersten Geschäftsjahres.

## 3. Kontrollwerte

| Stand | Erwartung |
|---|---|
| 1800 zum 30.09.2026 | {de(s[3].closing, '€')} |
| 1800 zum 16.10.2026 | {de(s[4].closing, '€')} |
| Umsatzerlöse 2025/26 netto | 24.000 + 10.560 + 13.200 + 8.800 + 3.520 = 60.080,00 € (Abschlag Domstadt bleibt Anzahlung) |
| Personalaufwand | 3 × 4.000 = 12.000,00 € |
| Offene Posten zum 30.09. | Forderung Kleinkram 4.188,80 € (wertberichtigt); Verbindlichkeit Weber 4.641,00 € |
"""
