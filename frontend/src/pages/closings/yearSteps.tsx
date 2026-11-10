import React, { useEffect, useState } from 'react';
import { Api } from '../../services/api';
import { useWriteLock } from '../../components/WriteLock';
import { formatCents, formatDate, formatPermille } from '../../utils/formatters';
import type { Route } from '../../navigation';
import type {
  Accrual,
  ClosingStepKey,
  ClosingStepView,
  ForeignCurrencyValuation,
  InputTaxCorrectionYear,
  NotesSectionText,
  Provision,
  ProvisionMirror,
  SizeClass,
  TaxProvisionPreview,
  VatSettlement,
  WriteUpReport,
} from '../../types';
import {
  Help,
  Button,
  DetailList,
  Disclosure,
  Field,
  Input,
  Notice,
  SkeletonRows,
  StatusBadge,
  Table,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
  cn,
} from '../../components/ui';
import { message } from './shared';

/**
 * Was ein Baustein des Jahresabschlusses in der Oberfläche zeigt.
 *
 * Gebucht werden die Bausteine von Claude. Die App zeigt je Baustein die
 * Kartei, die dahinter steht, damit ein Prüfer sie ohne Claude einsehen kann
 * (§ 147 Abs. 6 AO). Feststellen, Ergebnisverwendung und Offenlegung haben je
 * eine eigene Zeile in der Schrittliste (`row`): Feststellen und Offenlegen
 * tut der Mensch, die Ergebnisverwendung bucht Claude nach dem Beschluss der
 * Gesellschafter, oder der Mensch erfasst sie dort.
 *
 * scripts/check_closing_steps.py hält diese Zuordnung gegen die Schlüssel in
 * internal/domain/closing_step.go.
 */
type StepView =
  | { kind: 'kartei'; kartei: Kartei; route?: Route; routeLabel?: string }
  | { kind: 'route'; route: Route; routeLabel: string }
  | { kind: 'row' };

type Kartei =
  | 'accruals'
  | 'provisions'
  | 'vatSettlement'
  | 'taxProvision'
  | 'inputTax'
  | 'currency'
  | 'writeUps'
  | 'statement'
  | 'none';

export const STEP_VIEWS: Record<ClosingStepKey, StepView> = {
  depreciation: { kind: 'route', route: { area: 'books', tab: 'assets' }, routeLabel: 'Zu den Anlagen' },
  write_up: { kind: 'kartei', kartei: 'writeUps' },
  currency_valuation: { kind: 'kartei', kartei: 'currency' },
  receivable_allowance: { kind: 'kartei', kartei: 'none' },
  work_in_progress: { kind: 'kartei', kartei: 'none' },
  accruals: { kind: 'kartei', kartei: 'accruals' },
  provisions: { kind: 'kartei', kartei: 'provisions' },
  vat_settlement: { kind: 'kartei', kartei: 'vatSettlement' },
  input_tax_correction: { kind: 'kartei', kartei: 'inputTax' },
  tax_provision: { kind: 'kartei', kartei: 'taxProvision' },
  check_run: { kind: 'route', route: { area: 'closings', tab: 'month' }, routeLabel: 'Zum Monatsabschluss' },
  statement: { kind: 'kartei', kartei: 'statement', route: { area: 'books', tab: 'statements' }, routeLabel: 'Zu Bilanz und GuV' },
  adoption: { kind: 'row' },
  disclosure: { kind: 'row' },
  appropriation: { kind: 'row' },
};

/** Ist der Schlüssel ein Baustein des Abschlusses? */
export function isStepKey(value: string | undefined): value is ClosingStepKey {
  return value !== undefined && Object.prototype.hasOwnProperty.call(STEP_VIEWS, value);
}

/** Die Abschlussbuchungen, ohne die Bausteine mit eigener Zeile in der Schrittliste. */
export function isBookingStep(step: ClosingStepView): boolean {
  return STEP_VIEWS[step.key]?.kind !== 'row';
}

/** Der Stand eines Bausteins als Abzeichen; übersprungen steht als Wort. */
export const StepState: React.FC<{ step: ClosingStepView }> = ({ step }) =>
  step.state === 'skipped' ? (
    <span className="inline-flex items-center gap-1.5 text-caption text-ink-subtle">
      <span className="mark-diamond bg-ink-faint" aria-hidden="true" />
      Übersprungen
    </span>
  ) : (
    <StatusBadge status={step.state === 'done' ? 'erledigt' : 'offen'} />
  );

