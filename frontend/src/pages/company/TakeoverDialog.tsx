import React, { useEffect, useState } from 'react';
import { Plus, Trash2 } from 'lucide-react';
import { Api } from '../../services/api';
import type { Contact, DepreciationMethod, Receipt, Side, TakeoverPreview } from '../../types';
import { useWriteLock } from '../../components/WriteLock';
import { formatCents, formatDate, parseCents } from '../../utils/formatters';
import {
  buildTakeoverRequest,
  takeoverTotals,
  type TakeoverAccountDraft,
  type TakeoverAssetDraft,
  type TakeoverDraft,
  type TakeoverItemDraft,
  type TakeoverTax,
} from '../../utils/takeover';
import {
  Button,
  Dialog,
  Field,
  FormGrid,
  Input,
  Notice,
  Select,
  Table,
  Tabs,
  TabPanel,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
  toast,
} from '../../components/ui';

type Step = 'accounts' | 'receivables' | 'payables' | 'assets';

const SIDES: { value: Side; label: string }[] = [
  { value: 'S', label: 'Soll' },
  { value: 'H', label: 'Haben' },
];

const TAXES: { value: TakeoverTax; label: string }[] = [
  { value: '', label: 'unbekannt' },
  { value: '19', label: '19 %' },
  { value: '7', label: '7 %' },
  { value: '0', label: 'nicht steuerbar' },
];

const METHODS: { value: DepreciationMethod | ''; label: string }[] = [
  { value: '', label: 'nach Anlagekonto' },
  { value: 'linear', label: 'Linear' },
  { value: 'degressive', label: 'Degressiv' },
  { value: 'building_linear', label: 'Gebäude, fester Satz' },
  { value: 'electric_vehicle', label: 'Staffel für Elektrofahrzeuge' },
  { value: 'pool', label: 'Sammelposten' },
  { value: 'none', label: 'Keine planmäßige Abschreibung' },
];

const emptyAccount = (): TakeoverAccountDraft => ({ account: '', side: 'S', amount: '' });
const emptyItem = (): TakeoverItemDraft => ({
  contactId: 0,
  documentNumber: '',
  documentDate: '',
  dueDate: '',
  amount: '',
  tax: '',
});
const emptyAsset = (): TakeoverAssetDraft => ({
  name: '',
  account: '',
  acquisitionDate: '',
  acquisitionCost: '',
  accumulatedDepreciation: '',
  usefulLifeMonths: '',
  method: '',
});

function replaceAt<T>(rows: T[], index: number, patch: Partial<T>): T[] {
  return rows.map((row, i) => (i === index ? { ...row, ...patch } : row));
}

const RemoveRow: React.FC<{ label: string; onClick: () => void }> = ({ label, onClick }) => (
  <Button variant="quiet" size="sm" iconOnly title="Zeile entfernen" aria-label={label} onClick={onClick}>
    <Trash2 className="w-4 h-4" strokeWidth={1.5} />
  </Button>
);

const AddRow: React.FC<{ label: string; onClick: () => void }> = ({ label, onClick }) => (
  <Button
    variant="quiet"
    size="sm"
    className="-ml-2.5 mt-3"
    icon={<Plus className="w-4 h-4" strokeWidth={1.5} />}
    onClick={onClick}
  >
    {label}
  </Button>
);

