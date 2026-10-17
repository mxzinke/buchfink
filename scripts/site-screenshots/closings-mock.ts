/**
 * Beispieldaten der Abschlüsse für die Screenshots: Pflichtenkalender,
 * Jahreserklärungen und die Karteien hinter den Abschlussbausteinen.
 *
 * Die Zahlen gehören zur Nordlicht Systeme GmbH aus `mock-bridge.ts`: Stand
 * 25.08.2026, vierteljährliche Voranmeldung, das erste Halbjahr ist
 * festgeschrieben und übermittelt, das Geschäftsjahr 2025 ist festgestellt.
 *
 * `declarations.json` enthält Erklärungen, wie sie das Paket
 * internal/declaration für einen frischen Mandanten rechnet; Werte, Angaben
 * und Befunde sind auf die Nordlicht gesetzt.
 */

import DECLARATIONS from './declarations.json';

const YEAR = 2026;
const TODAY = `${YEAR}-08-25`;
const pad = (n: number) => String(n).padStart(2, '0');
const c = (euro: number) => Math.round(euro * 100);

// -------------------------------------------------------------------------
// Pflichtenkalender

interface ObligationSpec {
  kind: string;
  title: string;
  basis: string;
  description: string;
  channel: string;
  evidenceSource: 'data' | 'human' | 'either';
  responsibility?: string;
}

const KINDS: Record<string, ObligationSpec> = {
  festschreibung: {
    kind: 'festschreibung',
    title: 'Monatsfestschreibung',
    basis: '§ 146 Abs. 4 AO',
    description: 'Der Monat wird festgeschrieben, bevor seine Voranmeldung übermittelt wird.',
    channel: 'intern',
    evidenceSource: 'data',
  },
  ustva: {
    kind: 'ustva',
    title: 'Umsatzsteuer-Voranmeldung',
    basis: '§ 18 Abs. 1 UStG',
    description: 'Die Voranmeldung ist bis zum 10. Tag nach Ablauf des Zeitraums zu übermitteln.',
    channel: 'elster',
    evidenceSource: 'data',
  },
  'lohnsteuer-anmeldung': {
    kind: 'lohnsteuer-anmeldung',
    title: 'Lohnsteuer-Anmeldung',
    basis: '§ 41a EStG',
    description: 'Das Lohnbüro übermittelt die Anmeldung; Buchfink kennt sie nur aus Ihrer Bestätigung.',
    channel: 'lohnanbieter',
    evidenceSource: 'human',
    responsibility: 'human_confirms',
  },
  'kst-vorauszahlung': {
    kind: 'kst-vorauszahlung',
    title: 'Körperschaftsteuer-Vorauszahlung',
    basis: '§ 31 KStG, § 37 EStG',
    description: 'Vorauszahlung laut Bescheid, fällig am 10. März, Juni, September und Dezember.',
    channel: 'elster',
    evidenceSource: 'either',
  },
  'gewst-vorauszahlung': {
    kind: 'gewst-vorauszahlung',
    title: 'Gewerbesteuer-Vorauszahlung',
    basis: '§ 19 GewStG',
    description: 'Vorauszahlung laut Bescheid der Gemeinde, fällig am 15. Februar, Mai, August und November.',
    channel: 'elster',
    evidenceSource: 'either',
  },
  abschlussbuchungen: {
    kind: 'abschlussbuchungen',
    title: 'Abschlussbuchungen zum Bilanzstichtag',
    basis: '§ 242 HGB',
    description: 'Abschreibung, Abgrenzung, Rückstellungen und Steuern zum Stichtag.',
    channel: 'intern',
    evidenceSource: 'data',
  },
  aufstellung: {
    kind: 'aufstellung',
    title: 'Aufstellung des Jahresabschlusses',
    basis: '§ 264 Abs. 1 HGB',
    description: 'Die Geschäftsführung stellt den Abschluss in den ersten sechs Monaten auf.',
    channel: 'intern',
    evidenceSource: 'data',
  },
  feststellung: {
    kind: 'feststellung',
    title: 'Feststellung des Jahresabschlusses',
    basis: '§ 42a Abs. 2 GmbHG',
    description: 'Die Gesellschafter beschließen innerhalb von elf Monaten über den Abschluss.',
    channel: 'intern',
    evidenceSource: 'data',
  },
  ergebnisverwendung: {
    kind: 'ergebnisverwendung',
    title: 'Beschluss über die Ergebnisverwendung',
    basis: '§ 29 GmbHG',
    description: 'Die Gesellschafter beschließen, was mit dem Ergebnis geschieht.',
    channel: 'intern',
    evidenceSource: 'data',
  },
  offenlegung: {
    kind: 'offenlegung',
    title: 'Offenlegung im Unternehmensregister',
    basis: '§ 325 HGB',
    description: 'Einreichung bis zum Ende des zwölften Monats nach dem Stichtag.',
    channel: 'unternehmensregister',
    evidenceSource: 'data',
  },
  'e-bilanz': {
    kind: 'e-bilanz',
    title: 'E-Bilanz',
    basis: '§ 5b EStG',
    description: 'Übermittlung von Bilanz und GuV nach amtlichem Datensatz über ein Programm mit ERiC.',
    channel: 'elster',
    evidenceSource: 'human',
  },
  'kst-erklaerung': {
    kind: 'kst-erklaerung',
    title: 'Körperschaftsteuererklärung',
    basis: '§ 31 KStG, § 149 AO',
    description: 'Abgabe in Mein ELSTER, ohne Steuerberater bis zum 31. Juli des Folgejahres.',
    channel: 'elster',
    evidenceSource: 'either',
  },
  'gewst-erklaerung': {
    kind: 'gewst-erklaerung',
    title: 'Gewerbesteuererklärung',
    basis: '§ 14a GewStG, § 149 AO',
    description: 'Abgabe in Mein ELSTER, ohne Steuerberater bis zum 31. Juli des Folgejahres.',
    channel: 'elster',
    evidenceSource: 'either',
  },
  'ust-erklaerung': {
    kind: 'ust-erklaerung',
    title: 'Umsatzsteuer-Jahreserklärung',
    basis: '§ 18 Abs. 3 UStG, § 149 AO',
    description: 'Abgabe in Mein ELSTER, ohne Steuerberater bis zum 31. Juli des Folgejahres.',
    channel: 'elster',
    evidenceSource: 'either',
  },
};

