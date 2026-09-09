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
  // Leer: die Anlegerstellung für § 20 InvStG folgt aus der Rechtsform.
  investorOverride: '',
  // Ab diesem Bruttobetrag verlangt der Belegweg den Leistungsnachweis
  // (RECH-08); die Voreinstellung des Backends sind 1.000 Euro.
  invoiceCheckThreshold: c(1000),
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
    kind: 'invoice',
    // Die Kopfdaten (BEL-02): bei einer E-Rechnung stehen sie im Datensatz und
    // werden beim Ablegen übernommen.
    documentDate: '2026-08-10',
    issuerName: 'Techpartner Nord GmbH',
    subject: 'Notebooks, 4 Stück',
    grossAmount: c(3232.04),
    taxAmount: c(516.04),
    currency: EUR,
    retentionClass: 'vouchers',
    retentionUntil: '2034-12-31',
    earliestDeletion: '2035-01-01',
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
  receipt(2, 'BE-2026-0230', 'filed', 'Stadtwerke_Abschlag_08-2026.pdf', '2026-08-21',
    'Stadtwerke Nord AG', 'Abschlag Strom August 2026', 428.4, undefined),
  receipt(3, 'BE-2026-0229', 'sealed', 'ER-2026-0238_Telefon-August.pdf', '2026-08-15',
    'Nordfunk Telekommunikation GmbH', 'Telefon und Anschluss August 2026', 321.3, 52),
  receipt(4, 'BE-2026-0228', 'sealed', 'ER-2026-0212_Cloud-Hosting.pdf', '2026-08-01',
    'Hanse Cloud Services GmbH', 'Cloud-Hosting August 2026', 1475.6, 42),
  receipt(5, 'BE-2026-0227', 'sealed', 'ER-2026-0218_Miete-August.pdf', '2026-08-01',
    'Kontorhaus Verwaltung GmbH', 'Miete Büro August 2026', 3570, 45),
  {
    ...receipt(6, 'BE-2026-0226', 'discarded', 'ER-2026-0209_doppelt.pdf', '2026-07-29',
      'Hanse Cloud Services GmbH', 'Cloud-Hosting Juli 2026', 1475.6, undefined),
    discardReason: 'Doppelt eingegangen, gebucht ist BE-2026-0225.',
  },
];

function receipt(
  id: number,
  receiptNumber: string,
  status: string,
  fileName: string,
  receivedAt: string,
  issuerName: string,
  subject: string,
  grossEuro: number,
  journalEntryId?: number,
) {
  const gross = c(grossEuro);
  // 19 % im Bruttobetrag: der Steueranteil ist Brutto minus Brutto/1,19.
  const tax = gross - Math.round(gross / 1.19);
  return {
    id,
    fiscalYear: YEAR,
    receiptNumber,
    direction: 'incoming',
    status,
    kind: 'invoice',
    documentDate: receivedAt,
    issuerName,
    subject,
    grossAmount: gross,
    taxAmount: tax,
    currency: EUR,
    retentionClass: 'vouchers',
    retentionUntil: `${YEAR + 8}-12-31`,
    earliestDeletion: `${YEAR + 9}-01-01`,
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
    // Bearbeiterkennung und Programmfassung stehen an jeder Zeile (UNV-04,
    // UNV-06); die Ansicht „Nachweise" führt beide als eigene Spalte.
    actor: 'anna@nordlicht-mbp',
    appVersion: '0.9.4',
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
    specialDue: 0,
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
  asset(9, 'AN-2026-0009', 'Fertigungslinie (im Bau)', 'tangible', '0700',
    'Geleistete Anzahlungen und Anlagen im Bau', '', '2026-01-15', c(80000), 0, 0, 0, 'none', 0),
  asset(10, 'AN-2026-0010', 'Fertigungsroboter KR-210', 'tangible', '0440', 'Maschinen',
    '6220', '2026-01-05', c(100000), 0, 0, c(10000), 'linear', 120,
    {
      specialPermille: 400,
      specialYears: 5,
      specialAccount: '6241',
      specialReason: 'Gewinn 2025: 142.000 €; ausschließlich betriebliche Nutzung',
      specialDue: c(8000),
    }),
  asset(12, 'AN-2025-0012', 'MSCI-World-ETF (thesaurierend)', 'financial', '0900',
    'Wertpapiere des Anlagevermögens', '', '2025-01-02', c(100000), 0, 0, 0, 'none', 0,
    {
      identifier: 'IE00B4L5Y983',
      fundClass: 'equity',
      quantity: 1200 * 10000,
      unitsHeld: 1200 * 10000,
      vorabpauschalen: c(1771),
    }),
  asset(13, 'AN-2024-0013', 'Darlehen an Werftgrund GmbH', 'financial', '0940',
    'Darlehen', '', '2024-03-01', c(50000), 0, 0, 0, 'none', 0,
    { maturityDate: '2027-03-01' }),
  asset(11, 'AN-2025-0011', 'US-Staatsanleihe 2032', 'financial', '0920',
    'Festverzinsliche Wertpapiere', '', '2025-05-12', c(10000), 0, 0, 0, 'none', 0,
    {
      identifier: 'US912828Z294',
      currency: 'USD',
      maturityDate: '2032-05-12',
      foreignCost: c(12000),
      quantity: 100 * 10000,
      unitsHeld: 100 * 10000,
    }),
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
  { number: '0700', name: 'Geleistete Anzahlungen und Anlagen im Bau', class: 'tangible', group: 'Anlagen im Bau', depreciable: false, inProgress: true, hint: 'Mit der Fertigstellung wird umgebucht, und die AfA beginnt.' },
  { number: '0850', name: 'Beteiligungen an Kapitalgesellschaften', class: 'financial', group: 'Anteile und Beteiligungen', depreciable: false },
  { number: '0920', name: 'Festverzinsliche Wertpapiere', class: 'financial', group: 'Wertpapiere', depreciable: false },
  { number: '0900', name: 'Wertpapiere des Anlagevermögens', class: 'financial', group: 'Wertpapiere', depreciable: false },
  { number: '0940', name: 'Darlehen', class: 'financial', group: 'Ausleihungen', depreciable: false },
];

const ASSET_RULES = {
  fiscalYear: YEAR,
  gwgImmediateLimit: c(800),
  gwgRecordFrom: c(250),
  poolLowerLimit: c(250),
  poolUpperLimit: c(1000),
  poolYears: 5,
  degressiveWindows: [
    { From: '2009-01-01', Until: '2010-12-31', FactorPermille: 2500, MaxPermille: 250, Source: '§ 7 Abs. 2 EStG in der Fassung des Gesetzes vom 21.12.2008' },
    { From: '2020-01-01', Until: '2022-12-31', FactorPermille: 2500, MaxPermille: 250, Source: '§ 7 Abs. 2 EStG in der Fassung des Zweiten Corona-Steuerhilfegesetzes' },
    { From: '2024-04-01', Until: '2024-12-31', FactorPermille: 2000, MaxPermille: 200, Source: '§ 7 Abs. 2 EStG in der Fassung des Wachstumschancengesetzes' },
    { From: '2025-07-01', Until: '2027-12-31', FactorPermille: 3000, MaxPermille: 300, Source: '§ 7 Abs. 2 Sätze 1 und 2 EStG' },
  ],
  specialMaxPermille: 400,
  specialPeriodYears: 5,
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
    { assetId: 1, inventoryNumber: 'AN-2024-0001', name: 'Büroeinrichtung Konferenzraum', account: '0650', expenseAccount: '6220', method: 'linear', rateLabel: '7,7 %', months: 12, planned: c(738.46), booked: 0, due: c(738.46), bookValueBefore: c(8430.77), bookValueAfter: c(7692.31), specialPlanned: 0, specialBooked: 0, specialDue: 0 },
    { assetId: 2, inventoryNumber: 'AN-2025-0002', name: 'Pkw VW ID.4 · HH-NS 412', account: '0520', expenseAccount: '6222', method: 'linear', rateLabel: '16,7 %', months: 12, planned: c(7000), booked: 0, due: c(7000), bookValueBefore: c(36166.67), bookValueAfter: c(29166.67), specialPlanned: 0, specialBooked: 0, specialDue: 0 },
    { assetId: 3, inventoryNumber: 'AN-2026-0003', name: 'CNC-Fräse Haas VF-2', account: '0440', expenseAccount: '6220', method: 'linear', rateLabel: '12,5 %', months: 11, planned: c(2750), booked: 0, due: c(2750), bookValueBefore: c(24000), bookValueAfter: c(21250), note: 'Zeitanteilig für 11 von 12 Monaten (§ 7 Abs. 1 Satz 4 EStG).', specialPlanned: 0, specialBooked: 0, specialDue: 0 },
    { assetId: 5, inventoryNumber: 'AN-2026-0005', name: 'Sammelposten 2026', account: '0675', expenseAccount: '6264', method: 'pool', rateLabel: '1/5', months: 12, planned: c(840), booked: 0, due: c(840), bookValueBefore: c(4200), bookValueAfter: c(3360), note: 'Auflösung des Sammelpostens 2026 mit einem Fünftel, ohne Zeitanteil (§ 6 Abs. 2a Satz 2 EStG).', specialPlanned: 0, specialBooked: 0, specialDue: 0 },
    { assetId: 8, inventoryNumber: 'AN-2025-0008', name: 'ERP-Lizenz Warenwirtschaft', account: '0135', expenseAccount: '6200', method: 'linear', rateLabel: '33,3 %', months: 12, planned: c(4000), booked: 0, due: c(4000), bookValueBefore: c(10000), bookValueAfter: c(6000), specialPlanned: 0, specialBooked: 0, specialDue: 0 },
    // Die Sonderabschreibung läuft neben der planmäßigen AfA und auf einem
    // eigenen Aufwandskonto (§ 7g Abs. 5 EStG, § 7a Abs. 4 EStG).
    { assetId: 10, inventoryNumber: 'AN-2026-0010', name: 'Fertigungsroboter KR-210', account: '0440', expenseAccount: '6220', method: 'linear', rateLabel: '10 %', months: 12, planned: c(10000), booked: 0, due: c(10000), bookValueBefore: c(100000), bookValueAfter: c(82000), specialAccount: '6241', specialPlanned: c(8000), specialBooked: 0, specialDue: c(8000) },
  ],
  total: c(33328.46),
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
    transfers: 0,
    depreciationOpening,
    depreciationYear,
    writeUpsYear: 0,
    depreciationDisposal: 0,
    depreciationTransfer: 0,
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
      transfers: acc.transfers + row.transfers,
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
      transfers: 0,
      depreciationOpening: 0,
      depreciationYear: 0,
      writeUpsYear: 0,
      depreciationDisposal: 0,
      depreciationClosing: 0,
      depreciationTransfer: 0,
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
    { fiscalYear: 2025, months: 10, method: 'linear', rateLabel: '16,7 %', openingBookValue: c(42000), amount: c(5833.33), closingBookValue: c(36166.67), booked: c(5833.33), due: 0, specialBooked: 0, specialDue: 0, status: 'gebucht', note: 'Zeitanteilig für 10 von 12 Monaten (§ 7 Abs. 1 Satz 4 EStG).' },
    { fiscalYear: 2026, months: 12, method: 'linear', rateLabel: '16,7 %', openingBookValue: c(36166.67), amount: c(7000), closingBookValue: c(29166.67), booked: 0, due: c(7000), specialBooked: 0, specialDue: 0, status: 'offen' },
    { fiscalYear: 2027, months: 12, method: 'linear', rateLabel: '16,7 %', openingBookValue: c(29166.67), amount: c(7000), closingBookValue: c(22166.67), booked: 0, due: c(7000), specialBooked: 0, specialDue: 0, status: 'geplant' },
    { fiscalYear: 2028, months: 12, method: 'linear', rateLabel: '16,7 %', openingBookValue: c(22166.67), amount: c(7000), closingBookValue: c(15166.67), booked: 0, due: c(7000), specialBooked: 0, specialDue: 0, status: 'geplant' },
    { fiscalYear: 2029, months: 12, method: 'linear', rateLabel: '16,7 %', openingBookValue: c(15166.67), amount: c(7000), closingBookValue: c(8166.67), booked: 0, due: c(7000), specialBooked: 0, specialDue: 0, status: 'geplant' },
    { fiscalYear: 2030, months: 12, method: 'linear', rateLabel: '16,7 %', openingBookValue: c(8166.67), amount: c(7000), closingBookValue: c(1166.67), booked: 0, due: c(7000), specialBooked: 0, specialDue: 0, status: 'geplant' },
    { fiscalYear: 2031, months: 2, method: 'linear', rateLabel: 'Restwert', openingBookValue: c(1166.67), amount: c(1166.67), closingBookValue: 0, booked: 0, due: c(1166.67), specialBooked: 0, specialDue: 0, status: 'geplant' },
  ],
};

// Der Fertigungsroboter zeigt beide Phasen: den Begünstigungszeitraum mit der
// Sonderabschreibung neben der planmäßigen AfA, und danach die
// Restwertverteilung des § 7a Abs. 9 EStG.
ASSET_SCHEDULES[10] = [
  { fiscalYear: 2026, months: 12, method: 'linear', rateLabel: '10 %', openingBookValue: c(100000), amount: c(10000), specialAmount: c(8000), closingBookValue: c(82000), booked: 0, due: c(10000), specialBooked: 0, specialDue: c(8000), status: 'offen', note: 'Zusätzlich Sonderabschreibung nach § 7g Abs. 5 EStG: 8.000,00 €. Sie tritt neben die planmäßige AfA, die daneben unverändert weiterläuft (§ 7a Abs. 4 EStG).' },
  { fiscalYear: 2027, months: 12, method: 'linear', rateLabel: '10 %', openingBookValue: c(82000), amount: c(10000), specialAmount: c(8000), closingBookValue: c(64000), booked: 0, due: c(10000), specialBooked: 0, specialDue: c(8000), status: 'geplant' },
  { fiscalYear: 2028, months: 12, method: 'linear', rateLabel: '10 %', openingBookValue: c(64000), amount: c(10000), specialAmount: c(8000), closingBookValue: c(46000), booked: 0, due: c(10000), specialBooked: 0, specialDue: c(8000), status: 'geplant' },
  { fiscalYear: 2029, months: 12, method: 'linear', rateLabel: '10 %', openingBookValue: c(46000), amount: c(10000), specialAmount: c(8000), closingBookValue: c(28000), booked: 0, due: c(10000), specialBooked: 0, specialDue: c(8000), status: 'geplant' },
  { fiscalYear: 2030, months: 12, method: 'linear', rateLabel: '10 %', openingBookValue: c(28000), amount: c(10000), specialAmount: c(8000), closingBookValue: c(10000), booked: 0, due: c(10000), specialBooked: 0, specialDue: c(8000), status: 'geplant' },
  { fiscalYear: 2031, months: 12, method: 'linear', rateLabel: 'linear auf 60 Restmonate', openingBookValue: c(10000), amount: c(2000), closingBookValue: c(8000), booked: 0, due: c(2000), specialBooked: 0, specialDue: 0, status: 'geplant', note: 'Der Begünstigungszeitraum der Sonderabschreibung ist abgelaufen: der Restwert verteilt sich von hier an auf die Restnutzungsdauer (§ 7a Abs. 9 EStG).' },
  { fiscalYear: 2032, months: 12, method: 'linear', rateLabel: 'linear auf 48 Restmonate', openingBookValue: c(8000), amount: c(2000), closingBookValue: c(6000), booked: 0, due: c(2000), specialBooked: 0, specialDue: 0, status: 'geplant' },
  { fiscalYear: 2033, months: 12, method: 'linear', rateLabel: 'linear auf 36 Restmonate', openingBookValue: c(6000), amount: c(2000), closingBookValue: c(4000), booked: 0, due: c(2000), specialBooked: 0, specialDue: 0, status: 'geplant' },
  { fiscalYear: 2034, months: 12, method: 'linear', rateLabel: 'linear auf 24 Restmonate', openingBookValue: c(4000), amount: c(2000), closingBookValue: c(2000), booked: 0, due: c(2000), specialBooked: 0, specialDue: 0, status: 'geplant' },
  { fiscalYear: 2035, months: 12, method: 'linear', rateLabel: 'Restwert', openingBookValue: c(2000), amount: c(2000), closingBookValue: 0, booked: 0, due: c(2000), specialBooked: 0, specialDue: 0, status: 'geplant' },
];

const ASSET_MOVEMENTS: Record<number, unknown[]> = {
  2: [
    { id: 1, assetId: 2, kind: 'acquisition', date: '2025-03-15', fiscalYear: 2025, costAmount: c(42000), depreciationAmount: 0, entryNumber: '2025-000112', note: 'Zugang', createdAt: '2025-03-15T09:00:00Z' },
    { id: 2, assetId: 2, kind: 'depreciation', date: '2025-12-31', fiscalYear: 2025, costAmount: 0, depreciationAmount: c(5833.33), entryNumber: '2025-000488', note: 'AfA 2025', createdAt: '2025-12-31T09:00:00Z' },
  ],
};

