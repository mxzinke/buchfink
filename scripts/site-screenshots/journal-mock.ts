/**
 * Das Journal der Beispielfirma Nordlicht Systeme GmbH, aus dem sich alles
 * Weitere ableitet: Konten, Summen- und Saldenliste, Kontoblätter, Bilanz und
 * GuV, Bankumsätze und offene Posten.
 *
 * Es beginnt mit dem Saldenvortrag aus 2025, führt Januar bis Juli Monat für
 * Monat und endet mit dem August bis zum 25., dem Stand der Beispieldaten.
 * Die Werte sind so gewählt, dass die Jahreserklärungen in `declarations.json`
 * aufgehen: Erlöse 486.200 €, Vorsteuer 22.898,80 €, Einkünfte vor den nicht
 * abziehbaren Steuern 126.544 €.
 *
 * Die Firma meldet die Umsatzsteuer monatlich mit Dauerfristverlängerung: die
 * Zahllast 2025 lag weit über 9.000 €. Jede Voranmeldung ist einen Monat
 * später fällig, die Sondervorauszahlung im Februar.
 *
 * Bis zum 3. August bucht Anna Bauer in der App, danach Claude Code.
 */

export const YEAR = 2026;

export type MockLine = [side: 'S' | 'H', account: string, euro: number];

export interface BankInfo {
  counterparty: string;
  iban: string;
  remittance: string;
}

export interface MockEntry {
  /** Kennung in der Reihenfolge der Erfassung, wie im Backend. */
  id: number;
  /** Die Augustbuchungen tragen eine feste Marke 41 bis 55, auf die andere Daten zeigen. */
  ref?: number;
  date: string;
  documentDate: string;
  description: string;
  source: string;
  documentNumber?: string;
  contactId?: number;
  receiptId?: number;
  kind?: 'normal' | 'reversal';
  reversalOfId?: number;
  reversalReason?: string;
  /** Erfassungsweg: die App (`ui`) oder eine verbundene App (`agent:<kurzname>`). */
  channel?: string;
  bankTxId?: number;
  /** Der Bankumsatz, der zur Buchung gehört; ohne ihn entsteht keiner. */
  bank?: BankInfo;
  /** Ein Eingangsbeleg: er bekommt seine Belegnummer nach dem Eingang. */
  doc?: { issuer: string; subject: string };
  lines: MockLine[];
  /** Laufende Nummer, vergeben nach Datum. */
  number: string;
}

type Draft = Omit<MockEntry, 'number' | 'id'> & {
  /** Die Zahlung gleicht diesen Beleg aus; der Verwendungszweck nennt seine Nummer. */
  settles?: Draft | string;
  /** Wie `settles`, für die Augustbuchungen über ihre Marke. */
  settlesId?: number;
  /** Storno und Neubuchung tragen die Belegnummer der ersten Buchung. */
  sameDocAs?: number;
};

const round = (euro: number) => Math.round(euro * 100) / 100;
const vat = (net: number) => round(net * 0.19);
const pad = (n: number) => String(n).padStart(2, '0');

const MONTHS = ['Januar', 'Februar', 'März', 'April', 'Mai', 'Juni', 'Juli', 'August'];

// -------------------------------------------------------------------------
// Geschäftspartner

interface Partner {
  id: number;
  name: string;
  iban: string;
  /** Zahlungsziel in Tagen. */
  terms: number;
}

const CUSTOMERS: Record<string, Partner> = {
  nordwind: { id: 1, name: 'Nordwind Handels GmbH', iban: 'DE44100500001234567890', terms: 14 },
  elbtal: { id: 2, name: 'Elbtal Logistik KG', iban: 'DE72200400003456789012', terms: 30 },
  werft: { id: 3, name: 'Werft & Co. KG', iban: 'DE61200800005678901234', terms: 14 },
  marschland: { id: 4, name: 'Marschland Energie AG', iban: 'DE11217500000012345678', terms: 30 },
};

const VENDORS: Record<string, Partner> = {
  hanse: { id: 11, name: 'Hanse Cloud Services GmbH', iban: 'DE55201207007890123456', terms: 14 },
  kontorhaus: { id: 12, name: 'Kontorhaus Verwaltung GmbH', iban: 'DE89200505501234500012', terms: 0 },
  techpartner: { id: 13, name: 'Techpartner Nord GmbH', iban: 'DE93370400444567890123', terms: 14 },
  reise: { id: 14, name: 'Hanseatische Reisestelle GmbH', iban: 'DE27200400600987654321', terms: 0 },
  nordfunk: { id: 15, name: 'Nordfunk Telekommunikation GmbH', iban: 'DE05300606016789012345', terms: 0 },
  fachportal: { id: 16, name: 'Fachportal Media GmbH', iban: 'DE18500105172345678901', terms: 21 },
  lorenz: { id: 18, name: 'Kai Lorenz IT-Beratung', iban: 'DE68200505501011223344', terms: 10 },
  kanzlei: { id: 19, name: 'Kanzlei Elbufer Rechtsanwälte', iban: 'DE41200300000055667788', terms: 14 },
};

// -------------------------------------------------------------------------
// Bausteine

