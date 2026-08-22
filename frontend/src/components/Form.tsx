import React from 'react';
import { X } from 'lucide-react';

/**
 * Die geteilten Formular-Bausteine.
 *
 * `inputClass` und `Field` waren dreimal wortgleich in JournalPage,
 * InvoicesPage und BankImportPage kopiert. Jede neue Maske hätte die vierte
 * Kopie erzeugt.
 */

export const inputClass =
  'w-full px-2.5 py-1.5 text-sm rounded-lg border border-stone-200 focus:border-amber-500 focus:ring-1 focus:ring-amber-500 outline-none disabled:bg-stone-50 disabled:text-stone-400';

export const Field: React.FC<{
  label: string;
  hint?: string;
  children: React.ReactNode;
  className?: string;
}> = ({ label, hint, children, className }) => (
  <label className={`block ${className ?? ''}`}>
    <span className="block text-xs font-medium text-stone-600 mb-1">
      {label}
      {hint && <span className="text-stone-400 font-normal"> · {hint}</span>}
    </span>
    {children}
  </label>
);

/** Ein roter Kasten für Backend-Meldungen. Fachliche Fehler kommen von dort. */
export const ErrorBox: React.FC<{ message?: string | null }> = ({ message }) =>
  message ? (
    <div className="rounded-lg bg-rose-50 border border-rose-200 px-3 py-2 text-xs text-rose-700 whitespace-pre-line">
      {message}
    </div>
  ) : null;

export const Modal: React.FC<{
  title: string;
  onClose: () => void;
  children: React.ReactNode;
  footer?: React.ReactNode;
  width?: string;
}> = ({ title, onClose, children, footer, width = 'max-w-3xl' }) => (
  <div className="fixed inset-0 bg-stone-900/40 flex items-start justify-center p-4 overflow-y-auto z-50">
    <div className={`bg-white rounded-2xl border border-stone-200 shadow-xl w-full ${width} my-8`}>
      <div className="flex items-center justify-between px-5 py-3 border-b border-stone-100">
        <h2 className="font-semibold text-stone-900">{title}</h2>
        <button type="button" onClick={onClose} className="text-stone-400 hover:text-stone-700">
          <X className="w-5 h-5" />
        </button>
      </div>
      {children}
      {footer && (
        <div className="flex justify-end gap-2 px-5 py-3 border-t border-stone-100">{footer}</div>
      )}
    </div>
  </div>
);

export const PrimaryButton: React.FC<React.ButtonHTMLAttributes<HTMLButtonElement>> = ({
  className,
  ...props
}) => (
  <button
    {...props}
    className={`px-3.5 py-2 rounded-lg bg-amber-700 text-white text-xs font-semibold hover:bg-amber-800 disabled:bg-stone-300 disabled:cursor-not-allowed transition-colors ${className ?? ''}`}
  />
);

export const SecondaryButton: React.FC<React.ButtonHTMLAttributes<HTMLButtonElement>> = ({
  className,
  ...props
}) => (
  <button
    {...props}
    className={`px-3.5 py-2 rounded-lg border border-stone-200 text-stone-600 text-xs font-semibold hover:bg-stone-50 transition-colors ${className ?? ''}`}
  />
);
