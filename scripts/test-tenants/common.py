"""Gemeinsame Bausteine: Beträge, Prüfziffern, E-Rechnung (CII, UBL), CAMT.053, Typst."""

from __future__ import annotations

import datetime as dt
import json
import os
import shutil
import subprocess
from dataclasses import dataclass, field
from decimal import ROUND_HALF_UP, Decimal as D
from xml.sax.saxutils import escape

GEN = os.path.dirname(os.path.abspath(__file__))
# Zielordner der Testdaten; gen.py setzt ihn aus seinem Argument.
BASE = os.path.abspath(os.environ.get("BUCHFINK_TESTDATEN")
                       or os.path.join(GEN, "..", "..", ".cache", "testdaten"))
BUILD = os.path.join(BASE, "_build")
# Typst liest Vorlage, Daten und Anhänge nur unterhalb seiner Wurzel.
ROOT = os.path.commonpath([GEN, BASE])
MARKER = "Testdokument für Buchfink · alle Angaben erfunden"

# ---------------------------------------------------------------- Beträge, Datum

def r2(x) -> D:
    return D(x).quantize(D("0.01"), ROUND_HALF_UP)


def de(x, cur: str = "") -> str:
    """1234.5 -> '1.234,50'"""
    v = r2(x)
    sign = "-" if v < 0 else ""
    v = abs(v)
    whole, frac = f"{v:.2f}".split(".")
    groups = []
    while len(whole) > 3:
        groups.insert(0, whole[-3:])
        whole = whole[:-3]
    groups.insert(0, whole)
    s = f"{sign}{'.'.join(groups)},{frac}"
    return f"{s} {cur}".strip() if cur else s


def en(x, cur: str = "") -> str:
    v = r2(x)
    s = f"{v:,.2f}"
    return f"{cur}{s}" if cur in ("$", "€") else (f"{s} {cur}".strip())


def qty_de(q) -> str:
    q = D(q)
    if q == q.to_integral():
        return str(int(q))
    return f"{q.normalize()}".replace(".", ",")


def ddate(iso: str) -> str:
    return dt.date.fromisoformat(iso).strftime("%d.%m.%Y")


def cdate(iso: str) -> str:
    return iso.replace("-", "")


def d(iso: str) -> dt.date:
    return dt.date.fromisoformat(iso)


# TARGET2-Feiertage und Wochenenden sind keine Buchungstage.
TARGET_HOLIDAYS = {"2025-12-25", "2025-12-26", "2026-01-01", "2026-04-03", "2026-04-06", "2026-05-01", "2026-12-25", "2026-12-26"}


def assert_bankday(iso: str):
    x = d(iso)
    if x.weekday() >= 5 or iso in TARGET_HOLIDAYS:
        raise ValueError(f"{iso} ist kein Bankarbeitstag ({x.strftime('%a')})")


# ---------------------------------------------------------------- Prüfziffern

def iban(country: str, bban: str) -> str:
    s = bban + country + "00"
    num = "".join(str(int(c, 36)) for c in s)
    return f"{country}{98 - int(num) % 97:02d}{bban}"


def fmt_iban(i: str) -> str:
    return " ".join(i[k:k + 4] for k in range(0, len(i), 4))


def de_vat(base8: str) -> str:
    product = 10
    for ch in base8:
        s = (int(ch) + product) % 10
        if s == 0:
            s = 10
        product = (2 * s) % 11
    c = 11 - product
    if c == 10:
        c = 0
    return f"DE{base8}{c}"


def dk_vat(base7: str) -> str:
    w = [2, 7, 6, 5, 4, 3, 2]
    s = sum(int(a) * b for a, b in zip(base7, w))
    last = (-s) % 11
    if last == 10:
        raise ValueError("DK: andere Basis wählen")
    return f"DK{base7}{last}"


def nl_vat(base8: str) -> str:
    s = sum(int(a) * w for a, w in zip(base8, range(9, 1, -1)))
    last = s % 11
    if last == 10:
        raise ValueError("NL: andere Basis wählen")
    return f"NL{base8}{last}B01"


def se_vat(base9: str) -> str:
    digits = [int(c) for c in base9]
    total = 0
    for i, x in enumerate(digits):
        y = x * (2 if i % 2 == 0 else 1)
        total += y - 9 if y > 9 else y
    check = (10 - total % 10) % 10
    return f"SE{base9}{check}01"


def ie_vat(base7: str) -> str:
    s = sum(int(a) * w for a, w in zip(base7, range(8, 1, -1)))
    return f"IE{base7}{'WABCDEFGHIJKLMNOPQRSTUV'[s % 23]}"


def creditor_id(national11: str) -> str:
    num = "".join(str(int(c, 36)) for c in national11 + "DE00")
    return f"DE{98 - int(num) % 97:02d}ZZZ{national11}"


def leitweg(base: str) -> str:
    raw = base.replace("-", "") + "00"
    num = "".join(str(int(c, 36)) for c in raw.upper())
    return f"{base}-{98 - int(num) % 97:02d}"


# ---------------------------------------------------------------- Rechnung

@dataclass
class Party:
    name: str
    street: str
    zip: str
    city: str
    country: str = "DE"
    country_name: str = ""
    vat_id: str = ""
    tax_no: str = ""
    email: str = ""
    phone: str = ""
    contact: str = ""
    iban: str = ""
    bic: str = ""
    bank: str = ""
    register: str = ""
    gf: str = ""
    web: str = ""

    def address_lines(self, extra_top: list[str] | None = None) -> list[str]:
        out = list(extra_top or [])
        out += [self.name, self.street, f"{self.zip} {self.city}"]
        if self.country != "DE":
            out.append(self.country_name or self.country)
        return out

    def one_line(self) -> str:
        return f"{self.name} · {self.street} · {self.zip} {self.city}"


