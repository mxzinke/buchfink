import React, { useState } from 'react';
import { CheckCircle2, ArrowRight, ArrowLeft } from 'lucide-react';
import { CompanySettings } from '../types';
import { Api } from '../services/api';

interface OnboardingModalProps {
  isOpen: boolean;
  onClose: () => void;
  initialSettings: CompanySettings | null;
  onCompleted: () => void;
}

export const OnboardingModal: React.FC<OnboardingModalProps> = ({
  isOpen,
  onClose,
  initialSettings,
  onCompleted,
}) => {
  const [step, setStep] = useState<1 | 2 | 3>(1);
  const [form, setForm] = useState<CompanySettings>({
    companyName: initialSettings?.companyName || '',
    legalForm: initialSettings?.legalForm || 'GmbH',
    fiscalYear: initialSettings?.fiscalYear || new Date().getFullYear(),
    taxNumber: initialSettings?.taxNumber || '',
    vatId: initialSettings?.vatId || '',
    taxOffice: initialSettings?.taxOffice || '',
    iban: initialSettings?.iban || '',
    bic: initialSettings?.bic || '',
    bankName: initialSettings?.bankName || '',
    street: initialSettings?.street || '',
    zipCity: initialSettings?.zipCity || '',
    country: initialSettings?.country || 'Deutschland',
    currency: 'EUR',
    skr: 'SKR04',
  });
  const [isSaving, setIsSaving] = useState(false);

  if (!isOpen) return null;

  const handleFinish = async () => {
    setIsSaving(true);
    try {
      await Api.updateCompanySettings(form);
      onCompleted();
      onClose();
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 bg-stone-950/70 backdrop-blur-sm flex items-center justify-center p-4 overflow-y-auto">
      <div className="bg-white rounded-2xl shadow-2xl max-w-xl w-full p-6 border border-stone-200 animate-in fade-in my-8 space-y-6">
        {/* Step Indicator */}
        <div className="flex items-center justify-between border-b border-stone-100 pb-4">
          <div className="flex items-center gap-3">
            <img src="/buchfink-logo.svg" alt="Buchfink" className="w-8 h-8 rounded-lg bg-stone-100 p-0.5" />
            <div>
              <h2 className="text-base font-bold text-stone-900">Setup-Assistent</h2>
              <p className="text-xs text-stone-500">
                Schritt {step} von 3: {step === 1 ? 'Unternehmensdaten' : step === 2 ? 'Bankverbindung' : 'Abschluss'}
              </p>
            </div>
          </div>
          <div className="flex items-center gap-1.5">
            {[1, 2, 3].map((s) => (
              <div
                key={s}
                className={`w-2.5 h-2.5 rounded-full transition-all ${
                  step === s ? 'bg-amber-600 w-6' : step > s ? 'bg-amber-400' : 'bg-stone-200'
                }`}
              />
            ))}
          </div>
        </div>

        {/* Step 1: Company Profile */}
        {step === 1 && (
          <div className="space-y-4 text-xs">
            <div>
              <label className="font-semibold text-stone-700 block mb-1">Firmen- / Unternehmensname:</label>
              <input
                type="text"
                placeholder="z. B. Meine Firma GmbH oder Max Mustermann"
                value={form.companyName}
                onChange={(e) => setForm({ ...form, companyName: e.target.value })}
                className="w-full p-2.5 bg-stone-50 border border-stone-200 rounded-lg text-xs"
                required
              />
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="font-semibold text-stone-700 block mb-1">Rechtsform:</label>
                <select
                  value={form.legalForm}
                  onChange={(e) => setForm({ ...form, legalForm: e.target.value })}
                  className="w-full p-2.5 bg-stone-50 border border-stone-200 rounded-lg text-xs"
                >
                  <option value="Einzelunternehmen">Einzelunternehmen / Freelancer</option>
                  <option value="UG (haftungsbeschränkt)">UG (haftungsbeschränkt)</option>
                  <option value="GmbH">GmbH</option>
                  <option value="GbR">GbR</option>
                  <option value="AG">AG</option>
                </select>
              </div>

              <div>
                <label className="font-semibold text-stone-700 block mb-1">Geschäftsjahr:</label>
                <input
                  type="number"
                  value={form.fiscalYear}
                  onChange={(e) => setForm({ ...form, fiscalYear: Number(e.target.value) })}
                  className="w-full p-2.5 bg-stone-50 border border-stone-200 rounded-lg text-xs font-mono"
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="font-semibold text-stone-700 block mb-1">Steuernummer (Finanzamt):</label>
                <input
                  type="text"
                  placeholder="12/345/67890"
                  value={form.taxNumber}
                  onChange={(e) => setForm({ ...form, taxNumber: e.target.value })}
                  className="w-full p-2.5 bg-stone-50 border border-stone-200 rounded-lg text-xs font-mono"
                />
              </div>
              <div>
                <label className="font-semibold text-stone-700 block mb-1">USt-IdNr. (Optional):</label>
                <input
                  type="text"
                  placeholder="DE123456789"
                  value={form.vatId}
                  onChange={(e) => setForm({ ...form, vatId: e.target.value })}
                  className="w-full p-2.5 bg-stone-50 border border-stone-200 rounded-lg text-xs font-mono"
                />
              </div>
            </div>
          </div>
        )}

        {/* Step 2: Bank Connection */}
        {step === 2 && (
          <div className="space-y-4 text-xs">
            <div>
              <label className="font-semibold text-stone-700 block mb-1">Geschäftskonto IBAN:</label>
              <input
                type="text"
                placeholder="DE89 3704 0044 0532 0130 00"
                value={form.iban}
                onChange={(e) => setForm({ ...form, iban: e.target.value })}
                className="w-full p-2.5 bg-stone-50 border border-stone-200 rounded-lg text-xs font-mono"
              />
              <p className="text-[11px] text-stone-500 mt-1">
                Wird im SKR04 automatisch mit dem Haupt-Bankkonto (Konto 1800) verknüpft.
              </p>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="font-semibold text-stone-700 block mb-1">Bankname:</label>
                <input
                  type="text"
                  placeholder="z. B. Commerzbank, Qonto, N26"
                  value={form.bankName}
                  onChange={(e) => setForm({ ...form, bankName: e.target.value })}
                  className="w-full p-2.5 bg-stone-50 border border-stone-200 rounded-lg text-xs"
                />
              </div>
              <div>
                <label className="font-semibold text-stone-700 block mb-1">BIC (Optional):</label>
                <input
                  type="text"
                  placeholder="COBADEFFXXX"
                  value={form.bic}
                  onChange={(e) => setForm({ ...form, bic: e.target.value })}
                  className="w-full p-2.5 bg-stone-50 border border-stone-200 rounded-lg text-xs font-mono"
                />
              </div>
            </div>

            <div className="p-3.5 bg-stone-50 rounded-xl border border-stone-200 text-stone-600 space-y-1">
              <span className="font-semibold text-stone-900 block">Kontenrahmen: SKR04 (Standard)</span>
              <p className="text-[11px] leading-relaxed">
                Buchfink verwendet die Abschlussgliederung des Standardkontenrahmens 04 mit vorinstallierten Standardkonten.
              </p>
            </div>
          </div>
        )}

        {/* Step 3: Confirmation */}
        {step === 3 && (
          <div className="space-y-4 text-xs">
            <div className="p-4 bg-amber-50/60 rounded-xl border border-amber-200/80 space-y-2">
              <span className="font-bold text-amber-900 block">Zusammenfassung:</span>
              <div className="space-y-1 text-stone-700">
                <div><strong>Unternehmen:</strong> {form.companyName || 'Nicht angegeben'} ({form.legalForm})</div>
                <div><strong>Geschäftsjahr:</strong> {form.fiscalYear}</div>
                <div><strong>Steuernummer:</strong> {form.taxNumber || '—'}</div>
                <div><strong>Bank (Konto 1800):</strong> {form.bankName || 'Hausbank'} ({form.iban || '—'})</div>
                <div><strong>Speicherort:</strong> Lokale SQLite-Datenbank ({form.fiscalYear}.sqlite)</div>
              </div>
            </div>
            <p className="text-[11px] text-stone-500">
              Sie können alle Angaben jederzeit in den Einstellungen ändern.
            </p>
          </div>
        )}

        {/* Action Buttons */}
        <div className="flex items-center justify-between border-t border-stone-100 pt-4">
          {step > 1 ? (
            <button
              type="button"
              onClick={() => setStep((s) => (s - 1) as any)}
              className="px-4 py-2 text-xs font-semibold text-stone-600 hover:bg-stone-100 rounded-lg flex items-center gap-1.5"
            >
              <ArrowLeft className="w-3.5 h-3.5" /> Zurück
            </button>
          ) : (
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 text-xs font-semibold text-stone-500 hover:bg-stone-100 rounded-lg"
            >
              Später einrichten
            </button>
          )}

          {step < 3 ? (
            <button
              type="button"
              disabled={step === 1 && !form.companyName.trim()}
              onClick={() => setStep((s) => (s + 1) as any)}
              className="px-5 py-2 text-xs font-semibold bg-amber-600 hover:bg-amber-700 disabled:opacity-50 text-white rounded-lg shadow-xs flex items-center gap-1.5"
            >
              Weiter <ArrowRight className="w-3.5 h-3.5" />
            </button>
          ) : (
            <button
              type="button"
              disabled={isSaving}
              onClick={handleFinish}
              className="px-6 py-2 text-xs font-semibold bg-amber-600 hover:bg-amber-700 text-white rounded-lg shadow-xs flex items-center gap-1.5"
            >
              <CheckCircle2 className="w-4 h-4" />
              {isSaving ? 'Wird gespeichert...' : 'Buchhaltung jetzt starten'}
            </button>
          )}
        </div>
      </div>
    </div>
  );
};
