/**
 * Beispieldaten für die Screenshots der Projektseite.
 *
 * Diese Datei tritt beim Screenshot-Lauf an die Stelle von
 * `frontend/src/services/bridge.ts`. Die Oberfläche bleibt unverändert: es
 * laufen dieselben Seiten, dieselben Bausteine und dieselben Formatierer, nur
 * antwortet statt der Wails-Laufzeit dieses Modul.
 *
 * Die Zahlen gehören zu einer erfundenen Nordlicht Systeme GmbH. Sie sind
 * untereinander stimmig gerechnet — Soll gleich Haben, Umsatzsteuer 19 % auf
 * das Entgelt, Zahllast gleich Umsatzsteuer minus Vorsteuer —, damit die
 * Screenshots keine Zahlen zeigen, die es so nicht geben kann.
 */

// Die Oberfläche prüft die Laufzeit über `window._wails`, bevor sie einen
// Bridge-Aufruf zulässt (siehe services/api.ts). Für den Screenshot-Lauf wird
// die Prüfung erfüllt, der Aufruf selbst läuft hier weiter.
if (typeof window !== 'undefined') {
  (window as any)._wails = { screenshotMock: true };
}

const YEAR = 2026;
const EUR = 'EUR';

/** Alle Beträge in Cent, wie im Backend. */
const c = (euro: number) => Math.round(euro * 100);

/**
 * Ein Hash, der wie ein Hash aussieht und reproduzierbar bleibt.
 * Für Screenshots zählt nur die Gestalt: 64 Hexstellen, immer dieselben.
 */
function fakeHash(seed: string): string {
  let h1 = 0x811c9dc5;
  let out = '';
  for (let round = 0; round < 8; round++) {
    for (let i = 0; i < seed.length; i++) {
      h1 ^= seed.charCodeAt(i) + round * 31;
      h1 = Math.imul(h1, 0x01000193) >>> 0;
    }
    out += h1.toString(16).padStart(8, '0');
  }
  return out.slice(0, 64);
}

const later = <T>(value: T, ms = 0): Promise<T> =>
  new Promise((resolve) => setTimeout(() => resolve(value), ms));

// -------------------------------------------------------------------------
// Mandanten, Einrichtung, Stammdaten

const TENANTS = [
  {
    id: 'nordlicht',
    name: 'Nordlicht Systeme GmbH',
    dataDir: '/Users/anna/Buchfink/nordlicht',
    createdAt: '2024-01-08T09:12:00Z',
  },
  {
    id: 'werkbank',
    name: 'Werkbank Ost UG (haftungsbeschränkt)',
    dataDir: '/Users/anna/Buchfink/werkbank',
    createdAt: '2025-03-02T11:40:00Z',
  },
];

const SETTINGS = {
  companyName: 'Nordlicht Systeme GmbH',
  legalForm: 'GmbH',
  fiscalYear: YEAR,
  fiscalYearStartMonth: 1,
  taxNumber: '27/123/45678',
  vatId: 'DE312345678',
  taxOffice: 'Finanzamt Hamburg-Nord',
  iban: 'DE02120300000000202051',
  bic: 'BYLADEM1001',
  bankName: 'Deutsche Kreditbank',
  street: 'Hafenstraße 14',
  zipCity: '20359 Hamburg',
  country: 'DE',
  currency: EUR,
  skr: 'SKR04',
  vatPeriod: 'quarter',
  taxationType: 'Soll',
};

// -------------------------------------------------------------------------
// Konten (SKR04 2026)

interface MockAccount {
  number: string;
  name: string;
  type: string;
  kontenklasse: number;
  kontenklasseName: string;
  balanceSide: string;
  statementType: string;
  debit: number;
  credit: number;
  bookings: number;
}

const KLASSEN: Record<number, string> = {
  0: 'Anlagevermögenskonten',
  1: 'Umlaufvermögenskonten',
  2: 'Eigenkapitalkonten / Fremdkapitalkonten',
  3: 'Fremdkapitalkonten',
  4: 'Betriebliche Erträge',
  5: 'Betriebliche Aufwendungen (Material/Fremdleistungen)',
  6: 'Betriebliche Aufwendungen (Personal/AfA/Sonstige)',
  7: 'Weitere Erträge und Aufwendungen (Finanz/Steuern)',
};

const RAW_ACCOUNTS: MockAccount[] = [
  // Aktiva
  a('0520', 'Pkw', 'asset', 0, 'Aktiva', 'Bilanz', 24500, 0, 2),
  a('1200', 'Forderungen aus Lieferungen und Leistungen', 'asset', 1, 'Aktiva', 'Bilanz', 554400, 517699, 46),
  a('1406', 'Abziehbare Vorsteuer 19 %', 'asset', 1, 'Aktiva', 'Bilanz', 22898.8, 17613, 88),
  a('1600', 'Kasse', 'asset', 1, 'Aktiva', 'Bilanz', 1800, 1180, 11),
  a('1800', 'Bank', 'asset', 1, 'Aktiva', 'Bilanz', 492160, 319481.26, 164),
  a('1900', 'Aktive Rechnungsabgrenzung', 'asset', 1, 'Aktiva', 'Bilanz', 2400, 0, 1),
  // Passiva
  a('2900', 'Gezeichnetes Kapital', 'equity', 2, 'Passiva', 'Bilanz', 0, 25000, 1),
  a('2970', 'Gewinnvortrag vor Verwendung', 'equity', 2, 'Passiva', 'Bilanz', 0, 61230, 1),
  a('3300', 'Verbindlichkeiten aus Lieferungen und Leistungen', 'liability', 3, 'Passiva', 'Bilanz', 415856.46, 421290, 74),
  a('3806', 'Umsatzsteuer 19 %', 'liability', 3, 'Passiva', 'Bilanz', 68400, 92378, 52),
  // Erträge
  a('4400', 'Erlöse 19 % USt', 'revenue', 4, 'GuV', 'GuV', 0, 486200, 46),
  a('4830', 'Sonstige betriebliche Erträge', 'revenue', 4, 'GuV', 'GuV', 0, 3750, 3),
  // Aufwendungen
  a('5906', 'Fremdleistungen', 'expense', 5, 'GuV', 'GuV', 74300, 0, 24),
  a('6020', 'Gehälter', 'expense', 6, 'GuV', 'GuV', 213600, 0, 8),
  a('6260', 'Sofortabschreibungen geringwertiger Wirtschaftsgüter', 'expense', 6, 'GuV', 'GuV', 4180, 0, 6),
  a('6310', 'Miete (unbewegliche Wirtschaftsgüter)', 'expense', 6, 'GuV', 'GuV', 28800, 0, 8),
  a('6495', 'Wartungskosten für Hard- und Software', 'expense', 6, 'GuV', 'GuV', 12470, 0, 14),
  a('6600', 'Werbekosten', 'expense', 6, 'GuV', 'GuV', 9850, 0, 9),
  a('6650', 'Reisekosten Arbeitnehmer', 'expense', 6, 'GuV', 'GuV', 6240, 0, 12),
  a('6805', 'Telefon', 'expense', 6, 'GuV', 'GuV', 2160, 0, 8),
  a('6815', 'Bürobedarf', 'expense', 6, 'GuV', 'GuV', 3420, 0, 17),
  a('6825', 'Rechts- und Beratungskosten', 'expense', 6, 'GuV', 'GuV', 7900, 0, 5),
  a('6855', 'Nebenkosten des Geldverkehrs', 'expense', 6, 'GuV', 'GuV', 486, 0, 8),
];

/** Unbebuchte Konten, damit der Kontenrahmen nicht aus lauter Salden besteht. */
const UNUSED_ACCOUNTS: MockAccount[] = [
  a('0440', 'Technische Anlagen und Maschinen', 'asset', 0, 'Aktiva', 'Bilanz', 0, 0, 0),
  a('1370', 'Durchlaufende Posten', 'asset', 1, 'Aktiva', 'Bilanz', 0, 0, 0),
  a('1401', 'Abziehbare Vorsteuer 7 %', 'asset', 1, 'Aktiva', 'Bilanz', 0, 0, 0),
  a('1408', 'Abziehbare Vorsteuer nach § 13b UStG', 'asset', 1, 'Aktiva', 'Bilanz', 0, 0, 0),
  a('1460', 'Geldtransit', 'asset', 1, 'Aktiva', 'Bilanz', 0, 0, 0),
  a('3801', 'Umsatzsteuer 7 %', 'liability', 3, 'Passiva', 'Bilanz', 0, 0, 0),
  a('3835', 'Umsatzsteuer nach § 13b UStG', 'liability', 3, 'Passiva', 'Bilanz', 0, 0, 0),
  a('3900', 'Passive Rechnungsabgrenzung', 'liability', 3, 'Passiva', 'Bilanz', 0, 0, 0),
  a('4736', 'Gewährte Skonti 19 % USt', 'revenue', 4, 'GuV', 'GuV', 0, 0, 0),
  a('5400', 'Wareneingang', 'expense', 5, 'GuV', 'GuV', 0, 0, 0),
  a('5736', 'Erhaltene Skonti 19 % Vorsteuer', 'expense', 5, 'GuV', 'GuV', 0, 0, 0),
  a('6300', 'Sonstige betriebliche Aufwendungen', 'expense', 6, 'GuV', 'GuV', 0, 0, 0),
  a('6400', 'Versicherungen', 'expense', 6, 'GuV', 'GuV', 0, 0, 0),
  a('6420', 'Beiträge', 'expense', 6, 'GuV', 'GuV', 0, 0, 0),
  a('6640', 'Bewirtungskosten', 'expense', 6, 'GuV', 'GuV', 0, 0, 0),
  a('6644', 'Nicht abzugsfähige Bewirtungskosten', 'expense', 6, 'GuV', 'GuV', 0, 0, 0),
  a('6821', 'Fortbildungskosten', 'expense', 6, 'GuV', 'GuV', 0, 0, 0),
  a('6827', 'Abschluss- und Prüfungskosten', 'expense', 6, 'GuV', 'GuV', 0, 0, 0),
  a('7300', 'Zinsen und ähnliche Aufwendungen', 'expense', 7, 'GuV', 'GuV', 0, 0, 0),
];