/** Eine Ausgangsrechnung: Forderung gegen Erlös und Umsatzsteuer. */
function invoice(date: string, number: string, customer: Partner, net: number, text: string, channel = 'ui'): Draft {
  return {
    date, documentDate: date, description: `${text}, Rechnung ${number}`,
    source: 'invoice', documentNumber: number, contactId: customer.id, channel,
    lines: [['S', '1200', round(net + vat(net))], ['H', '4400', net], ['H', '3806', vat(net)]],
  };
}

/** Der Zahlungseingang zu einer Rechnung. */
function receipt(date: string, number: string, customer: Partner, gross: number, channel = 'ui'): Draft {
  return {
    date, documentDate: date, description: `Zahlungseingang zu ${number}`,
    source: 'payment', contactId: customer.id, channel,
    bank: { counterparty: customer.name, iban: customer.iban, remittance: `${number}` },
    lines: [['S', '1800', gross], ['H', '1200', gross]],
  };
}

/** Eine Eingangsrechnung auf Ziel: Aufwand und Vorsteuer gegen Verbindlichkeit. */
function bill(date: string, vendor: Partner, account: string, net: number, text: string, channel = 'ui'): Draft {
  return {
    date, documentDate: date, description: text,
    source: 'receipt', contactId: vendor.id, channel, doc: { issuer: vendor.name, subject: text },
    lines: [['S', account, net], ['S', '1406', vat(net)], ['H', '3300', round(net + vat(net))]],
  };
}

/**
 * Die Zahlung einer Eingangsrechnung. Den Verwendungszweck mit der
 * Belegnummer setzt `settles`, sobald die Nummern vergeben sind.
 */
function payment(date: string, of: Draft | string, vendor: Partner, gross: number, channel = 'ui'): Draft {
  return {
    date, documentDate: date, description: 'Zahlung',
    source: 'payment', contactId: vendor.id, channel,
    bank: { counterparty: vendor.name, iban: vendor.iban, remittance: typeof of === 'string' ? of : '' },
    settles: of,
    lines: [['S', '3300', gross], ['H', '1800', gross]],
  } as Draft;
}

/** Ein Aufwand, der direkt vom Bankkonto abgeht. */
function direct(
  date: string, vendor: Partner, account: string, net: number, text: string,
  options: { withVat?: boolean; channel?: string } = {},
): Draft {
  const withVat = options.withVat ?? true;
  const gross = withVat ? round(net + vat(net)) : net;
  const lines: MockLine[] = withVat
    ? [['S', account, net], ['S', '1406', vat(net)], ['H', '1800', gross]]
    : [['S', account, net], ['H', '1800', gross]];
  return {
    date, documentDate: date, description: text,
    source: 'receipt', contactId: vendor.id, channel: options.channel ?? 'ui',
    doc: { issuer: vendor.name, subject: text },
    bank: { counterparty: vendor.name, iban: vendor.iban, remittance: text },
    lines,
  };
}

/** Eine Buchung ohne Beleg eines Geschäftspartners: Gehälter, Steuern, Erstattungen. */
function plain(date: string, text: string, lines: MockLine[], bank?: BankInfo, source = 'manual'): Draft {
  return { date, documentDate: date, description: text, source, channel: 'ui', bank, lines };
}

/** Das Geschäftskonto bei der Deutschen Kreditbank, wie in den Stammdaten. */
const OWN_BANK = 'DE02120300000000202051';

const BANK: Partner = { id: 17, name: 'Deutsche Kreditbank', iban: '', terms: 0 };

/** Die Sondervorauszahlung 2026: ein Elftel der Vorauszahlungen 2025. */
export const SPECIAL_PREPAYMENT = 5221;

/**
 * Fälligkeit der Voranmeldungen Januar bis August mit Dauerfristverlängerung:
 * der 10. des übernächsten Monats, am Wochenende der nächste Werktag.
 */
export const VAT_DUE = [
  `${YEAR}-03-10`, `${YEAR}-04-10`, `${YEAR}-05-11`, `${YEAR}-06-10`,
  `${YEAR}-07-10`, `${YEAR}-08-10`, `${YEAR}-09-10`, `${YEAR}-10-12`,
];

/** Die Zahllast eines Monats aus den Entwürfen: Umsatzsteuer minus Vorsteuer. */
function vatPayable(drafts: Draft[], month: number): number {
  const prefix = `${YEAR}-${pad(month)}`;
  let tax = 0;
  for (const d of drafts) {
    if (!d.date.startsWith(prefix)) continue;
    for (const [side, account, euro] of d.lines) {
      if (account === '3806') tax += side === 'H' ? euro : -euro;
      if (account === '1406') tax -= side === 'S' ? euro : -euro;
    }
  }
  return round(tax);
}

// -------------------------------------------------------------------------
// Saldenvortrag

/**
 * Die Eröffnungswerte zum 01.01.2026 aus der Schlussbilanz 2025, gebucht
 * gegen 9000. Die Rückstellungen und Abgrenzungen stehen auch in den
 * Karteien der Abschlüsse (`closings-mock.ts`).
 */
