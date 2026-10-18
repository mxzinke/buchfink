import React, { useEffect, useMemo, useState } from 'react';
import type {
  AcquisitionCandidate,
  Anlagenspiegel,
  AnlagenspiegelRow,
  AssetClass,
  AssetDetail,
  DepreciationMethod,
  FixedAsset,
} from '../../types';
import { useNavigation, useRouteObject } from '../../navigation';
import { useWorkspace } from '../../workspace';
import { Api } from '../../services/api';
import { formatCents, formatDate, formatUnits } from '../../utils/formatters';
import {
  DetailList,
  DetailPanel,
  Disclosure,
  EmptyState,
  Help,
  Notice,
  RowButton,
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
} from '../../components/ui';
import { PanelHeading, errorText } from './shared';
import { splitNorms } from '../../utils/norms';

const CLASS_ORDER: AssetClass[] = ['intangible', 'tangible', 'financial'];

const CLASS_LABEL: Record<AssetClass, string> = {
  intangible: 'Immaterielle Werte',
  tangible: 'Sachanlagen',
  financial: 'Finanzanlagen',
};

const METHOD_LABEL: Record<DepreciationMethod, string> = {
  linear: 'Linear',
  degressive: 'Degressiv',
  electric_vehicle: 'Staffel für Elektrofahrzeuge',
  building_linear: 'Gebäude, fester Satz',
  pool: 'Sammelposten',
  immediate: 'Sofortabzug',
  none: 'Keine planmäßige Abschreibung',
};

const MOVEMENT_LABEL: Record<string, string> = {
  acquisition: 'Zugang',
  subsequent_cost: 'Nachträgliche Anschaffungskosten',
  cost_reduction: 'Minderung der Anschaffungskosten',
  depreciation: 'Planmäßige Abschreibung',
  special_depreciation: 'Sonderabschreibung',
  impairment: 'Außerplanmäßige Abschreibung',
  write_up: 'Zuschreibung',
  income: 'Laufender Ertrag',
  vorabpauschale: 'Vorabpauschale',
  disposal: 'Abgang',
  transfer: 'Umbuchung',
};

/** Der Zustand eines Anlageguts in einem Wort, mit dem offenen Betrag daneben. */
const AssetState: React.FC<{ asset: FixedAsset }> = ({ asset }) => {
  switch (asset.status) {
    case 'depreciate_due':
      return (
        <span className="inline-flex items-center gap-2" title={asset.statusNote}>
          <StatusBadge status="offen" />
          <span className="text-caption text-ink-subtle num">
            {formatCents(asset.dueAmount + asset.specialDue)}
          </span>
        </span>
      );
    case 'unbooked':
      return <StatusBadge status="entwurf" />;
    case 'disposed':
      return (
        <span className="text-caption text-ink-subtle">{`Abgang ${formatDate(asset.disposalDate ?? '')}`}</span>
      );
    case 'fully_written':
      return <span className="text-caption text-ink-subtle">Abgeschrieben</span>;
    default:
      return <span className="text-caption text-ink-subtle">Im Bestand</span>;
  }
};

/**
 * Das Anlagevermögen zum Nachlesen: Verzeichnis, Abschreibungsplan je
 * Anlagegut und Anlagenspiegel. Erfasst, abgeschrieben und abgegangen wird
 * über Claude; die App zeigt den Stand.
 */
