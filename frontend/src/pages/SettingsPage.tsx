import React, { useEffect, useState } from 'react';
import { Save, Shield } from 'lucide-react';
import { CompanySettings, AppConfig, InvestorType, LegalFormInfo } from '../types';
import { Api } from '../services/api';
import {
  Button,
  Field,
  HelpPopover,
  Input,
  PageHeader,
  RadioGroup,
  Section,
  Select,
  SkeletonRows,
  toast,
} from '../components/ui';

/**
 * Einstellungen.
 *
 * Stammdaten des Mandanten: Sie gelten über alle Geschäftsjahre hinweg. Was
 * Buchfink bewusst nicht kann — Istversteuerung, Kleinunternehmerregelung —
 * steht hier als Erklärung am jeweiligen Feld, nicht als Absatz auf der Seite.
 */

/**
 * Die Anlegerstellung nach § 20 InvStG. „Nicht festgelegt" ist die
 * Voreinstellung und bleibt eine gültige Antwort: geraten wird hier nichts.
 */
/** Was eine abgeleitete Anlegerstellung in einem Halbsatz bedeutet. */
function investorHint(investor: InvestorType): string {
  switch (investor) {
    case 'corporate':
      return 'Investmentanteile: 80 % Teilfreistellung';
    case 'individual_business':
      return 'Investmentanteile: 60 % Teilfreistellung';
    case 'basic':
      return 'Investmentanteile: 30 % Teilfreistellung';
    default:
      return 'Investmentanteile: Teilfreistellung noch offen';
  }
}

/**
 * Die Anlegerstellung nach § 20 InvStG, als Ausnahme von der Ableitung.
 * „Aus der Rechtsform" ist die Voreinstellung und bleibt eine gültige Antwort.
 */
/**
 * Der Platzhalter für „keine Festlegung". Ein leerer Wert wäre für das
 * Auswahlfeld kein Wert und deshalb nicht wieder wählbar — gespeichert wird
 * trotzdem leer, denn leer heißt: aus der Rechtsform.
 */
const DERIVE = 'derive';

const INVESTOR_TYPES = [
  { value: 'corporate', label: 'Körperschaft — 80 %' },
  { value: 'individual_business', label: 'Natürliche Person, Anteile im Betriebsvermögen — 60 %' },
  { value: 'basic', label: 'Grundsatz — 30 % (auch für Versicherer, Handelsbestand, Pensionsfonds)' },
  { value: 'mixed', label: 'Gesellschafter unterschiedlich besteuert — kein einheitlicher Satz' },
];

const MONTHS = [
  { value: 1, label: 'Januar · Kalenderjahr' },
  { value: 2, label: 'Februar · 1. Feb bis 31. Jan' },
  { value: 3, label: 'März · 1. Mär bis 28./29. Feb' },
  { value: 4, label: 'April · 1. Apr bis 31. Mär' },
  { value: 5, label: 'Mai · 1. Mai bis 30. Apr' },
  { value: 6, label: 'Juni · 1. Jun bis 31. Mai' },
  { value: 7, label: 'Juli · 1. Jul bis 30. Jun' },
  { value: 8, label: 'August · 1. Aug bis 31. Jul' },
  { value: 9, label: 'September · 1. Sep bis 31. Aug' },
  { value: 10, label: 'Oktober · 1. Okt bis 30. Sep' },
  { value: 11, label: 'November · 1. Nov bis 31. Okt' },
  { value: 12, label: 'Dezember · 1. Dez bis 30. Nov' },
];

const VAT_PERIODS = [
  { value: 'quarter', label: 'Vierteljährlich' },
  { value: 'month', label: 'Monatlich' },
  { value: 'year', label: 'Jährlich, nur die Jahreserklärung' },
];

/** Wo der Schlüssel im jeweiligen Betriebssystem zu finden ist. */
function keychainHint(): string {
  if (navigator.platform.startsWith('Mac')) return 'In der Schlüsselbundverwaltung unter diesem Eintrag.';
  if (navigator.platform.startsWith('Win'))
    return 'In der Anmeldeinformationsverwaltung unter Windows-Anmeldeinformationen.';
  return 'Im Secret Service, etwa dem GNOME-Schlüsselbund, unter diesem Dienst und Konto.';
}

