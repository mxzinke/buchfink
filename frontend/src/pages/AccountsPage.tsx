import React, { useEffect, useMemo, useState } from 'react';
import { ArrowLeft, ChevronDown, ChevronRight } from 'lucide-react';
import type { Account, AccountLedger, AccountType, SuSaOverview } from '../types';
import type { NavigateFn } from '../components/Sidebar';
import { Api } from '../services/api';
import { formatCents, formatDate } from '../utils/formatters';
import {
  Button,
  EmptyState,
  HelpPopover,
  HelpTooltip,
  PageHeader,
  SearchInput,
  Section,
  SkeletonRows,
  Stat,
  StatRow,
  TabPanel,
  Table,
  Tabs,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
  toast,
} from '../components/ui';

/**
 * Kontenaufstellung.
 *
 * Von 1.855 Katalogeinträgen bebucht ein Unternehmen ein paar Dutzend. Die
 * Seite zeigt deshalb zuerst die bebuchten Konten und macht den Kontenrahmen
 * erst auf Wunsch auf. Das Grundprinzip — Aktiv- und Aufwandskonten tragen
 * einen Sollsaldo, Passiv-, Kapital- und Ertragskonten einen Habensaldo —
 * steht als Erklärung an der Übersicht, nicht als Legende auf jeder Zeile.
 */

type Tab = 'konten' | 'susa';

const CLASS_NAMES: Record<number, string> = {
  0: 'Anlagevermögen',
  1: 'Umlaufvermögen',
  2: 'Eigen- & Fremdkapital',
  3: 'Fremdkapital',
  4: 'Betriebliche Erträge',
  5: 'Material & Fremdleistungen',
  6: 'Personal, Abschreibungen & sonstige Aufwendungen',
  7: 'Finanzen & Steuern',
  8: 'Von DATEV freigehalten',
  9: 'Vorträge & statistische Konten',
};

/** Die Kontoart ist eine Einordnung, kein Zustand. Deshalb Wort statt Farbe (§3.4). */
const TYPE_LABELS: Record<AccountType, string> = {
  asset: 'Aktiva',
  liability: 'Passiva',
  equity: 'Eigenkapital',
  revenue: 'Ertrag',
  expense: 'Aufwand',
  statistical: 'Statistisch',
};

/** Auf welcher Seite ein Konto seinen normalen Saldo trägt. */
function naturalSide(type: AccountType): 'Soll' | 'Haben' {
  return type === 'liability' || type === 'equity' || type === 'revenue' ? 'Haben' : 'Soll';
}

/** Beschreibt den Saldo eines Kontos ausgeschrieben. */
function balanceHint(account: Account): string {
  const natural = naturalSide(account.type);
  if (account.balance === 0) return 'ausgeglichen';
  if (account.balance > 0) return `${natural}saldo`;
  return `${natural === 'Soll' ? 'Haben' : 'Soll'}saldo, ungewöhnlich`;
}

const BALANCE_HELP =
  'Aktiv- und Aufwandskonten tragen ihren Saldo im Soll, Passiv-, Kapital- und Ertragskonten im ' +
  'Haben. Steht der Saldo auf der anderen Seite, weist die Zeile darauf hin.';

export interface AccountsPageProps {
  /**
   * Kontonummer aus dem Navigationsziel. Wer aus der Bilanz auf ein Konto
   * klickt, will das Kontoblatt sehen und nicht die Kontenliste (GOB-02).
   */
  initialAccount?: string;
  onNavigate?: NavigateFn;
}

