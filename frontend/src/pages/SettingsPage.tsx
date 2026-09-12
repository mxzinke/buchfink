import React, { useEffect, useState } from 'react';
import { ChevronDown, Save, Shield } from 'lucide-react';
import { InvoiceNumberRange } from '../components/InvoiceNumberRange';
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
  RetentionRules,
  SpecialPrepaymentSuggestion,
  VatPeriodProposal,
} from '../types';
import { RETENTION_CLASS_LABELS } from '../types';
import { Api } from '../services/api';
import { useWriteLock } from '../components/WriteLock';
import type { NavigateFn } from '../components/Sidebar';
import { formatCents, formatCentsPlain, formatDate, parseCents } from '../utils/formatters';
import {
  Button,
  Checkbox,
  Field,
  FieldValue,
  FormGrid,
  Help,
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

export const SettingsPage: React.FC<{ year: number; onNavigate?: NavigateFn }> = ({ year, onNavigate }) => {
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
  const [showNumberRanges, setShowNumberRanges] = useState(false);
  // Die Hinweise zu Rechtsform, Speicherort und Steuerfällen kommen aus dem
  // Backend: welche Rechtsform Entnahmen kennt und welcher Pfad in einem
  // Synchronisationsordner liegt, ist Recht bzw. Umgebung — beides gehört an
  // eine Stelle und nicht in eine zweite Liste hier (BEW-13, UST-08, ARC-06).
  const [hints, setHints] = useState<ComplianceHints | null>(null);
  // Der Umstellungszeitpunkt aus einem Altsystem. Er wird hier gepflegt und
  // nicht in einer Übersicht: er ist eine Angabe über dieses Unternehmen wie
  // Rechtsform und Geschäftsjahresbeginn, und eine Übersicht, die nur berichtet
  // was in Ordnung ist, ist der falsche Ort für ein Eingabefeld.
  const [systemChangeDate, setSystemChangeDate] = useState('');
  // Die Freitexte der Organisationsanweisung. Gepflegt werden sie bei der
  // Betriebsprüfung, wo die Verfahrensdokumentation entsteht, die sie
  // aufnimmt; hier steht, ob und wie viel davon hinterlegt ist — ein Feld, das
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
  // wäre hier eine Lücke, nicht eine Antwort.
  const [captureDaysText, setCaptureDaysText] = useState('10');
  // Die Nachfrist zur Festschreibung ebenso; hier ist die 0 allerdings eine
  // Antwort: keine Nachfrist über das Ende des Folgemonats hinaus.
  const [graceDaysText, setGraceDaysText] = useState('0');
  const [suggestion, setSuggestion] = useState<SpecialPrepaymentSuggestion | null>(null);
  // Der Voranmeldungszeitraum, der sich aus der Steuer des Vorjahres ergibt
  // (§ 18 Abs. 2 UStG), und die Fristentabelle mit ihrem Rechtsstand. Beides
  // kommt aus dem Backend: ein Vorschlag und eine Frist sind Rechtsfolgen und
  // keine Wahl der Oberfläche.
  const [vatProposal, setVatProposal] = useState<VatPeriodProposal | null>(null);
  const [retentionRules, setRetentionRules] = useState<RetentionRules | null>(null);
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
  // Ziffer ist keine 0 — und eine 0 bedeutet hier die Voreinstellung, nicht
  // „kein Nachweis" (§8.3).
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
        const loaded = await Api.getComplianceHints();
        setHints(loaded);
        setSystemChangeDate(loaded?.systemChangeDate ?? '');
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
        // Nebenauskunft: fehlt der Vorschlag — etwa weil das Vorjahr noch
        // nicht angemeldet ist —, bleibt der Zeitraum von Hand wählbar.
        setVatProposal(await Api.getVatPeriodProposal(s.fiscalYear || 0));
      } catch {
        setVatProposal(null);
      }
      try {
        setRetentionRules(await Api.getRetentionRules());
      } catch {
        setRetentionRules(null);
      }
      try {
        // Basiszins und gelernte Regeln ebenso: sie werden in eigenen
        // Diensten gespeichert, und ohne sie bleiben die Stammdaten bedienbar.
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
      if (path) { toast.success(`Wiederherstellungsschlüssel gespeichert: ${path}`); setAppConfig(await Api.getAppConfig()); }
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
   * Sofort und nicht über den Knopf im Seitenkopf: der Satz wird über einen
   * eigenen Dienst gespeichert, und ein Wert, der bis zum nächsten
   * „Speichern" nur in der Maske stünde, wäre für die Zinsrechnung nicht da.
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
      // Nur wenn er sich geändert hat: der Umstellungszeitpunkt liegt in einem
      // eigenen Schlüssel und schreibt bei jedem Setzen einen Protokolleintrag.
      // Ihn bei jedem Speichern der Einstellungen mitzuschreiben füllte das
      // Änderungsprotokoll mit Einträgen über eine Änderung, die niemand
      // vorgenommen hat.
      if (systemChangeDate !== (hints?.systemChangeDate ?? '')) {
        await Api.setSystemChangeDate(systemChangeDate);
      }
      // Der Hinweis zur Rechtsform richtet sich nach der gespeicherten
      // Rechtsform; nach dem Speichern gilt eine neue, also wird er neu geholt.
      setSavedLegalForm(settings.legalForm || '');
      try {
        const loaded = await Api.getComplianceHints();
        setHints(loaded);
        setSystemChangeDate(loaded?.systemChangeDate ?? '');
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
        <FormGrid>
          <Field label="Firmen- oder Inhabername">
            <Input value={settings.companyName} onChange={(e) => patch({ companyName: e.target.value })} />
          </Field>
          <Field helpSummary="Die Rechtsform beeinflusst, wie Ihr Unternehmen steuerlich behandelt wird."
            label="Rechtsform"
            hint={derivedInvestor?.label}
            explain={derivedInvestor?.note}
          >
            <Select
              items={legalFormItems}
              value={settings.legalForm || ''}
              onValueChange={(next) => patch({ legalForm: String(next) })}
            />
          </Field>
          <Field helpSummary="Geben Sie Ihre Steuernummer oder Umsatzsteuer-ID für die Rechnungsstellung an."
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

          {/*
            Die Anlegerstellung für § 20 InvStG folgt aus der Rechtsform.
            Sichtbar wird sie nur, wo diese sie nicht hergibt — bei einer
            Personengesellschaft — oder wo jemand sie ausdrücklich anders
            festlegen will. Als eigenes Pflichtfeld stünde hier sonst eine
            Rechtsfrage, die die meisten nie beantworten müssten. Sie steht im
            Raster der übrigen Felder und nicht darunter: ein Feld, das seine
            eigene Breite mitbringt, sieht aus wie ein Nachtrag.
          */}
          {(needsInvestorChoice || showInvestorChoice) && (
            <Field helpSummary="Die Art des Anlegers bestimmt den steuerfreien Anteil von Fondserträgen."
              label="Anlegerstellung für Investmentanteile"
              optional={!needsInvestorChoice}
              hint="nur für die Teilfreistellung"
              explain="Der Satz richtet sich nach dem Anleger. Bei einer Personengesellschaft bestimmt ihn der einzelne Gesellschafter (§ 20 Abs. 3a InvStG). Und auch eine Körperschaft hat nicht immer 80 %: für Lebens- und Krankenversicherer, für Kreditinstitute mit Handelsbestand und für Pensionsfonds nehmen § 20 Abs. 1 Sätze 4 und 5 die Erhöhung zurück."
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
        </FormGrid>

        {!needsInvestorChoice && !showInvestorChoice && (
          <Button
            variant="quiet"
            size="sm"
            className="mt-4 -ml-2.5"
            onClick={() => setShowInvestorChoice(true)}
          >
            Anlegerstellung für Investmentanteile abweichend festlegen
          </Button>
        )}

        {/* Die Grenze des Funktionsumfangs bei Rechtsformen mit Entnahmen
            (BEW-13). Sie steht als Hinweisfläche und nicht hinter dem
            Erklärzeichen: sie erklärt, was Buchfink für diesen Mandanten
            nicht rechnet, und nicht das Feld daneben — das soll niemand
            erst beim Jahresabschluss erfahren. */}
        {hints?.legalFormNote && settings.legalForm === savedLegalForm && (
          <Notice className="mt-5" text={hints.legalFormNote} />
        )}
      </Section>

      <Section title="Anschrift">
        <FormGrid>
          <Field label="Straße und Hausnummer" className="md:col-span-2">
            <Input value={settings.street} onChange={(e) => patch({ street: e.target.value })} />
          </Field>
          <Field label="PLZ und Ort">
            <Input value={settings.zipCity} onChange={(e) => patch({ zipCity: e.target.value })} />
          </Field>
          <Field label="Land">
            <Input value={settings.country} onChange={(e) => patch({ country: e.target.value })} />
          </Field>
        </FormGrid>
      </Section>

      <Section helpSummary="Legen Sie Rechnungsnummern und Kontaktdaten für Ihre Rechnungen fest."
        title="Rechnungsstellung"
        explain={
          <>
            Die Systematik des Nummernkreises gehört in die Verfahrensdokumentation und steht
            deshalb als Einstellung: Ein Mandant mit vorhandener Buchhaltung führt seine Systematik
            fort. Ansprechpartner, Telefon und E-Mail sind bei einer XRechnung Pflichtangaben
            (BR-DE-2 bis BR-DE-7) — eine Behörde, die nicht zurückfragen kann, weist die Rechnung
            zurück.
          </>
        }
      >
        <FormGrid>
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
        </FormGrid>
        <Button
          variant="quiet"
          size="sm"
          className="mt-4 -ml-2.5"
          aria-expanded={showNumberRanges}
          aria-controls="invoice-number-ranges"
          onClick={() => setShowNumberRanges((open) => !open)}
        >
          <ChevronDown
            className={`w-3.5 h-3.5 transition-transform duration-120 ${showNumberRanges ? 'rotate-180' : ''}`}
            strokeWidth={1.5}
            aria-hidden="true"
          />
          Nummernkreise
        </Button>
        {showNumberRanges && (
          <div id="invoice-number-ranges" className="mt-5 space-y-6">
            <FormGrid>
              <Field helpSummary="Legen Sie fest, wie Jahr und laufende Nummer in Ihrer Rechnungsnummer erscheinen."
                label="Rechnungsnummernformat"
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
              <Field helpSummary="Legen Sie fest, wie Buchfink Ihre eingehenden Belege nummeriert."
                label="Belegnummernformat"
                hint="{JAHR} und {NR:4}"
                explain="Dieselben zwei Platzhalter für den Belegnummernkreis: {JAHR} für das Geschäftsjahr, {NR:4} für den Zähler mit vier Stellen. Leer heißt ER-{JAHR}-{NR:4}. Bestehende Belegnummern bleiben gültig (BEL-02)."
              >
                <Input
                  className="code-num"
                  placeholder="ER-{JAHR}-{NR:4}"
                  value={settings.receiptNumberFormat}
                  onChange={(e) => patch({ receiptNumberFormat: e.target.value })}
                />
              </Field>
            </FormGrid>
            <InvoiceNumberRange key={year} year={year} />
          </div>
        )}
      </Section>

      {/* Ohne diese drei Angaben fehlt dem Jahresabschluss der Kopf, den
          § 264 Abs. 1a HGB verlangt. Sie standen bisher nur im Gründungsweg. */}
      <Section helpSummary="Diese Registerangaben erscheinen im Kopf Ihres Jahresabschlusses."
        title="Registereintragung"
        explain={
          <>
            Auf jedem Jahresabschluss einer Kapitalgesellschaft sind Firma, Sitz, Registergericht
            und Registernummer anzugeben (§ 264 Abs. 1a HGB). Buchfink setzt sie in den Kopf von
            Bilanz, Gewinn- und Verlustrechnung und E-Bilanz.
          </>
        }
      >
        <FormGrid>
          <Field label="Sitz der Gesellschaft">
            <Input
              value={settings.seat}
              onChange={(e) => patch({ seat: e.target.value })}
              placeholder="Ort laut Satzung"
            />
          </Field>
          <Field label="Geschäftsführung" hint="Alle Geschäftsführer mit ausgeschriebenem Vor- und Nachnamen; bei GmbH und UG Pflicht auf Rechnungen.">
            <Input value={settings.managingDirectors || ''} onChange={(e) => patch({ managingDirectors: e.target.value })} placeholder="Mara Beispiel, Timo Muster" />
          </Field>
          <Field label="Aufsichtsratsvorsitz" optional hint="Ausfüllen, wenn ein Aufsichtsrat mit Vorsitz besteht.">
            <Input value={settings.supervisoryBoardChair || ''} onChange={(e) => patch({ supervisoryBoardChair: e.target.value })} />
          </Field>
          <Field label="Verkäuferkennung für E-Rechnungen" optional hint="Erforderlich, wenn weder USt-ID noch Registernummer vorhanden ist. Verwenden Sie eine feste, mit Ihren Kunden abgestimmte Kennung.">
            <Input value={settings.sellerIdentifier || ''} onChange={(e) => patch({ sellerIdentifier: e.target.value })} placeholder="Zum Beispiel Ihre Lieferantennummer" />
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
        </FormGrid>
      </Section>

      <Section helpSummary="Buchfink ordnet Ihre Vorgänge anhand ihres Datums einem Geschäftsjahr zu."
        title="Geschäftsjahr"
        explain={
          <>
            Belege, Rechnungen und Zahlungen haben ein Datum, daraus ergibt sich das Geschäftsjahr
            von selbst. Im Grenzbereich zum Jahreswechsel lässt sich die Zuordnung einer Buchung im
            Journal übersteuern.
          </>
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
          <FormGrid className="mt-5">
            <Field label="Beginn des Wirtschaftsjahres">
              <Select
                items={MONTHS}
                value={startMonth}
                onValueChange={(month) => patch({ fiscalYearStartMonth: month })}
              />
            </Field>
          </FormGrid>
        )}
      </Section>

      {/* Die Steuerfälle außerhalb des Funktionsumfangs — OSS/IOSS und die
          Kleinunternehmerregelung — stehen hinter dem Erklärzeichen und nicht
          als Absatz auf der Seite (UST-08). Sie kommen aus dem Backend, weil
          dieselbe Aufzählung in der Verfahrensdokumentation steht. */}
      <Section helpSummary="Hier sehen Sie die unterstützten Steuerfälle und Grenzen der Umsatzsteuerfunktionen."
        title="Umsatzsteuer"
        explain={
          (hints?.taxCaseHints ?? []).length > 0 && (
            <>
              <span className="block">Buchfink bildet diese Steuerfälle nicht ab:</span>
              <ul className="mt-2 flex flex-col gap-2">
                {(hints?.taxCaseHints ?? []).map((hint) => (
                  <li key={hint}>{hint}</li>
                ))}
              </ul>
            </>
          )
        }
      >
        <FormGrid>
          <Field helpSummary="Wählen Sie, ob Sie Ihre Umsatzsteuer monatlich oder vierteljährlich melden."
            label="Voranmeldezeitraum"
            hint={
              vatProposal
                ? `Vorschlag aus ${vatProposal.basedOnYear}: ${
                    VAT_PERIODS.find((item) => item.value === vatProposal.proposed)?.label ??
                    vatProposal.proposed
                  }`
                : undefined
            }
            // Der allgemeine Satz und die Herleitung des Vorschlags stehen
            // hinter demselben Fragezeichen: Zwei nebeneinander sahen aus wie
            // ein Fehler, und wer die Frage stellt, will beides wissen.
            explain={
              vatProposal
                ? `Monatlich gilt bei Neugründung und hoher Zahllast, vierteljährlich ist der Regelfall. ${
                    vatProposal.reference
                  } Die Steuer des Vorjahres betrug ${formatCents(
                    vatProposal.priorYearTax,
                  )} (Kennziffer 83).${
                    vatProposal.complete
                      ? ''
                      : ` Für ${vatProposal.missingPeriods} Zeiträume des Vorjahres liegt keine übermittelte Anmeldung vor; der Vorschlag ist deshalb unvollständig.`
                  }`
                : 'Monatlich gilt bei Neugründung und hoher Zahllast, vierteljährlich ist der Regelfall.'
            }
          >
            <Select
              items={VAT_PERIODS}
              value={settings.vatPeriod || 'quarter'}
              onValueChange={(period) => patch({ vatPeriod: period as CompanySettings['vatPeriod'] })}
            />
          </Field>
          <Field helpSummary="Buchfink unterstützt die Besteuerung nach erbrachter Leistung, auch vor der Zahlung."
            label="Besteuerungsart"
            hint="nach vereinbarten Entgelten"
            explain="Buchfink rechnet nach § 16 Abs. 1 Satz 1 UStG. Bei Istversteuerung entstünde die Steuer erst mit der Vereinnahmung, die Buchungen sähen anders aus — der Buchungskern weist sie deshalb ab, statt sie stillschweigend falsch zu behandeln."
          >
            <Select items={[{ value: 'SOLL', label: 'Sollversteuerung' }]} value="SOLL" disabled />
          </Field>
        </FormGrid>

        {/* Der Vorschlag wird nicht stillschweigend übernommen: der Zeitraum
            wird vom Finanzamt festgesetzt, und die Befreiung von der Abgabe ist
            dessen Entscheidung. Buchfink nennt ihn und überlässt die Umstellung
            dem Anwender (UST-03 K1). */}
        {vatProposal?.changes && (
          <Notice
            className="mt-5"
            text={
              vatProposal.note ||
              `Aus der Steuer des Jahres ${vatProposal.basedOnYear} folgt ein anderer Voranmeldungszeitraum als der eingestellte.`
            }
            action={
              <Button
                variant="secondary"
                size="sm"
                onClick={() =>
                  patch({ vatPeriod: vatProposal.proposed as CompanySettings['vatPeriod'] })
                }
              >
                Vorschlag übernehmen
              </Button>
            }
          />
        )}

        <Checkbox
          className="mt-5"
          checked={settings.permanentExtension}
          onCheckedChange={(next) => patch({ permanentExtension: Boolean(next) })}
          label={
            <span className="flex items-center">
              Dauerfristverlängerung
              <Help summary="Eine genehmigte Dauerfristverlängerung gibt Ihnen einen Monat mehr Zeit für die Voranmeldung." label="Erklärung zur Dauerfristverlängerung">
                Mit der Dauerfristverlängerung wird jede Voranmeldung einen Monat später fällig
                (§§ 46 bis 48 UStDV). Wer monatlich anmeldet, hat dafür bis zum 10. Februar eine
                Sondervorauszahlung von einem Elftel der Vorauszahlungen des Vorjahres anzumelden
                und zu zahlen; angerechnet wird sie in der letzten Voranmeldung des Jahres.
              </Help>
            </span>
          }
          hint="verschiebt jede Fälligkeit um einen Monat"
        />

        {settings.permanentExtension && (
          <FormGrid className="mt-5">
            <Field helpSummary="Tragen Sie die Sondervorauszahlung ein, die Sie beim Finanzamt angemeldet haben."
              label="Angemeldete Sondervorauszahlung"
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
          </FormGrid>
        )}
      </Section>

      {/* Anzeige und keine Einstellung: eine Aufbewahrungsfrist ist keine Wahl
          des Anwenders (ARC-01 K4). Sie steht hier, weil die Einstellungen der
          Ort ist, an dem nachgesehen wird, was gilt. */}
      <Section helpSummary="Hier sehen Sie, wie lange Sie die verschiedenen Unterlagen mindestens aufbewahren müssen."
        title="Aufbewahrungsfristen"
        context={
          retentionRules
            ? `${retentionRules.source} · gültig ab ${formatDate(retentionRules.validFrom)}`
            : 'Gesetzesstand'
        }
        explain={
          <>
            Die Fristen kommen aus einer datierten Tabelle und nicht aus dem Programmcode: ändert
            der Gesetzgeber sie, gilt die neue Frist ab ihrem Stichtag, und die alte bleibt für die
            Unterlagen davor stehen. Am einzelnen Beleg lässt sich die Frist verlängern, nie
            verkürzen.
          </>
        }
      >
        {retentionRules === null ? (
          <SkeletonRows rows={3} />
        ) : (
          <Table>
            <Thead>
              <Tr>
                <Th>Klasse</Th>
                <Th numeric>Jahre</Th>
                <Th>Grundlage</Th>
                <Th>Vorige Frist</Th>
              </Tr>
            </Thead>
            <Tbody>
              {(retentionRules.classes ?? []).map((entry) => (
                <Tr key={entry.class}>
                  <Td>{entry.label || RETENTION_CLASS_LABELS[entry.class]}</Td>
                  <Td numeric>{entry.years}</Td>
                  <Td className="text-ink-muted">{entry.legalBasis}</Td>
                  <Td className="text-ink-subtle num">
                    {entry.previous
                      ? `${entry.previous.years} Jahre bis ${formatDate(entry.previous.replacedFrom)}`
                      : '—'}
                  </Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
      </Section>

      <Section helpSummary="Legen Sie fest, wann Buchfink auf noch nicht erfasste Belege hinweisen soll."
        title="Prüfläufe"
        explain={
          <>
            Der Prüflauf vor der Festschreibung meldet abgelegte, aber nicht gebuchte Belege. Die
            GoBD nennt in Rz. 47 zehn Tage für die Erfassung unbarer Geschäftsvorfälle; wer anders
            arbeitet, setzt hier seinen eigenen Wert. Die Nachfrist zur Festschreibung entscheidet
            daneben, ab wann ein nicht festgeschriebener Monat als überfällig gilt — in der
            Fristenliste wie im Prüfbericht.
          </>
        }
      >
        <FormGrid>
          <Field label="Belege spätestens erfassen nach" hint="Tage nach Eingang">
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
        </FormGrid>
      </Section>



      {/*
        Die Abschlussbausteine rechnen mit diesen drei Angaben: ohne sie liefe
        jede Installation mit 400 % Hebesatz, monatsgenauer Abgrenzung und 800
        Euro Schwelle, als wäre das gewählt worden.
      */}
      <Section helpSummary="Diese Einstellungen beeinflussen die Berechnungen Ihres Jahresabschlusses."
        title="Jahresabschluss"
        context="Steuert die Abschlussbausteine"
        explain={
          <>
            Der Hebesatz geht in die Steuerrückstellung ein, die Abgrenzungsmethode in jeden
            Abgrenzungsposten und die Vorschlagsschwelle allein in die Vorschlagsliste. Alle drei
            gelten für den ganzen Mandanten und über alle Geschäftsjahre; die Änderung wird
            protokolliert.
          </>
        }
      >
        <FormGrid>
          <Field helpSummary="Tragen Sie den Gewerbesteuer-Hebesatz Ihrer Gemeinde ein."
            label="Gewerbesteuer-Hebesatz"
            hint="Prozent der Gemeinde"
            error={tradeTaxError || undefined}
            explain="Der Hebesatz der Gemeinde, in der die Betriebsstätte liegt. Er steht im Gewerbesteuermessbescheid und auf der Website der Gemeinde; mindestens 200 % (§ 16 Abs. 4 Satz 2 GewStG)."
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
                    `Buchfink unterstützt Hebesätze zwischen ${TRADE_TAX_MIN} % und ${TRADE_TAX_MAX} %.`,
                  );
                  return;
                }
                setTradeTaxError('');
                setClosing({ ...closing, tradeTaxRatePercent: percent });
                setTradeTaxText(String(percent));
              }}
            />
          </Field>
          <Field helpSummary="Wählen Sie, ob Buchfink zeitliche Abgrenzungen nach Monaten oder Tagen verteilt."
            label="Abgrenzungsmethode"
            explain="Monatsgenau verteilt nach Zwölfteln, taggenau nach Kalendertagen. Beides ist zulässig; § 252 Abs. 1 Nr. 6 HGB verlangt nur, dass es dabei bleibt — die Wahl gilt deshalb für alle Posten."
          >
            <Select
              items={ACCRUAL_METHODS}
              value={closing.accrualMethod}
              onValueChange={(next) =>
                setClosing({ ...closing, accrualMethod: next as AccrualMethod })
              }
            />
          </Field>
          <Field helpSummary="Ab diesem Betrag schlägt Buchfink eine zeitliche Abgrenzung vor."
            label="Vorschlagsschwelle der Abgrenzung"
            hint="unterhalb nur Anzeige"
            explain="Nur für die Vorschlagsliste: Handelsrechtlich gibt es keine Grenze, jeder Posten ist abzugrenzen (§ 250 HGB). Die 800 Euro sind das steuerliche Wahlrecht des § 6 Abs. 2 EStG, das die Finanzverwaltung auch für die Abgrenzung zulässt — wer es nicht nutzen will, trägt hier 0,00 ein."
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
          <Field helpSummary="Wählen Sie, ob vorgetragene Abgrenzungen monatlich oder einmal im Jahr aufgelöst werden."
            label="Auflösung im Folgejahr"
            explain="Der Saldenvortrag bucht die Auflösung mit. Einmal je Jahr hält die Zahl der Abschlussbuchungen klein; monatlich braucht, wer unterjährig auswertet — sonst trägt der Januar den gesamten Vorjahresaufwand."
          >
            <Select
              items={ACCRUAL_RELEASES}
              value={closing.accrualRelease}
              onValueChange={(next) =>
                setClosing({ ...closing, accrualRelease: next as AccrualReleaseCycle })
              }
            />
          </Field>
        </FormGrid>
      </Section>

      <Section title="Bankverbindung">
        <FormGrid>
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
          <Field helpSummary="Buchfink verwendet den Kontenrahmen SKR04 für die doppelte Buchführung."
            label="Kontenrahmen"
            explain="Buchfink richtet sich an bilanzierende Gesellschaften und bucht im SKR04. Die Kleinunternehmerregelung nach § 19 UStG wird nicht unterstützt; ein Kleinunternehmer als Lieferant ist dagegen ein normaler Fall und wird am Kontakt hinterlegt."
          >
            <FieldValue>SKR04 · Bilanz und GuV</FieldValue>
          </Field>
        </FormGrid>
      </Section>

      {/* Die Adressen der Netzdienste und die Umsatzsteuer-Umrechnungskurse
          stehen auf der Seite „Nebenpflichten": dort, wo die Kurshistorie und
          der Verlauf der Bestätigungsabfragen liegen, zu denen sie gehören.
          Der Verweis steht hier, weil sie Einstellungen sind und niemand sie
          zuerst unter „Nebenpflichten" sucht. */}
      <Section helpSummary="Hier finden Sie externe Abfragedienste und die gespeicherten Umrechnungskurse."
        title="Netzdienste und Umrechnungskurse"
        context="Auf der Seite „Nebenpflichten“: BZSt, Kursdienst, USt-Durchschnittskurse"
        explain={
          <>
            Die Adressen der beiden Netzdienste und die monatlichen
            Umsatzsteuer-Umrechnungskurse nach § 16 Abs. 6 UStG stehen bei der Kurshistorie und
            beim Verlauf der Bestätigungsabfragen: eine geänderte Adresse und ein nachgetragener
            Kurs sind dort sofort an den Daten zu sehen, die sie betreffen.
          </>
        }
      >
        {/* Zwei Wege, zwei Knöpfe: der eine führt auf die Kurse, der andere auf
            den Verlauf der Bestätigungsabfragen. Beide sehen gleich aus, weil
            beide dasselbe tun — eine Seite aufschlagen. */}
        {onNavigate && (
          <div className="flex flex-wrap gap-2">
            <Button
              variant="secondary"
              onClick={() => onNavigate('obligations', { obligationsTab: 'currency' })}
            >
              Kurse und Endpunkte
            </Button>
            <Button
              variant="secondary"
              onClick={() => onNavigate('obligations', { obligationsTab: 'vatid' })}
            >
              Bestätigung der USt-IdNr.
            </Button>
          </div>
        )}
      </Section>

      {/* Der Umstellungszeitpunkt ist eine Angabe über dieses Unternehmen und
          steht deshalb hier, bei Rechtsform und Geschäftsjahr. Die
          Organisationsanweisung ist ein halbes Dutzend Absätze Fließtext: sie
          steht dort, wo die Verfahrensdokumentation entsteht, die sie aufnimmt
          (ARC-05, PRF-03), und hier nur als Auskunft darüber, ob sie
          hinterlegt ist. */}
      <Section title="Verfahren und Umstellung" context="Herkunft der Daten und Organisation">
        <FormGrid>
          <Field helpSummary="Halten Sie fest, wann Sie von Ihrem bisherigen Buchhaltungsprogramm umgestiegen sind."
            label="Umstellungszeitpunkt aus einem Altsystem"
            optional
            hint={systemChangeDate ? undefined : 'nicht hinterlegt'}
            explain={
              hints?.systemChangeNote ||
              'Wer die Buchführung aus einem anderen System übernimmt, hält den Umstellungszeitpunkt fest. Ab ihm läuft die Fünfjahresfrist des § 147 Abs. 6 Satz 6 AO: so lange muss das Altsystem für den Datenzugriff verfügbar bleiben.'
            }
          >
            <Input
              type="date"
              value={systemChangeDate}
              disabled={writeLock.locked}
              title={writeLock.hint}
              onChange={(e) => setSystemChangeDate(e.target.value)}
            />
          </Field>
          <Field helpSummary="Beschreiben Sie, wer in Ihrem Unternehmen Belege erfasst, prüft und freigibt."
            label="Organisationsanweisung"
            hint={orgFilledCount > 0 ? undefined : 'nicht hinterlegt'}
            explain="Wer scannt, wer prüft, wer freigibt und wie vertreten wird: die Teile der Verfahrensdokumentation, die nur das Unternehmen kennt. Buchfink gibt Muster vor."
          >
            <span className="flex items-center gap-3 min-w-0">
              <FieldValue className="truncate">
                {orgFilledCount > 0 ? organisationPreview : '—'}
              </FieldValue>
              {onNavigate && (
                <Button variant="quiet" size="sm" onClick={() => onNavigate('taxaudit')}>
                  Pflegen
                </Button>
              )}
            </span>
          </Field>
        </FormGrid>
      </Section>

      {/*
        Rechnungsprüfung und Mahnwesen (RECH-08, QUE-05). Beide Einstellungen
        steuern Vorgänge und keine Buchung: die eine sagt, ab welchem Betrag ein
        Eingangsbeleg einen Prüfvermerk braucht, die andere, wann welche
        Mahnung vorgeschlagen wird.
      */}
      <Section helpSummary="Legen Sie fest, wann eine Leistungsprüfung nötig ist und wie Sie offene Rechnungen anmahnen."
        title="Rechnungsprüfung und Mahnwesen"
        context="Leistungsnachweis und Mahnstufen"
        explain={
          <>
            Der Vorsteuerabzug setzt eine tatsächlich bezogene Leistung voraus (§ 15 UStG); der
            Leistungsnachweis am Beleg hält fest, wer das bestätigt hat. Die Mahnstufen sind eine
            Vereinbarung des Unternehmens mit sich selbst — das Gesetz kennt keine „erste Mahnung".
            Die Gebühr ist ein Vorschlag und nur ersatzfähig, soweit sie tatsächlich entstandener
            Verzugsschaden ist.
          </>
        }
      >
        <FormGrid>
          <Field label="Leistungsnachweis ab" hint="Bruttobetrag des Eingangsbelegs">
            <Input
              align="right"
              value={checkThresholdText}
              onChange={(e) => setCheckThresholdText(e.target.value)}
              onBlur={() => {
                const value = parseCents(checkThresholdText);
                // Eine Null ist hier ein leeres Feld, keine Abschaltung: wer
                // keinen Nachweis verlangt, setzt die Grenze hoch. Das
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
        </FormGrid>

        <div className="mt-8">
          <DunningLevelsTable
            levels={settings.dunningLevels ?? []}
            onChange={(levels) => patch({ dunningLevels: levels })}
          />
        </div>
      </Section>

      <Section helpSummary="Der veröffentlichte Basiszinssatz ist die Grundlage für die Berechnung von Verzugszinsen."
        title="Basiszinssatz"
        context="Grundlage der Verzugszinsen"
        explain={
          <>
            Die Deutsche Bundesbank setzt den Basiszinssatz zum 1. Januar und zum 1. Juli neu fest
            und gibt ihn im Bundesanzeiger bekannt (§ 247 BGB). Die Verzugszinsen liegen neun
            Prozentpunkte darüber, gegenüber einem Verbraucher fünf (§ 288 BGB). Ein Wert, der als
            „zu prüfen" steht, ist fortgeschrieben und nicht bekanntgegeben — er ist gegen die
            Bekanntgabe abzugleichen.
          </>
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

      <Section helpSummary="Buchfink merkt sich bestätigte Zuordnungen für wiederkehrende Bankumsätze."
        title="Gelernte Bankregeln"
        context="Muster wiederkehrender Umsätze"
        explain={
          <>
            Miete, Kontoführungsentgelt und Gehalt kommen jeden Monat wieder und haben keinen
            offenen Posten. Buchfink merkt sich, wogegen ein solcher Umsatz zuletzt gebucht wurde,
            und schlägt es beim nächsten Mal vor. Gebucht wird davon nichts von selbst: Eine Regel,
            die selbst bucht, würde aus einem einmaligen Griff eine Gewohnheit machen, die niemand
            mehr prüft.
          </>
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
        {/* Eine Spalte über die volle Breite: hier stehen Pfade, und ein Pfad
            in einem halben Feld ist ein abgeschnittener Pfad. */}
        <FormGrid cols={1}>
          {/* Der Datenordner in einem Synchronisationsordner: § 146 Abs. 2, 2a
              AO. Der Satz kommt aus dem Backend, weil nur dort bekannt ist,
              welche Pfadbestandteile als Cloud-Ordner gelten (ARC-06). */}
          {hints?.cloudWarning && <Notice tone="negative" text={hints.cloudWarning} />}

          {/* Kein Knopf zum Wählen: Buchfink zieht einen Datenordner nicht um,
              und ein Knopf, der nur die Anzeige ändert, verspräche das (ARC-06). */}
          <Field helpSummary="Hier liegen die Buchhaltungsdaten und Belege Ihres Unternehmens."
            label="Ordner für Buchungsdaten und Belege"
            explain="Der Ordner steht beim Anlegen des Mandanten fest und lässt sich hier nicht umziehen."
          >
            <Input className="code-num" value={appConfig?.dataDir || ''} readOnly />
          </Field>

          <Field helpSummary="Wählen Sie einen Zielordner, damit Buchfink Sicherungen erstellen kann."
            label="Sicherungsordner"
            hint="Einzurichten unter Datenzugriff"
            explain="Ohne Sicherungsordner schreibt Buchfink keine Sicherung — weder von Hand noch beim Beenden."
          >
            <Input
              className="code-num"
              value={appConfig?.backupDir || ''}
              readOnly
              placeholder="Nicht eingerichtet"
            />
          </Field>

          <Field helpSummary="Die Programmversion hilft, Exporte und Sicherungen dem verwendeten Stand zuzuordnen." label="Programmversion" explain="Sie steht in jedem Export und in jeder Sicherung.">
            <Input className="code-num" value={appConfig?.programVersion || 'dev'} readOnly />
          </Field>

          <Field helpSummary="Hier sehen Sie, ob der Schlüssel für Ihre Daten auf diesem Rechner verfügbar ist." label="Schlüssel im Schlüsselbund des Betriebssystems" explain={keychainHint()}>
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
              <Help summary="Bewahren Sie die Wiederherstellungsdatei getrennt von Rechner und Datensicherung auf." label="Erklärung zum Recovery-Schlüssel">
                Geht dieser Rechner verloren, ist der Schlüsselbund weg und die verschlüsselten
                Daten sind ohne Recovery-Datei unwiederbringlich. Die Datei gehört an einen anderen
                Ort als die Datensicherung, damit der Besitz der Sicherung allein keinen Zugriff auf die verschlüsselten Daten ermöglicht.
              </Help>
            </h3>
            <p className="text-body text-ink-muted mt-1">
              {appConfig?.tenants.find((t) => t.id === appConfig.activeTenantId)?.recoveryExportedAt ? 'Ein Wiederherstellungsschlüssel wurde bereits exportiert. Bewahren Sie die Datei getrennt und sicher auf.' : 'Noch keine Schlüsselsicherung vermerkt. Ohne diese Datei sind die verschlüsselten Daten bei Verlust des Rechners nicht wiederherstellbar.'}
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
        </FormGrid>
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

  /** Entfernt eine Stufe. Die Entwürfe werden geleert: Ihr Schlüssel ist der Index, und der verschiebt sich nach dem Entfernen. */
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
