import React, { useEffect, useState } from 'react';
import { Save } from 'lucide-react';
import { Api } from '../../services/api';
import type {
  AccrualMethod,
  AccrualReleaseCycle,
  BankRule,
  ClosingSettings,
  CompanySettings,
  ComplianceHints,
  RetentionRules,
} from '../../types';
import { RETENTION_CLASS_LABELS } from '../../types';
import { useWorkspace } from '../../workspace';
import { useWriteLock } from '../../components/WriteLock';
import { formatCentsPlain, formatDate, parseCents } from '../../utils/formatters';
import {
  Button,
  Disclosure,
  Field,
  FieldValue,
  FormGrid,
  Input,
  Notice,
  Select,
  SkeletonRows,
  Table,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
  toast,
} from '../../components/ui';

function message(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

const ACCRUAL_METHODS = [
  { value: 'monthly', label: 'Monatsgenau · nach Zwölfteln' },
  { value: 'daily', label: 'Taggenau · nach Kalendertagen' },
];

const ACCRUAL_RELEASES = [
  { value: 'yearly', label: 'Einmal je Geschäftsjahr · am ersten Tag' },
  { value: 'monthly', label: 'Monatlich · für unterjährige Auswertungen' },
];

/** Eine Zahl aus einem Textfeld; leer oder ungültig ergibt den Ersatzwert. */
function wholeNumber(text: string, fallback: number, min: number): number {
  const value = Math.trunc(Number(text));
  return Number.isFinite(value) && value >= min && text.trim() !== '' ? value : fallback;
}

/**
 * Was selten jemand anfasst: Fristen der Prüfläufe, Verfahren der Abgrenzung,
 * die Grenze des Leistungsnachweises, gelernte Bankregeln, Aufbewahrungsfristen
 * und Speicherort. Gespeichert wird über einen Knopf.
 */
export const AdvancedSettings: React.FC<{ onManageTenants?: () => void }> = ({ onManageTenants }) => {
  const workspace = useWorkspace();
  const writeLock = useWriteLock();
  const [settings, setSettings] = useState<CompanySettings | null>(null);
  const [closing, setClosing] = useState<ClosingSettings | null>(null);
  const [hints, setHints] = useState<ComplianceHints | null>(null);
  const [retention, setRetention] = useState<RetentionRules | null>(null);
  const [rules, setRules] = useState<BankRule[]>([]);
  const [captureDays, setCaptureDays] = useState('10');
  const [graceDays, setGraceDays] = useState('0');
  const [threshold, setThreshold] = useState('');
  const [checkThreshold, setCheckThreshold] = useState('');
  const [systemChangeDate, setSystemChangeDate] = useState('');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    let active = true;
    (async () => {
      try {
        const [s, c] = await Promise.all([Api.getCompanySettings(), Api.getClosingSettings()]);
        if (!active) return;
        setSettings(s);
        setClosing(c);
        setCaptureDays(String(s.receiptCaptureDays > 0 ? s.receiptCaptureDays : 10));
        setGraceDays(String(s.commitGraceDays > 0 ? s.commitGraceDays : 0));
        setCheckThreshold(s.invoiceCheckThreshold ? formatCentsPlain(s.invoiceCheckThreshold) : '');
        setThreshold(c?.accrualThreshold ? formatCentsPlain(c.accrualThreshold) : '0,00');
      } catch (e) {
        if (active) setError(message(e));
      }
      // Nebenauskünfte: fehlen sie, bleiben die Einstellungen bedienbar.
      Api.getComplianceHints()
        .then((h) => {
          if (!active) return;
          setHints(h);
          setSystemChangeDate(h?.systemChangeDate ?? '');
        })
        .catch(() => undefined);
      Api.getRetentionRules()
        .then((r) => active && setRetention(r))
        .catch(() => undefined);
      Api.getBankRules()
        .then((r) => active && setRules(r))
        .catch(() => undefined);
    })();
    return () => {
      active = false;
    };
  }, []);

  async function save() {
    if (!settings || !closing || saving || writeLock.locked) return;
    setSaving(true);
    setError('');
    try {
      const fresh = await Api.getCompanySettings();
      const checkCents = parseCents(checkThreshold);
      await Api.updateCompanySettings({
        ...fresh,
        receiptCaptureDays: wholeNumber(captureDays, 10, 1),
        commitGraceDays: wholeNumber(graceDays, 0, 0),
        invoiceCheckThreshold: checkCents && checkCents > 0 ? checkCents : fresh.invoiceCheckThreshold,
      });
      // Der Hebesatz steht in den Stammdaten; hier bleibt er, wie er gespeichert ist.
      const current = await Api.getClosingSettings();
      const thresholdCents = parseCents(threshold);
      setClosing(
        await Api.saveClosingSettings({
          ...current,
          accrualMethod: closing.accrualMethod,
          accrualRelease: closing.accrualRelease,
          accrualThreshold: thresholdCents !== null && thresholdCents >= 0 ? thresholdCents : current.accrualThreshold,
        }),
      );
      if (systemChangeDate !== (hints?.systemChangeDate ?? '')) {
        await Api.setSystemChangeDate(systemChangeDate);
        setHints((prev) => (prev ? { ...prev, systemChangeDate } : prev));
      }
      toast.success('Einstellungen gespeichert.');
    } catch (e) {
      setError(message(e));
    } finally {
      setSaving(false);
    }
  }

  async function deleteRule(id: number) {
    try {
      await Api.deleteBankRule(id);
      setRules(await Api.getBankRules());
      toast.success('Regel gelöscht. Sie entsteht neu, sobald ein solcher Umsatz wieder zugeordnet wird.');
    } catch (e) {
      toast.error(message(e));
    }
  }

  if (!settings || !closing) {
    return error ? <Notice tone="negative" text={error} /> : <SkeletonRows rows={6} />;
  }

  return (
    <div className="max-w-[900px]">
      {error && <Notice tone="negative" className="mb-5" text={error} />}

      <FormGrid>
        <Field
          label="Belege spätestens erfassen nach"
          hint="Tage nach Eingang"
          helpSummary="Danach meldet der Prüflauf einen abgelegten, ungebuchten Beleg."
          explain="Die GoBD nennen in Rz. 47 zehn Tage für die Erfassung unbarer Geschäftsvorfälle."
        >
          <Input type="number" min={1} align="right" value={captureDays} onChange={(e) => setCaptureDays(e.target.value)} />
        </Field>
        <Field
          label="Nachfrist Festschreibung"
          hint="Tage nach dem Folgemonat"
          helpSummary="Ab wann ein nicht festgeschriebener Monat als überfällig gilt."
          explain="0 heißt: Der Monat ist bis zum Ende des Folgemonats festzuschreiben."
        >
          <Input type="number" min={0} align="right" value={graceDays} onChange={(e) => setGraceDays(e.target.value)} />
        </Field>
        <Field
          label="Abgrenzungsmethode"
          helpSummary="Wie ein Abgrenzungsposten auf die Jahre verteilt wird."
          explain="Monatsgenau verteilt nach Zwölfteln, taggenau nach Kalendertagen. Die Wahl gilt für alle Posten, weil § 252 Abs. 1 Nr. 6 HGB Stetigkeit verlangt."
        >
          <Select
            items={ACCRUAL_METHODS}
            value={closing.accrualMethod}
            onValueChange={(next) => setClosing({ ...closing, accrualMethod: next as AccrualMethod })}
          />
        </Field>
        <Field
          label="Auflösung im Folgejahr"
          helpSummary="Wie oft der Saldenvortrag die Auflösung bucht."
          explain="Einmal je Jahr hält die Zahl der Buchungen klein; monatlich braucht, wer unterjährig auswertet."
        >
          <Select
            items={ACCRUAL_RELEASES}
            value={closing.accrualRelease}
            onValueChange={(next) => setClosing({ ...closing, accrualRelease: next as AccrualReleaseCycle })}
          />
        </Field>
        <Field
          label="Vorschlagsschwelle der Abgrenzung"
          hint="Darunter nur Anzeige"
          helpSummary="Ab diesem Betrag schlägt Buchfink eine Abgrenzung vor."
          explain="Handelsrechtlich ist jeder Posten abzugrenzen (§ 250 HGB). Die 800 Euro folgen dem steuerlichen Wahlrecht des § 6 Abs. 2 EStG; wer es nicht nutzt, trägt 0,00 ein."
        >
          <Input
            align="right"
            placeholder="0,00"
            value={threshold}
            onChange={(e) => setThreshold(e.target.value)}
            onBlur={() => {
              const cents = parseCents(threshold);
              if (cents !== null && cents >= 0) setThreshold(formatCentsPlain(cents));
            }}
          />
        </Field>
        <Field
          label="Leistungsnachweis ab"
          hint="Bruttobetrag des Eingangsbelegs"
          helpSummary="Ab diesem Betrag braucht ein Eingangsbeleg einen Prüfvermerk."
          explain="Der Vorsteuerabzug setzt eine tatsächlich bezogene Leistung voraus (§ 15 UStG). Der Vermerk am Beleg hält fest, wer das bestätigt hat."
        >
          <Input
            align="right"
            value={checkThreshold}
            onChange={(e) => setCheckThreshold(e.target.value)}
            onBlur={() => {
              const cents = parseCents(checkThreshold);
              if (cents !== null && cents > 0) setCheckThreshold(formatCentsPlain(cents));
            }}
          />
        </Field>
        <Field
          label="Umstellung aus einem Altsystem"
          optional
          hint={systemChangeDate ? undefined : 'nicht hinterlegt'}
          helpSummary="Wann Sie von Ihrem bisherigen Buchhaltungsprogramm umgestiegen sind."
          explain={
            hints?.systemChangeNote ||
            'Ab diesem Tag läuft die Frist, in der das Altsystem für den Datenzugriff verfügbar bleiben muss (§ 147 Abs. 6 AO).'
          }
        >
          <Input type="date" value={systemChangeDate} onChange={(e) => setSystemChangeDate(e.target.value)} />
        </Field>
      </FormGrid>

      <div className="mt-6">
        <Button
          variant="primary"
          loading={saving}
          disabled={writeLock.locked}
          title={writeLock.hint}
          icon={<Save className="w-4 h-4" strokeWidth={1.5} />}
          onClick={() => void save()}
        >
          Einstellungen speichern
        </Button>
      </div>

      <div className="mt-8 border-t border-line">
        <Disclosure
          variant="row"
          title="Gelernte Bankregeln"
          context={rules.length === 0 ? 'Noch keine Regel gelernt' : `${rules.length} Muster wiederkehrender Umsätze`}
        >
          {rules.length === 0 ? (
            <p className="text-body text-ink-muted">Eine Regel entsteht, sobald ein Umsatz ohne Beleg zugeordnet wird.</p>
          ) : (
            <Table density="kompakt">
              <Thead>
                <Tr>
                  <Th>Muster</Th>
                  <Th className="w-24">Richtung</Th>
                  <Th className="w-28">Gegenkonto</Th>
                  <Th numeric className="w-20">Treffer</Th>
                  <Th className="w-24" aria-label="Aktionen" />
                </Tr>
              </Thead>
              <Tbody>
                {rules.map((rule) => (
                  <Tr key={rule.id}>
                    <Td className="max-w-[24rem] truncate" title={rule.pattern}>
                      {rule.label || rule.pattern}
                    </Td>
                    <Td className="text-ink-muted">{rule.moneyIn ? 'Eingang' : 'Ausgang'}</Td>
                    <Td code>{rule.counterAccount}</Td>
                    <Td numeric className="text-ink-subtle">{rule.hits}</Td>
                    <Td className="pl-0 text-right">
                      <Button
                        variant="quiet"
                        size="sm"
                        disabled={writeLock.locked}
                        title={writeLock.hint ?? 'Die Regel entsteht neu, sobald ein solcher Umsatz bestätigt zugeordnet wird.'}
                        onClick={() => void deleteRule(rule.id)}
                      >
                        Löschen
                      </Button>
                    </Td>
                  </Tr>
                ))}
              </Tbody>
            </Table>
          )}
        </Disclosure>

        <Disclosure
          variant="row"
          title="Aufbewahrungsfristen"
          context={
            retention
              ? `Gesetzesstand ${retention.version} · gültig ab ${formatDate(retention.validFrom)}`
              : 'Gesetzesstand'
          }
        >
          {retention === null ? (
            <SkeletonRows rows={3} />
          ) : (
            <Table density="kompakt">
              <Thead>
                <Tr>
                  <Th>Unterlagen</Th>
                  <Th numeric className="w-20">Jahre</Th>
                  <Th>Grundlage</Th>
                </Tr>
              </Thead>
              <Tbody>
                {(retention.classes ?? []).map((entry) => (
                  <Tr key={entry.class}>
                    <Td>{entry.label || RETENTION_CLASS_LABELS[entry.class]}</Td>
                    <Td numeric>{entry.years}</Td>
                    <Td className="text-ink-muted">{entry.legalBasis}</Td>
                  </Tr>
                ))}
              </Tbody>
            </Table>
          )}
        </Disclosure>

        <Disclosure variant="row" title="Speicherort und Mandant" context="Datenordner, Programmfassung, Mandanten">
          {hints?.cloudWarning && <Notice tone="negative" className="mb-4" text={hints.cloudWarning} />}
          <FormGrid cols={1}>
            <Field label="Ordner für Buchungsdaten und Belege" hint="Steht beim Anlegen fest">
              <FieldValue className="code-num">{workspace.appConfig.dataDir || '—'}</FieldValue>
            </Field>
            <Field label="Programmfassung">
              <FieldValue className="code-num">{workspace.appConfig.programVersion || 'dev'}</FieldValue>
            </Field>
          </FormGrid>
          {onManageTenants && (
            <div className="mt-5 flex flex-wrap gap-2">
              <Button variant="secondary" size="sm" onClick={onManageTenants}>
                Mandanten verwalten
              </Button>
            </div>
          )}
        </Disclosure>
      </div>

    </div>
  );
};
