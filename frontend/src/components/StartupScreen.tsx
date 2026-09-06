import React, { useState } from 'react';
import { ArrowRight, Plus, RotateCcw, Trash2 } from 'lucide-react';
import { CompanySettings, TenantConfig } from '../types';
import { Api } from '../services/api';
import { useWriteLock } from './WriteLock';
import { GermanFlag } from './GermanFlag';
import { ConfirmDialog, SHELL_BUTTON, SHELL_PANEL, cn, toast } from './ui';

interface StartupScreenProps {
  settings: CompanySettings | null;
  tenants: TenantConfig[];
  activeTenant: TenantConfig | null;
  onSwitchTenant: (tenantId: string) => Promise<void>;
  onRefreshTenants: () => Promise<void>;
  onAddTenant: () => void;
  /** Öffnet die Buchhaltung des aktiven Mandanten. */
  onStartDashboard: () => void;
}

/**
 * Der Startschirm steht vor dem Arbeitsbereich und gehört zur Schale (§16):
 * dunkler Grund, eine erhöhte Fläche, genau eine Primäraktion.
 *
 * Er beantwortet eine einzige Frage: mit welchem Mandanten wird gearbeitet.
 * Bis hierher stand daneben eine zweite Navigation — vier Sprungmarken in
 * Bankumsätze, Rechnungen, Journal und Konten — und darüber die Stammdaten des
 * aktiven Mandanten. Beides gibt es im Arbeitsbereich schon, und eine zweite
 * Navigation vor der ersten ist ein Umweg, kein Einstieg.
 */
export const StartupScreen: React.FC<StartupScreenProps> = ({
  settings,
  tenants,
  activeTenant,
  onSwitchTenant,
  onRefreshTenants,
  onAddTenant,
  onStartDashboard,
}) => {
  // Einen Mandanten aus der Liste zu nehmen ist eine Änderung an der
  // Anwendungsverwaltung und im Prüfermodus gesperrt; Öffnen, Anlegen und
  // Wiederherstellen bleiben möglich (§10.4).
  const writeLock = useWriteLock();
  const [removing, setRemoving] = useState<TenantConfig | null>(null);
  const [restoring, setRestoring] = useState(false);
  const [opening, setOpening] = useState<string | null>(null);

  /**
   * Wiederherstellung vom Startbildschirm.
   *
   * Sie gehört hierher und nicht nur in die Einstellungen: gebraucht wird sie
   * auf einem Rechner, auf dem noch keine Bücher offen sind — und dorthin
   * käme man über eine Ansicht innerhalb eines Mandanten nicht.
   */
  async function restoreBackup() {
    setRestoring(true);
    try {
      const zip = await Api.selectBackupFile('Sicherung zum Wiederherstellen auswählen');
      if (!zip) return;
      const target = await Api.selectDirectoryDialog('Leeren Zielordner für die Daten wählen');
      if (!target) return;
      const tenant = await Api.restoreFromBackup(zip, target);
      await onRefreshTenants();
      toast.success(`${tenant.name} wiederhergestellt und geprüft.`);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setRestoring(false);
    }
  }

  async function removeTenant(tenant: TenantConfig) {
    try {
      await Api.deleteTenant(tenant.id);
      toast.success(`Mandant ${tenant.name} aus der Liste entfernt.`);
      await onRefreshTenants();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    }
  }

  /**
   * Ein Klick auf den Mandanten öffnet ihn — der Wechsel und das Betreten der
   * Bücher sind hier derselbe Vorgang. Getrennt wären es zwei Schritte für eine
   * Absicht: erst umschalten, dann noch einmal auf „Zur Buchhaltung".
   */
  async function open(tenant: TenantConfig) {
    if (tenant.id === activeTenant?.id) {
      onStartDashboard();
      return;
    }
    setOpening(tenant.id);
    try {
      await onSwitchTenant(tenant.id);
      onStartDashboard();
    } finally {
      setOpening(null);
    }
  }

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

      <div className="relative z-10 w-full max-w-2xl mx-auto px-6 py-12 flex-1 flex flex-col justify-center gap-6">
        <header className="flex items-center gap-3">
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
        </header>

        <div className={cn(SHELL_PANEL, 'p-6')}>
          <h2 className="text-overline uppercase text-shell-text-muted">
            Mandanten · {tenants.length}
          </h2>

          <ul className="mt-2 divide-y divide-shell-line">
            {tenants.map((tenant) => {
              const active = tenant.id === activeTenant?.id;
              return (
                <li key={tenant.id} className="group relative">
                  {/* Die ganze Zeile ist der Knopf: der Mandant wird geöffnet,
                      nicht ausgewählt und dann geöffnet. */}
                  <button
                    type="button"
                    onClick={() => void open(tenant)}
                    disabled={opening !== null}
                    aria-label={`${tenant.name} öffnen`}
                    className={cn(
                      'w-full flex items-center gap-3 rounded-control px-3 py-3 -mx-3 text-left',
                      'transition-colors duration-120 ease-quiet',
                      active ? 'bg-shell-raised' : 'hover:bg-shell-raised/60',
                      opening !== null && !active && 'opacity-60',
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
                        {active && settings?.fiscalYear && (
                          <span className="text-shell-text-muted">
                            {' '}
                            · Geschäftsjahr {settings.fiscalYear}
                          </span>
                        )}
                      </span>
                      <span
                        className="block code-num text-caption text-shell-text-muted truncate"
                        title={tenant.dataDir}
                      >
                        {tenant.dataDir}
                      </span>
                    </span>
                    <span className="shrink-0 flex items-center gap-2 text-caption text-shell-text-muted">
                      {opening === tenant.id ? 'wird geöffnet …' : 'Öffnen'}
                      <ArrowRight className="w-4 h-4" strokeWidth={1.5} />
                    </span>
                    {/* Platz für den Entfernen-Knopf, der über der Zeile liegt. */}
                    {tenants.length > 1 && <span className="w-8 shrink-0" aria-hidden="true" />}
                  </button>

                  {tenants.length > 1 && (
                    <button
                      type="button"
                      onClick={() => setRemoving(tenant)}
                      disabled={writeLock.locked}
                      title={writeLock.hint ?? 'Aus der Liste entfernen'}
                      aria-label={`${tenant.name} aus der Liste entfernen`}
                      className={cn(
                        SHELL_BUTTON.quiet,
                        'absolute right-0 top-1/2 -translate-y-1/2 h-8 w-8 px-0 opacity-0',
                        'group-hover:opacity-100 focus-visible:opacity-100',
                        'disabled:cursor-not-allowed',
                      )}
                    >
                      <Trash2 className="w-4 h-4" strokeWidth={1.5} />
                    </button>
                  )}
                </li>
              );
            })}
          </ul>

          <div className="mt-5 pt-5 border-t border-shell-line flex flex-wrap gap-2">
            <button type="button" onClick={onAddTenant} className={SHELL_BUTTON.secondary}>
              <Plus className="w-4 h-4" strokeWidth={1.5} />
              Mandant hinzufügen
            </button>
            <button
              type="button"
              onClick={() => void restoreBackup()}
              disabled={restoring}
              className={SHELL_BUTTON.quiet}
            >
              <RotateCcw className="w-4 h-4" strokeWidth={1.5} />
              Aus Sicherung wiederherstellen
            </button>
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
