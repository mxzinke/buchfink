import React, { useCallback, useEffect, useState } from 'react';
import { Plug } from 'lucide-react';
import { Api } from '../../services/api';
import type { ConnectedApp, MCPServerStatus } from '../../types';
import { formatDateOfTimestamp, formatDateTime } from '../../utils/formatters';
import { useRouteObject } from '../../navigation';
import { useWriteLock } from '../../components/WriteLock';
import { AgentClientCommands } from '../../components/AgentClients';
import {
  Button,
  ConfirmDialog,
  DetailList,
  DetailPanel,
  Dialog,
  EmptyState,
  Field,
  Input,
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
  return `Nicht erreichbar · Port ${status.port}`;
}

/**
 * Apps verbinden: die Befehle je App, die zugelassenen Apps und der Stand des
 * Servers, über den sie zugreifen. Entziehen wirkt sofort: der nächste Aufruf
 * der App scheitert, bis sie erneut zugelassen wird.
 */
export const AppsPanel: React.FC<{ id?: string }> = ({ id }) => {
  const writeLock = useWriteLock();
  const [status, setStatus] = useState<MCPServerStatus | null>(null);
  const [apps, setApps] = useState<ConnectedApp[] | null>(null);
  const [error, setError] = useState('');
  const [revoking, setRevoking] = useState<ConnectedApp | null>(null);
  const [portOpen, setPortOpen] = useState(false);
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
      setError(`Die Apps konnten nicht geladen werden. ${message(e)}`);
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
      helpSummary="Eine KI-App wie Claude erledigt die Buchhaltung über diese Verbindung."
      explain={
        <>
          Tragen Sie Buchfink mit dem Befehl Ihrer App ein und melden Sie sich an. Die Zustimmung
          erteilen Sie danach in diesem Fenster, nie im Browser. Buchungen einer App speichern ihren
          Namen und lassen sich bis zur Festschreibung stornieren. Was ein Werkzeug zurückgibt,
          schickt die App an den Anbieter ihres Sprachmodells; klären Sie vorher, ob Ihr Vertrag mit
          ihm das abdeckt.
        </>
      }
      action={
        <Button
          variant="quiet"
          size="sm"
          disabled={!status}
          onClick={() => setPortOpen(true)}
        >
          Port ändern
        </Button>
      }
    >
      {status && !status.running && (
        <Notice
          tone="negative"
          className="mb-5"
          text={status.error || 'Der Zugang für KI-Apps läuft nicht.'}
          action={
            <Button variant="secondary" size="sm" onClick={() => setPortOpen(true)}>
              Anderen Port wählen
            </Button>
          }
        />
      )}
      <AgentClientCommands url={status?.url} className="max-w-[900px]" />

      <h3 className="text-label text-ink mt-10 mb-3">Zugelassene Apps</h3>
      {error ? (
        <Notice
          tone="negative"
          text={error}
          action={
            <Button variant="quiet" size="sm" onClick={() => void load()}>
              Erneut laden
            </Button>
          }
        />
      ) : apps === null ? (
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
              <Th>Rechte</Th>
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
                  {app.tenants.length === 1
                    ? rightsOf(app.tenants[0].scopes)
                    : `${app.tenants.length} Unternehmen`}
                </Td>
                <Td className="num text-ink-subtle">{formatDateOfTimestamp(app.approvedAt)}</Td>
                <Td className="num text-ink-subtle">
                  {app.lastUsedAt ? formatDateOfTimestamp(app.lastUsedAt) : 'nie'}
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
        context={selected ? `Zugelassen am ${formatDateOfTimestamp(selected.approvedAt)}` : undefined}
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
              {
                label: 'Erfassungsweg',
                value: <span className="code-num">{selected.channel}</span>,
              },
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

      {portOpen && status && (
        <PortDialog
          status={status}
          onClose={() => setPortOpen(false)}
          onChanged={(next) => {
            setStatus(next);
            setPortOpen(false);
          }}
        />
      )}
    </Section>
  );
};

/**
 * Der Port gehört zur Adresse, die jede App gespeichert hat. Ein neuer Port
 * heißt: jede App mit der neuen Adresse eintragen und neu anmelden.
 */
const PortDialog: React.FC<{
  status: MCPServerStatus;
  onClose: () => void;
  onChanged: (status: MCPServerStatus) => void;
}> = ({ status, onClose, onChanged }) => {
  const [text, setText] = useState(String(status.port || ''));
  const [error, setError] = useState('');
  const [saving, setSaving] = useState(false);

  async function save() {
    const port = text.trim() === '' ? 0 : Number(text);
    if (!Number.isInteger(port) || (port !== 0 && (port < 1024 || port > 65535))) {
      setError('Ein Port liegt zwischen 1024 und 65535.');
      return;
    }
    setSaving(true);
    setError('');
    try {
      const next = await Api.setMCPPort(port);
      toast.success(`Erreichbar unter ${next.url}. Tragen Sie die neue Adresse in Ihren Apps ein.`);
      onChanged(next);
    } catch (e) {
      setError(message(e));
    } finally {
      setSaving(false);
    }
  }

  return (
    <Dialog
      open
      onOpenChange={(open) => !open && onClose()}
      title="Port für KI-Apps"
      width="max-w-md"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Abbrechen
          </Button>
          <Button variant="primary" loading={saving} onClick={() => void save()}>
            Port übernehmen
          </Button>
        </>
      }
    >
      <Field
        label="Port"
        hint="Leer heißt 7465"
        error={error || undefined}
        helpSummary="Ändern Sie den Port nur, wenn ein anderes Programm ihn belegt."
        explain="Die Adresse mit dem Port steht in jeder verbundenen App. Nach einem Wechsel tragen Sie Buchfink dort mit der neuen Adresse ein und melden sich erneut an."
      >
        <Input
          type="number"
          min={1024}
          max={65535}
          align="right"
          value={text}
          onChange={(e) => setText(e.target.value)}
        />
      </Field>
    </Dialog>
  );
};
