/**
 * Beispieldaten der Abschlüsse für die Screenshots: Pflichtenkalender,
 * Jahreserklärungen und die Karteien hinter den Abschlussbausteinen.
 *
 * Die Zahlen gehören zur Nordlicht Systeme GmbH aus `mock-bridge.ts`: Stand
 * 25.08.2026, monatliche Voranmeldung mit Dauerfristverlängerung, Januar bis
 * Juni sind festgeschrieben und übermittelt, das Geschäftsjahr 2025 ist
 * festgestellt.
 *
 * `declarations.json` enthält Erklärungen, wie sie das Paket
 * internal/declaration für einen frischen Mandanten rechnet; Werte, Angaben
 * und Befunde sind auf die Nordlicht gesetzt.
 */

import DECLARATIONS from './declarations.json';
import { VAT_DUE, prepaymentEntry } from './journal-mock';

const YEAR = 2026;
const TODAY = `${YEAR}-08-25`;
const pad = (n: number) => String(n).padStart(2, '0');
const c = (euro: number) => Math.round(euro * 100);

/** Der nächste Werktag; Feiertage kennt die Vorschau nicht. */
function workday(iso: string): { date: string; shift?: string } {
  const d = new Date(`${iso}T00:00:00Z`);
  const day = d.getUTCDay();
  if (day !== 0 && day !== 6) return { date: iso };
  d.setUTCDate(d.getUTCDate() + (day === 6 ? 2 : 1));
  return { date: d.toISOString().slice(0, 10), shift: day === 6 ? 'Sonnabend' : 'Sonntag' };
}

/** Festgeschrieben wird in der ersten Woche des Folgemonats, Januar bis Juni. */
export function commitDate(month: number): string {
  return `${YEAR}-${pad(month + 1)}-0${month % 3 === 0 ? 7 : 6}`;
}

