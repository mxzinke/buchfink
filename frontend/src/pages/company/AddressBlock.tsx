import React, { useState } from 'react';
import type { CompanySettings, Country } from '../../types';
import { DetailList, Field, FormGrid, Input, Select } from '../../components/ui';
import { EditPanel, Missing, ProfileBlock, PROFILE_LIST } from './ProfileBlock';

type Draft = Pick<
  CompanySettings,
  'street' | 'houseNumber' | 'addressAddition' | 'postalCode' | 'city' | 'countryCode'
>;

function draftOf(s: CompanySettings): Draft {
  return {
    street: s.street,
    houseNumber: s.houseNumber ?? '',
    addressAddition: s.addressAddition,
    postalCode: s.postalCode,
    city: s.city,
    countryCode: s.countryCode || 'DE',
  };
}

/** Die Geschäftsanschrift, wie sie auf Rechnungen und E-Rechnungen steht. */
export const AddressBlock: React.FC<{
  settings: CompanySettings;
  countries: Country[];
  onSave: (patch: Partial<CompanySettings>) => Promise<void>;
}> = ({ settings, countries, onSave }) => {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState<Draft>(() => draftOf(settings));
  const patch = (next: Partial<Draft>) => setDraft((prev) => ({ ...prev, ...next }));

  const missing = [
    !settings.street && 'Straße',
    !settings.postalCode && 'Postleitzahl',
    !settings.city && 'Ort',
  ].filter(Boolean) as string[];
  const country = countries.find((c) => c.code === (settings.countryCode || 'DE'))?.name ?? settings.countryCode;
  const streetLine = [settings.street, settings.houseNumber].filter(Boolean).join(' ');

  return (
    <ProfileBlock
      id="anschrift"
      title="Anschrift"
      missing={missing}
      context="Auf Rechnungen und in der E-Rechnung"
      helpSummary="Rechnungen nennen die vollständige Anschrift des Ausstellers."
      explain="§ 14 Abs. 4 Nr. 1 UStG verlangt die vollständige Anschrift; ein Postfach genügt. Die E-Rechnung führt Straße, Postleitzahl, Ort und Land in getrennten Feldern."
      onEdit={() => {
        setDraft(draftOf(settings));
        setEditing(true);
      }}
    >
      <DetailList
        className={PROFILE_LIST}
        items={[
          { label: 'Straße', value: streetLine || <Missing /> },
          ...(settings.addressAddition ? [{ label: 'Zusatz', value: settings.addressAddition }] : []),
          {
            label: 'Ort',
            value: settings.postalCode || settings.city ? `${settings.postalCode} ${settings.city}`.trim() : <Missing />,
          },
          { label: 'Land', value: country },
        ]}
      />

      <EditPanel open={editing} title="Anschrift" onClose={() => setEditing(false)} onSave={() => onSave(draft)}>
        <FormGrid>
          <Field label="Straße" hint="oder Postfach">
            <Input value={draft.street} onChange={(e) => patch({ street: e.target.value })} />
          </Field>
          <Field label="Hausnummer" optional>
            <Input value={draft.houseNumber} onChange={(e) => patch({ houseNumber: e.target.value })} />
          </Field>
          <Field label="Zusatz" optional className="md:col-span-2">
            <Input
              value={draft.addressAddition}
              placeholder="c/o, Gebäude oder Etage"
              onChange={(e) => patch({ addressAddition: e.target.value })}
            />
          </Field>
          <Field label="Postleitzahl">
            <Input className="code-num" value={draft.postalCode} onChange={(e) => patch({ postalCode: e.target.value })} />
          </Field>
          <Field label="Ort">
            <Input value={draft.city} onChange={(e) => patch({ city: e.target.value })} />
          </Field>
          <Field label="Land">
            <Select
              items={countries.map((c) => ({ value: c.code, label: c.name }))}
              value={draft.countryCode}
              onValueChange={(code) => patch({ countryCode: String(code) })}
            />
          </Field>
        </FormGrid>
      </EditPanel>
    </ProfileBlock>
  );
};
