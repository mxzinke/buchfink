/**
 * Wer eine Buchung erfasst hat: ein Mensch in der App, ein verbundener Agent
 * oder ein Programmlauf. Jede Buchung speichert den Bearbeiter und den
 * Erfassungsweg (`ui`, `system` oder `agent:<kurzname>`).
 */
export type ActorKind = 'agent' | 'system' | 'person';

const AGENT_PREFIX = 'agent:';

/** Schreibweisen bekannter Apps; sonst wird der Kurzname wortweise großgeschrieben. */
const APP_NAMES: Record<string, string> = {
  'claude-code': 'Claude Code',
  'claude-desktop': 'Claude Desktop',
  claude: 'Claude',
  opencode: 'OpenCode',
  codex: 'Codex',
  pi: 'Pi',
};

export function actorKind(channel?: string): ActorKind {
  if (channel?.startsWith(AGENT_PREFIX)) return 'agent';
  if (channel === 'system') return 'system';
  return 'person';
}

/** Der Name der App aus dem Erfassungsweg `agent:<kurzname>`. */
export function agentName(channel?: string): string {
  const slug = channel?.startsWith(AGENT_PREFIX) ? channel.slice(AGENT_PREFIX.length) : '';
  if (!slug) return 'Agent';
  return (
    APP_NAMES[slug] ??
    slug
      .split('-')
      .filter(Boolean)
      .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
      .join(' ')
  );
}

/** Wer erfasst hat und auf welchem Weg, etwa „Claude Code · anna@nordlicht-mbp“. */
export function actorWho(actor?: string, channel?: string): string {
  switch (actorKind(channel)) {
    case 'agent':
      return actor ? `${agentName(channel)} · ${actor}` : agentName(channel);
    case 'system':
      return 'Buchfink (Programmlauf)';
    default:
      return actor ? `${actor} in Buchfink` : 'In Buchfink';
  }
}

/** Der Satz für den Tooltip. */
export function actorLabel(actor?: string, channel?: string): string {
  return `Erfasst von ${actorWho(actor, channel)}`;
}

/**
 * Der Vermerk zum Erfassungsweg neben dem Bearbeiter. Die Bedienung in der App
 * ist der Regelfall und bekommt keinen Vermerk; ein Programmlauf oder eine App
 * steht mit ihrem Anzeigenamen da.
 */
export function channelNote(channel?: string): string | null {
  if (!channel || channel === 'ui') return null;
  if (channel === 'system') return 'Programmlauf';
  if (channel.startsWith(AGENT_PREFIX)) return `über ${agentName(channel)}`;
  return channel;
}