function a(
  number: string,
  name: string,
  type: string,
  kontenklasse: number,
  balanceSide: string,
  statementType: string,
  debit: number,
  credit: number,
  bookings: number,
): MockAccount {
  return {
    number,
    name,
    type,
    kontenklasse,
    kontenklasseName: KLASSEN[kontenklasse] ?? '',
    balanceSide,
    statementType,
    debit,
    credit,
    bookings,
  };
}

const ACCOUNTS = [...RAW_ACCOUNTS, ...UNUSED_ACCOUNTS]
  .map((m, index) => ({
    id: index + 1,
    number: m.number,
    name: m.name,
    type: m.type,
    category: m.kontenklasseName,
    subcategory: '',
    kontenklasse: m.kontenklasse,
    kontenklasseName: m.kontenklasseName,
    positionId: '',
    posten: '',
    balanceSide: m.balanceSide,
    hgbCode: '',
    statementType: m.statementType,
    taxRate: 0,
    hauptfunktion: '',
    hauptfunktionDesc: '',
    zusatzfunktion: '',
    zusatzfunktionDesc: '',
    abschlusszweck: '',
    isRange: false,
    rangeStart: '',
    rangeEnd: '',
    isReserved: false,
    description: '',
    isActive: true,
    debitSum: c(m.debit),
    creditSum: c(m.credit),
    balance: Math.abs(c(m.debit) - c(m.credit)),
    bookingsCount: m.bookings,
  }))
  .sort((x, y) => x.number.localeCompare(y.number));

const accountName = (number: string) => ACCOUNTS.find((x) => x.number === number)?.name ?? '';

// -------------------------------------------------------------------------
// Journal

type MockLine = [side: 'S' | 'H', account: string, euro: number];

interface MockEntry {
  n: number;
  number: string;
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
  lines: MockLine[];
}

const RAW_ENTRIES: MockEntry[] = [
  {
    n: 41, number: 'B-2026-0041', date: '2026-08-03', documentDate: '2026-08-03',
    description: 'Rahmenvertrag Portalpflege, Rechnung RE-2026-0118',
    source: 'invoice', documentNumber: 'RE-2026-0118', contactId: 1,
    lines: [['S', '1200', 34510], ['H', '4400', 29000], ['H', '3806', 5510]],
  },
  {
    n: 42, number: 'B-2026-0042', date: '2026-08-04', documentDate: '2026-08-01',
    description: 'Cloud-Hosting August 2026',
    source: 'receipt', documentNumber: 'ER-2026-0212', contactId: 11, receiptId: 4,
    lines: [['S', '5906', 1480], ['S', '1406', 281.2], ['H', '3300', 1761.2]],
  },
  {
    n: 43, number: 'B-2026-0043', date: '2026-08-06', documentDate: '2026-07-31',
    description: 'Gehälter Juli 2026', source: 'manual',
    lines: [['S', '6020', 26700], ['H', '1800', 26700]],
  },
  {
    n: 44, number: 'B-2026-0044', date: '2026-08-07', documentDate: '2026-08-07',
    description: 'Zahlungseingang zu RE-2026-0112', source: 'payment', contactId: 2,
    lines: [['S', '1800', 18921], ['H', '1200', 18921]],
  },
  {
    n: 45, number: 'B-2026-0045', date: '2026-08-10', documentDate: '2026-08-01',
    description: 'Büromiete August 2026', source: 'receipt', documentNumber: 'ER-2026-0218', contactId: 12,
    lines: [['S', '6310', 3600], ['H', '1800', 3600]],
  },
  {
    n: 46, number: 'B-2026-0046', date: '2026-08-11', documentDate: '2026-08-10',
    description: 'Notebooks, 4 Stück (geringwertige Wirtschaftsgüter)',
    source: 'receipt', documentNumber: 'ER-2026-0231', contactId: 13, receiptId: 1,
    lines: [['S', '6260', 2716], ['S', '1406', 516.04], ['H', '3300', 3232.04]],
  },
  {
    n: 47, number: 'B-2026-0047', date: '2026-08-12', documentDate: '2026-08-09',
    description: 'Reisekosten Fachmesse Berlin', source: 'receipt', documentNumber: 'ER-2026-0233', contactId: 14,
    lines: [['S', '6650', 842], ['S', '1406', 159.98], ['H', '1800', 1001.98]],
  },
  {
    n: 48, number: 'B-2026-0048', date: '2026-08-13', documentDate: '2026-08-12',
    description: 'Wartungsvertrag Datenbank, Q3 2026',
    source: 'receipt', documentNumber: 'ER-2026-0236', contactId: 11,
    lines: [['S', '6495', 1240], ['S', '1406', 235.6], ['H', '3300', 1475.6]],
  },
  {
    n: 49, number: 'B-2026-0049', date: '2026-08-14', documentDate: '2026-08-12',
    description: 'Generalumkehr zu B-2026-0048',
    source: 'receipt', documentNumber: 'ER-2026-0236', contactId: 11,
    kind: 'reversal', reversalOfId: 48,
    reversalReason: 'Aufwand gehört auf 5906 Fremdleistungen, nicht auf 6495.',
    lines: [['S', '3300', 1475.6], ['H', '6495', 1240], ['H', '1406', 235.6]],
  },
  {
    n: 50, number: 'B-2026-0050', date: '2026-08-14', documentDate: '2026-08-12',
    description: 'Wartungsvertrag Datenbank, Q3 2026 (Neubuchung)',
    source: 'receipt', documentNumber: 'ER-2026-0236', contactId: 11,
    lines: [['S', '5906', 1240], ['S', '1406', 235.6], ['H', '3300', 1475.6]],
  },
  {
    n: 51, number: 'B-2026-0051', date: '2026-08-17', documentDate: '2026-08-17',
    description: 'Wartungspauschale Q3, Rechnung RE-2026-0119',
    source: 'invoice', documentNumber: 'RE-2026-0119', contactId: 1,
    lines: [['S', '1200', 8211], ['H', '4400', 6900], ['H', '3806', 1311]],
  },
  {
    n: 52, number: 'B-2026-0052', date: '2026-08-18', documentDate: '2026-08-15',
    description: 'Telefon und Internet August 2026',
    source: 'receipt', documentNumber: 'ER-2026-0238', contactId: 15, receiptId: 3,
    lines: [['S', '6805', 270], ['S', '1406', 51.3], ['H', '1800', 321.3]],
  },
  {
    n: 53, number: 'B-2026-0053', date: '2026-08-20', documentDate: '2026-08-20',
    description: 'Zahlung Lieferantenrechnung ER-2026-0212', source: 'payment', contactId: 11,
    lines: [['S', '3300', 1761.2], ['H', '1800', 1761.2]],
  },
  {
    n: 54, number: 'B-2026-0054', date: '2026-08-21', documentDate: '2026-08-19',
    description: 'Werbekampagne Fachportal', source: 'receipt', documentNumber: 'ER-2026-0240', contactId: 16,
    lines: [['S', '6600', 1850], ['S', '1406', 351.5], ['H', '3300', 2201.5]],
  },
  {
    n: 55, number: 'B-2026-0055', date: '2026-08-24', documentDate: '2026-08-24',
    description: 'Kontoführung August 2026', source: 'receipt', documentNumber: 'ER-2026-0241', contactId: 17,
    lines: [['S', '6855', 62], ['H', '1800', 62]],
  },
];

let previous = fakeHash('genesis');
const ENTRIES = RAW_ENTRIES.map((e) => {
  const entryHash = fakeHash(e.number + previous);
  const entry = {
    id: e.n,
    fiscalYear: YEAR,
    entryNumber: e.number,
    bookingDate: e.date,
    documentDate: e.documentDate,
    serviceDateFrom: e.documentDate,
    serviceDateTo: e.documentDate,
    valueDate: e.date,
    description: e.description,
    source: e.source,
    documentNumber: e.documentNumber,
    receiptId: e.receiptId,
    receiptHash: e.receiptId ? fakeHash(`receipt-${e.receiptId}`) : undefined,
    taxTreatment: 'domestic',
    contactId: e.contactId,
    kind: e.kind ?? 'normal',
    reversalOfId: e.reversalOfId,
    reversalReason: e.reversalReason,
    currency: EUR,
    exchangeRateMicros: 1_000_000,
    postingRuleVersion: '2026.1',
    lines: e.lines.map((line, i) => ({
      id: e.n * 100 + i,
      entryId: e.n,
      position: i + 1,
      side: line[0],
      amount: c(line[2]),
      account: line[1],
      accountName: accountName(line[1]),
      taxKey: line[1] === '1406' || line[1] === '3806' ? '9' : undefined,
    })),
    previousHash: previous,
    entryHash,
    createdAt: `${e.date}T09:24:00Z`,
  };
  previous = entryHash;
  return entry;
});

const CHAIN_HEAD = previous;

// -------------------------------------------------------------------------
// Kontakte, offene Posten, Bank

const CONTACTS = [
  contact(1, 'customer', '10001', 'Nordwind Handels GmbH', 'Am Sandtorkai 3, 20457 Hamburg', 'DE289512347', 8211, 14),
  contact(2, 'customer', '10002', 'Elbtal Logistik KG', 'Billstraße 88, 20539 Hamburg', 'DE274119803', 24920, 30),
  contact(3, 'customer', '10003', 'Werft & Co. KG', 'Steinwerder 12, 20457 Hamburg', 'DE301447221', 3570, 14),
  contact(4, 'customer', '10004', 'Marschland Energie AG', 'Deichstraße 5, 25813 Husum', 'DE256330914', 0, 30),
  contact(11, 'vendor', '70011', 'Hanse Cloud Services GmbH', 'Kehrwieder 9, 20457 Hamburg', 'DE317744002', 0, 14),
  contact(13, 'vendor', '70013', 'Techpartner Nord GmbH', 'Ruhrstraße 42, 22761 Hamburg', 'DE263558107', 3232.04, 14),
  contact(16, 'vendor', '70016', 'Fachportal Media GmbH', 'Kaiser-Wilhelm-Straße 4, 20355 Hamburg', 'DE298001554', 2201.5, 21),
];

function contact(
  id: number,
  type: string,
  ledgerAccount: string,
  name: string,
  address: string,
  vatId: string,
  openEuro: number,
  terms: number,
) {
  return {
    id,
    type,
    ledgerAccount,
    name,
    company: name,
    email: `buchhaltung@${name.split(' ')[0].toLowerCase().replace(/[^a-z]/g, '')}.example`,
    address,
    taxId: '',
    vatId,
    countryCode: 'DE',
    iban: '',
    bic: '',
    paymentTermsDays: terms,
    isPrivate: false,
    isSmallBusiness: false,
    openAmount: c(openEuro),
    createdAt: '2024-02-11T08:00:00Z',
  };
}