export const AssetsTab: React.FC = () => {
  const { year } = useWorkspace();
  const { id, open, close } = useRouteObject('books', 'assets');
  const [assets, setAssets] = useState<FixedAsset[]>([]);
  const [candidates, setCandidates] = useState<AcquisitionCandidate[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    Promise.all([Api.getFixedAssets(''), Api.getAssetAcquisitionCandidates()])
      .then(([list, pending]) => {
        if (cancelled) return;
        setAssets(list ?? []);
        setCandidates(pending ?? []);
        setError(null);
      })
      .catch((e) => !cancelled && setError(errorText(e)))
      .finally(() => !cancelled && setLoading(false));
    return () => {
      cancelled = true;
    };
  }, [year]);

  const inStock = assets.filter((a) => a.status !== 'disposed');
  const sum = (pick: (a: FixedAsset) => number) => inStock.reduce((total, a) => total + pick(a), 0);
  const due = sum((a) => a.dueAmount + a.specialDue);

  const grouped = useMemo(
    () =>
      CLASS_ORDER.map((assetClass) => ({
        assetClass,
        rows: assets
          .filter((a) => a.class === assetClass)
          .sort((a, b) => a.inventoryNumber.localeCompare(b.inventoryNumber)),
      })).filter((group) => group.rows.length > 0),
    [assets],
  );

  if (error) return <Notice tone="negative" text={error} />;
  if (loading) return <SkeletonRows rows={8} />;

  return (
    <div>
      <StatRow>
        <Stat
          label="Anschaffungskosten"
          value={formatCents(sum((a) => a.cost))}
          context={`${inStock.length} im Bestand`}
        />
        <Stat label="Buchwert" value={formatCents(sum((a) => a.bookValue))} />
        <Stat
          label={`Abschreibung ${year}`}
          value={formatCents(sum((a) => a.yearAmount))}
          context={due > 0 ? `${formatCents(due)} noch nicht gebucht` : 'vollständig gebucht'}
        />
      </StatRow>

      {candidates.length > 0 && (
        <Notice
          className="mt-6"
          text={
            candidates.length === 1
              ? 'Eine Buchung auf einem Anlagekonto hat noch kein Anlagegut im Verzeichnis.'
              : `${candidates.length} Buchungen auf Anlagekonten haben noch kein Anlagegut im Verzeichnis.`
          }
        />
      )}

      <div className="mt-8">
        {assets.length === 0 ? (
          <EmptyState
            title="Noch kein Anlagevermögen"
            description="Was länger als ein Jahr genutzt wird und über der Wertgrenze liegt, steht hier."
          />
        ) : (
          <Table density="kompakt">
            <Thead sticky>
              <Tr>
                <Th className="w-32">Inventarnummer</Th>
                <Th>Bezeichnung</Th>
                <Th optional className="w-28">
                  Anschaffung
                </Th>
                <Th numeric optional className="w-36">
                  Anschaffungskosten
                </Th>
                <Th numeric className="w-32">
                  Buchwert
                </Th>
                <Th numeric className="w-28">{`AfA ${year}`}</Th>
                <Th className="w-40">Stand</Th>
              </Tr>
            </Thead>
            <Tbody>
              {grouped.map((group) => (
                <React.Fragment key={group.assetClass}>
                  <Tr>
                    <Td colSpan={7} className="bg-sunken text-overline text-ink-subtle">
                      {CLASS_LABEL[group.assetClass]}
                    </Td>
                  </Tr>
                  {group.rows.map((asset) => (
                    <Tr
                      key={asset.id}
                      variant={String(asset.id) === id ? 'selected' : 'default'}
                      className="cursor-pointer"
                      onClick={() => open(String(asset.id))}
                    >
                      <Td code>
                        <RowButton onOpen={() => open(String(asset.id))}>{asset.inventoryNumber}</RowButton>
                      </Td>
                      <Td fill title={asset.name}>
                        {asset.name}
                      </Td>
                      <Td optional className="text-ink-subtle num">
                        {formatDate(asset.acquisitionDate)}
                      </Td>
                      <Td numeric optional>
                        {formatCents(asset.cost)}
                      </Td>
                      <Td numeric>{formatCents(asset.bookValue)}</Td>
                      <Td numeric>{asset.yearAmount === 0 ? '—' : formatCents(asset.yearAmount)}</Td>
                      <Td>
                        <AssetState asset={asset} />
                      </Td>
                    </Tr>
                  ))}
                </React.Fragment>
              ))}
            </Tbody>
          </Table>
        )}
      </div>

      <Disclosure title="Anlagenspiegel" context={`Entwicklung des Anlagevermögens ${year}`}>
        <Spiegel year={year} />
      </Disclosure>

      <AssetPanel assetId={id ? Number(id) : undefined} onClose={close} />
    </div>
  );
};

