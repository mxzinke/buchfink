import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { FileDown } from 'lucide-react';
import { Api } from '../../services/api';
import { useWriteLock } from '../../components/WriteLock';
import { formatCents, formatDate, formatDateTime, formatShortHash, parseCents } from '../../utils/formatters';
import { useAnchorScroll, type NavigateFn } from '../../navigation';
import type {
  Declaration,
  DeclarationInput,
  DeclarationInputSource,
  DeclarationKind,
  DeclarationLine,
  DeclarationLineStatus,
  DeclarationOwner,
  DeclarationValue,
  DeclarationView,
  NonDeductibleReport,
  Obligation,
} from '../../types';
import {
  Button,
  DetailList,
  DetailPanel,
  Dialog,
  Disclosure,
  Field,
  Help,
  Input,
  Notice,
  PageHeader,
  Section,
  Select,
  SkeletonRows,
  Stat,
  StatRow,
  StatusBadge,
  Switch,
  Table,
  Tbody,
  Td,
  Textarea,
  Th,
  Thead,
  Tr,
  cn,
  toast,
} from '../../components/ui';
import {
  ObligationDecisionDialog,
  ObligationState,
  isManual,
  message,
  obligationKindOf,
  todayISO,
  loadErrorText,
} from './shared';
import { EBilanzView } from './EBilanzView';
import { splitNorms } from '../../utils/norms';

/**
 * Die Jahreserklärungen: Körperschaftsteuer, Gewerbesteuer, Umsatzsteuer und
 * E-Bilanz. Die Liste zeigt Frist und Stand; die Erklärung selbst geht auf
 * Klick auf, Vordruck für Vordruck, mit der Herkunft jeder Zeile in der
 * Seitenleiste. Buchfink übermittelt nicht; die Werte sind eine Eingabehilfe
 * für Mein ELSTER.
 */

type Selection = DeclarationKind | 'e-bilanz';

interface Entry {
  value: Selection;
  title: string;
  obligationKind: string;
  channel: string;
}

const ENTRIES: Entry[] = [
  { value: 'kst', title: 'Körperschaftsteuererklärung', obligationKind: 'kst-erklaerung', channel: 'Mein ELSTER' },
  { value: 'gewst', title: 'Gewerbesteuererklärung', obligationKind: 'gewst-erklaerung', channel: 'Mein ELSTER' },
  { value: 'ust_jahr', title: 'Umsatzsteuererklärung', obligationKind: 'ust-erklaerung', channel: 'Mein ELSTER' },
  { value: 'e-bilanz', title: 'E-Bilanz', obligationKind: 'e-bilanz', channel: 'E-Bilanz-Dienst' },
];

/** Stand der Vorauszahlungen, eine überfällige auch im eingeklappten Zustand. */
function prepaymentContext(prepayments: Obligation[]): string {
  const paid = prepayments.filter((o) => o.status === 'done').length;
  const overdue = prepayments.filter((o) => o.status === 'overdue').length;
  const text = `${paid} von ${prepayments.length} bezahlt`;
  return overdue > 0 ? `${text} · ${overdue} überfällig` : text;
}

/** Welche Erklärung ein Ziel meint: die Art selbst oder eine Pflicht. */
function selectionOf(id: string | undefined): Selection | null {
  if (!id) return null;
  if (ENTRIES.some((entry) => entry.value === id)) return id as Selection;
  switch (obligationKindOf(id)) {
    case 'kst-erklaerung':
    case 'kst-vorauszahlung':
      return 'kst';
    case 'gewst-erklaerung':
    case 'gewst-vorauszahlung':
      return 'gewst';
    case 'ust-erklaerung':
      return 'ust_jahr';
    case 'e-bilanz':
      return 'e-bilanz';
    default:
      return null;
  }
}

const DECLARATION_KINDS = ['kst-erklaerung', 'gewst-erklaerung', 'ust-erklaerung', 'e-bilanz'];

export interface DeclarationsTabProps {
  year: number;
  obligations: Obligation[];
  /** Der Pflichtenkalender ließ sich nicht laden. */
  obligationsError?: string;
  onObligationsChanged: () => Promise<void>;
  id?: string;
  onNavigate: NavigateFn;
}

