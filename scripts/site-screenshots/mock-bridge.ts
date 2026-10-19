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

import * as closings from './closings-mock';
import { JOURNAL, RECEIPT_DOCS, SPECIAL_PREPAYMENT, VAT_DUE, entryByRef, entryNumber } from './journal-mock';

// Die Oberfläche prüft die Laufzeit über `window._wails`, bevor sie einen
// Bridge-Aufruf zulässt (siehe services/api.ts). Für den Screenshot-Lauf wird
// die Prüfung erfüllt, der Aufruf selbst läuft hier weiter. Ergänzt statt
// ersetzt: die Wails-Laufzeit hängt dort `dispatchWailsEvent` an, über das die
// Aufnahmen Ereignisse des Backends nachspielen.
if (typeof window !== 'undefined') {
  const w = window as any;
  w._wails = Object.assign(w._wails ?? {}, { screenshotMock: true });
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
  // Abgeleitet aus Steuernummer und Bundesland, wie GetCompanySettings sie liefert.
  taxNumberElster: '2227012345678',
  federalState: 'HH',
  vatId: 'DE312345678',
  taxOffice: 'Finanzamt Hamburg-Nord',
  iban: 'DE02120300000000202051',
  bic: 'BYLADEM1001',
  bankName: 'Deutsche Kreditbank',
  street: 'Hafenstraße',
  houseNumber: '14',
  addressAddition: '',
  postalCode: '20359',
  city: 'Hamburg',
  countryCode: 'DE',
  // Gegründet 2019, seit 2024 in Buchfink: ein Umstieg zum Beginn eines Geschäftsjahres.
  foundedOn: '2019-05-14',
  seat: 'Hamburg',
  registerCourt: 'Amtsgericht Hamburg',
  registerNumber: 'HRB 148223',
  registeredOn: '2019-06-03',
  managingDirectors: 'Anna Bauer',
  supervisoryBoardChair: '',
  sellerIdentifier: '',
  notary: 'Dr. Helene Voss, Hamburg',
  deedNumber: 'UR-Nr. 812/2019',
  shareCapital: c(25000),
  shareholders: [
    { name: 'Anna Bauer', shareCapital: c(15000) },
    { name: 'Jonas Peters', shareCapital: c(10000) },
  ],
  contactName: 'Anna Bauer',
  contactPhone: '040 3290 1180',
  contactEmail: 'rechnung@nordlicht-systeme.example',
  invoiceNumberFormat: '',
  receiptNumberFormat: '',
  currency: EUR,
  skr: 'SKR04',
  vatPeriod: 'month',
  taxationType: 'SOLL',
  permanentExtension: true,
  specialPrepayment: c(SPECIAL_PREPAYMENT),
  receiptCaptureDays: 10,
  commitGraceDays: 0,
  taxAdvisor: { engaged: false, sinceYear: 0, declarations: [] as string[] },
  nonProfit: false,
  inLiquidation: false,
  fiscalUnity: false,
  // Leer: die Anlegerstellung für § 20 InvStG folgt aus der Rechtsform.
  investorOverride: '',
  // Ab diesem Bruttobetrag verlangt der Belegweg den Leistungsnachweis
  // (RECH-08); die Voreinstellung des Backends sind 1.000 Euro.
  invoiceCheckThreshold: c(1000),
};

// -------------------------------------------------------------------------
// Konten (SKR04 2026)

/** Ein Konto des Kontenrahmens; Summen und Zahl der Buchungen kommen aus dem Journal. */
interface MockAccount {
  number: string;
  name: string;
  type: string;
  kontenklasse: number;
  kontenklasseName: string;
  balanceSide: string;
  statementType: string;
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
  9: 'Vortrags-, Kapital- und statistische Konten',
};

/** Die Konten der Beispielfirma, bebucht oder nicht; Namen aus internal/accounting/skr04_2026.json. */
const CHART: MockAccount[] = [
  a('0135', 'EDV-Software', 'asset', 0, 'Aktiva', 'Bilanz'),
  a('0440', 'Maschinen', 'asset', 0, 'Aktiva', 'Bilanz'),
  a('0520', 'Pkw', 'asset', 0, 'Aktiva', 'Bilanz'),
  a('0650', 'Büroeinrichtung', 'asset', 0, 'Aktiva', 'Bilanz'),
  a('0670', 'Geringwertige Wirtschaftsgüter', 'asset', 0, 'Aktiva', 'Bilanz'),
  a('0675', 'Wirtschaftsgüter (Sammelposten)', 'asset', 0, 'Aktiva', 'Bilanz'),
  a('1200', 'Forderungen aus Lieferungen und Leistungen', 'asset', 1, 'Aktiva', 'Bilanz'),
  a('1370', 'Durchlaufende Posten', 'asset', 1, 'Aktiva', 'Bilanz'),
  a('1401', 'Abziehbare Vorsteuer 7 %', 'asset', 1, 'Aktiva', 'Bilanz'),
  a('1406', 'Abziehbare Vorsteuer 19 %', 'asset', 1, 'Aktiva', 'Bilanz'),
  a('1408', 'Abziehbare Vorsteuer nach § 13b UStG', 'asset', 1, 'Aktiva', 'Bilanz'),
  a('1460', 'Geldtransit', 'asset', 1, 'Aktiva', 'Bilanz'),
  a('1800', 'Bank', 'asset', 1, 'Aktiva', 'Bilanz'),
  a('1900', 'Aktive Rechnungsabgrenzung', 'asset', 1, 'Aktiva', 'Bilanz'),
  a('2900', 'Gezeichnetes Kapital', 'equity', 2, 'Passiva', 'Bilanz'),
  a('2970', 'Gewinnvortrag vor Verwendung', 'equity', 2, 'Passiva', 'Bilanz'),
  a('3035', 'Gewerbesteuerrückstellung nach § 4 Abs. 5b EStG', 'liability', 3, 'Passiva', 'Bilanz'),
  a('3040', 'Körperschaftsteuerrückstellung', 'liability', 3, 'Passiva', 'Bilanz'),
  a('3095', 'Rückstellungen für Abschluss- und Prüfungskosten', 'liability', 3, 'Passiva', 'Bilanz'),
  a('3096', 'Rückstellungen zur Erfüllung der Aufbewahrungspflichten', 'liability', 3, 'Passiva', 'Bilanz'),
  a('3300', 'Verbindlichkeiten aus Lieferungen und Leistungen', 'liability', 3, 'Passiva', 'Bilanz'),
  a('3642', 'Verbindlichkeiten gegenüber GmbH-Gesellschaftern – Restlaufzeit 1 bis 5 Jahre', 'liability', 3, 'Passiva', 'Bilanz'),
  a('3801', 'Umsatzsteuer 7 %', 'liability', 3, 'Passiva', 'Bilanz'),
  a('3806', 'Umsatzsteuer 19 %', 'liability', 3, 'Passiva', 'Bilanz'),
  a('3820', 'Umsatzsteuer-Vorauszahlungen', 'liability', 3, 'Passiva', 'Bilanz'),
  a('3830', 'Umsatzsteuer-Vorauszahlungen 1/11', 'liability', 3, 'Passiva', 'Bilanz'),
  a('3835', 'Umsatzsteuer nach § 13b UStG', 'liability', 3, 'Passiva', 'Bilanz'),
  a('3841', 'Umsatzsteuerverbindlichkeiten Vorjahr', 'liability', 3, 'Passiva', 'Bilanz'),
  a('3900', 'Passive Rechnungsabgrenzung', 'liability', 3, 'Passiva', 'Bilanz'),
  a('4400', 'Erlöse 19 % USt', 'revenue', 4, 'GuV', 'GuV'),
  a('4736', 'Gewährte Skonti 19 % USt', 'revenue', 4, 'GuV', 'GuV'),
  a('4830', 'Sonstige betriebliche Erträge', 'revenue', 4, 'GuV', 'GuV'),
  a('5400', 'Wareneingang', 'expense', 5, 'GuV', 'GuV'),
  a('5736', 'Erhaltene Skonti 19 % Vorsteuer', 'expense', 5, 'GuV', 'GuV'),
  a('5906', 'Fremdleistungen 19 % Vorsteuer', 'expense', 5, 'GuV', 'GuV'),
  a('6020', 'Gehälter', 'expense', 6, 'GuV', 'GuV'),
  a('6260', 'Sofortabschreibungen geringwertiger Wirtschaftsgüter', 'expense', 6, 'GuV', 'GuV'),
  a('6300', 'Sonstige betriebliche Aufwendungen', 'expense', 6, 'GuV', 'GuV'),
  a('6310', 'Miete (unbewegliche Wirtschaftsgüter)', 'expense', 6, 'GuV', 'GuV'),
  a('6400', 'Versicherungen', 'expense', 6, 'GuV', 'GuV'),
  a('6420', 'Beiträge', 'expense', 6, 'GuV', 'GuV'),
  a('6495', 'Wartungskosten für Hard- und Software', 'expense', 6, 'GuV', 'GuV'),
  a('6600', 'Werbekosten', 'expense', 6, 'GuV', 'GuV'),
  a('6640', 'Bewirtungskosten', 'expense', 6, 'GuV', 'GuV'),
  a('6644', 'Nicht abzugsfähige Bewirtungskosten', 'expense', 6, 'GuV', 'GuV'),
  a('6650', 'Reisekosten Arbeitnehmer', 'expense', 6, 'GuV', 'GuV'),
  a('6805', 'Telefon', 'expense', 6, 'GuV', 'GuV'),
  a('6815', 'Bürobedarf', 'expense', 6, 'GuV', 'GuV'),
  a('6821', 'Fortbildungskosten', 'expense', 6, 'GuV', 'GuV'),
  a('6825', 'Rechts- und Beratungskosten', 'expense', 6, 'GuV', 'GuV'),
  a('6827', 'Abschluss- und Prüfungskosten', 'expense', 6, 'GuV', 'GuV'),
  a('6855', 'Nebenkosten des Geldverkehrs', 'expense', 6, 'GuV', 'GuV'),
  a('7300', 'Zinsen und ähnliche Aufwendungen', 'expense', 7, 'GuV', 'GuV'),
  a('7600', 'Körperschaftsteuer', 'expense', 7, 'GuV', 'GuV'),
  a('7608', 'Solidaritätszuschlag', 'expense', 7, 'GuV', 'GuV'),
  a('7610', 'Gewerbesteuer', 'expense', 7, 'GuV', 'GuV'),
  a('9000', 'Saldenvorträge, Sachkonten', 'statistical', 9, '', ''),
];

function a(
  number: string,
  name: string,
  type: string,
  kontenklasse: number,
  balanceSide: string,
  statementType: string,
): MockAccount {
  return { number, name, type, kontenklasse, kontenklasseName: KLASSEN[kontenklasse] ?? '', balanceSide, statementType };
}

/** Summen je Konto aus dem Journal. */
const TURNOVER = (() => {
  const out = new Map<string, { debit: number; credit: number; count: number }>();
  for (const entry of JOURNAL) {
    for (const [side, account, euro] of entry.lines) {
      const t = out.get(account) ?? { debit: 0, credit: 0, count: 0 };
      if (side === 'S') t.debit += c(euro);
      else t.credit += c(euro);
      t.count++;
      out.set(account, t);
    }
  }
  return out;
})();

/** Aktiva und Aufwand tragen ihren Saldo im Soll, Passiva und Ertrag im Haben. */
const debitNatural = (type: string) => type === 'asset' || type === 'expense' || type === 'statistical';

const ACCOUNTS = CHART.map((m, index) => {
  const t = TURNOVER.get(m.number) ?? { debit: 0, credit: 0, count: 0 };
  return {
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
    debitSum: t.debit,
    creditSum: t.credit,
    balance: debitNatural(m.type) ? t.debit - t.credit : t.credit - t.debit,
    bookingsCount: t.count,
  };
}).sort((x, y) => x.number.localeCompare(y.number));

const accountName = (number: string) => ACCOUNTS.find((x) => x.number === number)?.name ?? '';

// -------------------------------------------------------------------------
// Journal

/** Der Saldenvortrag aus 2025, gebucht vor der ersten Festschreibung des Jahres. */
const CARRIED_FORWARD_AT = `${YEAR}-01-08T09:30:00Z`;

/** Erfasst am Morgen des Buchungstags; der Saldenvortrag am 08.01. */
const ENTRIES = (() => {
  let previous = fakeHash('genesis');
  return JOURNAL.map((e) => {
    const entryHash = fakeHash(e.number + previous);
    const entry = {
      id: e.id,
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
      actor: 'anna@nordlicht-mbp',
      channel: e.channel ?? 'ui',
      bankTxId: e.bankTxId,
      currency: EUR,
      exchangeRateMicros: 1_000_000,
      postingRuleVersion: '2026.1',
      lines: e.lines.map((line, i) => ({
        id: e.id * 100 + i,
        entryId: e.id,
        position: i + 1,
        side: line[0],
        amount: c(line[2]),
        account: line[1],
        accountName: accountName(line[1]),
        taxKey: line[1] === '1406' || line[1] === '3806' ? '9' : undefined,
      })),
      previousHash: previous,
      entryHash,
      createdAt: e.source === 'opening' ? CARRIED_FORWARD_AT : `${e.date}T07:24:00Z`,
    };
    previous = entryHash;
    return entry;
  });
})();

const CHAIN_HEAD = ENTRIES[ENTRIES.length - 1].entryHash;

// -------------------------------------------------------------------------
// Kontakte, offene Posten, Bank

const CONTACTS = [
  contact(1, 'customer', '10001', 'Nordwind Handels GmbH', 'Am Sandtorkai 3, 20457 Hamburg', 'DE289512347', 14),
  contact(2, 'customer', '10002', 'Elbtal Logistik KG', 'Billstraße 88, 20539 Hamburg', 'DE274119803', 30),
  contact(3, 'customer', '10003', 'Werft & Co. KG', 'Steinwerder 12, 20457 Hamburg', 'DE301447221', 14),
  contact(4, 'customer', '10004', 'Marschland Energie AG', 'Deichstraße 5, 25813 Husum', 'DE256330914', 30),
  contact(11, 'vendor', '70011', 'Hanse Cloud Services GmbH', 'Kehrwieder 9, 20457 Hamburg', 'DE317744002', 14),
  contact(12, 'vendor', '70012', 'Kontorhaus Verwaltung GmbH', 'Brandstwiete 1, 20457 Hamburg', 'DE118803442', 0),
  contact(13, 'vendor', '70013', 'Techpartner Nord GmbH', 'Ruhrstraße 42, 22761 Hamburg', 'DE263558107', 14),
  contact(14, 'vendor', '70014', 'Hanseatische Reisestelle GmbH', 'Ballindamm 7, 20095 Hamburg', 'DE204417761', 0),
  contact(15, 'vendor', '70015', 'Nordfunk Telekommunikation GmbH', 'Überseering 30, 22297 Hamburg', 'DE190227315', 0),
  contact(16, 'vendor', '70016', 'Fachportal Media GmbH', 'Kaiser-Wilhelm-Straße 4, 20355 Hamburg', 'DE298001554', 21),
  contact(17, 'vendor', '70017', 'Deutsche Kreditbank AG', 'Taubenstraße 7, 10117 Berlin', '', 0),
  contact(18, 'vendor', '70018', 'Kai Lorenz IT-Beratung', 'Lehmweg 14, 20251 Hamburg', 'DE331905528', 10),
  contact(19, 'vendor', '70019', 'Kanzlei Elbufer Rechtsanwälte', 'Elbchaussee 120, 22763 Hamburg', 'DE287700143', 14),
  contact(20, 'vendor', '70020', 'Anna Bauer', 'Isestraße 61, 20149 Hamburg', '', 0, ''),
];