/** Der Anlagenspiegel: Anschaffungskosten und Abschreibungen je Konto, mit Summen je Klasse. */
const Spiegel: React.FC<{ year: number }> = ({ year }) => {
  const [spiegel, setSpiegel] = useState<Anlagenspiegel | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    Api.getAnlagenspiegel()
      .then(setSpiegel)
      .catch((e) => setError(errorText(e)));
  }, []);

  if (error) return <Notice tone="negative" text={error} />;
  if (!spiegel) return <SkeletonRows rows={6} />;
  if (spiegel.rows.length === 0) return <EmptyState title="Noch kein Anlagevermögen" />;

  const cell = 'px-2.5';
  const row = (r: AnlagenspiegelRow, key: string, sum = false) => (
    <Tr key={key} variant={sum ? 'sum' : 'default'}>
      <Td code className={cell}>
        {sum ? '' : r.account}
      </Td>
      <Td className={`${cell} max-w-[14rem] truncate`} title={r.accountName}>
        {r.accountName}
      </Td>
      <Td numeric className={cell}>
        {formatCents(r.costOpening)}
      </Td>
      <Td numeric className={cell}>
        {formatCents(r.additions)}
      </Td>
      <Td numeric className={cell}>
        {formatCents(r.disposals)}
      </Td>
      <Td numeric className={cell}>
        {formatCents(r.costClosing)}
      </Td>
      <Td numeric className={cell}>
        {formatCents(r.depreciationYear)}
      </Td>
      <Td numeric className={cell}>
        {formatCents(r.depreciationClosing)}
      </Td>
      <Td numeric className={cell}>
        {formatCents(r.bookValueClosing)}
      </Td>
    </Tr>
  );

  return (
    <Table density="kompakt">
      <Thead>
        <Tr>
          <Th className={`${cell} w-16`}>Konto</Th>
          <Th className={cell}>Position</Th>
          <Th numeric className={cell}>
            Stand 01.01.
          </Th>
          <Th numeric className={cell}>
            Zugänge
          </Th>
          <Th numeric className={cell}>
            Abgänge
          </Th>
          <Th numeric className={cell}>
            Stand 31.12.
          </Th>
          <Th numeric className={cell}>{`AfA ${year}`}</Th>
          <Th numeric className={cell}>
            AfA gesamt
          </Th>
          <Th numeric className={cell}>
            Buchwert
          </Th>
        </Tr>
      </Thead>
      <Tbody>
        {CLASS_ORDER.map((assetClass) => {
          const rows = spiegel.rows.filter((r) => r.class === assetClass);
          if (rows.length === 0) return null;
          const total = spiegel.classTotals.find((t) => t.class === assetClass);
          return (
            <React.Fragment key={assetClass}>
              <Tr>
                <Td colSpan={9} className="bg-sunken text-overline text-ink-subtle">
                  {CLASS_LABEL[assetClass]}
                </Td>
              </Tr>
              {rows.map((r) => row(r, `${assetClass}-${r.account}`))}
              {total && row(total, `${assetClass}-total`, true)}
            </React.Fragment>
          );
        })}
        {row(spiegel.totals, 'total', true)}
      </Tbody>
    </Table>
  );
};

