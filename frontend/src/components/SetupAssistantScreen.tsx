import React, { useEffect, useState } from 'react';
import {
  ArrowLeft,
  ArrowRight,
  Check,
  Database,
  FolderOpen,
  Plus,
  PlusCircle,
  Scale,
  ShieldCheck,
  Trash2,
} from 'lucide-react';
import { CompanySettings, ContributionKind, FoundationRules } from '../types';
import { Api } from '../services/api';
import { formatCents, parseCents } from '../utils/formatters';
import { GermanFlag } from './GermanFlag';
import { HelpPopover, SHELL_BUTTON, SHELL_CONTROL, SHELL_PANEL, cn } from './ui';

interface SetupAssistantScreenProps {
  onSetupCompleted: () => void;
  onCancel?: () => void;
  isAdditionalTenant?: boolean;
}

/**
 * Die Rechtsformen der Ersteinrichtung.
 *
 * Sie stehen hier als Liste und nicht als Abfrage aus der Bridge: die
 * Einrichtung läuft, bevor ein Mandant existiert. Die Schreibweisen müssen zu
 * `domain.LegalFormCatalog` passen — aus ihnen leitet Buchfink später die
 * Anlegerstellung für § 20 InvStG ab, und eine abweichende Schreibweise fiele
 * dort aus dem Katalog.
 */
const LEGAL_FORMS = [
  'Einzelunternehmen',
  'Eingetragener Kaufmann (e. K.)',
  'Freiberufliche Praxis',
  'GbR',
  'OHG',
  'KG',
  'GmbH & Co. KG',
  'Partnerschaftsgesellschaft',
  'UG (haftungsbeschränkt)',
  'GmbH',
  'AG',
  'SE',
  'eG',
  'e. V.',
  'Stiftung',
  'Sonstige',
];

/** Eine Zeile der Gesellschafterliste, solange sie in der Maske steht. */
interface ShareholderDraft {
  name: string;
  shareCapital: string;
  paidIn: string;
  kind: ContributionKind;
}

const emptyShareholder = (): ShareholderDraft => ({
  name: '',
  shareCapital: '',
  paidIn: '',
  kind: 'cash',
});

/**
 * Feld auf der Schale. Gleiche Anordnung wie `Field`, dunkle Rollen (§16).
 *
 * `labelHidden` blendet die Beschriftung aus, ohne sie wegzulassen: In einer
 * Liste gleichartiger Zeilen steht sie nur über der ersten, jede weitere braucht
 * sie trotzdem — für die Vorlesesoftware und damit die Zeilen gleich hoch
 * bleiben.
 */
const ShellField: React.FC<{
  label: string;
  hint?: string;
  labelHidden?: boolean;
  className?: string;
  children: React.ReactNode;
}> = ({ label, hint, labelHidden = false, className, children }) => (
  <label className={cn('flex flex-col gap-1 min-w-0', className)}>
    <span className={cn('text-label text-shell-text-muted', labelHidden && 'sr-only')}>{label}</span>
    {children}
    {hint && <span className="text-caption text-shell-text-muted">{hint}</span>}
  </label>
);

/**
 * Der Einrichtungsassistent. Je Schritt eine Entscheidung, und auf jedem Schirm
 * genau eine Primäraktion (§8.2). Er gehört zur Schale (§16) und steht deshalb
 * auf dunklem Grund.
 *
 * Die Zahl der Schritte hängt an der Rechtsform: Nur eine Kapitalgesellschaft
 * durchläuft eine Vorgesellschaft, und nur dort gibt es eine Unterbilanzhaftung,
 * die Buchfink von Anfang an mitrechnen muss.
 */