/** Der Inhalt der Seitenleiste eines Bausteins. */
export const StepDetails: React.FC<{
  step: ClosingStepView;
  year: number;
}> = ({ step, year }) => {
  const view = STEP_VIEWS[step.key];
  return (
    <div className="flex flex-col gap-6">
      <DetailList
        items={[
          { label: 'Stand', value: <StepState step={step} /> },
          ...(step.detail ? [{ label: 'Woran', value: step.detail }] : []),
          ...(step.state === 'skipped' && step.reason
            ? [{ label: 'Grund', value: step.reason }]
            : []),
          ...(step.changedOn ? [{ label: 'Geändert', value: formatDate(step.changedOn) }] : []),
        ]}
      />
      <p className="text-body text-ink-muted">
        {step.hint}
        {step.reference && (
          <Help summary="Auf diesen Vorschriften beruht der Baustein." label={`Grundlage zu ${step.label}`}>
            {step.reference}
          </Help>
        )}
      </p>
      {view?.kind === 'kartei' && <KarteiView kartei={view.kartei} year={year} />}
    </div>
  );
};

const KarteiView: React.FC<{ kartei: Kartei; year: number }> = ({ kartei, year }) => {
  switch (kartei) {
    case 'accruals':
      return <AccrualsKartei year={year} />;
    case 'provisions':
      return <ProvisionsKartei year={year} />;
    case 'vatSettlement':
      return <VatSettlementKartei year={year} />;
    case 'taxProvision':
      return <TaxProvisionKartei year={year} />;
    case 'inputTax':
      return <InputTaxKartei year={year} />;
    case 'currency':
      return <CurrencyKartei year={year} />;
    case 'writeUps':
      return <WriteUpKartei year={year} />;
    case 'statement':
      return <StatementKartei year={year} />;
    default:
      return null;
  }
};

