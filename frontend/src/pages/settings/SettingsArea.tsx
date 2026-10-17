import React, { useEffect, useState } from 'react';
import { useAnchorScroll, useNavigation, type TabOf } from '../../navigation';
import { useWorkspace } from '../../workspace';
import { Button, Disclosure, PageFrame, TabPanel, Tabs } from '../../components/ui';
import { AuditPage } from '../AuditPage';
import { BackupPage } from '../BackupPage';
import { SettingsPage } from '../SettingsPage';
import { TaxAuditPage } from '../TaxAuditPage';
import { AppsPanel } from './AppsPanel';
import { TenantsPanel } from './TenantsPanel';

type SettingsSection = TabOf<'settings'>;

/** Kennung im Abschnitt „Prüfung & GoBD", die die Betriebsprüfung aufschlägt. */
const TAX_AUDIT_ID = 'betriebspruefung';

type AuditView = 'uebersicht' | 'betriebspruefung';

/**
 * Die Einstellungen als eine Seite: oben, was man braucht (Apps, Sicherung,
 * Mandanten), darunter eingeklappt Prüfung und Erweitertes. Der Reiter im Ziel
 * benennt den Abschnitt, der aufgeklappt und angesprungen wird.
 */
export const SettingsArea: React.FC = () => {
  const { route, navigate } = useNavigation();
  const workspace = useWorkspace();
  const section: SettingsSection | undefined = route.area === 'settings' ? route.tab : undefined;
  const id = route.area === 'settings' ? route.id : undefined;

  const [open, setOpen] = useState({ audit: section === 'audit', advanced: section === 'advanced' });
  const [auditView, setAuditView] = useState<AuditView>(
    id === TAX_AUDIT_ID ? 'betriebspruefung' : 'uebersicht',
  );
  // Mit einem Mandanten gibt es nichts zu verwalten; der Abschnitt erscheint
  // erst, wenn er gefragt ist.
  const showTenants = workspace.tenants.length > 1 || section === 'tenants';

  useEffect(() => {
    if (section === 'audit' || section === 'advanced') {
      setOpen((prev) => ({ ...prev, [section]: true }));
    }
    if (section === 'audit') setAuditView(id === TAX_AUDIT_ID ? 'betriebspruefung' : 'uebersicht');
  }, [section, id]);
  // Der erste Abschnitt steht oben; dorthin zu springen hieße, den Titel aus
  // dem Blick zu schieben.
  useAnchorScroll(section && section !== 'apps' ? `einstellungen-${section}` : undefined, id);

  return (
    <PageFrame title="Einstellungen" context="Apps, Sicherung und Prüfung">
      <AppsPanel id="einstellungen-apps" />

      <section id="einstellungen-backup" className="mt-8 pt-8 border-t border-line">
        <BackupPage
          appConfig={workspace.appConfig}
          onAppConfigChange={workspace.setAppConfig}
          onRestored={workspace.reload}
        />
      </section>

      {showTenants && <TenantsPanel id="einstellungen-tenants" />}

      <Disclosure
        id="einstellungen-audit"
        title="Prüfung & GoBD"
        context="Integrität, Änderungsprotokoll, Prüfermodus, Datenüberlassung"
        open={open.audit}
        onOpenChange={(next) => setOpen((prev) => ({ ...prev, audit: next }))}
      >
        <Tabs<AuditView>
          items={[
            { value: 'uebersicht', label: 'Prüfübersicht' },
            { value: 'betriebspruefung', label: 'Betriebsprüfung' },
          ]}
          value={auditView}
          onValueChange={setAuditView}
        >
          <TabPanel value="uebersicht">
            {auditView === 'uebersicht' && (
              <AuditPage
                integrity={workspace.integrity}
                isChecking={workspace.isCheckingIntegrity}
                onRefreshIntegrity={() => void workspace.refreshIntegrity()}
                initialRule={id && id !== TAX_AUDIT_ID ? id : undefined}
              />
            )}
          </TabPanel>
          <TabPanel value="betriebspruefung">
            {auditView === 'betriebspruefung' && (
              <TaxAuditPage
                year={workspace.year}
                availableYears={workspace.availableYears}
                appConfig={workspace.appConfig}
                onAppConfigChange={workspace.setAppConfig}
              />
            )}
          </TabPanel>
        </Tabs>
      </Disclosure>

      <Disclosure
        id="einstellungen-advanced"
        title="Erweitert"
        context="Fristen, Prüfläufe, Abschluss, Speicherort"
        open={open.advanced}
        onOpenChange={(next) => setOpen((prev) => ({ ...prev, advanced: next }))}
      >
        {!showTenants && (
          <div className="mb-6">
            <Button
              variant="secondary"
              size="sm"
              onClick={() => navigate({ area: 'settings', tab: 'tenants' })}
            >
              Mandanten verwalten
            </Button>
          </div>
        )}
        <SettingsPage
          key={workspace.activeTenant?.id}
          part="advanced"
          year={workspace.year}
          onNavigate={navigate}
          onTenantDeleted={workspace.reload}
          onSaved={workspace.reloadSettings}
        />
      </Disclosure>
    </PageFrame>
  );
};