const OPEN_ITEMS = [
  openItem(51, 'B-2026-0051', 1, 'Nordwind Handels GmbH', 'customer', '10001', 'RE-2026-0119', '2026-08-17', '2026-08-31', 8211, 0),
  openItem(38, 'B-2026-0038', 2, 'Elbtal Logistik KG', 'customer', '10002', 'RE-2026-0117', '2026-08-06', '2026-09-05', 24920, 0),
  openItem(36, 'B-2026-0036', 3, 'Werft & Co. KG', 'customer', '10003', 'RE-2026-0116', '2026-07-28', '2026-08-27', 7140, 3570),
  openItem(46, 'B-2026-0046', 13, 'Techpartner Nord GmbH', 'vendor', '70013', 'ER-2026-0231', '2026-08-10', '2026-08-30', 3232.04, 0),
  openItem(54, 'B-2026-0054', 16, 'Fachportal Media GmbH', 'vendor', '70016', 'ER-2026-0240', '2026-08-19', '2026-09-03', 2201.5, 0),
];

function openItem(
  entryId: number,
  entryNumber: string,
  contactId: number,
  contactName: string,
  contactType: string,
  ledgerAccount: string,
  documentNumber: string,
  documentDate: string,
  dueDate: string,
  grossEuro: number,
  settledEuro: number,
) {
  return {
    entryId,
    entryNumber,
    contactId,
    contactName,
    contactType,
    ledgerAccount,
    documentNumber,
    documentDate,
    dueDate,
    grossAmount: c(grossEuro),
    settledAmount: c(settledEuro),
    openAmount: c(grossEuro - settledEuro),
    taxRate: 1900,
    taxTreatment: 'domestic',
  };
}

const BANK_TX = [
  tx(1, '2026-08-25', 8211, 'Nordwind Handels GmbH', 'DE44100500001234567890', 'RE-2026-0119 Wartungspauschale Q3', 'unmatched'),
  tx(2, '2026-08-25', -2201.5, 'Fachportal Media GmbH', 'DE18500105172345678901', 'Rechnung 2026-8814 Kampagne August', 'unmatched'),
  tx(3, '2026-08-24', 12460, 'Elbtal Logistik KG', 'DE72200400003456789012', 'RE-2026-0117 Teilzahlung', 'unmatched'),
  tx(4, '2026-08-24', -3232.04, 'Techpartner Nord GmbH', 'DE93370400444567890123', 'ER-2026-0231 Notebooks, Zahlung nach Skontofrist', 'unmatched'),
  tx(5, '2026-08-21', -890, 'Stadtwerke Nord AöR', 'DE29200505501122334455', 'Abschlag Strom 08/2026, Kundennummer 4471-92', 'unmatched'),
  tx(6, '2026-08-20', 3570, 'Werft & Co. KG', 'DE61200800005678901234', 'RE-2026-0116 Restzahlung', 'unmatched'),
  tx(7, '2026-08-18', -321.3, 'Telekommunikation Nord GmbH', 'DE05300606016789012345', 'ER-2026-0238 Telefon August', 'matched'),
  tx(8, '2026-08-20', -1761.2, 'Hanse Cloud Services GmbH', 'DE55201207007890123456', 'ER-2026-0212 Cloud-Hosting August', 'matched'),
];

function tx(
  id: number,
  bookingDate: string,
  euro: number,
  counterpartyName: string,
  counterpartyIban: string,
  remittanceInfo: string,
  matchStatus: string,
) {
  return {
    id,
    fiscalYear: YEAR,
    accountIban: SETTINGS.iban,
    bookingDate,
    valueDate: bookingDate,
    amount: c(euro),
    currency: EUR,
    counterpartyName,
    counterpartyIban,
    remittanceInfo,
    endToEndId: `NOTPROVIDED-${id}`,
    matchStatus,
    ledgerAccount: '1800',
    matchedAmount: matchStatus === 'matched' ? c(Math.abs(euro)) : 0,
  };
}

// -------------------------------------------------------------------------
// Belege

const RECEIPTS = [
  {
    id: 1,
    fiscalYear: YEAR,
    receiptNumber: 'BE-2026-0231',
    direction: 'incoming',
    status: 'filed',
    files: [
      file(1, 1, 'original', 'ER-2026-0231_Techpartner-Nord.pdf', 'application/pdf', 184_213, false),
      file(2, 1, 'structured', 'factur-x.xml', 'application/xml', 12_884, true),
    ],
    receiptHash: fakeHash('receipt-1'),
    receivedAt: '2026-08-10',
    receivedVia: 'E-Mail',
    detectedFormat: 'ZUGFeRD / Factur-X',
    detectedProfile: 'EN 16931 (Comfort)',
    validatedAt: '2026-08-10T07:41:00Z',
    validationRuleset: 'EN 16931 / CIUS XRechnung',
    validationVersion: '3.0.2',
    validationCoverage: 'partial',
    validationErrors: 0,
    validationFindings: JSON.stringify([
      {
        rule: 'BR-DE-15',
        severity: 'warning',
        terms: ['BT-13'],
        message: 'Die Bestellreferenz des Käufers fehlt. Für den Vorsteuerabzug ist sie nicht nötig.',
      },
    ]),
    createdAt: '2026-08-10T07:40:00Z',
    updatedAt: '2026-08-10T07:41:00Z',
  },
  receipt(2, 'BE-2026-0230', 'filed', 'Stadtwerke_Abschlag_08-2026.pdf', '2026-08-21'),
  receipt(3, 'BE-2026-0229', 'sealed', 'ER-2026-0238_Telefon-August.pdf', '2026-08-15', 52),
  receipt(4, 'BE-2026-0228', 'sealed', 'ER-2026-0212_Cloud-Hosting.pdf', '2026-08-01', 42),
  receipt(5, 'BE-2026-0227', 'sealed', 'ER-2026-0218_Miete-August.pdf', '2026-08-01', 45),
  {
    ...receipt(6, 'BE-2026-0226', 'discarded', 'ER-2026-0209_doppelt.pdf', '2026-07-29'),
    discardReason: 'Doppelt eingegangen, gebucht ist BE-2026-0225.',
  },
];

function receipt(
  id: number,
  receiptNumber: string,
  status: string,
  fileName: string,
  receivedAt: string,
  journalEntryId?: number,
) {
  return {
    id,
    fiscalYear: YEAR,
    receiptNumber,
    direction: 'incoming',
    status,
    files: [file(id * 10, id, 'original', fileName, 'application/pdf', 96_400 + id * 517, false)],
    receiptHash: fakeHash(`receipt-${id}`),
    receivedAt,
    receivedVia: 'E-Mail',
    journalEntryId,
    validationErrors: 0,
    createdAt: `${receivedAt}T08:00:00Z`,
    updatedAt: `${receivedAt}T08:00:00Z`,
  };
}

function file(
  id: number,
  receiptId: number,
  role: string,
  fileName: string,
  mimeType: string,
  size: number,
  derived: boolean,
) {
  return {
    id,
    receiptId,
    position: 1,
    role,
    fileName,
    mimeType,
    size,
    sha256: fakeHash(fileName),
    derived,
    storedPath: `belege/2026/${fileName}`,
    createdAt: '2026-08-10T07:40:00Z',
  };
}

/**
 * Die Belegvorschau. Im Betrieb kommt sie aus der Belegablage; für den
 * Screenshot-Lauf legt `shoot.mjs` das Bild vorab unter
 * `window.__receiptPreview` ab.
 */
const receiptPreview = () =>
  (typeof window !== 'undefined' && (window as any).__receiptPreview) || '';

// -------------------------------------------------------------------------
// Kontierung

const POSTING_GROUPS = [
  group('wareneingang', 'Wareneingang', 'Material & Fremdleistungen', '5400'),
  group('fremdleistungen', 'Fremdleistungen', 'Material & Fremdleistungen', '5906'),
  group('miete', 'Miete & Pacht', 'Raumkosten', '6310', 0, 'exempt'),
  group('raumkosten', 'Nebenkosten & sonstige Raumkosten', 'Raumkosten', '6345'),
  group('buerobedarf', 'Bürobedarf', 'Verwaltung', '6815'),
  group('telefon', 'Telefon & Internet', 'Verwaltung', '6805'),
  group('software', 'Software & IT-Wartung', 'Verwaltung', '6495'),
  group('beratung', 'Rechts- & Beratungskosten', 'Verwaltung', '6825'),
  group('versicherungen', 'Versicherungen', 'Verwaltung', '6400', 0, 'exempt'),
  group('gehaelter', 'Gehälter', 'Personal', '6020', 0, 'not_taxable'),
  group('reisekosten', 'Reisekosten', 'Personal', '6650'),
  group('werbung', 'Werbekosten', 'Vertrieb', '6600'),
  group('gwg', 'Geringwertige Wirtschaftsgüter (Sofortabschreibung)', 'Anlagen', '6260'),
];

function group(
  key: string,
  label: string,
  category: string,
  account: string,
  defaultRate = 1900,
  defaultTreatment?: string,
) {
  return { key, label, category, direction: 'incoming', account, defaultRate, defaultTreatment };
}

const TAX_TREATMENTS = [
  treatment('domestic', 'Inland, steuerpflichtig', 'Der Regelfall: deutsche Rechnung mit ausgewiesener Umsatzsteuer.', true, false),
  treatment('reverse_charge', 'Reverse Charge (§ 13b UStG)', 'Die Steuer schuldet der Leistungsempfänger. Gezahlt wird netto.', true, true),
  treatment('intra_community_acquisition', 'Innergemeinschaftlicher Erwerb', 'Erwerb aus einem anderen EU-Mitgliedstaat mit Umsatzsteuer-Identifikationsnummer.', true, true),
  treatment('exempt', 'Steuerfrei', 'Ohne Umsatzsteuer, etwa Miete oder Versicherung.', false, false),
  treatment('not_taxable', 'Nicht steuerbar', 'Kein Leistungsaustausch, etwa Gehälter oder Beiträge.', false, false),
];

function treatment(
  key: string,
  label: string,
  hint: string,
  requiresRate: boolean,
  requiresVatId: boolean,
) {
  return { treatment: key, label, hint, direction: 'incoming', requiresRate, requiresVatId };
}

