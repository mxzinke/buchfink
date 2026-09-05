import React, { useMemo, useState } from 'react';
import { ChevronDown, ChevronRight } from 'lucide-react';
import type { NavigateFn } from './Sidebar';
import type {
  Cents,
  Deadline,
  FinancialStatement,
  MaturityRow,
  SizeAssessment,
  SizeClass,
  StatementAccount,
  StatementDepth,
  StatementLine,
} from '../types';
import { formatCents, formatDate } from '../utils/formatters';
import {
  Button,
  EmptyState,
  HelpPopover,
  HelpTooltip,
  Notice,
  Section,
  Select,
  Stat,
  StatRow,
  Table,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
  cn,
} from './ui';

/**
 * Bilanz und Gewinn- und Verlustrechnung nach den §§ 266 und 275 HGB.
 *
 * Die Ansicht steht als Baustein hier und nicht als eigene Seite: sie ist eine
 * von mehreren Sichten der Auswertungen und teilt sich deren Reiter mit der
 * Umsatzsteuer.
 *
 * Jede Zahl kommt fertig gegliedert aus `GetStatement`. Die frühere Auswertung
 * filterte im Frontend nach Kontenklasse und kam damit zu einer zweiten,
 * abweichenden Wahrheit — Gliederung, Vorzeichen und Vorjahresspalte gehören
 * zur Rechnungslegung und nicht zur Darstellung.
 */

/** Die Sichten auf den Abschluss, als Reiter der Auswertungen. */
export type StatementTab = 'bilanz' | 'guv' | 'angaben' | 'klasse';

/** „Auto" ist die Tiefe, die die Größenklasse vorgibt (§ 266 Abs. 1 HGB). */
export type DepthChoice = 'auto' | StatementDepth;

const DEPTH_ITEMS: { value: DepthChoice; label: string }[] = [
  { value: 'auto', label: 'Tiefe der Größenklasse' },
  { value: 'full', label: 'Vollgliederung (§ 266 Abs. 2 und 3 HGB)' },
  { value: 'short', label: 'Verkürzt (§ 266 Abs. 1 Satz 3 HGB)' },
  { value: 'letters', label: 'Buchstaben (§ 266 Abs. 1 Satz 4 HGB)' },
];

const CLASS_LABELS: Record<string, string> = {
  micro: 'Kleinstkapitalgesellschaft',
  small: 'Kleine Kapitalgesellschaft',
  medium: 'Mittelgroße Kapitalgesellschaft',
  large: 'Große Kapitalgesellschaft',
};

const DEPTH_LABELS: Record<StatementDepth, string> = {
  full: 'Vollgliederung',
  short: 'Verkürzte Gliederung',
  letters: 'Buchstabengliederung',
};

/** Einrückung nach der Ebene: Buchstabe, römische Ziffer, arabische Ziffer. */
const INDENT: Record<number, string> = { 1: 'pl-4', 2: 'pl-8', 3: 'pl-12' };

/**
 * Die Posten, die der Abschluss ausweist. Welche nach § 265 Abs. 8 HGB
 * entfallen, entscheidet der Aufbau im Backend und nicht die Ansicht: PDF und
 * CSV folgen demselben Merker, und so zeigen Bildschirm und Datei dieselben
 * Zeilen.
 */
function shown(lines: StatementLine[]): StatementLine[] {
  return lines.filter((line) => !line.omitted);
}

export interface StatementViewProps {
  /** Der fertige Abschluss aus `GetStatement`. */
  data: FinancialStatement;
  /** Welche Sicht dieser Reiter zeigt. */
  view: StatementTab;
  depth: DepthChoice;
  onDepthChange: (depth: DepthChoice) => void;
  /** Weg von der Gliederungszeile über das Konto ins Kontoblatt (GOB-02). */
  onNavigate?: NavigateFn;
}