export const AccountsPage: React.FC<AccountsPageProps> = ({ initialAccount, onNavigate }) => {
  const [tab, setTab] = useState<Tab>('konten');
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [susa, setSusa] = useState<SuSaOverview | null>(null);
  const [ledger, setLedger] = useState<AccountLedger | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadingLedger, setLoadingLedger] = useState(false);

  const [search, setSearch] = useState('');
  const [showCatalog, setShowCatalog] = useState(false);
  const [openClasses, setOpenClasses] = useState<Record<number, boolean>>({});

  useEffect(() => {
    void load();
  }, []);

  // Das Kontoblatt folgt dem Navigationsziel in beide Richtungen: mit
  // Kontonummer öffnet es sich, ohne schließt es. Sonst bliebe ein Kontoblatt
  // stehen, das über die Navigation gar nicht angesteuert wurde.
  useEffect(() => {
    if (initialAccount) void openLedger(initialAccount);
    else setLedger(null);
  }, [initialAccount]);

  async function load() {
    setLoading(true);
    try {
      const [accountList, overview] = await Promise.all([Api.getAccounts(), Api.getSuSaOverview()]);
      setAccounts(accountList);
      setSusa(overview);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }

  async function openLedger(accountNumber: string) {
    setLoadingLedger(true);
    try {
      setLedger(await Api.getAccountLedger(accountNumber));
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setLoadingLedger(false);
    }
  }

  /** Bebuchte Konten — das, womit tatsächlich gearbeitet wird. */
  const inUse = useMemo(
    () => accounts.filter((a) => a.bookingsCount > 0).sort((a, b) => a.number.localeCompare(b.number)),
    [accounts],
  );

  /** Bebuchbarer Katalog: ohne reservierte Einträge und ohne die freigehaltene Klasse 8. */
  const catalog = useMemo(
    () => accounts.filter((a) => !a.isReserved && a.kontenklasse !== 8),
    [accounts],
  );

  const searchResults = useMemo(() => {
    const query = search.trim().toLowerCase();
    if (!query) return [];
    return catalog
      .filter((a) => a.number.toLowerCase().includes(query) || a.name.toLowerCase().includes(query))
      .slice(0, 60);
  }, [catalog, search]);

  const catalogByClass = useMemo(() => {
    const groups = new Map<number, Account[]>();
    for (const account of catalog) {
      const list = groups.get(account.kontenklasse) ?? [];
      list.push(account);
      groups.set(account.kontenklasse, list);
    }
    return [...groups.entries()]
      .sort(([a], [b]) => a - b)
      .map(([kontenklasse, list]) => ({
        kontenklasse,
        accounts: list.sort((a, b) => a.number.localeCompare(b.number)),
      }));
  }, [catalog]);

  if (ledger) {
    return (
      <LedgerView
        ledger={ledger}
        loading={loadingLedger}
        onBack={() => setLedger(null)}
        onOpenEntry={
          onNavigate && ((entryNumber: string) => onNavigate('journal', { entryNumber }))
        }
      />
    );
  }

  return (
    <div className="max-w-[1200px] mx-auto px-8 py-8">
      <PageHeader title="Konten" context="Kontenrahmen SKR04 der DATEV, Fassung 2026" />

      <Tabs
        items={[
          { value: 'konten' as Tab, label: 'Konten' },
          { value: 'susa' as Tab, label: 'Summen & Salden' },
        ]}
        value={tab}
        onValueChange={setTab}
        className="mt-6"
      >
        <TabPanel value="konten">
          {loading ? (
            <SkeletonRows rows={8} />
          ) : (
            <>
              <Section
                title="Bebuchte Konten"
                context={`${inUse.length} von ${catalog.length} bebuchbaren Konten`}
                divider={false}
                action={
                  <HelpPopover label="Erklärung zu den bebuchten Konten">
                    Der SKR04 enthält über 1.600 nutzbare Konten, ein Unternehmen bebucht davon
                    typischerweise ein paar Dutzend. Neue Konten entstehen von selbst, sobald eine
                    Buchungsgruppe sie zum ersten Mal verwendet. {BALANCE_HELP}
                  </HelpPopover>
                }
              >
                {inUse.length === 0 ? (
                  <EmptyState
                    title="Noch keine Buchungen vorhanden"
                    description="Sobald der erste Beleg erfasst ist, erscheinen hier die Konten, die er berührt."
                  />
                ) : (
                  <AccountTable accounts={inUse} onSelect={openLedger} showTurnover />
                )}
              </Section>

              <Section
                title="Kontenrahmen"
                context="Alle bebuchbaren Konten des SKR04"
                action={
                  <div className="flex items-center gap-2">
                    <Button variant="quiet" onClick={() => setShowCatalog((v) => !v)}>
                      {showCatalog ? 'Rahmen ausblenden' : 'Rahmen anzeigen'}
                    </Button>
                    <HelpPopover label="Erklärung zum Kontenrahmen">
                      Bereichskonten wie 4400-4409 sind eine Kurzschreibweise für zehn nutzbare
                      Konten, keine eigenen Konten. Kontenklasse 8 hält die DATEV im SKR04 frei und
                      ist hier ausgeblendet, weil dort nicht gebucht werden darf. Erlöse liegen
                      anders als im SKR03 in Klasse 4.
                    </HelpPopover>
                  </div>
                }
              >
                <SearchInput
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  placeholder="Kontonummer oder Bezeichnung, z. B. 6815 oder Bürobedarf"
                  aria-label="Kontenrahmen durchsuchen"
                />

                <div className="mt-4">
                  {search.trim() ? (
                    searchResults.length ? (
                      <AccountTable accounts={searchResults} onSelect={openLedger} />
                    ) : (
                      <EmptyState
                        title="Kein Konto gefunden"
                        description="Erlöse liegen im SKR04 in Klasse 4, etwa 4400, nicht in Klasse 8 wie im SKR03."
                      />
                    )
                  ) : showCatalog ? (
                    <div className="divide-y divide-line border-t border-line">
                      {catalogByClass.map(({ kontenklasse, accounts: classAccounts }) => {
                        const isOpen = openClasses[kontenklasse] ?? false;
                        return (
                          <div key={kontenklasse}>
                            <button
                              type="button"
                              aria-expanded={isOpen}
                              onClick={() =>
                                setOpenClasses((prev) => ({ ...prev, [kontenklasse]: !prev[kontenklasse] }))
                              }
                              className="w-full flex items-center gap-2 px-2 py-2.5 -mx-2 rounded-control text-left
                                         transition-colors duration-120 ease-quiet hover:bg-sunken"
                            >
                              {isOpen ? (
                                <ChevronDown className="w-4 h-4 shrink-0 text-ink-faint" strokeWidth={1.5} />
                              ) : (
                                <ChevronRight className="w-4 h-4 shrink-0 text-ink-faint" strokeWidth={1.5} />
                              )}
                              <span className="code-num text-caption text-ink-subtle">
                                Klasse {kontenklasse}
                              </span>
                              <span className="text-body text-ink truncate">
                                {CLASS_NAMES[kontenklasse] ?? ''}
                              </span>
                              <span className="ml-auto text-caption text-ink-subtle num">
                                {classAccounts.length}
                              </span>
                            </button>
                            {isOpen && (
                              <div className="pb-4">
                                <AccountTable accounts={classAccounts} onSelect={openLedger} />
                              </div>
                            )}
                          </div>
                        );
                      })}
                    </div>
                  ) : null}
                </div>
              </Section>
            </>
          )}
        </TabPanel>

        <TabPanel value="susa">
          {loading ? <SkeletonRows rows={8} /> : <SuSaView susa={susa} onSelect={openLedger} />}
        </TabPanel>
      </Tabs>
    </div>
  );
};