const DIFFERENCE_KINDS = [
  { kind: 'none', label: 'Keine Differenz', hint: 'Der Zahlbetrag entspricht dem offenen Posten.' },
  { kind: 'skonto', label: 'Skonto', hint: 'Entgeltminderung; die Steuer wird nach § 17 UStG berichtigt.' },
  { kind: 'bank_fee', label: 'Bankgebühr', hint: 'Die Bank hat vom Betrag einbehalten.' },
  { kind: 'rounding', label: 'Rundungsdifferenz', hint: 'Centbetrag ohne eigene Aussage.' },
  { kind: 'currency', label: 'Kursdifferenz', hint: 'Unterschied aus der Umrechnung einer Fremdwährung.' },
];

// -------------------------------------------------------------------------
// Umsatzsteuer

/** Nettoerlöse zu 19 % je Monat, Januar bis August. */
const REVENUE_BY_MONTH = [52400, 58900, 61300, 57800, 63400, 66200, 59700, 66500];
/** Vorsteuerpflichtige Aufwendungen netto je Monat. */
const INPUT_BASE_BY_MONTH = [13800, 15100, 16400, 14200, 15900, 17300, 13600, 14220];

function monthsIn(from: string, to: string): number[] {
  if (!from || !to) return REVENUE_BY_MONTH.map((_, i) => i);
  const first = Number(from.slice(5, 7));
  const last = Number(to.slice(5, 7));
  const months: number[] = [];
  for (let m = first; m <= last; m++) if (m <= REVENUE_BY_MONTH.length) months.push(m - 1);
  return months;
}

function vatSummary(from: string, to: string) {
  const months = monthsIn(from, to);
  const net19 = months.reduce((sum, m) => sum + REVENUE_BY_MONTH[m], 0);
  const inputBase = months.reduce((sum, m) => sum + INPUT_BASE_BY_MONTH[m], 0);
  const outputTax = Math.round(c(net19) * 0.19);
  const inputTax = Math.round(c(inputBase) * 0.19);
  return {
    fiscalYear: YEAR,
    periodFrom: from,
    periodTo: to,
    taxableRevenue: net19 > 0 ? [{ rate: 1900, net: c(net19), tax: outputTax }] : [],
    exemptRevenue: 0,
    intraCommunitySupply: 0,
    export: 0,
    reverseChargeSupply: 0,
    outputTax,
    reverseChargeTax: 0,
    reverseChargeBase: 0,
    intraCommunityAcquisitionTax: 0,
    intraCommunityAcquisitionBase: 0,
    totalOwedTax: outputTax,
    inputTax,
    payable: outputTax - inputTax,
  };
}

// -------------------------------------------------------------------------
// Auswertung, Integrität, Protokoll

const SUMMARY = {
  totalRevenue: c(489950),
  totalExpenses: c(363406),
  netIncome: c(126544),
  bankBalance: c(172678.74),
  openReceivables: c(36701),
  openPayables: c(5433.54),
  cashflowHistory: REVENUE_BY_MONTH.map((net, i) => ({
    month: `${YEAR}-${String(i + 1).padStart(2, '0')}`,
    label: ['Jan', 'Feb', 'Mär', 'Apr', 'Mai', 'Jun', 'Jul', 'Aug'][i],
    inflow: c(net * 1.19),
    outflow: c(INPUT_BASE_BY_MONTH[i] * 1.19 + 26700),
    net: c(net * 1.19 - INPUT_BASE_BY_MONTH[i] * 1.19 - 26700),
  })),
};

const INTEGRITY = {
  isValid: true,
  totalEntries: 55,
  checkedEntries: 55,
  message: 'Alle 55 Buchungen sind unverändert. Die Hash-Kette ist lückenlos.',
  lastVerifiedHash: CHAIN_HEAD,
  checkedAt: '2026-08-26T08:12:00Z',
};

const AUDIT_LOGS = [
  log(214, '2026-08-24T09:31:00Z', 'journal.post', 'JournalEntry', 'B-2026-0055', 'Kontoführung August 2026, 62,00 €'),
  log(213, '2026-08-21T14:02:00Z', 'journal.post', 'JournalEntry', 'B-2026-0054', 'Werbekampagne Fachportal, 2.201,50 €'),
  log(212, '2026-08-20T11:18:00Z', 'payment.settle', 'JournalEntry', 'B-2026-0053', 'Offener Posten ER-2026-0212 ausgeglichen'),
  log(211, '2026-08-18T08:47:00Z', 'receipt.seal', 'Receipt', 'BE-2026-0229', 'Beleg gebucht und versiegelt'),
  log(210, '2026-08-14T16:05:00Z', 'journal.post', 'JournalEntry', 'B-2026-0050', 'Neubuchung nach Generalumkehr'),
  log(209, '2026-08-14T16:04:00Z', 'journal.reverse', 'JournalEntry', 'B-2026-0049', 'Generalumkehr zu B-2026-0048'),
  log(208, '2026-08-13T10:22:00Z', 'journal.post', 'JournalEntry', 'B-2026-0048', 'Wartungsvertrag Datenbank, 1.475,60 €'),
  log(207, '2026-08-12T09:14:00Z', 'contact.update', 'Contact', '70013', 'Zahlungsziel von 21 auf 14 Tage geändert'),
  log(206, '2026-08-11T07:58:00Z', 'bank.import', 'BankStatement', 'camt053-2026-08-11', '34 Umsätze aus CAMT.053 eingelesen'),
  log(205, '2026-08-10T07:41:00Z', 'receipt.file', 'Receipt', 'BE-2026-0231', 'ZUGFeRD-Beleg abgelegt, EN 16931 geprüft'),
];

function log(
  id: number,
  timestamp: string,
  action: string,
  entityType: string,
  entityId: string,
  details: string,
) {
  return {
    id,
    timestamp,
    action,
    entityType,
    entityId,
    details,
    previousHash: fakeHash(`log-${id - 1}`),
    entryHash: fakeHash(`log-${id}`),
  };
}

const FESTSCHREIBUNGEN = [
  {
    id: 3,
    fiscalYear: YEAR,
    periodType: 'quarter',
    periodLabel: 'Q2 2026',
    cutoffDate: '2026-06-30',
    chainHead: fakeHash('festschreibung-q2'),
    entryCount: 37,
    tsaName: 'freeTSA',
    tsaGenTime: '2026-07-10T06:15:00Z',
    timestampStatus: 'confirmed',
    createdAt: '2026-07-10T06:15:00Z',
  },
  {
    id: 2,
    fiscalYear: YEAR,
    periodType: 'quarter',
    periodLabel: 'Q1 2026',
    cutoffDate: '2026-03-31',
    chainHead: fakeHash('festschreibung-q1'),
    entryCount: 19,
    tsaName: 'freeTSA',
    tsaGenTime: '2026-04-09T05:52:00Z',
    timestampStatus: 'confirmed',
    createdAt: '2026-04-09T05:52:00Z',
  },
  {
    id: 1,
    fiscalYear: 2025,
    periodType: 'year',
    periodLabel: 'Geschäftsjahr 2025',
    cutoffDate: '2025-12-31',
    chainHead: fakeHash('festschreibung-2025'),
    entryCount: 148,
    tsaName: 'freeTSA',
    tsaGenTime: '2026-02-28T10:04:00Z',
    timestampStatus: 'confirmed',
    createdAt: '2026-02-28T10:04:00Z',
  },
];

/**
 * Ein Auszug der erzeugten E-Bilanz. Für den Screenshot reicht der Kopf mit
 * Kontexten, Einheiten und den ersten Positionen — die Gestalt der Datei ist
 * das, was die Seite zeigen soll.
 */
const XBRL = `<?xml version="1.0" encoding="UTF-8"?>
<xbrli:xbrl xmlns:xbrli="http://www.xbrl.org/2003/instance"
            xmlns:de-gaap-ci="http://www.xbrl.de/taxonomies/de-gaap-ci-2026-04-01"
            xmlns:de-gcd="http://www.xbrl.de/taxonomies/de-gcd-2026-04-01"
            xmlns:iso4217="http://www.xbrl.org/2003/iso4217"
            xmlns:link="http://www.xbrl.org/2003/linkbase"
            xmlns:xlink="http://www.w3.org/1999/xlink">
  <link:schemaRef xlink:type="simple"
                  xlink:href="http://www.xbrl.de/taxonomies/de-gaap-ci-2026-04-01-shell-fiscal.xsd"/>

  <xbrli:context id="D-2026">
    <xbrli:entity>
      <xbrli:identifier scheme="http://www.rzf-nrw.de/">27/123/45678</xbrli:identifier>
    </xbrli:entity>
    <xbrli:period>
      <xbrli:startDate>2026-01-01</xbrli:startDate>
      <xbrli:endDate>2026-12-31</xbrli:endDate>
    </xbrli:period>
  </xbrli:context>
  <xbrli:context id="I-2026-12-31">
    <xbrli:entity>
      <xbrli:identifier scheme="http://www.rzf-nrw.de/">27/123/45678</xbrli:identifier>
    </xbrli:entity>
    <xbrli:period><xbrli:instant>2026-12-31</xbrli:instant></xbrli:period>
  </xbrli:context>

  <xbrli:unit id="EUR">
    <xbrli:measure>iso4217:EUR</xbrli:measure>
  </xbrli:unit>

  <de-gcd:genInfo.company.name contextRef="D-2026">Nordlicht Systeme GmbH</de-gcd:genInfo.company.name>
  <de-gcd:genInfo.company.location.city contextRef="D-2026">Hamburg</de-gcd:genInfo.company.location.city>
  <de-gcd:genInfo.report.id.type contextRef="D-2026">Bilanz</de-gcd:genInfo.report.id.type>
  <de-gcd:genInfo.report.accountingStandard contextRef="D-2026">HGB</de-gcd:genInfo.report.accountingStandard>
  <de-gcd:genInfo.report.balSheetType contextRef="D-2026">Kontoform</de-gcd:genInfo.report.balSheetType>

  <!-- Bilanz, Aktiva -->
  <de-gaap-ci:bs.ass.fixAss.tan.othEquip contextRef="I-2026-12-31" unitRef="EUR" decimals="2">24500.00</de-gaap-ci:bs.ass.fixAss.tan.othEquip>
  <de-gaap-ci:bs.ass.currAss.receiv.trade contextRef="I-2026-12-31" unitRef="EUR" decimals="2">36701.00</de-gaap-ci:bs.ass.currAss.receiv.trade>
  <de-gaap-ci:bs.ass.currAss.cashEquiv contextRef="I-2026-12-31" unitRef="EUR" decimals="2">173298.74</de-gaap-ci:bs.ass.currAss.cashEquiv>
  <de-gaap-ci:bs.ass.deferralItem contextRef="I-2026-12-31" unitRef="EUR" decimals="2">2400.00</de-gaap-ci:bs.ass.deferralItem>

  <!-- Gewinn- und Verlustrechnung -->
  <de-gaap-ci:is.netIncome.regular.operatingTrade.sales contextRef="D-2026" unitRef="EUR" decimals="2">486200.00</de-gaap-ci:is.netIncome.regular.operatingTrade.sales>
  <de-gaap-ci:is.netIncome.regular.operatingTrade.otherOpInc contextRef="D-2026" unitRef="EUR" decimals="2">3750.00</de-gaap-ci:is.netIncome.regular.operatingTrade.otherOpInc>
  <de-gaap-ci:is.netIncome.regular.operatingTrade.staff.wages contextRef="D-2026" unitRef="EUR" decimals="2">213600.00</de-gaap-ci:is.netIncome.regular.operatingTrade.staff.wages>
  <de-gaap-ci:is.netIncome.regular.operatingTrade.otherOpExp contextRef="D-2026" unitRef="EUR" decimals="2">149806.00</de-gaap-ci:is.netIncome.regular.operatingTrade.otherOpExp>

  <!-- Kontennachweis, Auszug -->
  <de-gaap-ci:accountBalance.account.number contextRef="D-2026">4400</de-gaap-ci:accountBalance.account.number>
  <de-gaap-ci:accountBalance.account.name contextRef="D-2026">Erlöse 19 % USt</de-gaap-ci:accountBalance.account.name>
  <de-gaap-ci:accountBalance.account.balance contextRef="D-2026" unitRef="EUR" decimals="2">486200.00</de-gaap-ci:accountBalance.account.balance>
</xbrli:xbrl>
`;