@dataclass
class Line:
    text: str
    qty: D
    price: D
    rate: D = D("19")
    cat: str = "S"
    unit: str = "C62"
    unit_label: str = "Stk."
    detail: str = ""

    @property
    def net(self) -> D:
        return r2(D(self.qty) * D(self.price))


EXEMPTION = {
    "AE": ("Steuerschuldnerschaft des Leistungsempfängers", "VATEX-EU-AE"),
    "K": ("Innergemeinschaftliche Lieferung, steuerfrei", "VATEX-EU-IC"),
    "E": ("Steuerbefreit", "VATEX-EU-132"),
    "O": ("Nicht steuerbar", "VATEX-EU-O"),
}


@dataclass
class Invoice:
    number: str
    date: str
    seller: Party
    buyer: Party
    lines: list[Line]
    type_code: str = "380"
    currency: str = "EUR"
    delivery: str = ""
    period: tuple[str, str] | None = None
    due: str = ""
    means: dict = field(default_factory=dict)
    terms: str = ""
    skonto: tuple[int, D] | None = None
    prepaid: D = D("0")
    buyer_ref: str = ""
    order_ref: str = ""
    customer_no: str = ""
    preceding: tuple[str, str] | None = None
    notes: list[str] = field(default_factory=list)
    guideline: str = "en16931"

    def groups(self):
        g: dict[tuple[str, D], D] = {}
        for l in self.lines:
            k = (l.cat, D(l.rate))
            g[k] = g.get(k, D("0")) + l.net
        out = []
        for (cat, rate), basis in sorted(g.items(), key=lambda kv: (-kv[0][1], kv[0][0])):
            tax = r2(basis * rate / 100) if cat == "S" else D("0.00")
            out.append((cat, rate, r2(basis), tax))
        return out

    @property
    def net(self) -> D:
        return r2(sum((l.net for l in self.lines), D("0")))

    @property
    def tax(self) -> D:
        return r2(sum((g[3] for g in self.groups()), D("0")))

    @property
    def gross(self) -> D:
        return self.net + self.tax

    @property
    def payable(self) -> D:
        return self.gross - self.prepaid

    def terms_note(self) -> str:
        parts = []
        if self.skonto:
            days, pct = self.skonto
            parts.append(f"#SKONTO#TAGE={days}#PROZENT={D(pct):.2f}#")
        if self.terms:
            parts.append(self.terms)
        return "\n".join(parts)


def _dt(tag: str, iso: str) -> str:
    return f'<ram:{tag}><udt:DateTimeString format="102">{cdate(iso)}</udt:DateTimeString></ram:{tag}>'


def _party_cii(tag: str, p: Party, with_contact: bool, electronic: bool) -> str:
    x = [f"<ram:{tag}>", f"<ram:Name>{escape(p.name)}</ram:Name>"]
    if p.register:
        x.append(f"<ram:SpecifiedLegalOrganization><ram:ID>{escape(p.register)}</ram:ID></ram:SpecifiedLegalOrganization>")
    if with_contact and (p.contact or p.phone or p.email):
        x.append("<ram:DefinedTradeContact>")
        if p.contact:
            x.append(f"<ram:PersonName>{escape(p.contact)}</ram:PersonName>")
        if p.phone:
            x.append(f"<ram:TelephoneUniversalCommunication><ram:CompleteNumber>{escape(p.phone)}</ram:CompleteNumber></ram:TelephoneUniversalCommunication>")
        if p.email:
            x.append(f"<ram:EmailURIUniversalCommunication><ram:URIID>{escape(p.email)}</ram:URIID></ram:EmailURIUniversalCommunication>")
        x.append("</ram:DefinedTradeContact>")
    x.append("<ram:PostalTradeAddress>"
             f"<ram:PostcodeCode>{escape(p.zip)}</ram:PostcodeCode>"
             f"<ram:LineOne>{escape(p.street)}</ram:LineOne>"
             f"<ram:CityName>{escape(p.city)}</ram:CityName>"
             f"<ram:CountryID>{p.country}</ram:CountryID></ram:PostalTradeAddress>")
    if electronic and p.email:
        x.append(f'<ram:URIUniversalCommunication><ram:URIID schemeID="EM">{escape(p.email)}</ram:URIID></ram:URIUniversalCommunication>')
    if p.vat_id:
        x.append(f'<ram:SpecifiedTaxRegistration><ram:ID schemeID="VA">{p.vat_id}</ram:ID></ram:SpecifiedTaxRegistration>')
    if p.tax_no:
        x.append(f'<ram:SpecifiedTaxRegistration><ram:ID schemeID="FC">{escape(p.tax_no)}</ram:ID></ram:SpecifiedTaxRegistration>')
    x.append(f"</ram:{tag}>")
    return "".join(x)


def _num(v: D) -> str:
    return f"{r2(v):.2f}"


def _rate(v: D) -> str:
    v = D(v)
    return f"{v:.2f}"


