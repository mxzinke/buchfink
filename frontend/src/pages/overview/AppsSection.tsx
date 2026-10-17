import React from 'react';
import type { ConnectedApp, MCPServerStatus } from '../../types';
import { formatDateOfTimestamp } from '../../utils/formatters';
import type { NavigateFn } from '../../navigation';
import { Button, Notice, Section, SkeletonRows, cn } from '../../components/ui';

const SCOPE_LABELS: Record<string, string> = {
  'buchfink:lesen': 'Lesen',
  'buchfink:buchen': 'Buchen',
};

/** Was eine App im geöffneten Mandanten darf. */
function rightsOf(app: ConnectedApp, tenantId: string | undefined): string {
  const tenant = app.tenants.find((t) => t.id === tenantId);
  if (!tenant) return 'Kein Zugriff auf diesen Mandanten';
  return tenant.scopes.map((scope) => SCOPE_LABELS[scope] ?? scope).join(' und ') || 'Kein Recht';
}

function serverContext(apps: ConnectedApp[], status: MCPServerStatus | null): string {
  const count = apps.length === 1 ? '1 App zugelassen' : `${apps.length} Apps zugelassen`;
  if (!status) return count;
  return status.running ? `${count} · Server läuft auf Port ${status.port}` : `${count} · Server läuft nicht`;
}

export interface AppsSectionProps {
  apps: ConnectedApp[] | null;
  status: MCPServerStatus | null;
  tenantId: string | undefined;
  onNavigate: NavigateFn;
}

/**
 * Die verbundenen Apps in Kürze: Anzahl, Stand des Servers und je App, was sie
 * darf und wann sie zuletzt zugegriffen hat. Verbinden und Entziehen stehen in
 * den Einstellungen.
 */
export const AppsSection: React.FC<AppsSectionProps> = ({ apps, status, tenantId, onNavigate }) => {
  const toSettings = () => onNavigate({ area: 'settings', tab: 'apps' });
  return (
    <Section
      title="Verbundene Apps"
      context={apps ? serverContext(apps, status) : 'Apps mit Zugriff auf Ihre Buchhaltung'}
      helpSummary="Diese Apps dürfen auf Ihre Buchhaltung zugreifen."
      explain={
        <>
          Claude und andere Apps greifen über einen Server auf diesem Rechner auf Buchfink zu. Jede
          App fragt einmal nach Ihrer Zustimmung, die Sie in diesem Fenster erteilen. Entziehen
          können Sie den Zugriff jederzeit in den Einstellungen.
        </>
      }
      action={
        <Button variant="quiet" onClick={toSettings}>
          Apps verwalten
        </Button>
      }
    >
      {status && !status.running && status.error && (
        <Notice tone="negative" className="mb-4" text={status.error} />
      )}
      {!apps ? (
        <SkeletonRows rows={2} />
      ) : apps.length === 0 ? (
        <p className="text-body text-ink-muted">
          Noch keine App verbunden. Den Befehl für Claude finden Sie unter Einstellungen.
        </p>
      ) : (
        <ul className="flex flex-col">
          {apps.map((app) => (
            <li
              key={app.clientId}
              className="flex items-center gap-3 py-2 border-t border-line first:border-t-0"
            >
              <span
                className={cn(
                  'mark-diamond shrink-0',
                  status?.running ? 'bg-positive' : 'bg-ink-faint',
                )}
                aria-hidden="true"
              />
              <span className="min-w-0 flex-1 truncate text-body text-ink">{app.name}</span>
              <span className="text-caption text-ink-subtle">{rightsOf(app, tenantId)}</span>
              <span className="w-40 text-right text-caption text-ink-subtle num">
                {app.lastUsedAt ? `zuletzt ${formatDateOfTimestamp(app.lastUsedAt)}` : 'noch nicht benutzt'}
              </span>
            </li>
          ))}
        </ul>
      )}
    </Section>
  );
};