export const SettingsPage: React.FC = () => {
  const [settings, setSettings] = useState<CompanySettings | null>(null);
  const [appConfig, setAppConfig] = useState<AppConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [exporting, setExporting] = useState(false);
  const [legalForms, setLegalForms] = useState<LegalFormInfo[]>([]);
  const [showInvestorChoice, setShowInvestorChoice] = useState(false);

  useEffect(() => {
    void loadSettings();
  }, []);

  async function loadSettings() {
    setLoading(true);
    try {
      const [s, cfg, forms] = await Promise.all([
        Api.getCompanySettings(),
        Api.getAppConfig(),
        Api.getLegalForms(),
      ]);
      setSettings(s);
      setAppConfig(cfg);
      setLegalForms(forms ?? []);
      // Wer die Anlegerstellung schon einmal abweichend festgelegt hat, soll
      // sie auch wiederfinden.
      if (s.investorOverride) setShowInvestorChoice(true);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }

  async function pickDirectory() {
    try {
      const selected = await Api.selectDirectoryDialog('Buchfink Datenordner ändern');
      if (selected && appConfig) setAppConfig({ ...appConfig, dataDir: selected });
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    }
  }

  async function exportRecovery() {
    setExporting(true);
    try {
      const path = await Api.exportRecoveryKey();
      if (path) toast.success(`Recovery-Schlüssel gespeichert: ${path}`);
    } catch (e) {
      const message = e instanceof Error ? e.message : String(e);
      // Der abgebrochene Ordnerdialog ist kein Fehler, den jemand lesen muss.
      if (!message.includes('kein Zielordner')) toast.error(message);
    } finally {
      setExporting(false);
    }
  }

  async function save(e: React.FormEvent) {
    e.preventDefault();
    if (!settings || saving) return;
    setSaving(true);
    try {
      await Api.updateCompanySettings(settings);
      toast.success('Einstellungen gespeichert.');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
    }
  }

  if (loading || !settings) {
    return (
      <div className="max-w-[900px] mx-auto px-8 py-8">
        <SkeletonRows rows={8} />
      </div>
    );
  }

  // Eine gespeicherte Rechtsform, die nicht im Katalog steht, bleibt wählbar:
  // sonst überschriebe das Öffnen der Einstellungen still, was dort stand.
  const known = legalForms.some((form) => form.name === settings.legalForm);
  const legalFormItems = [
    ...(settings.legalForm && !known ? [{ value: settings.legalForm, label: settings.legalForm }] : []),
    ...legalForms.map((form) => ({ value: form.name, label: form.name })),
  ];
  const selectedForm = legalForms.find((form) => form.name === settings.legalForm);
  // Was die Rechtsform für Investmentanteile bedeutet — als Hinweis am Feld,
  // nicht als zweite Frage.
  const derivedInvestor = settings.investorOverride
    ? {
        label: 'Anlegerstellung abweichend festgelegt',
        note: 'Die Rechtsform entscheidet hier nicht; unten steht, was stattdessen gilt.',
      }
    : selectedForm
      ? { label: investorHint(selectedForm.investor), note: selectedForm.note }
      : undefined;
  // Gefragt wird nur, wo die Rechtsform die Anlegerstellung offen lässt.
  const needsInvestorChoice = Boolean(selectedForm) && !selectedForm?.investor;

  const patch = (next: Partial<CompanySettings>) => setSettings({ ...settings, ...next });
  const startMonth = settings.fiscalYearStartMonth || 1;
  const deviating = startMonth !== 1;

  return (
    <form onSubmit={save} className="max-w-[900px] mx-auto px-8 py-8">
      <PageHeader
        title="Einstellungen"
        context="Stammdaten des Mandanten · gelten über alle Geschäftsjahre"
        action={
          <Button
            type="submit"
            variant="primary"
            loading={saving}
            icon={<Save className="w-4 h-4" strokeWidth={1.5} />}
          >
            Speichern
          </Button>
        }
      />

      <Section title="Unternehmen" divider={false} className="mt-8">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Field label="Firmen- oder Inhabername">
            <Input value={settings.companyName} onChange={(e) => patch({ companyName: e.target.value })} />
          </Field>
          <Field
            label="Rechtsform"
            hint={derivedInvestor?.label}
            help={derivedInvestor?.note}
          >
            <Select
              items={legalFormItems}
              value={settings.legalForm || ''}
              onValueChange={(next) => patch({ legalForm: String(next) })}
            />
          </Field>
          <Field label="Steuernummer">
            <Input
              className="code-num"
              value={settings.taxNumber}
              onChange={(e) => patch({ taxNumber: e.target.value })}
            />
          </Field>
          <Field label="Umsatzsteuer-Identifikationsnummer">
            <Input
              className="code-num"
              value={settings.vatId}
              onChange={(e) => patch({ vatId: e.target.value })}
            />
          </Field>
          <Field label="Zuständiges Finanzamt" className="md:col-span-2">
            <Input value={settings.taxOffice} onChange={(e) => patch({ taxOffice: e.target.value })} />
          </Field>
        </div>

        {/*
          Die Anlegerstellung für § 20 InvStG folgt aus der Rechtsform. Sichtbar
          wird sie nur, wo diese sie nicht hergibt — bei einer
          Personengesellschaft — oder wo jemand sie ausdrücklich anders
          festlegen will. Als eigenes Pflichtfeld stünde hier sonst eine
          Rechtsfrage, die die meisten nie beantworten müssten.
        */}
        {(needsInvestorChoice || showInvestorChoice) && (
          <Field
            label="Anlegerstellung für Investmentanteile"
            optional={!needsInvestorChoice}
            hint="nur für die Teilfreistellung nach § 20 InvStG"
            help="Der Satz hängt am Anleger. Bei einer Personengesellschaft bestimmt ihn der einzelne Gesellschafter (§ 20 Abs. 3a InvStG). Und auch eine Körperschaft trägt nicht immer 80 %: für Lebens- und Krankenversicherer, für Kreditinstitute mit Handelsbestand und für Pensionsfonds nehmen § 20 Abs. 1 Sätze 4 und 5 die Erhöhung zurück."
            className="mt-4 max-w-2xl"
          >
            <Select
              items={[
                {
                  value: DERIVE,
                  label: needsInvestorChoice ? 'Noch nicht festgelegt' : 'Aus der Rechtsform',
                },
                ...INVESTOR_TYPES,
              ]}
              value={settings.investorOverride || DERIVE}
              onValueChange={(next) =>
                patch({
                  investorOverride: (next === DERIVE
                    ? ''
                    : next) as CompanySettings['investorOverride'],
                })
              }
            />
          </Field>
        )}
        {!needsInvestorChoice && !showInvestorChoice && (
          <Button
            variant="quiet"
            size="sm"
            className="mt-4 -ml-3"
            onClick={() => setShowInvestorChoice(true)}
          >
            Anlegerstellung für Investmentanteile abweichend festlegen
          </Button>
        )}
      </Section>

      <Section title="Anschrift">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Field label="Straße und Hausnummer" className="md:col-span-2">
            <Input value={settings.street} onChange={(e) => patch({ street: e.target.value })} />
          </Field>
          <Field label="PLZ und Ort">
            <Input value={settings.zipCity} onChange={(e) => patch({ zipCity: e.target.value })} />
          </Field>
          <Field label="Land">
            <Input value={settings.country} onChange={(e) => patch({ country: e.target.value })} />
          </Field>
        </div>
      </Section>

      {/* Ohne diese drei Angaben trägt der Jahresabschluss den Kopf nicht, den
          § 264 Abs. 1a HGB verlangt. Sie standen bisher nur im Gründungsweg. */}
      <Section
        title="Registereintragung"
        action={
          <HelpPopover label="Erklärung zur Registereintragung">
            Auf jedem Jahresabschluss einer Kapitalgesellschaft sind Firma, Sitz, Registergericht
            und Registernummer anzugeben (§ 264 Abs. 1a HGB). Buchfink setzt sie in den Kopf von
            Bilanz, Gewinn- und Verlustrechnung und E-Bilanz.
          </HelpPopover>
        }
      >
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Field label="Sitz der Gesellschaft">
            <Input
              value={settings.seat}
              onChange={(e) => patch({ seat: e.target.value })}
              placeholder="Ort laut Satzung"
            />
          </Field>
          <Field label="Registergericht">
            <Input
              value={settings.registerCourt}
              onChange={(e) => patch({ registerCourt: e.target.value })}
              placeholder="Amtsgericht München"
            />
          </Field>
          <Field label="Registernummer">
            <Input
              value={settings.registerNumber}
              onChange={(e) => patch({ registerNumber: e.target.value })}
              placeholder="HRB 123456"
            />
          </Field>
        </div>
      </Section>

      <Section
        title="Geschäftsjahr"
        action={
          <HelpPopover label="Erklärung zur Jahreszuordnung">
            Belege, Rechnungen und Zahlungen tragen ein Datum, daraus ergibt sich das Geschäftsjahr
            von selbst. Im Grenzbereich zum Jahreswechsel lässt sich die Zuordnung einer Buchung im
            Journal übersteuern.
          </HelpPopover>
        }
      >
        <RadioGroup
          options={[
            {
              value: 'calendar',
              label: 'Kalenderjahr',
              hint: '1. Januar bis 31. Dezember',
            },
            {
              value: 'deviating',
              label: 'Abweichendes Wirtschaftsjahr',
              hint: 'beginnt in einem anderen Monat',
            },
          ]}
          value={deviating ? 'deviating' : 'calendar'}
          onValueChange={(next) => patch({ fiscalYearStartMonth: next === 'deviating' ? 7 : 1 })}
          inline
        />

        {deviating && (
          <Field label="Beginn des Wirtschaftsjahres" className="mt-4 max-w-sm">
            <Select
              items={MONTHS}
              value={startMonth}
              onValueChange={(month) => patch({ fiscalYearStartMonth: month })}
            />
          </Field>
        )}
      </Section>

      <Section title="Umsatzsteuer">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Field
            label="Voranmeldezeitraum"
            help="Monatlich gilt bei Neugründung und hoher Zahllast, vierteljährlich ist der Regelfall."
          >
            <Select
              items={VAT_PERIODS}
              value={settings.vatPeriod || 'quarter'}
              onValueChange={(period) => patch({ vatPeriod: period as CompanySettings['vatPeriod'] })}
            />
          </Field>
          <Field
            label="Besteuerungsart"
            hint="nach vereinbarten Entgelten"
            help="Buchfink rechnet nach § 16 Abs. 1 Satz 1 UStG. Bei Istversteuerung entstünde die Steuer erst mit der Vereinnahmung, die Buchungen sähen anders aus — der Buchungskern weist sie deshalb ab, statt sie stillschweigend falsch zu behandeln."
          >
            <Select items={[{ value: 'SOLL', label: 'Sollversteuerung' }]} value="SOLL" disabled />
          </Field>
        </div>
      </Section>



      <Section title="Bankverbindung">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Field label="Bankname">
            <Input value={settings.bankName} onChange={(e) => patch({ bankName: e.target.value })} />
          </Field>
          <Field label="IBAN">
            <Input
              className="code-num"
              value={settings.iban}
              onChange={(e) => patch({ iban: e.target.value })}
            />
          </Field>
          <Field label="BIC">
            <Input
              className="code-num"
              value={settings.bic}
              onChange={(e) => patch({ bic: e.target.value })}
            />
          </Field>
          <Field
            label="Kontenrahmen"
            help="Buchfink richtet sich an bilanzierende Gesellschaften und bucht im SKR04. Die Kleinunternehmerregelung nach § 19 UStG wird nicht unterstützt; ein Kleinunternehmer als Lieferant ist dagegen ein normaler Fall und wird am Kontakt hinterlegt."
          >
            <Input value="SKR04 · Bilanz und GuV" disabled readOnly />
          </Field>
        </div>
      </Section>

      <Section title="Speicherort und Schlüssel" context="Gilt für diesen Mandanten">
        <div className="flex flex-col gap-4 max-w-2xl">
          <Field label="Ordner für Buchungsdaten und Belege">
            <div className="flex gap-2">
              <Input className="code-num" value={appConfig?.dataDir || ''} readOnly />
              <Button variant="secondary" onClick={() => void pickDirectory()} className="shrink-0">
                Ordner wählen
              </Button>
            </div>
          </Field>

          <Field label="Schlüssel im Schlüsselbund des Betriebssystems" help={keychainHint()}>
            <p className="flex items-center gap-2 text-body text-ink-muted">
              <span className="mark-diamond bg-positive" aria-hidden="true" />
              Dienst <span className="code-num text-ink">org.buchfink.app</span> · Konto{' '}
              <span className="code-num text-ink">{appConfig?.activeTenantId || '—'}</span>
            </p>
          </Field>

          {/* Hinweisfläche nach §6.2, Fall 4: Der Verlust ist endgültig, das
              darf einmal laut werden. */}
          <div className="rounded-control border border-attention-line bg-attention-soft px-4 py-3">
            <h3 className="flex items-center text-label text-attention-text">
              Recovery-Schlüssel getrennt sichern
              <HelpPopover label="Erklärung zum Recovery-Schlüssel">
                Geht dieser Rechner verloren, ist der Schlüsselbund weg und die verschlüsselten
                Daten sind ohne Recovery-Datei unwiederbringlich. Die Datei gehört an einen anderen
                Ort als das Datenbackup — ein Backup, das beide enthält, schützt vor nichts.
              </HelpPopover>
            </h3>
            <p className="text-body text-ink-muted mt-1">
              Ohne diese Datei sind die Daten verloren, wenn der Rechner abhandenkommt.
            </p>
            <Button
              variant="secondary"
              size="sm"
              loading={exporting}
              onClick={() => void exportRecovery()}
              icon={<Shield className="w-3.5 h-3.5" strokeWidth={1.5} />}
              className="mt-3"
            >
              Recovery-Schlüssel exportieren
            </Button>
          </div>
        </div>
      </Section>
    </form>
  );
};