// -------------------------------------------------------------------------

const AccountTable: React.FC<{
  accounts: Account[];
  onSelect: (accountNumber: string) => void;
  showTurnover?: boolean;
}> = ({ accounts, onSelect, showTurnover = false }) => (
  <Table>
    <Thead>
      <Tr>
        <Th className="w-28">Konto</Th>
        <Th>Bezeichnung</Th>
        <Th className="w-32">Art</Th>
        {showTurnover && (
          <>
            <Th numeric className="w-32">
              Soll
            </Th>
            <Th numeric className="w-32">
              Haben
            </Th>
          </>
        )}
        <Th numeric className="w-40">
          Saldo
        </Th>
      </Tr>
    </Thead>
    <Tbody>
      {accounts.map((account) => {
        const postable = !account.isRange;
        // Ein Bereich ist keine Zeile zum Anklicken. Er wird ruhiger gesetzt,
        // statt ihn mit einer eigenen Fläche hervorzuheben.
        return (
          <Tr
            key={account.number}
            onClick={postable ? () => onSelect(account.number) : undefined}
            className={postable ? 'cursor-pointer' : 'text-ink-subtle'}
          >
            <Td code>{account.number}</Td>
            <Td className="max-w-[26rem]">
              <span className="block truncate">{account.name}</span>
              {account.isRange ? (
                <span className="block text-caption text-ink-subtle">
                  Kurzschreibweise für die Konten dieses Bereichs
                </span>
              ) : account.aggregatedAccounts ? (
                <span className="block text-caption text-ink-subtle">
                  verdichtet aus {account.aggregatedAccounts} Personenkonten
                </span>
              ) : null}
            </Td>
            <Td className="text-ink-muted">{TYPE_LABELS[account.type] ?? '—'}</Td>
            {showTurnover && (
              <>
                <Td numeric className="text-ink-subtle">
                  {account.debitSum ? formatCents(account.debitSum) : '—'}
                </Td>
                <Td numeric className="text-ink-subtle">
                  {account.creditSum ? formatCents(account.creditSum) : '—'}
                </Td>
              </>
            )}
            <Td numeric>
              {formatCents(account.balance)}
              {account.bookingsCount > 0 && account.balance < 0 && (
                <span className="block text-caption text-attention-text">{balanceHint(account)}</span>
              )}
            </Td>
          </Tr>
        );
      })}
    </Tbody>
  </Table>
);