export const OPENING: [account: string, side: 'S' | 'H', euro: number][] = [
  ['0135', 'S', 10000],
  ['0440', 'S', 21250],
  ['0520', 'S', 36166.67],
  ['0650', 'S', 8430.77],
  ['0675', 'S', 3360],
  ['1200', 'S', 17850],
  ['1800', 'S', 28229.96],
  ['1900', 'S', 2220],
  ['2900', 'H', 25000],
  ['2970', 'H', 61230],
  ['3035', 'H', 3906],
  ['3040', 'H', 2950.4],
  ['3095', 'H', 1450],
  ['3096', 'H', 1570],
  ['3300', 'H', 4760],
  ['3642', 'H', 20000],
  ['3841', 'H', 6641],
];

function openingEntries(): Draft[] {
  return OPENING.map(([account, side, euro]) => ({
    date: `${YEAR}-01-01`,
    documentDate: `${YEAR}-01-01`,
    description: `Saldenvortrag ${account} aus 2025`,
    source: 'opening',
    documentNumber: `SV ${YEAR}`,
    channel: 'system',
    lines: side === 'S' ? [['S', account, euro], ['H', '9000', euro]] : [['S', '9000', euro], ['H', account, euro]],
  }));
}

// -------------------------------------------------------------------------
// Januar bis Juli

/** Die Ausgangsrechnungen: Nummer, Datum, Kunde, Netto, Leistung. */
const SALES: [number: string, date: string, customer: keyof typeof CUSTOMERS, net: number, text: string, paid?: string][] = [
  ['RE-2026-0099', '01-05', 'nordwind', 30000, 'Rahmenvertrag Portalpflege Januar', '01-19'],
  ['RE-2026-0100', '01-19', 'elbtal', 22400, 'Schnittstelle Lagerverwaltung', '02-18'],
  ['RE-2026-0101', '02-02', 'nordwind', 30000, 'Rahmenvertrag Portalpflege Februar', '02-09'],
  ['RE-2026-0102', '02-16', 'marschland', 28900, 'Kundenportal Netzanschluss, Abnahme 1', '03-18'],
  ['RE-2026-0103', '03-02', 'nordwind', 30000, 'Rahmenvertrag Portalpflege März', '03-09'],
  ['RE-2026-0104', '03-17', 'werft', 31300, 'Projektverwaltung Dock 4', '03-31'],
  ['RE-2026-0105', '04-01', 'nordwind', 30000, 'Rahmenvertrag Portalpflege April', '04-15'],
  ['RE-2026-0106', '04-20', 'elbtal', 27800, 'Tourenplanung, Erweiterung', '05-20'],
  ['RE-2026-0107', '05-04', 'nordwind', 30000, 'Rahmenvertrag Portalpflege Mai', '05-18'],
  ['RE-2026-0108', '05-18', 'marschland', 33400, 'Kundenportal Netzanschluss, Abnahme 2', '06-17'],
  ['RE-2026-0109', '06-01', 'nordwind', 30000, 'Rahmenvertrag Portalpflege Juni', '06-15'],
  ['RE-2026-0110', '06-15', 'werft', 36200, 'Projektverwaltung Dock 4, Abschluss', '06-29'],
  ['RE-2026-0111', '07-01', 'nordwind', 22700, 'Rahmenvertrag Portalpflege Juli', '07-15'],
  ['RE-2026-0112', '07-06', 'elbtal', 15900, 'Tourenplanung, Mobilansicht'],
  ['RE-2026-0113', '07-07', 'marschland', 12300, 'Kundenportal, Wartung Juli'],
  ['RE-2026-0114', '07-08', 'elbtal', 4800, 'Schulung Tourenplanung'],
  ['RE-2026-0115', '07-14', 'marschland', 12400, 'Kundenportal, Lastprofile', '07-30'],
  ['RE-2026-0116', '07-28', 'werft', 6000, 'Projektverwaltung Dock 4, Nachträge'],
  ['RE-2026-0117', '07-31', 'elbtal', 21000, 'Tourenplanung, Disposition'],
];

/** Fremdleistung Entwicklung je Monat; dazu je 1.480 € Hosting. */
const FREELANCE = [7210, 6930, 11096, 8410, 6750, 11556, 9268];
const SALARY = 31150;
const BANK_FEES = [58, 62, 60, 61, 59, 62, 62];
const LAST_BANK_DAY = ['01-30', '02-27', '03-31', '04-30', '05-29', '06-30', '07-31'];

/** Die Rechnung der freien Mitarbeit im Juli; Claude zahlt sie im August. */
let julyFreelance: Draft | undefined;