/** Übermittelt wurden die Voranmeldungen Januar bis Juni wenige Tage vor der Fälligkeit. */
export const VAT_SUBMITTED = [
  `${YEAR}-03-06`, `${YEAR}-04-08`, `${YEAR}-05-07`, `${YEAR}-06-08`, `${YEAR}-07-08`, `${YEAR}-08-07`,
];

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
    basis: '§ 18 Abs. 1 und 2 UStG, § 46 UStDV',
    description:
      'Übermittlung und Zahlung bis zum 10. Tag nach Ablauf des Voranmeldungszeitraums, mit Dauerfristverlängerung einen Monat später. Erledigt mit dem Transferticket an der Voranmeldung.',
    channel: 'elster',
    evidenceSource: 'data',
  },
  sondervorauszahlung: {
    kind: 'sondervorauszahlung',
    title: 'Sondervorauszahlung',
    basis: '§ 47 Abs. 1, § 48 Abs. 2 UStDV',
    description:
      'Bei monatlicher Voranmeldung mit Dauerfristverlängerung: ein Elftel der Vorauszahlungen des Vorjahres, zu berechnen, anzumelden und zu entrichten bis zum 10. Februar. Erledigt mit der Buchung auf 3830.',
    channel: 'elster',
    evidenceSource: 'either',
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
    description: 'Übermittlung von Bilanz und GuV nach amtlichem Datensatz über einen E-Bilanz-Dienst.',
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

/** „2026-08-15“ als „15.08.2026“, wie das Backend Vorauszahlungen benennt. */
function germanDate(iso: string): string {
  return `${iso.slice(8, 10)}.${iso.slice(5, 7)}.${iso.slice(0, 4)}`;
}

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
    title: `${spec.title} ${o.periodLabel}`,
    description: spec.description,
    basis: spec.basis,
    channel: spec.channel,
    responsibility: spec.responsibility ?? 'agent_prepares',
    evidenceSource: spec.evidenceSource,
    skillRef: `buchfink/references/grundlagen/pflichten.md#${spec.kind}`,
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
      // Mit Dauerfristverlängerung: Festschreibung bis zum Ende des
      // übernächsten Monats, Voranmeldung bis zum 10. des übernächsten.
      const dueMonth = ((m + 1) % 12) + 1;
      const dueYear = m >= 11 ? y + 1 : y;
      out.push(
        obligation(y, KINDS.festschreibung, {
          periodKey: `${y}-${pad(m)}`,
          periodLabel: `${MONTH_NAMES[m - 1]} ${y}`,
          from: `${y}-${pad(m)}-01`,
          to,
          due: lastDay(dueYear, dueMonth),
          done: m <= 6 ? { date: commitDate(m), kind: 'commitment', reference: `${MONTH_NAMES[m - 1]} ${y}` } : undefined,
        }),
      );
      const vatDue = workday(`${dueYear}-${pad(dueMonth)}-10`);
      out.push(
        obligation(y, KINDS.ustva, {
          periodKey: `${y}-${pad(m)}`,
          periodLabel: `${MONTH_NAMES[m - 1]} ${y}`,
          from: `${y}-${pad(m)}-01`,
          to,
          due: VAT_DUE[m - 1] ?? vatDue.date,
          statutory: `${dueYear}-${pad(dueMonth)}-10`,
          shift: vatDue.shift,
          done: m <= 6 ? { date: VAT_SUBMITTED[m - 1], kind: 'transfer_ticket', reference: `et26m${pad(m)}x7k2m9p4` } : undefined,
        }),
      );
      const wageMonth = m === 12 ? 1 : m + 1;
      const wageYear = m === 12 ? y + 1 : y;
      const wageDue = workday(`${wageYear}-${pad(wageMonth)}-10`);
      out.push(
        obligation(y, KINDS['lohnsteuer-anmeldung'], {
          periodKey: `${y}-${pad(m)}`,
          periodLabel: `${MONTH_NAMES[m - 1]} ${y}`,
          from: `${y}-${pad(m)}-01`,
          to,
          due: wageDue.date,
          statutory: `${wageYear}-${pad(wageMonth)}-10`,
          shift: wageDue.shift,
          done: m <= 7 ? { date: `${wageYear}-${pad(wageMonth)}-08`, kind: 'transfer_ticket', reference: `LSt-${pad(m)}-88${m}31`, automatic: false } : undefined,
        }),
      );
    }
    out.push(
      obligation(y, KINDS.sondervorauszahlung, {
        periodKey: String(y),
        periodLabel: String(y),
        due: `${y}-02-10`,
        done: { date: `${y}-02-10`, kind: 'booking', reference: prepaymentEntry(`Sondervorauszahlung ${y}`) },
      }),
    );
    const kstDue = ['2026-03-10', '2026-06-10', '2026-09-10', '2026-12-10'];
    const gewstDue = ['2026-02-16', '2026-05-15', '2026-08-17', '2026-11-16'];
    for (let q = 1; q <= 4; q++) {
      out.push(
        obligation(y, KINDS['kst-vorauszahlung'], {
          periodKey: `${y}-Q${q}`,
          periodLabel: germanDate(kstDue[q - 1]),
          due: kstDue[q - 1],
          done: q <= 2
            ? { date: kstDue[q - 1], kind: 'booking', reference: prepaymentEntry(`Körperschaftsteuer-Vorauszahlung ${germanDate(kstDue[q - 1])}`) }
            : undefined,
        }),
      );
      // Die Vorauszahlung zum 15.08. ist noch nicht gezahlt.
      const statutory = `${y}-${gewstDue[q - 1].slice(5, 7)}-15`;
      out.push(
        obligation(y, KINDS['gewst-vorauszahlung'], {
          periodKey: `${y}-Q${q}`,
          periodLabel: germanDate(statutory),
          due: gewstDue[q - 1],
          statutory,
          shift: statutory !== gewstDue[q - 1] ? workday(statutory).shift : undefined,
          done: q <= 2
            ? { date: gewstDue[q - 1], kind: 'booking', reference: prepaymentEntry(`Gewerbesteuer-Vorauszahlung ${germanDate(statutory)}`) }
            : undefined,
        }),
      );
    }
  }
  const annual = (kind: string, due: string, done?: Options['done']) =>
    obligation(y, KINDS[kind], { periodKey: String(y), periodLabel: String(y), due, done });
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
  } else if (y === YEAR - 2) {
    out.push(
      annual('abschlussbuchungen', '2025-03-31', { date: '2025-03-10', kind: 'status' }),
      annual('aufstellung', '2025-06-30', { date: '2025-03-18', kind: 'status' }),
      annual('feststellung', '2025-11-30', { date: '2025-05-20', kind: 'status' }),
      annual('ergebnisverwendung', '2025-11-30', { date: '2025-05-20', kind: 'resolution' }),
      annual('offenlegung', '2025-12-31', { date: '2025-11-12', kind: 'status' }),
      annual('e-bilanz', '2025-07-31', { date: '2025-06-20', kind: 'transfer_ticket', reference: 'ET-2024-71204', automatic: false }),
      annual('kst-erklaerung', '2025-07-31', { date: '2025-06-26', kind: 'transfer_ticket', reference: 'TT-2024-KST-41108' }),
      annual('gewst-erklaerung', '2025-07-31', { date: '2025-06-26', kind: 'transfer_ticket', reference: 'TT-2024-GEW-41109' }),
      annual('ust-erklaerung', '2025-07-31', { date: '2025-06-26', kind: 'transfer_ticket', reference: 'TT-2024-UST-41110' }),
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
// Karteien der Abschlussbausteine

/**
 * Die Abgrenzungen aus dem Abschluss 2025. Sie stehen auf 1900 und werden mit
 * dem Abschluss 2026 aufgelöst; im laufenden Jahr zeigt die Kartei dieselben.
 */
export function accruals(year: number) {
  const formed = Math.min(year, YEAR - 1);
  return [
    {
      id: 1,
      fiscalYear: formed,
      kind: 'active',
      text: 'Kfz-Versicherung Pkw HH-NS 412, Dezember bis November',
      totalAmount: c(1440),
      deferredAmount: c(1320),
      startDate: `${formed}-12-01`,
      endDate: `${formed + 1}-11-30`,
      cutoffDate: `${formed}-12-31`,
      account: '6520',
      method: 'monthly',
      formationEntryId: 9411,
      releases: [],
      createdAt: `${formed + 1}-02-12T10:00:00Z`,
    },
    {
      id: 2,
      fiscalYear: formed,
      kind: 'active',
      text: 'Domains und Zertifikate, Oktober bis September',
      totalAmount: c(1200),
      deferredAmount: c(900),
      startDate: `${formed}-10-01`,
      endDate: `${formed + 1}-09-30`,
      cutoffDate: `${formed}-12-31`,
      account: '6495',
      method: 'monthly',
      formationEntryId: 9412,
      releases: [],
      createdAt: `${formed + 1}-02-12T10:02:00Z`,
    },
  ];
}

function mirrorRow(kind: string, label: string, account: string, opening: number, additions: number, used: number) {
  return { kind, label, account, opening, additions, used, released: 0, unwinding: 0, closing: opening + additions - used };
}

/**
 * Der Rückstellungsspiegel. 2025 bildet die Rückstellungen, die als
 * Saldenvortrag im Journal 2026 stehen; 2026 verbraucht die Steuern 2025 mit
 * den Abschlusszahlungen im Juli.
 */
export function provisionMirror(year: number) {
  const rows =
    year >= YEAR
      ? [
          mirrorRow('closing_costs', 'Abschluss- und Prüfungskosten', '3095', c(1450), 0, 0),
          mirrorRow('retention_costs', 'Aufbewahrung der Geschäftsunterlagen', '3096', c(1570), 0, 0),
          mirrorRow('tax_income', 'Körperschaftsteuer', '3040', c(2950.4), 0, c(2950.4)),
          mirrorRow('tax_trade', 'Gewerbesteuer', '3035', c(3906), 0, c(3906)),
        ]
      : [
          mirrorRow('closing_costs', 'Abschluss- und Prüfungskosten', '3095', c(1300), c(1450), c(1300)),
          mirrorRow('retention_costs', 'Aufbewahrung der Geschäftsunterlagen', '3096', c(1450), c(120), 0),
          mirrorRow('tax_income', 'Körperschaftsteuer', '3040', c(1650), c(2950.4), c(1650)),
          mirrorRow('tax_trade', 'Gewerbesteuer', '3035', c(2100), c(3906), c(2100)),
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

/** Die Rückstellungen aus dem Abschluss 2025, die am Stichtag noch bestehen. */
export function provisions(year: number) {
  const formed = Math.min(year, YEAR - 1);
  return [
    {
      id: 1, fiscalYear: formed, kind: 'closing_costs', text: `Erstellung und Offenlegung Jahresabschluss ${formed}`,
      settlementAmount: c(1450), expectedDate: `${formed + 1}-12-31`, discountedAmount: c(1450),
      balanceAccount: '3095', expenseAccount: '6827', reason: 'Gesetzliche Pflicht zur Aufstellung und Offenlegung',
      movements: [], createdAt: `${formed + 1}-02-14T09:00:00Z`,
    },
    {
      id: 2, fiscalYear: formed, kind: 'retention_costs', text: 'Aufbewahrung der Unterlagen, zehn Jahre',
      settlementAmount: c(1570), expectedDate: `${formed + 10}-12-31`, discountedAmount: c(1570),
      balanceAccount: '3096', expenseAccount: '6827', reason: 'Aufbewahrungspflicht für Bücher und Belege',
      movements: [], createdAt: `${formed + 1}-02-14T09:05:00Z`,
    },
  ];
}

export function vatSettlement(year: number) {
  return {
    fiscalYear: year, cutoff: `${year}-12-31`, rows: [],
    inputTax: c(22898.8), outputTax: c(92378), prepaid: c(56008), payable: c(13471.2), refund: 0,
    lines: [], bookingDate: `${year}-12-31`, explanation: '',
  };
}

export function taxProvision(year: number) {
  return {
    fiscalYear: year, cutoff: `${year}-12-31`,
    input: {
      profitBeforeTax: c(126544), nonDeductible: 0, tradeTaxRatePercent: 470,
      prepaidCorporate: c(3500), prepaidSolidarity: c(192.5), prepaidTrade: c(6000), date: `${year}-12-31`,
    },
    taxableIncome: c(126544), corporateTax: c(18981), solidarity: c(1043.95),
    tradeIncome: c(126500), tradeBase: c(4427.5), tradeTax: c(20809.25),
    incomeProvision: c(16332.45), tradeProvision: c(14809.25), incomeRefund: 0, tradeRefund: 0,
    ratesUsed: 'KSt 15 %, SolZ 5,5 %, Hebesatz 470 %', lines: [], explanation: '',
    warning: 'Die Rechnung ist eine Schätzung auf den Stand von heute. Gebucht wird erst zum Abschluss.',
  };
}

/** Die Ergebnisverwendung: 2024 ausgeschüttet, 2025 auf neue Rechnung vorgetragen. */
export function appropriation(year: number) {
  if (year === YEAR - 2) {
    return {
      year, decisionDate: '2025-05-20', text: 'Gesellschafterbeschluss vom 20.05.2025',
      netIncome: c(38400), legalReserve: 0, otherReserves: 0, distribution: c(38400),
      withholdingTax: c(9600), solidarityOnWithholding: c(528), carryForward: 0, createdAt: '2025-05-20T14:00:00Z',
    };
  }
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
    { section: 'board', label: 'Geschäftsführung', hint: '', basis: '§ 285 Nr. 10 HGB', text: 'Anna Bauer, Kauffrau, Hamburg' },
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

/** Geschenke und Bewirtungen hat die Beispielfirma im Jahr nicht gebucht. */
export function nonDeductibleReport(year: number) {
  return {
    fiscalYear: year,
    giftLimit: c(50),
    categories: [
      { key: 'gifts', label: 'Geschenke an Geschäftspartner', reference: '', note: '', deductibleAmount: 0, nonDeductibleAmount: 0, total: 0, count: 0 },
      { key: 'entertainment', label: 'Bewirtung aus geschäftlichem Anlass', reference: '', note: '', deductibleAmount: 0, nonDeductibleAmount: 0, total: 0, count: 0 },
    ],
    recipients: [],
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
