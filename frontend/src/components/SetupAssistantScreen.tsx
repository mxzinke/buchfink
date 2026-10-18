import React, { useEffect, useState } from 'react';
import { AlertTriangle, ArrowLeft, ArrowRight, Check, Database, FolderOpen, HardDrive } from 'lucide-react';
import type { CompanySettings, ConnectedApp, FolderCheck, MCPServerStatus } from '../types';
import { Api } from '../services/api';
import { FEDERAL_STATES, LEGAL_FORMS } from '../utils/companyLabels';
import { defaultInbox } from '../utils/inbox';
import { AgentClientCommands } from './AgentClients';
import { GermanFlag } from './GermanFlag';
import { Checkbox, Help, SHELL_BUTTON, SHELL_CONTROL, SHELL_PANEL, cn } from './ui';

interface SetupAssistantScreenProps {
  onSetupCompleted: () => void;
  onCancel?: () => void;
  isAdditionalTenant?: boolean;
}

type Start = 'founding' | 'switch';

/** Ein Pfad, der nach einem Trennzeichen umbricht statt mitten im Ordnernamen. */
function breakable(path: string): React.ReactNode {
  return path.split(/([\\/])/).map((part, index) => (
    <React.Fragment key={index}>
      {part}
      {(part === '/' || part === '\\') && <wbr />}
    </React.Fragment>
  ));
}

/** Ein Ordnername aus der Firma: „Nordlicht Systeme GmbH" → „nordlicht-systeme-gmbh". */
function folderOf(company: string): string {
  const slug = company
    .toLowerCase()
    .replace(/ä/g, 'ae')
    .replace(/ö/g, 'oe')
    .replace(/ü/g, 'ue')
    .replace(/ß/g, 'ss')
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '');
  return `~/Buchfink/${slug || 'mandant'}`;
}

const MONTHS = [
  'Januar',
  'Februar',
  'März',
  'April',
  'Mai',
  'Juni',
  'Juli',
  'August',
  'September',
  'Oktober',
  'November',
  'Dezember',
];

/** Das Geschäftsjahr, in das ein Tag fällt, wie `domain.GetFiscalYearForDate`. */
function fiscalYearOf(date: string, startMonth: number): number {
  const year = Number(date.slice(0, 4));
  const month = Number(date.slice(5, 7));
  return startMonth > 1 && month < startMonth ? year - 1 : year;
}

/** Die Angaben, die die Einrichtung nicht fragt, mit den Voreinstellungen des Backends. */
const DEFAULTS: CompanySettings = {
  companyName: '',
  legalForm: 'GmbH',
  fiscalYear: new Date().getFullYear(),
  fiscalYearStartMonth: 1,
  taxNumber: '',
  vatId: '',
  taxOffice: '',
  iban: '',
  bic: '',
  bankName: '',
  street: '',
  addressAddition: '',
  postalCode: '',
  city: '',
  countryCode: 'DE',
  foundedOn: '',
  notary: '',
  deedNumber: '',
  shareCapital: 0,
  shareholders: [],
  contactName: '',
  contactPhone: '',
  contactEmail: '',
  invoiceNumberFormat: '',
  receiptNumberFormat: '',
  seat: '',
  registerCourt: '',
  registerNumber: '',
  currency: 'EUR',
  skr: 'SKR04',
  vatPeriod: 'unknown',
  taxationType: 'SOLL',
  permanentExtension: false,
  specialPrepayment: 0,
  receiptCaptureDays: 10,
  commitGraceDays: 0,
  invoiceCheckThreshold: 0,
  investorOverride: '',
};

const ShellField: React.FC<{
  label: string;
  hint?: string;
  error?: string;
  className?: string;
  children: React.ReactNode;
}> = ({ label, hint, error, className, children }) => (
  <div className={cn('flex flex-col gap-1 min-w-0', className)}>
    <label className="flex flex-col gap-1 min-w-0">
      <span className="text-label text-shell-text-muted">{label}</span>
      {children}
    </label>
    {error ? (
      <span role="alert" className="text-caption text-shell-negative">
        {error}
      </span>
    ) : (
      hint && <span className="text-caption text-shell-text-muted">{hint}</span>
    )}
  </div>
);