function monthEntries(): Draft[] {
  const out: Draft[] = [];
  const y = (md: string) => `${YEAR}-${md}`;

  // Was aus 2025 offen war.
  out.push(receipt(y('01-08'), 'RE-2025-0098', CUSTOMERS.nordwind, 17850));
  out.push(payment(y('01-07'), 'ER-2025-0159', VENDORS.lorenz, 4760));
  // Die Voranmeldungen November und Dezember 2025; der Dezember abzüglich der Sondervorauszahlung 2025.
  out.push(plain(y('01-12'), 'Umsatzsteuer November 2025', [['S', '3841', 4980], ['H', '1800', 4980]],
    { counterparty: 'Finanzamt Hamburg-Mitte', iban: 'DE38200000000020001500', remittance: 'USt 11/2025' }, 'tax'));
  out.push(plain(y('02-10'), 'Umsatzsteuer Dezember 2025', [['S', '3841', 1661], ['H', '1800', 1661]],
    { counterparty: 'Finanzamt Hamburg-Mitte', iban: 'DE38200000000020001500', remittance: 'USt 12/2025' }, 'tax'));

  for (const [number, md, key, net, text, paid] of SALES) {
    const customer = CUSTOMERS[key];
    out.push(invoice(y(md), number, customer, net, text));
    if (paid) out.push(receipt(y(paid), number, customer, round(net + vat(net))));
  }
  // RE-2026-0114 ging an den falschen Ansprechpartner und ist storniert.
  out.push({
    date: y('07-10'), documentDate: y('07-10'), description: 'Storno zu RE-2026-0114',
    source: 'invoice', documentNumber: 'RE-2026-0114', contactId: CUSTOMERS.elbtal.id, channel: 'ui',
    kind: 'reversal', reversalReason: 'Rechnung an die falsche Gesellschaft adressiert.',
    lines: [['S', '4400', 4800], ['S', '3806', 912], ['H', '1200', 5712]],
  });

  for (let m = 1; m <= 7; m++) {
    const mm = pad(m);
    const name = `${MONTHS[m - 1]} ${YEAR}`;

    out.push({
      ...direct(y(`${mm}-01`), VENDORS.kontorhaus, '6310', 3600, `Büromiete ${name}`, { withVat: false }),
      bank: { counterparty: VENDORS.kontorhaus.name, iban: VENDORS.kontorhaus.iban, remittance: `Miete ${name}` },
    });

    const hosting = bill(y(`${mm}-02`), VENDORS.hanse, '5906', 1480, `Cloud-Hosting ${name}`);
    out.push(hosting, payment(y(`${mm}-16`), hosting, VENDORS.hanse, round(1480 + vat(1480))));

    out.push(direct(y(`${mm}-15`), VENDORS.nordfunk, '6805', 270, `Telefon und Internet ${name}`));

    const freelance = bill(y(`${mm}-25`), VENDORS.lorenz, '5906', FREELANCE[m - 1], `Entwicklung, freie Mitarbeit ${name}`);
    out.push(freelance);
    // Die Juli-Rechnung zahlt Claude im August.
    if (m < 7) {
      out.push(payment(y(`${pad(m + 1)}-05`), freelance, VENDORS.lorenz, round(FREELANCE[m - 1] + vat(FREELANCE[m - 1]))));
    } else {
      julyFreelance = freelance;
    }

    // Die Gehälter für Juli gehen am 6. August hinaus (Buchung 43).
    if (m < 7) {
      out.push(plain(y(LAST_BANK_DAY[m - 1]), `Gehälter ${name}`, [['S', '6020', SALARY], ['H', '1800', SALARY]],
        { counterparty: 'Gehaltszahlungen Sammelauftrag', iban: OWN_BANK, remittance: `Löhne und Gehälter ${name}` }));
    }
    out.push({
      ...direct(y(LAST_BANK_DAY[m - 1]), BANK, '6855', BANK_FEES[m - 1], `Kontoführung ${name}`, { withVat: false }),
      bank: { counterparty: BANK.name, iban: OWN_BANK, remittance: `Entgelt Kontoführung ${name}` },
    });
  }

  // Die übrigen Aufwendungen, wie sie über das Jahr anfielen.
  const billed: [md: string, vendor: keyof typeof VENDORS, account: string, net: number, text: string, paid: string][] = [
    ['01-12', 'techpartner', '6495', 4200, 'Jahreslizenz Entwicklungsumgebung 2026', '01-26'],
    ['02-10', 'fachportal', '6600', 2500, 'Anzeigen Fachportal Logistik, Februar', '03-03'],
    ['02-24', 'kanzlei', '6825', 2900, 'Rechtsberatung AGB und Lizenzverträge', '03-10'],
    ['03-10', 'hanse', '6495', 1890, 'Wartung Datenbankserver, 1. Quartal', '03-24'],
    ['04-08', 'techpartner', '6495', 1240, 'Lizenzerweiterung Datenbank', '04-22'],
    ['04-14', 'fachportal', '6600', 1500, 'Anzeigen Fachportal Logistik, April', '05-05'],
    ['05-12', 'fachportal', '6600', 2200, 'Messeauftritt Fachportal, Mai', '06-02'],
    ['05-19', 'kanzlei', '6825', 2500, 'Rechtsberatung Rahmenvertrag Marschland', '06-02'],
    ['06-09', 'techpartner', '6495', 2400, 'Supportvertrag Server, 2. Halbjahr', '06-23'],
    ['07-13', 'techpartner', '6495', 2740, 'Lizenzen Monitoring, 2. Halbjahr', '07-27'],
    ['07-14', 'fachportal', '6600', 1800, 'Anzeigen Fachportal Logistik, Juli', '07-28'],
    ['07-20', 'kanzlei', '6825', 2500, 'Rechtsberatung Datenschutzvereinbarungen', '07-31'],
  ];
  for (const [md, key, account, net, text, paid] of billed) {
    const doc = bill(y(md), VENDORS[key], account, net, text);
    out.push(doc, payment(y(paid), doc, VENDORS[key], round(net + vat(net))));
  }

  const paidDirect: [md: string, vendor: keyof typeof VENDORS, account: string, net: number, text: string][] = [
    ['01-14', 'techpartner', '6815', 640, 'Druckerpapier und Toner'],
    ['02-05', 'techpartner', '6260', 1020, 'Bürostühle, 2 Stück'],
    ['03-11', 'techpartner', '6815', 380, 'Bürobedarf März'],
    ['03-20', 'reise', '6650', 1284, 'Reisekosten Kundentermin Husum'],
    ['04-20', 'techpartner', '6260', 780, 'Notebook Entwicklung'],
    ['04-21', 'techpartner', '6815', 520, 'Headsets, 4 Stück'],
    ['05-13', 'techpartner', '6815', 610, 'Bürobedarf Mai'],
    ['05-22', 'reise', '6650', 2090, 'Reisekosten Messe Bremen'],
    ['06-12', 'reise', '6650', 1124, 'Reisekosten Kundentermin Husum'],
    ['06-16', 'techpartner', '6815', 470, 'Kabel und Adapter'],
    ['07-09', 'techpartner', '6815', 800, 'Whiteboards Besprechungsraum'],
    ['07-21', 'reise', '6650', 900, 'Reisekosten Projektabnahme Werft'],
  ];
  for (const [md, key, account, net, text] of paidDirect) {
    out.push(direct(y(md), VENDORS[key], account, net, text));
  }

  // Erstattungen der Krankenkasse aus der Umlage U2.
  for (const md of ['03-25', '04-24', '05-26']) {
    out.push(plain(y(md), 'Erstattung Umlage U2', [['S', '1800', 1250], ['H', '4830', 1250]],
      { counterparty: 'Techniker Krankenkasse', iban: 'DE13200400000622277000', remittance: 'Erstattung U2' }));
  }

  // Umsatzsteuer: die Sondervorauszahlung, ein Elftel der Vorauszahlungen 2025
  // (57.431 €), und die Zahllast je Monat auf 3820. Verrechnet wird zum Abschluss.
  const fa = (text: string) => ({ counterparty: 'Finanzamt Hamburg-Mitte', iban: 'DE38200000000020001500', remittance: text });
  out.push(plain(y('02-10'), 'Sondervorauszahlung 2026', [['S', '3830', SPECIAL_PREPAYMENT], ['H', '1800', SPECIAL_PREPAYMENT]],
    fa('USt-SVZ 2026'), 'tax'));
  for (let m = 1; m <= 5; m++) {
    const due = VAT_DUE[m - 1];
    const payable = vatPayable(out, m);
    out.push(plain(due, `Umsatzsteuer-Vorauszahlung ${MONTHS[m - 1]} ${YEAR}`, [['S', '3820', payable], ['H', '1800', payable]],
      fa(`USt ${pad(m)}/${YEAR}`), 'tax'));
  }

  // Die Steuern 2025 nach den Bescheiden, aus den Rückstellungen.
  out.push(plain(y('07-20'), 'Körperschaftsteuer 2025, Abschlusszahlung', [['S', '3040', 2950.4], ['H', '1800', 2950.4]],
    fa('KSt 2025'), 'tax'));
  out.push(plain(y('07-27'), 'Gewerbesteuer 2025, Abschlusszahlung', [['S', '3035', 3906], ['H', '1800', 3906]],
    { counterparty: 'Freie und Hansestadt Hamburg', iban: 'DE60200000000020101500', remittance: 'GewSt 2025' }, 'tax'));

  // Vorauszahlungen laut Bescheid.
  const gemeinde = { counterparty: 'Freie und Hansestadt Hamburg', iban: 'DE60200000000020101500', remittance: '' };
  out.push(plain(y('02-16'), 'Gewerbesteuer-Vorauszahlung 15.02.2026', [['S', '7610', 3000], ['H', '1800', 3000]],
    { ...gemeinde, remittance: 'GewSt VZ I/2026' }, 'tax'));
  out.push(plain(y('05-15'), 'Gewerbesteuer-Vorauszahlung 15.05.2026', [['S', '7610', 3000], ['H', '1800', 3000]],
    { ...gemeinde, remittance: 'GewSt VZ II/2026' }, 'tax'));
  out.push(plain(y('03-10'), 'Körperschaftsteuer-Vorauszahlung 10.03.2026',
    [['S', '7600', 1750], ['S', '7608', 96.25], ['H', '1800', 1846.25]], fa('KSt VZ I/2026'), 'tax'));
  out.push(plain(y('06-10'), 'Körperschaftsteuer-Vorauszahlung 10.06.2026',
    [['S', '7600', 1750], ['S', '7608', 96.25], ['H', '1800', 1846.25]], fa('KSt VZ II/2026'), 'tax'));

  return out;
}