export const SetupAssistantScreen: React.FC<SetupAssistantScreenProps> = ({
  onSetupCompleted,
  onCancel,
  isAdditionalTenant = false,
}) => {
  const currentYear = new Date().getFullYear();
  const [setupChoice, setSetupChoice] = useState<'new' | 'existing' | null>(null);
  const [step, setStep] = useState(1);

  const [dataDir, setDataDir] = useState('~/.buchfink/data');
  const [existingDbPath, setExistingDbPath] = useState('');

  const [settings, setSettings] = useState<CompanySettings>({
    companyName: '',
    legalForm: 'GmbH',
    fiscalYear: currentYear,
    fiscalYearStartMonth: 1,
    taxNumber: '',
    vatId: '',
    taxOffice: '',
    iban: '',
    bic: '',
    bankName: '',
    street: '',
    zipCity: '',
    country: 'Deutschland',
    currency: 'EUR',
    skr: 'SKR04',
    vatPeriod: 'quarter',
    taxationType: 'SOLL',
    // Leer heißt: die Anlegerstellung für § 20 InvStG folgt aus der
    // Rechtsform. Gefragt wird sie nur, wo diese sie offen lässt.
    investorOverride: '',
  });

  /**
   * Die Kapitalaufbringungsregeln kommen aus dem Backend, nicht aus einer
   * zweiten Liste hier: Mindestkapital und Einzahlungsquote sind Recht, und
   * Recht gehört an eine Stelle. Der Aufruf braucht keinen Mandanten — er liest
   * einen Katalog, keine Datenbank. Bleibt er ohne Antwort, läuft die
   * Einrichtung ohne Gründungsschritt weiter; ohne Backend ließe sie sich
   * ohnehin nicht abschließen.
   */
  const [foundationRules, setFoundationRules] = useState<FoundationRules[]>([]);
  useEffect(() => {
    Api.getFoundationRules()
      .then(setFoundationRules)
      .catch(() => setFoundationRules([]));
  }, []);

  const rules = foundationRules.find((r) => r.legalForm === settings.legalForm) ?? null;
  const isCapitalCompany = rules !== null;

  const [isFoundingCase, setIsFoundingCase] = useState<boolean | null>(null);
  const [notarizedOn, setNotarizedOn] = useState('');
  const [registeredOn, setRegisteredOn] = useState('');
  const [registerCourt, setRegisterCourt] = useState('');
  const [registerNumber, setRegisterNumber] = useState('');
  const [shareCapital, setShareCapital] = useState('');
  const [foundationCostCap, setFoundationCostCap] = useState('');
  const [shareholders, setShareholders] = useState<ShareholderDraft[]>([emptyShareholder()]);
  const [vatReason, setVatReason] = useState('');

  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const patch = (next: Partial<CompanySettings>) => setSettings({ ...settings, ...next });

  // Die Schritte des Neuanlage-Wegs. Der Gründungsschritt steht zwischen
  // Stammdaten und Bankverbindung, weil er die Rechtsform aus dem Schritt davor
  // braucht — und weil aus dem Beurkundungsdatum das Rumpfgeschäftsjahr folgt.
  const stepTitles = [
    'Speicherort',
    'Verschlüsselung',
    'Unternehmen und Steuern',
    ...(isCapitalCompany ? ['Gründung'] : []),
    'Bankverbindung',
  ];
  const stepCount = stepTitles.length;
  const currentTitle = stepTitles[step - 1] ?? '';
  const isFoundingStep = isCapitalCompany && step === 4;

  async function pickDataDirectory() {
    try {
      const selected = await Api.selectDirectoryDialog('Buchfink Datenordner auswählen');
      if (selected) setDataDir(selected);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function pickDatabaseFile() {
    try {
      const selected = await Api.selectDatabaseFileDialog('Buchfink Buchhaltungsdatei auswählen');
      if (selected) setExistingDbPath(selected);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  /**
   * Aus dem Beurkundungsdatum folgen zwei Angaben, die sonst zu raten wären:
   * das Rumpfgeschäftsjahr und der Voranmeldungszeitraum. Den zweiten
   * beantwortet das Backend, weil § 18 Abs. 2 UStG dafür ein Stichjahr hat.
   */
  async function applyFoundingDate(date: string) {
    setNotarizedOn(date);
    if (date.length !== 10) return;
    const year = Number(date.slice(0, 4));
    if (!Number.isFinite(year) || year < 1900) return;

    setSettings((prev) => ({ ...prev, fiscalYear: year }));
    try {
      const recommendation = await Api.getRecommendedVatPeriod(year);
      setSettings((prev) => ({
        ...prev,
        fiscalYear: year,
        vatPeriod: recommendation.period as CompanySettings['vatPeriod'],
      }));
      setVatReason(recommendation.reason);
    } catch {
      // Ohne Antwort bleibt die Auswahl aus Schritt 3 stehen.
    }
  }

  const subscribedCapital = shareholders.reduce(
    (sum, s) => sum + (parseCents(s.shareCapital) ?? 0),
    0
  );
  const capitalTarget = parseCents(shareCapital) ?? 0;
  const paidInTotal = shareholders.reduce((sum, s) => sum + (parseCents(s.paidIn) ?? 0), 0);

  /**
   * Was die Anmeldung verlangt, gerechnet wie im Backend: ein Anteil je
   * Geschäftsanteil, eine Untergrenze für die Summe. Hier steht es nur, um es
   * schon während der Eingabe zu zeigen — verbindlich prüft der Dienst.
   */
  const requiredPaidIn = (() => {
    if (!rules) return 0;
    const perShare = shareholders.reduce((sum, s) => {
      const share = parseCents(s.shareCapital) ?? 0;
      if (s.kind === 'kind') return sum + share;
      return sum + Math.ceil(share * rules.paidInPerShareQuota);
    }, 0);
    const floor = rules.paidInFloorIsFullCapital ? capitalTarget : rules.paidInFloor;
    return Math.max(perShare, floor);
  })();

  const capitalMatches = capitalTarget > 0 && subscribedCapital === capitalTarget;
  const foundingComplete =
    isFoundingCase === false ||
    (isFoundingCase === true &&
      notarizedOn.length === 10 &&
      capitalMatches &&
      shareholders.every((s) => s.name.trim() !== '' && (parseCents(s.shareCapital) ?? 0) > 0));

  function patchShareholder(index: number, next: Partial<ShareholderDraft>) {
    setShareholders((prev) => prev.map((s, i) => (i === index ? { ...s, ...next } : s)));
  }

  async function finish() {
    setSubmitting(true);
    setError(null);
    try {
      // Die Verschlüsselung entsteht im Hintergrund über den Schlüsselbund des
      // Betriebssystems. Der Recovery-Schlüssel wird danach in den
      // Einstellungen exportiert.
      await Api.setupApplication(dataDir, settings);

      // Die Gründung erst danach: sie gehört in die Datenbank des Mandanten,
      // den der Aufruf oben gerade angelegt hat.
      if (isCapitalCompany && isFoundingCase && notarizedOn.length === 10) {
        await Api.saveFoundation({
          notarizedOn,
          registeredOn: registeredOn.length === 10 ? registeredOn : '',
          registerCourt: registerCourt.trim(),
          registerNumber: registerNumber.trim(),
          shareCapital: capitalTarget,
          foundationCostCap: parseCents(foundationCostCap) ?? 0,
          shareholders: shareholders.map((s) => ({
            id: 0,
            foundationId: 0,
            name: s.name.trim(),
            shareCapital: parseCents(s.shareCapital) ?? 0,
            paidIn: parseCents(s.paidIn) ?? 0,
            kind: s.kind,
          })),
        });
      }
      onSetupCompleted();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setSubmitting(false);
    }
  }

  async function loadExisting(e: React.FormEvent) {
    e.preventDefault();
    if (!existingDbPath.trim()) return;
    setSubmitting(true);
    setError(null);
    try {
      await Api.loadExistingDatabase(existingDbPath.trim());
      onSetupCompleted();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="relative min-h-screen flex flex-col overflow-y-auto bg-shell-deep text-shell-text">
      <div
        className="absolute inset-0 bg-cover bg-center pointer-events-none"
        style={{ backgroundImage: "url('/bg-startupscreen_unsplash-steven-kamenar.jpg')" }}
      />
      <div className="absolute inset-0 bg-shell-deep/85 pointer-events-none" />
      <div className="absolute inset-0 bg-gradient-to-t from-shell-deep to-transparent pointer-events-none" />

      <div className="relative z-10 w-full max-w-2xl mx-auto px-6 py-12 flex-1 flex flex-col justify-center gap-6">
        <header className="flex items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <span className="relative shrink-0">
              <img
                src="/buchfink-logo.svg"
                alt=""
                className="w-11 h-11 rounded-control border border-shell-line bg-shell-raised p-1.5"
              />
              <span className="absolute -bottom-1 -right-1">
                <GermanFlag className="w-4 h-3 rounded-[2px] border border-shell-deep" />
              </span>
            </span>
            <span>
              <h1 className="text-heading text-white">Buchfink</h1>
              <p className="text-caption text-shell-text-muted">
                {isAdditionalTenant ? 'Weiteren Mandanten hinzufügen' : 'Einrichtung'}
              </p>
            </span>
          </div>

          {isAdditionalTenant && onCancel && (
            <button type="button" onClick={onCancel} className={SHELL_BUTTON.quiet}>
              Abbrechen
            </button>
          )}
        </header>

        <div className={cn(SHELL_PANEL, 'p-7')}>
          {error && (
            <p className="mb-5 rounded-control border border-negative/50 bg-negative/15 px-4 py-3 text-body text-shell-negative">
              {error}
            </p>
          )}

          {setupChoice === null ? (
            <>
              <h2 className="text-heading text-white">Willkommen bei Buchfink</h2>
              <p className="flex items-center text-caption text-shell-text-muted mt-1">
                Für Unternehmen mit doppelter Buchführung
                <HelpPopover
                  label="Erklärung zum Anwendungsbereich"
                  className="text-shell-text-muted hover:text-shell-text data-[popup-open]:text-shell-text"
                >
                  Buchfink ist für Unternehmen gebaut, die zur doppelten Buchführung und
                  Bilanzierung verpflichtet sind — UG, GmbH, AG, bilanzierende Kaufleute. Für die
                  Einnahmen-Überschuss-Rechnung kleiner Selbstständiger und Freiberufler ist es
                  nicht geeignet.
                </HelpPopover>
              </p>

              <div className="mt-5 divide-y divide-shell-line border-t border-shell-line">
                <button
                  type="button"
                  onClick={() => setSetupChoice('new')}
                  className="group w-full flex items-start gap-3 py-4 text-left transition-colors duration-120 ease-quiet cursor-pointer"
                >
                  <PlusCircle className="w-5 h-5 mt-0.5 shrink-0 text-accent-light" strokeWidth={1.5} />
                  <span className="min-w-0 flex-1">
                    <span className="block text-body text-white">Neue Buchhaltung anlegen</span>
                    <span className="block text-caption text-shell-text-muted mt-0.5">
                      Kontenrahmen SKR04, Geschäftsjahr {currentYear}, lokaler Speicherort.
                    </span>
                  </span>
                  <ArrowRight
                    className="w-4 h-4 mt-0.5 shrink-0 text-shell-text-muted group-hover:text-accent-soft transition-all
                               duration-120 ease-quiet group-hover:translate-x-0.5"
                    strokeWidth={1.5}
                  />
                </button>

                <button
                  type="button"
                  onClick={() => setSetupChoice('existing')}
                  className="group w-full flex items-start gap-3 py-4 text-left transition-colors duration-120 ease-quiet cursor-pointer"
                >
                  <Database className="w-5 h-5 mt-0.5 shrink-0 text-shell-text-muted" strokeWidth={1.5} />
                  <span className="min-w-0 flex-1">
                    <span className="block text-body text-white">Bestehende Buchhaltung öffnen</span>
                    <span className="block text-caption text-shell-text-muted mt-0.5">
                      Eine vorhandene Buchfink-Datei von der Festplatte oder aus einem Backup.
                    </span>
                  </span>
                  <ArrowRight
                    className="w-4 h-4 mt-0.5 shrink-0 text-shell-text-muted group-hover:text-accent-soft transition-all
                               duration-120 ease-quiet group-hover:translate-x-0.5"
                    strokeWidth={1.5}
                  />
                </button>
              </div>
            </>
          ) : setupChoice === 'new' ? (
            <>
              <div className="flex items-end justify-between gap-4 pb-4 border-b border-shell-line">
                <div>
                  <span className="text-overline uppercase text-accent-light">
                    Schritt {step} von {stepCount}
                  </span>
                  <h2 className="text-heading text-white mt-1">{currentTitle}</h2>
                </div>
                <div className="flex items-center gap-1.5 pb-1" aria-hidden="true">
                  {stepTitles.map((title, index) => (
                    <span
                      key={title}
                      className={cn(
                        'h-0.5 rounded-full transition-all duration-180 ease-quiet',
                        step === index + 1
                          ? 'w-6 bg-accent-light'
                          : index + 1 < step
                            ? 'w-2 bg-accent'
                            : 'w-2 bg-shell-line'
                      )}
                    />
                  ))}
                </div>
              </div>

              <div className="py-6">
                {step === 1 && (
                  <div className="flex flex-col gap-4">
                    <p className="text-body text-shell-text-muted">
                      Buchungsdaten und Belege liegen auf dieser Festplatte, nicht in einer Cloud.
                    </p>
                    <ShellField label="Ordner für Buchungsdaten und Belege">
                      <span className="flex gap-2">
                        <input
                          type="text"
                          value={dataDir}
                          onChange={(e) => setDataDir(e.target.value)}
                          className={cn(SHELL_CONTROL, 'code-num')}
                        />
                        <button
                          type="button"
                          onClick={() => void pickDataDirectory()}
                          className={cn(SHELL_BUTTON.secondary, 'shrink-0')}
                        >
                          <FolderOpen className="w-4 h-4" strokeWidth={1.5} />
                          Wählen
                        </button>
                      </span>
                    </ShellField>
                  </div>
                )}

                {step === 2 && (
                  <div className="flex flex-col gap-4">
                    <p className="text-body text-shell-text-muted">
                      Buchungstexte, Verwendungszwecke und Kontaktdaten werden mit AES-256
                      verschlüsselt. Der Schlüssel liegt im Schlüsselbund des Betriebssystems, ein
                      Passwort ist nicht zu merken.
                    </p>
                    <p className="rounded-control border border-attention/50 bg-attention/15 px-4 py-3">
                      <span className="flex items-center text-label text-attention-line">
                        Recovery-Schlüssel gleich danach exportieren
                        <HelpPopover
                          label="Erklärung zum Recovery-Schlüssel"
                          className="text-shell-text-muted hover:text-shell-text data-[popup-open]:text-shell-text"
                        >
                          Geht dieser Rechner verloren, ist der Schlüsselbund weg und die
                          verschlüsselten Daten sind ohne Recovery-Datei unwiederbringlich — auch
                          aus einem Backup. Die Datei gehört an einen anderen Ort als das
                          Datenbackup.
                        </HelpPopover>
                      </span>
                      <span className="block text-body text-shell-text-muted mt-1">
                        In den Einstellungen unter Speicherort und Schlüssel.
                      </span>
                    </p>
                  </div>
                )}

                {step === 3 && (
                  <div className="flex flex-col gap-4">
                    <ShellField label="Firmen- oder Inhabername">
                      <input
                        type="text"
                        placeholder="Musterfirma GmbH"
                        value={settings.companyName}
                        onChange={(e) => patch({ companyName: e.target.value })}
                        className={SHELL_CONTROL}
                      />
                    </ShellField>

                    <div className="grid grid-cols-2 gap-4">
                      <ShellField label="Rechtsform">
                        <select
                          value={settings.legalForm}
                          onChange={(e) => patch({ legalForm: e.target.value })}
                          className={SHELL_CONTROL}
                        >
                          {LEGAL_FORMS.map((form) => (
                            <option key={form} value={form}>
                              {form}
                            </option>
                          ))}
                        </select>
                      </ShellField>
                      <ShellField label="Erstes Geschäftsjahr">
                        <input
                          type="number"
                          value={settings.fiscalYear}
                          onChange={(e) => patch({ fiscalYear: Number(e.target.value) })}
                          className={cn(SHELL_CONTROL, 'num')}
                        />
                      </ShellField>
                    </div>

                    <div className="grid grid-cols-2 gap-4">
                      <ShellField label="Steuernummer">
                        <input
                          type="text"
                          placeholder="12/345/67890"
                          value={settings.taxNumber}
                          onChange={(e) => patch({ taxNumber: e.target.value })}
                          className={cn(SHELL_CONTROL, 'code-num')}
                        />
                      </ShellField>
                      <ShellField label="USt-IdNr. · optional">
                        <input
                          type="text"
                          placeholder="DE123456789"
                          value={settings.vatId}
                          onChange={(e) => patch({ vatId: e.target.value })}
                          className={cn(SHELL_CONTROL, 'code-num')}
                        />
                      </ShellField>
                    </div>

                    <ShellField label="USt-Voranmeldezeitraum">
                      <select
                        value={settings.vatPeriod || 'quarter'}
                        onChange={(e) =>
                          patch({ vatPeriod: e.target.value as CompanySettings['vatPeriod'] })
                        }
                        className={SHELL_CONTROL}
                      >
                        <option value="quarter">Vierteljährlich</option>
                        <option value="month">Monatlich</option>
                        <option value="year">Jährlich, nur die Jahreserklärung</option>
                      </select>
                    </ShellField>

                    <p className="flex items-center text-caption text-shell-text-muted">
                      Gebucht wird nach vereinbarten Entgelten
                      <HelpPopover
                        label="Erklärung zur Besteuerungsart"
                        className="text-shell-text-muted hover:text-shell-text data-[popup-open]:text-shell-text"
                      >
                        Sollversteuerung nach § 16 Abs. 1 Satz 1 UStG: Eine Rechnung wird mit ihrem
                        Datum gebucht, die Zahlung ist ein späterer, eigener Vorgang. Istversteuerung
                        und die Kleinunternehmerregelung nach § 19 UStG unterstützt Buchfink nicht.
                      </HelpPopover>
                    </p>
                  </div>
                )}

                {isFoundingStep && (
                  <div className="flex flex-col gap-4">
                    {isFoundingCase === null ? (
                      <>
                        <p className="flex items-center text-body text-shell-text-muted">
                          Wo steht die {settings.legalForm} heute?
                          <HelpPopover
                            label="Erklärung zur Vorgesellschaft"
                            className="text-shell-text-muted hover:text-shell-text data-[popup-open]:text-shell-text"
                          >
                            Zwischen der notariellen Beurkundung und der Eintragung besteht die
                            Vorgesellschaft. Sie ist bereits buchführungspflichtig, die
                            Haftungsbeschränkung greift aber noch nicht: Wer in ihrem Namen handelt,
                            haftet persönlich (§ 11 Abs. 2 GmbHG), und bleibt das Reinvermögen am
                            Tag der Eintragung hinter dem Stammkapital zurück, schulden die
                            Gesellschafter die Differenz.
                          </HelpPopover>
                        </p>

                        <div className="divide-y divide-shell-line border-t border-shell-line">
                          <button
                            type="button"
                            onClick={() => setIsFoundingCase(true)}
                            className="group w-full flex items-start gap-3 py-4 text-left transition-colors duration-120 ease-quiet cursor-pointer"
                          >
                            <Scale className="w-5 h-5 mt-0.5 shrink-0 text-accent-light" strokeWidth={1.5} />
                            <span className="min-w-0 flex-1">
                              <span className="block text-body text-white">
                                Gerade gegründet oder in Gründung
                              </span>
                              <span className="block text-caption text-shell-text-muted mt-0.5">
                                Buchfink führt die Fristen und rechnet die Unterbilanz mit.
                              </span>
                            </span>
                            <ArrowRight
                              className="w-4 h-4 mt-0.5 shrink-0 text-shell-text-muted group-hover:text-accent-soft transition-all
                                         duration-120 ease-quiet group-hover:translate-x-0.5"
                              strokeWidth={1.5}
                            />
                          </button>

                          <button
                            type="button"
                            onClick={() => setIsFoundingCase(false)}
                            className="group w-full flex items-start gap-3 py-4 text-left transition-colors duration-120 ease-quiet cursor-pointer"
                          >
                            <Check className="w-5 h-5 mt-0.5 shrink-0 text-shell-text-muted" strokeWidth={1.5} />
                            <span className="min-w-0 flex-1">
                              <span className="block text-body text-white">
                                Länger im Handelsregister eingetragen
                              </span>
                              <span className="block text-caption text-shell-text-muted mt-0.5">
                                Kein Gründungsfall, dieser Schritt entfällt.
                              </span>
                            </span>
                            <ArrowRight
                              className="w-4 h-4 mt-0.5 shrink-0 text-shell-text-muted group-hover:text-accent-soft transition-all
                                         duration-120 ease-quiet group-hover:translate-x-0.5"
                              strokeWidth={1.5}
                            />
                          </button>
                        </div>
                      </>
                    ) : isFoundingCase === false ? (
                      <p className="text-body text-shell-text-muted">
                        Kein Gründungsfall. Der Gründungsabschnitt der Steuerfristen bleibt leer.
                      </p>
                    ) : (
                      <>
                        <div className="grid grid-cols-2 gap-4">
                          <ShellField
                            label="Beurkundung des Gesellschaftsvertrags"
                            hint="Setzt Geschäftsjahr und Fristen"
                          >
                            <input
                              type="date"
                              value={notarizedOn}
                              onChange={(e) => void applyFoundingDate(e.target.value)}
                              className={cn(SHELL_CONTROL, 'num')}
                            />
                          </ShellField>
                          <ShellField
                            label={`${rules?.legalForm === 'AG' ? 'Grundkapital' : 'Stammkapital'} laut Satzung`}
                            hint={`Mindestens ${formatCents(rules?.minShareCapital ?? 0)}`}
                          >
                            <input
                              type="text"
                              inputMode="decimal"
                              placeholder={formatCents(rules?.minShareCapital ?? 0)}
                              value={shareCapital}
                              onChange={(e) => setShareCapital(e.target.value)}
                              className={cn(SHELL_CONTROL, 'num')}
                            />
                          </ShellField>
                        </div>

                        {vatReason && (
                          <p className="flex items-center text-caption text-shell-text-muted">
                            Voranmeldung{' '}
                            {settings.vatPeriod === 'month' ? 'monatlich' : 'vierteljährlich'}
                            <HelpPopover
                              label="Erklärung zum Voranmeldezeitraum bei Neugründung"
                              className="text-shell-text-muted hover:text-shell-text data-[popup-open]:text-shell-text"
                            >
                              {vatReason}
                            </HelpPopover>
                          </p>
                        )}

                        <div>
                          <div className="flex items-center justify-between gap-4 pb-2 border-b border-shell-line">
                            <span className="flex items-center text-label text-shell-text-muted">
                              Gesellschafter
                              <HelpPopover
                                label="Erklärung zu den Geschäftsanteilen"
                                className="text-shell-text-muted hover:text-shell-text data-[popup-open]:text-shell-text"
                              >
                                Die Summe der übernommenen Geschäftsanteile muss dem Stammkapital
                                entsprechen (§ 5 Abs. 3 Satz 2 GmbHG). Nach ihr richtet sich später
                                auch, wer welchen Teil einer Unterbilanz trägt.
                              </HelpPopover>
                            </span>
                            <button
                              type="button"
                              onClick={() => setShareholders((prev) => [...prev, emptyShareholder()])}
                              className={cn(SHELL_BUTTON.quiet, 'h-8 px-2')}
                            >
                              <Plus className="w-4 h-4" strokeWidth={1.5} />
                              Gesellschafter hinzufügen
                            </button>
                          </div>

                          <ul className="divide-y divide-shell-line">
                            {shareholders.map((holder, index) => (
                              <li key={index} className="flex items-end gap-2 py-3">
                                <ShellField label="Name" labelHidden={index > 0} className="flex-1">
                                  <input
                                    type="text"
                                    placeholder="Anna Bauer"
                                    value={holder.name}
                                    onChange={(e) => patchShareholder(index, { name: e.target.value })}
                                    className={SHELL_CONTROL}
                                  />
                                </ShellField>
                                <ShellField label="Übernommener Anteil" labelHidden={index > 0} className="w-32">
                                  <input
                                    type="text"
                                    inputMode="decimal"
                                    placeholder="15.000,00"
                                    value={holder.shareCapital}
                                    onChange={(e) =>
                                      patchShareholder(index, { shareCapital: e.target.value })
                                    }
                                    className={cn(SHELL_CONTROL, 'num')}
                                  />
                                </ShellField>
                                <ShellField label="Geleistet" labelHidden={index > 0} className="w-32">
                                  <input
                                    type="text"
                                    inputMode="decimal"
                                    placeholder="7.500,00"
                                    value={holder.paidIn}
                                    onChange={(e) => patchShareholder(index, { paidIn: e.target.value })}
                                    className={cn(SHELL_CONTROL, 'num')}
                                  />
                                </ShellField>
                                {!rules?.cashOnly && (
                                  <ShellField label="Art der Einlage" labelHidden={index > 0} className="w-32">
                                    <select
                                      value={holder.kind}
                                      onChange={(e) =>
                                        patchShareholder(index, {
                                          kind: e.target.value as ContributionKind,
                                        })
                                      }
                                      className={SHELL_CONTROL}
                                    >
                                      <option value="cash">Bar</option>
                                      <option value="kind">Sache</option>
                                    </select>
                                  </ShellField>
                                )}
                                <button
                                  type="button"
                                  disabled={shareholders.length === 1}
                                  onClick={() =>
                                    setShareholders((prev) => prev.filter((_, i) => i !== index))
                                  }
                                  title="Gesellschafter entfernen"
                                  aria-label={`${holder.name || `Gesellschafter ${index + 1}`} entfernen`}
                                  className={cn(SHELL_BUTTON.quiet, 'h-9 w-9 px-0 shrink-0')}
                                >
                                  <Trash2 className="w-4 h-4" strokeWidth={1.5} />
                                </button>
                              </li>
                            ))}
                          </ul>

                          <dl className="flex flex-wrap gap-x-6 gap-y-1 pt-3 border-t border-shell-line">
                            <span className="flex items-baseline gap-2">
                              <dt className="text-caption text-shell-text-muted">Übernommen</dt>
                              <dd
                                className={cn(
                                  'text-body num',
                                  capitalTarget > 0 && !capitalMatches
                                    ? 'text-shell-negative'
                                    : 'text-shell-text'
                                )}
                              >
                                {formatCents(subscribedCapital)}
                              </dd>
                            </span>
                            <span className="flex items-baseline gap-2">
                              <dt className="text-caption text-shell-text-muted">Geleistet</dt>
                              <dd className="text-body num text-shell-text">
                                {formatCents(paidInTotal)}
                              </dd>
                            </span>
                            <span className="flex items-baseline gap-2">
                              <dt className="text-caption text-shell-text-muted">
                                Vor der Anmeldung nötig
                              </dt>
                              <dd
                                className={cn(
                                  'text-body num',
                                  paidInTotal >= requiredPaidIn && requiredPaidIn > 0
                                    ? 'text-shell-positive'
                                    : 'text-shell-text'
                                )}
                              >
                                {formatCents(requiredPaidIn)}
                              </dd>
                            </span>
                          </dl>
                          {capitalTarget > 0 && !capitalMatches && (
                            <p className="mt-2 text-caption text-shell-negative">
                              Die Anteile ergeben {formatCents(subscribedCapital)}, das Kapital
                              beträgt {formatCents(capitalTarget)}.
                            </p>
                          )}
                        </div>

                        <div className="grid grid-cols-2 gap-4">
                          <ShellField
                            label="Gründungsaufwand laut Satzung · optional"
                            hint="Begrenzt die Unterbilanzhaftung"
                          >
                            <input
                              type="text"
                              inputMode="decimal"
                              placeholder="2.500,00"
                              value={foundationCostCap}
                              onChange={(e) => setFoundationCostCap(e.target.value)}
                              className={cn(SHELL_CONTROL, 'num')}
                            />
                          </ShellField>
                          <ShellField label="Eintragung im Handelsregister · falls schon erfolgt">
                            <input
                              type="date"
                              value={registeredOn}
                              onChange={(e) => setRegisteredOn(e.target.value)}
                              className={cn(SHELL_CONTROL, 'num')}
                            />
                          </ShellField>
                        </div>

                        {registeredOn.length === 10 && (
                          <div className="grid grid-cols-2 gap-4">
                            <ShellField label="Registergericht">
                              <input
                                type="text"
                                placeholder="Amtsgericht München"
                                value={registerCourt}
                                onChange={(e) => setRegisterCourt(e.target.value)}
                                className={SHELL_CONTROL}
                              />
                            </ShellField>
                            <ShellField label="Registernummer">
                              <input
                                type="text"
                                placeholder="HRB 123456"
                                value={registerNumber}
                                onChange={(e) => setRegisterNumber(e.target.value)}
                                className={cn(SHELL_CONTROL, 'code-num')}
                              />
                            </ShellField>
                          </div>
                        )}
                      </>
                    )}
                  </div>
                )}

                {step === stepCount && (
                  <div className="flex flex-col gap-4">
                    <ShellField
                      label="IBAN des Geschäftskontos"
                      hint="Wird mit dem Bankkonto 1800 verknüpft."
                    >
                      <input
                        type="text"
                        placeholder="DE89 3704 0044 0532 0130 00"
                        value={settings.iban}
                        onChange={(e) => patch({ iban: e.target.value })}
                        className={cn(SHELL_CONTROL, 'code-num')}
                      />
                    </ShellField>
                    <ShellField label="Bankname">
                      <input
                        type="text"
                        placeholder="Sparkasse, Volksbank, Qonto"
                        value={settings.bankName}
                        onChange={(e) => patch({ bankName: e.target.value })}
                        className={SHELL_CONTROL}
                      />
                    </ShellField>
                    <p className="text-caption text-shell-text-muted">
                      Der Kontenrahmen SKR04 bringt alle gängigen Konten für Erlöse, Aufwendungen,
                      Steuern und Bankverkehr mit.
                    </p>
                  </div>
                )}
              </div>

              <div className="flex items-center justify-between gap-4 pt-4 border-t border-shell-line">
                <button
                  type="button"
                  onClick={() => {
                    if (isFoundingStep && isFoundingCase !== null) {
                      setIsFoundingCase(null);
                      return;
                    }
                    if (step > 1) setStep(step - 1);
                    else setSetupChoice(null);
                  }}
                  className={SHELL_BUTTON.quiet}
                >
                  <ArrowLeft className="w-4 h-4" strokeWidth={1.5} />
                  Zurück
                </button>

                {step < stepCount ? (
                  <button
                    type="button"
                    disabled={
                      (step === 3 && !settings.companyName.trim()) ||
                      (isFoundingStep && !foundingComplete)
                    }
                    onClick={() => setStep(step + 1)}
                    className={SHELL_BUTTON.primary}
                  >
                    Weiter
                    <ArrowRight className="w-4 h-4" strokeWidth={1.5} />
                  </button>
                ) : (
                  <button
                    type="button"
                    disabled={submitting}
                    onClick={() => void finish()}
                    className={SHELL_BUTTON.primary}
                  >
                    <Check className="w-4 h-4" strokeWidth={1.5} />
                    {submitting ? 'Wird eingerichtet …' : 'Buchhaltung anlegen'}
                  </button>
                )}
              </div>
            </>
          ) : (
            <form onSubmit={loadExisting}>
              <h2 className="text-heading text-white pb-4 border-b border-shell-line">
                Bestehende Buchhaltung öffnen
              </h2>

              <div className="py-6">
                <ShellField label="Pfad zur Buchhaltungsdatei">
                  <span className="flex gap-2">
                    <input
                      type="text"
                      placeholder="/Pfad/zur/buchfink-datei"
                      value={existingDbPath}
                      onChange={(e) => setExistingDbPath(e.target.value)}
                      className={cn(SHELL_CONTROL, 'code-num')}
                      required
                    />
                    <button
                      type="button"
                      onClick={() => void pickDatabaseFile()}
                      className={cn(SHELL_BUTTON.secondary, 'shrink-0')}
                    >
                      <FolderOpen className="w-4 h-4" strokeWidth={1.5} />
                      Wählen
                    </button>
                  </span>
                </ShellField>
              </div>

              <div className="flex items-center justify-between gap-4 pt-4 border-t border-shell-line">
                <button
                  type="button"
                  onClick={() => setSetupChoice(null)}
                  className={SHELL_BUTTON.quiet}
                >
                  <ArrowLeft className="w-4 h-4" strokeWidth={1.5} />
                  Zurück
                </button>
                <button
                  type="submit"
                  disabled={submitting || !existingDbPath.trim()}
                  className={SHELL_BUTTON.primary}
                >
                  {submitting ? 'Wird geladen …' : 'Buchhaltung öffnen'}
                </button>
              </div>
            </form>
          )}
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-3 gap-6 pt-6 border-t border-shell-line">
          <p className="flex items-start gap-2.5">
            <ShieldCheck className="w-4 h-4 mt-0.5 shrink-0 text-shell-positive" strokeWidth={1.5} />
            <span>
              <span className="block text-label text-white">Unveränderbar</span>
              <span className="block text-caption text-shell-text-muted mt-0.5">
                Eine Hashkette schützt die Buchungen lückenlos, wie es die GoBD verlangen.
              </span>
            </span>
          </p>
          <p className="flex items-start gap-2.5">
            <FolderOpen className="w-4 h-4 mt-0.5 shrink-0 text-shell-text-muted" strokeWidth={1.5} />
            <span>
              <span className="block text-label text-white">Lokal</span>
              <span className="block text-caption text-shell-text-muted mt-0.5">
                Daten und Schlüssel bleiben auf diesem Rechner. Kein Cloud-Zwang.
              </span>
            </span>
          </p>
          <p className="flex items-start gap-2.5">
            <Scale className="w-4 h-4 mt-0.5 shrink-0 text-accent-light" strokeWidth={1.5} />
            <span>
              <span className="block text-label text-white">Auswertungen</span>
              <span className="block text-caption text-shell-text-muted mt-0.5">
                GuV, Bilanz und E-Bilanz für Finanzamt und Steuerberatung.
              </span>
            </span>
          </p>
        </div>
      </div>
    </div>
  );
};
