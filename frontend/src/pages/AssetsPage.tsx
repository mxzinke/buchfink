import React, { useEffect, useMemo, useState } from 'react';
import { AlertCircle, ArrowLeft, Building2, Plus } from 'lucide-react';
import {
  AcquisitionAdvice,
  AcquisitionCandidate,
  Anlagenspiegel,
  AnlagenspiegelRow,
  Account,
  AssetAccountInfo,
  AssetClass,
  AssetDetail,
  AssetRules,
  Cents,
  Contact,
  DepreciationRun,
  DisposalKind,
  DisposalPreview,
  DisposalRequest,
  FixedAsset,
  JournalLine,
  Settlement,
  TaxTreatment,
} from '../types';
import { Api } from '../services/api';
import { formatCents, formatCentsPlain, formatDate, parseCents } from '../utils/formatters';
import {
  Button,
  Checkbox,
  Combobox,
  Dialog,
  EmptyState,
  Field,
  HelpPopover,
  Input,
  PageHeader,
  RadioGroup,
  Section,
  Select,
  SkeletonRows,
  Stat,
  StatRow,
  StatusBadge,
  TabPanel,
  Table,
  Tabs,
  Tbody,
  Td,
  Textarea,
  Th,
  Thead,
  Tr,
  toast,
} from '../components/ui';

/**
 * Anlagevermögen: Verzeichnis, Abschreibungen und Anlagenspiegel.
 *
 * Die Ansicht erklärt sich selbst, rechnet aber nichts nach. Wertgrenzen,
 * Zeitfenster der degressiven AfA, die Wahl des Erlöskontos beim Abgang und die
 * Sätze zu jedem einzelnen Anlagegut kommen aus dem Backend — eine zweite
 * Fassung derselben Regel im Frontend driftet, sobald sich eine davon ändert.
 *
 * Die Trennung nach Sach- und Finanzanlagen ist keine Kosmetik: Sachanlagen
 * nutzen sich ab und werden planmäßig abgeschrieben, Finanzanlagen nicht. Für
 * sie gibt es nur die außerplanmäßige Abschreibung und die Zuschreibung. Zwei
 * Reiter, zwei Erklärungen, zwei Aktionsleisten.
 */

type Tab = 'tangible' | 'financial' | 'intangible' | 'depreciation' | 'spiegel';

const CLASS_TABS: AssetClass[] = ['tangible', 'financial', 'intangible'];

const CLASS_LABEL: Record<AssetClass, string> = {
  tangible: 'Sachanlagen',
  financial: 'Finanzanlagen',
  intangible: 'Immaterielle Werte',
};

/** Die Anlagenklasse einer Kontonummer nach dem Aufbau der Kontenklasse 0. */
function classOfAccount(account: string): AssetClass | null {
  const number = Number(account);
  if (!Number.isFinite(number) || number < 100 || number >= 1000) return null;
  if (number < 200) return 'intangible';
  if (number < 800) return 'tangible';
  return 'financial';
}

/**
 * Erklärungen laufen über die drei Stufen aus §15.2: eine Zeile Kontext in der
 * Ansicht, bis zu drei Sätze im Popover, alles Weitere im Dialog hinter „Mehr
 * dazu".
 *
 * Eine Arbeitsansicht enthält keinen Fließtext. Wer täglich damit arbeitet,
 * liest den Erklärsatz beim zwanzigsten Mal nicht mehr, sondern scrollt an ihm
 * vorbei — und die Anlagenbuchhaltung hat genug zu erklären, um eine Ansicht
 * damit zuzuschütten.
 */
interface Explanation {
  title: string;
  /** Eine Zeile, die in der Ansicht stehen bleibt. */
  line: string;
  /** Bis drei Sätze im Popover. */
  short: React.ReactNode;
  /** Der lange Text, nur auf Klick. */
  full: React.ReactNode;
}

const ExplainLine: React.FC<{ explanation: Explanation; onMore: () => void }> = ({
  explanation,
  onMore,
}) => (
  <div className="flex items-center gap-1 text-caption text-ink-subtle">
    <span>{explanation.line}</span>
    <HelpPopover label={`Erklärung zu ${explanation.title}`} onMore={onMore}>
      {explanation.short}
    </HelpPopover>
  </div>
);

const ExplainDialog: React.FC<{ explanation: Explanation | null; onClose: () => void }> = ({
  explanation,
  onClose,
}) => (
  <Dialog
    open={explanation !== null}
    onOpenChange={(next) => !next && onClose()}
    title={explanation?.title ?? ''}
    width="max-w-2xl"
    footer={
      <Button variant="secondary" onClick={onClose}>
        Schließen
      </Button>
    }
  >
    <div className="text-body text-ink-muted space-y-3">{explanation?.full}</div>
  </Dialog>
);

/** In den Masken bleibt es bei zwei Stufen — ein Dialog im Dialog hilft niemandem. */
const FormHint: React.FC<{ label: string; line: string; children: React.ReactNode }> = ({
  label,
  line,
  children,
}) => (
  <div className="flex items-center gap-1 text-caption text-ink-subtle">
    <span>{line}</span>
    <HelpPopover label={label}>{children}</HelpPopover>
  </div>
);

const Notice: React.FC<{ tone?: 'attention' | 'negative'; children: React.ReactNode }> = ({
  tone = 'attention',
  children,
}) => (
  <div
    className={
      tone === 'negative'
        ? 'flex items-start gap-2.5 rounded-control border border-negative-line bg-negative-soft px-4 py-3'
        : 'flex items-start gap-2.5 rounded-control border border-attention-line bg-attention-soft px-4 py-3'
    }
  >
    <AlertCircle
      className={`w-4 h-4 mt-0.5 shrink-0 ${tone === 'negative' ? 'text-negative' : 'text-attention'}`}
      strokeWidth={1.5}
    />
    <div className={`text-body ${tone === 'negative' ? 'text-negative-text' : 'text-attention-text'}`}>
      {children}
    </div>
  </div>
);

type Topic = 'tangible' | 'financial' | 'intangible' | 'depreciation' | 'spiegel';

/**
 * Die Texte stehen beieinander, damit die Ansicht nur noch eine Zeile davon
 * zeigt. Die Wertgrenzen kommen aus dem Backend — sie stehen hier nicht ein
 * zweites Mal als Zahl im Text.
 */