// -------------------------------------------------------------------------
// August

/**
 * Die Buchungen des Augusts, auf die Belege, offene Posten und Aufnahmen
 * zeigen. Sie tragen die Marke 41 bis 55; `reversalOfId`, `settlesId` und
 * `sameDocAs` nennen hier Marken.
 */
const AUGUST: Draft[] = [
  {
    ref: 41, channel: 'agent:claude-code', date: '2026-08-03', documentDate: '2026-08-03',
    description: 'Rahmenvertrag Portalpflege, Rechnung RE-2026-0118',
    source: 'invoice', documentNumber: 'RE-2026-0118', contactId: 1,
    lines: [['S', '1200', 34510], ['H', '4400', 29000], ['H', '3806', 5510]],
  },
  {
    ref: 42, channel: 'agent:claude-code', date: '2026-08-04', documentDate: '2026-08-01',
    description: 'Cloud-Hosting August 2026',
    source: 'receipt', contactId: 11, receiptId: 4,
    doc: { issuer: VENDORS.hanse.name, subject: 'Cloud-Hosting August 2026' },
    lines: [['S', '5906', 1480], ['S', '1406', 281.2], ['H', '3300', 1761.2]],
  },
  {
    ref: 43, channel: 'agent:claude-code', date: '2026-08-06', documentDate: '2026-07-31',
    description: 'Gehälter Juli 2026', source: 'manual',
    bank: { counterparty: 'Gehaltszahlungen Sammelauftrag', iban: OWN_BANK, remittance: 'Löhne und Gehälter Juli 2026' },
    lines: [['S', '6020', 26700], ['H', '1800', 26700]],
  },
  {
    ref: 44, channel: 'agent:claude-code', date: '2026-08-07', documentDate: '2026-08-07',
    description: 'Zahlungseingang zu RE-2026-0112', source: 'payment', contactId: 2,
    bank: { counterparty: CUSTOMERS.elbtal.name, iban: CUSTOMERS.elbtal.iban, remittance: 'RE-2026-0112' },
    lines: [['S', '1800', 18921], ['H', '1200', 18921]],
  },
  {
    ref: 45, channel: 'agent:claude-code', date: '2026-08-10', documentDate: '2026-08-01',
    description: 'Büromiete August 2026', source: 'receipt', contactId: 12, receiptId: 5,
    bank: { counterparty: VENDORS.kontorhaus.name, iban: VENDORS.kontorhaus.iban, remittance: 'Miete August 2026' },
    doc: { issuer: VENDORS.kontorhaus.name, subject: 'Miete Büro August 2026' },
    lines: [['S', '6310', 3600], ['H', '1800', 3600]],
  },
  {
    ref: 46, channel: 'agent:claude-code', date: '2026-08-11', documentDate: '2026-08-03',
    description: 'Monitore und Dockingstationen, 4 Stück (geringwertige Wirtschaftsgüter)',
    source: 'receipt', contactId: 13,
    doc: { issuer: VENDORS.techpartner.name, subject: 'Monitore und Dockingstationen, 4 Stück' },
    lines: [['S', '6260', 2380], ['S', '1406', 452.2], ['H', '3300', 2832.2]],
  },
  {
    ref: 47, channel: 'agent:claude-code', date: '2026-08-12', documentDate: '2026-08-09',
    description: 'Reisekosten Fachmesse Berlin', source: 'receipt', contactId: 14,
    bank: { counterparty: VENDORS.reise.name, iban: VENDORS.reise.iban, remittance: 'Reisekosten Fachmesse Berlin' },
    doc: { issuer: VENDORS.reise.name, subject: 'Reisekosten Fachmesse Berlin' },
    lines: [['S', '6650', 842], ['S', '1406', 159.98], ['H', '1800', 1001.98]],
  },
  {
    ref: 48, channel: 'agent:claude-code', date: '2026-08-13', documentDate: '2026-08-12',
    description: 'Wartungsvertrag Datenbank, Q3 2026',
    source: 'receipt', contactId: 11,
    doc: { issuer: VENDORS.hanse.name, subject: 'Wartungsvertrag Datenbank, Q3 2026' },
    lines: [['S', '6495', 1240], ['S', '1406', 235.6], ['H', '3300', 1475.6]],
  },
  {
    ref: 49, channel: 'agent:claude-code', date: '2026-08-14', documentDate: '2026-08-12',
    description: 'Generalumkehr zu Wartungsvertrag Datenbank, Q3 2026',
    source: 'receipt', contactId: 11,
    kind: 'reversal', reversalOfId: 48,
    reversalReason: 'Aufwand gehört auf 5906 Fremdleistungen, nicht auf 6495.',
    sameDocAs: 48,
    lines: [['S', '3300', 1475.6], ['H', '6495', 1240], ['H', '1406', 235.6]],
  },
  {
    ref: 50, channel: 'agent:claude-code', date: '2026-08-14', documentDate: '2026-08-12',
    description: 'Wartungsvertrag Datenbank, Q3 2026 (Neubuchung)',
    source: 'receipt', contactId: 11,
    sameDocAs: 48,
    lines: [['S', '5906', 1240], ['S', '1406', 235.6], ['H', '3300', 1475.6]],
  },
  {
    ref: 51, channel: 'ui', date: '2026-08-17', documentDate: '2026-08-17',
    description: 'Wartungspauschale Q3, Rechnung RE-2026-0119',
    source: 'invoice', documentNumber: 'RE-2026-0119', contactId: 1,
    lines: [['S', '1200', 8211], ['H', '4400', 6900], ['H', '3806', 1311]],
  },
  {
    ref: 52, channel: 'agent:claude-code', bankTxId: 7, date: '2026-08-18', documentDate: '2026-08-15',
    description: 'Telefon und Internet August 2026',
    source: 'receipt', contactId: 15, receiptId: 3,
    doc: { issuer: VENDORS.nordfunk.name, subject: 'Telefon und Anschluss August 2026' },
    lines: [['S', '6805', 270], ['S', '1406', 51.3], ['H', '1800', 321.3]],
  },
  {
    ref: 53, channel: 'agent:claude-code', bankTxId: 8, date: '2026-08-20', documentDate: '2026-08-20',
    description: 'Zahlung', source: 'payment', contactId: 11, settlesId: 42,
    lines: [['S', '3300', 1761.2], ['H', '1800', 1761.2]],
  },
  {
    ref: 54, channel: 'agent:claude-code', date: '2026-08-21', documentDate: '2026-08-19',
    description: 'Werbekampagne Fachportal', source: 'receipt', contactId: 16,
    doc: { issuer: VENDORS.fachportal.name, subject: 'Werbekampagne August 2026' },
    lines: [['S', '6600', 1850], ['S', '1406', 351.5], ['H', '3300', 2201.5]],
  },
  {
    ref: 55, channel: 'agent:claude-code', date: '2026-08-24', documentDate: '2026-08-24',
    description: 'Kontoführung August 2026', source: 'receipt', contactId: 17,
    bank: { counterparty: BANK.name, iban: OWN_BANK, remittance: 'Entgelt Kontoführung August 2026' },
    doc: { issuer: BANK.name, subject: 'Kontoführung August 2026' },
    lines: [['S', '6855', 62], ['H', '1800', 62]],
  },
];

