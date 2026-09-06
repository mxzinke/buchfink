import React from 'react';
import {
  Archive,
  BookCheck,
  BookOpen,
  Building2,
  Calendar,
  ClipboardCheck,
  FileSpreadsheet,
  FileText,
  HandCoins,
  Landmark,
  ListChecks,
  ListOrdered,
  ListTodo,
  Loader2,
  Percent,
  Receipt,
  Scale,
  ScrollText,
  Settings,
  ShieldAlert,
  Users,
  X,
} from 'lucide-react';
import { CompanySettings, IntegrityCheckResult, TenantConfig } from '../types';
import { TenantSwitcher } from './TenantSwitcher';
import { Button, cn } from './ui';
import { formatTime } from '../utils/formatters';

export type TabType =
  | 'welcome'
  | 'tasks'
  | 'bank'
  | 'receipts'
  | 'invoices'
  | 'advances'
  | 'journal'
  | 'assets'
  | 'contacts'
  | 'accounts'
  | 'reports'
  | 'closing'
  | 'closingmodules'
  | 'vat'
  | 'obligations'
  | 'deadlines'
  | 'ebilanz'
  | 'audit'
  | 'backup'
  | 'taxaudit'
  | 'settings';

/**
 * Ziel einer Navigation mit Parameter.
 *
 * Der Weg von der Bilanzzeile über das Konto zur Buchung (GOB-02) endet sonst
 * an der Seitengrenze: die Kontenübersicht wüsste nicht, welches Kontoblatt sie
 * öffnen soll, und das Journal nicht, auf welche Buchung es filtern soll.
 */
export interface NavigationParams {
  /** Kontonummer, deren Kontoblatt die Kontenübersicht öffnet. */
  account?: string;
  /** Buchungsnummer, auf die das Journal filtert. */
  entryNumber?: string;
  /**
   * Konto, mit dem das Journal die Zeilenauswertung öffnet (PRF-01 K3).
   *
   * Getrennt von `entryNumber`: das eine sucht eine Buchung in der Liste, das
   * andere öffnet den Reiter „Zeilen auswerten" mit vorbelegtem Konto — der
   * Weg aus dem Kontoblatt in die Auswertung desselben Kontos.
   */
  filterAccount?: string;
  /**
   * Reiter, den die Seite „Nebenpflichten" öffnet. Aus der Schrittliste des
   * Abschlusses führt sonst kein Weg zu der Arbeit, die die Zeile benennt: der
   * Reiter wäre zu erraten.
   */
  obligationsTab?: string;
  /** Belegfilter, mit dem die Belegseite öffnet, etwa „filed" für abgelegte. */
  receiptStatus?: string;
  /**
   * Ansicht, mit der „Bank & Zahlungen" öffnet: der Abgleich oder das
   * Mahnwesen. Beide wohnen auf derselben Seite, und eine Aufgabe, die das
   * Mahnwesen meint, landete sonst auf dem Kontoauszug.
   */
  bankView?: string;
  /**
   * Baustein, den die Abschlussbausteine gleich aufschlagen. Aus dem geführten
   * Weg des Jahresabschlusses wäre der Reiter sonst zu erraten.
   */
  closingTab?: string;
  /**
   * Rechnung, deren Nachweisbelege die Seite „Nebenpflichten" gleich aufschlägt.
   * Ohne sie bräche der Weg „je Lieferung" an der Seitengrenze ab: der Anwender
   * käme auf einer Liste an und müsste die Rechnung, von der er kommt, dort
   * erneut suchen.
   */
  invoiceId?: number;
  /**
   * Beleg, den die Belegseite gleich aufschlägt. Er schließt den Weg von der
   * Bilanzposition über Konto und Buchung bis zum Beleg (GOB-02): ohne ihn
   * endete der Weg im Journal, und der Beleg wäre in der Belegliste erneut zu
   * suchen.
   */
  receiptId?: number;
  /**
   * Das Geschäftsjahr, in dem das Ziel steht. Die Abschlussansichten folgen dem
   * Jahr aus der Kopfzeile; eine Aufgabe zum Vorjahresabschluss führte ohne
   * diesen Parameter auf die Abschlussseite des laufenden Jahres — also auf die
   * falsche Auskunft.
   */
  year?: number;
  /**
   * Prüfregel, deren Befunde die Prüfübersicht hervorhebt. Ohne sie stünde die
   * Aufgabe „Befunde klären" vor einer Liste von Prüfläufen ohne Anker.
   */
  auditRule?: string;
  /** Frist, die die Fristenansicht hervorhebt. */
  deadlineKey?: string;
}

export type NavigateFn = (tab: TabType, params?: NavigationParams) => void;

