import React, { useEffect, useState } from 'react';
import type { NumberGapReason, NumberGapReasonOption, NumberGapReport } from '../types';
import { Api } from '../services/api';
import { usePostingLock } from './WriteLock';
import { formatDate } from '../utils/formatters';
import {
  Button,
  Dialog,
  EmptyState,
  Field,
  Input,
  Notice,
  Section,
  Select,
  SkeletonRows,
  Table,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
} from './ui';

export const InvoiceNumberRange: React.FC<{ year: number }> = ({ year }) => {
  const writeLock = usePostingLock();
  const [gaps, setGaps] = useState<NumberGapReport | null>(null);
  const [reasons, setReasons] = useState<NumberGapReasonOption[]>([]);
  const [gapReason, setGapReason] = useState<{ sequence: number; number: string } | null>(null);
  const [loading, setLoading] = useState(true);
  const [failure, setFailure] = useState<string | null>(null);

  useEffect(() => {
    void load();
  }, [year]);

  async function load() {
    setLoading(true);
    setFailure(null);
    try {
      const [report, options] = await Promise.all([
        Api.getInvoiceNumberGaps(year),
        Api.getNumberGapReasons(),
      ]);
      if (!report) throw new Error('Der Lückenbericht konnte nicht geladen werden.');
      setGaps(report);
      setReasons(options);
    } catch (e) {
      setFailure(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }

  return (
    <>
      <Section helpSummary="Jede ausgestellte Rechnung erhält eine eigene fortlaufende Nummer."
        title="Rechnungsnummern prüfen"
        divider={false}
        context={
          loading
            ? 'Bericht wird geladen'
            : failure || !gaps
              ? 'Bericht nicht verfügbar'
              : `${gaps.fiscalYear} · ${gaps.issued} ${gaps.issued === 1 ? 'Nummer' : 'Nummern'} vergeben · ${gaps.used} mit Dokument · ${gaps.gaps.length} ohne`
        }
        explain={
          <>
            Buchfink vergibt Rechnungsnummern automatisch. Hier sehen Sie, welche vergebenen
            Nummern kein Dokument haben. Halten Sie den Grund einer Lücke fest, damit er bei
            einer späteren Prüfung nachvollziehbar bleibt.
          </>
        }
      >
        {loading ? (
          <SkeletonRows rows={2} />
        ) : failure ? (
          <Notice
            tone="negative"
            text={failure}
            action={
              <Button variant="quiet" size="sm" onClick={() => void load()}>
                Erneut laden
              </Button>
            }
          />
        ) : gaps && gaps.gaps.length === 0 ? (
          <EmptyState
            title="Keine Lücke im Rechnungsnummernkreis"
            description="Jede vergebene Nummer hat ein Dokument."
          />
        ) : (
          <Table density="kompakt">
            <Thead>
              <Tr>
                <Th className="w-40">Nummer</Th>
                <Th className="w-56">Grund</Th>
                <Th>Vermerk</Th>
                <Th className="w-40">Festgehalten</Th>
                <Th className="w-32" aria-label="Aktionen" />
              </Tr>
            </Thead>
            <Tbody>
              {gaps?.gaps.map((gap) => (
                <Tr key={gap.sequence} className="group">
                  <Td code>{gap.number}</Td>
                  <Td className={gap.reason === 'unknown' ? 'text-attention-text' : 'text-ink-muted'}>
                    {gap.label}
                  </Td>
                  <Td className="max-w-[24rem] truncate">{gap.detail || '—'}</Td>
                  <Td className="text-ink-subtle num">
                    {gap.recordedAt ? formatDate(gap.recordedAt.split('T')[0]) : '—'}
                  </Td>
                  <Td className="pl-0">
                    <Button
                      variant="quiet"
                      size="sm"
                      disabled={writeLock.locked}
                      title={writeLock.hint}
                      className="opacity-0 transition-opacity duration-120 ease-quiet
                                 group-hover:opacity-100 focus-visible:opacity-100"
                      onClick={() => setGapReason({ sequence: gap.sequence, number: gap.number })}
                    >
                      Begründen
                    </Button>
                  </Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
      </Section>

      <GapReasonDialog
        gap={gapReason}
        reasons={reasons}
        year={year}
        onClose={() => setGapReason(null)}
        onDone={() => {
          setGapReason(null);
          void load();
        }}
      />
    </>
  );
};

/** Der Grund einer Lücke im Nummernkreis — die Frage der Betriebsprüfung. */
const GapReasonDialog: React.FC<{
  gap: { sequence: number; number: string } | null;
  reasons: NumberGapReasonOption[];
  year: number;
  onClose: () => void;
  onDone: () => void;
}> = ({ gap, reasons, year, onClose, onDone }) => {
  const writeLock = usePostingLock();
  const [reason, setReason] = useState<NumberGapReason>('aborted');
  const [detail, setDetail] = useState('');
  // Der Grund kommt aus einer Auswahl mit Voreinstellung, der Vermerk ist
  // freiwillig: es gibt keine Pflichtangabe, die am Feld fehlen könnte. Was
  // zurückkommt, ist die Ablehnung des Backends (§10.4).
  const [failure, setFailure] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!gap) return;
    setReason('aborted');
    setDetail('');
    setFailure(null);
  }, [gap]);

  async function submit() {
    setFailure(null);
    setBusy(true);
    try {
      await Api.recordInvoiceNumberGapReason(year, gap!.sequence, reason, detail);
      // Kein Toast: Der Grund steht danach in der Zeile des Lückenberichts
      // (§8.5).
      onDone();
    } catch (e) {
      setFailure(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog
      open={gap !== null}
      onOpenChange={(next) => !next && onClose()}
      title={`Lücke ${gap?.number ?? ''} begründen`}
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
            Grund festhalten
          </Button>
        </>
      }
    >
      <Field label="Grund">
        <Select
          items={reasons.map((o) => ({ value: o.reason, label: o.label }))}
          value={reason}
          onValueChange={setReason}
        />
      </Field>
      <Field label="Vermerk" optional className="mt-4">
        <Input
          value={detail}
          onChange={(e) => setDetail(e.target.value)}
          placeholder="Abbruch beim Erzeugen des Dokuments"
        />
      </Field>

      {failure && <Notice tone="negative" text={failure} className="mt-6" />}
    </Dialog>
  );
};