/** Was Claude im August sonst noch aus den Kontoauszügen gebucht hat. */
function augustPayments(earlier: Draft[]): Draft[] {
  const agent = 'agent:claude-code';
  const lorenzJuly = round(FREELANCE[6] + vat(FREELANCE[6]));
  const june = vatPayable(earlier, 6);
  return [
    {
      ...plain(VAT_DUE[5], `Umsatzsteuer-Vorauszahlung Juni ${YEAR}`, [['S', '3820', june], ['H', '1800', june]],
        { counterparty: 'Finanzamt Hamburg-Mitte', iban: 'DE38200000000020001500', remittance: `USt 06/${YEAR}` }, 'tax'),
      channel: agent,
    },
    { ...payment('2026-08-04', julyFreelance!, VENDORS.lorenz, lorenzJuly, agent) },
    { ...receipt('2026-08-06', 'RE-2026-0113', CUSTOMERS.marschland, round(12300 + vat(12300)), agent) },
    {
      ...receipt('2026-08-12', 'RE-2026-0116', CUSTOMERS.werft, 3570, agent),
      description: 'Zahlungseingang zu RE-2026-0116, erste Hälfte',
    },
    { ...receipt('2026-08-17', 'RE-2026-0118', CUSTOMERS.nordwind, 34510, agent) },
  ];
}

