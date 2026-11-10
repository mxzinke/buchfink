"""Mandant 2: Kranich Software GmbH, München. Laufender Betrieb von Juni bis Oktober 2026."""

import shutil

from common import *

T = Tenant("2-Kranich-Software-GmbH")
F1, F2, F3, F4, F5 = "01-Gruendung-Juni", "02-Juli", "03-August", "04-September", "05-Oktober"

ME = Party("Kranich Software GmbH", "Schwanthalerstraße 64", "80336", "München",
           vat_id=de_vat("28374615"), tax_no="143/112/50117", email="buchhaltung@kranich-software.example",
           phone="089 4523 8810", contact="Dr. Mira Kranich",
           iban=iban("DE", "70069911" + "0004457120"), bic="GENODEF1ITB", bank="Isartal Testbank eG",
           register="Amtsgericht München HRB 291837", gf="Dr. Mira Kranich", web="kranich-software.example")
ME_IG = Party(ME.name + " i. G.", ME.street, ME.zip, ME.city)
ME_B = Party(ME.name, ME.street, ME.zip, ME.city)
ME_V = Party(ME.name, ME.street, ME.zip, ME.city, vat_id=ME.vat_id, email=ME.email)
CARD = "8812"

MIRA_IBAN = iban("DE", "70150000" + "0071442388")
LUKAS_IBAN = iban("DE", "70020270" + "0015532901")

ISAR = Party("Isarhost GmbH", "Landsberger Straße 155", "80687", "München", vat_id=de_vat("26173948"),
             email="rechnung@isarhost.example", phone="089 2154 990", contact="Kundenservice Abrechnung",
             iban=iban("DE", "70150000" + "1002233440"), bic="SSKMDEMMXXX", bank="Testsparkasse München",
             register="Amtsgericht München HRB 201554", gf="Florian Wimmer", web="isarhost.example")
ISAR_CI, ISAR_MANDATE = creditor_id("00000771234"), "IH-K-50331"
APFEL = Party("Apfelgrün IT-Handel GmbH", "Leopoldstraße 82", "80802", "München", vat_id=de_vat("13958264"),
              email="rechnung@apfelgruen.example", phone="089 3399 410", contact="Verkauf Business",
              iban=iban("DE", "70070010" + "0811402200"), bic="DEUTDEMMXXX", bank="Testbank München",
              register="Amtsgericht München HRB 177302", gf="Sven Aigner")
BRANDL = Party("Brandl Büromöbel GmbH", "Rosenheimer Straße 145", "81671", "München", vat_id=de_vat("21184735"),
               email="rechnung@brandl-bueromoebel.example", phone="089 4090 1170", contact="Theresa Brandl",
               iban=iban("DE", "70150000" + "0905177300"), bic="SSKMDEMMXXX", bank="Testsparkasse München",
               register="Amtsgericht München HRB 64410", gf="Theresa Brandl, Josef Brandl")
BRANDL_FAKE = Party(**{**BRANDL.__dict__, "email": "rechnung@brandl-buero-moebel.example",
                       "iban": iban("LT", "73000" + "10199882231"), "bic": "HABALT22XXX", "bank": "Baltic Testbank"})
WEISS = Party("Druckerei Weiß OHG", "Lindwurmstraße 129", "80337", "München", vat_id=de_vat("16480297"),
              email="auftrag@druckerei-weiss.example", phone="089 7214 33", contact="Maria Weiß",
              iban=iban("DE", "70190000" + "0003318840"), bic="GENODEF1M01", bank="Münchner Testbank eG",
              register="Amtsgericht München HRA 98112", gf="Anton Weiß, Maria Weiß (persönlich haftende Gesellschafter)")
MUCF = Party("MUC Fiber GmbH", "Elsenheimerstraße 45", "80687", "München", vat_id=de_vat("30195528"),
             email="e-rechnung@mucfiber.example", phone="089 9922 0", contact="Rechnungsservice",
             iban=iban("DE", "70050000" + "0004015500"), bic="BYLADEMMXXX", bank="Testlandesbank",
             register="Amtsgericht München HRB 255181", gf="Dr. Korbinian Schmid")
MUCF_CI, MUCF_MANDATE = creditor_id("00000928001"), "MF-7781230-01"
NTF = Party("Nordic Type Foundry AB", "Södermannagatan 22", "116 23", "Stockholm", country="SE", country_name="Schweden",
            vat_id=se_vat("556871234"), email="invoice@nordictype.example", contact="Elin Lindqvist", phone="+46 8 555 120 30",
            iban=iban("SE", "500" + "00000054391025512"), bic="ESSESESS", bank="Testbanken Stockholm")
CODEF = Party("Codeforge Inc.", "500 Howard Street, Suite 400", "CA 94105", "San Francisco", country="US",
              country_name="United States", email="billing@codeforge.example", register="EIN 88-4127730")
BEELD = Party("Beeldscherm Direct B.V.", "Weena 505", "3013 AL", "Rotterdam", country="NL", country_name="Niederlande",
              vat_id=nl_vat("85340917"), email="sales@beeldscherm-direct.example", register="KvK 61234870",
              iban=iban("NL", "RABO" + "0315220781"), bic="RABONL2U", bank="Testbank Rotterdam")
HOTEL = Party("Hafenkante Hotel GmbH", "Vorsetzen 53", "20459", "Hamburg", vat_id=de_vat("25618390"),
              email="rezeption@hafenkante-hotel.example", phone="040 3011 770",
              register="Amtsgericht Hamburg HRB 120477", gf="Lars Ohlsen")
VINO = Party("Vinothek Sendling GmbH", "Plinganserstraße 40", "81369", "München", vat_id=de_vat("27703915"),
             email="laden@vinothek-sendling.example", phone="089 7678 220",
             iban=iban("DE", "70150000" + "0090077120"), bic="SSKMDEMMXXX", bank="Testsparkasse München",
             register="Amtsgericht München HRB 188920", gf="Benedikt Rauch")
NOTAR = Party("Matthias Lechner, Notar", "Maximilianstraße 31", "80539", "München", tax_no="145/201/40117",
              email="kanzlei@notar-lechner.example", phone="089 2104 660",
              iban=iban("DE", "70020270" + "0062114500"), bic="HYVEDEMMXXX", bank="Testbank München")
LJK_IBAN = iban("DE", "77000000" + "0077001500")
FA_IBAN = iban("DE", "70000000" + "0070001054")
BKK_IBAN = iban("DE", "70050000" + "0000611330")
ALPEN = Party("Alpenblick Versicherung AG", "Leopoldstraße 244", "80807", "München", vat_id=de_vat("12907314"),
              iban=iban("DE", "70020270" + "0659013300"))
BERG = Party("Bergsteiger Outdoor GmbH", "Münchener Straße 18", "83022", "Rosenheim", vat_id=de_vat("20438817"))
NORDSO = Party("Nordsø Analytics ApS", "Store Kongensgade 68", "1264", "København K", country="DK", country_name="Dänemark",
               vat_id=dk_vat("3518472"), iban=iban("DK", "3000" + "0010479552"))


def isar(no, date, period, month_label, debit):
    return Invoice(no, date, ISAR, ME_B, [
        Line(f"Managed Server M (4 vCPU, 16 GB RAM), {month_label}", 1, D("79.00"), unit_label="Monat"),
        Line(f"Backup-Speicher 200 GB, {month_label}", 1, D("10.00"), unit_label="Monat"),
    ], period=period, due=debit, customer_no="K-50331",
        means={"code": "59", "debtor_iban": ME.iban, "mandate": ISAR_MANDATE, "creditor_id": ISAR_CI},
        terms=f"Der Betrag wird am {ddate(debit)} per SEPA-Lastschrift eingezogen (Mandat {ISAR_MANDATE}, Gläubiger-ID {ISAR_CI}).")


ISAR_STYLE = dict(font="Helvetica Neue", accent="#0f6cbd", tagline="Hosting aus München")