/** Zwei bis drei Wahlmöglichkeiten nebeneinander, auf der Schale. */
function ShellChoice<T extends string>({
  label,
  options,
  value,
  onChange,
}: {
  label: string;
  options: { value: T; label: string; hint?: string }[];
  value: T;
  onChange: (value: T) => void;
}) {
  return (
    <div className="flex flex-col gap-1">
      <span className="text-label text-shell-text-muted">{label}</span>
      <div role="radiogroup" aria-label={label} className="grid grid-cols-1 sm:grid-cols-2 gap-2">
        {options.map((option) => {
          const active = option.value === value;
          return (
            <button
              key={option.value}
              type="button"
              role="radio"
              aria-checked={active}
              onClick={() => onChange(option.value)}
              className={cn(
                'rounded-control border px-3 py-2 text-left transition-colors duration-120 ease-quiet',
                active
                  ? 'border-accent-light bg-shell-raised text-white'
                  : 'border-shell-line text-shell-text hover:bg-shell-raised',
              )}
            >
              <span className="block text-body">{option.label}</span>
              {option.hint && <span className="block text-caption text-shell-text-muted mt-0.5">{option.hint}</span>}
            </button>
          );
        })}
      </div>
    </div>
  );
}

/**
 * Die Einrichtung in zwei Schritten: das Unternehmen mit dem, was die Bücher
 * von Anfang an brauchen, und die Verbindung zu einer KI-App. Alles Weitere
 * erledigt der Agent im Gespräch oder der Mensch später in den Stammdaten.
 */
