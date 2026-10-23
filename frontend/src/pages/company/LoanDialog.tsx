import React, { useEffect, useState } from 'react';
import { Api } from '../../services/api';
import type {
  CompanyShareholder,
  Contact,
  LoanDayCount,
  LoanDirection,
  ShareholderLoanRequest,
  ShareholderLoanView,
} from '../../types';
import { useWriteLock } from '../../components/WriteLock';
import { formatCentsPlain, parseCents } from '../../utils/formatters';
import {
  Button,
  Checkbox,
  Dialog,
  Field,
  FormGrid,
  Input,
  Notice,
  RadioGroup,
  Select,
  toast,
} from '../../components/ui';

const DAY_COUNTS: { value: LoanDayCount; label: string }[] = [
  { value: 'act/365', label: 'Taggenau, Jahr mit 365 Tagen' },
  { value: 'act/360', label: 'Taggenau, Jahr mit 360 Tagen' },
  { value: '30/360', label: 'Deutsche Zinsmethode, 30/360' },
];

function normalize(text: string): string {
  return text.trim().toLowerCase().replace(/\s+/g, ' ');
}

/** Die Kontakte, deren Name in der Gesellschafterliste steht. */
export function shareholderContacts(contacts: Contact[], shareholders: CompanyShareholder[]): Contact[] {
  const names = shareholders.map((s) => normalize(s.name)).filter(Boolean);
  return contacts.filter((c) =>
    [c.name, c.company].some((name) => name && names.includes(normalize(name))),
  );
}

function rateText(bp: number): string {
  return (bp / 100).toFixed(2).replace('.', ',');
}

function parseRate(text: string): number | null {
  const value = Number(text.replace(/\s|%/g, '').replace(',', '.'));
  return Number.isFinite(value) && value >= 0 ? Math.round(value * 100) : null;
}

interface LoanDialogProps {
  /** Der Vertrag, der berichtigt wird; ohne ihn entsteht ein neuer. */
  loan?: ShareholderLoanView;
  shareholders: CompanyShareholder[];
  onClose: () => void;
  onSaved: () => void;
}

/**
 * Ein Darlehensvertrag mit einem Gesellschafter. Konten leitet Buchfink aus
 * Richtung, Laufzeit und Beteiligung ab; Auszahlung, Tilgung und Zinsen bucht
 * danach der Agent oder die Bank.
 */