const MONTH_NAMES = [
  'Januar', 'Februar', 'März', 'April', 'Mai', 'Juni',
  'Juli', 'August', 'September', 'Oktober', 'November', 'Dezember',
];

function lastDay(year: number, month: number): string {
  return `${year}-${pad(month)}-${pad(new Date(Date.UTC(year, month, 0)).getUTCDate())}`;
}

interface Options {
  periodKey: string;
  periodLabel: string;
  from?: string;
  to?: string;
  due: string;
  statutory?: string;
  shift?: string;
  done?: { date: string; kind: string; reference?: string; automatic?: boolean };
  prepared?: string;
}

function obligation(year: number, spec: ObligationSpec, o: Options) {
  const status = o.done ? 'done' : o.due < TODAY ? 'overdue' : o.prepared ? 'prepared' : 'open';
  return {
    key: `${spec.kind}.${o.periodKey}`,
    kind: spec.kind,
    title: spec.title,
    description: spec.description,
    basis: spec.basis,
    channel: spec.channel,
    responsibility: spec.responsibility ?? 'agent_prepares',
    evidenceSource: spec.evidenceSource,
    skillRef: `buchfink-grundlagen/references/pflichten.md#${spec.kind}`,
    periodKey: o.periodKey,
    periodLabel: o.periodLabel,
    periodFrom: o.from,
    periodTo: o.to,
    fiscalYear: year,
    statutoryDate: o.statutory ?? o.due,
    dueDate: o.due,
    shiftReason: o.shift,
    status,
    proof: o.done
      ? { date: o.done.date, kind: o.done.kind, reference: o.done.reference, automatic: o.done.automatic ?? true }
      : undefined,
    preparation: o.prepared
      ? { at: o.prepared, channel: 'agent:claude-code', actor: 'anna via Claude Code' }
      : undefined,
    hints: [],
  };
}

