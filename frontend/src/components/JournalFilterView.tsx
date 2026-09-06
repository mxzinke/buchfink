import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Download } from 'lucide-react';
import { Api } from '../services/api';
import { formatCents, formatDate, parseCents } from '../utils/formatters';
import type { Account, JournalFilter, JournalFilterResult } from '../types';
import type { NavigateFn } from './Sidebar';
import {
  Button,
  Combobox,
  EmptyState,
  Field,
  FieldRow,
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
  toast,
} from './ui';

/**
 * Die gefilterte Journalansicht (PRF-01 K3).
 *
 * Gefiltert wird über Zeilen und nicht über Buchungen: „Konto 6300" ist eine
 * Eigenschaft der Zeile, und ein Prüfer, der nach einem Konto fragt, will die
 * Zeilen dieses Kontos sehen. Gefiltert und summiert wird im Backend — zehn-
 * tausend Zeilen in die Ansicht zu laden, um darin fünf zu suchen, wäre der
 * falsche Weg, und eine zweite Summenrechnung in der Maske liefe von der des
 * CSV-Ausgabewegs irgendwann ab.
 */
interface JournalFilterViewProps {
  accounts: Account[];
  /** Das Geschäftsjahr — nur der Vorschlag für den Dateinamen der CSV. */
  year?: number;
  /** Vorbelegtes Konto beim Sprung aus dem Kontoblatt (PRF-01 K3). */
  initialAccount?: string;
  onNavigate?: NavigateFn;
}