// Die Anleihe wird in Stück geführt: Zugang und Nachkauf tragen ihre
// Stückzahl, sonst ergäbe sich der Bestand nach einem Teilabgang nicht mehr.
ASSET_MOVEMENTS[11] = [
  { id: 30, assetId: 11, kind: 'acquisition', date: '2025-05-12', fiscalYear: 2025, costAmount: c(10000), depreciationAmount: 0, quantity: 100 * 10000, entryNumber: '2025-000231', note: 'Zugang · 12.000,00 USD', createdAt: '2025-05-12T09:00:00Z' },
  { id: 31, assetId: 11, kind: 'income', date: `${YEAR}-06-30`, fiscalYear: YEAR, costAmount: 0, depreciationAmount: 0, entryNumber: `${YEAR}-000318`, note: '250,00 € auf 7010. Halbjahreszins', createdAt: `${YEAR}-06-30T09:00:00Z` },
];

// Verträge und Papiere zum Anlagegut — abgelegt, nicht gebucht.
const ASSET_DOCUMENTS: Record<number, unknown[]> = {
  3: [
    { id: 1, assetId: 3, kind: 'contract', title: 'Kaufvertrag Haas Automation', fileName: 'kaufvertrag-vf2.pdf', mimeType: 'application/pdf', size: 184320, sha256: 'a'.repeat(64), storedPath: 'dokumente/contract/a.pdf', documentDate: '2026-01-28', createdAt: '2026-02-01T09:00:00Z' },
    { id: 2, assetId: 3, kind: 'maintenance', title: 'Abnahmeprotokoll und Einweisung', fileName: 'abnahme.pdf', mimeType: 'application/pdf', size: 92160, sha256: 'b'.repeat(64), storedPath: 'dokumente/maintenance/b.pdf', documentDate: '2026-02-03', createdAt: '2026-02-04T09:00:00Z' },
    { id: 3, assetId: 3, kind: 'insurance', title: 'Maschinenbruchversicherung', fileName: 'police.pdf', mimeType: 'application/pdf', size: 61440, sha256: 'c'.repeat(64), storedPath: 'dokumente/insurance/c.pdf', documentDate: '2026-02-10', validUntil: `${YEAR}-12-31`, createdAt: '2026-02-10T09:00:00Z' },
  ],
  13: [
    { id: 4, assetId: 13, kind: 'contract', title: 'Darlehensvertrag vom 01.03.2024', fileName: 'darlehensvertrag.pdf', mimeType: 'application/pdf', size: 133120, sha256: 'd'.repeat(64), storedPath: 'dokumente/contract/d.pdf', documentDate: '2024-03-01', validUntil: '2027-03-01', createdAt: '2024-03-01T09:00:00Z' },
    { id: 5, assetId: 13, kind: 'statement', title: 'Tilgungsplan', fileName: 'tilgungsplan.csv', mimeType: 'text/csv', size: 2048, sha256: 'e'.repeat(64), storedPath: 'dokumente/statement/e.csv', createdAt: '2024-03-01T09:00:00Z' },
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
    // Die Dokumente hängen am Anlagegut, nicht an der Detailansicht — im
    // Backend lädt das Repository sie mit.
    asset: { ...found, documents: ASSET_DOCUMENTS[id] ?? [] },
    // Ohne planmäßige AfA ist die kumulierte Abschreibung eine außerplanmäßige —
    // und damit genau der Betrag, der zugeschrieben werden dürfte.
    writeUpCeiling: found.method === 'none' ? found.accumulated : 0,
    schedule: ASSET_SCHEDULES[id] ?? [],
    movements: ASSET_MOVEMENTS[id] ?? [
      { id: 100 + id, assetId: id, kind: 'acquisition', date: found.acquisitionDate, fiscalYear: Number(found.acquisitionDate.slice(0, 4)), costAmount: found.cost, depreciationAmount: 0, note: 'Zugang', createdAt: `${found.acquisitionDate}T09:00:00Z` },
    ],
    notes: ASSET_NOTES[found.class] ?? [],
  };
}

/** Fondsarten, Anlegerstellungen und der Satz, der sich aus beidem ergibt. */
const INVESTMENT_RULES = {
  fundClasses: [
    { class: '', label: 'Kein Investmentanteil' },
    { class: 'equity', label: 'Aktienfonds (mindestens 51 % Kapitalbeteiligungen)' },
    { class: 'mixed', label: 'Mischfonds (mindestens 25 % Kapitalbeteiligungen)' },
    { class: 'real_estate', label: 'Immobilienfonds' },
    { class: 'foreign_real_estate', label: 'Auslands-Immobilienfonds' },
    { class: 'other', label: 'Investmentfonds ohne Teilfreistellung' },
  ],
  investorTypes: [
    { type: 'corporate', label: 'Anleger unterliegt dem Körperschaftsteuergesetz' },
    { type: 'individual_business', label: 'Natürliche Person, Anteile im Betriebsvermögen' },
    { type: 'basic', label: 'Grundsatz (Privatvermögen oder Ausnahme nach § 20 Abs. 1 Sätze 4 und 5 InvStG)' },
    { type: 'mixed', label: 'Personengesellschaft mit gemischt besteuerten Gesellschaftern' },
  ],
  investorType: 'corporate',
  investorLabel: 'Anleger unterliegt dem Körperschaftsteuergesetz',
  investorReason:
    'Eine Körperschaft unterliegt dem Körperschaftsteuergesetz. Für Investmentanteile heißt das: 80 % Aktienteilfreistellung (§ 20 Abs. 1 Satz 3 InvStG).',
  legalForm: 'GmbH',
  exemptions: [
    { class: 'equity', label: 'Aktienfonds', permille: 800, source: '§ 20 Abs. 1 Satz 3 InvStG', explanation: 'Der Anleger unterliegt dem Körperschaftsteuergesetz: die Aktienteilfreistellung beträgt 80 %.' },
    { class: 'mixed', label: 'Mischfonds', permille: 400, source: '§ 20 Abs. 1 Satz 3 i. V. m. § 20 Abs. 2 InvStG', explanation: 'Bei Mischfonds ist die Hälfte der Aktienteilfreistellung anzusetzen.' },
    { class: 'real_estate', label: 'Immobilienfonds', permille: 600, source: '§ 20 Abs. 3 Satz 1 InvStG', explanation: 'Der Satz hängt nicht vom Anleger ab.' },
    { class: 'foreign_real_estate', label: 'Auslands-Immobilienfonds', permille: 800, source: '§ 20 Abs. 3 Satz 2 InvStG', explanation: 'Der Satz hängt nicht vom Anleger ab.' },
    { class: 'other', label: 'Investmentfonds ohne Teilfreistellung', permille: 0, source: '§ 20 InvStG', explanation: 'Der Fonds erreicht keine der Quoten.' },
  ],
};

/** Der Rechtsformkatalog, je mit der Anlegerstellung, die aus ihr folgt. */
const LEGAL_FORMS = [
  { name: 'Einzelunternehmen', investor: 'individual_business', note: 'Das Unternehmen wird von einer natürlichen Person geführt. Für Investmentanteile im Betriebsvermögen heißt das: 60 % Aktienteilfreistellung (§ 20 Abs. 1 Satz 2 InvStG).' },
  { name: 'Eingetragener Kaufmann (e. K.)', investor: 'individual_business', note: 'Das Unternehmen wird von einer natürlichen Person geführt. Für Investmentanteile im Betriebsvermögen heißt das: 60 % Aktienteilfreistellung (§ 20 Abs. 1 Satz 2 InvStG).' },
  { name: 'Freiberufliche Praxis', investor: 'individual_business', note: 'Das Unternehmen wird von einer natürlichen Person geführt. Für Investmentanteile im Betriebsvermögen heißt das: 60 % Aktienteilfreistellung (§ 20 Abs. 1 Satz 2 InvStG).' },
  { name: 'GbR', investor: '', note: 'Bei einer Personengesellschaft bestimmt sich die Teilfreistellung nach dem einzelnen Gesellschafter (§ 20 Abs. 3a InvStG).' },
  { name: 'OHG', investor: '', note: 'Bei einer Personengesellschaft bestimmt sich die Teilfreistellung nach dem einzelnen Gesellschafter (§ 20 Abs. 3a InvStG).' },
  { name: 'KG', investor: '', note: 'Bei einer Personengesellschaft bestimmt sich die Teilfreistellung nach dem einzelnen Gesellschafter (§ 20 Abs. 3a InvStG).' },
  { name: 'GmbH & Co. KG', investor: '', note: 'Bei einer Personengesellschaft bestimmt sich die Teilfreistellung nach dem einzelnen Gesellschafter (§ 20 Abs. 3a InvStG). Sind alle Gesellschafter natürliche Personen, sind es 60 %; ist eine Körperschaft beteiligt, gilt für deren Anteil 80 %.' },
  { name: 'Partnerschaftsgesellschaft', investor: '', note: 'Bei einer Personengesellschaft bestimmt sich die Teilfreistellung nach dem einzelnen Gesellschafter (§ 20 Abs. 3a InvStG).' },
  { name: 'UG (haftungsbeschränkt)', investor: 'corporate', note: 'Eine Körperschaft unterliegt dem Körperschaftsteuergesetz. Für Investmentanteile heißt das: 80 % Aktienteilfreistellung (§ 20 Abs. 1 Satz 3 InvStG).' },
  { name: 'GmbH', investor: 'corporate', note: 'Eine Körperschaft unterliegt dem Körperschaftsteuergesetz. Für Investmentanteile heißt das: 80 % Aktienteilfreistellung (§ 20 Abs. 1 Satz 3 InvStG).' },
  { name: 'AG', investor: 'corporate', note: 'Eine Körperschaft unterliegt dem Körperschaftsteuergesetz. Für Investmentanteile heißt das: 80 % Aktienteilfreistellung (§ 20 Abs. 1 Satz 3 InvStG).' },
  { name: 'SE', investor: 'corporate', note: 'Eine Körperschaft unterliegt dem Körperschaftsteuergesetz. Für Investmentanteile heißt das: 80 % Aktienteilfreistellung (§ 20 Abs. 1 Satz 3 InvStG).' },
  { name: 'eG', investor: 'corporate', note: 'Eine Körperschaft unterliegt dem Körperschaftsteuergesetz. Für Investmentanteile heißt das: 80 % Aktienteilfreistellung (§ 20 Abs. 1 Satz 3 InvStG).' },
  { name: 'e. V.', investor: 'corporate', note: 'Eine Körperschaft unterliegt dem Körperschaftsteuergesetz. Für Investmentanteile heißt das: 80 % Aktienteilfreistellung (§ 20 Abs. 1 Satz 3 InvStG).' },
  { name: 'Stiftung', investor: 'corporate', note: 'Eine Körperschaft unterliegt dem Körperschaftsteuergesetz. Für Investmentanteile heißt das: 80 % Aktienteilfreistellung (§ 20 Abs. 1 Satz 3 InvStG).' },
  { name: 'Sonstige', investor: '', note: 'Aus dieser Rechtsform folgt die Anlegerstellung nicht. Wenn du Investmentanteile hältst, lege sie fest.' },
];

const ASSET_DOCUMENT_KINDS = [
  { kind: 'contract', label: 'Vertrag' },
  { kind: 'invoice', label: 'Rechnung (Kopie)' },
  { kind: 'statement', label: 'Abrechnung' },
  { kind: 'valuation', label: 'Gutachten' },
  { kind: 'registration', label: 'Register- oder Zulassungspapier' },
  { kind: 'insurance', label: 'Versicherung' },
  { kind: 'maintenance', label: 'Wartung und Prüfung' },
  { kind: 'photo', label: 'Bild' },
  { kind: 'other', label: 'Sonstiges' },
];

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
    specialDue: list.reduce((sum, a: any) => sum + (a.specialDue ?? 0), 0),
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
  const found = ASSETS.find((a) => a.id === request.assetId)! as any;
  const catchUp = found.dueAmount;
  const held = found.unitsHeld ?? 0;
  // Wo Stücke geführt werden, ist die Stückzahl die Vorgabe und der Betrag das
  // Ergebnis — genau wie im Dienst.
  const byUnits = (request.quantity ?? 0) > 0 && held > 0;
  const shareFromUnits = byUnits ? Math.round((found.cost * request.quantity) / held) : 0;
  const requested = byUnits ? shareFromUnits : (request.costShare ?? 0);
  const partial = requested > 0 && requested < found.cost;
  const costShare = partial ? requested : found.cost;
  const depreciationShare = partial
    ? Math.round((found.accumulated * costShare) / found.cost)
    : found.accumulated;
  const bookValue = Math.max(costShare - depreciationShare - catchUp, 0);
  const proceeds = request.kind === 'scrapped' ? 0 : (request.proceeds ?? 0);
  const result = proceeds - bookValue;
  const isGain = result > 0;
  const tax =
    request.taxTreatment === 'domestic' ? Math.round((proceeds * (request.taxRate ?? 1900)) / 10000) : 0;
  const gross = proceeds + tax;
  // Der SKR04 wählt nach Anlagenklasse *und* Ergebnis — beides bildet der Mock ab,
  // sonst zeigen die Screenshots eine Kontierung, die es so nicht gibt.
  const financial = found.class === 'financial';
  const accounts = isGain
    ? {
        revenue: financial ? '4851' : '4845',
        bookValue: financial ? '4857' : '4855',
        explanation:
          'Der Verkaufserlös liegt über dem Restbuchwert: es entsteht ein Buchgewinn. Der SKR04 führt Erlös und Restbuchwert dann unter den sonstigen betrieblichen Erträgen.',
      }
    : {
        revenue: financial ? '6891' : '6885',
        bookValue: financial ? '6897' : '6895',
        explanation:
          'Der Verkaufserlös liegt unter dem Restbuchwert: es entsteht ein Buchverlust. Derselbe Vorgang läuft im SKR04 dann über die sonstigen betrieblichen Aufwendungen.',
      };

  const lines: any[] = [];
  if (proceeds > 0) {
    lines.push({ id: 1, position: 1, side: 'S', account: request.paymentAccount ?? '1800', accountName: 'Bank', amount: gross });
    lines.push({
      id: 2, position: 2, side: 'H', account: accounts.revenue,
      accountName: financial
        ? `Erlöse aus Verkäufen Finanzanlagen (bei ${isGain ? 'Buchgewinn' : 'Buchverlust'})`
        : `Erlöse aus Verkäufen Sachanlagevermögen 19 % USt (bei ${isGain ? 'Buchgewinn' : 'Buchverlust'})`,
      amount: proceeds,
    });
    if (tax > 0) {
      lines.push({ id: 3, position: 3, side: 'H', account: '3806', accountName: 'Umsatzsteuer 19 %', amount: tax, taxKey: 'UST19', taxBase: proceeds });
    }
  }
  if (bookValue > 0) {
    lines.push({
      id: 4, position: 4, side: 'S', account: accounts.bookValue,
      accountName: financial ? 'Anlagenabgänge Finanzanlagen' : 'Anlagenabgänge Sachanlagen',
      amount: bookValue,
    });
    lines.push({ id: 5, position: 5, side: 'H', account: found.account, accountName: found.accountName, amount: bookValue });
  }

  return {
    partial,
    costShare,
    quantityShare: byUnits ? request.quantity : held,
    unitsRemaining: byUnits ? held - request.quantity : 0,
    depreciationShare,
    catchUpAmount: catchUp,
    specialCatchUp: found.specialDue ?? 0,
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
    investment: investmentNote(found.id, result, true),
  };
}


/**
 * Die Umrechnung zum Devisenkassamittelkurs des Stichtags (§ 256a HGB), nach
 * oben begrenzt durch die Anschaffungskosten (§ 253 Abs. 1 Satz 1 HGB).
 */
