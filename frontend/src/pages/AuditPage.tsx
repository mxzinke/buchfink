import React, { useEffect, useState } from 'react';
import { ShieldCheck, ShieldAlert, RefreshCw } from 'lucide-react';
import { AuditLogEntry, IntegrityCheckResult } from '../types';
import { Api } from '../services/api';
import { HelpTooltip } from '../components/HelpTooltip';

export const AuditPage: React.FC = () => {
  const [logs, setLogs] = useState<AuditLogEntry[]>([]);
  const [integrity, setIntegrity] = useState<IntegrityCheckResult | null>(null);
  const [isVerifying, setIsVerifying] = useState(false);

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    try {
      const [l, res] = await Promise.all([
        Api.getAuditLogs(),
        Api.verifyIntegrity(),
      ]);
      setLogs(l);
      setIntegrity(res);
    } catch (e) {
      console.error(e);
    }
  };

  const handleReverify = async () => {
    setIsVerifying(true);
    try {
      const res = await Api.verifyIntegrity();
      setIntegrity(res);
    } finally {
      setIsVerifying(false);
    }
  };

  return (
    <div className="p-8 max-w-7xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold text-stone-900 tracking-tight flex items-center">
            GoBD & Audit-Log (Verfahrensdokumentation)
            <HelpTooltip
              title="GoBD-Nachweisbarkeit ab v1"
              content="Die GoBD fordern Nachvollziehbarkeit, Unveränderbarkeit und lückenlose Protokollierung. Jede Buchung ist über SHA256 mit ihrem Vorgänger verkettet (Hash-Chain). Manipulationen fallen bei der Prüfung sofort auf."
            />
          </h2>
          <p className="text-xs text-stone-500 mt-1">
            Kryptografischer Integritätsbeweis & Protokoll aller Stammdatenänderungen
          </p>
        </div>

        <button
          onClick={handleReverify}
          disabled={isVerifying}
          className="flex items-center gap-2 px-3.5 py-2 rounded-lg bg-stone-900 text-stone-100 text-xs font-semibold hover:bg-stone-800 transition-colors shadow-xs"
        >
          <RefreshCw className={`w-3.5 h-3.5 ${isVerifying ? 'animate-spin text-amber-400' : ''}`} />
          Hash-Chain jetzt verifizieren
        </button>
      </div>

      {/* Integrity Card */}
      {integrity && (
        <div
          className={`p-6 rounded-xl border flex flex-col md:flex-row items-start md:items-center justify-between gap-4 ${
            integrity.isValid
              ? 'bg-emerald-50/70 border-emerald-200 text-emerald-950'
              : 'bg-rose-50 border-rose-200 text-rose-950'
          }`}
        >
          <div className="flex items-start gap-3.5">
            <div
              className={`p-2.5 rounded-lg shrink-0 ${
                integrity.isValid ? 'bg-emerald-600 text-white' : 'bg-rose-600 text-white'
              }`}
            >
              {integrity.isValid ? (
                <ShieldCheck className="w-5 h-5" />
              ) : (
                <ShieldAlert className="w-5 h-5" />
              )}
            </div>
            <div>
              <h3 className="font-bold text-sm">
                {integrity.isValid
                  ? 'Kryptografische Hash-Chain ist vollständig intakt'
                  : 'Achtung: Integritätsprüfung fehlgeschlagen!'}
              </h3>
              <p className="text-xs text-stone-600 mt-1">{integrity.message}</p>
              <div className="mt-2 text-[11px] font-mono text-stone-500">
                Letzter gültiger Block-Hash: {integrity.lastVerifiedHash} &bull; Geprüft am:{' '}
                {integrity.checkedAt}
              </div>
            </div>
          </div>

          <div className="text-right shrink-0">
            <span className="px-3 py-1 rounded-full text-xs font-bold bg-emerald-100 text-emerald-800 border border-emerald-300">
              {integrity.checkedEntries}/{integrity.totalEntries} Buchungen valide
            </span>
          </div>
        </div>
      )}

      {/* Audit Log Table */}
      <div className="bg-white rounded-xl border border-stone-200/90 shadow-xs overflow-hidden">
        <div className="p-4 border-b border-stone-200 bg-stone-50/60 flex items-center justify-between">
          <h3 className="text-xs font-bold text-stone-900 uppercase tracking-wider">
            Unveränderliches Audit-Log
          </h3>
          <span className="text-[11px] text-stone-500">GoBD-Prüfpfad</span>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs">
            <thead className="bg-stone-50 border-b border-stone-200 text-stone-500 font-medium">
              <tr>
                <th className="py-3 px-4 w-40">Zeitstempel</th>
                <th className="py-3 px-4 w-28">Aktion</th>
                <th className="py-3 px-4 w-32">Objekt</th>
                <th className="py-3 px-4">Details</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-stone-100">
              {logs.length === 0 ? (
                <tr>
                  <td colSpan={4} className="py-8 text-center text-stone-400">
                    Keine Logeinträge vorhanden.
                  </td>
                </tr>
              ) : (
                logs.map((l) => (
                  <tr key={l.id} className="hover:bg-stone-50 transition-colors">
                    <td className="py-3 px-4 text-stone-600 font-mono text-[11px]">
                      {l.timestamp}
                    </td>
                    <td className="py-3 px-4">
                      <span className="px-2 py-0.5 rounded text-[10px] font-mono font-bold bg-stone-100 text-stone-800">
                        {l.action}
                      </span>
                    </td>
                    <td className="py-3 px-4 text-stone-600 font-mono text-[11px]">
                      {l.entityType} ({l.entityId})
                    </td>
                    <td className="py-3 px-4 text-stone-900 font-medium">{l.details}</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};
