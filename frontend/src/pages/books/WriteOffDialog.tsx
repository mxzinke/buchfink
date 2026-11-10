import React, { useEffect, useState } from 'react';
import type { JournalEntry, OpenItem, Receipt } from '../../types';
import { Api } from '../../services/api';
import { Button, Dialog, Field, Input, Notice, Select } from '../../components/ui';
import { usePostingLock } from '../../components/WriteLock';
import { formatCents, parseCents } from '../../utils/formatters';
import { PostingLines, errorText, todayISO } from './shared';

export const WriteOffDialog: React.FC<{
  item: OpenItem;
  onClose: () => void;
  onBooked: () => void;
}> = ({ item, onClose, onBooked }) => {
  const lock = usePostingLock();
  const [date, setDate] = useState(todayISO());
  const [amount, setAmount] = useState((item.openAmount / 100).toFixed(2));
  const [reason, setReason] = useState('');
  const [receipt, setReceipt] = useState('');
  const [receipts, setReceipts] = useState<Receipt[]>([]);
  const [preview, setPreview] = useState<JournalEntry | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    void Api.getReceipts().then(setReceipts).catch((e) => setError(errorText(e)));
  }, []);

  const change = (setter: (value: string) => void) => (e: React.ChangeEvent<HTMLInputElement>) => {
    setter(e.target.value);
    setPreview(null);
  };

  const submit = async (onlyPreview: boolean) => {
    setBusy(true);
    setError('');
    try {
      const cents = parseCents(amount);
      if (cents == null || cents <= 0) throw new Error('Bitte einen positiven Bruttobetrag eingeben.');
      const result = await Api.writeOffOpenItem(
        { openItemEntryId: item.entryId, date, amount: cents, reason, receiptId: Number(receipt) },
        onlyPreview,
      );
      if (onlyPreview) setPreview(result);
      else onBooked();
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Dialog
      open
      title="Forderung ausbuchen"
      onOpenChange={(open) => !open && onClose()}
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>Abbrechen</Button>
          <Button
            variant="primary"
            disabled={lock.locked || busy || !reason.trim() || !receipt || !date}
            loading={busy}
            onClick={() => void submit(!preview)}
          >
            {preview ? 'Ausbuchen' : 'Vorschau'}
          </Button>
        </>
      }
    >
      <p>{item.documentNumber} · {item.contactName} · {formatCents(item.openAmount)} offen</p>
      <Field label="Uneinbringlich seit">
        <Input disabled={busy} type="date" value={date} onChange={change(setDate)} />
      </Field>
      <Field label="Bruttobetrag in Euro">
        <Input disabled={busy} value={amount} onChange={change(setAmount)} />
      </Field>
      <Field label="Grund">
        <Input disabled={busy} value={reason} onChange={change(setReason)} />
      </Field>
      <Field label="Nachweis der Uneinbringlichkeit">
        <Select
          disabled={busy}
          value={receipt}
          onValueChange={(value) => { setReceipt(value); setPreview(null); }}
          items={receipts.filter((r) => r.status !== 'discarded').map((r) => ({
            value: String(r.id), label: `${r.receiptNumber} · ${r.subject}`,
          }))}
        />
      </Field>
      {error && <Notice tone="negative" text={error} />}
      {Boolean(item.specificAllowance) && <p>Einzelwertberichtigung: {formatCents(item.specificAllowance ?? 0)}</p>}
      {item.allowanceUnassigned && (
        <Notice tone="negative" text="Es bestehen Einzelwertberichtigungen ohne Zuordnung. Bitte zuerst fachlich klären." />
      )}
      {preview && <PostingLines lines={preview.lines} />}
    </Dialog>
  );
};