/** Die Pflichten eines Geschäftsjahres, wie GetObligations sie liefert. */
export function obligations(year: number) {
  const y = year || YEAR;
  const out: ReturnType<typeof obligation>[] = [];
  if (y === YEAR) {
    for (let m = 1; m <= 12; m++) {
      const to = lastDay(y, m);
      const dueMonth = m === 12 ? 1 : m + 1;
      const dueYear = m === 12 ? y + 1 : y;
      out.push(
        obligation(y, KINDS.festschreibung, {
          periodKey: `${y}-${pad(m)}`,
          periodLabel: `${MONTH_NAMES[m - 1]} ${y}`,
          from: `${y}-${pad(m)}-01`,
          to,
          due: lastDay(dueYear, dueMonth),
          done: m <= 6 ? { date: m <= 3 ? '2026-04-09' : '2026-07-10', kind: 'commitment', reference: m <= 3 ? 'Q1 2026' : 'Q2 2026' } : undefined,
        }),
      );
      out.push(
        obligation(y, KINDS['lohnsteuer-anmeldung'], {
          periodKey: `${y}-${pad(m)}`,
          periodLabel: `${MONTH_NAMES[m - 1]} ${y}`,
          from: `${y}-${pad(m)}-01`,
          to,
          due: `${dueYear}-${pad(dueMonth)}-10`,
          done: m <= 7 ? { date: `${dueYear}-${pad(dueMonth)}-08`, kind: 'transfer_ticket', reference: `LSt-${pad(m)}-88${m}31`, automatic: false } : undefined,
        }),
      );
    }
    const quarterDue = ['2026-04-10', '2026-07-10', '2026-10-12', '2027-01-11'];
    for (let q = 1; q <= 4; q++) {
      out.push(
        obligation(y, KINDS.ustva, {
          periodKey: `${y}-Q${q}`,
          periodLabel: `${q}. Quartal ${y}`,
          from: `${y}-${pad(q * 3 - 2)}-01`,
          to: lastDay(y, q * 3),
          due: quarterDue[q - 1],
          statutory: q === 3 ? '2026-10-10' : q === 4 ? '2027-01-10' : undefined,
          shift: q === 3 ? 'Der 10.10.2026 ist ein Samstag.' : q === 4 ? 'Der 10.01.2027 ist ein Sonntag.' : undefined,
          done: q <= 2 ? { date: quarterDue[q - 1], kind: 'transfer_ticket', reference: `TT-2026${pad(q * 3)}-4711${q}` } : undefined,
        }),
      );
    }
    const kstDue = ['2026-03-10', '2026-06-10', '2026-09-10', '2026-12-10'];
    const gewstDue = ['2026-02-16', '2026-05-15', '2026-08-17', '2026-11-16'];
    for (let q = 1; q <= 4; q++) {
      out.push(
        obligation(y, KINDS['kst-vorauszahlung'], {
          periodKey: `${y}-Q${q}`,
          periodLabel: `${q}. Quartal ${y}`,
          due: kstDue[q - 1],
          done: q <= 2 ? { date: kstDue[q - 1], kind: 'booking', reference: `B-2026-0${q === 1 ? '041' : '118'}` } : undefined,
        }),
      );
      out.push(
        obligation(y, KINDS['gewst-vorauszahlung'], {
          periodKey: `${y}-Q${q}`,
          periodLabel: `${q}. Quartal ${y}`,
          due: gewstDue[q - 1],
          done: q <= 3 ? { date: gewstDue[q - 1], kind: 'booking', reference: `B-2026-0${q === 1 ? '022' : q === 2 ? '097' : '171'}` } : undefined,
        }),
      );
    }
  }
  const annual = (kind: string, due: string, done?: Options['done']) =>
    obligation(y, KINDS[kind], { periodKey: String(y), periodLabel: `Geschäftsjahr ${y}`, due, done });
  if (y === YEAR) {
    out.push(
      annual('abschlussbuchungen', '2027-03-31'),
      annual('aufstellung', '2027-06-30'),
      annual('feststellung', '2027-11-30'),
      annual('ergebnisverwendung', '2027-11-30'),
      annual('offenlegung', '2027-12-31'),
      annual('e-bilanz', '2027-08-02'),
      annual('kst-erklaerung', '2027-08-02'),
      annual('gewst-erklaerung', '2027-08-02'),
      annual('ust-erklaerung', '2027-08-02'),
    );
  } else if (y === YEAR - 1) {
    out.push(
      annual('abschlussbuchungen', '2026-03-31', { date: '2026-03-12', kind: 'status' }),
      annual('aufstellung', '2026-06-30', { date: '2026-03-20', kind: 'status' }),
      annual('feststellung', '2026-11-30', { date: '2026-05-14', kind: 'status' }),
      annual('ergebnisverwendung', '2026-11-30', { date: '2026-05-14', kind: 'resolution' }),
      annual('offenlegung', '2026-12-31'),
      annual('e-bilanz', '2026-07-31', { date: '2026-06-18', kind: 'transfer_ticket', reference: 'ET-2025-90817', automatic: false }),
      annual('kst-erklaerung', '2026-07-31', { date: '2026-06-24', kind: 'transfer_ticket', reference: 'TT-2025-KST-55310' }),
      annual('gewst-erklaerung', '2026-07-31', { date: '2026-06-24', kind: 'transfer_ticket', reference: 'TT-2025-GEW-55311' }),
      annual('ust-erklaerung', '2026-07-31', { date: '2026-06-24', kind: 'transfer_ticket', reference: 'TT-2025-UST-55312' }),
    );
  }
  return out.sort((a, b) => (a.dueDate < b.dueDate ? -1 : a.dueDate > b.dueDate ? 1 : 0));
}

