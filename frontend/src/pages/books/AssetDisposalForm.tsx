import { useEffect, useState } from 'react';
import type { AssetDisposalPreview, Contact, FixedAsset } from '../../types';
import { Api } from '../../services/api';
import { Button, Field, Input, Notice, Select } from '../../components/ui';
import { formatCents } from '../../utils/formatters';
import { errorText } from './shared';

export function AssetDisposalForm({ asset, onSaved }: { asset: FixedAsset; onSaved: () => Promise<void> }) {
  const [kind, setKind] = useState<'sale' | 'scrapped'>('sale');
  const [date, setDate] = useState('');
  const [proceeds, setProceeds] = useState('');
  const [quantity, setQuantity] = useState('');
  const [contact, setContact] = useState('');
  const [contacts, setContacts] = useState<Contact[]>([]);
  const [poolItem, setPoolItem] = useState('');
  const [note, setNote] = useState('');
  const [preview, setPreview] = useState<AssetDisposalPreview | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  useEffect(() => { void Api.getContacts().then(setContacts).catch(e => setError(errorText(e))); }, []);
  const request = {
    poolItemMovementId: Number(poolItem), assetId: asset.id, date, kind, proceeds: kind === 'sale' ? Math.round(Number(proceeds) * 100) : 0,
    quantity: Math.round(Number(quantity) * 10000), contactId: Number(contact), note,
    taxTreatment: asset.class === 'financial' ? 'exempt' : 'domestic', taxRate: 0,
  };
  return <form className="space-y-4" onChange={() => setPreview(null)} onSubmit={async event => {
    event.preventDefault();
    if (busy) return;
    setBusy(true); setError(null);
    try {
      if (preview) { await Api.disposeAsset(request); await onSaved(); }
      else setPreview(await Api.previewAssetDisposal(request));
    } catch (e) { setError(errorText(e)); } finally { setBusy(false); }
  }}>
    <Field label="Art"><Select value={kind} disabled={busy} items={[
      { value: 'sale', label: 'Verkauf' }, { value: 'scrapped', label: 'Verschrottung' },
    ]} onValueChange={value => { setKind(value as 'sale' | 'scrapped'); setPreview(null); }} /></Field>
    {asset.method === 'pool' && <Field label="Wirtschaftsgut im Sammelposten">
      <Select value={poolItem} disabled={busy} items={(asset.movements ?? [])
        .filter(m => m.kind === 'acquisition' || m.kind === 'subsequent_cost')
        .map(m => ({ value: String(m.id), label: `${m.date} · ${m.note ?? ''} · ${formatCents(m.costAmount)}` }))}
        onValueChange={value => { setPoolItem(value); setPreview(null); }} />
    </Field>}
    <Field label="Abgangsdatum"><Input type="date" required disabled={busy} value={date}
      onChange={e => setDate(e.target.value)} /></Field>
    {kind === 'sale' && <>
      <Field label="Nettoerlös in Euro"><Input type="number" min="0" step="0.01" required disabled={busy}
        value={proceeds} onChange={e => setProceeds(e.target.value)} /></Field>
      <Field label="Käufer"><Select value={contact} disabled={busy}
        items={contacts.filter(c => c.type === 'customer').map(c => ({ value: String(c.id), label: c.name }))}
        onValueChange={value => { setContact(value); setPreview(null); }} /></Field>
    </>}
    {(asset.quantity ?? 0) > 0 && <Field label="Abgehende Stückzahl" optional hint="Ohne Angabe geht der gesamte Bestand ab">
      <Input type="number" min="0" step="0.0001" disabled={busy} value={quantity}
        onChange={e => setQuantity(e.target.value)} /></Field>}
    <Field label="Grund"><Input required disabled={busy} value={note} onChange={e => setNote(e.target.value)} /></Field>
    {preview && <div className="text-caption">
      <p>AfA bis zum Abgang: {formatCents(preview.catchUpAmount)}</p>
      <p>Restbuchwert: {formatCents(preview.bookValue)}</p>
      <p>Umsatzsteuer: {formatCents(preview.tax)}</p>
      <p>Ergebnis: {formatCents(preview.result)}</p>
      {preview.warnings?.map(w => <Notice key={w} tone="attention" text={w} />)}
    </div>}
    {error && <Notice tone="negative" text={error} />}
    <Button type="submit" disabled={busy}>{preview ? 'Abgang buchen' : 'Abgang prüfen'}</Button>
  </form>;
}

export function AssetDisposalReversal({ asset, onSaved }: { asset: FixedAsset; onSaved: () => Promise<void> }) {
 const [movement, setMovement] = useState('');
 const [reason, setReason] = useState('');
 const [error, setError] = useState<string | null>(null);
 const [busy, setBusy] = useState(false);
 return <form className="space-y-4" onSubmit={async event => {
  event.preventDefault(); if (busy) return;
  setBusy(true); setError(null);
  try { await Api.reverseAssetDisposal(asset.id, Number(movement), reason); await onSaved(); }
  catch (e) { setError(errorText(e)); } finally { setBusy(false); }
 }}>
  <Field label="Abgang"><Select value={movement} disabled={busy} items={(asset.movements ?? [])
   .filter(m => m.kind === 'disposal').map(m => ({ value: String(m.id), label: `${m.date} · ${m.note ?? 'Abgang'}` }))}
   onValueChange={setMovement} /></Field>
  <Field label="Stornogrund"><Input required disabled={busy} value={reason} onChange={e => setReason(e.target.value)} /></Field>
  {error && <Notice tone="negative" text={error} />}
  <Button type="submit" disabled={busy || !movement}>Abgang zurücknehmen</Button>
 </form>;
}
