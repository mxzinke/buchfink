import React, { useState } from 'react';
import { ArrowRight, BookOpen, FileText, Landmark, ListOrdered, Plus, Trash2 } from 'lucide-react';
import { CompanySettings, TenantConfig } from '../types';
import { Api } from '../services/api';
import { GermanFlag } from './GermanFlag';
import { ConfirmDialog, SHELL_BUTTON, SHELL_PANEL, cn, toast } from './ui';

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

const LAUNCHERS: { tab: string; label: string; icon: React.ReactNode }[] = [
  { tab: 'bank', label: 'Bankumsätze', icon: <Landmark className="w-4 h-4" strokeWidth={1.5} /> },
  { tab: 'invoices', label: 'Rechnungen', icon: <FileText className="w-4 h-4" strokeWidth={1.5} /> },
  { tab: 'journal', label: 'Journal', icon: <ListOrdered className="w-4 h-4" strokeWidth={1.5} /> },
  { tab: 'accounts', label: 'Konten', icon: <BookOpen className="w-4 h-4" strokeWidth={1.5} /> },
];

/**
 * Der Startschirm steht vor dem Arbeitsbereich und gehört zur Schale (§16):
 * dunkler Grund, eine erhöhte Fläche, genau eine Primäraktion.
 */
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
  const [removing, setRemoving] = useState<TenantConfig | null>(null);

  async function removeTenant(tenant: TenantConfig) {
    try {
      await Api.deleteTenant(tenant.id);
      toast.success(`Mandant ${tenant.name} aus der Liste entfernt.`);
      await onRefreshTenants();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    }
  }

  const companyName = settings?.companyName || activeTenant?.name || 'Hauptmandant';
  const fiscalYear = settings?.fiscalYear || new Date().getFullYear();

  const meta = [
    settings?.legalForm || 'GmbH',
    `${settings?.taxationType || 'SOLL'}-Versteuerung`,
    settings?.taxNumber ? `St.-Nr. ${settings.taxNumber}` : null,
  ].filter(Boolean);

  return (
    <div className="relative min-h-full flex flex-col overflow-y-auto bg-shell-deep text-shell-text">
      {/* Das Bild ist der einzige Zierrat der Anwendung und bleibt dem Schirm
          vor der Arbeit vorbehalten. */}
      <div
        className="absolute inset-0 bg-cover bg-center pointer-events-none"
        style={{ backgroundImage: "url('/bg-startupscreen_unsplash-steven-kamenar.jpg')" }}
      />
      <div className="absolute inset-0 bg-shell-deep/85 pointer-events-none" />
      <div className="absolute inset-0 bg-gradient-to-t from-shell-deep to-transparent pointer-events-none" />

      <div className="relative z-10 w-full max-w-3xl mx-auto px-6 py-12 flex-1 flex flex-col justify-center gap-6">
        <header className="flex items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <span className="relative shrink-0">
              <img
                src="/buchfink-logo.svg"
                alt=""
                className="w-11 h-11 rounded-control border border-shell-line bg-shell-raised p-1.5"
              />
              <span className="absolute -bottom-1 -right-1">
                <GermanFlag className="w-4 h-3 rounded-[2px] border border-shell-deep" />
              </span>
            </span>
            <span>
              <h1 className="text-heading text-white">Buchfink</h1>
              <p className="text-caption text-shell-text-muted">
                Doppelte Buchführung und Bilanzierung, SKR04
              </p>
            </span>
          </div>

          <button type="button" onClick={onAddTenant} className={SHELL_BUTTON.secondary}>
            <Plus className="w-4 h-4" strokeWidth={1.5} />
            Mandant hinzufügen
          </button>
        </header>

        <div className={cn(SHELL_PANEL, 'p-6')}>
          <div className="flex flex-wrap items-start justify-between gap-4 pb-5 border-b border-shell-line">
            <div className="min-w-0">
              <h2 className="text-display text-white truncate">{companyName}</h2>
              <p className="text-caption text-shell-text-muted mt-1">
                <span className="text-accent-light">Geschäftsjahr {fiscalYear}</span>
                {meta.map((entry) => (
                  <React.Fragment key={String(entry)}> · {entry}</React.Fragment>
                ))}
              </p>
            </div>

            <button type="button" onClick={onStartDashboard} className={SHELL_BUTTON.primary}>
              Zur Buchhaltung
              <ArrowRight className="w-4 h-4" strokeWidth={1.5} />
            </button>
          </div>

          <div className="mt-5">
            <h3 className="text-overline uppercase text-shell-text-muted">
              Mandanten · {tenants.length}
            </h3>

            <ul className="mt-2 divide-y divide-shell-line">
              {tenants.map((tenant) => {
                const active = tenant.id === activeTenant?.id;
                return (
                  <li key={tenant.id}>
                    <div
                      className={cn(
                        'group flex items-center gap-3 rounded-control px-3 py-2.5 -mx-3',
                        'transition-colors duration-120 ease-quiet',
                        active ? 'bg-shell-raised' : 'hover:bg-shell-raised/60',
                      )}
                    >
                      <span
                        className={cn(
                          'mark-diamond shrink-0',
                          active ? 'bg-accent-light' : 'bg-shell-line',
                        )}
                        aria-hidden="true"
                      />
                      <span className="min-w-0 flex-1">
                        <span className="block text-body text-shell-text truncate">
                          {tenant.name}
                        </span>
                        <span
                          className="block code-num text-caption text-shell-text-muted truncate"
                          title={tenant.dataDir}
                        >
                          {tenant.dataDir}
                        </span>
                      </span>

                      {active ? (
                        <span className="shrink-0 text-caption text-shell-text-muted">Aktiv</span>
                      ) : (
                        <button
                          type="button"
                          onClick={() => void onSwitchTenant(tenant.id)}
                          className={cn(SHELL_BUTTON.secondary, 'h-8 px-2.5 shrink-0')}
                        >
                          Öffnen
                        </button>
                      )}

                      {tenants.length > 1 && (
                        <button
                          type="button"
                          onClick={() => setRemoving(tenant)}
                          title="Aus der Liste entfernen"
                          aria-label={`${tenant.name} aus der Liste entfernen`}
                          className={cn(
                            SHELL_BUTTON.quiet,
                            'h-8 w-8 px-0 shrink-0 opacity-0',
                            'group-hover:opacity-100 focus-visible:opacity-100',
                          )}
                        >
                          <Trash2 className="w-4 h-4" strokeWidth={1.5} />
                        </button>
                      )}
                    </div>
                  </li>
                );
              })}
            </ul>
          </div>

          <div className="mt-5 pt-5 border-t border-shell-line flex flex-wrap gap-1">
            {LAUNCHERS.map((launcher) => (
              <button
                key={launcher.tab}
                type="button"
                onClick={() => onNavigate(launcher.tab)}
                className={SHELL_BUTTON.quiet}
              >
                {launcher.icon}
                {launcher.label}
              </button>
            ))}
          </div>
        </div>
      </div>

      <ConfirmDialog
        open={removing !== null}
        onOpenChange={(next) => !next && setRemoving(null)}
        title="Mandant aus der Liste entfernen"
        description="Die Buchhaltungsdaten auf der Festplatte bleiben unberührt. Der Mandant lässt sich später über den Einrichtungsassistenten wieder hinzufügen."
        confirmLabel="Entfernen"
        destructive
        onConfirm={() => {
          const tenant = removing;
          setRemoving(null);
          if (tenant) void removeTenant(tenant);
        }}
      />
    </div>
  );
};
