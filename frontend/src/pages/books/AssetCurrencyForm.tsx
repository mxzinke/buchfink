import { useState } from 'react';
import type { AcquisitionCandidate, FixedAsset, JournalEntry } from '../../types';
import { Api } from '../../services/api';
import { useWorkspace } from '../../workspace';
import { Button, Field, Input, Notice, Select } from '../../components/ui';
import { formatCents } from '../../utils/formatters';
import { errorText } from './shared';

export function AssetCurrencyForm({ asset, onSaved }: { asset: FixedAsset; onSaved: () => Promise<void> }) {
  const { year } = useWorkspace();
  const [date, setDate] = useState(`${year}-12-31`);
  const [rate, setRate] = useState('');
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
          const entry = await Api.bookAssetCurrency({
            assetId: asset.id,
            date,
            ratePerEuro: rate ? Math.round(Number(rate) * 1000000) : 0,
            preview: !preview,
          });
          if (preview) {
            setPreview(null);
            await onSaved();
          } else setPreview(entry);
        } catch (e) {
          setError(errorText(e));
        } finally {
          setBusy(false);
        }
      }}
    >
      <Field label="Bewertungsdatum">
        <Input required type="date" value={date} disabled={busy} onChange={(e) => setDate(e.target.value)} />
      </Field>
      <Field label={`${asset.currency} je Euro`} optional hint="Ohne Angabe wird der Tageskurs abgerufen">
        <Input
          type="number"
          step="0.000001"
          min="0.000001"
          value={rate}
          disabled={busy}
          onChange={(e) => setRate(e.target.value)}
        />
      </Field>
      {preview?.lines.map((line, i) => (
        <p key={i} className="text-caption">
          {line.side} {line.account}: {formatCents(line.amount)}
        </p>
      ))}
      {error && <Notice tone="negative" text={error} />}
      <Button type="submit" disabled={busy}>
        {preview ? 'Kursbewertung buchen' : 'Kursbewertung prüfen'}
      </Button>
    </form>
  );
}

export function ForeignAssetAcquisitionForm({
  candidates,
  onSaved,
}: {
  candidates: AcquisitionCandidate[];
  onSaved: () => Promise<void>;
}) {
  const [key, setKey] = useState('');
  const [name, setName] = useState('');
  const [currency, setCurrency] = useState('USD');
  const [cost, setCost] = useState('');
  const [rate, setRate] = useState('');
  const [source, setSource] = useState('');
  const [quantity, setQuantity] = useState('');
  const [maturity, setMaturity] = useState('');
  const [preview, setPreview] = useState<FixedAsset | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const selected = candidates.find((c) => `${c.entryId}:${c.account}` === key);
  return (
    <form
      className="space-y-4"
      onChange={() => setPreview(null)}
      onSubmit={async (event) => {
        event.preventDefault();
        if (busy || !selected) return;
        setBusy(true);
        setError(null);
        try {
          const asset = await Api.recordForeignAsset({
            entryId: selected.entryId,
            account: selected.account,
            date: selected.bookingDate,
            name,
            currency,
            foreignCost: Math.round(Number(cost) * 100),
            quantity: Math.round(Number(quantity) * 10000),
            ratePerEuro: rate ? Math.round(Number(rate) * 1000000) : 0,
            maturityDate: maturity,
            rateSource: source,
            preview: !preview,
          });
          if (preview) {
            setPreview(null);
            await onSaved();
          } else setPreview(asset);
        } catch (e) {
          setError(errorText(e));
        } finally {
          setBusy(false);
        }
      }}
    >
      <Field label="Gebuchter Zugang">
        <Select
          value={key}
          disabled={busy}
          items={candidates
            .filter((c) => c.account.startsWith('08') || c.account.startsWith('09'))
            .map((c) => ({
              value: `${c.entryId}:${c.account}`,
              label: `${c.entryNumber} · ${c.accountName} · ${formatCents(c.amount)}`,
            }))}
          onValueChange={(value) => {
            setKey(value);
            setPreview(null);
          }}
        />
      </Field>
      <Field label="Bezeichnung">
        <Input required disabled={busy} value={name} onChange={(e) => setName(e.target.value)} />
      </Field>
      <Field label="Währung">
        <Input
          required
          maxLength={3}
          disabled={busy}
          value={currency}
          onChange={(e) => setCurrency(e.target.value.toUpperCase())}
        />
      </Field>
      <Field label="Anschaffungskosten in Fremdwährung">
        <Input
          required
          type="number"
          min="0.01"
          step="0.01"
          disabled={busy}
          value={cost}
          onChange={(e) => setCost(e.target.value)}
        />
      </Field>
      <Field label="Fälligkeit bei Ausleihungen" optional>
        <Input type="date" disabled={busy} value={maturity} onChange={(e) => setMaturity(e.target.value)} />
      </Field>
      <Field label="Stückzahl" optional>
        <Input
          type="number"
          min="0"
          step="0.0001"
          disabled={busy}
          value={quantity}
          onChange={(e) => setQuantity(e.target.value)}
        />
      </Field>
      <Field label="Fremdwährung je Euro" optional hint="Ohne Angabe wird der Tageskurs abgerufen">
        <Input
          type="number"
          min="0.000001"
          step="0.000001"
          disabled={busy}
          value={rate}
          onChange={(e) => setRate(e.target.value)}
        />
      </Field>
      {rate && (
        <Field label="Kursquelle">
          <Input required disabled={busy} value={source} onChange={(e) => setSource(e.target.value)} />
        </Field>
      )}
      {preview && <p className="text-caption">Anschaffungskosten: {formatCents(preview.acquisitionCost)}</p>}
      {error && <Notice tone="negative" text={error} />}
      <Button type="submit" disabled={busy || !selected}>
        {preview ? 'Zugang aufnehmen' : 'Zugang prüfen'}
      </Button>
    </form>
  );
}