function explanations(rules: AssetRules | null, year: number): Record<Topic, Explanation> {
  const gwg = formatCents(rules?.gwgImmediateLimit ?? 0);
  const poolFrom = formatCents(rules?.poolLowerLimit ?? 0);
  const poolTo = formatCents(rules?.poolUpperLimit ?? 0);
  const record = formatCents(rules?.gwgRecordFrom ?? 0);

  return {
    tangible: {
      title: 'Sachanlagen',
      line: 'Erst die Wertgrenze, dann die Abschreibungsmethode.',
      short: (
        <>
          Bis {gwg} netto ist der Sofortabzug möglich, bis {poolTo} der Sammelposten, darüber wird
          aktiviert und über die Nutzungsdauer abgeschrieben. Die eigentliche Hürde ist dabei nicht
          der Betrag, sondern die selbständige Nutzbarkeit. Buchfink fragt sie beim Erfassen ab,
          statt sie zu raten.
        </>
      ),
      full: (
        <>
          <p>
            Bei jeder Anschaffung steht eine Entscheidung <em>vor</em> der Abschreibungsmethode, und
            sie bestimmt den ganzen weiteren Verlauf:
          </p>
          <ul className="list-disc pl-5 space-y-1">
            <li>
              <strong>Sofortabzug</strong> bis {gwg} netto — voller Aufwand im Anschaffungsjahr
              (§ 6 Abs. 2 Satz 1 EStG).
            </li>
            <li>
              <strong>Sammelposten</strong> von {poolFrom} bis {poolTo} — ein Pool je
              Wirtschaftsjahr, aufgelöst mit je einem Fünftel über {rules?.poolYears ?? 5} Jahre
              (§ 6 Abs. 2a EStG).
            </li>
            <li>
              <strong>Aktivierung</strong> darüber — planmäßige AfA über die betriebsgewöhnliche
              Nutzungsdauer (§ 7 Abs. 1 EStG).
            </li>
          </ul>
          <p>
            Zwei Fallen stecken darin. Die eigentliche Hürde ist nicht der Betrag, sondern die{' '}
            <strong>selbständige Nutzbarkeit</strong>: ein Bildschirm für 300 € ist ohne Rechner
            nicht nutzbar und damit kein GWG. Und das{' '}
            <strong>Sammelposten-Wahlrecht gilt einheitlich</strong> für alle Wirtschaftsgüter eines
            Jahres — wer einmal poolt, poolt für dieses Jahr durchgehend.
          </p>
          <p>
            Ab {record} muss ein geringwertiges Wirtschaftsgut in ein laufend geführtes Verzeichnis
            (§ 6 Abs. 2 Satz 4 EStG) — dieses Verzeichnis erfüllt das.
          </p>
        </>
      ),
    },
    financial: {
      title: 'Finanzanlagen',
      line: 'Finanzanlagen werden nicht planmäßig abgeschrieben.',
      short: (
        <>
          Sie nutzen sich nicht ab und stehen mit ihren Anschaffungskosten in der Bilanz. Wertverlust
          wird außerplanmäßig erfasst — bei Finanzanlagen auch bei einer nur vorübergehenden
          Wertminderung (§ 253 Abs. 3 Satz 6 HGB). Fällt der Grund später weg, ist wieder
          zuzuschreiben.
        </>
      ),
      full: (
        <>
          <p>
            Hier stehen Beteiligungen, Anteile an verbundenen Unternehmen, Wertpapiere des
            Anlagevermögens und Ausleihungen — alles, was <em>dauernd</em> dem Geschäftsbetrieb
            dienen soll. Was nur vorübergehend gehalten wird, gehört ins Umlaufvermögen und
            unterliegt dort strengeren Bewertungsregeln.
          </p>
          <p>
            Für Finanzanlagen gilt das <em>gemilderte</em> Niederstwertprinzip: bei voraussichtlich
            dauernder Wertminderung <em>ist</em> abzuschreiben, bei einer nicht dauernden{' '}
            <em>darf</em> abgeschrieben werden (§ 253 Abs. 3 Sätze 5 und 6 HGB). Fällt der Grund
            später weg, ist wieder zuzuschreiben — höchstens bis zu den Anschaffungskosten
            (§ 253 Abs. 5 Satz 1 HGB). Das ist ein Gebot, kein Wahlrecht.
          </p>
          <p>
            Beide Vorgänge stehen im Anlagegut selbst: öffne eine Zeile und wähle „Außerplanmäßig
            abschreiben" oder „Zuschreiben". Der Grund gehört zwingend dazu — ohne ihn kann später
            niemand mehr nachvollziehen, warum der Wert gefallen ist.
          </p>
        </>
      ),
    },
    intangible: {
      title: 'Immaterielle Vermögensgegenstände',
      line: 'Nur entgeltlich erworbene Werte gehören hierher.',
      short: (
        <>
          Software, Lizenzen, Konzessionen und der Geschäfts- oder Firmenwert. Selbst geschaffene
          Werte dürfen handelsrechtlich nur wahlweise aktiviert werden (§ 248 Abs. 2 HGB), steuerlich
          gar nicht. Abgeschrieben wird planmäßig über die Nutzungsdauer wie bei den Sachanlagen.
        </>
      ),
      full: (
        <>
          <p>
            Software, Lizenzen, Konzessionen, gewerbliche Schutzrechte und der Geschäfts- oder
            Firmenwert. Angesetzt werden dürfen nur <em>entgeltlich erworbene</em> Werte; für selbst
            geschaffene besteht handelsrechtlich lediglich ein Wahlrecht (§ 248 Abs. 2 HGB) und
            steuerlich ein Ansatzverbot.
          </p>
          <p>
            Abgeschrieben wird planmäßig über die Nutzungsdauer wie bei den Sachanlagen. Der
            Geschäfts- oder Firmenwert ist der Sonderfall: steuerlich über 15 Jahre
            (§ 7 Abs. 1 Satz 3 EStG), und eine Zuschreibung auf ihn ist ausgeschlossen
            (§ 253 Abs. 5 Satz 2 HGB).
          </p>
        </>
      ),
    },
    depreciation: {
      title: 'Abschreibungslauf',
      line: 'Die Abschreibung ist eine Abschlussbuchung zum Bilanzstichtag, kein laufender Geschäftsvorfall.',
      short: (
        <>
          Buchfink bucht sie deshalb nie im Hintergrund: hier steht, was für {year} fällig ist, und
          gebucht wird auf Freigabe. Gerechnet wird monatsgenau ab dem Anschaffungsmonat
          (§ 7 Abs. 1 Satz 4 EStG). Vor der Festschreibung eines ganzen Jahres prüft Buchfink, ob
          hier noch etwas offen ist.
        </>
      ),
      full: (
        <>
          <p>
            Die AfA entsteht nicht nebenbei im Lauf des Jahres, sondern zum Bilanzstichtag. Buchfink
            bucht sie deshalb nie im Hintergrund: hier steht, was für {year} fällig ist, und gebucht
            wird auf Freigabe — eine Buchung je Anlagegut, damit der Bezug in beide Richtungen trägt.
          </p>
          <p>
            Gerechnet wird monatsgenau ab dem Anschaffungsmonat (§ 7 Abs. 1 Satz 4 EStG). Ein im
            September angeschafftes Wirtschaftsgut trägt im ersten Jahr vier Zwölftel.
          </p>
          <p>
            Vor der Festschreibung eines ganzen Jahres prüft Buchfink, ob hier noch etwas offen ist.
            Ein festgeschriebenes Jahr nimmt keine Buchung mehr auf — die fehlende Abschreibung ließe
            sich danach nicht mehr nachholen.
          </p>
        </>
      ),
    },
    spiegel: {
      title: 'Anlagenspiegel',
      line: 'Die Entwicklung jeder Position über das Geschäftsjahr.',
      short: (
        <>
          Anfangsbestand, Zugänge, Abgänge, Abschreibungen und Buchwert am Ende. Für
          Kapitalgesellschaften ist der Anlagenspiegel Bestandteil des Anhangs (§ 284 Abs. 3 HGB);
          kleine Kapitalgesellschaften sind davon befreit (§ 288 Abs. 1 Nr. 1 HGB). Er ist keine
          Buchung, sondern eine Auswertung.
        </>
      ),
      full: (
        <>
          <p>
            Der Anlagenspiegel zeigt für jeden Posten, was am Anfang da war, was hinzukam, was abging
            und wie viel abgeschrieben wurde. Für Kapitalgesellschaften ist er Bestandteil des
            Anhangs (§ 284 Abs. 3 HGB); kleine Kapitalgesellschaften sind davon befreit
            (§ 288 Abs. 1 Nr. 1 HGB).
          </p>
          <p>
            Er ist keine zusätzliche Buchung, sondern eine Auswertung — aber eine, die nur
            funktioniert, weil die Anlagenkartei jahresübergreifend geführt wird. Ein 2019
            angeschafftes Wirtschaftsgut steht {year} noch mit seinen Zugängen und seiner kumulierten
            Abschreibung da, obwohl das Journal pro Geschäftsjahr organisiert ist.
          </p>
        </>
      ),
    },
  };
}

