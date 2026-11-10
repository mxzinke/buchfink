"""Mandant 1: Nordlicht Digital GmbH, Hamburg. Gründung mit Halbeinzahlung."""

from common import *

T = Tenant("1-Nordlicht-Digital-GmbH")

F1, F2, F3, F4 = "01-Gruendung-August", "02-August", "03-September", "04-Oktober"

ME = Party("Nordlicht Digital GmbH", "Am Kaiserkai 19", "20457", "Hamburg",
           vat_id=de_vat("31846275"), tax_no="47/716/02345", email="hallo@nordlicht-digital.example",
           phone="040 3290 4410", contact="Jana Petersen",
           iban=iban("DE", "20041155" + "0742138800"), bic="HATBDEHHXXX", bank="Hanseatische Testbank AG",
           register="Amtsgericht Hamburg HRB 187654", gf="Jana Petersen", web="nordlicht-digital.example")
ME_IG = Party(ME.name + " i. G.", ME.street, ME.zip, ME.city)
ME_NOVAT = Party(ME.name, ME.street, ME.zip, ME.city)

JANA_IBAN = iban("DE", "20050550" + "1288430071")
OLE_IBAN = iban("DE", "43060967" + "2047719300")

ELBTECH = Party("Elbtech Computer GmbH", "Mönckebergstraße 7", "20095", "Hamburg", vat_id=de_vat("27519384"),
                email="service@elbtech.example", phone="040 6688 120", contact="Kai Lüders",
                iban=iban("DE", "20070000" + "0631977400"), bic="DEUTDEHHXXX", bank="Testbank Hamburg",
                register="Amtsgericht Hamburg HRB 99817", gf="Kai Lüders, Mareike Holm", web="elbtech.example")
COWORK = Party("Speicherstadt Coworking GmbH", "Bei St. Annen 2", "20457", "Hamburg", vat_id=de_vat("29473816"),
               email="buchhaltung@speicherstadt-cowork.example", phone="040 2263 7700", contact="Femke Jansen",
               iban=iban("DE", "20050550" + "1014332200"), bic="HASPDEHHXXX", bank="Hamburger Testsparkasse",
               register="Amtsgericht Hamburg HRB 140233", gf="Femke Jansen")
COWORK_CI = creditor_id("00000394712")
COWORK_MANDATE = "SC-NL-0042"
NOTAR = Party("Dr. Henrike Albers, Notarin", "Neuer Wall 42", "20354", "Hamburg", tax_no="46/201/33817",
              email="kanzlei@notarin-albers.example", phone="040 3570 990",
              iban=iban("DE", "20030000" + "0081556100"), bic="HYVEDEMM300", bank="Notaranderkonto Testbank")
ADWISE = Party("Adwise Ireland Ltd.", "4 Grand Canal Quay", "D02 X525", "Dublin 2", country="IE", country_name="Ireland",
               vat_id=ie_vat("6388047"), email="billing@adwise.example", register="Registered in Ireland, No. 368047",
               iban=iban("IE", "ADWI" + "990051" + "12345678"), bic="ADWIIE2DXXX", bank="Testbank Ireland")
FIGMO = Party("Figmo B.V.", "Herengracht 520", "1017 CC", "Amsterdam", country="NL", country_name="Netherlands",
              vat_id=nl_vat("85620371"), email="billing@figmo.example", register="KvK 74125530",
              iban=iban("NL", "INGB" + "0007731554"), bic="INGBNL2A", bank="Testbank Amsterdam")
FISCHMARKT = Party("Fischmarkt Feinkost GmbH", "Große Elbstraße 133", "22767", "Hamburg", vat_id=de_vat("19274638"),
                   email="einkauf@fischmarkt-feinkost.example", iban=iban("DE", "20050550" + "1209883100"))
BEZIRKSAMT_LEITWEG = leitweg("02000000-BAN0417")

LHK_IBAN = iban("DE", "20000000" + "0020001520")


