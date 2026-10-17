import React from 'react';
import {
  AREA_TABS,
  TAB_LABELS,
  activeTab,
  useAnchorScroll,
  useNavigation,
  type TabOf,
} from '../../navigation';
import { useWorkspace } from '../../workspace';
import { PageFrame, type FrameTab } from '../../components/ui';
import { ContactsPanel } from './ContactsPanel';
import { ProfilePage } from './ProfilePage';

type CompanyTab = TabOf<'company'>;

const TABS: FrameTab<CompanyTab>[] = AREA_TABS.company.map((value) => ({
  value,
  label: TAB_LABELS.company[value],
}));

/**
 * Das Unternehmen: Stammdaten, die nur der Mensch pflegt, und die Kontakte.
 * Die Kennung im Reiter „Stammdaten" ist der Block oder die Pflicht, zu der
 * gesprungen wird (firma, anschrift, steuern, bankkonten, gesellschafter); bei
 * den Kontakten die Kennung des Kontakts.
 */
export const CompanyArea: React.FC = () => {
  const { route, navigate } = useNavigation();
  const { activeTenant } = useWorkspace();
  const tab = activeTab(route, 'company');
  const id = route.area === 'company' ? route.id : undefined;

  useAnchorScroll(tab === 'profile' ? id : undefined);

  return (
    <PageFrame
      title="Unternehmen"
      context="Stammdaten pflegen nur Sie, keine verbundene App"
      tabs={TABS}
      tab={tab}
      onTabChange={(next) => navigate({ area: 'company', tab: next })}
    >
      {tab === 'profile' && <ProfilePage key={activeTenant?.id} />}
      {tab === 'contacts' && <ContactsPanel key={activeTenant?.id} />}
    </PageFrame>
  );
};