const INVOICES = [
  invoice(1, 'RE-2026-0119', '2026-08-17', '2026-08-31', 1, 'Nordwind Handels GmbH', 6900, 'issued', 0),
  invoice(2, 'RE-2026-0118', '2026-08-03', '2026-08-17', 1, 'Nordwind Handels GmbH', 29000, 'paid', 34510),
  invoice(3, 'RE-2026-0117', '2026-08-06', '2026-09-05', 2, 'Elbtal Logistik KG', 20941.18, 'issued', 0),
  invoice(4, 'RE-2026-0116', '2026-07-28', '2026-08-27', 3, 'Werft & Co. KG', 6000, 'issued', 3570),
  invoice(5, 'RE-2026-0115', '2026-07-20', '2026-08-03', 4, 'Marschland Energie AG', 12400, 'paid', 14756),
  invoice(6, 'RE-2026-0114', '2026-07-14', '2026-07-28', 2, 'Elbtal Logistik KG', 4800, 'cancelled', 0),
];

function invoice(
  id: number,
  invoiceNumber: string,
  date: string,
  dueDate: string,
  contactId: number,
  contactName: string,
  netEuro: number,
  status: string,
  paidEuro: number,
) {
  const net = c(netEuro);
  const tax = Math.round(net * 0.19);
  return {
    id,
    fiscalYear: YEAR,
    invoiceNumber,
    date,
    serviceDateFrom: date,
    serviceDateTo: date,
    dueDate,
    contactId,
    contactName,
    items: [
      {
        position: 1,
        description: 'Wartung und Pflege des Kundenportals',
        quantityMilli: 1000,
        unit: 'Pauschale',
        unitPrice: net,
        taxRate: 1900,
      },
    ],
    taxTreatment: 'domestic',
    netAmount: net,
    taxAmount: tax,
    grossAmount: net + tax,
    currency: EUR,
    status,
    receiptId: id + 20,
    paidAmount: c(paidEuro),
    createdAt: `${date}T10:00:00Z`,
  };
}

// -------------------------------------------------------------------------
// Die Bridge


// -------------------------------------------------------------------------
// Anlagevermögen

/** Ein Anlagegut samt der Werte, die das Backend sonst aus den Bewegungen rechnet. */
function asset(
  id: number,
  inventoryNumber: string,
  name: string,
  assetClass: 'tangible' | 'financial' | 'intangible',
  account: string,
  accountName: string,
  depreciationAccount: string,
  acquisitionDate: string,
  cost: number,
  accumulated: number,
  yearAmount: number,
  dueAmount: number,
  method: 'linear' | 'degressive' | 'pool' | 'immediate' | 'none',
  usefulLifeMonths: number,
  extra: Record<string, unknown> = {},
) {
  const status =
    dueAmount > 0
      ? 'depreciate_due'
      : cost - accumulated === 0 && method !== 'none'
        ? 'fully_written'
        : 'active';
  return {
    id,
    inventoryNumber,
    name,
    class: assetClass,
    account,
    accountName,
    depreciationAccount,
    acquisitionDate,
    acquisitionCost: cost,
    method,
    usefulLifeMonths,
    createdAt: `${acquisitionDate}T09:00:00Z`,
    updatedAt: `${acquisitionDate}T09:00:00Z`,
    cost,
    accumulated,
    bookValue: cost - accumulated,
    yearAmount,
    dueAmount,
    status,
    ...extra,
  };
}

const ASSETS = [
  asset(1, 'AN-2024-0001', 'Büroeinrichtung Konferenzraum', 'tangible', '0650', 'Büroeinrichtung',
    '6220', '2024-06-01', c(9600), c(1169.23), 0, c(738.46), 'linear', 156),
  asset(2, 'AN-2025-0002', 'Pkw VW ID.4 · HH-NS 412', 'tangible', '0520', 'Pkw',
    '6222', '2025-03-15', c(42000), c(5833.33), 0, c(7000), 'linear', 72),
  asset(3, 'AN-2026-0003', 'CNC-Fräse Haas VF-2', 'tangible', '0440', 'Maschinen',
    '6220', '2026-02-01', c(24000), 0, 0, c(2750), 'linear', 96),
  asset(4, 'AN-2026-0004', 'Notebook Entwicklung', 'tangible', '0670',
    'Geringwertige Wirtschaftsgüter', '6260', '2026-04-20', c(780), c(780), c(780), 0, 'immediate', 0),
  asset(5, 'AN-2026-0005', 'Sammelposten 2026', 'tangible', '0675',
    'Wirtschaftsgüter (Sammelposten)', '6264', '2026-01-01', c(4200), 0, 0, c(840), 'pool', 0,
    { poolYear: YEAR }),
  asset(6, 'AN-2023-0006', 'Beteiligung Werftgrund GmbH', 'financial', '0850',
    'Beteiligungen an Kapitalgesellschaften', '', '2023-09-01', c(50000), c(5000), 0, 0, 'none', 0,
    { identifier: 'HRB 148223', holdingPermille: 260, taxPrivileged: true }),
  asset(7, 'AN-2024-0007', 'Festverzinsliche Anleihe 2031', 'financial', '0920',
    'Festverzinsliche Wertpapiere', '', '2024-11-04', c(25000), 0, 0, 0, 'none', 0,
    { identifier: 'DE000A2LQ5H0' }),
  asset(8, 'AN-2025-0008', 'ERP-Lizenz Warenwirtschaft', 'intangible', '0135', 'EDV-Software',
    '6200', '2025-07-01', c(12000), c(2000), 0, c(4000), 'linear', 36),
];

const ASSET_ACCOUNTS = [
  { number: '0135', name: 'EDV-Software', class: 'intangible', group: 'Konzessionen, Lizenzen und Software', depreciationAccount: '6200', depreciable: true },
  { number: '0150', name: 'Geschäfts- oder Firmenwert', class: 'intangible', group: 'Geschäfts- oder Firmenwert', depreciationAccount: '6205', depreciable: true, defaultUsefulLifeMonths: 180, usefulLifeSource: '§ 7 Abs. 1 Satz 3 EStG' },
  { number: '0440', name: 'Maschinen', class: 'tangible', group: 'Technische Anlagen und Maschinen', depreciationAccount: '6220', depreciable: true, usefulLifeSource: 'AfA-Tabelle des BMF' },
  { number: '0520', name: 'Pkw', class: 'tangible', group: 'Fahrzeuge', depreciationAccount: '6222', depreciable: true, defaultUsefulLifeMonths: 72, usefulLifeSource: 'AfA-Tabelle AV (BMF): Personenkraftwagen sechs Jahre' },
  { number: '0650', name: 'Büroeinrichtung', class: 'tangible', group: 'Betriebs- und Geschäftsausstattung', depreciationAccount: '6220', depreciable: true, defaultUsefulLifeMonths: 156 },
  { number: '0670', name: 'Geringwertige Wirtschaftsgüter', class: 'tangible', group: 'Geringwertige Wirtschaftsgüter', depreciationAccount: '6260', depreciable: true },
  { number: '0675', name: 'Wirtschaftsgüter (Sammelposten)', class: 'tangible', group: 'Geringwertige Wirtschaftsgüter', depreciationAccount: '6264', depreciable: true },
  { number: '0215', name: 'Unbebaute Grundstücke', class: 'tangible', group: 'Grundstücke und Bauten', depreciable: false, hint: 'Grund und Boden nutzt sich nicht ab.' },
  { number: '0850', name: 'Beteiligungen an Kapitalgesellschaften', class: 'financial', group: 'Anteile und Beteiligungen', depreciable: false },
  { number: '0920', name: 'Festverzinsliche Wertpapiere', class: 'financial', group: 'Wertpapiere', depreciable: false },
];