export const StatementView: React.FC<StatementViewProps> = ({
  data,
  view,
  depth,
  onDepthChange,
  onNavigate,
}) => {
  const stmt = data.statement;
  const header = data.header;

  const assets = useMemo(() => shown(stmt.assets), [stmt]);
  const liabilities = useMemo(() => shown(stmt.liabilities), [stmt]);
  const income = useMemo(() => shown(stmt.income), [stmt]);
  const statistical = useMemo(() => shown(stmt.statistical), [stmt]);

  const findings = [...stmt.assignment.unassigned, ...stmt.assignment.wrongSign];
  const priorYear = stmt.priorYear;
  const hasPrior = stmt.hasPrior;
  const openAccount = onNavigate ? (account: string) => onNavigate('accounts', { account }) : undefined;

  const depthSelect = (
    <Select<DepthChoice> items={DEPTH_ITEMS} value={depth} onValueChange={onDepthChange} className="w-72" />
  );

  if (view === 'angaben') {
    return (
      <Section
        title="Restlaufzeiten"
        context={`Stichtag ${formatDate(data.maturities.closingDate)} · ${data.maturities.reference}`}
        divider={false}
      >
        <MaturityView rows={data.maturities.rows} />
      </Section>
    );
  }

  if (view === 'klasse') {
    return <SizeClassView sizeClass={data.sizeClass} deadlines={data.deadlines} />;
  }

  return (
    <>
      <HeaderFacts
        statement={data}
        onOpenSettings={onNavigate && (() => onNavigate('settings'))}
      />

      {findings.length > 0 && (
        <Notice
          className="mt-6"
          text={`${findings.length} Konten mit Saldo stehen unter „Nicht zugeordnet" oder widersprechen der Richtung ihrer Position.`}
        />
      )}

      {view === 'bilanz' ? (
        <>
          <Section
            title={`Bilanz zum ${formatDate(header.closingDate)}`}
            context={`${DEPTH_LABELS[stmt.depth]} · Kontoform nach § 266 Abs. 1 Satz 1 HGB`}
            action={depthSelect}
          >
            {assets.length === 0 && liabilities.length === 0 ? (
              <EmptyState
                title="Keine Bestände im Geschäftsjahr"
                description={`Im Geschäftsjahr ${stmt.fiscalYear} ist auf kein Bestandskonto gebucht worden.`}
              />
            ) : (
              <div className="grid grid-cols-1 xl:grid-cols-2 gap-8">
                <LineTable
                  caption="Aktiva"
                  lines={assets}
                  year={stmt.fiscalYear}
                  priorYear={priorYear}
                  hasPrior={hasPrior}
                  totalLabel="Summe Aktiva"
                  total={stmt.totalAssets}
                  totalPrior={stmt.totalAssetsPrior}
                  onOpenAccount={openAccount}
                />
                <LineTable
                  caption="Passiva"
                  lines={liabilities}
                  year={stmt.fiscalYear}
                  priorYear={priorYear}
                  hasPrior={hasPrior}
                  totalLabel="Summe Passiva"
                  total={stmt.totalLiabilities}
                  totalPrior={stmt.totalLiabilitiesPrior}
                  onOpenAccount={openAccount}
                />
              </div>
            )}
          </Section>

          {statistical.length > 0 && (
            <Section
              title="Statistische Konten"
              context="Nicht Bestandteil der Bilanz — ein Saldo hier bleibt zu klären"
            >
              <LineTable
                caption="Klasse 9"
                lines={statistical}
                year={stmt.fiscalYear}
                priorYear={priorYear}
                hasPrior={hasPrior}
                onOpenAccount={openAccount}
              />
            </Section>
          )}

          {findings.length > 0 && (
            <Section
              title="Konten ohne tragfähige Zuordnung"
              context={'Vor der E-Bilanz zu klären, hier stehen sie unter „Nicht zugeordnet"'}
            >
              <FindingTable accounts={findings} onOpenAccount={openAccount} />
            </Section>
          )}
        </>
      ) : (
        <Section
          title={`Gewinn- und Verlustrechnung ${stmt.fiscalYear}`}
          context="Staffelform, Gesamtkostenverfahren nach § 275 Abs. 2 HGB"
          action={depthSelect}
        >
          {income.length === 0 ? (
            <EmptyState
              title="Keine Erträge und Aufwendungen"
              description={`Im Geschäftsjahr ${stmt.fiscalYear} ist auf kein Erfolgskonto gebucht worden.`}
            />
          ) : (
            <LineTable
              caption="Staffel"
              lines={income}
              year={stmt.fiscalYear}
              priorYear={priorYear}
              hasPrior={hasPrior}
              onOpenAccount={openAccount}
            />
          )}
        </Section>
      )}
    </>
  );
};

