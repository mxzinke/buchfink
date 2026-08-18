import React, { useEffect, useState } from 'react';
import { toast } from 'sonner';
import { Save, CheckCircle2, Building, DollarSign, FolderOpen, Shield, ReceiptText, Loader2, Check, AlertCircle, Calendar, Info } from 'lucide-react';
import { CompanySettings, AppConfig } from '../types';
import { Api } from '../services/api';
import { HelpTooltip } from '../components/HelpTooltip';

export const SettingsPage: React.FC = () => {
  const [settings, setSettings] = useState<CompanySettings | null>(null);
  const [appConfig, setAppConfig] = useState<AppConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [savedMessage, setSavedMessage] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const monthNames = [
    { value: 1, label: 'Januar (Standard Kalenderjahr)' },
    { value: 2, label: 'Februar (1. Feb – 31. Jan)' },
    { value: 3, label: 'März (1. Mär – 28./29. Feb)' },
    { value: 4, label: 'April (1. Apr – 31. Mär)' },
    { value: 5, label: 'Mai (1. Mai – 30. Apr)' },
    { value: 6, label: 'Juni (1. Jun – 31. Mai)' },
    { value: 7, label: 'Juli (1. Jul – 30. Jun)' },
    { value: 8, label: 'August (1. Aug – 31. Jul)' },
    { value: 9, label: 'September (1. Sep – 31. Aug)' },
    { value: 10, label: 'Oktober (1. Okt – 30. Sep)' },
    { value: 11, label: 'November (1. Nov – 31. Okt)' },
    { value: 12, label: 'Dezember (1. Dez – 30. Nov)' },
  ];

  useEffect(() => {
    loadSettings();
  }, []);

  const loadSettings = async () => {
    setLoading(true);
    try {
      const [s, cfg] = await Promise.all([
        Api.getCompanySettings(),
        Api.getAppConfig(),
      ]);
      setSettings(s);
      setAppConfig(cfg);
    } finally {
      setLoading(false);
    }
  };

  const handlePickDirectory = async () => {
    try {
      const selected = await Api.selectDirectoryDialog('Buchfink Datenordner ändern');
      if (selected && appConfig) {
        setAppConfig({ ...appConfig, dataDir: selected });
      }
    } catch (e) {
      console.error(e);
    }
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!settings || isSaving) return;
    setIsSaving(true);
    setErrorMessage(null);
    try {
      await Api.updateCompanySettings(settings);
      setSavedMessage(true);
      toast.success('Einstellungen gespeichert', {
        description: 'Ihre Unternehmensdaten wurden erfolgreich aktualisiert.',
      });
      setTimeout(() => setSavedMessage(false), 3500);
    } catch (err: any) {
      const msg = err?.message || 'Fehler beim Speichern der Einstellungen.';
      setErrorMessage(msg);
      toast.error('Fehler beim Speichern', {
        description: msg,
      });
      setTimeout(() => setErrorMessage(null), 5000);
    } finally {
      setIsSaving(false);
    }
  };

  if (loading || !settings) {
    return <div className="p-8 text-center text-stone-500 text-xs">Einstellungen werden geladen...</div>;
  }

  const startMonth = settings.fiscalYearStartMonth || 1;
  const isDeviating = startMonth !== 1;

  return (
    <div className="p-8 max-w-4xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold text-stone-900 tracking-tight flex items-center">
            Einstellungen
            <HelpTooltip
              title="Einstellungen & Unternehmensdaten"
              content="Diese Stammdaten gelten übergeordnet für das gesamte Unternehmen. Buchungen und Rechnungen werden automatisch dem passenden Geschäftsjahr zugeordnet."
            />
          </h2>
          <p className="text-xs text-stone-500 mt-1">
            Übergeordnete Stammdaten &bull; Lokal &amp; privat auf Ihrem Rechner gespeichert
          </p>
        </div>

        {savedMessage && (
          <div className="px-3 py-1.5 rounded-lg bg-emerald-50 border border-emerald-200 text-emerald-800 text-xs flex items-center gap-1.5 font-medium animate-in fade-in transition-all">
            <CheckCircle2 className="w-3.5 h-3.5 text-emerald-600 shrink-0" />
            Änderungen gespeichert
          </div>
        )}
      </div>

      <form onSubmit={handleSave} className="space-y-6">
        {/* Storage & Key */}
        <div className="bg-white p-6 rounded-xl border border-stone-200/80 shadow-xs space-y-4">
          <h3 className="text-sm font-bold text-stone-900 flex items-center gap-2 border-b border-stone-100 pb-2">
            <FolderOpen className="w-4 h-4 text-amber-700" />
            Speicherort &amp; Sicherheitsschlüssel dieses Mandanten
          </h3>

          <div className="space-y-3 text-xs">
            <div>
              <label className="font-semibold text-stone-700 block mb-1">Speicherort für Buchungsdaten &amp; Belege:</label>
              <div className="flex gap-2">
                <input
                  type="text"
                  value={appConfig?.dataDir || ''}
                  readOnly
                  className="w-full p-2 bg-stone-50 border border-stone-200 rounded-lg text-stone-700 text-xs font-mono"
                />
                <button
                  type="button"
                  onClick={handlePickDirectory}
                  className="px-3 py-2 rounded-lg bg-stone-100 hover:bg-stone-200 text-stone-800 font-semibold border border-stone-200 transition-colors shrink-0 text-xs"
                >
                  Ordner wählen...
                </button>
              </div>
            </div>

            <div>
              <label className="font-semibold text-stone-700 block mb-1">Sicherheitsschlüssel (Signatur):</label>
              <div className="p-2.5 bg-stone-50 rounded-lg border border-stone-200 text-stone-600 flex items-center gap-2 text-xs font-mono">
                <Shield className="w-3.5 h-3.5 text-emerald-600 shrink-0" />
                <span className="truncate">{appConfig?.certPath || 'Standard-Sicherheitsschlüssel'}</span>
              </div>
            </div>
          </div>
        </div>

        {/* Company Identity */}
        <div className="bg-white p-6 rounded-xl border border-stone-200/80 shadow-xs space-y-4">
          <h3 className="text-sm font-bold text-stone-900 flex items-center gap-2 border-b border-stone-100 pb-2">
            <Building className="w-4 h-4 text-amber-700" />
            Unternehmensdaten (Stammdaten)
          </h3>

          <div className="p-3 bg-amber-50/60 rounded-lg border border-amber-200/60 text-xs text-amber-900 leading-relaxed">
            <strong>Stammdaten:</strong> Diese Firmendaten gelten übergeordnet für alle Geschäftsjahre dieses Mandanten.
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-xs">
            <div>
              <label className="font-semibold text-stone-700 block mb-1">Firmen- oder Inhabername:</label>
              <input
                type="text"
                value={settings.companyName}
                onChange={(e) => setSettings({ ...settings, companyName: e.target.value })}
                className="w-full p-2 bg-stone-50 border border-stone-200 rounded-lg focus:outline-none focus:border-amber-600 focus:ring-2 focus:ring-amber-500/20 text-xs text-stone-800"
              />
            </div>

            <div>
              <label className="font-semibold text-stone-700 block mb-1">Rechtsform:</label>
              <input
                type="text"
                value={settings.legalForm}
                onChange={(e) => setSettings({ ...settings, legalForm: e.target.value })}
                className="w-full p-2 bg-stone-50 border border-stone-200 rounded-lg focus:outline-none focus:border-amber-600 focus:ring-2 focus:ring-amber-500/20 text-xs text-stone-800"
              />
            </div>

            <div>
              <label className="font-semibold text-stone-700 block mb-1">Steuernummer:</label>
              <input
                type="text"
                value={settings.taxNumber}
                onChange={(e) => setSettings({ ...settings, taxNumber: e.target.value })}
                className="w-full p-2 bg-stone-50 border border-stone-200 rounded-lg focus:outline-none focus:border-amber-600 focus:ring-2 focus:ring-amber-500/20 font-mono text-xs text-stone-800"
              />
            </div>

            <div>
              <label className="font-semibold text-stone-700 block mb-1">Umsatzsteuer-Identifikationsnummer (USt-IdNr.):</label>
              <input
                type="text"
                value={settings.vatId}
                onChange={(e) => setSettings({ ...settings, vatId: e.target.value })}
                className="w-full p-2 bg-stone-50 border border-stone-200 rounded-lg focus:outline-none focus:border-amber-600 focus:ring-2 focus:ring-amber-500/20 font-mono text-xs text-stone-800"
              />
            </div>

            <div className="md:col-span-2">
              <label className="font-semibold text-stone-700 block mb-1">Zuständiges Finanzamt:</label>
              <input
                type="text"
                value={settings.taxOffice}
                onChange={(e) => setSettings({ ...settings, taxOffice: e.target.value })}
                className="w-full p-2 bg-stone-50 border border-stone-200 rounded-lg focus:outline-none focus:border-amber-600 focus:ring-2 focus:ring-amber-500/20 text-xs text-stone-800"
              />
            </div>
          </div>
        </div>

        {/* Fiscal Year Configuration (Standard vs Deviating) */}
        <div className="bg-white p-6 rounded-xl border border-stone-200/80 shadow-xs space-y-4">
          <h3 className="text-sm font-bold text-stone-900 flex items-center gap-2 border-b border-stone-100 pb-2">
            <Calendar className="w-4 h-4 text-amber-700" />
            Geschäftsjahr &amp; Wirtschaftsjahr
          </h3>

          <div className="space-y-4 text-xs">
            <div>
              <label className="font-semibold text-stone-700 block mb-1">
                Geschäftsjahr-Typ:
              </label>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <label className={`p-3 rounded-xl border cursor-pointer flex items-start gap-2.5 transition-all ${
                  !isDeviating ? 'bg-amber-50/70 border-amber-600 ring-1 ring-amber-600/30' : 'bg-stone-50 border-stone-200 hover:bg-stone-100'
                }`}>
                  <input
                    type="radio"
                    name="fy_type"
                    checked={!isDeviating}
                    onChange={() => setSettings({ ...settings, fiscalYearStartMonth: 1 })}
                    className="mt-0.5"
                  />
                  <div>
                    <span className="font-bold text-stone-900 block">Kalenderjahr (Standard)</span>
                    <span className="text-[11px] text-stone-500">
                      Läuft vom 1. Januar bis zum 31. Dezember.
                    </span>
                  </div>
                </label>

                <label className={`p-3 rounded-xl border cursor-pointer flex items-start gap-2.5 transition-all ${
                  isDeviating ? 'bg-amber-50/70 border-amber-600 ring-1 ring-amber-600/30' : 'bg-stone-50 border-stone-200 hover:bg-stone-100'
                }`}>
                  <input
                    type="radio"
                    name="fy_type"
                    checked={isDeviating}
                    onChange={() => setSettings({ ...settings, fiscalYearStartMonth: 7 })}
                    className="mt-0.5"
                  />
                  <div>
                    <span className="font-bold text-stone-900 block">Abweichendes Geschäftsjahr</span>
                    <span className="text-[11px] text-stone-500">
                      Beginnt an einem anderen Monat (z. B. 1. Juli oder 1. April).
                    </span>
                  </div>
                </label>
              </div>
            </div>

            {isDeviating && (
              <div className="p-4 bg-amber-50/50 rounded-xl border border-amber-200/70 space-y-3 animate-in fade-in">
                <div>
                  <label className="font-semibold text-stone-800 block mb-1">
                    Beginn des Wirtschaftsjahres (Startmonat):
                  </label>
                  <select
                    value={settings.fiscalYearStartMonth || 7}
                    onChange={(e) => setSettings({ ...settings, fiscalYearStartMonth: Number(e.target.value) })}
                    className="w-full p-2 bg-white border border-stone-300 rounded-lg text-xs text-stone-800 focus:outline-none focus:border-amber-600"
                  >
                    {monthNames.map((m) => (
                      <option key={m.value} value={m.value}>
                        {m.label}
                      </option>
                    ))}
                  </select>
                </div>
              </div>
            )}

            <div className="p-3.5 bg-stone-50 rounded-xl border border-stone-200/70 flex items-start gap-2.5 text-stone-600">
              <Info className="w-4 h-4 text-amber-700 shrink-0 mt-0.5" />
              <p className="text-[11px] leading-relaxed">
                <strong>Automatische Jahreszuordnung:</strong> Da Belege, Rechnungen und Bankzahlungen jeweils ein Datum aufweisen, weist Buchfink jeden Vorgang automatisch dem passenden Geschäftsjahr zu. Bei Buchungen im Grenzbereich (Übergangsfrist / Jahresabschluss) können Sie das Geschäftsjahr bei Bedarf im Journal manuell übersteuern.
              </p>
            </div>
          </div>
        </div>

        {/* Taxes & VAT Configuration */}
        <div className="bg-white p-6 rounded-xl border border-stone-200/80 shadow-xs space-y-4">
          <h3 className="text-sm font-bold text-stone-900 flex items-center gap-2 border-b border-stone-100 pb-2">
            <ReceiptText className="w-4 h-4 text-amber-700" />
            Steuern &amp; Umsatzsteuer
          </h3>

          <div className="space-y-4 text-xs">
            <div>
              <label className="font-semibold text-stone-700 block mb-1">
                Umsatzsteuer-Status:
              </label>
              <select
                value={settings.isSmallBusiness ? 'small_business' : 'standard'}
                onChange={(e) => {
                  const isSmall = e.target.value === 'small_business';
                  setSettings({
                    ...settings,
                    isSmallBusiness: isSmall,
                    vatPeriod: isSmall ? 'exempt' : settings.vatPeriod || 'quarter',
                  });
                }}
                className="w-full p-2 bg-stone-50 border border-stone-200 rounded-lg text-stone-800 focus:outline-none focus:border-amber-600 focus:ring-2 focus:ring-amber-500/20 text-xs"
              >
                <option value="standard">Regelbesteuerung (Umsatzsteuerpflichtig mit Vorsteuerabzug)</option>
                <option value="small_business">Kleinunternehmer nach § 19 UStG (Keine Umsatzsteuer auf Rechnungen)</option>
              </select>
            </div>

            {!settings.isSmallBusiness ? (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4 pt-1">
                <div>
                  <label className="font-semibold text-stone-700 block mb-1">
                    USt-Voranmeldezeitraum:
                  </label>
                  <select
                    value={settings.vatPeriod || 'quarter'}
                    onChange={(e) =>
                      setSettings({
                        ...settings,
                        vatPeriod: e.target.value as any,
                      })
                    }
                    className="w-full p-2 bg-stone-50 border border-stone-200 rounded-lg text-stone-800 focus:outline-none focus:border-amber-600 focus:ring-2 focus:ring-amber-500/20 text-xs"
                  >
                    <option value="quarter">Quartalsweise / Vierteljährlich (Standard)</option>
                    <option value="month">Monatlich (z. B. Neugründung oder hohe Zahllast)</option>
                    <option value="year">Jährlich (nur USt-Jahreserklärung)</option>
                  </select>
                </div>

                <div>
                  <label className="font-semibold text-stone-700 block mb-1">
                    Besteuerungsart:
                  </label>
                  <select
                    value={settings.taxationType || 'IST'}
                    onChange={(e) =>
                      setSettings({
                        ...settings,
                        taxationType: e.target.value as any,
                      })
                    }
                    className="w-full p-2 bg-stone-50 border border-stone-200 rounded-lg text-stone-800 focus:outline-none focus:border-amber-600 focus:ring-2 focus:ring-amber-500/20 text-xs"
                  >
                    <option value="IST">IST-Versteuerung (nach vereinnahmten Entgelten / Zahlungseingang)</option>
                    <option value="SOLL">SOLL-Versteuerung (nach vereinbarten Entgelten / Rechnungsdatum)</option>
                  </select>
                </div>
              </div>
            ) : (
              <div className="p-3 bg-amber-50/70 rounded-lg border border-amber-200/60 text-stone-600 leading-relaxed">
                Als Kleinunternehmer nach § 19 UStG weisen Sie auf Ihren Rechnungen keine Umsatzsteuer aus und führen keine USt-Voranmeldungen durch.
              </div>
            )}
          </div>
        </div>

        {/* Address */}
        <div className="bg-white p-6 rounded-xl border border-stone-200/80 shadow-xs space-y-4">
          <h3 className="text-sm font-bold text-stone-900 border-b border-stone-100 pb-2">
            Anschrift
          </h3>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-xs">
            <div className="md:col-span-2">
              <label className="font-semibold text-stone-700 block mb-1">Straße &amp; Hausnummer:</label>
              <input
                type="text"
                value={settings.street}
                onChange={(e) => setSettings({ ...settings, street: e.target.value })}
                className="w-full p-2 bg-stone-50 border border-stone-200 rounded-lg focus:outline-none focus:border-amber-600 focus:ring-2 focus:ring-amber-500/20 text-xs text-stone-800"
              />
            </div>

            <div>
              <label className="font-semibold text-stone-700 block mb-1">PLZ &amp; Ort:</label>
              <input
                type="text"
                value={settings.zipCity}
                onChange={(e) => setSettings({ ...settings, zipCity: e.target.value })}
                className="w-full p-2 bg-stone-50 border border-stone-200 rounded-lg focus:outline-none focus:border-amber-600 focus:ring-2 focus:ring-amber-500/20 text-xs text-stone-800"
              />
            </div>

            <div>
              <label className="font-semibold text-stone-700 block mb-1">Land:</label>
              <input
                type="text"
                value={settings.country}
                onChange={(e) => setSettings({ ...settings, country: e.target.value })}
                className="w-full p-2 bg-stone-50 border border-stone-200 rounded-lg focus:outline-none focus:border-amber-600 focus:ring-2 focus:ring-amber-500/20 text-xs text-stone-800"
              />
            </div>
          </div>
        </div>

        {/* Banking */}
        <div className="bg-white p-6 rounded-xl border border-stone-200/80 shadow-xs space-y-4">
          <h3 className="text-sm font-bold text-stone-900 flex items-center gap-2 border-b border-stone-100 pb-2">
            <DollarSign className="w-4 h-4 text-amber-700" />
            Bankverbindung &amp; Kontenrahmen
          </h3>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-xs">
            <div>
              <label className="font-semibold text-stone-700 block mb-1">Bankname:</label>
              <input
                type="text"
                value={settings.bankName}
                onChange={(e) => setSettings({ ...settings, bankName: e.target.value })}
                className="w-full p-2 bg-stone-50 border border-stone-200 rounded-lg focus:outline-none focus:border-amber-600 focus:ring-2 focus:ring-amber-500/20 text-xs text-stone-800"
              />
            </div>

            <div>
              <label className="font-semibold text-stone-700 block mb-1">IBAN:</label>
              <input
                type="text"
                value={settings.iban}
                onChange={(e) => setSettings({ ...settings, iban: e.target.value })}
                className="w-full p-2 bg-stone-50 border border-stone-200 rounded-lg focus:outline-none focus:border-amber-600 focus:ring-2 focus:ring-amber-500/20 font-mono text-xs text-stone-800"
              />
            </div>

            <div>
              <label className="font-semibold text-stone-700 block mb-1">BIC:</label>
              <input
                type="text"
                value={settings.bic}
                onChange={(e) => setSettings({ ...settings, bic: e.target.value })}
                className="w-full p-2 bg-stone-50 border border-stone-200 rounded-lg focus:outline-none focus:border-amber-600 focus:ring-2 focus:ring-amber-500/20 font-mono text-xs text-stone-800"
              />
            </div>

            <div>
              <label className="font-semibold text-stone-700 block mb-1">Kontenrahmen:</label>
              <input
                type="text"
                disabled
                value="Standard SKR04 (Bilanz &amp; GuV)"
                className="w-full p-2 bg-stone-100 border border-stone-200 rounded-lg text-stone-600 cursor-not-allowed text-xs"
              />
            </div>
          </div>
        </div>

        {/* Action Bar / Submit */}
        <div className="flex flex-col sm:flex-row items-center justify-between gap-4 pt-2 border-t border-stone-200/60">
          <div className="flex items-center gap-2">
            {savedMessage && (
              <div className="px-3.5 py-2 rounded-lg bg-emerald-50 border border-emerald-200 text-emerald-800 text-xs flex items-center gap-2 font-medium animate-in fade-in slide-in-from-bottom-1">
                <CheckCircle2 className="w-4 h-4 text-emerald-600 shrink-0" />
                <span>Einstellungen wurden erfolgreich gespeichert.</span>
              </div>
            )}
            {errorMessage && (
              <div className="px-3.5 py-2 rounded-lg bg-rose-50 border border-rose-200 text-rose-800 text-xs flex items-center gap-2 font-medium animate-in fade-in slide-in-from-bottom-1">
                <AlertCircle className="w-4 h-4 text-rose-600 shrink-0" />
                <span>{errorMessage}</span>
              </div>
            )}
          </div>

          <div className="flex items-center gap-3 ml-auto">
            <button
              type="submit"
              disabled={isSaving}
              className={`flex items-center gap-2 px-6 py-2.5 rounded-lg text-xs font-semibold transition-all shadow-xs disabled:opacity-70 disabled:cursor-not-allowed ${
                savedMessage
                  ? 'bg-emerald-700 hover:bg-emerald-800 text-white'
                  : 'bg-amber-700 hover:bg-amber-800 text-white'
              }`}
            >
              {isSaving ? (
                <>
                  <Loader2 className="w-4 h-4 animate-spin" />
                  <span>Wird gespeichert...</span>
                </>
              ) : savedMessage ? (
                <>
                  <Check className="w-4 h-4" />
                  <span>Gespeichert!</span>
                </>
              ) : (
                <>
                  <Save className="w-4 h-4" />
                  <span>Einstellungen speichern</span>
                </>
              )}
            </button>
          </div>
        </div>
      </form>
    </div>
  );
};