const ASSET_RULES = {
  fiscalYear: YEAR,
  gwgImmediateLimit: c(800),
  gwgRecordFrom: c(250),
  poolLowerLimit: c(250),
  poolUpperLimit: c(1000),
  poolYears: 5,
  degressiveWindows: [
    { From: '2025-07-01', Until: '2027-12-31', FactorTimes: 3, MaxPermille: 300, Source: '§ 7 Abs. 2 Sätze 1 und 2 EStG' },
  ],
  methods: [
    { method: 'linear', label: 'Linear (§ 7 Abs. 1 EStG)', classes: ['intangible', 'tangible'], hint: 'Gleichmäßig über die betriebsgewöhnliche Nutzungsdauer, zeitanteilig ab dem Anschaffungsmonat.' },
    { method: 'degressive', label: 'Degressiv (§ 7 Abs. 2 EStG)', classes: ['tangible'], hint: 'Vom Restbuchwert, höchstens das Dreifache des linearen Satzes und höchstens 30 %.' },
    { method: 'pool', label: 'Sammelposten (§ 6 Abs. 2a EStG)', classes: ['tangible'], hint: 'Ein Pool je Wirtschaftsjahr, aufgelöst mit je einem Fünftel.' },
    { method: 'immediate', label: 'Sofortabzug GWG (§ 6 Abs. 2 EStG)', classes: ['tangible'], hint: 'Voller Aufwand im Anschaffungsjahr.' },
    { method: 'none', label: 'Keine planmäßige Abschreibung', classes: ['intangible', 'tangible', 'financial'], hint: 'Für alles, was sich nicht abnutzt.' },
  ],
};

const DEPRECIATION_RUN = {
  fiscalYear: YEAR,
  bookingDate: `${YEAR}-12-31`,
  due: [
    { assetId: 1, inventoryNumber: 'AN-2024-0001', name: 'Büroeinrichtung Konferenzraum', account: '0650', expenseAccount: '6220', method: 'linear', rateLabel: '7,7 %', months: 12, planned: c(738.46), booked: 0, due: c(738.46), bookValueBefore: c(8430.77), bookValueAfter: c(7692.31) },
    { assetId: 2, inventoryNumber: 'AN-2025-0002', name: 'Pkw VW ID.4 · HH-NS 412', account: '0520', expenseAccount: '6222', method: 'linear', rateLabel: '16,7 %', months: 12, planned: c(7000), booked: 0, due: c(7000), bookValueBefore: c(36166.67), bookValueAfter: c(29166.67) },
    { assetId: 3, inventoryNumber: 'AN-2026-0003', name: 'CNC-Fräse Haas VF-2', account: '0440', expenseAccount: '6220', method: 'linear', rateLabel: '12,5 %', months: 11, planned: c(2750), booked: 0, due: c(2750), bookValueBefore: c(24000), bookValueAfter: c(21250), note: 'Zeitanteilig für 11 von 12 Monaten (§ 7 Abs. 1 Satz 4 EStG).' },
    { assetId: 5, inventoryNumber: 'AN-2026-0005', name: 'Sammelposten 2026', account: '0675', expenseAccount: '6264', method: 'pool', rateLabel: '1/5', months: 12, planned: c(840), booked: 0, due: c(840), bookValueBefore: c(4200), bookValueAfter: c(3360), note: 'Auflösung des Sammelpostens 2026 mit einem Fünftel, ohne Zeitanteil (§ 6 Abs. 2a Satz 2 EStG).' },
    { assetId: 8, inventoryNumber: 'AN-2025-0008', name: 'ERP-Lizenz Warenwirtschaft', account: '0135', expenseAccount: '6200', method: 'linear', rateLabel: '33,3 %', months: 12, planned: c(4000), booked: 0, due: c(4000), bookValueBefore: c(10000), bookValueAfter: c(6000) },
  ],
  total: c(15328.46),
};

function spiegelRow(
  assetClass: 'intangible' | 'tangible' | 'financial',
  account: string,
  accountName: string,
  assetCount: number,
  costOpening: number,
  additions: number,
  disposals: number,
  depreciationOpening: number,
  depreciationYear: number,
) {
  const costClosing = costOpening + additions - disposals;
  const depreciationClosing = depreciationOpening + depreciationYear;
  return {
    class: assetClass,
    account,
    accountName,
    assetCount,
    costOpening,
    additions,
    disposals,
    costClosing,
    depreciationOpening,
    depreciationYear,
    writeUpsYear: 0,
    depreciationDisposal: 0,
    depreciationClosing,
    bookValueOpening: costOpening - depreciationOpening,
    bookValueClosing: costClosing - depreciationClosing,
  };
}

const SPIEGEL_ROWS = [
  spiegelRow('intangible', '0135', 'EDV-Software', 1, c(12000), 0, 0, c(2000), c(4000)),
  spiegelRow('tangible', '0440', 'Maschinen', 1, 0, c(24000), 0, 0, c(2750)),
  spiegelRow('tangible', '0520', 'Pkw', 1, c(42000), 0, 0, c(5833.33), c(7000)),
  spiegelRow('tangible', '0650', 'Büroeinrichtung', 1, c(9600), 0, 0, c(1169.23), c(738.46)),
  spiegelRow('tangible', '0670', 'Geringwertige Wirtschaftsgüter', 1, 0, c(780), 0, 0, c(780)),
  spiegelRow('tangible', '0675', 'Wirtschaftsgüter (Sammelposten)', 1, 0, c(4200), 0, 0, c(840)),
  spiegelRow('financial', '0850', 'Beteiligungen an Kapitalgesellschaften', 1, c(50000), 0, 0, c(5000), 0),
  spiegelRow('financial', '0920', 'Festverzinsliche Wertpapiere', 1, c(25000), 0, 0, 0, 0),
];

function sumSpiegel(rows: typeof SPIEGEL_ROWS, accountName: string, assetClass = '' as any) {
  const total = rows.reduce(
    (acc, row) => ({
      ...acc,
      assetCount: acc.assetCount + row.assetCount,
      costOpening: acc.costOpening + row.costOpening,
      additions: acc.additions + row.additions,
      disposals: acc.disposals + row.disposals,
      costClosing: acc.costClosing + row.costClosing,
      depreciationOpening: acc.depreciationOpening + row.depreciationOpening,
      depreciationYear: acc.depreciationYear + row.depreciationYear,
      depreciationClosing: acc.depreciationClosing + row.depreciationClosing,
      bookValueOpening: acc.bookValueOpening + row.bookValueOpening,
      bookValueClosing: acc.bookValueClosing + row.bookValueClosing,
    }),
    {
      class: assetClass,
      account: '',
      accountName,
      assetCount: 0,
      costOpening: 0,
      additions: 0,
      disposals: 0,
      costClosing: 0,
      depreciationOpening: 0,
      depreciationYear: 0,
      writeUpsYear: 0,
      depreciationDisposal: 0,
      depreciationClosing: 0,
      bookValueOpening: 0,
      bookValueClosing: 0,
    },
  );
  return total;
}

const ANLAGENSPIEGEL = {
  fiscalYear: YEAR,
  rows: SPIEGEL_ROWS,
  totals: sumSpiegel(SPIEGEL_ROWS, 'Anlagevermögen gesamt'),
  classTotals: (['intangible', 'tangible', 'financial'] as const).map((assetClass) =>
    sumSpiegel(
      SPIEGEL_ROWS.filter((r) => r.class === assetClass),
      assetClass === 'intangible'
        ? 'Immaterielle Vermögensgegenstände'
        : assetClass === 'tangible'
          ? 'Sachanlagen'
          : 'Finanzanlagen',
      assetClass,
    ),
  ),
};

const ASSET_CANDIDATES = [
  {
    entryId: 41,
    entryNumber: `${YEAR}-000041`,
    bookingDate: `${YEAR}-05-12`,
    description: 'Hebebühne Werkstatt',
    account: '0440',
    accountName: 'Maschinen',
    amount: c(6800),
  },
];

const ASSET_SCHEDULES: Record<number, unknown[]> = {
  2: [
    { fiscalYear: 2025, months: 10, method: 'linear', rateLabel: '16,7 %', openingBookValue: c(42000), amount: c(5833.33), closingBookValue: c(36166.67), booked: c(5833.33), due: 0, status: 'gebucht', note: 'Zeitanteilig für 10 von 12 Monaten (§ 7 Abs. 1 Satz 4 EStG).' },
    { fiscalYear: 2026, months: 12, method: 'linear', rateLabel: '16,7 %', openingBookValue: c(36166.67), amount: c(7000), closingBookValue: c(29166.67), booked: 0, due: c(7000), status: 'offen' },
    { fiscalYear: 2027, months: 12, method: 'linear', rateLabel: '16,7 %', openingBookValue: c(29166.67), amount: c(7000), closingBookValue: c(22166.67), booked: 0, due: c(7000), status: 'geplant' },
    { fiscalYear: 2028, months: 12, method: 'linear', rateLabel: '16,7 %', openingBookValue: c(22166.67), amount: c(7000), closingBookValue: c(15166.67), booked: 0, due: c(7000), status: 'geplant' },
    { fiscalYear: 2029, months: 12, method: 'linear', rateLabel: '16,7 %', openingBookValue: c(15166.67), amount: c(7000), closingBookValue: c(8166.67), booked: 0, due: c(7000), status: 'geplant' },
    { fiscalYear: 2030, months: 12, method: 'linear', rateLabel: '16,7 %', openingBookValue: c(8166.67), amount: c(7000), closingBookValue: c(1166.67), booked: 0, due: c(7000), status: 'geplant' },
    { fiscalYear: 2031, months: 2, method: 'linear', rateLabel: 'Restwert', openingBookValue: c(1166.67), amount: c(1166.67), closingBookValue: 0, booked: 0, due: c(1166.67), status: 'geplant' },
  ],
};

const ASSET_MOVEMENTS: Record<number, unknown[]> = {
  2: [
    { id: 1, assetId: 2, kind: 'acquisition', date: '2025-03-15', fiscalYear: 2025, costAmount: c(42000), depreciationAmount: 0, entryNumber: '2025-000112', note: 'Zugang', createdAt: '2025-03-15T09:00:00Z' },
    { id: 2, assetId: 2, kind: 'depreciation', date: '2025-12-31', fiscalYear: 2025, costAmount: 0, depreciationAmount: c(5833.33), entryNumber: '2025-000488', note: 'AfA 2025', createdAt: '2025-12-31T09:00:00Z' },
  ],
};

