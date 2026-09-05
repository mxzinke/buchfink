import React, { useEffect, useState } from 'react';
import { Save, Shield } from 'lucide-react';
import {
  AccrualMethod,
  AccrualReleaseCycle,
  ClosingSettings,
  CompanySettings,
  AppConfig,
  InvestorType,
  LegalFormInfo,
  SpecialPrepaymentSuggestion,
} from '../types';
import { Api } from '../services/api';
import { useWriteLock } from '../components/WriteLock';
import { formatCents, formatCentsPlain, parseCents } from '../utils/formatters';
import {
  Button,
  Checkbox,
  Field,
  FieldValue,
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

/**
 * Die Verteilung eines Abgrenzungspostens. Beide Verfahren sind zulässig; die
 * Wahl gilt für den ganzen Mandanten, weil § 252 Abs. 1 Nr. 6 HGB die Stetigkeit
 * der Bewertungsmethoden verlangt.
 */
const ACCRUAL_METHODS = [
  { value: 'monthly', label: 'Monatsgenau · nach Zwölfteln' },
  { value: 'daily', label: 'Taggenau · nach Kalendertagen' },
];

/** Wie oft die Auflösung im Folgejahr gebucht wird. */
const ACCRUAL_RELEASES = [
  { value: 'yearly', label: 'Einmal je Geschäftsjahr · am ersten Tag' },
  { value: 'monthly', label: 'Monatlich · für unterjährige Auswertungen' },
];

/** Die Voreinstellungen des Dienstes, bis er geantwortet hat. */
const DEFAULT_CLOSING_SETTINGS: ClosingSettings = {
  tradeTaxRatePercent: 400,
  accrualMethod: 'monthly',
  accrualThreshold: 80000,
  accrualRelease: 'yearly',
};

/** Wo der Schlüssel im jeweiligen Betriebssystem zu finden ist. */
function keychainHint(): string {
  if (navigator.platform.startsWith('Mac')) return 'In der Schlüsselbundverwaltung unter diesem Eintrag.';
  if (navigator.platform.startsWith('Win'))
    return 'In der Anmeldeinformationsverwaltung unter Windows-Anmeldeinformationen.';
  return 'Im Secret Service, etwa dem GNOME-Schlüsselbund, unter diesem Dienst und Konto.';
}

export const SettingsPage: React.FC = () => {
  // Die Stammdaten stehen in jeder Buchung und jeder Meldung: sie zu ändern ist
  // im Prüfermodus gesperrt. Der Schlüsselexport bleibt möglich (§10.4).
  const writeLock = useWriteLock();
  const [settings, setSettings] = useState<CompanySettings | null>(null);
  const [appConfig, setAppConfig] = useState<AppConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [exporting, setExporting] = useState(false);
  const [legalForms, setLegalForms] = useState<LegalFormInfo[]>([]);
  const [showInvestorChoice, setShowInvestorChoice] = useState(false);
  // Die Sondervorauszahlung wird als Text erfasst und erst beim Verlassen des
  // Feldes in Cent umgerechnet (§8.3).
  const [prepaymentText, setPrepaymentText] = useState('');
  // Die Erfassungsfrist ebenso: ein leeres Zahlenfeld ist kein Wert, und die 0
  // wäre hier keine Antwort, sondern eine Lücke.
  const [captureDaysText, setCaptureDaysText] = useState('10');
  // Die Nachfrist zur Festschreibung ebenso; hier ist die 0 allerdings eine
  // Antwort: keine Nachfrist über das Ende des Folgemonats hinaus.
  const [graceDaysText, setGraceDaysText] = useState('0');
  const [suggestion, setSuggestion] = useState<SpecialPrepaymentSuggestion | null>(null);
  // Die Einstellungen der Abschlussbausteine stehen in eigenen Schlüsseln und
  // werden über einen eigenen Dienst gespeichert; sie hängen deshalb neben den
  // Stammdaten und nicht in ihnen.
  const [closing, setClosing] = useState<ClosingSettings>(DEFAULT_CLOSING_SETTINGS);
  // Hebesatz und Schwelle werden als Text geführt und erst beim Verlassen des
  // Feldes umgerechnet (§8.3): eine gelöschte Ziffer ist keine 0.
  const [tradeTaxText, setTradeTaxText] = useState('400');
  const [thresholdText, setThresholdText] = useState('');

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
      setPrepaymentText(s.specialPrepayment ? formatCentsPlain(s.specialPrepayment) : '');
      setCaptureDaysText(String(s.receiptCaptureDays > 0 ? s.receiptCaptureDays : 10));
      setGraceDaysText(String(s.commitGraceDays > 0 ? s.commitGraceDays : 0));
      try {
        // Der Vorschlag ist eine Nebenauskunft: fehlt er, bleiben die
        // Einstellungen benutzbar.
        setSuggestion(await Api.getSpecialPrepaymentSuggestion(s.fiscalYear || 0));
      } catch {
        setSuggestion(null);
      }
      try {
        // Ohne aktiven Mandanten antwortet der Dienst nicht; die übrigen
        // Einstellungen bleiben dann trotzdem bedienbar.
        const values = await Api.getClosingSettings();
        if (values) applyClosing(values);
      } catch {
        applyClosing(DEFAULT_CLOSING_SETTINGS);
      }
      // Wer die Anlegerstellung schon einmal abweichend festgelegt hat, soll
      // sie auch wiederfinden.
      if (s.investorOverride) setShowInvestorChoice(true);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }

  /** Übernimmt die Antwort des Dienstes in Zustand und Textfelder. */
  function applyClosing(values: ClosingSettings) {
    setClosing(values);
    setTradeTaxText(String(values.tradeTaxRatePercent));
    setThresholdText(values.accrualThreshold ? formatCentsPlain(values.accrualThreshold) : '0,00');
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
    // Auch die Eingabetaste in einem Feld löst das Formular aus; der gesperrte
    // Knopf allein hielte den Prüfermodus deshalb nicht.
    if (!settings || saving || writeLock.locked) return;
    setSaving(true);
    try {
      await Api.updateCompanySettings(settings);
      // Der eine Knopf speichert beides. Die Abschluss-Einstellungen liegen in
      // eigenen Schlüsseln, aber es wäre eine Zumutung, dafür einen zweiten
      // „Speichern" zu suchen; der Dienst prüft die Grenzen und meldet sich mit
      // seinem eigenen Satz, wenn ihm ein Wert nicht passt.
      applyClosing(await Api.saveClosingSettings(closing));
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
  // Ohne Steuernummer und ohne USt-IdNr. lässt sich keine Rechnung ausstellen
  // (§ 14 Abs. 4 Nr. 2 UStG). Der Hinweis steht am Feld und nicht erst in der
  // Fehlermeldung des Rechnungsdialogs.
  const identifierMissing = !settings.taxNumber && !settings.vatId;
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
            disabled={writeLock.locked}
            title={writeLock.hint}
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
          <Field
            label="Steuernummer"
            hint={identifierMissing ? 'für Rechnungen nötig' : undefined}
            explain="§ 14 Abs. 4 Nr. 2 UStG verlangt auf jeder Rechnung die Steuernummer oder die USt-IdNr. des Ausstellers. Buchfink schreibt die USt-IdNr., wenn sie vorliegt (BT-31), sonst die Steuernummer (BT-32); ohne beide wird keine Rechnung ausgestellt."
          >
            <Input
              className="code-num"
              value={settings.taxNumber}
              onChange={(e) => patch({ taxNumber: e.target.value })}
            />
          </Field>
          <Field
            label="Umsatzsteuer-Identifikationsnummer"
            hint={identifierMissing ? 'für Rechnungen nötig' : undefined}
          >
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

      <Section
        title="Rechnungsstellung"
        action={
          <HelpPopover label="Erklärung zur Rechnungsstellung">
            Die Systematik des Nummernkreises gehört in die Verfahrensdokumentation und steht
            deshalb als Einstellung: Ein Mandant mit vorhandener Buchhaltung führt seine Systematik
            fort. Ansprechpartner, Telefon und E-Mail sind bei einer XRechnung Pflichtangaben
            (BR-DE-2 bis BR-DE-7) — eine Behörde, die nicht zurückfragen kann, weist die Rechnung
            zurück.
          </HelpPopover>
        }
      >
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Field
            label="Nummernformat"
            hint="{JAHR} und {NR:4}"
            explain="Zwei Platzhalter: {JAHR} für das Geschäftsjahr, {NR:4} für den Zähler mit vier Stellen. Ohne {NR} trüge jede Rechnung dieselbe Nummer; ein solches Format weist Buchfink zurück (§ 14 Abs. 4 Nr. 4 UStG). Leer heißt RE-{JAHR}-{NR:4}."
          >
            <Input
              className="code-num"
              placeholder="RE-{JAHR}-{NR:4}"
              value={settings.invoiceNumberFormat}
              onChange={(e) => patch({ invoiceNumberFormat: e.target.value })}
            />
          </Field>
          <Field label="Ansprechpartner" hint="bei XRechnung Pflicht">
            <Input
              value={settings.contactName}
              onChange={(e) => patch({ contactName: e.target.value })}
            />
          </Field>
          <Field label="Telefon" hint="bei XRechnung Pflicht">
            <Input
              value={settings.contactPhone}
              onChange={(e) => patch({ contactPhone: e.target.value })}
            />
          </Field>
          <Field label="E-Mail für Rückfragen" hint="bei XRechnung Pflicht">
            <Input
              type="email"
              value={settings.contactEmail}
              onChange={(e) => patch({ contactEmail: e.target.value })}
            />
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

        <Checkbox
          className="mt-4"
          checked={settings.permanentExtension}
          onCheckedChange={(next) => patch({ permanentExtension: Boolean(next) })}
          label={
            <span className="flex items-center">
              Dauerfristverlängerung
              <HelpPopover label="Erklärung zur Dauerfristverlängerung">
                Mit der Dauerfristverlängerung wird jede Voranmeldung einen Monat später fällig
                (§§ 46 bis 48 UStDV). Wer monatlich anmeldet, hat dafür bis zum 10. Februar eine
                Sondervorauszahlung von einem Elftel der Vorauszahlungen des Vorjahres anzumelden
                und zu zahlen; angerechnet wird sie in der letzten Voranmeldung des Jahres.
              </HelpPopover>
            </span>
          }
          hint="verschiebt jede Fälligkeit um einen Monat"
        />

        {settings.permanentExtension && (
          <Field
            label="Angemeldete Sondervorauszahlung"
            className="mt-4 max-w-sm"
            hint={
              suggestion && suggestion.amount > 0
                ? `Vorschlag aus ${suggestion.basedOnYear}: ${formatCents(suggestion.amount)}`
                : 'ein Elftel der Vorauszahlungen des Vorjahres'
            }
            explain={suggestion?.note}
          >
            <div className="flex gap-2">
              <Input
                align="right"
                value={prepaymentText}
                onChange={(e) => setPrepaymentText(e.target.value)}
                onBlur={() => {
                  const cents = parseCents(prepaymentText);
                  patch({ specialPrepayment: cents ?? 0 });
                  setPrepaymentText(cents ? formatCentsPlain(cents) : '');
                }}
                placeholder="0,00"
              />
              {suggestion && suggestion.amount > 0 && (
                <Button
                  variant="secondary"
                  className="shrink-0"
                  onClick={() => {
                    patch({ specialPrepayment: suggestion.amount });
                    setPrepaymentText(formatCentsPlain(suggestion.amount));
                  }}
                >
                  Vorschlag übernehmen
                </Button>
              )}
            </div>
          </Field>
        )}
      </Section>

      <Section
        title="Prüfläufe"
        action={
          <HelpPopover label="Erklärung zu den Schwellenwerten">
            Der Prüflauf vor der Festschreibung meldet abgelegte, aber nicht gebuchte Belege. Die
            GoBD nennt in Rz. 47 zehn Tage für die Erfassung unbarer Geschäftsvorfälle; wer anders
            arbeitet, setzt hier seinen eigenen Wert. Die Nachfrist zur Festschreibung entscheidet
            daneben, ab wann ein nicht festgeschriebener Monat als überfällig gilt — in der
            Fristenliste wie im Prüfbericht.
          </HelpPopover>
        }
      >
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Field label="Belege spätestens erfassen nach" hint="Tage nach Eingang · GoBD Rz. 47">
            {/* Der Wert wird als Text geführt und erst beim Verlassen des Feldes
                normalisiert (§8.3). Vorher zeigte das Feld die Voreinstellung
                an, während im Zustand die 0 stand — angezeigter und
                gespeicherter Wert stimmten nur zufällig überein. */}
            <Input
              type="number"
              min={1}
              align="right"
              value={captureDaysText}
              onChange={(e) => setCaptureDaysText(e.target.value)}
              onBlur={() => {
                const days = Number(captureDaysText);
                const value = Number.isFinite(days) && days > 0 ? Math.trunc(days) : 10;
                patch({ receiptCaptureDays: value });
                setCaptureDaysText(String(value));
              }}
            />
          </Field>
          <Field label="Nachfrist Festschreibung" hint="Tage nach dem Folgemonat">
            {/* Wie die Erfassungsfrist daneben: als Text geführt und erst beim
                Verlassen des Feldes normalisiert (§8.3). Bei jedem Tastendruck
                zu speichern machte aus einer gelöschten Ziffer eine 0 — ein
                Wert, den niemand eingegeben hat. */}
            <Input
              type="number"
              min={0}
              align="right"
              value={graceDaysText}
              onChange={(e) => setGraceDaysText(e.target.value)}
              onBlur={() => {
                const days = Number(graceDaysText);
                const value = Number.isFinite(days) && days > 0 ? Math.trunc(days) : 0;
                patch({ commitGraceDays: value });
                setGraceDaysText(String(value));
              }}
            />
          </Field>
        </div>
      </Section>



      {/*
        Die Abschlussbausteine rechnen mit diesen drei Angaben: ohne sie liefe
        jede Installation mit 400 % Hebesatz, monatsgenauer Abgrenzung und 800
        Euro Schwelle, als wäre das gewählt worden.
      */}
      <Section
        title="Jahresabschluss"
        context="Steuert die Abschlussbausteine"
        action={
          <HelpPopover label="Erklärung zu den Abschluss-Einstellungen">
            Der Hebesatz geht in die Steuerrückstellung ein, die Abgrenzungsmethode in jeden
            Abgrenzungsposten und die Vorschlagsschwelle allein in die Vorschlagsliste. Alle drei
            gelten für den ganzen Mandanten und über alle Geschäftsjahre; die Änderung wird
            protokolliert.
          </HelpPopover>
        }
      >
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Field
            label="Gewerbesteuer-Hebesatz"
            hint="Prozent · § 16 GewStG"
            help="Der Hebesatz der Gemeinde, in der die Betriebsstätte liegt. Er steht im Gewerbesteuermessbescheid und auf der Website der Gemeinde; mindestens 200 % (§ 16 Abs. 4 Satz 2 GewStG)."
          >
            <Input
              type="number"
              min={200}
              max={1000}
              align="right"
              value={tradeTaxText}
              onChange={(e) => setTradeTaxText(e.target.value)}
              onBlur={() => {
                const percent = Number(tradeTaxText);
                const value =
                  Number.isFinite(percent) && percent > 0 ? Math.trunc(percent) : closing.tradeTaxRatePercent;
                setClosing({ ...closing, tradeTaxRatePercent: value });
                setTradeTaxText(String(value));
              }}
            />
          </Field>
          <Field
            label="Abgrenzungsmethode"
            help="Monatsgenau verteilt nach Zwölfteln, taggenau nach Kalendertagen. Beides ist zulässig; § 252 Abs. 1 Nr. 6 HGB verlangt nur, dass es dabei bleibt — die Wahl gilt deshalb für alle Posten."
          >
            <Select
              items={ACCRUAL_METHODS}
              value={closing.accrualMethod}
              onValueChange={(next) =>
                setClosing({ ...closing, accrualMethod: next as AccrualMethod })
              }
            />
          </Field>
          <Field
            label="Vorschlagsschwelle der Abgrenzung"
            hint="unterhalb nur Anzeige"
            help="Nur für die Vorschlagsliste: Handelsrechtlich gibt es keine Grenze, jeder Posten ist abzugrenzen (§ 250 HGB). Die 800 Euro sind das steuerliche Wahlrecht des § 6 Abs. 2 EStG, das die Finanzverwaltung auch für die Abgrenzung zulässt — wer es nicht nutzen will, trägt hier 0,00 ein."
          >
            <Input
              align="right"
              placeholder="0,00"
              value={thresholdText}
              onBlur={() => {
                const cents = parseCents(thresholdText);
                const value = cents !== null && cents >= 0 ? cents : closing.accrualThreshold;
                setClosing({ ...closing, accrualThreshold: value });
                setThresholdText(formatCentsPlain(value));
              }}
              onChange={(e) => setThresholdText(e.target.value)}
            />
          </Field>
          <Field
            label="Auflösung im Folgejahr"
            help="Der Saldenvortrag bucht die Auflösung mit. Einmal je Jahr hält die Zahl der Abschlussbuchungen klein; monatlich braucht, wer unterjährig auswertet — sonst trägt der Januar den gesamten Vorjahresaufwand."
          >
            <Select
              items={ACCRUAL_RELEASES}
              value={closing.accrualRelease}
              onValueChange={(next) =>
                setClosing({ ...closing, accrualRelease: next as AccrualReleaseCycle })
              }
            />
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
            <FieldValue>SKR04 · Bilanz und GuV</FieldValue>
          </Field>
        </div>
      </Section>

      <Section title="Speicherort und Schlüssel" context="Gilt für diesen Mandanten">
        <div className="flex flex-col gap-4 max-w-2xl">
          {/* Kein Knopf zum Wählen: Buchfink zieht einen Datenordner nicht um,
              und ein Knopf, der nur die Anzeige ändert, verspräche das (ARC-06). */}
          <Field
            label="Ordner für Buchungsdaten und Belege"
            help="Der Ordner steht beim Anlegen des Mandanten fest und lässt sich hier nicht umziehen."
          >
            <Input className="code-num" value={appConfig?.dataDir || ''} readOnly />
          </Field>

          <Field
            label="Sicherungsordner"
            hint="Einzurichten unter Datenzugriff"
            help="Ohne Sicherungsordner schreibt Buchfink keine Sicherung — weder von Hand noch beim Beenden."
          >
            <Input
              className="code-num"
              value={appConfig?.backupDir || ''}
              readOnly
              placeholder="Nicht eingerichtet"
            />
          </Field>

          <Field label="Programmversion" help="Sie steht in jedem Export und in jeder Sicherung.">
            <Input className="code-num" value={appConfig?.programVersion || 'dev'} readOnly />
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