export const AssetsPage: React.FC = () => {
  const [tab, setTab] = useState<Tab>('tangible');
  const [loading, setLoading] = useState(true);
  const [assets, setAssets] = useState<FixedAsset[]>([]);
  const [rules, setRules] = useState<AssetRules | null>(null);
  const [run, setRun] = useState<DepreciationRun | null>(null);
  const [spiegel, setSpiegel] = useState<Anlagenspiegel | null>(null);
  const [candidates, setCandidates] = useState<AcquisitionCandidate[]>([]);
  const [accounts, setAccounts] = useState<AssetAccountInfo[]>([]);
  const [contacts, setContacts] = useState<Contact[]>([]);
  const [paymentAccounts, setPaymentAccounts] = useState<Account[]>([]);

  const [editing, setEditing] = useState<Partial<FixedAsset> | null>(null);
  const [detailId, setDetailId] = useState<number | null>(null);
  const [topic, setTopic] = useState<Topic | null>(null);

  useEffect(() => {
    void loadAll();
  }, []);

  async function loadAll() {
    setLoading(true);
    try {
      const [list, ruleSet, runData, spiegelData, candidateList, accountList, contactList, payment] =
        await Promise.all([
          Api.getFixedAssets(''),
          Api.getAssetRules(),
          Api.getDepreciationRun(),
          Api.getAnlagenspiegel(),
          Api.getAssetAcquisitionCandidates(),
          Api.getAssetAccounts(''),
          Api.getContacts(),
          Api.getPaymentAccounts(),
        ]);
      setAssets(list ?? []);
      setRules(ruleSet);
      setRun(runData);
      setSpiegel(spiegelData);
      setCandidates(candidateList ?? []);
      setAccounts(accountList ?? []);
      setContacts(contactList ?? []);
      setPaymentAccounts(payment ?? []);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }

  const year = rules?.fiscalYear ?? new Date().getFullYear();
  const activeClass: AssetClass | null = CLASS_TABS.includes(tab as AssetClass)
    ? (tab as AssetClass)
    : null;

  const byClass = useMemo(() => {
    const map: Record<AssetClass, FixedAsset[]> = { tangible: [], financial: [], intangible: [] };
    for (const asset of assets) map[asset.class]?.push(asset);
    return map;
  }, [assets]);

  const dueCount = run?.due.filter((d) => d.due > 0).length ?? 0;
  const explain = useMemo(() => explanations(rules, year), [rules, year]);

  return (
    <div className="max-w-[1200px] mx-auto px-8 py-8">
      <PageHeader
        title="Anlagevermögen"
        context={`Geschäftsjahr ${year} · Verzeichnis, Abschreibungen und Anlagenspiegel`}
        action={
          <Button
            variant="primary"
            icon={<Plus className="w-4 h-4" strokeWidth={1.5} />}
            onClick={() =>
              setEditing({
                class: activeClass ?? 'tangible',
                method: activeClass === 'financial' ? 'none' : 'linear',
                acquisitionDate: new Date().toISOString().slice(0, 10),
              })
            }
          >
            Anlagegut erfassen
          </Button>
        }
      />

      <Tabs
        items={[
          { value: 'tangible' as Tab, label: 'Sachanlagen', count: byClass.tangible.length },
          { value: 'financial' as Tab, label: 'Finanzanlagen', count: byClass.financial.length },
          { value: 'intangible' as Tab, label: 'Immaterielle Werte', count: byClass.intangible.length },
          { value: 'depreciation' as Tab, label: 'Abschreibungen', count: dueCount },
          { value: 'spiegel' as Tab, label: 'Anlagenspiegel' },
        ]}
        value={tab}
        onValueChange={setTab}
        className="mt-6"
      >
        {CLASS_TABS.map((assetClass) => (
          <TabPanel key={assetClass} value={assetClass}>
            {loading ? (
              <SkeletonRows rows={6} />
            ) : (
              <RegisterTab
                assetClass={assetClass}
                assets={byClass[assetClass]}
                explanation={explain[assetClass]}
                onExplain={() => setTopic(assetClass)}
                year={year}
                candidates={candidates.filter((c) => classOfAccount(c.account) === assetClass)}
                onOpen={setDetailId}
                onCreate={(prefill) =>
                  setEditing({
                    class: assetClass,
                    method: assetClass === 'financial' ? 'none' : 'linear',
                    acquisitionDate: new Date().toISOString().slice(0, 10),
                    ...prefill,
                  })
                }
              />
            )}
          </TabPanel>
        ))}

        <TabPanel value="depreciation">
          {loading ? (
            <SkeletonRows rows={6} />
          ) : (
            <DepreciationTab
              run={run}
              year={year}
              explanation={explain.depreciation}
              onExplain={() => setTopic('depreciation')}
              onBooked={loadAll}
            />
          )}
        </TabPanel>

        <TabPanel value="spiegel">
          {loading ? (
            <SkeletonRows rows={8} />
          ) : (
            <SpiegelTab
              spiegel={spiegel}
              year={year}
              explanation={explain.spiegel}
              onExplain={() => setTopic('spiegel')}
            />
          )}
        </TabPanel>
      </Tabs>

      <AssetFormDialog
        draft={editing}
        accounts={accounts}
        rules={rules}
        candidates={candidates}
        contacts={contacts}
        year={year}
        onClose={() => setEditing(null)}
        onSaved={async (asset) => {
          setEditing(null);
          toast.success(`${asset.inventoryNumber} gespeichert.`);
          await loadAll();
        }}
      />

      <ExplainDialog
        explanation={topic ? explain[topic] : null}
        onClose={() => setTopic(null)}
      />

      <AssetDetailDialog
        assetId={detailId}
        contacts={contacts}
        paymentAccounts={paymentAccounts}
        onClose={() => setDetailId(null)}
        onEdit={(asset) => {
          setDetailId(null);
          setEditing(asset);
        }}
        onChanged={loadAll}
      />
    </div>
  );
};

// -------------------------------------------------------------------------
// Verzeichnis je Anlagenklasse
// -------------------------------------------------------------------------

const RegisterTab: React.FC<{
  assetClass: AssetClass;
  assets: FixedAsset[];
  explanation: Explanation;
  onExplain: () => void;
  year: number;
  candidates: AcquisitionCandidate[];
  onOpen: (id: number) => void;
  onCreate: (prefill: Partial<FixedAsset>) => void;
}> = ({ assetClass, assets, explanation, onExplain, year, candidates, onOpen, onCreate }) => {
  const inStock = assets.filter((a) => a.status !== 'disposed');
  const disposed = assets.filter((a) => a.status === 'disposed');

  const sum = (pick: (a: FixedAsset) => Cents) => inStock.reduce((total, a) => total + pick(a), 0);

  return (
    <div className="space-y-6">
      <ExplainLine explanation={explanation} onMore={onExplain} />

      <StatRow>
        <Stat label="Anschaffungskosten" value={formatCents(sum((a) => a.cost))} context={`${inStock.length} im Bestand`} />
        <Stat label="Kumulierte Abschreibungen" value={formatCents(sum((a) => a.accumulated))} />
        <Stat label="Buchwert" value={formatCents(sum((a) => a.bookValue))} context={`Stand Geschäftsjahr ${year}`} />
        <Stat
          label={`Abschreibung ${year}`}
          value={formatCents(sum((a) => a.yearAmount))}
          context={sum((a) => a.dueAmount) > 0 ? `${formatCents(sum((a) => a.dueAmount))} noch offen` : 'vollständig gebucht'}
        />
      </StatRow>

      {candidates.length > 0 && (
        <Notice>
          <p>
            {candidates.length === 1
              ? 'Eine Buchung liegt auf einem Anlagekonto, ohne dass es dazu ein Anlagegut gibt.'
              : `${candidates.length} Buchungen liegen auf Anlagekonten, ohne dass es dazu ein Anlagegut gibt.`}
          </p>
          <ul className="mt-2 space-y-1">
            {candidates.slice(0, 4).map((candidate) => (
              <li key={`${candidate.entryId}-${candidate.account}`} className="flex items-center gap-2">
                <span className="code-num text-caption">{candidate.account}</span>
                <span className="truncate">{candidate.description}</span>
                <span className="num">{formatCents(candidate.amount)}</span>
                <Button
                  variant="quiet"
                  size="sm"
                  onClick={() =>
                    onCreate({
                      account: candidate.account,
                      name: candidate.description,
                      acquisitionCost: candidate.amount,
                      acquisitionDate: candidate.bookingDate,
                      acquisitionEntryId: candidate.entryId,
                      contactId: candidate.contactId,
                    })
                  }
                >
                  Erfassen
                </Button>
              </li>
            ))}
          </ul>
        </Notice>
      )}

      {assets.length === 0 ? (
        <EmptyState
          icon={<Building2 className="w-6 h-6" strokeWidth={1.5} />}
          title={`Noch keine ${CLASS_LABEL[assetClass]} erfasst`}
          description={
            assetClass === 'financial'
              ? 'Beteiligungen, Wertpapiere und Ausleihungen, die dauernd dem Geschäftsbetrieb dienen sollen.'
              : 'Was länger als ein Jahr genutzt wird und über der Wertgrenze liegt, gehört hierher.'
          }
          action={<Button variant="primary" onClick={() => onCreate({})}>Anlagegut erfassen</Button>}
        />
      ) : (
        <Table>
          <Thead sticky>
            <Tr>
              <Th className="w-32">Inventarnummer</Th>
              <Th>Bezeichnung</Th>
              <Th className="w-24">Konto</Th>
              <Th className="w-28">Anschaffung</Th>
              <Th numeric className="w-32">Anschaffungskosten</Th>
              <Th numeric className="w-32">Kumulierte AfA</Th>
              <Th numeric className="w-28">Buchwert</Th>
              <Th numeric className="w-28">{`AfA ${year}`}</Th>
              <Th className="w-32">Zustand</Th>
            </Tr>
          </Thead>
          <Tbody>
            {[...inStock, ...disposed].map((asset) => (
              <Tr key={asset.id} className="cursor-pointer" onClick={() => onOpen(asset.id)}>
                <Td code>{asset.inventoryNumber}</Td>
                <Td className="max-w-[20rem] truncate" title={asset.name}>
                  {asset.name}
                  {asset.identifier && <span className="text-ink-subtle"> · {asset.identifier}</span>}
                </Td>
                <Td code title={asset.accountName}>{asset.account}</Td>
                <Td className="text-ink-muted">{formatDate(asset.acquisitionDate)}</Td>
                <Td numeric>{formatCents(asset.cost)}</Td>
                <Td numeric>{formatCents(asset.accumulated)}</Td>
                <Td numeric>{formatCents(asset.bookValue)}</Td>
                <Td numeric>{asset.yearAmount === 0 ? '—' : formatCents(asset.yearAmount)}</Td>
                <Td>
                  <AssetStatusCell asset={asset} />
                </Td>
              </Tr>
            ))}
          </Tbody>
        </Table>
      )}
    </div>
  );
};

const AssetStatusCell: React.FC<{ asset: FixedAsset }> = ({ asset }) => {
  switch (asset.status) {
    case 'depreciate_due':
      return (
        <span title={asset.statusNote} className="inline-flex items-center gap-2">
          <StatusBadge status="offen" />
          <span className="text-caption text-ink-subtle num">{formatCents(asset.dueAmount)}</span>
        </span>
      );
    case 'unbooked':
      return <span title={asset.statusNote}><StatusBadge status="entwurf" /></span>;
    case 'disposed':
      return (
        <span className="text-ink-subtle text-caption">
          Abgegangen {formatDate(asset.disposalDate ?? '')}
        </span>
      );
    case 'fully_written':
      return <span className="text-ink-subtle text-caption">Abgeschrieben</span>;
    default:
      return <span className="text-ink-subtle text-caption">Im Bestand</span>;
  }
};

// -------------------------------------------------------------------------
// Abschreibungslauf
// -------------------------------------------------------------------------

const DepreciationTab: React.FC<{
  run: DepreciationRun | null;
  year: number;
  explanation: Explanation;
  onExplain: () => void;
  onBooked: () => Promise<void>;
}> = ({ run, year, explanation, onExplain, onBooked }) => {
  const [selected, setSelected] = useState<number[]>([]);
  const [bookingDate, setBookingDate] = useState(run?.bookingDate ?? `${year}-12-31`);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (run?.bookingDate) setBookingDate(run.bookingDate);
    setSelected(run?.due.filter((d) => d.due > 0).map((d) => d.assetId) ?? []);
  }, [run]);

  const due = run?.due ?? [];
  const bookable = due.filter((d) => d.due > 0);
  const total = bookable
    .filter((d) => selected.includes(d.assetId))
    .reduce((sum, d) => sum + d.due, 0);

  async function book() {
    setBusy(true);
    try {
      const result = await Api.bookDepreciationRun({
        fiscalYear: year,
        bookingDate,
        assetIds: selected,
      });
      toast.success(
        `${result.entries.length} Abschreibungsbuchungen über ${formatCents(result.total)} geschrieben.`,
      );
      await onBooked();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="space-y-6">
      <ExplainLine explanation={explanation} onMore={onExplain} />

      {run?.missingPriorYears && run.missingPriorYears.length > 0 && (
        <Notice>
          Für {run.missingPriorYears.join(', ')} fehlt noch Abschreibung — sie gehört in ihr eigenes
          Geschäftsjahr und wird hier nicht nachgeholt.
        </Notice>
      )}

      {bookable.length === 0 ? (
        <EmptyState
          title={`Für ${year} ist keine Abschreibung offen`}
          description={
            due.length > 0
              ? 'Einzelne Anlagegüter konnten nicht gerechnet werden — siehe die Hinweise unten.'
              : 'Alle planmäßigen Abschreibungen dieses Geschäftsjahres sind gebucht.'
          }
        />
      ) : (
        <>
          <div className="flex flex-wrap items-end justify-between gap-4">
            <Field
              label="Buchungsdatum"
              hint="Bilanzstichtag des Geschäftsjahres"
              className="w-48"
              help="Die AfA gehört in das Jahr, das sie betrifft. Ein Datum außerhalb wird abgelehnt."
            >
              <Input
                type="date"
                value={bookingDate}
                onChange={(e) => setBookingDate(e.target.value)}
              />
            </Field>
            <div className="flex items-center gap-4">
              <span className="text-body text-ink-muted">
                {selected.length} von {bookable.length} ausgewählt ·{' '}
                <span className="num text-ink">{formatCents(total)}</span>
              </span>
              <Button variant="primary" loading={busy} disabled={selected.length === 0} onClick={book}>
                Abschreibung buchen
              </Button>
            </div>
          </div>

          <Table>
            <Thead sticky>
              <Tr>
                <Th className="w-10" aria-label="Auswahl" />
                <Th className="w-32">Inventarnummer</Th>
                <Th>Bezeichnung</Th>
                <Th className="w-40">Buchungssatz</Th>
                <Th className="w-36">Methode</Th>
                <Th numeric className="w-28">Buchwert vorher</Th>
                <Th numeric className="w-28">Abschreibung</Th>
                <Th numeric className="w-28">Buchwert nachher</Th>
              </Tr>
            </Thead>
            <Tbody>
              {bookable.map((row) => (
                <Tr key={row.assetId}>
                  <Td>
                    <Checkbox
                      label=""
                      checked={selected.includes(row.assetId)}
                      onCheckedChange={(checked) =>
                        setSelected((prev) =>
                          checked ? [...prev, row.assetId] : prev.filter((id) => id !== row.assetId),
                        )
                      }
                    />
                  </Td>
                  <Td code>{row.inventoryNumber}</Td>
                  <Td className="max-w-[16rem] truncate" title={row.note}>
                    {row.name}
                  </Td>
                  <Td code>
                    {row.expenseAccount} an {row.account}
                  </Td>
                  <Td className="text-ink-muted">
                    {row.rateLabel}
                    {row.months < 12 && <span className="text-ink-subtle"> · {row.months}/12</span>}
                  </Td>
                  <Td numeric>{formatCents(row.bookValueBefore)}</Td>
                  <Td numeric>{formatCents(row.due)}</Td>
                  <Td numeric>{formatCents(row.bookValueAfter)}</Td>
                </Tr>
              ))}
              <Tr variant="sum">
                <Td colSpan={6}>Summe der ausgewählten Abschreibungen</Td>
                <Td numeric>{formatCents(total)}</Td>
                <Td />
              </Tr>
            </Tbody>
          </Table>
        </>
      )}

      {due.some((d) => d.due <= 0 && d.note) && (
        <Section title="Nicht rechenbare Anlagegüter" context="Diese Zeilen bleiben ungebucht">
          <ul className="space-y-2 text-body text-ink-muted">
            {due
              .filter((d) => d.due <= 0 && d.note)
              .map((d) => (
                <li key={d.assetId}>
                  <span className="code-num text-caption">{d.inventoryNumber}</span> {d.name}: {d.note}
                </li>
              ))}
          </ul>
        </Section>
      )}
    </div>
  );
};

