#!/usr/bin/env python3

"""
DATEV SKR 04 (BilRUG 2026) Parser & Position Generator
Extracts all accounts, positions, categories, tax functions, footnotes and hierarchy
directly from the official DATEV SKR04 PDF with exact bounding boxes.

The source PDF is not part of this repository. Obtain it from DATEV and place
it at assets/DATEV-SKR04-BilrUg-2026.pdf before running this script.
"""

import pdfplumber
import re
import json
import os
from collections import OrderedDict

# ----------------------------------------------------------------------
# 1. Official Definitions & Metadata
# ----------------------------------------------------------------------

FOOTNOTE_EXPLANATIONS = {
    1: "Konto für das Buchungsjahr 2026 neu eingeführt.",
    2: "Bilanz- und GuV-Posten: Auswertung für große Kapitalgesellschaften, GuV-Gesamtkostenverfahren.",
    3: "Diese Konten können mit BU-Schlüssel 10 bebucht werden. Das EU-Land und der ausländische Steuersatz werden über das EU-Fenster eingegeben.",
    4: "Kontenbezogene Kennzeichnung der Programmverbindung in Rechnungswesen-Programmen zu Umsatzsteuererklärung (U), Gewerbesteuer (G) und Körperschaftsteuer (K). Abschlusszweck: HB (Handelsbilanz), SB (Steuerbilanz), EÜR (§ 4 Abs. 3 EStG).",
    5: "Dieses Konto kann mit BU-Schlüssel 44 bebucht werden. Das EU-Land und der ausländische Steuersatz werden über das EU-Fenster eingegeben.",
    6: "Das Konto gilt als Hauptkonto für Sachverhalte, die in diesen Kontenbereichen nicht als spezieller Sachverhalt auf Einzelkonten dargestellt sind.",
    7: "Diese Konten werden für Branchen-BWA-Formen mit statistischen Mengeneinheiten bebucht und wurden mit der Umrechnungssperre, Funktion 18000, belegt.",
    8: "Kontenbeschriftung in 2026 geändert.",
    9: "An der Schnittstelle zu Gewerbesteuer werden ab VAZ 2009 die Erträge zu 40 % als steuerfrei und die Aufwendungen zu 40 % als nicht abziehbar behandelt. An der Schnittstelle zu Körperschaftsteuer werden die Erträge zu 100 % als steuerfrei und die Aufwendungen zu 100 % als nicht abziehbar behandelt. Siehe §§ 3 Nr. 40 und 3c EStG bzw. § 8b KStG.",
    10: "Diese Konten haben nicht mehr die Zusatzfunktion KU. Bitte verwenden Sie diese Konten nur noch in Verbindung mit einem Gegenkonto mit Geldkontenfunktion.",
    11: "Das Konto wird nur noch für Auswertungen mit Vorjahresvergleich benötigt und wird im folgenden Jahr gelöscht.",
    12: "Das Konto wird im folgenden Jahr gelöscht, der Kontenzweck bleibt erhalten.",
    15: "Das Konto wurde zur Aufteilung nach Steuersätzen am Jahresende eingerichtet und sollte unterjährig nicht bebucht werden. Beachten Sie das Dokument Buchungsregeln: Zusätzliche Buchungen zum korrekten Befüllen der Anlage EÜR (Dok.-Nr. 0906057).",
    16: "Das Konto wird in Körperschaftsteuer nur bei Organgesellschaften berücksichtigt.",
    17: "Das Konto wird in Körperschaftsteuer ausschließlich in die Positionen „Eigen-/Nennkapital zum Schluss des vorangegangenen Wirtschaftsjahres“ übernommen.",
    18: "Da das EÜR-Formular einen differenzierten Ausweis der Reisekosten und Fahrzeugkosten fordert, darf dieses Konto von EÜR-Anwendern nicht genutzt werden.",
    21: "Diese Konten können mit BU-Schlüssel 94 (Konto mit Vorsteuerabzug) bzw. mit BU-Schlüssel 95 (Konto ohne Vorsteuerabzug) gebucht werden. Der Tatbestand des § 13b UStG ist anschließend zu erfassen.",
    22: "Ab dem Buchungsjahr 2019 dürfen die Konten nur noch für Einzelunternehmer verwendet werden.",
    23: "Diese Konten fließen im EÜR-Formular in die Zeile Ergebnisanteile aus Beteiligungen an Personengesellschaften.",
    24: "Diese Konten werden für die BWA-Formen der Branchenlösung bebucht.",
    28: "Das Konto wird nur in den Auswertungen für Sonderbilanzen abgefragt."
}

FUNCTION_DEFINITIONS = {
    "AV": "Automatische Errechnung der Vorsteuer",
    "AM": "Automatische Errechnung der Umsatzsteuer",
    "S": "Sammelkonto",
    "F": "Konto mit allgemeiner Funktion",
    "R": "Reserviertes Konto (darf erst bebucht werden, wenn ihm eine andere Funktion zugeteilt wurde)",
    "S/AM": "Sammelkonto / Automatische Errechnung der Umsatzsteuer",
    "S/AV": "Sammelkonto / Automatische Errechnung der Vorsteuer"
}