// -------------------------------------------------------------------------

/**
 * Der Kopf des Abschlusses: die Pflichtangaben des § 264 Abs. 1a HGB und die
 * drei Zahlen, an denen alles Weitere hängt.
 */
const HeaderFacts: React.FC<{
  statement: FinancialStatement;
  onOpenSettings?: () => void;
}> = ({ statement, onOpenSettings }) => {
  const { header, statement: stmt, sizeClass } = statement;
  const register = [header.registerCourt, header.registerNumber].filter(Boolean).join(' ');

  return (
    <>
      <dl className="grid grid-cols-2 md:grid-cols-4 gap-x-8 gap-y-4">
        <Fact label="Firma" value={header.companyName} context={header.legalForm} />
        <Fact label="Sitz" value={header.seat} />
        <Fact label="Registergericht und -nummer" value={register} />
        <Fact
          label="Geschäftsjahr"
          value={`${formatDate(header.startDate)} – ${formatDate(header.closingDate)}`}
          context={header.isShortYear ? 'Rumpfgeschäftsjahr' : undefined}
        />
      </dl>

      {header.missing.length > 0 && (
        <Notice
          className="mt-6"
          text={`Pflichtangaben nach ${header.reference} fehlen: ${header.missing.join(', ')}.`}
          action={
            onOpenSettings && (
              <Button variant="secondary" size="sm" onClick={onOpenSettings}>
                Ergänzen
              </Button>
            )
          }
        />
      )}

      <div className="mt-6">
        <StatRow>
          <Stat
            label={
              <>
                Bilanzsumme
                <HelpTooltip
                  label="Erklärung zur Bilanzsumme"
                  content="Summe der Posten A bis E der Aktivseite ohne die nicht eingeforderten ausstehenden Einlagen (§ 267 Abs. 4a HGB)."
                />
              </>
            }
            value={formatCents(stmt.balanceSheetTotal)}
            context={stmt.hasPrior ? `Vorjahr ${formatCents(stmt.balanceSheetTotalPrior)}` : undefined}
          />
          <Stat
            label="Jahresergebnis"
            value={formatCents(stmt.netIncome)}
            context={stmt.hasPrior ? `Vorjahr ${formatCents(stmt.netIncomePrior)}` : 'Nummer 17 der Staffel'}
            tone={stmt.netIncome >= 0 ? 'positive' : 'negative'}
          />
          <Stat
            label={
              <>
                Größenklasse
                <HelpPopover label="Erklärung zur Größenklasse">{sizeClass.reason}</HelpPopover>
              </>
            }
            value={CLASS_LABELS[sizeClass.class] ?? sizeClass.class}
            context={sizeClass.isFirstYear ? 'Erster Abschlussstichtag (§ 267 Abs. 4 Satz 2 HGB)' : undefined}
          />
        </StatRow>
      </div>
    </>
  );
};

const Fact: React.FC<{ label: string; value: string; context?: string }> = ({
  label,
  value,
  context,
}) => (
  <div className="min-w-0">
    <dt className="text-caption text-ink-subtle">{label}</dt>
    <dd className={cn('text-body mt-0.5 truncate', value ? 'text-ink' : 'text-ink-faint')}>
      {value || 'fehlt'}
    </dd>
    {context && <dd className="text-caption text-ink-subtle truncate">{context}</dd>}
  </div>
);