export const JournalFilterView: React.FC<JournalFilterViewProps> = ({
  accounts,
  year,
  initialAccount,
  onNavigate,
}) => {
  const [filter, setFilter] = useState<JournalFilter>(
    initialAccount ? { account: initialAccount } : {},
  );
  const [amountFrom, setAmountFrom] = useState('');
  const [amountTo, setAmountTo] = useState('');
  const [result, setResult] = useState<JournalFilterResult | null>(null);
  const [searching, setSearching] = useState(false);
  const [error, setError] = useState('');
  const [touched, setTouched] = useState(false);
  const [exporting, setExporting] = useState(false);

  /** Der Filter, wie er an das Backend geht: die Beträge erst hier gelesen. */
  const activeFilter = useMemo<JournalFilter>(() => {
    const next: JournalFilter = { ...filter };
    const from = parseCents(amountFrom);
    const to = parseCents(amountTo);
    if (from !== null) next.amountFrom = from;
    if (to !== null) next.amountTo = to;
    return next;
  }, [filter, amountFrom, amountTo]);

  const search = useCallback(async () => {
    setSearching(true);
    setError('');
    setTouched(true);
    try {
      setResult(await Api.getFilteredJournal(activeFilter));
    } catch (e) {
      setResult(null);
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setSearching(false);
    }
  }, [activeFilter]);

  useEffect(() => {
    // Die erste Anzeige ist die ungefilterte Menge: eine leere Tabelle mit
    // Filterfeldern darüber sähe aus, als gäbe es keine Buchungen.
    void search();
    // Nur beim Öffnen der Ansicht; danach löst der Knopf die Suche aus.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  /**
   * Die gefilterte Menge als Datei. Ein Fehler des Backends bleibt als
   * Hinweisfläche über der Tabelle stehen (§10.4) — er nennt den Grund, aus
   * dem die Datei nicht geschrieben wurde, und der überlebt keine fünf
   * Sekunden Meldung.
   */
  const exportCSV = async () => {
    setExporting(true);
    setError('');
    try {
      const path = await Api.selectSaveFilePath(
        'Gefilterte Buchungen speichern',
        `journal-${year ?? new Date().getFullYear()}.csv`,
      );
      // Leerer Pfad heißt: der Speichern-Dialog wurde abgebrochen.
      if (!path) return;
      await Api.saveFilteredJournalCSV(activeFilter, path);
      toast.success('Die gefilterte Menge ist geschrieben.');
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setExporting(false);
    }
  };

  const resetFilter = () => {
    setFilter({});
    setAmountFrom('');
    setAmountTo('');
    setTouched(false);
  };

  const accountOptions = useMemo(
    () =>
      accounts
        .filter((a) => !a.isRange && !a.isReserved)
        .map((a) => ({ value: a.number, label: `${a.number} ${a.name}`, meta: a.kontenklasseName })),
    [accounts],
  );

  /** Nummer → Bezeichnung, falls das Backend den Namen an der Zeile nicht führt. */
  const accountNames = useMemo(
    () => new Map((accounts ?? []).map((a) => [a.number, a.name])),
    [accounts],
  );

  const filterActive = useMemo(
    () => Object.values(activeFilter).some((value) => value !== undefined && value !== ''),
    [activeFilter],
  );

  const rows = result?.rows ?? [];

  return (
    <>
      <Section
        title="Filter"
        context={
          result
            ? `${result.rowCount} Zeilen aus ${result.entryCount} Buchungen`
            : 'Konto, Betrag, Steuerschlüssel, Bearbeiter, Beleg'
        }
        divider={false}
        className="mt-6"
        action={
          <div className="flex gap-2">
            <Button variant="quiet" onClick={resetFilter}>
              Filter zurücksetzen
            </Button>
            <Button
              variant="secondary"
              loading={exporting}
              icon={<Download className="w-4 h-4" strokeWidth={1.5} />}
              onClick={exportCSV}
            >
              Als CSV speichern
            </Button>
            <Button variant="secondary" loading={searching} onClick={search}>
              Suchen
            </Button>
          </div>
        }
      >
        <FieldRow>
          <Field label="Von">
            <Input
              type="date"
              value={filter.from ?? ''}
              onChange={(e) => setFilter({ ...filter, from: e.target.value })}
            />
          </Field>
          <Field label="Bis">
            <Input
              type="date"
              value={filter.to ?? ''}
              onChange={(e) => setFilter({ ...filter, to: e.target.value })}
            />
          </Field>
          <Field label="Konto" className="flex-1">
            <Combobox
              items={accountOptions}
              value={filter.account ?? null}
              onValueChange={(next) => setFilter({ ...filter, account: next ?? '' })}
              placeholder="Alle Konten"
            />
          </Field>
          <Field
            label="Gegenkonto"
            className="flex-1"
            explain="Konto auf der anderen Seite derselben Buchung"
          >
            <Combobox
              items={accountOptions}
              value={filter.counterAccount ?? null}
              onValueChange={(next) => setFilter({ ...filter, counterAccount: next ?? '' })}
              placeholder="Alle Konten"
            />
          </Field>
        </FieldRow>
        <FieldRow className="mt-4">
          <Field label="Betrag von">
            <Input
              align="right"
              inputMode="decimal"
              value={amountFrom}
              placeholder="0,00"
              onChange={(e) => setAmountFrom(e.target.value)}
            />
          </Field>
          <Field label="Betrag bis">
            <Input
              align="right"
              inputMode="decimal"
              value={amountTo}
              placeholder="0,00"
              onChange={(e) => setAmountTo(e.target.value)}
            />
          </Field>
          <Field label="Steuerschlüssel">
            <Input
              value={filter.taxKey ?? ''}
              placeholder="VST19"
              onChange={(e) => setFilter({ ...filter, taxKey: e.target.value })}
            />
          </Field>
          <Field label="Bearbeiter">
            <Input
              value={filter.actor ?? ''}
              onChange={(e) => setFilter({ ...filter, actor: e.target.value })}
            />
          </Field>
          <Field label="Beleg">
            <Select
              items={[
                { value: 'any', label: 'Alle' },
                { value: 'yes', label: 'Mit Beleg' },
                { value: 'no', label: 'Ohne Beleg' },
              ]}
              value={filter.hasReceipt === undefined ? 'any' : filter.hasReceipt ? 'yes' : 'no'}
              onValueChange={(next) =>
                setFilter({
                  ...filter,
                  hasReceipt: next === 'any' ? undefined : next === 'yes',
                })
              }
              aria-label="Buchungen mit oder ohne Beleg"
            />
          </Field>
          <Field label="Suchtext" className="flex-1">
            <Input
              value={filter.text ?? ''}
              placeholder="Buchungstext, Beleg- oder Buchungsnummer"
              onChange={(e) => setFilter({ ...filter, text: e.target.value })}
            />
          </Field>
        </FieldRow>
      </Section>

      <Section
        title="Gefundene Zeilen"
        context={
          result
            ? `Soll ${formatCents(result.totalDebit)} · Haben ${formatCents(result.totalCredit)}`
            : undefined
        }
        explain={
          <>
            Gefiltert wird über Zeilen und nicht über Buchungen: „Konto 6300" ist eine Eigenschaft
            der Zeile. Die Summe unten summiert deshalb Soll und Haben der gefilterten Zeilen. Sie
            ist nicht notwendig null — ein Filter schneidet Buchungen an, und das soll er.
          </>
        }
      >
        {error && <Notice tone="negative" text={error} className="mb-5" />}
        {searching ? (
          <SkeletonRows rows={6} />
        ) : rows.length === 0 ? (
          <EmptyState
            variant={filterActive || touched ? 'gefiltert' : 'leer'}
            title={filterActive ? 'Keine Buchungen für diesen Filter' : 'Noch nichts gebucht'}
            description={
              filterActive
                ? 'Die Einschränkungen oben treffen keine Zeile.'
                : 'Sobald eine Buchung erfasst ist, steht sie hier.'
            }
            action={
              filterActive ? (
                <Button variant="secondary" onClick={resetFilter}>
                  Filter zurücksetzen
                </Button>
              ) : undefined
            }
          />
        ) : (
          <Table>
            <Thead sticky>
              <Tr>
                <Th>Buchung</Th>
                <Th>Datum</Th>
                <Th>Buchungstext</Th>
                <Th>Konto</Th>
                <Th>Steuer</Th>
                <Th>Bearbeiter</Th>
                <Th numeric>Soll</Th>
                <Th numeric>Haben</Th>
              </Tr>
            </Thead>
            <Tbody>
              {rows.map((row) => (
                <Tr
                  key={`${row.entryId}-${row.position}`}
                  variant={row.kind === 'reversal' ? 'storno' : 'default'}
                >
                  <Td code>{row.entryNumber}</Td>
                  <Td className="text-ink-subtle num">{formatDate(row.bookingDate)}</Td>
                  <Td className="max-w-[20rem] truncate" title={row.description}>
                    {row.description}
                  </Td>
                  {/* Nummer und Bezeichnung zusammen (§11.1): „6300" allein sagt
                      niemandem, welches Konto gemeint ist. */}
                  <Td>
                    <div className="flex items-baseline gap-2">
                      {onNavigate ? (
                        <Button
                          variant="quiet"
                          size="sm"
                          className="code-num"
                          onClick={() => onNavigate('accounts', { account: row.account })}
                        >
                          {row.account}
                        </Button>
                      ) : (
                        <span className="code-num">{row.account}</span>
                      )}
                      <span className="text-ink-muted truncate max-w-[12rem]">
                        {row.accountName ?? accountNames.get(row.account) ?? ''}
                      </span>
                    </div>
                  </Td>
                  <Td className="text-ink-subtle code-num">{row.taxKey ?? '—'}</Td>
                  <Td className="text-ink-subtle">{row.actor ?? '—'}</Td>
                  <Td numeric>{row.side === 'S' ? formatCents(row.amount) : ''}</Td>
                  <Td numeric>{row.side === 'H' ? formatCents(row.amount) : ''}</Td>
                </Tr>
              ))}
              {result && (
                <Tr className="rule-total">
                  <Td colSpan={6} className="text-label text-ink">
                    {`Summe · ${result.rowCount} Zeilen`}
                  </Td>
                  <Td numeric className="text-label text-ink">
                    {formatCents(result.totalDebit)}
                  </Td>
                  <Td numeric className="text-label text-ink">
                    {formatCents(result.totalCredit)}
                  </Td>
                </Tr>
              )}
            </Tbody>
          </Table>
        )}
      </Section>
    </>
  );
};
