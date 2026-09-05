import React, { useEffect, useMemo, useState } from 'react';
import { HandCoins, MoreHorizontal, Plus, Trash2 } from 'lucide-react';
import type {
  Account,
  AdvanceItem,
  Contact,
  InvoiceGroup,
  TaxRate,
  UnitCode,
  VendorAdvance,
} from '../types';
import { TAX_RATE_NONE, TAX_RATE_REDUCED, TAX_RATE_STANDARD } from '../types';
import { Api } from '../services/api';
import { useWriteLock } from '../components/WriteLock';
import { formatCents, formatDate, formatTaxRate, parseCents } from '../utils/formatters';
import {
  Button,
  Dialog,
  EmptyState,
  Field,
  HelpPopover,
  Input,
  Menu,
  MenuItem,
  Notice,
  PageHeader,
  Section,
  Select,
  SkeletonRows,
  Stat,
  StatRow,
  StatusBadge,
  Table,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
  cn,
  toast,
  type Status,
} from '../components/ui';

/**
 * Anzahlungen.
 *
 * Der Rechnungsverbund hält zusammen, was fachlich ein Vorgang ist: der
 * vereinbarte Gesamtbetrag, die Abschlagsrechnungen mit ihrem Zahlungsstand und
 * die Schlussrechnung mit der Verrechnung. Er steht in einer eigenen Ansicht und
 * nicht bei den Ausgangsrechnungen, weil hier zwei Zeitpunkte auseinanderfallen,
 * die dort immer zusammenliegen: Die Abschlagsrechnung wird beim Ausstellen
 * nicht gebucht — die Steuer entsteht erst mit der Vereinnahmung
 * (§ 13 Abs. 1 Nr. 1 Buchst. a Satz 4 UStG).
 *
 * Die zweite Hälfte des Themas ist die Eingangsseite: eine geleistete Anzahlung
 * ist kein Aufwand, sondern ein eigener Bilanzposten, und ihre Vorsteuer hängt
 * an der Zahlung (§ 15 Abs. 1 Satz 1 Nr. 1 Satz 3 UStG).
 */

/**
 * Der Stand eines Abschlags in einem Wort (§11.3).
 *
 * Die Abschlagsrechnung ist ein offener Posten, und das Vokabular eines offenen
 * Postens kennt zwischen „Offen" und „Ausgeglichen" nichts weiter: mit der
 * Vereinnahmung ist er ausgeglichen. „Gebucht" gehört zur Buchung im Journal
 * und wäre hier ein geliehenes Wort für denselben Zustand.
 *
 * Die Verrechnung in der Schlussrechnung ist kein dritter Zustand des Postens —
 * sie setzt einen ausgeglichenen Abschlag ab (DeductibleInFinal verlangt die
 * Vereinnahmung) und steht deshalb als Vermerk in der Zeile, nicht im
 * Statuswort.
 */
const ADVANCE_STATUS = (advance: AdvanceItem): Status => {
  if (advance.cancelled) return 'storniert';
  if (advance.settledAt) return 'ausgeglichen';
  return 'offen';
};

const todayISO = () => new Date().toISOString().split('T')[0];

/**
 * Die Gründe, aus denen eine Aktion an einem Verbund oder Abschlag nicht
 * offensteht — `undefined`, solange sie offensteht.
 *
 * §10.4 verlangt für jeden deaktivierten Knopf eine Erklärung im `title`. Sie
 * stehen hier zusammen und nicht als Bedingung im JSX, weil `disabled` und
 * Begründung sonst getrennt gepflegt werden und auseinanderlaufen: gesperrt
 * ist genau, was einen Grund hat.
 */
function closedGroupHint(group: InvoiceGroup): string | undefined {
  return group.closed
    ? 'Der Verbund ist mit der Schlussrechnung abgeschlossen und nimmt keine weitere Rechnung mehr auf.'
    : undefined;
}

function settleBlockedReason(advance: AdvanceItem): string | undefined {
  if (advance.cancelled) return 'Die Abschlagsrechnung ist storniert.';
  if (advance.settledAt) return 'Der Abschlag ist bereits vereinnahmt.';
  return undefined;
}

function refundBlockedReason(advance: AdvanceItem): string | undefined {
  if (advance.cancelled) return 'Die Abschlagsrechnung ist storniert.';
  if (!advance.settledAt)
    return 'Zurückgezahlt wird nur, was vereinnahmt wurde; dieser Abschlag ist noch offen.';
  if (advance.settledInFinal)
    return 'Die Anzahlung ist in der Schlussrechnung abgesetzt und damit bereits verrechnet.';
  return undefined;
}

