import React, { useEffect, useMemo, useState } from 'react';
import { Trash2 } from 'lucide-react';
import type {
  Account,
  Contact,
  OpenOpeningItem,
  OpeningBalanceLine,
  OpeningBalancePreview,
  OpeningBalanceRequest,
  Receipt,
  Side,
} from '../types';
import { Api } from '../services/api';
import { useWriteLock } from '../components/WriteLock';
import type { NavigateFn } from './Sidebar';
import { formatCents, parseCents } from '../utils/formatters';
import {
  Button,
  Combobox,
  Dialog,
  Field,
  HelpPopover,
  Input,
  Notice,
  Select,
  SkeletonRows,
  Table,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
  cn,
} from './ui';

/**
 * Die Eröffnungsbilanz des Umsteigers (ARC-05).
 *
 * Der Erfassungsweg für den, der aus einem Altsystem kommt: die Schlussbilanz
 * von dort wird zu den Anfangsbeständen hier. Sachkonten gegen 9000,
 * Personenkonten je offenem Posten gegen 9008 und 9009 — die Konten setzt das
 * Backend, hier stehen nur die Werte.
 *
 * Offene Posten werden einzeln erfasst und nicht als Summe auf dem
 * Sammelkonto: eine Forderung ohne Nummer und Fälligkeit ließe sich nach der
 * Übernahme nicht mehr ausgleichen, und die Offene-Posten-Liste des ersten
 * Jahres bliebe leer, während in der Bilanz Forderungen stünden.
 *
 * Vorschau und Buchung teilen sich denselben Aufbau im Backend: was hier zu
 * sehen ist, wird auch gebucht.
 */

/** Eine Zeile, solange sie in der Maske steht — Beträge als Text (§8.3). */
interface DraftAccount {
  account: string;
  side: Side;
  amount: string;
  legacyRef: string;
}

interface DraftItem {
  contactId: number | null;
  amount: string;
  documentNumber: string;
  documentDate: string;
  dueDate: string;
  legacyRef: string;
}

const emptyAccount = (): DraftAccount => ({ account: '', side: 'S', amount: '', legacyRef: '' });

const emptyItem = (documentDate: string): DraftItem => ({
  contactId: null,
  amount: '',
  documentNumber: '',
  documentDate,
  dueDate: '',
  legacyRef: '',
});