/**
 * Standard-Buchungsgruppen: Miete, Hosting und Telefon hat Anna Bauer
 * bestätigt; für die freie Mitarbeit und das Fachportal hat Claude eine
 * Gruppe vorgeschlagen, die sie noch bestätigen muss. Die Gruppen passen zu
 * den Konten, auf die das Journal die Rechnungen bucht.
 */
const POSTING_GROUP_STATE: Record<number, object> = {
  11: { defaultPostingGroup: 'fremdleistungen', defaultPostingGroupConfirmedAt: `${YEAR}-03-02T09:14:00Z`, defaultPostingGroupConfirmedBy: 'Anna Bauer' },
  12: { defaultPostingGroup: 'miete', defaultPostingGroupConfirmedAt: `${YEAR}-02-02T08:30:00Z`, defaultPostingGroupConfirmedBy: 'Anna Bauer' },
  15: { defaultPostingGroup: 'telefon', defaultPostingGroupConfirmedAt: `${YEAR}-02-16T10:05:00Z`, defaultPostingGroupConfirmedBy: 'Anna Bauer' },
  16: {
    proposedPostingGroup: 'werbung',
    proposedPostingGroupNote: 'Fünf Rechnungen 2026, alle auf Werbekosten gebucht.',
    proposedPostingGroupBy: 'agent:claude-code',
  },
  18: {
    proposedPostingGroup: 'fremdleistungen',
    proposedPostingGroupNote: 'Sieben Rechnungen 2026 für Entwicklung, alle auf Fremdleistungen gebucht.',
    proposedPostingGroupBy: 'agent:claude-code',
  },
};
for (const entry of CONTACTS) Object.assign(entry, POSTING_GROUP_STATE[entry.id] ?? {});

/** Zerlegt „Straße 1, 12345 Ort" in die Felder des Kontakts. */
function postalParts(address: string) {
  const [street, postalLine] = address.split(', ');
  const [postalCode, ...city] = postalLine.split(' ');
  return { street, postalCode, city: city.join(' ') };
}

function contact(
  id: number,
  type: string,
  ledgerAccount: string,
  name: string,
  address: string,
  vatId: string,
  terms: number,
  company = name,
) {
  return {
    id,
    type,
    ledgerAccount,
    name,
    company,
    email: `buchhaltung@${name.split(' ')[0].toLowerCase().replace(/[^a-z]/g, '')}.example`,
    ...postalParts(address),
    taxId: '',
    vatId,
    countryCode: 'DE',
    iban: '',
    bic: '',
    paymentTermsDays: terms,
    isPrivate: false,
    isSmallBusiness: false,
    // Was offen ist, rechnen die offenen Posten aus dem Journal.
    get openAmount() {
      return OPEN_ITEMS.filter((item) => item.contactId === id).reduce((sum, item) => sum + item.openAmount, 0);
    },
    createdAt: '2024-02-11T08:00:00Z',
  };
}

/**
 * Die offenen Posten aus dem Journal: je Rechnung oder Eingangsbeleg der
 * Betrag abzüglich der Zahlungen, die ihre Nummer nennen. Stornierte
 * Buchungen zählen nicht mit.
 */
const OPEN_ITEMS = (() => {
  const items: ReturnType<typeof openItem>[] = [];
  const reversed = new Set(JOURNAL.map((e) => e.reversalOfId).filter(Boolean));
  const byDocument = new Map<string, (typeof JOURNAL)[number][]>();
  for (const e of JOURNAL) {
    if (e.kind === 'reversal' || reversed.has(e.id) || !e.documentNumber || e.source === 'opening') continue;
    if (!e.lines.some(([s, acc]) => (acc === '1200' && s === 'S') || (acc === '3300' && s === 'H'))) continue;
    byDocument.set(e.documentNumber, [...(byDocument.get(e.documentNumber) ?? []), e]);
  }
  for (const [number, entries] of byDocument) {
    const last = entries[entries.length - 1];
    const customer = last.lines.some(([s, acc]) => acc === '1200' && s === 'S');
    const account = customer ? '1200' : '3300';
    const gross = entries
      .flatMap((e) => e.lines)
      .filter(([s, acc]) => acc === account && s === (customer ? 'S' : 'H'))
      .reduce((sum, [, , euro]) => sum + euro, 0);
    const settled = settledBy(number, account, customer ? 'H' : 'S');
    if (Math.abs(gross - settled) < 0.005) continue;
    const contact = CONTACTS.find((x) => x.id === last.contactId);
    items.push(
      openItem(
        last.id, last.number, last.contactId ?? 0, contact?.name ?? '', customer ? 'customer' : 'vendor',
        contact?.ledgerAccount ?? '', number, last.documentDate, addDays(last.documentDate, contact?.paymentTermsDays ?? 14),
        gross, settled,
      ),
    );
  }
  return items;
})();

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

/** Was der letzte Kontoauszug brachte und noch nicht gebucht ist. */
const UNBOOKED_TX = [
  tx(1, '2026-08-25', 8211, 'Nordwind Handels GmbH', 'DE44100500001234567890', 'RE-2026-0119 Wartungspauschale Q3', 'unmatched'),
  tx(2, '2026-08-25', -2201.5, 'Fachportal Media GmbH', 'DE18500105172345678901', 'Rechnung 2026-8814 Kampagne August', 'unmatched'),
  tx(3, '2026-08-24', 12495, 'Elbtal Logistik KG', 'DE72200400003456789012', 'RE-2026-0117 Teilzahlung', 'unmatched'),
  tx(4, '2026-08-24', -2832.2, 'Techpartner Nord GmbH', 'DE93370400444567890123', 'RG TP-26-4471 Monitore und Dockingstationen', 'unmatched'),
  tx(5, '2026-08-21', -428.4, 'Stadtwerke Nord AG', 'DE29200505501122334455', 'Abschlag Strom 08/2026, Kundennummer 4471-92', 'unmatched'),
  tx(6, '2026-08-20', 3570, 'Werft & Co. KG', 'DE61200800005678901234', 'RE-2026-0116 Restzahlung', 'unmatched'),
];

/** Die Bankseite jeder Buchung: ein zugeordneter Umsatz je Bewegung auf 1800. */
const BOOKED_TX = JOURNAL.filter((e) => e.bankTxId).map((e) => {
  const amount = e.lines
    .filter(([, acc]) => acc === '1800')
    .reduce((sum, [side, , euro]) => sum + (side === 'S' ? euro : -euro), 0);
  const settled = e.ref === 53 ? entryByRef(42) : undefined;
  const bank =
    e.bank ??
    (e.ref === 52
      ? { counterparty: 'Nordfunk Telekommunikation GmbH', iban: 'DE05300606016789012345', remittance: 'Telefon und Internet August 2026' }
      : { counterparty: 'Hanse Cloud Services GmbH', iban: 'DE55201207007890123456', remittance: `${settled?.documentNumber ?? ''} Cloud-Hosting August 2026` });
  return tx(e.bankTxId!, e.date, Math.round(amount * 100) / 100, bank.counterparty, bank.iban, bank.remittance, 'matched');
});

const BANK_TX = [...UNBOOKED_TX, ...BOOKED_TX].sort(
  (x, y) => y.bookingDate.localeCompare(x.bookingDate) || x.id - y.id,
);

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

/** Der Dateiname eines Belegs: Belegnummer und Aussteller. */
const receiptFileName = (number: string, issuer: string) =>
  `${number}_${issuer.split(' ').slice(0, 2).join('-').replace(/[^A-Za-zÄÖÜäöüß-]/g, '')}.pdf`;

/**
 * Die Belege aus dem Journal. Der Notebook-Beleg (Kennung 1) ist eine
 * ZUGFeRD-Rechnung und trägt die Prüfangaben der E-Rechnung.
 */
const RECEIPTS = RECEIPT_DOCS.map((doc) => {
  const base = receipt(doc);
  if (doc.id !== 1) return base;
  return {
    ...base,
    files: [
      file(1, 1, 'original', receiptFileName(doc.number, doc.issuer), 'application/pdf', 184_213, false, doc.date),
      file(2, 1, 'structured', 'factur-x.xml', 'application/xml', 12_884, true, doc.date),
    ],
    detectedFormat: 'ZUGFeRD / Factur-X',
    detectedProfile: 'EN 16931 (Comfort)',
    validatedAt: `${doc.date}T07:41:00Z`,
    validationRuleset: 'EN 16931 / CIUS XRechnung',
    validationVersion: '3.0.2',
    validationCoverage: 'partial',
    validationFindings: JSON.stringify([
      {
        rule: 'BR-DE-15',
        severity: 'warning',
        terms: ['BT-13'],
        message: 'Die Bestellreferenz des Käufers fehlt. Für den Vorsteuerabzug ist sie nicht nötig.',
      },
    ]),
    createdAt: `${doc.date}T07:40:00Z`,
    updatedAt: `${doc.date}T07:41:00Z`,
  };
}).sort((x, y) => y.receiptNumber.localeCompare(x.receiptNumber));

