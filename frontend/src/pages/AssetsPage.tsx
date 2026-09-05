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
  AssetScheduleYear,
  AcquisitionOption,
  Cents,
  DepreciationMethod,
  Contact,
  DepreciationRun,
  DisposalKind,
  DisposalPreview,
  DisposalRequest,
  FixedAsset,
  AssetDocument,
  AssetDocumentKind,
  AssetDocumentKindInfo,
  CurrencyValuation,
  FundClass,
  InvestmentRules,
  InvestmentTaxNote,
  JournalLine,
  Vorabpauschale,
  Settlement,
  TaxTreatment,
} from '../types';
import { RATE_SCALE, UNIT_SCALE } from '../types';
import { Api } from '../services/api';
import { useWriteLock } from '../components/WriteLock';
import { formatCents, formatCentsPlain, formatDate, formatUnits, parseCents } from '../utils/formatters';
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
            Eine Sonderabschreibung nach § 7g Abs. 5 EStG erscheint in einer eigenen Spalte und wird
            mit derselben Buchung erfasst, aber auf einem eigenen Aufwandskonto: sie tritt{' '}
            <em>neben</em> die Absetzung für Abnutzung und ersetzt sie nicht. Nach dem
            Begünstigungszeitraum verteilt § 7a Abs. 9 EStG den Restwert auf die Restnutzungsdauer.
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
  const writeLock = useWriteLock();
  const [tab, setTab] = useState<Tab>('tangible');
  const [loading, setLoading] = useState(true);
  const [assets, setAssets] = useState<FixedAsset[]>([]);
  const [rules, setRules] = useState<AssetRules | null>(null);
  const [investment, setInvestment] = useState<InvestmentRules | null>(null);
  const [documentKinds, setDocumentKinds] = useState<AssetDocumentKindInfo[]>([]);
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
      const [
        list,
        ruleSet,
        runData,
        spiegelData,
        candidateList,
        accountList,
        contactList,
        payment,
        investmentRules,
        kinds,
      ] = await Promise.all([
        Api.getFixedAssets(''),
        Api.getAssetRules(),
        Api.getDepreciationRun(),
        Api.getAnlagenspiegel(),
        Api.getAssetAcquisitionCandidates(),
        Api.getAssetAccounts(''),
        Api.getContacts(),
        Api.getPaymentAccounts(),
        Api.getInvestmentRules(),
        Api.getAssetDocumentKinds(),
      ]);
      setAssets(list ?? []);
      setRules(ruleSet);
      setRun(runData);
      setSpiegel(spiegelData);
      setCandidates(candidateList ?? []);
      setAccounts(accountList ?? []);
      setContacts(contactList ?? []);
      setPaymentAccounts(payment ?? []);
      setInvestment(investmentRules);
      setDocumentKinds(kinds ?? []);
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
            disabled={writeLock.locked}
            title={writeLock.hint}
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
        investment={investment}
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
        accounts={accounts}
        documentKinds={documentKinds}
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
  const writeLock = useWriteLock();
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
          context={
            sum((a) => a.dueAmount + a.specialDue) > 0
              ? `${formatCents(sum((a) => a.dueAmount + a.specialDue))} noch offen`
              : 'vollständig gebucht'
          }
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
          action={
            <Button
              variant="primary"
              disabled={writeLock.locked}
              title={writeLock.hint}
              onClick={() => onCreate({})}
            >
              Anlagegut erfassen
            </Button>
          }
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
          <span className="text-caption text-ink-subtle num">
            {formatCents(asset.dueAmount + asset.specialDue)}
          </span>
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
  const writeLock = useWriteLock();
  const [selected, setSelected] = useState<number[]>([]);
  const [bookingDate, setBookingDate] = useState(run?.bookingDate ?? `${year}-12-31`);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (run?.bookingDate) setBookingDate(run.bookingDate);
    setSelected(run?.due.filter((d) => d.due > 0).map((d) => d.assetId) ?? []);
  }, [run]);

  const due = run?.due ?? [];
  const bookable = due.filter((d) => d.due > 0 || d.specialDue > 0);
  const total = bookable
    .filter((d) => selected.includes(d.assetId))
    .reduce((sum, d) => sum + d.due + d.specialDue, 0);
  // Die Sonderabschreibung bekommt nur dann eine Spalte, wenn eine läuft.
  const showSpecial = bookable.some((d) => d.specialDue > 0);

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
              <Button
                variant="primary"
                loading={busy}
                disabled={selected.length === 0 || writeLock.locked}
                title={writeLock.hint}
                onClick={book}
              >
                Abschreibung buchen
              </Button>
            </div>
          </div>

          <Table className={showSpecial ? '[&_td]:px-2.5 [&_th]:px-2.5' : undefined}>
            <Thead sticky>
              <Tr>
                <Th className="w-10" aria-label="Auswahl" />
                <Th className="w-32">Inventarnummer</Th>
                <Th>Bezeichnung</Th>
                <Th className="w-40">Buchungssatz</Th>
                <Th className={showSpecial ? 'w-24' : 'w-36'}>Methode</Th>
                <Th numeric className="w-28">Buchwert vorher</Th>
                <Th numeric className="w-28">Abschreibung</Th>
                {showSpecial && (
                  <Th numeric className="w-28">
                    Sonderabschreibung
                  </Th>
                )}
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
                    {row.expenseAccount}
                    {row.specialDue > 0 && row.specialAccount ? ` + ${row.specialAccount}` : ''} an{' '}
                    {row.account}
                  </Td>
                  <Td className="text-ink-muted">
                    {row.rateLabel}
                    {row.months < 12 && <span className="text-ink-subtle"> · {row.months}/12</span>}
                  </Td>
                  <Td numeric>{formatCents(row.bookValueBefore)}</Td>
                  <Td numeric>{formatCents(row.due)}</Td>
                  {showSpecial && (
                    <Td numeric>{row.specialDue === 0 ? '—' : formatCents(row.specialDue)}</Td>
                  )}
                  <Td numeric>{formatCents(row.bookValueAfter)}</Td>
                </Tr>
              ))}
              <Tr variant="sum">
                <Td colSpan={showSpecial ? 7 : 6}>Summe der ausgewählten Abschreibungen</Td>
                <Td numeric>{formatCents(total)}</Td>
                <Td />
              </Tr>
            </Tbody>
          </Table>
        </>
      )}

      {due.some((d) => d.due <= 0 && d.specialDue <= 0 && d.note) && (
        <Section title="Nicht rechenbare Anlagegüter" context="Diese Zeilen bleiben ungebucht">
          <ul className="space-y-2 text-body text-ink-muted">
            {due
              .filter((d) => d.due <= 0 && d.specialDue <= 0 && d.note)
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
  const showTransfers = rows.some((row) => row.transfers !== 0);
  const columnCount = 10 + (showWriteUps ? 1 : 0) + (showTransfers ? 1 : 0);
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
      {showTransfers && <Td numeric className={cell}>{formatCents(row.transfers)}</Td>}
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
              {showTransfers && (
                <Th numeric className={cell}>Umbuchungen</Th>
              )}
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

const ADVICE_LABEL: Record<AcquisitionOption, string> = {
  immediate: 'Sofortabzug',
  pool: 'Sammelposten',
  activate: 'Aktivieren und abschreiben',
};

const AssetFormDialog: React.FC<{
  draft: Partial<FixedAsset> | null;
  accounts: AssetAccountInfo[];
  rules: AssetRules | null;
  investment: InvestmentRules | null;
  candidates: AcquisitionCandidate[];
  contacts: Contact[];
  year: number;
  onClose: () => void;
  onSaved: (asset: FixedAsset) => Promise<void>;
}> = ({ draft, accounts, rules, investment, candidates, contacts, year, onClose, onSaved }) => {
  const writeLock = useWriteLock();
  const [asset, setAsset] = useState<Partial<FixedAsset>>(draft ?? {});
  // Beträge stehen als Text im Formular und werden erst beim Speichern gelesen.
  // Ein Feld, das bei jedem Tastendruck neu formatiert, lässt sich nicht tippen.
  const [costText, setCostText] = useState('');
  const [foreignText, setForeignText] = useState('');
  const [selfUsable, setSelfUsable] = useState(true);
  const [advice, setAdvice] = useState<AcquisitionAdvice | null>(null);
  const [plan, setPlan] = useState<AssetScheduleYear[]>([]);
  const [pool, setPool] = useState<FixedAsset | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (draft) {
      setAsset(draft);
      setCostText(draft.acquisitionCost ? formatCentsPlain(draft.acquisitionCost) : '');
      setForeignText(draft.foreignCost ? formatCentsPlain(draft.foreignCost) : '');
      setError(null);
      setAdvice(null);
      setPlan([]);
    }
  }, [draft]);

  const assetClass = (asset.class ?? 'tangible') as AssetClass;
  const isNew = !asset.id;
  const catalog = accounts.filter((a) => a.class === assetClass);
  const selectedAccount = catalog.find((a) => a.number === asset.account);

  const cost = parseCents(costText);
  // § 7g Abs. 5 EStG begünstigt nur abnutzbare bewegliche Wirtschaftsgüter, und
  // die Sonderabschreibung tritt neben die Absetzung für Abnutzung „nach § 7
  // Absatz 1 oder Absatz 2" — also neben die lineare wie neben die degressive.
  // Beides entscheidet hier darüber, ob das Feld überhaupt erscheint, statt erst
  // beim Speichern als Fehlermeldung.
  const specialAvailable =
    assetClass === 'tangible' &&
    (asset.method === 'linear' || asset.method === 'degressive') &&
    Boolean(selectedAccount) &&
    !selectedAccount?.immovable &&
    !selectedAccount?.inProgress;
  const usesSpecial = (asset.specialPermille ?? 0) > 0;
  const costError =
    costText.trim() !== '' && (cost === null || cost <= 0)
      ? 'Erwartet wird ein Betrag wie 1.234,56.'
      : undefined;

  // Die Einordnung nach § 6 Abs. 2 und 2a EStG rechnet das Backend. Sie wird
  // angefragt, sobald Betrag und Datum stehen — und nur für Sachanlagen, weil
  // die Wertgrenzen nur dort greifen.
  useEffect(() => {
    if (assetClass !== 'tangible' || !cost || cost <= 0 || !asset.acquisitionDate) {
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
  }, [assetClass, cost, asset.acquisitionDate, selfUsable]);

  // Der Plan wird nicht hier gerechnet, sondern gefragt — dieselbe Rechnung, die
  // später auch bucht. So steht schon in der Maske, was die Nutzungsdauer
  // bedeutet, statt erst nach dem Speichern.
  useEffect(() => {
    if (!cost || cost <= 0 || !asset.acquisitionDate || !asset.method || asset.method === 'none') {
      setPlan([]);
      return;
    }
    if ((asset.method === 'linear' || asset.method === 'degressive') && !asset.usefulLifeMonths) {
      setPlan([]);
      return;
    }
    let cancelled = false;
    Api.previewDepreciationPlan({
      acquisitionDate: asset.acquisitionDate,
      cost,
      usefulLifeMonths: asset.usefulLifeMonths ?? 0,
      method: asset.method,
      poolYear: asset.poolYear || year,
      specialPermille: asset.specialPermille ?? 0,
      specialYears: asset.specialYears ?? 0,
    })
      .then((rows) => {
        if (!cancelled) setPlan(rows ?? []);
      })
      .catch(() => setPlan([]));
    return () => {
      cancelled = true;
    };
  }, [
    cost,
    asset.acquisitionDate,
    asset.method,
    asset.usefulLifeMonths,
    asset.poolYear,
    asset.specialPermille,
    asset.specialYears,
    year,
  ]);

  // Es gibt genau einen Sammelposten je Wirtschaftsjahr. Besteht er schon, wird
  // das Gut dort eingestellt statt ein zweiter Posten angelegt.
  useEffect(() => {
    if (asset.method !== 'pool' || !isNew) {
      setPool(null);
      return;
    }
    let cancelled = false;
    Api.getSammelposten(asset.poolYear || year)
      .then((found) => {
        if (!cancelled) setPool(found);
      })
      .catch(() => setPool(null));
    return () => {
      cancelled = true;
    };
  }, [asset.method, asset.poolYear, isNew, year]);

  function set(patch: Partial<FixedAsset>) {
    setAsset((prev) => ({ ...prev, ...patch }));
  }

  /**
   * Die Zugangsbuchung kennt Konto, Betrag, Datum und Lieferant bereits. Sie
   * abzutippen ist genau die Art Arbeit, die eine Buchhaltung nicht braucht.
   */
  function pickEntry(entryId: number) {
    if (entryId === 0) {
      set({ acquisitionEntryId: undefined });
      return;
    }
    const candidate = candidates.find((c) => c.entryId === entryId);
    if (!candidate) {
      set({ acquisitionEntryId: entryId });
      return;
    }
    const entry = accounts.find((a) => a.number === candidate.account);
    setCostText(formatCentsPlain(candidate.amount));
    set({
      acquisitionEntryId: entryId,
      account: candidate.account,
      acquisitionDate: candidate.bookingDate,
      contactId: candidate.contactId,
      name: asset.name || candidate.description,
      depreciationAccount: entry?.depreciationAccount ?? asset.depreciationAccount ?? '',
      usefulLifeMonths: entry?.defaultUsefulLifeMonths || asset.usefulLifeMonths || 0,
      method: entry && !entry.depreciable ? 'none' : asset.method ?? 'linear',
    });
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

  /**
   * Warum eine Methode hier nicht wählbar ist — oder undefined, wenn sie es ist.
   *
   * Die Gründe stehen alle schon fest, bevor jemand speichert: die Wertgrenzen aus
   * der Einordnung, das Zeitfenster der degressiven AfA und die Frage, ob sich das
   * gewählte Konto überhaupt abnutzt. Sie erst beim Speichern als Fehler zu
   * zeigen, wäre eine Maske, die etwas anbietet und es dann verweigert.
   */
  function methodBlocked(method: DepreciationMethod): string | undefined {
    if (selectedAccount && !selectedAccount.depreciable && method !== 'none') {
      return `${selectedAccount.name} nutzt sich nicht ab.`;
    }
    if (method === 'degressive' && asset.acquisitionDate) {
      const open = (rules?.degressiveWindows ?? []).some(
        (w) => asset.acquisitionDate! >= w.From && asset.acquisitionDate! <= w.Until,
      );
      if (!open) {
        return 'Für dieses Anschaffungsdatum ist die degressive AfA nicht zulässig.';
      }
    }
    if (advice && (method === 'immediate' || method === 'pool')) {
      if (!advice.allowed.includes(method === 'immediate' ? 'immediate' : 'pool')) {
        return method === 'immediate'
          ? `Über ${formatCents(advice.limits.immediate)} netto ist der Sofortabzug ausgeschlossen.`
          : `Der Sammelposten gilt von ${formatCents(advice.limits.poolLowerLimit)} bis ${formatCents(
              advice.limits.poolUpperLimit,
            )} netto.`;
      }
    }
    return undefined;
  }

  const currentOption: AcquisitionOption =
    asset.method === 'immediate' ? 'immediate' : asset.method === 'pool' ? 'pool' : 'activate';

  const methods = (rules?.methods ?? []).filter((m) => m.classes.includes(assetClass));
  const activeMethod = methods.find((m) => m.method === asset.method);
  const blockedReason = asset.method ? methodBlocked(asset.method) : undefined;
  const needsUsefulLife = asset.method === 'linear' || asset.method === 'degressive';

  /** Übernimmt den Vorschlag der Einordnung samt des Kontos, das dazu gehört. */
  function applyAdvice(option: AcquisitionOption) {
    if (option === 'immediate') {
      set({ method: 'immediate', account: '0670', depreciationAccount: '6260', usefulLifeMonths: 0 });
      return;
    }
    if (option === 'pool') {
      set({
        method: 'pool',
        account: '0675',
        depreciationAccount: '6264',
        usefulLifeMonths: 0,
        poolYear: asset.poolYear || year,
      });
      return;
    }
    set({ method: 'linear' });
  }

  /** Das Gut in den bestehenden Sammelposten des Jahres einstellen. */
  async function addToPool() {
    if (!pool || !cost || cost <= 0) {
      setError('Für den Zugang zum Sammelposten fehlt ein lesbarer Betrag.');
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const updated = await Api.recordAssetCostAdjustment({
        assetId: pool.id,
        date: asset.acquisitionDate ?? new Date().toISOString().slice(0, 10),
        amount: cost,
        note: asset.name || 'Zugang',
      });
      await onSaved(updated);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  async function submit() {
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
        // Wo die Sonderabschreibung nicht offensteht, darf auch kein Rest von
        // ihr mitgespeichert werden: ein Methodenwechsel würde sie sonst still
        // mitnehmen.
        ...(specialAvailable && usesSpecial
          ? {}
          : { specialPermille: 0, specialYears: 0, specialAccount: '', specialReason: '' }),
        ...(assetClass === 'financial' && asset.currency
          ? { foreignCost: parseCents(foreignText) ?? 0 }
          : { currency: '', foreignCost: 0 }),
        // Eine Fondsart und eine Fälligkeit an einer Maschine wären nicht
        // falsch, sondern sinnlos — und eine sinnlose Angabe wird später als
        // bedeutsam gelesen.
        ...(assetClass === 'financial' ? {} : { fundClass: '' as FundClass, maturityDate: '' }),
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
          error={costError}
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
        <Field
          label="Zugangsbuchung"
          optional
          help="Die gewählte Buchung füllt Konto, Betrag, Datum und Lieferant — sie weiß das alles bereits."
        >
          <Select
            items={[
              { value: 0, label: 'Nicht verknüpft' },
              ...candidates.map((c) => ({
                value: c.entryId,
                label: `${c.entryNumber} · ${c.account} · ${formatCents(c.amount)}`,
              })),
              ...(asset.acquisitionEntryId &&
              !candidates.some((c) => c.entryId === asset.acquisitionEntryId)
                ? [{ value: asset.acquisitionEntryId, label: 'Bereits verknüpft' }]
                : []),
            ]}
            value={asset.acquisitionEntryId ?? 0}
            onValueChange={(next) => pickEntry(Number(next))}
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
            <div className="flex items-start justify-between gap-4 rounded-control border border-accent-line bg-accent-soft px-4 py-3">
              <p className="text-body text-ink-muted">
                {ADVICE_LABEL[advice.recommended]} — {advice.reason}
                {advice.poolNote ? ` ${advice.poolNote}` : ''}
              </p>
              {advice.recommended !== currentOption && (
                <Button
                  variant="secondary"
                  size="sm"
                  className="shrink-0"
                  onClick={() => applyAdvice(advice.recommended)}
                >
                  Übernehmen
                </Button>
              )}
            </div>
          )}
        </div>
      )}

      <div className="grid grid-cols-3 gap-4 mt-4">
        <Field
          label="Abschreibungsmethode"
          hint={activeMethod?.hint}
          error={blockedReason}
        >
          <Select
            items={methods.map((m) => {
              const blocked = methodBlocked(m.method);
              return {
                value: m.method,
                label: blocked ? `${m.label} — ${blocked}` : m.label,
                disabled: Boolean(blocked),
              };
            })}
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

      {specialAvailable && (
        <div className="mt-4 space-y-3">
          <Checkbox
            checked={usesSpecial}
            onCheckedChange={(checked) =>
              set(
                checked
                  ? { specialPermille: rules?.specialMaxPermille ?? 400, specialYears: 1 }
                  : { specialPermille: 0, specialYears: 0, specialReason: '' },
              )
            }
            label="Sonderabschreibung nach § 7g Abs. 5 EStG in Anspruch nehmen"
            hint="Bis 40 % der Anschaffungskosten, zusätzlich zur Absetzung für Abnutzung — verteilbar auf das Anschaffungsjahr und die vier folgenden."
          />
          {usesSpecial && (
            <>
              <div className="grid grid-cols-3 gap-4">
                <Field
                  label="Satz in Prozent"
                  hint={`höchstens ${(rules?.specialMaxPermille ?? 400) / 10} %`}
                  help="Der Satz bemisst sich an den Anschaffungskosten, nicht am Restbuchwert. Die planmäßige AfA läuft daneben unverändert weiter — § 7g Abs. 5 EStG lässt die Sonderabschreibung neben der linearen wie neben der degressiven zu."
                >
                  <Input
                    type="number"
                    min={0}
                    max={(rules?.specialMaxPermille ?? 400) / 10}
                    step={1}
                    align="right"
                    value={(asset.specialPermille ?? 0) / 10}
                    onChange={(e) => set({ specialPermille: Math.round(Number(e.target.value) * 10) })}
                  />
                </Field>
                <Field
                  label="Verteilt auf Jahre"
                  hint={`eins bis ${rules?.specialPeriodYears ?? 5}`}
                  help="Wie der Betrag über den Begünstigungszeitraum verteilt wird, entscheidest du. Danach verteilt § 7a Abs. 9 EStG den Restwert auf die Restnutzungsdauer."
                >
                  <Input
                    type="number"
                    min={1}
                    max={rules?.specialPeriodYears ?? 5}
                    step={1}
                    align="right"
                    value={asset.specialYears ?? 1}
                    onChange={(e) => set({ specialYears: Number(e.target.value) })}
                  />
                </Field>
                <Field label="Aufwandskonto" hint="folgt aus dem Anlagekonto" optional>
                  <Input
                    value={asset.specialAccount ?? (selectedAccount?.depreciationAccount === '6222' ? '6242' : '6241')}
                    disabled
                  />
                </Field>
              </div>
              <Field
                label="Voraussetzungen nach § 7g Abs. 6 EStG"
                hint="Gewinn des Vorjahres höchstens 200.000 €, fast ausschließlich betriebliche Nutzung"
                help="Zwei Sachverhalte, die Buchfink nicht kennen kann. Halte fest, worauf sich die Inanspruchnahme stützt — die Angabe steht später bei der Buchung."
              >
                <Textarea
                  rows={2}
                  value={asset.specialReason ?? ''}
                  onChange={(e) => set({ specialReason: e.target.value })}
                />
              </Field>
            </>
          )}
        </div>
      )}

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
          <Field
            label="Stückzahl"
            optional
            hint="Anteile, Stücke, Nominale"
            help="Wird sie geführt, rechnet Buchfink beim Teilabgang den Anteil der Anschaffungskosten aus der Stückzahl. Ohne sie wird der Teilabgang als Betrag angegeben."
          >
            <Input
              type="number"
              min={0}
              step="any"
              align="right"
              value={asset.quantity ? asset.quantity / UNIT_SCALE : ''}
              onChange={(e) =>
                set({ quantity: Math.round(Number(e.target.value || '0') * UNIT_SCALE) })
              }
            />
          </Field>
          <Field
            label="Notierungswährung"
            optional
            hint="ISO-Code, leer heißt Euro"
            help="Nur nötig, wo das Papier tatsächlich in einer anderen Währung notiert. Aus Fremdbetrag und Euro-Anschaffungskosten ergibt sich der Anschaffungskurs, gegen den § 256a HGB den Stichtagskurs hält."
          >
            <Input
              className="code-num uppercase"
              maxLength={3}
              value={asset.currency ?? ''}
              onChange={(e) => set({ currency: e.target.value.toUpperCase() })}
            />
          </Field>
          <Field
            label="Fondsart"
            optional
            hint="entscheidet über die Teilfreistellung"
            help="Ein Investmentanteil steht in der Bilanz wie jedes andere Wertpapier. Steuerlich legt das Investmentsteuergesetz zwei Rechnungen daneben: die Teilfreistellung nach § 20 InvStG und die Vorabpauschale nach § 18 InvStG. Für eine Einzelaktie und eine Anleihe gibt es beides nicht."
          >
            <Select
              items={(investment?.fundClasses ?? [{ class: '' as FundClass, label: 'Kein Investmentanteil' }]).map(
                (f) => ({ value: f.class, label: f.label }),
              )}
              value={asset.fundClass ?? ''}
              onValueChange={(next) => set({ fundClass: next as FundClass })}
            />
          </Field>
          <Field
            label="Fälligkeit"
            optional
            hint="bei einer Ausleihung"
            help="Sie entscheidet über die Bewertung: § 256a Satz 2 HGB nimmt Posten mit einer Restlaufzeit von höchstens einem Jahr vom Anschaffungskostenprinzip aus — dort schlägt ein gestiegener Kurs voll durch."
          >
            <Input
              type="date"
              value={asset.maturityDate ?? ''}
              onChange={(e) => set({ maturityDate: e.target.value })}
            />
          </Field>
          {asset.currency ? (
            <Field label={`Anschaffungskosten in ${asset.currency}`} hint="Betrag in der Notierungswährung">
              <Input
                align="right"
                inputMode="decimal"
                placeholder="0,00"
                value={foreignText}
                onChange={(e) => setForeignText(e.target.value)}
              />
            </Field>
          ) : null}
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

      {pool && (
        <div className="mt-4 flex items-start justify-between gap-4 rounded-control border border-attention-line bg-attention-soft px-4 py-3">
          <p className="text-body text-attention-text">
            Für {asset.poolYear || year} besteht bereits der Sammelposten {pool.inventoryNumber} über{' '}
            {formatCents(pool.cost)}. § 6 Abs. 2a EStG kennt genau einen je Wirtschaftsjahr.
          </p>
          <Button
            variant="secondary"
            size="sm"
            className="shrink-0"
            loading={busy}
            onClick={addToPool}
          >
            Dort aufnehmen
          </Button>
        </div>
      )}

      {plan.length > 0 && (
        <div className="mt-4 rounded-control border border-line bg-sunken px-4 py-3">
          <div className="text-overline text-ink-subtle mb-2">So läuft die Abschreibung</div>
          <div className="flex flex-wrap gap-x-8 gap-y-1 text-body text-ink-muted">
            <span>
              {plan[0].fiscalYear}:{' '}
              <span className="num text-ink">{formatCents(plan[0].amount)}</span>
              {plan[0].months < 12 && (
                <span className="text-ink-subtle"> · {plan[0].months} von 12 Monaten</span>
              )}
            </span>
            {plan.length > 1 && (
              <span>
                {plan[1].fiscalYear}:{' '}
                <span className="num text-ink">{formatCents(plan[1].amount)}</span>
              </span>
            )}
            <span className="text-ink-subtle">
              vollständig abgeschrieben {plan[plan.length - 1].fiscalYear}
            </span>
          </div>
        </div>
      )}

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

type DetailAction =
  | 'impairment'
  | 'writeUp'
  | 'cost'
  | 'disposal'
  | 'transfer'
  | 'maintenance'
  | 'income'
  | 'currency'
  | 'document'
  | 'vorabpauschale'
  | null;

const AssetDetailDialog: React.FC<{
  assetId: number | null;
  contacts: Contact[];
  paymentAccounts: Account[];
  accounts: AssetAccountInfo[];
  documentKinds: AssetDocumentKindInfo[];
  onClose: () => void;
  onEdit: (asset: FixedAsset) => void;
  onChanged: () => Promise<void>;
}> = ({
  assetId,
  contacts,
  paymentAccounts,
  accounts,
  documentKinds,
  onClose,
  onEdit,
  onChanged,
}) => {
  const writeLock = useWriteLock();
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
  // Von einer Anlage im Bau wird umgebucht — der Katalog sagt, welche Konten das sind.
  const inProgress = Boolean(
    asset && accounts.find((a) => a.number === asset.account)?.inProgress,
  );

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
            <Button
              variant="quiet"
              disabled={writeLock.locked}
              title={writeLock.hint}
              onClick={() => setAction('cost')}
            >
              Erweiterung erfassen
            </Button>
            {!asset.disposalDate && inProgress && (
              <Button
                variant="secondary"
                disabled={writeLock.locked}
                title={writeLock.hint}
                onClick={() => setAction('transfer')}
              >
                Fertigstellung buchen
              </Button>
            )}
            {!asset.disposalDate && asset.class !== 'financial' && (
              <Button
                variant="quiet"
                disabled={writeLock.locked}
                title={writeLock.hint}
                onClick={() => setAction('maintenance')}
              >
                Erhaltungsaufwand
              </Button>
            )}
            {!asset.disposalDate && asset.class === 'financial' && (
              <Button
                variant="quiet"
                disabled={writeLock.locked}
                title={writeLock.hint}
                onClick={() => setAction('income')}
              >
                Ertrag buchen
              </Button>
            )}
            {!asset.disposalDate && asset.currency && (
              <Button
                variant="quiet"
                disabled={writeLock.locked}
                title={writeLock.hint}
                onClick={() => setAction('currency')}
              >
                Währung bewerten
              </Button>
            )}
            {!asset.disposalDate && asset.fundClass && (
              <Button
                variant="quiet"
                disabled={writeLock.locked}
                title={writeLock.hint}
                onClick={() => setAction('vorabpauschale')}
              >
                Vorabpauschale
              </Button>
            )}
            <Button
              variant="quiet"
              disabled={writeLock.locked}
              title={writeLock.hint}
              onClick={() => setAction('document')}
            >
              Dokument ablegen
            </Button>
            {!asset.disposalDate && (
              <>
                <Button
                  variant="secondary"
                  disabled={writeLock.locked}
                  title={writeLock.hint}
                  onClick={() => setAction('impairment')}
                >
                  Außerplanmäßig abschreiben
                </Button>
                <Button
                  variant="secondary"
                  disabled={writeLock.locked}
                  title={writeLock.hint}
                  onClick={() => setAction('writeUp')}
                >
                  Zuschreiben
                </Button>
                <Button
                  variant="secondary"
                  disabled={writeLock.locked}
                  title={writeLock.hint}
                  onClick={() => onEdit(asset)}
                >
                  Bearbeiten
                </Button>
                <Button
                  variant="primary"
                  disabled={writeLock.locked}
                  title={writeLock.hint}
                  onClick={() => setAction('disposal')}
                >
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
        <WriteUpForm asset={asset} ceiling={detail.writeUpCeiling} onDone={afterBooking} />
      ) : action === 'cost' ? (
        <CostAdjustmentForm asset={asset} onDone={afterBooking} />
      ) : action === 'transfer' ? (
        <TransferForm asset={asset} accounts={accounts} onDone={afterBooking} />
      ) : action === 'maintenance' ? (
        <MaintenanceForm
          asset={asset}
          contacts={contacts}
          paymentAccounts={paymentAccounts}
          onDone={afterBooking}
        />
      ) : action === 'income' ? (
        <AssetIncomeForm
          asset={asset}
          contacts={contacts}
          paymentAccounts={paymentAccounts}
          onDone={afterBooking}
        />
      ) : action === 'currency' ? (
        <CurrencyForm asset={asset} onDone={afterBooking} />
      ) : action === 'document' ? (
        <DocumentForm asset={asset} kinds={documentKinds} onDone={afterBooking} />
      ) : action === 'vorabpauschale' ? (
        <VorabpauschaleForm asset={asset} onDone={afterBooking} />
      ) : action === 'disposal' ? (
        <DisposalForm
          asset={asset}
          contacts={contacts}
          paymentAccounts={paymentAccounts}
          onDone={afterBooking}
        />
      ) : (
        <AssetOverview
          detail={detail}
          onDocumentsChanged={async () => {
            if (assetId !== null) await load(assetId);
            await onChanged();
          }}
        />
      )}
    </Dialog>
  );
};

const AssetOverview: React.FC<{
  detail: AssetDetail;
  onDocumentsChanged: () => Promise<void>;
}> = ({ detail, onDocumentsChanged }) => {
  const { asset, schedule, movements, notes } = detail;
  // Die Spalte erscheint nur, wo es eine Sonderabschreibung gibt. Eine leere
  // Spalte in jedem Plan wäre eine Frage, die sich niemand gestellt hat.
  const hasSpecial = schedule.some((row) => (row.specialAmount ?? 0) > 0);
  const tracksUnits = movements.some((movement) => Boolean(movement.quantity));

  return (
    <div className="space-y-6">
      <StatRow>
        <Stat label="Anschaffungskosten" value={formatCents(asset.cost)} context={formatDate(asset.acquisitionDate)} />
        <Stat label="Kumulierte Abschreibungen" value={formatCents(asset.accumulated)} />
        <Stat label="Buchwert" value={formatCents(asset.bookValue)} />
        {asset.unitsHeld ? (
          <Stat
            label="Bestand"
            value={`${formatUnits(asset.unitsHeld)} Stück`}
            context={asset.identifier}
          />
        ) : (
          <Stat
            label="Konto"
            value={<span className="code-num text-body">{asset.account}</span>}
            context={asset.accountName}
          />
        )}
      </StatRow>

      {asset.currency && asset.foreignCost ? (
        <div className="rounded-control border border-line bg-sunken px-4 py-3 text-body text-ink-muted">
          Notiert in {asset.currency}: {formatCentsPlain(asset.foreignCost)} {asset.currency} zu
          Anschaffungskosten von {formatCents(asset.acquisitionCost)}. Zum Abschlussstichtag ist zum
          Devisenkassamittelkurs umzurechnen (§ 256a HGB) — nach oben begrenzt durch die
          Anschaffungskosten.
        </div>
      ) : null}

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
          <Table density="kompakt" className={hasSpecial ? '[&_td]:px-2.5 [&_th]:px-2.5' : undefined}>
            <Thead>
              <Tr>
                <Th className="w-20">Jahr</Th>
                <Th className="w-20">Monate</Th>
                <Th>Satz</Th>
                <Th numeric>Buchwert Anfang</Th>
                <Th numeric>Abschreibung</Th>
                {hasSpecial && <Th numeric>Sonderabschreibung</Th>}
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
                  {hasSpecial && (
                    <Td numeric>
                      {(row.specialAmount ?? 0) === 0 ? '—' : formatCents(row.specialAmount ?? 0)}
                    </Td>
                  )}
                  <Td numeric>{formatCents(row.booked + row.specialBooked)}</Td>
                  <Td numeric>{formatCents(row.closingBookValue)}</Td>
                  <Td className="text-caption text-ink-subtle">{row.status}</Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        </Section>
      )}

      <DocumentSection
        asset={asset}
        onChanged={async () => {
          await onDocumentsChanged();
        }}
      />

      <Section title="Bewegungen" context="Jede Wertänderung mit ihrer Buchung">
        <Table density="kompakt">
          <Thead>
            <Tr>
              <Th className="w-28">Datum</Th>
              <Th>Vorgang</Th>
              <Th numeric>Anschaffungskosten</Th>
              <Th numeric>Abschreibungen</Th>
              {tracksUnits && <Th numeric>Stück</Th>}
              <Th className="w-32">Buchung</Th>
            </Tr>
          </Thead>
          <Tbody>
            {movements.map((movement) => (
              <Tr key={movement.id}>
                <Td className="text-ink-muted">{formatDate(movement.date)}</Td>
                <Td className="max-w-[18rem] truncate" title={movement.note}>
                  {MOVEMENT_LABEL[movement.kind] ?? movement.kind}
                  {movement.account && movement.kind === 'transfer' && (
                    <span className="text-ink-subtle code-num text-caption"> · {movement.account}</span>
                  )}
                </Td>
                <Td numeric>{movement.costAmount === 0 ? '—' : formatCents(movement.costAmount)}</Td>
                <Td numeric>
                  {movement.depreciationAmount === 0 ? '—' : formatCents(movement.depreciationAmount)}
                </Td>
                {tracksUnits && (
                  <Td numeric>{movement.quantity ? formatUnits(movement.quantity) : '—'}</Td>
                )}
                <Td code>{movement.entryNumber ?? '—'}</Td>
              </Tr>
            ))}
          </Tbody>
        </Table>
      </Section>
    </div>
  );
};

