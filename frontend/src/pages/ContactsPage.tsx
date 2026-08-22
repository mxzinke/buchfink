import React, { useEffect, useState } from 'react';
import { toast } from 'sonner';
import { Plus, Building, User, Mail, CreditCard, Pencil } from 'lucide-react';
import { Contact, ContactType, TaxTreatmentInfo } from '../types';
import { Api } from '../services/api';
import { formatCents } from '../utils/formatters';
import { HelpTooltip } from '../components/HelpTooltip';
import {
  ErrorBox,
  Field,
  Modal,
  PrimaryButton,
  SecondaryButton,
  inputClass,
} from '../components/Form';

export const ContactsPage: React.FC = () => {
  const [contacts, setContacts] = useState<Contact[]>([]);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState<Partial<Contact> | null>(null);

  useEffect(() => {
    loadContacts();
  }, []);

  const loadContacts = async () => {
    setLoading(true);
    try {
      const list = await Api.getContacts();
      setContacts(list);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="p-3 sm:p-6 md:p-8 max-w-7xl mx-auto space-y-4 sm:space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold text-stone-900 tracking-tight flex items-center">
            Kunden & Lieferanten
            <HelpTooltip
              title="Kontakte & Zahlungsübersicht"
              content="Verwalten Sie Ihre Kunden und Lieferanten, um Rechnungen schnell zuzuordnen und offene Rechnungsbeträge nachzuvollziehen."
            />
          </h2>
          <p className="text-xs text-stone-500 mt-1">
            Kontaktdaten und offene Beträge im Überblick
          </p>
        </div>

        <button
          onClick={() => setEditing({ type: 'vendor', countryCode: 'DE', paymentTermsDays: 14 })}
          className="flex items-center gap-1.5 px-3.5 py-2 rounded-lg bg-amber-700 text-white text-xs font-semibold hover:bg-amber-800 transition-colors shadow-xs"
        >
          <Plus className="w-3.5 h-3.5" />
          Neuer Kontakt
        </button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5">
        {loading ? (
          <div className="col-span-full py-12 text-center text-stone-400 text-xs">
            Kontakte werden geladen...
          </div>
        ) : contacts.length === 0 ? (
          <div className="col-span-full py-12 text-center text-stone-400 text-xs">
            Noch keine Kontakte angelegt.
          </div>
        ) : (
          contacts.map((c) => (
            <div
              key={c.id}
              className="bg-white p-5 rounded-xl border border-stone-200/80 shadow-xs space-y-4 hover:border-amber-500/40 transition-colors"
            >
              <div className="flex items-start justify-between">
                <div className="flex items-center gap-3">
                  <div
                    className={`p-2.5 rounded-lg ${
                      c.type === 'customer'
                        ? 'bg-amber-50 text-amber-700'
                        : 'bg-stone-100 text-stone-700'
                    }`}
                  >
                    {c.company ? <Building className="w-4 h-4" /> : <User className="w-4 h-4" />}
                  </div>
                  <div>
                    <h3 className="font-bold text-sm text-stone-900">{c.name}</h3>
                    <div className="text-xs font-mono text-stone-500">{c.ledgerAccount}</div>
                  </div>
                </div>

                <span
                  className={`px-2 py-0.5 rounded text-[10px] font-semibold ${
                    c.type === 'customer'
                      ? 'bg-amber-100 text-amber-800'
                      : 'bg-stone-200 text-stone-800'
                  }`}
                >
                  {c.type === 'customer' ? 'Kunde' : 'Lieferant'}
                </span>
              </div>

              <div className="space-y-1 text-xs text-stone-600">
                {c.email && (
                  <div className="flex items-center gap-2 truncate">
                    <Mail className="w-3.5 h-3.5 text-stone-400 shrink-0" />
                    <span>{c.email}</span>
                  </div>
                )}
                {c.iban && (
                  <div className="flex items-center gap-2 truncate font-mono text-xs">
                    <CreditCard className="w-3.5 h-3.5 text-stone-400 shrink-0" />
                    <span>{c.iban}</span>
                  </div>
                )}
              </div>

              <div className="pt-3 border-t border-stone-100 flex items-center justify-between text-xs">
                <span className="text-stone-500">Offener Betrag:</span>
                <span className="font-mono font-bold text-stone-900">
                  {formatCents(c.openAmount)}
                </span>
              </div>
              <button
                onClick={() => setEditing(c)}
                className="w-full pt-2 text-[11px] text-stone-400 hover:text-amber-700 flex items-center justify-center gap-1"
              >
                <Pencil className="w-3 h-3" />
                Bearbeiten
              </button>
            </div>
          ))
        )}
      </div>

      {editing && (
        <ContactForm
          contact={editing}
          onClose={() => setEditing(null)}
          onSaved={async (name) => {
            setEditing(null);
            toast.success(`${name} gespeichert`);
            await loadContacts();
          }}
        />
      )}
    </div>
  );
};

/**
 * Kunden und Lieferanten.
 *
 * Das Personenkonto vergibt das Backend beim Speichern aus den
 * DATEV-Nummernkreisen (10000–69999 Debitoren, 70000–99999 Kreditoren) — es wird
 * hier nicht eingegeben und nie wiederverwendet.
 *
 * Welche Stammdaten ein Steuerfall verlangt, sagt ebenfalls das Backend über
 * `requiresVatId`. Diese Maske zeigt es an, statt die Regel nachzubauen.
 */
const ContactForm: React.FC<{
  contact: Partial<Contact>;
  onClose: () => void;
  onSaved: (name: string) => Promise<void>;
}> = ({ contact, onClose, onSaved }) => {
  const [draft, setDraft] = useState<Partial<Contact>>(contact);
  const [treatments, setTreatments] = useState<TaxTreatmentInfo[]>([]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const isNew = !draft.id;
  const direction = draft.type === 'customer' ? 'outgoing' : 'incoming';

  useEffect(() => {
    let cancelled = false;
    Api.getTaxTreatments(direction)
      .then((list) => {
        if (!cancelled) setTreatments(list ?? []);
      })
      .catch(() => undefined);
    return () => {
      cancelled = true;
    };
  }, [direction]);

  function set(patch: Partial<Contact>) {
    setDraft((prev) => ({ ...prev, ...patch }));
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const saved = await Api.saveContact(draft);
      await onSaved(saved.name);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  // Steuerfälle, die ohne USt-IdNr. des Partners nicht gehen — die Meldung dazu
  // kommt beim Buchen aus dem Backend, hier steht nur der Hinweis vorweg.
  const needVatID = treatments.filter((t) => t.requiresVatId).map((t) => t.label);

  return (
    <Modal
      title={isNew ? 'Neuer Kontakt' : draft.name || 'Kontakt bearbeiten'}
      onClose={onClose}
      width="max-w-2xl"
    >
      <form onSubmit={submit}>
        <div className="p-5 space-y-3">
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <Field
              label="Art"
              hint={isNew ? undefined : 'nicht änderbar — das Personenkonto hängt daran'}
            >
              <select
                className={inputClass}
                value={draft.type ?? 'vendor'}
                disabled={!isNew}
                onChange={(e) => set({ type: e.target.value as ContactType })}
              >
                <option value="vendor">Lieferant (Kreditor)</option>
                <option value="customer">Kunde (Debitor)</option>
              </select>
            </Field>
            <Field label="Personenkonto" hint="wird beim Anlegen vergeben">
              <input className={inputClass} value={draft.ledgerAccount ?? ''} disabled />
            </Field>
          </div>

          <Field label="Name">
            <input
              className={inputClass}
              value={draft.name ?? ''}
              onChange={(e) => set({ name: e.target.value })}
              required
            />
          </Field>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <Field label="Firma">
              <input
                className={inputClass}
                value={draft.company ?? ''}
                onChange={(e) => set({ company: e.target.value })}
              />
            </Field>
            <Field label="E-Mail">
              <input
                type="email"
                className={inputClass}
                value={draft.email ?? ''}
                onChange={(e) => set({ email: e.target.value })}
              />
            </Field>
          </div>

          <Field label="Anschrift">
            <textarea
              className={inputClass}
              rows={2}
              value={draft.address ?? ''}
              onChange={(e) => set({ address: e.target.value })}
            />
          </Field>

          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
            <Field label="Land" hint="ISO-Code, z. B. DE">
              <input
                className={inputClass}
                maxLength={2}
                value={draft.countryCode ?? ''}
                onChange={(e) => set({ countryCode: e.target.value.toUpperCase() })}
              />
            </Field>
            <Field label="Steuernummer">
              <input
                className={inputClass}
                value={draft.taxId ?? ''}
                onChange={(e) => set({ taxId: e.target.value })}
              />
            </Field>
            <Field
              label="USt-IdNr."
              hint={needVatID.length > 0 ? `nötig für: ${needVatID.join(', ')}` : undefined}
            >
              <input
                className={inputClass}
                value={draft.vatId ?? ''}
                onChange={(e) => set({ vatId: e.target.value })}
              />
            </Field>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
            <Field label="IBAN">
              <input
                className={`${inputClass} font-mono`}
                value={draft.iban ?? ''}
                onChange={(e) => set({ iban: e.target.value.replace(/\s/g, '').toUpperCase() })}
              />
            </Field>
            <Field label="BIC">
              <input
                className={`${inputClass} font-mono`}
                value={draft.bic ?? ''}
                onChange={(e) => set({ bic: e.target.value.toUpperCase() })}
              />
            </Field>
            <Field label="Zahlungsziel" hint="Tage">
              <input
                type="number"
                min={0}
                className={inputClass}
                value={draft.paymentTermsDays ?? 14}
                onChange={(e) => set({ paymentTermsDays: Number(e.target.value) })}
              />
            </Field>
          </div>

          <div className="space-y-1.5 pt-1">
            <label className="flex items-start gap-2 text-xs text-stone-600">
              <input
                type="checkbox"
                className="mt-0.5 accent-amber-700"
                checked={draft.isPrivate ?? false}
                onChange={(e) => set({ isPrivate: e.target.checked })}
              />
              <span>
                Privatperson, kein Unternehmer
                <span className="block text-[11px] text-stone-400">
                  Dann besteht für Rechnungen dieses Partners keine E-Rechnungspflicht.
                </span>
              </span>
            </label>
            <label className="flex items-start gap-2 text-xs text-stone-600">
              <input
                type="checkbox"
                className="mt-0.5 accent-amber-700"
                checked={draft.isSmallBusiness ?? false}
                onChange={(e) => set({ isSmallBusiness: e.target.checked })}
              />
              <span>
                Kleinunternehmer nach § 19 UStG
                <span className="block text-[11px] text-stone-400">
                  Darf nach § 34a UStDV immer eine sonstige Rechnung ausstellen.
                </span>
              </span>
            </label>
          </div>

          <ErrorBox message={error} />
        </div>

        <div className="flex justify-end gap-2 px-5 py-3 border-t border-stone-100">
          <SecondaryButton type="button" onClick={onClose}>
            Abbrechen
          </SecondaryButton>
          <PrimaryButton type="submit" disabled={busy}>
            {busy ? 'Wird gespeichert…' : 'Speichern'}
          </PrimaryButton>
        </div>
      </form>
    </Modal>
  );
};