/** Die Seitenleiste eines Anlageguts: Angaben, Abschreibungsplan, Bewegungen. */
const AssetPanel: React.FC<{ assetId: number | undefined; onClose: () => void }> = ({ assetId, onClose }) => {
  const { navigate } = useNavigation();
  const [detail, setDetail] = useState<AssetDetail | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!assetId) return;
    let cancelled = false;
    setError(null);
    setDetail((current) => (current?.asset.id === assetId ? current : null));
    Api.getFixedAsset(assetId)
      .then((result) => !cancelled && setDetail(result))
      .catch((e) => !cancelled && setError(errorText(e)));
    return () => {
      cancelled = true;
    };
  }, [assetId]);

  const asset = detail?.asset;
  const facts = asset
    ? [
        { label: 'Konto', value: `${asset.account} · ${asset.accountName ?? ''}` },
        { label: 'Angeschafft', value: formatDate(asset.acquisitionDate), numeric: true },
        ...(asset.inServiceDate
          ? [{ label: 'In Betrieb seit', value: formatDate(asset.inServiceDate), numeric: true }]
          : []),
        { label: 'Anschaffungskosten', value: formatCents(asset.cost), numeric: true },
        { label: 'Abschreibungen', value: formatCents(asset.accumulated), numeric: true },
        { label: 'Buchwert', value: formatCents(asset.bookValue), numeric: true },
        {
          label: 'Methode',
          value:
            METHOD_LABEL[asset.method] +
            (asset.usefulLifeMonths ? ` · ${asset.usefulLifeMonths} Monate` : ''),
        },
        ...(asset.unitsHeld
          ? [{ label: 'Bestand', value: `${formatUnits(asset.unitsHeld)} Stück`, numeric: true }]
          : []),
        ...(asset.identifier ? [{ label: 'Kennung', value: asset.identifier }] : []),
        ...(asset.usefulLifeReason ? [{ label: 'Nutzungsdauer', value: asset.usefulLifeReason }] : []),
      ]
    : [];

  return (
    <DetailPanel
      open={Boolean(assetId)}
      onOpenChange={(next) => !next && onClose()}
      title={asset ? asset.name : ''}
      context={asset ? `${asset.inventoryNumber} · ${CLASS_LABEL[asset.class]}` : undefined}
      meta={asset ? <AssetState asset={asset} /> : undefined}
      width="lg"
    >
      {error ? (
        <Notice tone="negative" text={error} />
      ) : !detail || !asset ? (
        <SkeletonRows rows={6} />
      ) : (
        <>
          <DetailList items={facts} />

          {detail.schedule.length > 0 && (
            <div className="mt-6">
              <PanelHeading>Abschreibungsplan</PanelHeading>
              <table className="w-full text-body">
                <thead>
                  <tr className="[&>th]:h-7 [&>th]:text-label [&>th]:font-medium [&>th]:text-ink-subtle [&>th]:border-b [&>th]:border-line-strong">
                    <th className="text-left w-14">Jahr</th>
                    <th className="text-left">Satz</th>
                    <th className="text-right w-28">Anfang</th>
                    <th className="text-right w-28">Abschreibung</th>
                    <th className="text-right w-28">Ende</th>
                    <th className="text-left w-20 pl-3">Stand</th>
                  </tr>
                </thead>
                <tbody>
                  {detail.schedule.map((row) => (
                    <tr key={row.fiscalYear} className="[&>td]:h-8 [&>td]:border-b [&>td]:border-line">
                      <td className="num">{row.fiscalYear}</td>
                      <td className="text-ink-muted truncate max-w-0" title={row.note}>
                        {row.rateLabel}
                      </td>
                      <td className="text-right num">{formatCents(row.openingBookValue)}</td>
                      <td className="text-right num">{formatCents(row.amount + (row.specialAmount ?? 0))}</td>
                      <td className="text-right num">{formatCents(row.closingBookValue)}</td>
                      <td className="pl-3 text-caption text-ink-subtle">{row.status}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          <div className="mt-6">
            <PanelHeading>Bewegungen</PanelHeading>
            <ul className="divide-y divide-line border-y border-line">
              {detail.movements.map((movement) => (
                <li key={movement.id} className="flex items-center gap-3 py-2 text-body">
                  <span className="num text-ink-subtle w-24 shrink-0">{formatDate(movement.date)}</span>
                  <span className="flex-1 truncate" title={movement.note}>
                    {MOVEMENT_LABEL[movement.kind] ?? movement.kind}
                  </span>
                  <span className="num">
                    {formatCents(
                      movement.costAmount !== 0 ? movement.costAmount : movement.depreciationAmount,
                    )}
                  </span>
                  {movement.entryNumber ? (
                    <button
                      type="button"
                      className="code-num text-caption text-accent-text hover:underline w-28 text-right"
                      onClick={() => navigate({ area: 'books', tab: 'journal', id: movement.entryNumber })}
                    >
                      {movement.entryNumber}
                    </button>
                  ) : (
                    <span className="w-28" />
                  )}
                </li>
              ))}
            </ul>
          </div>

          {detail.notes.length > 0 && (
            <div className="mt-6 space-y-2">
              {detail.notes.map((note, index) => {
                const { text, norms } = splitNorms(note);
                return (
                  <p key={index} className="text-caption text-ink-subtle">
                    {text}
                    {norms.length > 0 && (
                      <Help summary="Auf diesen Vorschriften beruht der Hinweis." label="Grundlage des Hinweises">
                        {norms.join(', ')}
                      </Help>
                    )}
                  </p>
                );
              })}
            </div>
          )}
        </>
      )}
    </DetailPanel>
  );
};