def cii(inv: Invoice) -> str:
    xr = inv.guideline == "xrechnung"
    guideline = "urn:cen.eu:en16931:2017#compliant#urn:xeinkauf.de:kosit:xrechnung_3.0" if xr else "urn:cen.eu:en16931:2017"
    out = ['<?xml version="1.0" encoding="UTF-8"?>',
           '<rsm:CrossIndustryInvoice xmlns:rsm="urn:un:unece:uncefact:data:standard:CrossIndustryInvoice:100" '
           'xmlns:ram="urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100" '
           'xmlns:qdt="urn:un:unece:uncefact:data:standard:QualifiedDataType:100" '
           'xmlns:udt="urn:un:unece:uncefact:data:standard:UnqualifiedDataType:100">',
           "<rsm:ExchangedDocumentContext>"]
    if xr:
        out.append("<ram:BusinessProcessSpecifiedDocumentContextParameter><ram:ID>urn:fdc:peppol.eu:2017:poacc:billing:01.0</ram:ID></ram:BusinessProcessSpecifiedDocumentContextParameter>")
    out.append(f"<ram:GuidelineSpecifiedDocumentContextParameter><ram:ID>{guideline}</ram:ID></ram:GuidelineSpecifiedDocumentContextParameter>")
    out.append("</rsm:ExchangedDocumentContext>")
    out.append("<rsm:ExchangedDocument>"
               f"<ram:ID>{escape(inv.number)}</ram:ID><ram:TypeCode>{inv.type_code}</ram:TypeCode>"
               + _dt("IssueDateTime", inv.date))
    for n in inv.notes:
        out.append(f"<ram:IncludedNote><ram:Content>{escape(n)}</ram:Content></ram:IncludedNote>")
    seller_reg = inv.seller.register
    if seller_reg:
        out.append(f"<ram:IncludedNote><ram:Content>{escape(inv.seller.name)}, {escape(seller_reg)}, Geschäftsführung: {escape(inv.seller.gf)}</ram:Content><ram:SubjectCode>REG</ram:SubjectCode></ram:IncludedNote>")
    out.append("</rsm:ExchangedDocument><rsm:SupplyChainTradeTransaction>")
    for i, l in enumerate(inv.lines, 1):
        out.append("<ram:IncludedSupplyChainTradeLineItem>"
                   f"<ram:AssociatedDocumentLineDocument><ram:LineID>{i}</ram:LineID></ram:AssociatedDocumentLineDocument>"
                   f"<ram:SpecifiedTradeProduct><ram:Name>{escape(l.text)}</ram:Name>"
                   + (f"<ram:Description>{escape(l.detail)}</ram:Description>" if l.detail else "")
                   + "</ram:SpecifiedTradeProduct>"
                   f"<ram:SpecifiedLineTradeAgreement><ram:NetPriceProductTradePrice><ram:ChargeAmount>{D(l.price):.2f}</ram:ChargeAmount></ram:NetPriceProductTradePrice></ram:SpecifiedLineTradeAgreement>"
                   f'<ram:SpecifiedLineTradeDelivery><ram:BilledQuantity unitCode="{l.unit}">{D(l.qty)}</ram:BilledQuantity></ram:SpecifiedLineTradeDelivery>'
                   "<ram:SpecifiedLineTradeSettlement><ram:ApplicableTradeTax><ram:TypeCode>VAT</ram:TypeCode>"
                   f"<ram:CategoryCode>{l.cat}</ram:CategoryCode><ram:RateApplicablePercent>{_rate(l.rate)}</ram:RateApplicablePercent></ram:ApplicableTradeTax>"
                   f"<ram:SpecifiedTradeSettlementLineMonetarySummation><ram:LineTotalAmount>{_num(l.net)}</ram:LineTotalAmount></ram:SpecifiedTradeSettlementLineMonetarySummation>"
                   "</ram:SpecifiedLineTradeSettlement></ram:IncludedSupplyChainTradeLineItem>")
    out.append("<ram:ApplicableHeaderTradeAgreement>")
    if inv.buyer_ref:
        out.append(f"<ram:BuyerReference>{escape(inv.buyer_ref)}</ram:BuyerReference>")
    out.append(_party_cii("SellerTradeParty", inv.seller, True, xr))
    out.append(_party_cii("BuyerTradeParty", inv.buyer, False, xr))
    if inv.order_ref:
        out.append(f"<ram:BuyerOrderReferencedDocument><ram:IssuerAssignedID>{escape(inv.order_ref)}</ram:IssuerAssignedID></ram:BuyerOrderReferencedDocument>")
    out.append("</ram:ApplicableHeaderTradeAgreement><ram:ApplicableHeaderTradeDelivery>")
    if inv.delivery:
        out.append("<ram:ActualDeliverySupplyChainEvent>" + _dt("OccurrenceDateTime", inv.delivery) + "</ram:ActualDeliverySupplyChainEvent>")
    out.append("</ram:ApplicableHeaderTradeDelivery><ram:ApplicableHeaderTradeSettlement>")
    m = inv.means
    if m.get("creditor_id"):
        out.append(f"<ram:CreditorReferenceID>{m['creditor_id']}</ram:CreditorReferenceID>")
    out.append(f"<ram:PaymentReference>{escape(inv.number)}</ram:PaymentReference>")
    out.append(f"<ram:InvoiceCurrencyCode>{inv.currency}</ram:InvoiceCurrencyCode>")
    if m:
        out.append(f"<ram:SpecifiedTradeSettlementPaymentMeans><ram:TypeCode>{m['code']}</ram:TypeCode>")
        if m.get("card"):
            out.append(f"<ram:ApplicableTradeSettlementFinancialCard><ram:ID>{m['card']}</ram:ID><ram:CardholderName>{escape(m.get('holder', ''))}</ram:CardholderName></ram:ApplicableTradeSettlementFinancialCard>")
        if m.get("debtor_iban"):
            out.append(f"<ram:PayerPartyDebtorFinancialAccount><ram:IBANID>{m['debtor_iban']}</ram:IBANID></ram:PayerPartyDebtorFinancialAccount>")
        if m.get("iban"):
            out.append(f"<ram:PayeePartyCreditorFinancialAccount><ram:IBANID>{m['iban']}</ram:IBANID>"
                       + (f"<ram:AccountName>{escape(m['account_name'])}</ram:AccountName>" if m.get("account_name") else "")
                       + "</ram:PayeePartyCreditorFinancialAccount>")
            if m.get("bic"):
                out.append(f"<ram:PayeeSpecifiedCreditorFinancialInstitution><ram:BICID>{m['bic']}</ram:BICID></ram:PayeeSpecifiedCreditorFinancialInstitution>")
        out.append("</ram:SpecifiedTradeSettlementPaymentMeans>")
    for cat, rate, basis, tax in inv.groups():
        out.append(f"<ram:ApplicableTradeTax><ram:CalculatedAmount>{_num(tax)}</ram:CalculatedAmount><ram:TypeCode>VAT</ram:TypeCode>")
        if cat in EXEMPTION:
            out.append(f"<ram:ExemptionReason>{EXEMPTION[cat][0]}</ram:ExemptionReason>")
        out.append(f"<ram:BasisAmount>{_num(basis)}</ram:BasisAmount><ram:CategoryCode>{cat}</ram:CategoryCode>")
        if cat in EXEMPTION:
            out.append(f"<ram:ExemptionReasonCode>{EXEMPTION[cat][1]}</ram:ExemptionReasonCode>")
        out.append(f"<ram:RateApplicablePercent>{_rate(rate)}</ram:RateApplicablePercent></ram:ApplicableTradeTax>")
    if inv.period:
        out.append("<ram:BillingSpecifiedPeriod>" + _dt("StartDateTime", inv.period[0]) + _dt("EndDateTime", inv.period[1]) + "</ram:BillingSpecifiedPeriod>")
    note = inv.terms_note()
    if note or inv.due or m.get("mandate"):
        out.append("<ram:SpecifiedTradePaymentTerms>")
        if note:
            out.append(f"<ram:Description>{escape(note)}\n</ram:Description>" if inv.skonto else f"<ram:Description>{escape(note)}</ram:Description>")
        if inv.due:
            out.append(_dt("DueDateDateTime", inv.due))
        if m.get("mandate"):
            out.append(f"<ram:DirectDebitMandateID>{escape(m['mandate'])}</ram:DirectDebitMandateID>")
        out.append("</ram:SpecifiedTradePaymentTerms>")
    out.append("<ram:SpecifiedTradeSettlementHeaderMonetarySummation>"
               f"<ram:LineTotalAmount>{_num(inv.net)}</ram:LineTotalAmount>"
               f"<ram:TaxBasisTotalAmount>{_num(inv.net)}</ram:TaxBasisTotalAmount>"
               f'<ram:TaxTotalAmount currencyID="{inv.currency}">{_num(inv.tax)}</ram:TaxTotalAmount>'
               f"<ram:GrandTotalAmount>{_num(inv.gross)}</ram:GrandTotalAmount>"
               + (f"<ram:TotalPrepaidAmount>{_num(inv.prepaid)}</ram:TotalPrepaidAmount>" if inv.prepaid else "")
               + f"<ram:DuePayableAmount>{_num(inv.payable)}</ram:DuePayableAmount>"
               "</ram:SpecifiedTradeSettlementHeaderMonetarySummation>")
    if inv.preceding:
        out.append("<ram:InvoiceReferencedDocument>"
                   f"<ram:IssuerAssignedID>{escape(inv.preceding[0])}</ram:IssuerAssignedID>"
                   '<ram:FormattedIssueDateTime><qdt:DateTimeString format="102">' + cdate(inv.preceding[1]) + "</qdt:DateTimeString></ram:FormattedIssueDateTime>"
                   "</ram:InvoiceReferencedDocument>")
    out.append("</ram:ApplicableHeaderTradeSettlement></rsm:SupplyChainTradeTransaction></rsm:CrossIndustryInvoice>")
    return "\n".join(out) + "\n"