/** Die Zeilen der offenen Posten einer Seite. */
const ItemRows: React.FC<{
  label: string;
  rows: TakeoverItemDraft[];
  contacts: Contact[];
  onChange: (rows: TakeoverItemDraft[]) => void;
}> = ({ label, rows, contacts, onChange }) => (
  <>
    <ul className="flex flex-col gap-2">
      {rows.map((row, index) => (
        <li
          key={index}
          className="grid grid-cols-[minmax(0,1.4fr)_minmax(0,1fr)_9.5rem_9.5rem_8rem_9rem_2.25rem] gap-2 items-center"
        >
          <Select
            aria-label={`${label} ${index + 1}: Kontakt`}
            placeholder="Kontakt"
            items={contacts.map((c) => ({ value: c.id, label: c.company || c.name }))}
            value={row.contactId || undefined}
            onValueChange={(id) => onChange(replaceAt(rows, index, { contactId: Number(id) }))}
          />
          <Input
            aria-label={`${label} ${index + 1}: Rechnungsnummer`}
            placeholder="Rechnungsnummer"
            value={row.documentNumber}
            onChange={(e) => onChange(replaceAt(rows, index, { documentNumber: e.target.value }))}
          />
          <Input
            type="date"
            aria-label={`${label} ${index + 1}: Rechnungsdatum`}
            title="Rechnungsdatum"
            value={row.documentDate}
            onChange={(e) => onChange(replaceAt(rows, index, { documentDate: e.target.value }))}
          />
          <Input
            type="date"
            aria-label={`${label} ${index + 1}: fällig am`}
            title="Fällig am"
            value={row.dueDate}
            onChange={(e) => onChange(replaceAt(rows, index, { dueDate: e.target.value }))}
          />
          <Input
            align="right"
            aria-label={`${label} ${index + 1}: offener Betrag`}
            placeholder="0,00"
            value={row.amount}
            onChange={(e) => onChange(replaceAt(rows, index, { amount: e.target.value }))}
          />
          <Select
            aria-label={`${label} ${index + 1}: Umsatzsteuer`}
            items={TAXES}
            value={row.tax}
            onValueChange={(tax) => onChange(replaceAt(rows, index, { tax }))}
          />
          <RemoveRow
            label={`${label} ${index + 1} entfernen`}
            onClick={() => onChange(rows.filter((_, i) => i !== index))}
          />
        </li>
      ))}
    </ul>
    <AddRow label="Offenen Posten hinzufügen" onClick={() => onChange([...rows, emptyItem()])} />
  </>
);

/**
 * Die Eröffnung eines Umsteigers: Bestände der Schlussbilanz, offene Posten je
 * Rechnung und Anlagen je Anlagegut. Gebucht wird nach der Vorschau, und nur,
 * wenn sie keine Differenz nennt (TakeoverService.Book).
 */
