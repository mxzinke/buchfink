import React, { useCallback, useEffect, useState } from 'react';
import { ChevronDown, ChevronRight, Download } from 'lucide-react';
import type { Cents, FinancialStatement, StatementAccount, StatementLine } from '../../types';
import { useNavigation } from '../../navigation';
import { useWorkspace } from '../../workspace';
import { Api } from '../../services/api';
import { downloadBlob, downloadCSV } from '../../utils/download';
import { formatCents, formatDate } from '../../utils/formatters';
import {
  Button,
  Disclosure,
  EmptyState,
  Help,
  Notice,
  Section,
  SkeletonRows,
  Stat,
  StatRow,
  Table,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
  cn,
  toast,
} from '../../components/ui';
import { errorText } from './shared';

/** Einrückung nach der Ebene: Buchstabe, römische Ziffer, arabische Ziffer. */
const INDENT: Record<number, string> = { 1: 'pl-4', 2: 'pl-8', 3: 'pl-12' };

/** Posten ohne Betrag in beiden Jahren entfallen; das entscheidet das Backend. */
const shown = (lines: StatementLine[]) => lines.filter((line) => !line.omitted);

function bytesFromBase64(base64: string): ArrayBuffer {
  const binary = atob(base64);
  const buffer = new ArrayBuffer(binary.length);
  const bytes = new Uint8Array(buffer);
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
  return buffer;
}

/**
 * Bilanz und Gewinn- und Verlustrechnung des Geschäftsjahres. Jede Zeile
 * klappt auf ihre Konten auf, ein Konto öffnet sein Kontoblatt. So führt der
 * Weg von der Bilanzposition über Konto und Buchung bis zum Beleg.
 */