def _party_ubl(tag: str, p: Party, contact: bool) -> str:
    x = [f"<cac:{tag}><cac:Party>"]
    if p.email:
        x.append(f'<cbc:EndpointID schemeID="EM">{escape(p.email)}</cbc:EndpointID>')
    x.append(f"<cac:PostalAddress><cbc:StreetName>{escape(p.street)}</cbc:StreetName><cbc:CityName>{escape(p.city)}</cbc:CityName>"
             f"<cbc:PostalZone>{escape(p.zip)}</cbc:PostalZone><cac:Country><cbc:IdentificationCode>{p.country}</cbc:IdentificationCode></cac:Country></cac:PostalAddress>")
    if p.vat_id:
        x.append(f"<cac:PartyTaxScheme><cbc:CompanyID>{p.vat_id}</cbc:CompanyID><cac:TaxScheme><cbc:ID>VAT</cbc:ID></cac:TaxScheme></cac:PartyTaxScheme>")
    x.append(f"<cac:PartyLegalEntity><cbc:RegistrationName>{escape(p.name)}</cbc:RegistrationName></cac:PartyLegalEntity>")
    if contact and (p.contact or p.phone or p.email):
        x.append("<cac:Contact>"
                 + (f"<cbc:Name>{escape(p.contact)}</cbc:Name>" if p.contact else "")
                 + (f"<cbc:Telephone>{escape(p.phone)}</cbc:Telephone>" if p.phone else "")
                 + (f"<cbc:ElectronicMail>{escape(p.email)}</cbc:ElectronicMail>" if p.email else "")
                 + "</cac:Contact>")
    x.append(f"</cac:Party></cac:{tag}>")
    return "".join(x)