function currencyValuation(request: any) {
  const found = ASSETS.find((a) => a.id === request.assetId)! as any;
  const foreign = found.foreignCost ?? 0;
  const rate = request.ratePerEuro ?? 1_000_000;
  const valueAtRate = Math.round((foreign * 1_000_000) / rate);
  const difference = valueAtRate - found.bookValue;
  const ceiling = Math.max(found.acquisitionCost - found.bookValue, 0);
  const proposedAmount = difference < 0 ? -difference : Math.min(difference, ceiling);
  const proposal = difference < 0 ? 'impairment' : proposedAmount > 0 ? 'write_up' : 'none';
  return {
    currency: found.currency ?? 'EUR',
    foreignAmount: foreign,
    acquisitionRate: found.acquisitionCost > 0 ? Math.round((foreign * 1_000_000) / found.acquisitionCost) : 0,
    ratePerEuro: rate,
    valueAtRate,
    bookValue: found.bookValue,
    difference,
    shortTerm: Boolean(
      found.maturityDate &&
        found.maturityDate <=
          new Date(new Date(request.date ?? `${YEAR}-12-31`).getTime() + 365 * 864e5)
            .toISOString()
            .slice(0, 10),
    ),
    proposal,
    proposedAmount,
    explanation:
      difference < 0
        ? 'Zum Kurs des Stichtags ist der Bestand weniger wert als gebucht. Die Differenz ist außerplanmäßig abzuschreiben — bei Finanzanlagen auch dann, wenn die Wertminderung voraussichtlich nicht von Dauer ist (§ 253 Abs. 3 Satz 6 HGB).'
        : proposedAmount > 0
          ? 'Der Kurs ist gestiegen: zuzuschreiben ist höchstens bis zu den fortgeführten Anschaffungskosten (§ 253 Abs. 5 Satz 1 HGB).'
          : 'Der Kurs ist gestiegen, zuzuschreiben ist trotzdem nichts — über die Anschaffungskosten hinaus darf nicht bewertet werden (§ 253 Abs. 1 Satz 1 HGB).',
  };
}

/** Der Abschreibungsplan einer Eingabe, wie ihn das Backend rechnet. */
function previewPlan(request: any) {
  const cost = request.cost ?? 0;
  if (cost <= 0) return [];
  const start = Number(String(request.acquisitionDate).slice(0, 4));

  const specialTotal = Math.round((cost * (request.specialPermille ?? 0)) / 1000);
  const specialYears = Math.min(Math.max(request.specialYears ?? 0, 0), 5);

  if (request.method === 'immediate') {
    return [{ fiscalYear: start, months: 1, method: 'immediate', rateLabel: '100 %', openingBookValue: cost, amount: cost, closingBookValue: 0, specialBooked: 0, specialDue: 0 }];
  }
  if (request.method === 'pool') {
    const share = Math.round(cost / 5);
    return Array.from({ length: 5 }, (_, i) => ({
      fiscalYear: (request.poolYear || start) + i,
      months: 12,
      method: 'pool',
      rateLabel: '1/5',
      openingBookValue: cost - share * i,
      amount: i === 4 ? cost - share * 4 : share,
      closingBookValue: cost - share * (i + 1),
      specialBooked: 0,
      specialDue: 0,
    }));
  }

  const life = request.usefulLifeMonths ?? 0;
  if (life <= 0) return [];
  const month = Number(String(request.acquisitionDate).slice(5, 7));
  const firstMonths = 13 - month;
  const annual = Math.round((cost * 12) / life);
  const rows: any[] = [];
  let book = cost;
  let remaining = life;
  let year = start;
  let months = Math.min(firstMonths, life);
  let specialLeft = specialTotal;
  const specialShare = specialYears > 0 ? Math.round(specialTotal / specialYears) : 0;
  while (remaining > 0 && book > 0) {
    const inPeriod = specialYears > 0 && year < start + specialYears;
    // Die Sonderabschreibung wird im Anschaffungsjahr nicht zeitanteilig
    // gekürzt und kommt zur planmäßigen AfA hinzu, nicht an ihre Stelle.
    let special = 0;
    if (inPeriod && specialLeft > 0) {
      special = year === start + specialYears - 1 ? specialLeft : Math.min(specialShare, specialLeft);
      specialLeft -= special;
    }
    // Nach dem Begünstigungszeitraum verteilt § 7a Abs. 9 EStG den Restwert.
    const residual = specialTotal > 0 && year >= start + 5;
    const planned = residual ? Math.round((book * months) / remaining) : Math.round((annual * months) / 12);
    const amount = remaining - months <= 0 ? book - special : planned;
    rows.push({
      fiscalYear: year,
      months,
      method: 'linear',
      rateLabel: residual ? `linear auf ${remaining} Restmonate` : `${Math.round((1200 / life) * 10) / 10} %`,
      openingBookValue: book,
      amount,
      specialAmount: special,
      closingBookValue: book - amount - special,
      specialBooked: 0,
      specialDue: 0,
    });
    book -= amount + special;
    remaining -= months;
    year += 1;
    months = Math.min(12, remaining);
  }
  return rows;
}

/** Beträge wie im Backend: zwei Nachkommastellen, deutsche Schreibweise. */
const euro = (cents: number) =>
  (cents / 100).toLocaleString('de-DE', { minimumFractionDigits: 2, maximumFractionDigits: 2 });

/** Die Vorabpauschale nach § 18 InvStG, wie sie das Backend rechnet. */
function vorabpauschale(request: any) {
  const opening = request.openingPrice ?? 0;
  const closing = request.closingPrice ?? 0;
  const distributions = request.distributions ?? 0;
  const points = request.basisPoints ?? 0;
  let basis = Math.round((opening * points * 70) / (10_000 * 100));
  const growth = Math.max(closing - opening + distributions, 0);
  const capped = basis > growth;
  if (capped) basis = growth;
  let amount = Math.max(basis - distributions, 0);

  // § 18 Abs. 2 InvStG kürzt im Erwerbsjahr um ein Zwölftel je vollem Monat.
  const found = ASSETS.find((a) => a.id === request.assetId)! as any;
  let months = 12;
  if (String(found.acquisitionDate).slice(0, 4) === String(request.year)) {
    months = 13 - Number(String(found.acquisitionDate).slice(5, 7));
    amount = Math.round((amount * months) / 12);
  }
  return {
    year: request.year,
    basisReturn: basis,
    growth,
    capped,
    distributions,
    monthsCounted: months,
    amount,
    accruedOn: `${request.year + 1}-01-02`,
    explanation:
      `Basisertrag ${request.year}: ${euro(opening)} € × 70 % von ` +
      `${(points / 100).toFixed(2).replace('.', ',')} % = ${euro(basis)} €` +
      (capped ? ', begrenzt auf den Wertzuwachs (§ 18 Abs. 1 Satz 3 InvStG)' : '') +
      (months < 12 ? `, gekürzt auf ${months} Zwölftel für das Erwerbsjahr (§ 18 Abs. 2 InvStG)` : '') +
      `. Handelsrechtlich ist das kein Ertrag — es wird nichts gebucht.`,
  };
}

/** Die Teilfreistellung nach § 20 InvStG, mit den Vorabpauschalen beim Abgang. */
function investmentNote(assetId: number, gross: number, disposal: boolean) {
  const found = ASSETS.find((a) => a.id === assetId)! as any;
  if (!found.fundClass) return null;
  const exemption = INVESTMENT_RULES.exemptions.find((e) => e.class === found.fundClass)!;
  const vorab = disposal ? (found.vorabpauschalen ?? 0) : 0;
  const taxable = gross - vorab;
  const exempt = taxable > 0 ? Math.round((taxable * exemption.permille) / 1000) : 0;
  return {
    fundClass: found.fundClass,
    fundClassLabel: exemption.label,
    exemption: {
      permille: exemption.permille,
      determined: true,
      source: exemption.source,
      explanation: exemption.explanation,
    },
    grossAmount: gross,
    vorabpauschalen: vorab,
    exemptAmount: exempt,
    taxableAmount: taxable - exempt,
    explanation:
      `${disposal ? 'Der Buchgewinn' : 'Der Ertrag'} beträgt ${euro(gross)} €.` +
      (vorab > 0
        ? ` Davon gehen ${euro(vorab)} € an Vorabpauschalen ab, die über die Besitzzeit bereits versteuert wurden.`
        : '') +
      ` ${exemption.explanation} Steuerfrei bleiben ${euro(exempt)} €.`,
  };
}

// -------------------------------------------------------------------------
// Jahresabschluss: Bilanz, Gewinn- und Verlustrechnung, Größenklasse
//
// Die Gliederung wird aus denselben Konten gerechnet, aus denen auch die
// Summen- und Saldenliste entsteht — nicht abgetippt. Damit stimmt sie mit den
// übrigen Ansichten überein, und die Bilanz geht auf: die Summe der Aktiva ist
// die Summe der Passiva einschließlich des Jahresergebnisses.

/**
 * Der vorzeichenbehaftete Saldo eines Kontos.
 *
 * Aktiva und Aufwendungen tragen den Sollsaldo, Passiva und Erträge den
 * Habensaldo. Das gespeicherte `balance` ist der Betrag ohne Vorzeichen und
 * taugt deshalb nicht zum Addieren über eine Position hinweg.
 */
function signedBalance(account: (typeof ACCOUNTS)[number]): number {
  const debitSide = account.type === 'asset' || account.type === 'expense';
  return debitSide
    ? account.debitSum - account.creditSum
    : account.creditSum - account.debitSum;
}

/** Die bebuchten Konten einer Kontenklasse, ohne die ausgenommenen. */
function classAccounts(kontenklasse: number, except: string[] = []) {
  return ACCOUNTS.filter(
    (account) =>
      account.kontenklasse === kontenklasse &&
      account.bookingsCount > 0 &&
      !except.includes(account.number),
  );
}

/** Eine Gliederungszeile mit ihren Konten; der Betrag ist deren Summe. */
function line(
  key: string,
  ordinal: string,
  label: string,
  level: number,
  section: string,
  accounts: (typeof ACCOUNTS)[number][],
  extra: { amount?: number; isSubtotal?: boolean } = {},
) {
  const amount = extra.amount ?? accounts.reduce((sum, a) => sum + signedBalance(a), 0);
  return {
    key,
    ordinal,
    label,
    level,
    section,
    isSubtotal: extra.isSubtotal ?? false,
    isFallback: false,
    omitted: false,
    amount,
    priorAmount: 0,
    accounts: accounts.map((account) => ({
      number: account.number,
      name: account.name,
      positionId: key,
      position: account.name,
      amount: signedBalance(account),
      priorAmount: 0,
    })),
  };
}

const REVENUE = classAccounts(4);
const EXPENSES = [...classAccounts(5), ...classAccounts(6), ...classAccounts(7)];
const NET_INCOME =
  REVENUE.reduce((sum, a) => sum + signedBalance(a), 0) -
  EXPENSES.reduce((sum, a) => sum + signedBalance(a), 0);

const FIXED_ASSETS = classAccounts(0);
const CURRENT_ASSETS = classAccounts(1, ['1900']);
const PREPAID = classAccounts(1).filter((a) => a.number === '1900');
const EQUITY = classAccounts(2);
const LIABILITIES = classAccounts(3);

const ASSET_LINES = [
  line('aktiva.A', 'A.', 'Anlagevermögen', 1, 'aktiva', FIXED_ASSETS),
  line('aktiva.B', 'B.', 'Umlaufvermögen', 1, 'aktiva', CURRENT_ASSETS),
  line('aktiva.C', 'C.', 'Rechnungsabgrenzungsposten', 1, 'aktiva', PREPAID),
];

const LIABILITY_LINES = [
  line('passiva.A', 'A.', 'Eigenkapital', 1, 'passiva', EQUITY, {
    amount: EQUITY.reduce((sum, a) => sum + signedBalance(a), 0) + NET_INCOME,
  }),
  line('passiva.C', 'C.', 'Verbindlichkeiten', 1, 'passiva', LIABILITIES),
];

const INCOME_LINES = [
  line('guv.1', '1.', 'Umsatzerlöse', 1, 'guv', classAccounts(4).filter((a) => a.number === '4400')),
  line(
    'guv.4',
    '4.',
    'Sonstige betriebliche Erträge',
    1,
    'guv',
    classAccounts(4).filter((a) => a.number !== '4400'),
  ),
  line('guv.5', '5.', 'Materialaufwand', 1, 'guv', classAccounts(5), {
    amount: -classAccounts(5).reduce((sum, a) => sum + signedBalance(a), 0),
  }),
  line('guv.6', '6.', 'Personalaufwand', 1, 'guv', classAccounts(6).filter((a) => a.number === '6020'), {
    amount: -classAccounts(6)
      .filter((a) => a.number === '6020')
      .reduce((sum, a) => sum + signedBalance(a), 0),
  }),
  line(
    'guv.8',
    '8.',
    'Sonstige betriebliche Aufwendungen',
    1,
    'guv',
    classAccounts(6).filter((a) => a.number !== '6020'),
    {
      amount: -classAccounts(6)
        .filter((a) => a.number !== '6020')
        .reduce((sum, a) => sum + signedBalance(a), 0),
    },
  ),
  line('guv.17', '17.', 'Jahresüberschuss', 1, 'guv', [], {
    amount: NET_INCOME,
    isSubtotal: true,
  }),
];

/** Die Geschäftsjahre mit ihrem Abschlussstand; das laufende ist offen. */
const FISCAL_YEARS = [
  {
    year: YEAR - 1,
    startDate: `${YEAR - 1}-01-01`,
    endDate: `${YEAR - 1}-12-31`,
    isShort: false,
    status: 'adopted',
    adoptedOn: `${YEAR}-05-14`,
  },
  {
    year: YEAR,
    startDate: `${YEAR}-01-01`,
    endDate: `${YEAR}-12-31`,
    isShort: false,
    status: 'open',
  },
];

const TOTAL_ASSETS = ASSET_LINES.reduce((sum, l) => sum + l.amount, 0);
const TOTAL_LIABILITIES = LIABILITY_LINES.reduce((sum, l) => sum + l.amount, 0);

const SIZE_CRITERIA = {
  balanceSheetTotal: TOTAL_ASSETS,
  revenue: REVENUE.reduce((sum, a) => sum + signedBalance(a), 0),
  employees: 9,
};

const SIZE_THRESHOLDS = {
  validFrom: '2024-01-01',
  reference: '§ 267 HGB in der Fassung des BEG IV',
  micro: { balanceSheetTotal: c(450_000), revenue: c(900_000), employees: 10 },
  small: { balanceSheetTotal: c(7_500_000), revenue: c(15_000_000), employees: 50 },
  medium: { balanceSheetTotal: c(25_000_000), revenue: c(50_000_000), employees: 250 },
};

const SIZE_CLASS = {
  year: YEAR,
  closingDate: `${YEAR}-12-31`,
  class: 'small',
  criteria: SIZE_CRITERIA,
  current: {
    year: YEAR,
    closingDate: `${YEAR}-12-31`,
    criteria: SIZE_CRITERIA,
    class: 'small',
    met: ['Bilanzsumme', 'Umsatzerlöse'],
    thresholds: SIZE_THRESHOLDS,
  },
  isFirstYear: false,
  reason:
    'Bilanzsumme und Umsatzerlöse liegen unter den Schwellen des § 267 Abs. 1 HGB; die Zahl der Arbeitnehmer ebenso. Zwei von drei Merkmalen genügen, und sie sind an zwei aufeinanderfolgenden Stichtagen erfüllt.',
  obligations: {
    depth: 'short',
    depthReference: '§ 266 Abs. 1 Satz 3 HGB',
    notesRequired: true,
    notesReference: '§ 264 Abs. 1 Satz 1 HGB',
    managementReport: false,
    managementReportReference: '§ 264 Abs. 1 Satz 4 HGB',
    auditRequired: false,
    auditReference: '§ 316 Abs. 1 Satz 1 HGB',
    preparationMonths: 6,
    preparationReference: '§ 264 Abs. 1 Satz 4 HGB',
    disclosureMonths: 12,
    disclosureReference: '§ 325 Abs. 1a HGB',
    disclosureScope: 'Bilanz und Anhang ohne die Angaben zur Gewinn- und Verlustrechnung',
    disclosureScopeReference: '§ 326 Abs. 1 HGB',
  },
};

const STATEMENT = {
  header: {
    companyName: SETTINGS.companyName,
    // Die Firma des Bilanzkopfes: der Name mit der Rechtsform, hier ohne den
    // Zusatz „i. G." — die Gesellschaft ist eingetragen.
    firmName: SETTINGS.companyName,
    legalForm: SETTINGS.legalForm,
    seat: SETTINGS.zipCity,
    registerCourt: 'Amtsgericht Hamburg',
    registerNumber: 'HRB 148223',
    fiscalYear: YEAR,
    startDate: `${YEAR}-01-01`,
    closingDate: `${YEAR}-12-31`,
    priorYear: YEAR - 1,
    isShortYear: false,
    reference: '§ 264 Abs. 1a HGB',
    missing: [],
  },
  statement: {
    fiscalYear: YEAR,
    priorYear: YEAR - 1,
    hasPrior: false,
    depth: 'short',
    assets: ASSET_LINES,
    liabilities: LIABILITY_LINES,
    income: INCOME_LINES,
    statistical: [],
    assignment: { unassigned: [], wrongSign: [], signSwitches: [], fallbacks: [] },
    totalAssets: TOTAL_ASSETS,
    totalAssetsPrior: 0,
    totalLiabilities: TOTAL_LIABILITIES,
    totalLiabilitiesPrior: 0,
    balanceSheetTotal: TOTAL_ASSETS,
    balanceSheetTotalPrior: 0,
    netIncome: NET_INCOME,
    netIncomePrior: 0,
    revenue: SIZE_CRITERIA.revenue,
    revenuePrior: 0,
  },
  sizeClass: SIZE_CLASS,
  maturities: {
    closingDate: `${YEAR}-12-31`,
    reference: '§ 268 Abs. 4 und 5 HGB',
    rows: [
      {
        key: 'verbindlichkeiten',
        label: 'Verbindlichkeiten aus Lieferungen und Leistungen',
        total: LIABILITY_LINES[1].amount,
        upToOneYear: LIABILITY_LINES[1].amount,
        overOneYear: 0,
        overFiveYears: 0,
        items: 3,
        undated: 0,
      },
    ],
  },
  notes: {
    reference: '§§ 284 bis 288 HGB',
    texts: [],
    provisionMirror: {
      fiscalYear: YEAR,
      rows: [],
      total: {
        kind: 'sonstige',
        label: 'Summe',
        account: '',
        opening: 0,
        additions: 0,
        used: 0,
        released: 0,
        unwinding: 0,
        closing: 0,
      },
    },
    reconciliation: {
      fiscalYear: YEAR,
      cutoff: `${YEAR}-12-31`,
      rows: [],
      equityEffect: 0,
      note: 'Es bestehen keine Abweichungen zwischen Handels- und Steuerbilanz.',
    },
  },
  deadlines: [],
};

