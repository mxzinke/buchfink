import React, { useState } from 'react';
import { Api } from '../../services/api';
import type { CompanySettings, Deadline, FoundationState } from '../../types';
import { useWriteLock } from '../../components/WriteLock';
import { LEGAL_FORMS, fiscalYearLabel } from '../../utils/companyLabels';
import { formatDate } from '../../utils/formatters';
import {
  Button,
  DetailList,
  Dialog,
  Field,
  FieldValue,
  FormGrid,
  Input,
  Notice,
  Select,
  toast,
} from '../../components/ui';
import { EditPanel, Missing, ProfileBlock, PROFILE_LIST } from './ProfileBlock';
import { firmGaps } from '../../utils/profileGaps';
import { ObligationList, obligationsOf } from './ObligationList';
import { COMPANY_OBLIGATIONS } from '../../utils/obligations';

interface FirmBlockProps {
  settings: CompanySettings;
  foundation: FoundationState | null;
  deadlines: Deadline[];
  onSave: (patch: Partial<CompanySettings>) => Promise<void>;
  onReload: () => void;
}

type Draft = Pick<
  CompanySettings,
  | 'companyName'
  | 'legalForm'
  | 'foundedOn'
  | 'seat'
  | 'managingDirectors'
  | 'supervisoryBoardChair'
  | 'registerCourt'
  | 'registerNumber'
  | 'registeredOn'
  | 'sellerIdentifier'
>;

function draftOf(s: CompanySettings): Draft {
  return {
    companyName: s.companyName,
    legalForm: s.legalForm,
    foundedOn: s.foundedOn,
    seat: s.seat,
    managingDirectors: s.managingDirectors ?? '',
    supervisoryBoardChair: s.supervisoryBoardChair ?? '',
    registerCourt: s.registerCourt,
    registerNumber: s.registerNumber,
    registeredOn: s.registeredOn ?? '',
    sellerIdentifier: s.sellerIdentifier ?? '',
  };
}