export const AdvancesPage: React.FC = () => {
  // Abschlag, Vereinnahmung und Schlussrechnung sind Buchungen; im Prüfermodus
  // bleibt die Ansicht lesbar (§10.4).
  const writeLock = useWriteLock();
  const [groups, setGroups] = useState<InvoiceGroup[]>([]);
  const [contacts, setContacts] = useState<Contact[]>([]);
  const [paymentAccounts, setPaymentAccounts] = useState<Account[]>([]);
  const [units, setUnits] = useState<UnitCode[]>([]);
  const [vendorAdvances, setVendorAdvances] = useState<VendorAdvance[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedId, setSelectedId] = useState<number | null>(null);

  const [creatingGroup, setCreatingGroup] = useState(false);
  const [advanceFor, setAdvanceFor] = useState<InvoiceGroup | null>(null);
  const [finalFor, setFinalFor] = useState<InvoiceGroup | null>(null);
  const [settling, setSettling] = useState<AdvanceItem | null>(null);
  const [refunding, setRefunding] = useState<AdvanceItem | null>(null);

  useEffect(() => {
    void load();
  }, []);

  async function load() {
    setLoading(true);
    try {
      const [groupList, contactList, accounts, unitList, vendorList] = await Promise.all([
        Api.getInvoiceGroups(),
        Api.getContacts(),
        Api.getPaymentAccounts(),
        Api.getUnitCodes(),
        Api.getOpenVendorAdvances(),
      ]);
      setGroups(groupList);
      setContacts(contactList.filter((c) => c.type === 'customer'));
      setPaymentAccounts(accounts);
      setUnits(unitList);
      setVendorAdvances(vendorList);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }

  const selected = useMemo(
    () => groups.find((g) => g.id === selectedId) ?? null,
    [groups, selectedId],
  );

  // Abgerechnet, vereinnahmt und offen kommen als `group.progress` aus dem
  // Backend (domain.GroupProgress). Diese Seite hat die Summen früher selbst
  // gebildet; welche Abschläge mitzählen, ist aber eine fachliche Regel und
  // keine Anzeige — ein stornierter fällt heraus (§ 14 Abs. 5 Satz 2 UStG
  // verlangt eine Rechnung), und vereinnahmt ist erst, was bezahlt wurde.
  // Hier wird nur noch über die Verbünde aufaddiert.
  const totals = useMemo(() => {
    let agreed = 0;
    let received = 0;
    let open = 0;
    for (const group of groups) {
      const p = group.progress;
      agreed += group.totalNet;
      received += p.receivedGross;
      if (!group.closed) open += p.openNet;
    }
    return { agreed, received, open };
  }, [groups]);

  const openGroups = groups.filter((g) => !g.closed);

  return (
    <div className="max-w-[1200px] mx-auto px-8 py-8">
      <PageHeader
        title="Anzahlungen"
        context={
          loading
            ? undefined
            : `${groups.length} Verbünde · ${openGroups.length} ohne Schlussrechnung`
        }
        action={
          <div className="flex items-center gap-2">
            <HelpPopover label="Erklärung zur Abschlagsrechnung">
              Eine Abschlagsrechnung wird beim Ausstellen nicht gebucht: Die Umsatzsteuer entsteht
              erst mit der Vereinnahmung. Sie steht bis dahin als offener Posten der Quelle
              „Abschlag" in Bank &amp; Zahlungen. Die Schlussrechnung setzt alle vereinnahmten
              Anzahlungen ab — wer das vergisst, weist die Steuer zweimal aus und schuldet den
              Mehrbetrag (§ 14c Abs. 1 UStG).
            </HelpPopover>
            <Button
              variant="primary"
              icon={<Plus className="w-4 h-4" strokeWidth={1.5} />}
              disabled={contacts.length === 0 || writeLock.locked}
              title={
                writeLock.hint ??
                (contacts.length === 0 ? 'Zuerst einen Kunden in den Stammdaten anlegen' : undefined)
              }
              onClick={() => setCreatingGroup(true)}
            >
              Neuer Verbund
            </Button>
          </div>
        }
      />

      {!loading && groups.length > 0 && (
        <StatRow className="mt-8">
          <Stat label="Vereinbart, netto" value={formatCents(totals.agreed)} />
          <Stat label="Vereinnahmt, brutto" value={formatCents(totals.received)} />
          <Stat
            label="Noch nicht abgerechnet"
            value={formatCents(totals.open)}
            context="netto, offene Verbünde"
          />
        </StatRow>
      )}

      <Section title="Rechnungsverbünde" divider={groups.length > 0} className="mt-8">
        {loading ? (
          <SkeletonRows rows={5} />
        ) : groups.length === 0 ? (
          <EmptyState
            icon={<HandCoins className="w-6 h-6" strokeWidth={1.5} />}
            title="Noch kein Rechnungsverbund angelegt"
            description="Ein Verbund hält Gesamtbetrag, Abschlagsrechnungen und Schlussrechnung eines Auftrags zusammen."
            action={
              // Sekundär, weil „Neuer Verbund" im Kopf der Seite schon steht:
              // genau eine Primäraktion je Ansicht (§10.4). Der zweite Weg zu
              // derselben Handlung bleibt hier, wo der Anwender hinsieht, tritt
              // aber nicht als zweiter Vorschlag auf.
              <Button
                variant="secondary"
                disabled={contacts.length === 0 || writeLock.locked}
                title={
                  writeLock.hint ??
                  (contacts.length === 0
                    ? 'Zuerst einen Kunden in den Stammdaten anlegen'
                    : undefined)
                }
                onClick={() => setCreatingGroup(true)}
              >
                Neuer Verbund
              </Button>
            }
          />
        ) : (
          <Table>
            <Thead>
              <Tr>
                <Th>Auftrag</Th>
                <Th>Auftraggeber</Th>
                <Th numeric className="w-36">
                  Vereinbart
                </Th>
                <Th numeric className="w-36">
                  Abgerechnet
                </Th>
                <Th numeric className="w-36">
                  Vereinnahmt
                </Th>
                <Th className="w-36">Stand</Th>
                <Th className="w-20" aria-label="Aktionen" />
              </Tr>
            </Thead>
            <Tbody>
              {groups.map((group) => {
                const p = group.progress;
                const contact = contacts.find((c) => c.id === group.contactId);
                return (
                  <Tr
                    key={group.id}
                    variant={selectedId === group.id ? 'selected' : 'default'}
                    className="group cursor-pointer"
                    onClick={() => setSelectedId(group.id === selectedId ? null : group.id)}
                  >
                    <Td className="max-w-[20rem] truncate">{group.title}</Td>
                    <Td className="text-ink-muted max-w-[16rem] truncate">
                      {contact?.name ?? '—'}
                    </Td>
                    <Td numeric>{formatCents(group.totalNet)}</Td>
                    <Td numeric className="text-ink-muted">
                      {formatCents(p.billedNet)}
                    </Td>
                    <Td numeric className="text-ink-muted">
                      {formatCents(p.receivedGross)}
                    </Td>
                    <Td>
                      {/* Der Verbund ist kein offener Posten: „Ausgeglichen"
                          verspräche einen Zahlungseingang, den die
                          Schlussrechnung gerade erst anfordert. Das Abzeichen
                          bleibt dem Verbund vorbehalten, der noch Rechnungen
                          aufnimmt; der abgeschlossene sagt es als Wort und
                          holt keine Aufmerksamkeit mehr (§11.3). */}
                      {group.closed ? (
                        <span className="text-body text-ink-subtle">Abgeschlossen</span>
                      ) : (
                        <StatusBadge status="offen" />
                      )}
                    </Td>
                    <Td className="pl-0">
                      <span className="flex items-center justify-end">
                        <Menu
                          trigger={
                            <Button
                              variant="quiet"
                              size="sm"
                              iconOnly
                              title="Aktionen zu diesem Verbund"
                              aria-label={`Aktionen zu ${group.title}`}
                            >
                              <MoreHorizontal className="w-4 h-4" strokeWidth={1.5} />
                            </Button>
                          }
                        >
                          <MenuItem
                            disabled={group.closed || writeLock.locked}
                            title={writeLock.hint ?? closedGroupHint(group)}
                            onClick={() => setAdvanceFor(group)}
                          >
                            Abschlagsrechnung ausstellen
                          </MenuItem>
                          <MenuItem
                            disabled={group.closed || writeLock.locked}
                            title={writeLock.hint ?? closedGroupHint(group)}
                            onClick={() => setFinalFor(group)}
                          >
                            Schlussrechnung ausstellen
                          </MenuItem>
                        </Menu>
                      </span>
                    </Td>
                  </Tr>
                );
              })}
            </Tbody>
          </Table>
        )}
      </Section>

      {selected && (
        <Section
          title={`Abschläge zu ${selected.title}`}
          context={`Steuersatz ${formatTaxRate(selected.taxRate)} · offen ${formatCents(
            selected.progress.openNet,
          )} von ${formatCents(selected.totalNet)}`}
        >
          {selected.advances.length === 0 ? (
            <EmptyState
              title="Noch keine Abschlagsrechnung in diesem Verbund"
              description="Der Abschlag wird ohne Buchung ausgestellt; gebucht wird er mit dem Zahlungseingang."
              action={
                <Button
                  variant="secondary"
                  disabled={selected.closed || writeLock.locked}
                  title={writeLock.hint ?? closedGroupHint(selected)}
                  onClick={() => setAdvanceFor(selected)}
                >
                  Abschlagsrechnung ausstellen
                </Button>
              }
            />
          ) : (
            <Table density="kompakt">
              <Thead>
                <Tr>
                  <Th className="w-36">Nummer</Th>
                  <Th className="w-28">Datum</Th>
                  <Th numeric className="w-32">
                    Netto
                  </Th>
                  <Th numeric className="w-32">
                    Brutto
                  </Th>
                  <Th className="w-28">Vereinnahmt</Th>
                  <Th className="w-40">Stand</Th>
                  <Th className="w-20" aria-label="Aktionen" />
                </Tr>
              </Thead>
              <Tbody>
                {selected.advances.map((advance) => (
                  <Tr
                    key={advance.id}
                    variant={advance.cancelled ? 'storno' : 'default'}
                    className="group"
                  >
                    <Td code>
                      {advance.invoiceNumber}
                      {advance.settledInFinal && (
                        <span className="block text-caption text-ink-subtle">
                          in Schlussrechnung verrechnet
                        </span>
                      )}
                    </Td>
                    <Td className="text-ink-subtle num">{formatDate(advance.invoiceDate)}</Td>
                    <Td numeric className="text-ink-muted">
                      {formatCents(advance.netAmount)}
                    </Td>
                    <Td numeric>{formatCents(advance.grossAmount)}</Td>
                    <Td className="text-ink-subtle num">
                      {advance.settledAt ? formatDate(advance.settledAt) : '—'}
                    </Td>
                    <Td>
                      <StatusBadge status={ADVANCE_STATUS(advance)} />
                    </Td>
                    <Td className="pl-0">
                      <span className="flex items-center justify-end">
                        <Menu
                          trigger={
                            <Button
                              variant="quiet"
                              size="sm"
                              iconOnly
                              title="Aktionen zu diesem Abschlag"
                              aria-label={`Aktionen zu ${advance.invoiceNumber}`}
                            >
                              <MoreHorizontal className="w-4 h-4" strokeWidth={1.5} />
                            </Button>
                          }
                        >
                          <MenuItem
                            disabled={
                              advance.cancelled || Boolean(advance.settledAt) || writeLock.locked
                            }
                            title={writeLock.hint ?? settleBlockedReason(advance)}
                            onClick={() => setSettling(advance)}
                          >
                            Zahlungseingang buchen
                          </MenuItem>
                          <MenuItem
                            disabled={
                              advance.cancelled ||
                              !advance.settledAt ||
                              advance.settledInFinal ||
                              writeLock.locked
                            }
                            title={writeLock.hint ?? refundBlockedReason(advance)}
                            onClick={() => setRefunding(advance)}
                          >
                            Anzahlung zurückzahlen
                          </MenuItem>
                        </Menu>
                      </span>
                    </Td>
                  </Tr>
                ))}
              </Tbody>
            </Table>
          )}
        </Section>
      )}

      <Section
        title="Geleistete Anzahlungen"
        context={
          loading
            ? undefined
            : `${vendorAdvances.length} noch nicht durch eine Schlussrechnung verrechnet`
        }
        action={
          <HelpPopover label="Erklärung zur geleisteten Anzahlung">
            Bezahlt ist etwas, geliefert nichts: Die geleistete Anzahlung ist kein Aufwand, sondern
            ein eigener Posten im Vermögen (§ 266 Abs. 2 A I 4, A II 4, B I 4 HGB). Erfasst wird sie
            im Belegweg mit dem Kennzeichen „Anzahlung"; die Schlussrechnung des Lieferanten setzt
            sie dort wieder ab.
          </HelpPopover>
        }
      >
        {loading ? (
          <SkeletonRows rows={3} />
        ) : vendorAdvances.length === 0 ? (
          <EmptyState
            title="Keine offene geleistete Anzahlung"
            description="Im Belegweg macht das Kennzeichen „Anzahlung“ einen Beleg dazu."
          />
        ) : (
          <Table density="kompakt">
            <Thead>
              <Tr>
                <Th className="w-40">Beleg</Th>
                <Th className="w-28">Bezahlt am</Th>
                <Th className="w-28">Konto</Th>
                <Th numeric className="w-32">
                  Netto
                </Th>
                <Th numeric className="w-32">
                  Vorsteuer
                </Th>
                <Th numeric className="w-32">
                  Brutto
                </Th>
              </Tr>
            </Thead>
            <Tbody>
              {vendorAdvances.map((advance) => (
                <Tr key={advance.id}>
                  <Td code>{advance.documentNumber}</Td>
                  <Td className="text-ink-subtle num">{formatDate(advance.paidAt)}</Td>
                  <Td code>{advance.account}</Td>
                  <Td numeric className="text-ink-muted">
                    {formatCents(advance.netAmount)}
                  </Td>
                  <Td numeric className="text-ink-muted">
                    {formatCents(advance.taxAmount)}
                  </Td>
                  <Td numeric>{formatCents(advance.grossAmount)}</Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
      </Section>

      <GroupDialog
        open={creatingGroup}
        contacts={contacts}
        onClose={() => setCreatingGroup(false)}
        onDone={async (title) => {
          setCreatingGroup(false);
          toast.success(`Rechnungsverbund „${title}" angelegt.`);
          await load();
        }}
      />

      <AdvanceDialog
        group={advanceFor}
        onClose={() => setAdvanceFor(null)}
        onDone={async (number) => {
          setAdvanceFor(null);
          toast.success(`Abschlagsrechnung ${number} ausgestellt.`);
          await load();
        }}
      />

      <FinalDialog
        group={finalFor}
        units={units}
        onClose={() => setFinalFor(null)}
        onDone={async (number) => {
          setFinalFor(null);
          toast.success(`Schlussrechnung ${number} ausgestellt.`);
          await load();
        }}
      />

      <SettleDialog
        advance={settling}
        paymentAccounts={paymentAccounts}
        onClose={() => setSettling(null)}
        onDone={async () => {
          setSettling(null);
          await load();
        }}
      />

      <RefundDialog
        advance={refunding}
        paymentAccounts={paymentAccounts}
        onClose={() => setRefunding(null)}
        onDone={async () => {
          setRefunding(null);
          await load();
        }}
      />
    </div>
  );
};

// -------------------------------------------------------------------------

const GroupDialog: React.FC<{
  open: boolean;
  contacts: Contact[];
  onClose: () => void;
  onDone: (title: string) => void;
}> = ({ open, contacts, onClose, onDone }) => {
  const writeLock = useWriteLock();
  const [contactId, setContactId] = useState(0);
  const [title, setTitle] = useState('');
  const [totalNet, setTotalNet] = useState('');
  const [taxRate, setTaxRate] = useState<TaxRate>(TAX_RATE_STANDARD);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!open) return;
    setContactId(contacts[0]?.id ?? 0);
    setTitle('');
    setTotalNet('');
    setTaxRate(TAX_RATE_STANDARD);
    setError(null);
  }, [open, contacts]);

  async function submit() {
    const net = parseCents(totalNet);
    if (!title.trim()) {
      setError('Ohne Bezeichnung lässt sich der Verbund später nicht zuordnen.');
      return;
    }
    if (net === null || net <= 0) {
      setError('Der vereinbarte Gesamtbetrag ist der Rahmen der Abschläge und muss größer als null sein.');
      return;
    }
    setBusy(true);
    try {
      await Api.createInvoiceGroup({ contactId, title, totalNet: net, taxRate });
      onDone(title);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => !next && onClose()}
      title="Neuer Rechnungsverbund"
      width="max-w-xl"
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
            Verbund anlegen
          </Button>
        </>
      }
    >
      <Field label="Auftraggeber">
        <Select
          items={contacts.map((c) => ({
            value: c.id,
            label: `${c.name} · Debitor ${c.ledgerAccount}`,
          }))}
          value={contactId}
          onValueChange={setContactId}
        />
      </Field>

      <Field label="Bezeichnung des Auftrags" className="mt-4" error={error ?? undefined}>
        <Input
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder="Umbau Ladenlokal"
        />
      </Field>

      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 mt-4">
        <Field
          label="Vereinbarter Gesamtbetrag"
          hint="netto"
          explain="Er ist die Obergrenze der Abschläge und die Bemessungsgrundlage der Schlussrechnung. Buchfink weist einen Abschlag zurück, der die Summe darüber hebt."
        >
          <Input
            align="right"
            inputMode="decimal"
            placeholder="0,00"
            value={totalNet}
            onChange={(e) => setTotalNet(e.target.value)}
          />
        </Field>
        <Field label="Steuersatz">
          <Select
            items={[TAX_RATE_STANDARD, TAX_RATE_REDUCED, TAX_RATE_NONE].map((rate) => ({
              value: rate,
              label: formatTaxRate(rate),
            }))}
            value={taxRate}
            onValueChange={setTaxRate}
          />
        </Field>
      </div>
    </Dialog>
  );
};

// -------------------------------------------------------------------------

const AdvanceDialog: React.FC<{
  group: InvoiceGroup | null;
  onClose: () => void;
  onDone: (invoiceNumber: string) => void;
}> = ({ group, onClose, onDone }) => {
  const writeLock = useWriteLock();
  const [date, setDate] = useState(todayISO());
  const [description, setDescription] = useState('');
  const [net, setNet] = useState('');
  const [receivedAt, setReceivedAt] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!group) return;
    setDate(todayISO());
    setDescription('');
    setNet('');
    setReceivedAt('');
    setError(null);
  }, [group]);

  async function submit() {
    const amount = parseCents(net);
    if (amount === null || amount <= 0) {
      setError('Der Abschlagsbetrag muss größer als null sein.');
      return;
    }
    setBusy(true);
    try {
      const invoice = await Api.issueAdvanceInvoice({
        groupId: group!.id,
        date,
        description,
        net: amount,
        paymentReceivedAt: receivedAt || undefined,
      });
      onDone(invoice.invoiceNumber);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  const remaining = group?.progress.openNet ?? 0;

  return (
    <Dialog
      open={group !== null}
      onOpenChange={(next) => !next && onClose()}
      title="Abschlagsrechnung ausstellen"
      width="max-w-xl"
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
            Ausstellen
          </Button>
        </>
      }
    >
      {group && (
        <>
          <p className="text-body text-ink-muted">
            {group.title} · noch nicht abgerechnet{' '}
            <span className="num text-ink">{formatCents(remaining)}</span>
            <HelpPopover label="Erklärung zur Abschlagsrechnung">
              Die Abschlagsrechnung trägt den Typcode 386 und wird beim Ausstellen nicht gebucht.
              Sie erscheint als offener Posten der Quelle „Abschlag"; mit dem Zahlungseingang bucht
              Buchfink gegen die erhaltenen Anzahlungen und die Umsatzsteuer.
            </HelpPopover>
          </p>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 mt-4">
            <Field label="Rechnungsdatum">
              <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
            </Field>
            <Field
              label="Abschlagsbetrag"
              hint="netto"
              error={error ?? undefined}
            >
              <Input
                align="right"
                inputMode="decimal"
                placeholder="0,00"
                value={net}
                onChange={(e) => setNet(e.target.value)}
              />
            </Field>
          </div>

          <Field label="Bezeichnung" optional className="mt-4">
            <Input
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder={`Abschlag auf ${group.title}`}
            />
          </Field>

          <Field
            label="Vereinnahmt am"
            optional
            className="mt-4 max-w-sm"
            explain="Steht der Zeitpunkt der Vereinnahmung beim Ausstellen schon fest — etwa bei einer Rechnung nach der Zahlung —, tritt er auf dem Dokument an die Stelle des Leistungszeitpunkts (§ 14 Abs. 4 Nr. 6 UStG). Vor dem Geldeingang bleibt das Feld leer."
          >
            <Input type="date" value={receivedAt} onChange={(e) => setReceivedAt(e.target.value)} />
          </Field>
        </>
      )}
    </Dialog>
  );
};