// -------------------------------------------------------------------------
// Zusammenführen

/** Ein Eingangsbeleg im Belegarchiv. */
export interface ReceiptDoc {
  id: number;
  number: string;
  date: string;
  issuer: string;
  subject: string;
  gross: number;
  tax: number;
  status: 'sealed' | 'filed' | 'discarded';
  entryIds: number[];
  contactId?: number;
  discardReason?: string;
}

/** Belege ohne Buchung: abgelegt, aber noch offen, und ein verworfenes Doppel. */
const LOOSE_DOCS: Omit<ReceiptDoc, 'number' | 'entryIds'>[] = [
  { id: 6, date: '2026-07-29', issuer: VENDORS.hanse.name, subject: 'Cloud-Hosting Juli 2026', gross: 1761.2, tax: 281.2, status: 'discarded', contactId: 11 },
  { id: 1, date: '2026-08-10', issuer: VENDORS.techpartner.name, subject: 'Notebooks, 4 Stück', gross: 3232.04, tax: 516.04, status: 'filed', contactId: 13 },
  { id: 2, date: '2026-08-21', issuer: 'Stadtwerke Nord AG', subject: 'Abschlag Strom August 2026', gross: 428.4, tax: 68.4, status: 'filed' },
];

/** Die Belegnummer läuft über die Jahre weiter; 2025 endete sie bei 0159. */
const FIRST_RECEIPT = 160;