/** Lädt eine Kartei und zeigt Ladezustand und Fehler gleich. */
function useKartei<T>(load: () => Promise<T>, deps: unknown[]): { data: T | null; error: string } {
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState('');
  useEffect(() => {
    let cancelled = false;
    setData(null);
    setError('');
    load()
      .then((next) => !cancelled && setData(next))
      .catch((e) => !cancelled && setError(message(e)));
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);
  return { data, error };
}

const KarteiFrame: React.FC<{
  title: string;
  error: string;
  loading: boolean;
  empty?: string;
  children?: React.ReactNode;
}> = ({ title, error, loading, empty, children }) => (
  <section>
    <h3 className="text-label text-ink-muted mb-2">{title}</h3>
    {error ? (
      <Notice tone="negative" text={error} />
    ) : loading ? (
      <SkeletonRows rows={3} />
    ) : empty ? (
      <p className="text-body text-ink-subtle">{empty}</p>
    ) : (
      children
    )}
  </section>
);

const ACCRUAL_KIND: Record<string, string> = {
  active: 'Ausgabe für spätere Jahre',
  passive: 'Einnahme für spätere Jahre',
  disagio: 'Disagio',
};

const AccrualsKartei: React.FC<{ year: number }> = ({ year }) => {
  const { data, error } = useKartei<Accrual[]>(() => Api.getAccruals(year), [year]);
  return (
    <KarteiFrame
      title="Rechnungsabgrenzung"
      error={error}
      loading={data === null}
      empty={data?.length === 0 ? 'Keine Posten abgegrenzt.' : undefined}
    >
      <Table density="kompakt">
        <Thead>
          <Tr>
            <Th>Posten</Th>
            <Th className="w-40">Zeitraum</Th>
            <Th numeric className="w-32">
              Abgegrenzt
            </Th>
          </Tr>
        </Thead>
        <Tbody>
          {(data ?? []).map((a) => (
            <Tr key={a.id}>
              <Td className="whitespace-normal">
                <span className="block">{a.text}</span>
                <span className="block text-caption text-ink-subtle">
                  {`${ACCRUAL_KIND[a.kind] ?? a.kind} · Konto ${a.account}${a.formationEntryId ? '' : ' · vorgeschlagen'}`}
                </span>
              </Td>
              <Td className="num text-ink-subtle">{`${formatDate(a.startDate)} – ${formatDate(a.endDate)}`}</Td>
              <Td numeric>{formatCents(a.deferredAmount)}</Td>
            </Tr>
          ))}
        </Tbody>
      </Table>
    </KarteiFrame>
  );
};

const ProvisionsKartei: React.FC<{ year: number }> = ({ year }) => {
  const { data, error } = useKartei<[ProvisionMirror, Provision[]]>(
    () => Promise.all([Api.getProvisionMirror(year), Api.getProvisions(year)]),
    [year],
  );
  const mirror = data?.[0];
  const provisions = data?.[1] ?? [];
  return (
    <KarteiFrame
      title="Rückstellungsspiegel"
      error={error}
      loading={data === null}
      empty={mirror && mirror.rows.length === 0 ? 'Keine Rückstellungen gebildet.' : undefined}
    >
      {mirror && (
        <Table density="kompakt">
          <Thead>
            <Tr>
              <Th>Art</Th>
              <Th numeric>Anfang</Th>
              <Th numeric>Zugang</Th>
              <Th numeric>Verbrauch</Th>
              <Th numeric>Auflösung</Th>
              <Th numeric>Ende</Th>
            </Tr>
          </Thead>
          <Tbody>
            {mirror.rows.map((row) => (
              <Tr key={`${row.kind}-${row.account}`}>
                <Td className="whitespace-normal">{row.label}</Td>
                <Td numeric className="text-ink-muted">{formatCents(row.opening)}</Td>
                <Td numeric>{formatCents(row.additions + row.unwinding)}</Td>
                <Td numeric className="text-ink-muted">{formatCents(row.used)}</Td>
                <Td numeric className="text-ink-muted">{formatCents(row.released)}</Td>
                <Td numeric>{formatCents(row.closing)}</Td>
              </Tr>
            ))}
            <Tr variant="sum">
              <Td>Summe</Td>
              <Td numeric>{formatCents(mirror.total.opening)}</Td>
              <Td numeric>{formatCents(mirror.total.additions + mirror.total.unwinding)}</Td>
              <Td numeric>{formatCents(mirror.total.used)}</Td>
              <Td numeric>{formatCents(mirror.total.released)}</Td>
              <Td numeric>{formatCents(mirror.total.closing)}</Td>
            </Tr>
          </Tbody>
        </Table>
      )}
      {provisions.length > 0 && (
        <ul className="mt-4 flex flex-col">
          {provisions.map((p) => (
            <li key={p.id} className="py-2 border-t border-line first:border-t-0">
              <span className="flex items-baseline justify-between gap-4">
                <span className="text-body text-ink">{p.text}</span>
                <span className="text-body num">{formatCents(p.discountedAmount)}</span>
              </span>
              <span className="block text-caption text-ink-subtle">
                {`${p.reason}${p.expectedDate ? ` · erwartet ${formatDate(p.expectedDate)}` : ''}${p.settledOn ? ` · erledigt ${formatDate(p.settledOn)}` : ''}`}
              </span>
            </li>
          ))}
        </ul>
      )}
    </KarteiFrame>
  );
};

const VatSettlementKartei: React.FC<{ year: number }> = ({ year }) => {
  const { data, error } = useKartei<VatSettlement>(() => Api.previewVatSettlement(year), [year]);
  return (
    <KarteiFrame title="Umsatzsteuer zum Stichtag" error={error} loading={data === null}>
      {data && (
        <DetailList
          items={[
            { label: 'Umsatzsteuer', value: formatCents(data.outputTax), numeric: true },
            { label: 'Vorsteuer', value: formatCents(data.inputTax), numeric: true },
            { label: 'Vorauszahlungen', value: formatCents(data.prepaid), numeric: true },
            data.refund > 0
              ? { label: 'Erstattung', value: formatCents(data.refund), numeric: true }
              : { label: 'Zahllast', value: formatCents(data.payable), numeric: true },
          ]}
        />
      )}
    </KarteiFrame>
  );
};

const TaxProvisionKartei: React.FC<{ year: number }> = ({ year }) => {
  const { data, error } = useKartei<TaxProvisionPreview>(() => Api.previewTaxProvision(year), [year]);
  return (
    <KarteiFrame title="Steuern vom Einkommen und Ertrag" error={error} loading={data === null}>
      {data && (
        <>
          {data.warning && <Notice className="mb-4" text={data.warning} />}
          <DetailList
            items={[
              { label: 'Zu versteuerndes Einkommen', value: formatCents(data.taxableIncome), numeric: true },
              { label: 'Körperschaftsteuer', value: formatCents(data.corporateTax), numeric: true },
              { label: 'Solidaritätszuschlag', value: formatCents(data.solidarity), numeric: true },
              { label: 'Gewerbesteuer', value: formatCents(data.tradeTax), numeric: true },
              {
                label: 'Rückstellung',
                value: formatCents(data.incomeProvision + data.tradeProvision),
                numeric: true,
              },
              ...(data.incomeRefund + data.tradeRefund > 0
                ? [{ label: 'Erstattung', value: formatCents(data.incomeRefund + data.tradeRefund), numeric: true }]
                : []),
            ]}
          />
        </>
      )}
    </KarteiFrame>
  );
};

const InputTaxKartei: React.FC<{ year: number }> = ({ year }) => {
  const { data, error } = useKartei<InputTaxCorrectionYear>(() => Api.getInputTaxCorrections(year), [year]);
  return (
    <KarteiFrame
      title="Vorsteuerberichtigung"
      error={error}
      loading={data === null}
      empty={data?.rows.length === 0 ? 'Kein Wirtschaftsgut im Berichtigungszeitraum.' : undefined}
    >
      {data && (
        <Table density="kompakt">
          <Thead>
            <Tr>
              <Th>Wirtschaftsgut</Th>
              <Th numeric className="w-24">
                Anteil
              </Th>
              <Th numeric className="w-32">
                Berichtigung
              </Th>
              <Th className="w-24">Stand</Th>
            </Tr>
          </Thead>
          <Tbody>
            {data.rows.map((row) => (
              <Tr key={row.correction.id}>
                <Td className="whitespace-normal">
                  <span className="block">{row.correction.label}</span>
                  <span className="block text-caption text-ink-subtle">
                    {`${row.correction.firstFiscalYear}–${row.correction.lastFiscalYear}`}
                  </span>
                </Td>
                <Td numeric>{formatPermille(row.permille)}</Td>
                <Td numeric>{formatCents(row.assessment.amount)}</Td>
                <Td>{row.booked ? <StatusBadge status="gebucht" /> : <StatusBadge status="offen" />}</Td>
              </Tr>
            ))}
          </Tbody>
        </Table>
      )}
    </KarteiFrame>
  );
};

const CurrencyKartei: React.FC<{ year: number }> = ({ year }) => {
  const { data, error } = useKartei<ForeignCurrencyValuation>(
    () => Api.previewCurrencyValuation(year),
    [year],
  );
  return (
    <KarteiFrame
      title="Fremdwährung zum Stichtag"
      error={error}
      loading={data === null}
      empty={data?.items.length === 0 ? 'Keine Posten in fremder Währung.' : undefined}
    >
      {data && (
        <Table density="kompakt">
          <Thead>
            <Tr>
              <Th>Posten</Th>
              <Th numeric className="w-32">
                Buchwert
              </Th>
              <Th numeric className="w-32">
                Stichtag
              </Th>
              <Th numeric className="w-28">
                Wirkung
              </Th>
            </Tr>
          </Thead>
          <Tbody>
            {data.items.map((item) => (
              <Tr key={`${item.kind}-${item.entryNumber}-${item.account}`}>
                <Td className="whitespace-normal">
                  <span className="block">{item.description}</span>
                  <span className="block text-caption text-ink-subtle">
                    {`${formatCents(item.foreignAmount, item.currency)}${item.recognised ? '' : ' · nicht gebucht'}`}
                  </span>
                </Td>
                <Td numeric className="text-ink-muted">{formatCents(item.bookValue)}</Td>
                <Td numeric>{formatCents(item.valueAtCutoff)}</Td>
                <Td numeric>{formatCents(item.amount)}</Td>
              </Tr>
            ))}
          </Tbody>
        </Table>
      )}
    </KarteiFrame>
  );
};

const WriteUpKartei: React.FC<{ year: number }> = ({ year }) => {
  const { data, error } = useKartei<WriteUpReport>(() => Api.getWriteUpReport(year), [year]);
  return (
    <KarteiFrame
      title="Abgewertete Anlagen"
      error={error}
      loading={data === null}
      empty={data?.candidates.length === 0 ? 'Kein Anlagegut außerplanmäßig abgeschrieben.' : undefined}
    >
      {data && (
        <Table density="kompakt">
          <Thead>
            <Tr>
              <Th>Anlagegut</Th>
              <Th numeric className="w-32">
                Buchwert
              </Th>
              <Th numeric className="w-36">
                Höchstens zuschreiben
              </Th>
              <Th className="w-28">Grund besteht</Th>
            </Tr>
          </Thead>
          <Tbody>
            {data.candidates.map((c) => (
              <Tr key={c.assetId}>
                <Td>{`${c.inventoryNumber} · ${c.name}`}</Td>
                <Td numeric>{formatCents(c.bookValue)}</Td>
                <Td numeric>{formatCents(c.maxWriteUp)}</Td>
                <Td>{c.confirmed ? 'bestätigt' : 'offen'}</Td>
              </Tr>
            ))}
          </Tbody>
        </Table>
      )}
    </KarteiFrame>
  );
};

const SIZE_LABELS: Record<string, string> = {
  micro: 'Kleinstkapitalgesellschaft',
  small: 'Kleine Kapitalgesellschaft',
  medium: 'Mittelgroße Kapitalgesellschaft',
  large: 'Große Kapitalgesellschaft',
};

/**
 * Größenklasse und Anhang. Die Arbeitnehmerzahl ist das einzige Merkmal, das
 * Buchfink nicht aus den Büchern kennt; sie steht deshalb hier als Feld.
 */
const StatementKartei: React.FC<{ year: number }> = ({ year }) => {
  const writeLock = useWriteLock();
  const [reload, setReload] = useState(0);
  const { data, error } = useKartei<[SizeClass, NotesSectionText[]]>(
    () => Promise.all([Api.getSizeClass(year), Api.getNotesTexts(year)]),
    [year, reload],
  );
  const [employees, setEmployees] = useState('');
  const [fieldError, setFieldError] = useState('');
  const [saving, setSaving] = useState(false);
  const size = data?.[0];
  const notes = data?.[1] ?? [];

  useEffect(() => {
    if (size) setEmployees(String(size.criteria.employees));
  }, [size]);

  async function saveEmployees() {
    const count = Number.parseInt(employees, 10);
    if (!Number.isFinite(count) || count < 0) {
      setFieldError('Erwartet wird eine ganze Zahl ab null.');
      return;
    }
    setSaving(true);
    try {
      await Api.setAverageEmployees(year, count);
      setFieldError('');
      setReload((value) => value + 1);
    } catch (e) {
      setFieldError(message(e));
    } finally {
      setSaving(false);
    }
  }

  return (
    <>
      <KarteiFrame title="Größenklasse" error={error} loading={data === null}>
        {size && (
          <>
            <DetailList
              items={[
                { label: 'Klasse', value: SIZE_LABELS[size.class] ?? size.class },
                { label: 'Bilanzsumme', value: formatCents(size.criteria.balanceSheetTotal), numeric: true },
                { label: 'Umsatzerlöse', value: formatCents(size.criteria.revenue), numeric: true },
                { label: 'Offenlegung', value: size.obligations.disclosureScope },
              ]}
            />
            <Field
              label="Arbeitnehmer im Jahresdurchschnitt"
              hint="ohne Auszubildende"
              error={fieldError || undefined}
              className="mt-4 max-w-sm"
            >
              <div className="flex items-center gap-2">
                <Input
                  value={employees}
                  inputMode="numeric"
                  className="num"
                  onChange={(e) => {
                    setEmployees(e.target.value);
                    setFieldError('');
                  }}
                />
                <Button
                  variant="secondary"
                  loading={saving}
                  disabled={writeLock.locked}
                  title={writeLock.hint}
                  onClick={() => void saveEmployees()}
                >
                  Übernehmen
                </Button>
              </div>
            </Field>
          </>
        )}
      </KarteiFrame>
      {notes.length > 0 && (
        <Disclosure
          title="Anhang"
          context={`${notes.filter((n) => n.text.trim()).length} von ${notes.length} Abschnitten ausgefüllt`}
          className="mt-2"
        >
          <ul className="flex flex-col">
            {notes.map((note) => (
              <li key={note.section} className="py-2 border-t border-line first:border-t-0">
                <span className="block text-label text-ink-muted">{note.label}</span>
                <span className={cn('block text-body mt-0.5', note.text.trim() ? 'text-ink' : 'text-ink-subtle')}>
                  {note.text.trim() || 'noch nicht ausgefüllt'}
                </span>
              </li>
            ))}
          </ul>
        </Disclosure>
      )}
    </>
  );
};