// -------------------------------------------------------------------------
// Anlagenspiegel
// -------------------------------------------------------------------------

const SpiegelTab: React.FC<{
  spiegel: Anlagenspiegel | null;
  year: number;
  explanation: Explanation;
  onExplain: () => void;
}> = ({ spiegel, year, explanation, onExplain }) => {
  const rows = spiegel?.rows ?? [];
  // Die Spalte Zuschreibungen steht nur da, wo es welche gibt. Elf Spalten
  // brauchen jeden Millimeter, und eine Spalte aus lauter Nullen erklärt nichts.
  const showWriteUps = rows.some((row) => row.writeUpsYear !== 0);
  const columnCount = showWriteUps ? 11 : 10;
  const cell = 'px-2.5';

  const renderRow = (row: AnlagenspiegelRow, key: string, variant?: 'sum') => (
    <Tr key={key} variant={variant}>
      <Td code className={cell}>{variant ? '' : row.account}</Td>
      <Td className={`${cell} max-w-[14rem] truncate`} title={row.accountName}>
        {row.accountName}
      </Td>
      <Td numeric className={cell}>{formatCents(row.costOpening)}</Td>
      <Td numeric className={cell}>{formatCents(row.additions)}</Td>
      <Td numeric className={cell}>{formatCents(row.disposals)}</Td>
      <Td numeric className={cell}>{formatCents(row.costClosing)}</Td>
      <Td numeric className={cell}>{formatCents(row.depreciationYear)}</Td>
      {showWriteUps && <Td numeric className={cell}>{formatCents(row.writeUpsYear)}</Td>}
      <Td numeric className={cell}>{formatCents(row.depreciationClosing)}</Td>
      <Td numeric className={cell}>{formatCents(row.bookValueOpening)}</Td>
      <Td numeric className={cell}>{formatCents(row.bookValueClosing)}</Td>
    </Tr>
  );

  return (
    <div className="space-y-6">
      <ExplainLine explanation={explanation} onMore={onExplain} />

      {rows.length === 0 ? (
        <EmptyState
          title="Noch kein Anlagevermögen erfasst"
          description="Sobald das erste Anlagegut im Verzeichnis steht, füllt sich der Spiegel von selbst."
        />
      ) : (
        <Table density="kompakt">
          <Thead sticky>
            <Tr>
              <Th className={`${cell} w-16`}>Konto</Th>
              <Th className={cell}>Position</Th>
              <Th numeric className={cell}>AHK 01.01.</Th>
              <Th numeric className={cell}>Zugänge</Th>
              <Th numeric className={cell}>Abgänge</Th>
              <Th numeric className={cell}>AHK 31.12.</Th>
              <Th numeric className={cell}>AfA {year}</Th>
              {showWriteUps && (
                <Th numeric className={cell}>Zuschreibung</Th>
              )}
              <Th numeric className={cell}>Kum. AfA</Th>
              <Th numeric className={cell}>Buchwert {year - 1}</Th>
              <Th numeric className={cell}>Buchwert {year}</Th>
            </Tr>
          </Thead>
          <Tbody>
            {(['intangible', 'tangible', 'financial'] as AssetClass[]).map((assetClass) => {
              const classRows = rows.filter((r) => r.class === assetClass);
              if (classRows.length === 0) return null;
              const total = spiegel?.classTotals.find((t) => t.class === assetClass);
              return (
                <React.Fragment key={assetClass}>
                  <Tr>
                    <Td colSpan={columnCount} className="bg-sunken text-overline text-ink-subtle">
                      {CLASS_LABEL[assetClass]}
                    </Td>
                  </Tr>
                  {classRows.map((row) => renderRow(row, `${assetClass}-${row.account}`))}
                  {total && renderRow(total, `${assetClass}-total`, 'sum')}
                </React.Fragment>
              );
            })}
            {spiegel && renderRow(spiegel.totals, 'total', 'sum')}
          </Tbody>
        </Table>
      )}
    </div>
  );
};