interface SidebarProps {
  currentTab: TabType;
  onSelectTab: (tab: TabType) => void;
  settings: CompanySettings | null;
  integrity: IntegrityCheckResult | null;
  isCheckingIntegrity: boolean;
  /** Mandantenwahl am Kopf der Navigation, dort wo der Name des Mandanten steht. */
  tenants: TenantConfig[];
  activeTenant: TenantConfig | null;
  onSwitchTenant: (tenantId: string) => void;
  onAddTenant: () => void;
  onShowTenants: () => void;
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
    // Die Aufgabenliste ist die Startseite und der einzige Eintrag dieser Gruppe
    // (Architektur 6.1): wer keine Buchhalterin ist, weiß nach dem Start nicht,
    // was heute dran ist — und ein Bankguthaben beantwortet das nicht. Die
    // Kennzahlen und die zuletzt erfassten Vorgänge stehen auf derselben Seite
    // unter der Liste; eine zweite Seite daneben beantwortete dieselbe Frage
    // ein zweites Mal und wäre irgendwann die veraltete von beiden.
    items: [{ id: 'tasks', label: 'Aufgaben', icon: <ListTodo className={icon} /> }],
  },
  {
    label: 'Buchhaltung',
    items: [
      { id: 'bank', label: 'Bank & Zahlungen', icon: <Landmark className={icon} /> },
      { id: 'receipts', label: 'Belege', icon: <Receipt className={icon} /> },
      { id: 'invoices', label: 'Rechnungen', icon: <FileText className={icon} /> },
      { id: 'advances', label: 'Anzahlungen', icon: <HandCoins className={icon} /> },
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
      {
        id: 'closingmodules',
        label: 'Abschlussbausteine',
        icon: <ListChecks className={icon} />,
      },
      { id: 'vat', label: 'Umsatzsteuer', icon: <Percent className={icon} /> },
      // Die Nebenpflichten stehen neben der Umsatzsteuer, weil vier ihrer sechs
      // Bausteine aus dem Umsatzsteuerrecht kommen — und weil man sie nur
      // findet, wenn sie in der Navigation stehen.
      { id: 'obligations', label: 'Nebenpflichten', icon: <ClipboardCheck className={icon} /> },
      { id: 'deadlines', label: 'Steuerfristen', icon: <Calendar className={icon} /> },
      { id: 'ebilanz', label: 'E-Bilanz', icon: <FileSpreadsheet className={icon} /> },
    ],
  },
  {
    label: 'Verwaltung',
    // Drei Einträge nach dem, was der Anwender vorhat, und nicht nach der
    // Herkunft der Daten: seine Daten sichern, eine Prüfung bedienen,
    // einstellen. Was die Buchführung über sich selbst weiß — Kette,
    // Protokoll, Prüfläufe, Systemhistorie — steht in der Prüfübersicht und
    // hängt am Zustandsanzeiger in der Fußzeile: sie beantwortet die Frage
    // „ist etwas nicht in Ordnung", und die stellt sich erst, wenn der
    // Anzeiger etwas anderes sagt als „unverändert".
    items: [
      { id: 'backup', label: 'Datensicherung', icon: <Archive className={icon} /> },
      { id: 'taxaudit', label: 'Betriebsprüfung', icon: <ScrollText className={icon} /> },
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

/**
 * Der Integritätszustand steht dauerhaft hier und nie in einem Toast (§11.4).
 * Drei Zustände, drei Formulierungen.
 *
 * Er ist zugleich der Weg in die Prüfübersicht — Kette, Protokoll, Prüfläufe,
 * Systemhistorie —, die deshalb keinen eigenen Navigationseintrag hat: die
 * Frage „ist etwas nicht in Ordnung" wird hier beantwortet, und wer sie
 * genauer wissen will, klickt auf die Antwort. Nachgeprüft wird auf der Seite
 * selbst; ein Klick, der beides täte, machte aus einem Blick auf den Zustand
 * einen Lauf über die ganze Buchführung.
 */
const IntegrityStatus: React.FC<{
  integrity: IntegrityCheckResult | null;
  isChecking: boolean;
  isActive: boolean;
  onOpen: () => void;
}> = ({ integrity, isChecking, isActive, onOpen }) => {
  const broken = integrity !== null && !integrity.isValid;

  return (
    <button
      type="button"
      onClick={onOpen}
      aria-current={isActive ? 'page' : undefined}
      title={integrity?.message || 'Prüfübersicht öffnen'}
      className={cn(
        `w-full flex items-start gap-2.5 px-2 py-1.5 rounded-control text-left
         transition-colors duration-120 ease-quiet hover:bg-shell-raised`,
        // Die geöffnete Prüfübersicht wird hier markiert und nicht in der
        // Liste darüber: sonst stünde der Anwender auf einer Seite, die in
        // der Navigation nirgends hervorgehoben ist.
        isActive && 'bg-shell-raised',
      )}
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
            {/* Die Uhrzeit mit Zeitzone und aus formatters.ts: eine eigene
                Formatierung an dieser Stelle wäre die einzige Zeitangabe der
                Anwendung ohne Zone (QUE-04, Entscheidung 8). Das Datum steht
                im Zusammenhang des Arbeitstags. */}
            Geprüft um {formatTime(integrity.checkedAt)}
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
  isCheckingIntegrity,
  tenants,
  activeTenant,
  onSwitchTenant,
  onAddTenant,
  onShowTenants,
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
            <div className="flex-1 min-w-0">
              <TenantSwitcher
                tenants={tenants}
                activeTenant={activeTenant}
                settings={settings}
                onSwitchTenant={(id) => {
                  onSwitchTenant(id);
                  onCloseMobile?.();
                }}
                onAddTenant={onAddTenant}
                onShowAll={onShowTenants}
              />
            </div>

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
            isActive={currentTab === 'audit'}
            onOpen={() => handleItemClick('audit')}
          />
        </div>
      </aside>
    </>
  );
};