export const StatementsTab: React.FC = () => {
  const { navigate } = useNavigation();
  const { year } = useWorkspace();
  const [data, setData] = useState<FinancialStatement | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [exporting, setExporting] = useState<'pdf' | 'csv' | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      setData(await Api.getStatement(year));
      setError(null);
    } catch (e) {
      setData(null);
      setError(errorText(e));
    } finally {
      setLoading(false);
    }
  }, [year]);

  useEffect(() => {
    void load();
  }, [load]);

  async function exportAs(kind: 'pdf' | 'csv') {
    setExporting(kind);
    try {
      if (kind === 'pdf') {
        const base64 = await Api.exportStatementPDF(year);
        downloadBlob(
          `jahresabschluss-${year}.pdf`,
          new Blob([bytesFromBase64(base64)], { type: 'application/pdf' }),
        );
      } else {
        downloadCSV(`jahresabschluss-${year}.csv`, await Api.exportStatementCSV(year));
      }
    } catch (e) {
      toast.error(errorText(e));
    } finally {
      setExporting(null);
    }
  }

  const openAccount = (account: string) => navigate({ area: 'books', tab: 'accounts', id: account });

  if (loading) return <SkeletonRows rows={10} />;
  if (error) {
    return (
      <Notice
        tone="negative"
        text={error}
        action={
          <Button variant="secondary" size="sm" onClick={() => void load()}>
            Erneut aufstellen
          </Button>
        }
      />
    );
  }
  if (!data) return <EmptyState title={`Kein Abschluss für ${year}`} />;

  const stmt = data.statement;
  const findings = [...stmt.assignment.unassigned, ...stmt.assignment.wrongSign];

  return (
    <div>
      <div className="flex flex-wrap items-end justify-between gap-6">
        <StatRow className="flex-1 min-w-[24rem]">
          <Stat
            label="Bilanzsumme"
            value={formatCents(stmt.balanceSheetTotal)}
            context={stmt.hasPrior ? `Vorjahr ${formatCents(stmt.balanceSheetTotalPrior)}` : undefined}
          />
          <Stat
            label="Jahresergebnis"
            value={formatCents(stmt.netIncome)}
            context={
              stmt.hasPrior ? `Vorjahr ${formatCents(stmt.netIncomePrior)}` : 'bisher im Geschäftsjahr'
            }
            tone={stmt.netIncome >= 0 ? 'positive' : 'negative'}
          />
        </StatRow>
        <div className="flex gap-2">
          <Button
            variant="quiet"
            size="sm"
            icon={<Download className="w-3.5 h-3.5" strokeWidth={1.5} />}
            loading={exporting === 'csv'}
            disabled={exporting !== null}
            onClick={() => void exportAs('csv')}
          >
            CSV
          </Button>
          <Button
            variant="quiet"
            size="sm"
            icon={<Download className="w-3.5 h-3.5" strokeWidth={1.5} />}
            loading={exporting === 'pdf'}
            disabled={exporting !== null}
            onClick={() => void exportAs('pdf')}
          >
            PDF
          </Button>
        </div>
      </div>

      {findings.length > 0 && (
        <Notice
          className="mt-6"
          text={`${findings.length} Konten mit Saldo haben keine passende Bilanzposition.`}
        />
      )}

      <Section title={`Bilanz zum ${formatDate(data.header.closingDate)}`} context="Aktiva und Passiva">
        <div className="grid grid-cols-1 xl:grid-cols-2 gap-8">
          <LineTable
            caption="Aktiva"
            lines={shown(stmt.assets)}
            year={stmt.fiscalYear}
            priorYear={stmt.priorYear}
            hasPrior={stmt.hasPrior}
            total={{ label: 'Summe Aktiva', amount: stmt.totalAssets, prior: stmt.totalAssetsPrior }}
            onOpenAccount={openAccount}
          />
          <LineTable
            caption="Passiva"
            lines={shown(stmt.liabilities)}
            year={stmt.fiscalYear}
            priorYear={stmt.priorYear}
            hasPrior={stmt.hasPrior}
            total={{
              label: 'Summe Passiva',
              amount: stmt.totalLiabilities,
              prior: stmt.totalLiabilitiesPrior,
            }}
            onOpenAccount={openAccount}
          />
        </div>
      </Section>

      <Section
        title={`Gewinn- und Verlustrechnung ${stmt.fiscalYear}`}
        context="Erträge und Aufwendungen des Geschäftsjahres"
        helpSummary="Die Gewinn- und Verlustrechnung zeigt, wie Ihr Jahresergebnis entsteht."
        explain="Die Gliederung folgt dem Gesamtkostenverfahren nach § 275 Abs. 2 HGB."
      >
        {shown(stmt.income).length === 0 ? (
          <EmptyState
            title="Keine Erträge und Aufwendungen"
            description={`Im Geschäftsjahr ${stmt.fiscalYear} ist auf kein Erfolgskonto gebucht.`}
          />
        ) : (
          <LineTable
            caption="Staffel"
            lines={shown(stmt.income)}
            year={stmt.fiscalYear}
            priorYear={stmt.priorYear}
            hasPrior={stmt.hasPrior}
            onOpenAccount={openAccount}
          />
        )}
      </Section>

      {findings.length > 0 && (
        <Disclosure title="Konten ohne passende Position" context={`${findings.length} Konten`}>
          <AccountList accounts={findings} onOpenAccount={openAccount} />
        </Disclosure>
      )}
    </div>
  );
};

interface LineTableProps {
  caption: string;
  lines: StatementLine[];
  year: number;
  priorYear: number;
  hasPrior: boolean;
  total?: { label: string; amount: Cents; prior: Cents };
  onOpenAccount: (account: string) => void;
}

