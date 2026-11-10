import React from 'react';
import {
  BookCheck,
  BookOpen,
  Building2,
  LayoutDashboard,
  Loader2,
  Settings,
  ShieldAlert,
  X,
} from 'lucide-react';
import { IntegrityCheckResult } from '../types';
import { AREA_LABELS, Area } from '../navigation';
import { GermanFlag } from './GermanFlag';
import { Button, cn } from './ui';
import { formatTime } from '../utils/formatters';

interface SidebarProps {
  currentArea: Area;
  onSelectArea: (area: Area) => void;
  integrity: IntegrityCheckResult | null;
  integrityError: string | null;
  isCheckingIntegrity: boolean;
  /** Der Zustandsanzeiger öffnet „Prüfung & GoBD" in den Einstellungen. */
  onOpenAudit: () => void;
  isOpenMobile?: boolean;
  onCloseMobile?: () => void;
}

const icon = 'w-4 h-4 shrink-0';

/**
 * Fünf Einträge. Die Arbeit erledigt der Agent; die Oberfläche zeigt den
 * Stand, die Bücher und was nur der Mensch freigeben kann.
 */
const ITEMS: { id: Area; icon: React.ReactNode }[] = [
  { id: 'overview', icon: <LayoutDashboard className={icon} strokeWidth={1.5} /> },
  { id: 'books', icon: <BookOpen className={icon} strokeWidth={1.5} /> },
  { id: 'closings', icon: <BookCheck className={icon} strokeWidth={1.5} /> },
  { id: 'company', icon: <Building2 className={icon} strokeWidth={1.5} /> },
  { id: 'settings', icon: <Settings className={icon} strokeWidth={1.5} /> },
];

const NAV_ITEM =
  'w-full flex items-center gap-2 h-8 pl-1.5 pr-2 rounded-control ' +
  'text-body text-left transition-colors duration-120 ease-quiet';

const NavItem: React.FC<{
  label: string;
  icon: React.ReactNode;
  isActive: boolean;
  onSelect: () => void;
}> = ({ label, icon: itemIcon, isActive, onSelect }) => (
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
    <span className="shrink-0 ml-0.5">{itemIcon}</span>
    <span className="truncate">{label}</span>
  </button>
);

type IntegrityState = 'checking' | 'valid' | 'broken' | 'failed' | 'unchecked';

/**
 * „Daten unverändert" steht nur nach einer gelungenen Prüfung. Ist sie
 * fehlgeschlagen oder noch nicht gelaufen, sagt die Anzeige genau das.
 */
function integrityState(
  integrity: IntegrityCheckResult | null,
  error: string | null,
  isChecking: boolean,
): IntegrityState {
  if (isChecking) return 'checking';
  if (integrity) return integrity.isValid ? 'valid' : 'broken';
  return error ? 'failed' : 'unchecked';
}

const INTEGRITY_LABEL: Record<IntegrityState, string> = {
  checking: 'Prüfung läuft',
  valid: 'Daten unverändert',
  broken: 'Integrität verletzt',
  failed: 'Prüfung fehlgeschlagen',
  unchecked: 'Noch nicht geprüft',
};

/**
 * Der Integritätszustand steht dauerhaft hier und nie in einem Toast (§11.4).
 * Ein Klick führt zu „Prüfung & GoBD"; nachgeprüft wird dort.
 */
const IntegrityStatus: React.FC<{
  integrity: IntegrityCheckResult | null;
  error: string | null;
  isChecking: boolean;
  onOpen: () => void;
}> = ({ integrity, error, isChecking, onOpen }) => {
  const state = integrityState(integrity, error, isChecking);
  const alarming = state === 'broken' || state === 'failed';

  return (
    <button
      type="button"
      onClick={onOpen}
      title={integrity?.message || error || 'Prüfung & GoBD öffnen'}
      className="w-full flex items-start gap-2.5 px-2 py-1.5 rounded-control text-left
                 transition-colors duration-120 ease-quiet hover:bg-shell-raised"
    >
      <span className="mt-1 shrink-0">
        {state === 'checking' ? (
          <Loader2 className="w-3.5 h-3.5 animate-spin text-shell-text-muted" strokeWidth={1.5} />
        ) : alarming ? (
          <ShieldAlert className="w-3.5 h-3.5 text-shell-negative" strokeWidth={1.5} />
        ) : state === 'valid' ? (
          <span className="mark-diamond bg-shell-positive block" />
        ) : (
          <span className="mark-diamond bg-shell-text-muted block" />
        )}
      </span>
      <span className="min-w-0">
        <span className={cn('block text-caption', alarming ? 'text-shell-negative' : 'text-shell-text')}>
          {INTEGRITY_LABEL[state]}
        </span>
        {integrity?.checkedAt && state !== 'checking' ? (
          <span className="block text-caption text-shell-text-muted num">
            Geprüft um {formatTime(integrity.checkedAt)}
          </span>
        ) : (
          state !== 'checking' &&
          state !== 'valid' && (
            <span className="block text-caption text-shell-text-muted">In Prüfung &amp; GoBD prüfen</span>
          )
        )}
      </span>
    </button>
  );
};

export const Sidebar: React.FC<SidebarProps> = ({
  currentArea,
  onSelectArea,
  integrity,
  integrityError,
  isCheckingIntegrity,
  onOpenAudit,
  isOpenMobile = false,
  onCloseMobile,
}) => (
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
        <div className="pt-8 md:pt-9 pb-3 px-3 border-b border-shell-line flex items-center gap-3">
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
          <span className="flex-1 min-w-0 text-body font-semibold text-white">Buchfink</span>
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

        <nav className="flex-1 overflow-y-auto p-3 space-y-0.5 window-no-drag">
          {ITEMS.map((item) => (
            <NavItem
              key={item.id}
              label={AREA_LABELS[item.id]}
              icon={item.icon}
              isActive={currentArea === item.id}
              onSelect={() => {
                onSelectArea(item.id);
                onCloseMobile?.();
              }}
            />
          ))}
        </nav>
      </div>

      <div className="p-3 border-t border-shell-line window-no-drag">
        <IntegrityStatus
          integrity={integrity}
          error={integrityError}
          isChecking={isCheckingIntegrity}
          onOpen={() => {
            onOpenAudit();
            onCloseMobile?.();
          }}
        />
      </div>
    </aside>
  </>
);