def ubl(inv: Invoice) -> str:
    cur = inv.currency
    out = ['<?xml version="1.0" encoding="UTF-8"?>',
           '<Invoice xmlns="urn:oasis:names:specification:ubl:schema:xsd:Invoice-2" '
           'xmlns:cac="urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2" '
           'xmlns:cbc="urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2">',
           "<cbc:CustomizationID>urn:cen.eu:en16931:2017#compliant#urn:fdc:peppol.eu:2017:poacc:billing:3.0</cbc:CustomizationID>",
           "<cbc:ProfileID>urn:fdc:peppol.eu:2017:poacc:billing:01.0</cbc:ProfileID>",
           f"<cbc:ID>{escape(inv.number)}</cbc:ID>",
           f"<cbc:IssueDate>{inv.date}</cbc:IssueDate>"]
    if inv.due:
        out.append(f"<cbc:DueDate>{inv.due}</cbc:DueDate>")
    out.append(f"<cbc:InvoiceTypeCode>{inv.type_code}</cbc:InvoiceTypeCode>")
    for n in inv.notes:
        out.append(f"<cbc:Note>{escape(n)}</cbc:Note>")
    out.append(f"<cbc:DocumentCurrencyCode>{cur}</cbc:DocumentCurrencyCode>")
    if inv.buyer_ref:
        out.append(f"<cbc:BuyerReference>{escape(inv.buyer_ref)}</cbc:BuyerReference>")
    if inv.period:
        out.append(f"<cac:InvoicePeriod><cbc:StartDate>{inv.period[0]}</cbc:StartDate><cbc:EndDate>{inv.period[1]}</cbc:EndDate></cac:InvoicePeriod>")
    if inv.order_ref:
        out.append(f"<cac:OrderReference><cbc:ID>{escape(inv.order_ref)}</cbc:ID></cac:OrderReference>")
    out.append(_party_ubl("AccountingSupplierParty", inv.seller, True))
    out.append(_party_ubl("AccountingCustomerParty", inv.buyer, False))
    if inv.delivery:
        out.append(f"<cac:Delivery><cbc:ActualDeliveryDate>{inv.delivery}</cbc:ActualDeliveryDate></cac:Delivery>")
    m = inv.means
    if m:
        out.append(f"<cac:PaymentMeans><cbc:PaymentMeansCode>{m['code']}</cbc:PaymentMeansCode><cbc:PaymentID>{escape(inv.number)}</cbc:PaymentID>")
        if m.get("iban"):
            out.append(f"<cac:PayeeFinancialAccount><cbc:ID>{m['iban']}</cbc:ID>"
                       + (f"<cbc:Name>{escape(m['account_name'])}</cbc:Name>" if m.get("account_name") else "")
                       + (f"<cac:FinancialInstitutionBranch><cbc:ID>{m['bic']}</cbc:ID></cac:FinancialInstitutionBranch>" if m.get("bic") else "")
                       + "</cac:PayeeFinancialAccount>")
        out.append("</cac:PaymentMeans>")
    if inv.terms:
        out.append(f"<cac:PaymentTerms><cbc:Note>{escape(inv.terms)}</cbc:Note></cac:PaymentTerms>")
    out.append(f'<cac:TaxTotal><cbc:TaxAmount currencyID="{cur}">{_num(inv.tax)}</cbc:TaxAmount>')
    for cat, rate, basis, tax in inv.groups():
        out.append(f'<cac:TaxSubtotal><cbc:TaxableAmount currencyID="{cur}">{_num(basis)}</cbc:TaxableAmount>'
                   f'<cbc:TaxAmount currencyID="{cur}">{_num(tax)}</cbc:TaxAmount><cac:TaxCategory><cbc:ID>{cat}</cbc:ID>'
                   f"<cbc:Percent>{_rate(rate)}</cbc:Percent>"
                   + (f"<cbc:TaxExemptionReasonCode>{EXEMPTION[cat][1]}</cbc:TaxExemptionReasonCode><cbc:TaxExemptionReason>{EXEMPTION[cat][0]}</cbc:TaxExemptionReason>" if cat in EXEMPTION else "")
                   + "<cac:TaxScheme><cbc:ID>VAT</cbc:ID></cac:TaxScheme></cac:TaxCategory></cac:TaxSubtotal>")
    out.append("</cac:TaxTotal>")
    out.append("<cac:LegalMonetaryTotal>"
               f'<cbc:LineExtensionAmount currencyID="{cur}">{_num(inv.net)}</cbc:LineExtensionAmount>'
               f'<cbc:TaxExclusiveAmount currencyID="{cur}">{_num(inv.net)}</cbc:TaxExclusiveAmount>'
               f'<cbc:TaxInclusiveAmount currencyID="{cur}">{_num(inv.gross)}</cbc:TaxInclusiveAmount>'
               f'<cbc:PayableAmount currencyID="{cur}">{_num(inv.payable)}</cbc:PayableAmount>'
               "</cac:LegalMonetaryTotal>")
    for i, l in enumerate(inv.lines, 1):
        out.append(f"<cac:InvoiceLine><cbc:ID>{i}</cbc:ID>"
                   f'<cbc:InvoicedQuantity unitCode="{l.unit}">{D(l.qty)}</cbc:InvoicedQuantity>'
                   f'<cbc:LineExtensionAmount currencyID="{cur}">{_num(l.net)}</cbc:LineExtensionAmount>'
                   f"<cac:Item><cbc:Name>{escape(l.text)}</cbc:Name><cac:ClassifiedTaxCategory><cbc:ID>{l.cat}</cbc:ID>"
                   f"<cbc:Percent>{_rate(l.rate)}</cbc:Percent><cac:TaxScheme><cbc:ID>VAT</cbc:ID></cac:TaxScheme></cac:ClassifiedTaxCategory></cac:Item>"
                   f'<cac:Price><cbc:PriceAmount currencyID="{cur}">{D(l.price):.2f}</cbc:PriceAmount></cac:Price></cac:InvoiceLine>')
    out.append("</Invoice>")
    return "\n".join(out) + "\n"


# ---------------------------------------------------------------- Rechnungs-PDF

LABELS = {
    "de": dict(no="Rechnungsnummer", date="Rechnungsdatum", cust="Kundennummer", deliv="Leistungsdatum", period="Leistungszeitraum",
               pos="Pos.", desc="Beschreibung", qty="Menge", unit="Einheit", price="Einzelpreis", vat="USt", total="Gesamt",
               net="Nettobetrag", vat_on="zzgl. {r} % USt auf {b}", gross="Rechnungsbetrag", prepaid="bereits bezahlt", payable="Zahlbetrag",
               title="Rechnung", vatid="Ihre USt-IdNr.", order="Ihre Bestellung"),
    "en": dict(no="Invoice number", date="Invoice date", cust="Customer ID", deliv="Date of supply", period="Service period",
               pos="#", desc="Description", qty="Qty", unit="Unit", price="Unit price", vat="VAT", total="Amount",
               net="Subtotal", vat_on="VAT {r} % on {b}", gross="Total", prepaid="Paid", payable="Amount due",
               title="Invoice", vatid="Your VAT ID", order="Your order"),
}