export const TakeoverDialog: React.FC<{
  date: string;
  onClose: () => void;
  onBooked: () => void;
}> = ({ date, onClose, onBooked }) => {
  const writeLock = useWriteLock();
  const [receipts, setReceipts] = useState<Receipt[]>([]);
  const [contacts, setContacts] = useState<Contact[]>([]);
  const [step, setStep] = useState<Step>('accounts');
  const [draft, setDraft] = useState<TakeoverDraft>({
    receiptId: 0,
    legacySystem: '',
    accounts: [emptyAccount(), emptyAccount()],
    receivables: [],
    payables: [],
    assets: [],
  });
  const [preview, setPreview] = useState<TakeoverPreview | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    Api.getReceipts('filed')
      .then(setReceipts)
      .catch(() => setReceipts([]));
    Api.getSelectableContacts()
      .then(setContacts)
      .catch(() => setContacts([]));
  }, []);

  const change = (patch: Partial<TakeoverDraft>) => {
    setDraft((prev) => ({ ...prev, ...patch }));
    setPreview(null);
  };
  const totals = takeoverTotals(draft.accounts, parseCents);
  const customers = contacts.filter((c) => c.type === 'customer');
  const vendors = contacts.filter((c) => c.type === 'vendor');

  async function run(book: boolean) {
    const built = buildTakeoverRequest(draft, parseCents);
    if (built.error !== undefined) return setError(built.error);
    setBusy(true);
    setError('');
    try {
      const result = await Api.takeover(built.request, !book);
      if (book) {
        toast.success('Eröffnung gebucht.');
        onBooked();
      } else {
        setPreview(result);
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  const bookable = preview !== null && preview.differences.length === 0;

  return (
    <Dialog
      open
      onOpenChange={(open) => !open && onClose()}
      title={`Eröffnung zum ${formatDate(date)} erfassen`}
      width="max-w-6xl"
      dirty
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Abbrechen
          </Button>
          <Button variant="secondary" loading={busy && !bookable} onClick={() => void run(false)}>
            Vorschau
          </Button>
          <Button
            variant="primary"
            loading={busy && bookable}
            disabled={!bookable || writeLock.locked}
            title={writeLock.hint || (bookable ? undefined : 'Erst die Vorschau ohne Differenz')}
            onClick={() => void run(true)}
          >
            Eröffnung buchen
          </Button>
        </>
      }
    >
      <FormGrid>
        <Field
          label="Schlussbilanz des Vorsystems"
          hint="Abgelegter Beleg"
          helpSummary="Die Schlussbilanz oder Summen- und Saldenliste belegt die übernommenen Bestände."
          explain="Legen Sie die Schlussbilanz oder die Summen- und Saldenliste des Vorsystems zuerst unter Belege ab. Buchfink verbindet sie mit jeder Eröffnungsbuchung."
        >
          <Select
            placeholder="Beleg wählen"
            items={receipts.map((r) => ({
              value: r.id,
              label: [r.receiptNumber, r.subject || r.issuerName, r.documentDate && formatDate(r.documentDate)]
                .filter(Boolean)
                .join(' · '),
            }))}
            value={draft.receiptId || undefined}
            onValueChange={(id) => change({ receiptId: Number(id) })}
          />
        </Field>
        <Field label="Vorsystem" optional>
          <Input value={draft.legacySystem} onChange={(e) => change({ legacySystem: e.target.value })} />
        </Field>
      </FormGrid>

      <Tabs<Step>
        className="mt-6"
        value={step}
        onValueChange={setStep}
        items={[
          { value: 'accounts', label: 'Bestände', count: draft.accounts.length },
          { value: 'receivables', label: 'Forderungen', count: draft.receivables.length },
          { value: 'payables', label: 'Verbindlichkeiten', count: draft.payables.length },
          { value: 'assets', label: 'Anlagen', count: draft.assets.length },
        ]}
      >
        <TabPanel value="accounts">
          <ul className="flex flex-col gap-2">
            {draft.accounts.map((row, index) => (
              <li key={index} className="grid grid-cols-[8rem_9rem_10rem_2.25rem] gap-2 items-center">
                <Input
                  className="code-num"
                  aria-label={`Bestand ${index + 1}: Konto`}
                  placeholder="Konto"
                  value={row.account}
                  onChange={(e) => change({ accounts: replaceAt(draft.accounts, index, { account: e.target.value }) })}
                />
                <Select
                  aria-label={`Bestand ${index + 1}: Seite`}
                  items={SIDES}
                  value={row.side}
                  onValueChange={(side) => change({ accounts: replaceAt(draft.accounts, index, { side }) })}
                />
                <Input
                  align="right"
                  aria-label={`Bestand ${index + 1}: Betrag`}
                  placeholder="0,00"
                  value={row.amount}
                  onChange={(e) => change({ accounts: replaceAt(draft.accounts, index, { amount: e.target.value }) })}
                />
                <RemoveRow
                  label={`Bestand ${index + 1} entfernen`}
                  onClick={() => change({ accounts: draft.accounts.filter((_, i) => i !== index) })}
                />
              </li>
            ))}
          </ul>
          <div className="flex flex-wrap items-center justify-between gap-3">
            <AddRow label="Konto hinzufügen" onClick={() => change({ accounts: [...draft.accounts, emptyAccount()] })} />
            <p className="text-caption text-ink-subtle mt-3 num">
              Soll {formatCents(totals.debit)} · Haben {formatCents(totals.credit)}
            </p>
          </div>
        </TabPanel>
        <TabPanel value="receivables">
          <ItemRows
            label="Forderung"
            rows={draft.receivables}
            contacts={customers}
            onChange={(receivables) => change({ receivables })}
          />
        </TabPanel>
        <TabPanel value="payables">
          <ItemRows
            label="Verbindlichkeit"
            rows={draft.payables}
            contacts={vendors}
            onChange={(payables) => change({ payables })}
          />
        </TabPanel>
        <TabPanel value="assets">
          <ul className="flex flex-col gap-2">
            {draft.assets.map((row, index) => (
              <li
                key={index}
                className="grid grid-cols-[minmax(0,1.4fr)_6rem_9.5rem_8rem_8rem_6rem_minmax(0,1fr)_2.25rem] gap-2 items-center"
              >
                <Input
                  aria-label={`Anlage ${index + 1}: Bezeichnung`}
                  placeholder="Bezeichnung"
                  value={row.name}
                  onChange={(e) => change({ assets: replaceAt(draft.assets, index, { name: e.target.value }) })}
                />
                <Input
                  className="code-num"
                  aria-label={`Anlage ${index + 1}: Anlagekonto`}
                  placeholder="Konto"
                  value={row.account}
                  onChange={(e) => change({ assets: replaceAt(draft.assets, index, { account: e.target.value }) })}
                />
                <Input
                  type="date"
                  aria-label={`Anlage ${index + 1}: Anschaffungsdatum`}
                  title="Anschaffungsdatum"
                  value={row.acquisitionDate}
                  onChange={(e) =>
                    change({ assets: replaceAt(draft.assets, index, { acquisitionDate: e.target.value }) })
                  }
                />
                <Input
                  align="right"
                  aria-label={`Anlage ${index + 1}: Anschaffungskosten`}
                  title="Historische Anschaffungskosten"
                  placeholder="Anschaffung"
                  value={row.acquisitionCost}
                  onChange={(e) =>
                    change({ assets: replaceAt(draft.assets, index, { acquisitionCost: e.target.value }) })
                  }
                />
                <Input
                  align="right"
                  aria-label={`Anlage ${index + 1}: bisherige Abschreibung`}
                  title="Bis zum Umstieg gebuchte Abschreibung"
                  placeholder="AfA bisher"
                  value={row.accumulatedDepreciation}
                  onChange={(e) =>
                    change({ assets: replaceAt(draft.assets, index, { accumulatedDepreciation: e.target.value }) })
                  }
                />
                <Input
                  align="right"
                  aria-label={`Anlage ${index + 1}: Nutzungsdauer in Monaten`}
                  title="Nutzungsdauer in Monaten ab Anschaffung"
                  placeholder="Monate"
                  value={row.usefulLifeMonths}
                  onChange={(e) =>
                    change({ assets: replaceAt(draft.assets, index, { usefulLifeMonths: e.target.value }) })
                  }
                />
                <Select
                  aria-label={`Anlage ${index + 1}: Abschreibung`}
                  items={METHODS}
                  value={row.method}
                  onValueChange={(method) => change({ assets: replaceAt(draft.assets, index, { method }) })}
                />
                <RemoveRow
                  label={`Anlage ${index + 1} entfernen`}
                  onClick={() => change({ assets: draft.assets.filter((_, i) => i !== index) })}
                />
              </li>
            ))}
          </ul>
          <AddRow label="Anlage hinzufügen" onClick={() => change({ assets: [...draft.assets, emptyAsset()] })} />
        </TabPanel>
      </Tabs>

      {error && <Notice tone="negative" className="mt-5" text={error} />}

      {preview && (
        <div className="mt-6 pt-6 border-t border-line">
          <h3 className="text-label text-ink">Vorschau</h3>
          <p className="text-caption text-ink-subtle mt-0.5 num">
            {preview.entries.length} Buchungen · Aktiva {formatCents(preview.debitTotal)} · Passiva{' '}
            {formatCents(preview.creditTotal)}
          </p>
          {preview.differences.map((difference) => (
            <Notice key={difference} tone="negative" className="mt-3" text={difference} />
          ))}
          {preview.assets.length > 0 && (
            <div className="mt-4">
              <Table density="kompakt">
                <Thead>
                  <Tr>
                    <Th>Anlage</Th>
                    <Th className="w-24">Konto</Th>
                    <Th numeric className="w-36">Anschaffung</Th>
                    <Th numeric className="w-36">AfA bisher</Th>
                    <Th numeric className="w-36">Buchwert</Th>
                    <Th numeric className="w-40">AfA im ersten Jahr</Th>
                  </Tr>
                </Thead>
                <Tbody>
                  {preview.assets.map((asset, index) => (
                    <Tr key={index}>
                      <Td>{asset.name}</Td>
                      <Td className="code-num">{asset.account}</Td>
                      <Td numeric>{formatCents(asset.acquisitionCost)}</Td>
                      <Td numeric>{formatCents(asset.accumulatedDepreciation)}</Td>
                      <Td numeric>{formatCents(asset.bookValue)}</Td>
                      <Td numeric>{formatCents(asset.firstYearDepreciation)}</Td>
                    </Tr>
                  ))}
                </Tbody>
              </Table>
            </div>
          )}
        </div>
      )}
    </Dialog>
  );
};
