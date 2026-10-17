import React, { useEffect, useRef, useState } from 'react';
import { Check, Copy, ExternalLink } from 'lucide-react';
import { Clipboard } from '@wailsio/runtime';
import { Api } from '../services/api';
import type { AgentClient } from '../types';
import { cn, toast } from './ui';

/** Die Adresse, solange der Server noch keine gemeldet hat. */
export const DEFAULT_MCP_URL = 'http://127.0.0.1:7465/mcp';

interface ClientStep {
  label: string;
  /** Der Text zum Kopieren; ohne ihn steht `text` als Satz da. */
  command?: (url: string) => string;
  text?: string;
}

interface ClientInfo {
  id: AgentClient;
  name: string;
  steps: ClientStep[];
}

/**
 * Je App der kürzeste Weg zur Verbindung. Die Befehle folgen den Anleitungen
 * in docs/nutzung/ki-agenten/; dort stehen Skills, Rechte und Fehlerbilder.
 */
export const AGENT_CLIENTS: ClientInfo[] = [
  {
    id: 'claude-code',
    name: 'Claude Code',
    steps: [
      {
        label: 'Eintragen',
        command: (url) => `claude mcp add --transport http --scope user buchfink ${url}`,
      },
      { label: 'Anmelden', command: () => 'claude mcp login buchfink' },
    ],
  },
  {
    id: 'claude-desktop',
    name: 'Claude Desktop',
    steps: [
      { label: 'Erweiterung', text: 'buchfink.mcpb aus der Anleitung laden und doppelklicken' },
      { label: 'Im Chat', command: () => 'Rufe das Buchfink-Werkzeug status auf.' },
    ],
  },
  {
    id: 'opencode',
    name: 'OpenCode',
    steps: [
      {
        label: 'In opencode.json unter „mcp“',
        command: (url) => `"buchfink": { "type": "remote", "url": "${url}" }`,
      },
      { label: 'Anmelden', command: () => 'opencode mcp auth buchfink' },
    ],
  },
  {
    id: 'codex',
    name: 'Codex',
    steps: [
      { label: 'Eintragen', command: (url) => `codex mcp add buchfink --url ${url}` },
      { label: 'Anmelden', command: () => 'codex mcp login buchfink' },
    ],
  },
  {
    id: 'pi',
    name: 'Pi',
    steps: [
      {
        label: 'Eintragen',
        command: (url) => `pi mcp add buchfink --url ${url} --exposure direct`,
      },
      { label: 'Anmelden', command: () => 'pi mcp login buchfink' },
    ],
  },
];

/** Die Rollen auf Papier und auf der dunklen Schale der Einrichtung (§12). */
const TONES = {
  paper: {
    switcher: 'border-line',
    option: 'text-ink-subtle hover:text-ink',
    optionActive: 'text-ink font-semibold',
    marker: 'bg-accent',
    label: 'text-ink-subtle',
    code: 'border-line bg-surface text-ink',
    copy: 'text-ink-subtle hover:bg-sunken hover:text-ink',
    text: 'text-ink-muted',
    link: 'text-accent-text hover:underline',
    note: 'text-ink-subtle',
  },
  shell: {
    switcher: 'border-shell-line',
    option: 'text-shell-text-muted hover:text-shell-text',
    optionActive: 'text-white font-semibold',
    marker: 'bg-accent-light',
    label: 'text-shell-text-muted',
    code: 'border-shell-line bg-shell-deep text-shell-text',
    copy: 'text-shell-text-muted hover:bg-shell-raised hover:text-shell-text',
    text: 'text-shell-text',
    link: 'text-accent-light hover:underline',
    note: 'text-shell-text-muted',
  },
} as const;

/** Schreibt Text in die Zwischenablage, über die Laufzeit oder den Browser. */
async function copyText(text: string): Promise<void> {
  try {
    await Clipboard.SetText(text);
  } catch {
    await navigator.clipboard.writeText(text);
  }
}

