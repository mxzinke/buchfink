import React, { useCallback, useEffect, useState } from 'react';
import { Api } from '../../services/api';
import { useLiveReload } from '../../live';
import type {
  CompanySettings,
  Country,
  Deadline,
  FoundationRules,
  FoundationState,
} from '../../types';
import { useWorkspace } from '../../workspace';
import { Notice, SkeletonRows } from '../../components/ui';
import { AddressBlock } from './AddressBlock';
import { BankBlock } from './BankBlock';
import { CapitalBlock } from './CapitalBlock';
import { FirmBlock } from './FirmBlock';
import { COMPANY_OBLIGATIONS } from './ObligationList';
import { TaxBlock } from './TaxBlock';

interface ProfileData {
  settings: CompanySettings;
  tradeTaxRatePercent: number;
  foundation: FoundationState | null;
  rules: FoundationRules[];
  countries: Country[];
  deadlines: Deadline[];
}

const COMPANY_KINDS = [...COMPANY_OBLIGATIONS.firma, ...COMPANY_OBLIGATIONS.gesellschafter];

/** Die Pflichten zum Unternehmen aus allen Geschäftsjahren, jede einmal. */
async function companyDeadlines(years: number[]): Promise<Deadline[]> {
  const lists = await Promise.all(years.map((y) => Api.getDeadlines(y).catch(() => [] as Deadline[])));
  const byKey = new Map<string, Deadline>();
  for (const d of lists.flat()) {
    if (COMPANY_KINDS.some((kind) => d.key.startsWith(`${kind}.`))) byKey.set(d.key, d);
  }
  return Array.from(byKey.values()).sort((a, b) => a.dueDate.localeCompare(b.dueDate));
}

/**
 * Die Stammdaten in fünf Blöcken. Jeder Block zeigt, was fehlt, und die
 * wichtigsten Angaben; bearbeitet wird in der Seitenleiste. Die Stammdaten
 * pflegt nur der Mensch; keine verbundene App ändert sie.
 */
export const ProfilePage: React.FC = () => {
  const workspace = useWorkspace();
  const [data, setData] = useState<ProfileData | null>(null);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    try {
      const [settings, closing, rules, countries] = await Promise.all([
        Api.getCompanySettings(),
        Api.getClosingSettings(),
        Api.getFoundationRules(),
        Api.getCountries(),
      ]);
      const [foundation, deadlines] = await Promise.all([
        Api.getFoundationState().catch(() => null),
        companyDeadlines(workspace.availableYears),
      ]);
      setData({
        settings,
        tradeTaxRatePercent: closing?.tradeTaxRatePercent ?? 400,
        foundation,
        rules,
        countries,
        deadlines,
      });
      setError('');
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }, [workspace.availableYears]);

  useEffect(() => {
    void load();
  }, [load]);

  // Darlehen, Gründungsbuchungen und Fristen kann auch Claude ändern.
  useLiveReload(load, ['stammdaten', 'buchungen', 'abschluss']);

  /** Speichert einen Block auf dem frisch gelesenen Stand der übrigen. */
  const saveSettings = useCallback(
    async (patch: Partial<CompanySettings>) => {
      const fresh = await Api.getCompanySettings();
      await Api.updateCompanySettings({ ...fresh, ...patch });
      await load();
      await workspace.reloadSettings();
    },
    [load, workspace],
  );

  const saveTaxes = useCallback(
    async (patch: Partial<CompanySettings>, tradeTaxRatePercent: number) => {
      const fresh = await Api.getCompanySettings();
      await Api.updateCompanySettings({ ...fresh, ...patch });
      const closing = await Api.getClosingSettings();
      if (closing.tradeTaxRatePercent !== tradeTaxRatePercent) {
        await Api.saveClosingSettings({ ...closing, tradeTaxRatePercent });
      }
      await load();
      await workspace.reloadSettings();
    },
    [load, workspace],
  );

  if (!data) {
    return error ? <Notice tone="negative" text={error} /> : <SkeletonRows rows={10} />;
  }

  const { settings } = data;
  const rules = data.rules.find((r) => r.legalForm === settings.legalForm);
  const firstYear = workspace.availableYears.length ? Math.min(...workspace.availableYears) : workspace.year;

  return (
    <div key={workspace.activeTenant?.id}>
      <FirmBlock
        settings={settings}
        foundation={data.foundation}
        deadlines={data.deadlines}
        onSave={saveSettings}
        onReload={() => void load()}
      />
      <AddressBlock settings={settings} countries={data.countries} onSave={saveSettings} />
      <TaxBlock
        settings={settings}
        tradeTaxRatePercent={data.tradeTaxRatePercent}
        year={workspace.year}
        onSave={saveTaxes}
      />
      <BankBlock settings={settings} onSave={saveSettings} onChanged={() => void load()} />
      <CapitalBlock
        settings={settings}
        foundation={data.foundation}
        rules={rules}
        deadlines={data.deadlines}
        firstYear={firstYear}
        onSave={saveSettings}
        onReload={() => void load()}
      />
    </div>
  );
};