def build() -> str:
    # ---------------------------------------------------------- 01 Gründung
    vertrag = [{
        "kind": "a4", "font": "Georgia", "accent": "#222222", "footer": [],
        "blocks": [
            {"type": "lines", "lines": ["UR-Nr. 1187/2026 A", "Ausfertigung"], "size": 8.5},
            {"type": "spacer", "mm": 4},
            {"type": "title", "text": "Gründung einer Gesellschaft mit beschränkter Haftung", "sub": "Verhandelt zu Hamburg am 03. August 2026"},
            {"type": "para", "justify": True, "text": "Vor mir, Dr. Henrike Albers, Notarin mit dem Amtssitz in Hamburg, erschienen heute, persönlich bekannt: "
             "1. Frau Jana Petersen, geb. 14.03.1991, wohnhaft Eppendorfer Weg 201, 20253 Hamburg; 2. Herr Ole Brandt, geb. 02.11.1988, "
             "wohnhaft Bahrenfelder Chaussee 55, 22761 Hamburg. Die Erschienenen errichten hiermit eine Gesellschaft mit beschränkter Haftung "
             "und stellen den folgenden Gesellschaftsvertrag fest."},
            {"type": "h", "text": "Gesellschaftsvertrag der Nordlicht Digital GmbH"},
            {"type": "para", "justify": True, "text": [{"t": "§ 1 Firma und Sitz. ", "b": True}, "Die Firma der Gesellschaft lautet Nordlicht Digital GmbH. Sitz der Gesellschaft ist Hamburg."]},
            {"type": "para", "justify": True, "text": [{"t": "§ 2 Gegenstand. ", "b": True}, "Gegenstand des Unternehmens ist die Konzeption, Gestaltung und Entwicklung von Websites, Web- und Mobilanwendungen sowie die Beratung hierzu."]},
            {"type": "para", "justify": True, "text": [{"t": "§ 3 Stammkapital. ", "b": True},
             "Das Stammkapital beträgt 25.000,00 €. Es ist eingeteilt in den Geschäftsanteil Nr. 1 zu 15.000,00 €, übernommen von Frau Jana Petersen, "
             "und den Geschäftsanteil Nr. 2 zu 10.000,00 €, übernommen von Herrn Ole Brandt. Die Einlagen sind in Geld zu erbringen, "
             "und zwar je zur Hälfte sofort. Der Rest ist auf Anforderung durch Gesellschafterbeschluss zu leisten."]},
            {"type": "para", "justify": True, "text": [{"t": "§ 4 Geschäftsjahr. ", "b": True}, "Das Geschäftsjahr ist das Kalenderjahr. Das erste Geschäftsjahr beginnt mit der Errichtung der Gesellschaft und endet am 31. Dezember 2026."]},
            {"type": "para", "justify": True, "text": [{"t": "§ 5 Geschäftsführung und Vertretung. ", "b": True}, "Die Gesellschaft hat einen oder mehrere Geschäftsführer. Ist nur ein Geschäftsführer bestellt, vertritt er die Gesellschaft allein. Zur ersten Geschäftsführerin wird Frau Jana Petersen bestellt; sie ist von den Beschränkungen des § 181 BGB befreit."]},
            {"type": "para", "justify": True, "text": [{"t": "§ 6 Gründungsaufwand. ", "b": True}, "Die Gesellschaft trägt die Kosten der Gründung (Notar, Gericht, Bekanntmachung) bis zu einem Betrag von 2.500,00 €. Darüber hinausgehende Kosten tragen die Gesellschafter im Verhältnis ihrer Geschäftsanteile."]},
            {"type": "para", "justify": True, "text": [{"t": "§ 7 Schlussbestimmungen. ", "b": True}, "Sollten einzelne Bestimmungen unwirksam sein, bleibt die Wirksamkeit der übrigen unberührt."]},
            {"type": "h", "text": "Gesellschafterliste"},
            {"type": "table", "widths": [16, "1fr", 34, 30], "align": ["l", "l", "r", "r"],
             "header": ["Nr.", "Gesellschafter", "Nennbetrag", "Anteil"],
             "rows": [["1", "Jana Petersen, Hamburg, geb. 14.03.1991", "15.000,00 €", "60 %"],
                      ["2", "Ole Brandt, Hamburg, geb. 02.11.1988", "10.000,00 €", "40 %"]]},
            {"type": "para", "text": "Vorgelesen, genehmigt und eigenhändig unterschrieben:"},
            {"type": "sig", "items": [{"label": "Jana Petersen", "sig": "J. Petersen"}, {"label": "Ole Brandt", "sig": "Ole Brandt"},
                                       {"label": "Dr. Henrike Albers, Notarin", "sig": "Albers"}]},
        ]}]
    compile_doc(T.path(F1, "2026-08-03_Gesellschaftsvertrag_UR-1187-2026.pdf"), "Gesellschaftsvertrag Nordlicht Digital GmbH", "Notarin Dr. Albers", vertrag)
    T.add(F1, "2026-08-03_Gesellschaftsvertrag_UR-1187-2026.pdf", "2026-08-03", "PDF", "Notarin Dr. Albers", "25.000,00 € Stammkapital", "Zeichnung des Stammkapitals, Halbeinzahlung (1298/2910 an 2900)")

    notar = Invoice("K-2026-1187", "2026-08-12", NOTAR, ME_IG, [
        Line("Beurkundung des Gesellschaftsvertrags (Geschäftswert 25.000 €), 2,0-Gebühr", 1, D("230.00"), unit_label="pausch."),
        Line("Anmeldung zum Handelsregister (Geschäftswert 30.000 €), 0,5-Gebühr", 1, D("62.50"), unit_label="pausch."),
        Line("Vollzug: Gesellschafterliste, 0,5-Gebühr", 1, D("57.50"), unit_label="pausch."),
        Line("Erzeugung strukturierter Daten (XML), 0,3-Gebühr", 1, D("37.50"), unit_label="pausch."),
        Line("Beglaubigung von Abschriften", 1, D("10.00"), unit_label="pausch."),
        Line("Dokumentenpauschale", 1, D("15.00"), unit_label="pausch."),
        Line("Post- und Telekommunikationsentgelte", 1, D("20.00"), unit_label="pausch."),
    ], delivery="2026-08-03", due="2026-08-26")
    efile(T, F1, "2026-08-12_Notarin-Albers_Kostenberechnung_K-2026-1187.pdf", notar, fmt="pdf", issuer="Notarin Dr. Albers",
          case="Gründungskosten als Aufwand (§ 248 HGB); PDF ohne Datensatz → Hinweis E-Rechnung, Buchen nur mit Rückfrage",
          pdf_kwargs=dict(font="Georgia", accent="#5a3e1b", title="Kostenberechnung (§ 19 GNotKG)",
                          intro="UR-Nr. 1187/2026 A – Gründung der Nordlicht Digital GmbH vom 03.08.2026",
                          extra_notes=["Zahlbar ohne Abzug innerhalb von 14 Tagen auf das unten genannte Konto."]))

    ag = letter_pages(Party("Amtsgericht Hamburg – Registergericht", "Caffamacherreihe 20", "20355", "Hamburg"),
                      ME.address_lines(), meta=[["Kassenzeichen", "7012 3456 7890 1"], ["Datum", "27.08.2026"], ["Aktenzeichen", "HRB 187654"]],
                      title="Kostenrechnung", font="Arial", accent="#30475e",
                      paras=["In der Registersache Nordlicht Digital GmbH werden folgende Kosten erhoben:",
                             {"type": "table", "widths": ["1fr", 30], "align": ["l", "r"], "header": ["Gegenstand", "Betrag"],
                              "rows": [["Ersteintragung einer Gesellschaft mit beschränkter Haftung (Nr. 2100 GV HRegGebV), eingetragen am 24.08.2026", "150,00 €"]]},
                             {"type": "totals", "rows": [["Zu zahlen", "150,00 €", True]]},
                             "Bitte überweisen Sie den Betrag bis zum 10.09.2026 unter Angabe des Kassenzeichens an die Landeshauptkasse Hamburg, "
                             f"IBAN {fmt_iban(LHK_IBAN)}, BIC MARKDEF1200.",
                             "Diese Kostenrechnung wurde maschinell erstellt und ist ohne Unterschrift gültig."],
                      footer=[["Amtsgericht Hamburg", "Registergericht"], ["Landeshauptkasse Hamburg", f"IBAN {fmt_iban(LHK_IBAN)}"]])
    compile_doc(T.path(F1, "2026-08-27_Amtsgericht-Hamburg_Kostenrechnung_HRB-187654.pdf"), "Kostenrechnung HRB 187654", "Amtsgericht Hamburg", ag)
    T.add(F1, "2026-08-27_Amtsgericht-Hamburg_Kostenrechnung_HRB-187654.pdf", "2026-08-27", "PDF", "Amtsgericht Hamburg", "150,00 €",
          "Gerichtsgebühr ohne Umsatzsteuer (nicht steuerbar), Gründungskosten")

    # ---------------------------------------------------------- 02 August
    elb = Invoice("2026-40877", "2026-08-04", ELBTECH, ME_IG, [
        Line("Notebook ThinkBook 14 G7", 1, D("1249.00"), detail="32 GB RAM, 1 TB SSD, Seriennr. PF4KX2Q9"),
        Line("USB-C Dockingstation 100 W", 1, D("149.00")),
    ], delivery="2026-08-04", means={"code": "48", "card": "4417", "holder": "Jana Petersen"})
    elb.prepaid = elb.gross
    efile(T, F2, "2026-08-04_Elbtech_Rechnung_2026-40877.pdf", elb, fmt="zugferd", issuer="Elbtech Computer GmbH",
          case="Auslage der Gesellschafterin Jana Petersen (3640); Notebook als Anlage, Dockingstation sofort; Rechnung an „i. G.“",
          pdf_kwargs=dict(font="Avenir Next", accent="#0b7a75", tagline="Computer · Service · Beratung",
                          pay_lines=["Betrag dankend erhalten.", "Bezahlt am 04.08.2026 mit girocard, Karte endet auf 4417 (Karteninhaberin: Jana Petersen)."]))

    bon = [{"kind": "bon", "font": "Menlo", "lines": [
        {"t": "Papier & Co. Schreibwaren", "a": "c", "b": True, "s": 10},
        {"t": "Inh. Karin Möller e.K.", "a": "c"},
        {"t": "Grindelallee 88 · 20146 Hamburg", "a": "c"},
        {"t": "StNr 43/551/20094", "a": "c"},
        "---",
        ["11.08.2026 15:42", "Bon 0817-2231"],
        ["Kasse 2", "Bed. Lisa"],
        "---",
        ["Ordner A4 breit  5 x 3,49", "17,45 A"],
        ["Kopierpapier 500 Bl 2 x 5,99", "11,98 A"],
        ["Kugelschreiber 10er", "6,99 A"],
        ["Notizbuch A5 dotted 2 x 5,59", "11,18 A"],
        "---",
        [{"t": "SUMME EUR", "b": True}, {"t": "47,60", "b": True}],
        ["BAR", "50,00"],
        ["Rückgeld", "2,40"],
        "---",
        ["Steuer  Netto  MwSt", "Brutto"],
        ["A 19%   40,00  7,60", "47,60"],
        "---",
        {"t": "TSE-Seriennr. 7f3a…c91e", "s": 7},
        {"t": "TSE-Start 2026-08-11T15:42:03", "s": 7},
        {"t": "Signaturzähler 88213", "s": 7},
        "---",
        {"t": "Vielen Dank für Ihren Einkauf!", "a": "c"},
    ]}]
    compile_doc(T.path(F2, "2026-08-11_Kassenbon_Papier-und-Co.jpg"), "Kassenbon", "Papier & Co.", bon, fmt="jpg", ppi=170)
    T.add(F2, "2026-08-11_Kassenbon_Papier-und-Co.jpg", "2026-08-11", "Foto (JPG)", "Papier & Co. Schreibwaren", "47,60 €",
          "Kleinbetragsrechnung als Foto, bar bezahlt von Ole Brandt → Auslage (3640)")

    def cowork(no, date, period, lines, debit):
        inv = Invoice(no, date, COWORK, ME_IG if date < "2026-08-24" else ME_NOVAT, lines, period=period, due=debit,
                      means={"code": "59", "debtor_iban": ME.iban, "mandate": COWORK_MANDATE, "creditor_id": COWORK_CI},
                      terms=f"Der Rechnungsbetrag wird am {ddate(debit)} per SEPA-Lastschrift eingezogen (Mandat {COWORK_MANDATE}, Gläubiger-ID {COWORK_CI}).",
                      customer_no="M-0042")
        return inv

    cw8 = cowork("SC-2026-08-0412", "2026-08-17", ("2026-08-17", "2026-08-31"),
                 [Line("Fixed Desk, Mitgliedschaft anteilig 17.–31.08.2026", 2, D("112.50"), unit_label="Platz")], "2026-08-19")
    efile(T, F2, "2026-08-17_Speicherstadt-Coworking_SC-2026-08-0412.pdf", cw8, fmt="zugferd", issuer="Speicherstadt Coworking GmbH",
          case="Laufende Kosten per Lastschrift; 1. von 3 Rechnungen desselben Lieferanten",
          pdf_kwargs=dict(font="Futura", accent="#b5462c", tagline="Arbeitsplätze in der Speicherstadt",
                          extra_notes=["Der Betrag wird per SEPA-Lastschrift eingezogen."]))

    # ---------------------------------------------------------- 03 September
    fa = letter_pages(Party("Finanzamt Hamburg-Altona", "Postfach 50 11 20", "22711", "Hamburg", phone="040 42811-0"),
                      ME.address_lines(), meta=[["Steuernummer", "47/716/02345"], ["Datum", "21.09.2026"]],
                      title="Steuerliche Erfassung – Mitteilung der Steuernummer", font="Arial", accent="#30475e",
                      paras=["Sehr geehrte Damen und Herren,",
                             "aufgrund Ihres Fragebogens zur steuerlichen Erfassung vom 26.08.2026 wurde Ihr Unternehmen unter der oben genannten Steuernummer erfasst. "
                             "Bitte geben Sie diese bei allen Schreiben und Zahlungen an.",
                             {"type": "kv", "label_width": 70, "rows": [
                                 ["Körperschaftsteuer", "veranlagt; Vorauszahlungen 2026: 0,00 €"],
                                 ["Gewerbesteuer", "Messbetrag wird vom Finanzamt festgesetzt; Hebesatz Hamburg 470 %"],
                                 ["Umsatzsteuer", "Voranmeldungszeitraum: Kalendervierteljahr (§ 18 Abs. 2 UStG), Abgabe elektronisch bis zum 10. Tag nach Ablauf des Vierteljahres"],
                                 ["Dauerfristverlängerung", "nicht beantragt"],
                                 ["Lohnsteuer", "keine Arbeitnehmer gemeldet"],
                                 ["USt-IdNr.", "wird vom Bundeszentralamt für Steuern gesondert mitgeteilt"]]},
                             "Mit freundlichen Grüßen", "Ihr Finanzamt Hamburg-Altona"],
                      footer=[["Finanzamt Hamburg-Altona"], ["Dieses Schreiben wurde maschinell erstellt."]])
    compile_doc(T.path(F3, "2026-09-21_Finanzamt-Altona_Steuernummer.pdf"), "Mitteilung Steuernummer", "Finanzamt Hamburg-Altona", fa)
    T.add(F3, "2026-09-21_Finanzamt-Altona_Steuernummer.pdf", "2026-09-21", "PDF", "Finanzamt Hamburg-Altona", "–",
          "Kein Buchungsbeleg: Steuernummer, Finanzamt und Voranmeldungszeitraum schlägt der Agent vor (`stammdaten_vorschlagen`), der Mensch übernimmt in der App")

    cw9 = cowork("SC-2026-09-0533", "2026-09-01", ("2026-09-01", "2026-09-30"),
                 [Line("Fixed Desk, Mitgliedschaft September 2026", 2, D("225.00"), unit_label="Platz"),
                  Line("Meetingraum „Elbblick“, 10.09.2026", 2, D("25.00"), unit="HUR", unit_label="Std.")], "2026-09-03")
    efile(T, F3, "2026-09-01_Speicherstadt-Coworking_SC-2026-09-0533.pdf", cw9, fmt="zugferd", issuer="Speicherstadt Coworking GmbH",
          case="2. Rechnung desselben Lieferanten, zwei Positionen", pdf_kwargs=dict(font="Futura", accent="#b5462c", tagline="Arbeitsplätze in der Speicherstadt"))

    ads = Invoice("5307711842", "2026-09-30", ADWISE, ME_NOVAT, [
        Line("Online advertising – campaign “Nordlicht Launch”, clicks September 2026", 1, D("320.00"), rate=D("23"), unit_label="lot"),
    ], period=("2026-09-01", "2026-09-30"), due="2026-10-02", means={"code": "59"},
        terms="Amount will be collected by SEPA direct debit on 2 October 2026.")
    efile(T, F3, "2026-09-30_Adwise-Ireland_Invoice_5307711842.pdf", ads, fmt="pdf", issuer="Adwise Ireland Ltd.",
          case="Irische USt 23 % zu Unrecht, weil keine USt-IdNr. vorlag: Leistungsort Inland, § 13b; irische Steuer nicht abziehbar",
          pdf_kwargs=dict(lang="en", font="Helvetica Neue", accent="#4285f4", tagline="Advertising services",
                          extra_notes=["Irish VAT (23 %) has been charged because no valid VAT identification number was provided for this account. "
                                       "Add your VAT ID in the account settings to have future invoices issued under the reverse-charge mechanism."]))

    # ---------------------------------------------------------- 04 Oktober
    beschluss = [{
        "kind": "a4", "font": "Georgia", "accent": "#222222", "footer": [],
        "blocks": [
            {"type": "title", "text": "Gesellschafterbeschluss der Nordlicht Digital GmbH", "sub": "im Umlaufverfahren gemäß § 48 Abs. 2 GmbHG"},
            {"type": "para", "justify": True, "text": "Die Gesellschafter der Nordlicht Digital GmbH mit Sitz in Hamburg (Amtsgericht Hamburg HRB 187654) halten zusammen "
             "sämtliche Geschäftsanteile. Unter Verzicht auf alle Form- und Fristvorschriften beschließen sie einstimmig:"},
            {"type": "para", "justify": True, "text": [{"t": "1. Einforderung der ausstehenden Einlagen. ", "b": True},
             "Die nach § 3 des Gesellschaftsvertrags noch nicht geleisteten Einlagen werden eingefordert (§ 46 Nr. 2 GmbHG):"]},
            {"type": "table", "widths": [16, "1fr", 34, 34], "align": ["l", "l", "r", "r"],
             "header": ["GA", "Gesellschafter", "Nennbetrag", "eingefordert"],
             "rows": [["1", "Jana Petersen", "15.000,00 €", "7.500,00 €"], ["2", "Ole Brandt", "10.000,00 €", "5.000,00 €"],
                      [{"t": "", "b": True}, {"t": "Summe", "b": True}, {"t": "25.000,00 €", "b": True}, {"t": "12.500,00 €", "b": True}]]},
            {"type": "para", "justify": True, "text": [{"t": "2. Fälligkeit. ", "b": True}, f"Die eingeforderten Beträge sind bis zum 15.10.2026 auf das Geschäftskonto der Gesellschaft, IBAN {fmt_iban(ME.iban)}, einzuzahlen."]},
            {"type": "para", "text": "Hamburg, den 01.10.2026"},
            {"type": "sig", "items": [{"label": "Jana Petersen (GA Nr. 1)", "sig": "J. Petersen"}, {"label": "Ole Brandt (GA Nr. 2)", "sig": "Ole Brandt"}]},
        ]}]
    compile_doc(T.path(F4, "2026-10-01_Gesellschafterbeschluss_Einforderung-Resteinlagen.pdf"), "Gesellschafterbeschluss Einforderung", "Nordlicht Digital GmbH", beschluss)
    T.add(F4, "2026-10-01_Gesellschafterbeschluss_Einforderung-Resteinlagen.pdf", "2026-10-01", "PDF", "Gesellschafter", "12.500,00 €",
          "Einforderung der Resteinlagen (1298 an 2910), danach Einzahlungen über die Bank")

    cw10 = cowork("SC-2026-10-0617", "2026-10-01", ("2026-10-01", "2026-10-31"),
                  [Line("Fixed Desk, Mitgliedschaft Oktober 2026", 2, D("225.00"), unit_label="Platz")], "2026-10-05")
    efile(T, F4, "2026-10-01_Speicherstadt-Coworking_SC-2026-10-0617.pdf", cw10, fmt="zugferd", issuer="Speicherstadt Coworking GmbH",
          case="3. Rechnung desselben Lieferanten; im Voraus berechnet und vor Leistungsende bezahlt",
          pdf_kwargs=dict(font="Futura", accent="#b5462c", tagline="Arbeitsplätze in der Speicherstadt"))

    ihk = letter_pages(Party("Handelskammer Hamburg", "Adolphsplatz 1", "20457", "Hamburg", phone="040 36138-0"),
                       ME.address_lines(), meta=[["Beitragsnummer", "2026-4471882"], ["Datum", "02.10.2026"]],
                       title="Beitragsbescheid 2026", font="Helvetica Neue", accent="#7a1e2c",
                       paras=["Sehr geehrte Damen und Herren,",
                              "Ihr Unternehmen ist seit der Eintragung in das Handelsregister am 24.08.2026 Mitglied der Handelskammer Hamburg (§ 2 IHKG). "
                              "Für das Beitragsjahr 2026 setzen wir folgenden Beitrag fest:",
                              {"type": "table", "widths": ["1fr", 30], "align": ["l", "r"], "header": ["Beitragsart", "Betrag"],
                               "rows": [["Grundbeitrag 2026, anteilig ab 24.08.2026", "100,00 €"], ["Umlage (vorläufig, Gewerbeertrag noch nicht bekannt)", "0,00 €"]]},
                              {"type": "totals", "rows": [["Festgesetzter Beitrag", "100,00 €", True]]},
                              "Der Beitrag ist fällig am 02.11.2026. Der Beitrag unterliegt nicht der Umsatzsteuer.",
                              "Rechtsbehelfsbelehrung: Gegen diesen Bescheid kann innerhalb eines Monats nach Bekanntgabe Klage beim Verwaltungsgericht Hamburg erhoben werden."],
                       footer=[["Handelskammer Hamburg"], [f"IBAN {fmt_iban(iban('DE', '20050550' + '1280000110'))}"]])
    compile_doc(T.path(F4, "2026-10-02_Handelskammer_Beitragsbescheid-2026.pdf"), "Beitragsbescheid 2026", "Handelskammer Hamburg", ihk)
    T.add(F4, "2026-10-02_Handelskammer_Beitragsbescheid-2026.pdf", "2026-10-02", "PDF", "Handelskammer Hamburg", "100,00 €",
          "Beitrag ohne Umsatzsteuer, auf Ziel (fällig 02.11.2026, faellig_am) → bleibt offener Posten")

    bzst = letter_pages(Party("Bundeszentralamt für Steuern", "Dienstsitz Saarlouis", "66738", "Saarlouis"),
                        ME.address_lines(), meta=[["Datum", "05.10.2026"], ["Steuernummer", "47/716/02345"]],
                        title="Mitteilung über die Erteilung einer Umsatzsteuer-Identifikationsnummer", font="Arial", accent="#30475e",
                        paras=["Sehr geehrte Damen und Herren,",
                               [{"t": "für Ihr Unternehmen wurde folgende Umsatzsteuer-Identifikationsnummer erteilt: "}, {"t": ME.vat_id, "b": True}],
                               "Die Umsatzsteuer-Identifikationsnummer ist bei innergemeinschaftlichen Lieferungen und sonstigen Leistungen anzugeben "
                               "(§ 14a UStG). Die Zuständigkeit Ihres Finanzamts für die Besteuerung bleibt unberührt.",
                               "Dieses Schreiben wurde maschinell erstellt und ist nicht unterschrieben."])
    compile_doc(T.path(F4, "2026-10-05_BZSt_USt-IdNr.pdf"), "Mitteilung USt-IdNr.", "BZSt", bzst)
    T.add(F4, "2026-10-05_BZSt_USt-IdNr.pdf", "2026-10-05", "PDF", "Bundeszentralamt für Steuern", "–",
          "Kein Buchungsbeleg: USt-IdNr. schlägt der Agent vor (`stammdaten_vorschlagen`), der Mensch übernimmt in der App")

    fig = Invoice("FIG-2026-118842", "2026-10-06", FIGMO, ME, [
        Line("Professional plan – annual subscription, editor seat", 2, D("144.00"), rate=D("0"), cat="AE", unit_label="seat"),
    ], period=("2026-10-06", "2027-10-05"), due="2026-10-07", means={"code": "59"},
        terms="Paid by SEPA direct debit on 7 October 2026.")
    efile(T, F4, "2026-10-06_Figmo_Invoice_FIG-2026-118842.pdf", fig, fmt="pdf", issuer="Figmo B.V.",
          case="Sonstige Leistung aus der EU, § 13b Abs. 1 UStG (Kz 46/47/67); Jahresabo über den Bilanzstichtag → Rechnungsabgrenzung zum 31.12.",
          pdf_kwargs=dict(lang="en", font="Helvetica Neue", accent="#a259ff", tagline="Design, together",
                          extra_notes=["VAT reverse charged – Article 196 Council Directive 2006/112/EC. The customer is liable to account for the VAT."]))

    # ---------------------------------------------------------- Bank
    BANK = dict(owner="Nordlicht Digital GmbH", iban=ME.iban, bic=ME.bic, bank=ME.bank)
    s1 = Statement(**BANK, seq=1, start="2026-08-04", end="2026-08-31", opening=D("0"), txs=[
        Tx("2026-08-06", D("7500.00"), "Jana Petersen", JANA_IBAN, "Einzahlung Stammeinlage GA Nr. 1 Nordlicht Digital GmbH i.G. UR 1187/2026"),
        Tx("2026-08-07", D("5000.00"), "Ole Brandt", OLE_IBAN, "Stammeinlage Geschaeftsanteil Nr. 2 - 50 Prozent"),
        Tx("2026-08-14", -elb.gross, "Jana Petersen", JANA_IBAN, "Erstattung Auslage Elbtech RE 2026-40877 Notebook"),
        Tx("2026-08-14", D("-47.60"), "Ole Brandt", OLE_IBAN, "Erstattung Auslage Buerobedarf Papier und Co 11.08."),
        Tx("2026-08-17", -notar.gross, NOTAR.name, NOTAR.iban, "Kostenberechnung K-2026-1187 UR 1187/2026"),
        Tx("2026-08-19", -cw8.gross, COWORK.name, COWORK.iban, f"{cw8.number} Fixed Desk 17.-31.08.2026", kind="Lastschrift", mandate=COWORK_MANDATE, creditor_id=COWORK_CI),
        Tx("2026-08-31", D("-9.90"), "Hanseatische Testbank AG", "", "Kontofuehrungsentgelt 08/2026", kind="Entgelt"),
    ])
    s2 = Statement(**BANK, seq=2, start="2026-09-01", end="2026-09-30", opening=s1.closing, txs=[
        Tx("2026-09-02", D("-150.00"), "Landeshauptkasse Hamburg", LHK_IBAN, "Kassenzeichen 7012345678901 AG Hamburg HRB 187654"),
        Tx("2026-09-03", -cw9.gross, COWORK.name, COWORK.iban, f"{cw9.number} Fixed Desk 09/2026", kind="Lastschrift", mandate=COWORK_MANDATE, creditor_id=COWORK_CI),
        Tx("2026-09-30", D("-9.90"), "Hanseatische Testbank AG", "", "Kontofuehrungsentgelt 09/2026", kind="Entgelt"),
    ])
    s3 = Statement(**BANK, seq=3, start="2026-10-01", end="2026-10-16", opening=s2.closing, txs=[
        Tx("2026-10-02", -ads.gross, ADWISE.name, ADWISE.iban, "Adwise Ads invoice 5307711842", kind="Lastschrift", mandate="ADW-7731-2209"),
        Tx("2026-10-05", -cw10.gross, COWORK.name, COWORK.iban, f"{cw10.number} Fixed Desk 10/2026", kind="Lastschrift", mandate=COWORK_MANDATE, creditor_id=COWORK_CI),
        Tx("2026-10-06", D("7500.00"), "Jana Petersen", JANA_IBAN, "Resteinlage GA Nr. 1 lt. Beschluss vom 01.10.2026"),
        Tx("2026-10-07", -fig.gross, FIGMO.name, FIGMO.iban, "FIG-2026-118842 Figmo Professional annual", kind="Lastschrift", mandate="FIGMO-55120"),
        Tx("2026-10-09", D("5000.00"), "Ole Brandt", OLE_IBAN, "Resteinlage GA 2 Nordlicht"),
        Tx("2026-10-13", D("6473.60"), FISCHMARKT.name, FISCHMARKT.iban, "RE-2026-0001 Website Relaunch"),
    ])
    for folder, st, name in [(F2, s1, "2026-08-31_Kontoauszug_08-2026_camt053.xml"), (F3, s2, "2026-09-30_Kontoauszug_09-2026_camt053.xml"),
                             (F4, s3, "2026-10-16_Kontoauszug_10-2026_camt053.xml")]:
        T.write(folder, name, camt(st))
        T.add(folder, name, st.end, "CAMT.053", ME.bank, f"{len(st.txs)} Umsätze, Endsaldo {de(st.closing, '€')}",
              "Kontoauszug" + (" – den ersten liest der Mensch in der App ein" if st.seq == 1 else ""))

    vst_q3 = elb.tax + D("7.60") + notar.tax + cw8.tax + cw9.tax
    ust_q3 = D("1033.60")
    ads_13b = (ads.net * D("0.19")).quantize(D("0.01"))
    return readme(locals())