def lohnjournal(month: str, folder: str):
    mm, yyyy = month[5:], month[:4]
    rows = [
        ["1001", "Kranich, Dr. Mira", "GGF, SV-frei", "5.500,00", "1.128,41", "0,00", "0,00", "0,00", "4.371,59"],
        ["1002", "Bauer, Lukas", "StKl I, KV/RV/AV/PV", "3.800,00", "474,58", "0,00", "0,00", "826,50", "2.498,92"],
        [{"t": "", "b": True}, {"t": "Summe", "b": True}, "", {"t": "9.300,00", "b": True}, {"t": "1.602,99", "b": True},
         {"t": "0,00", "b": True}, {"t": "0,00", "b": True}, {"t": "826,50", "b": True}, {"t": "6.870,51", "b": True}],
    ]
    pages = [{
        "kind": "a4", "font": "Arial", "accent": "#475569", "footer": [["Lohnbüro Huber GmbH", "Bodenseestraße 4, 81241 München"], ["Mandant 4471 · Kranich Software GmbH"], ["Erstellt am " + ddate(f"{yyyy}-{mm}-25")]],
        "blocks": [
            {"type": "letterhead", "name": "Lohnbüro Huber GmbH", "size": 14, "tagline": "Lohn- und Gehaltsabrechnung", "right": ["Mandant 4471", "Kranich Software GmbH"]},
            {"type": "title", "text": f"Lohnjournal {mm}/{yyyy}", "sub": "Monatssummen je Arbeitnehmer · alle Beträge in Euro"},
            {"type": "table", "size": 8, "widths": [12, "1fr", 30, 20, 18, 12, 14, 20, 20], "align": ["l", "l", "l", "r", "r", "r", "r", "r", "r"],
             "header": ["PNr.", "Name", "Merkmale", "Brutto", "LSt", "SolZ", "KiSt", "SV-AN", "Netto"], "rows": rows},
            {"type": "h", "text": "Arbeitgeberanteile und Umlagen"},
            {"type": "table", "widths": ["1fr", 30], "align": ["l", "r"], "header": ["Position", "Betrag"], "rows": [
                ["Krankenversicherung AG (7,3 % + 1,45 %)", "332,50"], ["Rentenversicherung AG (9,3 %)", "353,40"],
                ["Arbeitslosenversicherung AG (1,3 %)", "49,40"], ["Pflegeversicherung AG (1,8 %)", "68,40"],
                [{"t": "Summe AG-Anteile Sozialversicherung", "b": True}, {"t": "803,70", "b": True}],
                ["Umlage U1 (1,1 %)", "41,80"], ["Umlage U2 (0,24 %)", "9,12"], ["Insolvenzgeldumlage (0,15 %)", "5,70"],
                [{"t": "Summe Umlagen", "b": True}, {"t": "56,62", "b": True}]]},
            {"type": "h", "text": "Zahlungen"},
            {"type": "table", "widths": ["1fr", 40, 30], "align": ["l", "l", "r"], "header": ["Empfänger", "fällig", "Betrag"], "rows": [
                ["Nettolöhne (2 Überweisungen)", "letzter Bankarbeitstag", "6.870,51"],
                ["BKK Testfink: SV-AN 826,50 + SV-AG 803,70 + Umlagen 56,62", "drittletzter Bankarbeitstag", "1.686,82"],
                ["Finanzamt München: Lohnsteuer", "10. des Folgemonats", "1.602,99"]]},
            {"type": "para", "size": 8.5, "text": "Dr. Mira Kranich ist beherrschende Gesellschafter-Geschäftsführerin und nicht sozialversicherungspflichtig (Statusfeststellung vom 12.06.2026). Werte erfunden, in sich stimmig."},
        ]}]
    name = f"{yyyy}-{mm}-25_Lohnbuero-Huber_Lohnjournal_{mm}-{yyyy}.pdf"
    compile_doc(T.path(folder, name), f"Lohnjournal {mm}/{yyyy}", "Lohnbüro Huber GmbH", pages)
    T.add(folder, name, f"{yyyy}-{mm}-25", "PDF", "Lohnbüro Huber GmbH", "Brutto 9.300,00 €",
          "Lohnjournal buchen (`lohnjournal_buchen`): GF-Gesellschafterin 6024, Gehalt 6020")


def card_statement(folder, date, debit, rows, total):
    pages = [{
        "kind": "a4", "font": "Helvetica Neue", "accent": "#1a1f71", "footer": [["Isartal Testbank eG", "Visa Business Card"], ["Kartenkonto 4917 •••• •••• 8812"], ["Fragen: 089 4040 500"]],
        "blocks": [
            {"type": "letterhead", "name": "Isartal Testbank eG", "size": 15, "tagline": "Visa Business Card", "right": ["Kartenabrechnung"]},
            {"type": "address", "sender_line": "Isartal Testbank eG · Postfach 20 11 55 · 80011 München", "lines": ME.address_lines(),
             "meta": [["Abrechnung vom", ddate(date)], ["Karte", "Visa Business •••• 8812"], ["Karteninhaberin", "Dr. Mira Kranich"]]},
            {"type": "title", "text": f"Kreditkartenabrechnung {date[5:7]}/{date[:4]}"},
            {"type": "table", "widths": [18, 18, "1fr", 26, 16, 24], "align": ["l", "l", "l", "r", "r", "r"],
             "header": ["Kauf", "Buchung", "Verwendung", "Fremdwährung", "Kurs", "EUR"], "rows": rows},
            {"type": "totals", "rows": [["Saldo dieser Abrechnung", de(total, "€"), True]]},
            {"type": "para", "text": f"Der Saldo wird am {ddate(debit)} von Ihrem Geschäftskonto IBAN {fmt_iban(ME.iban)} eingezogen."},
        ]}]
    name = f"{date}_Isartal-Testbank_Kreditkartenabrechnung_{date[5:7]}-{date[:4]}.pdf"
    compile_doc(T.path(folder, name), f"Kreditkartenabrechnung {date[5:7]}/{date[:4]}", "Isartal Testbank eG", pages)
    T.add(folder, name, date, "PDF", "Isartal Testbank eG", de(total, "€"),
          f"Monatsabrechnung der Firmenkarte (3610), Einzug am {ddate(debit)}; Auslandseinsatzentgelt als Bankgebühr")


