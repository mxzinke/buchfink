import React, { useEffect, useState } from 'react';
import { useAnchorScroll, useNavigation, type TabOf } from '../../navigation';
import { useWorkspace } from '../../workspace';
import { Disclosure, PageFrame } from '../../components/ui';
import { AdvancedSettings } from './AdvancedSettings';
import { AppsPanel } from './AppsPanel';
import { InboxPanel } from './InboxPanel';
import { BackupPanel } from './BackupPanel';
import { TenantsPanel } from './TenantsPanel';
import { AuditSection, auditAnchor } from './audit/AuditSection';

type SettingsSection = TabOf<'settings'>;

/**
 * Die Einstellungen als eine Seite: oben Apps verbinden, Eingang und Ordner
 * und die Sicherung, darunter eingeklappt Prüfung & GoBD und Erweitertes. Der Reiter im Ziel benennt den
 * Abschnitt, der aufgeklappt und angesprungen wird; die Kennung eine Zeile
 * darin.
 */
export const SettingsArea: React.FC = () => {
  const { route, navigate } = useNavigation();
  const workspace = useWorkspace();
  const section: SettingsSection | undefined = route.area === 'settings' ? route.tab : undefined;
  const id = route.area === 'settings' ? route.id : undefined;

  const [open, setOpen] = useState({ audit: section === 'audit', advanced: section === 'advanced' });
  // Mit einem Mandanten gibt es nichts zu verwalten; der Abschnitt erscheint
  // erst, wenn er gefragt ist.
  const showTenants = workspace.tenants.length > 1 || section === 'tenants';

  useEffect(() => {
    if (section === 'audit' || section === 'advanced') {
      setOpen((prev) => ({ ...prev, [section]: true }));
    }
  }, [section, id]);

  // Der erste Abschnitt steht oben; dorthin zu springen hieße, den Titel aus
  // dem Blick zu schieben. Eine Zeile der Prüfung hat ihren eigenen Anker.
  useAnchorScroll(
    section && section !== 'apps'
      ? (section === 'audit' && auditAnchor(id)) || `einstellungen-${section}`
      : undefined,
    id,
  );

  return (
    <PageFrame title="Einstellungen" context="Apps, Eingang, Sicherung und Prüfung">
      <AppsPanel id="einstellungen-apps" />

      <InboxPanel id="einstellungen-inbox" />

      <BackupPanel
        id="einstellungen-backup"
        appConfig={workspace.appConfig}
        onAppConfigChange={workspace.setAppConfig}
        onRestored={workspace.refreshTenants}
      />

      {showTenants && <TenantsPanel id="einstellungen-tenants" />}

      <Disclosure
        id="einstellungen-audit"
        title="Prüfung & GoBD"
        context="Integrität, Änderungsprotokoll, Prüfermodus, Datenüberlassung, Verfahrensdokumentation"
        open={open.audit}
        onOpenChange={(next) => setOpen((prev) => ({ ...prev, audit: next }))}
      >
        <AuditSection target={section === 'audit' ? id : undefined} />
      </Disclosure>

      <Disclosure
        id="einstellungen-advanced"
        title="Erweitert"
        context="Fristen der Prüfläufe, Abgrenzung, Bankregeln, Speicherort"
        open={open.advanced}
        onOpenChange={(next) => setOpen((prev) => ({ ...prev, advanced: next }))}
      >
        <AdvancedSettings
          key={workspace.activeTenant?.id}
          onManageTenants={showTenants ? undefined : () => navigate({ area: 'settings', tab: 'tenants' })}
        />
      </Disclosure>
    </PageFrame>
  );
};
