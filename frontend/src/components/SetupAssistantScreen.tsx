import React, { useState } from 'react';
import {
  ArrowLeft,
  ArrowRight,
  Check,
  Database,
  FolderOpen,
  PlusCircle,
  Scale,
  ShieldCheck,
} from 'lucide-react';
import { CompanySettings } from '../types';
import { Api } from '../services/api';
import { GermanFlag } from './GermanFlag';
import { HelpPopover, SHELL_BUTTON, SHELL_CONTROL, SHELL_PANEL, cn } from './ui';

interface SetupAssistantScreenProps {
  onSetupCompleted: () => void;
  onCancel?: () => void;
  isAdditionalTenant?: boolean;
}

const STEP_TITLES: Record<number, string> = {
  1: 'Speicherort',
  2: 'Verschlüsselung',
  3: 'Unternehmen und Steuern',
  4: 'Bankverbindung',
};

const LEGAL_FORMS = [
  'Einzelunternehmen',
  'UG (haftungsbeschränkt)',
  'GmbH',
  'GbR',
  'GmbH & Co. KG',
  'AG',
  'Sonstige',
];

/** Feld auf der Schale. Gleiche Anordnung wie `Field`, dunkle Rollen (§16). */
const ShellField: React.FC<{
  label: string;
  hint?: string;
  className?: string;
  children: React.ReactNode;
}> = ({ label, hint, className, children }) => (
  <label className={cn('flex flex-col gap-1 min-w-0', className)}>
    <span className="text-label text-shell-text-muted">{label}</span>
    {children}
    {hint && <span className="text-caption text-shell-text-muted">{hint}</span>}
  </label>
);

/**
 * Der Einrichtungsassistent. Vier Schritte, je einer pro Entscheidung, und auf
 * jedem Schirm genau eine Primäraktion (§8.2). Er gehört zur Schale (§16) und
 * steht deshalb auf dunklem Grund.
 */
export const SetupAssistantScreen: React.FC<SetupAssistantScreenProps> = ({
  onSetupCompleted,
  onCancel,
  isAdditionalTenant = false,
}) => {
  const currentYear = new Date().getFullYear();
  const [setupChoice, setSetupChoice] = useState<'new' | 'existing' | null>(null);
  const [step, setStep] = useState<1 | 2 | 3 | 4>(1);

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
  });

  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const patch = (next: Partial<CompanySettings>) => setSettings({ ...settings, ...next });

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

  async function finish() {
    setSubmitting(true);
    setError(null);
    try {
      // Die Verschlüsselung entsteht im Hintergrund über den Schlüsselbund des
      // Betriebssystems. Der Recovery-Schlüssel wird danach in den
      // Einstellungen exportiert.
      await Api.setupApplication(dataDir, settings);
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
                    Schritt {step} von 4
                  </span>
                  <h2 className="text-heading text-white mt-1">{STEP_TITLES[step]}</h2>
                </div>
                <div className="flex items-center gap-1.5 pb-1" aria-hidden="true">
                  {[1, 2, 3, 4].map((s) => (
                    <span
                      key={s}
                      className={cn(
                        'h-0.5 rounded-full transition-all duration-180 ease-quiet',
                        step === s ? 'w-6 bg-accent-light' : s < step ? 'w-2 bg-accent' : 'w-2 bg-shell-line',
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

                    <ShellField
                      label="USt-Voranmeldezeitraum"
                      hint="Vierteljährlich ist der Regelfall, monatlich gilt bei Neugründung."
                    >
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

                {step === 4 && (
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
                  onClick={() => (step > 1 ? setStep((s) => (s - 1) as 1 | 2 | 3) : setSetupChoice(null))}
                  className={SHELL_BUTTON.quiet}
                >
                  <ArrowLeft className="w-4 h-4" strokeWidth={1.5} />
                  Zurück
                </button>

                {step < 4 ? (
                  <button
                    type="button"
                    disabled={step === 3 && !settings.companyName.trim()}
                    onClick={() => setStep((s) => (s + 1) as 2 | 3 | 4)}
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
