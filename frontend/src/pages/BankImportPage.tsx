import React, { useEffect, useMemo, useState } from 'react';
import { ArrowDownLeft, ArrowUpRight, Ban, Landmark, MoreHorizontal, Upload } from 'lucide-react';
import type {
  Account,
  AllocationRequest,
  BankTransaction,
  DifferenceKind,
  DifferenceKindInfo,
  OpenItem,
} from '../types';
import { Api } from '../services/api';
import { useWriteLock } from '../components/WriteLock';
import { formatCents, formatDate, parseCents } from '../utils/formatters';
import {
  Button,
  Checkbox,
  Combobox,
  Dialog,
  EmptyState,
  Field,
  HelpPopover,
  Input,
  Menu,
  MenuItem,
  PageHeader,
  Section,
  Select,
  SkeletonRows,
  TabPanel,
  Table,
  Tabs,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
  cn,
  toast,
} from '../components/ui';

/**
 * Bank & Zahlungen.
 *
 * Ein Kontoauszug kennt zwei Fälle. Entweder gehört die Zahlung zu einem offenen
 * Posten — dann wird sie zugeordnet, samt Skonto oder Gebühr. Oder es gibt keinen
 * Beleg, etwa bei Kontoführungsentgelten — dann wird direkt gegen ein Konto
 * gebucht. Beides steht als Reiter nebeneinander, statt beides über dieselbe
 * Maske zu zwingen.
 */

/**
 * Der Grund, warum „Forderung ausbuchen" zu einem Posten nicht offensteht —
 * `undefined`, wenn die Aktion möglich ist.
 *
 * Ein stumm deaktivierter Menüeintrag lässt den Anwender raten, was er falsch
 * gemacht hat (§10.4: ein deaktivierter Knopf braucht seine Erklärung im
 * `title`). Beide Sperren haben einen fachlichen und keinen technischen Grund:
 * ein Abschlag ist bis zur Vereinnahmung ein Merkposten ohne Buchung, es gibt
 * also nichts auszubuchen, und eine Verbindlichkeit erlischt durch Zahlung oder
 * Verjährung, nicht durch einen Forderungsverlust nach § 17 UStG.
 */
function writeOffBlockedReason(item: OpenItem, lockHint?: string): string | undefined {
  if (lockHint) return lockHint;
  if (item.source === 'advance')
    return 'Ein Abschlag wird vereinnahmt, nicht ausgebucht: bis zur Zahlung ist er ein Merkposten ohne Buchung.';
  if (item.contactType !== 'customer')
    return 'Verbindlichkeiten werden nicht ausgebucht; ausgebucht wird nur eine uneinbringliche Forderung.';
  return undefined;
}