// -------------------------------------------------------------------------
// Anlagegut erfassen und ändern
// -------------------------------------------------------------------------

const AssetFormDialog: React.FC<{
  draft: Partial<FixedAsset> | null;
  accounts: AssetAccountInfo[];
  rules: AssetRules | null;
  candidates: AcquisitionCandidate[];
  contacts: Contact[];
  year: number;
  onClose: () => void;
  onSaved: (asset: FixedAsset) => Promise<void>;
}> = ({ draft, accounts, rules, candidates, contacts, year, onClose, onSaved }) => {
  const [asset, setAsset] = useState<Partial<FixedAsset>>(draft ?? {});
  // Beträge stehen als Text im Formular und werden erst beim Speichern gelesen.
  // Ein Feld, das bei jedem Tastendruck neu formatiert, lässt sich nicht tippen.
  const [costText, setCostText] = useState('');
  const [selfUsable, setSelfUsable] = useState(true);
  const [advice, setAdvice] = useState<AcquisitionAdvice | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (draft) {
      setAsset(draft);
      setCostText(draft.acquisitionCost ? formatCentsPlain(draft.acquisitionCost) : '');
      setError(null);
      setAdvice(null);
    }
  }, [draft]);

  const assetClass = (asset.class ?? 'tangible') as AssetClass;
  const isNew = !asset.id;
  const catalog = accounts.filter((a) => a.class === assetClass);
  const selectedAccount = catalog.find((a) => a.number === asset.account);

  // Die Einordnung nach § 6 Abs. 2 und 2a EStG rechnet das Backend. Sie wird
  // angefragt, sobald Betrag und Datum stehen — und nur für Sachanlagen, weil
  // die Wertgrenzen nur dort greifen.
  useEffect(() => {
    const cost = parseCents(costText) ?? 0;
    if (assetClass !== 'tangible' || cost <= 0 || !asset.acquisitionDate) {
      setAdvice(null);
      return;
    }
    let cancelled = false;
    Api.classifyAcquisition(cost, asset.acquisitionDate, selfUsable)
      .then((result) => {
        if (!cancelled) setAdvice(result);
      })
      .catch(() => setAdvice(null));
    return () => {
      cancelled = true;
    };
  }, [assetClass, costText, asset.acquisitionDate, selfUsable]);

  function set(patch: Partial<FixedAsset>) {
    setAsset((prev) => ({ ...prev, ...patch }));
  }

  function pickAccount(number: string | null) {
    const entry = catalog.find((a) => a.number === number);
    set({
      account: number ?? '',
      depreciationAccount: entry?.depreciationAccount ?? '',
      usefulLifeMonths: entry?.defaultUsefulLifeMonths || asset.usefulLifeMonths || 0,
      method: entry && !entry.depreciable ? 'none' : asset.method ?? 'linear',
    });
  }

  const methods = (rules?.methods ?? []).filter((m) => m.classes.includes(assetClass));
  const activeMethod = methods.find((m) => m.method === asset.method);
  const needsUsefulLife = asset.method === 'linear' || asset.method === 'degressive';

  async function submit() {
    const cost = parseCents(costText);
    if (cost === null || cost <= 0) {
      setError('Die Anschaffungskosten fehlen oder sind nicht lesbar. Erwartet wird ein Betrag wie 1.234,56.');
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const payload: Partial<FixedAsset> = {
        ...asset,
        class: assetClass,
        acquisitionCost: cost,
        poolYear: asset.method === 'pool' ? asset.poolYear || year : 0,
      };
      const saved = await Api.saveFixedAsset(payload);
      await onSaved(saved);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog
      open={draft !== null}
      onOpenChange={(next) => !next && onClose()}
      title={isNew ? 'Anlagegut erfassen' : `${asset.inventoryNumber ?? ''} bearbeiten`}
      width="max-w-3xl"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Abbrechen
          </Button>
          <Button variant="primary" loading={busy} onClick={submit}>
            Speichern
          </Button>
        </>
      }
    >
      <div className="grid grid-cols-2 gap-4">
        <Field
          label="Anlagenklasse"
          help="Die drei Blöcke des Anlagevermögens nach § 266 Abs. 2 A HGB. Sie entscheiden über Konten und Bewertung."
        >
          <Select
            items={CLASS_TABS.map((c) => ({ value: c, label: CLASS_LABEL[c] }))}
            value={assetClass}
            onValueChange={(next) =>
              set({
                class: next as AssetClass,
                account: '',
                depreciationAccount: '',
                method: next === 'financial' ? 'none' : 'linear',
              })
            }
          />
        </Field>
        <Field label="Inventarnummer" hint={isNew ? 'wird beim Speichern vergeben' : 'nicht änderbar'}>
          <Input value={asset.inventoryNumber ?? ''} disabled />
        </Field>
      </div>

      <Field label="Bezeichnung" className="mt-4">
        <Input value={asset.name ?? ''} onChange={(e) => set({ name: e.target.value })} />
      </Field>

      <Field
        label="Anlagekonto"
        className="mt-4"
        hint={selectedAccount?.hint}
        help="Der kuratierte Auszug aus der Kontenklasse 0. Zu jedem Konto gehört das Aufwandskonto, auf dem seine Abschreibung landet."
      >
        <Combobox
          items={catalog.map((a) => ({
            value: a.number,
            label: `${a.number} ${a.name}`,
            meta: a.depreciable ? a.group : `${a.group} · nicht abnutzbar`,
          }))}
          value={asset.account ?? null}
          onValueChange={pickAccount}
          placeholder="Konto suchen"
          emptyText="Kein Konto passt zur Suche."
        />
      </Field>

      <div className="grid grid-cols-3 gap-4 mt-4">
        <Field label="Anschaffungsdatum" help="Die AfA läuft monatsgenau ab diesem Monat.">
          <Input
            type="date"
            value={asset.acquisitionDate ?? ''}
            onChange={(e) => set({ acquisitionDate: e.target.value })}
          />
        </Field>
        <Field
          label="Anschaffungskosten"
          hint="netto, ohne Vorsteuer"
          help="Anschaffungspreis zuzüglich Nebenkosten wie Fracht und Montage, abzüglich Minderungen (§ 255 Abs. 1 HGB)."
        >
          <Input
            align="right"
            inputMode="decimal"
            placeholder="0,00"
            value={costText}
            onChange={(e) => setCostText(e.target.value)}
          />
        </Field>
        <Field label="Zugangsbuchung" optional help="Verknüpft das Anlagegut mit der Buchung, die den Zugang erfasst hat.">
          <Select
            items={[
              { value: 0, label: 'Nicht verknüpft' },
              ...candidates.map((c) => ({
                value: c.entryId,
                label: `${c.entryNumber} · ${formatCents(c.amount)}`,
              })),
              ...(asset.acquisitionEntryId &&
              !candidates.some((c) => c.entryId === asset.acquisitionEntryId)
                ? [{ value: asset.acquisitionEntryId, label: 'Bereits verknüpft' }]
                : []),
            ]}
            value={asset.acquisitionEntryId ?? 0}
            onValueChange={(next) => set({ acquisitionEntryId: next === 0 ? undefined : Number(next) })}
          />
        </Field>
      </div>

      {assetClass === 'tangible' && (
        <div className="mt-4 space-y-3">
          <Checkbox
            checked={selfUsable}
            onCheckedChange={(checked) => setSelfUsable(Boolean(checked))}
            label="Das Wirtschaftsgut ist selbständig nutzbar"
            hint="Ein Bildschirm ist es ohne Rechner nicht — und damit kein GWG, egal wie günstig er war."
          />
          {advice && (
            <div className="rounded-control border border-accent-line bg-accent-soft px-4 py-3 text-body text-ink-muted">
              <p>{advice.reason}</p>
              {advice.poolNote && <p className="mt-2">{advice.poolNote}</p>}
              <p className="mt-2 text-caption text-ink-subtle">
                Vorschlag:{' '}
                {advice.recommended === 'immediate'
                  ? 'Sofortabzug'
                  : advice.recommended === 'pool'
                    ? 'Sammelposten'
                    : 'Aktivieren und abschreiben'}
              </p>
            </div>
          )}
        </div>
      )}

      <div className="grid grid-cols-3 gap-4 mt-4">
        <Field label="Abschreibungsmethode" hint={activeMethod?.hint}>
          <Select
            items={methods.map((m) => ({ value: m.method, label: m.label }))}
            value={asset.method ?? 'linear'}
            onValueChange={(next) => set({ method: next as FixedAsset['method'] })}
          />
        </Field>
        {needsUsefulLife && (
          <Field
            label="Nutzungsdauer in Jahren"
            hint={
              asset.usefulLifeMonths
                ? `${asset.usefulLifeMonths} Monate${
                    selectedAccount?.usefulLifeSource ? ` · ${selectedAccount.usefulLifeSource}` : ''
                  }`
                : selectedAccount?.usefulLifeSource
            }
            help="Kommt aus den AfA-Tabellen des BMF. Die binden die Finanzverwaltung, nicht dich — eine begründete abweichende Nutzungsdauer ist zulässig."
          >
            <Input
              type="number"
              min={0}
              step={0.5}
              align="right"
              value={asset.usefulLifeMonths ? monthsToYears(asset.usefulLifeMonths) : ''}
              onChange={(e) => set({ usefulLifeMonths: Math.round(Number(e.target.value) * 12) })}
            />
          </Field>
        )}
        {asset.method === 'pool' && (
          <Field label="Wirtschaftsjahr des Sammelpostens" hint="ein Pool je Jahr">
            <Input
              type="number"
              align="right"
              value={asset.poolYear || year}
              onChange={(e) => set({ poolYear: Number(e.target.value) })}
            />
          </Field>
        )}
        <Field label="Abschreibungskonto" hint="folgt aus dem Anlagekonto" optional>
          <Input value={asset.depreciationAccount ?? ''} disabled />
        </Field>
      </div>

      {assetClass === 'financial' && (
        <div className="mt-4 grid grid-cols-3 gap-4">
          <Field label="Kennung" optional hint="ISIN, WKN oder Registernummer">
            <Input
              className="code-num"
              value={asset.identifier ?? ''}
              onChange={(e) => set({ identifier: e.target.value })}
            />
          </Field>
          <Field
            label="Beteiligungsquote"
            optional
            hint="in Prozent"
            help="Ab einem Fünftel des Kapitals vermutet § 271 Abs. 1 Satz 3 HGB eine Beteiligung."
          >
            <Input
              type="number"
              min={0}
              max={100}
              align="right"
              value={(asset.holdingPermille ?? 0) / 10}
              onChange={(e) => set({ holdingPermille: Math.round(Number(e.target.value) * 10) })}
            />
          </Field>
          <div className="flex items-end pb-2">
            <Checkbox
              checked={asset.taxPrivileged ?? false}
              onCheckedChange={(checked) => set({ taxPrivileged: Boolean(checked) })}
              label="Anteil an einer Kapitalgesellschaft"
              hint="Gewinn und Verlust laufen dann über eigene Konten — § 8b Abs. 2 KStG bzw. § 3 Nr. 40 EStG."
            />
          </div>
        </div>
      )}

      <div className="grid grid-cols-2 gap-4 mt-4">
        <Field label="Lieferant" optional>
          <Select
            items={[
              { value: 0, label: 'Nicht zugeordnet' },
              ...contacts.map((c) => ({ value: c.id, label: c.name })),
            ]}
            value={asset.contactId ?? 0}
            onValueChange={(next) => set({ contactId: next === 0 ? undefined : Number(next) })}
          />
        </Field>
        <Field label="Beschreibung" optional>
          <Input
            value={asset.description ?? ''}
            onChange={(e) => set({ description: e.target.value })}
          />
        </Field>
      </div>

      <div className="mt-5">
        <FormHint
          label="Erklärung zum Zugang"
          line="Der Zugang selbst wird über den Beleg gebucht, nicht hier."
        >
          Die Buchung entsteht mit Vorsteuer, Lieferant und Belegverweis im Belegflow. Das
          Verzeichnis führt das Anlagegut daneben fort: es kennt die Bemessungsgrundlage, den Plan
          und die Bewegungen über alle Jahre. Beides zusammen ergibt den Anlagenspiegel.
        </FormHint>
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

// -------------------------------------------------------------------------
// Anlagegut im Detail: Plan, Bewegungen und die Vorgänge daran
// -------------------------------------------------------------------------

type DetailAction = 'impairment' | 'writeUp' | 'cost' | 'disposal' | null;

const AssetDetailDialog: React.FC<{
  assetId: number | null;
  contacts: Contact[];
  paymentAccounts: Account[];
  onClose: () => void;
  onEdit: (asset: FixedAsset) => void;
  onChanged: () => Promise<void>;
}> = ({ assetId, contacts, paymentAccounts, onClose, onEdit, onChanged }) => {
  const [detail, setDetail] = useState<AssetDetail | null>(null);
  const [action, setAction] = useState<DetailAction>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (assetId === null) {
      setDetail(null);
      setAction(null);
      return;
    }
    void load(assetId);
  }, [assetId]);

  async function load(id: number) {
    setLoading(true);
    try {
      setDetail(await Api.getFixedAsset(id));
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }

  const asset = detail?.asset;

  async function afterBooking(message: string) {
    toast.success(message);
    setAction(null);
    if (assetId !== null) await load(assetId);
    await onChanged();
  }

  return (
    <Dialog
      open={assetId !== null}
      onOpenChange={(next) => !next && onClose()}
      title={asset ? `${asset.inventoryNumber} · ${asset.name}` : 'Anlagegut'}
      width="max-w-4xl"
      footer={
        action === null && asset ? (
          <>
            <Button variant="quiet" onClick={() => setAction('cost')}>
              Nachträgliche Kosten
            </Button>
            {!asset.disposalDate && (
              <>
                <Button variant="secondary" onClick={() => setAction('impairment')}>
                  Außerplanmäßig abschreiben
                </Button>
                <Button variant="secondary" onClick={() => setAction('writeUp')}>
                  Zuschreiben
                </Button>
                <Button variant="secondary" onClick={() => onEdit(asset)}>
                  Bearbeiten
                </Button>
                <Button variant="primary" onClick={() => setAction('disposal')}>
                  Abgang buchen
                </Button>
              </>
            )}
          </>
        ) : (
          <Button
            variant="secondary"
            icon={<ArrowLeft className="w-4 h-4" strokeWidth={1.5} />}
            onClick={() => setAction(null)}
          >
            Zurück
          </Button>
        )
      }
    >
      {loading || !detail || !asset ? (
        <SkeletonRows rows={5} />
      ) : action === 'impairment' ? (
        <ImpairmentForm asset={asset} onDone={afterBooking} />
      ) : action === 'writeUp' ? (
        <WriteUpForm asset={asset} onDone={afterBooking} />
      ) : action === 'cost' ? (
        <CostAdjustmentForm asset={asset} onDone={afterBooking} />
      ) : action === 'disposal' ? (
        <DisposalForm
          asset={asset}
          contacts={contacts}
          paymentAccounts={paymentAccounts}
          onDone={afterBooking}
        />
      ) : (
        <AssetOverview detail={detail} />
      )}
    </Dialog>
  );
};

