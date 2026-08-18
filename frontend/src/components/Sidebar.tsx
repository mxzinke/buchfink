import React from 'react';
import {
  LayoutDashboard,
  Landmark,
  FileText,
  ListOrdered,
  Users,
  BookOpen,
  Scale,
  FileSpreadsheet,
  ShieldCheck,
  Settings,
  Sparkles,
} from 'lucide-react';
import { CompanySettings } from '../types';

export type TabType =
  | 'welcome'
  | 'dashboard'
  | 'bank'
  | 'invoices'
  | 'journal'
  | 'contacts'
  | 'accounts'
  | 'reports'
  | 'ebilanz'
  | 'audit'
  | 'settings';

interface SidebarProps {
  currentTab: TabType;
  onSelectTab: (tab: TabType) => void;
  settings: CompanySettings | null;
  onOpenOnboarding: () => void;
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
  onOpenOnboarding,
}) => {
  const groups: NavGroup[] = [
    {
      label: 'Übersicht',
      items: [
        { id: 'dashboard', label: 'Dashboard', icon: <LayoutDashboard className="w-4 h-4" /> },
      ],
    },
    {
      label: 'Laufendes Geschäft',
      items: [
        { id: 'bank', label: 'Bank & Kasse', icon: <Landmark className="w-4 h-4" />, badge: 'Auto' },
        { id: 'invoices', label: 'Rechnungen', icon: <FileText className="w-4 h-4" />, badge: 'ZUGFeRD' },
        { id: 'journal', label: 'Buchungsjournal', icon: <ListOrdered className="w-4 h-4" /> },
      ],
    },
    {
      label: 'Stammdaten',
      items: [
        { id: 'contacts', label: 'Kunden & Lieferanten', icon: <Users className="w-4 h-4" /> },
        { id: 'accounts', label: 'SKR04 Kontenplan', icon: <BookOpen className="w-4 h-4" /> },
      ],
    },
    {
      label: 'Abschluss & Steuern',
      items: [
        { id: 'reports', label: 'Bilanz & GuV', icon: <Scale className="w-4 h-4" /> },
        { id: 'ebilanz', label: 'E-Bilanz (XBRL)', icon: <FileSpreadsheet className="w-4 h-4" /> },
      ],
    },
    {
      label: 'System',
      items: [
        { id: 'audit', label: 'GoBD Prüfpfad', icon: <ShieldCheck className="w-4 h-4" /> },
        { id: 'settings', label: 'Einstellungen', icon: <Settings className="w-4 h-4" /> },
      ],
    },
  ];

  return (
    <aside className="w-64 bg-stone-900 text-stone-200 flex flex-col justify-between shrink-0 select-none border-r border-stone-800/90 window-drag">
      <div className="flex-1 flex flex-col min-h-0">
        {/* Top Header with macOS Window Button Spacing (pt-9) */}
        <div className="pt-9 pb-3 px-4 border-b border-stone-800/80">
          <div
            onClick={() => onSelectTab('welcome')}
            className="flex items-center gap-3 p-1.5 -mx-1.5 rounded-xl hover:bg-stone-800/60 transition-colors cursor-pointer group window-no-drag"
          >
            <img
              src="/buchfink-logo.svg"
              alt="Buchfink"
              className="w-8 h-8 rounded-lg bg-white/5 p-0.5 border border-white/10 group-hover:scale-105 transition-transform shrink-0"
            />
            <div className="min-w-0 flex-1">
              <div className="flex items-center justify-between">
                <span className="font-bold text-sm text-stone-100 tracking-tight truncate">
                  {settings?.companyName || 'Buchfink'}
                </span>
              </div>
              <p className="text-[10px] text-stone-400 font-mono truncate">
                SKR04 &bull; {settings?.fiscalYear || 2024}
              </p>
            </div>
          </div>
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
                    onClick={() => onSelectTab(item.id)}
                    className={`w-full flex items-center justify-between px-2.5 py-1.5 rounded-lg text-xs font-medium transition-all ${
                      isActive
                        ? 'bg-amber-600/20 text-amber-300 border border-amber-500/30 shadow-xs'
                        : 'text-stone-400 hover:text-stone-100 hover:bg-stone-800/50'
                    }`}
                  >
                    <div className="flex items-center gap-2.5 min-w-0">
                      <span className={isActive ? 'text-amber-400' : 'text-stone-400 shrink-0'}>
                        {item.icon}
                      </span>
                      <span className="truncate">{item.label}</span>
                    </div>
                    {item.badge && (
                      <span
                        className={`text-[9px] px-1.5 py-0.2 rounded font-mono ${
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

      {/* Footer / Onboarding & Info */}
      <div className="p-3 border-t border-stone-800/80 bg-stone-950/40 text-[11px] space-y-2 window-no-drag">
        <button
          onClick={onOpenOnboarding}
          className="w-full flex items-center justify-center gap-1.5 px-2.5 py-1.5 rounded-lg bg-stone-800 hover:bg-stone-700 text-stone-300 hover:text-white text-[11px] font-medium transition-colors border border-stone-700/60"
        >
          <Sparkles className="w-3.5 h-3.5 text-amber-400" />
          <span>Setup-Assistent</span>
        </button>

        <div className="flex items-center justify-between text-[10px] text-stone-500 font-mono px-1">
          <span>Pure SQLite</span>
          <span className="text-emerald-400 font-sans">● Bereit</span>
        </div>
      </div>
    </aside>
  );
};
