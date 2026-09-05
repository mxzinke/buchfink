import React, { useState } from 'react';
import { BookOpen, Landmark } from 'lucide-react';
import { FoundationPostingPreview, FoundationState } from '../types';
import { Api } from '../services/api';
import { formatCents, formatDate, formatSide } from '../utils/formatters';
import {
  Button,
  ConfirmDialog,
  Dialog,
  HelpPopover,
  Section,
  Stat,
  StatRow,
  Table,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
  cn,
  toast,
} from './ui';

interface FoundationSectionProps {
  state: FoundationState;
  onChanged: () => void | Promise<void>;
}

/**
 * Die Gründung, solange sie läuft.
 *
 * Der Abschnitt steht auf der Fristenseite und verschwindet mit der Eintragung
 * wieder — er ist kein zweiter Dauerplatz in der Anwendung, sondern eine Phase.
 * Was er zeigt, ist die eine Zahl, die zwischen Beurkundung und Eintragung
 * niemand im Blick hat: um wie viel das Reinvermögen hinter dem Stammkapital
 * zurückbleibt und wer davon welchen Teil schuldet.
 */
export const FoundationSection: React.FC<FoundationSectionProps> = ({ state, onChanged }) => {
  const [preview, setPreview] = useState<FoundationPostingPreview | null>(null);
  const [registering, setRegistering] = useState(false);
  const [busy, setBusy] = useState(false);

  const [registerDate, setRegisterDate] = useState('');
  const [registerCourt, setRegisterCourt] = useState('');
  const [registerNumber, setRegisterNumber] = useState('');

  const foundation = state.foundation;
  const unterbilanz = state.unterbilanz;
  const anmeldung = state.anmeldung;
  if (!foundation || !unterbilanz || !anmeldung) return null;

  const paidIn = foundation.shareholders.reduce((sum, s) => sum + s.paidIn, 0);
  const liable = unterbilanz.amount > 0;

  async function openPreview() {
    try {
      setPreview(await Api.previewFoundationPostings());
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    }
  }

  async function bookPostings() {
    setBusy(true);
    try {
      // Eine Liste, die das Backend nie befüllt hat, kommt als `null` an; die
      // Länge davon wäre ein Fehler statt einer Rückmeldung.
      const created = (await Api.bookFoundationPostings()) ?? [];
      toast.success(
        created.length === 1
          ? 'Die Gründungsbuchung steht im Journal.'
          : `${created.length} Gründungsbuchungen stehen im Journal.`
      );
      setPreview(null);
      await onChanged();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  async function register() {
    setBusy(true);
    try {
      await Api.registerCompany(registerDate, registerCourt, registerNumber);
      toast.success(`Eingetragen am ${formatDate(registerDate)}. Die Unterbilanz steht damit fest.`);
      setRegistering(false);
      await onChanged();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Section
      title="Gründung"
      context={`Vorgesellschaft seit ${formatDate(foundation.notarizedOn)}`}
      action={
        <div className="flex items-center gap-2">
          <HelpPopover label="Erklärung zur Unterbilanzhaftung">
            Bleibt das Reinvermögen der Gesellschaft am Tag der Eintragung hinter dem Stammkapital
            zurück, schulden die Gesellschafter die Differenz — anteilig nach ihren
            Geschäftsanteilen. Gründungskosten dürfen nach § 248 Abs. 1 Nr. 1 HGB nicht aktiviert
            werden und mindern das Reinvermögen deshalb sofort. Bis zur Eintragung haftet außerdem
            persönlich, wer im Namen der Gesellschaft handelt (§ 11 Abs. 2 GmbHG).
          </HelpPopover>
          {!state.postingsBooked && (
            <Button
              variant="secondary"
              icon={<BookOpen className="w-4 h-4" strokeWidth={1.5} />}
              onClick={() => void openPreview()}
            >
              Gründung buchen
            </Button>
          )}
          <Button
            variant="primary"
            icon={<Landmark className="w-4 h-4" strokeWidth={1.5} />}
            onClick={() => setRegistering(true)}
          >
            Eintragung erfassen
          </Button>
        </div>
      }
    >
      <StatRow>
        <Stat
          label="Stammkapital"
          value={formatCents(foundation.shareCapital)}
          context="laut Gesellschaftsvertrag"
        />
        <Stat
          label="Davon geleistet"
          value={formatCents(paidIn)}
          context={
            anmeldung.isSatisfied
              ? 'Anmeldung möglich'
              : `${formatCents(anmeldung.requiredPaidIn)} nötig`
          }
          tone={anmeldung.isSatisfied ? 'positive' : 'neutral'}
        />
        <Stat
          label="Reinvermögen"
          value={formatCents(unterbilanz.netAssets)}
          context={`Stand ${formatDate(unterbilanz.asOf)}`}
        />
        <Stat
          label="Unterbilanz"
          value={formatCents(unterbilanz.amount)}
          context={
            unterbilanz.isFinal
              ? 'zum Tag der Eintragung'
              : liable
                ? 'vorläufig, wächst mit jeder Buchung'
                : 'vorläufig, derzeit keine'
          }
          tone={liable ? 'negative' : 'positive'}
        />
      </StatRow>

      {!anmeldung.isSatisfied && (
        <p className="mt-6 rounded-control border border-attention-line bg-attention-soft px-4 py-3 text-body text-attention-text">
          {anmeldung.findings[0]}
        </p>
      )}

      {unterbilanz.covered > 0 && (
        <p className="mt-6 text-caption text-ink-muted">
          Von {formatCents(unterbilanz.shortfall)} Unterdeckung deckt die Gründungsaufwandsklausel
          des Gesellschaftsvertrags {formatCents(unterbilanz.covered)}.
        </p>
      )}

      <div className="mt-6">
        <Table>
          <Thead>
            <Tr>
              <Th>Gesellschafter</Th>
              <Th className="w-32">Einlage</Th>
              <Th numeric className="w-36">
                Anteil
              </Th>
              <Th numeric className="w-36">
                Geleistet
              </Th>
              <Th numeric className="w-36">
                Offen
              </Th>
              <Th numeric className="w-40">
                Anteilige Haftung
              </Th>
            </Tr>
          </Thead>
          <Tbody>
            {(foundation.shareholders ?? []).map((holder) => {
              const share = (unterbilanz.shares ?? []).find((s) => s.shareholderId === holder.id);
              const open = holder.shareCapital - holder.paidIn;
              return (
                <Tr key={holder.id}>
                  <Td className="max-w-[20rem] truncate">{holder.name}</Td>
                  <Td className="text-ink-muted">
                    {holder.kind === 'kind' ? 'Sacheinlage' : 'Bareinlage'}
                  </Td>
                  <Td numeric>{formatCents(holder.shareCapital)}</Td>
                  <Td numeric>{formatCents(holder.paidIn)}</Td>
                  <Td numeric className={cn(open > 0 && 'text-attention-text')}>
                    {formatCents(open)}
                  </Td>
                  <Td numeric className={cn((share?.amount ?? 0) > 0 && 'text-negative-text')}>
                    {formatCents(share?.amount ?? 0)}
                  </Td>
                </Tr>
              );
            })}
            <Tr variant="sum">
              <Td>Summe</Td>
              <Td />
              <Td numeric>{formatCents(foundation.shareCapital)}</Td>
              <Td numeric>{formatCents(paidIn)}</Td>
              <Td numeric>{formatCents(foundation.shareCapital - paidIn)}</Td>
              <Td numeric>{formatCents(unterbilanz.amount)}</Td>
            </Tr>
          </Tbody>
        </Table>
      </div>

      <Dialog
        open={preview !== null}
        onOpenChange={(next) => !next && setPreview(null)}
        title="Gründungsbuchungen"
        footer={
          <>
            <Button variant="secondary" onClick={() => setPreview(null)}>
              Abbrechen
            </Button>
            <Button
              variant="primary"
              loading={busy}
              disabled={(preview?.postings.length ?? 0) === 0}
              onClick={() => void bookPostings()}
            >
              Buchen
            </Button>
          </>
        }
      >
        <Table>
          <Thead>
            <Tr>
              <Th>Vorgang</Th>
              <Th className="w-24">Datum</Th>
              <Th className="w-20">Seite</Th>
              <Th className="w-24">Konto</Th>
              <Th numeric className="w-36">
                Betrag
              </Th>
            </Tr>
          </Thead>
          <Tbody>
            {(preview?.postings ?? []).map((posting) =>
              (posting.lines ?? []).map((line, index) => (
                <Tr key={`${posting.title}-${index}`}>
                  <Td className="max-w-[18rem] truncate">{index === 0 ? posting.title : ''}</Td>
                  <Td className="num text-ink-subtle">
                    {index === 0 ? formatDate(posting.date) : ''}
                  </Td>
                  <Td className="text-ink-muted">{formatSide(line.side)}</Td>
                  <Td code>{line.account}</Td>
                  <Td numeric>{formatCents(line.amount)}</Td>
                </Tr>
              ))
            )}
          </Tbody>
        </Table>

        {(preview?.skipped ?? []).map((note) => (
          <p key={note} className="mt-4 text-caption text-ink-muted">
            {note}
          </p>
        ))}
      </Dialog>

      <Dialog
        open={registering}
        onOpenChange={setRegistering}
        title="Eintragung ins Handelsregister"
        width="max-w-lg"
        footer={
          <>
            <Button variant="secondary" onClick={() => setRegistering(false)}>
              Abbrechen
            </Button>
            <Button
              variant="primary"
              loading={busy}
              disabled={registerDate.length !== 10 || registerNumber.trim() === ''}
              onClick={() => void register()}
            >
              Eintragung erfassen
            </Button>
          </>
        }
      >
        <div className="flex flex-col gap-4">
          <p className="text-body text-ink-muted">
            Damit endet die Vorgesellschaft. Die Unterbilanz wird auf diesen Tag festgestellt und
            ändert sich danach nicht mehr.
          </p>
          <label className="flex flex-col gap-1">
            <span className="text-label text-ink-subtle">Tag der Eintragung</span>
            <input
              type="date"
              value={registerDate}
              min={foundation.notarizedOn}
              onChange={(e) => setRegisterDate(e.target.value)}
              className="h-9 w-full px-3 rounded-control border border-line-strong bg-paper text-body num
                         focus:outline-none focus:border-accent focus:ring-2 focus:ring-accent/30"
            />
          </label>
          <label className="flex flex-col gap-1">
            <span className="text-label text-ink-subtle">Registergericht</span>
            <input
              type="text"
              placeholder="Amtsgericht München"
              value={registerCourt}
              onChange={(e) => setRegisterCourt(e.target.value)}
              className="h-9 w-full px-3 rounded-control border border-line-strong bg-paper text-body
                         focus:outline-none focus:border-accent focus:ring-2 focus:ring-accent/30"
            />
          </label>
          <label className="flex flex-col gap-1">
            <span className="text-label text-ink-subtle">Registernummer</span>
            <input
              type="text"
              placeholder="HRB 123456"
              value={registerNumber}
              onChange={(e) => setRegisterNumber(e.target.value)}
              className="h-9 w-full px-3 rounded-control border border-line-strong bg-paper text-body code-num
                         focus:outline-none focus:border-accent focus:ring-2 focus:ring-accent/30"
            />
          </label>
        </div>
      </Dialog>
    </Section>
  );
};

/**
 * Bestätigung für das Zurücknehmen einer erledigten Gründungspflicht. Sie steht
 * hier, weil sie zum Gründungsteil gehört und die Fristenseite sonst zwei
 * Dialoge desselben Namens trüge.
 */
export const FoundationDutyResetDialog: React.FC<{
  title: string | null;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
}> = ({ title, onOpenChange, onConfirm }) => (
  <ConfirmDialog
    open={title !== null}
    onOpenChange={onOpenChange}
    title={`${title ?? ''} wieder als offen führen`}
    description="Das festgehaltene Erledigungsdatum wird gelöscht. Die Pflicht steht danach wieder als offen in der Liste."
    confirmLabel="Als offen führen"
    onConfirm={onConfirm}
  />
);
