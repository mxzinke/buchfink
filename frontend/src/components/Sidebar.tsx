import React from 'react';
import {
  LayoutDashboard,
  Landmark,
  FileText,
  ListOrdered,
  Receipt,
  Users,
  BookOpen,
  Scale,
  FileSpreadsheet,
  ShieldCheck,
  ShieldAlert,
  Settings,
  RefreshCw,
  Calendar,
  X,
} from 'lucide-react';
import { CompanySettings, IntegrityCheckResult } from '../types';
import { GermanFlag } from './GermanFlag';

export type TabType =
  | 'welcome'
  | 'dashboard'
  | 'bank'
  | 'receipts'
  | 'invoices'
  | 'journal'
  | 'contacts'
  | 'accounts'
  | 'reports'
  | 'deadlines'
  | 'ebilanz'
  | 'audit'
  | 'settings';

interface SidebarProps {
  currentTab: TabType;
  onSelectTab: (tab: TabType) => void;
  settings: CompanySettings | null;
  integrity: IntegrityCheckResult | null;
  onRefreshIntegrity: () => void;
  isCheckingIntegrity: boolean;
  isOpenMobile?: boolean;
  onCloseMobile?: () => void;
}

interface NavGroup {
  label: string;
  items: {
    id: TabType;
    label: string;
    icon: React.ReactNode;
    badge?: string;
  }[];
}