export const DeclarationsTab: React.FC<DeclarationsTabProps> = ({
  year,
  obligations,
  obligationsError,
  onObligationsChanged,
  id,
  onNavigate,
}) => {
  const writeLock = useWriteLock();
  const selected = selectionOf(id);
  const [previousOpen, setPreviousOpen] = useState<Obligation[]>([]);
  const [deciding, setDeciding] = useState<Obligation | null>(null);
  useAnchorScroll(selected ? 'erklaerung' : undefined, selected ?? undefined);

  useEffect(() => {
    Api.getObligations(year - 1)
      .then((list) =>
        setPreviousOpen(
          list.filter(
            (o) => DECLARATION_KINDS.includes(o.kind) && o.status !== 'done' && o.status !== 'waived',
          ),
        ),
      )
      .catch(() => setPreviousOpen([]));
  }, [year]);

  const byKind = (kind: string) => obligations.find((o) => o.kind === kind && o.fiscalYear === year);
  const prepayments = obligations.filter(
    (o) => o.kind === 'kst-vorauszahlung' || o.kind === 'gewst-vorauszahlung',
  );
  const select = (value: Selection) => onNavigate({ area: 'closings', tab: 'declarations', id: value });

  return (
    <div>
      <PageHeader
        title={`Steuererklärungen ${year}`}
        context="Eingabehilfe für Mein ELSTER, Zeile für Zeile"
        helpSummary="Buchfink rechnet die Erklärungen aus den Büchern. Übertragen und absenden tun Sie oder Claude."
        explain={
          <>
            Die Werte stehen so, wie sie im Vordruck einzutragen sind: Abschnitt, Zeile, Kennzahl.
            Hinter jeder Zeile steht, woher der Wert kommt. Was die Bücher nicht wissen, etwa einen
            Verlustvortrag aus dem Bescheid, tragen Sie oder Claude als Angabe ein. Nach dem
            Absenden in Mein ELSTER bestätigen Sie die Übermittlung mit dem Transferticket (§ 149
            AO, § 31 KStG, § 14a GewStG, § 18 Abs. 3 UStG).
          </>
        }
      />

      {obligationsError && (
        <Notice tone="negative" className="mt-6" text={loadErrorText(obligationsError)} />
      )}

      {previousOpen.length > 0 && (
        <Notice
          className="mt-6"
          text={`Für ${year - 1} ${previousOpen.length === 1 ? 'ist eine Erklärung' : `sind ${previousOpen.length} Erklärungen`} noch offen.`}
          action={
            <Button
              variant="secondary"
              size="sm"
              onClick={() => onNavigate({ area: 'closings', tab: 'declarations', year: year - 1 })}
            >
              {`Zu ${year - 1}`}
            </Button>
          }
        />
      )}

      <div className="mt-6">
        <Table>
          <Thead>
            <Tr>
              <Th>Erklärung</Th>
              <Th className="w-80">Stand</Th>
              <Th className="w-40">Abgabe über</Th>
            </Tr>
          </Thead>
          <Tbody>
            {ENTRIES.map((entry) => {
              const o = byKind(entry.obligationKind);
              return (
                <Tr key={entry.value} variant={selected === entry.value ? 'selected' : 'default'}>
                  <Td>
                    <button
                      type="button"
                      aria-current={selected === entry.value ? 'true' : undefined}
                      className="text-left text-ink underline-offset-2 hover:underline"
                      onClick={() => select(entry.value)}
                    >
                      {entry.title}
                    </button>
                  </Td>
                  <Td>{o ? <ObligationState obligation={o} /> : <span className="text-caption text-ink-subtle">—</span>}</Td>
                  <Td className="text-ink-muted">{entry.channel}</Td>
                </Tr>
              );
            })}
          </Tbody>
        </Table>
      </div>

      {prepayments.length > 0 && (
        <Disclosure
          title="Vorauszahlungen"
          context={prepaymentContext(prepayments)}
        >
          <ul className="flex flex-col">
            {prepayments.map((o) => (
              <li key={o.key} className="flex items-center gap-4 py-2 border-t border-line first:border-t-0">
                <span className="flex-1 text-body text-ink">{o.title}</span>
                <ObligationState obligation={o} />
                {isManual(o) ? (
                  <Button
                    variant="quiet"
                    size="sm"
                    disabled={writeLock.locked}
                    title={writeLock.hint}
                    onClick={() => setDeciding(o)}
                  >
                    Erledigt vermerken
                  </Button>
                ) : (
                  <span className="w-[8.5rem]" aria-hidden="true" />
                )}
              </li>
            ))}
          </ul>
        </Disclosure>
      )}

      {selected && (
        <div id="erklaerung" className="scroll-mt-6">
          {selected === 'e-bilanz' ? (
            <EBilanzView
              year={year}
              obligation={byKind('e-bilanz')}
              onObligationsChanged={onObligationsChanged}
            />
          ) : (
            <DeclarationDetail
              key={`${selected}-${year}`}
              kind={selected}
              year={year}
              onSubmitted={onObligationsChanged}
            />
          )}
        </div>
      )}

      <ObligationDecisionDialog
        obligation={deciding}
        onClose={() => setDeciding(null)}
        onDone={onObligationsChanged}
      />
    </div>
  );
};

// -------------------------------------------------------------------------
// Werte und Stände einer Zeile

/** Ein Wert so, wie er in den Vordruck gehört. */
export function formatDeclarationValue(value: DeclarationValue, unit?: string, rounding?: string): string {
  if (value.cent !== undefined && value.cent !== null) {
    if (unit === 'prozent') return `${(value.cent / 100).toLocaleString('de-DE')} %`;
    if (unit === 'euro_voll' && (rounding === 'abschneiden' || value.cent % 100 === 0)) {
      const whole = Math.trunc(value.cent / 100);
      return `${whole < 0 ? '−' : ''}${Math.abs(whole).toLocaleString('de-DE')} €`;
    }
    return formatCents(value.cent);
  }
  if (value.ja !== undefined && value.ja !== null) return value.ja ? 'ja' : 'nein';
  if (value.datum) return formatDate(value.datum);
  if (value.text !== undefined && value.text !== null) {
    return unit === 'prozent' && !value.text.includes('%') ? `${value.text} %` : value.text;
  }
  return '';
}

const LINE_STATUS: Record<DeclarationLineStatus, { label: string; mark: string }> = {
  auto: { label: 'aus den Büchern', mark: 'bg-positive' },
  eingabe: { label: 'Angabe', mark: 'bg-positive' },
  vorgeschlagen: { label: 'Vorschlag', mark: 'bg-attention' },
  offen: { label: 'offen', mark: 'bg-negative' },
  entfaellt: { label: 'entfällt', mark: 'bg-ink-faint' },
};

