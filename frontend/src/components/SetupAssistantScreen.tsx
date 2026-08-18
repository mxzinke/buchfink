import React, { useState } from 'react';
import {
  FolderOpen,
  Lock,
  ArrowRight,
  ArrowLeft,
  CheckCircle2,
  Database,
  PlusCircle,
  FileSearch,
  KeyRound,
  AlertTriangle,
  Info,
} from 'lucide-react';
import { CompanySettings } from '../types';
import { Api } from '../services/api';

interface SetupAssistantScreenProps {
  onSetupCompleted: () => void;
}

export const SetupAssistantScreen: React.FC<SetupAssistantScreenProps> = ({
  onSetupCompleted,
}) => {
  const currentYear = new Date().getFullYear(); // e.g. 2026
  const [setupChoice, setSetupChoice] = useState<'new' | 'existing' | null>(null);
  const [step, setStep] = useState<1 | 2 | 3 | 4>(1);

  // Form State
  const [dataDir, setDataDir] = useState('~/.buchfink/data');
  const [certDir, setCertDir] = useState('~/.buchfink/keys');
  const [password, setPassword] = useState('');
  const [existingDbPath, setExistingDbPath] = useState('');

  const [companySettings, setCompanySettings] = useState<CompanySettings>({
    companyName: '',
    legalForm: 'GmbH',
    fiscalYear: currentYear,
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

  const handlePickCertDirectory = async () => {
    try {
      const selected = await Api.selectDirectoryDialog('Speicherort für Sicherheitszertifikat auswählen');
      if (selected) {
        setCertDir(selected);
      }
    } catch (e) {
      console.error(e);
    }
  };

  const handlePickDatabaseFile = async () => {
    try {
      const selected = await Api.selectDatabaseFileDialog('Buchfink SQLite-Datenbank auswählen');
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
      await Api.setupApplication(dataDir, certDir, password, companySettings);
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
      setErrorMsg(e.message || 'Datenbankdatei konnte nicht geladen werden.');
    } finally {
      setIsSubmitting(false);
    }
  };

  const isSameDirectory = dataDir.trim() !== '' && certDir.trim() !== '' && dataDir.trim() === certDir.trim();

  return (
    <div className="relative min-h-screen flex flex-col justify-between bg-stone-950 text-stone-100 overflow-y-auto">
      {/* Background with warm moody tone */}
      <div
        className="absolute inset-0 bg-cover bg-center opacity-20 mix-blend-luminosity pointer-events-none scale-105 filter blur-xs"
        style={{ backgroundImage: "url('/bg-startupscreen_unsplash-steven-kamenar.jpg')" }}
      />
      <div className="absolute inset-0 bg-gradient-to-t from-stone-950 via-stone-900/90 to-stone-950 pointer-events-none" />

      {/* Main Content Area */}
      <div className="relative z-10 max-w-2xl mx-auto w-full px-6 py-12 flex-1 flex flex-col justify-center space-y-6">
        {/* Brand Header */}
        <div className="flex items-center gap-4">
          <img
            src="/buchfink-logo.svg"
            alt="Buchfink"
            className="w-14 h-14 rounded-2xl bg-white/10 p-1 border border-white/15 backdrop-blur-md drop-shadow-md"
          />
          <div>
            <div className="flex items-center gap-2">
              <h1 className="text-2xl font-extrabold text-white tracking-tight">Buchfink</h1>
              <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-amber-500/20 text-amber-300 border border-amber-500/40">
                SKR04
              </span>
            </div>
            <p className="text-xs text-stone-400 mt-0.5">
              Ersteinrichtung &bull; GoBD-Zertifikat &bull; Lokaler Speicherort
            </p>
          </div>
        </div>

        {/* Wizard Box */}
        <div className="bg-stone-900/90 backdrop-blur-xl border border-stone-800 rounded-2xl p-7 shadow-2xl space-y-6">
          {errorMsg && (
            <div className="p-3 bg-rose-500/10 border border-rose-500/30 text-rose-300 text-xs rounded-xl">
              {errorMsg}
            </div>
          )}

          {/* Initial Choice: New vs Existing */}
          {setupChoice === null ? (
            <div className="space-y-6">
              <div className="space-y-1">
                <h2 className="text-base font-bold text-white">Willkommen bei Buchfink</h2>
                <p className="text-xs text-stone-400">
                  Wählen Sie, wie Sie Ihre Buchhaltung starten möchten:
                </p>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {/* Option 1: New Setup */}
                <div
                  onClick={() => setSetupChoice('new')}
                  className="p-5 rounded-2xl bg-stone-950/80 border-2 border-stone-800 hover:border-amber-500 hover:bg-stone-900/90 transition-all cursor-pointer group space-y-3"
                >
                  <div className="w-10 h-10 rounded-xl bg-amber-500/20 text-amber-400 flex items-center justify-center border border-amber-500/30 group-hover:scale-105 transition-transform">
                    <PlusCircle className="w-5 h-5" />
                  </div>
                  <div>
                    <h3 className="font-bold text-sm text-white group-hover:text-amber-300 transition-colors">
                      Neue Buchhaltung anlegen
                    </h3>
                    <p className="text-xs text-stone-400 mt-1 leading-snug">
                      Erstellt eine neue SQLite-Datenbank für {currentYear}, initialisiert den SKR04-Kontenrahmen und erzeugt Ihr GoBD-Sicherheitszertifikat.
                    </p>
                  </div>
                  <div className="text-xs font-semibold text-amber-400 flex items-center gap-1 pt-1">
                    Jetzt einrichten &rarr;
                  </div>
                </div>

                {/* Option 2: Load Existing Database */}
                <div
                  onClick={() => setSetupChoice('existing')}
                  className="p-5 rounded-2xl bg-stone-950/80 border-2 border-stone-800 hover:border-amber-500 hover:bg-stone-900/90 transition-all cursor-pointer group space-y-3"
                >
                  <div className="w-10 h-10 rounded-xl bg-stone-800 text-stone-300 flex items-center justify-center border border-stone-700 group-hover:scale-105 transition-transform">
                    <Database className="w-5 h-5" />
                  </div>
                  <div>
                    <h3 className="font-bold text-sm text-white group-hover:text-amber-300 transition-colors">
                      Bestehende Datenbank öffnen
                    </h3>
                    <p className="text-xs text-stone-400 mt-1 leading-snug">
                      Laden Sie eine bereits vorhandene Buchfink-Datenbankdatei (*.sqlite) oder ein Backup von einem lokalen Pfad.
                    </p>
                  </div>
                  <div className="text-xs font-semibold text-stone-300 group-hover:text-amber-400 flex items-center gap-1 pt-1">
                    Datei auswählen &rarr;
                  </div>
                </div>
              </div>
            </div>
          ) : setupChoice === 'new' ? (
            /* New Setup Wizard */
            <>
              {/* Step indicator */}
              <div className="flex items-center justify-between border-b border-stone-800 pb-4">
                <div>
                  <span className="text-[11px] font-bold uppercase tracking-wider text-amber-400">
                    Schritt {step} von 4
                  </span>
                  <h2 className="text-sm font-bold text-white mt-0.5">
                    {step === 1 && '1. Speicherort der Buchhaltungsdaten'}
                    {step === 2 && '2. GoBD-Sicherheitszertifikat (Digitaler Schlüssel)'}
                    {step === 3 && '3. Unternehmensdaten & Geschäftsjahr'}
                    {step === 4 && '4. Bankverbindung (SKR04 Konto 1800)'}
                  </h2>
                </div>
                <div className="flex items-center gap-1.5">
                  {[1, 2, 3, 4].map((s) => (
                    <div
                      key={s}
                      className={`h-2 rounded-full transition-all ${
                        step === s ? 'bg-amber-500 w-6' : step > s ? 'bg-amber-500/50 w-2' : 'bg-stone-800 w-2'
                      }`}
                    />
                  ))}
                </div>
              </div>

              {/* Step 1: Storage Location for Data */}
              {step === 1 && (
                <div className="space-y-4 text-xs">
                  <div className="p-3.5 bg-stone-950/60 rounded-xl border border-stone-800 text-stone-400 flex items-start gap-2.5">
                    <Info className="w-4 h-4 text-amber-400 shrink-0 mt-0.5" />
                    <div className="space-y-1">
                      <span className="font-bold text-stone-200 block">Lokale SQLite-Speicherung</span>
                      <p className="text-[11px] leading-relaxed">
                        Buchfink speichert Ihre Buchungsdaten und Belege in einer separaten SQLite-Datei pro Geschäftsjahr auf Ihrer lokalen Festplatte.
                      </p>
                    </div>
                  </div>

                  <div>
                    <label className="font-semibold text-stone-300 block mb-1">
                      Datenordner (für Datenbanken & Belege):
                    </label>
                    <div className="flex gap-2">
                      <div className="relative flex-1">
                        <FolderOpen className="w-4 h-4 text-stone-500 absolute left-3 top-1/2 -translate-y-1/2" />
                        <input
                          type="text"
                          value={dataDir}
                          onChange={(e) => setDataDir(e.target.value)}
                          className="w-full pl-9 pr-3 py-2 bg-stone-950/80 border border-stone-700 rounded-xl font-mono text-stone-200 focus:border-amber-500 focus:outline-hidden"
                        />
                      </div>
                      <button
                        type="button"
                        onClick={handlePickDataDirectory}
                        className="px-3 py-2 rounded-xl bg-stone-800 hover:bg-stone-700 text-stone-200 font-medium text-xs border border-stone-700 transition-colors shrink-0 flex items-center gap-1.5"
                      >
                        <FolderOpen className="w-3.5 h-3.5 text-amber-400" />
                        Ordner auswählen...
                      </button>
                    </div>
                  </div>
                </div>
              )}

              {/* Step 2: Dedicated Certificate & Key Setup */}
              {step === 2 && (
                <div className="space-y-4 text-xs">
                  <div className="p-4 bg-stone-950/80 rounded-xl border border-stone-800 space-y-2">
                    <div className="flex items-center gap-2 font-bold text-amber-400">
                      <KeyRound className="w-4 h-4 text-amber-400" />
                      Warum ein separates Sicherheitszertifikat?
                    </div>
                    <p className="text-[11px] text-stone-300 leading-relaxed">
                      Buchfink schützt Ihre Buchungen mit einem digitalen Schlüssel (Zertifikat). Jede Buchung wird damit kryptografisch gestempelt (GoBD-Hash-Chain).
                    </p>
                    <div className="text-[11px] text-amber-200/90 bg-amber-500/10 p-2.5 rounded-lg border border-amber-500/20 leading-relaxed">
                      <strong>Wichtiger Sicherheitshinweis:</strong> Wie bei einem Tresor sollten Sie den digitalen Schlüssel an einem anderen Ort aufbewahren als Ihre Buchhaltungsdaten (z. B. in einem geschützten Schlüsselordner, einem separaten USB-Stick oder Dokumenten-Tresor).
                    </div>
                  </div>

                  <div>
                    <label className="font-semibold text-stone-300 block mb-1">
                      Speicherort für den digitalen Schlüssel:
                    </label>
                    <div className="flex gap-2">
                      <div className="relative flex-1">
                        <KeyRound className="w-4 h-4 text-stone-500 absolute left-3 top-1/2 -translate-y-1/2" />
                        <input
                          type="text"
                          value={certDir}
                          onChange={(e) => setCertDir(e.target.value)}
                          className="w-full pl-9 pr-3 py-2 bg-stone-950/80 border border-stone-700 rounded-xl font-mono text-stone-200 focus:border-amber-500 focus:outline-hidden"
                        />
                      </div>
                      <button
                        type="button"
                        onClick={handlePickCertDirectory}
                        className="px-3 py-2 rounded-xl bg-stone-800 hover:bg-stone-700 text-stone-200 font-medium text-xs border border-stone-700 transition-colors shrink-0 flex items-center gap-1.5"
                      >
                        <FolderOpen className="w-3.5 h-3.5 text-amber-400" />
                        Ordner auswählen...
                      </button>
                    </div>
                  </div>

                  {isSameDirectory && (
                    <div className="p-3 bg-amber-500/10 border border-amber-500/30 rounded-xl text-amber-300 text-[11px] flex items-start gap-2">
                      <AlertTriangle className="w-4 h-4 text-amber-400 shrink-0 mt-0.5" />
                      <span>
                        Hinweis: Schlüssel und Buchhaltungsdaten liegen im selben Ordner ({dataDir}). Für optimale Sicherheit empfehlen wir getrennte Pfade.
                      </span>
                    </div>
                  )}

                  <div>
                    <label className="font-semibold text-stone-300 block mb-1">
                      Zertifikats-Passwort (Optional für Zugriffsschutz):
                    </label>
                    <div className="relative">
                      <Lock className="w-4 h-4 text-stone-500 absolute left-3 top-1/2 -translate-y-1/2" />
                      <input
                        type="password"
                        placeholder="Optionales Kennwort zum Schutz des Schlüssels..."
                        value={password}
                        onChange={(e) => setPassword(e.target.value)}
                        className="w-full pl-9 pr-3 py-2 bg-stone-950/80 border border-stone-700 rounded-xl text-stone-200 focus:border-amber-500 focus:outline-hidden"
                      />
                    </div>
                  </div>
                </div>
              )}

              {/* Step 3: Company Info */}
              {step === 3 && (
                <div className="space-y-4 text-xs">
                  <div>
                    <label className="font-semibold text-stone-300 block mb-1">
                      Firmen- / Unternehmensname:
                    </label>
                    <input
                      type="text"
                      placeholder="z. B. Musterfirma GmbH oder Max Mustermann"
                      value={companySettings.companyName}
                      onChange={(e) =>
                        setCompanySettings({ ...companySettings, companyName: e.target.value })
                      }
                      className="w-full p-2.5 bg-stone-950/80 border border-stone-700 rounded-xl text-stone-200 focus:border-amber-500 focus:outline-hidden"
                      required
                    />
                  </div>

                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <label className="font-semibold text-stone-300 block mb-1">Rechtsform:</label>
                      <select
                        value={companySettings.legalForm}
                        onChange={(e) =>
                          setCompanySettings({ ...companySettings, legalForm: e.target.value })
                        }
                        className="w-full p-2.5 bg-stone-950/80 border border-stone-700 rounded-xl text-stone-200 focus:border-amber-500 focus:outline-hidden"
                      >
                        <option value="Einzelunternehmen">Einzelunternehmen / Freelancer</option>
                        <option value="UG (haftungsbeschränkt)">UG (haftungsbeschränkt)</option>
                        <option value="GmbH">GmbH</option>
                        <option value="GbR">GbR</option>
                        <option value="AG">AG</option>
                      </select>
                    </div>

                    <div>
                      <label className="font-semibold text-stone-300 block mb-1">
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
                        className="w-full p-2.5 bg-stone-950/80 border border-stone-700 rounded-xl text-stone-200 font-mono focus:border-amber-500 focus:outline-hidden"
                      />
                    </div>
                  </div>

                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <label className="font-semibold text-stone-300 block mb-1">
                        Steuernummer (Finanzamt):
                      </label>
                      <input
                        type="text"
                        placeholder="12/345/67890"
                        value={companySettings.taxNumber}
                        onChange={(e) =>
                          setCompanySettings({ ...companySettings, taxNumber: e.target.value })
                        }
                        className="w-full p-2.5 bg-stone-950/80 border border-stone-700 rounded-xl text-stone-200 font-mono focus:border-amber-500 focus:outline-hidden"
                      />
                    </div>
                    <div>
                      <label className="font-semibold text-stone-300 block mb-1">
                        USt-IdNr. (Optional):
                      </label>
                      <input
                        type="text"
                        placeholder="DE123456789"
                        value={companySettings.vatId}
                        onChange={(e) =>
                          setCompanySettings({ ...companySettings, vatId: e.target.value })
                        }
                        className="w-full p-2.5 bg-stone-950/80 border border-stone-700 rounded-xl text-stone-200 font-mono focus:border-amber-500 focus:outline-hidden"
                      />
                    </div>
                  </div>
                </div>
              )}

              {/* Step 4: Bank & SKR04 */}
              {step === 4 && (
                <div className="space-y-4 text-xs">
                  <div>
                    <label className="font-semibold text-stone-300 block mb-1">
                      Geschäftskonto IBAN:
                    </label>
                    <input
                      type="text"
                      placeholder="DE89 3704 0044 0532 0130 00"
                      value={companySettings.iban}
                      onChange={(e) =>
                        setCompanySettings({ ...companySettings, iban: e.target.value })
                      }
                      className="w-full p-2.5 bg-stone-950/80 border border-stone-700 rounded-xl text-stone-200 font-mono focus:border-amber-500 focus:outline-hidden"
                    />
                    <p className="text-[11px] text-stone-500 mt-1">
                      Wird im SKR04 automatisch mit dem Haupt-Finanzkonto 1800 (Bank) verknüpft.
                    </p>
                  </div>

                  <div>
                    <label className="font-semibold text-stone-300 block mb-1">Bankname:</label>
                    <input
                      type="text"
                      placeholder="z. B. Commerzbank, Qonto, N26, Sparkasse"
                      value={companySettings.bankName}
                      onChange={(e) =>
                        setCompanySettings({ ...companySettings, bankName: e.target.value })
                      }
                      className="w-full p-2.5 bg-stone-950/80 border border-stone-700 rounded-xl text-stone-200 focus:border-amber-500 focus:outline-hidden"
                    />
                  </div>

                  <div className="p-3.5 bg-stone-950/60 rounded-xl border border-stone-800 text-stone-400 space-y-1">
                    <span className="font-bold text-stone-200 block">
                      Kontenrahmen: SKR04 (Vorinstalliert)
                    </span>
                    <p className="text-[11px] leading-relaxed">
                      Inklusive Standardkonten für Einnahmen, Ausgaben, Vorsteuer und Bilanzierung.
                    </p>
                  </div>
                </div>
              )}

              {/* Actions */}
              <div className="flex items-center justify-between border-t border-stone-800 pt-4">
                <button
                  type="button"
                  onClick={() => (step > 1 ? setStep((s) => (s - 1) as any) : setSetupChoice(null))}
                  className="px-4 py-2 text-xs font-semibold text-stone-400 hover:text-white rounded-lg flex items-center gap-1.5"
                >
                  <ArrowLeft className="w-3.5 h-3.5" /> Zurück
                </button>

                {step < 4 ? (
                  <button
                    type="button"
                    disabled={step === 3 && !companySettings.companyName.trim()}
                    onClick={() => setStep((s) => (s + 1) as any)}
                    className="px-5 py-2.5 text-xs font-semibold bg-amber-600 hover:bg-amber-500 disabled:opacity-40 text-white rounded-xl shadow-lg flex items-center gap-1.5"
                  >
                    Weiter <ArrowRight className="w-3.5 h-3.5" />
                  </button>
                ) : (
                  <button
                    type="button"
                    disabled={isSubmitting}
                    onClick={handleFinishWizard}
                    className="px-6 py-2.5 text-xs font-semibold bg-amber-600 hover:bg-amber-500 text-white rounded-xl shadow-lg flex items-center gap-2"
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
              <div className="border-b border-stone-800 pb-3">
                <h2 className="text-sm font-bold text-white">Bestehende Datenbankdatei laden</h2>
                <p className="text-xs text-stone-400 mt-0.5">
                  Wählen Sie eine vorhandene SQLite-Datenbankdatei (*.sqlite) aus.
                </p>
              </div>

              <div>
                <label className="font-semibold text-stone-300 block mb-1">
                  Pfad zur SQLite-Datei:
                </label>
                <div className="flex gap-2">
                  <div className="relative flex-1">
                    <Database className="w-4 h-4 text-stone-500 absolute left-3 top-1/2 -translate-y-1/2" />
                    <input
                      type="text"
                      placeholder="/Pfad/zu/buchfink_2026.sqlite"
                      value={existingDbPath}
                      onChange={(e) => setExistingDbPath(e.target.value)}
                      className="w-full pl-9 pr-3 py-2 bg-stone-950/80 border border-stone-700 rounded-xl text-stone-200 font-mono text-xs focus:border-amber-500 focus:outline-hidden"
                      required
                    />
                  </div>
                  <button
                    type="button"
                    onClick={handlePickDatabaseFile}
                    className="px-3 py-2 rounded-xl bg-stone-800 hover:bg-stone-700 text-stone-200 font-medium text-xs border border-stone-700 transition-colors shrink-0 flex items-center gap-1.5"
                  >
                    <FileSearch className="w-3.5 h-3.5 text-amber-400" />
                    Datei auswählen...
                  </button>
                </div>
              </div>

              <div className="flex items-center justify-between border-t border-stone-800 pt-4">
                <button
                  type="button"
                  onClick={() => setSetupChoice(null)}
                  className="px-4 py-2 text-xs font-semibold text-stone-400 hover:text-white"
                >
                  Zurück zur Auswahl
                </button>
                <button
                  type="submit"
                  disabled={isSubmitting || !existingDbPath.trim()}
                  className="px-5 py-2.5 text-xs font-semibold bg-amber-600 hover:bg-amber-500 text-white rounded-xl shadow-lg flex items-center gap-1.5"
                >
                  <FolderOpen className="w-4 h-4" />
                  {isSubmitting ? 'Wird geladen...' : 'Datenbank öffnen'}
                </button>
              </div>
            </form>
          )}
        </div>
      </div>
    </div>
  );
};