// -------------------------------------------------------------------------

interface LineTableProps {
  caption: string;
  lines: StatementLine[];
  year: number;
  priorYear: number;
  hasPrior: boolean;
  totalLabel?: string;
  total?: Cents;
  totalPrior?: Cents;
  onOpenAccount?: (account: string) => void;
}

/**
 * Eine Seite der Bilanz oder die Staffel der GuV.
 *
 * Jede Zeile lässt sich auf die Konten aufklappen, die in ihr stehen — auch in
 * der verkürzten Gliederung, denn dort trägt die Buchstabenzeile die Konten der
 * Unterposten, die sie ersetzt (GOB-02).
 */
const LineTable: React.FC<LineTableProps> = ({
  caption,
  lines,
  year,
  priorYear,
  hasPrior,
  totalLabel,
  total,
  totalPrior,
  onOpenAccount,
}) => {
  const [open, setOpen] = useState<Record<string, boolean>>({});

  return (
    <div>
      <div className="text-overline text-ink-subtle mb-2">{caption}</div>
      <Table density="kompakt">
        <Thead>
          <Tr>
            <Th>Posten</Th>
            <Th numeric className="w-40">{`Geschäftsjahr ${year}`}</Th>
            <Th numeric className="w-36">
              {hasPrior ? `${priorYear}` : 'Vorjahr'}
            </Th>
          </Tr>
        </Thead>
        <Tbody>
          {lines.map((line) => {
            const accounts = line.accounts ?? [];
            const expandable = accounts.length > 0;
            const isOpen = Boolean(open[line.key]);

            return (
              <React.Fragment key={line.key}>
                <Tr
                  variant={line.isSubtotal ? 'sum' : 'default'}
                  onClick={expandable ? () => setOpen((s) => ({ ...s, [line.key]: !isOpen })) : undefined}
                  className={expandable ? 'cursor-pointer' : undefined}
                >
                  <Td className={cn('whitespace-normal', INDENT[line.level] ?? 'pl-4')}>
                    <span className="flex items-start gap-1.5">
                      <span className="w-4 shrink-0 pt-0.5 text-ink-faint">
                        {expandable ? (
                          isOpen ? (
                            <ChevronDown className="w-3.5 h-3.5" strokeWidth={1.5} />
                          ) : (
                            <ChevronRight className="w-3.5 h-3.5" strokeWidth={1.5} />
                          )
                        ) : null}
                      </span>
                      <span className="min-w-0">
                        {line.ordinal && (
                          <span className="code-num text-caption text-ink-subtle mr-2">{line.ordinal}</span>
                        )}
                        <span className={line.level === 1 ? 'font-medium' : undefined}>{line.label}</span>
                        {line.note && (
                          <HelpTooltip label={`Erklärung zu ${line.label}`} content={line.note} />
                        )}
                      </span>
                    </span>
                  </Td>
                  <Td numeric>{formatCents(line.amount)}</Td>
                  <Td numeric className="text-ink-subtle">
                    {hasPrior ? formatCents(line.priorAmount) : '—'}
                  </Td>
                </Tr>

                {isOpen &&
                  accounts.map((account) => (
                    <AccountRow
                      key={`${line.key}-${account.number}`}
                      account={account}
                      hasPrior={hasPrior}
                      onOpenAccount={onOpenAccount}
                    />
                  ))}
              </React.Fragment>
            );
          })}

          {totalLabel !== undefined && (
            <Tr variant="sum">
              <Td className="pl-4">{totalLabel}</Td>
              <Td numeric>{formatCents(total ?? 0)}</Td>
              <Td numeric>{hasPrior ? formatCents(totalPrior ?? 0) : '—'}</Td>
            </Tr>
          )}
        </Tbody>
      </Table>
    </div>
  );
};