// -------------------------------------------------------------------------
// Jahreserklärungen

/** Eine Jahreserklärung mit Übersicht, wie GetDeclaration sie liefert. */
export function declaration(kind: string, year: number) {
  const view = (DECLARATIONS as Record<string, unknown>)[kind];
  if (!view) throw new Error(`unbekannte Erklärung „${kind}" (erwartet ust_jahr, kst oder gewst)`);
  if ((year || YEAR) !== YEAR) {
    throw new Error(`Für ${year} liegt in den Beispieldaten keine Erklärung vor.`);
  }
  return structuredClone(view);
}

// -------------------------------------------------------------------------
// Prüfbericht vor einer Festschreibung

export function checkRun(cutoff: string, periodType: string) {
  const year = Number(cutoff.slice(0, 4)) || YEAR;
  const findings =
    periodType === 'year'
      ? []
      : [
          {
            id: 11,
            checkRunId: 0,
            rule: 'bank_unmatched',
            severity: 'warning',
            objectType: 'BANK_TX',
            objectId: '18',
            message: 'Ein Bankumsatz vom 28.07.2026 über 1.240,00 € ist keinem Beleg zugeordnet.',
          },
          {
            id: 12,
            checkRunId: 0,
            rule: 'receipt_unbooked',
            severity: 'warning',
            objectType: 'RECEIPT',
            objectId: '2',
            message: 'Der Beleg ER-2026-0142 vom 30.07.2026 ist abgelegt, aber nicht gebucht.',
          },
        ];
  return {
    id: 0,
    fiscalYear: year,
    cutoffDate: cutoff,
    periodType,
    checkedEntries: periodType === 'year' ? 214 : 31,
    checkedReceipts: periodType === 'year' ? 188 : 27,
    checkedBankTx: periodType === 'year' ? 402 : 58,
    findings,
    createdAt: `${TODAY}T08:40:00Z`,
  };
}