/**
 * Die Papiere zum Anlagegut.
 *
 * Sie sind kein zweiter Belegkreis: kein Nummernkreis, kein Geschäftsjahr,
 * keine Versiegelung. Ein Kaufvertrag wird nicht gebucht — er erklärt die
 * Anschaffung noch, wenn die Maschine zehn Jahre im Bestand ist.
 */
const DocumentSection: React.FC<{
  asset: FixedAsset;
  onChanged: () => Promise<void>;
}> = ({ asset, onChanged }) => {
  const writeLock = useWriteLock();
  const documents = asset.documents ?? [];
  const [busy, setBusy] = useState<number | null>(null);
  const today = new Date().toISOString().slice(0, 10);

  async function open(document: AssetDocument) {
    try {
      const preview = await Api.getAssetDocumentContent(document.id);
      if (!preview.intact) {
        toast.error(
          `${document.fileName} stimmt nicht mehr mit der Prüfsumme überein, unter der es abgelegt wurde.`,
        );
      }
      window.open(preview.dataUrl, '_blank');
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    }
  }

  async function remove(document: AssetDocument) {
    setBusy(document.id);
    try {
      await Api.removeAssetDocument(asset.id, document.id);
      toast.success(`${document.title || document.fileName} entfernt.`);
      await onChanged();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(null);
    }
  }

  if (documents.length === 0) return null;

  return (
    <Section
      title="Dokumente"
      context="Verträge, Gutachten und Papiere — abgelegt, nicht gebucht"
    >
      <Table density="kompakt">
        <Thead>
          <Tr>
            <Th className="w-44">Art</Th>
            <Th>Bezeichnung</Th>
            <Th className="w-28">Datum</Th>
            <Th className="w-28">Läuft ab</Th>
            <Th className="w-32" />
          </Tr>
        </Thead>
        <Tbody>
          {documents.map((document) => (
            <Tr key={document.id}>
              <Td className="text-ink-muted">{DOCUMENT_KIND_LABEL[document.kind] ?? document.kind}</Td>
              <Td className="max-w-[20rem] truncate" title={document.note || document.fileName}>
                <button
                  type="button"
                  className="text-left hover:underline"
                  onClick={() => void open(document)}
                >
                  {document.title || document.fileName}
                </button>
              </Td>
              <Td className="text-ink-muted">
                {document.documentDate ? formatDate(document.documentDate) : '—'}
              </Td>
              <Td
                className={
                  document.validUntil && document.validUntil <= today
                    ? 'text-negative-text'
                    : 'text-ink-muted'
                }
              >
                {document.validUntil ? formatDate(document.validUntil) : '—'}
              </Td>
              <Td>
                <Button
                  variant="quiet"
                  size="sm"
                  loading={busy === document.id}
                  disabled={writeLock.locked}
                  title={writeLock.hint}
                  onClick={() => void remove(document)}
                >
                  Entfernen
                </Button>
              </Td>
            </Tr>
          ))}
        </Tbody>
      </Table>
    </Section>
  );
};