// -------------------------------------------------------------------------
// Welle 7: Aufgabenliste, Monatsabschluss, Bankvorschlag, Mahnwesen, Prüfpfad

const TODAY = `${YEAR}-08-25`;

/** Die Aufgabenliste der Startseite in ihren drei Gruppen. */
const TASKS = {
  today: TODAY,
  overdue: [
    {
      key: 'deadline.ustva-2026-07',
      group: 'overdue',
      title: 'Voranmeldung Juli 2026 übermitteln',
      why: 'Die Voranmeldung ist bis zum zehnten Tag nach Ablauf des Zeitraums zu übermitteln.',
      reference: '§ 18 Abs. 1 UStG',
      target: { page: 'vat', params: {} },
      count: 0,
      amount: c(4318.4),
      dueDate: `${YEAR}-08-10`,
    },
  ],
  open: [
    {
      key: 'bank.unmatched',
      group: 'open',
      title: 'Bankumsätze ohne Zuordnung klären',
      why: 'Ein Umsatz ohne Zuordnung steht in keiner Buchung und fehlt damit in jeder Auswertung.',
      reference: 'GoBD Rz. 36',
      target: { page: 'bank', params: {} },
      count: 4,
      amount: c(12480.6),
    },
    {
      key: 'receipt.unbooked',
      group: 'open',
      title: 'Belege buchen',
      why: 'Abgelegte Belege sind erfasst, aber noch nicht gebucht.',
      target: { page: 'receipts', params: { status: 'filed' } },
      count: 2,
      amount: c(3630.94),
    },
    {
      key: 'dunning.overdue',
      group: 'open',
      title: 'Überfällige Forderungen mahnen',
      why: 'Zwei Kunden sind seit mehr als dreißig Tagen im Verzug.',
      reference: '§ 286 Abs. 3 BGB',
      target: { page: 'bank', params: { view: 'dunning' } },
      count: 2,
      amount: c(28311),
    },
  ],
  upcoming: [
    {
      key: 'deadline.ustva-2026-08',
      group: 'upcoming',
      title: 'Voranmeldung August 2026 vorbereiten',
      why: 'Der Zeitraum endet am Monatsende; die Meldung ist zehn Tage später fällig.',
      reference: '§ 18 Abs. 1 UStG',
      target: { page: 'deadlines', params: { key: 'ustva-2026-08' } },
      count: 0,
      amount: 0,
      dueDate: `${YEAR}-09-10`,
    },
  ],
};

/** Der Stand eines Monats in seinen drei Schritten. */
function monthCloseState(month: string) {
  const [year, index] = month.split('-').map(Number);
  if (!year || !index) throw new Error(`${month} ist kein Monat (erwartet JJJJ-MM).`);
  if (year !== YEAR) {
    throw new Error(
      `Der Monat gehört zum Geschäftsjahr ${year} — wechsle zuerst das Geschäftsjahr.`,
    );
  }
  const names = [
    'Januar', 'Februar', 'März', 'April', 'Mai', 'Juni',
    'Juli', 'August', 'September', 'Oktober', 'November', 'Dezember',
  ];
  const last = new Date(Date.UTC(year, index, 0)).getUTCDate();
  const label = `${names[index - 1]} ${year}`;
  // Das Quartal endet im März, Juni, September und Dezember; nur dort steht die
  // Voranmeldung an (der Mandant meldet vierteljährlich).
  const quarterEnd = index % 3 === 0;
  return {
    month,
    label,
    from: `${month}-01`,
    to: `${month}-${String(last).padStart(2, '0')}`,
    fiscalYear: YEAR,
    steps: [
      {
        number: 1,
        key: 'check',
        title: 'Prüfbericht',
        state: 'open',
        note: '2 Hinweise, die die Festschreibung nicht verhindern.',
      },
      {
        number: 2,
        key: 'commit',
        title: 'Festschreiben',
        state: 'open',
        note: 'Der Monat ist noch nicht festgeschrieben.',
      },
      quarterEnd
        ? {
            number: 3,
            key: 'vat',
            title: `Voranmeldung ${year}-Q${index / 3} bestätigen`,
            state: 'blocked',
            note: 'Das Kennziffernblatt steht bereit. Bestätigen lässt sich die Übermittlung erst nach der Festschreibung.',
          }
        : {
            number: 3,
            key: 'vat',
            title: 'Voranmeldung bestätigen',
            state: 'not_applicable',
            note: 'In diesem Monat ist keine Voranmeldung abzugeben.',
          },
    ],
    findings: [
      {
        id: 1,
        checkRunId: 0,
        rule: 'bank_unmatched',
        severity: 'warning',
        objectType: 'bank_transaction',
        objectId: '18',
        message: '4 Bankumsätze sind keiner Buchung zugeordnet.',
        reference: 'GoBD Rz. 36',
      },
      {
        id: 2,
        checkRunId: 0,
        rule: 'receipt_unbooked',
        severity: 'warning',
        objectType: 'receipt',
        objectId: '2',
        message: '2 abgelegte Belege sind noch nicht gebucht.',
      },
    ],
    blocking: 0,
    committed: false,
    committedTo: `${YEAR}-06-30`,
    vatApplies: quarterEnd,
    vatPeriodKey: quarterEnd ? `${year}-Q${index / 3}` : undefined,
    vatPeriodLabel: quarterEnd ? `${index / 3}. Quartal ${year}` : undefined,
    vatStatus: quarterEnd ? 'draft' : undefined,
    vatDueDate: quarterEnd ? `${year}-${String(index + 1).padStart(2, '0')}-10` : undefined,
  };
}

/**
 * Die Vorschläge zu einem Bankumsatz, der beste zuerst.
 *
 * Gerechnet und nicht abgetippt: der Vorschlag entsteht aus denselben offenen
 * Posten, die auch die Liste zeigt. Ein exakter Betrag und die Rechnungsnummer
 * im Verwendungszweck wiegen schwer, die gelernte Regel greift erst, wo es
 * keinen offenen Posten gibt.
 */
function bankSuggestions(bankTxId: number) {
  const tx = BANK_TX.find((t) => t.id === bankTxId);
  if (!tx) {
    return { bankTxId, amount: 0, suggestions: [], note: 'Der Umsatz ist nicht bekannt.' };
  }
  const suggestions: any[] = [];
  for (const item of OPEN_ITEMS) {
    const reasons: string[] = [];
    if (item.openAmount === Math.abs(tx.amount)) reasons.push('Betrag stimmt genau');
    if (tx.remittanceInfo.includes(item.documentNumber)) {
      reasons.push('Rechnungsnummer im Verwendungszweck');
    }
    if (reasons.length === 0) continue;
    suggestions.push({
      kind: 'open_item',
      score: reasons.length === 2 ? 95 : 60,
      reasons,
      label: `${item.documentNumber} · ${item.contactName}`,
      contactId: item.contactId,
      contactName: item.contactName,
      amount: item.openAmount,
      items: [item],
    });
  }
  suggestions.sort((a, b) => b.score - a.score);

  if (suggestions.length === 0) {
    const rule = BANK_RULES.find((r) => tx.remittanceInfo.toLowerCase().includes(r.pattern));
    if (rule) {
      suggestions.push({
        kind: 'rule',
        score: 70,
        reasons: [`Zuletzt ${rule.hits} Mal so zugeordnet`],
        label: rule.label,
        amount: Math.abs(tx.amount),
        items: [],
        counterAccount: rule.counterAccount,
        postingGroup: rule.postingGroup,
      });
    }
  }

  return {
    bankTxId,
    amount: tx.amount,
    suggestions,
    note:
      suggestions.length === 0
        ? 'Zu diesem Umsatz gibt es weder einen offenen Posten noch eine gelernte Regel.'
        : undefined,
  };
}

/** Die aus bestätigten Zuordnungen gelernten Regeln. */
const BANK_RULES = [
  {
    id: 1,
    pattern: 'büromiete',
    label: 'Miete (unbewegliche Wirtschaftsgüter)',
    counterAccount: '6310',
    postingGroup: 'miete',
    moneyIn: false,
    hits: 8,
    lastUsedAt: `${YEAR}-08-01T06:12:00Z`,
    createdAt: '2025-09-01T06:12:00Z',
    updatedAt: `${YEAR}-08-01T06:12:00Z`,
  },
  {
    id: 2,
    pattern: 'kontoführung',
    label: 'Nebenkosten des Geldverkehrs',
    counterAccount: '6855',
    postingGroup: 'geldverkehr',
    moneyIn: false,
    hits: 11,
    lastUsedAt: `${YEAR}-08-24T05:40:00Z`,
    createdAt: '2025-01-24T05:40:00Z',
    updatedAt: `${YEAR}-08-24T05:40:00Z`,
  },
];

/** Der Basiszinssatz als datierte Tabelle; der letzte Wert ist fortgeschrieben. */
const BASE_RATES = [
  { validFrom: '2024-01-01', basisPoints: 362, source: 'Deutsche Bundesbank', provisional: false, updatedAt: '2024-01-02T08:00:00Z' },
  { validFrom: '2024-07-01', basisPoints: 337, source: 'Deutsche Bundesbank', provisional: false, updatedAt: '2024-07-01T08:00:00Z' },
  { validFrom: '2025-01-01', basisPoints: 227, source: 'Deutsche Bundesbank', provisional: false, updatedAt: '2025-01-02T08:00:00Z' },
  { validFrom: '2025-07-01', basisPoints: 127, source: 'Deutsche Bundesbank', provisional: false, updatedAt: '2025-07-01T08:00:00Z' },
  { validFrom: '2026-01-01', basisPoints: 127, source: '', provisional: true, updatedAt: '2026-01-02T08:00:00Z' },
];

/** Die Mahnvorschläge: je Kunde die Posten, die Stufe, Zinsen und Gebühr. */
const DUNNING_PROPOSALS = [
  {
    contactId: 2,
    contactName: 'Elbe Werkzeug GmbH',
    isConsumer: false,
    level: 2,
    levelLabel: '1. Mahnung',
    noticeDate: TODAY,
    items: [
      {
        entryId: 44,
        documentNumber: 'RE-2026-0112',
        documentDate: `${YEAR}-06-18`,
        dueDate: `${YEAR}-07-02`,
        openAmount: c(18921),
        daysOverdue: 54,
        defaultFrom: `${YEAR}-07-19`,
        interestDays: 37,
        interest: c(197.02),
        level: 2,
        previousLevel: 1,
        lumpSum: c(40),
      },
    ],
    principal: c(18921),
    interest: c(197.02),
    fee: c(5),
    lumpSum: c(40),
    total: c(19163.02),
    note: '',
  },
  {
    contactId: 3,
    contactName: 'Anna Wieland',
    isConsumer: true,
    level: 1,
    levelLabel: 'Zahlungserinnerung',
    noticeDate: TODAY,
    items: [
      {
        entryId: 51,
        documentNumber: 'RE-2026-0119',
        documentDate: `${YEAR}-07-30`,
        dueDate: `${YEAR}-08-13`,
        openAmount: c(8211),
        daysOverdue: 12,
        defaultFrom: `${YEAR}-08-14`,
        interestDays: 11,
        interest: c(15.44),
        level: 1,
        previousLevel: 0,
        lumpSum: 0,
      },
    ],
    principal: c(8211),
    interest: c(15.44),
    fee: 0,
    lumpSum: 0,
    total: c(8226.44),
    note: 'Verbraucher: fünf Prozentpunkte über dem Basiszinssatz, keine Pauschale.',
  },
];

/** Die schon erzeugten Mahnschreiben, das jüngste zuerst. */
const DUNNING_NOTICES = [
  {
    id: 1,
    fiscalYear: YEAR,
    contactId: 2,
    contactName: 'Elbe Werkzeug GmbH',
    isConsumer: false,
    level: 1,
    levelLabel: 'Zahlungserinnerung',
    noticeDate: `${YEAR}-07-24`,
    dueDate: `${YEAR}-08-07`,
    principalAmount: c(18921),
    interestAmount: c(30.11),
    feeAmount: 0,
    lumpSumAmount: c(40),
    totalAmount: c(18991.11),
    documentName: 'Zahlungserinnerung_Elbe-Werkzeug_2026-07-24.pdf',
    documentPath: 'dokumente/mahnungen/2026/Zahlungserinnerung_Elbe-Werkzeug_2026-07-24.pdf',
    documentSha256: fakeHash('dunning-1'),
    items: [],
  },
];

/** Der Prüfpfad eines Belegs: Beleg → Buchung → Zahlung → Bankumsatz. */
function auditTrail(receiptId: number) {
  const found = RECEIPTS.find((r) => r.id === receiptId);
  if (!found) throw new Error('Der Beleg ist nicht bekannt.');
  const entry = ENTRIES.find((e) => e.receiptId === receiptId);
  return {
    receiptId,
    receiptNumber: found.receiptNumber,
    direction: 'incoming',
    documentDate: found.receivedAt,
    issuerName: 'Nordlicht Telekommunikation GmbH',
    grossAmount: c(321.3),
    orderReference: 'BST-2026-0044',
    serviceProof: 'geprüft gegen Bestellung BST-2026-0044 vom 02.08.2026',
    serviceProofAt: `${YEAR}-08-16`,
    steps: [
      {
        stage: 'receipt',
        title: `Beleg ${found.receiptNumber}`,
        date: found.receivedAt,
        reference: found.receiptNumber,
        amount: c(321.3),
        detail: 'Eingegangen per E-Mail, versiegelt.',
      },
      ...(entry
        ? [
            {
              stage: 'booking',
              title: `Buchung ${entry.entryNumber}`,
              date: entry.bookingDate,
              reference: entry.entryNumber,
              amount: c(321.3),
              detail: entry.description,
            },
          ]
        : []),
    ],
    note: entry
      ? 'Gebucht; eine Zahlung ist zu diesem Beleg noch nicht zugeordnet.'
      : 'Der Beleg ist noch nicht gebucht.',
  };
}

/** Die Einheiten aus UN/ECE Rec. 20, so weit der Rechnungsdialog sie anbietet. */
const UNIT_CODES = [
  { code: 'C62', label: 'Stück' },
  { code: 'HUR', label: 'Stunde' },
  { code: 'DAY', label: 'Tag' },
  { code: 'MON', label: 'Monat' },
  { code: 'KGM', label: 'Kilogramm' },
];

/** Die Zielformate, in denen eine Rechnung ausgestellt werden kann. */
const EINVOICE_PROFILES = [
  {
    profile: 'zugferd_en16931',
    label: 'ZUGFeRD / Factur-X (EN 16931)',
    hint: 'PDF mit eingebettetem Datensatz; für Unternehmen der Regelfall.',
  },
  {
    profile: 'xrechnung_cii',
    label: 'XRechnung (CII)',
    hint: 'Reiner Datensatz; von öffentlichen Auftraggebern verlangt.',
  },
  { profile: 'pdf_only', label: 'Nur PDF', hint: 'Ohne Datensatz; für Verbraucher.' },
];

/** Die Versandwege des Vermerks „Als versendet vermerken". */
const SENT_VIA_OPTIONS = [
  { via: 'email', label: 'E-Mail' },
  { via: 'portal', label: 'Portal' },
  { via: 'post', label: 'Post' },
  { via: 'other', label: 'Anderer Weg' },
];

/** Die Gründe, mit denen eine Lücke im Nummernkreis begründet wird. */
const NUMBER_GAP_REASONS = [
  { reason: 'aborted', label: 'Abgebrochene Ausstellung' },
  { reason: 'test', label: 'Testlauf' },
  { reason: 'cancelled', label: 'Storniert' },
  { reason: 'unknown', label: 'Unbekannt' },
];