TAX_FUNCTION_DEFINITIONS = {
    "KU": "Keine Errechnung der Umsatzsteuer möglich",
    "V": "Zusatzfunktion Vorsteuer",
    "M": "Zusatzfunktion Umsatzsteuer"
}

KLASSEN_INFO = {
    0: {"name": "Anlagevermögenskonten", "statement": "Bilanz", "type": "asset"},
    1: {"name": "Umlaufvermögenskonten", "statement": "Bilanz", "type": "asset"},
    2: {"name": "Eigenkapitalkonten / Fremdkapitalkonten", "statement": "Bilanz", "type": "equity"},
    3: {"name": "Fremdkapitalkonten", "statement": "Bilanz", "type": "liability"},
    4: {"name": "Betriebliche Erträge", "statement": "GuV", "type": "revenue"},
    5: {"name": "Betriebliche Aufwendungen (Material/Fremdleistungen)", "statement": "GuV", "type": "expense"},
    6: {"name": "Betriebliche Aufwendungen (Personal/AfA/Sonstige)", "statement": "GuV", "type": "expense"},
    7: {"name": "Weitere Erträge und Aufwendungen (Finanz/Steuern)", "statement": "GuV", "type": "other"},
    8: {"name": "Künftige Verwendung bei gesetzlichen Anforderungen durch DATEV möglich", "statement": "GuV", "type": "statistical"},
    9: {"name": "Vortrags-, Kapital-, Korrektur- und statistische Konten", "statement": "Statistisch", "type": "statistical"}
}

FOOTNOTE_IDS = set(FOOTNOTE_EXPLANATIONS.keys())


def slugify(text):
    if not text:
        return ""
    text = text.lower()
    text = text.replace('ä', 'ae').replace('ö', 'oe').replace('ü', 'ue').replace('ß', 'ss')
    text = re.sub(r'[^a-z0-9]+', '_', text)
    return text.strip('_')


def clean_german_text(tokens):
    if not tokens:
        return ""
    lines = [t.strip() for t in tokens if t.strip()]
    if not lines:
        return ""
    
    result = ""
    for line in lines:
        if not result:
            result = line
            continue
            
        if result.endswith("-"):
            first_word_next = line.split()[0] if line.split() else ""
            last_word_prev = result.split()[-1] if result.split() else ""
            
            keep_hyphen = False
            if result.endswith("-,") or result.endswith("- /"):
                keep_hyphen = True
            elif first_word_next in ("und", "oder", "bzw.", "sowie", "auch"):
                keep_hyphen = True
            elif any(last_word_prev.startswith(prefix) for prefix in ("GmbH-", "EU-", "EStG-", "USt-", "EDV-", "Kfz-", "BStBK-", "KapCo-", "BilMoG-", "GewSt-", "KSt-", "DBA-")):
                keep_hyphen = True
            elif last_word_prev in ("Geschäfts-,", "Fabrik-,", "Soll-", "Haben-", "Vor-", "Miet-", "Pacht-", "Roh-,", "Hilfs-", "Betriebs-", "Werbe-", "Repräsentations-"):
                keep_hyphen = True
            
            if keep_hyphen:
                result = result + " " + line
            else:
                result = result[:-1] + line
        else:
            result = result + " " + line
            
    result = re.sub(r'\s+', ' ', result).strip()
    return result


def clean_account_name_and_footnotes(name_str):
    fns = []
    
    # 1. Isolated footnote: e.g. "7)"
    if re.match(r'^\d{1,2}\)$', name_str.strip()):
        fn_val = int(name_str.strip()[:-1])
        if fn_val in FOOTNOTE_IDS:
            return "(Statistisches Mengenkonto)", [fn_val]

    # 2. Year + footnote: e.g. "20261)" -> "2026", fn 1
    m_year = re.search(r'2026(\d{1,2})\)', name_str)
    if m_year:
        fn_val = int(m_year.group(1))
        if fn_val in FOOTNOTE_IDS:
            fns.append(fn_val)
            name_str = name_str[:m_year.start()] + "2026" + name_str[m_year.end():]
            
    # 3. Account range + footnote: e.g. "9101-91028)" -> "9101-9102", fn 8
    m_range_fn = re.search(r'(\d{4}(?:-\d{4})?)(\d{1,2})\)', name_str)
    if m_range_fn:
        fn_val = int(m_range_fn.group(2))
        if fn_val in FOOTNOTE_IDS:
            fns.append(fn_val)
            name_str = name_str[:m_range_fn.start()] + m_range_fn.group(1) + name_str[m_range_fn.end():]

    # 4. Repeatedly find and strip footnote markers attached to word or after balanced parens
    while True:
        # Check attached to letter: e.g. "Kapital17)"
        m = re.search(r'([a-zA-ZäöüÄÖÜß])(\d{1,2})\)', name_str)
        if m and int(m.group(2)) in FOOTNOTE_IDS:
            fn_val = int(m.group(2))
            if fn_val not in fns:
                fns.append(fn_val)
            name_str = name_str[:m.start(2)] + name_str[m.end():]
            continue
            
        # Check attached after closed paren: e.g. ")22)"
        m = re.search(r'\)(\d{1,2})\)', name_str)
        if m and int(m.group(1)) in FOOTNOTE_IDS:
            fn_val = int(m.group(1))
            if fn_val not in fns:
                fns.append(fn_val)
            name_str = name_str[:m.start(1)] + name_str[m.end():]
            continue
            
        # Check trailing " \d+)" at end of name IF parens before it are balanced
        m = re.search(r'\s+(\d{1,2})\)$', name_str)
        if m and int(m.group(1)) in FOOTNOTE_IDS:
            prefix = name_str[:m.start()]
            if prefix.count('(') == prefix.count(')'):
                fn_val = int(m.group(1))
                if fn_val not in fns:
                    fns.append(fn_val)
                name_str = prefix
                continue
        break

    name_str = re.sub(r'\s+', ' ', name_str).strip()
    return name_str, sorted(list(set(fns)))