def build() -> str:
    # ------------------------------------------------------------ 01 Juni
    vertrag = [{
        "kind": "a4", "font": "Georgia", "accent": "#222222", "footer": [],
        "blocks": [
            {"type": "lines", "lines": ["UR-Nr. 0934/2026 L", "Ausfertigung"], "size": 8.5},
            {"type": "spacer", "mm": 4},
            {"type": "title", "text": "Errichtung einer Einpersonen-GmbH", "sub": "Verhandelt zu München am 01. Juni 2026"},
            {"type": "para", "justify": True, "text": "Vor mir, Matthias Lechner, Notar in München, erschien heute Frau Dr. Mira Kranich, geb. 22.07.1984, "
             "wohnhaft Herzog-Wilhelm-Straße 9, 80331 München, ausgewiesen durch Personalausweis. Die Erschienene errichtet hiermit eine "
             "Gesellschaft mit beschränkter Haftung mit dem folgenden Gesellschaftsvertrag."},
            {"type": "h", "text": "Gesellschaftsvertrag der Kranich Software GmbH"},
            {"type": "para", "justify": True, "text": [{"t": "§ 1 Firma, Sitz. ", "b": True}, "Kranich Software GmbH mit Sitz in München."]},
            {"type": "para", "justify": True, "text": [{"t": "§ 2 Gegenstand. ", "b": True}, "Entwicklung von Software, IT-Beratung und Schulungen."]},
            {"type": "para", "justify": True, "text": [{"t": "§ 3 Stammkapital. ", "b": True}, "Das Stammkapital beträgt 25.000,00 €. Die Gründerin übernimmt den Geschäftsanteil Nr. 1 zu 25.000,00 €. Die Einlage ist in Geld zu leisten und sofort in voller Höhe fällig."]},
            {"type": "para", "justify": True, "text": [{"t": "§ 4 Geschäftsjahr. ", "b": True}, "Das Geschäftsjahr ist das Kalenderjahr; das erste Geschäftsjahr endet am 31.12.2026."]},
            {"type": "para", "justify": True, "text": [{"t": "§ 5 Geschäftsführung. ", "b": True}, "Zur Geschäftsführerin wird Frau Dr. Mira Kranich bestellt. Sie vertritt allein und ist von den Beschränkungen des § 181 BGB befreit."]},
            {"type": "para", "justify": True, "text": [{"t": "§ 6 Gründungsaufwand. ", "b": True}, "Die Gesellschaft trägt die Gründungskosten bis zu 2.000,00 €."]},
            {"type": "sig", "items": [{"label": "Dr. Mira Kranich", "sig": "M. Kranich"}, {"label": "Matthias Lechner, Notar", "sig": "Lechner"}]},
        ]}]
    n = "2026-06-01_Gesellschaftsvertrag_UR-0934-2026.pdf"
    compile_doc(T.path(F1, n), "Gesellschaftsvertrag Kranich Software GmbH", "Notar Lechner", vertrag)
    T.add(F1, n, "2026-06-01", "PDF", "Notar Lechner", "25.000,00 € Stammkapital", "Zeichnung, volle Einzahlung (1298 an 2900)")

    notar = Invoice("KR 0934/2026", "2026-06-09", NOTAR, ME_IG, [
        Line("Beurkundung Gesellschaftsvertrag, Einpersonengründung (Geschäftswert 25.000 €), 1,0-Gebühr", 1, D("115.00"), unit_label="pausch."),
        Line("Anmeldung zum Handelsregister (Geschäftswert 30.000 €), 0,5-Gebühr", 1, D("62.50"), unit_label="pausch."),
        Line("Vollzug: Gesellschafterliste, 0,3-Gebühr", 1, D("34.50"), unit_label="pausch."),
        Line("Erzeugung strukturierter Daten (XML), 0,3-Gebühr", 1, D("37.50"), unit_label="pausch."),
        Line("Beglaubigung von Abschriften", 1, D("10.00"), unit_label="pausch."),
        Line("Dokumentenpauschale", 1, D("12.00"), unit_label="pausch."),
        Line("Post- und Telekommunikationsentgelte", 1, D("20.00"), unit_label="pausch."),
    ], delivery="2026-06-01", due="2026-06-23")
    efile(T, F1, "2026-06-09_Notar-Lechner_Kostenberechnung.pdf", notar, fmt="pdf", issuer="Notar Lechner",
          case="Gründungskosten, PDF ohne Datensatz",
          pdf_kwargs=dict(font="Georgia", accent="#5a3e1b", title="Kostenberechnung (§ 19 GNotKG)", intro="UR-Nr. 0934/2026 L – Errichtung der Kranich Software GmbH"))

    ag = letter_pages(Party("Amtsgericht München – Registergericht", "Infanteriestraße 5", "80325", "München"), ME.address_lines(),
                      meta=[["Kassenzeichen", "4711 0815 2026"], ["Datum", "18.06.2026"], ["Aktenzeichen", "HRB 291837"]],
                      title="Kostenrechnung", font="Arial", accent="#30475e",
                      paras=["In der Registersache Kranich Software GmbH werden folgende Kosten erhoben:",
                             {"type": "table", "widths": ["1fr", 30], "align": ["l", "r"], "header": ["Gegenstand", "Betrag"],
                              "rows": [["Ersteintragung einer GmbH (Nr. 2100 GV HRegGebV), eingetragen am 16.06.2026", "150,00 €"]]},
                             {"type": "totals", "rows": [["Zu zahlen", "150,00 €", True]]},
                             f"Zahlbar bis 02.07.2026 an die Landesjustizkasse Bamberg, IBAN {fmt_iban(LJK_IBAN)}, unter Angabe des Kassenzeichens."])
    n = "2026-06-18_Amtsgericht-Muenchen_Kostenrechnung_HRB-291837.pdf"
    compile_doc(T.path(F1, n), "Kostenrechnung HRB 291837", "Amtsgericht München", ag)
    T.add(F1, n, "2026-06-18", "PDF", "Amtsgericht München", "150,00 €", "Gerichtsgebühr, nicht steuerbar")

    apfel = Invoice("2026-11872", "2026-06-22", APFEL, ME_V, [
        Line("MacBook Pro 14\" M5, 32 GB, 1 TB SSD", 1, D("2149.00"), detail="Seriennummer C02XK1M5Q6L4"),
        Line("USB-C Multiport-Adapter", 1, D("39.00")),
    ], delivery="2026-06-22", due="2026-07-06", customer_no="50331",
        means={"code": "58", "iban": APFEL.iban, "bic": APFEL.bic, "account_name": APFEL.name},
        terms="Zahlbar innerhalb von 14 Tagen ohne Abzug.")
    efile(T, F1, "2026-06-22_Apfelgruen_Rechnung_2026-11872.pdf", apfel, fmt="zugferd", issuer="Apfelgrün IT-Handel GmbH",
          case="Anlage über 800 €: Notebook aktivieren, Adapter sofort; Überweisung",
          pdf_kwargs=dict(font="Helvetica", accent="#3a7d44", tagline="IT für Unternehmen"))

    i6 = isar("IH-2026-06-1188", "2026-06-30", ("2026-06-01", "2026-06-30"), "06/2026", "2026-07-06")
    efile(T, F1, "2026-06-30_Isarhost_IH-2026-06-1188.pdf", i6, fmt="zugferd", issuer="Isarhost GmbH",
          case="Hosting 1/4: Lieferant anlegen, Standardgruppe vorschlagen (Mensch bestätigt in der App)", pdf_kwargs=ISAR_STYLE)

    # ------------------------------------------------------------ 02 Juli
    dar = [{
        "kind": "a4", "font": "Georgia", "accent": "#222222", "footer": [],
        "blocks": [
            {"type": "title", "text": "Darlehensvertrag", "sub": "zwischen Frau Dr. Mira Kranich, Herzog-Wilhelm-Straße 9, 80331 München (Darlehensgeberin) und der Kranich Software GmbH, vertreten durch die Geschäftsführerin (Darlehensnehmerin)"},
            {"type": "para", "justify": True, "text": [{"t": "§ 1 Darlehen. ", "b": True}, f"Die Darlehensgeberin gewährt der Darlehensnehmerin ein Darlehen über 15.000,00 €. Es wird am 01.07.2026 auf das Geschäftskonto IBAN {fmt_iban(ME.iban)} ausgezahlt."]},
            {"type": "para", "justify": True, "text": [{"t": "§ 2 Zinsen. ", "b": True}, "Das Darlehen wird mit 4,00 % p. a. verzinst (Zinsmethode act/365). Die Zinsen sind jährlich nachträglich zum 30.06. fällig, erstmals am 30.06.2027."]},
            {"type": "para", "justify": True, "text": [{"t": "§ 3 Laufzeit und Tilgung. ", "b": True}, "Das Darlehen ist am 30.06.2029 in einer Summe zurückzuzahlen. Eine vorzeitige Rückzahlung ist jederzeit ohne Vorfälligkeitsentschädigung möglich."]},
            {"type": "para", "justify": True, "text": [{"t": "§ 4 Sicherheiten, Rang. ", "b": True}, "Sicherheiten werden nicht gestellt. Ein Rangrücktritt ist nicht vereinbart."]},
            {"type": "para", "justify": True, "text": [{"t": "§ 5 Fremdvergleich. ", "b": True}, "Die Isartal Testbank eG hat mit Schreiben vom 15.06.2026 ein ungesichertes Darlehen gleicher Laufzeit zu 6,20 % p. a. angeboten."]},
            {"type": "para", "justify": True, "text": [{"t": "§ 6 Sonstiges. ", "b": True}, "Die Geschäftsführerin ist von den Beschränkungen des § 181 BGB befreit. Änderungen bedürfen der Schriftform."]},
            {"type": "para", "text": "München, den 26.06.2026"},
            {"type": "sig", "items": [{"label": "Dr. Mira Kranich (Darlehensgeberin)", "sig": "M. Kranich"}, {"label": "Kranich Software GmbH, Dr. Mira Kranich (GF)", "sig": "M. Kranich"}]},
        ]}]
    n = "2026-06-26_Darlehensvertrag_Gesellschafterin.pdf"
    compile_doc(T.path(F2, n), "Darlehensvertrag", "Kranich Software GmbH", dar)
    T.add(F2, n, "2026-06-26", "PDF", "Dr. Mira Kranich / GmbH", "15.000,00 €",
          "Gesellschafterdarlehen an die GmbH (`darlehen_erfassen` mit `beleg_id`): 3642, Zinsen 7317 (Beteiligung > 25 %), Fremdvergleich 6,2 %; der Vertrag ist danach zugeordnet")

    brandl = Invoice("2026-0712", "2026-07-01", BRANDL, ME_V, [
        Line("Schreibtisch elektrisch höhenverstellbar, 160 × 80 cm", 2, D("849.00")),
        Line("Bürodrehstuhl ergonomisch, Modell Ergo 5", 2, D("489.00")),
        Line("Lieferung und Montage", 1, D("120.00"), unit_label="pausch."),
    ], delivery="2026-07-01", due="2026-07-31", customer_no="K-20871",
        means={"code": "58", "iban": BRANDL.iban, "bic": BRANDL.bic, "account_name": BRANDL.name},
        terms="Zahlbar innerhalb von 30 Tagen ohne Abzug.")
    efile(T, F2, "2026-07-01_Brandl-Bueromoebel_Rechnung_2026-0712.pdf", brandl, fmt="zugferd", issuer="Brandl Büromöbel GmbH",
          case="Tische > 800 € je Stück → Anlage (Büroeinrichtung), Stühle ≤ 800 € → GWG; Lieferung als Anschaffungsnebenkosten aufteilen",
          pdf_kwargs=dict(font="Gill Sans", accent="#8b5e34", tagline="Büroeinrichtung seit 1962"))

    fa = letter_pages(Party("Finanzamt München", "Deroystraße 18", "80335", "München", phone="089 1252-0"), ME.address_lines(),
                      meta=[["Steuernummer", ME.tax_no], ["Datum", "06.07.2026"]],
                      title="Steuerliche Erfassung, Dauerfristverlängerung und Sondervorauszahlung", font="Arial", accent="#30475e",
                      paras=["Sehr geehrte Damen und Herren,", "Ihr Unternehmen ist unter der oben genannten Steuernummer erfasst. Es gelten folgende Festlegungen:",
                             {"type": "kv", "label_width": 62, "rows": [
                                 ["Umsatzsteuer", "Voranmeldungszeitraum: Kalendermonat (voraussichtliche Steuer über 9.000 €)"],
                                 ["Dauerfristverlängerung", "gewährt ab Voranmeldungszeitraum Juni 2026 (§ 46 UStDV)"],
                                 ["Sondervorauszahlung 2026", "2.000,00 €, fällig 10.07.2026, Einzug per SEPA-Lastschrift; Anrechnung in der Voranmeldung Dezember 2026"],
                                 ["Lohnsteuer", "Anmeldungszeitraum: Kalendermonat"],
                                 ["Körperschaftsteuer", "Vorauszahlungen 2026: 0,00 €"],
                                 ["SEPA-Lastschriftmandat", "liegt vor (Mandatsreferenz FA9143-112-50117)"]]},
                             "Mit freundlichen Grüßen", "Ihr Finanzamt München"])
    n = "2026-07-06_Finanzamt-Muenchen_Erfassung-DFV.pdf"
    compile_doc(T.path(F2, n), "Steuerliche Erfassung", "Finanzamt München", fa)
    T.add(F2, n, "2026-07-06", "PDF", "Finanzamt München", "2.000,00 € SVZ",
          "Kein Buchungsbeleg: Stammdaten (monatlich, Dauerfristverlängerung) in der App; Sondervorauszahlung kommt über die Bank (3830)")

    cf7 = Invoice("CF-2026-07-88231", "2026-07-15", CODEF, ME_V, [
        Line("Team plan – seat (Jul 15 – Aug 14, 2026)", 4, D("21.00"), rate=D("0"), cat="O", unit_label="seat"),
    ], currency="USD", period=("2026-07-15", "2026-08-14"))
    efile(T, F2, "2026-07-15_Codeforge_Invoice_CF-2026-07-88231.pdf", cf7, fmt="pdf", issuer="Codeforge Inc. (USA)",
          case="Drittland, § 13b Abs. 2 Nr. 1 UStG (Kz 84/85), Fremdwährung USD, bezahlt mit Firmenkarte",
          pdf_kwargs=dict(lang="en", font="Helvetica Neue", accent="#24292f", tagline="Code hosting for teams",
                          extra_notes=["No VAT or sales tax charged. Business customers outside the United States are responsible for self-assessing VAT (reverse charge)."],
                          pay_lines=["Paid · Charged to Visa ending in 8812 on Jul 15, 2026"]))

    beeld = Invoice("BD-2026-071744", "2026-07-21", BEELD, ME_V, [
        Line("Monitor ProView U27, 27 inch, 4K UHD, USB-C", 2, D("274.50"), rate=D("0"), cat="K", unit_label="pcs"),
    ], delivery="2026-07-21")
    efile(T, F2, "2026-07-21_Beeldscherm-Direct_Invoice_BD-2026-071744.pdf", beeld, fmt="pdf", issuer="Beeldscherm Direct B.V. (NL)",
          case="Innergemeinschaftlicher Erwerb (1404/3804, Kz 89/61), zwei GWG zu je 274,50 €, bezahlt mit Firmenkarte",
          pdf_kwargs=dict(lang="en", font="Avenir Next", accent="#e85d04", tagline="Displays for professionals",
                          extra_notes=["Intra-Community supply of goods – VAT exempt under Article 138 Directive 2006/112/EC. Customer VAT ID: " + ME.vat_id + "."],
                          pay_lines=["Paid by Visa •••• 8812 on 21 Jul 2026. Shipped from Rotterdam (NL) to München (DE) on 21 Jul 2026."]))

    lohnjournal("2026-07", F2)
    i7 = isar("IH-2026-07-1302", "2026-07-31", ("2026-07-01", "2026-07-31"), "07/2026", "2026-08-05")
    efile(T, F2, "2026-07-31_Isarhost_IH-2026-07-1302.pdf", i7, fmt="zugferd", issuer="Isarhost GmbH", case="Hosting 2/4", pdf_kwargs=ISAR_STYLE)
    card_statement(F2, "2026-07-31", "2026-08-05", [
        ["15.07.", "16.07.", "CODEFORGE.COM SAN FRANCISCO", "USD 84,00", "1,1620", "72,29"],
        ["15.07.", "16.07.", "Auslandseinsatzentgelt 1,75 %", "", "", "1,27"],
        ["21.07.", "22.07.", "BEELDSCHERM DIRECT ROTTERDAM", "", "", "549,00"]], D("622.56"))

    # ------------------------------------------------------------ 03 August
    ticket = [{
        "kind": "a4", "font": "Helvetica", "accent": "#d00f2c", "footer": [["Südbahn Fernverkehr AG", "Arnulfstraße 19, 80335 München"], ["USt-IdNr. " + de_vat("11475592")], ["Fahrkarte gilt nur mit Lichtbildausweis"]],
        "blocks": [
            {"type": "letterhead", "name": "Südbahn Fernverkehr", "tagline": "Online-Ticket", "right": ["Auftragsnummer SB7K4Q", "Buchung 12.08.2026"]},
            {"type": "title", "text": "Fahrkarte und Rechnung", "sub": "Flexpreis · 2. Klasse · 1 Erwachsener · Reisende: Dr. Mira Kranich"},
            {"type": "table", "widths": [24, "1fr", 24, 24], "align": ["l", "l", "l", "r"], "header": ["Datum", "Strecke", "Zug", "Preis"], "rows": [
                ["13.08.2026", "München Hbf 07:16 → Hamburg Hbf 13:22", "SFV 588", "139,90 €"],
                ["14.08.2026", "Hamburg Hbf 16:01 → München Hbf 22:14", "SFV 597", "139,90 €"]]},
            {"type": "totals", "rows": [["Gesamtpreis", "279,80 €", True], ["darin 7 % MwSt.", "18,30 €"], ["Nettobetrag", "261,50 €"]]},
            {"type": "kv", "rows": [["Rechnungsempfänger", "Kranich Software GmbH, Schwanthalerstraße 64, 80336 München"],
                                    ["Zahlung", "Visa •••• 8812, belastet am 12.08.2026"], ["Rechnungsnummer", "SB-2026-0812-447190"]]},
            {"type": "note", "lines": ["Fahrausweise im Fernverkehr über 50 km unterliegen dem ermäßigten Steuersatz (§ 12 Abs. 2 Nr. 10 UStG)."]},
        ]}]
    n = "2026-08-12_Suedbahn_Online-Ticket_SB7K4Q.pdf"
    compile_doc(T.path(F3, n), "Online-Ticket", "Südbahn Fernverkehr AG", ticket)
    T.add(F3, n, "2026-08-12", "PDF", "Südbahn Fernverkehr AG", "279,80 €", "Fahrausweis 7 %, mit `brutto` gebucht: Vorsteuer 18,30 € aus dem Bruttobetrag (§ 35 UStDV); Reisekosten, Firmenkarte")

    bew = [{"kind": "bon", "font": "Courier New", "lines": [
        {"t": "KUTTERHUS AM HAFEN", "a": "c", "b": True, "s": 11},
        {"t": "Kutterhus am Hafen GmbH", "a": "c"}, {"t": "St. Pauli Fischmarkt 28, 22767 Hamburg", "a": "c"},
        {"t": "StNr 48/223/71155", "a": "c"}, "---",
        ["Rechnung Nr. 4471", "Tisch 12"], ["13.08.2026 22:47", "Bed. Jannik"], "---",
        ["1 Krabbensuppe", "14,80 B"], ["1 Hamburger Pannfisch", "26,90 B"], ["1 Labskaus", "22,50 B"], ["1 Scholle Finkenwerder", "31,50 B"],
        ["3 Alsterwasser 0,4 à 4,80", "14,40 A"], ["1 Riesling trocken 0,75", "42,00 A"], ["2 Mineralwasser 0,75 à 6,90", "13,80 A"],
        ["3 Espresso à 3,20", "9,60 A"], "---",
        [{"t": "SUMME EUR", "b": True}, {"t": "175,50", "b": True}], ["VISA •••• 8812", "190,00"], ["davon Trinkgeld", "14,50"], "---",
        ["Satz   Netto  MwSt", "Brutto"], ["A 19%  67,06  12,74", "79,80"], ["B  7%  89,44   6,26", "95,70"], "---",
        {"t": "TSE-Seriennr. 41c9…a07b · Sig.-Zähler 30217", "s": 6.5}, {"t": "Beginn 2026-08-13T19:02:11 Ende 2026-08-13T22:47:40", "s": 6.5}, "---",
        {"t": "Danke und bis bald!", "a": "c"}]},
        {"kind": "a4", "font": "Helvetica", "accent": "#333333", "footer": [], "blocks": [
            {"type": "title", "text": "Bewirtungsbeleg", "sub": "Angaben zum Nachweis der Höhe und der betrieblichen Veranlassung von Bewirtungsaufwendungen (§ 4 Abs. 5 Satz 1 Nr. 2 EStG)"},
            {"type": "kv", "label_width": 60, "rows": [
                ["Tag der Bewirtung", "13.08.2026"], ["Ort der Bewirtung", "Kutterhus am Hafen, St. Pauli Fischmarkt 28, 22767 Hamburg"],
                ["Bewirtete Personen", "Henrik Sørensen (CEO, Nordsø Analytics ApS, Kopenhagen); Line Madsen (Head of Data, Nordsø Analytics ApS)"],
                ["Gastgeberin", "Dr. Mira Kranich, Geschäftsführerin Kranich Software GmbH"],
                ["Anlass", "Projektanbahnung: Workshop „Data Platform Review“ (Auftrag vom 18.08.2026)"],
                ["Höhe der Aufwendungen", "175,50 € laut Rechnung Nr. 4471 zuzüglich 14,50 € Trinkgeld = 190,00 €, bezahlt mit Firmenkarte •••• 8812"],
                ["Ort, Datum", "München, 19.08.2026"]]},
            {"type": "sig", "items": [{"label": "Unterschrift Gastgeberin", "sig": "M. Kranich"}]}]}]
    n = "2026-08-13_Kutterhus_Bewirtung_Rechnung-4471.pdf"
    compile_doc(T.path(F3, n), "Bewirtung", "Kutterhus am Hafen", bew)
    T.add(F3, n, "2026-08-13", "PDF (Bon + Bewirtungsbeleg)", "Kutterhus am Hafen GmbH", "175,50 € + 14,50 € Trinkgeld",
          "Bewirtung: Speisen 7 % (seit 2026), Getränke 19 %; 70 % abziehbar (6640/6644), Vorsteuer voll; Trinkgeld als Position `nicht_steuerbar` im selben Beleg")

    hotel = Invoice("2026-08-3381", "2026-08-14", HOTEL, ME_B, [
        Line("Übernachtung Einzelzimmer, 13.–14.08.2026", 1, D("129.00"), rate=D("7"), unit_label="Nacht"),
        Line("Frühstück", 1, D("19.90"), rate=D("19"), unit_label="Pers."),
    ], period=("2026-08-13", "2026-08-14"))
    efile(T, F3, "2026-08-14_Hafenkante-Hotel_Rechnung_2026-08-3381.pdf", hotel, fmt="pdf", issuer="Hafenkante Hotel GmbH",
          case="Zwei Steuersätze (7 % Übernachtung, 19 % Frühstück), Reisekosten, Firmenkarte",
          pdf_kwargs=dict(font="Georgia", accent="#1d3557", tagline="Hotel am Hamburger Hafen",
                          intro="Gast: Dr. Mira Kranich · Zimmer 412 · Anreise 13.08.2026 · Abreise 14.08.2026",
                          pay_lines=["Bezahlt mit Visa •••• 8812 am 14.08.2026. Vielen Dank für Ihren Aufenthalt!"]))

    cf8 = Invoice("CF-2026-08-90412", "2026-08-15", CODEF, ME_V, [
        Line("Team plan – seat (Aug 15 – Sep 14, 2026)", 4, D("21.00"), rate=D("0"), cat="O", unit_label="seat"),
    ], currency="USD", period=("2026-08-15", "2026-09-14"))
    efile(T, F3, "2026-08-15_Codeforge_Invoice_CF-2026-08-90412.pdf", cf8, fmt="pdf", issuer="Codeforge Inc. (USA)",
          case="wie Juli: § 13b Abs. 2 Nr. 1, USD, Firmenkarte",
          pdf_kwargs=dict(lang="en", font="Helvetica Neue", accent="#24292f", tagline="Code hosting for teams",
                          extra_notes=["No VAT or sales tax charged. Business customers outside the United States are responsible for self-assessing VAT (reverse charge)."],
                          pay_lines=["Paid · Charged to Visa ending in 8812 on Aug 15, 2026"]))

    weiss = Invoice("RE 26-1408", "2026-08-18", WEISS, ME_V, [
        Line("Visitenkarten 4/4-farbig, 350 g, je 250 Stück", 2, D("45.00"), unit_label="Satz"),
        Line("Flyer DIN lang, 4/4-farbig, 2.000 Stück", 1, D("310.00"), unit_label="Auflage"),
        Line("Plakate A1, 4/0-farbig, 50 Stück", 1, D("600.00"), unit_label="Auflage"),
    ], delivery="2026-08-17", due="2026-09-17", customer_no="10488", skonto=(10, D("2")),
        means={"code": "58", "iban": WEISS.iban, "bic": WEISS.bic, "account_name": WEISS.name},
        terms="Zahlbar innerhalb von 30 Tagen ohne Abzug, innerhalb von 10 Tagen mit 2 % Skonto.")
    efile(T, F3, "2026-08-18_Druckerei-Weiss_RE-26-1408.pdf", weiss, fmt="zugferd", issuer="Druckerei Weiß OHG",
          case="Skonto 2 % bei Zahlung binnen 10 Tagen: Bank 1.166,20 → Skonto mindert Aufwand und Vorsteuer (3,80 €)",
          pdf_kwargs=dict(font="Futura", accent="#c1121f", tagline="Offset · Digital · Großformat",
                          extra_notes=["Zahlbar innerhalb von 30 Tagen ohne Abzug oder innerhalb von 10 Tagen mit 2 % Skonto (1.166,20 €)."]))

    vino = Invoice("2026-8812", "2026-08-28", VINO, ME_B, [
        Line("Präsentkarton „Bayerische Auslese“ (3 Flaschen Wein, Holzkiste, Grußkarte)", 3, D("79.00")),
    ], delivery="2026-08-28", due="2026-09-11",
        means={"code": "58", "iban": VINO.iban, "bic": VINO.bic, "account_name": VINO.name}, terms="Zahlbar innerhalb von 14 Tagen.")
    efile(T, F3, "2026-08-28_Vinothek-Sendling_Rechnung_2026-8812.pdf", vino, fmt="pdf", issuer="Vinothek Sendling GmbH",
          case="Geschenke über 50 € je Empfänger: nicht abziehbar (6620), keine Vorsteuer (§ 15 Abs. 1a UStG); Empfängerliste auf dem Beleg",
          pdf_kwargs=dict(font="Georgia", accent="#6d1a36", tagline="Wein · Feinkost · Präsente",
                          extra_notes=["Versand je eines Präsents an: Dr. Thomas Steiner und Sabine Wagner (Alpenblick Versicherung AG, München); "
                                       "Henrik Sørensen (Nordsø Analytics ApS, Kopenhagen). Grußkarte: „Danke für die gute Zusammenarbeit – Kranich Software“."]))

    i8 = isar("IH-2026-08-1421", "2026-08-31", ("2026-08-01", "2026-08-31"), "08/2026", "2026-09-07")
    efile(T, F3, "2026-08-31_Isarhost_IH-2026-08-1421.pdf", i8, fmt="zugferd", issuer="Isarhost GmbH", case="Hosting 3/4", pdf_kwargs=ISAR_STYLE)
    lohnjournal("2026-08", F3)
    card_statement(F3, "2026-08-31", "2026-09-04", [
        ["12.08.", "13.08.", "SUEDBAHN FERNVERKEHR ONLINE", "", "", "279,80"],
        ["13.08.", "14.08.", "KUTTERHUS AM HAFEN HAMBURG", "", "", "190,00"],
        ["14.08.", "15.08.", "HAFENKANTE HOTEL HAMBURG", "", "", "161,71"],
        ["15.08.", "17.08.", "CODEFORGE.COM SAN FRANCISCO", "USD 84,00", "1,1630", "72,23"],
        ["15.08.", "17.08.", "Auslandseinsatzentgelt 1,75 %", "", "", "1,26"]], D("705.00"))

    # ------------------------------------------------------------ 04 September
    dup = "2026-09-02_Brandl-Rechnung-2026-0712_nochmal-per-Mail.pdf"
    os.makedirs(os.path.dirname(T.path(F4, dup)), exist_ok=True)
    shutil.copy(T.path(F2, "2026-07-01_Brandl-Bueromoebel_Rechnung_2026-0712.pdf"), T.path(F4, dup))
    T.add(F4, dup, "2026-09-02", "ZUGFeRD (PDF + XML)", "Brandl Büromöbel GmbH", de(brandl.gross, "€"),
          "Dublette: dieselbe Datei wie im Juli, nur umbenannt → `bereits_abgelegt`, nicht erneut buchen")

    ntf = Invoice("NTF-2026-0388", "2026-09-14", NTF, ME_V, [
        Line("Annual brand font licence “Kranich Sans” family, 5 desktop users, 15 Sep 2026 – 14 Sep 2027", 1, D("450.00"), rate=D("0"), cat="AE"),
    ], period=("2026-09-15", "2027-09-14"), due="2026-09-28", buyer_ref="Kranich Software / M. Kranich",
        means={"code": "58", "iban": NTF.iban, "bic": NTF.bic, "account_name": NTF.name}, terms="Payment within 14 days.")
    efile(T, F4, "2026-09-14_Nordic-Type-Foundry_NTF-2026-0388.xml", ntf, fmt="ubl", issuer="Nordic Type Foundry AB (SE)",
          case="UBL ohne PDF, buchbar mit der Ansicht aus der XML; § 13b Abs. 1 (Kz 46/47); Jahreslizenz über den Bilanzstichtag → Abgrenzung zum 31.12.")

    mucf = Invoice("MF-2026-0912-77812", "2026-09-15", MUCF, ME_V, [
        Line("Glasfaser Business 500, Grundgebühr 15.08.–14.09.2026", 1, D("49.90"), unit_label="Monat"),
        Line("Feste IPv4-Adresse, 15.08.–14.09.2026", 1, D("10.00"), unit_label="Monat"),
    ], period=("2026-08-15", "2026-09-14"), due="2026-09-18", buyer_ref="KD-7781230",
        means={"code": "59", "debtor_iban": ME.iban, "mandate": MUCF_MANDATE, "creditor_id": MUCF_CI},
        terms=f"Der Betrag wird am 18.09.2026 per SEPA-Lastschrift eingezogen (Mandat {MUCF_MANDATE}).")
    efile(T, F4, "2026-09-15_MUC-Fiber_MF-2026-0912-77812.xml", mucf, fmt="xrechnung", issuer="MUC Fiber GmbH",
          case="Reine XRechnung (CII) ohne PDF: buchbar, die App zeigt die Ansicht aus der XML; Internetkosten (6810), Lastschrift")

    gs = Invoice("IH-GS-2026-0042", "2026-09-21", ISAR, ME_B, [
        Line("Gutschrift SLA: Ausfall Managed Server M am 27.08.2026 (6 Std.)", 1, D("20.00"), unit_label="pausch."),
    ], type_code="381", delivery="2026-08-27", preceding=("IH-2026-08-1421", "2026-08-31"), customer_no="K-50331",
        terms="Der Betrag wird am 25.09.2026 auf Ihr Konto überwiesen.")
    efile(T, F4, "2026-09-21_Isarhost_Gutschrift_IH-GS-2026-0042.pdf", gs, fmt="zugferd", issuer="Isarhost GmbH",
          case="Gutschrift (Typ 381) zur Rechnung IH-2026-08-1421: mit `gutschrift_zu` buchen, Erstattung gleicht das Guthaben aus",
          pdf_kwargs=dict(**ISAR_STYLE, title="Gutschrift", intro="Bezug: Rechnung IH-2026-08-1421 vom 31.08.2026"))

    fake = Invoice("2026-0958", "2026-09-22", BRANDL_FAKE, ME_V, [
        Line("Rollcontainer mit 3 Schubladen, abschließbar", 2, D("189.00")),
    ], delivery="2026-09-21", due="2026-10-06", customer_no="K-20871",
        means={"code": "58", "iban": BRANDL_FAKE.iban, "bic": BRANDL_FAKE.bic, "account_name": "Brandl Bueromoebel"},
        terms="Zahlbar innerhalb von 14 Tagen ohne Abzug.")
    efile(T, F4, "2026-09-22_Brandl-Bueromoebel_Rechnung_2026-0958.pdf", fake, fmt="zugferd", issuer="„Brandl Büromöbel GmbH“",
          case="Betrugsverdacht: neue Bankverbindung (Litauen) und abweichende Mail-Domain → `iban_weicht_ab`; nicht zahlen, beim Lieferanten nachfragen",
          pdf_kwargs=dict(font="Gill Sans", accent="#8b5e34", tagline="Büroeinrichtung seit 1962",
                          pay_lines=["WICHTIG: Unsere Bankverbindung hat sich geändert. Bitte überweisen Sie ab sofort ausschließlich auf",
                                     f"IBAN {fmt_iban(BRANDL_FAKE.iban)} · BIC {BRANDL_FAKE.bic} · Baltic Testbank. Zahlungen auf das alte Konto können nicht mehr zugeordnet werden."]))

    cf9 = Invoice("CF-2026-09-93107", "2026-09-15", CODEF, ME_V, [
        Line("Team plan – seat (Sep 15 – Oct 14, 2026)", 4, D("21.00"), rate=D("0"), cat="O", unit_label="seat"),
    ], currency="USD", period=("2026-09-15", "2026-10-14"))
    efile(T, F4, "2026-09-15_Codeforge_Invoice_CF-2026-09-93107.pdf", cf9, fmt="pdf", issuer="Codeforge Inc. (USA)",
          case="wie Juli und August: § 13b Abs. 2 Nr. 1, USD, Firmenkarte",
          pdf_kwargs=dict(lang="en", font="Helvetica Neue", accent="#24292f", tagline="Code hosting for teams",
                          extra_notes=["No VAT or sales tax charged. Business customers outside the United States are responsible for self-assessing VAT (reverse charge)."],
                          pay_lines=["Paid · Charged to Visa ending in 8812 on Sep 15, 2026"]))
    card_statement(F4, "2026-09-30", "2026-10-05", [
        ["15.09.", "16.09.", "CODEFORGE.COM SAN FRANCISCO", "USD 84,00", "1,1580", "72,54"],
        ["15.09.", "16.09.", "Auslandseinsatzentgelt 1,75 %", "", "", "1,27"]], D("73.81"))

    i9 = isar("IH-2026-09-1550", "2026-09-30", ("2026-09-01", "2026-09-30"), "09/2026", "2026-10-05")
    efile(T, F4, "2026-09-30_Isarhost_IH-2026-09-1550.pdf", i9, fmt="zugferd", issuer="Isarhost GmbH",
          case="Hosting 4/4: nach drei Buchungen und bestätigter Standardgruppe bucht `beleg_buchen` ohne Rückfrage", pdf_kwargs=ISAR_STYLE)
    lohnjournal("2026-09", F4)

    zins = letter_pages(Party("Isartal Testbank eG", "Sendlinger Straße 1", "80331", "München"), ME.address_lines(),
                        meta=[["Konto", fmt_iban(ME.iban)], ["Datum", "30.09.2026"]],
                        title="Zinsabrechnung III/2026", font="Helvetica Neue", accent="#1a1f71",
                        paras=["Für den Zeitraum 01.07.2026 bis 30.09.2026 rechnen wir Ihr Geschäftskonto wie folgt ab:",
                               {"type": "table", "widths": ["1fr", 30], "align": ["l", "r"], "header": ["Position", "Betrag"], "rows": [
                                   ["Habenzinsen 1,00 % p. a. auf das Durchschnittsguthaben", "100,00 €"],
                                   ["Kapitalertragsteuer 25 %", "−25,00 €"], ["Solidaritätszuschlag 5,5 %", "−1,37 €"]]},
                               {"type": "totals", "rows": [["Gutschrift, Wert 30.09.2026", "73,63 €", True]]},
                               "Eine Steuerbescheinigung nach Muster I erhalten Sie zum Jahresende. Kirchensteuer fällt bei Kapitalgesellschaften nicht an."])
    n = "2026-09-30_Isartal-Testbank_Zinsabrechnung-Q3.pdf"
    compile_doc(T.path(F4, n), "Zinsabrechnung", "Isartal Testbank eG", zins)
    T.add(F4, n, "2026-09-30", "PDF", "Isartal Testbank eG", "73,63 € (100,00 € brutto)",
          "Bankumsatz aufteilen: 7110 Zinsertrag 100,00, 7630 KapESt 25,00, 7633 Soli 1,37")

    # ------------------------------------------------------------ Bank
    B = dict(owner=ME.name, iban=ME.iban, bic=ME.bic, bank=ME.bank)
    fee = lambda date, m: Tx(date, D("-12.50"), "Isartal Testbank eG", "", f"Entgelt Kontofuehrung {m}", kind="Entgelt")
    sal = lambda date, m: [Tx(date, D("-2498.92"), "Lukas Bauer", LUKAS_IBAN, f"Gehalt {m}", kind="Lohn"),
                           Tx(date, D("-4371.59"), "Dr. Mira Kranich", MIRA_IBAN, f"Gehalt {m}", kind="Lohn")]
    sv = lambda date, m: Tx(date, D("-1686.82"), "BKK Testfink", BKK_IBAN, f"Beitraege {m} Betriebsnummer 81234567", kind="Lastschrift", mandate="BKKTF-81234567")
    isd = lambda date, inv: Tx(date, -inv.gross, ISAR.name, ISAR.iban, f"{inv.number} Kd K-50331", kind="Lastschrift", mandate=ISAR_MANDATE, creditor_id=ISAR_CI)
    fa = lambda date, amt, text: Tx(date, amt, "Finanzkasse Muenchen", FA_IBAN, f"StNr 143/112/50117 {text}", kind="Lastschrift" if amt < 0 else "Gutschrift", mandate="FA9143-112-50117" if amt < 0 else "")

    vst6 = notar.tax + apfel.tax + i6.tax
    ust7, vst7 = D("437.00") + D("1529.50"), brandl.tax + i7.tax
    ust8 = D("1748.00")
    vst8 = D("18.30") + hotel.tax + D("19.00") + weiss.tax - D("3.80") + i8.tax
    lst = D("1602.99")
    pay7, pay8 = ust7 - vst7, ust8 - vst8

    s1 = Statement(**B, seq=1, start="2026-06-02", end="2026-06-30", opening=D("0"), txs=[
        Tx("2026-06-03", D("25000.00"), "Dr. Mira Kranich", MIRA_IBAN, "Einzahlung Stammkapital Kranich Software GmbH i.G. UR 0934/2026"),
        Tx("2026-06-15", -notar.gross, NOTAR.name, NOTAR.iban, "KR 0934/2026 Gruendung Kranich Software"),
        Tx("2026-06-24", D("-150.00"), "Landesjustizkasse Bamberg", LJK_IBAN, "Kassenzeichen 4711 0815 2026 HRB 291837"),
        fee("2026-06-30", "06/2026")])
    s2 = Statement(**B, seq=2, start="2026-07-01", end="2026-07-31", opening=s1.closing, txs=[
        Tx("2026-07-01", D("15000.00"), "Dr. Mira Kranich", MIRA_IBAN, "Gesellschafterdarlehen lt. Vertrag vom 26.06.2026"),
        Tx("2026-07-03", -apfel.gross, APFEL.name, APFEL.iban, "RG 2026-11872 Kd 50331"),
        isd("2026-07-06", i6),
        fa("2026-07-10", D("-2000.00"), "Sondervorauszahlung USt 2026"),
        Tx("2026-07-28", -brandl.gross, BRANDL.name, BRANDL.iban, "Rechnung 2026-0712 Kd K-20871"),
        sv("2026-07-29", "07/2026"), *sal("2026-07-31", "07/2026"), fee("2026-07-31", "07/2026")])
    s3 = Statement(**B, seq=3, start="2026-08-01", end="2026-08-31", opening=s2.closing, txs=[
        Tx("2026-08-05", D("-622.56"), "Isartal Testbank eG Kartenservice", "", "Kreditkartenabrechnung 07/2026 Visa Business 8812", kind="Kartenabrechnung"),
        isd("2026-08-05", i7),
        fa("2026-08-10", -lst, "LSt 07/2026"),
        fa("2026-08-19", vst6, "Erstattung USt 06/2026"),
        Tx("2026-08-20", D("5000.00"), ALPEN.name, ALPEN.iban, "Teilzahlung RE-2026-0002"),
        Tx("2026-08-26", -(weiss.gross - D("23.80")), WEISS.name, WEISS.iban, "RE 26-1408 abzgl. 2% Skonto 23,80"),
        sv("2026-08-27", "08/2026"), *sal("2026-08-31", "08/2026"), fee("2026-08-31", "08/2026")])
    s4 = Statement(**B, seq=4, start="2026-09-01", end="2026-09-30", opening=s3.closing, txs=[
        Tx("2026-09-01", -vino.gross, VINO.name, VINO.iban, "Rg 2026-8812 Praesente"),
        Tx("2026-09-04", D("-705.00"), "Isartal Testbank eG Kartenservice", "", "Kreditkartenabrechnung 08/2026 Visa Business 8812", kind="Kartenabrechnung"),
        Tx("2026-09-04", D("4579.50"), ALPEN.name, ALPEN.iban, "Restzahlung RE-2026-0002"),
        isd("2026-09-07", i8),
        fa("2026-09-10", -(pay7 + lst), f"USt 07/2026 {de(pay7)} LSt 08/2026 {de(lst)}"),
        Tx("2026-09-15", D("10948.00"), ALPEN.name, ALPEN.iban, "RE-2026-0003"),
        Tx("2026-09-18", -mucf.gross, MUCF.name, MUCF.iban, f"{mucf.number} KD-7781230", kind="Lastschrift", mandate=MUCF_MANDATE, creditor_id=MUCF_CI),
        Tx("2026-09-21", D("3750.00"), NORDSO.name, NORDSO.iban, "Invoice RE-2026-0004 Data Platform Review"),
        Tx("2026-09-24", -ntf.gross, NTF.name, NTF.iban, "NTF-2026-0388 font licence"),
        Tx("2026-09-25", gs.gross, ISAR.name, ISAR.iban, "Gutschrift IH-GS-2026-0042 SLA"),
        sv("2026-09-28", "09/2026"), *sal("2026-09-30", "09/2026"),
        Tx("2026-09-30", D("73.63"), "Isartal Testbank eG", "", "Zinsen III/2026 100,00 abzgl. KapESt 25,00 SolZ 1,37", kind="Zinsen"),
        fee("2026-09-30", "09/2026")])
    s5 = Statement(**B, seq=5, start="2026-10-01", end="2026-10-16", opening=s4.closing, txs=[
        Tx("2026-10-05", D("-73.81"), "Isartal Testbank eG Kartenservice", "", "Kreditkartenabrechnung 09/2026 Visa Business 8812", kind="Kartenabrechnung"),
        isd("2026-10-05", i9),
        fa("2026-10-12", -(pay8 + lst), f"USt 08/2026 {de(pay8)} LSt 09/2026 {de(lst)}"),
        Tx("2026-10-14", D("9579.50"), ALPEN.name, ALPEN.iban, "RE-2026-0005")])
    for folder, st in [(F1, s1), (F2, s2), (F3, s3), (F4, s4), (F5, s5)]:
        name = f"{st.end}_Kontoauszug_{st.end[5:7]}-{st.end[:4]}_camt053.xml"
        T.write(folder, name, camt(st))
        T.add(folder, name, st.end, "CAMT.053", ME.bank, f"{len(st.txs)} Umsätze, Endsaldo {de(st.closing, '€')}",
              "Kontoauszug" + (" – den ersten liest der Mensch in der App ein" if st.seq == 1 else ""))
    return readme(locals())