/**
 * Das Kennziffernblatt eines Voranmeldungszeitraums.
 *
 * Gerechnet aus denselben Monatswerten wie die Umsatzsteuerübersicht: Zahllast
 * gleich Umsatzsteuer minus Vorsteuer. Der Mandant meldet vierteljährlich, der
 * Schlüssel ist deshalb „JJJJ-Qn".
 */
function vatReturn(periodKey: string) {
  const quarter = Number(periodKey.split('-Q')[1] || 0);
  if (!quarter) throw new Error(`Für ${periodKey} liegt kein Kennziffernblatt vor.`);
  const firstMonth = (quarter - 1) * 3 + 1;
  const from = `${YEAR}-${String(firstMonth).padStart(2, '0')}-01`;
  const lastMonth = firstMonth + 2;
  const lastDay = new Date(Date.UTC(YEAR, lastMonth, 0)).getUTCDate();
  const to = `${YEAR}-${String(lastMonth).padStart(2, '0')}-${lastDay}`;
  const summary = vatSummary(from, to);
  const base = summary.taxableRevenue[0]?.net ?? 0;
  const outputTax = summary.outputTax;
  return {
    id: quarter,
    fiscalYear: YEAR,
    periodType: 'quarter',
    periodKey,
    periodFrom: from,
    periodTo: to,
    isCorrection: false,
    status: 'draft',
    payable: summary.payable,
    dueDate: `${YEAR}-${String(lastMonth + 1).padStart(2, '0')}-10`,
    figures: [
      {
        code: '81',
        label: 'Steuerpflichtige Umsätze zum Steuersatz von 19 %',
        hasBase: true,
        base,
        hasTax: true,
        taxCode: '81',
        tax: outputTax,
        expectedTax: outputTax,
      },
      {
        code: '66',
        label: 'Vorsteuerbeträge aus Rechnungen von anderen Unternehmern',
        hasBase: false,
        base: 0,
        hasTax: true,
        tax: summary.inputTax,
        expectedTax: summary.inputTax,
      },
      {
        code: '83',
        label: 'Verbleibende Umsatzsteuer-Vorauszahlung',
        hasBase: false,
        base: 0,
        hasTax: true,
        tax: summary.payable,
        expectedTax: summary.payable,
      },
    ],
    lateEntries: [],
    createdAt: `${to}T18:00:00Z`,
  };
}

/** Der Hinweis zu einem langen Zahlungsziel; leer heißt: unauffällig. */
function paymentTermNotice(dueDays: number): string {
  if (dueDays <= 60) return '';
  return (
    `Ein Zahlungsziel von ${dueDays} Tagen liegt über sechzig Tagen. Eine solche Frist ist nur ` +
    'wirksam, wenn sie ausdrücklich vereinbart und für den Gläubiger nicht grob unbillig ist ' +
    '(§ 271a Abs. 1 BGB); andernfalls tritt Verzug früher ein, als die Rechnung erwarten lässt.'
  );
}

// -------------------------------------------------------------------------
// Meldezeiträume, Prüfläufe, Abschlussweg, Nebenpflichten und Nachweise
//
// Die Ansichten der Wellen 5b bis 7 holen ihre Auswertungen selbst. Was eine
// Seite beim Öffnen ruft, steht hier mit Beispieldaten; was sie erst auf
// Knopfdruck ruft, steht als `unsupported` in der Bridge — geschrieben wird in
// der Vorschau nichts.

const pad = (n: number) => String(n).padStart(2, '0');

/** Die vier Quartale des Jahres als Meldezeitraum. */
function quarters() {
  return [1, 2, 3, 4].map((q) => {
    const firstMonth = (q - 1) * 3 + 1;
    const lastMonth = firstMonth + 2;
    const lastDay = new Date(Date.UTC(YEAR, lastMonth, 0)).getUTCDate();
    const dueYear = lastMonth === 12 ? YEAR + 1 : YEAR;
    const dueMonth = lastMonth === 12 ? 1 : lastMonth + 1;
    return {
      q,
      key: `${YEAR}-Q${q}`,
      type: 'quarter',
      label: `${q}. Quartal ${YEAR}`,
      from: `${YEAR}-${pad(firstMonth)}-01`,
      to: `${YEAR}-${pad(lastMonth)}-${pad(lastDay)}`,
      year: YEAR,
      dueDate: `${dueYear}-${pad(dueMonth)}-10`,
    };
  });
}

/**
 * Die Voranmeldungszeiträume mit Stand. Das erste Halbjahr ist übermittelt und
 * festgeschrieben, das laufende Quartal steht als Entwurf.
 */
function vatPeriods() {
  return quarters().map((p) => {
    const submitted = p.q <= 2;
    return {
      key: p.key,
      type: p.type,
      label: p.label,
      from: p.from,
      to: p.to,
      year: p.year,
      dueDate: p.dueDate,
      status: submitted ? 'submitted' : 'draft',
      returnId: p.q,
      committed: submitted,
      payable: vatSummary(p.from, p.to).payable,
      submittedAt: submitted ? `${p.dueDate}T09:20:00Z` : undefined,
      isOverdue: !submitted && p.dueDate < TODAY,
    };
  });
}

/** Die übermittelten Voranmeldungen, wie sie in der Liste stehen. */
function savedVatReturns() {
  return quarters()
    .filter((p) => p.q <= 2)
    .map((p) => ({
      ...vatReturn(p.key),
      status: 'submitted',
      submittedAt: `${p.dueDate}T09:20:00Z`,
      transferTicket: `TT-${YEAR}${pad(p.q * 3)}-4711${p.q}`,
      submissionNote: 'In Mein ELSTER übermittelt.',
    }));
}

/** Die Meldezeiträume der Zusammenfassenden Meldung. */
function zmPeriods() {
  return quarters().map((p) => ({
    key: p.key,
    type: p.type,
    label: p.label,
    from: p.from,
    to: p.to,
    year: p.year,
    dueDate: `${p.dueDate.slice(0, 8)}25`,
    status: 'draft',
    committed: p.q <= 2,
    total: 0,
    isOverdue: false,
  }));
}

/**
 * Eine Meldung ohne Zeilen. Der Mandant liefert im Beispieljahr nicht
 * innergemeinschaftlich — eine erfundene Zeile stünde gegen die
 * Umsatzsteuerzahlen, die aus denselben Buchungen kommen.
 */
function zmReturnFor(periodKey: string) {
  const period = quarters().find((p) => p.key === periodKey);
  if (!period) throw new Error(`Für ${periodKey} liegt keine Meldung vor.`);
  return {
    id: period.q,
    fiscalYear: YEAR,
    periodType: 'quarter',
    periodKey,
    periodFrom: period.from,
    periodTo: period.to,
    isCorrection: false,
    status: 'draft',
    dueDate: `${period.dueDate.slice(0, 8)}25`,
    totalSupplies: 0,
    totalServices: 0,
    lines: [],
    findings: [],
    lateEntries: [],
    createdAt: `${period.to}T18:00:00Z`,
  };
}

/** Die Prüfläufe vor den beiden Festschreibungen des Jahres. */
const CHECK_RUNS = [
  {
    id: 2,
    fiscalYear: YEAR,
    cutoffDate: `${YEAR}-06-30`,
    periodType: 'quarter',
    checkedEntries: 37,
    checkedReceipts: 34,
    checkedBankTx: 96,
    timeliness: {
      measuredEntries: 37,
      captureDaysMedian: 3,
      captureDaysMax: 9,
      captureLimitDays: 10,
      lateEntries: 0,
      committedEntries: 37,
      commitDaysMedian: 10,
      commitDaysMax: 10,
      uncommittedEntries: 0,
    },
    findings: [
      {
        id: 4,
        checkRunId: 2,
        rule: 'bank_unmatched',
        severity: 'warning',
        objectType: 'bank_transaction',
        objectId: '4',
        message: 'Ein Bankumsatz vom 26.06.2026 ist keinem offenen Posten zugeordnet.',
        reference: 'GoBD Rz. 36',
      },
    ],
    createdAt: `${YEAR}-07-10T06:14:00Z`,
  },
  {
    id: 1,
    fiscalYear: YEAR,
    cutoffDate: `${YEAR}-03-31`,
    periodType: 'quarter',
    checkedEntries: 19,
    checkedReceipts: 17,
    checkedBankTx: 48,
    timeliness: {
      measuredEntries: 19,
      captureDaysMedian: 2,
      captureDaysMax: 7,
      captureLimitDays: 10,
      lateEntries: 0,
      committedEntries: 19,
      commitDaysMedian: 9,
      commitDaysMax: 9,
      uncommittedEntries: 0,
    },
    findings: [],
    createdAt: `${YEAR}-04-09T05:51:00Z`,
  },
];

/** Zuordnung der bebuchten Konten auf Gliederung und Taxonomie-Element. */
const EBILANZ_MAPPING: Record<string, [string, string]> = {
  '0520': ['Andere Anlagen, Betriebs- und Geschäftsausstattung', 'bs.ass.fixAss.tan.otherEquip'],
  '1200': ['Forderungen aus Lieferungen und Leistungen', 'bs.ass.currAss.receiv.trade'],
  '1406': ['Sonstige Vermögensgegenstände', 'bs.ass.currAss.receiv.other'],
  '1600': ['Kassenbestand, Guthaben bei Kreditinstituten', 'bs.ass.currAss.cashEquiv'],
  '1800': ['Kassenbestand, Guthaben bei Kreditinstituten', 'bs.ass.currAss.cashEquiv'],
  '1900': ['Rechnungsabgrenzungsposten', 'bs.ass.prepaidExp'],
  '2900': ['Gezeichnetes Kapital', 'bs.eqLiab.equity.subscribed'],
  '2970': ['Gewinnvortrag', 'bs.eqLiab.equity.retainedEarnings'],
  '3300': ['Verbindlichkeiten aus Lieferungen und Leistungen', 'bs.eqLiab.liab.trade'],
  '3806': ['Sonstige Verbindlichkeiten, davon aus Steuern', 'bs.eqLiab.liab.other.taxes'],
  '4400': ['Umsatzerlöse', 'is.netIncome.regular.operatingIncome.revenue'],
  '4830': ['Sonstige betriebliche Erträge', 'is.netIncome.regular.operatingIncome.otherOpIncome'],
  '5906': ['Aufwendungen für bezogene Leistungen', 'is.netIncome.regular.opEx.material.purchServices'],
  '6020': ['Löhne und Gehälter', 'is.netIncome.regular.opEx.personnel.wages'],
  '6260': ['Abschreibungen auf Sachanlagen', 'is.netIncome.regular.opEx.depreciation.tangible'],
};

/** Die Auffangposition, in der alles landet, was keine eigene Zeile hat. */
const EBILANZ_FALLBACK: [string, string] = [
  'Sonstige betriebliche Aufwendungen',
  'is.netIncome.regular.opEx.otherOpEx',
];

function mappingReport(year: number) {
  const booked = ACCOUNTS.filter((x) => x.bookingsCount > 0);
  const rows = booked.map((account) => {
    const known = EBILANZ_MAPPING[account.number];
    const [positionLabel, element] = known ?? EBILANZ_FALLBACK;
    return {
      account: account.number,
      name: account.name,
      balance: account.debitSum - account.creditSum,
      positionKey: element,
      positionLabel,
      element,
      // Die Elementnamen sind nach der Systematik gebildet und noch nicht gegen
      // die amtliche Taxonomie geprüft — genau wie im Programm.
      verified: false,
    };
  });
  const fallbackRows = rows.filter((row) => row.element === EBILANZ_FALLBACK[1]);
  return {
    fiscalYear: year || YEAR,
    taxonomyVersion: '6.9',
    taxonomyDate: '2026-01-01',
    taxonomyNote:
      'Die Taxonomie 6.9 gilt für Wirtschaftsjahre ab 2026. Ihre Elementnamen sind vor der ' +
      'ersten Übermittlung gegen die amtliche Fassung abzugleichen.',
    rows,
    blocking: [],
    fallbacks:
      fallbackRows.length > 0
        ? [
            {
              key: EBILANZ_FALLBACK[1],
              label: EBILANZ_FALLBACK[0],
              accounts: fallbackRows.length,
              amount: fallbackRows.reduce((sum, row) => sum + row.balance, 0),
            },
          ]
        : [],
    unverified: rows.length,
    canExport: true,
  };
}

/** Der Abschlussstand des laufenden Geschäftsjahres. */
function closingState(year: number) {
  const fiscalYear = {
    ...(FISCAL_YEARS.find((y) => y.year === (year || YEAR)) ?? FISCAL_YEARS[1]),
    averageEmployees: 6,
    priorYearRevenue: c(412800),
    createdAt: `${YEAR}-01-02T08:00:00Z`,
  };
  return {
    year: fiscalYear.year,
    fiscalYear,
    netIncome: SUMMARY.netIncome,
    hasYearCommitment: false,
    committedUntil: `${YEAR}-06-30`,
    nextYear: fiscalYear.year + 1,
    carriedForward: false,
    carryForwardCurrent: false,
    nextStatus: 'prepared',
    canAdopt: false,
    blocker:
      'Aufgestellt wird erst, wenn das Geschäftsjahr festgeschrieben ist (§ 146 Abs. 4 AO).',
  };
}

/** Die vierzehn Abschlussbausteine, wie sie im geführten Weg stehen. */
const CLOSING_STEP_DEFS: [string, string, string, boolean][] = [
  ['depreciation', 'Abschreibungen',
    'Die planmäßige AfA des Jahres. Sie ist eine Abschlussbuchung und lässt sich später nicht nachholen.', true],
  ['write_up', 'Wertaufholung prüfen',
    'Für jedes Anlagegut mit außerplanmäßiger Abschreibung: ist der Grund weggefallen? Dann ist zuzuschreiben (§ 253 Abs. 5 Satz 1 HGB).', true],
  ['currency_valuation', 'Fremdwährungsbewertung',
    'Offene Posten in Fremdwährung werden zum Stichtagskurs bewertet (§ 256a HGB).', true],
  ['accruals', 'Rechnungsabgrenzung',
    'Ausgaben und Einnahmen, die wirtschaftlich ins nächste Jahr gehören (§ 250 HGB).', false],
  ['provisions', 'Rückstellungen',
    'Verpflichtungen, deren Höhe oder Fälligkeit noch offen ist (§ 249 HGB).', false],
  ['inventory', 'Vorräte',
    'Der Inventurwert zum Stichtag und die Bestandsveränderung, die daraus folgt.', false],
  ['vat_settlement', 'Umsatzsteuer-Verrechnung',
    'Vorsteuer, Umsatzsteuer und Vorauszahlungen werden zu einem Saldo verrechnet.', false],
  ['input_tax_correction', 'Vorsteuerberichtigung § 15a',
    'Hat sich die Verwendung eines Wirtschaftsguts geändert, ist der Vorsteuerabzug anteilig zu berichtigen.', true],
  ['tax_provision', 'Steuerrückstellung',
    'Körperschaftsteuer, Solidaritätszuschlag und Gewerbesteuer auf das Ergebnis des Jahres.', false],
  ['check_run', 'Prüfbericht',
    'Der Prüflauf über Buchungen, Belege und Fristen vor der Festschreibung.', true],
  ['statement', 'Bilanz und GuV',
    'Die Gliederung nach §§ 266 und 275 HGB samt Anhang.', true],
  ['adoption', 'Aufstellen und Feststellen',
    'Die Aufstellung durch die Geschäftsführung und der Beschluss der Gesellschafter.', true],
  ['disclosure', 'E-Bilanz und Offenlegung',
    'Die Übermittlung nach § 5b EStG und die Offenlegung nach § 325 HGB.', true],
  ['appropriation', 'Vortrag und Ergebnisverwendung',
    'Der Saldenvortrag ins Folgejahr und der Beschluss über das Ergebnis.', false],
];

/** Zustand und Begründung je Baustein im Beispieljahr. */
const CLOSING_STEP_STATE: Record<string, { state: string; detail?: string; reason?: string }> = {
  depreciation: { state: 'done', detail: 'AfA für 4 Anlagegüter gebucht' },
  write_up: { state: 'done', detail: 'Keine außerplanmäßige Abschreibung offen' },
  currency_valuation: {
    state: 'skipped',
    reason: 'Im Geschäftsjahr steht kein offener Posten in Fremdwährung.',
  },
  accruals: { state: 'done', detail: '2 Abgrenzungsposten gebildet' },
  provisions: { state: 'done', detail: '1 Rückstellung gebildet, abgezinst' },
  inventory: { state: 'open', detail: 'Kein Inventurwert erfasst' },
  vat_settlement: { state: 'open', detail: 'Zahllast der vier Zeiträume noch nicht verrechnet' },
  input_tax_correction: { state: 'open', detail: '1 Wirtschaftsgut im Berichtigungszeitraum' },
  tax_provision: { state: 'open' },
  check_run: { state: 'open', detail: 'Letzter Lauf zum 30.06.2026' },
  statement: { state: 'open' },
  adoption: { state: 'open', detail: 'Ohne Jahres-Festschreibung nicht möglich' },
  disclosure: { state: 'open' },
  appropriation: { state: 'open' },
};

