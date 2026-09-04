import React from 'react';
import {
  BookCheck,
  BookOpen,
  Building2,
  Calendar,
  FileSpreadsheet,
  FileText,
  Landmark,
  LayoutDashboard,
  ListOrdered,
  Loader2,
  Receipt,
  Scale,
  Settings,
  ShieldAlert,
  ShieldCheck,
  Users,
  X,
} from 'lucide-react';
import { CompanySettings, IntegrityCheckResult } from '../types';
import { GermanFlag } from './GermanFlag';
import { Button, cn } from './ui';

export type TabType =
  | 'welcome'
  | 'dashboard'
  | 'bank'
  | 'receipts'
  | 'invoices'
  | 'journal'
  | 'assets'
  | 'contacts'
  | 'accounts'
  | 'reports'
  | 'closing'
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
  items: { id: TabType; label: string; icon: React.ReactNode }[];
}

const icon = 'w-4 h-4 shrink-0';

const GROUPS: NavGroup[] = [
  {
    label: 'Übersicht',
    items: [{ id: 'dashboard', label: 'Übersicht', icon: <LayoutDashboard className={icon} /> }],
  },
  {
    label: 'Buchhaltung',
    items: [
      { id: 'bank', label: 'Bank & Zahlungen', icon: <Landmark className={icon} /> },
      { id: 'receipts', label: 'Belege', icon: <Receipt className={icon} /> },
      { id: 'invoices', label: 'Rechnungen', icon: <FileText className={icon} /> },
      { id: 'journal', label: 'Journal', icon: <ListOrdered className={icon} /> },
      { id: 'assets', label: 'Anlagevermögen', icon: <Building2 className={icon} /> },
    ],
  },
  {
    label: 'Stammdaten',
    items: [
      { id: 'contacts', label: 'Kontakte', icon: <Users className={icon} /> },
      { id: 'accounts', label: 'Kontenübersicht', icon: <BookOpen className={icon} /> },
    ],
  },
  {
    label: 'Auswertungen',
    items: [
      { id: 'reports', label: 'GuV & Bilanz', icon: <Scale className={icon} /> },
      { id: 'closing', label: 'Jahresabschluss', icon: <BookCheck className={icon} /> },
      { id: 'deadlines', label: 'Steuerfristen', icon: <Calendar className={icon} /> },
      { id: 'ebilanz', label: 'E-Bilanz', icon: <FileSpreadsheet className={icon} /> },
    ],
  },
  {
    label: 'Verwaltung',
    items: [
      { id: 'audit', label: 'Sicherheit & Protokoll', icon: <ShieldCheck className={icon} /> },
      { id: 'settings', label: 'Einstellungen', icon: <Settings className={icon} /> },
    ],
  },
];

const NAV_ITEM =
  'w-full flex items-center gap-2 h-8 pl-1.5 pr-2 rounded-control ' +
  'text-body text-left transition-colors duration-120 ease-quiet';

const NavItem: React.FC<{
  item: NavGroup['items'][number];
  isActive: boolean;
  onSelect: () => void;
}> = ({ item, isActive, onSelect }) => (
  <button
    type="button"
    onClick={onSelect}
    aria-current={isActive ? 'page' : undefined}
    className={cn(
      NAV_ITEM,
      isActive
        ? 'bg-shell-raised text-white font-medium'
        : 'text-shell-text-muted hover:bg-shell-raised hover:text-shell-text',
    )}
  >
    {/* Die Markierung ist eine eigene Pille, keine einseitige Border: Eine
        Border folgt dem Eckenradius und wird an den Enden krumm. */}
    <span
      className={cn(
        'w-0.5 h-4 shrink-0 rounded-full',
        isActive ? 'bg-accent-light' : 'bg-transparent',
      )}
    />
    <span className="shrink-0 ml-0.5">{item.icon}</span>
    <span className="truncate">{item.label}</span>
  </button>
);

/** Zeigt nur die Uhrzeit, das Datum steht ohnehin im Kontext des Arbeitstags. */
function formatCheckedAt(iso: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return '';
  return new Intl.DateTimeFormat('de-DE', { hour: '2-digit', minute: '2-digit' }).format(date);
}

/**
 * Der Integritätszustand steht dauerhaft hier und nie in einem Toast (§11.4).
 * Drei Zustände, drei Formulierungen.
 */