/**
 * Die Ausleihungen aus dem Kontenkatalog. Nur sie werden getilgt — eine
 * Beteiligung und ein Wertpapier werden verkauft.
 */
const LOAN_ACCOUNTS = ['0810', '0880', '0930', '0940', '0990'];

const DOCUMENT_KIND_LABEL: Record<string, string> = {
  contract: 'Vertrag',
  invoice: 'Rechnung (Kopie)',
  valuation: 'Gutachten',
  registration: 'Register- oder Zulassungspapier',
  insurance: 'Versicherung',
  maintenance: 'Wartung und Prüfung',
  statement: 'Abrechnung',
  photo: 'Bild',
  other: 'Sonstiges',
};

const DocumentForm: React.FC<{
  asset: FixedAsset;
  kinds: AssetDocumentKindInfo[];
  onDone: (message: string) => Promise<void>;
}> = ({ asset, kinds, onDone }) => {
  const writeLock = useWriteLock();
  const [kind, setKind] = useState<AssetDocumentKind>('contract');
  const [paths, setPaths] = useState<string[]>([]);
  const [title, setTitle] = useState('');
  const [documentDate, setDocumentDate] = useState('');
  const [validUntil, setValidUntil] = useState('');
  const [note, setNote] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function pick() {
    try {
      const chosen = await Api.selectAssetDocumentsDialog('Dokumente zum Anlagegut auswählen');
      if (chosen?.length) setPaths(chosen);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function submit() {
    if (paths.length === 0) {
      setError('Es wurde keine Datei ausgewählt.');
      return;
    }
    setBusy(true);
    setError(null);
    try {
      for (const path of paths) {
        await Api.attachAssetDocument({
          assetId: asset.id,
          kind,
          path,
          // Ein Titel für mehrere Dateien wäre für alle bis auf eine falsch.
          title: paths.length === 1 ? title : '',
          documentDate,
          validUntil,
          note,
        });
      }
      await onDone(paths.length === 1 ? 'Dokument abgelegt.' : `${paths.length} Dokumente abgelegt.`);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="space-y-4">
      <FormHint
        label="Erklärung zu den Dokumenten"
        line="Was hier liegt, wird nicht gebucht — es gehört zum Wirtschaftsgut, nicht zum Geschäftsjahr."
      >
        Der Beleg zur Anschaffung steht im Belegkreis, mit Belegnummer und in der Journalkette. Ein
        Kaufvertrag, ein Gutachten, ein Fahrzeugbrief sind nichts davon: sie erklären das
        Wirtschaftsgut noch, wenn es zehn Jahre im Bestand ist. Abgelegt werden sie auf demselben
        Weg wie ein Beleg — unter ihrer eigenen Prüfsumme, sodass später feststeht, ob noch dort
        liegt, was abgelegt wurde. Die Aufbewahrungspflicht des § 147 AO ersetzt das nicht; sie
        trifft weiterhin das Original.
      </FormHint>

      <div className="grid grid-cols-2 gap-4">
        <Field label="Art des Dokuments">
          <Select
            items={kinds.map((k) => ({ value: k.kind, label: k.label }))}
            value={kind}
            onValueChange={(next) => setKind(next as AssetDocumentKind)}
          />
        </Field>
        <Field
          label="Datei"
          hint={
            paths.length === 0
              ? 'noch keine ausgewählt'
              : paths.length === 1
                ? paths[0].split(/[\\/]/).pop()
                : `${paths.length} Dateien`
          }
        >
          <Button variant="secondary" onClick={pick}>
            Datei auswählen
          </Button>
        </Field>
      </div>

      {paths.length === 1 && (
        <Field label="Bezeichnung" optional hint="was in der Liste steht; leer heißt der Dateiname">
          <Input value={title} onChange={(e) => setTitle(e.target.value)} />
        </Field>
      )}

      <div className="grid grid-cols-3 gap-4">
        <Field label="Datum des Dokuments" optional>
          <Input
            type="date"
            value={documentDate}
            onChange={(e) => setDocumentDate(e.target.value)}
          />
        </Field>
        <Field
          label="Läuft ab am"
          optional
          hint="Police, Frist, Fälligkeit"
          help="Ein Ablaufdatum, das niemand wieder liest, wäre keine Angabe. Buchfink beantwortet damit, was bis zu einem Stichtag ausläuft."
        >
          <Input type="date" value={validUntil} onChange={(e) => setValidUntil(e.target.value)} />
        </Field>
        <Field label="Notiz" optional>
          <Input value={note} onChange={(e) => setNote(e.target.value)} />
        </Field>
      </div>

      <FormError message={error} />

      <div className="flex justify-end">
        <Button
          variant="primary"
          loading={busy}
          disabled={paths.length === 0 || writeLock.locked}
          title={writeLock.hint}
          onClick={submit}
        >
          Ablegen
        </Button>
      </div>
    </div>
  );
};

/**
 * Die Vorabpauschale nach § 18 InvStG.
 *
 * Der Fall, um den es geht: ein thesaurierender Fonds schüttet nichts aus, und
 * zu versteuern ist trotzdem etwas. Gebucht wird nichts — handelsrechtlich
 * geschieht nichts —, festgehalten schon, weil der Betrag beim Abgang wieder
 * abgezogen wird.
 */
const VorabpauschaleForm: React.FC<{
  asset: FixedAsset;
  onDone: (message: string) => Promise<void>;
}> = ({ asset, onDone }) => {
  const writeLock = useWriteLock();
  const [year, setYear] = useState(new Date().getFullYear() - 1);
  const [opening, setOpening] = useState('');
  const [closing, setClosing] = useState('');
  const [distributions, setDistributions] = useState('');
  const [basisPercent, setBasisPercent] = useState('');
  const [result, setResult] = useState<Vorabpauschale | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const basisPoints = Math.round(Number(basisPercent.replace(',', '.') || '0') * 100);
  const request = {
    assetId: asset.id,
    year,
    openingPrice: parseCents(opening) ?? 0,
    closingPrice: parseCents(closing) ?? 0,
    distributions: parseCents(distributions) ?? 0,
    basisPoints,
  };

  // Gerechnet wird im Backend, auch die Vorschau: § 18 InvStG hat drei
  // Begrenzungen, und eine zweite Fassung davon hier driftete beim ersten
  // Sonderfall.
  useEffect(() => {
    if (basisPoints <= 0 || request.openingPrice <= 0) {
      setResult(null);
      return;
    }
    let cancelled = false;
    Api.computeVorabpauschale(request)
      .then((next) => {
        if (!cancelled) {
          setResult(next);
          setError(null);
        }
      })
      .catch((e) => {
        if (!cancelled) {
          setResult(null);
          setError(e instanceof Error ? e.message : String(e));
        }
      });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [asset.id, year, opening, closing, distributions, basisPercent]);

  async function record() {
    setBusy(true);
    setError(null);
    try {
      await Api.computeVorabpauschale({ ...request, record: true });
      await onDone(`Vorabpauschale ${year} festgehalten.`);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="space-y-4">
      <FormHint
        label="Erklärung zur Vorabpauschale"
        line="Zu versteuern, ohne dass Geld fließt — und deshalb nicht zu buchen."
      >
        Schüttet ein Fonds weniger aus als den Basisertrag, ist die Differenz zu versteuern
        (§ 18 Abs. 1 InvStG). Der Basisertrag sind 70 % des Basiszinses auf den Rücknahmepreis zu
        Jahresbeginn, begrenzt auf den Wertzuwachs des Jahres; im Erwerbsjahr wird um ein Zwölftel
        je vollem Monat vor dem Erwerb gekürzt. Handelsrechtlich geschieht nichts, deshalb entsteht
        keine Buchung. Festgehalten wird sie trotzdem: beim Abgang wird sie wieder abgezogen, weil
        sie über die Jahre schon versteuert wurde.
      </FormHint>

      <div className="grid grid-cols-4 gap-4">
        <Field label="Kalenderjahr" help="§ 18 InvStG rechnet nach Kalenderjahren, auch bei einem abweichenden Wirtschaftsjahr.">
          <Input
            type="number"
            align="right"
            value={year}
            onChange={(e) => setYear(Number(e.target.value))}
          />
        </Field>
        <Field label="Rücknahmepreis am Jahresanfang" hint="des gehaltenen Bestands">
          <Input
            align="right"
            inputMode="decimal"
            placeholder="0,00"
            value={opening}
            onChange={(e) => setOpening(e.target.value)}
          />
        </Field>
        <Field label="Rücknahmepreis am Jahresende">
          <Input
            align="right"
            inputMode="decimal"
            placeholder="0,00"
            value={closing}
            onChange={(e) => setClosing(e.target.value)}
          />
        </Field>
        <Field label="Ausschüttungen des Jahres" optional>
          <Input
            align="right"
            inputMode="decimal"
            placeholder="0,00"
            value={distributions}
            onChange={(e) => setDistributions(e.target.value)}
          />
        </Field>
      </div>

      <Field
        label="Basiszins in Prozent"
        hint="aus dem BMF-Schreiben im Bundessteuerblatt"
        help="Der Basiszins steht nicht im Gesetz. Die Bundesbank errechnet ihn auf den ersten Börsentag des Jahres, das Bundesministerium der Finanzen veröffentlicht ihn im Bundessteuerblatt (§ 18 Abs. 4 InvStG). Buchfink liefert ihn deshalb nicht mit — ein mitgelieferter Wert wäre im nächsten Jahr falsch."
        className="max-w-xs"
      >
        <Input
          align="right"
          inputMode="decimal"
          placeholder="2,53"
          value={basisPercent}
          onChange={(e) => setBasisPercent(e.target.value)}
        />
      </Field>

      {result && (
        <Section title="Was daraus folgt" divider={false}>
          <StatRow className="mb-4">
            <Stat label="Basisertrag" value={formatCents(result.basisReturn)} />
            <Stat
              label="Wertzuwachs"
              value={formatCents(result.growth)}
              context={result.capped ? 'begrenzt den Basisertrag' : undefined}
            />
            <Stat
              label="Vorabpauschale"
              value={formatCents(result.amount)}
              context={result.monthsCounted < 12 ? `${result.monthsCounted} von 12 Monaten` : undefined}
            />
            <Stat label="Gilt als zugeflossen" value={formatDate(result.accruedOn)} />
          </StatRow>
          <p className="text-body text-ink-muted">{result.explanation}</p>
        </Section>
      )}

      <FormError message={error} />

      <div className="flex justify-end">
        <Button
          variant="primary"
          loading={busy}
          disabled={!result || result.amount <= 0 || writeLock.locked}
          title={writeLock.hint}
          onClick={record}
        >
          Festhalten
        </Button>
      </div>
    </div>
  );
};

/** Die steuerliche Nebenrechnung eines Investmentanteils, neben der Buchung. */
const InvestmentNote: React.FC<{ note: InvestmentTaxNote }> = ({ note }) => (
  <div className="rounded-control border border-line bg-sunken px-4 py-3 space-y-2">
    <div className="text-overline text-ink-subtle">
      Steuerlich daneben · {note.fundClassLabel}
    </div>
    <div className="flex flex-wrap gap-x-8 gap-y-1 text-body">
      <span className="text-ink-muted">
        Vor Teilfreistellung <span className="num text-ink">{formatCents(note.grossAmount)}</span>
      </span>
      {note.vorabpauschalen > 0 && (
        <span className="text-ink-muted">
          Angesetzte Vorabpauschalen{' '}
          <span className="num text-ink">−{formatCents(note.vorabpauschalen)}</span>
        </span>
      )}
      <span className="text-ink-muted">
        Steuerfrei <span className="num text-ink">{formatCents(note.exemptAmount)}</span>
      </span>
      <span className="text-ink-muted">
        Zu versteuern <span className="num text-ink">{formatCents(note.taxableAmount)}</span>
      </span>
    </div>
    <p className="text-caption text-ink-subtle">{note.explanation}</p>
  </div>
);

const MOVEMENT_LABEL: Record<string, string> = {
  acquisition: 'Zugang',
  subsequent_cost: 'Nachträgliche Anschaffungskosten',
  cost_reduction: 'Anschaffungskostenminderung',
  depreciation: 'Planmäßige Abschreibung',
  special_depreciation: 'Sonderabschreibung (§ 7g Abs. 5 EStG)',
  impairment: 'Außerplanmäßige Abschreibung',
  write_up: 'Zuschreibung',
  maintenance: 'Erhaltungsaufwand',
  income: 'Laufender Ertrag',
  disposal: 'Abgang',
  transfer: 'Umbuchung',
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
  const writeLock = useWriteLock();
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
        <Field
          label="Betrag"
          hint={`höchstens ${formatCents(asset.bookValue)} — der heutige Buchwert`}
          error={
            (parseCents(amount) ?? 0) > asset.bookValue
              ? 'Mehr als den Buchwert kann ein Anlagegut nicht verlieren.'
              : undefined
          }
        >
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
        <Button
          variant="primary"
          loading={busy}
          disabled={writeLock.locked}
          title={writeLock.hint}
          onClick={submit}
        >
          Abschreibung buchen
        </Button>
      </div>
    </div>
  );
};

const WriteUpForm: React.FC<{
  asset: FixedAsset;
  /** Höchstbetrag aus dem Backend — dieselbe Grenze, an der das Buchen scheitern würde. */
  ceiling: Cents;
  onDone: (message: string) => Promise<void>;
}> = ({ asset, ceiling, onDone }) => {
  const writeLock = useWriteLock();
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
        <Field
          label="Betrag"
          hint={
            ceiling > 0
              ? `höchstens ${formatCents(ceiling)} — fortgeführte Anschaffungskosten`
              : 'Es gibt nichts zuzuschreiben: der Buchwert entspricht den fortgeführten Anschaffungskosten.'
          }
          error={
            (parseCents(amount) ?? 0) > ceiling && ceiling > 0
              ? `Mehr als ${formatCents(ceiling)} lässt § 253 Abs. 5 Satz 1 HGB nicht zu.`
              : undefined
          }
        >
          <Input
            align="right"
            inputMode="decimal"
            placeholder="0,00"
            disabled={ceiling <= 0}
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
        <Button
          variant="primary"
          loading={busy}
          disabled={
            ceiling <= 0 ||
            (parseCents(amount) ?? 0) <= 0 ||
            (parseCents(amount) ?? 0) > ceiling ||
            writeLock.locked
          }
          title={writeLock.hint}
          onClick={submit}
        >
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
  const writeLock = useWriteLock();
  const [date, setDate] = useState(new Date().toISOString().slice(0, 10));
  const [amount, setAmount] = useState('');
  const [reduction, setReduction] = useState(false);
  const [extendYears, setExtendYears] = useState('0');
  const [note, setNote] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Die Nutzungsdauer kann eine Erweiterung verlängern — aber nur dort, wo es
  // überhaupt einen Plan gibt, den sie verlängern könnte.
  const canExtend = asset.method === 'linear' || asset.method === 'degressive';
  const extendMonths = Math.round(Number(extendYears || '0') * 12);

  async function submit() {
    setBusy(true);
    setError(null);
    try {
      const value = parseCents(amount);
      if (value === null || value <= 0) {
        setError('Der Betrag fehlt oder ist nicht lesbar.');
        return;
      }
      await Api.recordAssetCostAdjustment({
        assetId: asset.id,
        date,
        amount: value,
        reduction,
        extendLifeMonths: reduction || !canExtend ? 0 : extendMonths,
        note,
      });
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
        label="Erklärung zu Erweiterungen und nachträglichen Anschaffungskosten"
        line="Erweiterung oder Reparatur? Nur die Erweiterung erhöht die Anschaffungskosten."
      >
        Aktiviert wird, was das Anlagegut erweitert oder über seinen ursprünglichen Zustand hinaus
        wesentlich verbessert (§ 255 Abs. 2 HGB) — ein Anbau, ein zusätzliches Modul. Eine Reparatur,
        die es nur im Zustand hält, ist Erhaltungsaufwand und gehört sofort in die Gewinn- und
        Verlustrechnung: dafür gibt es am Anlagegut die eigene Aktion „Erhaltungsaufwand". Fracht und
        Montage zählen zu den Anschaffungskosten, Finanzierungskosten nicht (§ 255 Abs. 1 HGB).
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

      {!reduction && canExtend && (
        <Field
          label="Nutzungsdauer verlängert sich um"
          hint={
            extendMonths > 0
              ? `${extendMonths} Monate — wirkt ab ${date.slice(0, 4)}, die gebuchten Jahre bleiben`
              : 'Jahre · 0, wenn die Erweiterung nichts daran ändert'
          }
          help="Ein Anbau hält oft so lange wie das Gebäude. Die Verlängerung wirkt nach vorn: sie verteilt den Restbuchwert auf mehr Restmonate, ohne die bereits gebuchten Jahre anzurühren."
          className="max-w-xs"
        >
          <Input
            type="number"
            min={0}
            step={0.5}
            align="right"
            value={extendYears}
            onChange={(e) => setExtendYears(e.target.value)}
          />
        </Field>
      )}

      <Field label="Notiz" optional>
        <Input value={note} onChange={(e) => setNote(e.target.value)} />
      </Field>

      <FormError message={error} />

      <div className="flex justify-end">
        <Button
          variant="primary"
          loading={busy}
          disabled={writeLock.locked}
          title={writeLock.hint}
          onClick={submit}
        >
          Erfassen
        </Button>
      </div>
    </div>
  );
};

const MaintenanceForm: React.FC<{
  asset: FixedAsset;
  contacts: Contact[];
  paymentAccounts: Account[];
  onDone: (message: string) => Promise<void>;
}> = ({ asset, contacts, paymentAccounts, onDone }) => {
  const writeLock = useWriteLock();
  const [date, setDate] = useState(new Date().toISOString().slice(0, 10));
  const [amount, setAmount] = useState('');
  const [settlement, setSettlement] = useState<Settlement>('paid');
  const [paymentAccount, setPaymentAccount] = useState(paymentAccounts[0]?.number ?? '');
  const [contactId, setContactId] = useState(0);
  const [note, setNote] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const vendors = contacts.filter((c) => c.type === 'vendor');

  async function submit() {
    setBusy(true);
    setError(null);
    try {
      const value = parseCents(amount);
      if (value === null || value <= 0) {
        setError('Der Betrag fehlt oder ist nicht lesbar.');
        return;
      }
      await Api.bookAssetMaintenance({
        assetId: asset.id,
        date,
        amount: value,
        settlement,
        paymentAccount: settlement === 'paid' ? paymentAccount : undefined,
        contactId: settlement === 'open' ? contactId : undefined,
        note,
      });
      await onDone('Erhaltungsaufwand gebucht.');
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="space-y-4">
      <FormHint
        label="Erklärung zum Erhaltungsaufwand"
        line="Was den Zustand nur erhält, ist sofort abziehbar — es erhöht die Anschaffungskosten nicht."
      >
        Aktiviert wird nur, was das Wirtschaftsgut erweitert oder über seinen ursprünglichen Zustand
        hinaus wesentlich verbessert (§ 255 Abs. 2 Satz 1 HGB). Eine Reparatur, die es im Zustand
        hält, gehört sofort in die Gewinn- und Verlustrechnung. Die Buchung wird hier trotzdem mit
        dem Anlagegut verknüpft: wer später fragt, was die Maschine gekostet hat, sieht beides und
        kann es auseinanderhalten. Das Aufwandskonto folgt aus dem Anlagekonto.
      </FormHint>

      <div className="grid grid-cols-3 gap-4">
        <Field label="Datum">
          <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
        </Field>
        <Field label="Betrag" hint="netto, ohne Vorsteuer">
          <Input
            align="right"
            inputMode="decimal"
            placeholder="0,00"
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
          />
        </Field>
        <Field label="Zahlung">
          <RadioGroup
            options={[
              { value: 'paid', label: 'Sofort bezahlt' },
              { value: 'open', label: 'Offener Posten' },
            ]}
            value={settlement}
            onValueChange={(next) => setSettlement(next as Settlement)}
          />
        </Field>
      </div>

      <div className="grid grid-cols-2 gap-4">
        {settlement === 'paid' ? (
          <Field label="Zahlungsmittel">
            <Select
              items={paymentAccounts.map((a) => ({ value: a.number, label: `${a.number} ${a.name}` }))}
              value={paymentAccount}
              onValueChange={(next) => setPaymentAccount(String(next))}
            />
          </Field>
        ) : (
          <Field label="Lieferant" hint="die Verbindlichkeit steht auf seinem Personenkonto">
            <Select
              items={vendors.map((c) => ({ value: c.id, label: c.name }))}
              value={contactId}
              onValueChange={(next) => setContactId(Number(next))}
            />
          </Field>
        )}
        <Field
          label="Abgrenzung zur Erweiterung"
          hint="wird mit der Buchung festgehalten"
          help="Die Unterscheidung ist eine Einschätzung, keine Rechnung. Ohne festgehaltene Begründung ist sie später nicht mehr nachvollziehbar."
        >
          <Input value={note} onChange={(e) => setNote(e.target.value)} />
        </Field>
      </div>

      <FormError message={error} />

      <div className="flex justify-end">
        <Button
          variant="primary"
          loading={busy}
          disabled={writeLock.locked}
          title={writeLock.hint}
          onClick={submit}
        >
          Aufwand buchen
        </Button>
      </div>
    </div>
  );
};

const AssetIncomeForm: React.FC<{
  asset: FixedAsset;
  contacts: Contact[];
  paymentAccounts: Account[];
  onDone: (message: string) => Promise<void>;
}> = ({ asset, contacts, paymentAccounts, onDone }) => {
  const writeLock = useWriteLock();
  const [date, setDate] = useState(new Date().toISOString().slice(0, 10));
  const [amount, setAmount] = useState('');
  const [withholding, setWithholding] = useState('');
  const [settlement, setSettlement] = useState<Settlement>('paid');
  const [paymentAccount, setPaymentAccount] = useState(paymentAccounts[0]?.number ?? '');
  const [contactId, setContactId] = useState(0);
  const [note, setNote] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const customers = contacts.filter((c) => c.type === 'customer');
  const gross = parseCents(amount) ?? 0;
  const tax = parseCents(withholding) ?? 0;

  async function submit() {
    setBusy(true);
    setError(null);
    try {
      if (gross <= 0) {
        setError('Der Betrag fehlt oder ist nicht lesbar.');
        return;
      }
      await Api.bookAssetIncome({
        assetId: asset.id,
        date,
        amount: gross,
        withholdingTax: tax,
        settlement,
        paymentAccount: settlement === 'paid' ? paymentAccount : undefined,
        contactId: settlement === 'open' ? contactId : undefined,
        note,
      });
      await onDone('Ertrag gebucht.');
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="space-y-4">
      <FormHint
        label="Erklärung zum laufenden Ertrag"
        line="Eine Ausschüttung ist Ertrag des Jahres, kein Rückfluss der Anschaffungskosten."
      >
        Der Buchwert des Anteils bleibt deshalb unberührt. Verknüpft wird der Ertrag trotzdem, sonst
        bliebe die Frage unbeantwortbar, was dieser Anteil eingebracht hat. Das Ertragskonto folgt
        aus der Art der Finanzanlage: die Gewinn- und Verlustrechnung weist Beteiligungserträge,
        Erträge aus Ausleihungen und sonstige Zinsen getrennt aus (§ 275 Abs. 2 HGB). Eine
        einbehaltene Kapitalertragsteuer mindert den Zufluss und nicht den Ertrag — sie ist eine
        Vorauszahlung auf die eigene Steuer.
      </FormHint>

      <div className="grid grid-cols-3 gap-4">
        <Field label="Datum">
          <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
        </Field>
        <Field label="Ertrag" hint="brutto, vor Kapitalertragsteuer">
          <Input
            align="right"
            inputMode="decimal"
            placeholder="0,00"
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
          />
        </Field>
        <Field
          label="Einbehaltene Kapitalertragsteuer"
          optional
          hint={gross > 0 ? `Zufluss ${formatCents(gross - tax)}` : 'samt Solidaritätszuschlag'}
          error={tax > gross ? 'Mehr als der Ertrag kann nicht einbehalten worden sein.' : undefined}
        >
          <Input
            align="right"
            inputMode="decimal"
            placeholder="0,00"
            value={withholding}
            onChange={(e) => setWithholding(e.target.value)}
          />
        </Field>
      </div>

      <div className="grid grid-cols-3 gap-4">
        <Field label="Zufluss">
          <RadioGroup
            options={[
              { value: 'paid', label: 'Sofort zugeflossen' },
              { value: 'open', label: 'Offene Forderung' },
            ]}
            value={settlement}
            onValueChange={(next) => setSettlement(next as Settlement)}
          />
        </Field>
        {settlement === 'paid' ? (
          <Field label="Zahlungsmittel">
            <Select
              items={paymentAccounts.map((a) => ({ value: a.number, label: `${a.number} ${a.name}` }))}
              value={paymentAccount}
              onValueChange={(next) => setPaymentAccount(String(next))}
            />
          </Field>
        ) : (
          <Field label="Schuldner" hint="die Forderung steht auf seinem Personenkonto">
            <Select
              items={customers.map((c) => ({ value: c.id, label: c.name }))}
              value={contactId}
              onValueChange={(next) => setContactId(Number(next))}
            />
          </Field>
        )}
        <Field label="Notiz" optional>
          <Input value={note} onChange={(e) => setNote(e.target.value)} />
        </Field>
      </div>

      <FormError message={error} />

      <div className="flex justify-end">
        <Button
          variant="primary"
          loading={busy}
          disabled={writeLock.locked}
          title={writeLock.hint}
          onClick={submit}
        >
          Ertrag buchen
        </Button>
      </div>
    </div>
  );
};

const CurrencyForm: React.FC<{
  asset: FixedAsset;
  onDone: (message: string) => Promise<void>;
}> = ({ asset, onDone }) => {
  const writeLock = useWriteLock();
  const [date, setDate] = useState(new Date().toISOString().slice(0, 10));
  const [rate, setRate] = useState('');
  const [valuation, setValuation] = useState<CurrencyValuation | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const rateValue = Math.round(Number(rate.replace(',', '.') || '0') * RATE_SCALE);

  // Die Umrechnung rechnet das Backend — dieselbe Rechnung, die auch die
  // Obergrenze der Zuschreibung kennt.
  useEffect(() => {
    if (rateValue <= 0) {
      setValuation(null);
      return;
    }
    let cancelled = false;
    Api.valuateAssetCurrency({ assetId: asset.id, date, ratePerEuro: rateValue })
      .then((result) => {
        if (!cancelled) {
          setValuation(result);
          setError(null);
        }
      })
      .catch((e) => {
        if (!cancelled) {
          setValuation(null);
          setError(e instanceof Error ? e.message : String(e));
        }
      });
    return () => {
      cancelled = true;
    };
  }, [asset.id, date, rateValue]);

  async function book() {
    if (!valuation || valuation.proposedAmount <= 0) return;
    setBusy(true);
    setError(null);
    try {
      // Gebucht wird über die Konten der Währungsumrechnung (6880/4840), nicht
      // über die der außerplanmäßigen Abschreibung: sonst sähe ein Kursverlust
      // aus wie eine Wertminderung des Papiers selbst.
      await Api.bookAssetCurrencyValuation({ assetId: asset.id, date, ratePerEuro: rateValue });
      await onDone(
        valuation.proposal === 'impairment'
          ? 'Kursverlust aus der Währungsumrechnung gebucht.'
          : 'Kursgewinn aus der Währungsumrechnung gebucht.',
      );
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="space-y-4">
      <FormHint
        label="Erklärung zur Fremdwährungsbewertung"
        line="Umgerechnet wird zum Devisenkassamittelkurs des Abschlussstichtags (§ 256a HGB)."
      >
        Nach oben begrenzt das Anschaffungskostenprinzip das Ergebnis (§ 253 Abs. 1 Satz 1 HGB): die
        Ausnahme des § 256a Satz 2 HGB gilt nur bei einer Restlaufzeit von höchstens einem Jahr und
        passt auf ein Anlagegut nicht, das dauernd dem Geschäftsbetrieb dienen soll. Ein gefallener
        Kurs führt deshalb zu einer außerplanmäßigen Abschreibung, ein gestiegener höchstens zu einer
        Zuschreibung bis zu den Anschaffungskosten. Buchfink rechnet den Betrag; gebucht wird er über
        dieselben Wege wie jede andere Wertänderung.
      </FormHint>

      <div className="grid grid-cols-3 gap-4">
        <Field label="Stichtag">
          <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
        </Field>
        <Field
          label={`Kurs in ${asset.currency} je Euro`}
          hint={
            valuation && valuation.acquisitionRate > 0
              ? `angeschafft zu ${(valuation.acquisitionRate / RATE_SCALE).toFixed(4)}`
              : 'Devisenkassamittelkurs des Stichtags'
          }
        >
          <Input
            align="right"
            inputMode="decimal"
            placeholder="1,0000"
            value={rate}
            onChange={(e) => setRate(e.target.value)}
          />
        </Field>
        <Field label="Bestand in Fremdwährung" hint="aus den Stammdaten">
          <Input align="right" disabled value={formatCentsPlain(asset.foreignCost ?? 0)} />
        </Field>
      </div>

      {valuation && (
        <Section title="Was der Stichtagskurs bedeutet" divider={false}>
          <StatRow className="mb-4">
            <Stat
              label="Wert zum Stichtagskurs"
              value={formatCents(valuation.valueAtRate)}
              context={`${formatCentsPlain(valuation.foreignAmount)} ${valuation.currency}`}
            />
            <Stat label="Buchwert" value={formatCents(valuation.bookValue)} />
            <Stat
              label="Unterschied"
              value={formatCents(valuation.difference)}
              tone={valuation.difference < 0 ? 'negative' : 'positive'}
            />
            <Stat
              label={
                valuation.proposal === 'impairment'
                  ? 'Abzuschreiben'
                  : valuation.proposal === 'write_up'
                    ? 'Zuzuschreiben'
                    : 'Zu buchen'
              }
              value={formatCents(valuation.proposedAmount)}
            />
          </StatRow>
          <p className="text-body text-ink-muted">{valuation.explanation}</p>
          {valuation.shortTerm && (
            <p className="mt-2 text-caption text-ink-subtle">
              Die Restlaufzeit beträgt höchstens ein Jahr: § 256a Satz 2 HGB nimmt den Posten damit
              vom Anschaffungskostenprinzip aus — anders als bei einer Beteiligung ohne Fälligkeit
              schlägt ein gestiegener Kurs hier voll durch.
            </p>
          )}
        </Section>
      )}

      <FormError message={error} />

      <div className="flex justify-end">
        <Button
          variant="primary"
          loading={busy}
          disabled={!valuation || valuation.proposedAmount <= 0 || writeLock.locked}
          title={writeLock.hint}
          onClick={book}
        >
          {valuation?.proposal === 'write_up' ? 'Kursgewinn buchen' : 'Kursverlust buchen'}
        </Button>
      </div>
    </div>
  );
};

const TransferForm: React.FC<{
  asset: FixedAsset;
  accounts: AssetAccountInfo[];
  onDone: (message: string) => Promise<void>;
}> = ({ asset, accounts, onDone }) => {
  const writeLock = useWriteLock();
  const [date, setDate] = useState(new Date().toISOString().slice(0, 10));
  const [account, setAccount] = useState<string | null>(null);
  const [method, setMethod] = useState<DepreciationMethod>('linear');
  const [lifeYears, setLifeYears] = useState('');
  const [note, setNote] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Zielkonten sind die endgültigen Anlagekonten derselben Klasse — nicht die,
  // von denen gerade umgebucht wird.
  const targets = accounts.filter((a) => a.class === asset.class && !a.inProgress);
  const target = targets.find((a) => a.number === account);
  const usefulLifeMonths = Math.round(Number(lifeYears || '0') * 12);

  function pick(number: string | null) {
    const entry = targets.find((a) => a.number === number);
    setAccount(number);
    if (entry) {
      setMethod(entry.depreciable ? 'linear' : 'none');
      if (entry.defaultUsefulLifeMonths) {
        setLifeYears(String(Math.round((entry.defaultUsefulLifeMonths / 12) * 10) / 10));
      }
    }
  }

  async function submit() {
    if (!account) {
      setError('Ohne Zielkonto lässt sich nicht umbuchen.');
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await Api.transferFixedAsset({
        assetId: asset.id,
        date,
        account,
        depreciationAccount: target?.depreciationAccount ?? '',
        method,
        usefulLifeMonths: method === 'linear' || method === 'degressive' ? usefulLifeMonths : 0,
        note,
      });
      await onDone('Fertigstellung gebucht — die Abschreibung beginnt.');
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="space-y-4">
      <FormHint
        label="Erklärung zur Fertigstellung"
        line="Erst mit der Betriebsbereitschaft beginnt die Abschreibung — nicht mit der ersten Anzahlung."
      >
        Solange eine Anlage nicht fertig ist, nutzt sie sich nicht ab; sie liegt auf dem Konto der
        Anlagen im Bau und wird nicht abgeschrieben. Mit der Fertigstellung wird sie auf ihr
        endgültiges Anlagekonto umgebucht, und die AfA läuft ab diesem Monat (§ 7 Abs. 1 Satz 4
        EStG). Im Anlagenspiegel erscheint das als Umbuchung: bei der einen Position ab, bei der
        anderen zu.
      </FormHint>

      <div className="grid grid-cols-2 gap-4">
        <Field label="Fertigstellung am" help="Ab diesem Monat wird abgeschrieben.">
          <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
        </Field>
        <Field label="Buchwert" hint={`von Konto ${asset.account}`}>
          <Input value={formatCents(asset.bookValue)} disabled align="right" />
        </Field>
      </div>

      <Field label="Zielkonto" hint={target?.hint}>
        <Combobox
          items={targets.map((a) => ({
            value: a.number,
            label: `${a.number} ${a.name}`,
            meta: a.group,
          }))}
          value={account}
          onValueChange={pick}
          placeholder="Konto suchen"
          emptyText="Kein Konto passt zur Suche."
        />
      </Field>

      <div className="grid grid-cols-2 gap-4">
        <Field label="Abschreibungsmethode">
          <Select
            items={[
              { value: 'linear', label: 'Linear (§ 7 Abs. 1 EStG)' },
              { value: 'degressive', label: 'Degressiv (§ 7 Abs. 2 EStG)' },
              { value: 'none', label: 'Keine planmäßige Abschreibung' },
            ]}
            value={method}
            onValueChange={(next) => setMethod(next as DepreciationMethod)}
          />
        </Field>
        {(method === 'linear' || method === 'degressive') && (
          <Field
            label="Nutzungsdauer in Jahren"
            hint={
              usefulLifeMonths > 0
                ? `${usefulLifeMonths} Monate ab der Fertigstellung`
                : target?.usefulLifeSource
            }
          >
            <Input
              type="number"
              min={0}
              step={0.5}
              align="right"
              value={lifeYears}
              onChange={(e) => setLifeYears(e.target.value)}
            />
          </Field>
        )}
      </div>

      <Field label="Notiz" optional>
        <Input value={note} onChange={(e) => setNote(e.target.value)} />
      </Field>

      <FormError message={error} />

      <div className="flex justify-end">
        <Button
          variant="primary"
          loading={busy}
          disabled={!account || writeLock.locked}
          title={writeLock.hint}
          onClick={submit}
        >
          Fertigstellung buchen
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
  const writeLock = useWriteLock();
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
  // Nur Finanzanlagen gehen anteilig ab — eine Tranche Anteile, die Tilgung
  // einer Ausleihung. Ein halber Pkw geht nicht ab.
  const partialPossible = asset.class === 'financial';
  const [partial, setPartial] = useState(false);
  const [costShareText, setCostShareText] = useState('');
  const [quantityText, setQuantityText] = useState('');
  // Wo Stücke geführt werden, ist die Stückzahl die Vorgabe und der Betrag das
  // Ergebnis. Beide Felder zugleich anzubieten hieße, den Nutzer zwei Wege
  // rechnen zu lassen, die auseinanderlaufen können.
  const tracksUnits = (asset.unitsHeld ?? 0) > 0;
  // Getilgt wird eine Ausleihung. Eine Beteiligung und ein Wertpapier werden
  // verkauft — der Katalog sagt, welches Konto welches ist.
  const isLoan = asset.class === 'financial' && LOAN_ACCOUNTS.includes(asset.account);
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
        <Field
          label="Art des Abgangs"
          hint={request.kind === 'repayment' ? 'kein Umsatz, kein Erlöskonto' : undefined}
          help={
            isLoan
              ? 'Eine Tilgung ist kein Verkauf: zurückgezahlt wird, was ausgeliehen wurde. Zum Buchwert entsteht dabei weder Erlös noch Buchgewinn — die Buchung ist Geld an Ausleihung.'
              : undefined
          }
        >
          <Select
            items={[
              { value: 'sale', label: 'Verkauf' },
              ...(isLoan ? [{ value: 'repayment', label: 'Tilgung' }] : []),
              { value: 'scrapped', label: 'Verschrottung ohne Erlös' },
            ]}
            value={request.kind}
            onValueChange={(next) => {
              if (next === 'scrapped') setProceedsText('');
              set({ kind: next as DisposalKind, proceeds: next === 'scrapped' ? 0 : request.proceeds });
            }}
          />
        </Field>
        <Field
          label={request.kind === 'repayment' ? 'Rückzahlung' : 'Erlös'}
          hint={request.kind === 'repayment' ? 'was zurückfließt' : 'netto'}
          disabled={request.kind === 'scrapped'}
        >
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

      {partialPossible && (
        <div className="grid grid-cols-2 gap-4">
          <div className="flex items-end pb-2">
            <Checkbox
              checked={partial}
              onCheckedChange={(checked) => {
                const next = Boolean(checked);
                setPartial(next);
                if (!next) {
                  setCostShareText('');
                  setQuantityText('');
                  set({ costShare: 0, quantity: 0 });
                }
              }}
              label="Nur ein Teil geht ab"
              hint="Eine Tranche von Anteilen, die Tilgung einer Ausleihung."
            />
          </div>
          {partial && tracksUnits ? (
            <Field
              label="Abgehende Stückzahl"
              hint={
                preview?.quantityShare
                  ? `von ${formatUnits(asset.unitsHeld ?? 0)} · Rest ${formatUnits(preview.unitsRemaining ?? 0)}`
                  : `von ${formatUnits(asset.unitsHeld ?? 0)} im Bestand`
              }
              help="Verkauft wird eine Tranche, kein Betrag. Den Anteil der Anschaffungskosten rechnet Buchfink daraus — samt der Abschreibungen, die im selben Verhältnis mit hinauswandern."
            >
              <Input
                type="number"
                min={0}
                step="any"
                align="right"
                value={quantityText}
                onChange={(e) => {
                  setQuantityText(e.target.value);
                  set({ quantity: Math.round(Number(e.target.value || '0') * UNIT_SCALE) });
                }}
              />
            </Field>
          ) : partial ? (
            <Field
              label="Abgehende Anschaffungskosten"
              hint={`von ${formatCents(asset.cost)}`}
              help="Die kumulierten Abschreibungen wandern im selben Verhältnis mit hinaus."
            >
              <Input
                align="right"
                inputMode="decimal"
                placeholder="0,00"
                value={costShareText}
                onChange={(e) => {
                  setCostShareText(e.target.value);
                  set({ costShare: parseCents(e.target.value) ?? 0 });
                }}
              />
            </Field>
          ) : null}
        </div>
      )}

      {request.kind === 'repayment' && (
        <Notice>
          Eine Rückzahlung ist kein Leistungsaustausch: sie ist nicht steuerbar, nicht bloß
          steuerfrei. Über ein Erlöskonto gebucht stünde in der Gewinn- und Verlustrechnung ein
          Umsatz, den es nie gab.
        </Notice>
      )}

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

      {request.kind !== 'scrapped' && (
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
            <Stat
              label={preview.partial ? 'Restbuchwert des Anteils' : 'Restbuchwert'}
              value={formatCents(preview.bookValue)}
              context={
                preview.partial
                  ? `${formatCents(preview.costShare)} AHK · ${formatCents(preview.depreciationShare)} AfA`
                  : undefined
              }
            />
            <Stat
              label={preview.result >= 0 ? 'Buchgewinn' : 'Buchverlust'}
              value={formatCents(Math.abs(preview.result))}
              tone={preview.result >= 0 ? 'positive' : 'negative'}
            />
            <Stat label="Zahlbetrag" value={formatCents(preview.gross)} context={`darin ${formatCents(preview.tax)} USt`} />
          </StatRow>

          <p className="text-caption text-ink-subtle mb-4">{preview.accounts.explanation}</p>

          {preview.investment && (
            <div className="mb-4">
              <InvestmentNote note={preview.investment} />
            </div>
          )}

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
        <Button
          variant="primary"
          loading={busy}
          disabled={Boolean(previewError) || writeLock.locked}
          title={writeLock.hint}
          onClick={submit}
        >
          {preview?.partial ? 'Teilabgang buchen' : 'Abgang buchen'}
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
