// SPDX-FileCopyrightText: 2026 Maximilian P.
// SPDX-License-Identifier: EUPL-1.2

import React from 'react';
import { toast } from 'sonner';
import {
  ArrowRight,
  Landmark,
  FileText,
  BookOpen,
  ListOrdered,
  Building2,
  Plus,
  Trash2,
  Check,
} from 'lucide-react';
import { CompanySettings, TenantConfig } from '../types';
import { Api } from '../services/api';
import { GermanFlag } from './GermanFlag';

interface StartupScreenProps {
  settings: CompanySettings | null;
  tenants: TenantConfig[];
  activeTenant: TenantConfig | null;
  onSwitchTenant: (tenantId: string) => Promise<void>;
  onRefreshTenants: () => Promise<void>;
  onAddTenant: () => void;
  onStartDashboard: () => void;
  onNavigate: (tab: any) => void;
}

export const StartupScreen: React.FC<StartupScreenProps> = ({
  settings,
  tenants,
  activeTenant,
  onSwitchTenant,
  onRefreshTenants,
  onAddTenant,
  onStartDashboard,
  onNavigate,
}) => {
  const currentCalendarYear = new Date().getFullYear();

  const handleDeleteTenant = async (e: React.MouseEvent, tenantId: string, name: string) => {
    e.stopPropagation();
    if (tenants.length <= 1) {
      toast.error('Löschen nicht möglich', {
        description: 'Mindestens ein Mandant muss erhalten bleiben.',
      });
      return;
    }
    if (!confirm(`Möchten Sie den Mandanten "${name}" wirklich aus der Liste entfernen? (Ihre Buchhaltungsdaten auf der Festplatte bleiben unberührt).`)) {
      return;
    }
    try {
      await Api.deleteTenant(tenantId);
      toast.success('Mandant entfernt', {
        description: `Mandant "${name}" wurde aus der Liste entfernt.`,
      });
      await onRefreshTenants();
    } catch (e: any) {
      toast.error('Fehler beim Entfernen des Mandanten');
    }
  };

  const activeCompanyName = settings?.companyName || activeTenant?.name || 'Hauptmandant';
  const activeYear = settings?.fiscalYear || currentCalendarYear;

  return (
    <div className="relative min-h-full flex flex-col justify-between overflow-y-auto bg-stone-900 text-stone-100">
      {/* Background with warm atmospheric view */}
      <div
        className="absolute inset-0 bg-cover bg-center opacity-85 pointer-events-none scale-100 transition-all duration-700"
        style={{ backgroundImage: "url('/bg-startupscreen_unsplash-steven-kamenar.jpg')" }}
      />
      <div className="absolute inset-0 bg-gradient-to-t from-[#1A1816]/90 via-[#1F1C1A]/60 to-[#1A1816]/40 pointer-events-none" />

      {/* Main Container */}
      <div className="relative z-10 max-w-4xl mx-auto w-full px-6 py-12 flex-1 flex flex-col justify-center space-y-6">
        {/* Clean Header */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3.5">
            <div className="relative">
              <img
                src="/buchfink-logo.svg"
                alt="Buchfink Logo"
                className="w-13 h-13 drop-shadow-lg rounded-2xl bg-white/15 p-1.5 border border-white/20 backdrop-blur-md"
              />
              <div className="absolute -bottom-1 -right-1">
                <GermanFlag className="w-4 h-3 shadow-md border border-[#1A1816] rounded-xs" />
              </div>
            </div>
            <div>
              <h1 className="text-2xl font-extrabold text-white tracking-tight drop-shadow-sm font-sans">
                Buchfink
              </h1>
              <p className="text-xs text-stone-300 font-medium">
                Doppelte Buchführung &amp; Bilanzierung (SKR04)
              </p>
            </div>
          </div>

          {/* Single clean button to add / import tenant via Setup Assistant */}
          <button
            onClick={onAddTenant}
            className="px-4 py-2 rounded-xl bg-amber-700 hover:bg-amber-600 text-white font-semibold text-xs transition-all flex items-center gap-1.5 shadow-lg shadow-amber-900/30"
          >
            <Plus className="w-4 h-4" />
            <span>Mandant hinzufügen</span>
          </button>
        </div>

        {/* Unified Main Card */}
        <div className="bg-[#24211E]/85 backdrop-blur-xl border border-white/15 rounded-2xl p-6 shadow-2xl space-y-6">
          {/* Active Mandant Header */}
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-white/10 pb-5">
            <div className="space-y-1">
              <h2 className="text-2xl font-bold text-white tracking-tight">
                {activeCompanyName}
              </h2>

              <div className="flex flex-wrap items-center gap-x-2.5 gap-y-1 text-xs text-stone-300">
                <span className="text-amber-300 font-medium">Geschäftsjahr {activeYear}</span>
                <span>&bull;</span>
                <span>{settings?.legalForm || 'GmbH'}</span>
                <span>&bull;</span>
                <span>{settings?.taxationType || 'SOLL'}-Besteuerung</span>
                {settings?.taxNumber && (
                  <>
                    <span>&bull;</span>
                    <span className="font-mono text-stone-400">St.-Nr. {settings.taxNumber}</span>
                  </>
                )}
                {settings?.iban && (
                  <>
                    <span>&bull;</span>
                    <span className="font-mono text-stone-400">IBAN: {settings.iban}</span>
                  </>
                )}
              </div>
            </div>

            <button
              onClick={onStartDashboard}
              className="px-6 py-3 rounded-xl bg-amber-700 hover:bg-amber-600 text-white font-semibold text-sm transition-all shadow-lg shadow-amber-900/40 flex items-center gap-2 group shrink-0"
            >
              <span>Zur Buchhaltung</span>
              <ArrowRight className="w-4 h-4 group-hover:translate-x-0.5 transition-transform" />
            </button>
          </div>

          {/* Mandanten-Liste */}
          <div className="space-y-2.5">
            <div className="flex items-center justify-between text-xs text-stone-400 px-0.5">
              <span className="font-semibold text-stone-300 text-xs flex items-center gap-1.5">
                <Building2 className="w-3.5 h-3.5 text-amber-400" />
                Mandanten ({tenants.length})
              </span>
              <span className="text-[11px] text-stone-400">Klick auf Zeile zum Mandantenwechsel</span>
            </div>

            <div className="bg-[#1D1B19]/80 rounded-xl border border-white/10 divide-y divide-white/5 overflow-hidden">
              {tenants.map((t) => {
                const isActive = t.id === activeTenant?.id;
                return (
                  <div
                    key={t.id}
                    onClick={() => onSwitchTenant(t.id)}
                    className={`px-3.5 py-2.5 flex items-center justify-between gap-3 transition-colors cursor-pointer ${
                      isActive ? 'bg-amber-500/10' : 'hover:bg-white/5'
                    }`}
                  >
                    <div className="flex items-center gap-3 min-w-0">
                      {/* Active Indicator Radio */}
                      <div className="shrink-0 flex items-center justify-center">
                        {isActive ? (
                          <div className="w-2 h-2 rounded-full bg-amber-400 ring-4 ring-amber-400/20" />
                        ) : (
                          <div className="w-2 h-2 rounded-full bg-stone-600" />
                        )}
                      </div>

                      <div className="min-w-0">
                        <div className="flex items-center gap-2">
                          <span className={`text-xs font-semibold truncate ${isActive ? 'text-amber-200 font-bold' : 'text-stone-200'}`}>
                            {t.name}
                          </span>
                          {isActive && (
                            <span className="text-[10px] text-amber-400/90 font-medium">
                              (aktiv)
                            </span>
                          )}
                        </div>
                        <div className="text-[11px] text-stone-400 font-mono truncate" title={t.dataDir}>
                          {t.dataDir}
                        </div>
                      </div>
                    </div>

                    <div className="flex items-center gap-2 shrink-0">
                      {!isActive ? (
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            onSwitchTenant(t.id);
                          }}
                          className="px-2.5 py-1 rounded-md bg-stone-800 hover:bg-stone-700 text-stone-300 hover:text-white text-[11px] font-medium border border-stone-700 transition-colors"
                        >
                          Öffnen
                        </button>
                      ) : (
                        <span className="text-[11px] text-emerald-400 font-medium flex items-center gap-1">
                          <Check className="w-3 h-3" /> Ausgewählt
                        </span>
                      )}

                      {tenants.length > 1 && (
                        <button
                          onClick={(e) => handleDeleteTenant(e, t.id, t.name)}
                          className="p-1 text-stone-500 hover:text-rose-400 transition-colors rounded"
                          title="Aus der Liste entfernen"
                        >
                          <Trash2 className="w-3 h-3" />
                        </button>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          </div>

          {/* Quick Launchers */}
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-2 pt-1 border-t border-white/10">
            <button
              onClick={() => onNavigate('bank')}
              className="p-2.5 rounded-xl bg-[#1D1B19]/60 hover:bg-[#1D1B19]/90 border border-white/5 hover:border-amber-400/40 text-left transition-all flex items-center gap-2.5 group"
            >
              <div className="w-7 h-7 rounded-lg bg-amber-500/15 text-amber-300 flex items-center justify-center shrink-0">
                <Landmark className="w-3.5 h-3.5" />
              </div>
              <div className="min-w-0">
                <div className="text-xs font-semibold text-white group-hover:text-amber-300 truncate">
                  Bankumsätze
                </div>
                <div className="text-[10px] text-stone-400 truncate">Kontoauszug abgleichen</div>
              </div>
            </button>

            <button
              onClick={() => onNavigate('invoices')}
              className="p-2.5 rounded-xl bg-[#1D1B19]/60 hover:bg-[#1D1B19]/90 border border-white/5 hover:border-amber-400/40 text-left transition-all flex items-center gap-2.5 group"
            >
              <div className="w-7 h-7 rounded-lg bg-amber-500/15 text-amber-300 flex items-center justify-center shrink-0">
                <FileText className="w-3.5 h-3.5" />
              </div>
              <div className="min-w-0">
                <div className="text-xs font-semibold text-white group-hover:text-amber-300 truncate">
                  Rechnungen
                </div>
                <div className="text-[10px] text-stone-400 truncate">E-Rechnung &amp; PDF</div>
              </div>
            </button>

            <button
              onClick={() => onNavigate('journal')}
              className="p-2.5 rounded-xl bg-[#1D1B19]/60 hover:bg-[#1D1B19]/90 border border-white/5 hover:border-amber-400/40 text-left transition-all flex items-center gap-2.5 group"
            >
              <div className="w-7 h-7 rounded-lg bg-amber-500/15 text-amber-300 flex items-center justify-center shrink-0">
                <ListOrdered className="w-3.5 h-3.5" />
              </div>
              <div className="min-w-0">
                <div className="text-xs font-semibold text-white group-hover:text-amber-300 truncate">
                  Journal
                </div>
                <div className="text-[10px] text-stone-400 truncate">Buchungssätze erfassen</div>
              </div>
            </button>

            <button
              onClick={() => onNavigate('accounts')}
              className="p-2.5 rounded-xl bg-[#1D1B19]/60 hover:bg-[#1D1B19]/90 border border-white/5 hover:border-amber-400/40 text-left transition-all flex items-center gap-2.5 group"
            >
              <div className="w-7 h-7 rounded-lg bg-amber-500/15 text-amber-300 flex items-center justify-center shrink-0">
                <BookOpen className="w-3.5 h-3.5" />
              </div>
              <div className="min-w-0">
                <div className="text-xs font-semibold text-white group-hover:text-amber-300 truncate">
                  Kontenübersicht
                </div>
                <div className="text-[10px] text-stone-400 truncate">SKR04 Saldenübersicht</div>
              </div>
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
