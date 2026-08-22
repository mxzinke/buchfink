import React, { useState } from 'react';
import {
  FolderOpen,
  ArrowRight,
  ArrowLeft,
  CheckCircle2,
  Database,
  PlusCircle,
  FileSearch,
  KeyRound,
  AlertTriangle,
  Info,
  Shield,
  ReceiptText,
  ShieldCheck,
  Scale,
} from 'lucide-react';
import { CompanySettings } from '../types';
import { Api } from '../services/api';
import { GermanFlag } from './GermanFlag';

interface SetupAssistantScreenProps {
  onSetupCompleted: () => void;
  onCancel?: () => void;
  isAdditionalTenant?: boolean;
}

export const SetupAssistantScreen: React.FC<SetupAssistantScreenProps> = ({
  onSetupCompleted,
  onCancel,
  isAdditionalTenant = false,
}) => {
  const currentYear = new Date().getFullYear(); // e.g. 2026
  const [setupChoice, setSetupChoice] = useState<'new' | 'existing' | null>(null);
  const [step, setStep] = useState<1 | 2 | 3 | 4>(1);

  // Form State
  const [dataDir, setDataDir] = useState('~/.buchfink/data');
  const [existingDbPath, setExistingDbPath] = useState('');

  const [companySettings, setCompanySettings] = useState<CompanySettings>({
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
    isSmallBusiness: false,
    vatPeriod: 'quarter',
    taxationType: 'SOLL',
  });

  const [isSubmitting, setIsSubmitting] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  const handlePickDataDirectory = async () => {
    try {
      const selected = await Api.selectDirectoryDialog('Buchfink Datenordner auswählen');
      if (selected) {
        setDataDir(selected);
      }
    } catch (e) {
      console.error(e);
    }
  };

  const handlePickDatabaseFile = async () => {
    try {
      const selected = await Api.selectDatabaseFileDialog('Buchfink Buchhaltungsdatei auswählen');
      if (selected) {
        setExistingDbPath(selected);
      }
    } catch (e) {
      console.error(e);
    }
  };

  const handleFinishWizard = async () => {
    setIsSubmitting(true);
    setErrorMsg(null);
    try {
      // Encryption is provisioned transparently via the OS keychain; the user is
      // guided to export a recovery key afterwards (Step 2 + Settings).
      await Api.setupApplication(dataDir, companySettings);
      onSetupCompleted();
    } catch (e: any) {
      setErrorMsg(e.message || 'Fehler bei der Ersteinrichtung.');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleLoadExisting = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!existingDbPath.trim()) return;
    setIsSubmitting(true);
    setErrorMsg(null);
    try {
      await Api.loadExistingDatabase(existingDbPath.trim());
      onSetupCompleted();
    } catch (e: any) {
      setErrorMsg(e.message || 'Buchhaltungsdatei konnte nicht geladen werden.');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="relative min-h-screen flex flex-col justify-between bg-stone-900 text-stone-100 overflow-y-auto">
      {/* Background with warm atmospheric view */}
      <div
        className="absolute inset-0 bg-cover bg-center opacity-85 pointer-events-none scale-100"
        style={{ backgroundImage: "url('/bg-startupscreen_unsplash-steven-kamenar.jpg')" }}
      />
      <div className="absolute inset-0 bg-gradient-to-t from-[#1A1816]/90 via-[#1F1C1A]/60 to-[#1A1816]/40 pointer-events-none" />

      {/* Main Content Area */}
      <div className="relative z-10 max-w-2xl mx-auto w-full px-6 py-12 flex-1 flex flex-col justify-center space-y-6">
        {/* Brand Header */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <div className="relative">
              <img
                src="/buchfink-logo.svg"
                alt="Buchfink"
                className="w-14 h-14 rounded-2xl bg-white/15 p-1.5 border border-white/20 backdrop-blur-md drop-shadow-lg"
              />
              <div className="absolute -bottom-1 -right-1">
                <GermanFlag className="w-4 h-3 shadow-sm border border-[#1A1816] rounded-xs" />
              </div>
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h1 className="text-2xl font-extrabold text-white tracking-tight drop-shadow-sm">Buchfink</h1>
                <span className="text-xs text-stone-300">Buchhaltung</span>
              </div>
              <p className="text-xs text-stone-200/90 mt-0.5">
                {isAdditionalTenant ? 'Weiteren Mandanten hinzufügen' : 'Ersteinrichtung in wenigen Schritten'}
              </p>
            </div>
          </div>

          {isAdditionalTenant && onCancel && (
            <button
              onClick={onCancel}
              className="px-3.5 py-1.5 text-xs font-medium text-stone-300 hover:text-white bg-stone-800/80 hover:bg-stone-700 rounded-xl border border-white/10 transition-colors"
            >
              Abbrechen
            </button>
          )}
        </div>

        {/* Wizard Box */}
        <div className="bg-[#24211E]/85 backdrop-blur-xl border border-white/15 rounded-2xl p-7 shadow-2xl space-y-6">
          {errorMsg && (
            <div className="p-3 bg-rose-500/20 border border-rose-500/40 text-rose-200 text-xs rounded-xl">
              {errorMsg}
            </div>
          )}

          {/* Initial Choice: New vs Existing */}
          {setupChoice === null ? (
            <div className="space-y-6">
              <div className="space-y-1">
                <h2 className="text-base font-bold text-white">Willkommen bei Buchfink</h2>
                <p className="text-xs text-stone-300">
                  Wählen Sie, wie Sie Ihre Buchhaltung starten möchten:
                </p>
              </div>

              {/* Accounting Scope & Double-entry Note */}
              <div className="p-3.5 bg-amber-500/10 border border-amber-500/30 rounded-xl text-stone-200 text-xs flex items-start gap-2.5">
                <Info className="w-4 h-4 text-amber-300 shrink-0 mt-0.5" />
                <div className="space-y-1">
                  <span className="font-semibold text-amber-200 block">Wichtiger Hinweis zum Anwendungsbereich</span>
                  <p className="text-stone-300 leading-relaxed text-[11px]">
                    Buchfink ist ausschließlich für Unternehmen konzipiert, die zur <strong>doppelten Buchführung und Bilanzierung</strong> verpflichtet sind (z.&nbsp;B. UG, GmbH, AG, bilanzierende Kaufleute). Es ist <strong>nicht für kleine Selbstständige oder Freiberufler</strong> mit einfacher Einnahmen-Überschuss-Rechnung (EÜR) geeignet.
                  </p>
                </div>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {/* Option 1: New Setup */}
                <div
                  onClick={() => setSetupChoice('new')}
                  className="p-5 rounded-2xl bg-[#1D1B19]/75 border-2 border-white/10 hover:border-amber-400 hover:bg-[#1D1B19]/90 transition-all cursor-pointer group space-y-3"
                >
                  <div className="w-10 h-10 rounded-xl bg-amber-500/20 text-amber-300 flex items-center justify-center border border-amber-500/30 group-hover:scale-105 transition-transform">
                    <PlusCircle className="w-5 h-5" />
                  </div>
                  <div>
                    <h3 className="font-semibold text-sm text-white group-hover:text-amber-300 transition-colors">
                      Neue Buchhaltung anlegen
                    </h3>
                    <p className="text-xs text-stone-300 mt-1 leading-snug">
                      Richtet Ihre Buchhaltung für {currentYear} mit Kontenrahmen und sicherem lokalen Speicherort ein.
                    </p>
                  </div>
                  <div className="text-xs font-semibold text-amber-300 flex items-center gap-1 pt-1">
                    Jetzt einrichten &rarr;
                  </div>
                </div>

                {/* Option 2: Load Existing Database */}
                <div
                  onClick={() => setSetupChoice('existing')}
                  className="p-5 rounded-2xl bg-[#1D1B19]/75 border-2 border-white/10 hover:border-amber-400 hover:bg-[#1D1B19]/90 transition-all cursor-pointer group space-y-3"
                >
                  <div className="w-10 h-10 rounded-xl bg-stone-800/80 text-stone-300 flex items-center justify-center border border-stone-700 group-hover:scale-105 transition-transform">
                    <Database className="w-5 h-5" />
                  </div>
                  <div>
                    <h3 className="font-semibold text-sm text-white group-hover:text-amber-300 transition-colors">
                      Bestehende Buchhaltung öffnen
                    </h3>
                    <p className="text-xs text-stone-300 mt-1 leading-snug">
                      Laden Sie eine bereits vorhandene Buchfink-Datei von Ihrer Festplatte oder ein Backup.
                    </p>
                  </div>
                  <div className="text-xs font-semibold text-stone-300 group-hover:text-amber-300 flex items-center gap-1 pt-1">
                    Datei auswählen &rarr;
                  </div>
                </div>
              </div>
            </div>
          ) : setupChoice === 'new' ? (
            /* New Setup Wizard */
            <>
              {/* Step indicator */}
              <div className="flex items-center justify-between border-b border-white/10 pb-4">
                <div>
                  <span className="text-xs font-semibold uppercase tracking-wider text-amber-300">
                    Schritt {step} von 4
                  </span>
                  <h2 className="text-sm font-semibold text-white mt-0.5">
                    {step === 1 && '1. Speicherort festlegen'}
                    {step === 2 && '2. Verschlüsselung'}
                    {step === 3 && '3. Unternehmensdaten & Geschäftsjahr'}
                    {step === 4 && '4. Bankverbindung & Konten'}
                  </h2>
                </div>
                <div className="flex items-center gap-1.5">
                  {[1, 2, 3, 4].map((s) => (
                    <div
                      key={s}
                      className={`h-2 rounded-full transition-all ${
                        step === s ? 'bg-amber-400 w-6' : step > s ? 'bg-amber-400/50 w-2' : 'bg-stone-700 w-2'
                      }`}
                    />
                  ))}
                </div>
              </div>

              {/* Step 1: Storage Location for Data */}
              {step === 1 && (
                <div className="space-y-4 text-xs">
                  <div className="p-3.5 bg-[#1D1B19]/70 rounded-xl border border-white/10 text-stone-300 flex items-start gap-2.5">
                    <Info className="w-4 h-4 text-amber-300 shrink-0 mt-0.5" />
                    <div className="space-y-1">
                      <span className="font-semibold text-white block">Lokale Speicherung auf Ihrem Computer</span>
                      <p className="text-xs leading-relaxed text-stone-300">
                        Buchfink speichert Ihre Buchungsdaten und Belege sicher auf Ihrer lokalen Festplatte. Sie behalten die volle Kontrolle über Ihre Finanzdaten.
                      </p>
                    </div>
                  </div>

                  <div>
                    <label className="font-medium text-stone-200 block mb-1">
                      Ordner für Buchungsdaten & Belege:
                    </label>
                    <div className="flex gap-2">
                      <div className="relative flex-1">
                        <FolderOpen className="w-4 h-4 text-stone-400 absolute left-3 top-1/2 -translate-y-1/2" />
                        <input
                          type="text"
                          value={dataDir}
                          onChange={(e) => setDataDir(e.target.value)}
                          className="w-full pl-9 pr-3 py-2 bg-[#1D1B19]/80 border border-stone-700 rounded-xl text-stone-200 focus:border-amber-400 focus:outline-hidden"
                        />
                      </div>
                      <button
                        type="button"
                        onClick={handlePickDataDirectory}
                        className="px-3 py-2 rounded-xl bg-stone-800 hover:bg-stone-700 text-stone-200 font-medium text-xs border border-stone-700 transition-colors shrink-0 flex items-center gap-1.5"
                      >
                        <FolderOpen className="w-3.5 h-3.5 text-amber-300" />
                        Ordner auswählen...
                      </button>
                    </div>
                  </div>
                </div>
              )}

              {/* Step 2: Encryption & Recovery Key */}
              {step === 2 && (
                <div className="space-y-4 text-xs">
                  <div className="p-4 bg-[#1D1B19]/70 rounded-xl border border-white/10 space-y-2">
                    <div className="flex items-center gap-2 font-semibold text-amber-300">
                      <Shield className="w-4 h-4 text-amber-300" />
                      Ihre Daten werden verschlüsselt
                    </div>
                    <p className="text-xs text-stone-300 leading-relaxed">
                      Sensible Inhalte (Buchungstexte, Verwendungszwecke, Kontaktdaten) werden mit AES-256
                      verschlüsselt. Der Zugriffsschlüssel wird automatisch und sicher im Schlüsselbund Ihres
                      Betriebssystems hinterlegt – Sie müssen sich kein Passwort merken. Original-Belege bleiben
                      aus rechtlichen Gründen unverändert erhalten.
                    </p>
                  </div>

                  <div className="p-4 bg-amber-500/10 rounded-xl border border-amber-500/30 space-y-2">
                    <div className="flex items-center gap-2 font-semibold text-amber-200">
                      <KeyRound className="w-4 h-4 text-amber-300" />
                      Wichtig: Recovery-Schlüssel für den Notfall
                    </div>
                    <p className="text-xs text-amber-100/90 leading-relaxed">
                      Geht dieser Rechner verloren oder defekt, ist der Schlüsselbund weg. Ohne externe Sicherung
                      sind die verschlüsselten Daten dann <strong>unwiederbringlich verloren</strong> – auch aus
                      einem Backup.
                    </p>
                    <div className="flex items-start gap-2 text-xs text-amber-100/90 bg-amber-500/10 p-2.5 rounded-lg border border-amber-500/20 leading-relaxed">
                      <AlertTriangle className="w-4 h-4 text-amber-300 shrink-0 mt-0.5" />
                      <span>
                        Exportieren Sie direkt nach der Einrichtung unter <strong>Einstellungen &rarr; Speicherort
                        &amp; Sicherheitsschlüssel</strong> einen <strong>Recovery-Schlüssel</strong> und bewahren Sie
                        ihn sicher und getrennt von Ihrem Datenbackup auf (z. B. USB-Stick, Tresor, Passwortmanager).
                      </span>
                    </div>
                  </div>
                </div>
              )}


              {/* Step 3: Company & Tax Info */}
              {step === 3 && (
                <div className="space-y-4 text-xs">
                  <div>
                    <label className="font-medium text-stone-200 block mb-1">
                      Firmen- oder Inhabername:
                    </label>
                    <input
                      type="text"
                      placeholder="z. B. Musterfirma GmbH oder Max Mustermann"
                      value={companySettings.companyName}
                      onChange={(e) =>
                        setCompanySettings({ ...companySettings, companyName: e.target.value })
                      }
                      className="w-full p-2.5 bg-[#1D1B19]/80 border border-stone-700 rounded-xl text-stone-200 focus:border-amber-400 focus:outline-hidden"
                      required
                    />
                  </div>

                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <label className="font-medium text-stone-200 block mb-1">Rechtsform:</label>
                      <select
                        value={companySettings.legalForm}
                        onChange={(e) => {
                          const form = e.target.value;
                          const isCorp = ['GmbH', 'UG (haftungsbeschränkt)', 'AG', 'GmbH & Co. KG'].includes(form);
                          setCompanySettings({
                            ...companySettings,
                            legalForm: form,
                            taxationType: isCorp ? 'SOLL' : companySettings.taxationType,
                          });
                        }}
                        className="w-full p-2.5 bg-[#1D1B19]/80 border border-stone-700 rounded-xl text-stone-200 focus:border-amber-400 focus:outline-hidden"
                      >
                        <option value="Einzelunternehmen">Einzelunternehmen / Freiberufler</option>
                        <option value="UG (haftungsbeschränkt)">UG (haftungsbeschränkt)</option>
                        <option value="GmbH">GmbH</option>
                        <option value="GbR">GbR</option>
                        <option value="GmbH & Co. KG">GmbH & Co. KG</option>
                        <option value="AG">AG</option>
                        <option value="Sonstige">Sonstige</option>
                      </select>
                    </div>

                    <div>
                      <label className="font-medium text-stone-200 block mb-1">
                        Start-Geschäftsjahr:
                      </label>
                      <input
                        type="number"
                        value={companySettings.fiscalYear}
                        onChange={(e) =>
                          setCompanySettings({
                            ...companySettings,
                            fiscalYear: Number(e.target.value),
                          })
                        }
                        className="w-full p-2.5 bg-[#1D1B19]/80 border border-stone-700 rounded-xl text-stone-200 font-mono focus:border-amber-400 focus:outline-hidden"
                      />
                    </div>
                  </div>

                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <label className="font-medium text-stone-200 block mb-1">
                        Steuernummer (Finanzamt):
                      </label>
                      <input
                        type="text"
                        placeholder="12/345/67890"
                        value={companySettings.taxNumber}
                        onChange={(e) =>
                          setCompanySettings({ ...companySettings, taxNumber: e.target.value })
                        }
                        className="w-full p-2.5 bg-[#1D1B19]/80 border border-stone-700 rounded-xl text-stone-200 font-mono focus:border-amber-400 focus:outline-hidden"
                      />
                    </div>
                    <div>
                      <label className="font-medium text-stone-200 block mb-1">
                        USt-IdNr. (optional):
                      </label>
                      <input
                        type="text"
                        placeholder="DE123456789"
                        value={companySettings.vatId}
                        onChange={(e) =>
                          setCompanySettings({ ...companySettings, vatId: e.target.value })
                        }
                        className="w-full p-2.5 bg-[#1D1B19]/80 border border-stone-700 rounded-xl text-stone-200 font-mono focus:border-amber-400 focus:outline-hidden"
                      />
                    </div>
                  </div>

                  {/* Taxes and VAT Configuration */}
                  <div className="p-3.5 bg-[#1D1B19]/70 rounded-xl border border-white/10 space-y-3">
                    <div className="flex items-center gap-2 font-semibold text-amber-300">
                      <ReceiptText className="w-4 h-4 text-amber-300" />
                      Umsatzsteuer & Besteuerungsart
                    </div>

                    <div>
                      <label className="font-medium text-stone-200 block mb-1">
                        Umsatzsteuerpflicht:
                      </label>
                      <select
                        value={companySettings.isSmallBusiness ? 'small_business' : 'standard'}
                        onChange={(e) => {
                          const isSmall = e.target.value === 'small_business';
                          setCompanySettings({
                            ...companySettings,
                            isSmallBusiness: isSmall,
                            vatPeriod: isSmall ? 'exempt' : 'quarter',
                            taxationType: 'IST',
                          });
                        }}
                        className="w-full p-2 bg-[#1D1B19]/90 border border-stone-700 rounded-lg text-stone-200 focus:border-amber-400 focus:outline-hidden"
                      >
                        <option value="standard">Regelbesteuerung (Umsatzsteuerpflichtig mit Vorsteuerabzug)</option>
                        <option value="small_business">Kleinunternehmer nach § 19 UStG (Keine Umsatzsteuer auf Rechnungen)</option>
                      </select>
                    </div>

                    {!companySettings.isSmallBusiness ? (
                      <div className="grid grid-cols-2 gap-3 pt-1">
                        <div>
                          <label className="font-medium text-stone-200 block mb-1">
                            USt-Voranmeldezeitraum:
                          </label>
                          <select
                            value={companySettings.vatPeriod || 'quarter'}
                            onChange={(e) =>
                              setCompanySettings({
                                ...companySettings,
                                vatPeriod: e.target.value as any,
                              })
                            }
                            className="w-full p-2 bg-[#1D1B19]/90 border border-stone-700 rounded-lg text-stone-200 focus:border-amber-400 focus:outline-hidden"
                          >
                            <option value="quarter">Quartalsweise (Standard)</option>
                            <option value="month">Monatlich (z. B. Neugründung)</option>
                            <option value="year">Jährlich (nur USt-Erklärung)</option>
                          </select>
                        </div>

                        <div>
                          <label className="font-medium text-stone-200 block mb-1">
                            Besteuerungsart:
                          </label>
                          <select
                            value={companySettings.taxationType || 'IST'}
                            onChange={(e) =>
                              setCompanySettings({
                                ...companySettings,
                                taxationType: e.target.value as any,
                              })
                            }
                            className="w-full p-2 bg-[#1D1B19]/90 border border-stone-700 rounded-lg text-stone-200 focus:border-amber-400 focus:outline-hidden"
                          >
                            <option value="IST">IST-Versteuerung (Zahlungseingang / Bank)</option>
                            <option value="SOLL">SOLL-Versteuerung (Rechnungsdatum)</option>
                          </select>
                        </div>
                      </div>
                    ) : (
                      <div className="p-2.5 bg-amber-500/10 rounded-lg border border-amber-500/20 text-[11px] text-amber-200 leading-relaxed">
                        Als Kleinunternehmer nach § 19 UStG weisen Sie auf Ausgangsrechnungen keine Umsatzsteuer aus und führen keine monatlichen oder quartalsweisen Voranmeldungen ab.
                      </div>
                    )}
                  </div>
                </div>
              )}

              {/* Step 4: Bank & Konten */}
              {step === 4 && (
                <div className="space-y-4 text-xs">
                  <div>
                    <label className="font-medium text-stone-200 block mb-1">
                      Geschäftskonto IBAN:
                    </label>
                    <input
                      type="text"
                      placeholder="DE89 3704 0044 0532 0130 00"
                      value={companySettings.iban}
                      onChange={(e) =>
                        setCompanySettings({ ...companySettings, iban: e.target.value })
                      }
                      className="w-full p-2.5 bg-[#1D1B19]/80 border border-stone-700 rounded-xl text-stone-200 font-mono focus:border-amber-400 focus:outline-hidden"
                    />
                    <p className="text-xs text-stone-400 mt-1">
                      Wird automatisch mit Ihrem Haupt-Bankkonto verknüpft.
                    </p>
                  </div>

                  <div>
                    <label className="font-medium text-stone-200 block mb-1">Bankname:</label>
                    <input
                      type="text"
                      placeholder="z. B. Sparkasse, Volksbank, Commerzbank, Qonto, N26"
                      value={companySettings.bankName}
                      onChange={(e) =>
                        setCompanySettings({ ...companySettings, bankName: e.target.value })
                      }
                      className="w-full p-2.5 bg-[#1D1B19]/80 border border-stone-700 rounded-xl text-stone-200 focus:border-amber-400 focus:outline-hidden"
                    />
                  </div>

                  <div className="p-3.5 bg-[#1D1B19]/70 rounded-xl border border-white/10 text-stone-300 space-y-1">
                    <span className="font-semibold text-white block">
                      Standard-Kontenrahmen
                    </span>
                    <p className="text-xs leading-relaxed text-stone-300">
                      Enthält alle gängigen Konten für Einnahmen, Ausgaben, Steuern und Bankverkehr mit einfachen Beschreibungen.
                    </p>
                  </div>
                </div>
              )}

              {/* Actions */}
              <div className="flex items-center justify-between border-t border-white/10 pt-4">
                <button
                  type="button"
                  onClick={() => (step > 1 ? setStep((s) => (s - 1) as any) : setSetupChoice(null))}
                  className="px-4 py-2 text-xs font-medium text-stone-300 hover:text-white rounded-lg flex items-center gap-1.5 transition-colors"
                >
                  <ArrowLeft className="w-3.5 h-3.5" /> Zurück
                </button>

                {step < 4 ? (
                  <button
                    type="button"
                    disabled={
                      (step === 3 && !companySettings.companyName.trim())
                    }
                    onClick={() => setStep((s) => (s + 1) as any)}
                    className="px-5 py-2.5 text-xs font-semibold bg-amber-700 hover:bg-amber-600 disabled:opacity-40 text-white rounded-xl shadow-md flex items-center gap-1.5 transition-all"
                  >
                    Weiter <ArrowRight className="w-3.5 h-3.5" />
                  </button>
                ) : (
                  <button
                    type="button"
                    disabled={isSubmitting}
                    onClick={handleFinishWizard}
                    className="px-6 py-2.5 text-xs font-semibold bg-amber-700 hover:bg-amber-600 text-white rounded-xl shadow-md flex items-center gap-2 transition-all"
                  >
                    <CheckCircle2 className="w-4 h-4" />
                    {isSubmitting ? 'Wird eingerichtet...' : 'Buchhaltung jetzt starten'}
                  </button>
                )}
              </div>
            </>
          ) : (
            /* Load Existing DB Mode */
            <form onSubmit={handleLoadExisting} className="space-y-4 text-xs">
              <div className="border-b border-white/10 pb-3">
                <h2 className="text-sm font-bold text-white">Bestehende Buchhaltungsdatei öffnen</h2>
                <p className="text-xs text-stone-300 mt-0.5">
                  Wählen Sie eine vorhandene Buchfink-Datei von Ihrer Festplatte aus.
                </p>
              </div>

              <div>
                <label className="font-medium text-stone-200 block mb-1">
                  Pfad zur Buchhaltungsdatei:
                </label>
                <div className="flex gap-2">
                  <div className="relative flex-1">
                    <Database className="w-4 h-4 text-stone-400 absolute left-3 top-1/2 -translate-y-1/2" />
                    <input
                      type="text"
                      placeholder="/Pfad/zur/buchfink_2026_datei"
                      value={existingDbPath}
                      onChange={(e) => setExistingDbPath(e.target.value)}
                      className="w-full pl-9 pr-3 py-2 bg-[#1D1B19]/80 border border-stone-700 rounded-xl text-stone-200 text-xs focus:border-amber-400 focus:outline-hidden"
                      required
                    />
                  </div>
                  <button
                    type="button"
                    onClick={handlePickDatabaseFile}
                    className="px-3 py-2 rounded-xl bg-stone-800 hover:bg-stone-700 text-stone-200 font-medium text-xs border border-stone-700 transition-colors shrink-0 flex items-center gap-1.5"
                  >
                    <FileSearch className="w-3.5 h-3.5 text-amber-300" />
                    Datei auswählen...
                  </button>
                </div>
              </div>

              <div className="flex items-center justify-between border-t border-white/10 pt-4">
                <button
                  type="button"
                  onClick={() => setSetupChoice(null)}
                  className="px-4 py-2 text-xs font-medium text-stone-300 hover:text-white transition-colors"
                >
                  Zurück zur Auswahl
                </button>
                <button
                  type="submit"
                  disabled={isSubmitting || !existingDbPath.trim()}
                  className="px-5 py-2.5 text-xs font-semibold bg-amber-700 hover:bg-amber-600 text-white rounded-xl shadow-md flex items-center gap-1.5 transition-all"
                >
                  <FolderOpen className="w-4 h-4" />
                  {isSubmitting ? 'Wird geladen...' : 'Buchhaltung öffnen'}
                </button>
              </div>
            </form>
          )}
        </div>

        {/* Feature & Security Advantages Footer */}
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-6 text-xs text-stone-300 border-t border-white/10 pt-6">
          <div className="flex items-start gap-3">
            <ShieldCheck className="w-5 h-5 text-emerald-400 shrink-0 mt-0.5" />
            <div>
              <p className="font-semibold text-white">Sicher &amp; unveränderbar</p>
              <p className="text-[11px] text-stone-300/80 mt-0.5 leading-snug">
                GoBD-konforme kryptografische Hashkette schützt Buchungen lückenlos vor Manipulation.
              </p>
            </div>
          </div>

          <div className="flex items-start gap-3">
            <FolderOpen className="w-5 h-5 text-amber-400 shrink-0 mt-0.5" />
            <div>
              <p className="font-semibold text-white">100% lokal &amp; privat</p>
              <p className="text-[11px] text-stone-300/80 mt-0.5 leading-snug">
                Kein Cloud-Zwang. Ihre Finanzdaten und Schlüssel bleiben vollständig auf Ihrem Computer.
              </p>
            </div>
          </div>

          <div className="flex items-start gap-3">
            <Scale className="w-5 h-5 text-sky-400 shrink-0 mt-0.5" />
            <div>
              <p className="font-semibold text-white">Auswertungen &amp; E-Bilanz</p>
              <p className="text-[11px] text-stone-300/80 mt-0.5 leading-snug">
                Rechtssichere GuV, Bilanz und automatisierte Exporte für Finanzamt und Steuerberater.
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