def resolve_subitem_name(prev_name, cur_name):
    if cur_name.startswith('–') or cur_name.startswith('- '):
        if '–' in prev_name:
            base = prev_name.split('–')[0].strip()
            return f"{base} {cur_name.strip()}"
        elif ' - ' in prev_name:
            base = prev_name.split(' - ')[0].strip()
            return f"{base} {cur_name.strip()}"
        else:
            return f"{prev_name.strip()} {cur_name.strip()}"
    return cur_name


def determine_hgb_code_and_section(statement_type, account_type, class_num, posten_text, acc_number_str):
    num = int(acc_number_str.split('-')[0])
    p_lower = posten_text.lower()
    
    if class_num == 0:
        if num < 100:
            return "Aktiva.AusstehendeEinlagen", "Anlagevermögen", "Ausstehende Einlagen", "Aktiva"
        elif num < 200:
            return "Aktiva.A.I", "Anlagevermögen", "Immaterielle Vermögensgegenstände", "Aktiva"
        elif num < 800:
            return "Aktiva.A.II", "Anlagevermögen", "Sachanlagen", "Aktiva"
        else:
            return "Aktiva.A.III", "Anlagevermögen", "Finanzanlagen", "Aktiva"
            
    elif class_num == 1:
        if num < 1200:
            return "Aktiva.B.I", "Umlaufvermögen", "Vorräte", "Aktiva"
        elif num < 1500:
            return "Aktiva.B.II", "Umlaufvermögen", "Forderungen und sonstige Vermögensgegenstände", "Aktiva"
        elif num < 1600:
            return "Aktiva.B.III", "Umlaufvermögen", "Wertpapiere", "Aktiva"
        elif num < 1900:
            return "Aktiva.B.IV", "Umlaufvermögen", "Kassenbestand, Bundesbankguthaben, Guthaben bei Kreditinstituten und Schecks", "Aktiva"
        else:
            return "Aktiva.C", "Rechnungsabgrenzungsposten", "Aktive Rechnungsabgrenzungsposten", "Aktiva"
            
    elif class_num == 2:
        if num < 2100:
            return "Passiva.A.I", "Eigenkapital", "Gezeichnetes Kapital / Kapitaleinlagen", "Passiva"
        elif num < 2500:
            return "Passiva.A.Privat.Vollhafter", "Eigenkapital", "Privatkonten (Vollhafter / Einzelunternehmer)", "Passiva"
        elif num < 2900:
            return "Passiva.D.8.Privat.Teilhafter", "Verbindlichkeiten", "Privatkonten (Teilhafter / Kommanditisten - Fremdkapital)", "Passiva"
        elif num < 2930:
            if "sonderposten" in p_lower or num < 2920:
                return "Passiva.B", "Sonderposten mit Rücklageanteil", "Sonderposten mit Rücklageanteil", "Passiva"
            else:
                return "Passiva.A.II", "Eigenkapital", "Kapitalrücklage", "Passiva"
        elif num < 2970:
            return "Passiva.A.III", "Eigenkapital", "Gewinnrücklagen", "Passiva"
        elif num < 2980:
            return "Passiva.A.IV", "Eigenkapital", "Gewinnvortrag / Verlustvortrag", "Passiva"
        else:
            return "Passiva.A.V", "Eigenkapital", "Jahresüberschuss / Jahresfehlbetrag", "Passiva"

    elif class_num == 3:
        if num < 3100:
            if "pension" in p_lower or num < 3020:
                return "Passiva.C.1", "Rückstellungen", "Rückstellungen für Pensionen und ähnliche Verpflichtungen", "Passiva"
            elif "steuer" in p_lower or num < 3060:
                return "Passiva.C.2", "Rückstellungen", "Steuerrückstellungen", "Passiva"
            else:
                return "Passiva.C.3", "Rückstellungen", "Sonstige Rückstellungen", "Passiva"
        elif num < 3900:
            if "anleihen" in p_lower or num < 3150:
                return "Passiva.D.1", "Verbindlichkeiten", "Anleihen", "Passiva"
            elif "kreditinstitute" in p_lower or "bank" in p_lower or num < 3250:
                return "Passiva.D.2", "Verbindlichkeiten", "Verbindlichkeiten gegenüber Kreditinstituten", "Passiva"
            elif "erhaltene anzahlungen" in p_lower or num < 3300:
                return "Passiva.D.3", "Verbindlichkeiten", "Erhaltene Anzahlungen auf Bestellungen", "Passiva"
            elif "lieferungen" in p_lower or "lul" in p_lower or num < 3500:
                return "Passiva.D.4", "Verbindlichkeiten", "Verbindlichkeiten aus Lieferungen und Leistungen", "Passiva"
            elif "wechsel" in p_lower or num < 3540:
                return "Passiva.D.5", "Verbindlichkeiten", "Verbindlichkeiten aus der Annahme gezogener Wechsel", "Passiva"
            elif "verbunden" in p_lower or num < 3600:
                return "Passiva.D.6", "Verbindlichkeiten", "Verbindlichkeiten gegenüber verbundenen Unternehmen", "Passiva"
            elif "beteiligung" in p_lower or num < 3640:
                return "Passiva.D.7", "Verbindlichkeiten", "Verbindlichkeiten gegenüber Unternehmen mit Beteiligungsverhältnis", "Passiva"
            else:
                return "Passiva.D.8", "Verbindlichkeiten", "Sonstige Verbindlichkeiten (inkl. Steuern und soziale Sicherheit)", "Passiva"
        elif num < 3950:
            return "Passiva.E", "Rechnungsabgrenzungsposten", "Passive Rechnungsabgrenzungsposten", "Passiva"
        else:
            return "Passiva.F", "Passive latente Steuern", "Passive latente Steuern", "Passiva"

    elif class_num == 4:
        if num < 4600:
            return "GuV.1", "Umsatzerlöse", "Umsatzerlöse", "GuV"
        elif num < 4700:
            return "GuV.2", "Bestandsveränderungen", "Erhöhung oder Verminderung des Bestands an fertigen und unfertigen Erzeugnissen", "GuV"
        elif num < 4800:
            return "GuV.3", "Aktivierte Eigenleistungen", "Andere aktivierte Eigenleistungen", "GuV"
        else:
            return "GuV.4", "Sonstige betriebliche Erträge", "Sonstige betriebliche Erträge", "GuV"

    elif class_num == 5:
        if num < 5900:
            return "GuV.5.a", "Materialaufwand", "Aufwendungen für Roh-, Hilfs- und Betriebsstoffe und bezogene Waren", "GuV"
        else:
            return "GuV.5.b", "Materialaufwand", "Aufwendungen für bezogene Leistungen (Fremdleistungen)", "GuV"

    elif class_num == 6:
        if num < 6100:
            return "GuV.6.a", "Personalaufwand", "Löhne und Gehälter", "GuV"
        elif num < 6200:
            return "GuV.6.b", "Personalaufwand", "Soziale Abgaben und Aufwendungen für Altersversorgung und Unterstützung", "GuV"
        elif num < 6300:
            if "umlauf" in p_lower:
                return "GuV.7.b", "Abschreibungen", "Abschreibungen auf Umlaufvermögen (außergewöhnlich)", "GuV"
            else:
                return "GuV.7.a", "Abschreibungen", "Abschreibungen auf immaterielle Vermögensgegenstände und Sachanlagen", "GuV"
        elif num < 7000:
            return "GuV.8", "Sonstige betriebliche Aufwendungen", "Sonstige betriebliche Aufwendungen", "GuV"
        else:
            return "GuV.Kalkulatorisch", "Kalkulatorische Kosten", "Kalkulatorische Kosten", "GuV"

    elif class_num == 7:
        if num < 7100:
            return "GuV.9", "Finanzergebnis", "Erträge aus Beteiligungen", "GuV"
        elif num < 7200:
            if "wertpapieren" in p_lower:
                return "GuV.10", "Finanzergebnis", "Erträge aus anderen Wertpapieren und Ausleihungen des Finanzanlagevermögens", "GuV"
            else:
                return "GuV.11", "Finanzergebnis", "Sonstige Zinsen und ähnliche Erträge", "GuV"
        elif num < 7300:
            return "GuV.ErtragVerlustuebernahme", "Finanzergebnis", "Erträge aus Verlustübernahme und Gewinngemeinschaften", "GuV"
        elif num < 7400:
            return "GuV.12", "Finanzergebnis", "Abschreibungen auf Finanzanlagen und Wertpapiere des Umlaufvermögens", "GuV"
        elif num < 7500:
            return "GuV.13", "Finanzergebnis", "Zinsen und ähnliche Aufwendungen", "GuV"
        elif num < 7600:
            return "GuV.AufwandVerlustuebernahme", "Finanzergebnis", "Aufwendungen aus Verlustübernahme / abgeführte Gewinne", "GuV"
        elif num < 7700:
            return "GuV.14", "Steuern", "Steuern vom Einkommen und vom Ertrag", "GuV"
        elif num < 7800:
            return "GuV.16", "Steuern", "Sonstige Steuern", "GuV"
        else:
            return "GuV.Ergebnisverwendung", "Ergebnisverwendung", "Ergebnisübertrag / Vortrag", "GuV"

    elif class_num == 8:
        return "GuV.KuenftigeVerwendung", "Künftige Verwendung", "Künftige Verwendung bei gesetzlichen Anforderungen durch DATEV möglich", "GuV"

    elif class_num == 9:
        if num < 9100:
            return "Statistisch.Vortragskonten", "Vortragskonten", "Saldenvortragskonten", "Statistisch"
        elif num < 9200:
            return "Statistisch.BWA", "Statistische Konten", "Statistische Konten für BWA", "Statistisch"
        elif num < 9300:
            return "Statistisch.Kapitalfluss", "Statistische Konten", "Statistische Konten für Kapitalflussrechnung und Haftungsverhältnisse", "Statistisch"
        elif num < 9800:
            return "Statistisch.Kapitalkontenentwicklung", "Statistische Konten", "Statistische Konten für Kapitalkontenentwicklung PersHG", "Statistisch"
        elif num < 9900:
            return "Statistisch.GewinnzuschlagEStG", "Statistische Konten", "Statistische Konten für Gewinnzuschlag und EÜR (§ 4 Abs. 3 EStG)", "Statistisch"
        else:
            return "Statistisch.Investitionsabzug7g", "Statistische Konten", "Statistische Konten für § 7g EStG, Gewinnkorrektur und Überleitung", "Statistisch"

    return "Sonstige", "Sonstige", "Sonstige Positionen", "Sonstige"