function closingSteps(year: number) {
  const steps = CLOSING_STEP_DEFS.map(([key, label, hint, automatic], index) => {
    const status = CLOSING_STEP_STATE[key] ?? { state: 'open' };
    return {
      key,
      order: index + 1,
      label,
      hint,
      automatic,
      state: status.state,
      reason: status.reason,
      changedOn: status.state === 'open' ? undefined : `${YEAR}-08-20`,
      detail: status.detail,
    };
  });
  return {
    fiscalYear: year || YEAR,
    cutoff: `${YEAR}-12-31`,
    steps,
    openCount: steps.filter((s) => s.state === 'open').length,
    doneCount: steps.filter((s) => s.state === 'done').length,
    skippedCount: steps.filter((s) => s.state === 'skipped').length,
    total: steps.length,
    reopenable: true,
  };
}

/** Der Vortragsstand ins Folgejahr; gebucht ist noch nichts. */
function carryForwardPreview(toYear: number) {
  const kinds: Record<string, string> = { '1200': 'debitor', '3300': 'kreditor' };
  const openItemCount: Record<string, number> = { '1200': 3, '3300': 2 };
  const rows = ACCOUNTS.filter((x) => x.bookingsCount > 0 && x.statementType === 'Bilanz').map(
    (account) => {
      const balance = account.debitSum - account.creditSum;
      return {
        account: account.number,
        name: account.name,
        kind: kinds[account.number] ?? 'sachkonto',
        closingBalance: balance,
        carried: 0,
        difference: balance,
        openItems: openItemCount[account.number],
      };
    },
  );
  return {
    fromYear: (toYear || YEAR + 1) - 1,
    toYear: toYear || YEAR + 1,
    bookingDate: `${toYear || YEAR + 1}-01-01`,
    deferred: false,
    rows,
    netIncome: SUMMARY.netIncome,
    resultAccount: '2970',
    resultAccountName: 'Gewinnvortrag vor Verwendung',
    alreadyCarried: false,
    needsCorrection: false,
    entries: 3,
    // Die Summe der Vortragswerte ist das Jahresergebnis; die Probe geht auf.
    balanceDifference: 0,
    isBalanced: true,
    accrualReleases: [],
  };
}

/** Ein Rechnungsverbund mit zwei Abschlägen und offener Schlussrechnung. */
const INVOICE_GROUPS = [
  {
    id: 1,
    fiscalYear: YEAR,
    contactId: 2,
    title: 'Lagerleitstand Billstraße, Ausbaustufe 1',
    totalNet: c(84000),
    taxRate: 1900,
    closed: false,
    advances: [
      advance(1, 1, 'RE-2026-0102', `${YEAR}-03-16`, 25200, `${YEAR}-03-30`),
      advance(2, 2, 'RE-2026-0113', `${YEAR}-06-15`, 25200, `${YEAR}-06-29`),
    ],
    progress: {
      agreedNet: c(84000),
      billedNet: c(50400),
      receivedNet: c(50400),
      receivedTax: c(9576),
      receivedGross: c(59976),
      openNet: c(33600),
      closed: false,
    },
    createdAt: `${YEAR}-03-02T09:00:00Z`,
  },
];

function advance(
  id: number,
  invoiceId: number,
  invoiceNumber: string,
  invoiceDate: string,
  netEuro: number,
  settledAt: string,
) {
  const net = c(netEuro);
  const tax = Math.round(net * 0.19);
  return {
    id,
    groupId: 1,
    invoiceId,
    contactId: 2,
    invoiceNumber,
    invoiceDate,
    netAmount: net,
    taxAmount: tax,
    grossAmount: net + tax,
    taxRate: 1900,
    settledAt,
    settlementEntryId: 20 + id,
    cancelled: false,
    settledInFinal: false,
  };
}

/**
 * Das Verzeichnis der Vorsteuerberichtigung nach § 15a UStG.
 *
 * Der Pkw ist 2025 mit vollem Abzug angeschafft; im Beispieljahr sinkt die
 * abzugsberechtigte Verwendung von 100 auf 90 Prozent. Ein Fünftel der
 * Vorsteuer entfällt auf das Jahr, davon zehn Prozent sind zu berichtigen.
 */
function inputTaxCorrections(year: number) {
  const netAmount = c(24500);
  const inputTax = Math.round(netAmount * 0.19);
  const yearlyShare = Math.round(inputTax / 5);
  const amount = -Math.round((yearlyShare * 100) / 1000);
  const correction = {
    id: 1,
    assetId: 1,
    label: 'Pkw, Inventarnummer A-2025-004',
    account: '0520',
    acquisitionDate: '2025-04-14',
    netAmount,
    inputTaxAmount: inputTax,
    originalPermille: 1000,
    immovable: false,
    correctionPeriodYears: 5,
    firstFiscalYear: 2025,
    lastFiscalYear: 2029,
    createdAt: '2025-04-14T10:00:00Z',
    updatedAt: `${YEAR}-08-18T10:00:00Z`,
    usages: [
      {
        correctionId: 1,
        fiscalYear: year || YEAR,
        permille: 900,
        confirmed: true,
        amount,
        updatedAt: `${YEAR}-08-18T10:00:00Z`,
      },
    ],
  };
  return {
    fiscalYear: year || YEAR,
    bookingDate: `${YEAR}-12-31`,
    rows: [
      {
        correction,
        inPeriod: true,
        permille: 900,
        confirmed: true,
        assessment: {
          amount,
          required: true,
          deferToAnnual: true,
          account: '1406',
          reason:
            'Der Berichtigungsbetrag bleibt unter 6.000 €; er wird erst bei der ' +
            'Steuerberechnung für das Kalenderjahr berücksichtigt (§ 44 Abs. 3 UStDV).',
        },
        booked: false,
      },
    ],
    totalAmount: amount,
    unconfirmed: 0,
    note:
      'Der Berichtigungszeitraum beträgt fünf Jahre; bei Grundstücken und Gebäuden sind es zehn ' +
      '(§ 15a Abs. 1 UStG).',
  };
}

/** Der Zustand der zweiten Kette: die des Änderungsprotokolls. */
const AUDIT_CHAIN = {
  isValid: true,
  totalEntries: AUDIT_LOGS.length + 204,
  checkedEntries: AUDIT_LOGS.length + 204,
  breaks: [],
  lastVerifiedHash: fakeHash(`log-${AUDIT_LOGS[0].id}`),
  checkedAt: '2026-08-26T08:12:00Z',
  message: 'Alle Protokolleinträge sind unverändert. Die Kette ist lückenlos.',
};

/** Die Sicherungsläufe: einer beim Beenden, einer täglich. */
const BACKUP_RUNS = [
  backupRun(3, 'automatic', '2026-08-25T18:40:12Z', 'nordlicht-2026-08-25.zip', 214, 48_218_411),
  backupRun(2, 'automatic', '2026-08-24T18:32:05Z', 'nordlicht-2026-08-24.zip', 212, 47_902_188),
  backupRun(1, 'manual', '2026-08-21T09:14:33Z', 'nordlicht-2026-08-21.zip', 209, 47_411_902),
];

function backupRun(
  id: number,
  kind: string,
  startedAt: string,
  file: string,
  fileCount: number,
  bytes: number,
) {
  return {
    id,
    kind,
    startedAt,
    finishedAt: startedAt.replace(/:(\d\d)Z$/, ':5$1Z').slice(0, 20) + 'Z',
    target: `/Users/anna/Sicherungen/${file}`,
    fileCount,
    bytes,
    success: true,
    message: 'Datenbank, Belege, Dokumente und Schlüsseldatei gesichert.',
    programVersion: '0.9.4',
    createdAt: startedAt,
  };
}

/** Das Schlüsselverzeichnis, das mit der Datenüberlassung hinausgeht. */
const KEY_DIRECTORY = [
  {
    category: 'Buchung',
    key: 'kind',
    label: 'Buchungsart',
    description: 'standard, storno, afa, closing, opening, payment',
  },
  {
    category: 'Buchung',
    key: 'source',
    label: 'Herkunft',
    description: 'receipt, invoice, bank, manual, closing, opening',
  },
  {
    category: 'Steuer',
    key: 'taxKey',
    label: 'Steuerschlüssel',
    description: '9 = Vorsteuer 19 %, 3 = Umsatzsteuer 19 %, 0 = ohne Steuer',
  },
  {
    category: 'Beleg',
    key: 'kind',
    label: 'Belegart',
    description: 'invoice, receipt, statement, letter, self_issued, other',
  },
  {
    category: 'Beleg',
    key: 'retentionClass',
    label: 'Aufbewahrungsklasse',
    description: 'books = 10 Jahre, vouchers = 8 Jahre, letters = 6 Jahre',
  },
];

/**
 * Die Verfahrensdokumentation, wie sie auf der Seite „Betriebsprüfung" steht:
 * eine erzeugte Fassung mit PDF daneben.
 */
const PROCEDURE_DOCS = [
  {
    id: 1,
    version: '2026-03-31.1',
    createdAt: `${YEAR}-03-31T09:12:00Z`,
    fiscalYear: YEAR,
    companyName: 'Pfennig Ventures GmbH',
    appVersion: '0.9.0',
    ruleVersion: '2026-01-01',
    actor: 'Anwender',
    fileName: 'Verfahrensdokumentation_Pfennig_Ventures_GmbH_2026-03-31.1.md',
    storedPath: 'dokumente/verfahrensdokumentation/2026-03-31.1.md',
    sha256: '4f2c19a7b83d5e0116c7d94af5b2c8e37a1d6b90f3c25e84d7a0b1c6e9f38d2a',
    size: 31_482,
    pdfFileName: 'Verfahrensdokumentation_Pfennig_Ventures_GmbH_2026-03-31.1.pdf',
    pdfStoredPath: 'dokumente/verfahrensdokumentation/2026-03-31.1.pdf',
    pdfSha256: '9b1e7c4d3a6f8c2d5e0a3b61c74e8d1a5b2f7c0e93d6a4b8c1e7f2059d3a6b47',
    pdfSize: 214_907,
  },
];

/**
 * Die Versionshistorie, wie sie aus dem Programm kommt: zerlegt in Fassungen.
 */
const CHANGELOG = [
  {
    version: 'v0.1',
    date: '2026-09-06',
    summary:
      'Erste Fassung. Buchfink führt die doppelte Buchführung einer Kapitalgesellschaft vom Beleg bis zur E-Bilanz, auf dem eigenen Rechner und ohne Konto bei irgendwem.',
    changes: [
      'Doppelte Buchführung auf dem SKR04: den Buchungssatz und die Steuer rechnet Buchfink aus dem Beleg, der Rechnung oder der zugeordneten Zahlung.',
      'Ausgangsrechnungen als ZUGFeRD-konformes PDF/A-3 oder als XRechnung im CII-Profil, Nummer und Buchung in einer Transaktion.',
      'Bankauszüge im Format CAMT.053 mit Zuordnungsvorschlag zum offenen Posten; gebucht wird nach Bestätigung.',
      'Unveränderbarkeit über Hash-Ketten, Festschreibung mit Zeitstempel nach RFC 3161 und Prüfbericht vor jeder Festschreibung.',
      'Datenüberlassung als Z3-Export, Prüferpaket, Prüfermodus und eine Verfahrensdokumentation aus dem laufenden System.',
    ],
  },
];

/**
 * Die Muster, mit denen Buchfink die Freitexte vorbelegt. Die Ansicht vergleicht
 * sie mit den erfassten Texten und sagt, welcher Abschnitt noch im Muster steht.
 */
const ORGANISATION_DEFAULTS = {
  responsibilities:
    'Die Buchführung wird von der Inhaberin bzw. dem Inhaber des Unternehmens selbst geführt.',
  receiptFlow:
    'Eingehende Belege werden unmittelbar nach Eingang in Buchfink abgelegt und dort erfasst.',
  scanning: 'Papierbelege werden in Farbe und mindestens 300 dpi als PDF erfasst.',
  approval: 'Buchung und Freigabe liegen in einer Hand.',
  substitution: 'Im Verhinderungsfall übernimmt der steuerliche Berater den Zugriff.',
  backup: 'Die Sicherung läuft nach dem hinterlegten Rhythmus auf ein Ziel außerhalb des Datenordners.',
  notes: '',
};

/** Die Freitexte der Organisationsanweisung, wie sie eingerichtet aussehen. */
const ORGANISATION_TEXTS = {
  responsibilities: 'Die Geschäftsführung führt die Buchführung selbst und verantwortet sie.',
  receiptFlow:
    'Eingangsbelege kommen per E-Mail oder auf Papier, werden am Tag des Eingangs abgelegt und wöchentlich gebucht.',
  scanning: 'Papierbelege werden mit 300 dpi in Farbe eingescannt und als PDF abgelegt.',
  approval: 'Gebucht und festgeschrieben wird von der Geschäftsführung nach dem Prüflauf.',
  substitution: 'Im Verhinderungsfall übernimmt die Steuerkanzlei.',
  backup: 'Die Sicherung läuft täglich auf ein zweites Laufwerk und wird monatlich zurückgespielt.',
  notes: '',
};

/** Hinweise zu Rechtsform, Speicherort und Steuerfällen. */
const COMPLIANCE_HINTS = {
  legalFormNote: '',
  cloudWarning: '',
  dataDir: '/Users/anwender/Buchfink/Pfennig Ventures GmbH',
  systemChangeDate: '',
  systemChangeNote: '',
  taxCaseHints: [
    'Organschaft und Konsolidierung bildet Buchfink nicht ab.',
    'Land- und Forstwirtschaft nach § 13a EStG ist nicht abgedeckt.',
  ],
};

/** Die Fristen je Geschäftsjahr. Gelöscht werden darf noch keines. */
function retentionOverview() {
  return {
    today: TODAY,
    years: [2024, 2025, YEAR].map((y) => retentionYear(y)),
    concept: [
      {
        category: 'Bücher, Abschlüsse, Verfahrensdokumentation',
        class: 'books',
        years: 10,
        legalBasis: '§ 147 Abs. 3 AO, § 257 Abs. 4 HGB',
        note: 'Die Frist beginnt am 31.12. des Jahres der letzten Eintragung.',
      },
      {
        category: 'Buchungsbelege und Rechnungen',
        class: 'vouchers',
        years: 8,
        legalBasis: '§ 147 Abs. 3 Satz 1 AO',
        note: 'Seit 2025 acht statt zehn Jahre.',
      },
      {
        category: 'Handels- und Geschäftsbriefe',
        class: 'letters',
        years: 6,
        legalBasis: '§ 147 Abs. 3 AO',
        note: 'Empfangene und abgesandte Schreiben.',
      },
    ],
  };
}

function retentionYear(fiscalYear: number) {
  const counts = {
    journalEntries: fiscalYear === YEAR ? 55 : 148,
    journalLines: fiscalYear === YEAR ? 171 : 452,
    receipts: fiscalYear === YEAR ? 6 : 84,
    receiptFiles: fiscalYear === YEAR ? 7 : 91,
    festschreibungen: fiscalYear === YEAR ? 2 : 5,
    checkRuns: fiscalYear === YEAR ? 2 : 5,
    vatReturns: fiscalYear === YEAR ? 2 : 4,
    invoices: fiscalYear === YEAR ? 19 : 46,
    bankTransactions: fiscalYear === YEAR ? 144 : 388,
    assetMovements: fiscalYear === YEAR ? 4 : 9,
    accruals: 2,
    provisions: 1,
    inventoryCounts: 0,
    zmReturns: 0,
    inputTaxUsages: 1,
    numberGaps: 0,
    clearedReferences: 0,
  };
  const classes = [
    row('books', 'Bücher und Abschlüsse', 10),
    row('vouchers', 'Buchungsbelege und Rechnungen', 8),
    row('letters', 'Handels- und Geschäftsbriefe', 6),
  ];
  function row(cls: string, label: string, years: number) {
    const end = `${fiscalYear + years}-12-31`;
    return {
      class: cls,
      label,
      years,
      retentionEnd: end,
      earliestDeletion: `${fiscalYear + years + 1}-01-01`,
      legalBasis: cls === 'books' ? '§ 147 Abs. 3 AO, § 257 Abs. 4 HGB' : '§ 147 Abs. 3 AO',
    };
  }
  return {
    fiscalYear,
    counts,
    classes,
    earliestDeletion: `${fiscalYear + 11}-01-01`,
    expired: false,
    deletable: false,
    note: `Die längste Frist läuft bis zum 31.12.${fiscalYear + 10}.`,
  };
}