export const SetupAssistantScreen: React.FC<SetupAssistantScreenProps> = ({
  onSetupCompleted,
  onCancel,
  isAdditionalTenant = false,
}) => {
  const [mode, setMode] = useState<'new' | 'existing'>('new');
  const [step, setStep] = useState<1 | 2>(1);

  const [companyName, setCompanyName] = useState('');
  const [legalForm, setLegalForm] = useState(LEGAL_FORMS[0].value);
  const [start, setStart] = useState<Start>('founding');
  const [startDate, setStartDate] = useState('');
  const [startMonth, setStartMonth] = useState(1);
  const [federalState, setFederalState] = useState('');
  const [nonProfit, setNonProfit] = useState(false);
  const [inLiquidation, setInLiquidation] = useState(false);
  const [fiscalUnity, setFiscalUnity] = useState(false);

  const [dataDir, setDataDir] = useState(folderOf(''));
  const [dirTouched, setDirTouched] = useState(false);
  const [dirOpen, setDirOpen] = useState(false);
  const [dirCheck, setDirCheck] = useState<{ path: string; error: string | null } | null>(null);
  // Ein Synchronisationsdienst über der Ablage; der Hinweis sperrt nichts.
  const [syncCheck, setSyncCheck] = useState<{ path: string; result: FolderCheck } | null>(null);
  // Leer heißt: der Eingang liegt in der Ablage und folgt ihr.
  const [inboxDir, setInboxDir] = useState('');

  const [existingPath, setExistingPath] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [status, setStatus] = useState<MCPServerStatus | null>(null);
  const [connected, setConnected] = useState<ConnectedApp[]>([]);
  const [tenantId, setTenantId] = useState('');

  // Der Ordner folgt der Firma, bis jemand ihn selbst wählt.
  useEffect(() => {
    if (!dirTouched) setDataDir(folderOf(companyName));
  }, [companyName, dirTouched]);

  useEffect(() => {
    if (mode !== 'new' || step !== 1) return;
    let active = true;
    const timer = window.setTimeout(() => {
      Api.validateTenantDirectory(dataDir).then(
        () => active && setDirCheck({ path: dataDir, error: null }),
        (e) => active && setDirCheck({ path: dataDir, error: e instanceof Error ? e.message : String(e) }),
      );
      Api.checkFolder(dataDir)
        .then((result) => active && setSyncCheck({ path: dataDir, result }))
        .catch(() => active && setSyncCheck(null));
    }, 250);
    return () => {
      active = false;
      window.clearTimeout(timer);
    };
  }, [dataDir, mode, step]);

  // Im zweiten Schritt: der Stand des Servers und die Apps, die sich schon
  // angemeldet haben. Die Zustimmung selbst steht als Dialog über dem Schirm.
  useEffect(() => {
    if (step !== 2) return;
    let active = true;
    const poll = () => {
      Api.getMCPServerStatus()
        .then((s) => active && setStatus(s))
        .catch(() => undefined);
      Api.getConnectedApps()
        .then((apps) => active && setConnected(apps.filter((app) => !app.revoked)))
        .catch(() => undefined);
    };
    poll();
    const timer = window.setInterval(poll, 3000);
    return () => {
      active = false;
      window.clearInterval(timer);
    };
  }, [step]);

  const dirError = dirCheck?.path === dataDir ? dirCheck.error : null;
  const dirValid = dirCheck?.path === dataDir && !dirCheck.error;
  const syncWarning = syncCheck?.path === dataDir ? syncCheck.result.warning : '';
  const inboxShown = inboxDir || defaultInbox(dataDir);
  const exclusions = [nonProfit, inLiquidation, fiscalUnity].some(Boolean);

  const dateComplete = /^\d{4}-\d{2}-\d{2}$/.test(startDate);
  const startError =
    dateComplete && start === 'switch' && !startDate.endsWith('-01')
      ? 'Ein Umstieg beginnt am ersten Tag eines Geschäftsjahres.'
      : dateComplete && Number(startDate.slice(0, 4)) > new Date().getFullYear() + 1
        ? 'So weit im Voraus lassen sich keine Bücher anlegen.'
        : '';
  const ready = companyName.trim() !== '' && dateComplete && !startError && federalState !== '' && dirValid;
  // Was „Weiter“ noch sperrt, steht sichtbar neben dem Knopf.
  const missing = [
    companyName.trim() === '' && 'Firma',
    !dateComplete && (start === 'founding' ? 'Tag der Beurkundung' : 'erster Tag in Buchfink'),
    federalState === '' && 'Bundesland',
    dirCheck?.path === dataDir && dirCheck.error && 'ein beschreibbarer Ordner',
  ].filter((item): item is string => Boolean(item));

  async function createTenant() {
    if (!ready) return;
    setSubmitting(true);
    setError(null);
    const month = start === 'switch' ? Number(startDate.slice(5, 7)) : startMonth;
    try {
      await Api.setupApplication(dataDir, inboxDir, {
        ...DEFAULTS,
        companyName: companyName.trim(),
        legalForm,
        federalState,
        fiscalYearStartMonth: month,
        fiscalYear: fiscalYearOf(startDate, month),
        // Bei einer Gründung beginnen die Bücher mit der Beurkundung; das
        // erste Geschäftsjahr richtet sich danach.
        foundedOn: start === 'founding' ? startDate : '',
        nonProfit,
        inLiquidation,
        fiscalUnity,
      });
      const tenant = await Api.getActiveTenant().catch(() => null);
      setTenantId(tenant?.id ?? '');
      setStep(2);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setSubmitting(false);
    }
  }

  async function pickDirectory() {
    try {
      const selected = await Api.selectDirectoryDialog('Ordner für Buchungsdaten und Belege wählen');
      if (selected) {
        setDirTouched(true);
        setDataDir(selected);
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function pickInbox() {
    try {
      const selected = await Api.selectDirectoryDialog('Eingang für Belege, E-Rechnungen und Kontoauszüge wählen');
      if (selected) setInboxDir(selected);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function pickDatabase() {
    try {
      const selected = await Api.selectDatabaseFileDialog('Buchfink-Datei auswählen');
      if (selected) setExistingPath(selected);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function openExisting(e: React.FormEvent) {
    e.preventDefault();
    if (!existingPath.trim()) return;
    setSubmitting(true);
    setError(null);
    try {
      await Api.loadExistingDatabase(existingPath.trim());
      onSetupCompleted();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSubmitting(false);
    }
  }

  const helpTone = 'text-shell-text-muted hover:text-shell-text data-[popup-open]:text-shell-text';
  const tenantName = companyName.trim();
  const connectedHere = connected.filter((app) => app.tenants.some((t) => t.id === tenantId));

  return (
    <div className="relative h-dvh overflow-hidden bg-shell-deep text-shell-text">
      <div
        className="absolute inset-0 bg-cover bg-center pointer-events-none"
        style={{ backgroundImage: "url('/bg-startupscreen_unsplash-steven-kamenar.jpg')" }}
      />
      <div className="absolute inset-0 bg-shell-deep/85 pointer-events-none" />
      <div className="absolute inset-0 bg-gradient-to-t from-shell-deep to-transparent pointer-events-none" />

      <div className="relative z-10 h-full overflow-y-auto">
        <div className="w-full min-h-full max-w-2xl mx-auto px-6 py-12 flex flex-col justify-center gap-6">
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
                  {isAdditionalTenant ? 'Weiteres Unternehmen einrichten' : 'Buchhaltung für GmbH und UG'}
                </p>
              </span>
            </div>
            {isAdditionalTenant && onCancel && step === 1 && (
              <button type="button" onClick={onCancel} className={SHELL_BUTTON.quiet}>
                Abbrechen
              </button>
            )}
          </header>

          <div className={cn(SHELL_PANEL, 'p-7')}>
            {error && (
              <p
                role="alert"
                className="mb-5 rounded-control border border-negative/50 bg-negative/15 px-4 py-3 text-body text-shell-negative"
              >
                {error}
              </p>
            )}

            {mode === 'existing' ? (
              <form onSubmit={openExisting}>
                <h2 className="text-heading text-white pb-4 border-b border-shell-line">
                  Vorhandene Buchhaltung öffnen
                </h2>
                <div className="py-6">
                  <ShellField label="Buchfink-Datei" hint="Etwa aus einer früheren Installation">
                    <span className="flex gap-2">
                      <input
                        type="text"
                        value={existingPath}
                        placeholder="/Pfad/zur/buchfink.db"
                        onChange={(e) => setExistingPath(e.target.value)}
                        className={cn(SHELL_CONTROL, 'code-num')}
                      />
                      <button
                        type="button"
                        onClick={() => void pickDatabase()}
                        className={cn(SHELL_BUTTON.secondary, 'shrink-0')}
                      >
                        <FolderOpen className="w-4 h-4" strokeWidth={1.5} />
                        Wählen
                      </button>
                    </span>
                  </ShellField>
                </div>
                <div className="flex items-center justify-between gap-4 pt-4 border-t border-shell-line">
                  <button type="button" onClick={() => setMode('new')} className={SHELL_BUTTON.quiet}>
                    <ArrowLeft className="w-4 h-4" strokeWidth={1.5} />
                    Zurück
                  </button>
                  <button
                    type="submit"
                    disabled={submitting || !existingPath.trim()}
                    className={SHELL_BUTTON.primary}
                  >
                    Buchhaltung öffnen
                  </button>
                </div>
              </form>
            ) : (
              <>
                <div className="flex items-end justify-between gap-4 pb-4 border-b border-shell-line">
                  <div>
                    <span className="text-overline uppercase text-accent-light">Schritt {step} von 2</span>
                    <h2 className="text-heading text-white mt-1">
                      {step === 1 ? 'Ihr Unternehmen' : 'Claude verbinden'}
                    </h2>
                  </div>
                  <div className="flex items-center gap-1.5 pb-1" aria-hidden="true">
                    <span className={cn('h-0.5 rounded-full', step === 1 ? 'w-6 bg-accent-light' : 'w-2 bg-accent')} />
                    <span className={cn('h-0.5 rounded-full', step === 2 ? 'w-6 bg-accent-light' : 'w-2 bg-shell-line')} />
                  </div>
                </div>

                {step === 1 ? (
                  <div className="py-6 flex flex-col gap-5">
                    <div className="grid grid-cols-1 sm:grid-cols-[minmax(0,1fr)_14rem] gap-4">
                      <ShellField label="Firma" hint="Mit Rechtsformzusatz, etwa GmbH">
                        <input
                          type="text"
                          value={companyName}
                          placeholder="Firma wie im Handelsregister"
                          onChange={(e) => setCompanyName(e.target.value)}
                          className={SHELL_CONTROL}
                        />
                      </ShellField>
                      <ShellField label="Rechtsform">
                        <select value={legalForm} onChange={(e) => setLegalForm(e.target.value)} className={SHELL_CONTROL}>
                          {LEGAL_FORMS.map((form) => (
                            <option key={form.value} value={form.value}>
                              {form.label}
                            </option>
                          ))}
                        </select>
                      </ShellField>
                    </div>

                    <ShellChoice<Start>
                      label="Wie beginnen die Bücher?"
                      value={start}
                      onChange={(next) => {
                        setStart(next);
                        setStartDate('');
                      }}
                      options={[
                        { value: 'founding', label: 'Neu gegründet', hint: 'Die Bücher beginnen mit der Beurkundung' },
                        { value: 'switch', label: 'Umstieg', hint: 'Die Gesellschaft besteht schon' },
                      ]}
                    />

                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                      <ShellField
                        label={start === 'founding' ? 'Tag der Beurkundung' : 'Erster Tag in Buchfink'}
                        hint={
                          start === 'founding'
                            ? 'Starttag der Bücher'
                            : 'Beginn eines Geschäftsjahres, etwa der 1. Januar'
                        }
                        error={startError || undefined}
                      >
                        <input
                          type="date"
                          value={startDate}
                          onChange={(e) => setStartDate(e.target.value)}
                          className={cn(SHELL_CONTROL, 'num')}
                        />
                      </ShellField>
                      {start === 'founding' ? (
                        <ShellField label="Geschäftsjahr" hint="Laut Gesellschaftsvertrag">
                          <select
                            value={startMonth}
                            onChange={(e) => setStartMonth(Number(e.target.value))}
                            className={SHELL_CONTROL}
                          >
                            {MONTHS.map((name, index) => (
                              <option key={name} value={index + 1}>
                                {index === 0 ? 'Kalenderjahr' : `Ab 1. ${name}`}
                              </option>
                            ))}
                          </select>
                        </ShellField>
                      ) : null}
                      <ShellField label="Bundesland" hint="Sitz der Gesellschaft, für Fristen und Steuernummer">
                        <select
                          value={federalState}
                          onChange={(e) => setFederalState(e.target.value)}
                          className={SHELL_CONTROL}
                        >
                          <option value="" disabled>
                            Bitte wählen
                          </option>
                          {FEDERAL_STATES.map((state) => (
                            <option key={state.value} value={state.value}>
                              {state.label}
                            </option>
                          ))}
                        </select>
                      </ShellField>
                    </div>

                    <div className="flex flex-col gap-2">
                      <span className="flex items-center text-label text-shell-text-muted">
                        Trifft eines davon zu?
                        <Help
                          summary="Für diese Gesellschaften rechnet Buchfink weder Jahresabschluss noch Steuerrückstellung."
                          label="Erklärung zu den Ausschlüssen"
                          className={helpTone}
                        >
                          Gemeinnützigkeit verlangt Sphärentrennung und Mittelverwendung, eine Liquidation
                          eigene Abwicklungsabschlüsse, eine Organschaft die Zurechnung des Einkommens zum
                          Organträger. Buchfink bildet das nicht ab. Buchen und auswerten können Sie
                          trotzdem; Abschluss und Erklärungen gehören dann zu einer Steuerberatung.
                        </Help>
                      </span>
                      <Checkbox tone="shell" checked={nonProfit} onCheckedChange={(v) => setNonProfit(Boolean(v))} label="Die Gesellschaft ist gemeinnützig" />
                      <Checkbox tone="shell" checked={inLiquidation} onCheckedChange={(v) => setInLiquidation(Boolean(v))} label="Die Gesellschaft ist in Liquidation" />
                      <Checkbox tone="shell" checked={fiscalUnity} onCheckedChange={(v) => setFiscalUnity(Boolean(v))} label="Die Gesellschaft ist Organgesellschaft einer Organschaft" />
                      {exclusions && (
                        <p className="mt-1 rounded-control border border-negative/50 bg-negative/15 px-4 py-3 text-body text-shell-negative">
                          Das deckt Buchfink nicht ab. Jahresabschluss und Steuererklärungen gehören zu einer
                          Steuerberatung; buchen können Sie trotzdem.
                        </p>
                      )}
                    </div>

                    <div className="flex flex-col gap-3 pt-4 border-t border-shell-line">
                      <p className="flex items-start gap-2.5">
                        <HardDrive className="w-4 h-4 mt-0.5 shrink-0 text-shell-text-muted" strokeWidth={1.5} />
                        <span className="block text-body text-shell-text">
                          Buchfink speichert lokal und unverschlüsselt. Schalten Sie die
                          Festplattenverschlüsselung des Rechners ein: FileVault, BitLocker oder LUKS.
                        </span>
                      </p>
                      <dl className="grid grid-cols-[4.5rem_minmax(0,1fr)] gap-x-3 gap-y-1.5 pl-[26px] text-caption">
                        <dt className="text-shell-text-muted">Ablage</dt>
                        <dd className="min-w-0">
                          <span className="code-num text-shell-text break-words">{breakable(dataDir)}</span>
                          {' · '}
                          <button
                            type="button"
                            onClick={() => setDirOpen((open) => !open)}
                            aria-expanded={dirOpen}
                            className="text-accent-light hover:underline"
                          >
                            Ordner ändern
                          </button>
                          {dirError && <span className="block mt-1 text-shell-negative">{dirError}</span>}
                        </dd>
                        <dt className="flex items-start text-shell-text-muted">
                          Eingang
                          <Help
                            summary="Hier sammeln Sie Belege, E-Rechnungen und Kontoauszüge für Claude."
                            label="Erklärung zum Eingang"
                            className={helpTone}
                          >
                            Legen Sie Dateien in diesen Ordner oder ziehen Sie sie später ins Fenster.
                            Claude legt sie ab, wenn Sie es sagen; danach verschiebt Buchfink sie in den
                            Unterordner Übernommen. Der Eingang darf auch in einem Cloud-Ordner liegen,
                            etwa wenn eine Scanner-App auf dem Telefon dorthin speichert.
                          </Help>
                        </dt>
                        <dd className="min-w-0">
                          <span className="code-num text-shell-text break-words">{breakable(inboxShown)}</span>
                          {' · '}
                          <button
                            type="button"
                            onClick={() => void pickInbox()}
                            className="text-accent-light hover:underline"
                          >
                            Anderen Ordner wählen
                          </button>
                          {inboxDir && (
                            <>
                              {' · '}
                              <button
                                type="button"
                                onClick={() => setInboxDir('')}
                                className="text-accent-light hover:underline"
                              >
                                Zurück in die Ablage
                              </button>
                            </>
                          )}
                        </dd>
                      </dl>
                      {dirOpen && (
                        <span className="flex gap-2">
                          <input
                            type="text"
                            aria-label="Ordner für Buchungsdaten und Belege"
                            value={dataDir}
                            onChange={(e) => {
                              setDirTouched(true);
                              setDataDir(e.target.value);
                            }}
                            className={cn(SHELL_CONTROL, 'code-num')}
                          />
                          <button type="button" onClick={() => void pickDirectory()} className={cn(SHELL_BUTTON.secondary, 'shrink-0')}>
                            <FolderOpen className="w-4 h-4" strokeWidth={1.5} />
                            Wählen
                          </button>
                        </span>
                      )}
                      {syncWarning && (
                        <p role="status" className="flex items-start gap-2.5 text-caption text-shell-attention">
                          <AlertTriangle className="w-4 h-4 shrink-0" strokeWidth={1.5} aria-hidden="true" />
                          <span>{syncWarning}</span>
                        </p>
                      )}
                    </div>
                  </div>
                ) : (
                  <div className="py-6 flex flex-col gap-5">
                    <p className="text-body text-shell-text">
                      {tenantName} ist eingerichtet. Verbinden Sie jetzt Claude oder eine andere KI-App; sie
                      erledigt die Buchhaltung, Sie geben frei.
                    </p>
                    <AgentClientCommands url={status?.url} tone="shell" />
                    <div className="flex items-start gap-2.5 text-caption text-shell-text-muted">
                      {connectedHere.length > 0 ? (
                        <>
                          <Check className="w-4 h-4 mt-px shrink-0 text-shell-positive" strokeWidth={1.5} />
                          <span className="text-shell-positive">
                            Verbunden: {connectedHere.map((app) => app.name).join(', ')}
                          </span>
                        </>
                      ) : (
                        <span className="flex items-center">
                          Meldet sich die App an, fragt Buchfink in diesem Fenster nach Ihrer Zustimmung.
                          <Help
                            summary="Was ein Werkzeug zurückgibt, schickt die App an den Anbieter ihres Sprachmodells."
                            label="Erklärung zum Datenschutz"
                            className={helpTone}
                          >
                            Buchfink läuft auf Ihrem Rechner. Was ein Werkzeug zurückgibt, schickt die KI-App an
                            den Anbieter ihres Sprachmodells: Buchungen, Beträge, Namen von Kunden und
                            Lieferanten, Texte aus Belegen. Klären Sie vorher, ob Ihr Vertrag mit dem Anbieter das
                            abdeckt (Art. 28 DSGVO).
                          </Help>
                        </span>
                      )}
                    </div>
                    {status && !status.running && (
                      <p className="text-caption text-shell-negative">
                        {status.error || 'Der Zugang für KI-Apps läuft nicht.'} Unter Einstellungen lässt sich
                        ein anderer Port wählen.
                      </p>
                    )}
                  </div>
                )}

                <div className="flex items-center justify-between gap-4 pt-4 border-t border-shell-line">
                  {step === 1 ? (
                    <>
                      <button type="button" onClick={() => setMode('existing')} className={SHELL_BUTTON.quiet}>
                        <Database className="w-4 h-4" strokeWidth={1.5} />
                        Vorhandene Buchhaltung öffnen
                      </button>
                      <span className="flex items-center gap-4">
                        {missing.length > 0 && (
                          <span id="einrichtung-fehlt" className="text-caption text-shell-text-muted text-right">
                            {`Es fehlt noch: ${missing.join(', ')}`}
                          </span>
                        )}
                        <button
                          type="button"
                          disabled={!ready || submitting}
                          aria-describedby={missing.length > 0 ? 'einrichtung-fehlt' : undefined}
                          onClick={() => void createTenant()}
                          className={cn(SHELL_BUTTON.primary, 'shrink-0')}
                        >
                          Weiter
                          <ArrowRight className="w-4 h-4" strokeWidth={1.5} />
                        </button>
                      </span>
                    </>
                  ) : (
                    <>
                      <span className="text-caption text-shell-text-muted">
                        {connectedHere.length === 0 ? 'Verbinden geht auch später unter Einstellungen.' : ''}
                      </span>
                      <button type="button" onClick={onSetupCompleted} className={cn(SHELL_BUTTON.primary, 'shrink-0')}>
                        Zur Übersicht
                        <ArrowRight className="w-4 h-4" strokeWidth={1.5} />
                      </button>
                    </>
                  )}
                </div>
              </>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};