def generate_position_id(hgb_code, posten_text, statement_type):
    st_prefix = "bilanz." if statement_type == "Bilanz" else ("guv." if statement_type == "GuV" else "statistisch.")
    hgb_clean = hgb_code.lower().replace('.', '_')
    posten_slug = slugify(posten_text)[:40].rstrip('_')
    return f"{st_prefix}{hgb_clean}.{posten_slug}".replace('..', '.')


def extract_all_accounts_from_pdf(pdf_path):
    pdf = pdfplumber.open(pdf_path)
    
    # 1. Parse all tax rules across all pages
    tax_rules = []
    for page_idx in range(36):
        p = pdf.pages[page_idx]
        words = sorted(p.extract_words(), key=lambda w: (w['top'], w['x0']))
        for i, w in enumerate(words):
            if w['text'] in ['KU', 'V', 'M']:
                if i + 1 < len(words):
                    next_w = words[i+1]
                    m = re.match(r'^(\d{4})(?:-(\d{4}))?(?:(\d{1,2})\))?$', next_w['text'])
                    if m and abs(next_w['top'] - w['top']) < 3 and next_w['x0'] > w['x0']:
                        s_acc = int(m.group(1))
                        e_acc = int(m.group(2)) if m.group(2) else s_acc
                        tax_rules.append((s_acc, e_acc, w['text']))
    tax_rules = sorted(list(set(tax_rules)), key=lambda x: x[0])
    
    all_accounts = []
    
    for page_idx in range(36):
        page_num = page_idx + 1
        is_odd = (page_num % 2 == 1)
        p = pdf.pages[page_idx]
        
        # Determine column geometry
        if is_odd:
            col_specs = [
                ('left', (75, 115, 319, 755), (75, 144.5), 144.5, 179.5, 194.5, 212.0),
                ('right', (319, 115, 565, 755), (319, 384.5), 384.5, 419.5, 434.5, 452.0)
            ]
        else:
            col_specs = [
                ('left', (50, 115, 292, 755), (50, 118.0), 118.0, 152.5, 167.5, 185.0),
                ('right', (292, 115, 540, 755), (292, 358.0), 358.0, 392.5, 407.5, 425.0)
            ]
            
        for side, bbox, (p_xmin, p_xmax), posten_x, prog_x, hf_x, num_x in col_specs:
            crop = p.crop(bbox)
            words = sorted(crop.extract_words(), key=lambda w: (w['top'], w['x0']))
            if not words:
                continue
                
            # Determine Posten bounding boxes for this column
            hlines = []
            for r in p.rects:
                if r['height'] < 2 and r['width'] > 30:
                    if p_xmin - 5 <= r['x0'] <= p_xmin + 15 and p_xmax - 15 <= r['x1'] <= p_xmax + 5:
                        hlines.append(r['top'])
            hlines = sorted(list(set(round(y, 1) for y in hlines)))
            
            posten_crop = p.crop((p_xmin, 115, p_xmax, 755))
            p_words = sorted(posten_crop.extract_words(), key=lambda w: (w['top'], w['x0']))
            
            posten_boxes = []
            for hi in range(len(hlines) - 1):
                y_start = hlines[hi]
                y_end = hlines[hi+1]
                b_words = [w for w in p_words if y_start - 2 <= w['top'] <= y_end + 2]
                if b_words:
                    b_text = clean_german_text([w['text'] for w in b_words])
                    posten_boxes.append((y_start, y_end, b_text))
                    
            # Determine Kontenklasse
            header_crop = p.crop((bbox[0], 75, bbox[2], 115))
            hw = header_crop.extract_words()
            class_num = None
            for w in hw:
                if w['text'].isdigit() and int(w['text']) in KLASSEN_INFO:
                    class_num = int(w['text'])
                    break
            if class_num is None:
                if page_num <= 2: class_num = 0
                elif page_num <= 6: class_num = 1
                elif page_num <= 10: class_num = 2
                elif page_num <= 14: class_num = 3 if page_num < 14 or side == 'left' else 4
                elif page_num <= 18: class_num = 4 if page_num < 18 or side == 'left' else 5
                elif page_num <= 21: class_num = 5 if page_num < 21 or side == 'left' else 6
                elif page_num <= 25: class_num = 6
                elif page_num <= 29: class_num = 7
                else: class_num = 9
                
            # Cluster visual lines
            lines = []
            for w in words:
                if not lines or abs(w['top'] - lines[-1][0]['top']) > 2.5:
                    lines.append([w])
                else:
                    lines[-1].append(w)
            for li in range(len(lines)):
                lines[li] = sorted(lines[li], key=lambda w: w['x0'])
                
            # Parse accounts in this column
            i = 0
            prev_acc_name = ""
            current_active_posten = ""
            
            while i < len(lines):
                line = lines[i]
                
                # Check for 4-digit number at account position
                num_word = None
                hf_word = None
                
                for w in line:
                    if prog_x - 3 <= w['x0'] <= num_x + 5:
                        if re.match(r'^\d{4}$', w['text']):
                            num_word = w
                        elif w['text'] in ['F', 'R', 'S', 'AM', 'AV', 'S/AM', 'S/AV', 'F/AM', 'F/AV']:
                            hf_word = w
                            
                is_zf_line = False
                line_text = ' '.join(w['text'] for w in line)
                if any(w['text'] in ['KU', 'V', 'M'] for w in line):
                    if re.search(r'\b(KU|V|M)\s+\d{4}', line_text):
                        is_zf_line = True
                        
                if num_word and not is_zf_line:
                    acc_num = num_word['text']
                    hf = hf_word['text'] if hf_word else None
                    acc_top = num_word['top']
                    
                    # Special check: class 8
                    if acc_num in ['8000', '8999'] or '8000-8999' in line_text:
                        class_num = 8
                    
                    name_words_first_line = [w for w in line if w['x0'] >= num_word['x1'] - 1 and w != num_word and w != hf_word]
                    prog_words_first_line = [w for w in line if posten_x - 2 < w['x0'] < prog_x + 2 and w != hf_word]
                    
                    all_name_words = list(name_words_first_line)
                    all_prog_words = list(prog_words_first_line)
                    
                    is_range = False
                    range_end = acc_num
                    
                    j = i + 1
                    while j < len(lines):
                        next_line = lines[j]
                        next_num_word = None
                        next_hf_word = None
                        for w in next_line:
                            if prog_x - 3 <= w['x0'] <= num_x + 5:
                                if re.match(r'^\d{4}$', w['text']):
                                    next_num_word = w
                                elif w['text'] in ['F', 'R', 'S', 'AM', 'AV', 'S/AM', 'S/AV']:
                                    next_hf_word = w
                                    
                        next_is_zf = False
                        if any(w['text'] in ['KU', 'V', 'M'] for w in next_line):
                            if re.search(r'\b(KU|V|M)\s+\d{4}', ' '.join(w['text'] for w in next_line)):
                                next_is_zf = True
                                
                        if (next_num_word and not next_is_zf) or (next_hf_word and not next_is_zf and any(re.match(r'^\d{4}$', w['text']) for w in next_line)):
                            break
                        if next_is_zf:
                            break
                            
                        # Check for range continuation word: e.g. -59, -89, -39, -07, -8999
                        range_word = None
                        for w in next_line:
                            if prog_x - 3 <= w['x0'] <= num_x + 10:
                                if re.match(r'^-\d{2,4}$', w['text']):
                                    range_word = w
                                    break
                                    
                        if range_word:
                            is_range = True
                            suffix = range_word['text'][1:]
                            if len(suffix) == 2:
                                range_end = acc_num[:2] + suffix
                            elif len(suffix) == 3:
                                range_end = acc_num[:1] + suffix
                            elif len(suffix) == 4:
                                range_end = suffix
                            r_name_words = [w for w in next_line if w['x0'] >= range_word['x1'] - 1 and w != range_word]
                            all_name_words.extend(r_name_words)
                        else:
                            c_name_words = [w for w in next_line if w['x0'] >= prog_x - 2]
                            all_name_words.extend(c_name_words)
                            
                        c_prog = [w for w in next_line if posten_x - 2 < w['x0'] < prog_x + 2]
                        all_prog_words.extend(c_prog)
                        j += 1
                        
                    raw_name = clean_german_text([w['text'] for w in all_name_words])
                    is_reserved = (hf == 'R')
                    
                    if not raw_name:
                        if is_reserved:
                            raw_name = "(Reserviert)"
                        elif class_num == 8:
                            raw_name = "Künftige Verwendung bei gesetzlichen Anforderungen durch DATEV möglich"
                        else:
                            raw_name = f"Konto {acc_num}"
                            
                    resolved_name = resolve_subitem_name(prev_acc_name, raw_name)
                    prev_acc_name = resolved_name
                    
                    clean_name, footnotes = clean_account_name_and_footnotes(resolved_name)
                    
                    # Determine Posten from Posten boxes
                    matching_posten = None
                    for y0, y1, ptext in posten_boxes:
                        if y0 - 3 <= acc_top <= y1 + 3:
                            matching_posten = ptext
                            break
                    if matching_posten:
                        current_active_posten = matching_posten
                        
                    active_posten = current_active_posten
                    
                    # Parse Prog and Abschlusszweck
                    full_prog = [w['text'] for w in all_prog_words]
                    az = None
                    pvs = []
                    for pt in full_prog:
                        if pt in ['HB', 'SB', 'EÜR']:
                            az = pt
                        elif pt in ['U', 'G', 'K']:
                            if pt not in pvs: pvs.append(pt)
                    if hf in ['S/AM', 'AM'] and 'U' not in pvs:
                        pvs.append('U')
                        
                    # Zusatzfunktion from tax rules
                    acc_int = int(acc_num)
                    zf = None
                    for s_acc, e_acc, tf in tax_rules:
                        if s_acc <= acc_int <= e_acc:
                            zf = tf
                            break
                            
                    st_type = "GuV" if class_num in (4, 5, 6, 7, 8) else ("Statistisch" if class_num == 9 else "Bilanz")
                    
                    acc_type = KLASSEN_INFO[class_num]['type']
                    if class_num == 2:
                        if any(w in active_posten for w in ("Fremdkapital", "Verbindlichkeiten", "Kommanditisten")):
                            acc_type = "liability"
                        else:
                            acc_type = "equity"
                    elif class_num == 7:
                        if any(w in active_posten.lower() or w in clean_name.lower() for w in ("erträge", "zinserträge", "gewinne")):
                            acc_type = "revenue"
                        elif any(w in active_posten.lower() or w in clean_name.lower() for w in ("aufwendungen", "zinsaufwendungen", "verluste", "steuern", "abschreibungen")):
                            acc_type = "expense"
                        else:
                            acc_type = "expense"
                            
                    num_str = f"{acc_num}-{range_end}" if is_range else acc_num
                    
                    hgb_code, main_group, group_name, balance_side = determine_hgb_code_and_section(
                        st_type, acc_type, class_num, active_posten, acc_num
                    )
                    
                    pos_id = generate_position_id(hgb_code, active_posten if active_posten else group_name, st_type)
                    
                    record = OrderedDict([
                        ("number", num_str),
                        ("name", clean_name),
                        ("position_id", pos_id),
                        ("kontenklasse", OrderedDict([
                            ("number", class_num),
                            ("name", KLASSEN_INFO[class_num]['name'])
                        ])),
                        ("category", main_group),
                        ("subcategory", group_name),
                        ("bilanzierung", OrderedDict([
                            ("statement_type", st_type),
                            ("balance_side", balance_side),
                            ("account_type", acc_type),
                            ("hgb_code", hgb_code),
                            ("position_id", pos_id),
                            ("posten", active_posten if active_posten else group_name)
                        ])),
                        ("steuer_funktion", OrderedDict([
                            ("programmverbindung", sorted(pvs)),
                            ("abschlusszweck", az),
                            ("hauptfunktion", hf),
                            ("hauptfunktion_description", FUNCTION_DEFINITIONS.get(hf)),
                            ("zusatzfunktion", zf),
                            ("zusatzfunktion_description", TAX_FUNCTION_DEFINITIONS.get(zf))
                        ])),
                        ("is_reserved", is_reserved),
                        ("is_range", is_range),
                        ("range_start", acc_num),
                        ("range_end", range_end),
                        ("footnotes", footnotes),
                        ("page", page_num)
                    ])
                    all_accounts.append(record)
                    i = j
                else:
                    i += 1
                    
    return all_accounts