/**
 * Die Steuertermine des Jahres. Erledigt ist, was aus den Daten folgt — die
 * übermittelte Voranmeldung, die Festschreibung —, und kein gesetzter Haken.
 */
const DEADLINES = [
  deadline('ustva-2026-q1', 'Umsatzsteuer-Voranmeldung 1. Quartal 2026', `${YEAR}-04-10`,
    '1. Quartal 2026', '§ 18 Abs. 1 UStG', true),
  deadline('ustva-2026-q2', 'Umsatzsteuer-Voranmeldung 2. Quartal 2026', `${YEAR}-07-10`,
    '2. Quartal 2026', '§ 18 Abs. 1 UStG', true),
  deadline('ustva-2026-q3', 'Umsatzsteuer-Voranmeldung 3. Quartal 2026', `${YEAR}-10-10`,
    '3. Quartal 2026', '§ 18 Abs. 1 UStG', false),
  deadline('zm-2026-q3', 'Zusammenfassende Meldung 3. Quartal 2026', `${YEAR}-10-25`,
    '3. Quartal 2026', '§ 18a Abs. 1 UStG', false),
  deadline('festschreibung-2026-q3', 'Drittes Quartal festschreiben', `${YEAR}-10-10`,
    '3. Quartal 2026', '§ 146 Abs. 4 AO', false),
  deadline('aufstellung-2025', 'Jahresabschluss 2025 aufstellen', `${YEAR}-06-30`,
    'Geschäftsjahr 2025', '§ 264 Abs. 1 HGB', true),
  deadline('offenlegung-2025', 'Jahresabschluss 2025 offenlegen', `${YEAR + 1}-12-31`,
    'Geschäftsjahr 2025', '§ 325 Abs. 1a HGB', false),
];

function deadline(
  key: string,
  title: string,
  dueDate: string,
  period: string,
  reference: string,
  isDone: boolean,
) {
  return {
    key,
    title,
    dueDate,
    period,
    reference,
    description: '',
    fiscalYear: YEAR,
    isDone,
    doneOn: isDone ? dueDate : undefined,
  };
}

/**
 * Der Gründungsstand. Die Beispielfirma ist seit 2024 eingetragen; der
 * Gründungsweg ist damit abgeschlossen und die Ansicht zeigt ihn nicht mehr.
 */
const FOUNDATION_STATE = {
  applies: true,
  hasFoundation: false,
  legalForm: 'GmbH',
  rules: {},
  stage: 'eingetragen',
  duties: [],
  postingsBooked: true,
};

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
      // Passend zum jüngsten Lauf in BACKUP_RUNS: ohne beides zeigte die Seite
      // „Noch keine Sicherung gelaufen" über einer Liste gelaufener Sicherungen.
      backupDir: '/Users/anna/Sicherungen',
      lastBackupAt: '2026-08-25T18:40:00Z',
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
  PreviewDepreciationPlan: (request: any) => later(previewPlan(request), 60),
  GetDepreciationRun: () => later(DEPRECIATION_RUN),
  BookDepreciationRun: unsupported('BookDepreciationRun'),
  BookAssetImpairment: unsupported('BookAssetImpairment'),
  BookAssetWriteUp: unsupported('BookAssetWriteUp'),
  TransferFixedAsset: unsupported('TransferFixedAsset'),
  PreviewAssetDisposal: (request: any) => later(disposalPreview(request), 80),
  BookAssetMaintenance: unsupported('BookAssetMaintenance'),
  BookAssetIncome: unsupported('BookAssetIncome'),
  ValuateAssetCurrency: (request: any) => later(currencyValuation(request), 60),
  BookAssetCurrencyValuation: unsupported('BookAssetCurrencyValuation'),
  SelectAssetDocumentsDialog: unsupported('SelectAssetDocumentsDialog'),
  AttachAssetDocument: unsupported('AttachAssetDocument'),
  RemoveAssetDocument: unsupported('RemoveAssetDocument'),
  GetAssetDocumentContent: unsupported('GetAssetDocumentContent'),
  GetAssetDocumentKinds: () => later(ASSET_DOCUMENT_KINDS),
  GetExpiringAssetDocuments: () => later([]),
  GetLegalForms: () => later(LEGAL_FORMS),
  GetInvestmentRules: () => later(INVESTMENT_RULES),
  ComputeVorabpauschale: (request: any) => later(vorabpauschale(request), 60),
  GetInvestmentNoteForIncome: (assetId: number, amount: number) =>
    later(investmentNote(assetId, amount, false), 60),
  DisposeFixedAsset: unsupported('DisposeFixedAsset'),
  GetAnlagenspiegel: () => later(ANLAGENSPIEGEL),
  GetAssetAcquisitionCandidates: () => later(ASSET_CANDIDATES),
  GetSammelposten: (fiscalYear: number) =>
    later(ASSETS.find((a) => a.method === 'pool' && a.poolYear === (fiscalYear || YEAR)) ?? null),

  // Nachgereichte Auskünfte, die eine Ansicht auf dem Weg von der Bilanz zum
  // Beleg nebenbei holt. Ohne sie bräche die Belegliste ab: sie lädt Belege,
  // Kontakte und Kontierung in einem Zug.
  GetFiscalYears: () => later(FISCAL_YEARS),
  GetSelectableContacts: () => later(CONTACTS),
  GetSuSaOverviewAt: () => later(susa()),
  GetOpenItemsAging: unsupported('GetOpenItemsAging'),
  GetPaymentAllocations: () => later([]),
  GetProvisions: () => later([]),
  GetAdvanceTargets: () => later([]),
  GetOpenAdvances: () => later([]),
  GetOpenVendorAdvances: () => later([]),
  ExportJournalCSV: unsupported('ExportJournalCSV'),

  // Jahresabschluss: Bilanz, GuV, Anhang, Größenklasse
  GetStatement: () => later(STATEMENT, 80),
  GetSizeClass: () => later(SIZE_CLASS),
  GetStatementDeadlines: () => later([]),
  ExportStatementPDF: unsupported('ExportStatementPDF'),
  ExportStatementCSV: unsupported('ExportStatementCSV'),

  // Auswahllisten des Rechnungsdialogs. Ohne sie scheitert das Laden der
  // Rechnungsseite in einem Zug, und die Maske bliebe ohne Kunden.
  GetUnitCodes: () => later(UNIT_CODES),
  GetEInvoiceProfiles: () => later(EINVOICE_PROFILES),
  GetInvoiceSentViaOptions: () => later(SENT_VIA_OPTIONS),
  GetNumberGapReasons: () => later(NUMBER_GAP_REASONS),
  GetInvoiceNumberGaps: () => later({ fiscalYear: YEAR, issued: 19, used: 19, gaps: [] }),
  GetSupplyEvidenceReport: () =>
    later({
      fiscalYear: YEAR,
      rows: [],
      incomplete: 0,
      note: 'Im Geschäftsjahr sind keine innergemeinschaftlichen Lieferungen abgerechnet.',
    }),

  // Umsatzsteuer: das Kennziffernblatt, das der Monatsabschluss zeigt.
  // Geschrieben wird hier nichts — Bestätigung und Berichtigung sind Vorgänge
  // mit Folgen und stehen deshalb als „nicht verfügbar" da.
  GetVatReturn: (periodKey: string) => later(vatReturn(periodKey), 60),
  SaveVatReturn: (periodKey: string) => later(vatReturn(periodKey), 60),
  ExportVatReturnCSV: () => later('Kennziffer;Bemessungsgrundlage;Steuer\n81;315200;59888\n'),
  ConfirmVatReturnSubmitted: unsupported('ConfirmVatReturnSubmitted'),

  // Welle 7: Aufgaben, Monatsabschluss, Bankvorschlag, Mahnwesen, Prüfpfad
  GetTasks: () => later(TASKS, 60),
  GetMonthCloseState: (month: string) => later(monthCloseState(month), 60),
  SuggestBankMatches: (bankTxId: number) => later(bankSuggestions(bankTxId), 60),
  GetBankRules: () => later(BANK_RULES),
  DeleteBankRule: unsupported('DeleteBankRule'),
  GetDunningProposals: () => later(DUNNING_PROPOSALS, 60),
  CreateDunningNotices: unsupported('CreateDunningNotices'),
  GetDunningNotices: (contactId: number) =>
    later(contactId ? DUNNING_NOTICES.filter((n) => n.contactId === contactId) : DUNNING_NOTICES),
  GetBaseRates: () => later(BASE_RATES),
  SaveBaseRate: unsupported('SaveBaseRate'),
  GetAuditTrail: (receiptId: number) => later(auditTrail(receiptId), 60),
  ExportAuditTrail: unsupported('ExportAuditTrail'),
  SaveServiceProof: unsupported('SaveServiceProof'),
  GetPaymentTermNotice: (dueDays: number) => later(paymentTermNotice(dueDays)),

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

  // Welle 5b bis 7: Meldezeiträume, Prüfläufe, Abschlussweg, Nebenpflichten
  GetVatPeriods: () => later(vatPeriods()),
  GetVatReturns: () => later(savedVatReturns()),
  GetZMPeriods: () => later(zmPeriods()),
  GetZMReturn: (periodKey: string) => later(zmReturnFor(periodKey), 60),
  GetZMReturns: () => later([]),
  GetCheckRuns: () => later(CHECK_RUNS),
  GetEBilanzMappingReport: (year: number) => later(mappingReport(year), 80),
  GetOpenItemsAt: () => later(OPEN_ITEMS),
  GetClosingState: (year: number) => later(closingState(year), 60),
  GetClosingSteps: (year: number) => later(closingSteps(year), 60),
  GetCarryForwardPreview: (toYear: number) => later(carryForwardPreview(toYear), 60),
  GetInvoiceGroups: () => later(INVOICE_GROUPS),
  GetInputTaxCorrections: (year: number) => later(inputTaxCorrections(year), 60),
  GetDeadlines: () => later(DEADLINES, 60),
  GetFoundationState: () => later(FOUNDATION_STATE),

  // Nachweise, Sicherung und Datenüberlassung
  GetAuditLogsFiltered: () => later(AUDIT_LOGS, 60),
  VerifyAuditChain: () => later(AUDIT_CHAIN, 200),
  GetRetentionOverview: () => later(retentionOverview(), 80),
  GetRetentionHolds: () => later([]),
  GetExpiredObjects: () => later([]),
  GetBackupRuns: () => later(BACKUP_RUNS),
  GetKeyDirectory: () => later(KEY_DIRECTORY),
  // Die Seite „Betriebsprüfung" liest die Verfahrensdokumentation beim Öffnen:
  // ohne Antwort stünde auf dem Bild für die Webseite eine Fehlermeldung.
  GetProcedureDocumentations: () => later(PROCEDURE_DOCS, 60),
  GetOrganisationTexts: () => later(ORGANISATION_TEXTS),
  GetOrganisationTextDefaults: () => later(ORGANISATION_DEFAULTS),
  GetComplianceHints: () => later(COMPLIANCE_HINTS),
  GetChangeLog: () => later(CHANGELOG),

  // Vorgänge mit Folgen — Schreiben, Dialoge, Übermittlungen. In der
  // Vorschau steht dafür eine Antwort und kein Loch (siehe README).
  RegenerateInvoiceDocument: unsupported('RegenerateInvoiceDocument'),
  CancelInvoiceWithDocument: unsupported('CancelInvoiceWithDocument'),
  CorrectInvoice: unsupported('CorrectInvoice'),
  MarkInvoiceSent: unsupported('MarkInvoiceSent'),
  RecordInvoiceNumberGapReason: unsupported('RecordInvoiceNumberGapReason'),
  CreateInvoiceGroup: unsupported('CreateInvoiceGroup'),
  IssueAdvanceInvoice: unsupported('IssueAdvanceInvoice'),
  SettleAdvance: unsupported('SettleAdvance'),
  RefundAdvance: unsupported('RefundAdvance'),
  IssueFinalInvoice: unsupported('IssueFinalInvoice'),
  WriteOffOpenItem: unsupported('WriteOffOpenItem'),
  GetLegacySpecialDepreciations: unsupported('GetLegacySpecialDepreciations'),
  SetAverageEmployees: unsupported('SetAverageEmployees'),
  SetPriorYearRevenue: unsupported('SetPriorYearRevenue'),
  CreateFiscalYear: unsupported('CreateFiscalYear'),
  CarryForward: unsupported('CarryForward'),
  SetFiscalYearStatus: unsupported('SetFiscalYearStatus'),
  ReopenFiscalYear: unsupported('ReopenFiscalYear'),
  SkipClosingStep: unsupported('SkipClosingStep'),
  ReopenClosingStep: unsupported('ReopenClosingStep'),
  MarkClosingStepDone: unsupported('MarkClosingStepDone'),
  ProposeAccruals: unsupported('ProposeAccruals'),
  PreviewAccrual: unsupported('PreviewAccrual'),
  BookAccrual: unsupported('BookAccrual'),
  GetAccruals: unsupported('GetAccruals'),
  GetAccrualReport: unsupported('GetAccrualReport'),
  PreviewProvision: unsupported('PreviewProvision'),
  BookProvisionFormation: unsupported('BookProvisionFormation'),
  BookProvisionIncrease: unsupported('BookProvisionIncrease'),
  BookProvisionRelease: unsupported('BookProvisionRelease'),
  BookProvisionConsumption: unsupported('BookProvisionConsumption'),
  BookProvisionUnwinding: unsupported('BookProvisionUnwinding'),
  SettleProvision: unsupported('SettleProvision'),
  GetProvisionMirror: unsupported('GetProvisionMirror'),
  GetDiscountRates: unsupported('GetDiscountRates'),
  GetDiscountRateMonths: unsupported('GetDiscountRateMonths'),
  SaveDiscountRates: unsupported('SaveDiscountRates'),
  ImportDiscountRatesCSV: unsupported('ImportDiscountRatesCSV'),
  GetInventoryAccounts: unsupported('GetInventoryAccounts'),
  PreviewInventory: unsupported('PreviewInventory'),
  BookInventory: unsupported('BookInventory'),
  PreviewVatSettlement: unsupported('PreviewVatSettlement'),
  BookVatSettlement: unsupported('BookVatSettlement'),
  PreviewTaxProvision: unsupported('PreviewTaxProvision'),
  BookTaxProvision: unsupported('BookTaxProvision'),
  PreviewAppropriation: unsupported('PreviewAppropriation'),
  BookAppropriation: unsupported('BookAppropriation'),
  GetAppropriation: unsupported('GetAppropriation'),
  GetNotesTexts: unsupported('GetNotesTexts'),
  SaveNotesText: unsupported('SaveNotesText'),
  GetClosingSettings: unsupported('GetClosingSettings'),
  SaveClosingSettings: unsupported('SaveClosingSettings'),
  GetTaxElectionRegister: unsupported('GetTaxElectionRegister'),
  ExportTaxElectionRegisterCSV: unsupported('ExportTaxElectionRegisterCSV'),
  GetReconciliation: unsupported('GetReconciliation'),
  CreateVatCorrection: unsupported('CreateVatCorrection'),
  GetSpecialPrepaymentSuggestion: unsupported('GetSpecialPrepaymentSuggestion'),
  SaveZMReturn: unsupported('SaveZMReturn'),
  ConfirmZMSubmitted: unsupported('ConfirmZMSubmitted'),
  CreateZMCorrection: unsupported('CreateZMCorrection'),
  ExportZMCSV: unsupported('ExportZMCSV'),
  RunChecks: unsupported('RunChecks'),
  MarkDeadlineDone: unsupported('MarkDeadlineDone'),
  SaveFoundation: unsupported('SaveFoundation'),
  GetFoundationRules: unsupported('GetFoundationRules'),
  GetRecommendedVatPeriod: unsupported('GetRecommendedVatPeriod'),
  PreviewFoundationPostings: unsupported('PreviewFoundationPostings'),
  BookFoundationPostings: unsupported('BookFoundationPostings'),
  RegisterCompany: unsupported('RegisterCompany'),
  CompleteFoundationDuty: unsupported('CompleteFoundationDuty'),
  ExportZ3: unsupported('ExportZ3'),
  ExportArchive: unsupported('ExportArchive'),
  ExportAuditPackage: unsupported('ExportAuditPackage'),
  ExportKeyDirectory: unsupported('ExportKeyDirectory'),
  SelectExportDirectoryDialog: unsupported('SelectExportDirectoryDialog'),
  SelectSaveFileDialog: unsupported('SelectSaveFileDialog'),
  SaveReceiptFileAs: unsupported('SaveReceiptFileAs'),
  VerifyReceiptFiles: unsupported('VerifyReceiptFiles'),
  GetAccountLedgerRange: unsupported('GetAccountLedgerRange'),
  ImportCAMT053File: unsupported('ImportCAMT053File'),
  SelectStatementFileDialog: unsupported('SelectStatementFileDialog'),
  SetBackupDir: unsupported('SetBackupDir'),
  CreateBackup: unsupported('CreateBackup'),
  VerifyBackup: unsupported('VerifyBackup'),
  RestoreFromBackup: unsupported('RestoreFromBackup'),
  SelectBackupDirDialog: unsupported('SelectBackupDirDialog'),
  SelectBackupFileDialog: unsupported('SelectBackupFileDialog'),
  PreviewInputTaxCorrection: unsupported('PreviewInputTaxCorrection'),
  RegisterInputTaxCorrection: unsupported('RegisterInputTaxCorrection'),
  CloseInputTaxCorrection: unsupported('CloseInputTaxCorrection'),
  SaveInputTaxUsage: unsupported('SaveInputTaxUsage'),
  BookInputTaxCorrection: unsupported('BookInputTaxCorrection'),
  GetVatIDChecks: unsupported('GetVatIDChecks'),
  GetVatIDStatus: unsupported('GetVatIDStatus'),
  CheckVatID: unsupported('CheckVatID'),
  GetSupplyEvidenceKinds: unsupported('GetSupplyEvidenceKinds'),
  GetSupplyEvidence: unsupported('GetSupplyEvidence'),
  AddSupplyEvidence: unsupported('AddSupplyEvidence'),
  SetSupplyTransport: unsupported('SetSupplyTransport'),
  RemoveSupplyEvidence: unsupported('RemoveSupplyEvidence'),
  GetNonDeductibleCategories: unsupported('GetNonDeductibleCategories'),
  GetNonDeductibleReport: unsupported('GetNonDeductibleReport'),
  RebookGiftsForRecipient: unsupported('RebookGiftsForRecipient'),
  GetExchangeRate: unsupported('GetExchangeRate'),
  GetExchangeRates: unsupported('GetExchangeRates'),
  SaveExchangeRate: unsupported('SaveExchangeRate'),
  GetVatExchangeRates: unsupported('GetVatExchangeRates'),
  SaveVatExchangeRate: unsupported('SaveVatExchangeRate'),
  ImportVatExchangeRatesCSV: unsupported('ImportVatExchangeRatesCSV'),
  PreviewCurrencyValuation: unsupported('PreviewCurrencyValuation'),
  BookCurrencyValuation: unsupported('BookCurrencyValuation'),
  GetAfaRules: unsupported('GetAfaRules'),
  GetWriteUpReport: unsupported('GetWriteUpReport'),
  ConfirmImpairmentPersists: unsupported('ConfirmImpairmentPersists'),
  GetPoolConsistencyReport: unsupported('GetPoolConsistencyReport'),
  CheckNearAcquisitionCost: unsupported('CheckNearAcquisitionCost'),
  CapitalizeNearAcquisitionCost: unsupported('CapitalizeNearAcquisitionCost'),
  GetExemptionCertificateWarnings: unsupported('GetExemptionCertificateWarnings'),
  GetServiceEndpoints: unsupported('GetServiceEndpoints'),
  SaveServiceEndpoints: unsupported('SaveServiceEndpoints'),
  GetSchemaMigrations: unsupported('GetSchemaMigrations'),
  GetMigrationRecords: unsupported('GetMigrationRecords'),
  SetSystemChangeDate: unsupported('SetSystemChangeDate'),
  SetRetentionHold: unsupported('SetRetentionHold'),
  ReleaseRetentionHold: unsupported('ReleaseRetentionHold'),
  ArchiveAndDeleteFiscalYear: unsupported('ArchiveAndDeleteFiscalYear'),
  GenerateProcedureDocumentation: unsupported('GenerateProcedureDocumentation'),
  SaveOrganisationTexts: unsupported('SaveOrganisationTexts'),
  SaveProcedureDocumentationAs: unsupported('SaveProcedureDocumentationAs'),
  OpenReleasesPage: unsupported('OpenReleasesPage'),
  BlockContact: unsupported('BlockContact'),
  SaveReceiptHeader: unsupported('SaveReceiptHeader'),
  CorrectEntry: unsupported('CorrectEntry'),
  PreviewOpeningBalance: unsupported('PreviewOpeningBalance'),
  BookOpeningBalance: unsupported('BookOpeningBalance'),
  EnableReadOnly: unsupported('EnableReadOnly'),
  DisableReadOnly: unsupported('DisableReadOnly'),
  GetProgramVersion: unsupported('GetProgramVersion'),

  // Welle 8: die laufende Buchhaltung. Gelesen wird aus denselben
  // Beispieldaten wie das Journal — geschrieben wird nichts: die Handbuchung,
  // der Eigenbeleg, das eigene Konto und die Fristverlängerung sind Vorgänge
  // mit Folgen und stehen deshalb als „nicht verfügbar" da.
  GetRetentionRules: () => later(RETENTION_RULES),
  GetVatRatePeriods: () => later(VAT_RATE_PERIODS),
  GetVatPeriodProposal: (year: number) => later(vatPeriodProposal(year)),
  GetCustomAccounts: () => later(CUSTOM_ACCOUNTS),
  GetStatementPositions: () => later(STATEMENT_POSITIONS),
  GetEntriesForReceipt: (receiptId: number) =>
    later(ENTRIES.filter((e: any) => e.receiptId === receiptId)),
  GetReceiptFindings: (receiptId: number) => later(receiptFindings(receiptId)),
  GetFilteredJournal: (filter: any) => later(filteredJournal(filter)),
  GetFilteredJournalCSV: () => later(filteredJournalCSV()),
  SaveFilteredJournalCSV: unsupported('SaveFilteredJournalCSV'),
  PostManualEntry: unsupported('PostManualEntry'),
  CreateSelfIssuedReceipt: unsupported('CreateSelfIssuedReceipt'),
  OverrideReceiptRetention: unsupported('OverrideReceiptRetention'),
  CreateCustomAccount: unsupported('CreateCustomAccount'),
  SetAccountBlocked: unsupported('SetAccountBlocked'),
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

  // Der fehlende Leistungsnachweis, wie ihn das Backend in der Vorschau meldet
  // (RECH-08). Die Maske macht daraus ab der Grenze ein Pflichtfeld — geprüft
  // wird gegen die Bestellung, und das geschieht vor dem Buchen oder gar nicht.
  const warnings: any[] = [];
  const receipt = RECEIPTS.find((r) => r.id === request.receiptId) as any;
  const threshold = SETTINGS.invoiceCheckThreshold;
  if (receipt && !receipt.serviceProof && threshold > 0 && gross >= threshold) {
    warnings.push({
      code: 'service_proof_required',
      severity: 'warning',
      title: 'Der Leistungsnachweis fehlt',
      detail:
        `Beleg ${receipt.receiptNumber} über ${euro(gross)} € trägt keinen Leistungsnachweis. ` +
        `Ab ${euro(threshold)} € verlangt die eigene Vorgabe den Vermerk, gegen welche Bestellung ` +
        'geprüft wurde. Ohne ihn wird der Beleg nicht gebucht.',
    });
  }

  return { lines, net, tax, gross, balanced: true, warnings };
}