// -------------------------------------------------------------------------

const SuSaView: React.FC<{ susa: SuSaOverview | null; onSelect: (n: string) => void }> = ({
  susa,
  onSelect,
}) => {
  if (!susa) return null;

  const classesWithBookings = susa.classes.filter((c) => c.accountsCount > 0);

  return (
    <>
      <StatRow>
        <Stat label="Summe Soll" value={formatCents(susa.totalDebit)} context="alle Konten" />
        <Stat label="Summe Haben" value={formatCents(susa.totalCredit)} context="alle Konten" />
        <Stat
          label={
            <>
              Abweichung
              <HelpTooltip
                label="Erklärung zur Abweichung"
                content="Die Prüfung ist exakt und nicht auf Cent gerundet: Jede Buchung wird schon beim Speichern auf Ausgeglichenheit geprüft."
              />
            </>
          }
          value={susa.isBalanced ? 'keine' : formatCents(susa.difference)}
          context={susa.isBalanced ? 'Soll und Haben stimmen überein' : 'Integrität im Protokoll prüfen'}
          tone={susa.isBalanced ? 'positive' : 'negative'}
        />
      </StatRow>

      {classesWithBookings.length === 0 ? (
        <div className="mt-8">
          <EmptyState title="Noch keine Buchungen im aktiven Geschäftsjahr" />
        </div>
      ) : (
        classesWithBookings.map((cls, index) => (
          <Section
            key={cls.kontenklasse}
            title={`Klasse ${cls.kontenklasse} · ${CLASS_NAMES[cls.kontenklasse] ?? cls.kontenklasseName}`}
            context={`Soll ${formatCents(cls.totalDebit)} · Haben ${formatCents(cls.totalCredit)}`}
            divider={index > 0}
            className={index === 0 ? 'mt-8' : undefined}
          >
            <AccountTable accounts={cls.accounts} onSelect={onSelect} showTurnover />
          </Section>
        ))
      )}
    </>
  );
};

// -------------------------------------------------------------------------