/** Firma und Register: was im Kopf jedes Abschlusses und jeder Rechnung steht. */
export const FirmBlock: React.FC<FirmBlockProps> = ({ settings, foundation, deadlines, onSave, onReload }) => {
  const writeLock = useWriteLock();
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState<Draft>(() => draftOf(settings));
  const [registering, setRegistering] = useState(false);

  const inFounding = Boolean(settings.inGruendung);
  const missing = firmGaps(settings);

  const register = inFounding ? (
    <span className="flex flex-wrap items-center gap-3">
      <Missing>Noch nicht eingetragen</Missing>
      {foundation?.hasFoundation && (
        <Button
          variant="quiet"
          size="sm"
          disabled={writeLock.locked}
          title={writeLock.hint}
          onClick={() => setRegistering(true)}
        >
          Eintragung festhalten
        </Button>
      )}
    </span>
  ) : settings.registerCourt || settings.registerNumber ? (
    [
      settings.registerCourt,
      settings.registerNumber,
      settings.registeredOn ? `eingetragen am ${formatDate(settings.registeredOn)}` : '',
    ]
      .filter(Boolean)
      .join(' · ')
  ) : (
    <Missing />
  );

  const patch = (next: Partial<Draft>) => setDraft((prev) => ({ ...prev, ...next }));

  return (
    <ProfileBlock
      id="firma"
      first
      title="Firma & Register"
      missing={missing}
      context="Steht im Kopf jedes Abschlusses und auf jeder Rechnung"
      helpSummary="Diese Angaben stehen auf Rechnungen und im Kopf des Jahresabschlusses."
      explain="Jahresabschluss und Geschäftsbriefe einer GmbH nennen Firma, Sitz, Registergericht und Registernummer (§ 264 Abs. 1a HGB, § 35a GmbHG), Rechnungen zusätzlich alle Geschäftsführer."
      onEdit={() => {
        setDraft(draftOf(settings));
        setEditing(true);
      }}
    >
      <DetailList
        className={PROFILE_LIST}
        items={[
          { label: 'Firma', value: inFounding ? `${settings.companyName} i. G.` : settings.companyName || <Missing /> },
          { label: 'Rechtsform', value: settings.legalForm || <Missing /> },
          { label: 'Sitz', value: settings.seat || <Missing /> },
          { label: 'Handelsregister', value: register },
          { label: 'Geschäftsführung', value: settings.managingDirectors || <Missing /> },
          { label: 'Gegründet', value: settings.foundedOn ? formatDate(settings.foundedOn) : '—', numeric: true },
          { label: 'Geschäftsjahr', value: fiscalYearLabel(settings.fiscalYearStartMonth) },
        ]}
      />

      <ObligationList deadlines={obligationsOf(deadlines, COMPANY_OBLIGATIONS.firma)} onConfirmed={onReload} />

      <EditPanel
        open={editing}
        title="Firma & Register"
        onClose={() => setEditing(false)}
        onSave={() => onSave(draft)}
      >
        <FormGrid>
          <Field label="Firma" hint="Wie im Handelsregister" className="md:col-span-2">
            <Input value={draft.companyName} onChange={(e) => patch({ companyName: e.target.value })} />
          </Field>
          <Field label="Rechtsform">
            <Select items={LEGAL_FORMS} value={draft.legalForm} onValueChange={(v) => patch({ legalForm: String(v) })} />
          </Field>
          <Field label="Gegründet am" hint="Tag der Beurkundung">
            <Input type="date" value={draft.foundedOn} onChange={(e) => patch({ foundedOn: e.target.value })} />
          </Field>
          <Field label="Sitz" hint="Ort laut Satzung">
            <Input value={draft.seat} onChange={(e) => patch({ seat: e.target.value })} />
          </Field>
          <Field label="Geschäftsführung" hint="Alle, mit Vor- und Nachnamen">
            <Input
              value={draft.managingDirectors}
              placeholder="Anna Bauer, Jonas Peters"
              onChange={(e) => patch({ managingDirectors: e.target.value })}
            />
          </Field>
          {!inFounding && (
            <>
              <Field label="Registergericht">
                <Input
                  value={draft.registerCourt}
                  placeholder="Amtsgericht Hamburg"
                  onChange={(e) => patch({ registerCourt: e.target.value })}
                />
              </Field>
              <Field label="Registernummer">
                <Input
                  className="code-num"
                  value={draft.registerNumber}
                  placeholder="HRB 123456"
                  onChange={(e) => patch({ registerNumber: e.target.value })}
                />
              </Field>
              <Field label="Eingetragen am" optional>
                <Input
                  type="date"
                  value={draft.registeredOn ?? ''}
                  onChange={(e) => patch({ registeredOn: e.target.value })}
                />
              </Field>
            </>
          )}
          <Field label="Aufsichtsratsvorsitz" optional>
            <Input
              value={draft.supervisoryBoardChair}
              onChange={(e) => patch({ supervisoryBoardChair: e.target.value })}
            />
          </Field>
          <Field
            label="Verkäuferkennung für E-Rechnungen"
            optional
            helpSummary="Nur nötig, wenn weder USt-IdNr. noch Registernummer vorliegt."
            explain="Eine mit den Kunden abgestimmte feste Kennung, etwa Ihre Lieferantennummer. Die E-Rechnung verlangt eine Kennung des Verkäufers (BT-29), wenn weder USt-IdNr. noch Registernummer angegeben sind."
          >
            <Input
              value={draft.sellerIdentifier}
              onChange={(e) => patch({ sellerIdentifier: e.target.value })}
            />
          </Field>
          <Field
            label="Geschäftsjahr"
            helpSummary="Das Geschäftsjahr steht seit der Einrichtung fest."
            explain="Ein Wechsel des Wirtschaftsjahres braucht ein Rumpfgeschäftsjahr und die Zustimmung des Finanzamts; Buchfink bildet ihn nicht ab."
          >
            <FieldValue>{fiscalYearLabel(settings.fiscalYearStartMonth)}</FieldValue>
          </Field>
        </FormGrid>
      </EditPanel>

      {registering && (
        <RegisterDialog
          defaults={{ court: settings.registerCourt, number: settings.registerNumber }}
          onClose={() => setRegistering(false)}
          onDone={() => {
            setRegistering(false);
            onReload();
          }}
        />
      )}
    </ProfileBlock>
  );
};

/**
 * Die Eintragung ins Handelsregister beendet die Vorgesellschaft. Danach steht
 * die Unterbilanz auf diesen Tag fest; deshalb nur mit ausdrücklicher Bestätigung.
 */
const RegisterDialog: React.FC<{
  defaults: { court: string; number: string };
  onClose: () => void;
  onDone: () => void;
}> = ({ defaults, onClose, onDone }) => {
  const [date, setDate] = useState('');
  const [court, setCourt] = useState(defaults.court);
  const [number, setNumber] = useState(defaults.number);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  async function save() {
    setBusy(true);
    setError('');
    try {
      await Api.registerCompany(date, court.trim(), number.trim());
      toast.success('Eintragung festgehalten. Die Firma steht ab jetzt ohne „i. G.“.');
      onDone();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog
      open
      onOpenChange={(open) => !open && onClose()}
      title="Eintragung ins Handelsregister"
      width="max-w-lg"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Abbrechen
          </Button>
          <Button variant="primary" loading={busy} disabled={!date} onClick={() => void save()}>
            Eintragung festhalten
          </Button>
        </>
      }
    >
      <p className="text-body text-ink-muted mb-5">
        Mit der Eintragung endet die Vorgesellschaft. Die Vorbelastung der Gesellschafter steht dann auf
        diesen Tag fest; zurücknehmen lässt sich das nicht.
      </p>
      <FormGrid>
        <Field label="Eingetragen am">
          <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
        </Field>
        <div />
        <Field label="Registergericht">
          <Input value={court} placeholder="Amtsgericht Hamburg" onChange={(e) => setCourt(e.target.value)} />
        </Field>
        <Field label="Registernummer">
          <Input className="code-num" value={number} placeholder="HRB 123456" onChange={(e) => setNumber(e.target.value)} />
        </Field>
      </FormGrid>
      {error && <Notice tone="negative" className="mt-5" text={error} />}
    </Dialog>
  );
};