// -------------------------------------------------------------------------
// Welle 8: Fristen, Steuersätze, eigene Konten, Beanstandungen, Journalfilter

/** Die Fristentabelle, wie sie das Backend aus `retention_rules.json` lädt. */
const RETENTION_RULES = {
  version: '2026.1',
  validFrom: '2025-01-01',
  source: 'HGB und AO in der Fassung des Vierten Bürokratieentlastungsgesetzes',
  note: 'Die verkürzte Frist für Buchungsbelege gilt für Unterlagen, deren Frist am 1. Januar 2025 noch lief.',
  classes: [
    {
      class: 'books',
      label: 'Handelsbücher und Abschlüsse',
      years: 10,
      legalBasis: '§ 257 Abs. 4 HGB, § 147 Abs. 3 AO',
      objects: ['Journal', 'Jahresabschluss', 'Inventar'],
      previous: null,
    },
    {
      class: 'vouchers',
      label: 'Buchungsbelege und Rechnungen',
      years: 8,
      legalBasis: '§ 257 Abs. 4 HGB, § 147 Abs. 3 AO',
      objects: ['Beleg', 'Rechnung', 'Kontoauszug'],
      previous: { years: 10, replacedFrom: '2025-01-01', note: 'bis zum Vierten Bürokratieentlastungsgesetz' },
    },
    {
      class: 'letters',
      label: 'Handelsbriefe und sonstige Unterlagen',
      years: 6,
      legalBasis: '§ 257 Abs. 4 HGB, § 147 Abs. 3 AO',
      objects: ['Handelsbrief', 'Angebot', 'Bestellung'],
      previous: null,
    },
  ],
};

/** Die datierte Tabelle der Steuersätze (UNV-03 K2). */
const VAT_RATE_PERIODS = [
  { validFrom: '2007-01-01', standard: 1900, reduced: 700, source: '§ 12 UStG' },
  { validFrom: '2020-07-01', standard: 1600, reduced: 500, source: 'Zweites Corona-Steuerhilfegesetz' },
  { validFrom: '2021-01-01', standard: 1900, reduced: 700, source: '§ 12 UStG' },
];

/** Der Vorschlag zum Voranmeldungszeitraum aus der Steuer des Vorjahres. */
function vatPeriodProposal(year: number) {
  const target = year || YEAR;
  return {
    year: target,
    basedOnYear: target - 1,
    priorYearTax: c(11_480),
    current: 'quarter',
    proposed: 'month',
    changes: true,
    complete: true,
    missingPeriods: 0,
    reference: '§ 18 Abs. 2 UStG',
    note:
      `Die Steuer des Jahres ${target - 1} lag über 9.000 Euro; ab ${target} ist monatlich ` +
      'anzumelden. Das Finanzamt setzt den Zeitraum fest — Buchfink schlägt ihn nur vor.',
  };
}

/** Zwei selbst angelegte Konten im freien Bereich des SKR04 (BEL-06 K2). */
const CUSTOM_ACCOUNTS = [
  {
    ...(ACCOUNTS[0] as any),
    id: 9001,
    number: '6899',
    name: 'Werkstattverbrauch Prototypen',
    posten: 'Sonstige betriebliche Aufwendungen',
    positionId: 'guv_sonstige_aufwendungen',
    statementType: 'guv',
    isCustom: true,
    isActive: true,
    bookingsCount: 14,
    debitSum: c(4_318.9),
    creditSum: 0,
    balance: c(4_318.9),
  },
  {
    ...(ACCOUNTS[0] as any),
    id: 9002,
    number: '6898',
    name: 'Messebeteiligung Nord',
    posten: 'Sonstige betriebliche Aufwendungen',
    positionId: 'guv_sonstige_aufwendungen',
    statementType: 'guv',
    isCustom: true,
    isActive: false,
    bookingsCount: 3,
    debitSum: c(1_190),
    creditSum: 0,
    balance: c(1_190),
  },
];

/** Die Gliederungspositionen, unter denen ein eigenes Konto stehen darf. */
const STATEMENT_POSITIONS = [
  {
    id: 'guv_sonstige_aufwendungen',
    name: 'Sonstige betriebliche Aufwendungen',
    statementType: 'guv',
    balanceSide: 'S',
    hgbCode: '§ 275 Abs. 2 Nr. 8 HGB',
    accountType: 'expense',
  },
  {
    id: 'guv_sonstige_ertraege',
    name: 'Sonstige betriebliche Erträge',
    statementType: 'guv',
    balanceSide: 'H',
    hgbCode: '§ 275 Abs. 2 Nr. 4 HGB',
    accountType: 'revenue',
  },
  {
    id: 'bilanz_sonstige_vermoegen',
    name: 'Sonstige Vermögensgegenstände',
    statementType: 'bilanz',
    balanceSide: 'S',
    hgbCode: '§ 266 Abs. 2 B. II. 4. HGB',
    accountType: 'asset',
  },
];

/**
 * Die Beanstandungen eines Belegs nach Fehlerklassen (RECH-07 K2).
 *
 * Der Beleg mit strukturiertem Teil trägt einen Geschäftsregel- und einen
 * Inhaltsfehler, jeder andere keinen — geprüft wird, was geprüft werden kann.
 */
function receiptFindings(receiptId: number) {
  const receipt = RECEIPTS.find((r: any) => r.id === receiptId) as any;
  const checked = Boolean(receipt?.validatedAt);
  const groups = checked
    ? [
        {
          class: 'business_rule',
          label: 'Geschäftsregelfehler',
          findings: [
            {
              class: 'business_rule',
              rule: 'BR-DE-15',
              severity: 'warning',
              where: '',
              message: 'Die Bestellreferenz des Käufers fehlt.',
              norm: 'CIUS XRechnung 3.0',
              inputTaxEffect: 'Ohne Folge für den Vorsteuerabzug.',
              blocking: false,
            },
          ],
        },
        {
          class: 'content',
          label: 'Inhaltsfehler',
          findings: [
            {
              class: 'content',
              rule: 'input_tax_service_date',
              severity: 'fatal',
              where: 'Kopfdaten',
              message: 'Der Zeitpunkt der Leistung ist nicht angegeben.',
              norm: '§ 14 Abs. 4 Satz 1 Nr. 6 UStG',
              inputTaxEffect: 'Der Vorsteuerabzug ist bis zur Berichtigung ausgeschlossen.',
              blocking: true,
            },
          ],
        },
      ]
    : [];
  const findings = groups.flatMap((g) => g.findings);
  return {
    receiptId,
    receiptNumber: receipt?.receiptNumber ?? '',
    checked,
    groups,
    total: findings.length,
    blocking: findings.filter((f) => f.blocking).length,
  };
}

/** Die Zeilen des Journals als flache Menge — die Grundlage des Filters. */
function journalRows() {
  const rows: any[] = [];
  for (const entry of ENTRIES as any[]) {
    for (const line of entry.lines) {
      rows.push({
        entryId: entry.id,
        entryNumber: entry.entryNumber,
        bookingDate: entry.bookingDate,
        documentDate: entry.documentDate,
        documentNumber: entry.documentNumber,
        description: entry.description,
        kind: entry.kind,
        actor: entry.actor ?? 'M. Schulte',
        hasReceipt: Boolean(entry.receiptId),
        position: line.position,
        side: line.side,
        account: line.account,
        accountName: line.accountName,
        amount: line.amount,
        taxKey: line.taxKey,
        taxBase: line.taxBase,
      });
    }
  }
  return rows;
}

/**
 * Die gefilterte Menge mit ihrer Summenzeile (PRF-01 K3).
 *
 * Gefiltert wird über dieselben Felder wie im Backend; die Vorschau nimmt
 * davon, was ohne Datenbank geht — Zeitraum, Konto, Betragsband, Steuer-
 * schlüssel, Bearbeiter, Beleg ja/nein und den Suchtext.
 */
function filteredJournal(filter: any) {
  const f = filter ?? {};
  const rows = journalRows().filter((row) => {
    if (f.from && row.bookingDate < f.from) return false;
    if (f.to && row.bookingDate > f.to) return false;
    if (f.account && row.account !== f.account) return false;
    if (f.amountFrom !== undefined && row.amount < f.amountFrom) return false;
    if (f.amountTo !== undefined && row.amount > f.amountTo) return false;
    if (f.taxKey && (row.taxKey ?? '') !== f.taxKey) return false;
    if (f.actor && !(row.actor ?? '').includes(f.actor)) return false;
    if (f.hasReceipt !== undefined && row.hasReceipt !== f.hasReceipt) return false;
    if (f.text) {
      const needle = String(f.text).toLowerCase();
      const haystack = `${row.entryNumber} ${row.description} ${row.documentNumber ?? ''}`;
      if (!haystack.toLowerCase().includes(needle)) return false;
    }
    return true;
  });
  const totalDebit = rows.filter((r) => r.side === 'S').reduce((sum, r) => sum + r.amount, 0);
  const totalCredit = rows.filter((r) => r.side === 'H').reduce((sum, r) => sum + r.amount, 0);
  return {
    rows,
    rowCount: rows.length,
    entryCount: new Set(rows.map((r) => r.entryId)).size,
    totalDebit,
    totalCredit,
    balance: totalDebit - totalCredit,
  };
}

/** Dieselbe Menge als CSV — dieselben Spalten wie im Z3-Export. */
function filteredJournalCSV() {
  const head = 'Buchung;Datum;Buchungstext;Konto;Steuerschlüssel;Bearbeiter;Soll;Haben';
  const body = journalRows().map((row) =>
    [
      row.entryNumber,
      row.bookingDate,
      row.description,
      row.account,
      row.taxKey ?? '',
      row.actor ?? '',
      row.side === 'S' ? euro(row.amount) : '',
      row.side === 'H' ? euro(row.amount) : '',
    ].join(';'),
  );
  return [head, ...body].join('\n');
}