def money_fmt(lang: str, cur: str):
    if lang == "en":
        sym = {"EUR": "€", "USD": "$"}.get(cur, cur + " ")
        return lambda v: (f"-{sym}{abs(r2(v)):,.2f}" if r2(v) < 0 else f"{sym}{r2(v):,.2f}")
    return lambda v: de(v, "€" if cur == "EUR" else cur)


def invoice_pages(inv: Invoice, *, lang="de", font="Helvetica", accent="#1f4e79", tagline="", title=None,
                  intro="", extra_notes=None, stamp=None, footer_extra=None, show_vat_col=True, pay_lines=None) -> list[dict]:
    L = LABELS[lang]
    fm = money_fmt(lang, inv.currency)
    s, b = inv.seller, inv.buyer
    right = [s.street, f"{s.zip} {s.city}"] + ([s.country_name] if s.country != "DE" else []) + [x for x in [s.phone and ("Tel. " if lang == "de" else "Phone ") + s.phone, s.email, s.web] if x]
    meta = [[L["no"], inv.number], [L["date"], ddate(inv.date) if lang == "de" else d(inv.date).strftime("%d %b %Y")]]
    if inv.customer_no:
        meta.append([L["cust"], inv.customer_no])
    if inv.order_ref:
        meta.append([L["order"], inv.order_ref])
    if inv.period:
        p0, p1 = inv.period
        meta.append([L["period"], f"{ddate(p0)} – {ddate(p1)}" if lang == "de" else f"{d(p0).strftime('%d %b %Y')} – {d(p1).strftime('%d %b %Y')}"])
    elif inv.delivery:
        meta.append([L["deliv"], ddate(inv.delivery) if lang == "de" else d(inv.delivery).strftime("%d %b %Y")])
    if b.vat_id:
        meta.append([L["vatid"], b.vat_id])
    header = [L["pos"], L["desc"], L["qty"], L["unit"], L["price"]] + ([L["vat"]] if show_vat_col else []) + [L["total"]]
    rows = []
    for i, l in enumerate(inv.lines, 1):
        desc = [{"t": l.text, "b": True}] + ([{"t": "\n" + l.detail}] if l.detail else [])
        row = [str(i), desc, qty_de(l.qty) if lang == "de" else str(D(l.qty).normalize()), l.unit_label, fm(l.price)]
        if show_vat_col:
            row.append(f"{D(l.rate).normalize()} %" if l.cat == "S" else ("0 %" if lang == "de" else "0 %"))
        row.append(fm(l.net))
        rows.append(row)
    widths = [10, "1fr", 14, 16, 24] + ([12] if show_vat_col else []) + [26]
    align = ["l", "l", "r", "l", "r"] + (["r"] if show_vat_col else []) + ["r"]
    tot = [[L["net"], fm(inv.net)]]
    for cat, rate, basis, tax in inv.groups():
        if cat == "S":
            tot.append([L["vat_on"].format(r=D(rate).normalize(), b=fm(basis)), fm(tax)])
    tot.append([L["gross"], fm(inv.gross), True])
    if inv.prepaid:
        tot.append([L["prepaid"], fm(-inv.prepaid)])
        tot.append([L["payable"], fm(inv.payable), True])
    blocks = [
        {"type": "letterhead", "name": s.name, "tagline": tagline, "right": right},
        {"type": "address", "sender_line": s.one_line(), "lines": b.address_lines(), "meta": meta},
        {"type": "title", "text": title or L["title"]},
    ]
    if intro:
        blocks.append({"type": "para", "text": intro})
    blocks += [{"type": "table", "widths": widths, "align": align, "header": header, "rows": rows},
               {"type": "totals", "rows": tot}]
    for n in (extra_notes or []):
        blocks.append({"type": "para", "text": n})
    if pay_lines:
        blocks.append({"type": "note", "lines": pay_lines})
    foot = [[s.name, s.street, f"{s.zip} {s.city}"] + ([s.country_name] if s.country != "DE" else [])]
    if s.iban:
        foot.append([s.bank or "Bank", f"IBAN {fmt_iban(s.iban)}", f"BIC {s.bic}" if s.bic else ""])
    ids = []
    if s.gf:
        ids.append(("Geschäftsführung: " if lang == "de" else "Directors: ") + s.gf)
    if s.register:
        ids.append(s.register)
    if s.tax_no:
        ids.append(("Steuernummer " if lang == "de" else "Tax no. ") + s.tax_no)
    if s.vat_id:
        ids.append(("USt-IdNr. " if lang == "de" else "VAT ID ") + s.vat_id)
    foot.append(ids)
    if footer_extra:
        foot.append(footer_extra)
    page = {"kind": "a4", "font": font, "accent": accent, "lang": lang, "blocks": blocks, "footer": foot}
    if stamp:
        page["stamp"] = stamp
    return [page]


# ---------------------------------------------------------------- Typst

def compile_doc(out_path: str, title: str, author: str, pages: list[dict], attach: list[tuple[str, str]] | None = None,
                fmt: str = "pdf", ppi: int = 200):
    """attach: [(Name im PDF, XML-Inhalt)]"""
    os.makedirs(BUILD, exist_ok=True)
    stem = os.path.splitext(os.path.basename(out_path))[0]
    tag = f"{abs(hash(out_path)) % 10**10}_{stem}"
    atts = []
    for name, content in attach or []:
        p = os.path.join(BUILD, f"{tag}_{name}")
        with open(p, "w", encoding="utf-8") as f:
            f.write(content)
        atts.append({"name": name, "path": "/" + os.path.relpath(p, ROOT), "rel": "alternative", "mime": "text/xml",
                     "desc": "Rechnungsdaten im ZUGFeRD-Format (EN 16931)"})
    data = {"title": title, "author": author, "marker": MARKER, "pages": pages, "attach": atts}
    jp = os.path.join(BUILD, f"{tag}.json")
    with open(jp, "w", encoding="utf-8") as f:
        json.dump(data, f, ensure_ascii=False, indent=1)
    os.makedirs(os.path.dirname(out_path), exist_ok=True)
    cmd = ["typst", "compile", "--root", ROOT, "--input", "data=/" + os.path.relpath(jp, ROOT)]
    if fmt == "pdf":
        if atts:
            cmd += ["--pdf-standard", "a-3b"]
        cmd += [os.path.join(GEN, "main.typ"), out_path]
        subprocess.run(cmd, check=True)
    else:
        png = os.path.join(BUILD, f"{tag}.png")
        cmd += ["--format", "png", "--ppi", str(ppi), os.path.join(GEN, "main.typ"), png]
        subprocess.run(cmd, check=True)
        if fmt == "jpg":
            to_jpeg(png, out_path)
        else:
            shutil.copy(png, out_path)


