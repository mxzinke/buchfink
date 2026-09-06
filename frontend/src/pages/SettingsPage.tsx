import React, { useEffect, useState } from 'react';
import { Save, Shield } from 'lucide-react';
import {
  AccrualMethod,
  AccrualReleaseCycle,
  BankRule,
  BaseRate,
  ClosingSettings,
  CompanySettings,
  ComplianceHints,
  AppConfig,
  DunningLevel,
  InvestorType,
  LegalFormInfo,
  OrganisationTexts,
  SpecialPrepaymentSuggestion,
} from '../types';
import { Api } from '../services/api';
import { useWriteLock } from '../components/WriteLock';
import type { NavigateFn } from '../components/Sidebar';
import { formatCents, formatCentsPlain, formatDate, parseCents } from '../utils/formatters';
import {
  Button,
  Checkbox,
  Field,
  FieldValue,
  HelpPopover,
  Input,
  Notice,
  PageHeader,
  RadioGroup,
  Section,
  Select,
  SkeletonRows,
  Table,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
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
 * Die Folge des Löschens einer gelernten Bankregel — am Knopf, bevor geklickt
 * wird, und im Toast danach.
 *
 * Was rückgängig zu machen ist, bekommt einen Rückgängig-Toast (§8.2). Diese
 * Regel ist es nicht: sie entsteht ausschließlich aus einer bestätigten
 * Zuordnung, von Hand anlegen lässt sie sich nicht. Dann ist die Folge zu
 * benennen, statt ein Rückgängig zu versprechen, das keines wäre.
 */
const DELETE_RULE_HINT =
  'Die Regel lässt sich nicht zurückholen. Sie entsteht neu, sobald ein solcher Umsatz das ' +
  'nächste Mal bestätigt zugeordnet wird.';

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

/**
 * Die Grenzen des Gewerbesteuer-Hebesatzes, die `ClosingSettingsService`
 * durchsetzt: mindestens 200 % nach § 16 Abs. 4 Satz 2 GewStG, nach oben die
 * Plausibilitätsgrenze des Dienstes.
 */
const TRADE_TAX_MIN = 200;
const TRADE_TAX_MAX = 1000;

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

export const SettingsPage: React.FC<{ onNavigate?: NavigateFn }> = ({ onNavigate }) => {
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
  // Die Hinweise zu Rechtsform, Speicherort und Steuerfällen kommen aus dem
  // Backend: welche Rechtsform Entnahmen kennt und welcher Pfad in einem
  // Synchronisationsordner liegt, ist Recht bzw. Umgebung — beides gehört an
  // eine Stelle und nicht in eine zweite Liste hier (BEW-13, UST-08, ARC-06).
  const [hints, setHints] = useState<ComplianceHints | null>(null);
  // Die Freitexte der Organisationsanweisung. Gepflegt werden sie unter
  // Nachweise; hier steht, ob und wie viel davon hinterlegt ist — ein Feld, das
  // nur wiederholt, was sein Label sagt, ist keine Auskunft (§15.3).
  const [orgTexts, setOrgTexts] = useState<OrganisationTexts | null>(null);
  // Die gespeicherte Rechtsform, gegen die der Hinweis gilt. Ein Hinweis zur
  // eben erst ausgewählten, noch nicht gespeicherten Rechtsform stammte aus
  // dem Backend und beschriebe den falschen Stand.
  const [savedLegalForm, setSavedLegalForm] = useState('');
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
  // Der Dienst weist einen Hebesatz außerhalb von 200 bis 1000 % ab (§ 16
  // Abs. 4 Satz 2 GewStG). Die Grenze steht deshalb schon am Feld: sonst
  // erführe der Anwender sie erst als Fehlermeldung des Speicherns, nachdem er
  // die ganze Seite ausgefüllt hat (§8.3).
  const [tradeTaxError, setTradeTaxError] = useState('');
  const [thresholdText, setThresholdText] = useState('');
  // Die Grenze des Leistungsnachweises ebenfalls als Text: eine gelöschte
  // Ziffer ist keine 0 — und eine 0 hieße hier nicht „kein Nachweis", sondern
  // die Voreinstellung (§8.3).
  const [checkThresholdText, setCheckThresholdText] = useState('');
  // Die Basiszinssätze und die gelernten Bankregeln stehen in eigenen Diensten
  // und nicht in den Stammdaten: sie werden sofort gespeichert bzw. gelöscht
  // und nicht über den Knopf im Seitenkopf.
  const [baseRates, setBaseRates] = useState<BaseRate[]>([]);
  const [bankRules, setBankRules] = useState<BankRule[]>([]);
  const [rateFrom, setRateFrom] = useState('');
  const [rateText, setRateText] = useState('');
  const [rateError, setRateError] = useState('');
  const [savingRate, setSavingRate] = useState(false);

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
      setSavedLegalForm(s.legalForm || '');
      try {
        // Nebenauskunft: fehlen die Hinweise, bleiben die Einstellungen
        // bedienbar — sie ändern nichts an dem, was hier gespeichert wird.
        setHints(await Api.getComplianceHints());
      } catch {
        setHints(null);
      }
      try {
        // Ebenfalls Nebenauskunft: fehlen die Freitexte, bleibt die Seite
        // bedienbar.
        setOrgTexts(await Api.getOrganisationTexts());
      } catch {
        setOrgTexts(null);
      }
      setPrepaymentText(s.specialPrepayment ? formatCentsPlain(s.specialPrepayment) : '');
      setCheckThresholdText(
        s.invoiceCheckThreshold ? formatCentsPlain(s.invoiceCheckThreshold) : '',
      );
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
        // Basiszins und gelernte Regeln ebenso: sie hängen an eigenen
        // Diensten, und ohne sie bleiben die Stammdaten bedienbar.
        const [rates, rules] = await Promise.all([Api.getBaseRates(), Api.getBankRules()]);
        setBaseRates(rates);
        setBankRules(rules);
      } catch {
        setBaseRates([]);
        setBankRules([]);
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
    setTradeTaxError('');
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

  /**
   * Trägt einen bekanntgegebenen Basiszinssatz nach.
   *
   * Sofort und nicht über den Knopf im Seitenkopf: der Satz hängt an einem
   * eigenen Dienst, und ein Wert, der bis zum nächsten „Speichern" nur in der
   * Maske stünde, wäre für die Zinsrechnung nicht da.
   */
  async function saveRate() {
    const percent = parsePercent(rateText);
    if (!rateFrom) {
      setRateError('Ohne Stichtag gilt der Satz für keinen Zeitraum.');
      return;
    }
    if (percent === null) {
      setRateError('Der Satz wird als Prozentzahl erfasst, etwa 1,27.');
      return;
    }
    setSavingRate(true);
    try {
      setBaseRates(await Api.saveBaseRate(rateFrom, percent));
      setRateFrom('');
      setRateText('');
      setRateError('');
      toast.success('Basiszinssatz nachgetragen.');
    } catch (e) {
      setRateError(e instanceof Error ? e.message : String(e));
    } finally {
      setSavingRate(false);
    }
  }

  /** Löscht eine gelernte Zuordnung; der nächste Umsatz wird wieder gefragt. */
  async function deleteRule(id: number) {
    try {
      await Api.deleteBankRule(id);
      setBankRules(await Api.getBankRules());
      // Kein Rückgängig-Toast (§8.2): das Backend kennt kein Anlegen einer
      // Regel von Hand — sie entsteht nur aus einer bestätigten Zuordnung. Ein
      // Knopf „Rückgängig", der nichts zurückholte, wäre schlimmer als keiner.
      // Deshalb steht hier die Folge, und sie steht schon vorher am Knopf.
      toast.success('Regel gelöscht. Sie entsteht neu, sobald ein solcher Umsatz wieder zugeordnet wird.');
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    }
  }

  async function save(e: React.FormEvent) {
    e.preventDefault();
    // Auch die Eingabetaste in einem Feld löst das Formular aus; der gesperrte
    // Knopf allein hielte den Prüfermodus deshalb nicht.
    if (!settings || saving || writeLock.locked) return;
    // Ein Hebesatz außerhalb der Grenzen wurde nicht übernommen; gespeichert
    // würde sonst der alte Wert, während der neue im Feld steht.
    if (tradeTaxError) return;
    setSaving(true);
    try {
      await Api.updateCompanySettings(settings);
      // Der eine Knopf speichert beides. Die Abschluss-Einstellungen liegen in
      // eigenen Schlüsseln, aber es wäre eine Zumutung, dafür einen zweiten
      // „Speichern" zu suchen; der Dienst prüft die Grenzen und meldet sich mit
      // seinem eigenen Satz, wenn ihm ein Wert nicht passt.
      applyClosing(await Api.saveClosingSettings(closing));
      // Der Hinweis zur Rechtsform hängt an der gespeicherten Rechtsform; nach
      // dem Speichern gilt eine neue, also wird er neu geholt.
      setSavedLegalForm(settings.legalForm || '');
      try {
        setHints(await Api.getComplianceHints());
      } catch {
        // Der Hinweis ist Beiwerk; das Speichern ist gelungen.
      }
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

  // Wie viel von der Organisationsanweisung hinterlegt ist, und der Anfang des
  // ersten belegten Textes. Der Zähler beantwortet die Frage, wegen der jemand
  // hier hinsieht — ist etwas hinterlegt? —, und die ersten Zeichen sagen, was.
  const organisationValues = Object.values(orgTexts ?? {}).map((text) => text.trim());
  const orgFilledCount = organisationValues.filter((text) => text !== '').length;
  const firstOrganisationText = organisationValues.find((text) => text !== '') ?? '';
  const organisationPreview = `${orgFilledCount} von ${organisationValues.length} Feldern · ${firstOrganisationText}`;

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
            hint="nur für die Teilfreistellung"
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

        {/* Die Grenze des Funktionsumfangs bei Rechtsformen mit Entnahmen
            (BEW-13). Sie steht als Hinweisfläche und nicht hinter dem
            Erklärzeichen: sie betrifft nicht das Feld daneben, sondern das,
            was Buchfink für diesen Mandanten nicht rechnet — und das soll
            niemand erst beim Jahresabschluss erfahren. */}
        {hints?.legalFormNote && settings.legalForm === savedLegalForm && (
          <Notice className="mt-5" text={hints.legalFormNote} />
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

      {/* Die Steuerfälle außerhalb des Funktionsumfangs — OSS/IOSS und die
          Kleinunternehmerregelung — stehen hinter dem Erklärzeichen und nicht
          als Absatz auf der Seite (UST-08). Sie kommen aus dem Backend, weil
          dieselbe Aufzählung in der Verfahrensdokumentation steht. */}
      <Section
        title="Umsatzsteuer"
        action={
          (hints?.taxCaseHints ?? []).length > 0 && (
            <HelpPopover label="Erklärung zu den Grenzen des Funktionsumfangs">
              <span className="block">Buchfink bildet diese Steuerfälle nicht ab:</span>
              <ul className="mt-2 flex flex-col gap-2">
                {(hints?.taxCaseHints ?? []).map((hint) => (
                  <li key={hint}>{hint}</li>
                ))}
              </ul>
            </HelpPopover>
          )
        }
      >
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
            hint="Prozent der Gemeinde"
            error={tradeTaxError || undefined}
            help="Der Hebesatz der Gemeinde, in der die Betriebsstätte liegt. Er steht im Gewerbesteuermessbescheid und auf der Website der Gemeinde; mindestens 200 % (§ 16 Abs. 4 Satz 2 GewStG)."
          >
            <Input
              type="number"
              min={TRADE_TAX_MIN}
              max={TRADE_TAX_MAX}
              align="right"
              value={tradeTaxText}
              onChange={(e) => {
                setTradeTaxText(e.target.value);
                setTradeTaxError('');
              }}
              onBlur={() => {
                const percent = Math.trunc(Number(tradeTaxText));
                // Ein Wert außerhalb der Grenzen bleibt im Feld stehen: ihn
                // stillschweigend auf den alten zurückzusetzen verschwiege,
                // dass die Eingabe nicht angekommen ist.
                if (
                  !Number.isFinite(percent) ||
                  percent < TRADE_TAX_MIN ||
                  percent > TRADE_TAX_MAX
                ) {
                  setTradeTaxError(
                    `Der Hebesatz liegt zwischen ${TRADE_TAX_MIN} % und ${TRADE_TAX_MAX} % (§ 16 Abs. 4 Satz 2 GewStG).`,
                  );
                  return;
                }
                setTradeTaxError('');
                setClosing({ ...closing, tradeTaxRatePercent: percent });
                setTradeTaxText(String(percent));
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

      {/* Die Adressen der Netzdienste und die Umsatzsteuer-Umrechnungskurse
          stehen auf der Seite „Nebenpflichten": dort, wo die Kurshistorie und
          der Verlauf der Bestätigungsabfragen liegen, zu denen sie gehören.
          Der Verweis steht hier, weil sie Einstellungen sind und niemand sie
          zuerst unter „Nebenpflichten" sucht. */}
      <Section
        title="Netzdienste und Umrechnungskurse"
        context="Auf der Seite „Nebenpflichten“: BZSt, Kursdienst, USt-Durchschnittskurse"
        action={
          <HelpPopover label="Erklärung zum Ort dieser Einstellungen">
            Die Adressen der beiden Netzdienste und die monatlichen
            Umsatzsteuer-Umrechnungskurse nach § 16 Abs. 6 UStG stehen bei der Kurshistorie und
            beim Verlauf der Bestätigungsabfragen: eine geänderte Adresse und ein nachgetragener
            Kurs sind dort sofort an den Daten zu sehen, die sie betreffen.
          </HelpPopover>
        }
      >
        <div className="flex flex-col gap-4 max-w-2xl">
          {onNavigate && (
            <div className="flex gap-2">
              <Button
                variant="secondary"
                size="sm"
                onClick={() => onNavigate('obligations', { obligationsTab: 'currency' })}
              >
                Kurse und Endpunkte öffnen
              </Button>
              <Button
                variant="quiet"
                size="sm"
                onClick={() => onNavigate('obligations', { obligationsTab: 'vatid' })}
              >
                Bestätigung der USt-IdNr.
              </Button>
            </div>
          )}
        </div>
      </Section>

      {/* Umstellungszeitpunkt und Organisationsanweisung stehen hier als
          Auskunft und werden unter „Nachweise" gepflegt. Zwei Masken für
          denselben Wert liefen auseinander, sobald jemand die eine benutzt und
          die andere offen hat; die Seite, die die Verfahrensdokumentation
          erzeugt, ist die richtige Stelle dafür (ARC-05, PRF-03). */}
      <Section
        title="Verfahren und Umstellung"
        context="Zu pflegen unter Nachweise"
      >
        {/* Zwei Verweise und nicht einer: der Umstellungszeitpunkt wird im
            Reiter „Versionen & Übernahmen" gepflegt, die Freitexte im Reiter
            „Verfahrensdokumentation". Ein gemeinsamer Knopf setzte den Leser
            in jedem zweiten Fall auf dem falschen Reiter ab. */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Field
            label="Umstellungszeitpunkt aus einem Altsystem"
            hint={hints?.systemChangeDate ? undefined : 'nicht hinterlegt'}
            help={
              hints?.systemChangeNote ||
              'Wer die Buchführung aus einem anderen System übernimmt, hält den Umstellungszeitpunkt fest. Ab ihm läuft die Fünfjahresfrist des § 147 Abs. 6 Satz 6 AO: so lange muss das Altsystem für den Datenzugriff verfügbar bleiben.'
            }
          >
            <span className="flex items-center gap-3 min-w-0">
              <FieldValue className="num">
                {hints?.systemChangeDate ? formatDate(hints.systemChangeDate) : '—'}
              </FieldValue>
              {onNavigate && (
                <Button
                  variant="quiet"
                  size="sm"
                  onClick={() => onNavigate('nachweise', { nachweiseTab: 'versionen' })}
                >
                  Pflegen
                </Button>
              )}
            </span>
          </Field>
          <Field
            label="Organisationsanweisung"
            hint={orgFilledCount > 0 ? undefined : 'nicht hinterlegt'}
            help="Wer scannt, wer prüft, wer freigibt und wie vertreten wird: die Teile der Verfahrensdokumentation, die nur das Unternehmen kennt. Buchfink gibt Muster vor."
          >
            <span className="flex items-center gap-3 min-w-0">
              <FieldValue className="truncate">
                {orgFilledCount > 0 ? organisationPreview : '—'}
              </FieldValue>
              {onNavigate && (
                <Button
                  variant="quiet"
                  size="sm"
                  onClick={() => onNavigate('nachweise', { nachweiseTab: 'verfahren' })}
                >
                  Pflegen
                </Button>
              )}
            </span>
          </Field>
        </div>
      </Section>

      {/*
        Rechnungsprüfung und Mahnwesen (RECH-08, QUE-05). Beide Einstellungen
        steuern Vorgänge und keine Buchung: die eine sagt, ab welchem Betrag ein
        Eingangsbeleg einen Prüfvermerk tragen soll, die andere, wann welche
        Mahnung vorgeschlagen wird.
      */}
      <Section
        title="Rechnungsprüfung und Mahnwesen"
        context="Leistungsnachweis und Mahnstufen"
        action={
          <HelpPopover label="Erklärung zur Rechnungsprüfung">
            Der Vorsteuerabzug setzt eine tatsächlich bezogene Leistung voraus (§ 15 UStG); der
            Leistungsnachweis am Beleg hält fest, wer das bestätigt hat. Die Mahnstufen sind eine
            Vereinbarung des Unternehmens mit sich selbst — das Gesetz kennt keine „erste Mahnung".
            Die Gebühr ist ein Vorschlag und nur ersatzfähig, soweit sie tatsächlich entstandener
            Verzugsschaden ist.
          </HelpPopover>
        }
      >
        <Field
          label="Leistungsnachweis ab"
          hint="Bruttobetrag des Eingangsbelegs"
          className="max-w-sm"
        >
          <Input
            align="right"
            value={checkThresholdText}
            onChange={(e) => setCheckThresholdText(e.target.value)}
            onBlur={() => {
              const value = parseCents(checkThresholdText);
              // Eine Null ist hier keine Abschaltung, sondern ein leeres Feld:
              // wer keinen Nachweis verlangt, setzt die Grenze hoch. Das
              // Backend behandelt die Null ebenso.
              if (value !== null && value > 0) {
                patch({ invoiceCheckThreshold: value });
                setCheckThresholdText(formatCentsPlain(value));
              } else {
                setCheckThresholdText(
                  settings.invoiceCheckThreshold
                    ? formatCentsPlain(settings.invoiceCheckThreshold)
                    : '',
                );
              }
            }}
          />
        </Field>

        <div className="mt-6">
          <DunningLevelsTable
            levels={settings.dunningLevels ?? []}
            onChange={(levels) => patch({ dunningLevels: levels })}
          />
        </div>
      </Section>

      <Section
        title="Basiszinssatz"
        context="Grundlage der Verzugszinsen"
        action={
          <HelpPopover label="Erklärung zum Basiszinssatz">
            Die Deutsche Bundesbank setzt den Basiszinssatz zum 1. Januar und zum 1. Juli neu fest
            und gibt ihn im Bundesanzeiger bekannt (§ 247 BGB). Die Verzugszinsen liegen neun
            Prozentpunkte darüber, gegenüber einem Verbraucher fünf (§ 288 BGB). Ein Wert, der als
            „zu prüfen" steht, ist fortgeschrieben und nicht bekanntgegeben — er ist gegen die
            Bekanntgabe abzugleichen.
          </HelpPopover>
        }
      >
        {baseRates.length === 0 ? (
          <p className="text-body text-ink-muted">Es ist kein Basiszinssatz hinterlegt.</p>
        ) : (
          <Table density="kompakt">
            <Thead>
              <Tr>
                <Th className="w-32">Gültig ab</Th>
                <Th numeric className="w-28">
                  Satz
                </Th>
                <Th>Quelle</Th>
              </Tr>
            </Thead>
            <Tbody>
              {baseRates.map((rate) => (
                <Tr key={rate.validFrom}>
                  <Td className="num">{formatDate(rate.validFrom)}</Td>
                  <Td numeric>{formatBasisPoints(rate.basisPoints)}</Td>
                  <Td className="text-ink-muted">
                    {rate.provisional ? 'fortgeschrieben · zu prüfen' : rate.source || '—'}
                  </Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}

        <div className="flex flex-wrap items-end gap-4 mt-5">
          <Field label="Gültig ab" className="w-44">
            <Input type="date" value={rateFrom} onChange={(e) => setRateFrom(e.target.value)} />
          </Field>
          <Field label="Satz in Prozent" error={rateError || undefined} className="w-44">
            <Input
              align="right"
              placeholder="1,27"
              value={rateText}
              onChange={(e) => {
                setRateText(e.target.value);
                if (rateError) setRateError('');
              }}
            />
          </Field>
          <Button
            variant="secondary"
            loading={savingRate}
            disabled={writeLock.locked}
            title={writeLock.hint}
            onClick={() => void saveRate()}
          >
            Satz nachtragen
          </Button>
        </div>
      </Section>

      <Section
        title="Gelernte Bankregeln"
        context="Muster wiederkehrender Umsätze"
        action={
          <HelpPopover label="Erklärung zu den gelernten Regeln">
            Miete, Kontoführungsentgelt und Gehalt kommen jeden Monat wieder und haben keinen
            offenen Posten. Buchfink merkt sich, wogegen ein solcher Umsatz zuletzt gebucht wurde,
            und schlägt es beim nächsten Mal vor. Gebucht wird davon nichts von selbst: Eine Regel,
            die selbst bucht, würde aus einem einmaligen Griff eine Gewohnheit machen, die niemand
            mehr prüft.
          </HelpPopover>
        }
      >
        {bankRules.length === 0 ? (
          <p className="text-body text-ink-muted">
            Es ist noch keine Regel gelernt. Sie entsteht, sobald ein Umsatz ohne Beleg zugeordnet
            wird.
          </p>
        ) : (
          <Table density="kompakt">
            <Thead>
              <Tr>
                <Th>Muster</Th>
                <Th className="w-28">Richtung</Th>
                <Th className="w-32">Gegenkonto</Th>
                <Th numeric className="w-24">
                  Treffer
                </Th>
                <Th className="w-32">Zuletzt</Th>
                <Th className="w-24" aria-label="Aktionen" />
              </Tr>
            </Thead>
            <Tbody>
              {bankRules.map((rule) => (
                <Tr key={rule.id}>
                  <Td className="max-w-[24rem] truncate" title={rule.pattern}>
                    {rule.label || rule.pattern}
                  </Td>
                  <Td className="text-ink-muted">{rule.moneyIn ? 'Eingang' : 'Ausgang'}</Td>
                  <Td code>{rule.counterAccount}</Td>
                  <Td numeric className="text-ink-subtle">
                    {rule.hits}
                  </Td>
                  <Td className="text-ink-subtle num">
                    {rule.lastUsedAt ? formatDate(rule.lastUsedAt) : '—'}
                  </Td>
                  <Td className="pl-0 text-right">
                    <Button
                      variant="quiet"
                      size="sm"
                      disabled={writeLock.locked}
                      title={writeLock.hint ?? DELETE_RULE_HINT}
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
      </Section>

      <Section title="Speicherort und Schlüssel" context="Gilt für diesen Mandanten">
        <div className="flex flex-col gap-4 max-w-2xl">
          {/* Der Datenordner in einem Synchronisationsordner: § 146 Abs. 2, 2a
              AO. Der Satz kommt aus dem Backend, weil nur dort bekannt ist,
              welche Pfadbestandteile als Cloud-Ordner gelten (ARC-06). */}
          {hints?.cloudWarning && <Notice tone="negative" text={hints.cloudWarning} />}

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


/**
 * Der Basiszinssatz in Hundertsteln eines Prozentpunktes: „1,27" wird zu 127.
 *
 * Ganzzahlig über die Brücke, weil die Bekanntgabe zwei Nachkommastellen hat
 * und eine Gleitkommazahl die Zinsrechnung um Cents verschöbe. Negative Sätze
 * sind zulässig: zwischen 2016 und 2022 lag der Basiszinssatz unter null.
 */
function parsePercent(input: string): number | null {
  const raw = input.trim().replace(/\s|%/g, '').replace(',', '.');
  if (!raw) return null;
  if (!/^[+-]?\d+(\.\d{1,2})?$/.test(raw)) return null;
  const value = Number(raw);
  if (!Number.isFinite(value)) return null;
  return Math.round(value * 100);
}

/** Und zurück: 127 wird „1,27 %". */
function formatBasisPoints(points: number): string {
  if (!Number.isFinite(points)) return '—';
  return `${(points / 100).toLocaleString('de-DE', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })} %`;
}

/**
 * Die Voreinstellung der Mahnstufen — dieselben drei Stufen wie in
 * domain.DefaultDunningLevels().
 *
 * Sie stehen hier nur, damit der Knopf „Voreinstellung eintragen" sie in die
 * Tabelle schreiben kann. Gerechnet wird mit ihnen nicht: der Mahnlauf nimmt
 * die Stufen aus den Einstellungen, und was dort fehlt, ersetzt das Backend.
 */
const DEFAULT_DUNNING_LEVELS: DunningLevel[] = [
  { level: 1, label: 'Zahlungserinnerung', daysAfterDue: 7, fee: 0 },
  { level: 2, label: '1. Mahnung', daysAfterDue: 21, fee: 500 },
  { level: 3, label: '2. Mahnung', daysAfterDue: 35, fee: 1000 },
];

/**
 * Die Stufenfolge des Mahnwesens.
 *
 * Die Stufe folgt aus der Reihenfolge und wird nicht eingegeben: das Backend
 * sortiert nach dem Abstand zur Fälligkeit und nummeriert neu. Eine Nummer im
 * Formular ließe zwei Stufen mit derselben Zahl zu, und der Mahnlauf wüsste
 * nicht, welche als Nächste kommt.
 */
const DunningLevelsTable: React.FC<{
  levels: DunningLevel[];
  onChange: (levels: DunningLevel[]) => void;
}> = ({ levels, onChange }) => {
  // Tage und Gebühr werden als Text geführt und erst beim Verlassen des Feldes
  // umgerechnet (§8.3): eine gelöschte Ziffer ist keine 0.
  const [drafts, setDrafts] = useState<Record<number, { days: string; fee: string }>>({});

  const draftOf = (index: number, level: DunningLevel) =>
    drafts[index] ?? { days: String(level.daysAfterDue), fee: formatCentsPlain(level.fee) };

  const update = (index: number, next: Partial<DunningLevel>) => {
    onChange(levels.map((level, i) => (i === index ? { ...level, ...next } : level)));
  };

  /**
   * Eine weitere Stufe. Die Nummer vergibt das Backend beim Speichern neu; hier
   * steht sie nur, damit die Zeile eine Anzeige hat.
   */
  const add = () => {
    const last = levels[levels.length - 1];
    onChange([
      ...levels,
      {
        level: levels.length + 1,
        label: `${levels.length + 1}. Mahnung`,
        daysAfterDue: (last?.daysAfterDue ?? 0) + 14,
        fee: last?.fee ?? 0,
      },
    ]);
  };

  /** Entfernt eine Stufe. Die Entwürfe rutschen mit — sie hängen am Index. */
  const remove = (index: number) => {
    setDrafts({});
    onChange(levels.filter((_, i) => i !== index));
  };

  const addButton = (
    <Button variant="secondary" size="sm" onClick={add}>
      Stufe hinzufügen
    </Button>
  );

  if (levels.length === 0) {
    return (
      <div className="flex flex-col items-start gap-3">
        {/* Eine leere Tabelle heißt nicht, dass nichts gespeichert ist: das
            Speichern lässt eine leere Liste bewusst aus, damit ein Formular,
            das die Stufen nicht kennt, sie nicht stumm löscht
            (repository/settings_gorm.go). Der Text darf deshalb nicht die
            Voreinstellung behaupten — die stellt der Knopf daneben her, indem
            er die drei Stufen einträgt, die dann auch gespeichert werden. */}
        <p className="text-body text-ink-muted">
          Es ist keine Stufe eingetragen. Bis Stufen gespeichert sind, gelten die zuletzt
          gespeicherten weiter; ist noch nie eine gespeichert worden, mahnt Buchfink nach
          7, 21 und 35 Tagen.
        </p>
        <div className="flex items-center gap-2">
          {addButton}
          <Button
            variant="secondary"
            size="sm"
            onClick={() => onChange(DEFAULT_DUNNING_LEVELS.map((level) => ({ ...level })))}
          >
            Voreinstellung eintragen
          </Button>
        </div>
      </div>
    );
  }

  return (
    <>
      <Table density="kompakt">
        <Thead>
          <Tr>
            <Th className="w-12" numeric>
              Nr.
            </Th>
            <Th>Bezeichnung</Th>
            <Th className="w-40">Tage nach Fälligkeit</Th>
            <Th className="w-40">Gebühr</Th>
            <Th className="w-28" aria-label="Aktion" />
          </Tr>
        </Thead>
        <Tbody>
          {levels.map((level, index) => {
            const draft = draftOf(index, level);
            return (
              <Tr key={`${level.level}-${index}`}>
                <Td numeric className="text-ink-subtle">
                  {level.level}
                </Td>
                <Td>
                  <Input
                    value={level.label}
                    onChange={(e) => update(index, { label: e.target.value })}
                  />
                </Td>
                <Td>
                  <Input
                    type="number"
                    min={0}
                    align="right"
                    value={draft.days}
                    onChange={(e) =>
                      setDrafts((prev) => ({
                        ...prev,
                        [index]: { ...draftOf(index, level), days: e.target.value },
                      }))
                    }
                    onBlur={() => {
                      const days = Number(draft.days);
                      const value = Number.isFinite(days) && days >= 0 ? Math.trunc(days) : level.daysAfterDue;
                      update(index, { daysAfterDue: value });
                      setDrafts((prev) => ({
                        ...prev,
                        [index]: { ...draftOf(index, level), days: String(value) },
                      }));
                    }}
                  />
                </Td>
                <Td>
                  <Input
                    align="right"
                    value={draft.fee}
                    onChange={(e) =>
                      setDrafts((prev) => ({
                        ...prev,
                        [index]: { ...draftOf(index, level), fee: e.target.value },
                      }))
                    }
                    onBlur={() => {
                      const fee = parseCents(draft.fee);
                      const value = fee !== null && fee >= 0 ? fee : level.fee;
                      update(index, { fee: value });
                      setDrafts((prev) => ({
                        ...prev,
                        [index]: { ...draftOf(index, level), fee: formatCentsPlain(value) },
                      }));
                    }}
                  />
                </Td>
                <Td className="text-right">
                  <Button variant="quiet" size="sm" onClick={() => remove(index)}>
                    Entfernen
                  </Button>
                </Td>
              </Tr>
            );
          })}
        </Tbody>
      </Table>
      <div className="mt-3">{addButton}</div>
    </>
  );
};
