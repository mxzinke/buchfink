import React from 'react';
import {
  ArrowRight,
  ShieldCheck,
  Landmark,
  FileText,
  BookOpen,
  Scale,
  FolderOpen,
} from 'lucide-react';
import { CompanySettings } from '../types';
import { GermanFlag } from './GermanFlag';

interface StartupScreenProps {
  settings: CompanySettings | null;
  onStartDashboard: () => void;
  onNavigate: (tab: any) => void;
}

export const StartupScreen: React.FC<StartupScreenProps> = ({
  settings,
  onStartDashboard,
  onNavigate,
}) => {
  return (
    <div className="relative min-h-full flex flex-col justify-between overflow-y-auto bg-stone-900 text-stone-100">
      {/* Background image with warm darkening overlay */}
      <div
        className="absolute inset-0 bg-cover bg-center opacity-25 mix-blend-luminosity pointer-events-none scale-105 filter blur-xs"
        style={{ backgroundImage: "url('/bg-startupscreen_unsplash-steven-kamenar.jpg')" }}
      />
      <div className="absolute inset-0 bg-gradient-to-t from-stone-950 via-stone-900/80 to-stone-950/70 pointer-events-none" />

      {/* Main Container */}
      <div className="relative z-10 max-w-5xl mx-auto w-full px-8 py-12 flex-1 flex flex-col justify-center space-y-10">
        {/* Header Branding */}
        <div className="space-y-4">
          <div className="flex items-center gap-4">
            <div className="relative">
              <img
                src="/buchfink-logo.svg"
                alt="Buchfink Logo"
                className="w-16 h-16 drop-shadow-md rounded-2xl bg-white/10 p-1 border border-white/10 backdrop-blur-md"
              />
              <div className="absolute -bottom-1.5 -right-1.5">
                <GermanFlag className="w-5 h-3.5 shadow-md border-2 border-stone-900 rounded-xs" />
              </div>
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h1 className="text-3xl font-extrabold tracking-tight text-white font-sans">
                  Buchfink
                </h1>
                <span className="px-2.5 py-0.5 rounded-full text-xs font-semibold bg-stone-800 text-stone-300 border border-stone-700">
                  GoBD-konform
                </span>
                <span className="px-2.5 py-0.5 rounded-full text-[11px] font-medium bg-emerald-500/20 text-emerald-300 border border-emerald-500/30">
                  100% Local-First
                </span>
              </div>
              <p className="text-sm text-stone-300 mt-1 font-medium">
                Moderne Buchhaltungssoftware für kleine Unternehmen & Selbstständige in Deutschland
              </p>
            </div>
          </div>
        </div>

        {/* Hero Card with Status and Primary CTA */}
        <div className="bg-stone-800/80 backdrop-blur-xl border border-stone-700/80 rounded-2xl p-6 shadow-2xl space-y-6">
          <div className="flex flex-col md:flex-row items-start md:items-center justify-between gap-4 border-b border-stone-700/60 pb-5">
            <div>
              <span className="text-[11px] font-bold tracking-wider text-amber-400 uppercase">
                Aktiver Mandant &bull; Geschäftsjahr {settings?.fiscalYear || new Date().getFullYear()}
              </span>
              <h2 className="text-xl font-bold text-white mt-0.5">
                {settings?.companyName || 'Musterfirma GmbH'}
              </h2>
              <div className="flex items-center gap-3 text-xs text-stone-400 mt-1 font-mono">
                <span>St.-Nr.: {settings?.taxNumber || '12/345/67890'}</span>
                <span>&bull;</span>
                <span>IBAN: {settings?.iban || 'DE89...3000'}</span>
              </div>
            </div>

            <button
              onClick={onStartDashboard}
              className="px-6 py-3 rounded-xl bg-amber-600 hover:bg-amber-500 text-white font-bold text-sm transition-all shadow-lg shadow-amber-600/30 flex items-center gap-2 group shrink-0"
            >
              <span>Zur Buchhaltungsübersicht</span>
              <ArrowRight className="w-4 h-4 group-hover:translate-x-1 transition-transform" />
            </button>
          </div>

          {/* Quick Flow Launchers */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div
              onClick={() => onNavigate('bank')}
              className="p-4 rounded-xl bg-stone-900/80 border border-stone-700/60 hover:border-amber-500/50 hover:bg-stone-900 transition-all cursor-pointer group space-y-2"
            >
              <div className="w-8 h-8 rounded-lg bg-amber-500/20 text-amber-400 flex items-center justify-center border border-amber-500/30 group-hover:scale-110 transition-transform">
                <Landmark className="w-4 h-4" />
              </div>
              <h3 className="font-bold text-sm text-white flex items-center justify-between">
                <span>Bankumsätze abstimmen</span>
                <ArrowRight className="w-3.5 h-3.5 text-stone-500 group-hover:text-amber-400 transition-colors" />
              </h3>
              <p className="text-xs text-stone-400 leading-snug">
                CAMT.053 Kontoauszüge importieren und Buchungen automatisch mit einem Klick auslösen.
              </p>
            </div>

            <div
              onClick={() => onNavigate('invoices')}
              className="p-4 rounded-xl bg-stone-900/80 border border-stone-700/60 hover:border-amber-500/50 hover:bg-stone-900 transition-all cursor-pointer group space-y-2"
            >
              <div className="w-8 h-8 rounded-lg bg-amber-500/20 text-amber-400 flex items-center justify-center border border-amber-500/30 group-hover:scale-110 transition-transform">
                <FileText className="w-4 h-4" />
              </div>
              <h3 className="font-bold text-sm text-white flex items-center justify-between">
                <span>ZUGFeRD Rechnung</span>
                <ArrowRight className="w-3.5 h-3.5 text-stone-500 group-hover:text-amber-400 transition-colors" />
              </h3>
              <p className="text-xs text-stone-400 leading-snug">
                E-Rechnungskonforme PDF/A-3 Rechnungen mit integrierter EN 16931 XML erstellen.
              </p>
            </div>

            <div
              onClick={() => onNavigate('accounts')}
              className="p-4 rounded-xl bg-stone-900/80 border border-stone-700/60 hover:border-amber-500/50 hover:bg-stone-900 transition-all cursor-pointer group space-y-2"
            >
              <div className="w-8 h-8 rounded-lg bg-amber-500/20 text-amber-400 flex items-center justify-center border border-amber-500/30 group-hover:scale-110 transition-transform">
                <BookOpen className="w-4 h-4" />
              </div>
              <h3 className="font-bold text-sm text-white flex items-center justify-between">
                <span>Kontenplan</span>
                <ArrowRight className="w-3.5 h-3.5 text-stone-500 group-hover:text-amber-400 transition-colors" />
              </h3>
              <p className="text-xs text-stone-400 leading-snug">
                Deutsche Standardkonten einsehen, Hilfserklärungen lesen und Salden prüfen.
              </p>
            </div>
          </div>
        </div>

        {/* Feature Highlights Footer */}
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-6 text-xs text-stone-400 border-t border-stone-800/80 pt-6">
          <div className="flex items-start gap-3">
            <ShieldCheck className="w-5 h-5 text-emerald-400 shrink-0 mt-0.5" />
            <div>
              <p className="font-semibold text-stone-200">GoBD-konforme Hash-Chain</p>
              <p className="text-[11px] text-stone-400 mt-0.5">
                Jede Zeile kryptografisch verkettet. Keine nachträgliche Manipulation möglich.
              </p>
            </div>
          </div>

          <div className="flex items-start gap-3">
            <FolderOpen className="w-5 h-5 text-amber-400 shrink-0 mt-0.5" />
            <div>
              <p className="font-semibold text-stone-200">100% Local-First</p>
              <p className="text-[11px] text-stone-400 mt-0.5">
                Eine SQLite-Datei pro Geschäftsjahr. Ihre sensiblen Finanzdaten bleiben bei Ihnen.
              </p>
            </div>
          </div>

          <div className="flex items-start gap-3">
            <Scale className="w-5 h-5 text-sky-400 shrink-0 mt-0.5" />
            <div>
              <p className="font-semibold text-stone-200">E-Bilanz & XBRL 6.7</p>
              <p className="text-[11px] text-stone-400 mt-0.5">
                Direkter XBRL-Export mit Kontennachweis zur Einreichung in Mein ELSTER.
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
