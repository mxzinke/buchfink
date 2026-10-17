import React, { useEffect, useState } from 'react';
import {
  AREA_TABS,
  TAB_LABELS,
  activeTab,
  useAnchorScroll,
  useNavigation,
  type TabOf,
} from '../../navigation';
import { useWorkspace } from '../../workspace';
import type { ClosingStepKey } from '../../types';
import { Disclosure, PageFrame, type FrameTab } from '../../components/ui';
import { ClosingModulesPage, STEP_OBLIGATIONS, STEP_TABS, isModuleTab } from '../ClosingModulesPage';
import { ClosingPage } from '../ClosingPage';
import { DeadlinesPage } from '../DeadlinesPage';
import { EBilanzPage } from '../EBilanzPage';
import { ObligationsPage, isObligationTab } from '../ObligationsPage';
import { VatPage } from '../VatPage';

type ClosingsTab = TabOf<'closings'>;

const TABS: FrameTab<ClosingsTab>[] = AREA_TABS.closings.map((value) => ({
  value,
  label: TAB_LABELS.closings[value],
}));

/**
 * Die Abschlüsse: Monat (Prüfbericht, Festschreibung, Voranmeldung), Jahr
 * (Schritte bis zur Offenlegung) und die Erklärungen. Unumkehrbare Schritte
 * bestätigt der Mensch hier.
 */
export const ClosingsArea: React.FC = () => {
  const { route, navigate } = useNavigation();
  const { year, refreshFiscalYears } = useWorkspace();
  const tab = activeTab(route, 'closings');
  const id = route.area === 'closings' ? route.id : undefined;

  return (
    <PageFrame
      title="Abschlüsse"
      context={`Geschäftsjahr ${year}`}
      tabs={TABS}
      tab={tab}
      onTabChange={(next) => navigate({ area: 'closings', tab: next })}
    >
      {tab === 'month' && (
        <>
          <DeadlinesPage initialKey={id} onNavigate={navigate} />
          <div className="mt-8 pt-8 border-t border-line">
            <VatPage year={year} onNavigate={navigate} />
          </div>
        </>
      )}
      {tab === 'year' && (
        <YearTab year={year} id={id} onFiscalYearChanged={refreshFiscalYears} />
      )}
      {tab === 'declarations' && <EBilanzPage year={year} />}
    </PageFrame>
  );
};

/**
 * Wohin eine Kennung im Reiter „Jahr" führt: ein Schritt des Abschlusses oder
 * ein Reiter der Bausteine oder Nebenpflichten.
 */
function yearTarget(id: string | undefined): { section: 'modules' | 'obligations'; tab: string } | null {
  if (!id) return null;
  const step = id as ClosingStepKey;
  const moduleTab = STEP_TABS[step];
  if (moduleTab) return { section: 'modules', tab: moduleTab };
  const obligationTab = STEP_OBLIGATIONS[step];
  if (obligationTab) return { section: 'obligations', tab: obligationTab };
  if (isModuleTab(id)) return { section: 'modules', tab: id };
  if (isObligationTab(id)) return { section: 'obligations', tab: id };
  return null;
}

const YearTab: React.FC<{
  year: number;
  id: string | undefined;
  onFiscalYearChanged: () => Promise<void>;
}> = ({ year, id, onFiscalYearChanged }) => {
  const { navigate } = useNavigation();
  const target = yearTarget(id);
  const [openSection, setOpenSection] = useState<Record<'modules' | 'obligations', boolean>>({
    modules: target?.section === 'modules',
    obligations: target?.section === 'obligations',
  });

  // Ein Verweis aus der Schrittliste oder von einer anderen Seite klappt den
  // Abschnitt auf und springt hin.
  useEffect(() => {
    if (target) setOpenSection((prev) => ({ ...prev, [target.section]: true }));
  }, [target?.section, target?.tab]);
  useAnchorScroll(target ? `jahr-${target.section}` : undefined, id);

  return (
    <>
      <ClosingPage year={year} onFiscalYearChanged={onFiscalYearChanged} onNavigate={navigate} />
      {/* Bausteine und Nebenpflichten bucht der Agent. Ihre Verzeichnisse
          bleiben lesbar, bis die Schrittliste sie selbst zeigt. */}
      <Disclosure
        id="jahr-modules"
        title="Abschlussbausteine"
        context="Abgrenzung, Rückstellungen, Steuern, Ergebnis, Anhang"
        open={openSection.modules}
        onOpenChange={(open) => setOpenSection((prev) => ({ ...prev, modules: open }))}
      >
        <ClosingModulesPage
          year={year}
          initialTab={target?.section === 'modules' ? target.tab : undefined}
          onNavigate={navigate}
        />
      </Disclosure>
      <Disclosure
        id="jahr-obligations"
        title="Nebenpflichten"
        context="Vorsteuerberichtigung, USt-IdNr., Fremdwährung, Anlagen"
        open={openSection.obligations}
        onOpenChange={(open) => setOpenSection((prev) => ({ ...prev, obligations: open }))}
      >
        <ObligationsPage
          year={year}
          initialTab={target?.section === 'obligations' ? target.tab : undefined}
          onNavigate={navigate}
        />
      </Disclosure>
    </>
  );
};
