import { useState } from 'react';
import type { FixedAsset } from '../../types';
import { Api } from '../../services/api';
import { Button, Field, Input, Notice, Select } from '../../components/ui';
import { errorText } from './shared';

export function AssetTransferForm({ asset, onSaved }: { asset: FixedAsset; onSaved: () => Promise<void> }) {
  const [date, setDate] = useState('');
  const [account, setAccount] = useState('');
  const [expense, setExpense] = useState(asset.class === 'intangible' ? '6200' : '6220');
  const [method, setMethod] = useState('linear');
  const [months, setMonths] = useState('');
  const [note, setNote] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  return (
    <form
      className="space-y-4"
      onSubmit={async (event) => {
        event.preventDefault();
        if (busy) return;
        setBusy(true);
        setError(null);
        try {
          await Api.transferAsset({
            assetId: asset.id,
            date,
            account,
            depreciationAccount: expense,
            method,
            usefulLifeMonths: Number(months),
            note,
          });
          await onSaved();
        } catch (e) {
          setError(errorText(e));
        } finally {
          setBusy(false);
        }
      }}
    >
      <Field label="Fertigstellung am">
        <Input type="date" required disabled={busy} value={date} onChange={(e) => setDate(e.target.value)} />
      </Field>
      <Field label="Anlagekonto">
        <Input required disabled={busy} value={account} onChange={(e) => setAccount(e.target.value)} />
      </Field>
      <Field label="Abschreibungskonto">
        <Input required disabled={busy} value={expense} onChange={(e) => setExpense(e.target.value)} />
      </Field>
      <Field label="Methode">
        <Select
          value={method}
          disabled={busy}
          onValueChange={setMethod}
          items={[
            { value: 'linear', label: 'Linear' },
            { value: 'degressive', label: 'Degressiv' },
            { value: 'none', label: 'Keine planmäßige Abschreibung' },
          ]}
        />
      </Field>
      <Field label="Nutzungsdauer in Monaten">
        <Input
          type="number"
          min="0"
          required
          disabled={busy}
          value={months}
          onChange={(e) => setMonths(e.target.value)}
        />
      </Field>
      <Field label="Grund">
        <Input required disabled={busy} value={note} onChange={(e) => setNote(e.target.value)} />
      </Field>
      {error && <Notice tone="negative" text={error} />}
      <Button type="submit" disabled={busy}>
        Fertigstellung buchen
      </Button>
    </form>
  );
}