function receipt(doc: (typeof RECEIPT_DOCS)[number]) {
  const fileName = receiptFileName(doc.number, doc.issuer);
  return {
    id: doc.id,
    fiscalYear: YEAR,
    receiptNumber: doc.number,
    direction: 'incoming',
    status: doc.status,
    kind: 'invoice',
    documentDate: doc.date,
    issuerName: doc.issuer,
    subject: doc.subject,
    grossAmount: c(doc.gross),
    taxAmount: c(doc.tax),
    currency: EUR,
    retentionClass: 'vouchers',
    retentionUntil: `${YEAR + 8}-12-31`,
    earliestDeletion: `${YEAR + 9}-01-01`,
    files: [file(doc.id * 10, doc.id, 'original', fileName, 'application/pdf', 96_400 + doc.id * 517, false, doc.date)],
    receiptHash: fakeHash(`receipt-${doc.id}`),
    receivedAt: doc.date,
    receivedVia: 'E-Mail',
    journalEntryId: doc.entryIds[0] as number | undefined,
    discardReason: doc.discardReason,
    validationErrors: 0,
    createdAt: `${doc.date}T08:00:00Z`,
    updatedAt: `${doc.date}T08:00:00Z`,
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
  receivedAt: string,
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
    createdAt: `${receivedAt}T07:40:00Z`,
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

// -------------------------------------------------------------------------
// Umsatzsteuer

/** Was ein Zeitraum auf einem Konto bewegt, Haben positiv; ohne die Zahlungen an das Finanzamt. */
function movement(account: string, from: string, to: string): number {
  return JOURNAL.filter((e) => e.source !== 'tax' && (!from || e.date >= from) && (!to || e.date <= to))
    .flatMap((e) => e.lines)
    .filter(([, acc]) => acc === account)
    .reduce((sum, [side, , euro]) => sum + (side === 'H' ? c(euro) : -c(euro)), 0);
}

/** Die Umsatzsteuer eines Zeitraums aus dem Journal. */
function vatSummary(from: string, to: string) {
  const net19 = movement('4400', from, to);
  const outputTax = movement('3806', from, to);
  const inputTax = -movement('1406', from, to);
  return {
    fiscalYear: YEAR,
    periodFrom: from,
    periodTo: to,
    taxableRevenue: net19 > 0 ? [{ rate: 1900, net: net19, tax: outputTax }] : [],
    exemptRevenue: 0,
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

const INTEGRITY = {
  isValid: true,
  totalEntries: ENTRIES.length,
  checkedEntries: ENTRIES.length,
  message: `Alle ${ENTRIES.length} Buchungen sind unverändert. Die Hash-Kette ist lückenlos.`,
  lastVerifiedHash: CHAIN_HEAD,
  // Geprüft beim Start der App am Morgen des 25.08.
  checkedAt: `${YEAR}-08-25T06:58:00Z`,
};

const receiptNumberOf = (id: number) => RECEIPT_DOCS.find((doc) => doc.id === id)?.number ?? '';

/** Die jüngsten Zeilen des Änderungsprotokolls; die Nummern kommen aus dem Journal. */
const AUDIT_LOGS = [
  log(214, '2026-08-25T06:52:00Z', 'bank.import', 'BankStatement', 'camt053-2026-08-25', '4 Umsätze aus CAMT.053 eingelesen'),
  log(213, '2026-08-24T09:31:00Z', 'journal.post', 'JournalEntry', entryNumber(55), 'Kontoführung August 2026, 62,00 €'),
  log(212, '2026-08-21T14:02:00Z', 'journal.post', 'JournalEntry', entryNumber(54), 'Werbekampagne Fachportal, 2.201,50 €'),
  log(211, '2026-08-20T11:18:00Z', 'payment.settle', 'JournalEntry', entryNumber(53), `Offener Posten ${receiptNumberOf(4)} ausgeglichen`),
  log(210, '2026-08-18T08:47:00Z', 'receipt.seal', 'Receipt', receiptNumberOf(3), 'Beleg gebucht und versiegelt'),
  log(209, '2026-08-14T16:05:00Z', 'journal.post', 'JournalEntry', entryNumber(50), 'Neubuchung nach Generalumkehr'),
  log(208, '2026-08-14T16:04:00Z', 'journal.reverse', 'JournalEntry', entryNumber(49), `Generalumkehr zu ${entryNumber(48)}`),
  log(207, '2026-08-13T10:22:00Z', 'journal.post', 'JournalEntry', entryNumber(48), 'Wartungsvertrag Datenbank, 1.475,60 €'),
  log(206, '2026-08-12T09:14:00Z', 'contact.update', 'Contact', '70013', 'Zahlungsziel von 21 auf 14 Tage geändert'),
  log(205, '2026-08-10T07:41:00Z', 'receipt.file', 'Receipt', receiptNumberOf(1), 'ZUGFeRD-Beleg abgelegt, EN 16931 geprüft'),
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

const MONTH_LABELS = [
  'Januar', 'Februar', 'März', 'April', 'Mai', 'Juni',
  'Juli', 'August', 'September', 'Oktober', 'November', 'Dezember',
];

/**
 * Die Festschreibungen: monatlich, jeweils in der ersten Woche des
 * Folgemonats; Juli steht noch aus. Davor die beiden Vorjahre im Ganzen.
 */
const FESTSCHREIBUNGEN = [
  ...[6, 5, 4, 3, 2, 1].map((m) => {
    const cutoff = `${YEAR}-${String(m).padStart(2, '0')}-${new Date(Date.UTC(YEAR, m, 0)).getUTCDate()}`;
    const at = `${closings.commitDate(m)}T06:15:00Z`;
    const upTo = ENTRIES.filter((e) => e.bookingDate <= cutoff);
    return {
      id: m + 2,
      fiscalYear: YEAR,
      periodType: 'month',
      periodLabel: `${MONTH_LABELS[m - 1]} ${YEAR}`,
      cutoffDate: cutoff,
      chainHead: upTo.length ? upTo[upTo.length - 1].entryHash : '',
      entryCount: upTo.length,
      tsaName: 'freeTSA',
      tsaGenTime: at,
      timestampStatus: 'confirmed',
      createdAt: at,
    };
  }),
  {
    id: 2,
    fiscalYear: YEAR - 1,
    periodType: 'year',
    periodLabel: `Geschäftsjahr ${YEAR - 1}`,
    cutoffDate: `${YEAR - 1}-12-31`,
    chainHead: fakeHash('festschreibung-2025'),
    entryCount: 236,
    tsaName: 'freeTSA',
    tsaGenTime: `${YEAR}-03-20T10:04:00Z`,
    timestampStatus: 'confirmed',
    createdAt: `${YEAR}-03-20T10:04:00Z`,
  },
  {
    id: 1,
    fiscalYear: YEAR - 2,
    periodType: 'year',
    periodLabel: `Geschäftsjahr ${YEAR - 2}`,
    cutoffDate: `${YEAR - 2}-12-31`,
    chainHead: fakeHash('festschreibung-2024'),
    entryCount: 148,
    tsaName: 'freeTSA',
    tsaGenTime: `${YEAR - 1}-03-18T09:40:00Z`,
    timestampStatus: 'confirmed',
    createdAt: `${YEAR - 1}-03-18T09:40:00Z`,
  },
];

/**
 * Was Zahlungen an einem Beleg ausgleichen: die Buchungen, deren Text seine
 * Nummer nennt, auf der Gegenseite des Personenkontos. Ein Storno gleicht
 * nichts aus; es nimmt die stornierte Buchung zurück.
 */
function settledBy(number: string, account: string, side: 'S' | 'H'): number {
  return JOURNAL.filter((e) => e.kind !== 'reversal' && e.description.includes(number))
    .flatMap((e) => e.lines)
    .filter(([s, acc]) => acc === account && s === side)
    .reduce((sum, [, , euro]) => sum + euro, 0);
}

function addDays(iso: string, days: number): string {
  const d = new Date(`${iso}T00:00:00Z`);
  d.setUTCDate(d.getUTCDate() + days);
  return d.toISOString().slice(0, 10);
}

/** Die Ausgangsrechnungen aus dem Journal, die jüngste zuerst. */
const INVOICES = JOURNAL.filter((e) => e.source === 'invoice' && e.kind !== 'reversal' && e.documentNumber)
  .map((e, index) => {
    const number = e.documentNumber!;
    const net = e.lines.find(([, acc]) => acc === '4400')![2];
    const gross = e.lines.find(([, acc]) => acc === '1200')![2];
    const cancelled = JOURNAL.some((x) => x.kind === 'reversal' && x.documentNumber === number);
    const paid = cancelled ? 0 : settledBy(number, '1200', 'H');
    const contact = CONTACTS.find((x) => x.id === e.contactId)!;
    const status = cancelled ? 'cancelled' : paid >= gross - 0.005 ? 'paid' : 'issued';
    return invoice(
      index + 1, number, e.documentDate, addDays(e.documentDate, contact.paymentTermsDays), contact.id,
      contact.name, e.description.replace(/, Rechnung RE-\d{4}-\d{4}$/, ''), net, status, paid, e.id,
    );
  })
  .reverse();

function invoice(
  id: number,
  invoiceNumber: string,
  date: string,
  dueDate: string,
  contactId: number,
  contactName: string,
  text: string,
  netEuro: number,
  status: string,
  paidEuro: number,
  journalEntryId: number,
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
        description: text,
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
    kind: 'invoice',
    journalEntryId,
    terms: { dueDays: Math.round((Date.parse(dueDate) - Date.parse(date)) / 86_400_000), discountPermille: 0, discountDays: 0 },
    precedingRefs: [],
    prepaidAmount: 0,
    receiptId: 500 + id,
    paidAmount: c(paidEuro),
    createdAt: `${date}T07:30:00Z`,
  };
}

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

/**
 * Das Anlagevermögen einer kleinen Software-GmbH. Die Buchwerte zum 01.01.
 * sind die Saldenvorträge der Anlagekonten im Journal; die AfA des Jahres
 * bucht der Jahresabschluss und steht bis dahin als offen.
 */
const ASSETS = [
  asset(1, 'AN-2024-0001', 'Büroeinrichtung Kontorhaus', 'tangible', '0650', 'Büroeinrichtung',
    '6220', '2024-06-01', c(9600), c(1169.23), 0, c(738.46), 'linear', 156),
  asset(2, 'AN-2024-0002', 'USV- und Klimaanlage Serverraum', 'tangible', '0440', 'Maschinen',
    '6220', '2024-07-01', c(25000), c(3750), 0, c(2500), 'linear', 120),
  asset(3, 'AN-2025-0003', 'Pkw VW ID.4 · HH-NS 412', 'tangible', '0520', 'Pkw',
    '6222', '2025-03-15', c(42000), c(5833.33), 0, c(7000), 'linear', 72),
  asset(4, 'AN-2025-0004', 'Projektverwaltung, Lizenz', 'intangible', '0135', 'EDV-Software',
    '6200', '2025-07-01', c(12000), c(2000), 0, c(4000), 'linear', 36),
  asset(5, 'AN-2025-0005', 'Sammelposten 2025', 'tangible', '0675',
    'Wirtschaftsgüter (Sammelposten)', '6264', '2025-01-01', c(4200), c(840), 0, c(840), 'pool', 0,
    { poolYear: YEAR - 1 }),
];

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

/** Der Anlagenspiegel je Konto aus dem Verzeichnis; Zugänge sind Anschaffungen des Jahres. */
const SPIEGEL_ROWS = Array.from(new Set(ASSETS.map((x) => x.account)))
  .sort()
  .map((account) => {
    const items = ASSETS.filter((x) => x.account === account);
    const before = items.filter((x) => Number(x.acquisitionDate.slice(0, 4)) < YEAR);
    const added = items.filter((x) => Number(x.acquisitionDate.slice(0, 4)) === YEAR);
    return spiegelRow(
      items[0].class, account, items[0].accountName, items.length,
      before.reduce((sum, x) => sum + x.cost, 0),
      added.reduce((sum, x) => sum + x.cost, 0),
      0,
      before.reduce((sum, x) => sum + x.accumulated - x.yearAmount, 0),
      items.reduce((sum, x) => sum + x.yearAmount, 0),
    );
  });

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
  classTotals: (['intangible', 'tangible'] as const).map((assetClass) =>
    sumSpiegel(
      SPIEGEL_ROWS.filter((r) => r.class === assetClass),
      assetClass === 'intangible' ? 'Immaterielle Vermögensgegenstände' : 'Sachanlagen',
      assetClass,
    ),
  ),
};

/** Der Abschreibungsplan des Pkw: zeitanteilig im ersten Jahr, Restwert im letzten. */
const ASSET_SCHEDULES: Record<number, unknown[]> = {
  3: [
    { fiscalYear: 2025, months: 10, method: 'linear', rateLabel: '16,7 %', openingBookValue: c(42000), amount: c(5833.33), closingBookValue: c(36166.67), booked: c(5833.33), due: 0, specialBooked: 0, specialDue: 0, status: 'gebucht', note: 'Zeitanteilig für 10 von 12 Monaten (§ 7 Abs. 1 Satz 4 EStG).' },
    { fiscalYear: 2026, months: 12, method: 'linear', rateLabel: '16,7 %', openingBookValue: c(36166.67), amount: c(7000), closingBookValue: c(29166.67), booked: 0, due: c(7000), specialBooked: 0, specialDue: 0, status: 'offen' },
    { fiscalYear: 2027, months: 12, method: 'linear', rateLabel: '16,7 %', openingBookValue: c(29166.67), amount: c(7000), closingBookValue: c(22166.67), booked: 0, due: c(7000), specialBooked: 0, specialDue: 0, status: 'geplant' },
    { fiscalYear: 2028, months: 12, method: 'linear', rateLabel: '16,7 %', openingBookValue: c(22166.67), amount: c(7000), closingBookValue: c(15166.67), booked: 0, due: c(7000), specialBooked: 0, specialDue: 0, status: 'geplant' },
    { fiscalYear: 2029, months: 12, method: 'linear', rateLabel: '16,7 %', openingBookValue: c(15166.67), amount: c(7000), closingBookValue: c(8166.67), booked: 0, due: c(7000), specialBooked: 0, specialDue: 0, status: 'geplant' },
    { fiscalYear: 2030, months: 12, method: 'linear', rateLabel: '16,7 %', openingBookValue: c(8166.67), amount: c(7000), closingBookValue: c(1166.67), booked: 0, due: c(7000), specialBooked: 0, specialDue: 0, status: 'geplant' },
    { fiscalYear: 2031, months: 2, method: 'linear', rateLabel: 'Restwert', openingBookValue: c(1166.67), amount: c(1166.67), closingBookValue: 0, booked: 0, due: c(1166.67), specialBooked: 0, specialDue: 0, status: 'geplant' },
  ],
};

const ASSET_MOVEMENTS: Record<number, unknown[]> = {
  3: [
    { id: 1, assetId: 3, kind: 'acquisition', date: '2025-03-15', fiscalYear: 2025, costAmount: c(42000), depreciationAmount: 0, entryNumber: 'B-2025-0052', note: 'Zugang', createdAt: '2025-03-15T09:00:00Z' },
    { id: 2, assetId: 3, kind: 'depreciation', date: '2025-12-31', fiscalYear: 2025, costAmount: 0, depreciationAmount: c(5833.33), entryNumber: 'B-2025-0229', note: 'AfA 2025', createdAt: '2026-02-20T09:00:00Z' },
  ],
};

const ASSET_NOTES: Record<string, string[]> = {
  tangible: [
    'Lineare Abschreibung über die betriebsgewöhnliche Nutzungsdauer (§ 7 Abs. 1 EStG), zeitanteilig ab dem Anschaffungsmonat (§ 7 Abs. 1 Satz 4 EStG).',
  ],
  intangible: [
    'Immaterielle Vermögensgegenstände des Anlagevermögens dürfen nur angesetzt werden, wenn sie entgeltlich erworben wurden (§ 248 Abs. 2 HGB).',
  ],
};

function assetDetail(id: number) {
  const found = ASSETS.find((a) => a.id === id)!;
  return {
    asset: found,
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

/** Beträge wie im Backend: zwei Nachkommastellen, deutsche Schreibweise. */
const euro = (cents: number) =>
  (cents / 100).toLocaleString('de-DE', { minimumFractionDigits: 2, maximumFractionDigits: 2 });

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
    // Ein ausgeglichenes Konto steht nicht unter der Position.
    accounts: accounts.filter((account) => signedBalance(account) !== 0).map((account) => ({
      number: account.number,
      name: account.name,
      positionId: key,
      position: account.name,
      amount: signedBalance(account),
      priorAmount: 0,
    })),
  };
}

/** Die Konten eines Nummernbereichs der Klasse 4 bis 7, wie die Gliederung sie zuordnet. */
const range = (from: string, to: string) =>
  ACCOUNTS.filter((a) => a.number >= from && a.number <= to && a.bookingsCount > 0 && a.kontenklasse >= 4);
const sumOf = (accounts: (typeof ACCOUNTS)[number][]) => accounts.reduce((sum, a) => sum + signedBalance(a), 0);

const SALES_ACCOUNTS = range('4000', '4799');
const OTHER_INCOME = range('4800', '4999');
const MATERIAL = range('5000', '5999');
const PERSONNEL = range('6000', '6199');
const DEPRECIATION = range('6200', '6299');
const OTHER_EXPENSES = range('6300', '6999');
const INCOME_TAXES = range('7600', '7699');

const REVENUE = [...SALES_ACCOUNTS, ...OTHER_INCOME];
const NET_INCOME =
  sumOf(REVENUE) - sumOf([...MATERIAL, ...PERSONNEL, ...DEPRECIATION, ...OTHER_EXPENSES, ...INCOME_TAXES]);

/** Der Saldo eines Kontos mit dem Vorzeichen seiner natürlichen Seite. */
const balanceOf = (number: string) => {
  const account = ACCOUNTS.find((x) => x.number === number);
  return account ? signedBalance(account) : 0;
};

const FIXED_ASSETS = classAccounts(0);
const CURRENT_ASSETS = classAccounts(1, ['1900']);
const PREPAID = classAccounts(1).filter((a) => a.number === '1900');
const EQUITY = classAccounts(2);
const PROVISIONS = classAccounts(3).filter((a) => a.number < '3100');
const LIABILITIES = classAccounts(3).filter((a) => a.number >= '3100');

const ASSET_LINES = [
  line('aktiva.A', 'A.', 'Anlagevermögen', 1, 'aktiva', FIXED_ASSETS),
  line('aktiva.B', 'B.', 'Umlaufvermögen', 1, 'aktiva', CURRENT_ASSETS),
  line('aktiva.C', 'C.', 'Rechnungsabgrenzungsposten', 1, 'aktiva', PREPAID),
].filter((l) => l.accounts.length > 0);

const LIABILITY_LINES = [
  line('passiva.A', 'A.', 'Eigenkapital', 1, 'passiva', EQUITY, {
    amount: sumOf(EQUITY) + NET_INCOME,
  }),
  line('passiva.B', 'B.', 'Rückstellungen', 1, 'passiva', PROVISIONS),
  line('passiva.C', 'C.', 'Verbindlichkeiten', 1, 'passiva', LIABILITIES),
];

/** Erträge positiv, Aufwendungen negativ, wie die Staffel im Backend. */
const expense = (key: string, ordinal: string, label: string, accounts: (typeof ACCOUNTS)[number][]) =>
  line(key, ordinal, label, 1, 'guv', accounts, { amount: -sumOf(accounts) });

const INCOME_LINES = [
  line('guv.1', '1.', 'Umsatzerlöse', 1, 'guv', SALES_ACCOUNTS),
  line('guv.4', '4.', 'sonstige betriebliche Erträge', 1, 'guv', OTHER_INCOME),
  expense('guv.5', '5.', 'Materialaufwand', MATERIAL),
  expense('guv.6', '6.', 'Personalaufwand', PERSONNEL),
  expense('guv.7', '7.', 'Abschreibungen', DEPRECIATION),
  expense('guv.8', '8.', 'sonstige betriebliche Aufwendungen', OTHER_EXPENSES),
  expense('guv.14', '14.', 'Steuern vom Einkommen und vom Ertrag', INCOME_TAXES),
  line('guv.15', '15.', 'Ergebnis nach Steuern', 1, 'guv', [], { amount: NET_INCOME, isSubtotal: true }),
  line('guv.17', '17.', 'Jahresüberschuss/Jahresfehlbetrag', 1, 'guv', [], {
    amount: NET_INCOME,
    isSubtotal: true,
  }),
].filter((l) => l.isSubtotal || l.accounts.length > 0);

/**
 * Die Geschäftsjahre mit ihrem Abschlussstand: 2024 ist offengelegt, 2025
 * festgestellt und noch nicht offengelegt, das laufende ist offen.
 */
const FISCAL_YEARS = [
  {
    year: YEAR - 2,
    startDate: `${YEAR - 2}-01-01`,
    endDate: `${YEAR - 2}-12-31`,
    isShort: false,
    status: 'disclosed',
    preparedOn: `${YEAR - 1}-03-18`,
    adoptedOn: `${YEAR - 1}-05-20`,
    disclosedOn: `${YEAR - 1}-11-12`,
  },
  {
    year: YEAR - 1,
    startDate: `${YEAR - 1}-01-01`,
    endDate: `${YEAR - 1}-12-31`,
    isShort: false,
    status: 'adopted',
    preparedOn: `${YEAR}-03-20`,
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
  revenue: sumOf(SALES_ACCOUNTS),
  employees: 6,
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
    seat: SETTINGS.seat,
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
        key: 'lieferungen',
        label: 'Verbindlichkeiten aus Lieferungen und Leistungen',
        total: balanceOf('3300'),
        upToOneYear: balanceOf('3300'),
        overOneYear: 0,
        overFiveYears: 0,
        items: OPEN_ITEMS.filter((item) => item.contactType === 'vendor').length,
        undated: 0,
      },
      {
        key: 'gesellschafter',
        label: 'Verbindlichkeiten gegenüber Gesellschaftern',
        total: balanceOf('3642'),
        upToOneYear: 0,
        overOneYear: balanceOf('3642'),
        overFiveYears: 0,
        items: 1,
        undated: 0,
      },
      {
        key: 'sonstige',
        label: 'sonstige Verbindlichkeiten',
        total: ['3806', '3820', '3830', '3841'].reduce((sum, n) => sum + balanceOf(n), 0),
        upToOneYear: ['3806', '3820', '3830', '3841'].reduce((sum, n) => sum + balanceOf(n), 0),
        overOneYear: 0,
        overFiveYears: 0,
        items: 0,
        undated: 0,
      },
    ],
  },
  notes: {
    reference: '§§ 284 bis 288 HGB',
    texts: [],
    provisionMirror: closings.provisionMirror(YEAR),
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
// Aufgabenliste, Monatsabschluss, Bankvorschlag

const TODAY = `${YEAR}-08-25`;

/**
 * Die Pflichten aus dem Pflichtenkalender, wie `GetNextObligations` sie
 * liefert: was noch etwas verlangt, nach Fälligkeit. Die Beispielfirma meldet
 * monatlich mit Dauerfristverlängerung, sitzt in Hamburg und hat
 * Vorauszahlungsbescheide zur Körperschaft- und Gewerbesteuer.
 */
const OBLIGATIONS = [
  obligation({
    key: 'gewst-vorauszahlung.2026-Q3',
    kind: 'gewst-vorauszahlung',
    title: 'Gewerbesteuer-Vorauszahlung 15.08.2026',
    description:
      'Zahlung an die Gemeinde in der Höhe des Vorauszahlungsbescheids. Erledigt mit der Buchung auf 7610.',
    basis: '§ 19 Abs. 1 und 2 GewStG',
    channel: 'intern',
    responsibility: 'human_confirms',
    evidenceSource: 'either',
    periodKey: '2026-Q3',
    periodLabel: '15.08.2026',
    periodFrom: '2026-07-01',
    periodTo: '2026-09-30',
    statutoryDate: '2026-08-15',
    dueDate: '2026-08-17',
    shiftReason: 'Sonnabend',
    status: 'overdue',
    hints: [
      'Das gesetzliche Fristende am 15.08.2026 ist kein Werktag (Sonnabend); die Frist endet am 17.08.2026 (§ 108 Abs. 3 AO).',
    ],
  }),
  obligation({
    key: 'festschreibung.2026-07',
    kind: 'festschreibung',
    title: 'Monatsfestschreibung Juli 2026',
    description:
      'Bis zum Ende des Folgemonats, mit Dauerfristverlängerung einen Monat später, zuzüglich der Nachfrist aus den Einstellungen. Nach der Festschreibung nimmt der Monat keine Buchung mehr auf.',
    basis: '§ 146 Abs. 4 AO, GoBD Rz. 107',
    channel: 'intern',
    responsibility: 'human_confirms',
    evidenceSource: 'data',
    periodKey: '2026-07',
    periodLabel: 'Juli 2026',
    periodFrom: '2026-07-01',
    periodTo: '2026-07-31',
    statutoryDate: '2026-09-30',
    dueDate: '2026-09-30',
    status: 'open',
  }),
  obligation({
    key: 'kst-vorauszahlung.2026-Q3',
    kind: 'kst-vorauszahlung',
    title: 'Körperschaftsteuer-Vorauszahlung 10.09.2026',
    description:
      'Zahlung an das Finanzamt in der Höhe des Vorauszahlungsbescheids, mit Solidaritätszuschlag. Erledigt mit der Buchung auf 7600 oder 7608.',
    basis: '§ 37 Abs. 1 EStG i. V. m. § 31 Abs. 1 und 2 KStG',
    channel: 'intern',
    responsibility: 'human_confirms',
    evidenceSource: 'either',
    periodKey: '2026-Q3',
    periodLabel: '10.09.2026',
    periodFrom: '2026-07-01',
    periodTo: '2026-09-30',
    statutoryDate: '2026-09-10',
    dueDate: '2026-09-10',
    status: 'open',
  }),
  obligation({
    key: 'lohnsteuer-anmeldung.2026-08',
    kind: 'lohnsteuer-anmeldung',
    title: 'Lohnsteuer-Anmeldung August 2026',
    description:
      'Anmeldung und Abführung bis zum 10. Tag nach Ablauf des Anmeldungszeitraums, meist durch den Lohnanbieter. Das gebuchte Lohnjournal bereitet sie vor; erledigt mit dem Transferticket oder der Bestätigung des Lohnanbieters.',
    basis: '§ 41a Abs. 1 und 2 EStG',
    channel: 'lohnanbieter',
    responsibility: 'human_confirms',
    evidenceSource: 'human',
    periodKey: '2026-08',
    periodLabel: 'August 2026',
    periodFrom: '2026-08-01',
    periodTo: '2026-08-31',
    statutoryDate: '2026-09-10',
    dueDate: '2026-09-10',
    status: 'prepared',
    preparation: {
      at: '2026-08-24T15:12:00Z',
      note: 'Lohnabrechnung August abgelegt (Lohnsteuer 4.812,33 €). Die Anmeldung übermittelt der Lohnanbieter.',
      channel: 'agent:claude-code',
      actor: 'anna@nordlicht-mbp',
    },
    hints: ['Monatlich: die Lohnsteuer des Vorjahres lag über 5.000 €.'],
  }),
  obligation({
    key: 'ustva.2026-07',
    kind: 'ustva',
    title: 'Umsatzsteuer-Voranmeldung Juli 2026',
    description:
      'Übermittlung und Zahlung bis zum 10. Tag nach Ablauf des Voranmeldungszeitraums, mit Dauerfristverlängerung einen Monat später. Erledigt mit dem Transferticket an der Voranmeldung.',
    basis: '§ 18 Abs. 1 und 2 UStG, § 46 UStDV',
    channel: 'elster',
    responsibility: 'agent_prepares',
    evidenceSource: 'data',
    periodKey: '2026-07',
    periodLabel: 'Juli 2026',
    periodFrom: '2026-07-01',
    periodTo: '2026-07-31',
    statutoryDate: '2026-09-10',
    dueDate: '2026-09-10',
    status: 'open',
  }),
  obligation({
    key: 'offenlegung.2025',
    kind: 'offenlegung',
    title: 'Offenlegung im Unternehmensregister 2025',
    description:
      'Übermittlung des festgestellten Abschlusses spätestens ein Jahr nach dem Abschlussstichtag. Kleine Gesellschaften legen Bilanz und Anhang offen, Kleinstgesellschaften nur die Bilanz und können sie hinterlegen. Erledigt mit dem Stand „offengelegt“.',
    basis: '§ 325 Abs. 1 und 1a, § 326 HGB',
    channel: 'unternehmensregister',
    responsibility: 'human_confirms',
    evidenceSource: 'data',
    periodKey: '2025',
    periodLabel: '2025',
    periodFrom: '2025-01-01',
    periodTo: '2025-12-31',
    fiscalYear: 2025,
    statutoryDate: '2026-12-31',
    dueDate: '2026-12-31',
    status: 'open',
  }),
];

/** Ziele wie service.ObligationTarget: Monat, Jahr, Erklärungen oder Stammdaten. */
function obligationTarget(kind: string, key: string) {
  const declarations = [
    'kst-erklaerung', 'gewst-erklaerung', 'ust-erklaerung', 'e-bilanz',
    'kst-vorauszahlung', 'gewst-vorauszahlung',
  ];
  const year = [
    'abschlussbuchungen', 'aufstellung', 'feststellung', 'ergebnisverwendung',
    'offenlegung', 'kapitalertragsteuer',
  ];
  const company = ['transparenzregister', 'gesellschafterliste', 'steuerliche-erfassung', 'gewerbeanmeldung'];
  if (declarations.includes(kind)) return { area: 'closings', tab: 'declarations', id: key };
  if (year.includes(kind)) return { area: 'closings', tab: 'year', id: key };
  if (company.includes(kind)) return { area: 'company', tab: 'profile', id: key };
  return { area: 'closings', tab: 'month', id: key };
}

function obligation(o: {
  key: string;
  kind: string;
  title: string;
  description: string;
  basis: string;
  channel: string;
  responsibility: string;
  evidenceSource: string;
  periodKey: string;
  periodLabel: string;
  periodFrom: string;
  periodTo: string;
  fiscalYear?: number;
  statutoryDate: string;
  dueDate: string;
  shiftReason?: string;
  status: string;
  preparation?: { at: string; note: string; channel: string; actor: string };
  hints?: string[];
}) {
  return {
    ...o,
    fiscalYear: o.fiscalYear ?? YEAR,
    skillRef: `buchfink/references/grundlagen/pflichten.md#${o.kind}`,
    hints: o.hints ?? [],
    target: obligationTarget(o.kind, o.key),
  };
}

/**
 * Der Pflichtenkalender eines Jahres aus `closings-mock.ts`. Die ausführlichen
 * Pflichten der Übersicht ersetzen die mit gleichem Schlüssel, damit Übersicht
 * und Abschlüsse denselben Stand zeigen.
 */
function obligationsOf(year: number) {
  const detailed = new Map(OBLIGATIONS.map((o) => [o.key, o]));
  const list = closings
    .obligations(year)
    .map((o) => detailed.get(o.key) ?? { ...o, target: obligationTarget(o.kind, o.key) });
  for (const o of OBLIGATIONS) {
    if (o.fiscalYear === year && !list.some((item) => item.key === o.key)) list.push(o);
  }
  return list.sort((a, b) => a.dueDate.localeCompare(b.dueDate));
}

/** Eine Pflicht als Zeile der Aufgabenliste, unter `pflicht.<schlüssel>`. */
function dutyTask(key: string, group: string) {
  const o = OBLIGATIONS.find((item) => item.key === key)!;
  return {
    key: `pflicht.${o.key}`,
    group,
    title: o.title,
    why: o.description,
    reference: o.basis,
    target: o.target,
    count: 0,
    amount: 0,
    dueDate: o.dueDate,
  };
}

/** Was Apps gebucht haben, die jüngste zuerst, mit der letzten Festschreibung. */
function agentBookings(limit: number) {
  const reversed = new Set(ENTRIES.map((e) => e.reversalOfId).filter(Boolean));
  const all = ENTRIES.filter((e) => e.channel.startsWith('agent:'))
    .map((e) => ({ ...e, reversed: reversed.has(e.id) }))
    .sort((a, b) => b.createdAt.localeCompare(a.createdAt) || b.id - a.id);
  return {
    lastCommit: FESTSCHREIBUNGEN[0],
    total: all.length,
    entries: limit > 0 ? all.slice(0, limit) : all,
  };
}

/** Die Belege, die die Aufgabe „Belege buchen“ zählt. */
const UNBOOKED_RECEIPTS = RECEIPTS.filter((r) => r.status === 'filed');

/** Die Bankumsätze, die die Aufgabe „Bankumsätze zuordnen“ zählt. */
const UNMATCHED_TX = BANK_TX.filter((t) => t.matchStatus === 'unmatched');

/** Die Aufgabenliste der Startseite in ihren drei Gruppen. */
const TASKS = {
  today: TODAY,
  overdue: [dutyTask('gewst-vorauszahlung.2026-Q3', 'overdue')],
  open: [
    dutyTask('festschreibung.2026-07', 'open'),
    {
      key: 'bank_unmatched',
      group: 'open',
      title: 'Bankumsätze zuordnen',
      why: 'Ein Umsatz ohne Zuordnung ist ein Geschäftsvorfall, der noch nicht gebucht ist.',
      reference: '§ 146 Abs. 1 AO',
      target: { area: 'books', tab: 'bank' },
      count: UNMATCHED_TX.length,
      amount: UNMATCHED_TX.reduce((sum, t) => sum + Math.abs(t.amount), 0),
    },
    {
      key: 'receipts_unbooked',
      group: 'open',
      title: 'Belege buchen',
      why: 'Abgelegte Belege sind erfasst, aber noch nicht gebucht.',
      target: { area: 'books', tab: 'receipts' },
      count: UNBOOKED_RECEIPTS.length,
      amount: UNBOOKED_RECEIPTS.reduce((sum, r) => sum + r.grossAmount, 0),
    },
  ],
  upcoming: [
    dutyTask('kst-vorauszahlung.2026-Q3', 'upcoming'),
    dutyTask('lohnsteuer-anmeldung.2026-08', 'upcoming'),
  ],
};

/** „2026-08-10“ als „10.08.2026“, wie das Backend Daten in Texte setzt. */
const germanDay = (iso: string) => `${iso.slice(8, 10)}.${iso.slice(5, 7)}.${iso.slice(0, 4)}`;

/** Bis wohin festgeschrieben ist: der letzte Monat der Festschreibungen dieses Jahres. */
const COMMITTED_TO = FESTSCHREIBUNGEN.filter((f) => f.fiscalYear === YEAR)
  .map((f) => f.cutoffDate)
  .sort()
  .reverse()[0];

/**
 * Die Befunde des Prüfberichts bis zu einem Stichtag, wie das Backend sie
 * rechnet: abgelegte, nicht gebuchte Belege und nicht zugeordnete
 * Bankumsätze. Was im Zeitraum liegt, verhindert die Festschreibung; ein
 * Beleg, der erst danach einging, ist ein Hinweis.
 */
function checkFindings(cutoff: string) {
  const receipts = RECEIPTS.filter((r) => r.status === 'filed').map((r) => ({
    rule: 'receipt_unbooked',
    severity: r.receivedAt <= cutoff ? 'blocking' : 'warning',
    objectType: 'RECEIPT',
    objectId: String(r.id),
    objectName: r.receiptNumber,
    message: `Beleg ${r.receiptNumber} (Eingang ${germanDay(r.receivedAt)}) ist abgelegt, aber nicht gebucht`,
    reference: 'GoBD Rz. 47',
  }));
  const bank = BANK_TX.filter((t) => t.matchStatus === 'unmatched' && t.bookingDate <= cutoff).map((t) => ({
    rule: 'bank_unmatched',
    severity: 'blocking',
    objectType: 'BANK_TX',
    objectId: String(t.id),
    objectName: t.counterpartyName,
    message: `Bankumsatz vom ${germanDay(t.bookingDate)} über ${euro(t.amount)} € (${t.counterpartyName}) ist keiner Buchung zugeordnet`,
    reference: '§ 146 Abs. 1 AO',
  }));
  return [...receipts, ...bank]
    .sort((a, b) => (a.severity === b.severity ? 0 : a.severity === 'blocking' ? -1 : 1))
    .map((f, i) => ({ id: i + 1, checkRunId: 0, ...f }));
}

/** Ein Prüflauf bis zum Stichtag, wie ihn der Festschreibedialog zeigt; gespeichert wird nichts. */
function checkRunFor(cutoff: string, periodType: string) {
  const entries = ENTRIES.filter((e) => e.bookingDate <= cutoff);
  return {
    id: 0,
    fiscalYear: Number(cutoff.slice(0, 4)) || YEAR,
    cutoffDate: cutoff,
    periodType,
    checkedEntries: entries.length,
    checkedReceipts: RECEIPTS.filter((r) => r.receivedAt <= cutoff).length,
    checkedBankTx: BANK_TX.filter((t) => t.bookingDate <= cutoff).length,
    timeliness: {
      measuredEntries: entries.length,
      captureDaysMedian: 1,
      captureDaysMax: 6,
      captureLimitDays: 10,
      lateEntries: 0,
      committedEntries: ENTRIES.filter((e) => e.bookingDate <= COMMITTED_TO).length,
      commitDaysMedian: 22,
      commitDaysMax: 37,
      uncommittedEntries: entries.filter((e) => e.bookingDate > COMMITTED_TO).length,
    },
    findings: checkFindings(cutoff),
    createdAt: `${TODAY}T08:11:00Z`,
  };
}

/** Der Stand eines Monats in seinen drei Schritten. */
function monthCloseState(month: string) {
  const [year, index] = month.split('-').map(Number);
  if (!year || !index) throw new Error(`„${month}“ ist kein Monat (erwartet JJJJ-MM).`);
  if (year !== YEAR) {
    throw new Error(`Für ${year} liegen in den Beispieldaten keine Monate vor.`);
  }
  const last = new Date(Date.UTC(year, index, 0)).getUTCDate();
  const label = `${MONTH_LABELS[index - 1]} ${year}`;
  // Der Mandant meldet monatlich: jeder Monat endet mit seiner Voranmeldung.
  const to = `${month}-${String(last).padStart(2, '0')}`;
  const committed = to <= COMMITTED_TO;
  const ret = vatReturn(month);
  const vat = {
    vatPeriodKey: ret.periodKey,
    vatPeriodLabel: label,
    vatStatus: ret.status,
    vatDueDate: ret.dueDate,
    vatSubmittedAt: ret.submittedAt,
    vatReturnId: ret.id,
  };
  const vatTitle = `Voranmeldung ${label} bestätigen`;
  const vatStep = ret.submittedAt
    ? { number: 3, key: 'vat', title: vatTitle, state: 'done', note: `Am ${germanDay(ret.submittedAt)} übermittelt.` }
    : committed
      ? { number: 3, key: 'vat', title: vatTitle, state: 'open', note: `Werte in Mein ELSTER eintragen, danach Datum und Transferticket erfassen. Fällig am ${germanDay(ret.dueDate)}.` }
      : { number: 3, key: 'vat', title: vatTitle, state: 'blocked', note: 'Das Kennziffernblatt steht bereit. Bestätigen lässt sich die Übermittlung erst nach der Festschreibung.' };

  // Ein festgeschriebener Monat hat seinen Prüfbericht hinter sich.
  const findings = committed ? [] : checkFindings(to);
  const blocking = findings.filter((f) => f.severity === 'blocking').length;
  const checkStep = committed
    ? { number: 1, key: 'check', title: 'Prüfbericht', state: 'done', note: 'Der Monat ist festgeschrieben; der Prüfbericht lief davor.' }
    : blocking > 0
      ? { number: 1, key: 'check', title: 'Prüfbericht', state: 'blocked', note: `${blocking} Befunde verhindern die Festschreibung. Klären oder mit Begründung übergehen.` }
      : findings.length > 0
        ? { number: 1, key: 'check', title: 'Prüfbericht', state: 'open', note: `${findings.length} Hinweise, die die Festschreibung nicht verhindern.` }
        : { number: 1, key: 'check', title: 'Prüfbericht', state: 'done', note: 'Der Prüfbericht ist ohne Befund.' };
  const commitStep = committed
    ? { number: 2, key: 'commit', title: 'Festschreiben', state: 'done', note: `Festgeschrieben bis zum ${germanDay(COMMITTED_TO)}.` }
    : blocking > 0
      ? { number: 2, key: 'commit', title: 'Festschreiben', state: 'blocked', note: 'Erst die blockierenden Befunde klären oder mit Begründung übergehen.' }
      : { number: 2, key: 'commit', title: 'Festschreiben', state: 'open', note: 'Der Monat ist noch nicht festgeschrieben.' };

  return {
    month,
    label,
    from: `${month}-01`,
    to,
    fiscalYear: YEAR,
    steps: [checkStep, commitStep, vatStep],
    findings,
    blocking,
    committed,
    committedTo: COMMITTED_TO,
    vatApplies: true,
    ...vat,
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

/** Die Gründe, mit denen eine Lücke im Nummernkreis begründet wird. */
const NUMBER_GAP_REASONS = [
  { reason: 'aborted', label: 'Abgebrochene Ausstellung' },
  { reason: 'test', label: 'Testlauf' },
  { reason: 'cancelled', label: 'Storniert' },
  { reason: 'unknown', label: 'Unbekannt' },
];

/**
 * Das Kennziffernblatt eines Voranmeldungszeitraums, gerechnet aus dem
 * Journal: Zahllast gleich Umsatzsteuer minus Vorsteuer. Der Mandant meldet
 * monatlich mit Dauerfristverlängerung, der Schlüssel ist „JJJJ-MM“.
 */
function vatReturn(periodKey: string) {
  const month = /^\d{4}-(\d{2})$/.exec(periodKey)?.[1];
  if (!month) throw new Error(`Für ${periodKey} liegt kein Kennziffernblatt vor.`);
  const m = Number(month);
  const from = `${YEAR}-${month}-01`;
  const to = `${YEAR}-${month}-${new Date(Date.UTC(YEAR, m, 0)).getUTCDate()}`;
  const summary = vatSummary(from, to);
  const base = summary.taxableRevenue[0]?.net ?? 0;
  const outputTax = summary.outputTax;
  const dueDate = VAT_DUE[m - 1] ?? `${YEAR + (m >= 11 ? 1 : 0)}-${pad(((m + 1) % 12) + 1)}-10`;
  // Übermittelt wurde vor der Fälligkeit; gezahlt ist am Fälligkeitstag.
  const submittedAt: string | undefined = closings.VAT_SUBMITTED[m - 1];
  return {
    id: m,
    fiscalYear: YEAR,
    periodType: 'month',
    periodKey,
    periodFrom: from,
    periodTo: to,
    isCorrection: false,
    status: submittedAt ? 'submitted' : 'draft',
    submittedAt,
    transferTicket: submittedAt ? `et${YEAR % 100}m${month}x7k2m9p4` : undefined,
    payable: summary.payable,
    dueDate,
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
    createdAt: submittedAt ? `${submittedAt}T07:30:00Z` : `${TODAY}T06:30:00Z`,
  };
}

// -------------------------------------------------------------------------
// Meldezeiträume, Prüfläufe, Abschlussweg, Nebenpflichten und Nachweise
//
// Was eine Seite beim Öffnen ruft, steht hier mit Beispieldaten; was sie erst
// auf Knopfdruck ruft, steht als `unsupported` in der Bridge — geschrieben wird
// in der Vorschau nichts.

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
    totalServices: 0,
    lines: [],
    findings: [],
    lateEntries: [],
    createdAt: `${TODAY}T06:30:00Z`,
  };
}

/**
 * Die Prüfberichte vor den Festschreibungen, je Monat eine Minute vor ihr.
 * Gezählt werden die Buchungen bis zum Stichtag und, wie im Backend, alle
 * Belege und Bankumsätze des Jahres, die es am Tag des Laufs gab.
 */
const CHECK_RUNS = FESTSCHREIBUNGEN.filter((f) => f.fiscalYear === YEAR).map((f) => {
  const entries = ENTRIES.filter((e) => e.bookingDate <= f.cutoffDate);
  const at = new Date(Date.parse(f.createdAt) - 60_000).toISOString().replace('.000Z', 'Z');
  const days = Math.round((Date.parse(f.createdAt) - Date.parse(`${f.cutoffDate}T00:00:00Z`)) / 86_400_000);
  return {
    id: f.id,
    fiscalYear: YEAR,
    cutoffDate: f.cutoffDate,
    periodType: 'month',
    checkedEntries: entries.length,
    checkedReceipts: RECEIPTS.filter((r) => r.receivedAt <= f.createdAt.slice(0, 10)).length,
    checkedBankTx: BANK_TX.filter((t) => t.bookingDate <= f.createdAt.slice(0, 10)).length,
    timeliness: {
      measuredEntries: entries.length,
      captureDaysMedian: 1,
      captureDaysMax: 6,
      captureLimitDays: 10,
      lateEntries: 0,
      committedEntries: entries.length,
      commitDaysMedian: days + 14,
      commitDaysMax: days + 30,
      uncommittedEntries: 0,
    },
    findings: [],
    createdAt: at,
  };
});

/** Zuordnung der bebuchten Konten auf Gliederung und Taxonomie-Element. */
const EBILANZ_MAPPING: Record<string, [string, string]> = {
  '0135': ['Entgeltlich erworbene Konzessionen, Lizenzen, EDV-Software', 'bs.ass.fixAss.intan.concessionBrands.software'],
  '0440': ['Technische Anlagen und Maschinen', 'bs.ass.fixAss.tan.machinery'],
  '0520': ['Andere Anlagen, Betriebs- und Geschäftsausstattung', 'bs.ass.fixAss.tan.otherEquipm'],
  '0650': ['Andere Anlagen, Betriebs- und Geschäftsausstattung', 'bs.ass.fixAss.tan.otherEquipm'],
  '0675': ['Andere Anlagen, Betriebs- und Geschäftsausstattung', 'bs.ass.fixAss.tan.otherEquipm'],
  '1200': ['Forderungen aus Lieferungen und Leistungen', 'bs.ass.currAss.receiv.trade'],
  '1406': ['Sonstige Vermögensgegenstände', 'bs.ass.currAss.receiv.other'],
  '1800': ['Kassenbestand, Guthaben bei Kreditinstituten', 'bs.ass.currAss.cashEquiv'],
  '1900': ['Rechnungsabgrenzungsposten', 'bs.ass.prepaidExp'],
  '2900': ['Gezeichnetes Kapital', 'bs.eqLiab.equity.subscribed'],
  '2970': ['Gewinnvortrag', 'bs.eqLiab.equity.retainedEarnings'],
  '3035': ['Steuerrückstellungen', 'bs.eqLiab.accruals.tax'],
  '3040': ['Steuerrückstellungen', 'bs.eqLiab.accruals.tax'],
  '3095': ['Sonstige Rückstellungen', 'bs.eqLiab.accruals.other'],
  '3096': ['Sonstige Rückstellungen', 'bs.eqLiab.accruals.other'],
  '3300': ['Verbindlichkeiten aus Lieferungen und Leistungen', 'bs.eqLiab.liab.trade'],
  '3820': ['Sonstige Verbindlichkeiten, aus Steuern', 'bs.eqLiab.liab.other.theroffTax'],
  '3830': ['Sonstige Verbindlichkeiten, aus Steuern', 'bs.eqLiab.liab.other.theroffTax'],
  '3642': ['Verbindlichkeiten gegenüber GmbH-Gesellschaftern', 'bs.eqLiab.liab.shareholders.gmbh'],
  '3806': ['Sonstige Verbindlichkeiten, aus Steuern', 'bs.eqLiab.liab.other.theroffTax'],
  '3841': ['Sonstige Verbindlichkeiten, aus Steuern', 'bs.eqLiab.liab.other.theroffTax'],
  '4400': ['Umsatzerlöse', 'is.netIncome.regular.operatingTC.grossTradingProfit.totalOutput.netSales'],
  '4830': ['Sonstige betriebliche Erträge', 'is.netIncome.regular.operatingTC.otherOpRevenue'],
  '5906': ['Aufwendungen für bezogene Leistungen', 'is.netIncome.regular.operatingTC.grossTradingProfit.materialServices.services'],
  '6020': ['Löhne und Gehälter', 'is.netIncome.regular.operatingTC.staff.salaries'],
  '6260': ['Abschreibungen auf Sachanlagen', 'is.netIncome.regular.operatingTC.deprAmort.fixAss.tan'],
  '7600': ['Steuern vom Einkommen und vom Ertrag, Körperschaftsteuer', 'is.netIncome.tax.kst'],
  '7608': ['Steuern vom Einkommen und vom Ertrag, Solidaritätszuschlag', 'is.netIncome.tax.soli'],
  '7610': ['Steuern vom Einkommen und vom Ertrag, Gewerbesteuer', 'is.netIncome.tax.gewst'],
};

/** Die Auffangposition, in der alles landet, was keine eigene Zeile hat. */
const EBILANZ_FALLBACK: [string, string] = [
  'Sonstige betriebliche Aufwendungen',
  'is.netIncome.regular.operatingTC.otherCost',
];

function mappingReport(year: number) {
  // Die Vortragskonten der Klasse 9 gehören weder in Bilanz noch GuV.
  const booked = ACCOUNTS.filter((x) => x.bookingsCount > 0 && x.kontenklasse < 9);
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
    accountingStandard: 'HAOE',
    transfers: [],
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

/**
 * Der Abschlussstand je Geschäftsjahr: 2025 ist aufgestellt, festgeschrieben,
 * festgestellt und vorgetragen, die Offenlegung steht aus; 2026 läuft noch.
 */
/** Ergebnis und Umsatz der Vorjahre; 2024 ist ausgeschüttet, 2025 vorgetragen. */
const PRIOR_YEARS: Record<number, { netIncome: number; revenue: number; adoptionNote: string; carriedAt: string }> = {
  [YEAR - 2]: { netIncome: c(38400), revenue: c(356900), adoptionNote: 'Gesellschafterbeschluss vom 20.05.2025', carriedAt: `${YEAR - 1}-03-18T10:00:00Z` },
  [YEAR - 1]: { netIncome: c(61230), revenue: c(412800), adoptionNote: 'Gesellschafterbeschluss vom 14.05.2026', carriedAt: CARRIED_FORWARD_AT },
};

/**
 * Der Abschlussstand je Geschäftsjahr: 2024 ist offengelegt, 2025 aufgestellt,
 * festgeschrieben, festgestellt und vorgetragen, die Offenlegung steht aus;
 * 2026 läuft noch.
 */
function closingState(year: number) {
  const y = year || YEAR;
  const base = FISCAL_YEARS.find((entry) => entry.year === y) ?? FISCAL_YEARS[FISCAL_YEARS.length - 1];
  const prior = PRIOR_YEARS[y];
  const fiscalYear = {
    ...base,
    ...(prior ? { adoptionNote: prior.adoptionNote } : {}),
    averageEmployees: y === YEAR ? 6 : 5,
    priorYearRevenue: PRIOR_YEARS[y - 1]?.revenue ?? c(298400),
    createdAt: `${y}-01-02T08:00:00Z`,
  };
  return {
    year: y,
    fiscalYear,
    netIncome: prior ? prior.netIncome : NET_INCOME,
    hasYearCommitment: Boolean(prior),
    committedUntil: prior ? `${y}-12-31` : COMMITTED_TO,
    nextYear: y + 1,
    carriedForward: Boolean(prior),
    carryForwardCurrent: Boolean(prior),
    carriedForwardAt: prior?.carriedAt,
    nextStatus: base.status === 'disclosed' ? undefined : prior ? 'disclosed' : 'prepared',
    canAdopt: true,
  };
}

/** Die Abschlussbausteine wie in internal/domain/closing_step.go: Schlüssel, Titel, Text, automatisch, Norm. */
const CLOSING_STEP_DEFS: [string, string, string, boolean, string][] = [
  ['depreciation', 'Abschreibungen',
    'Die planmäßige AfA des Jahres. Sie ist eine Abschlussbuchung und lässt sich später nicht nachholen.',
    true, '§ 253 Abs. 3 HGB, § 7 EStG'],
  ['write_up', 'Wertaufholung prüfen',
    'Für jedes Anlagegut mit außerplanmäßiger Abschreibung: ist der Grund weggefallen? Dann ist zuzuschreiben. Das ist ein Gebot und kein Wahlrecht.',
    true, '§ 253 Abs. 5 Satz 1 HGB'],
  ['currency_valuation', 'Fremdwährungsbewertung',
    'Offene Posten in Fremdwährung werden zum Stichtagskurs bewertet. Bei einer Restlaufzeit über einem Jahr wird nur der Verlust erfasst.',
    true, '§ 256a HGB'],
  ['receivable_allowance', 'Wertberichtigung auf Forderungen',
    'Forderungen mit Ausfallrisiko werden einzeln, die übrigen pauschal auf den niedrigeren Wert berichtigt, netto: die Umsatzsteuer folgt erst der Uneinbringlichkeit.',
    false, '§ 253 Abs. 4 Satz 2 HGB, § 17 Abs. 2 Nr. 1 UStG'],
  ['work_in_progress', 'Unfertige Leistungen',
    'Begonnene, am Stichtag nicht abgerechnete Projekte werden zu Herstellungskosten aktiviert und im Folgejahr aufgelöst; erhaltene Anzahlungen bleiben unter den Verbindlichkeiten oder werden offen abgesetzt.',
    false, '§ 255 Abs. 2 HGB, § 268 Abs. 5 Satz 2 HGB'],
  ['accruals', 'Rechnungsabgrenzung',
    'Ausgaben und Einnahmen, die wirtschaftlich ins nächste Jahr gehören.', false, '§ 250 HGB'],
  ['provisions', 'Rückstellungen',
    'Verpflichtungen, deren Höhe oder Fälligkeit noch offen ist.', false, '§ 249 HGB'],
  ['vat_settlement', 'Umsatzsteuer-Verrechnung',
    'Vorsteuer, Umsatzsteuer und Vorauszahlungen werden zu einem Saldo verrechnet.', false, ''],
  ['input_tax_correction', 'Vorsteuerberichtigung',
    'Hat sich die Verwendung eines Wirtschaftsguts im Berichtigungszeitraum geändert, ist der Vorsteuerabzug anteilig zu berichtigen. Der Betrag geht in die Kennziffer 64.',
    true, '§ 15a UStG'],
  ['tax_provision', 'Steuerrückstellung',
    'Körperschaftsteuer, Solidaritätszuschlag und Gewerbesteuer auf das Ergebnis des Jahres, abzüglich Vorauszahlungen und anrechenbarer Kapitalertragsteuer; eine Überzahlung ist ein Erstattungsanspruch.',
    false, '§ 249 Abs. 1 HGB'],
  ['check_run', 'Prüfbericht',
    'Der Prüflauf über Buchungen, Belege und Fristen vor der Festschreibung.', true, 'GoBD Rz. 100 ff.'],
  ['statement', 'Bilanz und GuV',
    'Die Gliederung von Bilanz und Gewinn- und Verlustrechnung samt Anhang.', true, '§§ 266, 275 HGB'],
  ['adoption', 'Aufstellen und Feststellen',
    'Die Aufstellung durch die Geschäftsführung und der Beschluss der Gesellschafter.', true,
    '§ 264 Abs. 1 HGB, § 46 Nr. 1 GmbHG'],
  ['disclosure', 'E-Bilanz und Offenlegung',
    'Die Übermittlung der E-Bilanz an das Finanzamt und die Offenlegung im Unternehmensregister.', true,
    '§ 5b EStG, § 325 HGB'],
  ['appropriation', 'Vortrag und Ergebnisverwendung',
    'Der Saldenvortrag ins Folgejahr und der Beschluss über das Ergebnis.', false, '§ 29 GmbHG'],
];

/** Zustand und Begründung je Baustein im Beispieljahr. */
const CLOSING_STEP_STATE: Record<string, { state: string; detail?: string; reason?: string }> = {
  depreciation: { state: 'done', detail: 'AfA für 5 Anlagegüter gebucht' },
  write_up: { state: 'done', detail: 'Keine außerplanmäßige Abschreibung offen' },
  currency_valuation: {
    state: 'skipped',
    reason: 'Im Geschäftsjahr steht kein offener Posten in Fremdwährung.',
  },
  accruals: { state: 'done', detail: '2 Abgrenzungsposten gebildet' },
  provisions: { state: 'done', detail: '2 Rückstellungen gebildet' },
  receivable_allowance: { state: 'done', detail: 'Keine Forderung mit Ausfallrisiko' },
  work_in_progress: { state: 'skipped', reason: 'Am Stichtag war kein Projekt begonnen und nicht abgerechnet.' },

  vat_settlement: { state: 'open', detail: 'Zahllast der vier Zeiträume noch nicht verrechnet' },
  input_tax_correction: { state: 'open', detail: '1 Wirtschaftsgut im Berichtigungszeitraum' },
  tax_provision: { state: 'open' },
  check_run: { state: 'open', detail: `Letzter Lauf zum ${germanDay(COMMITTED_TO)}` },
  statement: { state: 'open' },
  adoption: { state: 'open', detail: 'Ohne Jahres-Festschreibung nicht möglich' },
  disclosure: { state: 'open' },
  appropriation: { state: 'open' },
};

/** Was in einem abgeschlossenen Jahr an den Bausteinen steht. */
function closedYearDetail(year: number, key: string): string | undefined {
  const detail: Record<string, string> = {
    vat_settlement: `Zahllast ${year} verrechnet`,
    input_tax_correction: 'Keine Berichtigung nötig',
    tax_provision: 'Körperschaft- und Gewerbesteuer zurückgestellt',
    check_run: `Lauf zum 31.12.${year} ohne Befund`,
    statement: 'Kleine Kapitalgesellschaft, verkürzte Bilanz',
    adoption: year === YEAR - 1 ? 'Festgestellt am 14.05.2026' : 'Festgestellt am 20.05.2025',
    disclosure: 'Offengelegt am 12.11.2025',
    appropriation: year === YEAR - 1 ? 'Gewinnvortrag auf neue Rechnung' : 'Ausgeschüttet',
  };
  return detail[key];
}

/** Im laufenden Jahr steht jeder Baustein noch offen. */
const OPEN_YEAR_DETAIL: Record<string, string> = {
  depreciation: 'AfA für 5 Anlagegüter zum 31.12.2026',
  accruals: '2 Verträge reichen über den Stichtag',
  tax_provision: 'Schätzung auf den heutigen Stand',
  vat_settlement: 'Zahllast von zwölf Monaten und Sondervorauszahlung',
};

function closingSteps(year: number) {
  const y = year || YEAR;
  const closed = y < YEAR;
  const disclosed = FISCAL_YEARS.find((entry) => entry.year === y)?.status === 'disclosed';
  const steps = CLOSING_STEP_DEFS.map(([key, label, hint, automatic, reference], index) => {
    const status = closed
      ? key === 'disclosure' && !disclosed
        ? { state: 'open', detail: `Offenlegung bis 31.12.${y + 1}` }
        : CLOSING_STEP_STATE[key]?.state === 'skipped'
          ? CLOSING_STEP_STATE[key]
          : { state: 'done', detail: closedYearDetail(y, key) ?? CLOSING_STEP_STATE[key]?.detail }
      : { state: 'open', detail: OPEN_YEAR_DETAIL[key] };
    return {
      key,
      order: index + 1,
      label,
      hint,
      reference,
      automatic,
      state: status.state,
      reason: status.reason,
      changedOn: status.state === 'open' ? undefined : `${y + 1}-03-12`,
      detail: status.detail,
    };
  });
  return {
    fiscalYear: year || YEAR,
    cutoff: `${year || YEAR}-12-31`,
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
  const openItemCount: Record<string, number> = {
    '1200': OPEN_ITEMS.filter((item) => item.contactType === 'customer').length,
    '3300': OPEN_ITEMS.filter((item) => item.contactType === 'vendor').length,
  };
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
    netIncome: NET_INCOME,
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
  checkedAt: `${YEAR}-08-25T06:58:00Z`,
  message: 'Alle Protokolleinträge sind unverändert. Die Kette ist lückenlos.',
};

/** Die Sicherungsläufe: automatisch beim Beenden am Abend, dazu eine von Hand. */
const BACKUP_RUNS = [
  backupRun(3, 'automatic', '2026-08-24T18:40:12Z', 'nordlicht-2026-08-24.zip', 214, 48_218_411),
  backupRun(2, 'automatic', '2026-08-21T18:32:05Z', 'nordlicht-2026-08-21-abend.zip', 212, 47_902_188),
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
    message: 'Datenbank, Belege und Dokumente gesichert.',
    programVersion: '0.9.4',
    createdAt: startedAt,
  };
}

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

/** Hinweise zu Speicherort und Steuerfällen. */
const COMPLIANCE_HINTS = {
  cloudWarning: '',
  dataDir: TENANTS[0].dataDir,
  systemChangeDate: '',
  systemChangeNote: '',
  taxCaseHints: [
    'Organschaft und Konsolidierung bildet Buchfink nicht ab.',
    'Land- und Forstwirtschaft nach § 13a EStG ist nicht abgedeckt.',
  ],
};

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

/** Der Gründungsstand. Die Beispielfirma ist seit 2024 eingetragen. */
const FOUNDATION_STATE = {
  hasFoundation: false,
  legalForm: 'GmbH',
  rules: {},
  stage: 'eingetragen',
  postingsBooked: true,
};

// -------------------------------------------------------------------------
// Bankkonten, verbundene Apps, Länder

const BANK_ACCOUNTS = [
  {
    iban: SETTINGS.iban,
    name: 'Geschäftskonto',
    ledgerAccount: '1800',
    currency: EUR,
    bic: SETTINGS.bic,
    bankName: SETTINGS.bankName,
    isInvoiceAccount: true,
  },
];

const MCP_STATUS = {
  running: true,
  port: 7465,
  url: 'http://127.0.0.1:7465/mcp',
  error: '',
  fromEnvironment: false,
};

// -------------------------------------------------------------------------
// Eingang, Lese- und Exportordner

const DATA_DIR = TENANTS[0].dataDir;

/**
 * Drei Dateien im Eingang, die älteste zuerst: ein Foto vom 6. August, das
 * seit 19 Tagen liegt, eine E-Rechnung der Kanzlei und der Kontoauszug von
 * heute früh. Keine davon ist schon abgelegt.
 */
const INBOX_FILES = [
  {
    name: 'Bewirtung Kundentermin Husum.jpg',
    size: 1_284_311,
    modifiedAt: `${YEAR}-08-06T18:42:00+02:00`,
    kind: 'beleg',
  },
  {
    name: 'Kanzlei Elbufer RE-2026-311.pdf',
    size: 186_402,
    modifiedAt: `${YEAR}-08-24T16:05:00+02:00`,
    kind: 'e_rechnung',
  },
  {
    name: 'Kontoauszug DKB 2026-08-24.xml',
    size: 23_877,
    modifiedAt: `${YEAR}-08-25T07:58:00+02:00`,
    kind: 'kontoauszug',
  },
].map((file) => ({ ...file, path: `${DATA_DIR}/Eingang/${file.name}` }));

/** Weitere Leseordner: die Verträge, aus denen Claude Darlehen und Mietverträge liest. */
const READ_FOLDERS = ['/Users/anna/Dokumente/Verträge'];

const INBOX_INFO = {
  path: `${DATA_DIR}/Eingang`,
  default: true,
  count: INBOX_FILES.length,
  oldestDays: 19,
  readFolders: READ_FOLDERS,
  exportFolder: `${DATA_DIR}/Export`,
};

/** Die Regeln für Claude, wie eingerichtet: Grenzen voreingestellt, ein Leseordner. */
const AGENT_RULES = {
  stored: { directMaxGross: 0, directMinBookings: 0, extraReadFolders: READ_FOLDERS, exportFolder: '' },
  directMaxGross: c(1000),
  directMinBookings: 3,
  extraReadFolders: READ_FOLDERS,
  exportFolder: `${DATA_DIR}/Export`,
  defaults: { directMaxGross: c(1000), directMinBookings: 3 },
};

/**
 * Wie `syncdir.Detect`: ein Ordner unter iCloud Drive, Dropbox, OneDrive,
 * Google Drive oder Nextcloud. Die Einrichtung zeigt den Hinweis zur Ablage.
 */
function folderCheck(path: string) {
  const services: [RegExp, string][] = [
    [/Library\/Mobile Documents|CloudStorage\/iCloud|iCloudDrive/i, 'iCloud Drive'],
    [/OneDrive/i, 'OneDrive'],
    [/Google ?Drive|Meine Ablage/i, 'Google Drive'],
    [/Dropbox/i, 'Dropbox'],
    [/Nextcloud|ownCloud/i, 'Nextcloud/ownCloud'],
  ];
  const sync = services.find(([pattern]) => pattern.test(path))?.[1] ?? '';
  return {
    sync,
    warning: sync
      ? `Der Ordner liegt in ${sync}. Die Datenbank verträgt keine Synchronisation, während Buchfink läuft; ` +
        'besser ist ein lokaler Ordner mit einer Sicherung.'
      : '',
  };
}

/** Zwei zugelassene Apps: Claude Code bucht, Claude Desktop liest nur. */
const CONNECTED_APPS = [
  {
    clientId: 'c7f1e2a0-claude-code',
    name: 'Claude Code',
    channel: 'agent:claude-code',
    approvedAt: `${YEAR}-08-03T09:14:00Z`,
    lastUsedAt: `${YEAR}-08-25T06:55:00Z`,
    tenants: [
      {
        id: TENANTS[0].id,
        name: TENANTS[0].name,
        scopes: ['buchfink:lesen', 'buchfink:buchen'],
        grantedAt: `${YEAR}-08-03T09:14:00Z`,
      },
    ],
    revoked: false,
  },
  {
    clientId: '41b9d3c5-claude-desktop',
    name: 'Claude Desktop',
    channel: 'agent:claude-desktop',
    approvedAt: `${YEAR}-08-11T16:40:00Z`,
    lastUsedAt: `${YEAR}-08-22T17:05:00Z`,
    tenants: [
      {
        id: TENANTS[0].id,
        name: TENANTS[0].name,
        scopes: ['buchfink:lesen'],
        grantedAt: `${YEAR}-08-11T16:40:00Z`,
      },
    ],
    revoked: false,
  },
];

const COUNTRIES = [
  { code: 'DE', name: 'Deutschland' },
  { code: 'AT', name: 'Österreich' },
  { code: 'CH', name: 'Schweiz' },
  { code: 'NL', name: 'Niederlande' },
  { code: 'FR', name: 'Frankreich' },
];

/** Kapitalregeln der beiden Rechtsformen, die Buchfink führt. */
const FOUNDATION_RULES = [
  {
    legalForm: 'GmbH',
    minShareCapital: c(25000),
    paidInPerShareQuota: 0.25,
    paidInFloor: c(12500),
    paidInFloorIsFullCapital: false,
    cashOnly: false,
    legalReserve: false,
    reference: '§ 5 Abs. 1, § 7 Abs. 2 GmbHG',
    note: '',
  },
  {
    legalForm: 'UG (haftungsbeschränkt)',
    minShareCapital: c(1),
    paidInPerShareQuota: 1,
    paidInFloor: 0,
    paidInFloorIsFullCapital: true,
    cashOnly: true,
    legalReserve: true,
    reference: '§ 5a GmbHG',
    note: 'Ein Viertel des Jahresüberschusses fließt in die gesetzliche Rücklage.',
  },
];

/** Hebesatz Hamburg und die Vorgaben der Abgrenzung. */
const CLOSING_SETTINGS = {
  tradeTaxRatePercent: 470,
  accrualMethod: 'monthly',
  accrualThreshold: c(800),
  accrualRelease: 'yearly',
};

const FISCAL_YEAR_CANDIDATES = {
  next: { year: YEAR + 1, startDate: `${YEAR + 1}-01-01`, endDate: `${YEAR + 1}-12-31` },
  previous: {
    year: 2023,
    startDate: '2023-01-01',
    endDate: '2023-12-31',
    blocked: 'Die Bücher beginnen am 01.01.2024.',
  },
};

/** Das in der Kopfzeile gewählte Geschäftsjahr. */
let viewYear = YEAR;

// -------------------------------------------------------------------------
// Unternehmen, Einrichtung und Zustimmung

/** Ein Darlehen der Mehrheitsgesellschafterin an die GmbH. */
const SHAREHOLDER_LOANS = [
  {
    loan: {
      id: 1,
      contactId: 20,
      direction: 'to_company',
      principal: c(20000),
      interestRateBp: 350,
      dayCount: 'act/365',
      contractDate: '2025-02-20',
      startDate: '2025-03-01',
      maturityDate: '2028-02-29',
      account: '3642',
      interestAccount: '7317',
      collateral: '',
      subordinated: true,
    },
    contactName: 'Anna Bauer',
    outstanding: c(20000),
    firstMovement: '2025-03-03',
    interestBookedThrough: '2025-12-31',
    findings: [],
  },
];

/** Die Pflichten aus den Stammdaten: der Eintritt von Jonas Peters 2026. */
const COMPANY_DEADLINES = [
  {
    ...deadline('gesellschafterliste.2026-03', 'Gesellschafterliste einreichen (Eintritt Jonas Peters)',
      `${YEAR}-03-31`, 'März 2026', '§ 40 Abs. 2 GmbHG', true),
    doneOn: `${YEAR}-03-12`,
    description: 'Der Notar reicht die neue Liste beim Handelsregister ein.',
  },
  {
    ...deadline('transparenzregister.2026-03', 'Transparenzregister: wirtschaftlich Berechtigte melden',
      `${YEAR}-11-15`, 'März 2026', '§ 20 Abs. 1 GwG', false),
    description: 'Ändern sich die wirtschaftlich Berechtigten, ist das Transparenzregister zu aktualisieren.',
    confirmable: true,
  },
];

/** Steht `mock:consent` im lokalen Speicher, meldet sich Claude Code an. */
const CONSENT_REQUEST = {
  id: 'anfrage-1',
  nonce: 'nonce-1',
  clientName: 'Claude Code',
  clientId: 'c7f1e2a0-claude-code',
  clientRegisteredAt: `${YEAR}-08-25T08:11:00Z`,
  knownSince: null,
  redirectUri: 'http://127.0.0.1:53712/callback',
  redirectTo: '',
  scopes: ['buchfink:lesen', 'buchfink:buchen'],
  tenantId: TENANTS[0].id,
  tenantName: TENANTS[0].name,
  userCode: 'KXRT7M2Q',
  expiresAt: new Date(Date.now() + 5 * 60 * 1000).toISOString(),
};

/** Schalter für die Aufnahmen der Einrichtung und der Zustimmung. */
function mockFlag(name: string): string | null {
  try {
    return window.localStorage.getItem(`mock:${name}`);
  } catch {
    return null;
  }
}

const SCHEMA_MIGRATIONS = [
  { id: 2, runAt: `${YEAR}-08-01T07:58:00Z`, appVersion: '0.3.0', fromVersion: 41, toVersion: 44, result: 'ok', tables: 'shareholder_loans, obligation_records', message: '' },
  { id: 1, runAt: '2024-01-08T09:12:00Z', appVersion: '0.2.0', fromVersion: 0, toVersion: 41, result: 'ok', tables: 'Ersteinrichtung', message: '' },
];

const VAT_ID_STATUS = {
  contactId: 1,
  vatId: 'DE289512347',
  confirmed: true,
  latest: { checkedAt: `${YEAR}-07-02T10:14:00Z`, resultCode: '200', requestId: 'BZST-2026-07-02-4711' },
  validityDays: 365,
  note: 'Gültig bestätigt.',
};

const unsupported = (name: string) => () =>
  Promise.reject(new Error(`${name} ist in der Screenshot-Vorschau nicht verfügbar.`));

export const bridge = {
  // Mandanten
  GetTenants: () => later(TENANTS),
  GetActiveTenant: () => later(TENANTS[0]),
  SwitchTenant: () => later(undefined),
  DeleteTenant: unsupported('DeleteTenant'),

  GetBankAccounts: () => later(BANK_ACCOUNTS),
  PreviewBankStatement: unsupported('PreviewBankStatement'),
  ConfigureBankAccounts: unsupported('ConfigureBankAccounts'),
  SetInvoiceBankAccount: unsupported('SetInvoiceBankAccount'),

  // Eingang, Regeln für Apps, Standardgruppe
  GetInbox: () => later(INBOX_INFO),
  ListInbox: () => later(INBOX_FILES, 60),
  OpenInbox: () => later(undefined),
  AddToInbox: unsupported('AddToInbox'),
  SetInboxFolder: unsupported('SetInboxFolder'),
  CheckFolder: (path: string) => later(folderCheck(path)),
  GetAgentRules: () => later(AGENT_RULES),
  SaveAgentRules: unsupported('SaveAgentRules'),
  ConfirmDefaultPostingGroup: unsupported('ConfirmDefaultPostingGroup'),

  // Verbundene Apps
  GetMCPServerStatus: () => later(MCP_STATUS),
  SetMCPPort: unsupported('SetMCPPort'),
  GetConnectedApps: () => later(mockFlag('setup') ? [] : CONNECTED_APPS),
  RevokeConnectedApp: unsupported('RevokeConnectedApp'),
  GetPendingAppRequests: () => later(mockFlag('consent') ? [CONSENT_REQUEST] : []),
  ResolveAppRequest: () => {
    window.localStorage.removeItem('mock:consent');
    return later(undefined);
  },
  OpenAgentGuide: unsupported('OpenAgentGuide'),

  // Stammdaten: Darlehen, Eintragung, Steuernummer
  GetShareholderLoans: () => later(SHAREHOLDER_LOANS),
  RegisterShareholderLoan: unsupported('RegisterShareholderLoan'),
  CorrectShareholderLoan: unsupported('CorrectShareholderLoan'),
  RegisterCompany: unsupported('RegisterCompany'),
  FormatElsterTaxNumber: (taxNumber: string, state: string) => {
    const digits = taxNumber.replace(/\D/g, '');
    if (!digits || !state) return later('');
    // Hamburg: zehn Ziffern, Länderpräfix 22, eingefügte 0 an fünfter Stelle.
    if (state === 'HH' && digits.length === 10) return later(`22${digits.slice(0, 2)}0${digits.slice(2)}`);
    return Promise.reject(new Error(`Eine Steuernummer aus ${state} hat in der Vorschau zehn Ziffern.`));
  },
  GetCountries: () => later(COUNTRIES),
  GetFiscalYearCandidates: () => later(FISCAL_YEAR_CANDIDATES),
  BookLegalReserve: unsupported('BookLegalReserve'),

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
      lastBackupAt: '2026-08-24T18:40:12Z',
      // Mit `mock:setup` startet die Vorschau in der Einrichtung.
      ...(mockFlag('setup') === '1' ? { isConfigured: false, tenants: [], activeTenantId: '' } : {}),
    }),
  ValidateTenantDirectory: () => later(undefined, 80),
  SetupApplication: () => {
    window.localStorage.setItem('mock:setup', 'angelegt');
    return later(undefined, 300);
  },
  LoadExistingDatabase: unsupported('LoadExistingDatabase'),
  SelectDirectoryDialog: unsupported('SelectDirectoryDialog'),
  SelectDatabaseFileDialog: unsupported('SelectDatabaseFileDialog'),

  // Geschäftsjahr & Stammdaten
  // Die Kopfzeile schaltet das Geschäftsjahr um; die Vorschau merkt es sich,
  // damit die Ansicht nach dem Umschalten das gewählte Jahr behält.
  GetFiscalYear: () => later(viewYear),
  SetFiscalYear: (year: number) => {
    viewYear = year;
    return later(undefined);
  },
  GetAvailableFiscalYears: () => later([2024, 2025, 2026]),
  GetCompanySettings: () => later(SETTINGS),
  UpdateCompanySettings: () => later(undefined),

  // Konten
  GetAccounts: () => later(ACCOUNTS),
  GetAccountLedger: (accountNumber: string) => later(ledgerFor(accountNumber)),
  GetSuSaOverview: () => later(susa()),
  GetPaymentAccounts: () => later(ACCOUNTS.filter((x) => x.number === '1800')),

  // Kontierung
  GetPostingGroups: () => later(POSTING_GROUPS),
  GetTaxTreatments: () => later(TAX_TREATMENTS),

  // Journal
  GetJournalEntries: () => later(ENTRIES),
  GetAllJournalEntries: () => later(ENTRIES),
  PostIncomingReceipt: unsupported('PostIncomingReceipt'),
  VerifyIntegrity: () => later(INTEGRITY, 400),

  // Belege
  SelectReceiptFilesDialog: () => later([]),
  FileIncomingReceipt: unsupported('FileIncomingReceipt'),
  GetReceipts: (status: string) =>
    later(status ? RECEIPTS.filter((r) => r.status === status) : RECEIPTS),
  GetReceiptPreview: (receiptId: number) =>
    later({
      dataUrl: receiptPreview(),
      fileName: RECEIPTS.find((r) => r.id === receiptId)?.files[0].fileName ?? '',
      mimeType: 'image/png',
      intact: true,
    }),
  ProposeFromEInvoice: (receiptId: number) => later(proposalFor(receiptId)),
  GetEInvoiceRules: () => later(['BR-1', 'BR-2', 'BR-CO-10', 'BR-DE-1', 'BR-DE-15']),
  PreviewIncomingReceipt: (request: any) => later(previewIncoming(request), 120),
  PreviewOutgoingInvoice: (invoice: any) => later(previewOutgoing(invoice), 120),

  // Bank & Zahlungen
  GetBankTransactions: () => later(BANK_TX),
  BookBankTransactionDirect: unsupported('BookBankTransactionDirect'),
  SettlePayment: unsupported('SettlePayment'),

  // Kontakte & Rechnungen
  GetContacts: () => later(CONTACTS),
  SaveContact: unsupported('SaveContact'),
  GetInvoices: () => later(INVOICES),
  IssueInvoice: unsupported('IssueInvoice'),
  GetInvoiceDocument: (invoiceId: number) =>
    later({
      dataUrl: receiptPreview(),
      fileName: `${INVOICES.find((i) => i.id === invoiceId)?.invoiceNumber ?? 'Rechnung'}.pdf`,
      mimeType: 'image/png',
      intact: true,
    }),

  // Anlagevermögen
  GetFixedAssets: (assetClass: string) =>
    later(assetClass ? ASSETS.filter((a) => a.class === assetClass) : ASSETS),
  GetFixedAsset: (id: number) => later(assetDetail(id)),
  GetAnlagenspiegel: () => later(ANLAGENSPIEGEL),
  GetAssetAcquisitionCandidates: () => later([]),

  // Nachgereichte Auskünfte, die eine Ansicht auf dem Weg von der Bilanz zum
  // Beleg nebenbei holt. Ohne sie bräche die Belegliste ab: sie lädt Belege,
  // Kontakte und Kontierung in einem Zug.
  GetFiscalYears: () => later(FISCAL_YEARS),
  GetSelectableContacts: () => later(CONTACTS),
  GetPaymentAllocations: () => later([]),
  GetProvisions: (year: number) => later(closings.provisions(year || YEAR)),

  // Jahresabschluss: Bilanz, GuV, Anhang, Größenklasse
  GetStatement: () => later(STATEMENT, 80),
  GetSizeClass: () => later(SIZE_CLASS),
  ExportStatementPDF: unsupported('ExportStatementPDF'),
  ExportStatementCSV: unsupported('ExportStatementCSV'),

  // Auswahllisten des Rechnungsdialogs. Ohne sie scheitert das Laden der
  // Rechnungsseite in einem Zug, und die Maske bliebe ohne Kunden.
  GetUnitCodes: () => later(UNIT_CODES),
  GetEInvoiceProfiles: () => later(EINVOICE_PROFILES),
  GetNumberGapReasons: () => later(NUMBER_GAP_REASONS),
  GetInvoiceNumberGaps: () => later({ fiscalYear: YEAR, issued: INVOICES.length, used: INVOICES.length, gaps: [] }),

  // Umsatzsteuer: das Kennziffernblatt, das der Monatsabschluss zeigt.
  // Geschrieben wird hier nichts — Bestätigung und Berichtigung sind Vorgänge
  // mit Folgen und stehen deshalb als „nicht verfügbar" da.
  GetVatReturn: (periodKey: string) => later(vatReturn(periodKey), 60),
  SaveVatReturn: (periodKey: string) => later(vatReturn(periodKey), 60),
  ExportVatReturnCSV: (id: number) => {
    const r = vatReturn(`${YEAR}-${pad(id)}`);
    const rows = r.figures.map((f) => `${f.code};${f.hasBase ? f.base / 100 : ''};${f.tax / 100}`);
    return later(['Kennziffer;Bemessungsgrundlage;Steuer', ...rows].join('\n') + '\n');
  },
  ConfirmVatReturnSubmitted: unsupported('ConfirmVatReturnSubmitted'),

  // Aufgaben, Monatsabschluss, Bankvorschlag
  GetTasks: () => later(TASKS, 60),
  GetMonthCloseState: (month: string) => later(monthCloseState(month), 60),
  SuggestBankMatches: (bankTxId: number) => later(bankSuggestions(bankTxId), 60),
  GetBankRules: () => later(BANK_RULES),
  DeleteBankRule: unsupported('DeleteBankRule'),

  // Übersicht
  GetNextObligations: (limit: number) =>
    later(limit > 0 ? OBLIGATIONS.slice(0, limit) : OBLIGATIONS, 40),
  GetObligation: (key: string) => {
    const found = OBLIGATIONS.find((o) => o.key === key);
    return found
      ? later(found, 40)
      : Promise.reject(new Error(`die Pflicht „${key}“ gibt es für diesen Mandanten nicht`));
  },
  ConfirmObligation: unsupported('ConfirmObligation'),
  WaiveObligation: unsupported('WaiveObligation'),
  ReopenObligation: unsupported('ReopenObligation'),
  GetAgentBookings: (limit: number) => later(agentBookings(limit), 40),

  // E-Bilanz, Audit & Festschreibung
  SaveEBilanzAs: (year: number, format: string) =>
    later(
      format === 'susa_csv'
        ? `${DATA_DIR}/Export/ebilanz-${year || YEAR}-susa.csv`
        : `${DATA_DIR}/Export/e-bilanz-${year || YEAR}.xbrl`,
      400,
    ),
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

  // Meldezeiträume, Prüfläufe, Abschlussweg, Nebenpflichten
  GetZMPeriods: () => later(zmPeriods()),
  GetZMReturn: (periodKey: string) => later(zmReturnFor(periodKey), 60),
  GetCheckRuns: () => later(CHECK_RUNS),
  GetEBilanzMappingReport: (year: number) => later(mappingReport(year), 80),
  GetOpenItemsAt: () => later(OPEN_ITEMS),
  GetClosingState: (year: number) => later(closingState(year), 60),
  GetClosingSteps: (year: number) => later(closingSteps(year), 60),
  GetCarryForwardPreview: (toYear: number) => later(carryForwardPreview(toYear), 60),
  GetInputTaxCorrections: (year: number) => later(inputTaxCorrections(year), 60),
  // Die Oberfläche liest hier nur die Pflichten zum Unternehmen (Stammdaten).
  GetDeadlines: () => later(COMPANY_DEADLINES, 60),
  GetObligations: (year: number) => later(obligationsOf(year || YEAR), 60),
  GetDeclaration: (kind: string, year: number) =>
    new Promise((resolve, reject) =>
      setTimeout(() => {
        try {
          resolve(closings.declaration(kind, year || YEAR));
        } catch (e) {
          reject(e);
        }
      }, 80),
    ),
  GetFoundationState: () => later(FOUNDATION_STATE),

  // Nachweise, Sicherung und Datenüberlassung
  GetAuditLogsFiltered: () => later(AUDIT_LOGS, 60),
  VerifyAuditChain: () => later(AUDIT_CHAIN, 200),
  GetBackupRuns: () => later(BACKUP_RUNS),
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
  RecordInvoiceNumberGapReason: unsupported('RecordInvoiceNumberGapReason'),
  SettleAdvance: unsupported('SettleAdvance'),
  SetAverageEmployees: unsupported('SetAverageEmployees'),
  CreateFiscalYear: unsupported('CreateFiscalYear'),
  CarryForward: unsupported('CarryForward'),
  SetFiscalYearStatus: unsupported('SetFiscalYearStatus'),
  ReopenFiscalYear: unsupported('ReopenFiscalYear'),
  SkipClosingStep: unsupported('SkipClosingStep'),
  ReopenClosingStep: unsupported('ReopenClosingStep'),
  GetAccruals: (year: number) => later(closings.accruals(year || YEAR)),
  GetProvisionMirror: (year: number) => later(closings.provisionMirror(year || YEAR)),

  PreviewVatSettlement: (year: number) => later(closings.vatSettlement(year || YEAR)),
  PreviewTaxProvision: (year: number) => later(closings.taxProvision(year || YEAR)),
  PreviewAppropriation: unsupported('PreviewAppropriation'),
  BookAppropriation: unsupported('BookAppropriation'),
  GetAppropriation: (year: number) => later(closings.appropriation(year || YEAR - 1)),
  GetNotesTexts: () => later(closings.notesTexts()),
  GetClosingSettings: () => later(CLOSING_SETTINGS),
  SaveClosingSettings: unsupported('SaveClosingSettings'),
  GetTaxElectionRegister: (year: number) => later(closings.taxElectionRegister(year || YEAR)),
  GetReconciliation: (year: number) => later(closings.reconciliation(year || YEAR)),
  CreateVatCorrection: unsupported('CreateVatCorrection'),
  SaveZMReturn: unsupported('SaveZMReturn'),
  ConfirmZMSubmitted: unsupported('ConfirmZMSubmitted'),
  RunChecks: (cutoff: string, periodType: string) => later(checkRunFor(cutoff, periodType), 120),
  SetDeclarationInput: unsupported('SetDeclarationInput'),
  ConfirmDeclarationInput: unsupported('ConfirmDeclarationInput'),
  DeleteDeclarationInput: unsupported('DeleteDeclarationInput'),
  SubmitDeclaration: unsupported('SubmitDeclaration'),
  ExportDeclarationJSON: unsupported('ExportDeclarationJSON'),
  SaveFoundation: unsupported('SaveFoundation'),
  GetFoundationRules: () => later(FOUNDATION_RULES),
  ExportZ3: unsupported('ExportZ3'),
  ExportArchive: unsupported('ExportArchive'),
  ExportAuditPackage: unsupported('ExportAuditPackage'),
  SelectExportDirectoryDialog: unsupported('SelectExportDirectoryDialog'),
  SelectSaveFileDialog: unsupported('SelectSaveFileDialog'),
  SaveReceiptFileAs: unsupported('SaveReceiptFileAs'),
  VerifyReceiptFiles: unsupported('VerifyReceiptFiles'),
  ImportCAMT053File: unsupported('ImportCAMT053File'),
  SelectStatementFileDialog: unsupported('SelectStatementFileDialog'),
  SetBackupDir: unsupported('SetBackupDir'),
  CreateBackup: unsupported('CreateBackup'),
  VerifyBackup: unsupported('VerifyBackup'),
  RestoreFromBackup: unsupported('RestoreFromBackup'),
  SelectBackupDirDialog: unsupported('SelectBackupDirDialog'),
  SelectBackupFileDialog: unsupported('SelectBackupFileDialog'),
  GetVatIDStatus: (contactId: number) => later({ ...VAT_ID_STATUS, contactId }),
  GetNonDeductibleReport: (year: number) => later(closings.nonDeductibleReport(year || YEAR)),
  PreviewCurrencyValuation: (year: number) => later(closings.currencyValuation(year || YEAR)),
  GetWriteUpReport: (year: number) => later(closings.writeUpReport(year || YEAR)),
  GetSchemaMigrations: () => later(SCHEMA_MIGRATIONS),
  GetMigrationRecords: () => later([]),
  SetSystemChangeDate: unsupported('SetSystemChangeDate'),
  GenerateProcedureDocumentation: unsupported('GenerateProcedureDocumentation'),
  SaveOrganisationTexts: unsupported('SaveOrganisationTexts'),
  SaveProcedureDocumentationAs: unsupported('SaveProcedureDocumentationAs'),
  OpenReleasesPage: unsupported('OpenReleasesPage'),
  BlockContact: unsupported('BlockContact'),
  EnableReadOnly: unsupported('EnableReadOnly'),
  DisableReadOnly: unsupported('DisableReadOnly'),

  // Die laufende Buchhaltung, gelesen aus denselben Beispieldaten wie das Journal.
  GetRetentionRules: () => later(RETENTION_RULES),
  GetVatPeriodProposal: (year: number) => later(vatPeriodProposal(year)),
  GetEntriesForReceipt: (receiptId: number) =>
    later(ENTRIES.filter((e: any) => e.receiptId === receiptId)),
  GetFilteredJournal: (filter: any) => later(filteredJournal(filter)),
  SaveFilteredJournalCSV: unsupported('SaveFilteredJournalCSV'),
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
        .reduce((sum, x) => sum + x.debitSum - x.creditSum, 0),
      totalSaldoCredit: accounts
        .filter((x) => x.debitSum < x.creditSum)
        .reduce((sum, x) => sum + x.creditSum - x.debitSum, 0),
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

/**
 * Das Kontoblatt wie im Backend: der laufende Saldo trägt das Vorzeichen der
 * natürlichen Seite des Kontos. Der Saldenvortrag steht als erste Buchung
 * gegen 9000 im Blatt.
 */
function ledgerFor(accountNumber: string) {
  const account = ACCOUNTS.find((x) => x.number === accountNumber)!;
  const debitPositive = !['liability', 'equity', 'revenue'].includes(account.type);
  let running = 0;
  const rows = ENTRIES.flatMap((entry) =>
    entry.lines
      .filter((line) => line.account === accountNumber)
      .map((line) => {
        const debit = line.side === 'S' ? line.amount : 0;
        const credit = line.side === 'H' ? line.amount : 0;
        running += debitPositive ? debit - credit : credit - debit;
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
          runningBalance: running,
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
    closingBalance: running,
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
    invoiceNumber: 'TP-26-4528',
    grossAmount: c(3232.04),
    matchedContact: true,
    notes: ['Die Buchungsgruppe bleibt offen: Welches Konto zutrifft, sagt keine Rechnung.'],
  };
}

/** Der Buchungssatz einer Ausgangsrechnung: Forderung an Erlöse und Umsatzsteuer. */
function previewOutgoing(invoice: any) {
  const contact = CONTACTS.find((x) => x.id === invoice.contactId);
  const netOf = (item: any) => Math.round((item.quantityMilli * item.unitPrice) / 1000);
  const items = invoice.items ?? [];
  const net = items.reduce((sum: number, item: any) => sum + netOf(item), 0);
  const tax = items.reduce(
    (sum: number, item: any) => sum + Math.round((netOf(item) * item.taxRate) / 10000),
    0,
  );
  const receivable = contact?.ledgerAccount ?? '1200';
  const lines = [
    { id: 1, entryId: 0, position: 1, side: 'S', amount: net + tax, account: receivable, accountName: contact?.name ?? accountName('1200') },
    { id: 2, entryId: 0, position: 2, side: 'H', amount: net, account: '4400', accountName: accountName('4400') },
    ...(tax > 0
      ? [{ id: 3, entryId: 0, position: 3, side: 'H', amount: tax, account: '3806', accountName: accountName('3806'), taxKey: '3' }]
      : []),
  ];
  return { lines, net, tax, gross: net + tax, balanced: true, warnings: [], inputTaxFindings: [], issuingFindings: [] };
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

  return { lines, net, tax, gross, balanced: true, warnings, inputTaxFindings: [] };
}

// -------------------------------------------------------------------------
// Fristen, Voranmeldungszeitraum, Journalfilter

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

/** Der Vorschlag zum Voranmeldungszeitraum aus der Steuer des Vorjahres. */
function vatPeriodProposal(year: number) {
  const target = year || YEAR;
  return {
    year: target,
    basedOnYear: target - 1,
    priorYearTax: c(57_431),
    current: 'month',
    proposed: 'month',
    changes: false,
    complete: true,
    missingPeriods: 0,
    reference: '§ 18 Abs. 2 UStG',
    note:
      `Die Steuer des Jahres ${target - 1} lag über 9.000 Euro; auch ${target} ist monatlich ` +
      'anzumelden. Das Finanzamt setzt den Zeitraum fest — Buchfink schlägt ihn nur vor.',
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
