import React, { useEffect, useMemo, useState } from 'react';
import { AlertCircle, Plus } from 'lucide-react';
import { Contact, ContactType, EInvoiceProfileInfo, TaxTreatmentInfo } from '../types';
import { Api } from '../services/api';
import { useWriteLock } from '../components/WriteLock';
import { formatCents } from '../utils/formatters';
import {
  Button,
  Checkbox,
  Dialog,
  EmptyState,
  Field,
  HelpPopover,
  Input,
  Notice,
  PageHeader,
  SearchInput,
  Select,
  SkeletonRows,
  Table,
  Tbody,
  Td,
  Textarea,
  Th,
  Thead,
  Tr,
  toast,
} from '../components/ui';

export const ContactsPage: React.FC = () => {
  // Ein Kontakt trägt sein Personenkonto: ihn anzulegen ist eine Änderung an
  // den Stammdaten und im Prüfermodus gesperrt (§10.4).
  const writeLock = useWriteLock();
  const [contacts, setContacts] = useState<Contact[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [editing, setEditing] = useState<Partial<Contact> | null>(null);

  useEffect(() => {
    void loadContacts();
  }, []);

  async function loadContacts() {
    setLoading(true);
    try {
      setContacts(await Api.getContacts());
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }

  const filtered = useMemo(() => {
    const query = search.trim().toLowerCase();
    if (!query) return contacts;
    return contacts.filter(
      (c) =>
        c.name.toLowerCase().includes(query) ||
        (c.company ?? '').toLowerCase().includes(query) ||
        (c.email ?? '').toLowerCase().includes(query) ||
        c.ledgerAccount.includes(query),
    );
  }, [contacts, search]);

  const openTotal = contacts.reduce((sum, c) => sum + c.openAmount, 0);

  return (
    <div className="max-w-[1200px] mx-auto px-8 py-8">
      <PageHeader
        title="Kunden & Lieferanten"
        context={`${contacts.length} Kontakte · ${formatCents(openTotal)} offen`}
        action={
          <Button
            variant="primary"
            icon={<Plus className="w-4 h-4" strokeWidth={1.5} />}
            disabled={writeLock.locked}
            title={writeLock.hint}
            onClick={() => setEditing({ type: 'vendor', countryCode: 'DE', paymentTermsDays: 14 })}
          >
            Neuer Kontakt
          </Button>
        }
      />

      <div className="mt-6">
        <SearchInput
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Name, Firma, E-Mail oder Personenkonto"
          className="max-w-md"
        />
      </div>

      <div className="mt-5">
        {loading ? (
          <SkeletonRows rows={6} />
        ) : filtered.length === 0 ? (
          <EmptyState
            variant={contacts.length === 0 ? 'leer' : 'gefiltert'}
            title={
              contacts.length === 0 ? 'Noch keine Kontakte angelegt' : 'Kein Kontakt passt zur Suche'
            }
            description={
              contacts.length === 0
                ? 'Kontakte ordnen Rechnungen und Zahlungen einem Personenkonto zu.'
                : undefined
            }
            action={
              contacts.length === 0 ? (
                <Button
                  variant="primary"
                  disabled={writeLock.locked}
                  title={writeLock.hint}
                  onClick={() => setEditing({ type: 'vendor', countryCode: 'DE', paymentTermsDays: 14 })}
                >
                  Neuer Kontakt
                </Button>
              ) : (
                <Button variant="secondary" onClick={() => setSearch('')}>
                  Suche zurücksetzen
                </Button>
              )
            }
          />
        ) : (
          <Table>
            <Thead sticky>
              <Tr>
                <Th>Name</Th>
                <Th>Art</Th>
                <Th>Personenkonto</Th>
                <Th>E-Mail</Th>
                <Th>IBAN</Th>
                <Th numeric>Offener Betrag</Th>
                <Th className="w-24" aria-label="Aktionen" />
              </Tr>
            </Thead>
            <Tbody>
              {filtered.map((contact) => (
                <Tr key={contact.id} className="group">
                  <Td className="max-w-[18rem] truncate" title={contact.company || contact.name}>
                    {contact.name}
                    {contact.company && contact.company !== contact.name && (
                      <span className="text-ink-subtle"> · {contact.company}</span>
                    )}
                  </Td>
                  <Td className="text-ink-muted">
                    {contact.type === 'customer' ? 'Kunde' : 'Lieferant'}
                  </Td>
                  <Td code>{contact.ledgerAccount}</Td>
                  <Td className="max-w-[16rem] truncate text-ink-muted">{contact.email || '—'}</Td>
                  <Td code>{contact.iban || '—'}</Td>
                  <Td numeric>{formatCents(contact.openAmount)}</Td>
                  <Td className="pl-0">
                    <Button
                      variant="quiet"
                      size="sm"
                      onClick={() => setEditing(contact)}
                      className="opacity-0 transition-opacity duration-120 ease-quiet
                                 group-hover:opacity-100 focus-visible:opacity-100"
                    >
                      Bearbeiten
                    </Button>
                  </Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
      </div>

      <ContactForm
        contact={editing}
        onClose={() => setEditing(null)}
        onSaved={async (name) => {
          setEditing(null);
          toast.success(`${name} gespeichert.`);
          await loadContacts();
        }}
      />
    </div>
  );
};

/**
 * Kunden und Lieferanten.
 *
 * Das Personenkonto vergibt das Backend beim Speichern aus den
 * DATEV-Nummernkreisen (10000–69999 Debitoren, 70000–99999 Kreditoren). Es wird
 * hier nicht eingegeben und nie wiederverwendet.
 *
 * Welche Stammdaten ein Steuerfall verlangt, sagt ebenfalls das Backend über
 * `requiresVatId`. Diese Maske zeigt es an, statt die Regel nachzubauen.
 */
const ContactForm: React.FC<{
  contact: Partial<Contact> | null;
  onClose: () => void;
  onSaved: (name: string) => Promise<void>;
}> = ({ contact, onClose, onSaved }) => {
  const writeLock = useWriteLock();
  const [draft, setDraft] = useState<Partial<Contact>>(contact ?? {});
  const [treatments, setTreatments] = useState<TaxTreatmentInfo[]>([]);
  const [profiles, setProfiles] = useState<EInvoiceProfileInfo[]>([]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const isNew = !draft.id;
  const direction = draft.type === 'customer' ? 'outgoing' : 'incoming';
  const profileInfo = profiles.find(
    (p) => p.profile === (draft.eInvoiceProfile ?? 'zugferd_en16931'),
  );
  // Ein Bestandskontakt, dessen einzeilige Anschrift der Parser nicht trennen
  // konnte: die Felder sind leer, der alte Text steht noch da. Geraten wird
  // nichts — eine falsche Straße auf der Rechnung kostet den Empfänger den
  // Vorsteuerabzug.
  const addressIncomplete = Boolean(
    draft.address && !(draft.street && draft.postalCode && draft.city),
  );

  useEffect(() => {
    if (contact) {
      setDraft(contact);
      setError(null);
    }
  }, [contact]);

  // Die Zielformate kommen aus dem Backend und nicht aus einer zweiten Liste
  // hier: welches Format Buchfink erzeugen kann, weiß nur der Renderer.
  useEffect(() => {
    let cancelled = false;
    Api.getEInvoiceProfiles()
      .then((list) => {
        if (!cancelled) setProfiles(list);
      })
      .catch(() => undefined);
    return () => {
      cancelled = true;
    };
  }, []);

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

  async function submit() {
    if (!draft.name?.trim()) {
      setError('Ohne Namen lässt sich der Kontakt nicht zuordnen.');
      return;
    }
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

  // Steuerfälle, die ohne USt-IdNr. des Partners nicht gehen. Die Meldung dazu
  // kommt beim Buchen aus dem Backend, hier steht nur der Hinweis vorweg.
  const needVatID = treatments.filter((t) => t.requiresVatId).map((t) => t.label);

  return (
    <Dialog
      open={contact !== null}
      onOpenChange={(next) => !next && onClose()}
      title={isNew ? 'Neuer Kontakt' : draft.name || 'Kontakt bearbeiten'}
      width="max-w-2xl"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Abbrechen
          </Button>
          <Button
            variant="primary"
            loading={busy}
            disabled={writeLock.locked}
            title={writeLock.hint}
            onClick={submit}
          >
            Speichern
          </Button>
        </>
      }
    >
      <div className="grid grid-cols-2 gap-4">
        <Field
          label="Art"
          hint={isNew ? undefined : 'nicht änderbar'}
          help={isNew ? undefined : 'Das Personenkonto hängt an der Art und wird nie umgehängt.'}
        >
          <Select
            items={[
              { value: 'vendor', label: 'Lieferant (Kreditor)' },
              { value: 'customer', label: 'Kunde (Debitor)' },
            ]}
            value={draft.type ?? 'vendor'}
            disabled={!isNew}
            onValueChange={(type) => set({ type: type as ContactType })}
          />
        </Field>
        <Field label="Personenkonto" hint="wird beim Anlegen vergeben">
          <Input value={draft.ledgerAccount ?? ''} disabled />
        </Field>
      </div>

      <Field label="Name" className="mt-4">
        <Input value={draft.name ?? ''} onChange={(e) => set({ name: e.target.value })} />
      </Field>

      <div className="grid grid-cols-2 gap-4 mt-4">
        <Field label="Firma" optional>
          <Input value={draft.company ?? ''} onChange={(e) => set({ company: e.target.value })} />
        </Field>
        <Field label="E-Mail" optional>
          <Input
            type="email"
            value={draft.email ?? ''}
            onChange={(e) => set({ email: e.target.value })}
          />
        </Field>
      </div>

      {/* Straße, PLZ und Ort einzeln: § 14 Abs. 4 Nr. 1 UStG verlangt die
          vollständige Anschrift des Empfängers, und EN 16931 verlangt sie in
          Feldern (BT-50, BT-52, BT-53). Aus einer einzeiligen Anschrift wurde
          beim Empfänger die Stadt Teil der Straße. */}
      <div className="grid grid-cols-1 md:grid-cols-[minmax(0,2fr)_7rem_minmax(0,2fr)] gap-4 mt-4">
        <Field label="Straße und Hausnummer">
          <Input value={draft.street ?? ''} onChange={(e) => set({ street: e.target.value })} />
        </Field>
        <Field label="PLZ">
          <Input
            className="code-num"
            value={draft.postalCode ?? ''}
            onChange={(e) => set({ postalCode: e.target.value })}
          />
        </Field>
        <Field label="Ort">
          <Input value={draft.city ?? ''} onChange={(e) => set({ city: e.target.value })} />
        </Field>
      </div>

      {addressIncomplete && (
        <Notice
          className="mt-4"
          text="Die übernommene Anschrift ließ sich nicht in Straße, PLZ und Ort trennen; die Rechnung braucht alle drei."
        />
      )}

      <Field
        label="Übernommene Anschrift"
        className="mt-4"
        optional
        help="Die alte einzeilige Fassung. Sie bleibt als Nachweis stehen."
      >
        <Textarea
          rows={2}
          value={draft.address ?? ''}
          onChange={(e) => set({ address: e.target.value })}
        />
      </Field>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mt-4">
        <Field
          label="E-Rechnungsformat"
          hint={profileInfo?.hint ? undefined : 'für Ausgangsrechnungen'}
          explain={profileInfo?.hint}
        >
          <Select
            items={profiles.map((p) => ({ value: p.profile, label: p.label }))}
            value={draft.eInvoiceProfile ?? 'zugferd_en16931'}
            onValueChange={(profile) => set({ eInvoiceProfile: profile })}
          />
        </Field>
        <Field
          label="Leitweg-ID"
          optional={draft.eInvoiceProfile !== 'xrechnung_cii'}
          hint={draft.eInvoiceProfile === 'xrechnung_cii' ? 'bei XRechnung Pflicht' : undefined}
          explain="Die Route-ID des öffentlichen Auftraggebers (BT-10). Ohne sie findet die Rechnung ihren Empfänger in der Verwaltung nicht; Buchfink weist die Ausstellung dann zurück (BR-DE-15)."
        >
          {/* Leerzeichen gehören nicht in eine Leitweg-ID, das Abschneiden
              aber ans Verlassen des Feldes (§8.3): beim Tippen genommen,
              springt die Schreibmarke nach jedem versehentlichen Leerzeichen
              zurück. */}
          <Input
            className="code-num"
            value={draft.leitwegId ?? ''}
            onChange={(e) => set({ leitwegId: e.target.value })}
            onBlur={(e) => set({ leitwegId: e.target.value.trim() })}
          />
        </Field>
      </div>

      <div className="grid grid-cols-3 gap-4 mt-4">
        <Field label="Land" hint="ISO-Code">
          <Input
            maxLength={2}
            value={draft.countryCode ?? ''}
            onChange={(e) => set({ countryCode: e.target.value.toUpperCase() })}
          />
        </Field>
        <Field label="Steuernummer" optional>
          <Input value={draft.taxId ?? ''} onChange={(e) => set({ taxId: e.target.value })} />
        </Field>
        <Field
          label="USt-IdNr."
          optional
          help={
            needVatID.length > 0
              ? `Nötig für: ${needVatID.join(', ')}.`
              : undefined
          }
        >
          <Input value={draft.vatId ?? ''} onChange={(e) => set({ vatId: e.target.value })} />
        </Field>
      </div>

      <div className="grid grid-cols-3 gap-4 mt-4">
        <Field label="IBAN" optional>
          <Input
            className="code-num"
            value={draft.iban ?? ''}
            onChange={(e) => set({ iban: e.target.value.replace(/\s/g, '').toUpperCase() })}
          />
        </Field>
        <Field label="BIC" optional>
          <Input
            className="code-num"
            value={draft.bic ?? ''}
            onChange={(e) => set({ bic: e.target.value.toUpperCase() })}
          />
        </Field>
        <Field label="Zahlungsziel" hint="Tage">
          <Input
            type="number"
            min={0}
            align="right"
            value={draft.paymentTermsDays ?? 14}
            onChange={(e) => set({ paymentTermsDays: Number(e.target.value) })}
          />
        </Field>
      </div>

      <div className="mt-6 pt-5 border-t border-line flex flex-col gap-3">
        <Checkbox
          checked={draft.isPrivate ?? false}
          onCheckedChange={(checked) => set({ isPrivate: Boolean(checked) })}
          label="Privatperson, kein Unternehmer"
          hint="Für Rechnungen an diesen Partner besteht keine E-Rechnungspflicht."
        />
        <Checkbox
          checked={draft.isSmallBusiness ?? false}
          onCheckedChange={(checked) => set({ isSmallBusiness: Boolean(checked) })}
          label="Kleinunternehmer nach § 19 UStG"
          hint="Darf nach § 34a UStDV immer eine sonstige Rechnung ausstellen."
        />
      </div>

      <div className="mt-4">
        <HelpPopover label="Erklärung zum Personenkonto">
          Das Personenkonto vergibt Buchfink beim Anlegen aus den DATEV-Nummernkreisen: 10000 bis
          69999 für Debitoren, 70000 bis 99999 für Kreditoren. Eine einmal vergebene Nummer wird nie
          wiederverwendet.
        </HelpPopover>
      </div>

      {error && (
        <div className="mt-4 flex items-start gap-2.5 rounded-control border border-negative-line bg-negative-soft px-4 py-3">
          <AlertCircle className="w-4 h-4 mt-0.5 shrink-0 text-negative" strokeWidth={1.5} />
          <p className="text-body text-negative-text">{error}</p>
        </div>
      )}
    </Dialog>
  );
};