// -------------------------------------------------------------------------
// Karteien der Abschlussbausteine

export function accruals(year: number) {
  return [
    {
      id: 1,
      fiscalYear: year,
      kind: 'active',
      text: 'Kfz-Versicherung Pkw HH-NL 204, Dezember bis November',
      totalAmount: c(1440),
      deferredAmount: c(1320),
      startDate: `${year}-12-01`,
      endDate: `${year + 1}-11-30`,
      cutoffDate: `${year}-12-31`,
      account: '6520',
      method: 'monthly',
      formationEntryId: 411,
      releases: [],
      createdAt: `${year + 1}-02-12T10:00:00Z`,
    },
    {
      id: 2,
      fiscalYear: year,
      kind: 'active',
      text: 'Jahreslizenz Entwicklungsumgebung, ab Oktober',
      totalAmount: c(4320),
      deferredAmount: c(3240),
      startDate: `${year}-10-01`,
      endDate: `${year + 1}-09-30`,
      cutoffDate: `${year}-12-31`,
      account: '6495',
      method: 'monthly',
      formationEntryId: 412,
      releases: [],
      createdAt: `${year + 1}-02-12T10:02:00Z`,
    },
  ];
}

function mirrorRow(kind: string, label: string, account: string, opening: number, additions: number, used: number) {
  return { kind, label, account, opening, additions, used, released: 0, unwinding: 0, closing: opening + additions - used };
}

export function provisionMirror(year: number) {
  const rows = [
    mirrorRow('closing_costs', 'Jahresabschluss- und Prüfungskosten', '3070', c(2800), c(3200), c(2800)),
    mirrorRow('retention_costs', 'Aufbewahrung der Geschäftsunterlagen', '3070', c(1450), c(120), 0),
    mirrorRow('tax_trade', 'Gewerbesteuer', '3040', c(3906), 0, c(3906)),
  ];
  const sum = (key: 'opening' | 'additions' | 'used' | 'released' | 'unwinding' | 'closing') =>
    rows.reduce((total, row) => total + row[key], 0);
  return {
    fiscalYear: year,
    rows,
    total: {
      kind: 'total', label: 'Summe', account: '', opening: sum('opening'), additions: sum('additions'),
      used: sum('used'), released: sum('released'), unwinding: sum('unwinding'), closing: sum('closing'),
    },
  };
}

export function provisions(year: number) {
  return [
    {
      id: 1, fiscalYear: year, kind: 'closing_costs', text: `Erstellung Jahresabschluss ${year}`,
      settlementAmount: c(3200), expectedDate: `${year + 1}-04-30`, discountedAmount: c(3200),
      balanceAccount: '3070', expenseAccount: '6827', reason: 'Gesetzliche Pflicht zur Aufstellung',
      movements: [], createdAt: `${year + 1}-02-14T09:00:00Z`,
    },
    {
      id: 2, fiscalYear: year, kind: 'retention_costs', text: 'Aufbewahrung der Unterlagen, zehn Jahre',
      settlementAmount: c(1570), expectedDate: `${year + 10}-12-31`, discountedAmount: c(1570),
      balanceAccount: '3070', expenseAccount: '6827', reason: '§ 257 HGB, § 147 AO',
      movements: [], createdAt: `${year + 1}-02-14T09:05:00Z`,
    },
  ];
}

export function vatSettlement(year: number) {
  return {
    fiscalYear: year, cutoff: `${year}-12-31`, rows: [],
    inputTax: c(22898.8), outputTax: c(92378), prepaid: c(50787), payable: c(18692.2), refund: 0,
    lines: [], bookingDate: `${year}-12-31`, explanation: '',
  };
}

