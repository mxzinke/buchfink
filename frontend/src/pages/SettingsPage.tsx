import React, { useEffect, useState } from 'react';
import { Save, CheckCircle2, Building, DollarSign, FolderOpen, Shield } from 'lucide-react';
import { CompanySettings, AppConfig } from '../types';
import { Api } from '../services/api';
import { HelpTooltip } from '../components/HelpTooltip';

export const SettingsPage: React.FC = () => {
  const [settings, setSettings] = useState<CompanySettings | null>(null);
  const [appConfig, setAppConfig] = useState<AppConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [savedMessage, setSavedMessage] = useState(false);

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
    if (!settings) return;
    await Api.updateCompanySettings(settings);
    setSavedMessage(true);
    setTimeout(() => setSavedMessage(false), 3000);
  };

  if (loading || !settings) {
    return <div className="p-8 text-center text-stone-500 text-xs">Einstellungen werden geladen...</div>;
  }

  return (
    <div className="p-8 max-w-4xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold text-stone-900 tracking-tight flex items-center">
            Unternehmenseinstellungen & Stammdaten
            <HelpTooltip
              title="Stammdaten & GoBD"
              content="Diese Daten werden für ZUGFeRD-Rechnungen, den E-Bilanz-Export (GCD-Modul) und die Belegarchivierung verwendet."
            />
          </h2>
          <p className="text-xs text-stone-500 mt-1">
            Gespeichert in der lokalen SQLite-Datenbank &bull; Keine externen Cloud-Abhängigkeiten
          </p>
        </div>

        {savedMessage && (
          <div className="px-3 py-1.5 rounded-lg bg-emerald-50 border border-emerald-200 text-emerald-800 text-xs flex items-center gap-1.5 font-medium animate-in fade-in">
            <CheckCircle2 className="w-3.5 h-3.5 text-emerald-600" />
            Änderungen gespeichert
          </div>
        )}
      </div>

      <form onSubmit={handleSave} className="space-y-6">
        {/* Storage & Vault */}
        <div className="bg-white p-6 rounded-xl border border-stone-200/90 shadow-xs space-y-4">
          <h3 className="text-sm font-bold text-stone-900 flex items-center gap-2 border-b border-stone-100 pb-2">
            <FolderOpen className="w-4 h-4 text-amber-600" />
            Lokaler Speicherort & GoBD-Zertifikat
          </h3>

          <div className="space-y-3 text-xs">
            <div>
              <label className="font-semibold text-stone-700 block mb-1">Datenverzeichnis (SQLite & Belege):</label>
              <div className="flex gap-2">
                <input
                  type="text"
                  value={appConfig?.dataDir || ''}
                  readOnly
                  className="w-full p-2 bg-stone-50 border border-stone-200 rounded-lg font-mono text-stone-700"
                />
                <button
                  type="button"
                  onClick={handlePickDirectory}
                  className="px-3 py-2 rounded-lg bg-stone-100 hover:bg-stone-200 text-stone-800 font-semibold border border-stone-200 transition-colors shrink-0"
                >
                  Ordner wählen...
                </button>
              </div>
            </div>

            <div>
              <label className="font-semibold text-stone-700 block mb-1">GoBD Signaturzertifikat:</label>
              <div className="p-2.5 bg-stone-50 rounded-lg border border-stone-200 font-mono text-stone-600 flex items-center gap-2">
                <Shield className="w-3.5 h-3.5 text-emerald-600 shrink-0" />
                <span className="truncate">{appConfig?.certPath || 'Standard Ed25519 Zertifikat'}</span>
              </div>
            </div>
          </div>
        </div>

        {/* Company Identity */}
        <div className="bg-white p-6 rounded-xl border border-stone-200/90 shadow-xs space-y-4">
          <h3 className="text-sm font-bold text-stone-900 flex items-center gap-2 border-b border-stone-100 pb-2">
            <Building className="w-4 h-4 text-amber-600" />
            Unternehmensidentität & Steuern
          </h3>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-xs">
            <div>
              <label className="font-semibold text-stone-700 block mb-1">Firmenname:</label>
              <input
                type="text"
                value={settings.companyName}
                onChange={(e) => setSettings({ ...settings, companyName: e.target.value })}
                className="w-full p-2 bg-stone-50 border border-stone-200 rounded-lg focus:ring-2 focus:ring-amber-500/20"
              />
            </div>

            <div>
              <label className="font-semibold text-stone-700 block mb-1">Rechtsform:</label>
              <input
                type="text"
                value={settings.legalForm}
                onChange={(e) => setSettings({ ...settings, legalForm: e.target.value })}
                className="w-full p-2 bg-stone-50 border border-stone-200 rounded-lg focus:ring-2 focus:ring-amber-500/20"
              />
            </div>

            <div>
              <label className="font-semibold text-stone-700 block mb-1">Steuernummer:</label>
              <input
                type="text"
                value={settings.taxNumber}
                onChange={(e) => setSettings({ ...settings, taxNumber: e.target.value })}
                className="w-full p-2 bg-stone-50 border border-stone-200 rounded-lg focus:ring-2 focus:ring-amber-500/20 font-mono"
              />
            </div>

            <div>
              <label className="font-semibold text-stone-700 block mb-1">Umsatzsteuer-ID (USt-IdNr.):</label>
              <input
                type="text"
                value={settings.vatId}
                onChange={(e) => setSettings({ ...settings, vatId: e.target.value })}
                className="w-full p-2 bg-stone-50 border border-stone-200 rounded-lg focus:ring-2 focus:ring-amber-500/20 font-mono"
              />
            </div>

            <div className="md:col-span-2">
              <label className="font-semibold text-stone-700 block mb-1">Zuständiges Finanzamt:</label>
              <input
                type="text"
                value={settings.taxOffice}
                onChange={(e) => setSettings({ ...settings, taxOffice: e.target.value })}
                className="w-full p-2 bg-stone-50 border border-stone-200 rounded-lg focus:ring-2 focus:ring-amber-500/20"
              />
            </div>
          </div>
        </div>

        {/* Address */}
        <div className="bg-white p-6 rounded-xl border border-stone-200/90 shadow-xs space-y-4">
          <h3 className="text-sm font-bold text-stone-900 border-b border-stone-100 pb-2">
            Anschrift
          </h3>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-xs">
            <div className="md:col-span-2">
              <label className="font-semibold text-stone-700 block mb-1">Straße & Hausnummer:</label>
              <input
                type="text"
                value={settings.street}
                onChange={(e) => setSettings({ ...settings, street: e.target.value })}
                className="w-full p-2 bg-stone-50 border border-stone-200 rounded-lg focus:ring-2 focus:ring-amber-500/20"
              />
            </div>

            <div>
              <label className="font-semibold text-stone-700 block mb-1">PLZ & Stadt:</label>
              <input
                type="text"
                value={settings.zipCity}
                onChange={(e) => setSettings({ ...settings, zipCity: e.target.value })}
                className="w-full p-2 bg-stone-50 border border-stone-200 rounded-lg focus:ring-2 focus:ring-amber-500/20"
              />
            </div>

            <div>
              <label className="font-semibold text-stone-700 block mb-1">Land:</label>
              <input
                type="text"
                value={settings.country}
                onChange={(e) => setSettings({ ...settings, country: e.target.value })}
                className="w-full p-2 bg-stone-50 border border-stone-200 rounded-lg focus:ring-2 focus:ring-amber-500/20"
              />
            </div>
          </div>
        </div>

        {/* Banking */}
        <div className="bg-white p-6 rounded-xl border border-stone-200/90 shadow-xs space-y-4">
          <h3 className="text-sm font-bold text-stone-900 flex items-center gap-2 border-b border-stone-100 pb-2">
            <DollarSign className="w-4 h-4 text-amber-600" />
            Standard-Bankverbindung & EZB-Fremdwährung
          </h3>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-xs">
            <div>
              <label className="font-semibold text-stone-700 block mb-1">Bankname:</label>
              <input
                type="text"
                value={settings.bankName}
                onChange={(e) => setSettings({ ...settings, bankName: e.target.value })}
                className="w-full p-2 bg-stone-50 border border-stone-200 rounded-lg focus:ring-2 focus:ring-amber-500/20"
              />
            </div>

            <div>
              <label className="font-semibold text-stone-700 block mb-1">IBAN:</label>
              <input
                type="text"
                value={settings.iban}
                onChange={(e) => setSettings({ ...settings, iban: e.target.value })}
                className="w-full p-2 bg-stone-50 border border-stone-200 rounded-lg focus:ring-2 focus:ring-amber-500/20 font-mono"
              />
            </div>

            <div>
              <label className="font-semibold text-stone-700 block mb-1">BIC:</label>
              <input
                type="text"
                value={settings.bic}
                onChange={(e) => setSettings({ ...settings, bic: e.target.value })}
                className="w-full p-2 bg-stone-50 border border-stone-200 rounded-lg focus:ring-2 focus:ring-amber-500/20 font-mono"
              />
            </div>

            <div>
              <label className="font-semibold text-stone-700 block mb-1">Kontenrahmen:</label>
              <input
                type="text"
                disabled
                value={settings.skr}
                className="w-full p-2 bg-stone-100 border border-stone-200 rounded-lg font-mono text-stone-500 cursor-not-allowed"
              />
            </div>
          </div>
        </div>

        <div className="flex justify-end">
          <button
            type="submit"
            className="flex items-center gap-2 px-6 py-2.5 rounded-lg bg-amber-600 text-white text-xs font-semibold hover:bg-amber-700 transition-colors shadow-xs"
          >
            <Save className="w-4 h-4" />
            Einstellungen speichern
          </button>
        </div>
      </form>
    </div>
  );
};