const CommandLine: React.FC<{ value: string; tone: keyof typeof TONES }> = ({ value, tone }) => {
  const style = TONES[tone];
  const [copied, setCopied] = useState(false);
  const timer = useRef<number>();

  useEffect(() => () => window.clearTimeout(timer.current), []);

  async function copy() {
    try {
      await copyText(value);
      setCopied(true);
      window.clearTimeout(timer.current);
      timer.current = window.setTimeout(() => setCopied(false), 2000);
    } catch {
      toast.error('Die Zwischenablage ist nicht erreichbar. Markieren Sie den Befehl und kopieren Sie ihn von Hand.');
    }
  }

  return (
    <div className={cn('flex items-center gap-2 rounded-control border pl-3 pr-1 h-9 min-w-0', style.code)}>
      <code className="flex-1 min-w-0 truncate font-mono text-caption select-all" title={value}>
        {value}
      </code>
      <button
        type="button"
        onClick={() => void copy()}
        title={copied ? 'Kopiert' : 'Kopieren'}
        aria-label={copied ? 'Kopiert' : `Kopieren: ${value}`}
        className={cn(
          'grid place-items-center w-7 h-7 shrink-0 rounded-control transition-colors duration-120 ease-quiet',
          style.copy,
        )}
      >
        {copied ? (
          <Check className="w-3.5 h-3.5" strokeWidth={1.5} />
        ) : (
          <Copy className="w-3.5 h-3.5" strokeWidth={1.5} />
        )}
      </button>
    </div>
  );
};

export interface AgentClientCommandsProps {
  /** Die Adresse des Servers, wie Buchfink sie meldet. */
  url?: string;
  tone?: keyof typeof TONES;
  className?: string;
}

/**
 * Die fünf Apps als Umschalter, darunter für die gewählte die Befehle zum
 * Kopieren und der Weg zur Anleitung.
 */
export const AgentClientCommands: React.FC<AgentClientCommandsProps> = ({
  url,
  tone = 'paper',
  className,
}) => {
  const style = TONES[tone];
  const [selected, setSelected] = useState<AgentClient>('claude-code');
  const client = AGENT_CLIENTS.find((c) => c.id === selected) ?? AGENT_CLIENTS[0];
  const address = url || DEFAULT_MCP_URL;
  const customPort = address !== DEFAULT_MCP_URL;

  async function openGuide() {
    try {
      await Api.openAgentGuide(client.id);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    }
  }

  return (
    <div className={className}>
      <div
        role="radiogroup"
        aria-label="KI-App"
        className={cn('flex flex-wrap items-center gap-x-5 gap-y-1 border-b', style.switcher)}
      >
        {AGENT_CLIENTS.map((c) => {
          const active = c.id === selected;
          return (
            <button
              key={c.id}
              type="button"
              role="radio"
              aria-checked={active}
              onClick={() => setSelected(c.id)}
              className={cn(
                'relative -mb-px pb-2.5 pt-1 text-label transition-colors duration-120 ease-quiet',
                active ? style.optionActive : style.option,
              )}
            >
              {c.name}
              {active && (
                <span className={cn('absolute inset-x-0 bottom-0 h-0.5', style.marker)} aria-hidden="true" />
              )}
            </button>
          );
        })}
      </div>

      <ol className="mt-4 flex flex-col gap-3">
        {client.steps.map((step) => (
          <li key={step.label} className="grid grid-cols-1 sm:grid-cols-[9rem_minmax(0,1fr)] sm:items-center gap-1 sm:gap-4">
            <span className={cn('text-label', style.label)}>{step.label}</span>
            {step.command ? (
              <CommandLine value={step.command(address)} tone={tone} />
            ) : (
              <span className={cn('text-body', style.text)}>{step.text}</span>
            )}
          </li>
        ))}
      </ol>

      <div className="mt-4 flex flex-wrap items-center gap-x-4 gap-y-1">
        <button
          type="button"
          onClick={() => void openGuide()}
          className={cn('inline-flex items-center gap-1.5 text-label', style.link)}
        >
          Anleitung für {client.name}
          <ExternalLink className="w-3.5 h-3.5" strokeWidth={1.5} aria-hidden="true" />
        </button>
        {client.id === 'claude-desktop' && customPort && (
          <span className={cn('text-caption', style.note)}>
            In der Erweiterung die Adresse {address} eintragen.
          </span>
        )}
      </div>
    </div>
  );
};