export const Sidebar: React.FC<SidebarProps> = ({
  currentTab,
  onSelectTab,
  settings,
  integrity,
  onRefreshIntegrity,
  isCheckingIntegrity,
  isOpenMobile = false,
  onCloseMobile,
}) => {
  const groups: NavGroup[] = [
    {
      label: 'Übersicht',
      items: [
        { id: 'dashboard', label: 'Übersicht', icon: <LayoutDashboard className="w-4 h-4" /> },
      ],
    },
    {
      label: 'Buchhaltung',
      items: [
        { id: 'bank', label: 'Bank & Zahlungen', icon: <Landmark className="w-4 h-4" /> },
        { id: 'receipts', label: 'Belege', icon: <Receipt className="w-4 h-4" /> },
        { id: 'invoices', label: 'Rechnungen', icon: <FileText className="w-4 h-4" /> },
        { id: 'journal', label: 'Journal', icon: <ListOrdered className="w-4 h-4" /> },
      ],
    },
    {
      label: 'Stammdaten',
      items: [
        { id: 'contacts', label: 'Kontakte', icon: <Users className="w-4 h-4" /> },
        { id: 'accounts', label: 'Kontenübersicht', icon: <BookOpen className="w-4 h-4" /> },
      ],
    },
    {
      label: 'Auswertungen',
      items: [
        { id: 'reports', label: 'GuV & Bilanz', icon: <Scale className="w-4 h-4" /> },
        { id: 'deadlines', label: 'Steuerfristen', icon: <Calendar className="w-4 h-4" /> },
        { id: 'ebilanz', label: 'E-Bilanz', icon: <FileSpreadsheet className="w-4 h-4" /> },
      ],
    },
    {
      label: 'Verwaltung',
      items: [
        { id: 'audit', label: 'Sicherheit & Protokoll', icon: <ShieldCheck className="w-4 h-4" /> },
        { id: 'settings', label: 'Einstellungen', icon: <Settings className="w-4 h-4" /> },
      ],
    },
  ];

  const handleItemClick = (id: TabType) => {
    onSelectTab(id);
    if (onCloseMobile) {
      onCloseMobile();
    }
  };

  return (
    <>
      {/* Mobile Dimmed Backdrop */}
      {isOpenMobile && (
        <div
          className="fixed inset-0 bg-black/60 backdrop-blur-xs z-40 md:hidden transition-opacity"
          onClick={onCloseMobile}
        />
      )}

      {/* Sidebar Container */}
      <aside
        className={`fixed md:static inset-y-0 left-0 z-50 w-64 bg-[#24211E] text-stone-300 flex flex-col justify-between shrink-0 select-none border-r border-[#34302C] window-drag transition-transform duration-200 ease-in-out ${
          isOpenMobile ? 'translate-x-0 shadow-2xl' : '-translate-x-full md:translate-x-0'
        }`}
      >
        <div className="flex-1 flex flex-col min-h-0">
          {/* Top Header with macOS Window Button Spacing (pt-9 or pt-4 on mobile) */}
          <div className="pt-8 md:pt-9 pb-3 px-4 border-b border-[#34302C] flex items-center justify-between">
            <div
              onClick={() => handleItemClick('welcome')}
              className="flex items-center gap-3 p-1.5 -mx-1.5 rounded-xl hover:bg-white/5 transition-colors cursor-pointer group window-no-drag min-w-0 flex-1"
            >
              <div className="relative shrink-0">
                <img
                  src="/buchfink-logo.svg"
                  alt="Buchfink"
                  className="w-8 h-8 rounded-lg bg-white/10 p-0.5 border border-white/10 group-hover:scale-105 transition-transform"
                />
                <div className="absolute -bottom-1 -right-1">
                  <GermanFlag className="w-3.5 h-2.5 shadow-xs border border-stone-900" />
                </div>
              </div>
              <div className="min-w-0 flex-1">
                <div className="flex items-center justify-between">
                  <span className="font-bold text-sm text-stone-100 tracking-tight truncate">
                    {settings?.companyName || 'Buchfink'}
                  </span>
                </div>
                <p className="text-[11px] text-stone-400 truncate">
                  Geschäftsjahr {settings?.fiscalYear || new Date().getFullYear()}
                </p>
              </div>
            </div>

            {/* Mobile Close Button */}
            <button
              onClick={onCloseMobile}
              className="md:hidden p-1.5 text-stone-400 hover:text-white rounded-lg hover:bg-white/10 transition-colors ml-2 window-no-drag"
              title="Menü schließen"
            >
              <X className="w-5 h-5" />
            </button>
          </div>

          {/* Navigation Groups */}
          <nav className="flex-1 overflow-y-auto p-3 space-y-4 window-no-drag">
            {groups.map((group) => (
              <div key={group.label} className="space-y-1">
                <div className="px-2.5 py-1 text-[10px] font-bold uppercase tracking-wider text-stone-500">
                  {group.label}
                </div>
                {group.items.map((item) => {
                  const isActive = currentTab === item.id;
                  return (
                    <button
                      key={item.id}
                      onClick={() => handleItemClick(item.id)}
                      className={`w-full flex items-center justify-between px-2.5 py-1.5 rounded-lg text-xs font-medium transition-all cursor-pointer ${
                        isActive
                          ? 'bg-amber-500/15 text-amber-200 border border-amber-500/30 shadow-xs'
                          : 'text-stone-400 hover:text-stone-100 hover:bg-white/5'
                      }`}
                    >
                      <div className="flex items-center gap-2.5 min-w-0">
                        <span className={isActive ? 'text-amber-300' : 'text-stone-400 shrink-0'}>
                          {item.icon}
                        </span>
                        <span className="truncate">{item.label}</span>
                      </div>
                      {item.badge && (
                        <span
                          className={`text-[9px] px-1.5 py-0.2 rounded font-medium ${
                            isActive
                              ? 'bg-amber-500/30 text-amber-200'
                              : 'bg-stone-800 text-stone-400'
                          }`}
                        >
                          {item.badge}
                        </span>
                      )}
                    </button>
                  );
                })}
              </div>
            ))}
          </nav>
        </div>

        {/* Footer: Integrity & Storage Info */}
        <div className="p-3 border-t border-[#34302C] bg-[#1D1B19]/70 text-[11px] space-y-2 window-no-drag">
          {/* Integrity Status Bar */}
          <div
            className="flex items-center justify-between px-2.5 py-1.5 rounded-lg bg-[#24211E] border border-[#38332F] text-[11px]"
            title={integrity?.message || 'Status der Datenintegrität'}
          >
            <div className="flex items-center gap-2 truncate">
              {integrity?.isValid ? (
                <ShieldCheck className="w-3.5 h-3.5 text-emerald-400 shrink-0" />
              ) : (
                <ShieldAlert className="w-3.5 h-3.5 text-rose-400 shrink-0" />
              )}
              <span className={`truncate ${integrity?.isValid ? 'text-stone-300' : 'text-rose-300'}`}>
                {integrity?.isValid ? 'Daten unverändert' : 'Prüfung erforderlich'}
              </span>
            </div>
            <button
              onClick={onRefreshIntegrity}
              disabled={isCheckingIntegrity}
              className="text-stone-500 hover:text-stone-300 p-0.5 rounded transition-colors"
              title="Prüfung aktualisieren"
            >
              <RefreshCw className={`w-3 h-3 ${isCheckingIntegrity ? 'animate-spin text-amber-400' : ''}`} />
            </button>
          </div>

          <div className="flex items-center justify-between text-[11px] text-stone-500 px-1">
            <span>Lokal auf diesem Gerät</span>
            <span className="text-emerald-400 font-medium">● Bereit</span>
          </div>
        </div>
      </aside>
    </>
  );
};