def build_positions(accounts):
    """
    Extracts all unique Bilanzierungs-Positionen and aggregates all referenced account numbers.
    """
    positions_map = OrderedDict()
    
    for acc in accounts:
        pos_id = acc["position_id"]
        b = acc["bilanzierung"]
        
        if pos_id not in positions_map:
            positions_map[pos_id] = OrderedDict([
                ("id", pos_id),
                ("hgb_code", b["hgb_code"]),
                ("name", b["posten"]),
                ("statement_type", b["statement_type"]),
                ("balance_side", b["balance_side"]),
                ("account_type", b["account_type"]),
                ("kontenklasse", acc["kontenklasse"]),
                ("main_group", acc["category"]),
                ("group", acc["subcategory"]),
                ("account_numbers", []),
                ("accounts_count", 0)
            ])
            
        positions_map[pos_id]["account_numbers"].append(acc["number"])
        positions_map[pos_id]["accounts_count"] += 1
        
    return list(positions_map.values())


def build_hierarchy_tree(positions, accounts):
    """
    Builds a nested reporting tree (Bilanz -> Aktiva/Passiva -> Gruppen -> Positionen -> Konten).
    """
    tree = OrderedDict([
        ("bilanz", OrderedDict([
            ("aktiva", OrderedDict()),
            ("passiva", OrderedDict())
        ])),
        ("guv", OrderedDict()),
        ("statistisch", OrderedDict())
    ])
    
    for pos in positions:
        st = pos["statement_type"]
        bs = pos["balance_side"]
        mg = pos["main_group"]
        g = pos["group"]
        
        target_dict = None
        if st == "Bilanz":
            if bs == "Aktiva":
                target_dict = tree["bilanz"]["aktiva"]
            else:
                target_dict = tree["bilanz"]["passiva"]
        elif st == "GuV":
            target_dict = tree["guv"]
        else:
            target_dict = tree["statistisch"]
            
        mg_key = slugify(mg)
        if mg_key not in target_dict:
            target_dict[mg_key] = OrderedDict([
                ("name", mg),
                ("groups", OrderedDict())
            ])
            
        g_key = slugify(g)
        if g_key not in target_dict[mg_key]["groups"]:
            target_dict[mg_key]["groups"][g_key] = OrderedDict([
                ("name", g),
                ("positions", OrderedDict())
            ])
            
        pos_key = pos["id"].split('.')[-1]
        target_dict[mg_key]["groups"][g_key]["positions"][pos_key] = OrderedDict([
            ("position_id", pos["id"]),
            ("hgb_code", pos["hgb_code"]),
            ("name", pos["name"]),
            ("account_type", pos["account_type"]),
            ("account_numbers", pos["account_numbers"]),
            ("accounts_count", pos["accounts_count"])
        ])
        
    return tree