def to_jpeg(png: str, out_path: str):
    if shutil.which("sips"):
        cmd = ["sips", "-s", "format", "jpeg", "-s", "formatOptions", "82", png, "--out", out_path]
    elif shutil.which("magick"):
        cmd = ["magick", png, "-quality", "82", out_path]
    else:
        raise SystemExit("Für das Foto eines Kassenbons braucht der Generator sips (macOS) oder ImageMagick")
    subprocess.run(cmd, check=True, stdout=subprocess.DEVNULL)


# ---------------------------------------------------------------- CAMT.053

@dataclass
class Tx:
    date: str
    amount: D           # positiv = Gutschrift, negativ = Lastschrift
    name: str
    iban: str
    text: str
    kind: str = ""      # Überweisung, Lastschrift, Gutschrift, Entgelt, Zinsen
    mandate: str = ""
    creditor_id: str = ""
    e2e: str = "NOTPROVIDED"
    value_date: str = ""


@dataclass
class Statement:
    owner: str
    iban: str
    bic: str
    bank: str
    seq: int
    start: str
    end: str
    opening: D
    txs: list[Tx]

    @property
    def closing(self) -> D:
        return r2(self.opening + sum((t.amount for t in self.txs), D("0")))


BTC = {
    "Gutschrift": ("PMNT", "RCDT", "ESCT"),
    "Überweisung": ("PMNT", "ICDT", "ESCT"),
    "Lastschrift": ("PMNT", "IDDT", "ESDD"),
    "Kartenabrechnung": ("PMNT", "IDDT", "ESDD"),
    "Entgelt": ("ACMT", "MDOP", "CHRG"),
    "Zinsen": ("ACMT", "MDOP", "INTR"),
    "Lohn": ("PMNT", "ICDT", "SALA"),
}


def _bal(code: str, amount: D, date: str) -> str:
    ind = "CRDT" if amount >= 0 else "DBIT"
    return (f"<Bal><Tp><CdOrPrtry><Cd>{code}</Cd></CdOrPrtry></Tp><Amt Ccy=\"EUR\">{abs(r2(amount)):.2f}</Amt>"
            f"<CdtDbtInd>{ind}</CdtDbtInd><Dt><Dt>{date}</Dt></Dt></Bal>")


def camt(st: Statement) -> str:
    for t in st.txs:
        assert_bankday(t.date)
        if not (st.start <= t.date <= st.end):
            raise ValueError(f"{t.date} liegt außerhalb des Auszugs {st.start}–{st.end}")
    txs = sorted(st.txs, key=lambda t: t.date)
    msg = f"STMT{st.iban[-6:]}{cdate(st.end)}{st.seq:03d}"
    prev = (d(st.start) - dt.timedelta(days=1)).isoformat()
    out = ['<?xml version="1.0" encoding="UTF-8"?>',
           '<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.08" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">',
           "<BkToCstmrStmt>",
           f"<GrpHdr><MsgId>{msg}</MsgId><CreDtTm>{st.end}T22:15:00</CreDtTm></GrpHdr>",
           f"<Stmt><Id>{msg}</Id><ElctrncSeqNb>{st.seq}</ElctrncSeqNb><LglSeqNb>{st.seq}</LglSeqNb><CreDtTm>{st.end}T22:15:00</CreDtTm>",
           f"<FrToDt><FrDtTm>{st.start}T00:00:00</FrDtTm><ToDtTm>{st.end}T23:59:59</ToDtTm></FrToDt>",
           f"<Acct><Id><IBAN>{st.iban}</IBAN></Id><Ccy>EUR</Ccy><Ownr><Nm>{escape(st.owner)}</Nm></Ownr>"
           f"<Svcr><FinInstnId><BICFI>{st.bic}</BICFI><Nm>{escape(st.bank)}</Nm></FinInstnId></Svcr></Acct>",
           _bal("PRCD", st.opening, prev),
           _bal("CLBD", st.closing, st.end)]
    for i, t in enumerate(txs, 1):
        amt = abs(r2(t.amount))
        ind = "CRDT" if t.amount > 0 else "DBIT"
        dom, fam, sub = BTC.get(t.kind or ("Gutschrift" if t.amount > 0 else "Überweisung"))
        ref = f"{cdate(t.date)}{st.seq:02d}{i:04d}"
        x = [f"<Ntry><NtryRef>{ref}</NtryRef><Amt Ccy=\"EUR\">{amt:.2f}</Amt><CdtDbtInd>{ind}</CdtDbtInd><Sts><Cd>BOOK</Cd></Sts>",
             f"<BookgDt><Dt>{t.date}</Dt></BookgDt><ValDt><Dt>{t.value_date or t.date}</Dt></ValDt><AcctSvcrRef>{ref}</AcctSvcrRef>",
             f"<BkTxCd><Domn><Cd>{dom}</Cd><Fmly><Cd>{fam}</Cd><SubFmlyCd>{sub}</SubFmlyCd></Fmly></Domn></BkTxCd>",
             "<NtryDtls><TxDtls>",
             f"<Refs><AcctSvcrRef>{ref}</AcctSvcrRef><EndToEndId>{escape(t.e2e)}</EndToEndId>"
             + (f"<MndtId>{escape(t.mandate)}</MndtId>" if t.mandate else "") + "</Refs>",
             f"<Amt Ccy=\"EUR\">{amt:.2f}</Amt>"]
        if t.name:
            if t.amount > 0:
                x.append(f"<RltdPties><Dbtr><Pty><Nm>{escape(t.name)}</Nm></Pty></Dbtr>"
                         + (f"<DbtrAcct><Id><IBAN>{t.iban}</IBAN></Id></DbtrAcct>" if t.iban else "")
                         + f"<Cdtr><Pty><Nm>{escape(st.owner)}</Nm></Pty></Cdtr><CdtrAcct><Id><IBAN>{st.iban}</IBAN></Id></CdtrAcct></RltdPties>")
            else:
                x.append(f"<RltdPties><Dbtr><Pty><Nm>{escape(st.owner)}</Nm></Pty></Dbtr><DbtrAcct><Id><IBAN>{st.iban}</IBAN></Id></DbtrAcct>"
                         f"<Cdtr><Pty><Nm>{escape(t.name)}</Nm>"
                         + (f"<Id><PrvtId><Othr><Id>{t.creditor_id}</Id><SchmeNm><Prtry>SEPA</Prtry></SchmeNm></Othr></PrvtId></Id>" if t.creditor_id else "")
                         + "</Pty></Cdtr>"
                         + (f"<CdtrAcct><Id><IBAN>{t.iban}</IBAN></Id></CdtrAcct>" if t.iban else "")
                         + "</RltdPties>")
        x.append("<RmtInf>" + "".join(f"<Ustrd>{escape(part)}</Ustrd>" for part in _chunks(t.text, 140)) + "</RmtInf>")
        x.append(f"<AddtlTxInf>{escape(t.kind or ('Gutschrift' if t.amount > 0 else 'Überweisung'))}</AddtlTxInf>")
        x.append("</TxDtls></NtryDtls></Ntry>")
        out.append("".join(x))
    out.append("</Stmt></BkToCstmrStmt></Document>")
    return "\n".join(out) + "\n"


