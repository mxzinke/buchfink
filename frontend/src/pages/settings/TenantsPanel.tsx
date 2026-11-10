import React, { useState } from 'react';
import { Plus } from 'lucide-react';
import { Api } from '../../services/api';
import type { TenantConfig } from '../../types';
import { useWorkspace } from '../../workspace';
import { useWriteLock } from '../../components/WriteLock';
import {
  Button,
  ConfirmDialog,
  Section,
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

/**
 * Die Mandanten auf diesem Rechner. Der Agent arbeitet immer im geöffneten
 * Mandanten; ein Wechsel hier wechselt auch für ihn.
 */
export const TenantsPanel: React.FC<{ id?: string }> = ({ id }) => {
  const { tenants, activeTenant, switchTenant, refreshTenants, addTenant, reload } = useWorkspace();
  const writeLock = useWriteLock();
  const [removing, setRemoving] = useState<TenantConfig | null>(null);
  const [opening, setOpening] = useState<string | null>(null);

  async function open(tenant: TenantConfig) {
    setOpening(tenant.id);
    try {
      await switchTenant(tenant.id);
    } finally {
      setOpening(null);
    }
  }

  /** Entfernt einen Mandanten aus der Liste; beim geöffneten lädt der Arbeitsbereich neu. */
  async function remove(tenant: TenantConfig) {
    try {
      await Api.deleteTenant(tenant.id);
      toast.success(`Mandant ${tenant.name} aus der Liste entfernt. Die Dateien bleiben im Datenordner.`);
      if (tenant.id === activeTenant?.id) await reload();
      else await refreshTenants();
    } catch (e) {
      toast.error(message(e));
    }
  }

  return (
    <Section
      id={id}
      title="Mandanten"
      context={`${tenants.length} auf diesem Rechner`}
      action={
        <div className="flex gap-2">
          <Button
            variant="secondary"
            size="sm"
            icon={<Plus className="w-4 h-4" strokeWidth={1.5} />}
            onClick={addTenant}
          >
            Mandant hinzufügen
          </Button>
        </div>
      }
    >
      <Table>
        <Thead>
          <Tr>
            <Th>Mandant</Th>
            <Th>Ordner</Th>
            <Th className="w-56" aria-label="Aktionen" />
          </Tr>
        </Thead>
        <Tbody>
          {tenants.map((tenant) => {
            const active = tenant.id === activeTenant?.id;
            return (
              <Tr key={tenant.id}>
                <Td>{tenant.name}</Td>
                <Td code className="max-w-[24rem] truncate" title={tenant.dataDir}>
                  {tenant.dataDir}
                </Td>
                <Td className="text-right pl-0">
                  <span className="flex items-center justify-end gap-1">
                    {active ? (
                      <span className="px-2.5 text-caption text-ink-subtle">Geöffnet</span>
                    ) : (
                      <Button
                        variant="quiet"
                        size="sm"
                        loading={opening === tenant.id}
                        disabled={opening !== null}
                        onClick={() => void open(tenant)}
                      >
                        Öffnen
                      </Button>
                    )}
                    <Button
                      variant="quiet"
                      size="sm"
                      disabled={writeLock.locked}
                      title={writeLock.hint}
                      onClick={() => setRemoving(tenant)}
                    >
                      Entfernen
                    </Button>
                  </span>
                </Td>
              </Tr>
            );
          })}
        </Tbody>
      </Table>

      <ConfirmDialog
        open={removing !== null}
        onOpenChange={(next) => !next && setRemoving(null)}
        title={removing ? `„${removing.name}“ aus der Liste entfernen?` : 'Mandant aus der Liste entfernen'}
        description="Buchfink zeigt den Mandanten nicht mehr an. Buchungen, Belege und Sicherungen bleiben im Datenordner; über die Einrichtung lässt er sich wieder öffnen."
        confirmLabel="Entfernen"
        destructive
        onConfirm={() => {
          const tenant = removing;
          setRemoving(null);
          if (tenant) void remove(tenant);
        }}
      />
    </Section>
  );
};