const OWNER: Record<DeclarationOwner, string> = {
  buecher: 'die Bücher',
  agent_rueckfrage: 'Claude, mit Rückfrage bei Ihnen',
  mensch: 'Sie',
};

const ORIGIN: Record<string, string> = {
  konten: 'Salden der Konten',
  buchungen: 'Einzelne Buchungen',
  rechnung: 'Rechnung aus anderen Zeilen',
  stammdaten: 'Stammdaten des Unternehmens',
  eingabe: 'Angabe',
  voranmeldungen: 'Übermittelte Voranmeldungen',
  anlagenkartei: 'Anlagenkartei',
  abschluss: 'Jahresabschluss',
  keine: 'Noch ohne Wert',
};

const SOURCE_LABELS: Record<DeclarationInputSource, string> = {
  bescheid: 'Bescheid',
  beschluss: 'Beschluss',
  beleg: 'Beleg',
  mensch: 'eigene Angabe',
  buecher: 'Bücher',
};

const LineMark: React.FC<{ line: DeclarationLine }> = ({ line }) => (
  <span className="inline-flex items-center gap-1.5 text-caption text-ink-subtle">
    <span className={cn('mark-diamond', LINE_STATUS[line.status].mark)} aria-hidden="true" />
    {line.status === 'auto' && line.herkunft.art === 'stammdaten'
      ? 'aus den Stammdaten'
      : LINE_STATUS[line.status].label}
  </span>
);

// -------------------------------------------------------------------------
// Eine Erklärung