export const LoanDialog: React.FC<LoanDialogProps> = ({ loan, shareholders, onClose, onSaved }) => {
  const writeLock = useWriteLock();
  const [contacts, setContacts] = useState<Contact[]>([]);
  const [contactId, setContactId] = useState<number>(loan?.loan.contactId ?? 0);
  const [direction, setDirection] = useState<LoanDirection>(loan?.loan.direction ?? 'to_company');
  const [principal, setPrincipal] = useState(loan ? formatCentsPlain(loan.loan.principal) : '');
  const [rate, setRate] = useState(loan ? rateText(loan.loan.interestRateBp) : '');
  const [dayCount, setDayCount] = useState<LoanDayCount>(loan?.loan.dayCount ?? 'act/365');
  const [contractDate, setContractDate] = useState(loan?.loan.contractDate ?? '');
  const [startDate, setStartDate] = useState(loan?.loan.startDate ?? '');
  const [maturityDate, setMaturityDate] = useState(loan?.loan.maturityDate ?? '');
  const [collateral, setCollateral] = useState(loan?.loan.collateral ?? '');
  const [subordinated, setSubordinated] = useState(loan?.loan.subordinated ?? false);
  const [reason, setReason] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    Api.getSelectableContacts()
      .then((list) => {
        const candidates = shareholderContacts(list, shareholders);
        setContacts(candidates);
        if (!loan && candidates.length === 1) setContactId(candidates[0].id);
      })
      .catch(() => setContacts([]));
  }, [loan, shareholders]);

  async function save() {
    const cents = parseCents(principal);
    const bp = parseRate(rate);
    if (!contactId) return setError('Wählen Sie den Gesellschafter.');
    if (!cents || cents <= 0) return setError('Der Darlehensbetrag fehlt.');
    if (bp === null) return setError('Der Zinssatz fehlt; ein zinsloses Darlehen hat 0,00 %.');
    if (!startDate) return setError('Der Beginn fehlt.');
    if (loan && !reason.trim()) return setError('Eine Berichtigung braucht einen Grund.');
    const request: ShareholderLoanRequest = {
      contactId,
      direction,
      principal: cents,
      interestRateBp: bp,
      dayCount,
      contractDate: contractDate || undefined,
      startDate,
      maturityDate: maturityDate || undefined,
      collateral: collateral.trim() || undefined,
      subordinated,
    };
    setBusy(true);
    setError('');
    try {
      if (loan) {
        await Api.correctShareholderLoan(loan.loan.id, request, reason.trim());
        toast.success('Darlehensvertrag berichtigt.');
      } else {
        await Api.registerShareholderLoan(request);
        toast.success('Darlehensvertrag erfasst.');
      }
      onSaved();
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
      title={loan ? `Darlehen ${loan.contactName} berichtigen` : 'Darlehensvertrag erfassen'}
      width="max-w-2xl"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Abbrechen
          </Button>
          <Button
            variant="primary"
            loading={busy}
            disabled={writeLock.locked}
            title={writeLock.hint}
            onClick={() => void save()}
          >
            {loan ? 'Berichtigen' : 'Vertrag erfassen'}
          </Button>
        </>
      }
    >
      {contacts.length === 0 && (
        <Notice
          className="mb-5"
          text="Kein Kontakt passt zur Gesellschafterliste. Legen Sie den Gesellschafter unter Kontakte mit demselben Namen an."
        />
      )}
      <FormGrid>
        <Field label="Gesellschafter">
          <Select
            items={contacts.map((c) => ({ value: c.id, label: c.company || c.name }))}
            value={contactId || undefined}
            onValueChange={(id) => setContactId(Number(id))}
          />
        </Field>
        <Field label="Wer leiht">
          <RadioGroup<LoanDirection>
            options={[
              { value: 'to_company', label: 'Gesellschafter an die GmbH' },
              { value: 'to_shareholder', label: 'GmbH an den Gesellschafter' },
            ]}
            value={direction}
            onValueChange={setDirection}
          />
        </Field>
        <Field label="Betrag">
          <Input
            align="right"
            placeholder="0,00"
            value={principal}
            onChange={(e) => setPrincipal(e.target.value)}
            onBlur={() => {
              const cents = parseCents(principal);
              if (cents) setPrincipal(formatCentsPlain(cents));
            }}
          />
        </Field>
        <Field
          label="Zinssatz"
          hint="Prozent im Jahr"
          helpSummary="Ein unüblicher Zins ist kein Fehler, aber ein Befund für den Fremdvergleich."
          explain="Zahlt die GmbH mehr oder verlangt sie weniger, als ein fremder Dritter vereinbart hätte, kann eine verdeckte Gewinnausschüttung vorliegen (§ 8 Abs. 3 Satz 2 KStG). Buchfink zeigt das als Befund und sperrt nichts."
        >
          <Input align="right" placeholder="3,50" value={rate} onChange={(e) => setRate(e.target.value)} />
        </Field>
        <Field label="Zinsmethode">
          <Select items={DAY_COUNTS} value={dayCount} onValueChange={(v) => setDayCount(v as LoanDayCount)} />
        </Field>
        <Field label="Vertrag vom" optional>
          <Input type="date" value={contractDate} onChange={(e) => setContractDate(e.target.value)} />
        </Field>
        <Field label="Beginn" hint="Erste Auszahlung">
          <Input type="date" value={startDate} onChange={(e) => setStartDate(e.target.value)} />
        </Field>
        <Field label="Fällig am" optional hint="Leer: ohne feste Laufzeit">
          <Input type="date" value={maturityDate} onChange={(e) => setMaturityDate(e.target.value)} />
        </Field>
        <Field label="Sicherheiten" optional className="md:col-span-2">
          <Input value={collateral} onChange={(e) => setCollateral(e.target.value)} />
        </Field>
        <div className="md:col-span-2">
          <Checkbox
            checked={subordinated}
            onCheckedChange={(next) => setSubordinated(Boolean(next))}
            label="Rangrücktritt vereinbart"
            hint="Zählt bei der Prüfung auf Überschuldung nicht als Schuld"
          />
        </div>
        {loan && (
          <Field label="Grund der Berichtigung" hint="Steht im Änderungsprotokoll" className="md:col-span-2">
            <Input value={reason} placeholder="Nachtrag zum Vertrag" onChange={(e) => setReason(e.target.value)} />
          </Field>
        )}
      </FormGrid>
      {error && <Notice tone="negative" className="mt-5" text={error} />}
    </Dialog>
  );
};