/** Ein Konto unter der Position — von hier führt der Weg ins Kontoblatt. */
const AccountRow: React.FC<{
  account: StatementAccount;
  hasPrior: boolean;
  onOpenAccount?: (account: string) => void;
}> = ({ account, hasPrior, onOpenAccount }) => (
  <Tr>
    <Td className="pl-14 whitespace-normal">
      <span className="flex items-baseline gap-2">
        {onOpenAccount ? (
          <button
            type="button"
            onClick={(event) => {
              event.stopPropagation();
              onOpenAccount(account.number);
            }}
            className="code-num text-caption text-accent-text hover:underline"
          >
            {account.number}
          </button>
        ) : (
          <span className="code-num text-caption text-ink-muted">{account.number}</span>
        )}
        <span className="text-caption text-ink-muted">{account.name}</span>
        {account.note && (
          <HelpTooltip label={`Erklärung zu Konto ${account.number}`} content={account.note} />
        )}
      </span>
    </Td>
    <Td numeric className="text-caption text-ink-muted">
      {formatCents(account.amount)}
    </Td>
    <Td numeric className="text-caption text-ink-faint">
      {hasPrior ? formatCents(account.priorAmount) : '—'}
    </Td>
  </Tr>
);

/** Konten ohne Position und Konten, deren Saldo der Position widerspricht. */
const FindingTable: React.FC<{
  accounts: StatementAccount[];
  onOpenAccount?: (account: string) => void;
}> = ({ accounts, onOpenAccount }) => (
  <Table density="kompakt">
    <Thead>
      <Tr>
        <Th className="w-28">Konto</Th>
        <Th>Bezeichnung</Th>
        <Th>SKR04-Position</Th>
        <Th numeric className="w-40">
          Saldo
        </Th>
      </Tr>
    </Thead>
    <Tbody>
      {accounts.map((account) => (
        <Tr
          key={`${account.number}-${account.positionId}`}
          onClick={onOpenAccount ? () => onOpenAccount(account.number) : undefined}
          className={onOpenAccount ? 'cursor-pointer' : undefined}
        >
          <Td code>{account.number}</Td>
          <Td>{account.name}</Td>
          <Td className="text-ink-muted whitespace-normal">
            {account.position || account.positionId || '—'}
          </Td>
          <Td numeric>{formatCents(account.amount)}</Td>
        </Tr>
      ))}
    </Tbody>
  </Table>
);

// -------------------------------------------------------------------------

const MaturityView: React.FC<{ rows: MaturityRow[] }> = ({ rows }) => (
  <Table density="kompakt">
    <Thead>
      <Tr>
        <Th>Posten</Th>
        <Th numeric className="w-36">
          Gesamt
        </Th>
        <Th numeric className="w-36">
          bis 1 Jahr
        </Th>
        <Th numeric className="w-36">
          über 1 Jahr
        </Th>
        <Th numeric className="w-36">
          über 5 Jahre
        </Th>
        <Th numeric className="w-36">
          ohne Fälligkeit
        </Th>
      </Tr>
    </Thead>
    <Tbody>
      {rows.map((row) => (
        <Tr key={row.key}>
          <Td className="whitespace-normal">
            {row.label}
            {row.note && <HelpTooltip label={`Erklärung zu ${row.label}`} content={row.note} />}
          </Td>
          <Td numeric>{formatCents(row.total)}</Td>
          <Td numeric>{formatCents(row.upToOneYear)}</Td>
          <Td numeric>{formatCents(row.overOneYear)}</Td>
          <Td numeric>{formatCents(row.overFiveYears)}</Td>
          <Td numeric className="text-ink-subtle">
            {formatCents(row.undated)}
          </Td>
        </Tr>
      ))}
    </Tbody>
  </Table>
);

// -------------------------------------------------------------------------

const YES_NO = (value: boolean) => (value ? 'Ja' : 'Nein');