def _chunks(s: str, n: int):
    s = s.strip()
    return [s[i:i + n] for i in range(0, len(s), n)] or [""]


# ---------------------------------------------------------------- Ablage und Übersicht

@dataclass
class Entry:
    folder: str
    file: str
    date: str
    form: str
    issuer: str
    amount: str
    case: str


class Tenant:
    def __init__(self, dirname: str):
        self.dir = os.path.join(BASE, dirname)
        self.entries: list[Entry] = []
        self.counter: dict[str, int] = {}

    def path(self, folder: str, name: str) -> str:
        return os.path.join(self.dir, folder, name)

    def add(self, folder, file, date, form, issuer, amount, case):
        self.entries.append(Entry(folder, file, date, form, issuer, amount, case))

    def table(self, folder: str) -> str:
        rows = [e for e in self.entries if e.folder == folder]
        out = ["| Datei | Datum | Form | Von | Betrag | Testfall |", "|---|---|---|---|---:|---|"]
        for e in rows:
            out.append(f"| `{e.file}` | {ddate(e.date)} | {e.form} | {e.issuer} | {e.amount} | {e.case} |")
        return "\n".join(out)

    def write(self, folder: str, name: str, content: str):
        p = self.path(folder, name)
        os.makedirs(os.path.dirname(p), exist_ok=True)
        with open(p, "w", encoding="utf-8") as f:
            f.write(content)
        return p


def efile(t: Tenant, folder: str, name: str, inv: Invoice, *, fmt: str, issuer: str, case: str, pdf_kwargs=None):
    """fmt: zugferd | pdf | xrechnung | ubl"""
    kw = pdf_kwargs or {}
    amount = de(inv.gross, "€") if inv.currency == "EUR" else f"{inv.currency} {en(inv.gross)}"
    if fmt == "zugferd":
        compile_doc(t.path(folder, name), f"Rechnung {inv.number}", inv.seller.name, invoice_pages(inv, **kw), attach=[("factur-x.xml", cii(inv))])
        form = "ZUGFeRD (PDF + XML)"
    elif fmt == "pdf":
        compile_doc(t.path(folder, name), f"Rechnung {inv.number}", inv.seller.name, invoice_pages(inv, **kw))
        form = "PDF ohne Datensatz"
    elif fmt == "xrechnung":
        inv.guideline = "xrechnung"
        t.write(folder, name, cii(inv))
        form = "XRechnung (CII-XML)"
    elif fmt == "ubl":
        t.write(folder, name, ubl(inv))
        form = "UBL-XML (Peppol)"
    else:
        raise ValueError(fmt)
    t.add(folder, name, inv.date, form, issuer, amount, case)


def letter_pages(sender: Party, recipient_lines: list[str], *, meta, title, paras, font="Helvetica", accent="#333333",
                 tagline="", footer=None, sig=None, extra_blocks=None, sub=None, stamp=None) -> list[dict]:
    right = [sender.street, f"{sender.zip} {sender.city}"] + [x for x in [sender.phone and "Tel. " + sender.phone, sender.email, sender.web] if x]
    blocks = [{"type": "letterhead", "name": sender.name, "tagline": tagline, "right": right},
              {"type": "address", "sender_line": sender.one_line(), "lines": recipient_lines, "meta": meta},
              {"type": "title", "text": title, **({"sub": sub} if sub else {})}]
    for p in paras:
        blocks.append(p if isinstance(p, dict) else {"type": "para", "text": p})
    for b in extra_blocks or []:
        blocks.append(b)
    if sig:
        blocks.append({"type": "sig", "items": sig})
    page = {"kind": "a4", "font": font, "accent": accent, "blocks": blocks, "footer": footer or []}
    if stamp:
        page["stamp"] = stamp
    return [page]
