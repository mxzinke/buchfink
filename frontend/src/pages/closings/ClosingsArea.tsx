import React, { useCallback, useEffect, useState } from 'react';
import { AREA_TABS, TAB_LABELS, activeTab, useNavigation, type TabOf } from '../../navigation';
import { useWorkspace } from '../../workspace';
import { Api } from '../../services/api';
import type { Obligation } from '../../types';
import { PageFrame, type FrameTab } from '../../components/ui';
import { DeclarationsTab } from './DeclarationsTab';
import { MonthTab } from './MonthTab';
import { YearTab } from './YearTab';
import { message, obligationTab } from './shared';

type ClosingsTab = TabOf<'closings'>;

/**
 * Die Abschlüsse: Monat (Prüfbericht, Festschreibung, Voranmeldung), Jahr
 * (Schritte bis zur Offenlegung) und die Erklärungen. Fristen und Stand kommen
 * aus dem Pflichtenkalender. Unumkehrbare Schritte bestätigt der Mensch hier.
 */
export const ClosingsArea: React.FC = () => {
  const { route, navigate } = useNavigation();
  const { year, refreshFiscalYears } = useWorkspace();
  const tab = activeTab(route, 'closings');
  const id = route.area === 'closings' ? route.id : undefined;
  const [obligations, setObligations] = useState<Obligation[]>([]);
  const [error, setError] = useState('');

  const loadObligations = useCallback(async () => {
    try {
      setObligations(await Api.getObligations(year));
      setError('');
    } catch (e) {
      // Der Reiter zeigt den Fehler zusammen mit seinem eigenen.
      setObligations([]);
      setError(message(e));
    }
  }, [year]);

  useEffect(() => {
    void loadObligations();
  }, [loadObligations]);

  // Ein Reiter zählt, was an ihm überfällig ist.
  const overdue = (target: ClosingsTab) =>
    obligations.filter((o) => o.status === 'overdue' && obligationTab(o.kind) === target).length;
  const tabs: FrameTab<ClosingsTab>[] = AREA_TABS.closings.map((value) => ({
    value,
    label: TAB_LABELS.closings[value],
    count: overdue(value) || undefined,
  }));

  return (
    <PageFrame
      title="Abschlüsse"
      context={`Geschäftsjahr ${year}`}
      tabs={tabs}
      tab={tab}
      onTabChange={(next) => navigate({ area: 'closings', tab: next })}
    >
      {tab === 'month' && (
        <MonthTab
          key={year}
          year={year}
          obligations={obligations}
          obligationsError={error}
          onObligationsChanged={loadObligations}
          id={id}
          onNavigate={navigate}
        />
      )}
      {tab === 'year' && (
        <YearTab
          key={year}
          year={year}
          obligations={obligations}
          obligationsError={error}
          onObligationsChanged={loadObligations}
          onFiscalYearChanged={refreshFiscalYears}
          id={id}
          onNavigate={navigate}
        />
      )}
      {tab === 'declarations' && (
        <DeclarationsTab
          key={year}
          year={year}
          obligations={obligations}
          obligationsError={error}
          onObligationsChanged={loadObligations}
          id={id}
          onNavigate={navigate}
        />
      )}
    </PageFrame>
  );
};