const ASSET_NOTES: Record<string, string[]> = {
  tangible: [
    'Lineare Abschreibung über die betriebsgewöhnliche Nutzungsdauer (§ 7 Abs. 1 EStG), zeitanteilig ab dem Anschaffungsmonat (§ 7 Abs. 1 Satz 4 EStG).',
  ],
  financial: [
    'Finanzanlagen nutzen sich nicht ab und werden deshalb nicht planmäßig abgeschrieben. Sie stehen mit ihren Anschaffungskosten in der Bilanz, bis ein Grund für eine außerplanmäßige Abschreibung eintritt.',
    'Für Finanzanlagen gilt das gemilderte Niederstwertprinzip: bei voraussichtlich dauernder Wertminderung ist abzuschreiben, bei einer nicht dauernden darf abgeschrieben werden (§ 253 Abs. 3 Sätze 5 und 6 HGB).',
    'Fällt der Grund später weg, ist wieder zuzuschreiben — höchstens bis zu den Anschaffungskosten (§ 253 Abs. 5 Satz 1 HGB). Das ist ein Gebot, kein Wahlrecht.',
  ],
  intangible: [
    'Immaterielle Vermögensgegenstände des Anlagevermögens dürfen nur angesetzt werden, wenn sie entgeltlich erworben wurden (§ 248 Abs. 2 HGB).',
  ],
};

function assetDetail(id: number) {
  const found = ASSETS.find((a) => a.id === id)!;
  return {
    asset: found,
    schedule: ASSET_SCHEDULES[id] ?? [],
    movements: ASSET_MOVEMENTS[id] ?? [
      { id: 100 + id, assetId: id, kind: 'acquisition', date: found.acquisitionDate, fiscalYear: Number(found.acquisitionDate.slice(0, 4)), costAmount: found.cost, depreciationAmount: 0, note: 'Zugang', createdAt: `${found.acquisitionDate}T09:00:00Z` },
    ],
    notes: ASSET_NOTES[found.class] ?? [],
  };
}

function assetSummary(assetClass: string) {
  const list = ASSETS.filter((a) => !assetClass || a.class === assetClass);
  return {
    fiscalYear: YEAR,
    count: list.length,
    cost: list.reduce((sum, a) => sum + a.cost, 0),
    accumulated: list.reduce((sum, a) => sum + a.accumulated, 0),
    bookValue: list.reduce((sum, a) => sum + a.bookValue, 0),
    yearAmount: list.reduce((sum, a) => sum + a.yearAmount, 0),
    dueAmount: list.reduce((sum, a) => sum + a.dueAmount, 0),
    dueCount: list.filter((a) => a.dueAmount > 0).length,
  };
}


/** Die Einordnung nach § 6 Abs. 2 und 2a EStG, wie sie das Backend rechnet. */
function classifyAcquisition(netCost: number, selfUsable: boolean) {
  const limits = {
    immediate: c(800),
    recordFrom: c(250),
    poolLowerLimit: c(250),
    poolUpperLimit: c(1000),
  };
  if (!selfUsable) {
    return {
      recommended: 'activate',
      allowed: ['activate'],
      reason:
        'Das Wirtschaftsgut ist nicht selbständig nutzbar. Damit scheiden Sofortabzug und Sammelposten aus, ' +
        'unabhängig vom Betrag: § 6 Abs. 2 Satz 1 EStG setzt beim geringwertigen Wirtschaftsgut die selbständige ' +
        'Nutzbarkeit voraus.',
      limits,
    };
  }
  if (netCost <= limits.immediate) {
    return {
      recommended: 'immediate',
      allowed: ['immediate', 'pool', 'activate'],
      reason:
        'Bis 800,00 € netto ist der Sofortabzug nach § 6 Abs. 2 Satz 1 EStG möglich. Ab 250,00 € gehört das Gut ' +
        'in ein laufend geführtes Verzeichnis (§ 6 Abs. 2 Satz 4 EStG) — das Anlagenverzeichnis erfüllt das.',
      poolNote:
        'Das Wahlrecht zum Sammelposten gilt einheitlich für alle Wirtschaftsgüter eines Wirtschaftsjahres ' +
        '(§ 6 Abs. 2a Satz 5 EStG).',
      limits,
    };
  }
  if (netCost <= limits.poolUpperLimit) {
    return {
      recommended: 'pool',
      allowed: ['pool', 'activate'],
      reason:
        'Über 800,00 € ist der Sofortabzug ausgeschlossen. Bis 1.000,00 € netto kann das Gut in den Sammelposten ' +
        'des Wirtschaftsjahres eingestellt werden (§ 6 Abs. 2a Satz 1 EStG).',
      poolNote:
        'Das Wahlrecht zum Sammelposten gilt einheitlich für alle Wirtschaftsgüter eines Wirtschaftsjahres ' +
        '(§ 6 Abs. 2a Satz 5 EStG).',
      limits,
    };
  }
  return {
    recommended: 'activate',
    allowed: ['activate'],
    reason:
      'Über 1.000,00 € netto bleibt nur die Aktivierung: das Gut kommt auf ein Anlagekonto und wird über die ' +
      'betriebsgewöhnliche Nutzungsdauer abgeschrieben (§ 7 Abs. 1 EStG).',
    limits,
  };
}

/** Der Abgang, wie ihn das Backend vorrechnet: erst das Ergebnis, dann die Konten. */
function disposalPreview(request: any) {
  const found = ASSETS.find((a) => a.id === request.assetId)!;
  const catchUp = found.dueAmount;
  const bookValue = Math.max(found.bookValue - catchUp, 0);
  const proceeds = request.kind === 'scrapped' ? 0 : (request.proceeds ?? 0);
  const result = proceeds - bookValue;
  const isGain = result > 0;
  const tax =
    request.taxTreatment === 'domestic' ? Math.round((proceeds * (request.taxRate ?? 1900)) / 10000) : 0;
  const gross = proceeds + tax;
  const accounts = isGain
    ? { revenue: '4845', bookValue: '4855', explanation: 'Der Verkaufserlös liegt über dem Restbuchwert: es entsteht ein Buchgewinn. Der SKR04 führt Erlös und Restbuchwert dann unter den sonstigen betrieblichen Erträgen.' }
    : { revenue: '6885', bookValue: '6895', explanation: 'Der Verkaufserlös liegt unter dem Restbuchwert: es entsteht ein Buchverlust. Derselbe Vorgang läuft im SKR04 dann über die sonstigen betrieblichen Aufwendungen.' };

  const lines: any[] = [];
  if (proceeds > 0) {
    lines.push({ id: 1, position: 1, side: 'S', account: request.paymentAccount ?? '1800', accountName: 'Bank', amount: gross });
    lines.push({ id: 2, position: 2, side: 'H', account: accounts.revenue, accountName: isGain ? 'Erlöse aus Verkäufen Sachanlagevermögen 19 % USt (bei Buchgewinn)' : 'Erlöse aus Verkäufen Sachanlagevermögen 19 % USt (bei Buchverlust)', amount: proceeds });
    if (tax > 0) {
      lines.push({ id: 3, position: 3, side: 'H', account: '3806', accountName: 'Umsatzsteuer 19 %', amount: tax, taxKey: 'UST19', taxBase: proceeds });
    }
  }
  if (bookValue > 0) {
    lines.push({ id: 4, position: 4, side: 'S', account: accounts.bookValue, accountName: 'Anlagenabgänge Sachanlagen', amount: bookValue });
    lines.push({ id: 5, position: 5, side: 'H', account: found.account, accountName: found.accountName, amount: bookValue });
  }

  return {
    catchUpAmount: catchUp,
    catchUpLines: catchUp > 0
      ? [
          { id: 6, position: 1, side: 'S', account: found.depreciationAccount, accountName: 'Abschreibungen auf Fahrzeuge', amount: catchUp, text: 'AfA bis zum Abgangsmonat' },
          { id: 7, position: 2, side: 'H', account: found.account, accountName: found.accountName, amount: catchUp, text: 'AfA bis zum Abgangsmonat' },
        ]
      : [],
    bookValue,
    result,
    isGain,
    accounts,
    lines,
    gross,
    tax,
  };
}

const unsupported = (name: string) => () =>
  Promise.reject(new Error(`${name} ist in der Screenshot-Vorschau nicht verfügbar.`));