def readme(v) -> str:
    s1, s2, s3 = v["s1"], v["s2"], v["s3"]
    return f"""# Mandant 1: Nordlicht Digital GmbH (Hamburg)

**Schwerpunkt: Gründung.** Zwei Gesellschafter gründen eine GmbH mit 25.000 € Stammkapital und zahlen zunächst
die Hälfte ein. Die Bücher beginnen mit der Beurkundung am 03.08.2026, das erste Geschäftsjahr ist ein Rumpfjahr
bis 31.12.2026. Dazu kommen Auslagen vor der Kontoeröffnung, die ersten Lieferanten, Post vom Finanzamt, die
Einforderung der Resteinlagen und die ersten Ausgangsrechnungen, eine davon als XRechnung an eine Behörde.

## 1. Einrichtung in der App

Ablage und Eingang bitte in leeren Ordnern außerhalb dieses Testordners anlegen, z. B.
`~/Buchfink-Mandanten/nordlicht/` (Ablage) und `~/Buchfink-Mandanten/nordlicht/Eingang` (Eingang).

| Feld im Einrichtungsassistenten | Wert |
|---|---|
| Firma | Nordlicht Digital GmbH |
| Rechtsform | GmbH |
| Wie beginnen die Bücher? | Neu gegründet |
| Tag der Beurkundung | 03.08.2026 |
| Geschäftsjahr | Kalenderjahr (Januar) |
| Bundesland | Hamburg |
| Ausschlüsse | keiner (nicht gemeinnützig, nicht in Liquidation, keine Organschaft) |

**Unternehmensdaten** (Unternehmen › Stammdaten; der Agent schlägt vor, der Mensch übernimmt in der App):

| Feld | Wert | ab wann bekannt |
|---|---|---|
| Anschrift | Am Kaiserkai 19, 20457 Hamburg | sofort |
| Geschäftsführung | Jana Petersen | sofort |
| Ansprechpartner, Telefon, E-Mail | Jana Petersen, 040 3290 4410, hallo@nordlicht-digital.example | sofort (Pflicht für die XRechnung in Schritt 4) |
| Notar, Urkunde | Dr. Henrike Albers, Hamburg; UR-Nr. 1187/2026 A | sofort |
| Registergericht, Nummer | Amtsgericht Hamburg, HRB 187654 (eingetragen 24.08.2026) | Schritt 1 |
| Steuernummer, Finanzamt | 47/716/02345, Finanzamt Hamburg-Altona | Schritt 3 (Brief vom 21.09.) |
| Voranmeldungszeitraum | vierteljährlich, keine Dauerfristverlängerung | Schritt 3 |
| USt-IdNr. | {ME.vat_id} | Schritt 4 (Brief vom 05.10.) |
| Steuerliche Beratung | keine | sofort |
| Bankkonto | {ME.bank}, IBAN {fmt_iban(ME.iban)}, BIC {ME.bic}, Konto für Rechnungen | sofort |

**Gründung** (Unternehmen › Stammdaten; der Agent schlägt vor, der Mensch übernimmt in der App): Stammkapital 25.000,00 €; Geschäftsanteil Nr. 1 Jana Petersen
15.000,00 €, Nr. 2 Ole Brandt 10.000,00 €; Einlagen in Geld, je zur Hälfte sofort fällig; Gründungsaufwand laut
Satzung bis 2.500,00 €. Eintragung ins Handelsregister am 24.08.2026 erst nach Schritt 1 erfassen.

## 2. Agent verbinden

Einstellungen › Apps verbinden, Client mit `http://127.0.0.1:7465/mcp` verbinden, Abgleichcode im Fenster bestätigen.
Erster Prompt: „Ruf `status` auf und sag mir, was als Nächstes zu tun ist.“ Erwartet: Mandant Nordlicht Digital GmbH,
Rumpfjahr 2026, Pflichten der Gründung im Kalender (Fragebogen zur steuerlichen Erfassung, Transparenzregister,
Gewerbeanmeldung).

## 3. Testschritte

Alle Belege eines Schritts in das Fenster von Buchfink ziehen (oder in den Eingangsordner kopieren) und dem
Agenten sagen: „Buche die Belege im Eingang.“ Die Belege gelten als am Belegdatum eingegangen; der Agent soll
`eingang_am` entsprechend setzen, sonst landet die Vorsteuer im Oktober.

### Schritt 1 – Gründung (`{F1}/`)

{T.table(F1)}

Erwartete Buchungen:

| Vorgang | Buchung |
|---|---|
| Zeichnung (Agent, `buchung_erfassen` mit Rückfrage, Beleg Gesellschaftsvertrag) | 1298 Ausstehende Einlagen, eingefordert 12.500,00 + 2910 nicht eingefordert 12.500,00 an 2900 Gezeichnetes Kapital 25.000,00 |
| Notarin (Rückfrage, kein Datensatz) | 6825 Rechts- und Beratungskosten {de(v['notar'].net)} + 1406 Vorsteuer {de(v['notar'].tax)} an Kreditor {de(v['notar'].gross)} |
| Amtsgericht (Steuerfall nicht steuerbar) | Gruppe `gebuehren`: 6430 Sonstige Abgaben 150,00 an Kreditor 150,00; unter `beitraege` (6420) warnt die Vorschau |

Prüfpunkte: Die Gründung (Gesellschafter, Anteile, Eintragung) schlägt der Agent mit `stammdaten_vorschlagen` vor;
übernehmen kann sie nur der Mensch in der App. Den Gesellschaftsvertrag legt der Agent mit `art: sonstiges` ab, er ist der
Beleg der Zeichnung. Die Notarrechnung ist ein PDF ohne Datensatz; Buchfink zeigt den Hinweis zur E-Rechnungspflicht,
blockiert aber nicht. Nach dem Schritt die Eintragung (24.08.2026, HRB 187654, Amtsgericht Hamburg) vom Agenten vorschlagen
lassen und in der App übernehmen.

Pflichten der Eröffnungsbilanz (seit `94ae1db`): „Eröffnungsbilanz aufstellen und unterzeichnen“ fällig 03.11.2026
(drei Monate nach der Beurkundung, Unternehmen › Stammdaten › „Gesellschafter & Kapital“, Knopf „Als PDF speichern“),
„E-Bilanz der Eröffnungsbilanz“ fällig 02.08.2027 (Abschlüsse › Erklärungen, Export mit `bilanzart: eroeffnungsbilanz`).
Der Agent prüft die Zahlen und meldet höchstens „vorbereitet“. Im selben Block zeigt Buchfink die Unterbilanzrechnung
(für den Agenten `auswertung` mit `art: unterbilanz`): Gründungskosten 582,50 € netto liegen unter den 2.500 € der Satzung.

### Schritt 2 – August (`{F2}/`)

{T.table(F2)}

Erwartet:

| Vorgang | Buchung |
|---|---|
| Notebook als Auslage Jana Petersen | 0650 Büroeinrichtung bzw. EDV-Anlage 1.249,00 (Anlage, Nutzungsdauer klären) + Aufwand/GWG 149,00 + 1406 {de(v['elb'].tax)} an 3640 Gesellschafterin Jana Petersen {de(v['elb'].gross)} |
| Kassenbon als Auslage Ole Brandt | 6815 Bürobedarf 40,00 + 1406 7,60 an 3640 Ole Brandt 47,60 |
| Coworking August | Raumkosten/Miete 225,00 + 1406 {de(v['cw8'].tax)} an Kreditor {de(v['cw8'].gross)} |
| Einzahlungen 7.500 / 5.000 | 1800 an 1298, je Gesellschafter |
| Erstattungen der Auslagen | 3640 an 1800 (Kontakt der Gesellschafterin bzw. des Gesellschafters) |
| Notarin, Coworking | Zuordnung zum offenen Posten |
| Kontoführung 9,90 | 6855 Nebenkosten des Geldverkehrs an 1800 |

Endsaldo 31.08.: **{de(s1.closing, '€')}**. Prüfpunkte: Den ersten Kontoauszug liest der Mensch in der App ein
(Bücher › Bank); der Agent bittet darum, bevor er importiert. Liest der Agent ihn selbst ein, bleibt er „ungeprüft“,
der Hinweis nennt den fehlenden geprüften Saldo, und `bankumsatz_zuordnen` bucht nicht ohne Rückfrage. Die
Elbtech-Rechnung lautet auf „Nordlicht Digital GmbH i. G.“: kein Befund erwartet, die Vorgesellschaft und die
eingetragene GmbH sind dasselbe Unternehmen. Wer den Beleg ausgestellt hat (Elbtech), darf nicht zugleich die
Erstattung bekommen.

### Schritt 3 – September (`{F3}/`)

{T.table(F3)}

Dazu die erste Ausgangsrechnung über den Agenten (`rechnung_vorschau`, dann `rechnung_ausstellen`):

| Feld | Wert |
|---|---|
| Kunde (neu anlegen) | {FISCHMARKT.name}, {FISCHMARKT.street}, {FISCHMARKT.zip} {FISCHMARKT.city}, USt-IdNr. {FISCHMARKT.vat_id} |
| Rechnungsdatum | Tag des Tests (ein früheres Datum lehnt der Agent ab, § 14 Abs. 4 Satz 1 Nr. 3 UStG) |
| Leistungszeitraum | 01.09.–25.09.2026 |
| Positionen | „Konzeption und Screendesign Website-Relaunch“ 1 × 2.400,00 €; „Umsetzung Frontend“ 32 Std. × 95,00 € (19 %) |
| Zahlungsziel | 14 Tage |
| Erwartet | Nummer RE-2026-0001, netto 5.440,00 €, USt 1.033,60 €, brutto 6.473,60 €, PDF und ZUGFeRD im Exportordner |

Die Rechnung vor dem Oktober-Auszug ausstellen: dessen Zahlungseingang vom 13.10. nennt schon RE-2026-0001. Buchfink
bucht Erlös und Forderung zum Ende der Leistung am 25.09.2026 (§ 252 Abs. 1 Nr. 4 HGB), solange der September nicht
festgeschrieben ist; sonst zum Rechnungsdatum mit Hinweis. Die Umsatzsteuer gehört nach dem Leistungszeitraum in Q3
(§ 13 Abs. 1 Nr. 1 Buchst. a UStG).

Prüfpunkte: Den Brief des Finanzamts legt der Agent mit `art: brief` ab (kein Buchungsbeleg, blockiert den Prüfbericht
nicht) und schlägt Steuernummer, Finanzamt und Voranmeldungszeitraum mit `stammdaten_vorschlagen` vor; der Mensch
übernimmt in der App. Adwise berechnet irische Umsatzsteuer, obwohl die Werbeleistung an ein Unternehmen im Inland
steuerbar ist (§ 3a Abs. 2 UStG). Die GmbH schuldet die Steuer nach § 13b Abs. 1, 5 UStG: 6600 Werbekosten
{de(v['ads'].net)} + 1407 {de(v['ads_13b'])} an 3837 {de(v['ads_13b'])} + Kreditor {de(v['ads'].net)}. Die irische Steuer
{de(v['ads'].gross - v['ads'].net, '€')} ist nicht abziehbar; sie wird bei der Zahlung im Oktober Aufwand. Endsaldo 30.09.:
**{de(s2.closing, '€')}**.

**Voranmeldung III/2026** (fällig 12.10.2026, im Kalender also überfällig): Umsatzsteuer {de(v['ust_q3'], '€')} (Kz 81: 5.440),
§ 13b {de(v['ads_13b'], '€')} (Kz 46: {de(v['ads'].net)}), Vorsteuer {de(v['vst_q3'], '€')} (Kz 66) und {de(v['ads_13b'], '€')} (Kz 67),
Zahllast **{de(v['ust_q3'] - v['vst_q3'], '€')}**. Voraussetzung: alle Belege mit Eingang am Belegdatum gebucht.

### Schritt 4 – Oktober (`{F4}/`)

{T.table(F4)}

Dazu die zweite Ausgangsrechnung, diesmal an eine Behörde als XRechnung:

| Feld | Wert |
|---|---|
| Kunde (neu anlegen) | Bezirksamt Hamburg-Nord, Fachamt Digitalisierung, Kümmellstraße 6, 20249 Hamburg |
| E-Rechnungsformat, Leitweg-ID | XRechnung, `{BEZIRKSAMT_LEITWEG}` (der Agent legt den Kunden mit `leitweg_id` an) |
| Rechnungsdatum | Tag des Tests |
| Leistungszeitraum | 05.10.–08.10.2026 |
| Position | „Barrierefreiheitsprüfung Online-Formular nach BITV 2.0“, 12 Std. × 95,00 € (19 %) |
| Zahlungsziel | 30 Tage |
| Erwartet | RE-2026-0002, brutto 1.356,60 €, XRechnung (CII) ohne Befund; bleibt offen |

Erwartete Buchungen:

| Vorgang | Buchung |
|---|---|
| Einforderung (Beleg: Beschluss) | 1298 an 2910 12.500,00 |
| Resteinlagen 7.500 / 5.000 | 1800 an 1298 |
| Figmo, § 13b Abs. 1 | Software/EDV-Kosten 288,00 + 1407 Vorsteuer 54,72 an 3837 Umsatzsteuer 54,72 + Kreditor 288,00 |
| IHK | 6420 Beiträge 100,00 an Kreditor 100,00, `faellig_am` 02.11.2026 laut Bescheid (offen, nicht überfällig) |
| Zahlung Adwise | Kreditor {de(v['ads'].net)} + 6600 {de(v['ads'].gross - v['ads'].net)} (irische Steuer) an 1800 {de(v['ads'].gross)} |
| Zahlungseingang Fischmarkt | 1800 an Debitor 6.473,60 (Zuordnung über „RE-2026-0001“ im Verwendungszweck) |

Endsaldo 16.10.: **{de(s3.closing, '€')}**. Prüfpunkte: Den BZSt-Brief legt der Agent mit `art: brief` ab und schlägt die
USt-IdNr. vor; der Mensch übernimmt sie in der App. Figmo ist § 13b; weil vor Leistungsende bezahlt, gehört die Steuer
in Q4 2026 (§ 13b Abs. 4 Satz 2 UStG). Coworking Oktober ist vor Leistungsende bezahlt (Vorsteuer bei Zahlung). Die
Unterbilanzrechnung zum Eintragungstag muss die Gründungskosten bis 2.500 € decken. Figmo läuft bis 05.10.2027:
im Jahresabschluss 2026 wäre ein Rechnungsabgrenzungsposten fällig.

## 4. Kontrollwerte zum 16.10.2026

| Konto / Stand | Erwartung |
|---|---|
| 1800 Bank | {de(s3.closing, '€')} |
| 2900 Gezeichnetes Kapital | 25.000,00 € |
| 1298 / 2910 | 0,00 € / 0,00 € (alles eingefordert und eingezahlt) |
| 3640 Gesellschafter | 0,00 € (Auslagen erstattet) |
| Offene Posten | Forderung Bezirksamt 1.356,60 €; Verbindlichkeit Handelskammer 100,00 € |
"""