// -------------------------------------------------------------------------

interface FinalItem {
  description: string;
  quantity: string;
  unit: string;
  unitPrice: string;
  taxRate: TaxRate;
}

const FINAL_GRID =
  'grid grid-cols-[minmax(0,1fr)_5rem_7rem_7rem_6rem_2rem] gap-2 items-center';

const FinalDialog: React.FC<{
  group: InvoiceGroup | null;
  units: UnitCode[];
  onClose: () => void;
  onDone: (invoiceNumber: string) => void;
}> = ({ group, units, onClose, onDone }) => {
  const writeLock = useWriteLock();
  const [date, setDate] = useState(todayISO());
  const [serviceFrom, setServiceFrom] = useState(todayISO());
  const [serviceTo, setServiceTo] = useState(todayISO());
  const [dueDays, setDueDays] = useState('');
  const [items, setItems] = useState<FinalItem[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  // Die Gesamtleistung ist vorbelegt: der vereinbarte Gesamtbetrag des
  // Verbunds. Die Schlussrechnung rechnet über die ganze Leistung ab und setzt
  // die Anzahlungen davon ab — nicht über den Rest.
  useEffect(() => {
    if (!group) return;
    setDate(todayISO());
    setServiceFrom(todayISO());
    setServiceTo(todayISO());
    setDueDays('');
    setError(null);
    setItems([
      {
        description: group.title,
        quantity: '1',
        unit: 'C62',
        unitPrice: formatCents(group.totalNet, ''),
        taxRate: group.taxRate,
      },
    ]);
  }, [group]);

  function update(index: number, patch: Partial<FinalItem>) {
    setItems((prev) => prev.map((item, i) => (i === index ? { ...item, ...patch } : item)));
  }

  async function submit() {
    setBusy(true);
    try {
      const invoice = await Api.issueFinalInvoice({
        groupId: group!.id,
        date,
        serviceDateFrom: serviceFrom,
        serviceDateTo: serviceTo,
        terms: {
          dueDays: Number.parseInt(dueDays, 10) || 0,
          discountPermille: 0,
          discountDays: 0,
        },
        items: items.map((item, index) => ({
          position: index + 1,
          description: item.description,
          quantityMilli: Math.round((Number(item.quantity.replace(',', '.')) || 0) * 1000),
          unit: item.unit,
          unitPrice: parseCents(item.unitPrice) ?? 0,
          taxRate: item.taxRate,
        })),
      });
      onDone(invoice.invoiceNumber);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  // Abgesetzt wird, was vereinnahmt und nicht storniert ist
  // (AdvanceItem.DeductibleInFinal). Die Summe dazu rechnet diese Seite nicht
  // selbst nach: `progress.receivedGross` ist dieselbe Zahl aus dem Backend,
  // und die Regel, welcher Abschlag mitzählt, gehört an eine Stelle.
  const deducted = (group?.advances ?? []).filter((a) => !a.cancelled && a.settledAt);
  const prepaid = group?.progress?.receivedGross ?? 0;

  return (
    <Dialog
      open={group !== null}
      onOpenChange={(next) => !next && onClose()}
      title="Schlussrechnung ausstellen"
      width="max-w-3xl"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Abbrechen
          </Button>
          <Button
            variant="primary"
            loading={busy}
            disabled={items.length === 0 || writeLock.locked}
            title={writeLock.hint}
            onClick={submit}
          >
            Ausstellen und buchen
          </Button>
        </>
      }
    >
      {group && (
        <>
          <p className="text-body text-ink-muted">
            {group.title} · abzusetzen{' '}
            <span className="num text-ink">{formatCents(prepaid)}</span> aus {deducted.length}{' '}
            {deducted.length === 1 ? 'Anzahlung' : 'Anzahlungen'}
            <HelpPopover label="Erklärung zur Schlussrechnung">
              Die Schlussrechnung rechnet über die gesamte Leistung ab und setzt die vereinnahmten
              Anzahlungen ab (BT-113, § 14 Abs. 5 Satz 2 UStG). Gebucht wird der Gesamtbetrag gegen
              Erlös und Umsatzsteuer, dazu die Auflösung der erhaltenen Anzahlungen samt ihrer
              Steuer; als Forderung bleibt der Restbetrag. Ein noch offener Abschlag muss vorher
              vereinnahmt oder storniert sein.
            </HelpPopover>
          </p>

          <div className="grid grid-cols-1 sm:grid-cols-4 gap-4 mt-4">
            <Field label="Rechnungsdatum">
              <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
            </Field>
            <Field label="Leistung von">
              <Input
                type="date"
                value={serviceFrom}
                onChange={(e) => setServiceFrom(e.target.value)}
              />
            </Field>
            <Field label="Leistung bis">
              <Input type="date" value={serviceTo} onChange={(e) => setServiceTo(e.target.value)} />
            </Field>
            <Field label="Zahlungsziel" hint="Tage" optional>
              <Input
                align="right"
                inputMode="numeric"
                placeholder="14"
                value={dueDays}
                onChange={(e) => setDueDays(e.target.value)}
              />
            </Field>
          </div>

          <div className="mt-6 pt-6 border-t border-line">
            <div className="flex items-center justify-between mb-3">
              <h3 className="text-label text-ink-muted">Positionen der Gesamtleistung</h3>
              <Button
                variant="quiet"
                size="sm"
                icon={<Plus className="w-3.5 h-3.5" strokeWidth={1.5} />}
                onClick={() =>
                  setItems((prev) => [
                    ...prev,
                    {
                      description: '',
                      quantity: '1',
                      unit: 'C62',
                      unitPrice: '',
                      taxRate: group.taxRate,
                    },
                  ])
                }
              >
                Position hinzufügen
              </Button>
            </div>

            <div className={cn(FINAL_GRID, 'text-caption text-ink-subtle mb-1')}>
              <span>Bezeichnung</span>
              <span className="text-right">Menge</span>
              <span>Einheit</span>
              <span className="text-right">Einzelpreis</span>
              <span>USt</span>
              <span />
            </div>

            <div className="flex flex-col gap-2">
              {items.map((item, index) => (
                <div key={index} className={FINAL_GRID}>
                  <Input
                    value={item.description}
                    onChange={(e) => update(index, { description: e.target.value })}
                    aria-label={`Bezeichnung der Position ${index + 1}`}
                  />
                  <Input
                    align="right"
                    inputMode="decimal"
                    value={item.quantity}
                    onChange={(e) => update(index, { quantity: e.target.value })}
                    aria-label={`Menge der Position ${index + 1}`}
                  />
                  <Select
                    items={units.map((u) => ({ value: u.code, label: u.label }))}
                    value={item.unit}
                    onValueChange={(unit) => update(index, { unit })}
                    aria-label={`Einheit der Position ${index + 1}`}
                  />
                  <Input
                    align="right"
                    inputMode="decimal"
                    placeholder="0,00"
                    value={item.unitPrice}
                    onChange={(e) => update(index, { unitPrice: e.target.value })}
                    aria-label={`Einzelpreis der Position ${index + 1}`}
                  />
                  <Select
                    items={[TAX_RATE_STANDARD, TAX_RATE_REDUCED, TAX_RATE_NONE].map((rate) => ({
                      value: rate,
                      label: formatTaxRate(rate),
                    }))}
                    value={item.taxRate}
                    onValueChange={(taxRate) => update(index, { taxRate })}
                    aria-label={`Umsatzsteuersatz der Position ${index + 1}`}
                  />
                  {/* Hinzugefügt und nicht mehr gewollt: ohne diesen Knopf
                      bleibt die Position stehen, und der Anwender müsste den
                      Dialog verwerfen und von vorn beginnen. Die letzte
                      Position bleibt — eine Schlussrechnung ohne Position gibt
                      es nicht. */}
                  {items.length > 1 ? (
                    <Button
                      variant="quiet"
                      size="sm"
                      iconOnly
                      title="Position entfernen"
                      aria-label={`Position ${index + 1} entfernen`}
                      onClick={() => setItems((prev) => prev.filter((_, i) => i !== index))}
                    >
                      <Trash2 className="w-4 h-4" strokeWidth={1.5} />
                    </Button>
                  ) : (
                    <span />
                  )}
                </div>
              ))}
            </div>

          </div>

          {/* Der Fehler kommt aus dem Backend und gehört deshalb auf die
              Hinweisfläche in Rosé, unmittelbar über die Aktionen (§10.4) —
              nicht als kleingedruckte Zeile zwischen die Positionen. Was die
              Schlussrechnung ablehnt (ein offener Abschlag, eine überschrittene
              Summe), ist eine fachliche Aussage und keine Fußnote. */}
          {error && <Notice tone="negative" text={error} className="mt-6" />}
        </>
      )}
    </Dialog>
  );
};

// -------------------------------------------------------------------------

const SettleDialog: React.FC<{
  advance: AdvanceItem | null;
  paymentAccounts: Account[];
  onClose: () => void;
  onDone: () => void;
}> = ({ advance, paymentAccounts, onClose, onDone }) => {
  const writeLock = useWriteLock();
  const [date, setDate] = useState(todayISO());
  const [account, setAccount] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!advance) return;
    setDate(todayISO());
    setAccount(paymentAccounts[0]?.number ?? '');
    setError(null);
  }, [advance, paymentAccounts]);

  async function submit() {
    setBusy(true);
    try {
      await Api.settleAdvance({
        advanceId: advance!.invoiceId,
        paymentDate: date,
        paymentAccount: account,
      });
      toast.success(`Zahlungseingang auf ${advance!.invoiceNumber} gebucht.`);
      onDone();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog
      open={advance !== null}
      onOpenChange={(next) => !next && onClose()}
      title="Zahlungseingang auf den Abschlag"
      width="max-w-lg"
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
            Vereinnahmung buchen
          </Button>
        </>
      }
    >
      {advance && (
        <>
          <p className="text-body text-ink-muted">
            <span className="code-num text-ink">{advance.invoiceNumber}</span> über{' '}
            <span className="num text-ink">{formatCents(advance.grossAmount)}</span>
            <HelpPopover label="Erklärung zur Vereinnahmung">
              Mit der Zahlung entsteht die Umsatzsteuer. Gebucht wird das Zahlungsmittel gegen die
              erhaltenen, versteuerten Anzahlungen und die Umsatzsteuer; der Voranmeldungszeitraum
              folgt dem Zahlungsdatum. Kommt das Geld über den Kontoauszug, gehört die Zuordnung in
              Bank &amp; Zahlungen — dann stammen Konto und Datum aus dem Auszug.
            </HelpPopover>
          </p>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 mt-4">
            <Field label="Zahlungsdatum" error={error ?? undefined}>
              <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
            </Field>
            <Field label="Zahlungsmittel">
              <Select
                items={paymentAccounts.map((a) => ({
                  value: a.number,
                  label: `${a.number} · ${a.name}`,
                }))}
                value={account}
                onValueChange={setAccount}
              />
            </Field>
          </div>
        </>
      )}
    </Dialog>
  );
};