/** Merkmale, Schwellen, Folgen und Fristen — je mit der Norm daneben. */
const SizeClassView: React.FC<{ sizeClass: SizeClass; deadlines: Deadline[] }> = ({
  sizeClass,
  deadlines,
}) => {
  const thresholds = sizeClass.current.thresholds;
  const o = sizeClass.obligations;
  const history: SizeAssessment[] = sizeClass.history ?? [];

  return (
    <>
      <Section
        title={CLASS_LABELS[sizeClass.class] ?? sizeClass.class}
        context={`Stichtag ${formatDate(sizeClass.closingDate)} · ${sizeClass.reason}`}
        divider={false}
        action={
          <HelpPopover label="Erklärung zur Zweijahresregel">
            Die Rechtsfolgen treten nach § 267 Abs. 4 Satz 1 HGB erst ein, wenn zwei aufeinander
            folgende Abschlussstichtage dieselbe Klasse ergeben. Bei einer Neugründung gilt schon
            der erste Stichtag (Satz 2).
          </HelpPopover>
        }
      >
        <Table density="kompakt">
          <Thead>
            <Tr>
              <Th>Merkmal</Th>
              <Th numeric className="w-40">
                Geschäftsjahr
              </Th>
              <Th numeric className="w-36">
                Kleinst
              </Th>
              <Th numeric className="w-36">
                Klein
              </Th>
              <Th numeric className="w-40">
                Mittelgroß
              </Th>
            </Tr>
          </Thead>
          <Tbody>
            <Tr>
              <Td>Bilanzsumme</Td>
              <Td numeric>{formatCents(sizeClass.criteria.balanceSheetTotal)}</Td>
              <Td numeric className="text-ink-subtle">
                {formatCents(thresholds.micro.balanceSheetTotal)}
              </Td>
              <Td numeric className="text-ink-subtle">
                {formatCents(thresholds.small.balanceSheetTotal)}
              </Td>
              <Td numeric className="text-ink-subtle">
                {formatCents(thresholds.medium.balanceSheetTotal)}
              </Td>
            </Tr>
            <Tr>
              <Td>Umsatzerlöse</Td>
              <Td numeric>{formatCents(sizeClass.criteria.revenue)}</Td>
              <Td numeric className="text-ink-subtle">
                {formatCents(thresholds.micro.revenue)}
              </Td>
              <Td numeric className="text-ink-subtle">
                {formatCents(thresholds.small.revenue)}
              </Td>
              <Td numeric className="text-ink-subtle">
                {formatCents(thresholds.medium.revenue)}
              </Td>
            </Tr>
            <Tr>
              <Td>
                Arbeitnehmer im Jahresdurchschnitt
                <HelpTooltip
                  label="Erklärung zur Arbeitnehmerzahl"
                  content="Die Zahl lässt sich aus der Buchführung nicht ableiten; sie wird im Jahresabschluss erfasst."
                />
              </Td>
              <Td numeric>{sizeClass.criteria.employees}</Td>
              <Td numeric className="text-ink-subtle">
                {thresholds.micro.employees}
              </Td>
              <Td numeric className="text-ink-subtle">
                {thresholds.small.employees}
              </Td>
              <Td numeric className="text-ink-subtle">
                {thresholds.medium.employees}
              </Td>
            </Tr>
          </Tbody>
        </Table>
      </Section>

      {history.length > 1 && (
        <Section title="Beurteilte Stichtage" context="Grundlage der Zweijahresregel">
          <Table density="kompakt">
            <Thead>
              <Tr>
                <Th className="w-28">Stichtag</Th>
                <Th>Klasse</Th>
                <Th>Erfüllte Merkmale</Th>
              </Tr>
            </Thead>
            <Tbody>
              {history.map((assessment) => (
                <Tr key={assessment.year}>
                  <Td className="num">{formatDate(assessment.closingDate)}</Td>
                  <Td>{CLASS_LABELS[assessment.class] ?? assessment.class}</Td>
                  <Td className="text-ink-muted whitespace-normal">
                    {assessment.met.length > 0 ? assessment.met.join(', ') : '—'}
                  </Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        </Section>
      )}

      <Section title="Folgen der Größenklasse">
        <Table density="kompakt">
          <Thead>
            <Tr>
              <Th>Pflicht</Th>
              <Th className="w-80">Umfang</Th>
              <Th className="w-64">Norm</Th>
            </Tr>
          </Thead>
          <Tbody>
            <Tr>
              <Td>Gliederungstiefe</Td>
              <Td className="whitespace-normal">{DEPTH_LABELS[o.depth]}</Td>
              <Td className="text-ink-muted whitespace-normal">{o.depthReference}</Td>
            </Tr>
            <Tr>
              <Td>Anhang</Td>
              <Td>{YES_NO(o.notesRequired)}</Td>
              <Td className="text-ink-muted whitespace-normal">{o.notesReference}</Td>
            </Tr>
            <Tr>
              <Td>Lagebericht</Td>
              <Td>{YES_NO(o.managementReport)}</Td>
              <Td className="text-ink-muted whitespace-normal">
                {o.managementReportReference || '§ 264 Abs. 1 Satz 4 HGB'}
              </Td>
            </Tr>
            <Tr>
              <Td>Prüfung</Td>
              <Td>{YES_NO(o.auditRequired)}</Td>
              <Td className="text-ink-muted whitespace-normal">
                {o.auditReference || '§ 316 Abs. 1 Satz 1 HGB'}
              </Td>
            </Tr>
            <Tr>
              <Td>Aufstellungsfrist</Td>
              <Td>{`${o.preparationMonths} Monate`}</Td>
              <Td className="text-ink-muted whitespace-normal">{o.preparationReference}</Td>
            </Tr>
            <Tr>
              <Td>Offenlegungsfrist</Td>
              <Td>{`${o.disclosureMonths} Monate`}</Td>
              <Td className="text-ink-muted whitespace-normal">{o.disclosureReference}</Td>
            </Tr>
            <Tr>
              <Td>Offenlegungsumfang</Td>
              <Td className="whitespace-normal">{o.disclosureScope}</Td>
              <Td className="text-ink-muted whitespace-normal">{o.disclosureScopeReference}</Td>
            </Tr>
          </Tbody>
        </Table>
      </Section>

      <Section title="Termine" context="Aufstellung und Offenlegung des Abschlusses">
        {deadlines.length === 0 ? (
          <EmptyState
            title="Keine Termine"
            description="Ohne Abschlussstichtag lassen sich die Fristen nicht berechnen."
          />
        ) : (
          <Table density="kompakt">
            <Thead>
              <Tr>
                <Th>Termin</Th>
                <Th className="w-32">Fällig</Th>
                <Th className="w-56">Zeitraum</Th>
                <Th className="w-56">Norm</Th>
                <Th className="w-40">Stand</Th>
              </Tr>
            </Thead>
            <Tbody>
              {deadlines.map((deadline) => (
                <Tr key={deadline.key}>
                  <Td className="whitespace-normal">
                    {deadline.title}
                    {deadline.description && (
                      <HelpTooltip
                        label={`Erklärung zu ${deadline.title}`}
                        content={deadline.description}
                      />
                    )}
                  </Td>
                  <Td className="num">{formatDate(deadline.dueDate)}</Td>
                  <Td className="text-ink-muted whitespace-normal">{deadline.period}</Td>
                  <Td className="text-ink-muted whitespace-normal">{deadline.reference}</Td>
                  <Td className={deadline.isDone ? 'text-positive-text' : 'text-ink-subtle'}>
                    {deadline.isDone ? `Erledigt am ${formatDate(deadline.doneOn ?? '')}` : 'Offen'}
                  </Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
      </Section>
    </>
  );
};