def main():
    pdf_path = "assets/DATEV-SKR04-BilrUg-2026.pdf"
    if not os.path.exists(pdf_path):
        print(f"Error: {pdf_path} not found.")
        return
        
    print(f"Parsing {pdf_path}...")
    accounts = extract_all_accounts_from_pdf(pdf_path)
    positions = build_positions(accounts)
    hierarchy = build_hierarchy_tree(positions, accounts)
    
    print(f"Extracted {len(accounts)} accounts and {len(positions)} bilanzierungs-positionen.")
    
    total_accounts = len(accounts)
    reserved_accounts = sum(1 for a in accounts if a["is_reserved"])
    active_accounts = total_accounts - reserved_accounts
    range_accounts = sum(1 for a in accounts if a["is_range"])
    
    by_class = OrderedDict()
    for k in sorted(KLASSEN_INFO.keys()):
        cnt = sum(1 for a in accounts if a["kontenklasse"]["number"] == k)
        by_class[f"Klasse {k}: {KLASSEN_INFO[k]['name']}"] = cnt
        
    by_type = OrderedDict()
    for t in ["asset", "liability", "equity", "revenue", "expense", "statistical"]:
        cnt = sum(1 for a in accounts if a["bilanzierung"]["account_type"] == t)
        by_type[t] = cnt
        
    positions_by_side = OrderedDict([
        ("Aktiva (Bilanz)", sum(1 for p in positions if p["balance_side"] == "Aktiva")),
        ("Passiva (Bilanz)", sum(1 for p in positions if p["balance_side"] == "Passiva")),
        ("GuV (Gewinn- und Verlustrechnung)", sum(1 for p in positions if p["balance_side"] == "GuV")),
        ("Statistische Positionen", sum(1 for p in positions if p["balance_side"] == "Statistisch"))
    ])
    
    result = OrderedDict([
        ("metadata", OrderedDict([
            ("title", "DATEV-Kontenrahmen nach dem Bilanzrichtlinie-Umsetzungsgesetz (BilRUG)"),
            ("subtitle", "Standardkontenrahmen - Abschlussgliederungsprinzip (SKR 04)"),
            ("version", "2026"),
            ("validity_from", "2026-01-01"),
            ("article_number", "11175"),
            ("source_file", "assets/DATEV-SKR04-BilrUg-2026.pdf"),
            ("generated_at", "2026"),
            ("description", "Vollständige strukturierte Erfassung aller Konten, Bilanzierungs-Positionen (BilRUG / HGB § 266 und § 275), Bezeichnungen, Kontenklassen, Steuerkennzeichen und Funktionen gemäß offiziellem DATEV SKR04 2026.")
        ])),
        ("legend", OrderedDict([
            ("hauptfunktionen", FUNCTION_DEFINITIONS),
            ("zusatzfunktionen", TAX_FUNCTION_DEFINITIONS),
            ("abschlusszweck", {
                "HB": "Handelsbilanz (diese Konten sollten ausschließlich für die Handelsbilanz gebucht werden)",
                "SB": "Steuerbilanz (diese Konten sollten ausschließlich für die Steuerbilanz gebucht werden)",
                "EÜR": "Einnahmen-Überschuss-Rechnung (diese Konten sollten ausschließlich für die Gewinnermittlung nach § 4 Abs. 3 EStG gebucht werden)"
            }),
            ("programmverbindung", {
                "U": "Umsatzsteuererklärung",
                "G": "Gewerbesteuer",
                "K": "Körperschaftsteuer"
            }),
            ("footnotes", FOOTNOTE_EXPLANATIONS)
        ])),
        ("statistics", OrderedDict([
            ("total_accounts", total_accounts),
            ("active_accounts", active_accounts),
            ("reserved_accounts", reserved_accounts),
            ("range_accounts", range_accounts),
            ("total_positions", len(positions)),
            ("positions_by_side", positions_by_side),
            ("accounts_by_kontenklasse", by_class),
            ("accounts_by_type", by_type)
        ])),
        ("positions", positions),
        ("accounts", accounts),
        ("hierarchy", hierarchy)
    ])
    
    out_json_path = "assets/skr04_2026.json"
    with open(out_json_path, "w", encoding="utf-8") as f:
        json.dump(result, f, ensure_ascii=False, indent=2)
        
    alt_json_path = "assets/DATEV-SKR04-BilrUg-2026.json"
    with open(alt_json_path, "w", encoding="utf-8") as f:
        json.dump(result, f, ensure_ascii=False, indent=2)
        
    print(f"Successfully saved {total_accounts} accounts and {len(positions)} positions to {out_json_path} (size: {os.path.getsize(out_json_path):,} bytes)")


if __name__ == "__main__":
    main()