const AssetOverview: React.FC<{ detail: AssetDetail }> = ({ detail }) => {
  const { asset, schedule, movements, notes } = detail;

  return (
    <div className="space-y-6">
      <StatRow>
        <Stat label="Anschaffungskosten" value={formatCents(asset.cost)} context={formatDate(asset.acquisitionDate)} />
        <Stat label="Kumulierte Abschreibungen" value={formatCents(asset.accumulated)} />
        <Stat label="Buchwert" value={formatCents(asset.bookValue)} />
        <Stat
          label="Konto"
          value={<span className="code-num text-body">{asset.account}</span>}
          context={asset.accountName}
        />
      </StatRow>

      {notes.length > 0 && (
        <div className="rounded-card border border-line bg-surface px-5 py-4 space-y-2">
          {notes.map((note, index) => (
            <p key={index} className="text-body text-ink-muted">
              {note}
            </p>
          ))}
        </div>
      )}

      {schedule.length > 0 && (
        <Section title="Abschreibungsplan" divider={false} context="Gerechnet, nicht gespeichert — er folgt den Bewegungen">
          <Table density="kompakt">
            <Thead>
              <Tr>
                <Th className="w-20">Jahr</Th>
                <Th className="w-20">Monate</Th>
                <Th>Satz</Th>
                <Th numeric>Buchwert Anfang</Th>
                <Th numeric>Abschreibung</Th>
                <Th numeric>Gebucht</Th>
                <Th numeric>Buchwert Ende</Th>
                <Th className="w-24">Zustand</Th>
              </Tr>
            </Thead>
            <Tbody>
              {schedule.map((row) => (
                <Tr key={row.fiscalYear}>
                  <Td className="num">{row.fiscalYear}</Td>
                  <Td className="num">{row.months}</Td>
                  <Td className="text-ink-muted" title={row.note}>
                    {row.rateLabel}
                  </Td>
                  <Td numeric>{formatCents(row.openingBookValue)}</Td>
                  <Td numeric>{formatCents(row.amount)}</Td>
                  <Td numeric>{formatCents(row.booked)}</Td>
                  <Td numeric>{formatCents(row.closingBookValue)}</Td>
                  <Td className="text-caption text-ink-subtle">{row.status}</Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        </Section>
      )}

      <Section title="Bewegungen" context="Jede Wertänderung mit ihrer Buchung">
        <Table density="kompakt">
          <Thead>
            <Tr>
              <Th className="w-28">Datum</Th>
              <Th>Vorgang</Th>
              <Th numeric>Anschaffungskosten</Th>
              <Th numeric>Abschreibungen</Th>
              <Th className="w-32">Buchung</Th>
            </Tr>
          </Thead>
          <Tbody>
            {movements.map((movement) => (
              <Tr key={movement.id}>
                <Td className="text-ink-muted">{formatDate(movement.date)}</Td>
                <Td className="max-w-[18rem] truncate" title={movement.note}>
                  {MOVEMENT_LABEL[movement.kind] ?? movement.kind}
                </Td>
                <Td numeric>{movement.costAmount === 0 ? '—' : formatCents(movement.costAmount)}</Td>
                <Td numeric>
                  {movement.depreciationAmount === 0 ? '—' : formatCents(movement.depreciationAmount)}
                </Td>
                <Td code>{movement.entryNumber ?? '—'}</Td>
              </Tr>
            ))}
          </Tbody>
        </Table>
      </Section>
    </div>
  );
};

const MOVEMENT_LABEL: Record<string, string> = {
  acquisition: 'Zugang',
  subsequent_cost: 'Nachträgliche Anschaffungskosten',
  cost_reduction: 'Anschaffungskostenminderung',
  depreciation: 'Planmäßige Abschreibung',
  impairment: 'Außerplanmäßige Abschreibung',
  write_up: 'Zuschreibung',
  disposal: 'Abgang',
};

// -------------------------------------------------------------------------
// Vorgänge am Anlagegut
// -------------------------------------------------------------------------

const FormError: React.FC<{ message: string | null }> = ({ message }) =>
  message ? (
    <div className="flex items-start gap-2.5 rounded-control border border-negative-line bg-negative-soft px-4 py-3">
      <AlertCircle className="w-4 h-4 mt-0.5 shrink-0 text-negative" strokeWidth={1.5} />
      <p className="text-body text-negative-text">{message}</p>
    </div>
  ) : null;

const ImpairmentForm: React.FC<{
  asset: FixedAsset;
  onDone: (message: string) => Promise<void>;
}> = ({ asset, onDone }) => {
  const [date, setDate] = useState(new Date().toISOString().slice(0, 10));
  const [amount, setAmount] = useState('');
  const [permanent, setPermanent] = useState(true);
  const [reason, setReason] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const isFinancial = asset.class === 'financial';

  async function submit() {
    setBusy(true);
    setError(null);
    try {
      const value = parseCents(amount);
      if (value === null || value <= 0) {
        setError('Der Betrag fehlt oder ist nicht lesbar.');
        return;
      }
      await Api.bookAssetImpairment({ assetId: asset.id, date, amount: value, permanent, reason });
      await onDone('Außerplanmäßige Abschreibung gebucht.');
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="space-y-4">
      <FormHint
        label="Erklärung zur außerplanmäßigen Abschreibung"
        line="Ein Ermessensvorgang: Buchfink kann ihn erfassen, aber nicht auslösen."
      >
        Der Grund gehört deshalb zwingend an die Buchung — ohne ihn kann ihn später niemand mehr
        nachvollziehen.{' '}
        {isFinancial
          ? 'Bei Finanzanlagen darf auch bei einer nur vorübergehenden Wertminderung abgeschrieben werden (§ 253 Abs. 3 Satz 6 HGB); das Konto unterscheidet die beiden Fälle.'
          : 'Zulässig ist sie nur bei voraussichtlich dauernder Wertminderung (§ 253 Abs. 3 Satz 5 HGB); die Ausnahme für die nicht dauernde gilt allein für Finanzanlagen.'}
      </FormHint>

      <div className="grid grid-cols-3 gap-4">
        <Field label="Datum">
          <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
        </Field>
        <Field label="Betrag" hint={`Buchwert ${formatCents(asset.bookValue)}`}>
          <Input
            align="right"
            inputMode="decimal"
            placeholder="0,00"
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
          />
        </Field>
        <Field label="Art der Wertminderung">
          <RadioGroup
            options={[
              { value: 'permanent', label: 'Voraussichtlich dauernd' },
              {
                value: 'temporary',
                label: 'Nicht dauernd',
                disabled: !isFinancial,
                hint: isFinancial ? undefined : 'nur bei Finanzanlagen zulässig',
              },
            ]}
            value={permanent ? 'permanent' : 'temporary'}
            onValueChange={(next) => setPermanent(next === 'permanent')}
          />
        </Field>
      </div>

      <Field label="Grund" hint="wird mit der Buchung festgehalten">
        <Textarea rows={2} value={reason} onChange={(e) => setReason(e.target.value)} />
      </Field>

      <FormError message={error} />

      <div className="flex justify-end">
        <Button variant="primary" loading={busy} onClick={submit}>
          Abschreibung buchen
        </Button>
      </div>
    </div>
  );
};

const WriteUpForm: React.FC<{
  asset: FixedAsset;
  onDone: (message: string) => Promise<void>;
}> = ({ asset, onDone }) => {
  const [date, setDate] = useState(new Date().toISOString().slice(0, 10));
  const [amount, setAmount] = useState('');
  const [reason, setReason] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function submit() {
    setBusy(true);
    setError(null);
    try {
      const value = parseCents(amount);
      if (value === null || value <= 0) {
        setError('Der Betrag fehlt oder ist nicht lesbar.');
        return;
      }
      await Api.bookAssetWriteUp({ assetId: asset.id, date, amount: value, reason });
      await onDone('Zuschreibung gebucht.');
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="space-y-4">
      <FormHint
        label="Erklärung zur Zuschreibung"
        line="Zuschreiben ist ein Gebot, kein Wahlrecht (§ 253 Abs. 5 Satz 1 HGB)."
      >
        Fällt der Grund für eine frühere außerplanmäßige Abschreibung weg, ist zuzuschreiben. Die
        Obergrenze sind die fortgeführten Anschaffungskosten: der Buchwert, den das Anlagegut ohne
        die außerplanmäßige Abschreibung heute hätte. Buchfink rechnet diese Grenze und weist einen
        höheren Betrag ab.
      </FormHint>

      <div className="grid grid-cols-2 gap-4">
        <Field label="Datum">
          <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
        </Field>
        <Field label="Betrag" hint={`Buchwert ${formatCents(asset.bookValue)}`}>
          <Input
            align="right"
            inputMode="decimal"
            placeholder="0,00"
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
          />
        </Field>
      </div>

      <Field label="Grund" optional>
        <Textarea rows={2} value={reason} onChange={(e) => setReason(e.target.value)} />
      </Field>

      <FormError message={error} />

      <div className="flex justify-end">
        <Button variant="primary" loading={busy} onClick={submit}>
          Zuschreibung buchen
        </Button>
      </div>
    </div>
  );
};

const CostAdjustmentForm: React.FC<{
  asset: FixedAsset;
  onDone: (message: string) => Promise<void>;
}> = ({ asset, onDone }) => {
  const [date, setDate] = useState(new Date().toISOString().slice(0, 10));
  const [amount, setAmount] = useState('');
  const [reduction, setReduction] = useState(false);
  const [note, setNote] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function submit() {
    setBusy(true);
    setError(null);
    try {
      const value = parseCents(amount);
      if (value === null || value <= 0) {
        setError('Der Betrag fehlt oder ist nicht lesbar.');
        return;
      }
      await Api.recordAssetCostAdjustment({ assetId: asset.id, date, amount: value, reduction, note });
      await onDone(reduction ? 'Minderung erfasst.' : 'Nachträgliche Anschaffungskosten erfasst.');
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="space-y-4">
      <FormHint
        label="Erklärung zu nachträglichen Anschaffungskosten"
        line="Skonto auf eine Anlage mindert die Anschaffungskosten, nicht den Aufwand."
      >
        Fracht, Montage und Überführung gehören zu den Anschaffungskosten, Finanzierungskosten nicht
        (§ 255 Abs. 1 HGB). Gebucht wird hier nichts: der Betrag steht bereits über den Beleg oder
        die Zahlung auf dem Anlagekonto. Was hier entsteht, ist die Fortschreibung der Kartei — und
        damit ein neuer Plan für die Folgejahre.
      </FormHint>

      <div className="grid grid-cols-3 gap-4">
        <Field label="Datum">
          <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
        </Field>
        <Field label="Betrag" hint={`Bisher ${formatCents(asset.cost)}`}>
          <Input
            align="right"
            inputMode="decimal"
            placeholder="0,00"
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
          />
        </Field>
        <Field label="Art">
          <RadioGroup
            options={[
              { value: 'addition', label: 'Nachträgliche Kosten' },
              { value: 'reduction', label: 'Minderung (z. B. Skonto)' },
            ]}
            value={reduction ? 'reduction' : 'addition'}
            onValueChange={(next) => setReduction(next === 'reduction')}
          />
        </Field>
      </div>

      <Field label="Notiz" optional>
        <Input value={note} onChange={(e) => setNote(e.target.value)} />
      </Field>

      <FormError message={error} />

      <div className="flex justify-end">
        <Button variant="primary" loading={busy} onClick={submit}>
          Erfassen
        </Button>
      </div>
    </div>
  );
};

const DisposalForm: React.FC<{
  asset: FixedAsset;
  contacts: Contact[];
  paymentAccounts: Account[];
  onDone: (message: string) => Promise<void>;
}> = ({ asset, contacts, paymentAccounts, onDone }) => {
  const customers = contacts.filter((c) => c.type === 'customer');
  const [request, setRequest] = useState<DisposalRequest>({
    assetId: asset.id,
    date: new Date().toISOString().slice(0, 10),
    kind: 'sale',
    proceeds: 0,
    taxTreatment: asset.class === 'financial' ? 'exempt' : 'domestic',
    taxRate: 1900,
    settlement: 'paid',
    paymentAccount: paymentAccounts[0]?.number ?? '1800',
    contactId: customers[0]?.id,
    note: '',
  });
  const [proceedsText, setProceedsText] = useState('');
  const [preview, setPreview] = useState<DisposalPreview | null>(null);
  const [previewError, setPreviewError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    Api.previewAssetDisposal(request)
      .then((result) => {
        if (cancelled) return;
        setPreview(result);
        setPreviewError(null);
      })
      .catch((e) => {
        if (cancelled) return;
        setPreview(null);
        setPreviewError(e instanceof Error ? e.message : String(e));
      });
    return () => {
      cancelled = true;
    };
  }, [request]);

  function set(patch: Partial<DisposalRequest>) {
    setRequest((prev) => ({ ...prev, ...patch }));
  }

  async function submit() {
    setBusy(true);
    setError(null);
    try {
      const result = await Api.disposeFixedAsset(request);
      await onDone(result.message);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="space-y-4">
      <FormHint
        label="Erklärung zum Abgang"
        line="Der SKR04 wählt das Erlöskonto nach dem Ergebnis, nicht nach dem Vorgang."
      >
        Beim Abgang geschehen drei Dinge gleichzeitig: die Abschreibung wird bis zum Abgangsmonat
        nachgeholt, der Restbuchwert verschwindet, und ein Erlös entsteht. Die Differenz ist der
        Buchgewinn oder -verlust — und derselbe Verkauf steht damit einmal unter den Erträgen und
        einmal unter den Aufwendungen. Buchfink rechnet das Ergebnis deshalb zuerst und zeigt unten,
        welche Konten daraus folgen.
      </FormHint>

      <div className="grid grid-cols-3 gap-4">
        <Field label="Abgangsdatum" help="Im Abgangsmonat wird noch abgeschrieben, danach nicht mehr.">
          <Input type="date" value={request.date} onChange={(e) => set({ date: e.target.value })} />
        </Field>
        <Field label="Art des Abgangs">
          <Select
            items={[
              { value: 'sale', label: 'Verkauf' },
              { value: 'scrapped', label: 'Verschrottung ohne Erlös' },
            ]}
            value={request.kind}
            onValueChange={(next) => {
              if (next === 'scrapped') setProceedsText('');
              set({ kind: next as DisposalKind, proceeds: next === 'scrapped' ? 0 : request.proceeds });
            }}
          />
        </Field>
        <Field label="Erlös" hint="netto" disabled={request.kind === 'scrapped'}>
          <Input
            align="right"
            inputMode="decimal"
            placeholder="0,00"
            disabled={request.kind === 'scrapped'}
            value={proceedsText}
            onChange={(e) => {
              setProceedsText(e.target.value);
              set({ proceeds: parseCents(e.target.value) ?? 0 });
            }}
          />
        </Field>
      </div>

      {request.kind === 'sale' && (
        <div className="grid grid-cols-3 gap-4">
          <Field label="Steuerfall">
            <Select
              items={[
                { value: 'domestic', label: 'Inland, steuerpflichtig' },
                { value: 'intra_community_supply', label: 'Innergemeinschaftliche Lieferung' },
                { value: 'export', label: 'Ausfuhr in ein Drittland' },
                { value: 'exempt', label: 'Steuerfrei' },
              ]}
              value={request.taxTreatment ?? 'domestic'}
              onValueChange={(next) => set({ taxTreatment: next as TaxTreatment })}
            />
          </Field>
          <Field label="Steuersatz" disabled={request.taxTreatment !== 'domestic'}>
            <Select
              items={[
                { value: 1900, label: '19 %' },
                { value: 700, label: '7 %' },
              ]}
              value={request.taxRate ?? 1900}
              disabled={request.taxTreatment !== 'domestic'}
              onValueChange={(next) => set({ taxRate: Number(next) })}
            />
          </Field>
          <Field label="Zahlung">
            <Select
              items={[
                { value: 'paid', label: 'Sofort bezahlt' },
                { value: 'open', label: 'Offener Posten beim Käufer' },
              ]}
              value={request.settlement}
              onValueChange={(next) => set({ settlement: next as Settlement })}
            />
          </Field>
        </div>
      )}

      {request.kind === 'sale' && (
        <div className="grid grid-cols-2 gap-4">
          {request.settlement === 'paid' ? (
            <Field label="Zahlungsmittel">
              <Select
                items={paymentAccounts.map((a) => ({
                  value: a.number,
                  label: `${a.number} ${a.name}`,
                }))}
                value={request.paymentAccount ?? ''}
                onValueChange={(next) => set({ paymentAccount: String(next) })}
              />
            </Field>
          ) : (
            <Field label="Käufer" hint="die Forderung steht auf seinem Personenkonto">
              <Select
                items={customers.map((c) => ({ value: c.id, label: c.name }))}
                value={request.contactId ?? 0}
                onValueChange={(next) => set({ contactId: Number(next) })}
              />
            </Field>
          )}
          <Field label="Notiz" optional>
            <Input value={request.note ?? ''} onChange={(e) => set({ note: e.target.value })} />
          </Field>
        </div>
      )}

      {previewError && <Notice tone="negative">{previewError}</Notice>}

      {preview && (
        <Section title="Was gebucht wird" divider={false}>
          <StatRow className="mb-4">
            <Stat label="Nachgeholte AfA" value={formatCents(preview.catchUpAmount)} />
            <Stat label="Restbuchwert" value={formatCents(preview.bookValue)} />
            <Stat
              label={preview.result >= 0 ? 'Buchgewinn' : 'Buchverlust'}
              value={formatCents(Math.abs(preview.result))}
              tone={preview.result >= 0 ? 'positive' : 'negative'}
            />
            <Stat label="Zahlbetrag" value={formatCents(preview.gross)} context={`darin ${formatCents(preview.tax)} USt`} />
          </StatRow>

          <p className="text-caption text-ink-subtle mb-4">{preview.accounts.explanation}</p>

          {preview.warnings?.map((warning, index) => (
            <div key={index} className="mb-3">
              <Notice>{warning}</Notice>
            </div>
          ))}

          {preview.catchUpLines && preview.catchUpLines.length > 0 && (
            <LinesTable title="Abschreibung bis zum Abgangsmonat" lines={preview.catchUpLines} />
          )}
          {preview.lines.length > 0 && <LinesTable title="Abgangsbuchung" lines={preview.lines} />}
        </Section>
      )}

      <FormError message={error} />

      <div className="flex justify-end">
        <Button variant="primary" loading={busy} disabled={Boolean(previewError)} onClick={submit}>
          Abgang buchen
        </Button>
      </div>
    </div>
  );
};

const LinesTable: React.FC<{ title: string; lines: JournalLine[] }> = ({ title, lines }) => (
  <div className="mt-4">
    <div className="text-overline text-ink-subtle mb-2">{title}</div>
    <Table density="kompakt">
      <Thead>
        <Tr>
          <Th className="w-20">Seite</Th>
          <Th className="w-24">Konto</Th>
          <Th>Bezeichnung</Th>
          <Th numeric className="w-32">Betrag</Th>
        </Tr>
      </Thead>
      <Tbody>
        {lines.map((line, index) => (
          <Tr key={index}>
            <Td className="text-ink-muted">{line.side === 'S' ? 'Soll' : 'Haben'}</Td>
            <Td code>{line.account}</Td>
            <Td className="max-w-[20rem] truncate">{line.accountName ?? line.text ?? ''}</Td>
            <Td numeric>{formatCents(line.amount)}</Td>
          </Tr>
        ))}
      </Tbody>
    </Table>
  </div>
);

// -------------------------------------------------------------------------
// Kleine Helfer
// -------------------------------------------------------------------------

function monthsToYears(months: number | undefined): number {
  if (!months) return 0;
  return Math.round((months / 12) * 10) / 10;
}
