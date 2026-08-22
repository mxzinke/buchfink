// SPDX-FileCopyrightText: 2026 Maximilian Pfennig
// SPDX-License-Identifier: EUPL-1.2

import React, { useEffect, useState } from 'react';
import { Plus, Building, User, Mail, CreditCard } from 'lucide-react';
import { Contact } from '../types';
import { Api } from '../services/api';
import { formatCents } from '../utils/formatters';
import { HelpTooltip } from '../components/HelpTooltip';

export const ContactsPage: React.FC = () => {
  const [contacts, setContacts] = useState<Contact[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadContacts();
  }, []);

  const loadContacts = async () => {
    setLoading(true);
    try {
      const list = await Api.getContacts();
      setContacts(list);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="p-3 sm:p-6 md:p-8 max-w-7xl mx-auto space-y-4 sm:space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold text-stone-900 tracking-tight flex items-center">
            Kunden & Lieferanten
            <HelpTooltip
              title="Kontakte & Zahlungsübersicht"
              content="Verwalten Sie Ihre Kunden und Lieferanten, um Rechnungen schnell zuzuordnen und offene Rechnungsbeträge nachzuvollziehen."
            />
          </h2>
          <p className="text-xs text-stone-500 mt-1">
            Kontaktdaten und offene Beträge im Überblick
          </p>
        </div>

        <button
          onClick={() => alert('Kontakt anlegen')}
          className="flex items-center gap-1.5 px-3.5 py-2 rounded-lg bg-amber-700 text-white text-xs font-semibold hover:bg-amber-800 transition-colors shadow-xs"
        >
          <Plus className="w-3.5 h-3.5" />
          Neuer Kontakt
        </button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5">
        {loading ? (
          <div className="col-span-full py-12 text-center text-stone-400 text-xs">
            Kontakte werden geladen...
          </div>
        ) : contacts.length === 0 ? (
          <div className="col-span-full py-12 text-center text-stone-400 text-xs">
            Noch keine Kontakte angelegt.
          </div>
        ) : (
          contacts.map((c) => (
            <div
              key={c.id}
              className="bg-white p-5 rounded-xl border border-stone-200/80 shadow-xs space-y-4 hover:border-amber-500/40 transition-colors"
            >
              <div className="flex items-start justify-between">
                <div className="flex items-center gap-3">
                  <div
                    className={`p-2.5 rounded-lg ${
                      c.type === 'customer'
                        ? 'bg-amber-50 text-amber-700'
                        : 'bg-stone-100 text-stone-700'
                    }`}
                  >
                    {c.company ? <Building className="w-4 h-4" /> : <User className="w-4 h-4" />}
                  </div>
                  <div>
                    <h3 className="font-bold text-sm text-stone-900">{c.name}</h3>
                    <div className="text-xs font-mono text-stone-500">{c.ledgerAccount}</div>
                  </div>
                </div>

                <span
                  className={`px-2 py-0.5 rounded text-[10px] font-semibold ${
                    c.type === 'customer'
                      ? 'bg-amber-100 text-amber-800'
                      : 'bg-stone-200 text-stone-800'
                  }`}
                >
                  {c.type === 'customer' ? 'Kunde' : 'Lieferant'}
                </span>
              </div>

              <div className="space-y-1 text-xs text-stone-600">
                {c.email && (
                  <div className="flex items-center gap-2 truncate">
                    <Mail className="w-3.5 h-3.5 text-stone-400 shrink-0" />
                    <span>{c.email}</span>
                  </div>
                )}
                {c.iban && (
                  <div className="flex items-center gap-2 truncate font-mono text-xs">
                    <CreditCard className="w-3.5 h-3.5 text-stone-400 shrink-0" />
                    <span>{c.iban}</span>
                  </div>
                )}
              </div>

              <div className="pt-3 border-t border-stone-100 flex items-center justify-between text-xs">
                <span className="text-stone-500">Offener Betrag:</span>
                <span className="font-mono font-bold text-stone-900">
                  {formatCents(c.openAmount)}
                </span>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
};