def readme(v) -> str:
    s1, s2, s3, s4, s5 = (v[k] for k in ("s1", "s2", "s3", "s4", "s5"))
    pay7, pay8, lst = v["pay7"], v["pay8"], v["lst"]
    vst9 = v["mucf"].tax + v["i9"].tax - v["gs"].tax
    return f"""# Mandant 2: Kranich Software GmbH (München)

**Schwerpunkt: laufende Buchhaltung.** Einpersonen-GmbH, gegründet am 01.06.2026, voll eingezahlt. Ab Juli zwei
Gehälter, eine Firmenkreditkarte, Kunden im In- und Ausland. Die Belege decken die Fälle ab, an denen sich zeigt,
ob der Agent Routine und Rückfrage richtig trennt: E-Rechnungen eines Stammlieferanten bis zur Direktbuchung,
reine XRechnung und UBL, Gutschrift, Dublette, geänderte Bankverbindung, Bewirtung, Geschenke, zwei Steuersätze,
Skonto, Teilzahlung, igE, § 13b aus EU und Drittland, Fremdwährung, Lohn, Gesellschafterdarlehen, Zinsen mit
Kapitalertragsteuer, Steuerlastschriften und Dauerfristverlängerung.

## 1. Einrichtung in der App

| Feld im Einrichtungsassistenten | Wert |
|---|---|
| Firma | Kranich Software GmbH |
| Rechtsform | GmbH |
| Wie beginnen die Bücher? | Neu gegründet |
| Tag der Beurkundung | 01.06.2026 |
| Geschäftsjahr | Kalenderjahr |
| Bundesland | Bayern |

**Unternehmensdaten** (der Agent schlägt vor, der Mensch übernimmt in der App):

| Feld | Wert |
|---|---|
| Anschrift | Schwanthalerstraße 64, 80336 München |
| Geschäftsführung, Ansprechpartner | Dr. Mira Kranich, 089 4523 8810, buchhaltung@kranich-software.example |
| Register | Amtsgericht München HRB 291837, eingetragen 16.06.2026 |
| Notar, Urkunde | Matthias Lechner, München; UR-Nr. 0934/2026 L |
| Steuernummer, Finanzamt | 143/112/50117, Finanzamt München (bekannt ab Schritt 2) |
| USt-IdNr. | {ME.vat_id} |
| Voranmeldung | monatlich, Dauerfristverlängerung ab Juni 2026, Sondervorauszahlung 2.000 € |
| Steuerliche Beratung | Steuerkanzlei Obermaier, beauftragt ab 2026 für KSt, GewSt, USt-Jahreserklärung und E-Bilanz (nicht für Voranmeldungen) |
| Bankkonto | {ME.bank}, IBAN {fmt_iban(ME.iban)}, BIC {ME.bic}, Konto für Rechnungen |
| Firmenkreditkarte | Visa Business •••• 8812, Abrechnung monatlich, Einzug um den 5. vom Geschäftskonto |

**Gründung:** Stammkapital 25.000 €, Geschäftsanteil Nr. 1 Dr. Mira Kranich 25.000 € (100 %), sofort voll einzuzahlen,
Gründungsaufwand bis 2.000 €. **Gesellschafterin als Kontakt:** Dr. Mira Kranich (für Darlehen und Gehalt).

**Einstellungen › Buchen ohne Rückfrage:** Vorgaben lassen (Grenze 1.000 €, drei frühere Buchungen).

## 2. Testschritte

Die Belege gelten als am Belegdatum eingegangen (`eingang_am`). Die Ausgangsrechnungen stellt der Agent in genau
dieser Reihenfolge aus, damit die Nummern zu den Verwendungszwecken im Kontoauszug passen.

**Chronologie im Testlauf:** Buchfink stellt Rechnungen mit dem Tag des Testlaufs aus und bucht sie zum
Leistungstag. Die Nummern in den Verwendungszwecken stimmen, die Rechnungsdaten nicht: Kunden zahlen im Auszug
Rechnungen, die es an diesem Tag noch nicht gab. Sag dem Agenten, dass keine anderen Rechnungen existieren. Die
Mahnung am 20.08. und die Frist des § 14a UStG für Nordsø lassen sich so nicht prüfen; Voranmeldungen und
Lohnsteuer-Anmeldungen erscheinen als überfällig.

### Schritt 1 – Gründung und Juni (`{F1}/`)

{T.table(F1)}

Erwartet: Zeichnung 1298 an 2900 25.000,00; Einzahlung 1800 an 1298. Isarhost legt der Agent als Lieferanten mit
Standardgruppe `software` an; **die Standardgruppe bestätigt der Mensch in der App** (Unternehmen › Kontakte).
Voranmeldung Juni: Vorsteuer {de(v['vst6'], '€')}, Erstattung **{de(v['vst6'], '€')}**. Endsaldo 30.06.: **{de(s1.closing, '€')}**.

Die Pflicht „Eröffnungsbilanz aufstellen und unterzeichnen“ war am 01.09.2026 fällig und steht deshalb als überfällig im
Kalender. Die E-Bilanz der Eröffnungsbilanz hat die Frist der E-Bilanz 2026; weil die Steuerkanzlei sie übernimmt,
gilt deren Frist für die Körperschaftsteuererklärung 2026 (§ 149 Abs. 3 AO), und der Kalender nennt den Berater als
zuständig.

### Schritt 2 – Juli (`{F2}/`)

{T.table(F2)}

Ausgangsrechnungen (Agent, mit Vorschau und Rückfrage):

| Nr. (erwartet) | Kunde | Datum | Leistung | Positionen | Brutto |
|---|---|---|---|---|---:|
| RE-2026-0001 | Bergsteiger Outdoor GmbH, Münchener Straße 18, 83022 Rosenheim, USt-IdNr. {BERG.vat_id} | 15.07.2026 | 09.–10.07.2026 | „Code-Review und Architekturberatung Outdoor-App“, 2 Tage (DAY) × 1.150 € | 2.737,00 € |
| RE-2026-0002 | Alpenblick Versicherung AG, Leopoldstraße 244, 80807 München, USt-IdNr. {ALPEN.vat_id} | 31.07.2026 | 01.–31.07.2026 | „Beratung Schadenportal“, 7 Tage × 1.150 € | 9.579,50 € |

Zahlungsziele: Bergsteiger 14 Tage, Alpenblick 30 Tage.

Erwartete Buchungen und Prüfpunkte:

| Vorgang | Erwartung |
|---|---|
| Darlehen | `darlehen_erfassen` mit dem Vertrag als `beleg_id`: an die Gesellschaft, 15.000 €, ab 01.07.2026, fällig 30.06.2029, 4 %; Konto 3642, Zinsen 7317; Vergleichszins 6,2 % (Bankangebot). Auszahlung über den Bankumsatz 01.07. Der Vertrag ist dem Darlehen zugeordnet und kein Befund im Prüflauf |
| Brandl | Tische 2 × 849 € (+ anteilige Lieferung) auf Büroeinrichtung (0650), Stühle als GWG (0670/6260); Lieferung 120 € auf die Anschaffungskosten verteilen |
| Codeforge | Fremdwährung USD, § 13b Abs. 2 Nr. 1: 1407/3837 in gleicher Höhe, Kz 84/85; bezahlt mit Karte (3610). Umsatzsteuerkurs nach § 16 Abs. 6 UStG: mit Karte am 15.07. bezahlt, also Juli; ohne erfasste BMF-Monatskurse nimmt Buchfink den EZB-Tageskurs und sagt das. Mit `belastet` (Betrag laut Kartenabrechnung) steht die Belastung im Aufwand; sonst nimmt Buchfink die Abweichung beim Kartenausgleich als Kursdifferenz (4840/6880) an |
| Beeldscherm | igE: 1404 / 3804 je 104,31 €; GWG; Karte (3610) |
| Lohn Juli | 6020 3.800,00 + 6024 5.500,00 + 6110 803,70 + Umlagen 56,62 an 3720 6.870,51 + 3730 1.602,99 + 3740 1.686,82 |
| Bank | SVZ 2.000 € auf 3830; Löhne an 3720; Krankenkasse an 3740 |

Endsaldo 31.07.: **{de(s2.closing, '€')}**. Voranmeldung Juli: USt 1.966,50, Vorsteuer {de(v['vst7'], '€')} (ohne § 13b/igE,
die sich aufheben), Zahllast **{de(pay7, '€')}**, fällig mit Dauerfristverlängerung am 10.09.

### Schritt 3 – August (`{F3}/`)

{T.table(F3)}

Ausgangsrechnungen:

| Nr. (erwartet) | Kunde | Datum | Leistung | Positionen | Brutto |
|---|---|---|---|---|---:|
| RE-2026-0003 | Alpenblick Versicherung AG | 31.08.2026 | 01.–31.08.2026 | „Beratung Schadenportal“, 8 Tage × 1.150 € | 10.948,00 € |
| RE-2026-0004 | Nordsø Analytics ApS, Store Kongensgade 68, 1264 København K, DK, USt-IdNr. {NORDSO.vat_id} | 31.08.2026 | 25.–27.08.2026 | „Workshop Data Platform Review“, 3 Tage × 1.250 €, Steuerschuld des Empfängers | 3.750,00 € |

Dazu: Bergsteiger zahlt nicht. Die Mahnung (`mahnung_vermerken`) lässt sich im Testlauf nicht prüfen, weil
RE-2026-0001 das Datum des Testlaufs trägt und noch nicht fällig ist.

Prüfpunkte: Bewirtung mit 7 % Speisen und 19 % Getränken, 70/30 auf 6640/6644, Trinkgeld 14,50 € als Position
`nicht_steuerbar` im selben Beleg (ebenfalls 70/30, ohne Vorsteuer). Teilnehmer laut Bewirtungsbeleg: Henrik Sørensen,
Line Madsen, Dr. Mira Kranich. Fahrschein mit `brutto`: Vorsteuer 18,30 €. Hotel mit zwei Sätzen. Geschenke:
Empfänger nennen, 6620, keine Vorsteuer; der Agent sollte § 37b EStG ansprechen. Skonto Druckerei: Bank 1.166,20 €
gleicht 1.190,00 € aus, Skonto 20,00 € mindert die Werbekosten (6600), Vorsteuerkürzung 3,80 €. Teilzahlung
Alpenblick 5.000 € → Posten bleibt mit 4.579,50 € offen. Kreditkartenabrechnung Juli (Einzug 05.08.) gleicht 3610
aus; das Auslandseinsatzentgelt ist Bankgebühr (6855). Codeforge: die Abweichung zwischen dem Kurs der Rechnung und
dem Kartenkurs bucht Buchfink beim Kartenausgleich selbst als Kursdifferenz.

Endsaldo 31.08.: **{de(s3.closing, '€')}**. Voranmeldung August: USt 1.748,00, Vorsteuer {de(v['vst8'], '€')} (nach Skonto),
Zahllast **{de(pay8, '€')}**; Kz 21 mit 3.750 € (Nordsø) und Zusammenfassende Meldung III/2026.

### Schritt 4 – September (`{F4}/`)

{T.table(F4)}

Ausgangsrechnung: **RE-2026-0005** Alpenblick, 30.09.2026, Leistung 01.–30.09.2026, 7 Tage × 1.150 € = 9.579,50 € brutto.

Prüfpunkte:

- **Dublette:** Die umbenannte Brandl-Datei hat dieselbe Prüfsumme; `eingang` meldet `bereits_abgelegt`.
- **XRechnung MUC Fiber, UBL Nordic Type:** ohne PDF buchbar; die App zeigt die Ansicht aus der XML.
- **UBL Nordic Type:** § 13b Abs. 1, Jahreslizenz bis 14.09.2027 (Abgrenzung im Jahresabschluss).
- **Gutschrift Isarhost:** Typ 381, mit `gutschrift_zu` IH-2026-08-1421 gebucht: Hosting 20,00 und Vorsteuer 3,80 gemindert, Guthaben beim Kreditor; die Erstattung über die Bank am 25.09. gleicht es aus.
- **Codeforge September:** wie Juli; Kartenabrechnung 09/2026 mit Einzug am 05.10. im Oktober-Auszug.
- **Betrugsverdacht Brandl 2026-0958:** neue IBAN in Litauen, Mail-Domain `brandl-buero-moebel` statt `brandl-bueromoebel`. Erwartet: `iban_weicht_ab`, der Agent warnt und bucht nicht ohne Rückfrage.
- **Direktbuchung:** Isarhost 09/2026 ist die vierte Rechnung; wenn die Standardgruppe bestätigt ist, bucht `beleg_buchen` ohne Rückfrage. Vor Commit d9b1ebb meldete Buchfink hier fälschlich BR-61, und die Direktbuchung scheiterte an `e_rechnung_mit_befund`.
- **Zinsen:** Bankumsatz 73,63 € aufteilen in 7110 100,00 / 7630 25,00 / 7633 1,37.
- **Steuerlastschrift 10.09.:** ein Umsatz über {de(pay7 + lst, '€')} = USt 07 {de(pay7)} + LSt 08 {de(lst)} → Aufteilung auf 3820 und 3730.

Endsaldo 30.09.: **{de(s4.closing, '€')}**. Voranmeldung September (ohne die verdächtige Brandl-Rechnung): USt 1.529,50,
Vorsteuer {de(vst9, '€')}, Zahllast **{de(D('1529.50') - vst9, '€')}**.

### Schritt 5 – Oktober (`{F5}/`)

{T.table(F5)}

Prüfpunkte: Kartenabrechnung 09 am 05.10. (Kursdifferenz Codeforge). Steuerlastschrift 12.10. = USt 08 {de(pay8)} +
LSt 09 {de(lst)}. Zahlung RE-2026-0005. Danach Monatsabschluss
(Prüflauf, Festschreibung Juni bis September nur in der App), `fristen` mit Dauerfristverlängerung (USt 09 fällig 10.11.).

## 3. Kontrollwerte zum 16.10.2026

| Konto / Stand | Erwartung |
|---|---|
| 1800 Bank | {de(s5.closing, '€')} |
| 3610 Kreditkarte | 0,00 € |
| 3642 Darlehen Gesellschafterin | 15.000,00 € |
| Offene Forderungen | Bergsteiger RE-2026-0001 2.737,00 € (im Testlauf noch nicht fällig) |
| Offene Verbindlichkeiten | keine; Brandl 2026-0958 nur, wenn trotz Verdacht gebucht |
"""