const LedgerView: React.FC<{
  ledger: AccountLedger;
  loading: boolean;
  onBack: () => void;
  /** Weiter zur Buchung im Journal — der nächste Schritt des Drill-downs. */
  onOpenEntry?: (entryNumber: string) => void;
}> = ({ ledger, loading, onBack, onOpenEntry }) => {
  const account = ledger.account;
  const rows = ledger.rows ?? [];

  return (
    <div className="max-w-[1200px] mx-auto px-8 py-8">
      <Button
        variant="quiet"
        size="sm"
        onClick={onBack}
        icon={<ArrowLeft className="w-4 h-4" strokeWidth={1.5} />}
        className="-ml-2.5 mb-4"
      >
        Zurück zur Kontenübersicht
      </Button>

      <PageHeader
        title={`${account.number} · ${account.name}`}
        context={`${TYPE_LABELS[account.type] ?? ''} · Kontenklasse ${account.kontenklasse} · ${
          account.posten || account.category
        }`}
      />

      <div className="mt-6">
        <StatRow>
          <Stat
            label={
              <>
                Saldo
                <HelpTooltip label="Erklärung zum Saldo" content={BALANCE_HELP} />
              </>
            }
            value={formatCents(ledger.closingBalance)}
            context={`${balanceHint(account)} · Geschäftsjahr ${ledger.fiscalYear}`}
          />
          <Stat label="Summe Soll" value={formatCents(ledger.totalDebit)} />
          <Stat label="Summe Haben" value={formatCents(ledger.totalCredit)} />
          <Stat label="Zeilen" value={String(ledger.rowCount)} />
        </StatRow>
      </div>

      <Section title="Kontoblatt" context={`Alle Bewegungen im Geschäftsjahr ${ledger.fiscalYear}`}>
        {loading ? (
          <SkeletonRows rows={6} />
        ) : rows.length === 0 ? (
          <EmptyState
            title="Keine Bewegungen"
            description={`Auf diesem Konto wurde im Geschäftsjahr ${ledger.fiscalYear} nicht gebucht.`}
          />
        ) : (
          <Table>
            <Thead sticky>
              <Tr>
                <Th className="w-28">Datum</Th>
                <Th className="w-32">Buchung</Th>
                <Th>Buchungstext</Th>
                <Th className="w-48">
                  <span className="flex items-center">
                    Gegenkonten
                    <HelpTooltip
                      label="Erklärung zu den Gegenkonten"
                      content="Eine Buchung besteht aus beliebig vielen Zeilen, deshalb steht hier eine Liste und nicht ein einzelnes Gegenkonto."
                    />
                  </span>
                </Th>
                <Th numeric className="w-32">
                  Soll
                </Th>
                <Th numeric className="w-32">
                  Haben
                </Th>
                <Th numeric className="w-36">
                  Saldo
                </Th>
              </Tr>
            </Thead>
            <Tbody>
              {rows.map((row, index) => (
                <Tr
                  key={`${row.entryId}-${index}`}
                  variant={row.kind === 'reversal' ? 'storno' : 'default'}
                >
                  <Td className="text-ink-subtle num">{formatDate(row.bookingDate)}</Td>
                  <Td code>
                    {onOpenEntry ? (
                      <button
                        type="button"
                        onClick={() => onOpenEntry(row.entryNumber)}
                        className="code-num text-accent-text hover:underline"
                      >
                        {row.entryNumber}
                      </button>
                    ) : (
                      row.entryNumber
                    )}
                    {row.kind === 'reversal' && (
                      <span className="block text-negative-text">Generalumkehr</span>
                    )}
                  </Td>
                  <Td className="max-w-[24rem]">
                    <span className="block truncate">{row.description}</span>
                    {row.documentNumber && (
                      <span className="block code-num text-caption text-ink-subtle">
                        {row.documentNumber}
                      </span>
                    )}
                  </Td>
                  <Td
                    className="code-num text-caption text-ink-muted"
                    title={(row.counterAccounts ?? [])
                      .map((counter) => `${counter.account} ${counter.name} · ${formatCents(counter.amount)}`)
                      .join('\n')}
                  >
                    {(row.counterAccounts ?? []).map((counter) => counter.account).join(' · ') || '—'}
                  </Td>
                  <Td numeric>{row.debitAmount ? formatCents(row.debitAmount) : ''}</Td>
                  <Td numeric>{row.creditAmount ? formatCents(row.creditAmount) : ''}</Td>
                  <Td numeric className="font-medium">
                    {formatCents(row.runningBalance)}
                  </Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
      </Section>
    </div>
  );
};