/** Eine Seite der Bilanz oder die Staffel der GuV; jede Zeile klappt auf ihre Konten auf. */
const LineTable: React.FC<LineTableProps> = ({
  caption,
  lines,
  year,
  priorYear,
  hasPrior,
  total,
  onOpenAccount,
}) => {
  const [open, setOpen] = useState<Record<string, boolean>>({});
  const toggle = (key: string) => setOpen((state) => ({ ...state, [key]: !state[key] }));

  return (
    <div>
      <div className="text-overline text-ink-subtle mb-2">{caption}</div>
      <Table density="kompakt">
        <Thead>
          <Tr>
            <Th>Posten</Th>
            <Th numeric className="w-36">{`${year}`}</Th>
            {hasPrior && (
              <Th numeric className="w-32">
                {`${priorYear}`}
              </Th>
            )}
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
                  onClick={expandable ? () => toggle(line.key) : undefined}
                  tabIndex={expandable ? 0 : undefined}
                  aria-expanded={expandable ? isOpen : undefined}
                  onKeyDown={
                    expandable
                      ? (event) => {
                          if (event.key !== 'Enter' && event.key !== ' ') return;
                          event.preventDefault();
                          toggle(line.key);
                        }
                      : undefined
                  }
                  className={expandable ? 'cursor-pointer' : undefined}
                >
                  <Td className={cn('whitespace-normal', INDENT[line.level] ?? 'pl-4')}>
                    <span className="flex items-start gap-1.5">
                      <span className="w-4 shrink-0 pt-0.5 text-ink-faint">
                        {expandable &&
                          (isOpen ? (
                            <ChevronDown className="w-3.5 h-3.5" strokeWidth={1.5} />
                          ) : (
                            <ChevronRight className="w-3.5 h-3.5" strokeWidth={1.5} />
                          ))}
                      </span>
                      <span className="min-w-0">
                        {line.ordinal && (
                          <span className="code-num text-caption text-ink-subtle mr-2">{line.ordinal}</span>
                        )}
                        <span className={line.level === 1 ? 'font-medium' : undefined}>{line.label}</span>
                        {line.note && (
                          <Help
                            summary="Hier erfahren Sie, wie sich dieser Betrag zusammensetzt."
                            label={`Erklärung zu ${line.label}`}
                          >
                            {line.note}
                          </Help>
                        )}
                      </span>
                    </span>
                  </Td>
                  <Td numeric>{formatCents(line.amount)}</Td>
                  {hasPrior && (
                    <Td numeric className="text-ink-subtle">
                      {formatCents(line.priorAmount)}
                    </Td>
                  )}
                </Tr>
                {isOpen &&
                  accounts.map((account) => (
                    <Tr key={`${line.key}-${account.number}`}>
                      <Td className="pl-14 whitespace-normal">
                        <span className="flex items-baseline gap-2">
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
                          <span className="text-caption text-ink-muted">{account.name}</span>
                        </span>
                      </Td>
                      <Td numeric className="text-caption text-ink-muted">
                        {formatCents(account.amount)}
                      </Td>
                      {hasPrior && (
                        <Td numeric className="text-caption text-ink-subtle">
                          {formatCents(account.priorAmount)}
                        </Td>
                      )}
                    </Tr>
                  ))}
              </React.Fragment>
            );
          })}
          {total && (
            <Tr variant="sum">
              <Td className="pl-4">{total.label}</Td>
              <Td numeric>{formatCents(total.amount)}</Td>
              {hasPrior && <Td numeric>{formatCents(total.prior)}</Td>}
            </Tr>
          )}
        </Tbody>
      </Table>
    </div>
  );
};

/** Konten ohne passende Position; ein Klick öffnet das Kontoblatt. */
const AccountList: React.FC<{
  accounts: StatementAccount[];
  onOpenAccount: (account: string) => void;
}> = ({ accounts, onOpenAccount }) => (
  <Table density="kompakt">
    <Thead>
      <Tr>
        <Th className="w-24">Konto</Th>
        <Th>Bezeichnung</Th>
        <Th numeric className="w-40">
          Saldo
        </Th>
      </Tr>
    </Thead>
    <Tbody>
      {accounts.map((account) => (
        <Tr
          key={`${account.number}-${account.positionId}`}
          className="cursor-pointer"
          onClick={() => onOpenAccount(account.number)}
        >
          <Td code>{account.number}</Td>
          <Td>{account.name}</Td>
          <Td numeric>{formatCents(account.amount)}</Td>
        </Tr>
      ))}
    </Tbody>
  </Table>
);
