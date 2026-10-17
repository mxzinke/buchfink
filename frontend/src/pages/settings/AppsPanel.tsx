import React, { useCallback, useEffect, useState } from 'react';
import { Plug } from 'lucide-react';
import { Api } from '../../services/api';
import type { ConnectedApp, MCPServerStatus } from '../../types';
import { formatDate, formatDateTime } from '../../utils/formatters';
import { useRouteObject } from '../../navigation';
import { useWriteLock } from '../../components/WriteLock';
import {
  Button,
  ConfirmDialog,
  DetailList,
  DetailPanel,
  EmptyState,
  Notice,
  Section,
  SkeletonRows,
  Table,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
  toast,
} from '../../components/ui';

function message(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

const SCOPE_LABELS: Record<string, string> = {
  'buchfink:lesen': 'Lesen',
  'buchfink:buchen': 'Buchen',
};

function rightsOf(scopes: string[]): string {
  return scopes.map((scope) => SCOPE_LABELS[scope] ?? scope).join(' und ') || '—';
}

/** Stand des Servers in einer Zeile. */
function serverLine(status: MCPServerStatus | null): string {
  if (!status) return 'Stand wird gelesen';
  if (status.running) return `Erreichbar unter ${status.url}`;
  return 'Server läuft nicht';
}

/**
 * Die Apps, denen der Zugriff auf Buchfink erlaubt ist, und der Stand des
 * Servers, über den sie zugreifen. Entziehen wirkt sofort: der nächste Aufruf
 * der App scheitert, bis sie erneut zugelassen wird.
 */
export const AppsPanel: React.FC<{ id?: string }> = ({ id }) => {
  const writeLock = useWriteLock();
  const [status, setStatus] = useState<MCPServerStatus | null>(null);
  const [apps, setApps] = useState<ConnectedApp[] | null>(null);
  const [error, setError] = useState('');
  const [revoking, setRevoking] = useState<ConnectedApp | null>(null);
  // Die Einzelheiten einer App stehen in der Seitenleiste; die Kennung im Ziel
  // ist ihre Client-ID.
  const selection = useRouteObject('settings', 'apps');
  const selected = apps?.find((app) => app.clientId === selection.id) ?? null;

  const load = useCallback(async () => {
    try {
      const [nextStatus, nextApps] = await Promise.all([
        Api.getMCPServerStatus(),
        Api.getConnectedApps(),
      ]);
      setStatus(nextStatus);
      setApps(nextApps.filter((app) => !app.revoked));
      setError('');
    } catch (e) {
      setApps([]);
      setError(message(e));
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function revoke(app: ConnectedApp) {
    try {
      await Api.revokeConnectedApp(app.clientId);
      toast.success(`„${app.name}“ hat keinen Zugriff mehr.`);
      await load();
    } catch (e) {
      toast.error(message(e));
    }
  }

  return (
    <Section
      id={id}
      title="Apps verbinden"
      context={serverLine(status)}
      divider={false}
      helpSummary="Hier sehen Sie, welche KI-Apps auf Ihre Buchhaltung zugreifen dürfen."
      explain={
        <>
          Eine App meldet sich über die Adresse des Servers an. Die Zustimmung erteilen Sie in
          diesem Fenster. Buchungen der App sind mit ihrem Namen gekennzeichnet und lassen sich bis
          zur Festschreibung stornieren.
        </>
      }
    >
      {status && !status.running && status.error && (
        <Notice tone="negative" className="mb-5" text={status.error} />
      )}
      {error && <Notice tone="negative" className="mb-5" text={error} />}
      {apps === null ? (
        <SkeletonRows rows={3} />
      ) : apps.length === 0 ? (
        <EmptyState
          icon={<Plug className="w-6 h-6" strokeWidth={1.5} />}
          title="Noch keine App zugelassen"
          description="Sobald sich eine App anmeldet, fragt Buchfink in diesem Fenster nach Ihrer Zustimmung."
        />
      ) : (
        <Table>
          <Thead>
            <Tr>
              <Th>App</Th>
              <Th>Mandanten</Th>
              <Th className="w-36">Zugelassen</Th>
              <Th className="w-36">Zuletzt</Th>
              <Th className="w-28" aria-label="Aktion" />
            </Tr>
          </Thead>
          <Tbody>
            {apps.map((app) => (
              <Tr key={app.clientId}>
                <Td>
                  <button
                    type="button"
                    className="text-left text-ink underline-offset-2 hover:underline"
                    onClick={() => selection.open(app.clientId)}
                  >
                    {app.name}
                  </button>
                </Td>
                <Td className="text-ink-muted">
                  {app.tenants.map((tenant) => tenant.name).join(', ') || '—'}
                </Td>
                <Td className="num text-ink-subtle">{formatDate(app.approvedAt)}</Td>
                <Td className="num text-ink-subtle">
                  {app.lastUsedAt ? formatDate(app.lastUsedAt) : 'nie'}
                </Td>
                <Td className="text-right pl-0">
                  <Button
                    variant="quiet"
                    size="sm"
                    disabled={writeLock.locked}
                    title={writeLock.hint}
                    onClick={() => setRevoking(app)}
                  >
                    Entziehen
                  </Button>
                </Td>
              </Tr>
            ))}
          </Tbody>
        </Table>
      )}

      <DetailPanel
        open={selected !== null}
        onOpenChange={(open) => !open && selection.close()}
        title={selected?.name ?? ''}
        context={selected ? `Zugelassen am ${formatDate(selected.approvedAt)}` : undefined}
        footer={
          selected && (
            <Button
              variant="danger"
              disabled={writeLock.locked}
              title={writeLock.hint}
              onClick={() => setRevoking(selected)}
            >
              Zugriff entziehen
            </Button>
          )
        }
      >
        {selected && (
          <DetailList
            items={[
              { label: 'Erfassungsweg', value: <span className="code-num">{selected.channel}</span> },
              {
                label: 'Zuletzt',
                value: selected.lastUsedAt ? formatDateTime(selected.lastUsedAt) : 'nie',
              },
              ...selected.tenants.map((tenant) => ({
                label: tenant.name,
                value: rightsOf(tenant.scopes),
              })),
            ]}
          />
        )}
      </DetailPanel>

      <ConfirmDialog
        open={revoking !== null}
        onOpenChange={(open) => !open && setRevoking(null)}
        title={`Zugriff von „${revoking?.name ?? ''}“ entziehen?`}
        description="Die App kann danach weder lesen noch buchen. Für einen neuen Zugriff muss sie sich erneut anmelden und Sie müssen zustimmen."
        confirmLabel="Zugriff entziehen"
        destructive
        onConfirm={() => {
          const app = revoking;
          setRevoking(null);
          if (app) {
            selection.close();
            void revoke(app);
          }
        }}
      />
    </Section>
  );
};