export const BankImportPage: React.FC = () => {
  // Einlesen und Zuordnen sind schreibende Schritte und im Prüfermodus
  // gesperrt; die Liste bleibt lesbar (§10.4).
  const writeLock = useWriteLock();
  const [transactions, setTransactions] = useState<BankTransaction[]>([]);
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [paymentAccounts, setPaymentAccounts] = useState<Account[]>([]);
  const [openItems, setOpenItems] = useState<OpenItem[]>([]);
  const [differenceKinds, setDifferenceKinds] = useState<DifferenceKindInfo[]>([]);
  const [importAccount, setImportAccount] = useState('1800');
  const [active, setActive] = useState<BankTransaction | null>(null);
  const [writingOff, setWritingOff] = useState<OpenItem | null>(null);
  const [loading, setLoading] = useState(true);
  const [importing, setImporting] = useState(false);

  useEffect(() => {
    void load();
  }, []);

  async function load() {
    setLoading(true);
    try {
      const [txs, accs, payAccs, items, advances, kinds] = await Promise.all([
        Api.getBankTransactions(),
        Api.getAccounts(),
        Api.getPaymentAccounts(),
        Api.getOpenItems(),
        Api.getOpenAdvances(),
        Api.getDifferenceKinds(),
      ]);
      setTransactions(txs);
      setAccounts(accs);
      setPaymentAccounts(payAccs);
      // Die offenen Posten haben zwei Quellen: die gewöhnliche Buchung auf dem
      // Personenkonto und die ausgestellte Abschlagsrechnung, die noch keine
      // Buchung hat. Beide gehören in dieselbe Liste — der Anwender sieht auf
      // dem Kontoauszug nicht, welche Herkunft eine Zahlung hat.
      setOpenItems([...items, ...advances]);
      setDifferenceKinds(kinds);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }

  /**
   * Liest einen Kontoauszug über seinen Dateipfad ein.
   *
   * Nicht über den Inhalt: die CAMT-Datei ist der empfangene Beleg und wird vor
   * dem Auswerten als solcher abgelegt (GoBD Rz. 130 f.). Wer nur den Inhalt
   * übergibt, kann sie danach nicht mehr archivieren.
   */
  async function importStatement() {
    setImporting(true);
    try {
      const path = await Api.selectStatementFile();
      // Ein abgebrochener Dateidialog ist keine Fehlermeldung wert.
      if (!path) return;
      const count = await Api.importCAMTFile(path, importAccount);
      toast.success(`${count} Umsätze eingelesen, der Auszug ist als Beleg abgelegt.`);
      await load();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setImporting(false);
    }
  }

  const unmatched = transactions.filter((t) => t.matchStatus === 'unmatched');

  return (
    <div className="max-w-[1200px] mx-auto px-8 py-8">
      <PageHeader
        title="Bank & Zahlungen"
        context="Kontoauszüge einlesen und Zahlungen den offenen Posten zuordnen"
        action={
          <div className="flex items-center gap-2">
            <Select
              items={paymentAccounts.map((a) => ({ value: a.number, label: `${a.number} · ${a.name}` }))}
              value={importAccount}
              onValueChange={setImportAccount}
              placeholder="Bankkonto"
              className="w-56"
            />
            <Button
              variant="primary"
              icon={<Upload className="w-4 h-4" strokeWidth={1.5} />}
              loading={importing}
              disabled={writeLock.locked}
              title={writeLock.hint}
              onClick={() => void importStatement()}
            >
              CAMT.053 einlesen
            </Button>
          </div>
        }
      />

      <Section
        title="Offene Bankumsätze"
        context={loading ? undefined : `${unmatched.length} von ${transactions.length} noch nicht zugeordnet`}
        divider={false}
        className="mt-8"
        action={
          <HelpPopover label="Erklärung zur Kontierung">
            Buchfink schlägt bewusst kein Gegenkonto vor. Aus dem Verwendungszweck ein Aufwandskonto
            zu raten wäre eine unprüfbare Vermutung an genau der Stelle, an der die Kontierung
            entschieden wird — und für diese Entscheidung haftet das Unternehmen.
          </HelpPopover>
        }
      >
        {loading ? (
          <SkeletonRows rows={6} />
        ) : unmatched.length === 0 ? (
          <EmptyState
            icon={<Landmark className="w-6 h-6" strokeWidth={1.5} />}
            title="Keine offenen Bankumsätze"
            description="Alle eingelesenen Zahlungen sind zugeordnet."
          />
        ) : (
          <Table>
            <Thead>
              <Tr>
                <Th className="w-10" aria-label="Richtung" />
                <Th className="w-28">Datum</Th>
                <Th>Zahlungspartner</Th>
                <Th>Verwendungszweck</Th>
                <Th numeric className="w-36">
                  Betrag
                </Th>
                <Th className="w-28" aria-label="Aktionen" />
              </Tr>
            </Thead>
            <Tbody>
              {unmatched.map((tx) => {
                const incoming = tx.amount > 0;
                return (
                  <Tr key={tx.id} className="group cursor-pointer" onClick={() => setActive(tx)}>
                    <Td className="pr-0 text-ink-faint">
                      {incoming ? (
                        <ArrowDownLeft className="w-4 h-4" strokeWidth={1.5} aria-hidden="true" />
                      ) : (
                        <ArrowUpRight className="w-4 h-4" strokeWidth={1.5} aria-hidden="true" />
                      )}
                      <span className="sr-only">{incoming ? 'Eingang' : 'Ausgang'}</span>
                    </Td>
                    <Td className="text-ink-subtle num">{formatDate(tx.bookingDate)}</Td>
                    <Td className="max-w-[16rem] truncate">{tx.counterpartyName || '—'}</Td>
                    <Td className="max-w-[24rem] truncate text-ink-muted" title={tx.remittanceInfo}>
                      {tx.remittanceInfo}
                    </Td>
                    <Td numeric>{formatCents(tx.amount, tx.currency)}</Td>
                    <Td className="pl-0">
                      <Button
                        variant="quiet"
                        size="sm"
                        disabled={writeLock.locked}
                        title={writeLock.hint}
                        className="opacity-0 transition-opacity duration-120 ease-quiet
                                   group-hover:opacity-100 focus-visible:opacity-100"
                      >
                        Zuordnen
                      </Button>
                    </Td>
                  </Tr>
                );
              })}
            </Tbody>
          </Table>
        )}
      </Section>

      <Section
        title="Offene Posten"
        context={loading ? undefined : `${openItems.length} nicht ausgeglichen`}
        action={
          <HelpPopover label="Erklärung zur Ausbuchung">
            Eine uneinbringliche Forderung wird nicht über den Zahlungsausgleich geschlossen: Es
            fließt kein Geld, und eine Zahlung über null wäre eine Behauptung. Sie wird als
            Forderungsverlust gebucht, die Umsatzsteuer wird dabei berichtigt
            (§ 17 Abs. 2 Nr. 1 UStG). Ein Abschlag steht hier ohne Buchung — er wird über den
            Kontoauszug oder unter Anzahlungen vereinnahmt.
          </HelpPopover>
        }
      >
        {loading ? (
          <SkeletonRows rows={4} />
        ) : openItems.length === 0 ? (
          <EmptyState
            title="Keine offenen Posten"
            description="Jede Forderung und jede Verbindlichkeit ist ausgeglichen."
          />
        ) : (
          <Table density="kompakt">
            <Thead>
              <Tr>
                <Th className="w-40">Beleg</Th>
                <Th>Partner</Th>
                <Th className="w-28">Quelle</Th>
                <Th className="w-28">Fällig</Th>
                <Th numeric className="w-36">
                  Offen
                </Th>
                <Th className="w-20" aria-label="Aktionen" />
              </Tr>
            </Thead>
            <Tbody>
              {openItems.map((item) => (
                <Tr
                  key={item.source === 'advance' ? `advance-${item.advanceInvoiceId}` : item.entryId}
                  className="group"
                >
                  <Td code>{item.documentNumber || item.entryNumber}</Td>
                  <Td className="max-w-[20rem] truncate">{item.contactName}</Td>
                  <Td className="text-ink-muted">
                    {item.source === 'advance' ? 'Abschlag' : 'Buchung'}
                  </Td>
                  <Td className="text-ink-subtle num">
                    {item.dueDate ? formatDate(item.dueDate) : '—'}
                  </Td>
                  <Td numeric>{formatCents(item.openAmount)}</Td>
                  <Td className="pl-0">
                    <span className="flex items-center justify-end">
                      <Menu
                        trigger={
                          <Button
                            variant="quiet"
                            size="sm"
                            iconOnly
                            title="Aktionen zu diesem Posten"
                            aria-label={`Aktionen zu ${item.documentNumber || item.entryNumber}`}
                          >
                            <MoreHorizontal className="w-4 h-4" strokeWidth={1.5} />
                          </Button>
                        }
                      >
                        <MenuItem
                          disabled={Boolean(writeOffBlockedReason(item, writeLock.hint))}
                          title={writeOffBlockedReason(item, writeLock.hint)}
                          onClick={() => setWritingOff(item)}
                        >
                          Forderung ausbuchen
                        </MenuItem>
                      </Menu>
                    </span>
                  </Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
      </Section>

      <WriteOffDialog
        item={writingOff}
        onClose={() => setWritingOff(null)}
        onDone={async () => {
          setWritingOff(null);
          await load();
        }}
      />

      {active && (
        <AssignDialog
          tx={active}
          accounts={accounts}
          openItems={openItems}
          differenceKinds={differenceKinds}
          onClose={() => setActive(null)}
          onDone={async () => {
            setActive(null);
            await load();
          }}
        />
      )}
    </div>
  );
};

// -------------------------------------------------------------------------

const AssignDialog: React.FC<{
  tx: BankTransaction;
  accounts: Account[];
  openItems: OpenItem[];
  differenceKinds: DifferenceKindInfo[];
  onClose: () => void;
  onDone: () => void;
}> = ({ tx, accounts, openItems, differenceKinds, onClose, onDone }) => {
  const writeLock = useWriteLock();
  const [mode, setMode] = useState<'open_item' | 'direct'>('open_item');
  const [busy, setBusy] = useState<'submit' | 'ignore' | null>(null);

  // Zahlungseingänge gleichen Forderungen aus, Ausgänge Verbindlichkeiten.
  const relevant = useMemo(
    () => openItems.filter((i) => (tx.amount > 0 ? i.contactType === 'customer' : i.contactType === 'vendor')),
    [openItems, tx.amount],
  );

  const [selected, setSelected] = useState<Record<number, { amount: string; kind: DifferenceKind; diff: string }>>(
    {},
  );
  /**
   * Der gewählte Abschlag. Er steht neben den übrigen Posten und nicht in
   * ihnen: eine Abschlagsrechnung hat keine Buchung, gegen die ein Ausgleich
   * liefe, und wird über SettleAdvance vereinnahmt — mit dem ganzen Betrag,
   * denn erst damit entsteht die Steuer. Deshalb schließt seine Auswahl die
   * andere aus.
   */
  const [advance, setAdvance] = useState<OpenItem | null>(null);
  const [counterAccount, setCounterAccount] = useState<string | null>(null);
  const [description, setDescription] = useState(`${tx.counterpartyName} – ${tx.remittanceInfo}`);

  const postable = useMemo(
    () => accounts.filter((a) => !a.isRange && !a.isReserved && a.kontenklasse !== 8),
    [accounts],
  );

  const allocations: AllocationRequest[] = Object.entries(selected).map(([entryId, value]) => ({
    openItemEntryId: Number(entryId),
    settledAmount: parseCents(value.amount) ?? 0,
    differenceKind: value.kind,
    differenceAmount: value.kind === 'none' ? 0 : parseCents(value.diff) ?? 0,
  }));

  const cashTotal = allocations.reduce((sum, a) => {
    if (a.differenceKind === 'bank_fee') return sum + a.settledAmount + a.differenceAmount;
    if (a.differenceKind === 'none') return sum + a.settledAmount;
    return sum + a.settledAmount - a.differenceAmount;
  }, 0);
  const statementAmount = Math.abs(tx.amount);
  const matches = cashTotal === statementAmount && allocations.length > 0;

  function toggle(item: OpenItem) {
    if (item.source === 'advance') {
      setSelected({});
      setAdvance((prev) => (prev?.advanceInvoiceId === item.advanceInvoiceId ? null : item));
      return;
    }
    setAdvance(null);
    setSelected((prev) => {
      if (prev[item.entryId]) {
        const next = { ...prev };
        delete next[item.entryId];
        return next;
      }
      return {
        ...prev,
        [item.entryId]: { amount: formatCents(item.openAmount, ''), kind: 'none', diff: '' },
      };
    });
  }

  /** Ob dieser Posten in der Liste angehakt ist — gleich welcher Quelle. */
  function isSelected(item: OpenItem): boolean {
    return item.source === 'advance'
      ? advance?.advanceInvoiceId === item.advanceInvoiceId
      : Boolean(selected[item.entryId]);
  }

  async function submit() {
    setBusy('submit');
    try {
      if (mode === 'open_item' && advance) {
        // Konto, Datum und Betrag kommen aus dem Kontoauszug: keines davon ist
        // dann noch eine Eingabe, und keines kann sich vertippen.
        await Api.settleAdvance({
          advanceId: advance.advanceInvoiceId ?? 0,
          bankTxId: tx.id,
          paymentDate: tx.bookingDate,
          paymentAccount: tx.ledgerAccount,
        });
        toast.success(`Anzahlung ${advance.documentNumber} vereinnahmt.`);
      } else if (mode === 'open_item') {
        await Api.settlePayment({
          bankTxId: tx.id,
          paymentAccount: tx.ledgerAccount,
          paymentDate: tx.bookingDate,
          valueDate: tx.valueDate,
          allocations,
        });
        toast.success('Zahlung zugeordnet.');
      } else {
        await Api.bookBankTransactionDirect(tx.id, (counterAccount ?? '').trim(), description);
        toast.success('Bankumsatz gebucht.');
      }
      onDone();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(null);
    }
  }

  async function ignore() {
    setBusy('ignore');
    try {
      await Api.ignoreBankTransaction(tx.id);
      onDone();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(null);
    }
  }

  const canSubmit =
    mode === 'open_item' ? Boolean(advance) || matches : Boolean(counterAccount?.trim());

  return (
    <Dialog
      open
      onOpenChange={(next) => {
        if (!next) onClose();
      }}
      title="Bankumsatz zuordnen"
      width="max-w-3xl"
      footer={
        <>
          <Button
            variant="quiet"
            icon={<Ban className="w-4 h-4" strokeWidth={1.5} />}
            loading={busy === 'ignore'}
            disabled={writeLock.locked}
            title={writeLock.hint}
            onClick={ignore}
            className="mr-auto"
          >
            Nicht buchen
          </Button>
          <Button variant="secondary" onClick={onClose}>
            Abbrechen
          </Button>
          <Button
            variant="primary"
            loading={busy === 'submit'}
            disabled={!canSubmit || writeLock.locked}
            title={writeLock.hint}
            onClick={submit}
          >
            Buchen
          </Button>
        </>
      }
    >
      <p className="text-caption text-ink-subtle -mt-1 mb-5">
        {formatDate(tx.bookingDate)} · {tx.counterpartyName} ·{' '}
        <span className="num">{formatCents(tx.amount, tx.currency)}</span> auf Konto{' '}
        <span className="code-num">{tx.ledgerAccount}</span>
      </p>

      <Tabs
        items={[
          { value: 'open_item', label: 'Offenen Posten ausgleichen', count: relevant.length },
          { value: 'direct', label: 'Ohne Beleg direkt buchen' },
        ]}
        value={mode}
        onValueChange={setMode}
      >
        <TabPanel value="open_item">
          {relevant.length === 0 ? (
            <EmptyState
              title="Keine passenden offenen Posten"
              description="Wenn zu dieser Zahlung kein Beleg gehört, buche sie über den zweiten Reiter direkt."
            />
          ) : (
            <>
              <div className="divide-y divide-line">
                {relevant.map((item) => {
                  const entry = selected[item.entryId];
                  const isAdvance = item.source === 'advance';
                  return (
                    <div
                      key={isAdvance ? `advance-${item.advanceInvoiceId}` : item.entryId}
                      className="py-3 first:pt-0"
                    >
                      <div
                        className={cn(
                          'flex items-center gap-3',
                          isSelected(item) && '-mx-3 px-3 py-2 rounded-control bg-accent-soft',
                        )}
                      >
                        <Checkbox
                          checked={isSelected(item)}
                          onCheckedChange={() => toggle(item)}
                          label={`${item.documentNumber || item.entryNumber} · ${item.contactName}${
                            isAdvance ? ' · Abschlag' : ''
                          }`}
                          className="flex-1 min-w-0"
                        />
                        <span className="shrink-0 text-caption text-ink-subtle">
                          fällig {formatDate(item.dueDate)}
                        </span>
                        <span className="shrink-0 num">{formatCents(item.openAmount)}</span>
                      </div>

                      {entry && !isAdvance && (
                        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 mt-3">
                          <Field label="Ausgleichsbetrag">
                            <Input
                              align="right"
                              value={entry.amount}
                              onChange={(e) =>
                                setSelected((prev) => ({
                                  ...prev,
                                  [item.entryId]: { ...prev[item.entryId], amount: e.target.value },
                                }))
                              }
                            />
                          </Field>
                          <Field
                            label="Differenz"
                            help={`Skonto wird brutto erfasst. Buchfink teilt den Betrag in Entgelt und Steuer und korrigiert die Steuer nach § 17 UStG mit ${
                              item.taxRate ? `${item.taxRate / 100} %` : 'dem Satz des Belegs'
                            }.`}
                          >
                            <Select
                              items={differenceKinds
                                .filter((k) => !k.withoutPayment)
                                .map((k) => ({ value: k.kind, label: k.label }))}
                              value={entry.kind}
                              onValueChange={(kind) =>
                                setSelected((prev) => ({
                                  ...prev,
                                  [item.entryId]: { ...prev[item.entryId], kind },
                                }))
                              }
                            />
                          </Field>
                          {entry.kind !== 'none' && (
                            <Field label="Betrag der Differenz">
                              <Input
                                align="right"
                                placeholder="0,00"
                                value={entry.diff}
                                onChange={(e) =>
                                  setSelected((prev) => ({
                                    ...prev,
                                    [item.entryId]: { ...prev[item.entryId], diff: e.target.value },
                                  }))
                                }
                              />
                            </Field>
                          )}
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>

              {/* Doppellinie wie unter einer Summe: Die Zuordnung muss den
                  Kontoauszug treffen, sonst bleibt die Aktion gesperrt. */}
              <div
                className={cn(
                  'flex items-center justify-between gap-4 mt-4 pt-3 rule-total text-body',
                  matches ? 'text-positive-text' : 'text-ink-muted',
                )}
              >
                <span className="flex items-center gap-2">
                  {matches && <span className="mark-diamond bg-positive" aria-hidden="true" />}
                  {matches
                    ? 'Zuordnung passt zum Kontoauszug'
                    : `Noch ${formatCents(statementAmount - cashTotal)} offen`}
                </span>
                <span className="num">
                  {formatCents(cashTotal)} von {formatCents(statementAmount)}
                </span>
              </div>
            </>
          )}
        </TabPanel>

        <TabPanel value="direct">
          <div className="flex flex-col gap-4 max-w-md">
            <Field
              label="Gegenkonto"
              hint="Für Zinsen, Entgelte oder Umbuchungen"
              help="Die Bankseite kommt aus dem Kontoauszug, die Richtung kann nicht vertippt werden. Buchfink prüft, ob das Gegenkonto im SKR04 existiert und bebucht werden darf."
            >
              <Combobox
                items={postable.map((a) => ({
                  value: a.number,
                  label: `${a.number} ${a.name}`,
                }))}
                value={counterAccount}
                onValueChange={setCounterAccount}
                placeholder="Konto suchen, z. B. 6855"
              />
            </Field>
            <Field label="Buchungstext">
              <Input value={description} onChange={(e) => setDescription(e.target.value)} />
            </Field>
          </div>
        </TabPanel>
      </Tabs>
    </Dialog>
  );
};

// -------------------------------------------------------------------------

/**
 * Die Ausbuchung eines uneinbringlichen Postens.
 *
 * Sie läuft nicht über den Zahlungsausgleich: dort wird ein Zahlungsmittel
 * bewegt, hier fließt nichts. Gebucht werden der Forderungsverlust und die
 * Steuerkorrektur nach § 17 Abs. 2 Nr. 1 UStG — im Zeitraum der
 * Uneinbringlichkeit.
 */
const WriteOffDialog: React.FC<{
  item: OpenItem | null;
  onClose: () => void;
  onDone: () => void;
}> = ({ item, onClose, onDone }) => {
  const writeLock = useWriteLock();
  const [amount, setAmount] = useState('');
  const [date, setDate] = useState('');
  const [reason, setReason] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!item) return;
    setAmount(formatCents(item.openAmount, '').trim());
    setDate(new Date().toISOString().split('T')[0]);
    setReason('');
    setError(null);
  }, [item]);

  async function submit() {
    if (!reason.trim()) {
      setError(
        'Woran die Forderung gescheitert ist, entscheidet über die Uneinbringlichkeit. Bitte nennen Sie den Grund.',
      );
      return;
    }
    const value = parseCents(amount);
    if (value === null || value <= 0) {
      setError('Der auszubuchende Betrag ist nicht lesbar. Erwartet wird etwa 1234,56.');
      return;
    }
    setBusy(true);
    try {
      const entry = await Api.writeOffOpenItem({
        openItemEntryId: item!.entryId,
        amount: value,
        date,
        reason,
      });
      toast.success(`Forderung ausgebucht als ${entry.entryNumber}.`);
      onDone();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog
      open={item !== null}
      onOpenChange={(next) => !next && onClose()}
      title="Forderung ausbuchen"
      width="max-w-lg"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Abbrechen
          </Button>
          <Button
            variant="danger"
            loading={busy}
            disabled={writeLock.locked}
            title={writeLock.hint}
            onClick={submit}
          >
            Ausbuchen
          </Button>
        </>
      }
    >
      {item && (
        <>
          <p className="text-body text-ink-muted">
            <span className="code-num text-ink">{item.documentNumber || item.entryNumber}</span> ·{' '}
            {item.contactName} · offen{' '}
            <span className="num text-ink">{formatCents(item.openAmount)}</span>
            <HelpPopover label="Erklärung zur Ausbuchung">
              Gebucht werden der Forderungsverlust als Aufwand und die Steuerkorrektur nach § 17
              Abs. 2 Nr. 1 UStG gegen das Personenkonto. Der Zeitraum ist der der
              Uneinbringlichkeit, nicht der der Rechnung. Die Begründung steht im Protokoll: eine
              Ausbuchung ohne sie ist von einer vergessenen Forderung nicht zu unterscheiden.
            </HelpPopover>
          </p>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 mt-4">
            <Field label="Auszubuchender Betrag" hint="brutto">
              <Input
                align="right"
                inputMode="decimal"
                value={amount}
                onChange={(e) => setAmount(e.target.value)}
              />
            </Field>
            <Field label="Datum">
              <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
            </Field>
          </div>

          <Field label="Grund der Ausbuchung" className="mt-4" error={error ?? undefined}>
            <Input
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="Insolvenzverfahren mangels Masse abgewiesen"
            />
          </Field>
        </>
      )}
    </Dialog>
  );
};
