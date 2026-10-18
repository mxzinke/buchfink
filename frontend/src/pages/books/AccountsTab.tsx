import React, { useEffect, useMemo, useState } from 'react';
import type { Account, AccountLedger, OpenItem, SuSaOverview } from '../../types';
import { useNavigation, useRouteObject } from '../../navigation';
import { useWorkspace } from '../../workspace';
import { Api } from '../../services/api';
import { formatBalance, formatCents, formatDate } from '../../utils/formatters';
import {
  DetailList,
  DetailPanel,
  Disclosure,
  EmptyState,
  Notice,
  SearchInput,
  RowButton,
  SkeletonRows,
  StatusBadge,
  Table,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
  cn,
} from '../../components/ui';
import { PanelHeading, errorText, todayISO } from './shared';

/** Mehr Treffer zeigt die Suche im Kontenrahmen nicht; wer mehr braucht, tippt weiter. */
const SEARCH_LIMIT = 60;

/**
 * Die Konten: zuerst die bebuchten mit ihrem Saldo, die Suche reicht in den
 * ganzen Kontenrahmen. Ein Konto öffnet sein Kontoblatt in der Seitenleiste,
 * von dort führt jede Zeile zur Buchung im Journal. Summen und Salden sowie
 * die offenen Posten stehen eingeklappt darunter.
 */