function message(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

export interface OpeningBalanceDialogProps {
  open: boolean;
  /** Das erste Geschäftsjahr und sein erster Tag als Buchungstag. */
  fiscalYear: number;
  startDate: string;
  onOpenChange: (open: boolean) => void;
  /** Läuft nach der Buchung: die Abschlussansicht lädt danach neu. */
  onBooked: (preview: OpeningBalancePreview) => void | Promise<void>;
  /**
   * Der Weg zur Belegablage. Ohne ihn bliebe der Dialog eine Sackgasse, wenn
   * noch kein Beleg abgelegt ist: der Beleg der Schlussbilanz ist Pflicht, und
   * ablegen lässt er sich nur dort.
   */
  onNavigate?: NavigateFn;
}

export const OpeningBalanceDialog: React.FC<OpeningBalanceDialogProps> = ({
  open,
  fiscalYear,
  startDate,
  onOpenChange,
  onBooked,
  onNavigate,
}) => {
  const writeLock = useWriteLock();
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [contacts, setContacts] = useState<Contact[]>([]);
  const [receipts, setReceipts] = useState<Receipt[]>([]);
  const [loading, setLoading] = useState(true);

  const [legacySystem, setLegacySystem] = useState('');
  const [date, setDate] = useState(startDate);
  const [receiptId, setReceiptId] = useState<number | null>(null);
  const [lines, setLines] = useState<DraftAccount[]>([emptyAccount()]);
  const [receivables, setReceivables] = useState<DraftItem[]>([]);
  const [payables, setPayables] = useState<DraftItem[]>([]);

  const [preview, setPreview] = useState<OpeningBalancePreview | null>(null);
  const [busy, setBusy] = useState(false);
  const [booking, setBooking] = useState(false);
  const [error, setError] = useState('');

  // Konten, Geschäftspartner und Belege erst beim Öffnen: der Baustein wird
  // einmal im Leben eines Mandanten gebraucht.
  useEffect(() => {
    if (!open) return;
    let cancelled = false;
    setLoading(true);
    setError('');
    void (async () => {
      try {
        const [accountList, contactList, receiptList] = await Promise.all([
          Api.getAccounts(),
          Api.getSelectableContacts(),
          Api.getReceipts('filed'),
        ]);
        if (cancelled) return;
        setAccounts(accountList);
        setContacts(contactList);
        setReceipts(receiptList);
      } catch (e) {
        if (!cancelled) setError(message(e));
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [open]);

  useEffect(() => {
    if (open) setDate(startDate);
  }, [open, startDate]);

  const accountOptions = useMemo(
    () =>
      accounts
        .filter((account) => !account.isRange && !account.isReserved)
        .map((account) => ({
          value: account.number,
          label: `${account.number} ${account.name}`,
          meta: account.kontenklasseName,
        })),
    [accounts],
  );

  const contactItems = useMemo(
    () =>
      contacts.map((contact) => ({
        value: contact.id,
        label: `${contact.ledgerAccount} · ${contact.name}`,
      })),
    [contacts],
  );

  const receiptItems = useMemo(
    () =>
      receipts.map((receipt) => ({
        value: receipt.id,
        label: `${receipt.receiptNumber} · ${
          receipt.subject || receipt.issuerName || 'Schlussbilanz'
        }`,
      })),
    [receipts],
  );

  /**
   * Der Auftrag, so wie er ans Backend geht.
   *
   * Zeilen ohne Konto und Posten ohne Geschäftspartner fallen heraus: eine
   * halb ausgefüllte Zeile ist keine Angabe, und das Backend wiese sie mit
   * einem Satz zurück, der auf eine Zeile zeigt, die niemand gemeint hat.
   */
  function buildRequest(): OpeningBalanceRequest {
    const toItem = (item: DraftItem): OpenOpeningItem => ({
      contactId: item.contactId ?? 0,
      amount: parseCents(item.amount) ?? 0,
      documentNumber: item.documentNumber.trim(),
      documentDate: item.documentDate,
      dueDate: item.dueDate,
      legacyRef: item.legacyRef.trim(),
    });
    const accountLines: OpeningBalanceLine[] = lines
      .filter((line) => line.account.trim() !== '')
      .map((line) => ({
        account: line.account.trim(),
        side: line.side,
        amount: parseCents(line.amount) ?? 0,
        legacyRef: line.legacyRef.trim(),
      }));
    return {
      fiscalYear,
      date,
      receiptId: receiptId ?? undefined,
      legacySystem: legacySystem.trim(),
      accounts: accountLines,
      receivables: receivables.filter((item) => item.contactId !== null).map(toItem),
      payables: payables.filter((item) => item.contactId !== null).map(toItem),
    };
  }

  // Ein Betrag, den parseCents nicht lesen kann, darf nicht als 0 durchgehen.
  const unreadable = (value: string) => value.trim() !== '' && parseCents(value) === null;
  const hasUnreadable =
    lines.some((line) => unreadable(line.amount)) ||
    [...receivables, ...payables].some((item) => unreadable(item.amount));

  async function runPreview() {
    if (hasUnreadable) {
      setError('Ein Betrag ist nicht lesbar. Erwartet wird eine Zahl wie 1234,56.');
      return;
    }
    setBusy(true);
    setError('');
    try {
      setPreview(await Api.previewOpeningBalance(buildRequest()));
    } catch (e) {
      setPreview(null);
      setError(message(e));
    } finally {
      setBusy(false);
    }
  }

  async function book() {
    setBooking(true);
    setError('');
    try {
      const result = await Api.bookOpeningBalance(buildRequest());
      await onBooked(result);
    } catch (e) {
      setError(message(e));
    } finally {
      setBooking(false);
    }
  }

  const balanced = preview?.balanced ?? false;
  // Ohne abgelegten Beleg führt der Dialog nicht weiter: das Backend weist
  // Vorschau wie Buchung zurück (§ 146 Abs. 1 AO, Belegpflicht). Das gehört
  // vor die Arbeit und nicht in die Fehlermeldung danach.
  const noReceipt = !loading && receiptItems.length === 0;
  const noReceiptHint = noReceipt
    ? 'Ohne abgelegte Schlussbilanz des Altsystems lässt sich keine Eröffnungsbilanz erfassen.'
    : undefined;

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={`Eröffnungsbilanz ${fiscalYear} erfassen`}
      width="max-w-5xl"
      footer={
        <>
          <Button variant="secondary" onClick={() => onOpenChange(false)}>
            Abbrechen
          </Button>
          <Button
            variant="secondary"
            loading={busy}
            disabled={noReceipt}
            title={noReceiptHint}
            onClick={() => void runPreview()}
          >
            Vorschau
          </Button>
          <Button
            variant="primary"
            loading={booking}
            disabled={writeLock.locked || noReceipt || !balanced}
            title={
              writeLock.hint ??
              noReceiptHint ??
              (balanced
                ? undefined
                : 'Erst die Vorschau: gebucht wird nur eine Eröffnungsbilanz, die aufgeht.')
            }
            onClick={() => void book()}
          >
            Eröffnungsbilanz buchen
          </Button>
        </>
      }
    >
      {/* Die Ablehnung des Backends steht über den Aktionen und nicht als
          Toast: sie nennt die Zeile, die zu ändern ist (§11.4). */}
      {error && <Notice tone="negative" text={error} className="mb-5" />}

      {/* Der Ausweg statt der Sackgasse: das Auswahlfeld hätte sonst nur
          „Kein abgelegter Beleg vorhanden" zu bieten, und wo der Beleg
          herkommt, erführe man erst aus der Ablehnung nach der Vorschau. */}
      {noReceipt && (
        <Notice
          className="mb-5"
          text="Die Schlussbilanz des Altsystems ist noch nicht als Beleg abgelegt. Sie ist Pflicht: eine Eröffnungsbilanz ohne Beleg wäre eine Behauptung über Zahlen aus einem anderen System."
          action={
            onNavigate && (
              <Button
                variant="secondary"
                size="sm"
                onClick={() => {
                  onOpenChange(false);
                  onNavigate('receipts');
                }}
              >
                Zur Belegablage
              </Button>
            )
          }
        />
      )}

      {loading ? (
        <SkeletonRows rows={6} />
      ) : (
        <>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <Field
              label="Altsystem"
              optional
              help="Der Name des Programms, aus dem übernommen wird. Er steht später an jeder Eröffnungsbuchung."
            >
              <Input
                value={legacySystem}
                onChange={(e) => setLegacySystem(e.target.value)}
                placeholder="Vorheriges Buchhaltungsprogramm"
              />
            </Field>
            <Field label="Buchungstag" help="Voreingestellt der erste Tag des Geschäftsjahres.">
              <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
            </Field>
            <Field
              label="Beleg der Schlussbilanz"
              help="Pflicht: eine Eröffnungsbilanz ohne Beleg wäre eine Behauptung über Zahlen aus einem anderen System (§ 146 Abs. 1 AO). Die Schlussbilanz des Altsystems ist zuvor als Beleg abzulegen."
            >
              <Select<number>
                items={
                  receiptItems.length > 0
                    ? receiptItems
                    : [{ value: 0, label: 'Kein abgelegter Beleg vorhanden' }]
                }
                value={receiptId ?? 0}
                onValueChange={(value) => setReceiptId(value === 0 ? null : value)}
                aria-label="Beleg der Schlussbilanz"
              />
            </Field>
          </div>

          <div className="mt-8">
            <div className="flex items-center justify-between gap-4 mb-3">
              <span className="flex items-center text-label text-ink-muted">
                Sachkonten
                <HelpPopover label="Erklärung zu den Sachkonten">
                  Jeder Bestand der Schlussbilanz wird mit seinem Saldo erfasst: Aktivkonten im
                  Soll, Passiv- und Kapitalkonten im Haben. Die Gegenbuchung läuft über das
                  Saldenvortragskonto 9000 und ist weder Aufwand noch Ertrag.
                </HelpPopover>
              </span>
              <Button
                variant="quiet"
                size="sm"
                onClick={() => setLines((prev) => [...prev, emptyAccount()])}
              >
                Konto hinzufügen
              </Button>
            </div>

            <div className="grid grid-cols-[minmax(0,1fr)_7rem_10rem_9rem_2rem] gap-x-3 gap-y-2 items-center">
              <span className="text-caption text-ink-subtle">Konto</span>
              <span className="text-caption text-ink-subtle">Seite</span>
              <span className="text-caption text-ink-subtle text-right">Betrag</span>
              <span className="text-caption text-ink-subtle">Herkunft</span>
              <span />

              {lines.map((line, index) => (
                <React.Fragment key={index}>
                  <Combobox
                    items={accountOptions}
                    value={line.account || null}
                    onValueChange={(account) =>
                      setLines((prev) =>
                        prev.map((row, i) =>
                          i === index ? { ...row, account: account ?? '' } : row,
                        ),
                      )
                    }
                    placeholder="Konto suchen …"
                    emptyText="Kein Konto gefunden."
                  />
                  <Select<Side>
                    items={[
                      { value: 'S', label: 'Soll' },
                      { value: 'H', label: 'Haben' },
                    ]}
                    value={line.side}
                    onValueChange={(side) =>
                      setLines((prev) => prev.map((row, i) => (i === index ? { ...row, side } : row)))
                    }
                    aria-label={`Seite Zeile ${index + 1}`}
                  />
                  <Input
                    align="right"
                    inputMode="decimal"
                    value={line.amount}
                    onChange={(e) =>
                      setLines((prev) =>
                        prev.map((row, i) => (i === index ? { ...row, amount: e.target.value } : row)),
                      )
                    }
                    placeholder="0,00"
                    aria-label={`Betrag Zeile ${index + 1}`}
                  />
                  <Input
                    className="code-num"
                    value={line.legacyRef}
                    onChange={(e) =>
                      setLines((prev) =>
                        prev.map((row, i) =>
                          i === index ? { ...row, legacyRef: e.target.value } : row,
                        ),
                      )
                    }
                    placeholder="Altkennung"
                    aria-label={`Herkunftskennung Zeile ${index + 1}`}
                  />
                  {lines.length > 1 ? (
                    <Button
                      variant="quiet"
                      size="sm"
                      iconOnly
                      onClick={() => setLines((prev) => prev.filter((_, i) => i !== index))}
                      title="Zeile entfernen"
                      aria-label={`Zeile ${index + 1} entfernen`}
                    >
                      <Trash2 className="w-4 h-4" strokeWidth={1.5} />
                    </Button>
                  ) : (
                    <span />
                  )}
                </React.Fragment>
              ))}
            </div>
          </div>

          <OpenItemDraftTable
            title="Offene Forderungen"
            help="Jede Rechnung, die beim Umstieg noch offen war, einzeln — mit Nummer und Fälligkeit. Nur so lässt sie sich später ausgleichen."
            items={receivables}
            contactItems={contactItems}
            unreadable={unreadable}
            onChange={setReceivables}
            onAdd={() => setReceivables((prev) => [...prev, emptyItem(date)])}
          />

          <OpenItemDraftTable
            title="Offene Verbindlichkeiten"
            help="Die Eingangsrechnungen, die beim Umstieg noch nicht bezahlt waren."
            items={payables}
            contactItems={contactItems}
            unreadable={unreadable}
            onChange={setPayables}
            onAdd={() => setPayables((prev) => [...prev, emptyItem(date)])}
          />

          {preview && (
            <div className="mt-8 pt-6 border-t border-line">
              <div className="flex items-center justify-between gap-4 text-body">
                <span
                  className={cn(
                    'font-semibold',
                    preview.balanced ? 'text-positive-text' : 'text-ink-muted',
                  )}
                >
                  {preview.balanced
                    ? `${preview.entries.length} Eröffnungsbuchungen, Soll und Haben stimmen überein`
                    : 'Die Eröffnungsbilanz geht noch nicht auf'}
                </span>
                <span className="num text-ink">
                  Soll {formatCents(preview.debitTotal)} · Haben {formatCents(preview.creditTotal)}
                </span>
              </div>

              {preview.messages.length > 0 && (
                <ul className="mt-3 flex flex-col gap-1">
                  {preview.messages.map((note) => (
                    <li key={note} className="text-caption text-ink-muted">
                      {note}
                    </li>
                  ))}
                </ul>
              )}

              {preview.entries.length > 0 && (
                <Table density="kompakt" className="mt-4">
                  <Thead>
                    <Tr>
                      <Th className="w-28">Datum</Th>
                      <Th>Buchungstext</Th>
                      <Th className="w-28">Herkunft</Th>
                      <Th numeric className="w-36">
                        Betrag
                      </Th>
                    </Tr>
                  </Thead>
                  <Tbody>
                    {preview.entries.map((entry, index) => (
                      <Tr key={`${entry.entryNumber}-${index}`}>
                        <Td className="num text-ink-subtle">{entry.bookingDate}</Td>
                        <Td className="whitespace-normal">{entry.description}</Td>
                        <Td className="code-num text-ink-subtle">{entry.legacyRef || '—'}</Td>
                        <Td numeric className="num">
                          {formatCents(
                            entry.lines
                              .filter((line) => line.side === 'S')
                              .reduce((sum, line) => sum + line.amount, 0),
                          )}
                        </Td>
                      </Tr>
                    ))}
                  </Tbody>
                </Table>
              )}
            </div>
          )}
        </>
      )}
    </Dialog>
  );
};

/** Die offenen Posten einer Seite als Erfassungstabelle. */
const OpenItemDraftTable: React.FC<{
  title: string;
  help: string;
  items: DraftItem[];
  contactItems: { value: number; label: string }[];
  unreadable: (value: string) => boolean;
  onChange: (items: DraftItem[]) => void;
  onAdd: () => void;
}> = ({ title, help, items, contactItems, unreadable, onChange, onAdd }) => {
  function patch(index: number, next: Partial<DraftItem>) {
    onChange(items.map((item, i) => (i === index ? { ...item, ...next } : item)));
  }

  return (
    <div className="mt-8">
      <div className="flex items-center justify-between gap-4 mb-3">
        <span className="flex items-center text-label text-ink-muted">
          {title}
          <HelpPopover label={`Erklärung zu ${title}`}>{help}</HelpPopover>
        </span>
        <Button variant="quiet" size="sm" onClick={onAdd}>
          Posten hinzufügen
        </Button>
      </div>

      {items.length === 0 ? (
        <p className="text-caption text-ink-subtle">Keine offenen Posten übernommen.</p>
      ) : (
        <div className="grid grid-cols-[minmax(0,1fr)_9rem_8rem_8rem_8rem_2rem] gap-x-3 gap-y-2 items-center">
          <span className="text-caption text-ink-subtle">Geschäftspartner</span>
          <span className="text-caption text-ink-subtle text-right">Betrag</span>
          <span className="text-caption text-ink-subtle">Belegnummer</span>
          <span className="text-caption text-ink-subtle">Belegdatum</span>
          <span className="text-caption text-ink-subtle">Fällig am</span>
          <span />

          {items.map((item, index) => (
            <React.Fragment key={index}>
              <Select<number>
                items={
                  contactItems.length > 0
                    ? contactItems
                    : [{ value: 0, label: 'Kein Geschäftspartner angelegt' }]
                }
                value={item.contactId ?? 0}
                onValueChange={(value) => patch(index, { contactId: value === 0 ? null : value })}
                aria-label={`Geschäftspartner Posten ${index + 1}`}
              />
              <Input
                align="right"
                inputMode="decimal"
                value={item.amount}
                onChange={(e) => patch(index, { amount: e.target.value })}
                placeholder="0,00"
                aria-invalid={unreadable(item.amount) || undefined}
                aria-label={`Betrag Posten ${index + 1}`}
              />
              <Input
                className="code-num"
                value={item.documentNumber}
                onChange={(e) => patch(index, { documentNumber: e.target.value })}
                placeholder="RE-0042"
                aria-label={`Belegnummer Posten ${index + 1}`}
              />
              <Input
                type="date"
                value={item.documentDate}
                onChange={(e) => patch(index, { documentDate: e.target.value })}
                aria-label={`Belegdatum Posten ${index + 1}`}
              />
              <Input
                type="date"
                value={item.dueDate}
                onChange={(e) => patch(index, { dueDate: e.target.value })}
                aria-label={`Fälligkeit Posten ${index + 1}`}
              />
              <Button
                variant="quiet"
                size="sm"
                iconOnly
                onClick={() => onChange(items.filter((_, i) => i !== index))}
                title="Posten entfernen"
                aria-label={`Posten ${index + 1} entfernen`}
              >
                <Trash2 className="w-4 h-4" strokeWidth={1.5} />
              </Button>
            </React.Fragment>
          ))}
        </div>
      )}
    </div>
  );
};

export default OpeningBalanceDialog;
