import { useState } from 'react';
import type { FixedAsset, JournalEntry } from '../../types';
import { Api } from '../../services/api';
import { useWorkspace } from '../../workspace';
import { Button, Field, Input, Notice, Select } from '../../components/ui';
import { formatCents } from '../../utils/formatters';
import { errorText } from './shared';

export function AssetValuationForm({ asset, onSaved }: { asset: FixedAsset; onSaved: () => Promise<void> }) {
  const { year } = useWorkspace();
  const [kind, setKind] = useState('impairment');
  const [date, setDate] = useState(`${year}-12-31`);
  const [amount, setAmount] = useState('');
  const [permanent, setPermanent] = useState('yes');
  const [reason, setReason] = useState('');
  const [evidence, setEvidence] = useState('');
  const [preview, setPreview] = useState<JournalEntry | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  return (
    <form
      className="space-y-4"
      onChange={() => setPreview(null)}
      onSubmit={async (event) => {
        event.preventDefault();
        if (busy) return;
        setBusy(true);
        setError(null);
        try {
          if (kind === 'confirm') {
            await Api.confirmAssetImpairment(asset.id, year, reason, evidence);
            await onSaved();
          } else {
            const req = {
              assetId: asset.id,
              date,
              amount: Math.round(Number(amount) * 100),
              reason,
              evidence,
              permanent: permanent === 'yes',
              preview: !preview,
            };
            const entry =
              kind === 'impairment' ? await Api.bookAssetImpairment(req) : await Api.bookAssetWriteUp(req);
            if (preview) {
              setPreview(null);
              await onSaved();
            } else setPreview(entry);
          }
        } catch (e) {
          setError(errorText(e));
        } finally {
          setBusy(false);
        }
      }}
    >
      <Field label="Vorgang">
        <Select
          disabled={busy}
          value={kind}
          onValueChange={(value) => {
            setKind(value);
            setPreview(null);
          }}
          items={[
            { value: 'impairment', label: 'Außerplanmäßige Abschreibung' },
            { value: 'writeup', label: 'Zuschreibung' },
            { value: 'confirm', label: 'Wertminderung besteht fort' },
          ]}
        />
      </Field>
      {kind !== 'confirm' && (
        <>
          <Field label="Bewertungsdatum">
            <Input
              type="date"
              required
              disabled={busy}
              value={date}
              onChange={(e) => setDate(e.target.value)}
            />
          </Field>
          <Field label="Betrag in Euro">
            <Input
              type="number"
              min="0.01"
              step="0.01"
              required
              disabled={busy}
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
            />
          </Field>
        </>
      )}
      {kind === 'impairment' && (
        <Field label="Voraussichtlich dauernde Wertminderung">
          <Select
            disabled={busy}
            value={permanent}
            onValueChange={(value) => {
              setPermanent(value);
              setPreview(null);
            }}
            items={[
              { value: 'yes', label: 'Ja' },
              { value: 'no', label: 'Nein' },
            ]}
          />
        </Field>
      )}
      <Field label="Grund">
        <Input required disabled={busy} value={reason} onChange={(e) => setReason(e.target.value)} />
      </Field>
      <Field label="Nachweis" hint="Zum Beispiel Gutachten mit Datum und Fundstelle">
        <Input required disabled={busy} value={evidence} onChange={(e) => setEvidence(e.target.value)} />
      </Field>
      {preview?.lines.map((line, i) => (
        <p key={i} className="text-caption">
          {line.side} {line.account}: {formatCents(line.amount)}
        </p>
      ))}
      {error && <Notice tone="negative" text={error} />}
      <Button type="submit" disabled={busy}>
        {kind === 'confirm' ? 'Beibehaltung bestätigen' : preview ? 'Bewertung buchen' : 'Bewertung prüfen'}
      </Button>
    </form>
  );
}