const DeclarationDetail: React.FC<{
  kind: DeclarationKind;
  year: number;
  onSubmitted: () => Promise<void>;
}> = ({ kind, year, onSubmitted }) => {
  const writeLock = useWriteLock();
  const [view, setView] = useState<DeclarationView | null>(null);
  const [error, setError] = useState('');
  const [allLines, setAllLines] = useState(false);
  const [lineKey, setLineKey] = useState<string | null>(null);
  const [inputLine, setInputLine] = useState<DeclarationLine | null>(null);
  const [submitOpen, setSubmitOpen] = useState(false);
  const [exporting, setExporting] = useState(false);

  const load = useCallback(async () => {
    try {
      setView(await Api.getDeclaration(kind, year));
      setError('');
    } catch (e) {
      setView(null);
      setError(message(e));
    }
  }, [kind, year]);

  useEffect(() => {
    void load();
  }, [load]);

  const lines = useMemo(() => {
    const out = new Map<string, DeclarationLine>();
    for (const form of view?.declaration.formulare ?? []) {
      for (const section of form.abschnitte) for (const line of section.zeilen) out.set(line.schluessel, line);
    }
    return out;
  }, [view]);

  if (error) return <Notice tone="negative" className="mt-8" text={error} />;
  if (!view) {
    return (
      <div className="mt-8">
        <SkeletonRows rows={8} />
      </div>
    );
  }

  const { overview, declaration: d } = view;
  const submitted = d.uebermittlung;
  const applicable = d.formulare.filter((form) => form.erforderlich !== 'entfaellt');
  const waived = d.formulare.filter((form) => form.erforderlich === 'entfaellt');
  const blocking = d.befunde.filter((f) => f.schwere === 'blockiert');
  const others = d.befunde.filter((f) => f.schwere !== 'blockiert');
  const selectedLine = lineKey ? lines.get(lineKey) ?? null : null;
  const inputOf = (key: string): DeclarationInput | undefined => d.eingaben.find((i) => i.schluessel === key);
  const canConfirm = (line: DeclarationLine) => {
    const input = inputOf(line.schluessel);
    return line.eingabe_moeglich || (input !== undefined && !input.bestaetigt);
  };

  /**
   * Ein Vorschlag wird bestätigt: liegt eine Angabe von Claude vor, bestätigt
   * die App sie; sonst übernimmt die App den Wert als eigene Angabe.
   */
  async function confirmProposal(line: DeclarationLine) {
    try {
      const input = inputOf(line.schluessel);
      if (input && !input.bestaetigt) {
        await Api.confirmDeclarationInput(kind, year, line.schluessel);
      } else {
        await Api.setDeclarationInput({
          kind,
          year,
          key: line.schluessel,
          value: line.wert,
          source: 'mensch',
        });
      }
      toast.success(`„${line.bezeichnung}“ bestätigt.`);
      await load();
    } catch (e) {
      toast.error(message(e));
    }
  }

  async function removeInput(line: DeclarationLine) {
    try {
      await Api.deleteDeclarationInput(kind, year, line.schluessel);
      toast.success(`Angabe zu „${line.bezeichnung}“ zurückgenommen.`);
      await load();
    } catch (e) {
      toast.error(message(e));
    }
  }

  async function exportJSON() {
    setExporting(true);
    try {
      const result = await Api.exportDeclarationJSON(kind, year);
      if (result) toast.success(`Gespeichert unter ${result.pfad}`);
    } catch (e) {
      toast.error(message(e));
    } finally {
      setExporting(false);
    }
  }

  return (
    <Section
      title={d.titel}
      context={`Vordruck ${d.vordruckjahr} · gerechnet am ${formatDateTime(d.grundlage.gerechnet_am)} · ${d.grundlage.buchungen} Buchungen`}
      action={
        <div className="flex items-center gap-2">
          <Button
            variant="quiet"
            icon={<FileDown className="w-4 h-4" strokeWidth={1.5} />}
            loading={exporting}
            onClick={() => void exportJSON()}
          >
            Als JSON
          </Button>
          {submitted ? (
            <StatusBadge status="erledigt" />
          ) : (
            <Button
              variant={blocking.length > 0 ? 'secondary' : 'primary'}
              disabled={writeLock.locked}
              title={writeLock.hint}
              onClick={() => setSubmitOpen(true)}
            >
              Übermittlung bestätigen
            </Button>
          )}
        </div>
      }
    >
      {submitted ? (
        <p className="text-body text-ink-muted mb-6">
          {`Übermittelt am ${formatDate(submitted.am)} · Transferticket ${submitted.transferticket}`}
        </p>
      ) : blocking.length > 0 ? (
        <Notice
          className="mb-6"
          text={`${blocking.length} ${blocking.length === 1 ? 'Punkt sollte' : 'Punkte sollten'} vor der Abgabe geklärt sein. Sie stehen unten unter „Zu klären“ und „Befunde“.`}
        />
      ) : null}

      {overview.ergebnis.length > 0 && (
        <StatRow className="mb-6">
          {overview.ergebnis.slice(0, 4).map((figure) => {
            const line = lines.get(figure.schluessel);
            return (
              <Stat
                key={figure.schluessel}
                label={figure.kurz || figure.bezeichnung}
                value={formatDeclarationValue(figure.wert, line?.einheit, line?.rundung) || '—'}
              />
            );
          })}
        </StatRow>
      )}

      {overview.zu_klaeren.length > 0 && (
        <div className="mb-6">
          <h3 className="text-label text-ink-muted mb-2">Zu klären</h3>
          <ul className="flex flex-col">
            {overview.zu_klaeren.map((pending) => {
              const line = lines.get(pending.schluessel);
              return (
                <li
                  key={pending.schluessel}
                  className="flex items-center gap-4 py-2 border-t border-line first:border-t-0"
                >
                  <button
                    type="button"
                    className="flex-1 min-w-0 text-left"
                    onClick={() => setLineKey(pending.schluessel)}
                  >
                    <span className="block text-body text-ink truncate" title={pending.bezeichnung}>
                      {splitNorms(pending.bezeichnung).text}
                    </span>
                    <span className="block text-caption text-ink-subtle">
                      {pending.status === 'vorgeschlagen'
                        ? `Vorschlag ${formatDeclarationValue(pending.wert, line?.einheit, line?.rundung)} · wartet auf Ihre Bestätigung`
                        : `fehlt · ${pending.zustaendig === 'mensch' ? 'Angabe von Ihnen' : pending.zustaendig === 'agent_rueckfrage' ? 'Claude fragt bei Ihnen nach' : 'aus den Büchern'}`}
                    </span>
                  </button>
                  {pending.status === 'vorgeschlagen' && line && canConfirm(line) ? (
                    <Button
                      variant="secondary"
                      size="sm"
                      disabled={writeLock.locked}
                      title={writeLock.hint}
                      onClick={() => void confirmProposal(line)}
                    >
                      Bestätigen
                    </Button>
                  ) : line?.eingabe_moeglich ? (
                    <Button
                      variant="secondary"
                      size="sm"
                      disabled={writeLock.locked}
                      title={writeLock.hint}
                      onClick={() => setInputLine(line)}
                    >
                      Angabe eintragen
                    </Button>
                  ) : null}
                </li>
              );
            })}
          </ul>
        </div>
      )}

      {d.befunde.length > 0 && (
        <Disclosure
          title="Befunde"
          context={`${blocking.length} blockierend · ${others.length} Hinweise`}
          defaultOpen={false}
          className="mt-0"
        >
          <ul className="flex flex-col">
            {[...blocking, ...others].map((finding, index) => (
              <li
                key={`${finding.code}-${finding.zeile ?? index}`}
                className="flex items-start gap-3 py-2 border-t border-line first:border-t-0"
              >
                <span
                  className={cn(
                    'w-24 shrink-0 text-caption',
                    finding.schwere === 'blockiert' ? 'text-negative-text' : 'text-attention-text',
                  )}
                >
                  {finding.schwere === 'blockiert' ? 'Blockierend' : finding.schwere === 'warnung' ? 'Warnung' : 'Hinweis'}
                </span>
                <span className="flex-1 text-body text-ink-muted">{finding.text}</span>
                {finding.zeile && lines.has(finding.zeile) && (
                  <Button variant="quiet" size="sm" className="-my-1" onClick={() => setLineKey(finding.zeile!)}>
                    Zur Zeile
                  </Button>
                )}
              </li>
            ))}
          </ul>
        </Disclosure>
      )}

      <div className="mt-8 flex items-center justify-between gap-4">
        <h3 className="text-heading text-ink">Vordrucke</h3>
        <Switch checked={allLines} onCheckedChange={setAllLines} label="Auch leere Zeilen" />
      </div>
      {applicable.map((form) => {
        const rows = form.abschnitte
          .map((section) => ({
            section,
            lines: section.zeilen.filter(
              (line) =>
                allLines ||
                line.status === 'offen' ||
                line.status === 'vorgeschlagen' ||
                formatDeclarationValue(line.wert) !== '' && !isZero(line.wert),
            ),
          }))
          .filter((group) => group.lines.length > 0);
        const s = form.uebersicht;
        return (
          <Disclosure
            key={form.id}
            title={form.titel}
            context={
              form.erforderlich === 'kontrolle'
                ? 'Kontrollrechnung, nicht zu übertragen'
                : `${s.mit_wert} Zeilen mit Wert${s.offen > 0 ? ` · ${s.offen} offen` : ''}${s.vorgeschlagen > 0 ? ` · ${s.vorgeschlagen} ${s.vorgeschlagen === 1 ? 'Vorschlag' : 'Vorschläge'}` : ''}${form.erforderlich === 'bei_bedarf' ? ' · bei Bedarf' : ''}`
            }
            className="mt-4 pt-4"
          >
            {rows.length === 0 ? (
              <p className="text-body text-ink-subtle">Keine Zeile mit Wert.</p>
            ) : (
              rows.map(({ section, lines: sectionLines }) => (
                <div key={section.id} className="mb-5 last:mb-0">
                  <h4 className="text-label text-ink-muted mb-1.5">{section.titel}</h4>
                  <Table density="kompakt">
                    <Thead>
                      <Tr>
                        <Th className="w-16">Zeile</Th>
                        <Th className="w-16">Kz</Th>
                        <Th>Bezeichnung</Th>
                        <Th numeric className="w-40">
                          Wert
                        </Th>
                        <Th className="w-32">Stand</Th>
                      </Tr>
                    </Thead>
                    <Tbody>
                      {sectionLines.map((line) => (
                        <Tr key={line.schluessel} variant={lineKey === line.schluessel ? 'selected' : 'default'}>
                          <Td className="num text-ink-subtle">{line.zeile ?? ''}</Td>
                          <Td code>{line.kz ?? ''}</Td>
                          <Td className="whitespace-normal">
                            <button
                              type="button"
                              className="text-left text-ink underline-offset-2 hover:underline"
                              onClick={() => setLineKey(line.schluessel)}
                            >
                              {line.bezeichnung}
                            </button>
                          </Td>
                          <Td numeric className={cn(line.vorlaeufig && 'text-ink-muted')}>
                            {formatDeclarationValue(line.wert, line.einheit, line.rundung)}
                          </Td>
                          <Td>
                            <LineMark line={line} />
                          </Td>
                        </Tr>
                      ))}
                    </Tbody>
                  </Table>
                </div>
              ))
            )}
          </Disclosure>
        );
      })}

      {waived.length > 0 && (
        <Disclosure
          title="Entfallende Vordrucke"
          context={waived.map((form) => form.titel.replace(/\s*\(.*\)$/, '')).join(', ')}
          className="mt-4 pt-4"
        >
          <ul className="flex flex-col">
            {waived.map((form) => (
              <li key={form.id} className="py-2 border-t border-line first:border-t-0">
                <span className="block text-body text-ink">{form.titel}</span>
                {form.grund && <span className="block text-caption text-ink-subtle">{form.grund}</span>}
              </li>
            ))}
          </ul>
        </Disclosure>
      )}

      {kind === 'ust_jahr' && (d.abstimmung ?? []).length > 0 && (
        <Disclosure
          title="Abstimmung mit den Voranmeldungen"
          context={`${(d.abstimmung ?? []).filter((row) => row.differenz !== 0).length} Abweichungen`}
          className="mt-4 pt-4"
        >
          <Table density="kompakt">
            <Thead>
              <Tr>
                <Th className="w-16">Kz</Th>
                <Th>Bezeichnung</Th>
                <Th numeric className="w-36">
                  Jahr
                </Th>
                <Th numeric className="w-36">
                  Voranmeldungen
                </Th>
                <Th numeric className="w-32">
                  Differenz
                </Th>
              </Tr>
            </Thead>
            <Tbody>
              {(d.abstimmung ?? []).map((row) => (
                <Tr key={`${row.kz_voranmeldung}-${row.feld}`}>
                  <Td code>{row.kz_voranmeldung}</Td>
                  <Td className="whitespace-normal">
                    {row.bezeichnung}
                    {row.grund && <span className="block text-caption text-ink-subtle">{row.grund}</span>}
                  </Td>
                  <Td numeric>{formatCents(row.jahr)}</Td>
                  <Td numeric className="text-ink-muted">
                    {formatCents(row.voranmeldungen)}
                  </Td>
                  <Td numeric className={row.differenz !== 0 ? 'text-negative-text' : 'text-ink-faint'}>
                    {formatCents(row.differenz)}
                  </Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        </Disclosure>
      )}

      {kind === 'kst' && <NonDeductibleKartei year={year} />}

      <LinePanel
        line={selectedLine}
        input={selectedLine ? inputOf(selectedLine.schluessel) : undefined}
        lines={lines}
        onClose={() => setLineKey(null)}
        onSelect={setLineKey}
        onInput={(line) => setInputLine(line)}
        onConfirm={(line) => void confirmProposal(line)}
        onRemove={(line) => void removeInput(line)}
      />

      <InputDialog
        line={inputLine}
        kind={kind}
        year={year}
        input={inputLine ? inputOf(inputLine.schluessel) : undefined}
        onClose={() => setInputLine(null)}
        onDone={load}
      />

      <SubmitDialog
        open={submitOpen}
        declaration={d}
        onClose={() => setSubmitOpen(false)}
        onDone={async () => {
          await load();
          await onSubmitted();
        }}
      />
    </Section>
  );
};

function isZero(value: DeclarationValue): boolean {
  if (value.cent !== undefined && value.cent !== null) return value.cent === 0;
  if (value.ja !== undefined && value.ja !== null) return !value.ja;
  if (value.datum) return false;
  return !value.text;
}

/** Die Seitenleiste einer Zeile: Wert, Herkunft und was mit ihr zu tun ist. */
const LinePanel: React.FC<{
  line: DeclarationLine | null;
  input?: DeclarationInput;
  lines: Map<string, DeclarationLine>;
  onClose: () => void;
  onSelect: (key: string) => void;
  onInput: (line: DeclarationLine) => void;
  onConfirm: (line: DeclarationLine) => void;
  onRemove: (line: DeclarationLine) => void;
}> = ({ line, input, lines, onClose, onSelect, onInput, onConfirm, onRemove }) => {
  const writeLock = useWriteLock();
  const origin = line?.herkunft;
  const position = line ? [line.zeile ? `Zeile ${line.zeile}` : '', line.kz ? `Kz ${line.kz}` : ''].filter(Boolean).join(' · ') : '';
  return (
    <DetailPanel
      open={line !== null}
      onOpenChange={(open) => !open && onClose()}
      title={line?.bezeichnung ?? ''}
      context={position || undefined}
      meta={line ? <LineMark line={line} /> : undefined}
      width="lg"
      footer={
        line && (
          <>
            {input && (
              <Button variant="quiet" disabled={writeLock.locked} title={writeLock.hint} onClick={() => onRemove(line)}>
                Angabe zurücknehmen
              </Button>
            )}
            {line.eingabe_moeglich && (
              <Button variant="secondary" disabled={writeLock.locked} title={writeLock.hint} onClick={() => onInput(line)}>
                {input ? 'Angabe ändern' : 'Angabe eintragen'}
              </Button>
            )}
            {line.status === 'vorgeschlagen' && (line.eingabe_moeglich || (input && !input.bestaetigt)) && (
              <Button variant="primary" disabled={writeLock.locked} title={writeLock.hint} onClick={() => onConfirm(line)}>
                Vorschlag bestätigen
              </Button>
            )}
          </>
        )
      }
    >
      {line && origin && (
        <div className="flex flex-col gap-6">
          <DetailList
            items={[
              {
                label: 'Wert',
                value: formatDeclarationValue(line.wert, line.einheit, line.rundung) || '—',
                numeric: true,
              },
              ...(line.einheit === 'euro_voll' && line.wert.cent !== undefined && line.wert.cent % 100 !== 0
                ? [{ label: 'Genau', value: formatCents(line.wert.cent), numeric: true }]
                : []),
              { label: 'Herkunft', value: ORIGIN[origin.art] ?? origin.art },
              { label: 'Zuständig', value: OWNER[line.zustaendig] },
              ...(line.vorlaeufig ? [{ label: 'Vorläufig', value: 'rechnet aus einer offenen Zeile' }] : []),
            ]}
          />
          {origin.erlaeuterung && <p className="text-body text-ink-muted">{origin.erlaeuterung}</p>}
          {origin.rechnung && (
            <p className="text-body text-ink-muted">
              <span className="text-ink-subtle">Rechnung: </span>
              {origin.rechnung}
            </p>
          )}
          {(origin.konten ?? []).length > 0 && (
            <Table density="kompakt">
              <Thead>
                <Tr>
                  <Th className="w-20">Konto</Th>
                  <Th>Bezeichnung</Th>
                  <Th numeric className="w-32">
                    Saldo
                  </Th>
                  <Th numeric className="w-32">
                    Beitrag
                  </Th>
                </Tr>
              </Thead>
              <Tbody>
                {(origin.konten ?? []).map((part) => (
                  <Tr key={part.konto}>
                    <Td code>{part.konto}</Td>
                    <Td className="whitespace-normal">
                      {part.bezeichnung}
                      {part.anteil_nenner > 0 && part.anteil_zaehler !== part.anteil_nenner && (
                        <span className="text-ink-subtle">{` · ${part.anteil_zaehler}/${part.anteil_nenner}`}</span>
                      )}
                    </Td>
                    <Td numeric className="text-ink-muted">
                      {formatCents(part.saldo)}
                    </Td>
                    <Td numeric>{formatCents(part.beitrag)}</Td>
                  </Tr>
                ))}
              </Tbody>
            </Table>
          )}
          {(origin.aus_zeilen ?? []).length > 0 && (
            <div>
              <h3 className="text-label text-ink-muted mb-1.5">Aus den Zeilen</h3>
              <ul className="flex flex-col">
                {(origin.aus_zeilen ?? []).map((key) => {
                  const source = lines.get(key);
                  return (
                    <li key={key} className="flex items-baseline justify-between gap-4 py-1.5 border-t border-line first:border-t-0">
                      <button
                        type="button"
                        className="text-left text-body text-ink underline-offset-2 hover:underline"
                        onClick={() => onSelect(key)}
                      >
                        {source?.bezeichnung ?? key}
                      </button>
                      <span className="text-body num">
                        {source ? formatDeclarationValue(source.wert, source.einheit, source.rundung) : ''}
                      </span>
                    </li>
                  );
                })}
              </ul>
            </div>
          )}
          {(origin.buchungen ?? []).length > 0 && (
            <p className="text-body text-ink-muted">{`${origin.buchungen!.length} Buchungen tragen zu diesem Wert bei.`}</p>
          )}
          {input && (
            <DetailList
              items={[
                { label: 'Quelle der Angabe', value: SOURCE_LABELS[input.quelle] ?? input.quelle },
                ...(input.notiz ? [{ label: 'Notiz', value: input.notiz }] : []),
                { label: 'Erfasst', value: `${formatDateTime(input.erfasst_am)} · ${input.erfasst_von}` },
                {
                  label: 'Bestätigt',
                  value: input.bestaetigt
                    ? `${input.bestaetigt_am ? formatDateTime(input.bestaetigt_am) : 'ja'}${input.bestaetigt_von ? ` · ${input.bestaetigt_von}` : ''}`
                    : 'noch nicht',
                },
              ]}
            />
          )}
          {(line.hinweise ?? []).map((note) => (
            <Notice key={note} text={note} />
          ))}
          {((line.rechtsgrundlage ?? []).length > 0 || line.pruefvermerk.quelle) && (
            <Disclosure title="Rechtsgrundlage und Prüfvermerk" variant="row" className="border-b-0 -ml-2">
              <div className="flex flex-col gap-2 text-body text-ink-muted">
                {(line.rechtsgrundlage ?? []).length > 0 && <p>{(line.rechtsgrundlage ?? []).join('; ')}</p>}
                <p>
                  {line.pruefvermerk.am_vordruck_geprueft
                    ? 'Zeile und Kennzahl sind am amtlichen Vordruck geprüft.'
                    : 'Zeile und Kennzahl sind noch nicht am amtlichen Vordruck geprüft.'}
                  {line.pruefvermerk.quelle ? ` Quelle: ${line.pruefvermerk.quelle}` : ''}
                </p>
              </div>
            </Disclosure>
          )}
        </div>
      )}
    </DetailPanel>
  );
};

/** Eine Angabe zu einer Zeile: Wert, Quelle, Notiz. */
const InputDialog: React.FC<{
  line: DeclarationLine | null;
  kind: DeclarationKind;
  year: number;
  input?: DeclarationInput;
  onClose: () => void;
  onDone: () => Promise<void>;
}> = ({ line, kind, year, input, onClose, onDone }) => {
  const writeLock = useWriteLock();
  const [raw, setRaw] = useState('');
  const [source, setSource] = useState<DeclarationInputSource>('bescheid');
  const [note, setNote] = useState('');
  const [fieldError, setFieldError] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const unit = line?.einheit;
  const amount = unit === 'euro_voll' || unit === 'euro_cent';

  useEffect(() => {
    if (!line) return;
    const current = input?.wert ?? line.wert;
    setRaw(
      current.cent !== undefined && current.cent !== null
        ? formatCents(current.cent, '').trim()
        : current.ja !== undefined && current.ja !== null
          ? current.ja
            ? 'ja'
            : 'nein'
          : current.datum ?? current.text ?? '',
    );
    setSource(input?.quelle ?? 'bescheid');
    setNote(input?.notiz ?? '');
    setFieldError('');
    setError('');
  }, [line, input]);

  function value(): DeclarationValue | null {
    if (amount) {
      const cents = parseCents(raw);
      if (cents === null) return null;
      return { cent: cents };
    }
    if (unit === 'ja_nein') return { ja: raw === 'ja' };
    if (unit === 'datum') return raw ? { datum: raw } : null;
    return raw.trim() ? { text: raw.trim() } : null;
  }

  async function save() {
    if (!line) return;
    const next = value();
    if (!next) {
      setFieldError(amount ? 'Erwartet wird ein Betrag, etwa 12.500,00.' : 'Der Wert fehlt.');
      return;
    }
    if (unit === 'euro_voll' && next.cent !== undefined && next.cent % 100 !== 0) {
      setFieldError('Diese Zeile nimmt volle Euro ohne Cent.');
      return;
    }
    setBusy(true);
    try {
      await Api.setDeclarationInput({ kind, year, key: line.schluessel, value: next, source, note: note.trim() });
      toast.success(`Angabe zu „${line.bezeichnung}“ gespeichert.`);
      await onDone();
      onClose();
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog
      open={line !== null}
      onOpenChange={(open) => !open && onClose()}
      title={line?.bezeichnung ?? ''}
      width="max-w-lg"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Abbrechen
          </Button>
          <Button variant="primary" loading={busy} disabled={writeLock.locked} title={writeLock.hint} onClick={() => void save()}>
            Angabe speichern
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        <Field label="Wert" hint={unit === 'euro_voll' ? 'volle Euro' : undefined} error={fieldError || undefined}>
          {unit === 'ja_nein' ? (
            <Select
              items={[
                { value: 'ja', label: 'ja' },
                { value: 'nein', label: 'nein' },
              ]}
              value={raw || 'nein'}
              onValueChange={(next) => setRaw(String(next))}
            />
          ) : (
            <Input
              type={unit === 'datum' ? 'date' : 'text'}
              align={amount ? 'right' : 'left'}
              inputMode={amount ? 'decimal' : undefined}
              value={raw}
              onChange={(e) => {
                setRaw(e.target.value);
                setFieldError('');
              }}
            />
          )}
        </Field>
        <Field label="Woher der Wert stammt">
          <Select
            items={(['bescheid', 'beschluss', 'beleg', 'mensch'] as DeclarationInputSource[]).map((value) => ({
              value,
              label: SOURCE_LABELS[value],
            }))}
            value={source}
            onValueChange={(next) => setSource(next as DeclarationInputSource)}
          />
        </Field>
        <Field label="Notiz" optional>
          <Textarea rows={2} value={note} onChange={(e) => setNote(e.target.value)} />
        </Field>
        {error && <Notice tone="negative" text={error} />}
      </div>
    </Dialog>
  );
};

/** Die Übermittlung in Mein ELSTER bestätigen: Datum, Transferticket, Zahlenstand. */
const SubmitDialog: React.FC<{
  open: boolean;
  declaration: Declaration;
  onClose: () => void;
  onDone: () => Promise<void>;
}> = ({ open, declaration, onClose, onDone }) => {
  const writeLock = useWriteLock();
  const [date, setDate] = useState(todayISO());
  const [ticket, setTicket] = useState('');
  const [reason, setReason] = useState('');
  const [fieldError, setFieldError] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const blocking = declaration.befunde.filter((f) => f.schwere === 'blockiert').length;

  useEffect(() => {
    if (!open) return;
    setDate(todayISO());
    setTicket('');
    setReason('');
    setFieldError('');
    setError('');
  }, [open]);

  async function submit() {
    if (!ticket.trim()) {
      setFieldError('Ohne Transferticket gibt es keine Bestätigung.');
      return;
    }
    if (blocking > 0 && !reason.trim()) {
      setFieldError('');
      setError('Die Erklärung hat blockierende Befunde. Bitte begründen Sie die Übermittlung.');
      return;
    }
    setBusy(true);
    try {
      await Api.submitDeclaration({
        kind: declaration.art,
        year: declaration.zeitraum.jahr,
        submittedAt: date,
        transferTicket: ticket.trim(),
        figuresHash: declaration.zahlen_hash,
        overrideReason: reason.trim() || undefined,
      });
      toast.success(`${declaration.titel} als übermittelt vermerkt.`);
      await onDone();
      onClose();
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => !next && onClose()}
      title={`${declaration.titel} übermittelt`}
      width="max-w-lg"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Abbrechen
          </Button>
          <Button variant="primary" loading={busy} disabled={writeLock.locked} title={writeLock.hint} onClick={() => void submit()}>
            Übermittlung bestätigen
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        <p className="text-body text-ink">
          Bestätigt wird der Zahlenstand, der hier steht. Ändern sich die Bücher danach, ist eine
          berichtigte Erklärung nötig.
        </p>
        <div className="grid grid-cols-2 gap-4">
          <Field label="Übermittelt am">
            <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
          </Field>
          <Field label="Transferticket" hint="aus Mein ELSTER" error={fieldError || undefined}>
            <Input
              className="code-num"
              value={ticket}
              onChange={(e) => {
                setTicket(e.target.value);
                setFieldError('');
              }}
            />
          </Field>
        </div>
        <p className="text-caption text-ink-subtle">{`Zahlenstand ${formatShortHash(declaration.zahlen_hash)}`}</p>
        {blocking > 0 && (
          <Field label="Begründung" hint="steht im Protokoll">
            <Textarea rows={2} value={reason} onChange={(e) => setReason(e.target.value)} />
          </Field>
        )}
        {error && <Notice tone="negative" text={error} />}
      </div>
    </Dialog>
  );
};

/**
 * Die nicht abziehbaren Betriebsausgaben und Geschenke des Jahres. Sie gehen
 * als außerbilanzielle Korrektur in die Körperschaftsteuer ein; die Kartei
 * der Geschenke je Empfänger ist eine eigene Aufzeichnungspflicht.
 */
const NonDeductibleKartei: React.FC<{ year: number }> = ({ year }) => {
  const [report, setReport] = useState<NonDeductibleReport | null>(null);
  const [error, setError] = useState('');
  const [open, setOpen] = useState(false);
  useEffect(() => {
    if (!open || report) return;
    Api.getNonDeductibleReport(year)
      .then(setReport)
      .catch((e) => setError(message(e)));
  }, [open, report, year]);
  return (
    <Disclosure
      title="Nicht abziehbare Ausgaben und Geschenke"
      context="Kartei zu den außerbilanziellen Korrekturen"
      open={open}
      onOpenChange={setOpen}
      className="mt-4 pt-4"
    >
      {error ? (
        <Notice tone="negative" text={error} />
      ) : !report ? (
        <SkeletonRows rows={3} />
      ) : (
        <div className="flex flex-col gap-5">
          <Table density="kompakt">
            <Thead>
              <Tr>
                <Th>Art</Th>
                <Th numeric className="w-36">
                  Abziehbar
                </Th>
                <Th numeric className="w-36">
                  Nicht abziehbar
                </Th>
              </Tr>
            </Thead>
            <Tbody>
              {report.categories.map((row) => (
                <Tr key={row.key}>
                  <Td className="whitespace-normal">{row.label}</Td>
                  <Td numeric className="text-ink-muted">
                    {formatCents(row.deductibleAmount)}
                  </Td>
                  <Td numeric>{formatCents(row.nonDeductibleAmount)}</Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
          {report.recipients.length > 0 && (
            <div>
              <h4 className="text-label text-ink-muted mb-1.5 flex items-center">
                Geschenke je Empfänger
                <Help summary="Geschenke an Geschäftspartner sind nur bis zu einer Grenze je Empfänger und Jahr abziehbar." label="Erklärung zur Geschenkegrenze">
                  {`Die Freigrenze liegt bei ${formatCents(report.giftLimit)} je Empfänger und Wirtschaftsjahr (§ 4 Abs. 5 Satz 1 Nr. 1 EStG). Wird sie überschritten, ist alles an diesen Empfänger nicht abziehbar. Die Aufzeichnung je Empfänger verlangt § 4 Abs. 7 EStG.`}
                </Help>
              </h4>
              <Table density="kompakt">
                <Tbody>
                  {report.recipients.map((r) => (
                    <Tr key={r.recipientKey}>
                      <Td>{r.recipientName}</Td>
                      <Td className="w-40 text-caption text-ink-subtle">{`${r.bookings.length} Buchungen`}</Td>
                      <Td numeric className={cn('w-36', r.overLimit && 'text-negative-text')}>
                        {formatCents(r.total)}
                      </Td>
                    </Tr>
                  ))}
                </Tbody>
              </Table>
            </div>
          )}
        </div>
      )}
    </Disclosure>
  );
};
