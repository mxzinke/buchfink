import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Lock, Plus, Users } from 'lucide-react';
import { Api } from '../../services/api';
import type { Contact, ContactType, EInvoiceProfileInfo, PostingGroup, VatIDStatus } from '../../types';
import { useRouteObject } from '../../navigation';
import { useWriteLock } from '../../components/WriteLock';
import { useLiveReload } from '../../live';
import { agentName } from '../../utils/actor';
import { hasPendingProposal, postingGroupState, withPostingGroupOf } from '../../utils/contacts';
import { formatCents, formatDate, formatDateOfTimestamp, formatDateTime } from '../../utils/formatters';
import {
  Button,
  Checkbox,
  DetailList,
  DetailPanel,
  Dialog,
  Disclosure,
  EmptyState,
  Field,
  FormGrid,
  Help,
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
} from '../../components/ui';

function message(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

function typeLabel(type: ContactType): string {
  return type === 'customer' ? 'Kunde' : 'Lieferant';
}

const NEW_CONTACT: Partial<Contact> = { type: 'vendor', countryCode: 'DE', paymentTermsDays: 14 };

/**
 * Kunden und Lieferanten. Die Liste zeigt Personenkonto und offenen Betrag,
 * alles Weitere die Seitenleiste. Anlegen kann auch eine verbundene App; die
 * Standardgruppe eines Lieferanten bestätigt nur der Mensch.
 */
export const ContactsPanel: React.FC = () => {
  const writeLock = useWriteLock();
  const selection = useRouteObject('company', 'contacts');
  const [contacts, setContacts] = useState<Contact[] | null>(null);
  const [search, setSearch] = useState('');
  const [editing, setEditing] = useState<Partial<Contact> | null>(null);
  const [blocking, setBlocking] = useState<Contact | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      setContacts(await Api.getContacts());
      setLoadError(null);
    } catch (e) {
      setLoadError(`Die Kontakte konnten nicht geladen werden. ${message(e)}`);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  // Claude legt Kontakte an und schlägt Standardgruppen vor.
  useLiveReload(load, ['stammdaten', 'belege', 'buchungen']);

  const filtered = useMemo(() => {
    const list = contacts ?? [];
    const query = search.trim().toLowerCase();
    if (!query) return list;
    return list.filter(
      (c) =>
        c.name.toLowerCase().includes(query) ||
        (c.company ?? '').toLowerCase().includes(query) ||
        (c.email ?? '').toLowerCase().includes(query) ||
        c.ledgerAccount.includes(query),
    );
  }, [contacts, search]);

  const selected = contacts?.find((c) => String(c.id) === selection.id) ?? null;
  const openTotal = (contacts ?? []).reduce((sum, c) => sum + c.openAmount, 0);

  return (
    <div>
      <PageHeader
        title="Kunden & Lieferanten"
        context={contacts ? `${contacts.length} Kontakte · ${formatCents(openTotal)} offen` : undefined}
        action={
          <Button
            variant="secondary"
            icon={<Plus className="w-4 h-4" strokeWidth={1.5} />}
            disabled={writeLock.locked}
            title={writeLock.hint}
            onClick={() => setEditing(NEW_CONTACT)}
          >
            Neuer Kontakt
          </Button>
        }
      />

      {notice && (
        <Notice
          className="mt-6"
          text={notice}
          action={
            <Button variant="quiet" size="sm" onClick={() => setNotice(null)}>
              Verstanden
            </Button>
          }
        />
      )}

      <div className="mt-6 max-w-md">
        <SearchInput
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Name, Firma, E-Mail oder Personenkonto"
        />
      </div>

      <div className="mt-5">
        {loadError ? (
          <Notice
            tone="negative"
            text={loadError}
            action={
              <Button variant="quiet" size="sm" onClick={() => void load()}>
                Erneut laden
              </Button>
            }
          />
        ) : contacts === null ? (
          <SkeletonRows rows={6} />
        ) : filtered.length === 0 ? (
          <EmptyState
            variant={contacts.length === 0 ? 'leer' : 'gefiltert'}
            icon={<Users className="w-6 h-6" strokeWidth={1.5} />}
            title={contacts.length === 0 ? 'Noch keine Kontakte' : 'Kein Kontakt passt zur Suche'}
            description={
              contacts.length === 0
                ? 'Kontakte entstehen mit dem ersten Beleg oder der ersten Rechnung.'
                : undefined
            }
            action={
              contacts.length > 0 ? (
                <Button variant="secondary" onClick={() => setSearch('')}>
                  Suche zurücksetzen
                </Button>
              ) : undefined
            }
          />
        ) : (
          <Table>
            <Thead sticky>
              <Tr>
                <Th>Name</Th>
                <Th className="w-28">Art</Th>
                <Th className="w-32">Personenkonto</Th>
                <Th numeric className="w-36">Offen</Th>
              </Tr>
            </Thead>
            <Tbody>
              {filtered.map((contact) => (
                <Tr key={contact.id}>
                  <Td className="max-w-[28rem] truncate">
                    <button
                      type="button"
                      className="text-left text-ink underline-offset-2 hover:underline"
                      onClick={() => selection.open(String(contact.id))}
                    >
                      {contact.company || contact.name}
                    </button>
                    {contact.company && contact.name && contact.company !== contact.name && (
                      <span className="text-ink-subtle"> · {contact.name}</span>
                    )}
                    {contact.blocked && (
                      <span className="ml-2 text-caption text-negative-text whitespace-nowrap">Gesperrt</span>
                    )}
                    {hasPendingProposal(contact) && (
                      <span className="ml-2 text-caption text-attention-text whitespace-nowrap">
                        Standardgruppe bestätigen
                      </span>
                    )}
                  </Td>
                  <Td className="text-ink-muted">{typeLabel(contact.type)}</Td>
                  <Td code>{contact.ledgerAccount}</Td>
                  <Td numeric>{contact.openAmount ? formatCents(contact.openAmount) : '—'}</Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
      </div>

      <ContactDetail
        contact={selected}
        onClose={selection.close}
        onEdit={(contact) => setEditing(contact)}
        onBlock={(contact) => setBlocking(contact)}
        onChanged={(changed) =>
          setContacts((prev) => (prev ?? []).map((c) => (c.id === changed.id ? withPostingGroupOf(c, changed) : c)))
        }
      />

      {editing && (
        <ContactForm
          contact={editing}
          onClose={() => setEditing(null)}
          onSaved={async (saved) => {
            setEditing(null);
            toast.success(`${saved.company || saved.name} gespeichert.`);
            setNotice(saved.vatIdNotice ?? null);
            await load();
          }}
        />
      )}

      {blocking && (
        <BlockDialog
          contact={blocking}
          onClose={() => setBlocking(null)}
          onBlocked={async (answer) => {
            setBlocking(null);
            selection.close();
            setNotice(answer);
            await load();
          }}
        />
      )}
    </div>
  );
};

/** Die Einzelheiten eines Kontakts in der Seitenleiste. */
const ContactDetail: React.FC<{
  contact: Contact | null;
  onClose: () => void;
  onEdit: (contact: Contact) => void;
  onBlock: (contact: Contact) => void;
  onChanged: (contact: Contact) => void;
}> = ({ contact, onClose, onEdit, onBlock, onChanged }) => {
  const writeLock = useWriteLock();
  const [vatStatus, setVatStatus] = useState<VatIDStatus | null>(null);

  useEffect(() => {
    setVatStatus(null);
    if (!contact?.id || !contact.vatId) return;
    let active = true;
    Api.getVatIDStatus(contact.id)
      .then((status) => active && setVatStatus(status))
      .catch(() => undefined);
    return () => {
      active = false;
    };
  }, [contact?.id, contact?.vatId]);

  const address = contact
    ? [contact.street, [contact.postalCode, contact.city].filter(Boolean).join(' '), contact.countryCode]
        .filter(Boolean)
        .join(', ')
    : '';

  return (
    <DetailPanel
      open={contact !== null}
      onOpenChange={(open) => !open && onClose()}
      title={contact ? contact.company || contact.name : ''}
      context={contact ? `${typeLabel(contact.type)} · Personenkonto ${contact.ledgerAccount}` : undefined}
      footer={
        contact && (
          <>
            {!contact.blocked && (
              <Button
                variant="quiet"
                disabled={writeLock.locked}
                title={writeLock.hint ?? 'Nach einem Löschverlangen: der Kontakt verschwindet aus allen Auswahlen.'}
                icon={<Lock className="w-4 h-4" strokeWidth={1.5} />}
                onClick={() => onBlock(contact)}
              >
                Sperren
              </Button>
            )}
            <Button
              variant="secondary"
              disabled={writeLock.locked || contact.blocked}
              title={writeLock.hint ?? (contact.blocked ? 'Ein gesperrter Kontakt lässt sich nicht mehr ändern.' : undefined)}
              onClick={() => onEdit(contact)}
            >
              Bearbeiten
            </Button>
          </>
        )
      }
    >
      {contact && (
        <>
          {contact.blocked && (
            <Notice
              tone="negative"
              className="mb-5"
              text={`Gesperrt${contact.blockedAt ? ` seit ${formatDate(contact.blockedAt)}` : ''}${contact.blockedReason ? `: ${contact.blockedReason}` : ''}`}
            />
          )}
          <DetailList
            items={[
              { label: 'Offen', value: formatCents(contact.openAmount), numeric: true },
              ...(contact.company && contact.name !== contact.company ? [{ label: 'Ansprechpartner', value: contact.name }] : []),
              { label: 'Anschrift', value: address || '—' },
              { label: 'E-Mail', value: contact.email || '—' },
              {
                label: 'USt-IdNr.',
                value: contact.vatId ? (
                  <span>
                    <span className="code-num">{contact.vatId}</span>
                    {vatStatus && (
                      <span className={vatStatus.confirmed ? 'block text-caption text-positive-text' : 'block text-caption text-ink-subtle'}>
                        {vatStatus.confirmed && vatStatus.latest
                          ? `Bestätigt am ${formatDateTime(vatStatus.latest.checkedAt)}`
                          : vatStatus.note}
                      </span>
                    )}
                  </span>
                ) : (
                  '—'
                ),
              },
              ...(contact.taxId ? [{ label: 'Steuernummer', value: contact.taxId }] : []),
              {
                label: 'Bank',
                value: contact.iban ? <span className="code-num">{[contact.iban, contact.bic].filter(Boolean).join(' · ')}</span> : '—',
              },
              { label: 'Zahlungsziel', value: `${contact.paymentTermsDays} Tage` },
              ...(contact.leitwegId ? [{ label: 'Leitweg-ID', value: <span className="code-num">{contact.leitwegId}</span> }] : []),
              ...(contact.isPrivate ? [{ label: 'Privatperson', value: 'keine E-Rechnungspflicht' }] : []),
              ...(contact.isSmallBusiness ? [{ label: 'Kleinunternehmer', value: 'Rechnungen ohne Umsatzsteuer' }] : []),
              ...(contact.exemptionCertificateNumber
                ? [
                    {
                      label: 'Freistellung',
                      value: `${contact.exemptionCertificateNumber}${contact.exemptionCertificateValidUntil ? ` · gültig bis ${formatDate(contact.exemptionCertificateValidUntil)}` : ''}`,
                    },
                  ]
                : []),
            ]}
          />
          {contact.type === 'vendor' && <PostingGroupBlock contact={contact} onChanged={onChanged} />}
        </>
      )}
    </DetailPanel>
  );
};

/**
 * Die Standard-Buchungsgruppe eines Lieferanten: bestätigt, von Claude
 * vorgeschlagen oder keine. Bestätigen kann nur der Mensch; erst dann bucht
 * Claude E-Rechnungen des Lieferanten ohne Rückfrage, in den Grenzen unter
 * Einstellungen › Buchen ohne Rückfrage.
 */
const PostingGroupBlock: React.FC<{ contact: Contact; onChanged: (contact: Contact) => void }> = ({
  contact,
  onChanged,
}) => {
  const writeLock = useWriteLock();
  const [groups, setGroups] = useState<PostingGroup[]>([]);
  const [choosing, setChoosing] = useState(false);
  const [choice, setChoice] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const state = postingGroupState(contact);
  const proposal = contact.proposedPostingGroup ?? '';
  const confirmed = contact.defaultPostingGroup ?? '';

  useEffect(() => {
    Api.getPostingGroups('incoming')
      .then(setGroups)
      .catch(() => setGroups([]));
  }, []);

  useEffect(() => {
    setChoosing(false);
    setChoice(proposal || confirmed);
    setError('');
  }, [contact.id, proposal, confirmed]);

  const label = (key: string) => groups.find((g) => g.key === key)?.label ?? key;
  const name = contact.company || contact.name;

  async function confirm(group: string) {
    setBusy(true);
    setError('');
    try {
      const saved = await Api.confirmDefaultPostingGroup(contact.id, group);
      onChanged(saved);
      toast.success(
        group ? `Standardgruppe ${label(group)} für ${name} bestätigt.` : `${name} hat keine Standardgruppe mehr.`,
      );
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  }

  const picker = (
    <div className="mt-2 flex flex-wrap items-center gap-2">
      <div className="min-w-[14rem] flex-1">
        <Select
          aria-label="Standardgruppe"
          items={groups.map((g) => ({ value: g.key, label: g.label }))}
          value={choice}
          placeholder="Gruppe wählen"
          disabled={writeLock.locked}
          onValueChange={setChoice}
        />
      </div>
      <Button
        variant="secondary"
        size="sm"
        loading={busy}
        disabled={!choice || writeLock.locked}
        title={writeLock.hint}
        onClick={() => void confirm(choice)}
      >
        Bestätigen
      </Button>
      {choosing && (
        <Button variant="quiet" size="sm" onClick={() => setChoosing(false)}>
          Abbrechen
        </Button>
      )}
    </div>
  );

  return (
    <section className="mt-6 pt-5 border-t border-line">
      <h3 className="flex items-center text-label text-ink-muted">
        Standardgruppe
        <Help
          summary="Mit dieser Gruppe bucht Claude E-Rechnungen des Lieferanten ohne Rückfrage."
          label="Erklärung zur Standardgruppe"
        >
          Claude schlägt die Gruppe vor, nach der die Rechnungen eines Lieferanten gebucht werden,
          etwa Fremdleistungen. Erst wenn Sie sie bestätigen, bucht Claude seine E-Rechnungen ohne
          Rückfrage, und nur unter der Betragsgrenze und ab der Zahl früherer Buchungen, die Sie unter
          Einstellungen festlegen. Fotos und PDF ohne E-Rechnung legt Claude Ihnen immer vor.
        </Help>
      </h3>

      {choosing || state === 'none' ? (
        <>
          {state === 'none' && (
            <p className="mt-1 text-caption text-ink-subtle">
              Keine bestätigt. Bis dahin fragt Claude bei jeder Rechnung nach.
            </p>
          )}
          {picker}
        </>
      ) : state === 'confirmed' ? (
        <div className="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1">
          <span className="text-body text-ink">{label(confirmed)}</span>
          <span className="text-caption text-positive-text">
            {contact.defaultPostingGroupConfirmedAt
              ? `Bestätigt am ${formatDateOfTimestamp(contact.defaultPostingGroupConfirmedAt)}`
              : 'Bestätigt'}
          </span>
          <span className="ml-auto flex gap-1">
            <Button
              variant="quiet"
              size="sm"
              disabled={writeLock.locked}
              title={writeLock.hint}
              onClick={() => setChoosing(true)}
            >
              Ändern
            </Button>
            <Button
              variant="quiet"
              size="sm"
              loading={busy}
              disabled={writeLock.locked}
              title={writeLock.hint}
              onClick={() => void confirm('')}
            >
              Aufheben
            </Button>
          </span>
        </div>
      ) : (
        <div className="mt-2">
          <p className="text-body text-ink">
            {`Vorschlag von ${agentName(contact.proposedPostingGroupBy)}: `}
            <span className="font-semibold">{label(proposal)}</span>
          </p>
          {contact.proposedPostingGroupNote && (
            <p className="mt-0.5 text-caption text-ink-subtle">{contact.proposedPostingGroupNote}</p>
          )}
          {confirmed && (
            <p className="mt-0.5 text-caption text-ink-subtle">{`Bisher bestätigt: ${label(confirmed)}`}</p>
          )}
          <div className="mt-3 flex flex-wrap gap-2">
            <Button
              variant="secondary"
              size="sm"
              loading={busy}
              disabled={writeLock.locked}
              title={writeLock.hint}
              onClick={() => void confirm(proposal)}
            >
              Bestätigen
            </Button>
            <Button
              variant="quiet"
              size="sm"
              disabled={writeLock.locked}
              title={writeLock.hint}
              onClick={() => setChoosing(true)}
            >
              Andere Gruppe wählen
            </Button>
          </div>
        </div>
      )}

      {error && <Notice tone="negative" className="mt-3" text={error} />}
    </section>
  );
};

/** Anlegen und Bearbeiten; das Personenkonto vergibt Buchfink beim Anlegen. */
const ContactForm: React.FC<{
  contact: Partial<Contact>;
  onClose: () => void;
  onSaved: (saved: Contact) => Promise<void>;
}> = ({ contact, onClose, onSaved }) => {
  const writeLock = useWriteLock();
  const [draft, setDraft] = useState<Partial<Contact>>(contact);
  const [profiles, setProfiles] = useState<EInvoiceProfileInfo[]>([]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const isNew = !contact.id;
  const set = (patch: Partial<Contact>) => setDraft((prev) => ({ ...prev, ...patch }));
  const dirty = JSON.stringify(draft) !== JSON.stringify(contact);

  useEffect(() => {
    Api.getEInvoiceProfiles()
      .then(setProfiles)
      .catch(() => undefined);
  }, []);

  async function submit() {
    if (!draft.name?.trim() && !draft.company?.trim()) {
      setError('Ohne Namen lässt sich der Kontakt nicht zuordnen.');
      return;
    }
    setBusy(true);
    setError('');
    try {
      await onSaved(await Api.saveContact(draft));
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  }

  const profile = draft.eInvoiceProfile ?? 'zugferd_en16931';

  return (
    <Dialog
      open
      onOpenChange={(open) => !open && onClose()}
      title={isNew ? 'Neuer Kontakt' : `${contact.company || contact.name} bearbeiten`}
      width="max-w-2xl"
      dirty={dirty}
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Abbrechen
          </Button>
          <Button variant="primary" loading={busy} disabled={writeLock.locked} title={writeLock.hint} onClick={() => void submit()}>
            Speichern
          </Button>
        </>
      }
    >
      <FormGrid>
        <Field label="Art" hint={isNew ? undefined : 'Nach dem Anlegen fest'}>
          <Select
            items={[
              { value: 'vendor', label: 'Lieferant' },
              { value: 'customer', label: 'Kunde' },
            ]}
            value={draft.type ?? 'vendor'}
            disabled={!isNew}
            onValueChange={(type) => set({ type: type as ContactType })}
          />
        </Field>
        <Field label="Firma" hint="Mit Rechtsform" optional>
          <Input value={draft.company ?? ''} onChange={(e) => set({ company: e.target.value })} />
        </Field>
        <Field label="Name" hint="Person oder Ansprechpartner">
          <Input value={draft.name ?? ''} onChange={(e) => set({ name: e.target.value })} />
        </Field>
        <Field label="E-Mail" optional>
          <Input type="email" value={draft.email ?? ''} onChange={(e) => set({ email: e.target.value })} />
        </Field>
        <Field label="Straße und Hausnummer" className="md:col-span-2">
          <Input value={draft.street ?? ''} onChange={(e) => set({ street: e.target.value })} />
        </Field>
        <Field label="Postleitzahl">
          <Input className="code-num" value={draft.postalCode ?? ''} onChange={(e) => set({ postalCode: e.target.value })} />
        </Field>
        <Field label="Ort">
          <Input value={draft.city ?? ''} onChange={(e) => set({ city: e.target.value })} />
        </Field>
        <Field label="Land" hint="Zweistelliger Code">
          <Input
            maxLength={2}
            className="code-num"
            value={draft.countryCode ?? ''}
            onChange={(e) => set({ countryCode: e.target.value.toUpperCase() })}
          />
        </Field>
        <Field label="USt-IdNr." optional>
          <Input className="code-num" value={draft.vatId ?? ''} onChange={(e) => set({ vatId: e.target.value })} />
        </Field>
        <Field label="IBAN" optional>
          <Input
            className="code-num"
            value={draft.iban ?? ''}
            onChange={(e) => set({ iban: e.target.value.replace(/\s/g, '').toUpperCase() })}
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
      </FormGrid>

      <div className="mt-6 border-t border-line">
        <Disclosure variant="row" title="Weitere Angaben" context="E-Rechnung, Steuernummer, BIC, Freistellung">
          <FormGrid>
            <Field label="E-Rechnungsformat" hint="Für Ausgangsrechnungen">
              <Select
                items={profiles.map((p) => ({ value: p.profile, label: p.label }))}
                value={profile}
                onValueChange={(next) => set({ eInvoiceProfile: next })}
              />
            </Field>
            <Field
              label="Leitweg-ID"
              optional={profile !== 'xrechnung_cii'}
              hint={profile === 'xrechnung_cii' ? 'Bei XRechnung Pflicht' : undefined}
            >
              <Input
                className="code-num"
                value={draft.leitwegId ?? ''}
                onChange={(e) => set({ leitwegId: e.target.value })}
                onBlur={(e) => set({ leitwegId: e.target.value.trim() })}
              />
            </Field>
            <Field label="Steuernummer" optional>
              <Input value={draft.taxId ?? ''} onChange={(e) => set({ taxId: e.target.value })} />
            </Field>
            <Field label="BIC" optional>
              <Input className="code-num" value={draft.bic ?? ''} onChange={(e) => set({ bic: e.target.value.toUpperCase() })} />
            </Field>
            <Field
              label="Freistellungsbescheinigung"
              optional
              helpSummary="Nur bei Bauleistungen; Buchfink erinnert vor dem Ablauf."
              explain="Ohne gültige Bescheinigung ist bei einer Bauleistung ein Teil der Gegenleistung einzubehalten und abzuführen (§ 48 EStG). Buchfink rechnet diesen Abzug nicht, führt aber die Bescheinigung."
            >
              <Input
                className="code-num"
                value={draft.exemptionCertificateNumber ?? ''}
                onChange={(e) => set({ exemptionCertificateNumber: e.target.value })}
              />
            </Field>
            <Field label="Gültig bis" optional>
              <Input
                type="date"
                value={draft.exemptionCertificateValidUntil ?? ''}
                onChange={(e) => set({ exemptionCertificateValidUntil: e.target.value })}
              />
            </Field>
            <div className="md:col-span-2 flex flex-col gap-3">
              <Checkbox
                checked={draft.isPrivate ?? false}
                onCheckedChange={(checked) => set({ isPrivate: Boolean(checked) })}
                label="Privatperson"
                hint="Keine E-Rechnungspflicht"
              />
              <Checkbox
                checked={draft.isSmallBusiness ?? false}
                onCheckedChange={(checked) => set({ isSmallBusiness: Boolean(checked) })}
                label="Kleinunternehmer"
                hint="Rechnungen ohne Umsatzsteuer"
              />
            </div>
          </FormGrid>
        </Disclosure>
      </div>

      {error && <Notice tone="negative" className="mt-5" text={error} />}
    </Dialog>
  );
};

/**
 * Die Sperre nach einem Löschverlangen. Der Kontakt bleibt in Buchungen
 * stehen; Buchfink gibt die Antwort an die betroffene Person aus.
 */
const BlockDialog: React.FC<{
  contact: Contact;
  onClose: () => void;
  onBlocked: (answer: string) => Promise<void>;
}> = ({ contact, onClose, onBlocked }) => {
  const [reason, setReason] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  async function block() {
    if (!reason.trim()) {
      setError('Zur Sperre gehört der Grund; er steht im Änderungsprotokoll.');
      return;
    }
    setBusy(true);
    setError('');
    try {
      const result = await Api.blockContact(contact.id, reason.trim());
      await onBlocked(result.answer);
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog
      open
      onOpenChange={(open) => !open && onClose()}
      title={`${contact.company || contact.name} sperren`}
      width="max-w-lg"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Abbrechen
          </Button>
          <Button variant="danger" loading={busy} onClick={() => void block()}>
            Sperren
          </Button>
        </>
      }
    >
      <p className="text-body text-ink-muted mb-5">
        Der Kontakt verschwindet aus allen Auswahlen und lässt sich nicht mehr entsperren. In Buchungen und
        Rechnungen bleibt er stehen.
      </p>
      <Field
        label="Grund der Sperre"
        hint="Steht im Änderungsprotokoll"
        helpSummary="Ein gesperrter Kontakt ist nicht mehr wählbar; seine Buchungen bleiben."
        explain="Die Aufbewahrungspflicht geht dem Löschanspruch vor (Art. 17 Abs. 3 Buchst. b DSGVO, § 257 HGB, § 147 AO). Nach dem Sperren zeigt Buchfink die Antwort, die der betroffenen Person zusteht."
      >
        <Textarea rows={3} value={reason} placeholder="Löschverlangen vom …" onChange={(e) => setReason(e.target.value)} />
      </Field>
      {error && <Notice tone="negative" className="mt-5" text={error} />}
    </Dialog>
  );
};