// -------------------------------------------------------------------------

const RefundDialog: React.FC<{
  advance: AdvanceItem | null;
  paymentAccounts: Account[];
  onClose: () => void;
  onDone: () => void;
}> = ({ advance, paymentAccounts, onClose, onDone }) => {
  const writeLock = useWriteLock();
  const [date, setDate] = useState(todayISO());
  const [account, setAccount] = useState('');
  const [reason, setReason] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!advance) return;
    setDate(todayISO());
    setAccount(paymentAccounts[0]?.number ?? '');
    setReason('');
    setError(null);
  }, [advance, paymentAccounts]);

  async function submit() {
    if (!reason.trim()) {
      setError('Ohne Grund lässt sich die Rückzahlung später nicht nachvollziehen.');
      return;
    }
    setBusy(true);
    try {
      await Api.refundAdvance({
        advanceId: advance!.invoiceId,
        refundDate: date,
        paymentAccount: account,
        reason,
      });
      toast.success(`Rückzahlung zu ${advance!.invoiceNumber} gebucht.`);
      onDone();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog
      open={advance !== null}
      onOpenChange={(next) => !next && onClose()}
      title="Anzahlung zurückzahlen"
      width="max-w-lg"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Abbrechen
          </Button>
          <Button
            variant="danger"
            loading={busy}
            disabled={writeLock.locked}
            title={writeLock.hint}
            onClick={submit}
          >
            Rückzahlung buchen
          </Button>
        </>
      }
    >
      {advance && (
        <>
          <p className="text-body text-ink-muted">
            <span className="code-num text-ink">{advance.invoiceNumber}</span> über{' '}
            <span className="num text-ink">{formatCents(advance.grossAmount)}</span>
            <HelpPopover label="Erklärung zur Rückzahlung">
              Die Steuer einer Anzahlung entsteht mit der Vereinnahmung. Sie zu berichtigen setzt
              nach § 17 Abs. 2 Nr. 2 UStG voraus, dass das Entgelt zurückgezahlt worden ist —
              deshalb steht die Rückzahlung vor dem Storno einer bezahlten Abschlagsrechnung.
              Gebucht wird sie im Zeitraum der Rückzahlung.
            </HelpPopover>
          </p>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 mt-4">
            <Field label="Datum der Rückzahlung">
              <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
            </Field>
            <Field label="Zahlungsmittel">
              <Select
                items={paymentAccounts.map((a) => ({
                  value: a.number,
                  label: `${a.number} · ${a.name}`,
                }))}
                value={account}
                onValueChange={setAccount}
              />
            </Field>
          </div>

          <Field label="Grund der Rückzahlung" className="mt-4" error={error ?? undefined}>
            <Input
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="Auftrag storniert"
            />
          </Field>
        </>
      )}
    </Dialog>
  );
};