export function taxProvision(year: number) {
  return {
    fiscalYear: year, cutoff: `${year}-12-31`,
    input: { profitBeforeTax: c(126544), nonDeductible: 0, tradeTaxRatePercent: 470, prepaidCorporate: c(7000), prepaidTrade: c(12000), date: `${year}-12-31` },
    taxableIncome: c(126544), corporateTax: c(18981), solidarity: c(1043.95),
    tradeIncome: c(126500), tradeBase: c(4427.5), tradeTax: c(20809.25),
    incomeProvision: c(13024.95), tradeProvision: c(8809.25), incomeRefund: 0, tradeRefund: 0,
    ratesUsed: 'KSt 15 %, SolZ 5,5 %, Hebesatz 470 %', lines: [], explanation: '',
    warning: 'Die Rechnung ist eine Schätzung auf den Stand von heute. Gebucht wird erst zum Abschluss.',
  };
}

export function appropriation(year: number) {
  if (year !== YEAR - 1) return null;
  return {
    year, decisionDate: '2026-05-14', text: 'Gesellschafterbeschluss vom 14.05.2026',
    netIncome: c(61230), legalReserve: 0, otherReserves: 0, distribution: 0,
    withholdingTax: 0, solidarityOnWithholding: 0, carryForward: c(61230), createdAt: '2026-05-14T15:00:00Z',
  };
}

export function notesTexts() {
  return [
    { section: 'methods', label: 'Bilanzierungs- und Bewertungsmethoden', hint: '', basis: '§ 284 Abs. 2 Nr. 1 HGB', text: 'Die Gesellschaft bilanziert nach den Vorschriften für kleine Kapitalgesellschaften. Das Sachanlagevermögen ist zu Anschaffungskosten abzüglich planmäßiger linearer Abschreibungen bewertet.' },
    { section: 'board', label: 'Geschäftsführung', hint: '', basis: '§ 285 Nr. 10 HGB', text: 'Anna Lindqvist, Kauffrau, Hamburg' },
    { section: 'subsequent', label: 'Vorgänge nach dem Stichtag', hint: '', basis: '§ 285 Nr. 33 HGB', text: '' },
    { section: 'commitments', label: 'Sonstige finanzielle Verpflichtungen', hint: '', basis: '§ 285 Nr. 3a HGB', text: 'Mietvertrag Hafenstraße 14 bis 31.12.2028, jährlich 43.200 €.' },
  ];
}

export function reconciliation(year: number) {
  return { fiscalYear: year, cutoff: `${year}-12-31`, rows: [], equityEffect: 0, note: '' };
}

export function taxElectionRegister(year: number) {
  return { fiscalYear: year, rows: [], totalDifference: 0, totalBookValue: 0, totalTaxBookValue: 0, note: '' };
}

export function nonDeductibleReport(year: number) {
  return {
    fiscalYear: year,
    giftLimit: c(50),
    categories: [
      { key: 'gifts', label: 'Geschenke an Geschäftspartner', reference: '', note: '', deductibleAmount: c(186), nonDeductibleAmount: 0, total: c(186), count: 6 },
      { key: 'entertainment', label: 'Bewirtung aus geschäftlichem Anlass', reference: '', note: '', deductibleAmount: c(512.4), nonDeductibleAmount: c(219.6), total: c(732), count: 9 },
    ],
    recipients: [
      { recipientKey: 'kontor', recipientName: 'Kontor 7 Logistik GmbH', total: c(48), overLimit: false, toRebook: [], bookings: [{}, {}], note: '' },
      { recipientKey: 'haverkamp', recipientName: 'Haverkamp & Söhne KG', total: c(36), overLimit: false, toRebook: [], bookings: [{}], note: '' },
    ],
    note: '',
  };
}

export function currencyValuation(year: number) {
  return {
    fiscalYear: year, cutoff: `${year}-12-31`, reversalDate: `${year + 1}-01-01`,
    items: [], totalGain: 0, totalLoss: 0, note: '',
  };
}

export function writeUpReport(year: number) {
  return { fiscalYear: year, candidates: [], open: 0, note: '' };
}