export const bridge = {
  // Mandanten
  GetTenants: () => later(TENANTS),
  GetActiveTenant: () => later(TENANTS[0]),
  SwitchTenant: () => later(undefined),
  CreateTenant: unsupported('CreateTenant'),
  ImportTenant: unsupported('ImportTenant'),
  DeleteTenant: unsupported('DeleteTenant'),
  IsLocked: () => later(false),

  // Einrichtung
  GetAppConfig: () =>
    later({
      tenants: TENANTS,
      activeTenantId: TENANTS[0].id,
      dataDir: TENANTS[0].dataDir,
      isConfigured: true,
      lastFiscalYear: YEAR,
    }),
  SetupApplication: unsupported('SetupApplication'),
  LoadExistingDatabase: unsupported('LoadExistingDatabase'),
  SelectDirectoryDialog: unsupported('SelectDirectoryDialog'),
  SelectDatabaseFileDialog: unsupported('SelectDatabaseFileDialog'),
  SelectRecoveryFileDialog: unsupported('SelectRecoveryFileDialog'),
  ExportRecoveryKey: unsupported('ExportRecoveryKey'),
  RecoverActiveTenantFromFile: unsupported('RecoverActiveTenantFromFile'),

  // Geschäftsjahr & Stammdaten
  GetFiscalYear: () => later(YEAR),
  SetFiscalYear: () => later(undefined),
  GetAvailableFiscalYears: () => later([2024, 2025, 2026]),
  GetCompanySettings: () => later(SETTINGS),
  UpdateCompanySettings: () => later(undefined),

  // Konten
  GetAccounts: () => later(ACCOUNTS),
  GetAccountByNumber: (number: string) => later(ACCOUNTS.find((x) => x.number === number)),
  GetAccountLedger: (accountNumber: string) => later(ledgerFor(accountNumber)),
  GetSuSaOverview: () => later(susa()),
  GetPaymentAccounts: () => later(ACCOUNTS.filter((x) => ['1600', '1800'].includes(x.number))),

  // Kontierung
  GetPostingGroups: () => later(POSTING_GROUPS),
  GetTaxTreatments: () => later(TAX_TREATMENTS),
  GetDifferenceKinds: () => later(DIFFERENCE_KINDS),

  // Journal
  GetJournalEntries: () => later(ENTRIES),
  GetAllJournalEntries: () => later(ENTRIES),
  PostJournalEntry: unsupported('PostJournalEntry'),
  PostIncomingReceipt: unsupported('PostIncomingReceipt'),
  ReverseJournalEntry: unsupported('ReverseJournalEntry'),
  VerifyIntegrity: () => later(INTEGRITY, 400),
  GetFinancialSummary: () => later(SUMMARY),
  GetVatSummary: (from: string, to: string) => later(vatSummary(from, to)),

  // Belege
  SelectReceiptFilesDialog: () => later([]),
  FileIncomingReceipt: unsupported('FileIncomingReceipt'),
  AddReceiptFile: unsupported('AddReceiptFile'),
  RemoveReceiptFile: unsupported('RemoveReceiptFile'),
  GetReceipts: (status: string) =>
    later(status ? RECEIPTS.filter((r) => r.status === status) : RECEIPTS),
  GetReceipt: (id: number) => later(RECEIPTS.find((r) => r.id === id)),
  DiscardReceipt: unsupported('DiscardReceipt'),
  GetReceiptPreview: (receiptId: number) =>
    later({
      dataUrl: receiptPreview(),
      fileName: RECEIPTS.find((r) => r.id === receiptId)?.files[0].fileName ?? '',
      mimeType: 'image/png',
      intact: true,
    }),
  ExtractStructuredPart: unsupported('ExtractStructuredPart'),
  ProposeFromEInvoice: (receiptId: number) => later(proposalFor(receiptId)),
  ValidateEInvoice: unsupported('ValidateEInvoice'),
  GetEInvoiceRules: () => later(['BR-1', 'BR-2', 'BR-CO-10', 'BR-DE-1', 'BR-DE-15']),
  GetUncheckedEInvoiceRules: () =>
    later({
      'BR-CL-*': 'Codelisten-Regeln laufen gegen die mitgelieferten Listen, nicht gegen Schematron.',
    }),
  PreviewIncomingReceipt: (request: any) => later(previewIncoming(request), 120),
  PreviewOutgoingInvoice: unsupported('PreviewOutgoingInvoice'),

  // Bank & Zahlungen
  GetBankTransactions: () => later(BANK_TX),
  ImportCAMT053XML: unsupported('ImportCAMT053XML'),
  BookBankTransactionDirect: unsupported('BookBankTransactionDirect'),
  IgnoreBankTransaction: unsupported('IgnoreBankTransaction'),
  GetOpenItems: () => later(OPEN_ITEMS),
  SettlePayment: unsupported('SettlePayment'),

  // Kontakte & Rechnungen
  GetContacts: () => later(CONTACTS),
  SaveContact: unsupported('SaveContact'),
  DeleteContact: unsupported('DeleteContact'),
  GetInvoices: () => later(INVOICES),
  IssueInvoice: unsupported('IssueInvoice'),
  CancelInvoice: unsupported('CancelInvoice'),
  GenerateInvoiceZUGFeRD: unsupported('GenerateInvoiceZUGFeRD'),
  GetInvoiceDocument: unsupported('GetInvoiceDocument'),

  // Anlagevermögen
  GetFixedAssets: (assetClass: string) =>
    later(assetClass ? ASSETS.filter((a) => a.class === assetClass) : ASSETS),
  GetAssetSummary: (assetClass: string) => later(assetSummary(assetClass)),
  GetFixedAsset: (id: number) => later(assetDetail(id)),
  SaveFixedAsset: unsupported('SaveFixedAsset'),
  DeleteFixedAsset: unsupported('DeleteFixedAsset'),
  RecordAssetCostAdjustment: unsupported('RecordAssetCostAdjustment'),
  GetAssetAccounts: (assetClass: string) =>
    later(assetClass ? ASSET_ACCOUNTS.filter((a) => a.class === assetClass) : ASSET_ACCOUNTS),
  GetAssetRules: () => later(ASSET_RULES),
  ClassifyAcquisition: (netCost: number, _date: string, selfUsable: boolean) =>
    later(classifyAcquisition(netCost, selfUsable)),
  GetDepreciationRun: () => later(DEPRECIATION_RUN),
  BookDepreciationRun: unsupported('BookDepreciationRun'),
  BookAssetImpairment: unsupported('BookAssetImpairment'),
  BookAssetWriteUp: unsupported('BookAssetWriteUp'),
  PreviewAssetDisposal: (request: any) => later(disposalPreview(request), 80),
  DisposeFixedAsset: unsupported('DisposeFixedAsset'),
  GetAnlagenspiegel: () => later(ANLAGENSPIEGEL),
  GetAssetAcquisitionCandidates: () => later(ASSET_CANDIDATES),
  GetSammelposten: () => later(ASSETS.find((a) => a.method === 'pool') ?? null),

  // E-Bilanz, Audit & Festschreibung
  ExportEBilanzXBRL: () => later(XBRL, 400),
  GetAuditLogs: () => later(AUDIT_LOGS),
  GetFestschreibungen: () => later(FESTSCHREIBUNGEN),
  CommitPeriod: unsupported('CommitPeriod'),
  VerifyFestschreibung: (id: number) =>
    later({
      id,
      hasTimestamp: true,
      isValid: true,
      coversCurrent: true,
      genTime: FESTSCHREIBUNGEN.find((f) => f.id === id)?.tsaGenTime,
      tsaName: 'freeTSA',
      message: 'Der Zeitstempel deckt den festgeschriebenen Kettenkopf ab.',
    }),
};

// -------------------------------------------------------------------------

function susa() {
  const booked = ACCOUNTS.filter((x) => x.bookingsCount > 0);
  const totalDebit = booked.reduce((sum, x) => sum + x.debitSum, 0);
  const totalCredit = booked.reduce((sum, x) => sum + x.creditSum, 0);
  const classes = [...new Set(booked.map((x) => x.kontenklasse))].sort().map((kontenklasse) => {
    const accounts = booked.filter((x) => x.kontenklasse === kontenklasse);
    return {
      kontenklasse,
      kontenklasseName: KLASSEN[kontenklasse] ?? '',
      totalDebit: accounts.reduce((sum, x) => sum + x.debitSum, 0),
      totalCredit: accounts.reduce((sum, x) => sum + x.creditSum, 0),
      totalSaldoDebit: accounts
        .filter((x) => x.debitSum >= x.creditSum)
        .reduce((sum, x) => sum + x.balance, 0),
      totalSaldoCredit: accounts
        .filter((x) => x.debitSum < x.creditSum)
        .reduce((sum, x) => sum + x.balance, 0),
      accountsCount: accounts.length,
      accounts,
    };
  });
  return {
    fiscalYear: YEAR,
    totalDebit,
    totalCredit,
    totalSaldoDebit: classes.reduce((sum, x) => sum + x.totalSaldoDebit, 0),
    totalSaldoCredit: classes.reduce((sum, x) => sum + x.totalSaldoCredit, 0),
    isBalanced: totalDebit === totalCredit,
    difference: totalDebit - totalCredit,
    classes,
  };
}

function ledgerFor(accountNumber: string) {
  const account = ACCOUNTS.find((x) => x.number === accountNumber)!;
  let running = 0;
  const rows = ENTRIES.flatMap((entry) =>
    entry.lines
      .filter((line) => line.account === accountNumber)
      .map((line) => {
        const debit = line.side === 'S' ? line.amount : 0;
        const credit = line.side === 'H' ? line.amount : 0;
        running += debit - credit;
        return {
          entryId: entry.id,
          entryNumber: entry.entryNumber,
          bookingDate: entry.bookingDate,
          documentDate: entry.documentDate,
          documentNumber: entry.documentNumber,
          description: entry.description,
          kind: entry.kind,
          side: line.side,
          debitAmount: debit,
          creditAmount: credit,
          runningBalance: Math.abs(running),
          counterAccounts: entry.lines
            .filter((other) => other.account !== accountNumber)
            .map((other) => ({ account: other.account, name: other.accountName, amount: other.amount })),
          taxKey: line.taxKey,
        };
      }),
  );
  return {
    account,
    fiscalYear: YEAR,
    openingBalance: 0,
    totalDebit: rows.reduce((sum, r) => sum + r.debitAmount, 0),
    totalCredit: rows.reduce((sum, r) => sum + r.creditAmount, 0),
    closingBalance: Math.abs(running),
    rowCount: rows.length,
    rows,
  };
}

function proposalFor(receiptId: number) {
  if (receiptId !== 1) throw new Error('Dieser Beleg enthält keinen strukturierten Rechnungsdatensatz.');
  return {
    request: {
      contactId: 13,
      receiptId: 1,
      bookingDate: '2026-08-11',
      documentDate: '2026-08-10',
      serviceDateFrom: '2026-08-10',
      serviceDateTo: '2026-08-10',
      description: 'Notebooks, 4 Stück',
      taxTreatment: 'domestic',
      positions: [{ postingGroup: '', net: c(2716), taxRate: 1900, text: 'Notebook 14", 4 Stück' }],
      settlement: 'open',
    },
    format: 'ZUGFeRD / Factur-X',
    profile: 'EN 16931 (Comfort)',
    kind: '380',
    kindLabel: 'Rechnung',
    supplierName: 'Techpartner Nord GmbH',
    supplierVatId: 'DE263558107',
    invoiceNumber: 'ER-2026-0231',
    grossAmount: c(3232.04),
    matchedContact: true,
    notes: ['Die Buchungsgruppe bleibt offen: Welches Konto zutrifft, sagt keine Rechnung.'],
  };
}

/** Rechnet den Buchungssatz so, wie das Backend ihn liefern würde. */
function previewIncoming(request: any) {
  const lines: any[] = [];
  let net = 0;
  let tax = 0;
  let position = 1;

  for (const pos of request.positions ?? []) {
    const account = POSTING_GROUPS.find((g) => g.key === pos.postingGroup)?.account ?? '6300';
    lines.push({
      id: position,
      entryId: 0,
      position,
      side: 'S',
      amount: pos.net,
      account,
      accountName: accountName(account),
      text: pos.text,
    });
    position++;
    net += pos.net;
    if (request.taxTreatment === 'domestic' && pos.taxRate > 0) {
      tax += Math.round((pos.net * pos.taxRate) / 10000);
    }
  }

  if (tax > 0) {
    lines.push({
      id: position,
      entryId: 0,
      position,
      side: 'S',
      amount: tax,
      account: '1406',
      accountName: accountName('1406'),
      taxKey: '9',
      taxBase: net,
    });
    position++;
  }

  const gross = net + tax;
  const counter =
    request.settlement === 'paid' ? request.paymentAccount || '1800' : '3300';
  lines.push({
    id: position,
    entryId: 0,
    position,
    side: 'H',
    amount: gross,
    account: counter,
    accountName: accountName(counter),
  });

  return { lines, net, tax, gross, balanced: true, warnings: [] };
}
