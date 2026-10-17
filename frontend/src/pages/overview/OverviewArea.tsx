import React from 'react';
import { useNavigation } from '../../navigation';
import { useWorkspace } from '../../workspace';
import { PageFrame } from '../../components/ui';
import { TasksPage } from '../TasksPage';

/**
 * Die Startseite: was jetzt zu tun ist und was nur der Mensch freigeben kann.
 * Vorerst die Aufgabenliste mit Monatsabschluss und Kennzahlen.
 */
export const OverviewArea: React.FC = () => {
  const { navigate } = useNavigation();
  const { year } = useWorkspace();
  return (
    <PageFrame title="Übersicht" context={`Geschäftsjahr ${year}`}>
      <TasksPage onNavigate={navigate} />
    </PageFrame>
  );
};