const built = (() => {
  const months = monthEntries();
  const drafts = [...openingEntries(), ...months, ...augustPayments(months), ...AUGUST];
  // Stabil nach Datum; am selben Tag Saldenvortrag und Rechnungen vor Zahlungen.
  const order = (d: Draft) => (d.source === 'opening' ? 0 : d.source === 'invoice' ? 1 : 2);
  const sorted = drafts
    .map((d, index) => ({ d, index }))
    .sort((a, b) => a.d.date.localeCompare(b.d.date) || order(a.d) - order(b.d) || a.index - b.index)
    .map(({ d }) => d);
  const byDraft = new Map<Draft, MockEntry>();
  const entries: MockEntry[] = sorted.map((d, index) => {
    const { settles: _settles, settlesId: _settlesId, sameDocAs: _same, ...rest } = d;
    const entry: MockEntry = {
      ...rest,
      id: index + 1,
      number: `B-${YEAR}-${String(index + 1).padStart(4, '0')}`,
    };
    byDraft.set(d, entry);
    return entry;
  });
  const byRef = (ref: number) => entries.find((e) => e.ref === ref)!;
  for (const e of entries) {
    if (e.ref && e.reversalOfId) e.reversalOfId = byRef(e.reversalOfId).id;
  }

  // Belegnummern nach Eingang, über gebuchte und lose Belege hinweg.
  type Slot = { date: string; entry?: MockEntry; loose?: (typeof LOOSE_DOCS)[number] };
  const slots: Slot[] = [
    ...entries.filter((e) => e.doc).map((entry) => ({ date: entry.documentDate, entry })),
    ...LOOSE_DOCS.map((loose) => ({ date: loose.date, loose })),
  ].sort((a, b) => a.date.localeCompare(b.date));
  const KNOWN_RECEIPT: Record<number, number> = { 52: 3, 42: 4, 45: 5 };
  let nextReceipt = 100;
  const docs: ReceiptDoc[] = slots.map((slot, index) => {
    const number = `ER-${YEAR}-${String(FIRST_RECEIPT + index).padStart(4, '0')}`;
    if (slot.loose) return { ...slot.loose, number, entryIds: [] };
    const entry = slot.entry!;
    const id = (entry.ref && KNOWN_RECEIPT[entry.ref]) || nextReceipt++;
    entry.documentNumber = number;
    entry.receiptId = id;
    const gross = entry.lines.filter((l) => l[0] === 'H').reduce((sum, l) => sum + l[2], 0);
    const tax = entry.lines.find((l) => l[1] === '1406')?.[2] ?? 0;
    return {
      id, number, date: entry.documentDate, issuer: entry.doc!.issuer, subject: entry.doc!.subject,
      gross: round(gross), tax, status: 'sealed', entryIds: [entry.id], contactId: entry.contactId,
    };
  });

  // Storno und Neubuchung zum selben Beleg.
  for (const d of sorted) {
    if (!d.sameDocAs) continue;
    const entry = byDraft.get(d)!;
    const first = byRef(d.sameDocAs);
    entry.documentNumber = first.documentNumber;
    entry.receiptId = first.receiptId;
    docs.find((doc) => doc.id === first.receiptId)!.entryIds.push(entry.id);
  }

  // Der Verwendungszweck einer Zahlung nennt den Beleg, den sie ausgleicht.
  for (const d of sorted) {
    if (!d.settles && !d.settlesId) continue;
    const entry = byDraft.get(d)!;
    const settled =
      typeof d.settles === 'object'
        ? byDraft.get(d.settles)
        : d.settlesId
          ? byRef(d.settlesId)
          : undefined;
    const number = typeof d.settles === 'string' ? d.settles : settled?.documentNumber ?? '';
    entry.description = `Zahlung ${number}`;
    if (entry.bank) {
      entry.bank = { ...entry.bank, remittance: settled ? `${number} ${settled.description}` : number };
    }
  }

  // Die Generalumkehr des Juli zeigt auf ihre Rechnung.
  const cancelled = entries.find((e) => e.documentNumber === 'RE-2026-0114' && e.kind !== 'reversal');
  for (const e of entries) {
    if (e.kind === 'reversal' && e.documentNumber === 'RE-2026-0114') e.reversalOfId = cancelled?.id;
  }

  // Jede Buchung mit Bankbewegung hat ihren Umsatz aus dem Kontoauszug.
  let nextTx = 100;
  for (const entry of entries) {
    if (entry.bank && !entry.bankTxId) entry.bankTxId = nextTx++;
  }

  const discarded = docs.find((doc) => doc.status === 'discarded')!;
  const julyHosting = docs.find((doc) => doc.subject === 'Cloud-Hosting Juli 2026' && doc.status === 'sealed');
  discarded.discardReason = `Doppelt eingegangen, gebucht ist ${julyHosting?.number}.`;

  return { entries, docs };
})();

/** Alle Buchungen des Jahres nach Datum, mit fortlaufender Nummer. */
export const JOURNAL: MockEntry[] = built.entries;

/** Alle Eingangsbelege des Jahres, nach Belegnummer. */
export const RECEIPT_DOCS: ReceiptDoc[] = built.docs;

/** Die Augustbuchung mit dieser Marke. */
export function entryByRef(ref: number): MockEntry {
  return JOURNAL.find((e) => e.ref === ref)!;
}

/** Die Nummer der Augustbuchung mit dieser Marke. */
export function entryNumber(ref: number): string {
  return entryByRef(ref).number;
}

/** Die Buchung, die eine Steuervorauszahlung ausgleicht. */
export function prepaymentEntry(description: string): string {
  return JOURNAL.find((e) => e.description === description)?.number ?? '';
}