export const AccountsTab: React.FC = () => {
  const { year } = useWorkspace();
  const { id, open, close } = useRouteObject('books', 'accounts');
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [search, setSearch] = useState('');

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    Api.getAccounts()
      .then((list) => {
        if (cancelled) return;
        setAccounts(list ?? []);
        setError(null);
      })
      .catch((e) => !cancelled && setError(errorText(e)))
      .finally(() => !cancelled && setLoading(false));
    return () => {
      cancelled = true;
    };
  }, [year]);

  const inUse = useMemo(
    () => accounts.filter((a) => a.bookingsCount > 0).sort((a, b) => a.number.localeCompare(b.number)),
    [accounts],
  );

  const query = search.trim().toLowerCase();
  const results = useMemo(() => {
    if (!query) return [];
    return accounts
      .filter((a) => !a.isReserved && !a.isRange && a.kontenklasse !== 8)
      .filter((a) => a.number.startsWith(query) || a.name.toLowerCase().includes(query))
      .sort((a, b) => a.number.localeCompare(b.number))
      .slice(0, SEARCH_LIMIT);
  }, [accounts, query]);

  const shown = query ? results : inUse;

  return (
    <div>
      <div className="flex flex-wrap items-center gap-3">
        <SearchInput
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Kontonummer oder Bezeichnung im Kontenrahmen"
          aria-label="Konten durchsuchen"
          className="max-w-md"
        />
        <p className="text-caption text-ink-subtle">
          {loading ? '' : query ? `${results.length} Treffer im SKR04` : `${inUse.length} bebuchte Konten`}
        </p>
      </div>

      <div className="mt-4">
        {error ? (
          <Notice tone="negative" text={error} />
        ) : loading ? (
          <SkeletonRows rows={8} />
        ) : shown.length === 0 ? (
          query ? (
            <EmptyState variant="gefiltert" title="Kein Konto passt zur Suche" />
          ) : (
            <EmptyState
              title="Noch kein Konto bebucht"
              description={`Im Geschäftsjahr ${year} ist noch nichts gebucht.`}
            />
          )
        ) : (
          <Table density="kompakt">
            <Thead sticky>
              <Tr>
                <Th className="w-24">Konto</Th>
                <Th>Bezeichnung</Th>
                <Th numeric optional className="w-36">
                  Soll
                </Th>
                <Th numeric optional className="w-36">
                  Haben
                </Th>
                <Th numeric className="w-44">
                  Saldo
                </Th>
              </Tr>
            </Thead>
            <Tbody>
              {shown.map((account) => (
                <Tr
                  key={account.number}
                  variant={account.number === id ? 'selected' : 'default'}
                  className="cursor-pointer"
                  onClick={() => open(account.number)}
                >
                  <Td code>
                    <RowButton onOpen={() => open(account.number)}>{account.number}</RowButton>
                  </Td>
                  <Td fill title={account.name}>
                    {account.name}
                  </Td>
                  <Td numeric optional className="text-ink-subtle">
                    {account.debitSum ? formatCents(account.debitSum) : '—'}
                  </Td>
                  <Td numeric optional className="text-ink-subtle">
                    {account.creditSum ? formatCents(account.creditSum) : '—'}
                  </Td>
                  <Td numeric>{account.bookingsCount > 0 ? formatBalance(account.balance, account.type) : '—'}</Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
      </div>

      <Disclosure title="Summen und Salden" context="Alle Konten nach Kontenklassen">
        <SuSa onOpen={open} />
      </Disclosure>

      <Disclosure title="Offene Posten" context="Forderungen und Verbindlichkeiten zum heutigen Tag">
        <OpenItems />
      </Disclosure>

      <LedgerPanel account={id} onClose={close} />
    </div>
  );
};

/** Die Summen- und Saldenliste, je Kontenklasse eine Tabelle. */
const SuSa: React.FC<{ onOpen: (account: string) => void }> = ({ onOpen }) => {
  const [susa, setSusa] = useState<SuSaOverview | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    Api.getSuSaOverview()
      .then(setSusa)
      .catch((e) => setError(errorText(e)));
  }, []);

  if (error) return <Notice tone="negative" text={error} />;
  if (!susa) return <SkeletonRows rows={6} />;
  const classes = susa.classes.filter((c) => c.accountsCount > 0);

  return (
    <div>
      <DetailList
        items={[
          { label: 'Summe Soll', value: formatCents(susa.totalDebit), numeric: true },
          { label: 'Summe Haben', value: formatCents(susa.totalCredit), numeric: true },
          {
            label: 'Abweichung',
            value: susa.isBalanced ? (
              <StatusBadge status="ausgeglichen" />
            ) : (
              <span className="num text-negative-text">{formatCents(susa.difference)}</span>
            ),
          },
        ]}
      />
      {classes.map((cls) => (
        <div key={cls.kontenklasse} className="mt-6">
          <PanelHeading>{`Klasse ${cls.kontenklasse} · ${cls.kontenklasseName}`}</PanelHeading>
          <Table density="kompakt">
            <Thead>
              <Tr>
                <Th className="w-24">Konto</Th>
                <Th>Bezeichnung</Th>
                <Th numeric className="w-36">
                  Soll
                </Th>
                <Th numeric className="w-36">
                  Haben
                </Th>
                <Th numeric className="w-44">
                  Saldo
                </Th>
              </Tr>
            </Thead>
            <Tbody>
              {cls.accounts.map((account) => (
                <Tr key={account.number} className="cursor-pointer" onClick={() => onOpen(account.number)}>
                  <Td code>
                    <RowButton onOpen={() => onOpen(account.number)}>{account.number}</RowButton>
                  </Td>
                  <Td fill title={account.name}>
                    {account.name}
                  </Td>
                  <Td numeric className="text-ink-subtle">
                    {formatCents(account.debitSum)}
                  </Td>
                  <Td numeric className="text-ink-subtle">
                    {formatCents(account.creditSum)}
                  </Td>
                  <Td numeric>{formatBalance(account.balance, account.type)}</Td>
                </Tr>
              ))}
              <Tr variant="sum">
                <Td />
                <Td>Summe</Td>
                <Td numeric>{formatCents(cls.totalDebit)}</Td>
                <Td numeric>{formatCents(cls.totalCredit)}</Td>
                <Td />
              </Tr>
            </Tbody>
          </Table>
        </div>
      ))}
    </div>
  );
};

/**
 * Die offenen Posten. Forderungen und Verbindlichkeiten stehen getrennt: es
 * sind zwei Sachverhalte, und eine gemeinsame Summe wäre eine Saldierung.
 */
const OpenItems: React.FC = () => {
  const [items, setItems] = useState<OpenItem[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const today = todayISO();

  useEffect(() => {
    Api.getOpenItemsAt('')
      .then(setItems)
      .catch((e) => setError(errorText(e)));
  }, []);

  if (error) return <Notice tone="negative" text={error} />;
  if (!items) return <SkeletonRows rows={4} />;
  if (items.length === 0) {
    return <EmptyState title="Keine offenen Posten" description="Jede Rechnung ist ausgeglichen." />;
  }

  const sides: { title: string; rows: OpenItem[] }[] = [
    { title: 'Forderungen', rows: items.filter((i) => i.contactType === 'customer') },
    { title: 'Verbindlichkeiten', rows: items.filter((i) => i.contactType === 'vendor') },
  ];

  return (
    <div className="space-y-6">
      {sides
        .filter((side) => side.rows.length > 0)
        .map((side) => (
          <div key={side.title}>
            <PanelHeading>
              {`${side.title} · ${formatCents(side.rows.reduce((sum, row) => sum + row.openAmount, 0))}`}
            </PanelHeading>
            <Table density="kompakt">
              <Thead>
                <Tr>
                  <Th className="w-32">Beleg</Th>
                  <Th>Partner</Th>
                  <Th className="w-28">Fällig</Th>
                  <Th numeric className="w-36">
                    Offen
                  </Th>
                  <Th className="w-44">Zustand</Th>
                </Tr>
              </Thead>
              <Tbody>
                {side.rows.map((item) => {
                  const overdue = Boolean(item.dueDate) && item.dueDate < today;
                  return (
                    <Tr key={`${item.source}-${item.entryId}-${item.advanceInvoiceId ?? 0}`}>
                      <Td code>{item.documentNumber || item.entryNumber}</Td>
                      <Td className="max-w-[20rem] truncate">{item.contactName}</Td>
                      <Td className={cn('num', overdue ? 'text-negative-text' : 'text-ink-subtle')}>
                        {formatDate(item.dueDate)}
                      </Td>
                      <Td numeric>{formatCents(item.openAmount)}</Td>
                      <Td>
                        <StatusBadge
                          status={
                            overdue
                              ? 'ueberfaellig'
                              : item.settledAmount > 0
                                ? 'teilweiseAusgeglichen'
                                : 'offen'
                          }
                        />
                      </Td>
                    </Tr>
                  );
                })}
              </Tbody>
            </Table>
          </div>
        ))}
    </div>
  );
};

/** Das Kontoblatt in der Seitenleiste; jede Zeile führt zur Buchung im Journal. */
const LedgerPanel: React.FC<{ account: string | undefined; onClose: () => void }> = ({
  account,
  onClose,
}) => {
  const { navigate } = useNavigation();
  const [ledger, setLedger] = useState<AccountLedger | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!account) return;
    let cancelled = false;
    setError(null);
    setLedger((current) => (current?.account.number === account ? current : null));
    Api.getAccountLedger(account)
      .then((result) => !cancelled && setLedger(result))
      .catch((e) => !cancelled && setError(errorText(e)));
    return () => {
      cancelled = true;
    };
  }, [account]);

  const rows = ledger?.rows ?? [];
  const title = ledger ? `${ledger.account.number} · ${ledger.account.name}` : (account ?? '');

  return (
    <DetailPanel
      open={Boolean(account)}
      onOpenChange={(next) => !next && onClose()}
      title={title}
      context={
        ledger
          ? `Geschäftsjahr ${ledger.fiscalYear} · ${ledger.account.posten || ledger.account.category}`
          : undefined
      }
      width="lg"
    >
      {error ? (
        <Notice tone="negative" text={error} />
      ) : !ledger ? (
        <SkeletonRows rows={6} />
      ) : (
        <>
          <DetailList
            items={[
              { label: 'Saldo', value: formatBalance(ledger.closingBalance, ledger.account?.type), numeric: true },
              ...(ledger.openingBalance
                ? [{ label: 'Vortrag', value: formatBalance(ledger.openingBalance, ledger.account?.type), numeric: true }]
                : []),
              { label: 'Summe Soll', value: formatCents(ledger.totalDebit), numeric: true },
              { label: 'Summe Haben', value: formatCents(ledger.totalCredit), numeric: true },
            ]}
          />

          <PanelHeading className="mt-6">Kontoblatt</PanelHeading>
          {rows.length === 0 ? (
            <p className="text-body text-ink-subtle">Auf diesem Konto ist im Geschäftsjahr nichts gebucht.</p>
          ) : (
            <table className="w-full text-body">
              <thead>
                <tr className="[&>th]:h-7 [&>th]:text-label [&>th]:font-medium [&>th]:text-ink-subtle [&>th]:border-b [&>th]:border-line-strong">
                  <th className="text-left w-20">Datum</th>
                  <th className="text-left w-28">Buchung</th>
                  <th className="text-left">Text</th>
                  <th className="text-right w-24">Soll</th>
                  <th className="text-right w-24">Haben</th>
                  <th className="text-right w-24">Saldo</th>
                </tr>
              </thead>
              <tbody>
                {rows.map((row, index) => (
                  <tr
                    key={`${row.entryId}-${index}`}
                    className={cn(
                      '[&>td]:h-8 [&>td]:border-b [&>td]:border-line',
                      row.kind === 'reversal' && '[&>td]:bg-negative-soft/60',
                    )}
                  >
                    <td className="num text-ink-subtle">{formatDate(row.bookingDate)}</td>
                    <td>
                      <button
                        type="button"
                        onClick={() => navigate({ area: 'books', tab: 'journal', id: row.entryNumber })}
                        className="code-num text-caption text-accent-text hover:underline"
                      >
                        {row.entryNumber}
                      </button>
                    </td>
                    <td className="truncate max-w-0" title={row.description}>
                      {row.description}
                    </td>
                    <td className="text-right num">{row.debitAmount ? formatCents(row.debitAmount) : ''}</td>
                    <td className="text-right num">
                      {row.creditAmount ? formatCents(row.creditAmount) : ''}
                    </td>
                    <td className="text-right num font-medium">
                      {formatBalance(row.runningBalance, ledger.account?.type)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </>
      )}
    </DetailPanel>
  );
};