const IntegrityStatus: React.FC<{
  integrity: IntegrityCheckResult | null;
  isChecking: boolean;
  onRefresh: () => void;
}> = ({ integrity, isChecking, onRefresh }) => {
  const broken = integrity !== null && !integrity.isValid;

  return (
    <button
      type="button"
      onClick={onRefresh}
      title={integrity?.message || 'Integrität der Buchungskette prüfen'}
      className="w-full flex items-start gap-2.5 px-2 py-1.5 rounded-control text-left
                 transition-colors duration-120 ease-quiet hover:bg-shell-raised"
    >
      <span className="mt-1 shrink-0">
        {isChecking ? (
          <Loader2 className="w-3.5 h-3.5 animate-spin text-shell-text-muted" strokeWidth={1.5} />
        ) : broken ? (
          <ShieldAlert className="w-3.5 h-3.5 text-shell-negative" strokeWidth={1.5} />
        ) : (
          <span className="mark-diamond bg-shell-positive block" />
        )}
      </span>
      <span className="min-w-0">
        <span className={cn('block text-caption', broken ? 'text-shell-negative' : 'text-shell-text')}>
          {isChecking ? 'Prüfung läuft' : broken ? 'Integrität verletzt' : 'Daten unverändert'}
        </span>
        {integrity?.checkedAt && !isChecking && (
          <span className="block text-caption text-shell-text-muted num">
            Geprüft um {formatCheckedAt(integrity.checkedAt)} Uhr
          </span>
        )}
      </span>
    </button>
  );
};

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
  const handleItemClick = (id: TabType) => {
    onSelectTab(id);
    onCloseMobile?.();
  };

  return (
    <>
      {isOpenMobile && (
        <div
          className="fixed inset-0 z-40 bg-ink/60 md:hidden transition-opacity duration-180 ease-quiet"
          onClick={onCloseMobile}
        />
      )}

      <aside
        className={cn(
          'fixed md:static inset-y-0 left-0 z-50 w-60 shrink-0 select-none',
          'flex flex-col justify-between bg-shell text-shell-text',
          'border-r border-shell-line window-drag',
          'transition-transform duration-180 ease-quiet',
          isOpenMobile ? 'translate-x-0 shadow-dialog' : '-translate-x-full md:translate-x-0',
        )}
      >
        <div className="flex-1 flex flex-col min-h-0">
          {/* Oben Platz für die Fensterknöpfe unter macOS */}
          <div className="pt-8 md:pt-9 pb-3 px-3 border-b border-shell-line flex items-center gap-2">
            <button
              type="button"
              onClick={() => handleItemClick('welcome')}
              className="flex items-center gap-3 flex-1 min-w-0 p-1.5 -m-1.5 rounded-control
                         transition-colors duration-120 ease-quiet hover:bg-shell-raised window-no-drag"
            >
              <span className="relative shrink-0">
                <img
                  src="/buchfink-logo.svg"
                  alt=""
                  className="w-8 h-8 rounded-control bg-white/10 p-0.5 border border-white/10"
                />
                <span className="absolute -bottom-1 -right-1">
                  <GermanFlag className="w-3.5 h-2.5 border border-shell" />
                </span>
              </span>
              <span className="min-w-0 flex-1 text-left">
                <span className="block text-body font-semibold text-white truncate">
                  {settings?.companyName || 'Buchfink'}
                </span>
                <span className="block text-caption text-shell-text-muted truncate">
                  Geschäftsjahr {settings?.fiscalYear || new Date().getFullYear()}
                </span>
              </span>
            </button>

            {onCloseMobile && (
              <Button
                variant="quiet"
                size="sm"
                iconOnly
                onClick={onCloseMobile}
                title="Menü schließen"
                aria-label="Menü schließen"
                className="md:hidden text-shell-text-muted hover:bg-shell-raised hover:text-white window-no-drag"
              >
                <X className={icon} strokeWidth={1.5} />
              </Button>
            )}
          </div>

          <nav className="flex-1 overflow-y-auto p-3 space-y-5 window-no-drag">
            {GROUPS.map((group) => (
              <div key={group.label} className="space-y-0.5">
                <div className="px-2 pb-1 text-overline text-shell-text-muted">{group.label}</div>
                {group.items.map((item) => (
                  <NavItem
                    key={item.id}
                    item={item}
                    isActive={currentTab === item.id}
                    onSelect={() => handleItemClick(item.id)}
                  />
                ))}
              </div>
            ))}
          </nav>
        </div>

        <div className="p-3 border-t border-shell-line window-no-drag">
          <IntegrityStatus
            integrity={integrity}
            isChecking={isCheckingIntegrity}
            onRefresh={onRefreshIntegrity}
          />
        </div>
      </aside>
    </>
  );
};
